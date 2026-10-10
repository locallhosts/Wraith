package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/locallhosts/Wraith/backend-go/config"
)

func validProductionConfig() config.Config {
	return config.Config{
		DatabaseURL:      "postgres://wraith:secret@db/wraith?sslmode=require",
		WebhookSecret:    "configured-secret",
		SigningPublicKey: base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		ESAddr:           "https://elasticsearch.example.internal",
		Neo4jAddr:        "neo4j+s://neo4j.example.internal",
	}
}

func TestValidateProductionConfigAcceptsCompleteConfig(t *testing.T) {
	if err := validateProductionConfig(validProductionConfig(), "production"); err != nil {
		t.Fatalf("valid production configuration rejected: %v", err)
	}
}

func TestValidateProductionConfigRequiresDurableStore(t *testing.T) {
	cfg := validProductionConfig()
	cfg.DatabaseURL = ""
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "WRAITH_DATABASE_URL") {
		t.Fatalf("expected database requirement error, got %v", err)
	}
}

func TestValidateProductionConfigRequiresWebhookSecret(t *testing.T) {
	cfg := validProductionConfig()
	cfg.WebhookSecret = ""
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "WRAITH_WEBHOOK_SECRET") {
		t.Fatalf("expected webhook secret requirement error, got %v", err)
	}
}

func TestValidateProductionConfigRequiresValidSigningKey(t *testing.T) {
	cfg := validProductionConfig()
	cfg.SigningPublicKey = ""
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "WRAITH_SIGNING_PUBLIC_KEY") {
		t.Fatalf("expected signing key requirement error, got %v", err)
	}

	cfg.SigningPublicKey = base64.StdEncoding.EncodeToString([]byte("too short"))
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "base64-encoded Ed25519 public key") {
		t.Fatalf("expected signing key format error, got %v", err)
	}
}

func TestValidateProductionConfigRequiresValidationServices(t *testing.T) {
	cfg := validProductionConfig()
	cfg.ESAddr = ""
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "WRAITH_ES_ADDR") {
		t.Fatalf("expected Elasticsearch requirement error, got %v", err)
	}
	cfg = validProductionConfig()
	cfg.Neo4jAddr = ""
	if err := validateProductionConfig(cfg, "production"); err == nil || !strings.Contains(err.Error(), "WRAITH_NEO4J_ADDR") {
		t.Fatalf("expected Neo4j requirement error, got %v", err)
	}
}

func TestValidateProductionConfigKeepsDevelopmentFlexible(t *testing.T) {
	if err := validateProductionConfig(config.Config{}, "development"); err != nil {
		t.Fatalf("development config should retain local defaults: %v", err)
	}
	if err := validateProductionConfig(config.Config{}, ""); err != nil {
		t.Fatalf("unset environment should retain local defaults: %v", err)
	}
}
