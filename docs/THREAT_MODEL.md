# Wraith Threat Model

**Status:** Living engineering document  
**Scope:** Wraith API/control plane, frontend, validation pipeline, evidence/provenance, deployment gate, CI/CD, and deployment infrastructure.

Review this model when a trust boundary, integration, privileged operation, or deployment target changes. It complements [Security Architecture](SECURITY_ARCHITECTURE.md) and [Security Model](SECURITY_MODEL.md); it is not a certification or proof that a deployed environment is secure.

## 1. Security objectives

Wraith must:

1. Prevent unauthenticated or under-privileged callers from invoking control-plane actions.
2. Treat Sigma rules, webhook payloads, generated telemetry, translated queries, AI output, and external responses as untrusted.
3. Isolate validation workloads from production credentials and privileged infrastructure.
4. Bind evidence and approval to the exact rule content being promoted.
5. Preserve sufficient audit context to investigate changes and suspicious operations.
6. Fail closed when required verification material or deployment approvals are missing or invalid.
7. Bound CPU, memory, time, disk, request size, queue depth, and external-service usage for attacker-influenced work.

## 2. Assets and required properties

| Asset | Required property | Impact if compromised |
|---|---|---|
| API keys and identities | Confidentiality, revocability, least privilege | Unauthorized control-plane access |
| Signing private key | Confidentiality and integrity | Forged validation attestations |
| Trusted signing public key | Integrity and controlled rotation | Invalid evidence may be trusted |
| Rule source and deployed rule | Integrity and traceability | Detection bypass or unintended change |
| Run reports and provenance | Integrity, completeness, content binding | False assurance and unsafe promotion |
| Approval records | Integrity and attributable actor | Unauthorized production change |
| Audit events | Integrity, access control, retention | Reduced incident visibility |
| Database and configuration | Confidentiality, integrity, availability | Broad control-plane compromise |
| Validation workers and Docker authority | Isolation and constrained privilege | Host or neighboring workload compromise |
| CI identities, tokens, artifacts | Least privilege and integrity | Supply-chain compromise |
| External integrations | Scoped access and validated responses | Data exfiltration or unintended actions |
| Availability and spend | Limits and recovery | Queue starvation, outage, cost abuse |

## 3. Trust boundaries

| Boundary | Data crossing it | Attacker behavior to assume |
|---|---|---|
| GitHub/CI → webhook API | Event body, headers, repository metadata | Forgery, replay, oversized payload, unexpected event |
| Browser/API client → control plane | API key, JSON, IDs, deployment requests | Credential theft, malformed input, enumeration, privilege abuse |
| Internet → public Playground | Sigma text and request metadata | Flooding, parser abuse, resource exhaustion |
| API → queue/worker | Rule bytes, run IDs, job state | Injection, duplicate work, queue flooding, cancellation races |
| Worker → Python/tools | Arguments, environment, files, generated data | Command injection, unsafe parsing, path traversal, unbounded execution |
| Worker → Docker/test services | Container settings, mounts, queries | Escape attempts, resource abuse, lateral movement |
| Pipeline → evidence store | Reports, digests, metadata | Forgery, incomplete evidence, replay, stale evidence |
| Approval → deployment adapter | Actor, approval, attestation, rule content | Approval bypass, TOCTOU, tampering, replay |
| Wraith → external providers | Webhook destinations, optional AI/notification APIs | SSRF, leaked secrets, malicious response, provider outage |
| CI runner → repository/registry | Build scripts, dependencies, artifacts, tokens | Malicious PR, dependency compromise, poisoned artifact |
| Application → persistent stores | Queries, credentials, records | Injection, credential reuse, data tampering |

## 4. Threat scenarios and expected controls

Priority is an initial engineering estimate, not measured likelihood. Validate controls in code and deployment; documentation alone is not proof of enforcement.

| ID | Scenario | Priority | Expected prevention/detection | Verification evidence |
|---|---|---|---|---|
| TM-01 | Stolen API key invokes administrative or deployment routes | Critical | Server-side role checks, narrow key roles, revocation, TLS, rate limiting, audit | Negative authorization tests for each privileged route; key lifecycle tests |
| TM-02 | Malicious Sigma rule compromises its worker | Critical | No shell interpolation; isolated unprivileged worker; minimal environment; no host mounts or Docker socket exposed to untrusted code; CPU/memory/PID/time limits; egress denied by default | Adversarial parser and disposable-worker isolation tests |
| TM-03 | Forged/replayed webhook triggers work | High | Verify signature over exact raw body; reject missing/invalid signatures; cap body; validate event type/repository; track delivery IDs | Positive/negative signature and duplicate-delivery tests |
| TM-04 | Public Playground or pipeline is flooded | High | Per-client and global limits, body caps, queue quotas, bounded concurrency, deadlines, cancellation, overload responses | Burst/load test and queue saturation test |
| TM-05 | Attestation is replayed for modified rule content | Critical | Digest exact rule bytes under documented scheme; verify signature against separately configured trust anchor; compare digest immediately before deployment | Changed bytes, wrong key, altered metadata, stale approval tests |
| TM-06 | Approval is bypassed or raced with promotion | Critical | Server-side lead/admin authorization; bind approval to rule, digest, target, and policy; recheck immediately before write; serialize/conditionally update state | Negative RBAC and concurrent promotion tests |
| TM-07 | Python subprocess inherits production or signing secrets | Critical | Explicit minimal environment allowlist; generated workspace; no secret-bearing logs | Subprocess environment test using sentinel secrets |
| TM-08 | User-controlled ID/path escapes workspace | High | Strict identifier allowlists; path containment checks; generated temporary directories | Traversal tests for absolute paths, dot-dot, separators, encoded forms |
| TM-09 | Audit records are altered by compromised app/database identity | High | Restricted permissions; independent append-only/WORM sink; alert on gaps | Permission review and sink delivery/retention test |
| TM-10 | CI token/dependency modifies release artifact | High | Least-privilege workflow permissions, reviewed immutable action pins where practical, lockfiles, security scans, protected release environments | Workflow permission review and artifact provenance checks |
| TM-11 | AI/notification integration leaks data or fetches attacker URLs | High | Minimum necessary context, redaction, configured destinations, SSRF controls, egress restrictions, human review of generated content | Malicious-response tests and egress policy review |
| TM-12 | Stored rule/report content executes in operator UI | High | Contextual output encoding, safe URL handling, restrictive CSP, exact CORS origins | XSS regression tests with hostile rule titles/report fields |
| TM-13 | Store outage causes fail-open behavior | High | Fail closed for authorization, provenance, and approval; bounded retries/timeouts; backup/restore | Fault-injection tests for store failures |
| TM-14 | Ephemeral resources survive failed/cancelled runs | Medium–High | Run labels, TTL/reconciliation sweeper, cleanup on all terminal paths, isolated volumes | Forced worker crash and orphan sweep test |
| TM-15 | Per-replica rate limit permits aggregate abuse | Medium–High | Document per-replica behavior; shared limiter for scaled production; global limits | Multi-replica load test and configuration review |
| TM-16 | Synthetic validation is mistaken for production proof | High | Label evidence by backend/validation level; include corpus/version/limitations; require environment-specific acceptance | Report/UI assertions and release review |

## 5. Fail-closed requirements

- Missing or invalid webhook secret: reject webhook requests.
- Missing or invalid trusted provenance key: disable deployment verification rather than bypassing signature checks.
- Missing approval, wrong role, changed rule digest, unknown key, or malformed attestation: deny promotion.
- Database/authorization lookup failure: never infer authorization.
- Timeout, parser error, resource exhaustion, or incomplete validation: mark the run failed/incomplete, not passed.
- Playground requests must not enumerate keys, access run administration, deploy rules, query production stores, or invoke unrelated integrations.
- Generated SOAR/AI content remains an untrusted proposal and must not be executed or promoted solely because generation succeeded.

## 6. Assumptions and out of scope

Assumptions:
- Operators protect GitHub accounts, CI runners, cloud identities, deployment hosts, and signing-key custody.
- Production uses TLS at external boundaries and managed secret storage.
- The host kernel, container runtime, and cloud control plane are not already compromised.
- Operators review changes to authorization, validation, provenance, and deployment code.

Wraith cannot by itself prevent compromise of a trusted administrator account, compromised host/kernel, cloud control-plane compromise, all supply-chain attacks, or all semantic detection errors. Synthetic telemetry does not model every production environment. These limits must remain visible in release and deployment decisions.

## 7. Risk treatment and review cadence

Track each scenario as **implemented and tested**, **implemented but not independently tested**, **deployment-dependent**, or **open** in release reviews. Do not close a risk merely because a control is described here.

Review this model when authentication, roles, webhooks, subprocess execution, orchestration, evidence, approval, or deployment changes; before public Playground exposure or a new integration; after an incident or material architecture change; and at least quarterly for production.

## 8. Related documents

- [Security Architecture](SECURITY_ARCHITECTURE.md)
- [Security Model](SECURITY_MODEL.md)
- [Architecture](ARCHITECTURE.md)
- [Provenance](PROVENANCE.md)
- [Validation](VALIDATION.md)
- [CI Security](CI_SECURITY.md)
- [Deployment](DEPLOYMENT.md)
