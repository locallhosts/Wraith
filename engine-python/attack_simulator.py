"""
attack_simulator.py

Builds a multi-stage synthetic attack path in Neo4j (Initial Access ->
Execution -> Persistence -> ... -> Impact) based on the MITRE ATT&CK
technique(s) tagged on the Sigma rule under test, then injects the
corresponding telemetry into Elasticsearch on a realistic timeline so a
detection rule is exercised the way it would be against a real intrusion,
not just a single isolated log line.

Both the Neo4j write and the Elasticsearch bulk index are real calls
against the ephemeral containers provisioned by the Go orchestrator.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
import uuid
from pathlib import Path
from typing import List

import yaml
from elasticsearch import Elasticsearch, helpers
from neo4j import GraphDatabase

MITRE_MAPPINGS_PATH = Path(__file__).parent / "mitre_mappings.json"

# Canonical kill-chain order used to lay out the attack path even if the
# rule only tags a single mid-chain technique — we still build plausible
# preceding/following stages so multi-stage detections have something
# realistic to correlate against.
TACTIC_ORDER = [
    "initial-access", "execution", "persistence", "privilege-escalation",
    "defense-evasion", "credential-access", "discovery", "lateral-movement",
    "collection", "command-and-control", "exfiltration", "impact",
]


def load_mappings() -> dict:
    with open(MITRE_MAPPINGS_PATH) as f:
        return json.load(f)


def extract_techniques_from_rule(rule_path: str) -> List[str]:
    """Pull attack.tXXXX(.YYY) tags out of a Sigma rule's `tags:` field."""
    with open(rule_path) as f:
        rule = yaml.safe_load(f)
    tags = rule.get("tags", []) or []
    techniques = []
    for t in tags:
        m = re.match(r"attack\.(t\d{4}(?:\.\d{3})?)", t, re.IGNORECASE)
        if m:
            techniques.append(m.group(1).upper())
    return techniques


def build_attack_chain(techniques: List[str], mappings: dict) -> List[str]:
    """
    Given the technique(s) actually tagged on the rule, select a plausible
    surrounding chain from mitre_mappings.json ordered by tactic, so the
    rule is tested inside a realistic multi-stage narrative rather than in
    isolation.
    """
    known = [t for t in techniques if t in mappings]
    if not known:
        # Fall back to a generic but complete kill chain if the rule's tags
        # don't match our mapping table (still real data, just not tailored).
        known = list(mappings.keys())

    tactics_present = {mappings[t]["tactic"] for t in known}
    chain = []
    for tactic in TACTIC_ORDER:
        candidates = [t for t, m in mappings.items() if m["tactic"] == tactic]
        if not candidates:
            continue
        # Prefer a technique that was actually tagged on the rule for its tactic.
        tagged_for_tactic = [t for t in known if mappings[t]["tactic"] == tactic]
        chain.append(tagged_for_tactic[0] if tagged_for_tactic else candidates[0])
    return chain


def write_graph(driver, run_id: str, chain: List[str], mappings: dict, user: str, host: str) -> str:
    """Writes the attack path into Neo4j as a real (:Stage)-[:NEXT]->(:Stage)
    chain, linked to a synthetic (:User) and (:Host) node, and returns the
    Neo4j element id of the root stage for reference in reports."""
    with driver.session() as session:
        session.run("MATCH (n {run_id: $run_id}) DETACH DELETE n", run_id=run_id)

        session.run(
            """
            MERGE (u:User {name: $user, run_id: $run_id})
            MERGE (h:Host {name: $host, run_id: $run_id})
            MERGE (u)-[:LOGGED_INTO]->(h)
            """,
            user=user, host=host, run_id=run_id,
        )

        prev_id = None
        for i, technique in enumerate(chain):
            meta = mappings[technique]
            result = session.run(
                """
                CREATE (s:Stage {
                    run_id: $run_id,
                    seq: $seq,
                    technique_id: $tid,
                    technique_name: $tname,
                    tactic: $tactic
                })
                WITH s
                MATCH (h:Host {name: $host, run_id: $run_id})
                MERGE (s)-[:OBSERVED_ON]->(h)
                RETURN elementId(s) AS id
                """,
                run_id=run_id, seq=i, tid=technique, tname=meta["name"],
                tactic=meta["tactic"], host=host,
            )
            stage_id = result.single()["id"]
            if prev_id:
                session.run(
                    "MATCH (a:Stage) WHERE elementId(a) = $prev "
                    "MATCH (b:Stage) WHERE elementId(b) = $curr "
                    "MERGE (a)-[:NEXT]->(b)",
                    prev=prev_id, curr=stage_id,
                )
            prev_id = stage_id
        return prev_id


def inject_telemetry(es: Elasticsearch, run_id: str, chain: List[str], mappings: dict, user: str, host: str) -> int:
    """Turns each stage in the attack chain into one or more Elasticsearch
    documents, spaced a few minutes apart to mimic a real intrusion
    timeline, and bulk-indexes them into wraith-attack-<run_id>."""
    index_name = f"wraith-attack-{run_id}"
    start = dt.datetime.utcnow() - dt.timedelta(hours=2)

    def actions():
        for i, technique in enumerate(chain):
            meta = mappings[technique]
            ts = start + dt.timedelta(minutes=i * 7)
            doc = {
                "@timestamp": ts.isoformat(),
                "user": user,
                "host": host,
                "mitre_technique": technique,
                "mitre_tactic": meta["tactic"],
                "wraith_synthetic": True,
                "wraith_label": "attack",
                "wraith_run_id": run_id,
                **meta["event"],
            }
            yield {"_index": index_name, "_source": doc}

    success, errors = helpers.bulk(es, actions(), stats_only=True, raise_on_error=False)
    if errors:
        print(f"[attack_simulator] WARNING: {errors} attack docs failed to index", file=sys.stderr)
    es.indices.refresh(index=index_name)
    return success


def simulate(rule_path: str, es_addr: str, neo4j_addr: str, run_id: str,
             neo4j_user: str = "neo4j", neo4j_pass: str = "wraith-test-pw") -> dict:
    mappings = load_mappings()
    techniques = extract_techniques_from_rule(rule_path)
    chain = build_attack_chain(techniques, mappings)

    user = f"jdoe-{uuid.uuid4().hex[:4]}"
    host = f"WKS-ATK-{uuid.uuid4().hex[:4]}"

    driver = GraphDatabase.driver(neo4j_addr, auth=(neo4j_user, neo4j_pass))
    try:
        write_graph(driver, run_id, chain, mappings, user, host)
    finally:
        driver.close()

    es = Elasticsearch(es_addr)
    indexed = inject_telemetry(es, run_id, chain, mappings, user, host)

    return {
        "run_id": run_id,
        "rule_tagged_techniques": techniques,
        "simulated_chain": chain,
        "simulated_user": user,
        "simulated_host": host,
        "events_indexed": indexed,
    }


def main():
    parser = argparse.ArgumentParser(description="Simulate a graph-based multi-stage attack for rule testing")
    parser.add_argument("--rule", required=True)
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--neo4j-addr", required=True)
    parser.add_argument("--run-id", required=True)
    args = parser.parse_args()

    result = simulate(args.rule, args.es_addr, args.neo4j_addr, args.run_id)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
