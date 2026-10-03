package store

import (
	"context"
	"sort"
	"time"
)

func (m *MemoryStore) AppendRunEvent(_ context.Context, e *RunEvent) error {
	m.mu.Lock(); defer m.mu.Unlock()
	e.ID = int64(len(m.events[e.RunID]) + 1)
	e.CreatedAt = time.Now()
	cp := *e
	m.events[e.RunID] = append(m.events[e.RunID], &cp)
	return nil
}

func (m *MemoryStore) ListRunEvents(_ context.Context, runID string, limit int) ([]*RunEvent, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	items := m.events[runID]
	out := make([]*RunEvent, 0, len(items))
	for _, e := range items { cp := *e; out = append(out, &cp) }
	sort.Slice(out, func(i,j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit { out = out[:limit] }
	return out, nil
}

func (m *MemoryStore) EnqueuePipelineJob(_ context.Context, j *PipelineJob) error {
	m.mu.Lock(); defer m.mu.Unlock()
	for _, existing := range m.jobs {
		if existing.RunID == j.RunID && (existing.Status == "queued" || existing.Status == "running") { return nil }
	}
	m.nextJobID++
	cp := *j
	cp.ID = m.nextJobID
	if cp.Status == "" { cp.Status = "queued" }
	if cp.MaxAttempts <= 0 { cp.MaxAttempts = 3 }
	if cp.AvailableAt.IsZero() { cp.AvailableAt = time.Now() }
	cp.CreatedAt = time.Now(); cp.UpdatedAt = cp.CreatedAt
	m.jobs[cp.ID] = &cp
	return nil
}

func (m *MemoryStore) GetPipelineJob(_ context.Context, id int64) (*PipelineJob, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	j, ok := m.jobs[id]; if !ok { return nil, ErrNotFound }
	cp := *j; return &cp, nil
}

func (m *MemoryStore) ListPipelineJobs(_ context.Context, limit int) ([]*PipelineJob, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	out := make([]*PipelineJob, 0, len(m.jobs))
	for _, j := range m.jobs { cp := *j; out = append(out, &cp) }
	sort.Slice(out, func(i,j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit { out = out[:limit] }
	return out, nil
}

func (m *MemoryStore) RetryPipelineJob(_ context.Context, id int64) error {
	m.mu.Lock(); defer m.mu.Unlock()
	j, ok := m.jobs[id]; if !ok { return ErrNotFound }
	j.Status="queued"; j.Attempts=0; j.LastError=""; j.LockedBy=""; j.LockedAt=nil; j.AvailableAt=time.Now(); j.UpdatedAt=time.Now()
	return nil
}

func (m *MemoryStore) ClaimPipelineJob(_ context.Context, workerID string, lease time.Duration) (*PipelineJob, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	now:=time.Now(); var selected *PipelineJob
	for _, j := range m.jobs {
		if j.Status=="running" && j.LockedAt!=nil && now.Sub(*j.LockedAt)>lease { j.Status="queued"; j.LockedBy=""; j.LockedAt=nil }
		if j.Status=="queued" && !j.AvailableAt.After(now) && (selected==nil || j.ID<selected.ID) { selected=j }
	}
	if selected==nil { return nil, ErrNoJobAvailable }
	selected.Status="running"; selected.Attempts++; selected.LockedBy=workerID; t:=now; selected.LockedAt=&t; selected.UpdatedAt=now
	cp:=*selected; return &cp,nil
}

func (m *MemoryStore) CompletePipelineJob(_ context.Context,id int64,status,lastError string) error {
	m.mu.Lock(); defer m.mu.Unlock()
	j,ok:=m.jobs[id]; if !ok{return ErrNotFound}; j.Status=status; j.LastError=lastError; j.LockedBy=""; j.LockedAt=nil; j.UpdatedAt=time.Now(); return nil
}

func (m *MemoryStore) RequeuePipelineJob(_ context.Context,id int64,delay time.Duration,lastError string) error {
	m.mu.Lock(); defer m.mu.Unlock()
	j,ok:=m.jobs[id]; if !ok{return ErrNotFound}; j.Status="queued"; j.LastError=lastError; j.LockedBy=""; j.LockedAt=nil; j.AvailableAt=time.Now().Add(delay); j.UpdatedAt=time.Now(); return nil
}

func (m *MemoryStore) CancelPipelineJob(_ context.Context,runID string) error {
	m.mu.Lock(); defer m.mu.Unlock()
	for _, j := range m.jobs { if j.RunID==runID && (j.Status=="queued"||j.Status=="running") { j.Status="cancelled"; j.LastError="cancelled by operator"; j.LockedBy=""; j.LockedAt=nil; j.UpdatedAt=time.Now() } }
	return nil
}
