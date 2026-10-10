package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryEnqueuePipelineJobIsIdempotentForActiveAndSucceededJobs(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()
	job := &PipelineJob{RunID: "run-idempotent", RulePath: "rules/example.yml", Status: "queued", MaxAttempts: 3}
	if err := m.EnqueuePipelineJob(ctx, job); err != nil {
		t.Fatalf("enqueue initial job: %v", err)
	}
	if err := m.EnqueuePipelineJob(ctx, job); err != nil {
		t.Fatalf("enqueue duplicate active job: %v", err)
	}
	jobs, err := m.ListPipelineJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("duplicate enqueue created %d jobs, want 1", len(jobs))
	}

	if err := m.CompletePipelineJob(ctx, jobs[0].ID, "succeeded", ""); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	if err := m.EnqueuePipelineJob(ctx, job); err != nil {
		t.Fatalf("enqueue duplicate succeeded job: %v", err)
	}
	jobs, err = m.ListPipelineJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list jobs after duplicate: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != "succeeded" {
		t.Fatalf("succeeded job was duplicated or requeued: %#v", jobs)
	}
}

func TestMemoryEnqueuePipelineJobRequeuesFailedJobInPlace(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()
	if err := m.EnqueuePipelineJob(ctx, &PipelineJob{RunID: "run-retry", RulePath: "rules/old.yml", MaxAttempts: 3}); err != nil {
		t.Fatalf("enqueue initial job: %v", err)
	}
	claimed, err := m.ClaimPipelineJob(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if err := m.CompletePipelineJob(ctx, claimed.ID, "failed", "previous failure"); err != nil {
		t.Fatalf("mark job failed: %v", err)
	}

	if err := m.EnqueuePipelineJob(ctx, &PipelineJob{RunID: "run-retry", RulePath: "rules/new.yml", MaxAttempts: 5}); err != nil {
		t.Fatalf("requeue failed job: %v", err)
	}
	jobs, err := m.ListPipelineJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("requeue created a second job: %#v", jobs)
	}
	got := jobs[0]
	if got.ID != claimed.ID || got.Status != "queued" || got.Attempts != 0 || got.MaxAttempts != 5 ||
		got.RulePath != "rules/new.yml" || got.LastError != "" || got.LockedBy != "" || got.LockedAt != nil {
		t.Fatalf("failed job was not reset in place: %#v", got)
	}
}
