"""Deterministic multi-rule correlation checks over Wraith validation results.

The input is a JSON list of rule verdicts. A correlation manifest declares
required rule IDs and optional order. This keeps correlation logic backend-
agnostic while making multi-stage detection coverage testable in CI.
"""
from __future__ import annotations
import argparse, json
from pathlib import Path

def evaluate(manifest: dict, verdicts: list[dict]) -> dict:
    by_id={v.get("rule_id"):v for v in verdicts}
    required=manifest.get("rules",[])
    missing=[r for r in required if r not in by_id]
    failed=[r for r in required if r in by_id and not by_id[r].get("passed",False)]
    order=manifest.get("order", required)
    timestamps=[by_id[r].get("detected_at") for r in order if r in by_id]
    ordered=all(a <= b for a,b in zip([x for x in timestamps if x], [x for x in timestamps if x][1:]))
    return {"name":manifest.get("name","correlation"),"required_rules":required,"missing_rules":missing,
            "failed_rules":failed,"ordered":ordered,"passed":not missing and not failed and ordered}

def main():
    p=argparse.ArgumentParser(); p.add_argument("--manifest",required=True); p.add_argument("--verdicts",required=True); p.add_argument("--out",required=True)
    a=p.parse_args(); r=evaluate(json.loads(Path(a.manifest).read_text()),json.loads(Path(a.verdicts).read_text())); Path(a.out).write_text(json.dumps(r,indent=2)); raise SystemExit(0 if r["passed"] else 1)
if __name__=="__main__": main()
