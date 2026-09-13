"""
Unit tests for the pure-logic parts of attack_simulator.py — technique
extraction and attack-chain construction — that don't require live
Elasticsearch/Neo4j services. Run with: pytest tests/
"""
import json
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent / "engine-python"))

import attack_simulator  # noqa: E402


def test_extract_techniques_from_rule():
    rule_yaml = """
title: Test rule
id: 11111111-1111-1111-1111-111111111111
level: high
tags:
  - attack.execution
  - attack.t1059.001
logsource:
  category: process_creation
detection:
  selection:
    x: y
  condition: selection
"""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".yml", delete=False) as f:
        f.write(rule_yaml)
        path = f.name

    techniques = attack_simulator.extract_techniques_from_rule(path)
    assert techniques == ["T1059.001"]


def test_build_attack_chain_prefers_tagged_technique():
    mappings = json.loads(
        (Path(__file__).parent.parent / "engine-python" / "mitre_mappings.json").read_text()
    )
    chain = attack_simulator.build_attack_chain(["T1059.001"], mappings)

    assert "T1059.001" in chain
    # chain should be ordered by tactic per TACTIC_ORDER
    tactics_in_chain = [mappings[t]["tactic"] for t in chain]
    assert tactics_in_chain == sorted(
        tactics_in_chain, key=lambda t: attack_simulator.TACTIC_ORDER.index(t)
    )


def test_build_attack_chain_falls_back_when_unknown_technique():
    mappings = json.loads(
        (Path(__file__).parent.parent / "engine-python" / "mitre_mappings.json").read_text()
    )
    chain = attack_simulator.build_attack_chain(["T9999.999"], mappings)
    assert len(chain) > 0
    for t in chain:
        assert t in mappings
