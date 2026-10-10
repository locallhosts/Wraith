package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/locallhosts/Wraith/backend-go/config"
)

// validateProductionConfig prevents accidental startup with development-only
// storage or disabled trust boundaries. Non-production environments retain
// the existing local-development defaults.
func validateProductionConfig(cfg config.Config, environment string) error {
	if !strings.EqualFold(strings.TrimSpace(environment), "production") {
		return nil
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("WRAITH_DATABASE_URL is required when WRAITH_ENV=production")
	}
	if strings.TrimSpace(cfg.WebhookSecret) == "" {
		return fmt.Errorf("WRAITH_WEBHOOK_SECRET is required when WRAITH_ENV=production")
	}
	if strings.TrimSpace(cfg.SigningPublicKey) == "" {
		return fmt.Errorf("WRAITH_SIGNING_PUBLIC_KEY is required when WRAITH_ENV=production")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.SigningPublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return fmt.Errorf("WRAITH_SIGNING_PUBLIC_KEY must be a base64-encoded Ed25519 public key")
	}
	if strings.TrimSpace(cfg.ESAddr) == "" {
		return fmt.Errorf("WRAITH_ES_ADDR is required when WRAITH_ENV=production")
	}
	if strings.TrimSpace(cfg.Neo4jAddr) == "" {
		return fmt.Errorf("WRAITH_NEO4J_ADDR is required when WRAITH_ENV=production")
	}
	return nil
}
