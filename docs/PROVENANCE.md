# Provenance

## Purpose

WRAITH uses signed provenance to prevent a passing validation result from being silently reused for different rule content.

## Binding

The attestation includes validation evidence and a SHA-256 hash of the tested rule content. The Go provenance component signs the result with Ed25519.

## Verification

A deployment verifies:

1. the required approval exists;
2. the signature verifies against a trusted public key;
3. the current rule content hashes to the attested content hash.

The public key embedded in an untrusted attestation is not sufficient as a trust anchor.

## Key handling

The private signing key belongs in protected CI/secret infrastructure and must never be committed. Production deployments should prefer KMS/HSM-backed asymmetric signing where practical.

## Security property

The provenance system is intended to establish content integrity and evidence continuity. It does not prove that the detection logic is semantically perfect or that synthetic telemetry represents every production environment.
