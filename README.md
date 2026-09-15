# Wraith

**Detection engineering, validation, and security automation platform**

Wraith is a security engineering platform for developing, translating, testing, validating, and operationalizing detection rules across security environments.

The project focuses on the engineering lifecycle around detections rather than treating a detection rule as a static artifact. Wraith provides capabilities for rule analysis, Sigma translation, mutation testing, robustness testing, ATT&CK mapping, schema-drift analysis, correlation, quality scoring, compliance mapping, attack-chain simulation, and security automation.

> **Project status:** Active engineering project
> **Current local verification:** 21/21 Python tests passing under Python 3.13.3
> **Repository:** https://github.com/locallhosts/Wraith

---

## What Wraith Is

Security detections often fail for reasons that are not visible from the rule itself:

* a rule may be syntactically valid but too fragile
* small attacker-controlled variations may bypass it
* a field may disappear after a schema change
* a detection may translate differently between SIEM backends
* multiple detections may represent the same behavior without being correlated
* a rule may have weak ATT&CK coverage
* compliance mappings may become stale
* a rule may look sophisticated while providing little measurable coverage

Wraith is designed around these problems.

The platform provides an engineering workflow for:

```text
Detection Authoring
       │
       ▼
Rule Analysis
       │
       ├── Sigma Translation
       ├── Mutation Testing
       ├── Robustness Analysis
       ├── ATT&CK Mapping
       ├── Schema Drift
       ├── Correlation
       ├── Quality Scoring
       └── Compliance Mapping
       │
       ▼
Validation
       │
       ├── Unit Tests
       ├── Detection Tests
       ├── Adversarial Mutations
       └── Attack Simulation
       │
       ▼
Operationalization
       │
       ├── SIEM Translation
       ├── CI Security Checks
       ├── SOAR Integration
       └── GitHub Integration
```

---

# Core Capabilities

## Detection Engineering

Wraith provides functionality for analyzing and validating detection rules before they are deployed into production environments.

Capabilities include:

* detection rule parsing
* rule comparison
* rule fingerprinting
* rule-diff analysis
* deterministic mutation generation
* mutation scoring
* robustness testing
* schema-drift analysis
* event correlation
* quality scoring
* ATT&CK technique extraction
* attack-chain construction
* compliance mapping

---

## Sigma Translation

Wraith supports translating Sigma detections into multiple backend query languages through pySigma.

Current supported backends include:

| Backend         | Output                  |
| --------------- | ----------------------- |
| Elasticsearch   | Elasticsearch query DSL |
| OpenSearch      | OpenSearch query format |
| Splunk          | Splunk SPL              |
| Microsoft Kusto | KQL                     |
| CrowdStrike     | CrowdStrike LogScale    |
| Grafana Loki    | LogQL                   |

The translation layer is intentionally separated from the detection logic so that a detection can remain backend-independent while being translated for different environments.

### Current validation

The test suite verifies:

* supported backends produce non-empty output
* Elasticsearch output is valid JSON
* Splunk output contains the expected PowerShell reference
* Kusto output contains the expected encoded-command reference
* unsupported backends raise an error

The current local test environment has all supported optional SIEM backends installed.

---

# Mutation Testing

Wraith includes mutation-testing functionality for security detections.

Instead of only testing whether a rule detects the original event, Wraith can generate controlled mutations of the input behavior and evaluate whether the detection remains effective.

Examples of mutation dimensions include:

* case variation
* command-line variation
* PowerShell parameter aliases
* structural changes
* attacker-controlled representation changes

The purpose is to identify brittle detections that depend on a single representation of attacker behavior.

Conceptually:

```text
Original malicious behavior
          │
          ▼
     Detection Rule
          │
          ├── Original form
          ├── Case variation
          ├── Parameter variation
          ├── Command variation
          └── Structural variation
                    │
                    ▼
             Detection Results
                    │
                    ▼
             Mutation Score
```

A detection that only matches one exact representation can therefore be identified as less robust.

---

# Detection Robustness

Wraith includes robustness analysis designed to identify naive detections.

For example, a rule that only matches:

```text
powershell.exe -EncodedCommand ...
```

may be weaker than a detection capable of recognizing valid PowerShell parameter aliases or equivalent representations.

The robustness tests currently validate behaviors such as:

* case-sensitive detection weaknesses
* case-insensitive matching
* PowerShell parameter aliases
* generated command mutations
* robustness scoring

This is intended to model realistic attacker variation rather than simply testing the original known-good sample.

---

# Rule Diff and Fingerprinting

Detection changes should be observable and reviewable.

Wraith provides rule-diff functionality that can identify meaningful changes and generate deterministic fingerprints for detection content.

This allows detection engineering workflows to answer questions such as:

* Did the rule actually change?
* What changed?
* Did a detection change without an expected fingerprint change?
* Can a detection version be identified deterministically?

The implementation is useful for CI/CD workflows where detection changes should be traceable.

---

# ATT&CK Navigator Integration

Wraith can extract ATT&CK techniques from detection rules and generate ATT&CK Navigator-compatible representations.

This provides a way to visualize detection coverage and understand which techniques are represented by a detection set.

The workflow is:

```text
Detection Rules
      │
      ▼
Technique Extraction
      │
      ▼
ATT&CK Technique IDs
      │
      ▼
Navigator Layer
```

This can be used to identify coverage gaps and prioritize additional detections.

---

# Schema Drift Detection

Security detections depend heavily on event schemas.

A field that exists today may be:

* renamed
* removed
* changed in type
* moved to another event structure
* replaced by a vendor-specific field

Wraith includes schema-drift analysis to identify changes that can affect detection behavior.

The objective is to catch detection failures caused by telemetry changes before those failures reach production.

---

# Event Correlation

Single-event detections are not always sufficient for identifying multi-stage behavior.

Wraith includes correlation functionality for associating related events and evaluating sequences of security-relevant activity.

A simplified example:

```text
Process Creation
       │
       ▼
Encoded PowerShell
       │
       ▼
Network Connection
       │
       ▼
Credential Access
```

Correlation can provide additional context that is unavailable from an isolated event.

---

# Detection Quality Scoring

Wraith includes an explainable detection-quality scoring component.

The objective is not to produce an arbitrary "AI score", but to expose measurable dimensions that contribute to the resulting assessment.

The scoring functionality is designed to help evaluate factors such as:

* rule robustness
* mutation behavior
* coverage
* metadata
* ATT&CK alignment
* detection characteristics

The score is intentionally explainable so that engineers can understand why a detection received a particular assessment.

---

# Compliance Mapping

Wraith supports user-owned compliance mappings.

Mappings are represented as data rather than being hard-coded into the detection engine.

An example mapping file is provided:

```text
engine-python/compliance_mappings.example.json
```

This allows organizations to maintain mappings appropriate to their own compliance requirements and detection programs.

The platform therefore does not assume that one universal compliance mapping is correct for every environment.

---

# Attack Simulation

Wraith includes attack-simulation functionality for constructing attack chains from detection metadata.

The simulator can:

* extract techniques from rules
* prefer explicitly tagged ATT&CK techniques
* fall back when a technique is not directly known
* construct a representative attack chain

The purpose is to connect individual detections to broader adversary behavior.

---

# CI Security Integration

Detection validation can be integrated into CI workflows.

The repository includes GitHub Actions workflow functionality for detection-oriented CI checks.

The CI workflow can incorporate validation stages such as:

```text
Detection Changes
      │
      ▼
Rule Validation
      │
      ▼
Mutation Testing
      │
      ▼
Quality Evaluation
      │
      ▼
CI Result
```

This makes detection engineering part of the software-development lifecycle rather than a manual process performed after deployment.

---

# GitHub App Foundation

Wraith includes a GitHub App integration foundation under:

```text
integrations/github-app/
```

The integration provides the structure required for connecting Wraith's detection-engineering workflow with GitHub-based development processes.

The current repository includes:

```text
integrations/github-app/manifest.yml
integrations/github-app/README.md
```

This component should be considered an integration foundation rather than evidence of a fully deployed GitHub App service.

---

# SOAR / Automation

Wraith includes automation-oriented pipeline functionality and an Anthropic/GitHub integration path.

The project is designed to support security automation workflows where detection analysis can become part of a larger response or engineering process.

External integrations should be evaluated separately from the deterministic local detection-engineering components.

The local test suite primarily validates the project's Python logic and translation functionality. It should not be interpreted as proof that every external production API integration has been exercised against a live third-party service.

---

# Architecture

At a high level, Wraith is organized into several layers:

```text
                         ┌──────────────────────┐
                         │      Detection       │
                         │       Content        │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │   Detection Engine   │
                         └──────────┬───────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
              ▼                     ▼                     ▼
       Sigma Translation      Mutation Testing      Rule Analysis
              │                     │                     │
              ▼                     ▼                     ▼
        SIEM Backends          Robustness            Rule Diff
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ Validation / Analysis│
                         └──────────┬───────────┘
                                    │
          ┌───────────────┬─────────┼──────────┬──────────────┐
          │               │         │          │              │
          ▼               ▼         ▼          ▼              ▼
       ATT&CK         Schema     Correlation  Quality     Compliance
       Mapping         Drift                    Score       Mapping
          │               │         │          │              │
          └───────────────┴─────────┼──────────┴──────────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ CI / Automation /    │
                         │ Operational Workflows│
                         └──────────────────────┘
```

---

# Repository Structure

The repository is organized approximately as follows:

```text
Wraith/
├── engine-python/
│   ├── atomic_red_team.py
│   ├── attack_navigator.py
│   ├── compliance_map.py
│   ├── correlation.py
│   ├── mutation_runner.py
│   ├── mutation_testing.py
│   ├── quality_score.py
│   ├── rule_diff.py
│   ├── run_pipeline.py
│   ├── schema_drift.py
│   ├── sigma_translate.py
│   ├── requirements.txt
│   └── requirements-siem.txt
│
├── integrations/
│   └── github-app/
│       ├── manifest.yml
│       └── README.md
│
├── tests/
│   ├── test_advanced_features.py
│   ├── test_attack_simulator.py
│   ├── test_robustness_fuzzer.py
│   ├── test_sigma_translate.py
│   └── README.md
│
├── docs/
│   ├── API.md
│   ├── ARCHITECTURE.md
│   ├── CI_SECURITY.md
│   ├── DESIGN_DECISIONS.md
│   ├── DEPLOYMENT.md
│   ├── DETECTION_ENGINEERING.md
│   ├── ENTERPRISE.md
│   ├── FEATURES.md
│   ├── OPERATIONS.md
│   ├── PROVENANCE.md
│   ├── SECURITY_MODEL.md
│   ├── TESTING.md
│   ├── THREAT_MODEL.md
│   ├── USING_AS_ACTION.md
│   └── VALIDATION.md
│
├── .github/
│   └── workflows/
│       └── detection-ci.yml
│
├── Makefile
└── README.md
```

---

# Documentation

Detailed project documentation is available in the `docs/` directory.

| Document                                               | Description                                     |
| ------------------------------------------------------ | ----------------------------------------------- |
| [Architecture](docs/ARCHITECTURE.md)                   | System architecture and component relationships |
| [API](docs/API.md)                                     | API and interface documentation                 |
| [CI Security](docs/CI_SECURITY.md)                     | Detection validation in CI/CD                   |
| [Design Decisions](docs/DESIGN_DECISIONS.md)           | Important engineering decisions                 |
| [Deployment](docs/DEPLOYMENT.md)                       | Deployment considerations                       |
| [Detection Engineering](docs/DETECTION_ENGINEERING.md) | Detection development workflow                  |
| [Enterprise](docs/ENTERPRISE.md)                       | Enterprise deployment considerations            |
| [Features](docs/FEATURES.md)                           | Feature-level documentation                     |
| [Operations](docs/OPERATIONS.md)                       | Operational considerations                      |
| [Provenance](docs/PROVENANCE.md)                       | Detection and data provenance                   |
| [Security Model](docs/SECURITY_MODEL.md)               | Security boundaries and controls                |
| [Testing](docs/TESTING.md)                             | Testing strategy                                |
| [Threat Model](docs/THREAT_MODEL.md)                   | Threat model and security assumptions           |
| [Using as Action](docs/USING_AS_ACTION.md)             | GitHub/automation usage                         |
| [Validation](docs/VALIDATION.md)                       | Detection validation methodology                |

Additional test documentation:

* [Test Suite](tests/README.md)

---

# Installation

Wraith's Python components can be run in an isolated Python environment.

Python 3.13 has been used for the current local verification.

Create a virtual environment:

```bash
python3 -m venv .venv
```

Activate it:

### macOS / Linux

```bash
source .venv/bin/activate
```

### Windows

```powershell
.venv\Scripts\activate
```

Upgrade pip:

```bash
python -m pip install --upgrade pip
```

Install the core Python dependencies:

```bash
python -m pip install -r engine-python/requirements.txt
```

Install the optional SIEM translation backends:

```bash
python -m pip install -r engine-python/requirements-siem.txt
```

The optional backend file contains additional pySigma backends used by the Sigma translation functionality.

---

# Testing

Run the complete Python test suite with:

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

Runtime: 0.58s
```

The verified test groups currently include:

### Advanced detection engineering

* deterministic mutation generation
* mutation scoring
* rule-diff fingerprint changes
* ATT&CK Navigator extraction
* schema-drift detection
* correlation
* explainable quality scoring
* user-owned compliance mapping

### Attack simulation

* ATT&CK technique extraction
* tagged-technique preference
* fallback technique handling

### Robustness testing

* mutation generation
* case variation
* PowerShell parameter aliases
* robustness scoring

### Sigma translation

* supported backend output
* Elasticsearch JSON validation
* Splunk PowerShell references
* Kusto encoded-command references
* unknown-backend error handling

---

# Verification Status

The project intentionally distinguishes between implementation and external validation.

| Capability                      | Status                         |
| ------------------------------- | ------------------------------ |
| Detection analysis              | Implemented                    |
| Mutation testing                | Implemented and locally tested |
| Robustness testing              | Implemented and locally tested |
| Rule diff/fingerprinting        | Implemented and locally tested |
| ATT&CK Navigator extraction     | Implemented and locally tested |
| Schema-drift analysis           | Implemented and locally tested |
| Correlation                     | Implemented and locally tested |
| Quality scoring                 | Implemented and locally tested |
| Compliance mapping              | Implemented and locally tested |
| Attack-chain simulation         | Implemented and locally tested |
| Sigma translation               | Implemented and locally tested |
| Elasticsearch translation       | Locally verified               |
| OpenSearch translation          | Locally verified               |
| Splunk translation              | Locally verified               |
| Kusto translation               | Locally verified               |
| CrowdStrike translation         | Locally verified               |
| Loki translation                | Locally verified               |
| CI integration                  | Implemented                    |
| GitHub App foundation           | Implemented                    |
| SOAR integration path           | Implemented                    |
| Live third-party API validation | Environment-dependent          |
| Production deployment           | Not claimed by local tests     |

The capabilities described here are implemented and covered by the verification status above; where validation scope differs by backend or external service, the limitation is explicitly identified.

---

# Reproducibility

The project uses pinned Python dependencies.

Core dependencies are defined in:

```text
engine-python/requirements.txt
```

Optional SIEM translation dependencies are defined in:

```text
engine-python/requirements-siem.txt
```

Using a virtual environment is recommended.

For reproducible testing:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r engine-python/requirements.txt
python -m pip install -r engine-python/requirements-siem.txt
python -m pytest tests/ -v
```

---

# Security Considerations

Wraith is a security-engineering platform and should itself be treated as security-sensitive infrastructure.

Important considerations include:

* detection content can contain sensitive organizational information
* SIEM credentials must never be committed to the repository
* API tokens must be stored using appropriate secret-management mechanisms
* external integrations should follow least-privilege principles
* generated detection queries should be reviewed before production deployment
* compliance mappings should be validated against organizational requirements
* attack simulation should only be executed in authorized environments
* external AI/API integrations should be treated as separate trust boundaries

Wraith does not remove the need for security review of generated or translated detection content.

See:

* [Security Model](docs/SECURITY_MODEL.md)
* [Threat Model](docs/THREAT_MODEL.md)
* [Operations](docs/OPERATIONS.md)

---

# Threat Model

Wraith operates across several trust boundaries:

```text
Detection Author
       │
       ▼
Wraith Engine
       │
       ├──────────────► SIEM / Search Backend
       │
       ├──────────────► GitHub
       │
       ├──────────────► SOAR / Automation
       │
       └──────────────► External APIs
```

Security assumptions and threats include:

* malicious or malformed detection content
* untrusted event data
* compromised integration credentials
* unauthorized CI execution
* malicious pull requests
* dependency compromise
* unsafe generated queries
* excessive API permissions
* leakage of security telemetry

The detailed threat model is documented in:

[Threat Model](docs/THREAT_MODEL.md)

---

# Detection Engineering Workflow

A recommended Wraith workflow is:

### 1. Author

Create or modify the detection.

### 2. Analyze

Inspect metadata, fields, techniques, and detection logic.

### 3. Translate

Generate backend-specific queries where required.

### 4. Mutate

Generate controlled variations of attacker behavior.

### 5. Test

Evaluate detection behavior against original and mutated inputs.

### 6. Measure

Calculate robustness and quality metrics.

### 7. Correlate

Evaluate relationships between related events.

### 8. Map

Associate detections with ATT&CK and organizational compliance requirements.

### 9. Review

Inspect the resulting changes and validation evidence.

### 10. Deploy

Promote validated detection content through the organization's normal change-management process.

---

# Makefile

Common project operations are exposed through the Makefile where supported.

Examples include:

```bash
make mutation-test
make navigator
make quality-test
```

Run:

```bash
make
```

to inspect the available project targets.

---

# CI/CD

The repository contains a GitHub Actions workflow for detection-focused CI.

The workflow is intended to make detection validation part of the development process.

A simplified workflow is:

```text
Pull Request
     │
     ▼
Detection Validation
     │
     ├── Tests
     ├── Mutation Analysis
     ├── Quality Checks
     └── Detection Validation
             │
             ▼
        Review / Merge
```

See:

[CI Security](docs/CI_SECURITY.md)

---

# Design Principles

Wraith follows several engineering principles.

## 1. Detection logic should be testable

A detection should be treated as software-like logic rather than an untested text artifact.

## 2. Adversarial variation matters

Attackers do not necessarily reproduce the exact command or event representation used during detection development.

## 3. Validation should be measurable

Detection engineering should produce evidence rather than relying exclusively on intuition.

## 4. Backend translation should remain separate from detection logic

A detection should ideally remain portable while backend-specific translation occurs at the integration boundary.

## 5. Compliance mappings should be user-owned

Organizations should control their own mappings rather than depending on hard-coded assumptions.

## 6. External integrations are trust boundaries

A passing unit test does not constitute proof that an external production integration is correctly configured.

## 7. Explainability matters

Security engineers should be able to understand why a detection received a particular quality or robustness assessment.

---

# Limitations

Wraith is not intended to replace:

* a production SIEM
* an EDR
* a complete SOAR platform
* threat intelligence platforms
* human detection engineering review
* production change management
* organizational compliance programs

Detection validation is also not equivalent to proving that a detection catches every possible representation of an attack.

Mutation testing improves confidence by evaluating selected variations, but it cannot exhaustively enumerate adversarial behavior.

Similarly, a successful Sigma translation test confirms the translation behavior exercised by the test suite; it does not prove that every generated query is semantically optimal for every organization's telemetry.

---

# Project Roadmap

Planned areas of development include:

* expanded detection datasets
* broader adversarial mutation strategies
* additional SIEM/query backends
* deeper telemetry-schema validation
* larger-scale correlation testing
* expanded ATT&CK coverage analysis
* improved CI reporting
* richer GitHub integration
* production-oriented integration testing
* deployment automation
* interactive demonstration environment
* additional security validation and dependency analysis

Roadmap items should not be interpreted as currently implemented functionality.

---

# Development Philosophy

Wraith is being developed as a practical security-engineering project rather than as a collection of disconnected demonstrations.

The emphasis is on:

```text
Build
  ↓
Test
  ↓
Measure
  ↓
Attack the assumptions
  ↓
Improve
  ↓
Verify
  ↓
Document
```

The project intentionally records limitations and validation boundaries rather than treating every implemented code path as production-proven.

---

# Current Verification Snapshot

Latest local verification:

```text
Environment
-----------
Platform: macOS
Python:   3.13.3
pytest:   9.1.1

Test suite
-----------
Collected: 21
Passed:    21
Failed:     0

Result
------
21 passed in 0.58s
```

Git working tree at the time of this verification:

```text
clean
```

This snapshot describes the local development environment used for verification and should not be interpreted as a guarantee of identical behavior in every deployment environment.

---

# Contributing

Contributions should preserve the project's focus on measurable security engineering.

Before submitting changes:

```bash
python -m pytest tests/ -v
```

For detection-related changes, contributors should consider adding tests that demonstrate:

* expected detection behavior
* negative cases
* adversarial variations
* translation behavior
* metadata changes
* schema assumptions

Changes should also update relevant documentation when behavior or interfaces change.

---

# License

See the repository license file for the applicable terms.

---

# Author / Project

**Wraith**

Detection engineering, validation, and security automation.

GitHub:

https://github.com/locallhosts/Wraith

The project is developed as a security-engineering portfolio and research project, with an emphasis on detection quality, adversarial validation, automation, and reproducible engineering practices.
