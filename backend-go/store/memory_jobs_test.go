package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryPipelineJobLifecycle(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()
	run := &RunStatus{RunID:"run-1", RulePath:"rules/test.yml", RuleID:"rule.test", RuleTitle:"Test", Repo:"org/repo", StartedAt:time.Now()}
	if err := m.PutRun(ctx, run); err != nil { t.Fatal(err) }
	if err := m.EnqueuePipelineJob(ctx, &PipelineJob{RunID:"run-1", RulePath:run.RulePath, MaxAttempts:3}); err != nil { t.Fatal(err) }
	jobs, err := m.ListPipelineJobs(ctx, 10)
	if err != nil || len(jobs) != 1 { t.Fatalf("expected one job, got %d, err=%v", len(jobs), err) }
	job, err := m.GetPipelineJob(ctx, jobs[0].ID)
	if err != nil { t.Fatal(err) }
	if job.Status != "queued" { t.Fatalf("expected queued, got %q", job.Status) }
	if err := m.RetryPipelineJob(ctx, job.ID); err != nil { t.Fatal(err) }
	job, err = m.GetPipelineJob(ctx, job.ID)
	if err != nil { t.Fatal(err) }
	if job.Status != "queued" || job.Attempts != 0 { t.Fatalf("expected reset queued job, got status=%q attempts=%d", job.Status, job.Attempts) }
}
