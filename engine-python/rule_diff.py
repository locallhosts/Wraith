"""Rule diff and behavioral-regression report helpers."""
from __future__ import annotations
import argparse
import difflib
import hashlib
import json
from pathlib import Path
import yaml


def normalized(rule: dict) -> str:
    return yaml.safe_dump(rule, sort_keys=True, default_flow_style=False)


def diff(old: dict, new: dict) -> dict:
    a, b = normalized(old).splitlines(), normalized(new).splitlines()
    return {
        "changed": a != b,
        "old_sha256": hashlib.sha256(normalized(old).encode()).hexdigest(),
        "new_sha256": hashlib.sha256(normalized(new).encode()).hexdigest(),
        "unified_diff": "\n".join(difflib.unified_diff(a, b, fromfile="base", tofile="head", lineterm="")),
    }


def markdown(report: dict) -> str:
    status = "changed" if report["changed"] else "unchanged"
    body = ["## Wraith rule regression", f"**Status:** {status}",
            f"**Base SHA-256:** `{report['old_sha256']}`",
            f"**Head SHA-256:** `{report['new_sha256']}`"]
    if report["unified_diff"]:
        body += ["", "```diff", report["unified_diff"], "```"]
    return "\n".join(body)


def main() -> None:
    p = argparse.ArgumentParser(description="Compare two Sigma rule files")
    p.add_argument("--base", required=True)
    p.add_argument("--head", required=True)
    p.add_argument("--json")
    p.add_argument("--markdown")
    a = p.parse_args()
    r = diff(yaml.safe_load(Path(a.base).read_text()), yaml.safe_load(Path(a.head).read_text()))
    if a.json: Path(a.json).write_text(json.dumps(r, indent=2))
    if a.markdown: Path(a.markdown).write_text(markdown(r))
    if not a.json and not a.markdown: print(markdown(r))

if __name__ == "__main__": main()
