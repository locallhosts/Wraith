"""
sigma_to_es.py

Translate Sigma rules into native Elasticsearch Query DSL for Wraith's
synthetic validation indices.

pySigma is still responsible for parsing Sigma and applying Sigma semantics.
Wraith renders the resulting condition tree into Elasticsearch DSL because
the installed pySigma Elasticsearch Lucene backend emits query_string syntax,
while Wraith's synthetic schema uses text fields with .keyword multi-fields.
"""

from __future__ import annotations

import argparse
import json
from typing import Any

from sigma.collection import SigmaCollection
from sigma.conditions import (
    ConditionAND,
    ConditionNOT,
    ConditionOR,
    SigmaCondition,
)
from sigma.modifiers import SigmaContainsModifier
from sigma.types import (
    SigmaBool,
    SigmaNumber,
    SigmaString,
)


def _keyword_field(field: str) -> str:
    """Map Wraith string fields to their Elasticsearch keyword multi-field."""
    return f"{field}.keyword"


def _plain_sigma_string(value: SigmaString) -> str:
    """
    Convert a SigmaString to its textual representation.

    This intentionally relies on pySigma's parsed SigmaString rather than
    parsing the original YAML ourselves.
    """
    return str(value)


def _render_detection_item(item: Any) -> dict[str, Any]:
    """Render one SigmaDetectionItem into native Elasticsearch DSL."""

    field = _keyword_field(item.field)
    values = item.value

    contains = SigmaContainsModifier in item.modifiers

    rendered: list[dict[str, Any]] = []

    for value in values:
        if isinstance(value, SigmaString):
            text = _plain_sigma_string(value)

            if contains:
                # SigmaContainsModifier represents a substring match.
                # Native ES wildcard operates on the keyword field.
                rendered.append(
                    {
                        "wildcard": {
                            field: text,
                        }
                    }
                )
            else:
                rendered.append(
                    {
                        "term": {
                            field: text,
                        }
                    }
                )

        elif isinstance(value, SigmaBool):
            rendered.append(
                {
                    "term": {
                        item.field: value.boolean,
                    }
                }
            )

        elif isinstance(value, SigmaNumber):
            rendered.append(
                {
                    "term": {
                        item.field: value.number,
                    }
                }
            )

        else:
            raise NotImplementedError(
                f"Unsupported Sigma value type for field "
                f"{item.field!r}: {type(value).__name__}"
            )

    if not rendered:
        raise ValueError(f"Sigma detection item {item.field!r} has no values")

    if len(rendered) == 1:
        query = rendered[0]
    else:
        query = {
            "bool": {
                "should": rendered,
                "minimum_should_match": 1,
            }
        }

    if getattr(item, "negated", False):
        return {"bool": {"must_not": [query]}}

    return query


def _render_detection(detection: Any) -> dict[str, Any]:
    """Render a SigmaDetection whose items are implicitly AND-linked."""

    queries = [
        _render_detection_item(item)
        for item in detection.detection_items
    ]

    if not queries:
        raise ValueError("Sigma detection contains no detection items")

    if len(queries) == 1:
        return queries[0]

    return {
        "bool": {
            "must": queries,
        }
    }


def _render_condition(condition: Any) -> dict[str, Any]:
    """Render a parsed Sigma condition tree."""

    if isinstance(condition, ConditionAND):
        return {
            "bool": {
                "must": [
                    _render_condition(child)
                    for child in condition.args
                ]
            }
        }

    if isinstance(condition, ConditionOR):
        return {
            "bool": {
                "should": [
                    _render_condition(child)
                    for child in condition.args
                ],
                "minimum_should_match": 1,
            }
        }

    if isinstance(condition, ConditionNOT):
        return {
            "bool": {
                "must_not": [
                    _render_condition(child)
                    for child in condition.args
                ]
            }
        }

    if isinstance(condition, SigmaCondition):
        condition_name = condition.condition

        if isinstance(condition_name, str):
            detection_name = condition_name
        elif isinstance(condition_name, (list, tuple)) and len(condition_name) == 1:
            detection_name = condition_name[0]
        else:
            raise NotImplementedError(
                f"Unsupported SigmaCondition reference: {condition_name!r}"
            )

        detection = condition.detections.detections[detection_name]
        return _render_detection(detection)

    raise NotImplementedError(
        f"Unsupported Sigma condition type: {type(condition).__name__}"
    )


def translate(rule_path: str) -> dict[str, Any]:
    """
    Parse a Sigma rule and return native Elasticsearch Query DSL.
    """

    with open(rule_path, encoding="utf-8") as f:
        rule_yaml = f.read()

    collection = SigmaCollection.from_yaml(rule_yaml)

    if not collection.rules:
        raise ValueError("Sigma rule collection contains no rules")

    sigma_rule = collection.rules[0]

    conditions = sigma_rule.detection.parsed_condition

    if not conditions:
        raise ValueError("Sigma rule contains no parsed conditions")

    rendered = [
        _render_condition(condition)
        for condition in conditions
    ]

    if len(rendered) == 1:
        query = rendered[0]
    else:
        query = {
            "bool": {
                "must": rendered,
            }
        }

    return {
        "query": query,
    }


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Translate a Sigma rule into Wraith Elasticsearch DSL"
    )
    parser.add_argument("rule")
    args = parser.parse_args()

    result = translate(args.rule)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()