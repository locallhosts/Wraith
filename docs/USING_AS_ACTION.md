# Using WRAITH as a GitHub Action

## Minimal workflow

```yaml
name: Lint detection rules
on: pull_request
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: locallhosts/Wraith@v1
        with:
          rules-path: rules
          fail-on-warning: 'false'
```

## What it provides

The reusable action runs the Go Sigma linter and emits GitHub workflow annotations for findings such as missing required fields, missing ATT&CK tags, and documented-quality issues.

## What it does not require

The zero-setup Action does not require Elasticsearch, Neo4j, WRAITH API credentials, or an Anthropic key.

## Full pipeline

The complete detection pipeline is separate. It adds attack simulation, benign-baseline validation, robustness testing, mutation testing, reporting, and optional SOAR/provenance steps. It requires the services and secrets documented in the main repository README and CI workflow.

## Security note

Treat detection rules as untrusted repository content. Keep workflow permissions minimal and do not expose production secrets to jobs whose only purpose is linting untrusted changes.
