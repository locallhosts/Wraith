# Design Decisions

## Synthetic validation instead of production data

WRAITH uses generated attack and benign telemetry so contributors can reproduce validation without access to customer logs or a real SOC.

## Elasticsearch/OpenSearch as the live validation baseline

The orchestration layer provisions these backends for ephemeral CI validation. Other SIEMs are supported at translation level without falsely claiming identical live coverage.

## Sigma translation separate from behavioral validation

Native query generation and runtime behavior are distinct failure domains. Keeping them separate makes backend support claims measurable.

## Mutation testing is a separate layer

A test suite can pass while being too weak to detect regressions. Mutation testing deliberately weakens detections to measure whether the validation process notices.

## Human approval before deployment

Automated evidence is not treated as deployment authorization. A lead/admin approval remains required.

## Signed provenance

The tested rule content is bound to an Ed25519-signed attestation and a SHA-256 content hash so a different rule cannot silently reuse prior validation evidence.

## No generic Sigma -> YARA/Falco converter

Sigma, YARA, and Falco express different semantic models. WRAITH intentionally avoids a superficial converter that would create misleading support claims. Dedicated adapters can be added where the semantics can be preserved.

## Organization-owned compliance mapping

Compliance mappings are supplied as organization-owned control data. WRAITH does not claim that a generic ATT&CK mapping is an authoritative legal or audit determination.
