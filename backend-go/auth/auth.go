// Package auth implements API-key authentication and role-based access
// control for the WRAITH API. Keys are never stored in plaintext — only
// their SHA-256 hash is persisted (store.APIKey.KeyHash) — mirroring how
// GitHub, Stripe, etc. handle API credentials.
//
// Roles, least to most privileged:
//
//	viewer   - read-only: list/view runs and audit log
//	analyst  - viewer + trigger manual lints/runs
//	lead     - analyst + approve a passing run for production deploy
//	admin    - lead + manage API keys, view raw audit log with IPs
package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/store"
)

var roleRank = map[string]int{
	"viewer":  1,
	"analyst": 2,
	"lead":    3,
	"admin":   4,
}

const ctxKeyIdentity = "wraith_identity"

// Identity is attached to the gin.Context for downstream handlers and the
// audit logger to read.
type Identity struct {
	Label string
	Role  string
}

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Middleware validates the `Authorization: Bearer <key>` header against
// the store and attaches the resulting Identity to the request context.
// It does not enforce a minimum role — pair with RequireRole for that.
func Middleware(s store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}
		raw := strings.TrimPrefix(header, prefix)
		hash := HashKey(raw)

		key, err := s.GetAPIKey(c.Request.Context(), hash)
		if err != nil || key.Revoked {
			// Constant-time-ish path: we still hash+lookup even on a bad
			// key so response timing doesn't leak whether a prefix was
			// close to valid.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked API key"})
			return
		}

		c.Set(ctxKeyIdentity, Identity{Label: key.Label, Role: key.Role})
		c.Next()
	}
}

// RequireRole aborts the request with 403 unless the authenticated
// identity's role meets or exceeds minRole in privilege.
func RequireRole(minRole string) gin.HandlerFunc {
	minRank, ok := roleRank[minRole]
	if !ok {
		panic("auth: unknown role in RequireRole: " + minRole)
	}
	return func(c *gin.Context) {
		id, ok := GetIdentity(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		rank, ok := roleRank[id.Role]
		if !ok || rank < minRank {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":         "insufficient role",
				"required_role": minRole,
				"your_role":     id.Role,
			})
			return
		}
		c.Next()
	}
}

// GetIdentity reads the Identity attached by Middleware.
func GetIdentity(c *gin.Context) (Identity, bool) {
	v, ok := c.Get(ctxKeyIdentity)
	if !ok {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}

// ConstantTimeEqual is exposed for anywhere else in the codebase that
// needs to compare secrets (e.g. webhook secrets) without timing leaks.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// IdentityFromContext lets non-gin code (e.g. the pipeline trigger
// goroutine) carry an identity forward for audit logging.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKeyIdentity).(Identity)
	return id, ok
}
