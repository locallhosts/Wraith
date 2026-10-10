package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHashKey_Deterministic(t *testing.T) {
	a := HashKey("wraith-dev-admin-key")
	b := HashKey("wraith-dev-admin-key")
	if a != b {
		t.Fatalf("expected deterministic hash, got %s vs %s", a, b)
	}
	if len(a) != 64 { // hex-encoded sha256
		t.Fatalf("expected 64-char hex digest, got %d chars", len(a))
	}
}

func TestHashKey_DifferentInputsDifferentHashes(t *testing.T) {
	if HashKey("key-a") == HashKey("key-b") {
		t.Fatal("expected different keys to hash differently")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("secret", "secret") {
		t.Fatal("expected equal secrets to compare equal")
	}
	if ConstantTimeEqual("secret", "different") {
		t.Fatal("expected different secrets to compare unequal")
	}
}

func TestRequireRoleEnforcesPrivilegeHierarchy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		identity  string
		required  string
		wantCode  int
	}{
		{name: "missing identity", required: "viewer", wantCode: http.StatusUnauthorized},
		{name: "viewer can read", identity: "viewer", required: "viewer", wantCode: http.StatusNoContent},
		{name: "viewer cannot trigger analyst action", identity: "viewer", required: "analyst", wantCode: http.StatusForbidden},
		{name: "analyst can trigger analyst action", identity: "analyst", required: "analyst", wantCode: http.StatusNoContent},
		{name: "analyst cannot approve", identity: "analyst", required: "lead", wantCode: http.StatusForbidden},
		{name: "lead can approve", identity: "lead", required: "lead", wantCode: http.StatusNoContent},
		{name: "lead cannot manage keys", identity: "lead", required: "admin", wantCode: http.StatusForbidden},
		{name: "admin inherits lead privilege", identity: "admin", required: "lead", wantCode: http.StatusNoContent},
		{name: "unknown role fails closed", identity: "owner", required: "viewer", wantCode: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(c *gin.Context) {
				if tt.identity != "" {
					c.Set(ctxKeyIdentity, Identity{Label: "test", Role: tt.identity})
				}
			}, RequireRole(tt.required), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("RequireRole(%q) with identity %q: got status %d, want %d; body=%s",
					tt.required, tt.identity, rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}

func TestRequireRoleRejectsMalformedIdentityContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set(ctxKeyIdentity, "not-an-Identity")
	}, RequireRole("viewer"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("malformed identity must fail closed with 401, got %d", rec.Code)
	}
}
