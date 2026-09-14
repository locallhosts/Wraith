"""Export ATT&CK-tagged Sigma coverage as an ATT&CK Navigator layer."""
from __future__ import annotations
import argparse, json, re
from pathlib import Path
import yaml

TECH = re.compile(r"^attack\.(T[0-9]+(?:\.[0-9]+)?)$", re.I)

def techniques_from_rule(rule: dict) -> list[str]:
    found=[]
    for tag in rule.get("tags", []) or []:
        m=TECH.match(str(tag))
        if m: found.append(m.group(1).upper())
    return sorted(set(found))

def export_layer(rules: list[dict], name: str = "Wraith ATT&CK Coverage") -> dict:
    techniques=sorted({t for r in rules for t in techniques_from_rule(r)})
    return {
      "name": name, "versions": {"attack": "16", "navigator": "4.9.0", "layer": "4.5"},
      "domain": "enterprise-attack", "description": "Coverage derived from Sigma rule ATT&CK tags.",
      "filters": {"platforms": ["Windows", "Linux", "macOS", "Cloud", "Network", "Containers"]},
      "sorting": 3, "layout": {"layout": "side", "aggregateFunction": "average", "showID": True,
      "showName": True, "showDescription": False, "showAggregateScores": True, "showUnusedTechniques": False},
      "techniques": [{"techniqueID": t, "score": 1, "comment": "Detected by one or more Wraith Sigma rules"} for t in techniques],
      "gradient": {"colors": ["#ffffff", "#66b3ff", "#ff0000"], "minValue": 0, "maxValue": 1},
      "legendItems": [{"label": "Covered", "color": "#ff0000"}],
    }

def main():
    p=argparse.ArgumentParser(); p.add_argument("--rules", nargs="+", required=True); p.add_argument("--out", required=True)
    a=p.parse_args(); rules=[yaml.safe_load(Path(x).read_text()) for x in a.rules]
    Path(a.out).write_text(json.dumps(export_layer(rules), indent=2))
if __name__ == "__main__": main()
