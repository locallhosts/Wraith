"""Map ATT&CK-tagged detections to organization-supplied control mappings.

The mapping file is intentionally user-owned: Wraith does not present a
framework mapping as authoritative compliance advice.
"""
from __future__ import annotations
import argparse,json,re
from pathlib import Path
import yaml

TECH=re.compile(r"^attack\.(T[0-9]+(?:\.[0-9]+)?)$",re.I)

def rule_techniques(rule):
    return sorted({m.group(1).upper() for t in rule.get("tags",[]) or [] if (m:=TECH.match(str(t)))})

def map_rule(rule, mapping):
    rows=[]
    for t in rule_techniques(rule):
        item=mapping.get(t,{})
        rows.append({"technique":t,"controls":item.get("controls",[]),"source":item.get("source"),"mapped":bool(item.get("controls"))})
    return {"rule_id":rule.get("id"),"mappings":rows,"unmapped_techniques":[r["technique"] for r in rows if not r["mapped"]]}

def main():
    p=argparse.ArgumentParser(); p.add_argument("--rule",required=True); p.add_argument("--mapping",required=True); p.add_argument("--out",required=True)
    a=p.parse_args(); rule=yaml.safe_load(Path(a.rule).read_text()); mapping=json.loads(Path(a.mapping).read_text()); Path(a.out).write_text(json.dumps(map_rule(rule,mapping),indent=2))
if __name__=="__main__": main()
