# WRAITH

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)
[![Security Policy](https://img.shields.io/badge/security-policy-blue.svg)](SECURITY.md)

**Predictive detection engineering: every Sigma rule fights a synthetic attacker before it reaches your SIEM.**

Most detection-as-code pipelines answer one question: *does this rule fire?*
WRAITH answers three: does it fire on a **realistic, multi-stage attack**,
does it stay **silent on 24 hours of benign enterprise noise**, and — if it
passes — what should the **response playbook** look like?

## What this actually does for you, in plain terms

If you write detection rules for a SIEM (Sigma, Elastic Security, Splunk,
whatever), you already know the two failure modes that quietly hurt a SOC:

- **A rule that looks right but never actually fires** when the real
  attack technique happens — nobody finds out until an incident.
- **A rule that fires constantly on completely normal traffic** — the SOC
  mutes it within a month, and now it's dead weight that *looks* like
  coverage but isn't.

WRAITH catches both **before the rule ever merges**, automatically, on
every pull request: it simulates the actual attack technique your rule
claims to catch, floods the system with a day of realistic fake "normal"
traffic, and only lets the rule through if it fires on the attack and
stays silent on the noise. It also tests the rule against attacker
variations (different casing, different PowerShell parameter shortcuts,
etc.) instead of just the one textbook payload, and — if the rule
passes — drafts an incident-response playbook for it automatically.

**Who benefits and how:**

- **Detection engineers / SOC teams**: stop shipping rules on faith. Every
  rule gets a pass/fail grade backed by real (synthetic) attack data
  before it reaches production, plus a robustness score showing how
  easily it could be evaded.
- **Security engineers building a portfolio**: this is a complete,
  working, testable system spanning Go, Python, Terraform, Kubernetes,
  and a real dashboard — not a toy script. Everything claimed in this
  README has actually been built and tested (see "Verification status"
  below).
- **Anyone learning detection engineering or DevSecOps**: the whole
  pipeline is small enough to read end to end, and each stage
  (`backend-go/linter`, `engine-python/attack_simulator.py`, etc.) is a
  self-contained, commented example of one real technique — Sigma rule
  validation, MITRE ATT&CK-mapped attack simulation, synthetic baseline
  generation, cryptographic supply-chain attestation.

You don't need a real SOC or real production data to run this and see it
work — every attack and every "normal" log line it tests against is
synthetically generated, safely, on your own machine.

## Use this in your own repo — the zero-setup path

You don't need to fork or run any of this yourself to get value from it.
The Sigma rule linter is published as a **reusable GitHub Action** — drop
this into any repo with a `rules/` directory of Sigma rules:

```yaml
# .github/workflows/lint-detections.yml
name: Lint detection rules
on: pull_request
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: locallhosts/Wraith@v1
        with:
          rules-path: rules
          fail-on-warning: 'false'   # set 'true' to also block on missing MITRE tags, undocumented false positives, etc.
```

That's it — no Elasticsearch, no Neo4j, no API keys. Every PR touching a
rule gets inline annotations directly on the changed lines (missing
required fields, missing MITRE ATT&CK tags, undocumented false
positives) before a human ever has to eyeball it. See `action.yml` for
the full input/output contract.

For the complete pipeline (attack simulation, false-positive testing
against synthetic baseline traffic, adversarial robustness scoring, SOAR
playbook drafting) in your own repo, copy `.github/workflows/detection-ci.yml`
as a starting point — it needs Elasticsearch/Neo4j service containers and
(optionally) an `ANTHROPIC_API_KEY` secret, documented inline.

## Not an Elastic shop? Multiple SIEMs are supported

`engine-python/sigma_translate.py` translates a Sigma rule into the native
query language of six different platforms, not just Elasticsearch:

```bash
python sigma_translate.py --rule ../rules/your_rule.yml --backend splunk
python sigma_translate.py --rule ../rules/your_rule.yml --backend kusto        # Microsoft Sentinel / Defender
python sigma_translate.py --rule ../rules/your_rule.yml --backend crowdstrike  # LogScale
python sigma_translate.py --rule ../rules/your_rule.yml --backend loki        # Grafana Loki
python sigma_translate.py --rule ../rules/your_rule.yml --backend opensearch
python sigma_translate.py --rule ../rules/your_rule.yml --backend all         # everything at once
```

All six were run against this repo's real sample rule and verified
producing correct native output (see "Verification status" below). Today,
**live CI attack-simulation validation** (does the rule actually fire,
does it stay silent on baseline noise) only runs against
Elasticsearch/OpenSearch, since that's what the ephemeral test
infrastructure in `backend-go/orchestrator` provisions — the other four
backends give you correct translation now, with live validation as a
documented, well-scoped open contribution (see `CONTRIBUTING.md`).

Install the extra backends with `pip install -r requirements.txt -r requirements-siem.txt`.

## Quickstart (Ubuntu 22.04 / 24.04)

This gets the full stack running locally in about 5 minutes. Ubuntu 22.04
and 24.04 are both good choices — they're what GitHub Actions' own runners
use, so anything that works here will also work in CI.

```bash
# 1. Docker + Docker Compose (skip if you already have Docker installed)
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# let your user run docker without sudo (log out/in once after this)
sudo usermod -aG docker $USER
newgrp docker

# 2. Get the code
unzip wraith.zip
cd wraith

# 3. Bring up everything: Postgres, Elasticsearch, Neo4j, the Go API, the dashboard
docker compose up --build
```

That's it — no separate Go/Python/Node install needed for this path, since
Docker builds everything inside containers. Once it's up:

- Dashboard: http://localhost:3000
- API health check: http://localhost:8080/healthz
- Neo4j browser (see the attack graphs): http://localhost:7474 (login `neo4j` / `wraith-test-pw`)

To actually trigger a pipeline run against the sample rule without waiting
for a real GitHub pull request, open a second terminal:

```bash
docker compose exec backend wraith lint /app/rules
```

**If you want to develop on it (edit code, not just run it)**, you'll also want:

```bash
sudo apt-get install -y golang-go python3-pip python3-venv nodejs npm
```

— covered in more detail in "Running it locally" below.

```
Analyst commits Sigma rule
        │
        ▼
 GitHub Actions triggers
        │
        ▼
 ┌────────────────────┐
 │ Go rule linter       │  syntax + Sigma schema + MITRE tag checks, fails fast
 └─────────┬───────────┘
           ▼
 ┌────────────────────┐
 │ Ephemeral test range  │  Docker-provisioned Elasticsearch + Neo4j (Go SDK)
 └─────────┬───────────┘
           ▼
 ┌────────────────────┐
 │ Baseline generator     │  24h of synthetic benign telemetry -> Elasticsearch
 └─────────┬───────────┘
           ▼
 ┌────────────────────┐
 │ Attack simulator       │  MITRE-mapped multi-stage attack graph in Neo4j,
 │                        │  corresponding telemetry injected into Elasticsearch
 └─────────┬───────────┘
           ▼
 ┌────────────────────┐
 │ Validator               │  rule must fire on attack AND produce zero hits
 │                        │  on baseline, or the pipeline fails the PR
 └─────────┬───────────┘
           ▼
     pass ──┴── fail
      │            │
      ▼            ▼
 ┌──────────┐  PR blocked,
 │ SOAR       │  report posted
 │ playbook   │  as a check
 │ drafted    │  annotation
 │ via LLM,   │
 │ opened as  │
 │ a PR for   │
 │ SOC review │
 └──────────┘
```

**Enterprise features:** API-key auth + RBAC, Postgres-backed persistence,
an append-only audit trail, Prometheus metrics, Slack notifications, and a
human-in-the-loop approval gate before anything reaches production — see
[`docs/ENTERPRISE.md`](docs/ENTERPRISE.md). It also ships two things I
haven't seen in other detection-as-code pipelines: **signed provenance
attestations** (SLSA/supply-chain security concepts applied to detection
content, so a rule can't be quietly loosened after passing CI) and
**adversarial evasion robustness scoring** (testing rules against attacker
variations, not just the one canonical payload).

## Why this exists

Sigma rules ship to production the same way application code used to ship
before CI existed: someone writes it, someone eyeballs it, it goes live. The
two failure modes that actually hurt a SOC are invisible in that workflow:

- **False negatives** — the rule looks right but never actually fires on the
  technique it claims to cover.
- **Alert fatigue** — the rule fires constantly on completely benign traffic
  and gets muted within a month.

WRAITH catches both **before merge**, using real attack telemetry mapped to
MITRE ATT&CK and a real 24-hour baseline of synthetic-but-realistic noise —
not a single hand-picked test log line.

## What's actually in this repo

| Component | Language | Does |
|---|---|---|
| `backend-go/` | Go | GitHub webhook receiver, Sigma linter, Docker-based ephemeral infra orchestration, Elasticsearch validation queries, REST API for the dashboard |
| `engine-python/` | Python | Multi-SIEM Sigma translation (pySigma — Elasticsearch, OpenSearch, Splunk, Sentinel/Defender, CrowdStrike, Loki), synthetic baseline log generation (Faker), MITRE-mapped attack graph construction (Neo4j), SOAR playbook drafting (Anthropic API) + PR creation (GitHub API) |
| `tools/logblast/` | C | High-throughput synthetic log generator (libcurl + pthreads) for realistic-volume performance-overhead testing — see its own README for why C |
| `action.yml` | — | Reusable GitHub Action wrapping the Sigma linter — any repo can use this with zero setup, see above |
| `infra/terraform/` | Terraform | IaC for the persistent staging/prod deployment (backend + Elasticsearch + Neo4j) |
| `frontend/` | Next.js / TypeScript | Live dashboard of pipeline runs, pulling from the real Go API |
| `rules/` | Sigma (YAML) | Example detection rule used to exercise the whole pipeline |
| `.github/workflows/` | GitHub Actions | CI wiring: lint → provision → simulate → validate → SOAR draft, on every PR touching `rules/` |

Everything above talks to a real Elasticsearch, a real Neo4j, and (for the
SOAR step) the real Anthropic and GitHub APIs — there's no mocked data path.

| Enterprise component | Does |
|---|---|
| `backend-go/store/` | Postgres-backed run + audit persistence (in-memory fallback for local dev) |
| `backend-go/auth/` | API-key auth with SHA-256 hashing + 4-tier RBAC |
| `backend-go/provenance/` | Ed25519-signed, tamper-evident test-result attestations |
| `backend-go/deploy/` | Production promotion gate: requires approval + valid attestation |
| `backend-go/metrics/` | Prometheus counters/histograms |
| `backend-go/notify/` | Slack notifications |
| `engine-python/robustness_fuzzer.py` | Adversarial evasion variant generation + scoring |
| `k8s/`, `charts/wraith/` | Kubernetes manifests + Helm chart |
| `.github/workflows/security-scan.yml` | SAST, container scanning, secret scanning, dependency audit |
| `docs/openapi.yaml` | Full API contract |

Want to contribute? See [`CONTRIBUTING.md`](CONTRIBUTING.md) for
well-scoped starting points — the multi-SIEM live-validation gap above
and wiring `logblast` into the main pipeline are both concrete, welcome
first contributions.

## Running it locally (for development — editing the pipeline itself)

The Quickstart above is enough to just *use* WRAITH. This section is for
running individual pipeline stages by hand while developing — you'll want
Go 1.22+, Python 3.11+, and Node 20+ installed natively (not just Docker)
for this.

```bash
# 1. bring up Elasticsearch, Neo4j, Postgres, the Go API, and the dashboard
docker compose up --build

# 2. in another terminal, install the python engine deps if you want to
#    run pipeline stages directly (outside the containerized backend)
cd engine-python
pip install -r requirements.txt

# 3. lint the example rule
cd ../backend-go
go run . lint ../rules

# 4. run the full pipeline against the example rule by hand
cd ../engine-python
python run_pipeline.py \
  --rule ../rules/suspicious_powershell_encodedcommand.yml \
  --run-id local-test-1 \
  --es-addr http://localhost:9200 \
  --neo4j-addr bolt://localhost:7687

cat output/local-test-1/report.json
```

Open http://localhost:3000 for the dashboard and http://localhost:7474 to
browse the generated attack graph directly in Neo4j Browser
(`neo4j` / `wraith-test-pw`).

### Enabling the SOAR playbook + PR step

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export GITHUB_TOKEN=ghp_...
export GITHUB_REPO=locallhosts/Wraith
python run_pipeline.py --rule ../rules/suspicious_powershell_encodedcommand.yml \
  --run-id local-test-2 --es-addr http://localhost:9200 \
  --neo4j-addr bolt://localhost:7687 --open-pr
```

### Running in GitHub Actions

Set repo secrets `ANTHROPIC_API_KEY` (and rely on the built-in
`GITHUB_TOKEN`), then open a PR that touches a file under `rules/`. See
`.github/workflows/detection-ci.yml`.

## Setting up auth and the signing key

```bash
# generate the ed25519 keypair used for provenance attestations
cd backend-go && go run . keygen
# -> prints WRAITH_SIGNING_PUBLIC_KEY (goes on the server) and
#    WRAITH_SIGNING_PRIVATE_KEY (CI secret only, used by `wraith attest`)

# once WRAITH_DATABASE_URL is set, issue an API key for the dashboard/CI
go run . apikey create --label soc-dashboard --role lead
```

The full request lifecycle once a PR opens: webhook → lint → provision →
simulate → validate → robustness fuzz → SOAR draft → CI signs an
attestation (`wraith attest`) → a `lead` reviews in the dashboard and calls
`POST /runs/:id/approve` → `POST /runs/:id/deploy` re-verifies the
attestation against the rule's current content and writes it live.

## Deploying the persistent stack

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for the full walkthrough:
testing locally with kind/minikube before touching a real cluster,
Helm chart usage, what changes for a real managed cluster (registry,
secrets, storage class, ingress), and OS-specific guidance (short version:
Linux or Windows+WSL2 — see that doc for why native Windows causes
friction specifically for this project's bash/Docker tooling).

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars   # fill in real secrets
terraform init
terraform apply
```

This stands up the long-lived backend + Elasticsearch + Neo4j used for
production dashboards and rule-fire history — separate from the ephemeral
per-PR test containers the Go orchestrator spins up during CI, which need
sub-minute create/destroy cycles that Terraform apply/destroy isn't suited
for.

## Design notes / what I'd add next

- ~~Performance overhead profiling~~ — the measurement itself
  (`backend-go/validator/elastic.go`'s `ProfileQuery`) and a way to
  generate realistic index volume to measure it against
  (`tools/logblast/`, verified at ~229K events/sec) both exist now; wiring
  the two together into an automatic CI gate (fail if overhead >5%) is
  the remaining step — see `tools/logblast/README.md`.
- ~~Report viewer~~ — done: `GET /runs/:id/report` and `GET /runs/:id/attestation`
  serve the full pipeline JSON, rendered in `frontend/components/ReportViewer.tsx`
  (attack chain, validation results, robustness score with the specific
  evasion variants that slipped through, SOAR playbook link, attestation
  details). The Neo4j attack graph itself still isn't rendered inline —
  embedding a graph view (Neo4j Bloom or a D3 force-directed graph reading
  from the Neo4j Bolt driver) is the natural next piece.
- ~~Persistent run store~~ — done: `backend-go/store` now has a real
  Postgres-backed implementation (`store.PostgresStore`), integration-tested
  against a live database. The in-memory store remains as an explicit
  dev-only fallback when `WRAITH_DATABASE_URL` is unset.
- **Live CI validation for non-Elasticsearch backends** — `sigma_translate.py`
  correctly translates to Splunk/Sentinel/CrowdStrike/Loki today, but only
  Elasticsearch/OpenSearch get the full attack-simulation + false-positive
  pipeline, because that's the only ephemeral test infrastructure
  `backend-go/orchestrator` currently provisions. Adding a second backend
  (Splunk is the most-requested SIEM and has a well-documented REST API for
  ephemeral indexes) is a well-scoped, high-value open contribution — see
  `CONTRIBUTING.md`.


## Documentation

The `docs/` directory contains the engineering documentation behind the platform.

| Document | Covers |
|---|---|
| `ARCHITECTURE.md` | Components, data flow, Go/Python boundary, validation infrastructure |
| `DETECTION_ENGINEERING.md` | Detection lifecycle, attack testing, baseline testing, robustness, SIEM translation |
| `THREAT_MODEL.md` | Assets, trust boundaries, threats, assumptions, and mitigations |
| `SECURITY_MODEL.md` | Authentication, RBAC, audit, rate limiting, provenance, deployment controls |
| `VALIDATION.md` | Validation contract, evidence levels, backend scope, performance testing |
| `DESIGN_DECISIONS.md` | Major architectural decisions and trade-offs |
| `OPERATIONS.md` | Docker, Kubernetes, Terraform, secrets, observability, scaling |
| `API.md` | Current HTTP API surface and authorization model |
| `CI_SECURITY.md` | Security scanning, CI permissions, dependency exceptions, supply-chain controls |
| `PROVENANCE.md` | Ed25519 attestations, content binding, deployment verification |
| `TESTING.md` | Test strategy and verified engineering evidence |
| `USING_AS_ACTION.md` | Zero-setup reusable GitHub Action |
| `FEATURES.md` | Advanced detection-engineering features and their verification boundaries |
| `DEPLOYMENT.md` | Persistent deployment walkthrough |
| `ENTERPRISE.md` | Enterprise deployment and security details |

## Advanced detection-engineering features

The repository also includes the following implemented feature modules; their current verification scope is documented in `docs/FEATURES.md`.

| Feature | Implementation | Current evidence |
|---|---|---|
| Mutation testing | `engine-python/mutation_testing.py`, `mutation_runner.py` | 8/8 advanced-feature tests |
| Rule diff/regression report | `engine-python/rule_diff.py` | 8/8 advanced-feature tests |
| ATT&CK Navigator export | `engine-python/attack_navigator.py` | 8/8 advanced-feature tests |
| Schema drift detection | `engine-python/schema_drift.py` | 8/8 advanced-feature tests |
| Multi-rule correlation | `engine-python/correlation.py` | 8/8 advanced-feature tests |
| Detection Quality Score | `engine-python/quality_score.py` | 8/8 advanced-feature tests |
| Atomic Red Team import | `engine-python/atomic_red_team.py` | 8/8 advanced-feature tests; import-only by design |
| Compliance mapping | `engine-python/compliance_map.py` | 8/8 advanced-feature tests; organization-owned mappings |
| GitHub App foundation | `integrations/github-app/` | manifest/documentation foundation; production hosting still required |

The hosted playground remains intentionally deferred until the core engineering surface is finished.

## Verification status

Genuinely executed during development (not just read over), most recently:

| Component | How it was verified |
|---|---|
| `backend-go/provenance` (signed attestations) | Compiled + 5/5 unit tests passed, zero external deps |
| `backend-go/store` (Postgres persistence) | Compiled + 2/2 integration tests passed against a **live PostgreSQL 16 instance** — full run/audit/API-key CRUD, nullable timestamp handling, schema-reapply idempotency, re-run 3x with no flakiness |
| `backend-go/linter` (Sigma validation, including `--strict` mode) | Compiled + 4/4 unit tests passed |
| `engine-python/attack_simulator.py` | 3/3 unit tests passed |
| `engine-python/robustness_fuzzer.py` | 5/5 unit tests passed, including proof that a naive rule scores measurably worse than a well-written one |
| `tests/test_advanced_features.py` | 8/8 tests passed for mutation testing, rule diffing, ATT&CK Navigator export, schema drift, correlation, quality scoring, Atomic Red Team import, and compliance mapping |
| `engine-python/sigma_translate.py` (6 SIEM backends) | 5/5 unit tests passed; all 6 backends independently verified producing correct native output against the real sample rule (caught and fixed one real bug — a wrong class name for the CrowdStrike backend — during this verification) |
| `tools/logblast` (C, throughput testing) | Compiled clean with `-Wall -Wextra`, zero warnings; end-to-end tested against a protocol-validating mock endpoint at up to 2,000,000 events with zero malformed documents and zero HTTP errors (~229K events/sec on a single-vCPU test machine); one real bug found and fixed (`curl`'s `Expect: 100-continue` header causing a deadlock against naive HTTP servers) |
| `action.yml` (reusable GitHub Action) | Composite action YAML validated; backing Go CLI logic (`--strict`, GitHub annotation output) covered by the linter test suite above |
| `frontend/` (Next.js dashboard, incl. report viewer + approve/deploy UI) | **Real production build succeeded** (`next build`, Next.js 16.3.5), zero TypeScript errors, zero npm vulnerabilities after a dependency audit caught and fixed a critical Next.js CVE range and a high-severity PostCSS issue |
| Kubernetes manifests (`k8s/`) | Validated with `kubeconform` v0.8.0 in strict mode against real Kubernetes OpenAPI schemas — 8/8 resources valid, 0 errors |

Run the Postgres integration tests yourself:

```bash
export WRAITH_TEST_DATABASE_URL=postgres://wraith:wraith@localhost:5432/wraith?sslmode=disable
cd backend-go && go test -tags=integration ./store/... -v
```

Run the full Python test suite (13 tests, no external services required):

```bash
cd engine-python && pip install -r requirements.txt -r requirements-siem.txt
cd .. && python3 -m pytest tests/ -v
```

## License

MIT — see `LICENSE`.
