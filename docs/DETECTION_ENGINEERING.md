# Detection Engineering

## WRAITH's model

WRAITH treats a detection rule as a behavioral specification rather than a YAML document that only needs to parse.

The pipeline asks:

1. Does the rule detect the intended behavior?
2. Does it remain quiet on benign activity?
3. Does it survive meaningful attacker variations?
4. Can the tested content be bound to the content later promoted?

## Rule lifecycle

```text
Sigma
  -> lint
  -> ATT&CK mapping
  -> benign baseline
  -> attack scenario
  -> adversarial variants
  -> SIEM translation
  -> behavioral validation
  -> mutation testing
  -> report
  -> attestation
  -> approval
  -> deployment
```

## Validation layers

### Attack validation

The attack simulator maps Sigma ATT&CK tags to a multi-stage synthetic attack scenario and corresponding telemetry. A passing rule must match the intended attack telemetry.

### Baseline validation

The baseline generator produces synthetic enterprise activity over the validation window. A passing rule must remain silent on that baseline.

### Robustness

`robustness_fuzzer.py` creates documented adversarial variations. The score describes the variants actually tested; it is not a claim that all real-world bypasses have been modeled.

### Mutation testing

Mutation testing deliberately weakens a rule and checks whether the validation suite detects the regression. This tests the sensitivity of the detection test itself rather than only testing the original rule.

## Multi-SIEM translation

Verified native targets are Elasticsearch, OpenSearch, Splunk, Sentinel/Defender KQL, CrowdStrike LogScale, and Grafana Loki. Translation verification is intentionally separate from live attack/false-positive validation.

QRadar and Datadog are not presented as supported targets because their current pySigma backend requirements do not fit the pinned modern pySigma dependency set.

## Engineering principle

Syntax validity, intended detection, false-positive behavior, adversarial robustness, backend translation, mutation sensitivity, and provenance integrity are separate properties. Passing one does not imply passing the others.
