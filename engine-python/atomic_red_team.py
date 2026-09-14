"""Import MITRE Atomic Red Team tests without executing them.

Wraith treats atomics as attack fixtures. Execution remains an explicit,
separate operator action; CI never executes an atomic by default.
"""
from __future__ import annotations
import argparse,json
from pathlib import Path
import yaml

def load_atomic(path: str) -> dict:
    data=yaml.safe_load(Path(path).read_text())
    return {"attack_technique": data.get("attack_technique",{}), "atomic_tests": data.get("atomic_tests",[])}

def select(data: dict, test_id: str|None=None, technique: str|None=None) -> list[dict]:
    out=[]
    for test in data["atomic_tests"]:
        if test_id and test.get("auto_generated_guid") != test_id: continue
        if technique and data["attack_technique"].get("external_id") != technique: continue
        out.append(test)
    return out

def main():
    p=argparse.ArgumentParser(); p.add_argument("--atomics",required=True); p.add_argument("--technique"); p.add_argument("--test-id"); p.add_argument("--out",required=True)
    a=p.parse_args(); data=load_atomic(a.atomics); Path(a.out).write_text(json.dumps({"source":"Atomic Red Team","execution": "not performed", "tests":select(data,a.test_id,a.technique)},indent=2))
if __name__=="__main__": main()
