# Wraith

**Open-source detection engineering and validation platform**

Wraith is a security engineering platform for developing, translating, testing, validating, and operationalizing detection rules across security environments.

It is not a Python-only detection tool. Wraith combines a **Go control plane**, **Python detection and validation engine**, and **TypeScript/Next.js operations interface**, backed by PostgreSQL, Elasticsearch, and Neo4j.

> **Project status:** Active engineering project
>
> **Current local verification:** 21/21 Python engine tests passing under Python 3.13.3
>
> **Primary goal:** make detection quality measurable before a rule reaches production.

---

## Why Wraith

A detection rule can be syntactically valid and still fail in production.

It may:

- depend on one exact representation of attacker behavior
- break when telemetry schemas change
- behave differently after translation to another SIEM
- detect an attack but also match large amounts of benign activity
- fail against simple adversarial mutations
- provide incomplete ATT&CK coverage
- change without a clear provenance record

Wraith treats a detection as an engineering artifact that should be **analyzed, executed, challenged, measured, and traceable**.

The core workflow is:

```text
                         Detection Rule
                              │
                              ▼
                       Rule Analysis
                              │
                ┌─────────────┼─────────────┐
                ▼             ▼             ▼
          Sigma Translation  Diff       Fingerprint
                │
                ▼
          Attack Simulation
                │
                ▼
        Detection Validation
                │
        ┌───────┴────────┐
        ▼                ▼
   Benign Baseline   Attack Telemetry
        │                │
        └───────┬────────┘
                ▼
         Mutation Testing
                │
                ▼
       Robustness Analysis
                │
                ▼
        Quality Assessment
                │
                ▼
        Provenance / Audit
                │
                ▼
       Approval / Deployment
```

The objective is not to produce a decorative security dashboard. The platform is designed to connect **detection content to executable validation evidence**.

---

# Architecture

Wraith is intentionally split into components with different responsibilities.

```text
                              WRAITH
              Detection Engineering & Validation

 ┌───────────────────────┐     ┌────────────────────────┐
 │   TypeScript / Next.js│     │      Go Control Plane   │
 │                       │────▶│                        │
 │ Operations Dashboard  │ API │ Auth / RBAC             │
 │ Run Visibility        │     │ Orchestration           │
 │ Rule Workspace        │     │ Persistence / API       │
 └───────────────────────┘     └────────────┬───────────┘
                                           │
                                           ▼
                              ┌────────────────────────┐
                              │    Python Engine        │
                              │                        │
                              │ Sigma Translation      │
                              │ Detection Analysis     │
                              │ Attack Simulation       │
                              │ Mutation Testing       │
                              │ Robustness / Quality   │
                              └────────────┬───────────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    ▼                      ▼                      ▼
             PostgreSQL             Elasticsearch              Neo4j
             State / Audit             Telemetry          Attack / ATT&CK
                    │                      │                      │
                    └──────────────────────┼──────────────────────┘
                                           ▼
                              Validation Evidence
                                           │
                                           ▼
                               Attestation / Gates
```

### Component responsibilities

| Component | Responsibility |
| --- | --- |
| **Go** | API, control plane, authentication/RBAC, orchestration, persistence-facing services, approval/deployment workflow |
| **Python** | Detection analysis, Sigma translation, simulation, mutation testing, robustness and quality analysis |
| **TypeScript / Next.js** | Operations dashboard, run visibility, platform navigation, detection-engineering UI |
| **PostgreSQL** | Persistent run state and audit-oriented data |
| **Elasticsearch** | Synthetic telemetry and detection-validation queries |
| **Neo4j** | Attack-chain and relationship data used by the validation pipeline |
| **GitHub Actions** | CI/CD and security automation |

This separation allows the detection engine to evolve independently from the control plane and user interface.

---

# Detection Engineering Lifecycle

Wraith is organized around a repeatable detection-engineering lifecycle:

```text
Author
  ↓
Analyze
  ↓
Translate
  ↓
Simulate
  ↓
Validate
  ↓
Mutate
  ↓
Measure
  ↓
Attest
  ↓
Approve
  ↓
Deploy
```

Each stage produces evidence or an explicit result rather than assuming that a successful previous stage proves downstream effectiveness.

---

# Core Capabilities

## Detection Analysis

Wraith parses and analyzes detection content and exposes properties that can be used by the validation pipeline.

Current detection-engineering functionality includes:

- rule parsing
- rule comparison
- deterministic rule fingerprinting
- rule-diff analysis
- ATT&CK technique extraction
- metadata validation
- mutation generation
- mutation scoring
- robustness analysis
- schema-drift analysis
- event correlation
- quality scoring
- compliance mapping

The project distinguishes local implementation from externally validated integrations.

---

## Sigma Translation

Wraith uses Sigma and pySigma to keep detection logic independent from a particular SIEM query language.

The current translation layer supports:

| Backend | Output |
| --- | --- |
| Elasticsearch | Elasticsearch query DSL |
| OpenSearch | OpenSearch query format |
| Splunk | SPL |
| Microsoft Kusto | KQL |
| CrowdStrike | LogScale |
| Grafana Loki | LogQL |

The translation layer is separated from detection logic so the same detection can be evaluated against different backend representations.

Local tests verify supported backend output and selected backend-specific expectations. Live third-party deployment is not implied by those tests.

---

## Attack Simulation

Wraith can construct representative attack chains from detection metadata and ATT&CK techniques.

The validation pipeline can associate simulated stages with:

- ATT&CK technique IDs
- attack tactics
- synthetic users
- synthetic hosts
- generated telemetry
- ordered attack stages

The current local pipeline has been exercised with a seven-stage synthetic Windows attack chain and indexed the resulting telemetry into Elasticsearch.

The simulation is intended for **controlled validation**, not real-world attack execution against external systems.

---

## Detection Validation

Wraith validates detections against both attack telemetry and benign baseline telemetry.

A validation run can measure:

```text
Attack telemetry
      │
      ├── Did the rule fire?
      └── How many hits?

Benign baseline
      │
      ├── Did the rule fire?
      └── How many baseline documents were scanned?
```

For the current verified local example, a run successfully detected the simulated attack while producing zero hits across a 12,000-event synthetic benign baseline.

This is a **test-environment result**, not a claim of production accuracy.

---

## Mutation Testing

A detection should not be evaluated only against the exact event it was written for.

Wraith generates controlled variations of attacker behavior and measures whether the detection continues to fire.

Example mutation dimensions include:

- case variation
- whitespace variation
- PowerShell parameter aliases
- command-line representation changes
- structural variations

Conceptually:

```text
Original behavior
       │
       ▼
 Detection Rule
       │
       ├── Original
       ├── Case variation
       ├── Whitespace variation
       ├── Parameter alias
       └── Other controlled mutations
                    │
                    ▼
             Detection Results
                    │
                    ▼
             Robustness Measure
```

The goal is to expose brittle detections and provide evidence for where the rule can be improved.

---

## Robustness Analysis

Mutation results are converted into a measurable robustness result.

In a current local validation run:

```text
31 mutated variants tested
11 variants detected
20 variants not detected
35.48% mutation detection rate
```

The missed variants are retained as useful engineering evidence. A low mutation detection rate is not hidden behind a passing overall pipeline verdict.

---

## Rule Diff and Fingerprinting

Detection changes need to be observable and traceable.

Wraith provides deterministic fingerprinting and rule-diff functionality for detection content.

This supports workflows such as:

```text
Rule Version A
      │
      ▼
Fingerprint A
      │
      │ change
      ▼
Rule Version B
      │
      ▼
Fingerprint B
```

This is useful for CI/CD, review workflows, provenance, and deployment gates.

---

## ATT&CK Mapping

Wraith extracts ATT&CK techniques from detection metadata and uses those techniques during attack-chain construction and coverage analysis.

The project also includes ATT&CK Navigator-compatible output functionality.

The conceptual flow is:

```text
Detection Rules
      │
      ▼
Technique Extraction
      │
      ▼
ATT&CK IDs
      │
      ├── Coverage analysis
      ├── Attack simulation
      └── Navigator representation
```

---

## Schema Drift

Detection logic is dependent on telemetry schemas.

Wraith includes schema-drift analysis for changes such as:

- removed fields
- renamed fields
- type changes
- structural changes
- vendor-specific field differences

The purpose is to identify detection failures caused by telemetry changes before those changes silently reach production.

---

## Event Correlation

Wraith includes correlation functionality for relating security-relevant events and sequences.

A simplified example is:

```text
Process Creation
       │
       ▼
Encoded PowerShell
       │
       ▼
Network Activity
       │
       ▼
Credential Access
```

Correlation provides context that is not available from an isolated event.

---

## Quality Scoring

Wraith includes an explainable detection-quality scoring component.

The score is based on measurable components rather than being presented as an unexplained AI judgment.

A current local validation run produced:

```text
Behavioral component       40
False-positive component   20
Robustness component       7.10
Available weight           80
Score                      83.87
Rating                     acceptable
```

The score is intended as an engineering signal. It should not be interpreted as a universal measure of detection quality across environments.

---

## Compliance Mapping

Compliance mappings are represented as data rather than being hard-coded into the detection engine.

An example mapping is provided under:

```text
engine-python/compliance_mappings.example.json
```

Organizations can maintain mappings appropriate to their own requirements.

---

# Control Plane

The Go service provides the Wraith API and control-plane functionality.

The API includes authenticated operations for:

- validation-run visibility
- run details and reports
- attestations
- audit data
- rule linting
- approval workflow
- deployment workflow
- rule inspection

Authentication and role-based access control are enforced on protected API routes.

The control plane is deliberately separate from the Python engine so the frontend does not directly manage database, Elasticsearch, Neo4j, or engine credentials.

---

# Operations Dashboard

Wraith includes a TypeScript/Next.js operations interface.

The dashboard is intended to expose the actual Wraith control plane rather than behave as a generic SOC mockup.

Current UI surfaces include:

- platform introduction
- architecture overview
- validation-run overview
- run detail
- pipeline activity
- capability navigation
- controlled playground entry point
- API-key configuration
- light/dark theme support

The dashboard reads validation-run state from the Wraith API.

Planned workspaces are explicitly identified as planned rather than represented as completed functionality.

---

# Data and Infrastructure

The local Wraith stack uses real infrastructure components:

```text
┌───────────────────────────────────────────────┐
│                 Docker Compose                │
│                                               │
│  ┌─────────┐   ┌──────────────┐               │
│  │ Go API  │   │ Next.js UI   │               │
│  └────┬────┘   └──────────────┘               │
│       │                                        │
│  ┌────┴──────┬─────────────┬───────────────┐  │
│  │ PostgreSQL│ Elasticsearch│ Neo4j         │  │
│  └───────────┴─────────────┴───────────────┘  │
└───────────────────────────────────────────────┘
```

The stack is designed so a local validation run can exercise the same major component boundaries used by the platform architecture.

---

# Repository Structure

```text
Wraith/
├── backend-go/                 # Go control plane and API
│   ├── api/
│   ├── auth/
│   ├── config/
│   ├── deploy/
│   ├── linter/
│   ├── metrics/
│   ├── notify/
│   ├── orchestrator/
│   └── main.go
│
├── engine-python/              # Detection and validation engine
│   ├── attack_simulator.py
│   ├── attack_navigator.py
│   ├── compliance_map.py
│   ├── correlation.py
│   ├── mutation_runner.py
│   ├── mutation_testing.py
│   ├── quality_score.py
│   ├── rule_diff.py
│   ├── run_pipeline.py
│   ├── schema_drift.py
│   ├── sigma_to_es.py
│   ├── sigma_translate.py
│   └── requirements*.txt
│
├── frontend/                   # TypeScript / Next.js UI
│   ├── components/
│   ├── lib/
│   ├── pages/
│   └── styles.css
│
├── rules/                      # Sigma detection content
│
├── tests/                      # Python engine tests
│
├── integrations/               # External integration foundations
│   └── github-app/
│
├── docs/                       # Architecture and engineering documentation
│
├── .github/workflows/          # CI and security automation
│
├── docker-compose.yml          # Local platform stack
├── Makefile
├── action.yml
├── LICENSE
├── SECURITY.md
├── CONTRIBUTING.md
└── README.md
```

---

# Quick Start

Wraith is designed to be run as a local multi-service stack.

## Prerequisites

- Docker and Docker Compose
- Git
- Python 3.13 for engine development and tests
- Go toolchain for backend development
- Node.js for frontend development

## Start the local stack

```bash
git clone https://github.com/locallhosts/Wraith.git
cd Wraith

cp .env.example .env

docker compose up --build
```

The local stack exposes the API and frontend according to the ports configured in `docker-compose.yml`.

For engine-only development, create an isolated Python environment:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r engine-python/requirements.txt
python -m pip install -r engine-python/requirements-siem.txt
```

The Python environment is a **development path for the engine**, not the definition of the complete Wraith platform.

---

# Running a Validation Pipeline

A local pipeline can be executed against the local Elasticsearch and Neo4j services.

Example:

```bash
python engine-python/run_pipeline.py \
  --rule rules/suspicious_powershell_encodedcommand.yml \
  --run-id local-validation-004 \
  --target-os windows \
  --es-addr http://localhost:9200 \
  --neo4j-addr bolt://localhost:7687 \
  --neo4j-user neo4j \
  --neo4j-pass wraith-test-pw
```

The pipeline can produce:

- Sigma translation output
- synthetic attack telemetry
- ATT&CK attack-chain data
- Elasticsearch validation results
- benign baseline measurements
- adversarial mutation results
- robustness measurements
- quality scoring
- validation reports
- provenance/attestation data
- optional automation output

External provider integrations remain provider-dependent and are not treated as proof of deterministic local validation.

---

# Testing

## Python engine

```bash
python -m pytest tests/ -v
```

Current local verification:

```text
Python: 3.13.3
pytest: 9.1.1

21 tests collected
21 passed
0 failed
```

The verified Python test groups include:

- advanced detection engineering
- deterministic mutation generation
- mutation scoring
- rule-diff fingerprints
- ATT&CK Navigator extraction
- schema-drift detection
- correlation
- explainable quality scoring
- compliance mapping
- attack simulation
- robustness testing
- Sigma translation

## Go backend

Backend tests can be run from the repository with:

```bash
go test ./backend-go/...
```

## Frontend

Frontend validation should be performed with the project's Node/Next.js build and lint commands defined in `frontend/package.json`.

---

# Verification Status

Wraith intentionally separates **implemented code**, **local verification**, and **external/provider-dependent behavior**.

| Capability | Status |
| --- | --- |
| Detection analysis | Implemented |
| Mutation testing | Implemented and locally tested |
| Robustness testing | Implemented and locally tested |
| Rule diff / fingerprinting | Implemented and locally tested |
| ATT&CK technique extraction | Implemented and locally tested |
| ATT&CK Navigator output | Implemented and locally tested |
| Schema-drift analysis | Implemented and locally tested |
| Correlation | Implemented and locally tested |
| Quality scoring | Implemented and locally tested |
| Compliance mapping | Implemented and locally tested |
| Attack-chain simulation | Implemented and locally tested |
| Sigma translation | Implemented and locally tested |
| Elasticsearch translation | Locally verified |
| OpenSearch translation | Locally verified |
| Splunk translation | Locally verified |
| Kusto translation | Locally verified |
| CrowdStrike translation | Locally verified |
| Loki translation | Locally verified |
| Go control plane | Implemented |
| API authentication / RBAC | Implemented |
| PostgreSQL persistence | Implemented |
| Elasticsearch validation backend | Locally validated |
| Neo4j attack-chain persistence | Locally validated |
| Operations dashboard | Implemented |
| GitHub integration foundation | Implemented / foundation |
| SOAR / external AI integrations | Provider-dependent |
| Production deployment | Environment-dependent |

The table is intentionally conservative. Passing a local unit test does not imply production readiness, and an integration foundation is not described as a fully deployed service.

---

# CI/CD and Security

The repository includes GitHub Actions for detection-oriented CI and security checks.

The intended workflow is:

```text
Pull Request
     │
     ▼
Detection Changes
     │
     ▼
Rule / Code Validation
     │
     ▼
Security Checks
     │
     ▼
Detection Validation
     │
     ▼
Review / Approval
```

Security automation includes dependency and package checks, static analysis, secret scanning, container scanning, and detection-oriented validation where configured.

Some GitHub security features depend on repository-level settings and are therefore documented separately from the local application stack.

---

# Security Model

Wraith is designed around several security boundaries:

- authenticated API access
- role-based authorization
- controlled rule inspection
- separation between frontend and backend credentials
- audit-oriented persistence
- cryptographic provenance/attestation support
- approval gates for deployment workflows
- synthetic attack telemetry for local validation

Attack simulation is intended to generate controlled telemetry rather than execute arbitrary attacks against third-party infrastructure.

See [`SECURITY.md`](SECURITY.md) and the security documentation under [`docs/`](docs/) for additional details.

---

# Documentation

Detailed engineering documentation is available in `docs/`.

| Document | Description |
| --- | --- |
| [Architecture](docs/ARCHITECTURE.md) | System architecture and component relationships |
| [API](docs/API.md) | API and interface documentation |
| [CI Security](docs/CI_SECURITY.md) | Detection validation in CI/CD |
| [Design Decisions](docs/DESIGN_DECISIONS.md) | Important engineering decisions |
| [Deployment](docs/DEPLOYMENT.md) | Deployment considerations |
| [Detection Engineering](docs/DETECTION_ENGINEERING.md) | Detection development workflow |
| [Enterprise](docs/ENTERPRISE.md) | Enterprise deployment considerations |
| [Features](docs/FEATURES.md) | Feature-level documentation |
| [Operations](docs/OPERATIONS.md) | Operational considerations |
| [Provenance](docs/PROVENANCE.md) | Detection and data provenance |
| [Security Model](docs/SECURITY_MODEL.md) | Security boundaries and controls |
| [Testing](docs/TESTING.md) | Testing strategy |
| [Threat Model](docs/THREAT_MODEL.md) | Threat model and security assumptions |
| [Using as Action](docs/USING_AS_ACTION.md) | GitHub/automation usage |
| [Validation](docs/VALIDATION.md) | Detection validation methodology |

---

# Open Source

Wraith is developed as an open-source security engineering project.

The project is intended to make detection-engineering methodology inspectable and reproducible rather than hiding validation logic behind a hosted service.

Contributions should favor:

- reproducible tests
- explicit security assumptions
- measurable validation results
- small, reviewable changes
- real integrations over simulated interfaces
- clear separation between implemented and planned capabilities

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for contribution guidance.

---

# License

Wraith is licensed under the terms in [`LICENSE`](LICENSE).

---

# Project Direction

The long-term direction is to provide one engineering surface connecting:

```text
Detection Content
      │
      ▼
Engineering Analysis
      │
      ▼
Executable Validation
      │
      ▼
Adversarial Testing
      │
      ▼
Evidence
      │
      ▼
Governance
      │
      ▼
Deployment
```

The guiding principle is simple:

> **A detection should be measurable before it is trusted.**
