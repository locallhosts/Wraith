# Contributing to WRAITH

Thanks for considering it — genuinely. This project benefits from more
hands specifically because it spans several disconnected worlds (Sigma/
detection engineering, Go backend work, Python data generation, Neo4j,
Kubernetes, C for the performance tooling) and no one person is going to
be equally strong across all of them.

## Ways to contribute that don't require touching code

- **File an issue** if a Sigma rule you tested behaved unexpectedly, a
  backend translation produced wrong output, or a doc is unclear/wrong.
  See "Verification status" in the README for what's actually been
  tested vs. what's a documented known gap — if you hit a gap, that's
  useful signal, not a dumb question.
- **Add a Sigma rule** to `rules/` with good MITRE ATT&CK tagging — every
  rule added is another real test case for the pipeline itself.
- **Report a SIEM backend compatibility issue.** `engine-python/sigma_translate.py`
  documents exactly which pySigma backends are currently compatible with
  the pinned core version and which aren't (QRadar, Datadog, as of this
  writing) — if that's changed, a PR bumping `requirements-siem.txt` with
  proof it installs cleanly is extremely welcome.

## Ways to contribute that do

**High-value, well-scoped starting points** (see `docs/ENTERPRISE.md`'s
"known simplification" list and this README's "Design notes" for the
full backlog):

1. **Live validation for a second SIEM backend.** Today, only
   Elasticsearch/OpenSearch get real attack-simulation + false-positive
   testing (`backend-go/orchestrator` provisions ephemeral ES/Neo4j).
   Splunk, Sentinel, CrowdStrike, and Loki translation already works
   (`sigma_translate.py`) — wiring up ephemeral test infrastructure for
   one of them, plus the equivalent of `validator.Validate`, would be a
   substantial and very welcome contribution.
2. **Wire `logblast` (the C throughput tool) into `run_pipeline.py`**
   as an optional `--perf-events N` flag gating on
   `validator.ProfileQuery`'s overhead percentage — see
   `tools/logblast/README.md`'s "How it fits into the pipeline" section
   for the exact integration point.
3. **A report-viewer enhancement**: render the Neo4j attack graph inline
   in the dashboard (currently linked out to Neo4j Browser) — see
   `frontend/components/ReportViewer.tsx`.
4. **Redis-backed rate limiting** for horizontally-scaled deployments —
   `backend-go/ratelimit/limiter.go` documents exactly why the current
   per-replica limiter is a known simplification and what to replace it
   with.

## Development setup

See the README's "Quickstart" for running the whole stack, and "Running
it locally" for developing on individual pieces. In short:

```bash
# Go
cd backend-go && go mod tidy && go test ./...
gofmt -l .          # must produce no output before you open a PR

# Python
cd engine-python && pip install -r requirements.txt -r requirements-siem.txt
cd .. && python3 -m pytest tests/ -v

# Frontend
cd frontend && npm install && npx tsc --noEmit && npm run build

# C tool
cd tools/logblast && gcc -O2 -Wall -Wextra -pthread -o logblast logblast.c -lcurl
```

## Pull request expectations

- **Tests for anything you can test.** This project's whole premise is
  "verify before you ship" — a PR that adds an untested feature to a
  detection-testing pipeline is a bit of an irony we'd like to avoid.
  Look at `tests/test_sigma_translate.py` or
  `backend-go/provenance/attest_test.go` for the level of rigor we're
  going for: prove the thing actually works, not just that it doesn't
  crash.
- **Be honest about what's verified vs. assumed** in your PR
  description, the same way `docs/ENTERPRISE.md` and this file are. If
  you wired something up but couldn't test it end-to-end (e.g., no
  access to a real Splunk instance), say so — that's genuinely useful
  information for a reviewer and the next contributor, not a weakness in
  the PR.
- **`gofmt -l .` and `npx tsc --noEmit`** should both be silent before
  you open a PR touching Go or TypeScript.
- Security-sensitive changes (anything in `backend-go/auth`,
  `backend-go/provenance`, `backend-go/deploy`) get extra scrutiny —
  see `SECURITY.md` if you're reporting a vulnerability rather than
  fixing one, that goes through a different channel.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
Short version: be someone people want to collaborate with again.
