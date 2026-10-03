// Package store defines the persistence boundary for pipeline run state.
// backend-go/api uses this interface exclusively, so swapping the
// in-memory dev implementation for the Postgres-backed production one is a
// one-line change in main.go.
package store

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrNoJobAvailable = errors.New("no pipeline job available")

// RunStatus tracks one end-to-end pipeline execution for a rule/PR.
type RunStatus struct {
	RunID      string     `json:"run_id"`
	RulePath   string     `json:"rule_path"`
	RuleID     string     `json:"rule_id"`
	RuleTitle  string     `json:"rule_title"`
	Repo       string     `json:"repo"`
	PRNumber   int        `json:"pr_number"`
	Stage      string     `json:"stage"` // lint | provision | simulate | validate | soar | done | failed
	Passed     *bool      `json:"passed,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	DeployedAt *time.Time `json:"deployed_at,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AuditEntry is one append-only record of a state-changing action, kept
// for compliance/SOC2-style audit trails.
type RunStage struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // running | passed | failed | skipped
	Reason    string    `json:"reason,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type RunEvent struct {
	ID int64 `json:"id"`
	RunID string `json:"run_id"`
	Stage string `json:"stage"`
	Level string `json:"level"`
	Message string `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type PipelineJob struct {
	ID int64 `json:"id"`
	RunID string `json:"run_id"`
	RulePath string `json:"rule_path"`
	Status string `json:"status"`
	Attempts int `json:"attempts"`
	MaxAttempts int `json:"max_attempts"`
	AvailableAt time.Time `json:"available_at"`
	LockedBy string `json:"locked_by,omitempty"`
	LockedAt *time.Time `json:"locked_at,omitempty"`
	LastError string `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditEntry struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`      // API key label or "system"
	ActorRole string    `json:"actor_role"` // viewer | analyst | lead | admin
	Action    string    `json:"action"`     // e.g. "run.approve", "rule.deploy"
	Resource  string    `json:"resource"`   // e.g. run_id or rule_id
	Detail    string    `json:"detail"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// APIKey represents one issued credential and the role it carries.
type APIKey struct {
	ID        string    `json:"id"`
	KeyHash   string    `json:"-"` // never serialized
	Label     string    `json:"label"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// Store is the full persistence surface the API server depends on.
type Store interface {
	PutRun(ctx context.Context, r *RunStatus) error
	GetRun(ctx context.Context, runID string) (*RunStatus, error)
	ListRuns(ctx context.Context, limit int) ([]*RunStatus, error)
	AppendRunStage(ctx context.Context, stage *RunStage) error
	UpdateRunStage(ctx context.Context, stage *RunStage) error
	ListRunStages(ctx context.Context, runID string) ([]*RunStage, error)

	AppendRunEvent(ctx context.Context, e *RunEvent) error
	ListRunEvents(ctx context.Context, runID string, limit int) ([]*RunEvent, error)
	EnqueuePipelineJob(ctx context.Context, j *PipelineJob) error
	GetPipelineJob(ctx context.Context, jobID int64) (*PipelineJob, error)
	ListPipelineJobs(ctx context.Context, limit int) ([]*PipelineJob, error)
	RetryPipelineJob(ctx context.Context, jobID int64) error
	ClaimPipelineJob(ctx context.Context, workerID string, lease time.Duration) (*PipelineJob, error)
	CompletePipelineJob(ctx context.Context, jobID int64, status, lastError string) error
	RequeuePipelineJob(ctx context.Context, jobID int64, delay time.Duration, lastError string) error
	CancelPipelineJob(ctx context.Context, runID string) error

	AppendAudit(ctx context.Context, e *AuditEntry) error
	ListAudit(ctx context.Context, limit int) ([]*AuditEntry, error)

	GetAPIKey(ctx context.Context, keyHash string) (*APIKey, error)
	ListAPIKeys(ctx context.Context) ([]*APIKey, error)
	CreateAPIKey(ctx context.Context, id, keyHash, label, role string) error
	RevokeAPIKey(ctx context.Context, id string) error

	// Ping verifies connectivity, used by the /readyz endpoint.
	Ping(ctx context.Context) error
	Close() error
}
