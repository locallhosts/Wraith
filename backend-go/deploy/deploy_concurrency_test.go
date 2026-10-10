package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

func testESClient(t *testing.T, handler http.HandlerFunc) (*elasticsearch.Client, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{server.URL}})
	if err != nil {
		server.Close()
		t.Fatalf("create Elasticsearch client: %v", err)
	}
	return client, server.Close
}

func TestReadCurrentVersionCapturesConcurrencyMetadata(t *testing.T) {
	client, closeServer := testESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"_source":{"rule_id":"rule-1"},"_seq_no":7,"_primary_term":2,"found":true}`))
	})
	defer closeServer()

	source, seqNo, primaryTerm, err := readCurrentVersion(context.Background(), client, "rule-1")
	if err != nil {
		t.Fatalf("readCurrentVersion: %v", err)
	}
	if seqNo != 7 || primaryTerm != 2 {
		t.Fatalf("unexpected version metadata: seq_no=%d primary_term=%d", seqNo, primaryTerm)
	}
	var rule map[string]string
	if err := json.Unmarshal(source, &rule); err != nil || rule["rule_id"] != "rule-1" {
		t.Fatalf("unexpected source %s, err=%v", source, err)
	}
}

func TestIndexDocumentConditionalRejectsStaleVersion(t *testing.T) {
	client, closeServer := testESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("if_seq_no") != "7" || r.URL.Query().Get("if_primary_term") != "2" {
			t.Errorf("conditional write omitted expected version: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"type":"version_conflict_engine_exception"}}`))
	})
	defer closeServer()

	err := indexDocumentConditional(context.Background(), client, ProductionIndex, "rule-1", map[string]string{"rule_id": "rule-1"}, 7, 2)
	if err != ErrConcurrentChange {
		t.Fatalf("expected ErrConcurrentChange, got %v", err)
	}
}

func TestDeleteDocumentConditionalRejectsStaleVersion(t *testing.T) {
	client, closeServer := testESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Query().Get("if_seq_no") != "11" || r.URL.Query().Get("if_primary_term") != "3" {
			t.Errorf("conditional delete omitted expected version: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"type":"version_conflict_engine_exception"}}`))
	})
	defer closeServer()

	err := deleteDocumentConditional(context.Background(), client, ProductionIndex, "rule-1", 11, 3)
	if err != ErrConcurrentChange {
		t.Fatalf("expected ErrConcurrentChange, got %v", err)
	}
}

func TestIndexDocumentCreateUsesCreateOnlyOperation(t *testing.T) {
	client, closeServer := testESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("op_type") != "create" {
			t.Errorf("expected op_type=create, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"created"}`))
	})
	defer closeServer()

	if err := indexDocumentCreate(context.Background(), client, ProductionIndex, "rule-1", map[string]string{"rule_id": "rule-1"}); err != nil {
		t.Fatalf("indexDocumentCreate: %v", err)
	}
}
