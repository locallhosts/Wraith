# CI Security

## Pipeline controls

The repository uses CI security checks for source analysis, dependency auditing, secret detection, container scanning, and infrastructure validation.

## Permissions

GitHub Actions permissions should be kept at the minimum required level. Workflows that only lint or validate content should not receive write permissions unless the specific step needs them.

## Supply-chain controls

WRAITH uses signed provenance for detection content and scans dependencies and containers. CI should pin important actions and review dependency changes rather than treating scanners as a substitute for review.

## Dependency exceptions

A scanner finding without a fixed upstream release is documented rather than falsely marked as remediated. The repository previously tracked a `diskcache` advisory (`PYSEC-2026-2447`) where no fixed release was available; this is an accepted/monitored exception until an upstream fix exists.

## Pull request security

Rules are untrusted inputs. Validation infrastructure should remain ephemeral and resource-limited. Secrets must never be derived from attacker-controlled rule content or exposed to jobs that do not need them.

## Future hardening

Potential improvements include stronger action pinning, isolated runners for hostile content, automatic performance gates, broader live backend validation, and external key management for signing.
