# Wraith Architecture

Wraith validates Sigma detection rules, records evidence, and gates rule promotion. Its primary security objective is to make validation traceable to the exact rule content tested while keeping untrusted rules and public submissions away from privileged control-plane resources.

This describes logical responsibilities, not a claim that every deployment uses separate hosts or networks. See [Security Architecture](SECURITY_ARCHITECTURE.md), [Threat Model](THREAT_MODEL.md), and [Deployment](DEPLOYMENT.md).

## System context

Logical flow:

- GitHub pull requests and webhook events → Go API and control plane.
- Operator browser/API client → API-key authentication → server-side role authorization.
- API → configured store, pipeline queue/worker, and audit path.
- Pipeline → lint/translation → restricted per-run validation environment → test SIEM/graph services.
- Validation report → evidence store → provenance verification and approval gate → deployment adapter.
- Anonymous Playground → isolated limited validation path only; it must not connect to control-plane capabilities.

## Components and responsibilities

| Component | Responsibility | Security-sensitive inputs |
|---|---|---|
| Go API (backend-go/api) | HTTP routing, auth integration, webhooks, pipeline control, operator endpoints | API keys, webhook bodies, rule content, IDs, deploy requests |
| Authentication (backend-go/auth) | Resolve API-key identity and role | Authorization header, key verifier, identity metadata |
| Store (backend-go/store) | Persist runs, evidence, approvals, jobs, keys, and audit records | All records and database credentials |
| Pipeline worker | Coordinate queued validation | Rule bytes, run IDs, subprocess configuration |
| Linter/translation (backend-go/linter, engine-python) | Validate and translate Sigma rules | Untrusted YAML and query semantics |
| Orchestrator | Create and remove per-run test services | Docker API access, run identifiers, image settings |
| Validation services | Exercise rules against attack and benign scenarios | Generated telemetry and translated queries |
| Provenance | Sign and verify evidence bound to rule content | Reports, private signing key, trusted public key |
| Deployment gate | Verify approval and evidence before promotion | Approval state, attestation, current rule bytes |
| Frontend | Present runs, evidence, audit events, and controls | API responses and operator-entered API key |
| Public Playground | Limited anonymous validation | Anonymous request body and client IP |

## Authenticated validation flow

1. A client submits a request with an API key.
2. Authentication resolves identity; server-side route authorization decides whether the role may act. UI visibility is not a security boundary.
3. The API validates request shape and invokes or enqueues pipeline work.
4. The pipeline lints and translates the rule before behavioral validation.
5. Validation runs against configured test infrastructure and produces a report.
6. The configured store persists run state and evidence; state-changing actions should produce audit events.
7. Where configured, provenance binds evidence to a digest of the tested rule bytes. Passing validation alone is not production authorization.

## Promotion flow

1. An authorized operator requests promotion.
2. The server verifies required approval, signature against a separately trusted public key, and equality between the current rule digest and the attested digest.
3. Only after those checks does the deployment adapter write to its configured target.
4. The decision and result should be auditable. High-assurance deployments should export audit events to an independently controlled destination.

## Public Playground boundary

The Playground is an intentionally separate anonymous entry point. It must not inherit control-plane credentials or access to production Elasticsearch, Neo4j, deployment, key-management, or administrative operations. Enforce request-size and time limits, anonymous rate limits, exact allowed CORS origins, and a minimal subprocess environment. Keep it disabled unless exposure and controls have been reviewed.

## Language and process boundary

- Go owns the API/control plane, authentication integration, orchestration, validation coordination, and provenance/deployment gates.
- Python owns data-oriented pipeline tasks and libraries used for Sigma translation and synthetic telemetry/attack simulation.
- Treat all data crossing the Go-to-Python boundary as untrusted. Use discrete process arguments rather than shell interpolation, a minimal environment, bounded execution time, restricted filesystem permissions, and resource/network limits.
- Do not pass API credentials, database DSNs, signing keys, or cloud credentials to rule-processing subprocesses.
- If subprocess execution is replaced with RPC or a queue, preserve these controls at the new boundary.

## Infrastructure lifecycle

- Ephemeral validation resources must be run-scoped, labelled with an opaque run identifier, constrained, and removed on success, failure, cancellation, and timeout. Reconcile abandoned resources after worker crashes.
- Persistent services require separate network, access, patching, backup, recovery, and monitoring policies.
- Containerization alone is not a security sandbox. Production validation should run on a hardened, least-privileged worker with no production secrets and outbound network access denied by default.

## Validation contract and limits

The pipeline combines static checks, query translation, attack-scenario execution, benign-baseline evaluation, and robustness/mutation checks where configured. A pass means the rule behaved as expected against the tested corpus and backend; it does not establish universal detection coverage or equivalent behavior across SIEM products. See [Validation](VALIDATION.md) for backend-specific evidence scope.

## Operational invariants

1. Untrusted rule content never becomes a shell command or privileged configuration.
2. Anonymous Playground requests cannot reach authenticated control-plane capabilities.
3. Every privileged operation is authorized server-side.
4. A passing run is not deployment authorization.
5. Promotion requires approval and evidence bound to current rule bytes.
6. Trust comes from operator-configured key material, not a key supplied by an untrusted attestation.
7. Validation workers do not receive production credentials or signing private keys.
8. Audit evidence is exported off-host for high-assurance deployments.
9. Ephemeral resources have quotas, timeouts, cleanup, and reconciliation.
10. Security claims must be scoped to controls actually configured and tested.
