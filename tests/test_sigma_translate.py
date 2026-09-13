import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent / "engine-python"))

import sigma_translate  # noqa: E402

RULE_PATH = Path(__file__).parent.parent / "rules" / "suspicious_powershell_encodedcommand.yml"


def _rule_yaml() -> str:
    return RULE_PATH.read_text()


def test_all_supported_backends_produce_nonempty_output():
    """Every backend in SUPPORTED_BACKENDS must actually produce a query
    for the real shipped sample rule — this is the regression test that
    would have caught the CrowdStrike import-name bug immediately."""
    results = sigma_translate.translate_all(_rule_yaml())
    assert set(results.keys()) == set(sigma_translate.SUPPORTED_BACKENDS)
    for name, result in results.items():
        assert isinstance(result, sigma_translate.BackendResult), (
            f"backend {name!r} failed: {result}"
        )
        assert result.output.strip(), f"backend {name!r} produced empty output"


def test_elasticsearch_backend_produces_valid_json_dsl():
    result = sigma_translate.translate(_rule_yaml(), "elasticsearch")
    assert result.live_validation_supported is True
    import json
    parsed = json.loads(result.output)
    assert "query" in parsed


def test_splunk_backend_references_powershell():
    result = sigma_translate.translate(_rule_yaml(), "splunk")
    assert "powershell.exe" in result.output
    assert result.live_validation_supported is False


def test_kusto_backend_references_encoded_command():
    result = sigma_translate.translate(_rule_yaml(), "kusto")
    assert "EncodedCommand" in result.output


def test_unknown_backend_raises():
    import pytest
    with pytest.raises(ValueError):
        sigma_translate.translate(_rule_yaml(), "not-a-real-siem")
