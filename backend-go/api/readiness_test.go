package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/locallhosts/Wraith/backend-go/store"
)

type failingPingStore struct {
	store.Store
	err error
}

func (s failingPingStore) Ping(context.Context) error { return s.err }

func TestReadinessRedactsInternalDependencyErrors(t *testing.T) {
	internal := errors.New("dial tcp postgres.internal:5432: password=secret-token connection refused")
	s := testServer()
	s.Store = failingPingStore{Store: store.NewMemoryStore(), err: internal}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret-token") ||
		strings.Contains(rec.Body.String(), "postgres.internal") ||
		strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("readiness response leaked internal dependency details: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "dependency check failed") {
		t.Fatalf("expected stable public error message, got %s", rec.Body.String())
	}
}
