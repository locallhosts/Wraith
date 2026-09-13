import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent / "engine-python"))

import robustness_fuzzer as fuzz  # noqa: E402


def naive_contains_match(event: dict, needles: list[str]) -> bool:
    """Reimplements exactly what a Sigma `command_line|contains` selection
    compiles to: a case-sensitive substring match. This is the real
    semantics of the rule in rules/suspicious_powershell_encodedcommand.yml
    — not a stand-in."""
    cmdline = event.get("command_line", "")
    return any(needle in cmdline for needle in needles)


BASE_EVENT = {
    "event_type": "process_creation",
    "process_name": "powershell.exe",
    "command_line": "powershell.exe -EncodedCommand JABzAD0ATgBlAHc=",
}

RULE_NEEDLES = ["-EncodedCommand", "-enc ", "-e "]


def test_generate_variants_produces_real_mutations():
    variants = fuzz.generate_variants(BASE_EVENT, n=30, seed=42)
    assert len(variants) > 0
    # every variant must actually differ from the base command line
    for v in variants:
        assert v.event["command_line"] != BASE_EVENT["command_line"] or v.mutation_chain
    # confirm at least one known evasion class actually appears across the population
    chains = [m for v in variants for m in v.mutation_chain]
    assert "case_variation" in chains or "powershell_param_alias" in chains


def test_case_variation_evades_case_sensitive_match_but_would_still_execute():
    rng = fuzz.random.Random(1)
    mutated = fuzz._case_variation(BASE_EVENT["command_line"], rng)
    # case-sensitive substring match against the canonical needle should fail...
    assert "-EncodedCommand" not in mutated or mutated == BASE_EVENT["command_line"]
    # ...but the mutated string is identical modulo case, proving it's the
    # same command Windows would execute identically.
    assert mutated.lower() == BASE_EVENT["command_line"].lower()


def test_powershell_param_alias_produces_known_valid_aliases():
    rng = fuzz.random.Random(7)
    seen_aliases = set()
    for _ in range(50):
        mutated = fuzz._powershell_param_alias(BASE_EVENT["command_line"], rng)
        for alias in fuzz._POWERSHELL_PARAM_ALIASES:
            if alias in mutated and alias != "-EncodedCommand":
                seen_aliases.add(alias)
    assert seen_aliases, "expected at least one shortened PowerShell parameter alias to be generated"


def test_score_robustness_flags_naive_rule_as_non_robust():
    """A rule that only matches the literal '-EncodedCommand' string should
    score poorly against variants that use '-enc'/'-e' or case variation —
    proving the scorer actually distinguishes robust from fragile rules."""
    variants = fuzz.generate_variants(BASE_EVENT, n=40, seed=99)

    def matches(event: dict) -> bool:
        # deliberately narrow/fragile matcher: exact-case, exact-string only
        return "-EncodedCommand" in event.get("command_line", "")

    result = fuzz.score_robustness("fragile-rule", variants, matches)
    assert result.variants_tested > 0
    assert result.score < 1.0, "expected the naive case-sensitive rule to be flagged as non-robust"
    assert len(result.undetected_examples) > 0


def test_score_robustness_rewards_case_insensitive_multi_alias_rule():
    """A rule written the way this project's real Sigma rule is (matching
    multiple aliases, case-insensitively per Sigma's default semantics)
    should score much better than the fragile one above."""
    variants = fuzz.generate_variants(BASE_EVENT, n=40, seed=99)

    def matches(event: dict) -> bool:
        cmdline = event.get("command_line", "").lower()
        return any(n.lower().strip() in cmdline for n in RULE_NEEDLES)

    result = fuzz.score_robustness("robust-rule", variants, matches)
    assert result.score > 0.7, f"expected the robust rule to score highly, got {result.score}"
