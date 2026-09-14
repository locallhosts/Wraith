"""Run generated Sigma mutants against an isolated Wraith validation corpus."""
from __future__ import annotations
import argparse, json, tempfile
from pathlib import Path
import yaml
from elasticsearch import Elasticsearch
import mutation_testing
import sigma_to_es
import validate as validate_mod


def run(es_addr: str, run_id: str, rule: dict) -> dict:
    es = Elasticsearch(es_addr)
    results=[]
    for mutant in mutation_testing.generate_mutants(rule):
        with tempfile.NamedTemporaryFile("w", suffix=".yml") as f:
            yaml.safe_dump(mutant.rule, f, sort_keys=False)
            f.flush()
            query=sigma_to_es.translate(f.name)
        verdict=validate_mod.validate(es, run_id, query)
        results.append({"mutation_id":mutant.mutation_id,"operator":mutant.operator,
                        "description":mutant.description,"killed":not verdict["passed"],
                        "reason":verdict["reason"]})
    return {"status":"ok" if results else "no_mutants", **mutation_testing.mutation_score(results), "results":results}


def main():
    p=argparse.ArgumentParser(); p.add_argument("--rule",required=True); p.add_argument("--run-id",required=True); p.add_argument("--es-addr",required=True); p.add_argument("--out",required=True)
    a=p.parse_args(); rule=yaml.safe_load(Path(a.rule).read_text()); result=run(a.es_addr,a.run_id,rule); Path(a.out).write_text(json.dumps(result,indent=2)); print(json.dumps(result,indent=2)); raise SystemExit(0 if result["status"] in ("ok","no_mutants") else 1)
if __name__=="__main__": main()
