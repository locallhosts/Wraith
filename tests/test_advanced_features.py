import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[1] / "engine-python"))

import attack_navigator
import compliance_map
import correlation
import mutation_testing
import quality_score
import rule_diff
import schema_drift

RULE = {
    "id": "demo",
    "tags": ["attack.execution", "attack.t1059.001"],
    "detection": {
        "selection": {
            "process_name": "powershell.exe",
            "command_line|contains": ["-EncodedCommand", "-enc ", "-e "]
        },
        "condition": "selection"
    }
}


def test_mutants_are_deterministic_and_nonempty():
    a = mutation_testing.generate_mutants(RULE)
    b = mutation_testing.generate_mutants(RULE)
    assert a and [x.mutation_id for x in a] == [x.mutation_id for x in b]
    assert any(x.operator == "drop-field" for x in a)
    assert any(x.operator == "drop-value" for x in a)


def test_mutation_score():
    result = mutation_testing.mutation_score([{"killed": True}, {"killed": False}])
    assert result["mutation_score"] == 0.5


def test_rule_diff_changes_hash():
    changed = dict(RULE)
    changed["level"] = "high"
    result = rule_diff.diff(RULE, changed)
    assert result["changed"]
    assert result["old_sha256"] != result["new_sha256"]


def test_navigator_extracts_techniques():
    layer = attack_navigator.export_layer([RULE])
    assert layer["techniques"] == [{"techniqueID": "T1059.001", "score": 1, "comment": "Detected by one or more Wraith Sigma rules"}]


def test_schema_drift():
    result = schema_drift.compare(RULE, {"fields": ["process_name", "command_line"]})
    assert not result["drift"]
    result = schema_drift.compare(RULE, {"fields": ["process_name"]})
    assert result["drift"] and result["missing_fields"] == ["command_line"]


def test_correlation():
    manifest = {"name": "execution-chain", "rules": ["a", "b"], "order": ["a", "b"]}
    verdicts = [{"rule_id": "a", "passed": True, "detected_at": "2026-01-01T00:00:00Z"},
                {"rule_id": "b", "passed": True, "detected_at": "2026-01-01T00:01:00Z"}]
    assert correlation.evaluate(manifest, verdicts)["passed"]


def test_quality_score_is_explainable():
    report = {"stages": {"validate": {"passed": True, "baseline_hit_count": 0, "false_positive_rate": 0},
                          "robustness": {"score": 0.8},
                          "mutation_testing": {"mutation_score": 0.75}}}
    result = quality_score.score(report)
    assert result["score"] == 91.0
    assert set(result["components"]) == {"behavioral", "false_positive", "robustness", "mutation"}


def test_compliance_mapping_is_user_owned():
    result = compliance_map.map_rule(RULE, {"T1059.001": {"controls": ["ORG-01"], "source": "internal"}})
    assert result["mappings"][0]["controls"] == ["ORG-01"]
