# Security Policy

## Reporting a vulnerability

If you find a security issue in WRAITH itself (not in a detection rule
you wrote — that's a false negative, report it as a regular issue),
please **do not open a public GitHub issue**. Use GitHub's private
[Security Advisories](../../security/advisories/new) feature for this
repository instead, so the issue isn't public before a fix ships.

Please include:

- Which component (`backend-go/*`, `engine-python/*`, `frontend/*`,
  `tools/logblast/*`, the Terraform/Kubernetes manifests) is affected
- Steps to reproduce, or a minimal proof of concept
- What you'd expect to happen vs. what actually happens
- Your assessment of impact, if you have one — we'll form our own
  independently, but your reasoning helps triage faster

## What's especially worth reporting privately

Given what this project does, these categories get priority attention:

- **Anything that lets an unapproved rule reach the deploy endpoint** —
  i.e., a way around the approval-gate + provenance-attestation checks in
  `backend-go/deploy` and `backend-go/provenance`. This is the core
  security property of the whole system; a bypass here is critical.
- **Authentication/authorization bypasses** in `backend-go/auth` — e.g.,
  a way to escalate role, forge an API key, or access an endpoint without
  a valid key.
- **Webhook signature verification bypasses** in `backend-go/webhook`.
- **Injection vulnerabilities** anywhere user- or PR-controlled content
  (a Sigma rule's fields, a PR's metadata) reaches a shell command,
  SQL query, or the LLM prompt in `soar_playbook_generator.py`.

## What's out of scope

- Vulnerabilities in third-party dependencies (Elasticsearch, Neo4j,
  Postgres, the pySigma backends) — report those upstream. If a
  dependency version pinned in `requirements.txt`/`requirements-siem.txt`/
  `go.mod`/`package.json` has a known CVE, a PR bumping the pin (with
  proof it still installs cleanly and tests still pass — see
  `CONTRIBUTING.md`) is welcome as a normal PR, not a security report.
- The fact that `backend-go/ratelimit` is per-replica, not
  globally distributed — this is a documented, intentional simplification
  (see `docs/ENTERPRISE.md`), not a vulnerability.
- Missing live-validation support for SIEM backends other than
  Elasticsearch/OpenSearch — also a documented gap, not a vulnerability.

## Supported versions

This is a portfolio/reference project rather than a versioned product
with a formal support window — the `main` branch is the only supported
target. If you're running an older commit, please reproduce against
current `main` before reporting.
