# Validation

## Validation contract

A detection is considered behaviorally valid only when it demonstrates the expected attack behavior and does not trigger on the configured benign baseline. Additional stages measure robustness and test-suite sensitivity.

## Evidence levels

| Level | Meaning |
|---|---|
| Static | Syntax, schema, and metadata checks only |
| Translation | Correct native query generation for a backend |
| Behavioral | Attack and benign-baseline execution against live test infrastructure |
| Robustness | Behavior against generated attacker variations |
| Mutation | Evidence that weakened rules are detected as regressions |
| Provenance | Cryptographic binding of evidence to tested rule content |

## Backend scope

Full live attack/false-positive validation currently targets Elasticsearch/OpenSearch infrastructure. Splunk, Sentinel/Defender, CrowdStrike LogScale, and Grafana Loki have verified native translation but are not claimed to have equivalent live CI validation.

## Mutation testing

Mutation testing generates deliberately weakened rule variants and executes them against the same validation corpus. A mutant is killed when the validation suite detects the weakened behavior. The mutation score is the killed/tested fraction.

## Performance

`tools/logblast` provides high-volume synthetic event generation for performance experiments. It has been verified independently at up to 2,000,000 events with approximately 229K events/sec on a single-vCPU test machine. Automatic performance gating remains separate from that measurement utility.

## Limitations

Synthetic telemetry is not production telemetry. A passing result demonstrates behavior against the modeled scenarios and data, not universal detection coverage.
