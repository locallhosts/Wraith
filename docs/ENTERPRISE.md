# Enterprise features

This document covers what was added beyond the core detection-testing
pipeline to make WRAITH suitable for a real organization to run, plus two
features that are genuinely uncommon in detection-engineering tooling.

## Standout features (the "nobody else does this" part)

### 1. Cryptographic provenance attestation for detection rules

Software supply-chain security (SLSA, in-toto, cosign/Sigstore) solved the
problem of "how do I know this binary is actually the one that passed CI"
years ago. Detection engineering has never adopted the same idea, even
though the exact same gap exists: a rule can pass every check in CI and
then get hand-edited — loosened to kill an annoying false positive,
typically — before it reaches the SIEM, and nothing catches that.

`backend-go/provenance` signs a JSON attestation (ed25519, stdlib-only) for
every rule that passes: which techniques it was tested against, its
false-positive rate, its adversarial robustness score, and a SHA-256 hash
of the exact rule content that was tested. `backend-go/deploy` refuses to
promote a rule unless:

1. a human with `lead`+ role explicitly approved it, **and**
2. the attestation's signature verifies against a pinned public key
   (never the key embedded in the attestation — that would let anyone
   self-sign), **and**
3. hashing the rule's *current* on-disk content matches the hash inside
   the attestation — if someone edited it after signing, this fails and
   says so explicitly.

Fully tested with zero external dependencies — see
`backend-go/provenance/attest_test.go`, which includes a test that
specifically proves a post-signing edit gets caught.

### 2. Adversarial evasion robustness scoring

Every detection-as-code pipeline that exists tests a rule against exactly
one canonical payload (usually the matching Atomic Red Team test). That
tells you the rule catches the textbook case. It tells you nothing about
whether it catches an attacker who is one `-enc` vs `-EncodedCommand`
abbreviation away, or running the identical command with mixed case, or
using `%WINDIR%` instead of `C:\Windows\`.

`engine-python/robustness_fuzzer.py` generates a population of adversarial
variants using well-documented, publicly known technique classes (case
variation, PowerShell's own built-in parameter-prefix matching, env-var
path equivalence, whitespace variation), and `robustness_check.py` fires
each variant at the *actual* Elasticsearch query compiled from the rule —
not a Python reimplementation of Sigma semantics — to produce a
Robustness Score surfaced right next to the pass/fail verdict and fed into
the signed attestation. A rule that only matches one literal string looks
exactly as "passing" as a well-written one under the old model; under this
one, it visibly scores worse, before it ever reaches a SOC analyst who has
to find that out the hard way during a real intrusion.

## Standard enterprise hardening

| Area | What was added | Where |
|---|---|---|
| **Auth** | API-key auth (SHA-256 hashed at rest, never plaintext), 4-tier RBAC (viewer/analyst/lead/admin) | `backend-go/auth` |
| **Persistence** | Postgres-backed store (idempotent schema migration on boot) behind a `Store` interface; in-memory fallback clearly marked dev-only | `backend-go/store` |
| **Audit trail** | Append-only log of every state-changing action (actor, role, action, resource, IP, timestamp) | `store.AppendAudit`, called from every mutating handler |
| **Approval gate** | `POST /runs/:id/approve` requires `lead`+; `POST /runs/:id/deploy` requires approval **and** a valid attestation | `backend-go/api/server.go` |
| **Observability** | Prometheus metrics (`/metrics`): run pass/fail rate, duration histograms, false-positive rate distribution, robustness score distribution, approvals/deploys by outcome | `backend-go/metrics` |
| **Structured logging** | `log/slog` JSON logs for every request (actor, latency, status) — stdlib only, no extra dependency | `backend-go/api/server.go` |
| **Rate limiting** | Per-API-key token bucket (documented as per-replica; swap for Redis if horizontally scaling) | `backend-go/ratelimit` |
| **Notifications** | Slack webhook on run completion and on "awaiting approval" | `backend-go/notify` |
| **Kubernetes** | Deployment/Service/HPA/NetworkPolicy manifests + a parameterized Helm chart, non-root containers, read-only root filesystem, dropped capabilities | `k8s/`, `charts/wraith/` |
| **CI security scanning** | gosec (Go SAST), CodeQL (Go/Python/TS), Trivy (container CVEs), gitleaks (secret scanning), pip-audit, npm audit, tfsec (Terraform), Dependency Review — all on every PR | `.github/workflows/security-scan.yml` |
| **API contract** | Full OpenAPI 3.0 spec | `docs/openapi.yaml` |

## Secrets & key management

None of the above is meaningful if secrets sit in a `.env` file in
production. What ships here (`.env.example`, `k8s/01-secrets.template.yaml`)
is a **local-dev convenience**, not a production secrets strategy. For a
real deployment:

- **Signing key** (`WRAITH_SIGNING_PRIVATE_KEY`): should live in a KMS with
  asymmetric signing support (AWS KMS, GCP Cloud KMS, HashiCorp Vault
  Transit) so the raw private key material never exists outside the KMS —
  `backend-go/provenance.Sign` would call out to the KMS's Sign API instead
  of holding the key in memory. The current implementation takes a raw
  `ed25519.PrivateKey` specifically so that swap is a single function's
  worth of change.
- **Database credentials, webhook secret, Slack URL, Anthropic key**: an
  External Secrets Operator syncing from Vault/AWS Secrets Manager into the
  Kubernetes Secret referenced by `k8s/02-backend.yaml`, not committed
  manifests.
- **API keys**: only ever shown once, at creation (`wraith apikey create`),
  and stored hashed. Rotate by creating a new key and revoking the old
  row's `revoked` flag — there's intentionally no "show existing key"
  endpoint.

## What's still a known simplification (and the honest reason why)

- **Rate limiter is per-replica, not global.** A Redis-backed
  sliding-window limiter is the correct fix once you run more than a
  couple of API replicas; documented in `ratelimit/limiter.go` rather than
  silently pretending it's already global.
- **Attestations are read from a shared filesystem path** (
  `engine-python/output/<run_id>/attestation.json`) by the deploy endpoint.
  This works because the CI runner and API server share a volume in the
  docker-compose/k8s setups provided. At real scale, swap for signed
  attestations stored in S3/GCS or, better, a transparency log (Sigstore
  Rekor-style) so attestations are independently auditable outside the
  app's own database.
- **Deploy writes to an Elasticsearch index representing "production
  rules"** as a vendor-neutral stand-in. A real deployment would swap
  `backend-go/deploy.Deploy` to call the specific SIEM's native rule API
  (Elastic Security detection rules API, Splunk ES correlation searches,
  Sentinel analytics rules, etc.) — the gate logic (approval + attestation
  verification) is identical either way, which is the point of keeping it
  in its own package.
