# Wraith Security Architecture

This document describes how Wraith should enforce security at component boundaries and which controls are required before production exposure. It must be checked against implementation and deployment evidence; it is not a substitute for tests.

## 1. Design principles

1. **Deny by default:** missing identity, role, approval, trusted key, or verification evidence is a denial.
2. **Separate control plane from untrusted execution:** rule parsing, translation, and simulation must not run with administrative authority or production credentials.
3. **Make trust explicit:** a hash identifies content; a signature authenticates a statement from a key; neither proves detection quality by itself.
4. **Bind authorization to the operation:** verify actor, role, resource, target, and current content at promotion time.
5. **Constrain expensive work:** explicitly bound input size, queue depth, concurrency, execution time, memory, disk, process count, and outbound network access.
6. **Preserve independent evidence:** application audit records are useful but should be exported to a separately administered destination.
7. **Report evidence scope honestly:** distinguish static validation, query translation, behavioral execution, robustness testing, and cryptographic provenance.

## 2. Logical security zones

Logical flow: Untrusted browser/API clients and GitHub events enter the Go HTTP router. Authenticated routes pass through API-key identity and role checks before reaching the pipeline coordinator, persistent store, and audit path. The coordinator invokes the restricted lint/translation and per-run validation zone. Reports flow to evidence storage and provenance verification; the approval gate then permits the deployment adapter to write to the configured target. The anonymous Playground follows a separate limited path to lint/translation and must not connect to privileged control-plane services.

Boundary rules:
- The public Playground has no route or credential path into control-plane operations, persistent production stores, signing keys, or deployment.
- The validation zone must not read production secrets or write production rules.
- Only the smallest required signing component receives the signing private key. Validation code receives neither that key nor deployment credentials.
- The deployment gate uses a trusted public key configured independently of the attestation payload.
- Browser-side checks are usability features only. The API authorizes every protected request.

## 3. Identity and authorization

- Use individually scoped API keys for people and automation; avoid shared team keys.
- Store a verifier/hash rather than raw key material, and reveal a newly created secret only once.
- Roles viewer, analyst, lead, and admin are server-side policy inputs, not UI labels.
- Check authorization on every request and state transition. Do not rely on earlier UI checks or a previously authorized list response.
- Provide expiry/rotation and immediate revocation procedures. If compromise is suspected, revoke first and investigate.
- Protect transport with TLS outside local development; never put credentials in URLs, logs, crash reports, or telemetry.
- The current rate limiter is per replica; horizontally scaled production needs shared and global abuse limits.

## 4. Untrusted rule execution

Treat rule source, tags, titles, field names, generated telemetry, translated queries, and reports as untrusted data.

Required controls:
- Parse structured input with safe libraries; enforce schema, byte-size, nesting, and complexity limits before expensive processing.
- Avoid shell command construction. Pass arguments as discrete values and use generated workspaces rather than request-derived paths.
- Set explicit subprocess deadlines, cancellation, output-size caps, and a minimal environment allowlist.
- Run on isolated, unprivileged workers with no Docker socket or host mounts exposed to untrusted workload code. If orchestration requires Docker control, keep that authority in a trusted orchestrator separate from parser/simulation processes.
- Apply CPU, memory, PID, disk, and concurrency quotas. Deny outbound network by default; allow only required test endpoints.
- Use per-run filesystem/network isolation, cleanup on all terminal paths, and periodic orphan reconciliation.
- Treat timeout, parser failure, missing stage, or incomplete report as non-passing.

## 5. Evidence and promotion integrity

Promotion should require all of these, checked server-side close to the target write:

1. An authenticated actor with the required role.
2. A valid approval for the intended rule, digest, target, and policy version.
3. A valid signature over a well-defined attestation payload, verified using a trusted public key configured outside that payload.
4. Current rule bytes matching the attested digest.
5. A report identifying the expected run/rule and containing required completed validation stages.
6. A successful target write whose outcome is recorded.

Recommended hardening:
- Include schema version, algorithm identifier, key ID, rule ID, run ID, content digest, validation profile/backend, timestamps, and relevant policy version in the signed payload.
- Define canonical serialization and digest boundaries; do not sign ambiguous concatenated strings.
- Reject unknown key IDs and unsupported algorithms; document key rotation.
- Expire approvals and prevent replay across rules, environments, or targets.
- Recheck digest and approval immediately before writing to reduce time-of-check/time-of-use risk.
- Store signed attestation and verification result with run evidence.
- Test tampering, wrong keys, stale approval, wrong target, missing stage, and changed rule bytes.

A signature proves that a trusted key signed an attestation payload. It does not prove the rule is correct, the environment representative, or the signer uncompromised.

## 6. Audit and incident readiness

Record actor/key label, role, action, resource identifier, decision, timestamp, request/correlation ID, and safe source context. Never record raw API keys, signing private keys, authorization headers, webhook secrets, or secret-bearing environment values.

For production:
- Export audit events to a separately administered append-only or WORM-capable destination.
- Alert on repeated authentication failures, sensitive-route denials, deployment denials, key creation/revocation, signature failures, worker timeouts, queue saturation, and audit delivery gaps.
- Define retention, access, backup, restore, and incident-response procedures.
- Keep clocks synchronized and carry stable request/run IDs across API, worker, evidence, and deployment logs.

## 7. External integrations and browser boundary

- Verify webhook signatures over the exact raw request body before trusting payload fields; validate event type, repository identity, and delivery identifier.
- Restrict outbound destinations to operator-configured hosts. Apply SSRF controls before any privileged service fetches user-supplied URLs.
- Send minimum necessary data to optional AI/notification providers; redact secrets and sensitive telemetry.
- Treat provider responses and model-generated playbooks as untrusted; require validation and human review.
- Render rule metadata and reports with contextual output encoding. Avoid unsafe HTML insertion; use restrictive CSP and exact CORS allowlists.
- Public endpoints need independent body limits, per-IP limits, global concurrency limits, and explicit enablement.

## 8. Production readiness gates

- [ ] API authentication and route-level RBAC negative tests pass.
- [ ] Webhook invalid-signature and duplicate-delivery tests pass.
- [ ] Public Playground isolation is tested, or the feature remains disabled.
- [ ] Worker process environment contains no production or signing secrets.
- [ ] Resource limits, deadlines, egress restrictions, and cleanup are enforced.
- [ ] Changed-rule, wrong-key, tampered-attestation, stale-approval, and wrong-target tests are rejected.
- [ ] Database and signing keys are managed and backed up according to policy.
- [ ] Audit events reach an independent destination and alerting is configured.
- [ ] Restore, key rotation/revocation, and incident response have been exercised.
- [ ] Deployed configuration has been reviewed; green CI and documentation alone are not production certification.

## Related documents

- [Architecture](ARCHITECTURE.md)
- [Threat Model](THREAT_MODEL.md)
- [Security Model](SECURITY_MODEL.md)
- [Provenance](PROVENANCE.md)
- [Deployment](DEPLOYMENT.md)
