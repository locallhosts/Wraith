# Operations

## Local stack

The standard local path uses Docker Compose for PostgreSQL, Elasticsearch, Neo4j, the Go backend, and the frontend.

```bash
docker compose up --build
```

## CI

Pull requests touching detection rules run the linter and, in the full pipeline, provision ephemeral validation infrastructure. The reusable GitHub Action is intentionally lighter and requires no WRAITH services.

## Kubernetes

`k8s/` contains deployment manifests and `charts/wraith/` contains the Helm chart. Validate manifests before applying them to a cluster and use protected service accounts, network policies, resource limits, and external secrets for production.

## Terraform

`infra/terraform/` describes the persistent deployment. Keep real secrets out of Terraform source and state where possible; use the deployment environment's secret-management facilities.

## Observability

The backend exposes health/metrics surfaces used by the dashboard and monitoring stack. Audit events should be retained separately when stronger immutability is required.

## Scaling

The validation environment is designed for short-lived per-run infrastructure. Persistent API/database deployments can scale independently. The current API-key rate limiter is per replica and should be replaced with shared state for globally coordinated limits.

## Secrets

Never commit API keys, GitHub tokens, Anthropic keys, database passwords, or signing private keys. Development examples belong in `.env.example`; production secrets belong in protected secret management.

## Incident handling

If a signing key, GitHub token, or API key is exposed: revoke/rotate it first, preserve relevant audit and CI evidence, determine affected runs, and revalidate content before deployment.
