// Package config centralizes environment-based configuration so main.go
// and tests aren't scattered with os.Getenv calls, and so the full set of
// knobs a production operator needs is documented in one place (see also
// .env.example at the repo root).
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port              string
	RulesDir          string
	OutputDir         string // where run_pipeline.py writes report.json/attestation.json per run
	WebhookSecret     string
	DatabaseURL       string // if empty, falls back to in-memory store (dev only)
	ESAddr            string
	Neo4jAddr         string
	SlackWebhookURL   string
	SigningPublicKey  string // base64 ed25519 public key trusted for deploy verification
	MetricsEnabled    bool
	RequestsPerMinute int // rate limit per API key
}

func Load() Config {
	return Config{
		Port:              envOr("WRAITH_PORT", "8080"),
		RulesDir:          envOr("WRAITH_RULES_DIR", "./rules"),
		OutputDir:         envOr("WRAITH_OUTPUT_DIR", "engine-python/output"),
		WebhookSecret:     os.Getenv("WRAITH_WEBHOOK_SECRET"),
		DatabaseURL:       os.Getenv("WRAITH_DATABASE_URL"),
		ESAddr:            envOr("WRAITH_ES_ADDR", "http://localhost:9200"),
		Neo4jAddr:         envOr("WRAITH_NEO4J_ADDR", "bolt://localhost:7687"),
		SlackWebhookURL:   os.Getenv("WRAITH_SLACK_WEBHOOK_URL"),
		SigningPublicKey:  os.Getenv("WRAITH_SIGNING_PUBLIC_KEY"),
		MetricsEnabled:    envBool("WRAITH_METRICS_ENABLED", true),
		RequestsPerMinute: envInt("WRAITH_RATE_LIMIT_RPM", 120),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
