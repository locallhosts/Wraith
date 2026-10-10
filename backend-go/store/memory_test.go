package store

import (
	"context"
	"testing"
)

func TestGetAPIKeyReturnsDefensiveCopy(t *testing.T) {
	m := NewMemoryStore()
	keyHash := sha256Hex("wraith-dev-admin-key")

	key, err := m.GetAPIKey(context.Background(), keyHash)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	key.Role = "viewer"
	key.Label = "mutated"
	key.Revoked = true

	again, err := m.GetAPIKey(context.Background(), keyHash)
	if err != nil {
		t.Fatalf("GetAPIKey second read: %v", err)
	}
	if again.Role != "admin" || again.Label != "dev-admin" || again.Revoked {
		t.Fatalf("mutating returned key changed stored state: %#v", again)
	}
}

func TestAuditEntriesAreIsolatedFromCallers(t *testing.T) {
	m := NewMemoryStore()
	entry := &AuditEntry{Actor: "analyst", Action: "run.approve", Resource: "run-1", Detail: "approved"}
	if err := m.AppendAudit(context.Background(), entry); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}

	entry.Detail = "tampered after append"
	entries, err := m.ListAudit(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if len(entries) != 1 || entries[0].Detail != "approved" {
		t.Fatalf("caller mutation changed stored audit entry: %#v", entries)
	}

	entries[0].Detail = "tampered from read result"
	again, err := m.ListAudit(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAudit second read: %v", err)
	}
	if len(again) != 1 || again[0].Detail != "approved" {
		t.Fatalf("mutating returned audit entry changed stored record: %#v", again)
	}
}
