# Wraith

**Detection engineering, validation, and security assurance — from rule authoring to deployment evidence.**

Wraith is an open-source security engineering platform for building, translating, testing, and validating detection rules before they are promoted into production. It combines a Go control plane, a Python detection-validation engine, and a TypeScript/Next.js operations interface with data services for telemetry, state, and attack relationships.

The central idea is simple: **a detection rule should be treated as tested software, not just a query that parses.**

- **Repository:** [locallhosts/Wraith](https://github.com/locallhosts/Wraith)
- **Documentation:** [docs/](docs/)
- **Security reporting:** [SECURITY.md](SECURITY.md)
- **License:** See [LICENSE](LICENSE)

---

## Contents

- [Why Wraith](#why-wraith)
- [Capabilities](#capabilities)
- [How the pipeline works](#how-the-pipeline-works)
- [Architecture](#architecture)
- [Security engineering](#security-engineering)
- [Quick start](#quick-start)
- [Run the detection engine](#run-the-detection-engine)
- [Testing and validation](#testing-and-validation)
- [Repository layout](#repository-layout)
- [Documentation](#documentation)
- [Project scope and limitations](#project-scope-and-limitations)
- [Contributing](#contributing)

## Why Wraith

A detection can be syntactically correct and still be unreliable. It may miss a small variation in attacker behavior, produce excessive false positives, break when telemetry fields change, or behave differently after translation to another SIEM.

Wraith makes these risks part of the engineering workflow by connecting detection content to repeatable tests and inspectable evidence.

The platform is designed to help teams answer questions such as:

- Does the rule detect representative attack telemetry?
- Does it also fire on a benign baseline?
- How resilient is it to controlled changes in command or event representation?
- Which ATT&CK techniques does it cover?
- Did the rule change, and can that change be traced?
- Is there sufficient validation evidence to support review and deployment?

## Capabilities

### Detection analysis and change tracking

- Parse and inspect Sigma detection content.
- Compare rule revisions and generate deterministic fingerprints.
- Validate rule metadata and extract ATT&CK technique references.
- Analyze telemetry schema drift, including field and type changes.
- Produce explainable quality signals from measurable validation inputs.

### Multi-backend Sigma translation

The translation layer is designed to map Sigma rules to different query languages, including:

| Target | Query representation |
| --- | --- |
| Elasticsearch | Elasticsearch query DSL |
| OpenSearch | OpenSearch query format |
| Splunk | SPL |
| Microsoft Sentinel / Kusto | KQL |
| CrowdStrike LogScale | LogScale query language |
| Grafana Loki | LogQL |

Backend availability and translation behavior depend on the installed pySigma plugins and local configuration. Successful translation does not by itself prove that a rule has been tested against a live vendor environment.

### Controlled attack simulation and validation

- Build representative synthetic attack sequences associated with MITRE ATT&CK techniques.
- Generate telemetry for controlled validation runs.
- Evaluate detections against attack data and a benign baseline.
- Record hits and misses so false negatives and noisy rules remain visible.
- Keep simulation scoped to authorized, controlled test environments.

### Mutation and robustness testing

Wraith can generate controlled variations of event and command representations—such as case, whitespace, parameter aliases, and structural changes—and evaluate whether detections continue to fire. The results help identify brittle rules and provide concrete evidence for improvement.

### ATT&CK coverage and correlation

- Extract technique IDs from detection metadata.
- Associate simulation stages with tactics and techniques.
- Relate security events into ordered sequences.
- Produce ATT&CK Navigator-compatible output.

### Control plane and operations interface

The Go service provides API and orchestration boundaries for validation runs, run details, reports, rule inspection, audit data, attestations, approvals, and deployment workflows. The Next.js interface provides operational visibility into the platform and its validation activity.

### Provenance and deployment controls

Wraith includes provenance/attestation workflows and deployment-related controls intended to make changes reviewable and traceable. Approval and deployment decisions should be based on explicit policy and validation evidence rather than a dashboard status alone.

## How the pipeline works

The exact stages depend on configuration and enabled integrations. The intended lifecycle is:

```text
Detection Rule (Sigma)
        |
        v
Parse, Lint, and Fingerprint
        |
        v
Translate to Target Query
        |
        v
Construct Controlled Attack Telemetry
        |
        v
Validate Against Attack + Benign Baseline
        |
        v
Mutation / Robustness Testing
        |
        v
Quality Assessment and ATT&CK Coverage
        |
        v
Report, Audit, and Provenance Evidence
        |
        v
Human Review / Policy Gate
        |
        v
Deployment Workflow
```

Each stage should produce a result that can be inspected. Passing one stage is not proof that every downstream integration or production environment is healthy.

## Architecture

```text
                     WRAITH PLATFORM

  TypeScript / Next.js UI  <---->  Go API / Control Plane
                                      |
                                      v
                              Python Detection Engine
                                      |
                    +-----------------+------------------+
                    |                 |                  |
                    v                 v                  v
               PostgreSQL       Elasticsearch          Neo4j
               State / audit    Telemetry / queries   Relationships
                                     |                 /
                     +----------------+----------------+
                                      |
                                      v
                          Validation reports and
                           provenance / audit data
```

| Component | Responsibility |
| --- | --- |
| **Go — `backend-go/`** | API, authentication and authorization, orchestration, run management, and deployment-facing workflows |
| **Python — `engine-python/`** | Sigma translation, detection analysis, simulation, mutation testing, correlation, schema-drift analysis, and quality scoring |
| **TypeScript / Next.js — `frontend/`** | Operations UI, run visibility, platform navigation, and detection-engineering workflows |
| **PostgreSQL** | Persistent platform state and audit-oriented records |
| **Elasticsearch** | Synthetic telemetry indexing and detection queries |
| **Neo4j** | Attack-chain and relationship data |
| **GitHub Actions** | Continuous integration and security automation |

For the detailed component boundaries and design rationale, see [Architecture](docs/ARCHITECTURE.md) and [Design Decisions](docs/DESIGN_DECISIONS.md).

## Security engineering

Security is part of the platform implementation, not just the use case. The repository includes work on:

- **Threat modelling and security architecture** to document trust boundaries and risks.
- **Authentication and route-level RBAC** for protected control-plane operations.
- **Input and path hardening** for backend and webhook-related flows.
- **Startup and readiness behavior** intended to fail safely and avoid exposing sensitive error details.
- **Isolation and idempotency** for in-memory state, pipeline jobs, retries, and webhook run identifiers.
- **Rate-limit key isolation and trusted-proxy configuration** to reduce cross-client interference and unsafe proxy assumptions.
- **Deployment safety controls**, including compare-and-swap style safeguards.
- **CI and security scans** to catch regressions during development.

Relevant references: [Threat Model](docs/THREAT_MODEL.md), [Security Architecture](docs/SECURITY_ARCHITECTURE.md), [Security Model](docs/SECURITY_MODEL.md), [CI Security](docs/CI_SECURITY.md), and [Provenance](docs/PROVENANCE.md).

These controls reduce specific risks but do not constitute a claim that Wraith has undergone an independent security audit or is production-secure in every deployment. Review the threat model, configure the platform for your environment, and validate the controls before exposing services.

## Quick start

### Prerequisites

- Git
- Docker with the Docker Compose plugin
- Python 3.13 for engine development and tests
- A supported Go toolchain for backend development
- Node.js and npm for frontend development

### Start the local stack

```bash
git clone https://github.com/locallhosts/Wraith.git
cd Wraith

# Configure local environment values before starting services.
cp .env.example .env

docker compose up --build
```

Review `docker-compose.yml` and `.env.example` for the configured services, ports, and required variables. Replace development credentials with unique local values; do not reuse test secrets in shared or production environments.

### Develop the Python engine

```bash
python3 -m venv .venv
source .venv/bin/activate

python -m pip install --upgrade pip
python -m pip install -r engine-python/requirements.txt
python -m pip install -r engine-python/requirements-siem.txt
```

If your environment uses a different Python executable, substitute it for `python3`. The engine-only setup is for developing and testing the Python component; it does not start the complete platform or its backing services.

## Run the detection engine

The repository includes a pipeline entry point at `engine-python/run_pipeline.py`. A local invocation can look like this when Elasticsearch and Neo4j are running and the rule path and credentials match your configuration:

```bash
python engine-python/run_pipeline.py \
  --rule rules/suspicious_powershell_encodedcommand.yml \
  --run-id local-validation-001 \
  --target-os windows \
  --es-addr http://localhost:9200 \
  --neo4j-addr bolt://localhost:7687 \
  --neo4j-user neo4j \
  --neo4j-pass "$NEO4J_PASSWORD"
```

Set `NEO4J_PASSWORD` in your shell before running the command, and confirm the actual rule filename and service settings in your checkout. Do not commit credentials to source control.

Depending on configuration, a run may produce translated query output, synthetic telemetry, validation results, benign-baseline measurements, mutation results, quality signals, reports, and provenance data. See [Validation](docs/VALIDATION.md) and [Detection Engineering](docs/DETECTION_ENGINEERING.md) for the workflow and interpretation guidance.

## Testing and validation

### Python tests

From the repository root:

```bash
python -m pytest tests/ -v
```

For the broader verification process and component-specific checks, follow [Testing](docs/TESTING.md). Run the tests in the environment supported by the current dependency files.

### Go backend and frontend

Use the component-specific manifests and scripts in `backend-go/` and `frontend/` to run formatting, unit tests, type checks, and builds. The exact scripts may vary as the project evolves; inspect the relevant `Makefile`, `go.mod`, and frontend `package.json` before running commands.

### CI and security checks

Open the [GitHub Actions runs](https://github.com/locallhosts/Wraith/actions) to review the latest results. A passing CI run is evidence for the checks that actually ran—it is not a substitute for integration testing, deployment validation, or a security review.

## Repository layout

```text
Wraith/
├── backend-go/             # Go API and control plane
├── engine-python/          # Detection engineering and validation engine
├── frontend/               # TypeScript / Next.js operations UI
├── rules/                  # Sigma detection rules
├── tests/                  # Automated tests
├── integrations/           # Integration foundations
├── docs/                   # Architecture, security, operations, and API docs
├── .github/workflows/      # CI and security automation
├── docker-compose.yml      # Local multi-service environment
├── Makefile                # Common development tasks
├── action.yml              # GitHub Action metadata
├── SECURITY.md             # Vulnerability reporting policy
├── CONTRIBUTING.md         # Contribution guidance
└── README.md
```

## Documentation

| Guide | Description |
| --- | --- |
| [Architecture](docs/ARCHITECTURE.md) | Components and system boundaries |
| [Detection Engineering](docs/DETECTION_ENGINEERING.md) | Detection workflow and engineering concepts |
| [Validation](docs/VALIDATION.md) | Validation pipeline and evidence |
| [Testing](docs/TESTING.md) | Test strategy and verification |
| [API](docs/API.md) | API usage and endpoint notes |
| [OpenAPI specification](docs/openapi.yaml) | Machine-readable API definition |
| [Threat Model](docs/THREAT_MODEL.md) | Threats, trust boundaries, and mitigations |
| [Security Architecture](docs/SECURITY_ARCHITECTURE.md) | Security boundaries and controls |
| [Security Model](docs/SECURITY_MODEL.md) | Authentication and authorization model |
| [CI Security](docs/CI_SECURITY.md) | CI and security automation |
| [Deployment](docs/DEPLOYMENT.md) | Deployment configuration and considerations |
| [Operations](docs/OPERATIONS.md) | Operational guidance |
| [Enterprise considerations](docs/ENTERPRISE.md) | Enterprise deployment and control considerations |
| [Provenance](docs/PROVENANCE.md) | Evidence and attestation concepts |
| [Feature guide](docs/FEATURES.md) | Feature overview |
| [Using Wraith as a GitHub Action](docs/USING_AS_ACTION.md) | Action usage |
| [Security policy](SECURITY.md) | Reporting security vulnerabilities |
| [Contributing](CONTRIBUTING.md) | How to contribute |

## Project scope and limitations

Wraith is an actively engineered open-source project. Capabilities described here refer to repository implementation and documented workflows; integration maturity can differ by component and provider.

- Synthetic validation results should not be interpreted as production detection rates.
- Translation support does not guarantee semantic equivalence across SIEM products.
- External vendor integrations require their own credentials, configuration, and live-environment verification.
- A quality score is an engineering signal, not a universal measure of security effectiveness.
- Deployment gates should be configured and tested against your organization's own change-management requirements.
- Do not run simulations or tests against systems you do not own or have explicit permission to assess.

## Contributing

Contributions are welcome. Before opening a pull request:

1. Read [CONTRIBUTING.md](CONTRIBUTING.md) and the relevant architecture documentation.
2. Keep changes focused and document security-sensitive behavior.
3. Add or update tests for changed behavior.
4. Run the relevant local checks and include their results in the pull request.
5. Never commit secrets, real customer telemetry, or sensitive production data.

Please report vulnerabilities according to [SECURITY.md](SECURITY.md) rather than disclosing them in a public issue.

---

**Wraith — engineer detections with evidence, test them against controlled behavior, and make security changes traceable.**
