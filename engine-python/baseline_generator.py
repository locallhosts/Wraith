"""
baseline_generator.py

Generates a realistic 24-hour window of *benign* enterprise telemetry
(process creation, network connections, file activity, auth events) and
bulk-indexes it into the ephemeral Elasticsearch instance under
`wraith-baseline-*`. This is the corpus the CI pipeline checks new rules
against to catch false positives before they ever reach a real SOC.

This is real data generation (via Faker) and a real Elasticsearch bulk
API call — not a mock.
"""
from __future__ import annotations

import argparse
import datetime as dt
import random
import sys
from typing import Iterator

from elasticsearch import Elasticsearch, helpers
from faker import Faker

fake = Faker()

BENIGN_PROCESSES = [
    "chrome.exe", "outlook.exe", "teams.exe", "excel.exe", "winword.exe",
    "explorer.exe", "svchost.exe", "code.exe", "slack.exe", "onedrive.exe",
    "powershell.exe",  # legit admin usage too — good false-positive stress test
    "cmd.exe", "python.exe", "notepad.exe",
]

BENIGN_POWERSHELL_CMDLINES = [
    'powershell.exe -Command "Get-Process | Sort-Object CPU -Descending"',
    'powershell.exe -Command "Get-ADUser -Filter *"',
    'powershell.exe -File C:\\Scripts\\daily_report.ps1',
    'powershell.exe -Command "Import-Module ActiveDirectory"',
]

USERS = [fake.user_name() for _ in range(40)]
HOSTS = [f"WKS-{fake.random_int(1000, 9999)}" for _ in range(60)]


def _process_event(ts: dt.datetime) -> dict:
    proc = random.choice(BENIGN_PROCESSES)
    cmdline = proc
    if proc == "powershell.exe":
        cmdline = random.choice(BENIGN_POWERSHELL_CMDLINES)
    return {
        "@timestamp": ts.isoformat(),
        "event_type": "process_creation",
        "user": random.choice(USERS),
        "host": random.choice(HOSTS),
        "process_name": proc,
        "command_line": cmdline,
        "parent_process": "explorer.exe",
        "wraith_synthetic": True,
        "wraith_label": "baseline",
    }


def _network_event(ts: dt.datetime) -> dict:
    return {
        "@timestamp": ts.isoformat(),
        "event_type": "network_connection",
        "user": random.choice(USERS),
        "host": random.choice(HOSTS),
        "process_name": random.choice(["chrome.exe", "outlook.exe", "teams.exe"]),
        "dest_ip": fake.ipv4_public(),
        "dest_domain": fake.domain_name(),
        "dest_port": random.choice([443, 443, 443, 80, 993]),
        "wraith_synthetic": True,
        "wraith_label": "baseline",
    }


def _auth_event(ts: dt.datetime) -> dict:
    return {
        "@timestamp": ts.isoformat(),
        "event_type": "authentication",
        "user": random.choice(USERS),
        "host": random.choice(HOSTS),
        "logon_type": random.choice(["interactive", "network", "remote_interactive"]),
        "result": "success",
        "wraith_synthetic": True,
        "wraith_label": "baseline",
    }


def _file_event(ts: dt.datetime) -> dict:
    return {
        "@timestamp": ts.isoformat(),
        "event_type": "file_modification",
        "user": random.choice(USERS),
        "host": random.choice(HOSTS),
        "process_name": random.choice(["winword.exe", "excel.exe", "explorer.exe"]),
        "file_path": fake.file_path(depth=3),
        "wraith_synthetic": True,
        "wraith_label": "baseline",
    }


_GENERATORS = [_process_event, _network_event, _auth_event, _file_event]


def generate_events(hours: int, events_per_hour: int) -> Iterator[dict]:
    """Yield synthetic benign events spread evenly (with jitter) across the window."""
    start = dt.datetime.utcnow() - dt.timedelta(hours=hours)
    total = hours * events_per_hour
    for i in range(total):
        offset_seconds = (i / total) * hours * 3600 + random.uniform(-5, 5)
        ts = start + dt.timedelta(seconds=max(0, offset_seconds))
        gen = random.choices(_GENERATORS, weights=[0.5, 0.25, 0.15, 0.10])[0]
        yield gen(ts)


def index_baseline(es: Elasticsearch, run_id: str, hours: int = 24, events_per_hour: int = 500) -> int:
    """Bulk-indexes the generated baseline events. Returns count indexed."""
    index_name = f"wraith-baseline-{run_id}"

    def actions():
        for ev in generate_events(hours, events_per_hour):
            yield {"_index": index_name, "_source": ev}

    success, errors = helpers.bulk(es, actions(), stats_only=True, raise_on_error=False)
    if errors:
        print(f"[baseline_generator] WARNING: {errors} documents failed to index", file=sys.stderr)
    es.indices.refresh(index=index_name)
    return success


def main():
    parser = argparse.ArgumentParser(description="Generate and index synthetic baseline telemetry")
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--hours", type=int, default=24)
    parser.add_argument("--events-per-hour", type=int, default=500)
    args = parser.parse_args()

    es = Elasticsearch(args.es_addr)
    count = index_baseline(es, args.run_id, args.hours, args.events_per_hour)
    print(f"[baseline_generator] indexed {count} baseline events into wraith-baseline-{args.run_id}")


if __name__ == "__main__":
    main()
