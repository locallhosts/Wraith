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

	AppendAudit(ctx context.Context, e *AuditEntry) error
	ListAudit(ctx context.Context, limit int) ([]*AuditEntry, error)

	GetAPIKey(ctx context.Context, keyHash string) (*APIKey, error)

	// Ping verifies connectivity, used by the /readyz endpoint.
	Ping(ctx context.Context) error
	Close() error
}
