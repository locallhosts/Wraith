"""
attack_simulator.py

Builds a multi-stage synthetic attack path in Neo4j and injects
target-OS-specific telemetry into Elasticsearch.

WRAITH separates:

    Host OS:
        The operating system running WRAITH.

    Detection target:
        The operating system whose telemetry is being simulated.

Supported detection targets:

    windows
    linux
    macos

The host OS does not determine the detection target.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
import uuid
from pathlib import Path
from typing import List, Optional

import yaml
from elasticsearch import Elasticsearch, helpers
from neo4j import GraphDatabase


MITRE_MAPPINGS_PATH = Path(__file__).parent / "mitre_mappings.json"

SUPPORTED_TARGET_OS = {
    "windows",
    "linux",
    "macos",
}

TACTIC_ORDER = [
    "initial-access",
    "execution",
    "persistence",
    "privilege-escalation",
    "defense-evasion",
    "credential-access",
    "discovery",
    "lateral-movement",
    "collection",
    "command-and-control",
    "exfiltration",
    "impact",
]


def load_mappings() -> dict:
    with open(MITRE_MAPPINGS_PATH) as f:
        return json.load(f)


def normalize_target_os(target_os: Optional[str]) -> Optional[str]:
    """
    Normalize an explicitly supplied target OS.

    Returns:
        windows
        linux
        macos
        None
    """

    if target_os is None:
        return None

    value = target_os.strip().lower()

    aliases = {
        "win": "windows",
        "windows": "windows",
        "linux": "linux",
        "mac": "macos",
        "macos": "macos",
        "mac_os": "macos",
        "darwin": "macos",
    }

    normalized = aliases.get(value)

    if normalized is None:
        raise ValueError(
            f"Unsupported target OS '{target_os}'. "
            f"Supported values: {', '.join(sorted(SUPPORTED_TARGET_OS))}"
        )

    return normalized


def target_os_from_rule(rule_path: str) -> Optional[str]:
    """
    Infer the detection target from Sigma logsource.product.
    """

    with open(rule_path) as f:
        rule = yaml.safe_load(f)

    logsource = rule.get("logsource", {}) or {}
    product = str(logsource.get("product", "")).strip().lower()

    product_map = {
        "windows": "windows",
        "linux": "linux",
        "macos": "macos",
        "mac_os": "macos",
        "darwin": "macos",
    }

    return product_map.get(product)


def resolve_target_os(
    rule_path: str,
    target_os: Optional[str],
) -> tuple[Optional[str], str]:
    """
    Resolve the detection target.

    Explicit CLI target takes precedence over Sigma logsource.
    """

    explicit = normalize_target_os(target_os)

    if explicit:
        return explicit, "cli"

    inferred = target_os_from_rule(rule_path)

    if inferred:
        return inferred, "sigma-logsource"

    return None, "unspecified"


def extract_techniques_from_rule(rule_path: str) -> List[str]:
    """
    Pull attack.tXXXX(.YYY) tags out of a Sigma rule's tags field.
    """

    with open(rule_path) as f:
        rule = yaml.safe_load(f)

    tags = rule.get("tags", []) or []

    techniques = []

    for tag in tags:
        match = re.match(
            r"attack\.(t\d{4}(?:\.\d{3})?)",
            tag,
            re.IGNORECASE,
        )

        if match:
            techniques.append(match.group(1).upper())

    return techniques


def mapping_supports_platform(
    mapping: dict,
    target_os: Optional[str],
) -> bool:
    """
    Determine whether a telemetry mapping supports the requested OS.

    Older mappings without a platforms field remain compatible and are
    treated as platform-agnostic.
    """

    if target_os is None:
        return True

    platforms = mapping.get("platforms")

    if platforms is None:
        return True

    normalized = {
        normalize_target_os(platform)
        for platform in platforms
    }

    return target_os in normalized


def build_attack_chain(
    techniques: List[str],
    mappings: dict,
    target_os: Optional[str] = None,
) -> List[str]:
    """
    Given the technique(s) tagged on the rule, select a plausible
    surrounding chain from mitre_mappings.json ordered by tactic.

    Only telemetry mappings compatible with target_os are selected.
    """

    target_os = normalize_target_os(target_os)

    available = {
        technique: mapping
        for technique, mapping in mappings.items()
        if mapping_supports_platform(mapping, target_os)
    }

    known = [
        technique
        for technique in techniques
        if technique in available
    ]

    if not known:
        raise ValueError(
            "No MITRE telemetry mapping is available for the rule's "
            f"techniques and target OS '{target_os}'. "
            f"Tagged techniques: {techniques}"
        )

    chain = []

    for tactic in TACTIC_ORDER:
        candidates = [
            technique
            for technique, mapping in available.items()
            if mapping.get("tactic") == tactic
        ]

        if not candidates:
            continue

        tagged_for_tactic = [
            technique
            for technique in known
            if available[technique].get("tactic") == tactic
        ]

        if tagged_for_tactic:
            chain.append(tagged_for_tactic[0])
        else:
            chain.append(candidates[0])

    return chain


def write_graph(
    driver,
    run_id: str,
    chain: List[str],
    mappings: dict,
    user: str,
    host: str,
) -> str:
    """
    Write the attack path into Neo4j as:

        User -> Host
        Stage -> Host
        Stage -> Stage

    Returns the element ID of the final/root stage.
    """

    with driver.session() as session:
        session.run(
            "MATCH (n {run_id: $run_id}) DETACH DELETE n",
            run_id=run_id,
        )

        session.run(
            """
            MERGE (u:User {
                name: $user,
                run_id: $run_id
            })

            MERGE (h:Host {
                name: $host,
                run_id: $run_id
            })

            MERGE (u)-[:LOGGED_INTO]->(h)
            """,
            user=user,
            host=host,
            run_id=run_id,
        )

        previous_id = None

        for index, technique in enumerate(chain):
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

                MATCH (h:Host {
                    name: $host,
                    run_id: $run_id
                })

                MERGE (s)-[:OBSERVED_ON]->(h)

                RETURN elementId(s) AS id
                """,
                run_id=run_id,
                seq=index,
                tid=technique,
                tname=meta["name"],
                tactic=meta["tactic"],
                host=host,
            )

            stage_id = result.single()["id"]

            if previous_id:
                session.run(
                    """
                    MATCH (a:Stage)
                    WHERE elementId(a) = $previous

                    MATCH (b:Stage)
                    WHERE elementId(b) = $current

                    MERGE (a)-[:NEXT]->(b)
                    """,
                    previous=previous_id,
                    current=stage_id,
                )

            previous_id = stage_id

        return previous_id


def inject_telemetry(
    es: Elasticsearch,
    run_id: str,
    chain: List[str],
    mappings: dict,
    user: str,
    host: str,
    target_os: Optional[str],
) -> int:
    """
    Convert each attack stage into Elasticsearch telemetry.

    Each event contains explicit WRAITH metadata identifying the
    synthetic attack and its detection target.
    """

    index_name = f"wraith-attack-{run_id}"

    start = dt.datetime.now(dt.UTC) - dt.timedelta(hours=2)

    def actions():
        for index, technique in enumerate(chain):
            meta = mappings[technique]

            timestamp = start + dt.timedelta(
                minutes=index * 7
            )

            doc = {
                "@timestamp": timestamp.isoformat(),
                "user": user,
                "host": host,
                "mitre_technique": technique,
                "mitre_tactic": meta["tactic"],
                "wraith_synthetic": True,
                "wraith_label": "attack",
                "wraith_run_id": run_id,
                "wraith_target_os": target_os,
                **meta["event"],
            }

            yield {
                "_index": index_name,
                "_source": doc,
            }

    success, errors = helpers.bulk(
        es,
        actions(),
        stats_only=True,
        raise_on_error=False,
    )

    if errors:
        print(
            f"[attack_simulator] WARNING: "
            f"{errors} attack docs failed to index",
            file=sys.stderr,
        )

    es.indices.refresh(index=index_name)

    return success


def simulate(
    rule_path: str,
    es_addr: str,
    neo4j_addr: str,
    run_id: str,
    neo4j_user: str = "neo4j",
    neo4j_pass: str = "wraith-test-pw",
    target_os: Optional[str] = None,
) -> dict:
    """
    Simulate an attack against the requested detection target.

    The host OS running this Python process is intentionally irrelevant.
    """

    mappings = load_mappings()

    resolved_target_os, target_source = resolve_target_os(
        rule_path,
        target_os,
    )

    techniques = extract_techniques_from_rule(
        rule_path
    )

    chain = build_attack_chain(
        techniques,
        mappings,
        target_os=resolved_target_os,
    )

    user = f"jdoe-{uuid.uuid4().hex[:4]}"
    host = f"WKS-ATK-{uuid.uuid4().hex[:4]}"

    driver = GraphDatabase.driver(
        neo4j_addr,
        auth=(neo4j_user, neo4j_pass),
    )

    try:
        write_graph(
            driver,
            run_id,
            chain,
            mappings,
            user,
            host,
        )
    finally:
        driver.close()

    es = Elasticsearch(es_addr)

    indexed = inject_telemetry(
        es,
        run_id,
        chain,
        mappings,
        user,
        host,
        resolved_target_os,
    )

    return {
        "run_id": run_id,
        "target_os": resolved_target_os,
        "target_os_source": target_source,
        "rule_tagged_techniques": techniques,
        "simulated_chain": chain,
        "simulated_user": user,
        "simulated_host": host,
        "events_indexed": indexed,
    }


def main():
    parser = argparse.ArgumentParser(
        description=(
            "Simulate a graph-based multi-stage "
            "target-OS attack"
        )
    )

    parser.add_argument("--rule", required=True)
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--neo4j-addr", required=True)
    parser.add_argument("--run-id", required=True)

    parser.add_argument(
        "--neo4j-user",
        default="neo4j",
    )

    parser.add_argument(
        "--neo4j-pass",
        default="wraith-test-pw",
    )

    parser.add_argument(
        "--target-os",
        choices=sorted(SUPPORTED_TARGET_OS),
        default=None,
    )

    args = parser.parse_args()

    result = simulate(
        args.rule,
        args.es_addr,
        args.neo4j_addr,
        args.run_id,
        neo4j_user=args.neo4j_user,
        neo4j_pass=args.neo4j_pass,
        target_os=args.target_os,
    )

    print(
        json.dumps(
            result,
            indent=2,
        )
    )


if __name__ == "__main__":
    main()