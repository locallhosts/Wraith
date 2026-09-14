"""Mutation testing for Sigma detections.

Generates deliberately weakened variants of a rule so Wraith can measure
whether its validation corpus is strong enough to notice broken detections.
The module is intentionally independent of Elasticsearch: it produces
mutants and a machine-readable manifest; callers can feed each mutant into
whatever validation backend they already use.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Any
import yaml


@dataclass
class Mutant:
    mutation_id: str
    operator: str
    description: str
    rule: dict[str, Any]


def _id(rule: dict[str, Any], operator: str, n: int) -> str:
    seed = json.dumps(rule, sort_keys=True, separators=(",", ":"))
    return hashlib.sha256(f"{seed}:{operator}:{n}".encode()).hexdigest()[:12]


def generate_mutants(rule: dict[str, Any]) -> list[Mutant]:
    detection = rule.get("detection")
    if not isinstance(detection, dict):
        return []
    out: list[Mutant] = []

    # Remove one selection field: a common accidental broadening.
    for name, selection in detection.items():
        if name == "condition" or not isinstance(selection, dict):
            continue
        for field in list(selection):
            m = copy.deepcopy(rule)
            del m["detection"][name][field]
            out.append(Mutant(_id(rule, "drop-field", len(out)), "drop-field",
                              f"Remove detection field {name}.{field}", m))

    # Remove one condition from a list-valued selection.
    for name, selection in detection.items():
        if name == "condition" or not isinstance(selection, dict):
            continue
        for field, value in selection.items():
            if isinstance(value, list) and len(value) > 1:
                for idx in range(len(value)):
                    m = copy.deepcopy(rule)
                    del m["detection"][name][field][idx]
                    out.append(Mutant(_id(rule, "drop-value", len(out)), "drop-value",
                                      f"Remove value {idx} from {name}.{field}", m))

    # Weaken exact process names to a case variation. This tests case handling
    # without inventing a new field or changing the rule's semantics entirely.
    for name, selection in detection.items():
        if name == "condition" or not isinstance(selection, dict):
            continue
        for field, value in selection.items():
            if isinstance(value, str) and value and value.lower() != value.upper():
                m = copy.deepcopy(rule)
                m["detection"][name][field] = value.swapcase()
                out.append(Mutant(_id(rule, "case-variation", len(out)), "case-variation",
                                  f"Change case of {name}.{field}", m))

    # Replace a list with its first value: catches rules that accidentally lose
    # aliases/modifiers during editing.
    for name, selection in detection.items():
        if name == "condition" or not isinstance(selection, dict):
            continue
        for field, value in selection.items():
            if isinstance(value, list) and len(value) > 1:
                m = copy.deepcopy(rule)
                m["detection"][name][field] = [value[0]]
                out.append(Mutant(_id(rule, "single-value", len(out)), "single-value",
                                  f"Keep only the first value in {name}.{field}", m))
    return out


def mutation_score(results: list[dict[str, Any]]) -> dict[str, Any]:
    tested = len(results)
    killed = sum(1 for r in results if r.get("killed"))
    score = killed / tested if tested else None
    return {"mutants_tested": tested, "mutants_killed": killed, "mutation_score": score,
            "status": "ok" if tested else "no_mutants"}


def main() -> None:
    p = argparse.ArgumentParser(description="Generate and score Sigma detection mutants")
    p.add_argument("--rule", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--results", help="JSON array of {mutation_id,killed,...} to score")
    args = p.parse_args()
    rule = yaml.safe_load(Path(args.rule).read_text())
    mutants = generate_mutants(rule)
    payload = {"rule_id": rule.get("id"), "mutants": [asdict(m) for m in mutants]}
    if args.results:
        payload["score"] = mutation_score(json.loads(Path(args.results).read_text()))
    Path(args.out).write_text(json.dumps(payload, indent=2))


if __name__ == "__main__":
    main()
