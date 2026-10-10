package config

import (
	"reflect"
	"testing"
)

func TestLoadTrustedProxiesParsesExplicitRanges(t *testing.T) {
	t.Setenv("WRAITH_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8,")
	got := Load().TrustedProxies
	want := []string{"127.0.0.1", "10.0.0.0/8"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected trusted proxies: got %#v want %#v", got, want)
	}
}

func TestLoadTrustedProxiesDefaultsToEmpty(t *testing.T) {
	t.Setenv("WRAITH_TRUSTED_PROXIES", "")
	if got := Load().TrustedProxies; len(got) != 0 {
		t.Fatalf("trusted proxies should default to empty, got %#v", got)
	}
}
