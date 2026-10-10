package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/auth"
	"github.com/locallhosts/Wraith/backend-go/store"
)

func routerWithTestKeys(t *testing.T) (*gin.Engine, map[string]string) {
	t.Helper()
	st := store.NewMemoryStore()
	rawKeys := map[string]string{}
	for _, role := range []string{"viewer", "analyst", "lead", "admin", "owner"} {
		raw := "test-key-" + role
		rawKeys[role] = raw
		if err := st.CreateAPIKey(context.Background(), "test-id-"+role, auth.HashKey(raw), "shared-label", role); err != nil {
			t.Fatalf("create %s key: %v", role, err)
		}
	}
	s := testServer()
	s.Store = st
	return NewRouter(s), rawKeys
}

func TestControlPlaneRouteRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, keys := routerWithTestKeys(t)
	tests := []struct {
		name string
		role string
		method string
		path string
		body string
		want int
	}{
		{name: "viewer cannot read raw audit", role: "viewer", method: http.MethodGet, path: "/audit", want: http.StatusForbidden},
		{name: "viewer cannot lint", role: "viewer", method: http.MethodPost, path: "/lint", want: http.StatusForbidden},
		{name: "analyst cannot approve", role: "analyst", method: http.MethodPost, path: "/runs/run-1/approve", want: http.StatusForbidden},
		{name: "lead cannot manage keys", role: "lead", method: http.MethodPost, path: "/api-keys", body: `{"label":"new-key","role":"viewer"}`, want: http.StatusForbidden},
		{name: "unknown role fails closed", role: "owner", method: http.MethodGet, path: "/runs", want: http.StatusForbidden},
		{name: "admin can read audit", role: "admin", method: http.MethodGet, path: "/audit", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+keys[tt.role])
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("got HTTP %d, want %d; body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}
