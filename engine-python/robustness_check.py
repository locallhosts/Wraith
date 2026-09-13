"""
robustness_check.py

Bridges robustness_fuzzer.py (pure mutation logic, already unit-tested) to
a real Elasticsearch instance: indexes each adversarial variant
individually, then re-runs the rule's actual compiled query (the same
query_dsl produced by sigma_to_es.py and used by validate.py) scoped to
that single document, to determine whether the rule still catches it.

This is what turns "generate some mutated strings" into "prove the rule
does or doesn't still fire on them" — the whole point of the feature.
"""
from __future__ import annotations

import argparse
import copy
import json
import uuid
from typing import Callable

from elasticsearch import Elasticsearch

import robustness_fuzzer as fuzz


def make_es_matcher(es: Elasticsearch, index: str, query_dsl: dict) -> Callable[[dict], bool]:
    """Returns a predicate that indexes a single event and checks whether
    the rule's real query_dsl matches it — i.e. ground truth from the
    actual SIEM query engine, not a Python reimplementation of Sigma
    semantics that could drift from what Elasticsearch actually does."""

    def matches(event: dict) -> bool:
        doc_id = str(uuid.uuid4())
        es.index(index=index, id=doc_id, document=event, refresh=True)

        scoped_query = {
            "query": {
                "bool": {
                    "filter": [{"term": {"_id": doc_id}}],
                    "must": [query_dsl.get("query", {"match_all": {}})],
                }
            }
        }
        result = es.count(index=index, body=scoped_query, ignore_unavailable=True)
        return result["count"] > 0

    return matches


def run_robustness_check(es_addr: str, run_id: str, rule_id: str, query_dsl: dict,
                          base_event: dict, n_variants: int = 40) -> dict:
    es = Elasticsearch(es_addr)
    index = f"wraith-attack-fuzz-{run_id}"

    variants = fuzz.generate_variants(base_event, n=n_variants, seed=hash(run_id) % (2**31))
    if not variants:
        return {
            "rule_id": rule_id,
            "variants_tested": 0,
            "variants_detected": 0,
            "score": 1.0,
            "note": "no mutable command_line field found on the base event; robustness check skipped",
        }

    matcher = make_es_matcher(es, index, query_dsl)
    result = fuzz.score_robustness(rule_id, variants, matcher)

    return {
        "rule_id": result.rule_id,
        "variants_tested": result.variants_tested,
        "variants_detected": result.variants_detected,
        "score": result.score,
        "undetected_examples": result.undetected_examples,
    }


def main():
    parser = argparse.ArgumentParser(description="Run adversarial evasion robustness testing against real Elasticsearch")
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--rule-id", required=True)
    parser.add_argument("--query-dsl-file", required=True)
    parser.add_argument("--base-event-file", required=True, help="JSON file with the base attack event to mutate")
    parser.add_argument("--n-variants", type=int, default=40)
    args = parser.parse_args()

    with open(args.query_dsl_file) as f:
        query_dsl = json.load(f)
    with open(args.base_event_file) as f:
        base_event = json.load(f)

    result = run_robustness_check(args.es_addr, args.run_id, args.rule_id, query_dsl, base_event, args.n_variants)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
