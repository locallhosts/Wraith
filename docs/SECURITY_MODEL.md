# WRAITH Security Model

## Defense layers

```text
Authentication
    -> RBAC
    -> validation
    -> persistence/audit
    -> signed provenance
    -> human approval
    -> current-content verification
    -> deployment
```

## Authentication and RBAC

The Go API uses API-key authentication and four roles: `viewer`, `analyst`, `lead`, and `admin`. Authorization is enforced server-side; UI controls are not a security boundary.

## API keys

Raw API keys are not stored as plaintext. The service stores SHA-256 hashes. Keys should be created through the supported CLI/API flow, exposed only at creation, rotated by replacement and revocation, and never committed to source control.

## Rate limiting

The current limiter uses a per-API-key token bucket. It is per replica. Horizontally scaled deployments requiring globally coordinated limits should move limiter state to a shared system such as Redis.

## Audit

State-changing actions are recorded in the append-only audit path with actor, role, action, resource, source information, and timestamp. This is accountability evidence, not a substitute for external immutable logging in a high-assurance environment.

## Provenance

Passing rules can receive an Ed25519-signed attestation containing validation evidence and a SHA-256 hash of tested rule content. Deployment verifies approval, signature validity against a trusted public key, and current-content hash equality.

## Human approval

The deployment endpoint requires a `lead` or higher approval. Automated validation creates evidence; it does not independently authorize production change.

## Secrets

Development examples use `.env.example` and Kubernetes templates. Production secrets should use an external secret manager or equivalent protected mechanism. The signing private key should ideally be backed by KMS/HSM-style asymmetric signing rather than a long-lived raw private key in application configuration.

## SOAR output

LLM-generated playbooks are draft response content. They require human review and are not authoritative merely because generation succeeded.

## Deployment adapter

The current deploy implementation writes to an Elasticsearch index representing production rules. A real deployment should replace this adapter with the target SIEM's native rule-management API while retaining the approval and provenance gate.
