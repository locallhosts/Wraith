# API

WRAITH exposes a Go HTTP API used by the dashboard and CI integrations.

## Authentication

Protected endpoints use API-key authentication. Keys are stored as SHA-256 hashes and are associated with RBAC roles: `viewer`, `analyst`, `lead`, and `admin`.

## Core resources

The API provides lifecycle operations around pipeline runs, reports, attestations, approval, and deployment. The canonical machine-readable contract is `docs/openapi.yaml`.

## Security behavior

Authorization is enforced server-side. State-changing operations are audited. Rate limiting is applied per API key. Deployment requires a sufficiently privileged approval and re-verifies the attestation against current rule content.

## Report and attestation

The report endpoint exposes the pipeline evidence used by the dashboard. The attestation endpoint exposes the signed provenance artifact associated with a validated run.

## Deployment

Deployment is deliberately a gated operation. A valid signature alone is insufficient: the service also requires the required approval and a current-content hash match.

For exact paths, request schemas, and response schemas, use `docs/openapi.yaml` as the source of truth.
