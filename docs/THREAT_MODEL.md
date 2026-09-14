# WRAITH Threat Model

## Objective

WRAITH processes potentially attacker-controlled detection content and participates in CI/CD. Source-controlled rules and external event payloads are therefore treated as untrusted inputs.

## Assets

- Sigma rules
- validation reports
- provenance attestations
- API credentials and hashes
- GitHub tokens
- signing keys
- database records
- audit records
- generated SOAR content
- deployment authorization
- SIEM test data
- CI artifacts

## Trust boundaries

### GitHub -> WRAITH

Pull requests and webhook payloads cross into the platform. Threats include malicious rule content, forged events, oversized payloads, replay, and credential abuse.

### API -> control plane

External API clients cross the authentication boundary. Threats include unauthorized access, privilege escalation, brute force, and resource exhaustion.

### Validation environment

Rules, generated telemetry, and translated queries execute against ephemeral infrastructure. Threats include malicious inputs, query abuse, container escape attempts, and resource exhaustion.

### External APIs

GitHub and optional Anthropic integrations are separate trust domains. Returned content is not automatically trusted as security policy.

### Deployment

Promotion crosses from test evidence into production state. This is the highest-value integrity boundary.

## Threats and controls

| Threat | Control |
|---|---|
| Unauthorized API access | API-key authentication |
| Privilege escalation | Four-tier RBAC |
| Key exposure in database | SHA-256 key hashing |
| Malicious detection content | linting + ephemeral validation |
| Excessive API traffic | per-key rate limiting |
| Evidence tampering | Ed25519 attestation |
| Rule changed after validation | current-content hash verification |
| Deployment without review | lead/admin approval gate |
| Secret committed to source | Gitleaks |
| Vulnerable dependencies | pip-audit / npm audit / dependency review |
| Vulnerable container | Trivy |
| Infrastructure misconfiguration | Terraform/Kubernetes scanning |
| Audit manipulation | append-only audit records |

## Security assumptions

WRAITH is not a replacement for a hardened sandbox or a compromised-host defense. Production deployments should use least-privilege identities, protected CI runners, resource limits, restricted network access, external secret management, protected signing keys, and controlled deployment credentials.

## Out of scope

WRAITH cannot by itself prevent compromise of a trusted GitHub account, CI runner, underlying host, cloud control plane, unrestricted administrator abuse, all semantic detection errors, or all differences between synthetic and production telemetry.

## Primary security property

A passing result must remain cryptographically and semantically bound to the exact rule content being promoted.
