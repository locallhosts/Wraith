"""Explainable Detection Quality Score (0-100).

Only completed stages contribute to the denominator. This avoids penalizing a
rule merely because an optional validation stage was not requested.
"""
from __future__ import annotations
import argparse
import json
from pathlib import Path

WEIGHTS = {"behavioral": 40.0, "false_positive": 20.0, "robustness": 20.0, "mutation": 20.0}


def score(report: dict) -> dict:
    stages = report.get("stages", {})
    v = stages.get("validate", {})
    r = stages.get("robustness", {})
    m = stages.get("mutation_testing", {})
    parts: dict[str, float] = {}
    if "passed" in v:
        parts["behavioral"] = WEIGHTS["behavioral"] if v.get("passed") else 0.0
        parts["false_positive"] = (WEIGHTS["false_positive"] if v.get("baseline_hit_count", 1) == 0
                                    else max(0.0, WEIGHTS["false_positive"] * (1 - v.get("false_positive_rate", 1))))
    if isinstance(r.get("score"), (int, float)):
        parts["robustness"] = WEIGHTS["robustness"] * float(r["score"])
    if isinstance(m.get("mutation_score"), (int, float)):
        parts["mutation"] = WEIGHTS["mutation"] * float(m["mutation_score"])
    available = sum(WEIGHTS[k] for k in parts)
    raw = sum(parts.values())
    total = round((raw / available) * 100, 2) if available else None
    return {"score": total, "max": 100, "available_weight": available,
            "components": parts,
            "rating": ("strong" if total is not None and total >= 85 else
                       "acceptable" if total is not None and total >= 70 else
                       "needs-improvement" if total is not None else "not-rated")}


def main():
    p = argparse.ArgumentParser(); p.add_argument("--report", required=True); p.add_argument("--out", required=True)
    a = p.parse_args(); result = score(json.loads(Path(a.report).read_text())); Path(a.out).write_text(json.dumps(result, indent=2)); print(json.dumps(result, indent=2))

if __name__ == "__main__": main()
