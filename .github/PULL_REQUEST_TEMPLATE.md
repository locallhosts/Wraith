## What does this change?



## Which component(s)?

- [ ] backend-go
- [ ] engine-python
- [ ] frontend
- [ ] tools/logblast
- [ ] infra/terraform, k8s/, or charts/
- [ ] action.yml / CI workflows
- [ ] Documentation only

## How was this verified?

Be specific and honest — "ran the existing tests" is fine, "tested
end-to-end against a real X" is better, "wrote it and it looks right to
me" is also fine to say explicitly if that's genuinely as far as you got.
See CONTRIBUTING.md's "Pull request expectations" for why this matters
more here than in a typical repo.

## Checklist

- [ ] `gofmt -l .` is silent (if this PR touches `backend-go/`)
- [ ] `go test ./...` passes (if this PR touches `backend-go/`)
- [ ] `python3 -m pytest tests/ -v` passes (if this PR touches `engine-python/`)
- [ ] `npx tsc --noEmit` and `npm run build` succeed (if this PR touches `frontend/`)
- [ ] New behavior has a test, or I've explained above why it can't be
      tested in CI (e.g., requires a real SIEM instance we don't have
      ephemeral infra for yet)
- [ ] I updated relevant docs (README, docs/ENTERPRISE.md,
      docs/ARCHITECTURE.md) if this changes behavior described there
