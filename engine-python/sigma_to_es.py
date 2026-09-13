"""
sigma_to_es.py

Translates a Sigma rule file into an Elasticsearch Query DSL body using
the real pySigma library + its Elasticsearch backend (SigmaHQ project),
not a hand-rolled parser. This is what the Go validator runs against the
attack/baseline indices.
"""
from __future__ import annotations

import argparse
import json

from sigma.collection import SigmaCollection
from sigma.backends.elasticsearch import LuceneBackend


def translate(rule_path: str) -> dict:
    with open(rule_path) as f:
        rule_yaml = f.read()

    collection = SigmaCollection.from_yaml(rule_yaml)
    backend = LuceneBackend()

    # queries() returns Lucene query strings; for direct Query DSL use
    # the dsl_lucene output format supported by the backend.
    results = backend.convert(collection, output_format="dsl_lucene")
    if not results:
        raise ValueError("pySigma produced no query for this rule")

    # Each result is already a JSON-serializable ES query dict.
    query_dsl = results[0]
    if isinstance(query_dsl, str):
        query_dsl = json.loads(query_dsl)
    return query_dsl


def main():
    parser = argparse.ArgumentParser(description="Translate a Sigma rule into an Elasticsearch Query DSL body")
    parser.add_argument("--rule", required=True)
    parser.add_argument("--out", help="write JSON to this path instead of stdout")
    args = parser.parse_args()

    dsl = translate(args.rule)
    output = json.dumps(dsl, indent=2)
    if args.out:
        with open(args.out, "w") as f:
            f.write(output)
    else:
        print(output)


if __name__ == "__main__":
    main()
