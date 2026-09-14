# Wraith GitHub App

This directory contains the minimal manifest for installing Wraith as a
GitHub App. It uses the existing Wraith webhook/API surface rather than
introducing a second webhook implementation.

## Permissions

- `contents: read` — read Sigma rules and repository configuration.
- `metadata: read` — required GitHub repository metadata.
- `pull_requests: write` — publish validation/regression results when the
  installation is configured to do so.

The manifest intentionally uses a placeholder HTTPS host. Replace it with
the deployed Wraith webhook URL when registering the App. Installation,
private-key storage, webhook signature verification, and deployment remain
operator configuration; no private key belongs in this repository.
