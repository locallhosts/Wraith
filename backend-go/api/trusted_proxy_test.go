package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRouterDoesNotTrustForwardedIPByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(testServer())
	r.GET("/test-client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	req := httptest.NewRequest(http.MethodGet, "/test-client-ip", nil)
	req.RemoteAddr = "192.0.2.10:4567"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "192.0.2.10" {
		t.Fatalf("spoofed X-Forwarded-For was trusted by default: got %q", rec.Body.String())
	}
}


func TestValidateTrustedProxyList(t *testing.T) {
	tests := []struct {
		name string
		proxies []string
		wantErr bool
	}{
		{name: "empty means no trusted proxies", proxies: nil},
		{name: "single proxy IP", proxies: []string{"127.0.0.1"}},
		{name: "specific CIDR", proxies: []string{"10.0.0.0/8", "2001:db8::/32"}},
		{name: "IPv4 wildcard", proxies: []string{"0.0.0.0/0"}, wantErr: true},
		{name: "IPv6 wildcard", proxies: []string{"::/0"}, wantErr: true},
		{name: "invalid CIDR", proxies: []string{"10.0.0.0/77"}, wantErr: true},
		{name: "invalid IP", proxies: []string{"proxy.example.com"}, wantErr: true},
		{name: "empty entry", proxies: []string{""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTrustedProxyList(tt.proxies)
			if tt.wantErr && err == nil {
				t.Fatal("expected invalid proxy list to be rejected")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
