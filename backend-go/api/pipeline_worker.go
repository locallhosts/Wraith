package api

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/locallhosts/Wraith/backend-go/store"
)

// PipelineWorkerConfig controls the durable execution loop. Jobs are claimed
// from the store, so multiple API processes can consume work without running
// the same queued job concurrently.
type PipelineWorkerConfig struct {
	Workers     int
	PollInterval time.Duration
	Lease       time.Duration
	RetryBackoff time.Duration
}

func (c PipelineWorkerConfig) normalized() PipelineWorkerConfig {
	if c.Workers < 1 { c.Workers = 1 }
	if c.PollInterval <= 0 { c.PollInterval = time.Second }
	if c.Lease <= 0 { c.Lease = 20 * time.Minute }
	if c.RetryBackoff <= 0 { c.RetryBackoff = 10 * time.Second }
	return c
}

func StartPipelineWorkers(parent context.Context, s *Server, cfg PipelineWorkerConfig) context.CancelFunc {
	cfg = cfg.normalized()
	ctx, cancel := context.WithCancel(parent)
	for i := 0; i < cfg.Workers; i++ {
		workerID := fmt.Sprintf("wraith-%d-%d", time.Now().UnixNano(), i)
		go pipelineWorker(ctx, s, workerID, cfg)
	}
	return cancel
}

func pipelineWorker(ctx context.Context, s *Server, workerID string, cfg PipelineWorkerConfig) {
	jitter := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		if ctx.Err() != nil { return }
		job, err := s.Store.ClaimPipelineJob(ctx, workerID, cfg.Lease)
		if err == nil {
			executePipelineJob(ctx, s, workerID, job, cfg)
			continue
		}
		if err != store.ErrNoJobAvailable && s.Log != nil {
			s.Log.Error("pipeline worker claim failed", "worker_id", workerID, "error", err)
		}
		d := cfg.PollInterval
		if d > 250*time.Millisecond { d += time.Duration(jitter.Int63n(int64(d / 4))) }
		t := time.NewTimer(d)
		select { case <-ctx.Done(): t.Stop(); return; case <-t.C: }
	}
}

func executePipelineJob(ctx context.Context, s *Server, workerID string, job *store.PipelineJob, cfg PipelineWorkerConfig) {
	if s.Log != nil { s.Log.Info("pipeline job claimed", "worker_id", workerID, "job_id", job.ID, "run_id", job.RunID, "attempt", job.Attempts) }
	defer func() {
		if recovered := recover(); recovered != nil {
			reason := fmt.Sprintf("pipeline worker panic: %v", recovered)
			if s.Log != nil { s.Log.Error("pipeline worker panic", "worker_id", workerID, "run_id", job.RunID, "error", reason, "stack", string(debug.Stack())) }
			_ = s.Store.AppendRunEvent(context.Background(), &store.RunEvent{RunID: job.RunID, Stage: "failed", Level: "error", Message: reason})
			if job.Attempts < job.MaxAttempts {
				_ = s.Store.RequeuePipelineJob(context.Background(), job.ID, cfg.RetryBackoff*time.Duration(job.Attempts), reason)
			} else {
				_ = s.Store.CompletePipelineJob(context.Background(), job.ID, "failed", reason)
			}
		}
	}()

	if s.PipelineTrigger == nil {
		reason := "pipeline trigger is not configured"
		_ = s.Store.CompletePipelineJob(ctx, job.ID, "failed", reason)
		return
	}

	s.PipelineTrigger(job.RunID, job.RulePath)
	if ctx.Err() != nil { return }
	run, err := s.Store.GetRun(ctx, job.RunID)
	if err != nil {
		_ = s.Store.CompletePipelineJob(context.Background(), job.ID, "failed", fmt.Sprintf("reading final run state: %v", err))
		return
	}
	if run.Stage == "done" || run.Stage == "failed" {
		status := "succeeded"
		if run.Stage == "failed" && job.Attempts < job.MaxAttempts && run.Passed == nil {
			status = "failed"
		}
		_ = s.Store.CompletePipelineJob(context.Background(), job.ID, status, run.Reason)
		return
	}

	reason := fmt.Sprintf("pipeline worker returned with non-terminal stage %q", run.Stage)
	if job.Attempts < job.MaxAttempts {
		_ = s.Store.RequeuePipelineJob(context.Background(), job.ID, cfg.RetryBackoff*time.Duration(job.Attempts), reason)
	} else {
		_ = s.Store.CompletePipelineJob(context.Background(), job.ID, "failed", reason)
	}
}
