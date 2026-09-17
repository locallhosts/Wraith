// Package config centralizes environment-based configuration so main.go
// and tests aren't scattered with os.Getenv calls.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string
	RulesDir string
	OutputDir string
	WebhookSecret string
	DatabaseURL string
	ESAddr string
	Neo4jAddr string
	SlackWebhookURL string
	SigningPublicKey string
	MetricsEnabled bool
	RequestsPerMinute int
	PipelineWorkers int
	PipelinePollInterval time.Duration
	PipelineLease time.Duration
	PipelineRetryBackoff time.Duration
	PipelineMaxAttempts int
}

func Load() Config {
	return Config{
		Port: envOr("WRAITH_PORT", "8080"), RulesDir: envOr("WRAITH_RULES_DIR", "./rules"), OutputDir: envOr("WRAITH_OUTPUT_DIR", "engine-python/output"),
		WebhookSecret: os.Getenv("WRAITH_WEBHOOK_SECRET"), DatabaseURL: os.Getenv("WRAITH_DATABASE_URL"), ESAddr: envOr("WRAITH_ES_ADDR", "http://localhost:9200"), Neo4jAddr: envOr("WRAITH_NEO4J_ADDR", "bolt://localhost:7687"),
		SlackWebhookURL: os.Getenv("WRAITH_SLACK_WEBHOOK_URL"), SigningPublicKey: os.Getenv("WRAITH_SIGNING_PUBLIC_KEY"), MetricsEnabled: envBool("WRAITH_METRICS_ENABLED", true), RequestsPerMinute: envInt("WRAITH_RATE_LIMIT_RPM", 120),
		PipelineWorkers: envInt("WRAITH_PIPELINE_WORKERS", 2), PipelinePollInterval: envDuration("WRAITH_PIPELINE_POLL_INTERVAL", time.Second), PipelineLease: envDuration("WRAITH_PIPELINE_LEASE", 20*time.Minute), PipelineRetryBackoff: envDuration("WRAITH_PIPELINE_RETRY_BACKOFF", 10*time.Second), PipelineMaxAttempts: envInt("WRAITH_PIPELINE_MAX_ATTEMPTS", 3),
	}
}
func envOr(key,fallback string)string{if v:=os.Getenv(key);v!=""{return v};return fallback}
func envBool(key string,fallback bool)bool{v:=os.Getenv(key);if v==""{return fallback};b,err:=strconv.ParseBool(v);if err!=nil{return fallback};return b}
func envInt(key string,fallback int)int{v:=os.Getenv(key);if v==""{return fallback};i,err:=strconv.Atoi(v);if err!=nil||i<1{return fallback};return i}
func envDuration(key string,fallback time.Duration)time.Duration{v:=os.Getenv(key);if v==""{return fallback};d,err:=time.ParseDuration(v);if err!=nil||d<=0{return fallback};return d}
