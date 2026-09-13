"""
sigma_translate.py

Translates a Sigma rule into the native query language of whichever SIEM
you actually run — not just Elasticsearch. Detection engineers on Splunk,
Microsoft Sentinel/Defender, CrowdStrike, Grafana Loki, or OpenSearch get
the same rule-authoring workflow and the same CI validation this project
was originally built around for Elasticsearch.

Each backend below is a real, independently-maintained pySigma plugin
(all published on PyPI by the SigmaHQ / AttackIQ projects) — this module
is a thin, uniform CLI/API wrapper around them, not a reimplementation.

WHICH BACKENDS ARE INCLUDED, AND WHY OTHERS AREN'T (yet):
Only backends compatible with the current pySigma core (>=1.0,<2.0) are
wired in here. Two well-known backends are deliberately excluded because
they're currently incompatible with modern pySigma and appear unmaintained:

  - pysigma-backend-qradar   pins pysigma<0.10.0 (last released for a
                             pySigma core two major versions behind current)
  - pysigma-backend-datadog  pins pysigma<0.12.0 (same situation)

Installing either alongside the backends below in the same environment
will hit a real, unresolvable dependency conflict — pip will tell you so
explicitly rather than silently picking a broken combination. If you need
QRadar or Datadog output, translate with an isolated virtualenv pinned to
their required older pySigma version, or check whether either project has
published a compatible release since this was written.

LIMITATION BEING HONEST ABOUT: this module handles TRANSLATION for every
backend below. The rest of the pipeline's *live validation* (attack
simulation + baseline false-positive testing, in run_pipeline.py) only
runs against Elasticsearch/OpenSearch today, because that's what
backend-go/orchestrator provisions ephemeral test infrastructure for.
Translating to Splunk SPL or KQL and confirming it's syntactically valid
is real and immediately useful (catches broken field mappings, invalid
Sigma syntax, etc. before it reaches a human); confirming it actually
*fires correctly* against Splunk/Sentinel/CrowdStrike specifically would
require also building ephemeral test infrastructure for each of those
platforms, which is a natural (and substantial) next contribution — see
docs/ENTERPRISE.md's roadmap section.
"""
from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from typing import Callable

from sigma.collection import SigmaCollection

SUPPORTED_BACKENDS = (
    "elasticsearch", "opensearch", "splunk", "kusto", "crowdstrike", "loki",
)


@dataclass
class BackendResult:
    backend: str
    query_language: str
    output: str  # the translated query, as text (JSON DSL, SPL, KQL, LogQL, etc.)
    live_validation_supported: bool


def _translate_elasticsearch(rule_yaml: str) -> BackendResult:
    from sigma.backends.elasticsearch import LuceneBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(LuceneBackend(), collection, output_format="dsl_lucene")
    return BackendResult("elasticsearch", "Elasticsearch Query DSL", _stringify(results), True)


def _translate_opensearch(rule_yaml: str) -> BackendResult:
    from sigma.backends.opensearch import OpensearchLuceneBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(OpensearchLuceneBackend(), collection, output_format="dsl_lucene")
    return BackendResult("opensearch", "OpenSearch Query DSL", _stringify(results), True)


def _translate_splunk(rule_yaml: str) -> BackendResult:
    from sigma.backends.splunk import SplunkBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(SplunkBackend(), collection, output_format="default")
    return BackendResult("splunk", "Splunk SPL", _stringify(results), False)


def _translate_kusto(rule_yaml: str) -> BackendResult:
    from sigma.backends.kusto import KustoBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(KustoBackend(), collection, output_format="default")
    return BackendResult("kusto", "KQL (Microsoft Sentinel / Defender)", _stringify(results), False)


def _translate_crowdstrike(rule_yaml: str) -> BackendResult:
    from sigma.backends.crowdstrike import LogScaleBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(LogScaleBackend(), collection, output_format="default")
    return BackendResult("crowdstrike", "CrowdStrike LogScale query language", _stringify(results), False)


def _translate_loki(rule_yaml: str) -> BackendResult:
    from sigma.backends.loki import LogQLBackend
    collection = SigmaCollection.from_yaml(rule_yaml)
    results = _convert_get_first(LogQLBackend(), collection, output_format="default")
    return BackendResult("loki", "LogQL (Grafana Loki)", _stringify(results), False)


_TRANSLATORS: dict[str, Callable[[str], BackendResult]] = {
    "elasticsearch": _translate_elasticsearch,
    "opensearch": _translate_opensearch,
    "splunk": _translate_splunk,
    "kusto": _translate_kusto,
    "crowdstrike": _translate_crowdstrike,
    "loki": _translate_loki,
}


def _convert_get_first(backend, collection, output_format: str):
    results = backend.convert(collection, output_format=output_format)
    if not results:
        raise ValueError("backend produced no query for this rule")
    return results[0]


def _stringify(result) -> str:
    if isinstance(result, (dict, list)):
        return json.dumps(result, indent=2)
    return str(result)


def translate(rule_yaml: str, backend: str) -> BackendResult:
    if backend not in _TRANSLATORS:
        raise ValueError(f"unknown backend {backend!r}; supported: {', '.join(SUPPORTED_BACKENDS)}")
    return _TRANSLATORS[backend](rule_yaml)


def translate_all(rule_yaml: str) -> dict[str, BackendResult | str]:
    """Translates to every supported backend, capturing per-backend errors
    individually so one incompatible rule construct on one backend doesn't
    block seeing results for the others."""
    out: dict[str, BackendResult | str] = {}
    for name in SUPPORTED_BACKENDS:
        try:
            out[name] = translate(rule_yaml, name)
        except Exception as e:  # noqa: BLE001 - deliberately broad, see docstring
            out[name] = f"ERROR: {e}"
    return out


def main():
    parser = argparse.ArgumentParser(description="Translate a Sigma rule to the query language of your SIEM")
    parser.add_argument("--rule", required=True)
    parser.add_argument("--backend", choices=list(SUPPORTED_BACKENDS) + ["all"], default="elasticsearch")
    args = parser.parse_args()

    with open(args.rule) as f:
        rule_yaml = f.read()

    if args.backend == "all":
        results = translate_all(rule_yaml)
        for name, res in results.items():
            print(f"\n=== {name} ===")
            if isinstance(res, BackendResult):
                print(f"({res.query_language}, live CI validation: {'yes' if res.live_validation_supported else 'translation-only today'})")
                print(res.output)
            else:
                print(res, file=sys.stderr)
        return

    result = translate(rule_yaml, args.backend)
    print(result.output)


if __name__ == "__main__":
    main()
