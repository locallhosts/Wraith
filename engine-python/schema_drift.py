"""Detect fields used by Sigma rules that are absent from an observed schema."""
from __future__ import annotations
import argparse, json
from pathlib import Path
import yaml

META={"condition"}

def rule_fields(rule: dict) -> set[str]:
    fields=set()
    for name, selection in (rule.get("detection") or {}).items():
        if name in META or not isinstance(selection, dict): continue
        fields.update(str(k).split("|")[0] for k in selection)
    return fields

def compare(rule: dict, schema: dict) -> dict:
    observed=set(schema.get("fields", schema if isinstance(schema, list) else []))
    used=rule_fields(rule)
    missing=sorted(used-observed)
    return {"rule_id":rule.get("id"),"fields_used":sorted(used),"observed_fields":sorted(observed),
            "missing_fields":missing,"drift":bool(missing)}

def main():
    p=argparse.ArgumentParser(); p.add_argument("--rule",required=True); p.add_argument("--schema",required=True); p.add_argument("--out")
    a=p.parse_args(); r=compare(yaml.safe_load(Path(a.rule).read_text()),json.loads(Path(a.schema).read_text()))
    text=json.dumps(r,indent=2)
    if a.out: Path(a.out).write_text(text)
    else: print(text)
    raise SystemExit(1 if r["drift"] else 0)
if __name__=="__main__": main()
