"""
validate.py

Queries the real Elasticsearch instance to determine whether a rule fired
on the injected attack telemetry and whether it produced any false
positives against the baseline corpus. This mirrors backend-go/validator
so the Python pipeline can gate the GitHub Actions job (exit code) without
waiting on a round trip to the Go API in local/dev runs; the Go backend
performs the same check independently for the production API/dashboard.
"""
from __future__ import annotations

import argparse
import json

from elasticsearch import Elasticsearch


def validate(es: Elasticsearch, run_id: str, query_dsl: dict) -> dict:
    attack_index = f"wraith-attack-{run_id}"
    baseline_index = f"wraith-baseline-{run_id}"

    attack_count = es.count(index=attack_index, body=query_dsl, ignore_unavailable=True)["count"]
    baseline_count = es.count(index=baseline_index, body=query_dsl, ignore_unavailable=True)["count"]
    baseline_total = es.count(index=baseline_index, body={"query": {"match_all": {}}}, ignore_unavailable=True)["count"]

    fp_rate = (baseline_count / baseline_total) if baseline_total else 0.0

    if attack_count == 0:
        passed, reason = False, "rule did not fire on injected attack telemetry (false negative)"
    elif baseline_count > 0:
        passed, reason = False, f"rule fired {baseline_count} time(s) on benign baseline traffic (false positive)"
    else:
        passed, reason = True, "rule fired on attack telemetry with zero false positives on baseline"

    return {
        "run_id": run_id,
        "fired_on_attack": attack_count > 0,
        "attack_hit_count": attack_count,
        "fired_on_baseline": baseline_count > 0,
        "baseline_hit_count": baseline_count,
        "baseline_docs_scanned": baseline_total,
        "false_positive_rate": fp_rate,
        "passed": passed,
        "reason": reason,
    }


def main():
    parser = argparse.ArgumentParser(description="Validate rule fire/false-positive behavior against real ES data")
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--query-dsl-file", required=True)
    args = parser.parse_args()

    with open(args.query_dsl_file) as f:
        query_dsl = json.load(f)

    es = Elasticsearch(args.es_addr)
    result = validate(es, args.run_id, query_dsl)
    print(json.dumps(result, indent=2))

    if not result["passed"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
