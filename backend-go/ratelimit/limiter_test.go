package ratelimit

import (
	"testing"
	"time"
)

func TestAllowRefillNeverExceedsBucketCapacity(t *testing.T) {
	l := New(100, 1)
	key := "playground:127.0.0.1"
	l.buckets[key] = &bucket{tokens: 0, lastRefill: time.Now().Add(-time.Minute)}

	if !l.allow(key, l.playgroundRatePerMin, l.playgroundCapacity) {
		t.Fatal("expected refilled playground bucket to allow one request")
	}
	if got := l.buckets[key].tokens; got > l.playgroundCapacity {
		t.Fatalf("bucket refilled above configured capacity: got %.2f, capacity %.2f", got, l.playgroundCapacity)
	}
}

func TestAllowSeparatesKeys(t *testing.T) {
	l := New(1)
	if !l.allow("key-a", l.ratePerMin, l.capacity) {
		t.Fatal("first request should be allowed")
	}
	if l.allow("key-a", l.ratePerMin, l.capacity) {
		t.Fatal("second request for exhausted key should be rejected")
	}
	if !l.allow("key-b", l.ratePerMin, l.capacity) {
		t.Fatal("separate key should have its own bucket")
	}
}
