# Architecture

## Why two languages

**Go** owns everything that needs to be fast, concurrent, and close to
infrastructure: the webhook receiver, the Sigma linter (cheap fail-fast
stage before any containers spin up), Docker orchestration of ephemeral
test environments, and the Elasticsearch validation queries that gate the
pipeline. Go's goroutines make "provision two containers, poll both until
healthy, tear down on any failure" straightforward without callback soup.

**Python** owns everything that's data/ML shaped: translating Sigma to an
Elasticsearch query via the official `pySigma` library, generating
statistically realistic synthetic telemetry with Faker, building the MITRE
ATT&CK attack graph in Neo4j, and calling the Anthropic API to draft SOAR
playbooks. This is the ecosystem where those libraries live; reimplementing
pySigma's rule compiler in Go would be pure waste.

The two talk over a simple boundary: the Go API server shells out to
`engine-python/run_pipeline.py` for a given rule + run ID, and reads back
its exit code and JSON report. In a larger deployment this would become a
gRPC or message-queue boundary instead of a subprocess call — the subprocess
approach is intentionally the simplest thing that works for a single-node
CI runner.

## Two different "infrastructure as code" layers, on purpose

- **Ephemeral per-PR test range** (`backend-go/orchestrator/docker.go`):
  created and destroyed via the Docker Engine API directly, because CI runs
  need sub-minute lifecycle and per-run isolation (labeled by `run_id` so a
  crashed job can still be swept up later). Terraform's plan/apply/destroy
  cycle is the wrong tool for "spin up, use for 90 seconds, tear down."
- **Persistent staging/prod stack** (`infra/terraform/`): the long-lived
  Elasticsearch + Neo4j + backend that the dashboard and rule-fire history
  actually run against. This is exactly what Terraform is for.

## The validation contract

A rule only passes if, against the *same* Elasticsearch Query DSL translated
from its Sigma source:

1. `count(attack_index) > 0` — it actually detects the technique it claims to.
2. `count(baseline_index) == 0` — it produces zero false positives against
   24h of synthetic benign traffic across ~40 users / 60 hosts.

Both checks run against the real ephemeral Elasticsearch instance, not a
stub. See `backend-go/validator/elastic.go` and `engine-python/validate.py`
(the Python copy exists so a local `run_pipeline.py` invocation can gate on
exit code without a round trip through the Go API — see README "Design
notes" for the planned consolidation).

## Attack graph construction

`engine-python/attack_simulator.py` reads the `attack.tXXXX` MITRE tags off
the Sigma rule under test, looks them up in `mitre_mappings.json`, and
builds a full kill-chain (`initial-access` → ... → `impact`) in Neo4j,
preferring the rule's own tagged technique for whichever tactic it belongs
to and filling in a plausible surrounding chain for the rest. Each stage
becomes a `(:Stage)-[:NEXT]->(:Stage)` node, linked to synthetic `(:User)`
and `(:Host)` nodes, and is also converted into an Elasticsearch document
timestamped a few minutes apart from its neighbors — so a rule gets
exercised inside a believable incident timeline, not a single isolated
log line.
