package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.RWMutex
	runs   map[string]*RunStatus
	events map[string][]*RunEvent
	audit  []*AuditEntry
	keys   map[string]*APIKey
}

func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		runs:   map[string]*RunStatus{},
		events: map[string][]*RunEvent{},
		keys:   map[string]*APIKey{},
	}
	devKeyHash := sha256Hex("wraith-dev-admin-key")
	m.keys[devKeyHash] = &APIKey{KeyHash: devKeyHash, Label: "dev-admin", Role: "admin", CreatedAt: time.Now()}
	return m
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (m *MemoryStore) PutRun(_ context.Context, r *RunStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.UpdatedAt = time.Now()
	if r.StartedAt.IsZero() {
		if existing, ok := m.runs[r.RunID]; ok {
			r.StartedAt = existing.StartedAt
		} else {
			r.StartedAt = time.Now()
		}
	}
	cp := *r
	m.runs[r.RunID] = &cp
	return nil
}

func (m *MemoryStore) GetRun(_ context.Context, runID string) (*RunStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[runID]
	if !ok { return nil, ErrNotFound }
	cp := *r
	return &cp, nil
}

func (m *MemoryStore) ListRuns(_ context.Context, limit int) ([]*RunStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*RunStatus, 0, len(m.runs))
	for _, r := range m.runs { cp := *r; out = append(out, &cp) }
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	if limit > 0 && len(out) > limit { out = out[:limit] }
	return out, nil
}

func (m *MemoryStore) AppendRunEvent(_ context.Context, e *RunEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e.ID = int64(len(m.events[e.RunID]) + 1)
	e.CreatedAt = time.Now()
	cp := *e
	m.events[e.RunID] = append(m.events[e.RunID], &cp)
	return nil
}

func (m *MemoryStore) ListRunEvents(_ context.Context, runID string, limit int) ([]*RunEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.events[runID]
	out := make([]*RunEvent, 0, len(items))
	for _, e := range items { cp := *e; out = append(out, &cp) }
	if limit > 0 && len(out) > limit { out = out[len(out)-limit:] }
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 { out[i], out[j] = out[j], out[i] }
	return out, nil
}

func (m *MemoryStore) AppendAudit(_ context.Context, e *AuditEntry) error {
	m.mu.Lock(); defer m.mu.Unlock()
	e.ID = int64(len(m.audit) + 1); e.Timestamp = time.Now()
	cp := *e; m.audit = append(m.audit, &cp)
	return nil
}

func (m *MemoryStore) ListAudit(_ context.Context, limit int) ([]*AuditEntry, error) {
	m.mu.RLock(); defer m.mu.RUnlock()
	out := make([]*AuditEntry, len(m.audit)); copy(out, m.audit)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	if limit > 0 && len(out) > limit { out = out[:limit] }
	return out, nil
}

func (m *MemoryStore) GetAPIKey(_ context.Context, keyHash string) (*APIKey, error) {
	m.mu.RLock(); defer m.mu.RUnlock()
	k, ok := m.keys[keyHash]; if !ok { return nil, ErrNotFound }
	cp := *k
	return &cp, nil
}

func (m *MemoryStore) Ping(_ context.Context) error { return nil }
func (m *MemoryStore) Close() error { return nil }
