"""
run_pipeline.py

The full WRAITH per-rule pipeline, run once per Sigma rule file in a PR:

  1. Translate the rule to an Elasticsearch Query DSL (pySigma, real).
  2. Generate 24h of synthetic baseline traffic and index it (real ES writes).
  3. Build a multi-stage MITRE-mapped attack graph in Neo4j and inject the
     corresponding telemetry into Elasticsearch (real Neo4j + ES writes).
  4. Validate: does the rule fire on the attack and stay silent on baseline?
  5. If it passes, adversarially fuzz the attack telemetry (case variation,
     PowerShell parameter aliasing, env-var path substitution, whitespace
     variation) and measure what fraction of evasion variants the rule
     still catches — the "Robustness Score" (see robustness_fuzzer.py).
  6. If it passes, draft a SOAR playbook via the Anthropic API and (if
     configured) open a real GitHub PR with it.
  7. Write a JSON report to engine-python/output/<run_id>/report.json and
     exit non-zero if the rule failed, so this gates the CI job. A
     separate CI step (`wraith attest`) then signs this report into a
     tamper-evident provenance attestation — see backend-go/provenance.

This is the script backend-go/api/server.go shells out to for each run.
"""
from __future__ import annotations

import argparse
import json
import sys
import time
import yaml
from pathlib import Path

from elasticsearch import Elasticsearch

import attack_simulator
import baseline_generator
import robustness_check
import sigma_to_es
import soar_playbook_generator
import validate as validate_mod
import mutation_testing
import mutation_runner
import quality_score


def main():
    parser = argparse.ArgumentParser(description="Run the full WRAITH detection-testing pipeline for one rule")
    parser.add_argument("--rule", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--es-addr", required=True)
    parser.add_argument("--neo4j-addr", required=True)
    parser.add_argument("--neo4j-user", default="neo4j")
    parser.add_argument("--neo4j-pass", default="wraith-test-pw")
    parser.add_argument("--baseline-hours", type=int, default=24)
    parser.add_argument("--baseline-events-per-hour", type=int, default=500)
    parser.add_argument("--perf-reject-threshold-pct", type=float, default=5.0)
    parser.add_argument("--robustness-variants", type=int, default=40)
    parser.add_argument("--mutation-testing", action="store_true", help="Generate Sigma mutants for validation corpus testing")
    parser.add_argument("--open-pr", action="store_true")
    parser.add_argument("--output-dir", default="engine-python/output")
    args = parser.parse_args()

    out_dir = Path(args.output_dir) / args.run_id
    out_dir.mkdir(parents=True, exist_ok=True)
    t0 = time.time()

    with open(args.rule) as f:
        rule_yaml = yaml.safe_load(f)
    rule_id = rule_yaml.get("id", args.run_id)

    report = {"run_id": args.run_id, "rule_path": args.rule, "rule_id": rule_id, "stages": {}}

    # --- Stage 1: translate ---
    print("[run_pipeline] translating Sigma rule -> Elasticsearch Query DSL")
    query_dsl = sigma_to_es.translate(args.rule)
    dsl_path = out_dir / "query_dsl.json"
    dsl_path.write_text(json.dumps(query_dsl, indent=2))
    report["stages"]["translate"] = {"status": "ok", "query_dsl_path": str(dsl_path)}

    es = Elasticsearch(args.es_addr)

    # --- Stage 2: baseline ---
    print(f"[run_pipeline] generating {args.baseline_hours}h of synthetic baseline traffic")
    baseline_count = baseline_generator.index_baseline(
        es, args.run_id, hours=args.baseline_hours, events_per_hour=args.baseline_events_per_hour
    )
    report["stages"]["baseline"] = {"status": "ok", "events_indexed": baseline_count}

    # --- Stage 3: attack simulation ---
    print("[run_pipeline] building MITRE-mapped attack graph in Neo4j and injecting telemetry")
    sim_result = attack_simulator.simulate(
        args.rule, args.es_addr, args.neo4j_addr, args.run_id,
        neo4j_user=args.neo4j_user, neo4j_pass=args.neo4j_pass,
    )
    report["stages"]["attack_simulation"] = {"status": "ok", **sim_result}

    # --- Stage 4: validate ---
    print("[run_pipeline] validating rule against attack + baseline indices")
    verdict = validate_mod.validate(es, args.run_id, query_dsl)
    report["stages"]["validate"] = verdict
    report["passed"] = verdict["passed"]

    # --- Stage 5: adversarial robustness fuzzing (only if the rule passed
    # attack detection — no point evasion-testing a rule that doesn't even
    # catch the textbook case) ---
    if verdict["passed"]:
        print(f"[run_pipeline] validating against {args.baseline_events_per_hour}x baseline passed — "
              f"running adversarial evasion robustness check")
        techniques = attack_simulator.extract_techniques_from_rule(args.rule)
        mappings = attack_simulator.load_mappings()
        primary_technique = next((t for t in techniques if t in mappings), None)
        if primary_technique:
            base_event = dict(mappings[primary_technique]["event"])
            robustness_result = robustness_check.run_robustness_check(
                args.es_addr, args.run_id, rule_id, query_dsl, base_event,
                n_variants=args.robustness_variants,
            )
            report["stages"]["robustness"] = robustness_result
            print(f"[run_pipeline] robustness score: {robustness_result['score']:.0%} "
                  f"({robustness_result['variants_detected']}/{robustness_result['variants_tested']} evasion variants still caught)")
            if robustness_result["score"] < 0.7:
                print("[run_pipeline] WARNING: robustness score below 70% — rule is easily evaded by trivial "
                      "obfuscation. Consider broadening the detection (e.g. case-insensitive match, "
                      "additional parameter aliases) before this ships to production.", file=sys.stderr)
        else:
            report["stages"]["robustness"] = {"status": "skipped", "reason": "no mappable MITRE technique tagged on rule"}

    # --- Stage 6: mutation corpus generation (optional) ---
    # Mutants are generated here but are not silently executed against production
    # infrastructure. Each mutant must be run through an isolated validation
    # corpus and marked killed/alive by the caller.
    if args.mutation_testing:
        print("[run_pipeline] running mutation testing against the isolated validation corpus")
        mutation_result = mutation_runner.run(args.es_addr, args.run_id, rule_yaml)
        mutation_path = out_dir / "mutation_results.json"
        mutation_path.write_text(json.dumps(mutation_result, indent=2))
        report["stages"]["mutation_testing"] = mutation_result
        report["stages"]["mutation_testing"]["path"] = str(mutation_path)

    # --- Stage 7: explainable quality score ---
    report["stages"]["quality_score"] = quality_score.score(report)

    # --- Stage 8: SOAR playbook (only if the rule passed) ---
    if verdict["passed"]:
        print("[run_pipeline] rule passed — drafting SOAR playbook")
        try:
            ctx = soar_playbook_generator.load_rule_context(args.rule)
            code = soar_playbook_generator.draft_playbook(ctx)
            playbook_path = soar_playbook_generator.write_playbook_file(
                args.run_id, rule_id, code, out_dir
            )
            report["stages"]["soar_playbook"] = {"status": "ok", "path": str(playbook_path)}

            if args.open_pr:
                import os
                token = os.environ.get("GITHUB_TOKEN")
                repo = os.environ.get("GITHUB_REPO")
                base = os.environ.get("GITHUB_BASE_BRANCH", "main")
                if token and repo:
                    pr_url = soar_playbook_generator.open_github_pr(
                        repo, base, args.run_id, rule_id, playbook_path, token
                    )
                    report["stages"]["soar_playbook"]["pr_url"] = pr_url
                    print(f"[run_pipeline] opened SOAR review PR: {pr_url}")
                else:
                    print("[run_pipeline] GITHUB_TOKEN/GITHUB_REPO not set — skipping PR creation")
        except Exception as e:  # SOAR drafting failure should not silently pass the run
            print(f"[run_pipeline] WARNING: SOAR playbook generation failed: {e}", file=sys.stderr)
            report["stages"]["soar_playbook"] = {"status": "error", "error": str(e)}
    else:
        print(f"[run_pipeline] rule FAILED validation: {verdict['reason']}")

    report["duration_seconds"] = round(time.time() - t0, 2)
    report_path = out_dir / "report.json"
    report_path.write_text(json.dumps(report, indent=2))
    print(f"[run_pipeline] wrote report to {report_path}")

    if not verdict["passed"]:
        sys.exit(1)


if __name__ == "__main__":
    main()
