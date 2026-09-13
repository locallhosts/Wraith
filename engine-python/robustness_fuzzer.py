"""
robustness_fuzzer.py

Every detection pipeline I've seen tests a rule against exactly one
canonical Atomic Red Team payload. That answers "does this fire on the
textbook case" — it says nothing about whether a real attacker one
`-enc` vs `-EncodedCommand` abbreviation away from your exact string match
sails straight past it. This module generates a population of adversarial
variants of the tested technique's telemetry using well-documented,
publicly known obfuscation classes (case variation, PowerShell's own
built-in partial-parameter matching, environment-variable path expansion,
whitespace variation) and measures what fraction the rule still catches —
a "Robustness Score" surfaced right next to the pass/fail verdict.

This is a defensive purple-teaming technique (the same idea behind
DeTT&CT's "visibility vs. detection" scoring and MITRE's evaluations), not
an offensive obfuscation toolkit — every mutation class here is basic and
already assumed by any competent detection engineer; the value is in
running it systematically and automatically on every rule, every PR,
which nobody currently does.
"""
from __future__ import annotations

import copy
import itertools
import random
import re
from dataclasses import dataclass, field
from typing import Callable, List

# --- Mutation primitives -----------------------------------------------

def _case_variation(cmdline: str, rng: random.Random) -> str:
    """Randomly alters case per-character. Windows command execution is
    case-insensitive, so 'powershell.exe' and 'PoWeRsHeLL.exe' behave
    identically but defeat a naive case-sensitive substring match."""
    return "".join(c.upper() if rng.random() > 0.5 else c.lower() for c in cmdline)


# PowerShell accepts any unambiguous prefix of a full parameter name — this
# is a documented, intentional PowerShell feature (not a bypass technique),
# but it means a rule matching the literal string '-EncodedCommand' misses
# '-enc', '-en', '-e' etc., all of which PowerShell executes identically.
_POWERSHELL_PARAM_ALIASES = ["-e", "-en", "-enc", "-enco", "-EncodedCommand"]


def _powershell_param_alias(cmdline: str, rng: random.Random) -> str:
    if "-EncodedCommand" not in cmdline and "-encodedcommand" not in cmdline.lower():
        return cmdline
    alias = rng.choice(_POWERSHELL_PARAM_ALIASES)
    return re.sub(r"-EncodedCommand", alias, cmdline, flags=re.IGNORECASE)


# Common environment-variable equivalents Windows resolves identically at
# execution time to a literal path.
_ENV_VAR_SUBSTITUTIONS = {
    r"C:\\Windows\\": ["%WINDIR%\\", "%SystemRoot%\\"],
    r"C:\\Windows\\System32\\": ["%WINDIR%\\System32\\", "%SystemRoot%\\System32\\"],
}


def _env_var_substitution(cmdline: str, rng: random.Random) -> str:
    for literal, alternatives in _ENV_VAR_SUBSTITUTIONS.items():
        if re.search(literal, cmdline, flags=re.IGNORECASE):
            return re.sub(literal, rng.choice(alternatives), cmdline, count=1, flags=re.IGNORECASE)
    return cmdline


def _whitespace_variation(cmdline: str, rng: random.Random) -> str:
    """Windows argument parsing tolerates repeated/extra whitespace between
    tokens; a rule anchored on a single-space substring can miss this."""
    return re.sub(r" ", lambda _: " " * rng.randint(1, 3), cmdline)


MUTATORS: List[Callable[[str, random.Random], str]] = [
    _case_variation,
    _powershell_param_alias,
    _env_var_substitution,
    _whitespace_variation,
]


@dataclass
class Variant:
    mutation_chain: List[str]
    event: dict


def generate_variants(base_event: dict, n: int, seed: int | None = None,
                       max_mutations_per_variant: int = 2) -> List[Variant]:
    """Produces up to n adversarial variants of base_event by applying 1-2
    randomly chosen mutators to its command_line/process_name fields."""
    rng = random.Random(seed)
    variants: List[Variant] = []

    cmdline_field = None
    for candidate in ("command_line", "commandline", "CommandLine"):
        if candidate in base_event:
            cmdline_field = candidate
            break

    if cmdline_field is None:
        return []  # nothing to mutate for this event shape

    for _ in range(n):
        k = rng.randint(1, max_mutations_per_variant)
        chosen = rng.sample(MUTATORS, k=min(k, len(MUTATORS)))
        mutated = copy.deepcopy(base_event)
        chain_names = []
        for mutator in chosen:
            mutated[cmdline_field] = mutator(mutated[cmdline_field], rng)
            chain_names.append(mutator.__name__.strip("_"))
        mutated["wraith_label"] = "attack-fuzz"
        mutated["wraith_mutation_chain"] = chain_names
        variants.append(Variant(mutation_chain=chain_names, event=mutated))

    # de-duplicate identical resulting command lines (case variation can
    # occasionally collide with itself under a fixed seed at small n)
    seen = set()
    unique = []
    for v in variants:
        key = v.event[cmdline_field]
        if key not in seen:
            seen.add(key)
            unique.append(v)
    return unique


@dataclass
class RobustnessResult:
    rule_id: str
    variants_tested: int
    variants_detected: int
    score: float  # variants_detected / variants_tested, 1.0 = fully robust
    undetected_examples: List[dict] = field(default_factory=list)


def score_robustness(rule_id: str, variants: List[Variant],
                      matches_query: Callable[[dict], bool]) -> RobustnessResult:
    """Given a population of variants and a predicate that tells us whether
    a given event would match the rule's compiled query (in production:
    backed by a real Elasticsearch query against each variant, injected
    individually — see engine-python/run_pipeline.py), computes the
    robustness score and captures examples that slipped through for the
    CI report / SOC review."""
    detected = 0
    undetected = []
    for v in variants:
        if matches_query(v.event):
            detected += 1
        else:
            undetected.append({"mutation_chain": v.mutation_chain, "event": v.event})

    total = len(variants)
    score = (detected / total) if total else 1.0
    return RobustnessResult(
        rule_id=rule_id,
        variants_tested=total,
        variants_detected=detected,
        score=score,
        undetected_examples=undetected[:5],  # cap report size
    )
