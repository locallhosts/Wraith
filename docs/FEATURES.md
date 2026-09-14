# Advanced detection-engineering features

This document describes the advanced validation capabilities added around the
core Wraith pipeline.

## Mutation testing

`engine-python/mutation_testing.py` generates deliberately weakened Sigma
variants: dropped fields, dropped list values, case variation, and reduced
value sets. `mutation_runner.py` can then run those mutants against the same
isolated Elasticsearch attack/baseline corpus used by the pipeline and records
whether each mutant was **killed**. The mutation score is the fraction of tested
mutants rejected by the validation corpus. Production infrastructure is never
used by this stage.

This measures the quality of Wraith's test corpus, not the security of the
production rule by itself.

## Rule regression diff

`rule_diff.py` emits a normalized SHA-256 comparison and unified diff. CI can
place the Markdown result in `$GITHUB_STEP_SUMMARY` or a pull-request comment.
It intentionally does not claim behavioral regression unless the changed rule
has been revalidated.

## Atomic Red Team

`atomic_red_team.py` imports Atomic Red Team test YAML as an attack-fixture
source. It **does not execute atomics automatically**. Execution is a separate
operator-controlled activity because atomics can perform real security
behaviors on the host.

## ATT&CK Navigator

`attack_navigator.py` converts ATT&CK tags on Sigma rules into a Navigator
layer, making rule coverage reviewable outside the Wraith UI.

## Schema drift

`schema_drift.py` compares fields used by a rule against an observed schema
JSON file. Missing fields fail the check, allowing schema changes to become a
CI gate before a detection silently stops matching.

## Multi-rule correlation

`correlation.py` evaluates a declarative correlation manifest against existing
rule verdicts. It checks required rule IDs, individual pass/fail state, and an
optional detection order. Backend-specific event correlation can be layered on
top without changing this deterministic contract.

## Detection Quality Score

`quality_score.py` produces an explainable 0–100 score from behavioral
validation, false positives, adversarial robustness, and mutation testing.
The score is a triage signal, not a universal security grade.

## Compliance mapping

`compliance_map.py` maps ATT&CK techniques to an organization-supplied JSON
control map. Wraith deliberately does not ship authoritative regulatory
mappings; teams must review and own their mappings.

## GitHub App

`integrations/github-app/manifest.yml` provides a minimal App manifest for
installing Wraith with repository-read and pull-request-write permissions.
The existing webhook endpoint remains the processing surface.

## Falco and YARA

Wraith does **not** perform a generic Sigma→YARA conversion. YARA is a file/content
matching language, while Sigma models telemetry events. A mechanical converter
would create misleading detections. Falco has a closer event-oriented model,
but field semantics and rule condition syntax still require an explicit backend
contract. These integrations remain intentionally separate rather than
shipping a converter that produces plausible-looking but semantically wrong
rules.
