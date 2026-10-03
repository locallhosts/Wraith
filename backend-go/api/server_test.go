package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/locallhosts/Wraith/backend-go/store"
)

func testServer() *Server {
	return &Server{
		Store:            store.NewMemoryStore(),
		PublicPlayground: true,
		PublicOrigins:    map[string]bool{"https://playground.example.com": true},
	}
}

func TestPublicPlaygroundIsUnauthenticated(t *testing.T) {
	r := NewRouter(testServer())
	req := httptest.NewRequest(http.MethodPost, "/playground/validate", strings.NewReader(`{"rule": "title: test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("public playground unexpectedly requires authentication: %d", rec.Code)
	}
	if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected validation response, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestControlPlaneRemainsProtected(t *testing.T) {
	r := NewRouter(testServer())
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected control plane to return 401, got %d", rec.Code)
	}
}

func TestPublicPlaygroundRejectsOversizedRule(t *testing.T) {
	r := NewRouter(testServer())
	body := `{"rule":"` + strings.Repeat("A", 256*1024+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/playground/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPublicCORSUsesAllowlist(t *testing.T) {
	r := NewRouter(testServer())

	req := httptest.NewRequest(http.MethodOptions, "/playground/validate", nil)
	req.Header.Set("Origin", "https://playground.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected allowed preflight to return 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://playground.example.com" {
		t.Fatalf("unexpected CORS origin %q", got)
	}

	req = httptest.NewRequest(http.MethodOptions, "/playground/validate", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed preflight to return 403, got %d", rec.Code)
	}
}
