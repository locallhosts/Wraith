//go:build integration

// Run with a real Postgres available:
//
//	export WRAITH_TEST_DATABASE_URL=postgres://wraith:wraith@localhost:5432/wraith?sslmode=disable
//	go test -tags=integration ./store/... -v
//
// These were verified against a live PostgreSQL 16 instance during
// development — see docs/ENTERPRISE.md for the verification log. They are
// excluded from the default `go test ./...` / CI unit test run via the
// build tag above so contributors without a local Postgres aren't blocked;
// .github/workflows/detection-ci.yml runs them against a real `postgres:`
// service container.
package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func testDSN(t *testing.T) string {
	dsn := os.Getenv("WRAITH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WRAITH_TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	return dsn
}

// execCleanup removes rows this test suite owns, identified by fixed test
// IDs, so the suite can run repeatedly against a persistent database
// without colliding with leftovers from a prior run.
func execCleanup(ctx context.Context, p *PostgresStore, runID, apiKeyHash string) error {
	if _, err := p.db.ExecContext(ctx, `DELETE FROM audit_log WHERE resource = $1`, runID); err != nil {
		return err
	}
	if _, err := p.db.ExecContext(ctx, `DELETE FROM runs WHERE run_id = $1`, runID); err != nil {
		return err
	}
	if _, err := p.db.ExecContext(ctx, `DELETE FROM api_keys WHERE key_hash = $1`, apiKeyHash); err != nil {
		return err
	}
	return nil
}

func TestPostgresStore_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	p, err := NewPostgresStore(ctx, testDSN(t))
	if err != nil {
		t.Fatalf("connecting to real postgres: %v", err)
	}
	defer p.Close()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Clean up any leftover rows from a previous run of this test so it's
	// safely re-runnable against a persistent (non-ephemeral) database.
	t.Cleanup(func() {
		_ = execCleanup(ctx, p, "it-run-1", "test-hash-abc123")
	})
	_ = execCleanup(ctx, p, "it-run-1", "test-hash-abc123")

	run := &RunStatus{
		RunID: "it-run-1", RulePath: "rules/test.yml", RuleID: "rule-abc",
		RuleTitle: "Integration Test Rule", Repo: "org/repo", PRNumber: 42,
		Stage: "lint", StartedAt: time.Now(),
	}
	if err := p.PutRun(ctx, run); err != nil {
		t.Fatalf("put run: %v", err)
	}

	got, err := p.GetRun(ctx, "it-run-1")
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.RuleTitle != "Integration Test Rule" || got.Stage != "lint" {
		t.Fatalf("unexpected run after insert: %+v", got)
	}

	passed := true
	run.Stage = "done"
	run.Passed = &passed
	run.Reason = "all checks passed"
	if err := p.PutRun(ctx, run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	got, err = p.GetRun(ctx, "it-run-1")
	if err != nil {
		t.Fatalf("get run after update: %v", err)
	}
	if got.Stage != "done" || got.Passed == nil || !*got.Passed {
		t.Fatalf("expected updated run to show stage=done passed=true, got: %+v", got)
	}

	now := time.Now().UTC().Truncate(time.Second)
	got.ApprovedBy = "alice@soc"
	got.ApprovedAt = &now
	if err := p.PutRun(ctx, got); err != nil {
		t.Fatalf("put approval: %v", err)
	}
	got2, err := p.GetRun(ctx, "it-run-1")
	if err != nil {
		t.Fatalf("get run after approval: %v", err)
	}
	if got2.ApprovedBy != "alice@soc" || got2.ApprovedAt == nil {
		t.Fatalf("expected approval fields to persist, got: %+v", got2)
	}

	list, err := p.ListRuns(ctx, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	found := false
	for _, r := range list {
		if r.RunID == "it-run-1" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected inserted run to appear in ListRuns")
	}

	_, err = p.GetRun(ctx, "does-not-exist")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing run, got: %v", err)
	}

	entry := &AuditEntry{
		Actor: "alice@soc", ActorRole: "lead", Action: "run.approve",
		Resource: "it-run-1", Detail: "integration test", IPAddress: "127.0.0.1",
	}
	if err := p.AppendAudit(ctx, entry); err != nil {
		t.Fatalf("append audit: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected AppendAudit to populate the generated ID")
	}

	auditList, err := p.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(auditList) == 0 || auditList[0].Action != "run.approve" {
		t.Fatalf("expected audit entry to be listed, got: %+v", auditList)
	}

	keyHash := "test-hash-abc123"
	if err := p.CreateAPIKey(ctx, keyHash, "ci-bot", "analyst"); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	key, err := p.GetAPIKey(ctx, keyHash)
	if err != nil {
		t.Fatalf("get api key: %v", err)
	}
	if key.Label != "ci-bot" || key.Role != "analyst" || key.Revoked {
		t.Fatalf("unexpected api key: %+v", key)
	}

	_, err = p.GetAPIKey(ctx, "nonexistent-hash")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing key, got: %v", err)
	}
}

func TestPostgresStore_SchemaIsIdempotent(t *testing.T) {
	ctx := context.Background()
	dsn := testDSN(t)

	p1, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("first connect: %v", err)
	}
	defer p1.Close()

	p2, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("second connect (schema re-apply): %v", err)
	}
	defer p2.Close()

	if err := p2.Ping(ctx); err != nil {
		t.Fatalf("second connection not usable: %v", err)
	}
}
