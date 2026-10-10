package main

import (
	"testing"

	"github.com/locallhosts/Wraith/backend-go/auth"
)

func TestRateLimitIdentityKeyUsesUniqueKeyID(t *testing.T) {
	a := rateLimitIdentityKey(auth.Identity{KeyID: "key-1", Label: "shared-label"})
	b := rateLimitIdentityKey(auth.Identity{KeyID: "key-2", Label: "shared-label"})
	if a == b {
		t.Fatalf("different API keys with the same label shared a bucket: %q", a)
	}
	if a != "api-key:key-1" || b != "api-key:key-2" {
		t.Fatalf("unexpected bucket keys: %q, %q", a, b)
	}
}

func TestRateLimitIdentityKeyFallsBackForLegacyIdentity(t *testing.T) {
	if got := rateLimitIdentityKey(auth.Identity{Label: "legacy-label"}); got != "label:legacy-label" {
		t.Fatalf("unexpected fallback key: %q", got)
	}
	if got := rateLimitIdentityKey(auth.Identity{}); got != "" {
		t.Fatalf("anonymous identity should use IP fallback, got %q", got)
	}
}
