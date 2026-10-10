package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/store"
)

func TestMiddlewareAttachesStableAPIKeyID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", Middleware(store.NewMemoryStore()), func(c *gin.Context) {
		id, ok := GetIdentity(c)
		if !ok {
			t.Error("identity was not attached")
			c.Status(http.StatusInternalServerError)
			return
		}
		if id.KeyID != "dev-admin" || id.Label != "dev-admin" || id.Role != "admin" {
			t.Errorf("unexpected identity: %#v", id)
		}
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wraith-dev-admin-key")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
