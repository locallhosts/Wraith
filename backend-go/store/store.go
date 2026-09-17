// Package store defines the persistence boundary for pipeline run state.
package store

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrNoJobAvailable = errors.New("no pipeline job available")

type RunStatus struct {
	RunID      string     `json:"run_id"`
	RulePath   string     `json:"rule_path"`
	RuleID     string     `json:"rule_id"`
	RuleTitle  string     `json:"rule_title"`
	Repo       string     `json:"repo"`
	PRNumber   int        `json:"pr_number"`
	Stage      string     `json:"stage"`
	Passed     *bool      `json:"passed,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type RunEvent struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	Stage     string    `json:"stage"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type PipelineJob struct {
	ID          int64      `json:"id"`
	RunID       string     `json:"run_id"`
	RulePath    string     `json:"rule_path"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	AvailableAt time.Time  `json:"available_at"`
	LockedBy    string     `json:"locked_by,omitempty"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AuditEntry struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	ActorRole string    `json:"actor_role"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

type APIKey struct {
	KeyHash   string    `json:"-"`
	Label     string    `json:"label"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

type Store interface {
	PutRun(ctx context.Context, r *RunStatus) error
	GetRun(ctx context.Context, runID string) (*RunStatus, error)
	ListRuns(ctx context.Context, limit int) ([]*RunStatus, error)
	AppendRunEvent(ctx context.Context, e *RunEvent) error
	ListRunEvents(ctx context.Context, runID string, limit int) ([]*RunEvent, error)
	EnqueuePipelineJob(ctx context.Context, j *PipelineJob) error
	ClaimPipelineJob(ctx context.Context, workerID string, lease time.Duration) (*PipelineJob, error)
	CompletePipelineJob(ctx context.Context, jobID int64, status, lastError string) error
	RequeuePipelineJob(ctx context.Context, jobID int64, delay time.Duration, lastError string) error
	CancelPipelineJob(ctx context.Context, runID string) error
	AppendAudit(ctx context.Context, e *AuditEntry) error
	ListAudit(ctx context.Context, limit int) ([]*AuditEntry, error)
	GetAPIKey(ctx context.Context, keyHash string) (*APIKey, error)
	Ping(ctx context.Context) error
	Close() error
}
