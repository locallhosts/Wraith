"""
soar_playbook_generator.py

When a new detection rule passes validation, this drafts a Python SOAR
response playbook (e.g. "isolate host, revoke tokens, notify on-call") by
calling the real Anthropic Messages API, then opens a real GitHub pull
request containing the draft for a human analyst to review and approve.

Nothing here auto-executes on production systems — the whole point is a
human-in-the-loop PR, not an auto-responding bot.
"""
from __future__ import annotations

import argparse
import base64
import json
import os
import sys
from pathlib import Path

import requests
import yaml
from anthropic import Anthropic

SOAR_SYSTEM_PROMPT = """You are a SOC automation engineer drafting a SOAR \
(Security Orchestration, Automation and Response) playbook in Python for a \
SOAR platform with an incident-response SDK exposing:

    soar.isolate_host(hostname: str, reason: str) -> None
    soar.revoke_user_tokens(username: str) -> None
    soar.disable_user_account(username: str) -> None
    soar.notify_oncall(channel: str, message: str, severity: str) -> None
    soar.create_ticket(title: str, description: str, severity: str) -> str
    soar.quarantine_file(host: str, file_path: str, sha256: str = None) -> None
    soar.block_network_indicator(indicator: str, indicator_type: str) -> None

Given a Sigma detection rule (title, description, MITRE tags, level), \
write a single Python function `respond(alert: dict) -> None` that takes \
an alert dict with keys `host`, `user`, `technique`, `severity`, and \
`raw_event`, and calls an appropriate, conservative, and specifically \
scoped subset of the SDK above.

Rules:
- Always create a ticket first so there is an audit trail.
- Only isolate a host or disable/revoke a user for `high` or `critical` \
severity alerts — for lower severity, notify on-call and create a ticket \
only.
- Always notify on-call for anything credential-access, lateral-movement, \
or impact related, regardless of severity.
- Add a short docstring explaining the reasoning for a human reviewer.
- Output ONLY the Python code, no prose, no markdown fences.
"""


def load_rule_context(rule_path: str) -> dict:
    with open(rule_path) as f:
        rule = yaml.safe_load(f)
    return {
        "title": rule.get("title", ""),
        "description": rule.get("description", ""),
        "level": rule.get("level", "medium"),
        "tags": rule.get("tags", []),
        "logsource": rule.get("logsource", {}),
    }


def draft_playbook(rule_context: dict, api_key: str | None = None) -> str:
    client = Anthropic(api_key=api_key or os.environ.get("ANTHROPIC_API_KEY"))

    user_prompt = (
        f"Sigma rule title: {rule_context['title']}\n"
        f"Description: {rule_context['description']}\n"
        f"Level: {rule_context['level']}\n"
        f"MITRE tags: {', '.join(rule_context['tags'])}\n"
        f"Logsource: {json.dumps(rule_context['logsource'])}\n\n"
        "Draft the respond() function now."
    )

    message = client.messages.create(
        model="claude-sonnet-4-6",
        max_tokens=1200,
        system=SOAR_SYSTEM_PROMPT,
        messages=[{"role": "user", "content": user_prompt}],
    )

    text_parts = [block.text for block in message.content if block.type == "text"]
    return "\n".join(text_parts).strip()


def write_playbook_file(run_id: str, rule_id: str, code: str, output_dir: Path) -> Path:
    output_dir.mkdir(parents=True, exist_ok=True)
    path = output_dir / f"playbook_{rule_id}.py"
    header = (
        f'"""\n'
        f"Auto-drafted SOAR playbook — WRAITH pipeline run {run_id}\n"
        f"Rule ID: {rule_id}\n\n"
        f"THIS IS A DRAFT. A human analyst must review and approve this PR\n"
        f"before it is merged and wired into the SOAR platform.\n"
        f'"""\n\n'
    )
    path.write_text(header + code + "\n")
    return path


def open_github_pr(repo: str, base_branch: str, run_id: str, rule_id: str,
                    playbook_path: Path, github_token: str) -> str:
    """Creates a branch, commits the playbook file via the GitHub Contents
    API, and opens a real pull request against `base_branch`. Returns the
    PR URL."""
    api = f"https://api.github.com/repos/{repo}"
    headers = {
        "Authorization": f"Bearer {github_token}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    }

    base_ref = requests.get(f"{api}/git/ref/heads/{base_branch}", headers=headers)
    base_ref.raise_for_status()
    base_sha = base_ref.json()["object"]["sha"]

    branch_name = f"wraith/soar-playbook-{rule_id[:8]}-{run_id[:8]}"
    requests.post(
        f"{api}/git/refs", headers=headers,
        json={"ref": f"refs/heads/{branch_name}", "sha": base_sha},
    ).raise_for_status()

    content_b64 = base64.b64encode(playbook_path.read_bytes()).decode()
    remote_path = f"soar/playbooks/{playbook_path.name}"
    requests.put(
        f"{api}/contents/{remote_path}", headers=headers,
        json={
            "message": f"Draft SOAR playbook for rule {rule_id}",
            "content": content_b64,
            "branch": branch_name,
        },
    ).raise_for_status()

    pr_resp = requests.post(
        f"{api}/pulls", headers=headers,
        json={
            "title": f"[WRAITH] Draft SOAR playbook for rule {rule_id}",
            "head": branch_name,
            "base": base_branch,
            "body": (
                f"Auto-generated by the WRAITH detection pipeline for run `{run_id}`.\n\n"
                f"This rule **passed** attack-detection and false-positive testing "
                f"and now has a draft incident-response playbook attached.\n\n"
                "**A SOC analyst must review this before merge.** Nothing in this "
                "playbook executes automatically until approved and wired into "
                "the SOAR platform."
            ),
        },
    )
    pr_resp.raise_for_status()
    return pr_resp.json()["html_url"]


def main():
    parser = argparse.ArgumentParser(description="Draft a SOAR playbook and open a review PR")
    parser.add_argument("--rule", required=True)
    parser.add_argument("--rule-id", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--output-dir", default="engine-python/output")
    parser.add_argument("--open-pr", action="store_true", help="also open a GitHub PR (requires GITHUB_TOKEN, GITHUB_REPO)")
    args = parser.parse_args()

    ctx = load_rule_context(args.rule)
    code = draft_playbook(ctx)
    path = write_playbook_file(args.run_id, args.rule_id, code, Path(args.output_dir) / args.run_id)
    print(f"[soar_playbook_generator] wrote {path}")

    if args.open_pr:
        token = os.environ.get("GITHUB_TOKEN")
        repo = os.environ.get("GITHUB_REPO")
        base = os.environ.get("GITHUB_BASE_BRANCH", "main")
        if not token or not repo:
            print("GITHUB_TOKEN / GITHUB_REPO not set, skipping PR creation", file=sys.stderr)
            return
        url = open_github_pr(repo, base, args.run_id, args.rule_id, path, token)
        print(f"[soar_playbook_generator] opened PR: {url}")


if __name__ == "__main__":
    main()
