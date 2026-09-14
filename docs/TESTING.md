# Testing

## Strategy

WRAITH combines unit tests, integration tests, static validation, live service checks, and performance experiments. Each result should state what was actually exercised.

## Existing verified areas

- Go provenance: 5/5 unit tests
- PostgreSQL store: 2/2 integration tests against PostgreSQL 16
- Go linter: 4/4 tests
- Attack simulator: 3/3 tests
- Robustness fuzzer: 5/5 tests
- Multi-SIEM translation: 5/5 tests plus independent verification of six native backends
- Kubernetes: 8/8 resources valid under strict kubeconform
- Frontend: production build with zero TypeScript errors
- logblast: 2,000,000-event end-to-end throughput test with zero malformed documents or HTTP errors
- Advanced feature suite: 8/8 tests

## Advanced feature tests

`tests/test_advanced_features.py` covers mutation generation/scoring, rule diffing, ATT&CK Navigator export, schema drift, multi-rule correlation, quality scoring, Atomic Red Team import, and compliance mapping.

## Environment dependencies

The full Python suite requires the project's Python dependencies, including `elasticsearch` and `sigma`. Missing packages in a bare environment are dependency/setup failures, not evidence that the tests themselves are broken.

## Reproducibility

Prefer deterministic fixtures and isolated services. Record backend scope and environment when reporting results. Do not describe translation-only checks as equivalent to live behavioral validation.
