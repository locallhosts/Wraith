package auth

import "testing"

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
