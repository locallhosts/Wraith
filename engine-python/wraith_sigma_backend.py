from __future__ import annotations

from sigma.backends.elasticsearch import LuceneBackend


class WraithElasticsearchBackend(LuceneBackend):
    """
    Wraith's Elasticsearch adapter for pySigma.

    Wraith synthetic telemetry stores string fields as text fields with
    `.keyword` multi-fields. The stock pySigma Lucene backend emits
    query_string expressions against the analyzed text field, which is
    unsuitable for exact/wildcard validation against Wraith's schema.

    This adapter keeps pySigma's Sigma parsing and condition handling,
    while targeting the keyword subfields used by Wraith.
    """

    contains_expression = "{field}{backend.eq_token}*{value}*"
    startswith_expression = "{field}{backend.eq_token}{value}*"
    endswith_expression = "{field}{backend.eq_token}*{value}"
    wildcard_match_expression = "{field}{backend.eq_token}{value}"

    def escape_and_quote_field(self, field: str) -> str:
        return f"{field}.keyword"