// WRAITH backend — CLI + API server.
//
// Usage:
//
//	wraith serve                        # start the API server + webhook listener
//	wraith lint <dir>                   # lint all Sigma rules in a directory
//	wraith provision --run-id X         # spin up an ephemeral test environment
//	wraith teardown --run-id X          # tear it back down
//	wraith keygen                       # generate a new ed25519 signing keypair
//	wraith attest --report r.json --out a.json --signing-key <base64>
//	wraith verify --attestation a.json --rule rule.yml --public-key <base64>
//	wraith apikey create --label ci-bot --role analyst   # requires WRAITH_DATABASE_URL
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/api"
	"github.com/locallhosts/Wraith/backend-go/auth"
	"github.com/locallhosts/Wraith/backend-go/config"
	"github.com/locallhosts/Wraith/backend-go/linter"
	"github.com/locallhosts/Wraith/backend-go/notify"
	"github.com/locallhosts/Wraith/backend-go/orchestrator"
	"github.com/locallhosts/Wraith/backend-go/provenance"
	"github.com/locallhosts/Wraith/backend-go/ratelimit"
	"github.com/locallhosts/Wraith/backend-go/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: wraith <serve|lint|provision|teardown|keygen|attest|verify|apikey> [args]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		runServe()
	case "lint":
		runLint()
	case "provision":
		runProvision()
	case "teardown":
		runTeardown()
	case "keygen":
		runKeygen()
	case "attest":
		runAttest()
	case "verify":
		runVerify()
	case "apikey":
		runAPIKey()
	default:
		fmt.Printf("unknown command %q\n", os.Args[1])
		os.Exit(1)
	}
}

func runServe() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if cfg.WebhookSecret == "" {
		logger.Warn("WRAITH_WEBHOOK_SECRET is unset — webhook signature verification will reject all requests")
	}

	var st store.Store
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pg, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("failed to connect to postgres", "error", err)
			os.Exit(1)
		}
		st = pg
		logger.Info("using postgres store")
	} else {
		st = store.NewMemoryStore()
		logger.Warn("WRAITH_DATABASE_URL is unset — using in-memory store (NOT for production: no durability across restarts)")
	}
	defer st.Close()

	var slackNotifier *notify.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = notify.NewSlackNotifier(cfg.SlackWebhookURL)
	}

	var trustedKey ed25519.PublicKey
	if cfg.SigningPublicKey != "" {
		raw, err := base64.StdEncoding.DecodeString(cfg.SigningPublicKey)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			logger.Error("WRAITH_SIGNING_PUBLIC_KEY is set but invalid — deploy endpoint will be disabled", "error", err)
		} else {
			trustedKey = ed25519.PublicKey(raw)
		}
	} else {
		logger.Warn("WRAITH_SIGNING_PUBLIC_KEY is unset — /runs/:id/deploy will be disabled until configured")
	}

	limiter := ratelimit.New(cfg.RequestsPerMinute)

	srv := &api.Server{
		Store:             st,
		WebhookSecret:     cfg.WebhookSecret,
		RulesDir:          cfg.RulesDir,
		OutputDir:         cfg.OutputDir,
		Slack:             slackNotifier,
		TrustedSigningKey: trustedKey,
		Log:               logger,
		RateLimit: limiter.Middleware(func(c *gin.Context) string {
			if id, ok := auth.GetIdentity(c); ok {
				return id.Label
			}
			return ""
		}),
	}
	srv.PipelineTrigger = api.ManagedPythonPipelineTrigger(srv, cfg.ESAddr, cfg.Neo4jAddr)

	router := api.NewRouter(srv)
	logger.Info("wraith API listening", "port", cfg.Port, "rules_dir", cfg.RulesDir)
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func runLint() {
	dir := "./rules"
	strict := false
	for _, a := range os.Args[2:] {
		if a == "--strict" {
			strict = true
		} else if dir == "./rules" {
			dir = a
		}
	}
	results, err := linter.LintDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lint error:", err)
		os.Exit(1)
	}
	inGitHubActions := os.Getenv("GITHUB_ACTIONS") == "true"
	exitCode := 0
	for _, r := range results {
		rulePassed := r.Passed()
		if strict {
			rulePassed = r.PassedStrict()
		}
		status := "PASS"
		if !rulePassed {
			status = "FAIL"
			exitCode = 1
		}
		fmt.Printf("[%s] %s\n", status, r.Path)
		for _, issue := range r.Issues {
			fmt.Printf("    (%s) %s: %s\n", issue.Severity, issue.Field, issue.Message)
			if inGitHubActions {
				cmd := "notice"
				if issue.Severity == "error" || (strict && issue.Severity == "warning") {
					cmd = "error"
				}
				fmt.Printf("::%s file=%s::%s: %s\n", cmd, r.Path, issue.Field, issue.Message)
			}
		}
	}
	os.Exit(exitCode)
}

func runProvision() {
	runID := flagValue("--run-id")
	if runID == "" {
		fmt.Fprintln(os.Stderr, "--run-id is required")
		os.Exit(1)
	}
	cli, err := orchestrator.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "docker client error:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	env, err := orchestrator.Provision(ctx, cli, runID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision error:", err)
		os.Exit(1)
	}
	fmt.Printf("elasticsearch: http://localhost:%s\n", env.ElasticPort)
	fmt.Printf("neo4j http:    http://localhost:%s\n", env.Neo4jHTTPPort)
	fmt.Printf("neo4j bolt:    bolt://localhost:%s\n", env.Neo4jBoltPort)
}

func runTeardown() {
	runID := flagValue("--run-id")
	if runID == "" {
		fmt.Fprintln(os.Stderr, "--run-id is required")
		os.Exit(1)
	}
	cli, err := orchestrator.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "docker client error:", err)
		os.Exit(1)
	}
	if err := orchestrator.TeardownByRunID(context.Background(), cli, runID); err != nil {
		fmt.Fprintln(os.Stderr, "teardown error:", err)
		os.Exit(1)
	}
	fmt.Println("torn down run", runID)
}

func runKeygen() {
	pub, priv, err := provenance.GenerateKeypair()
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen error:", err)
		os.Exit(1)
	}
	fmt.Println("# Store the private key as a CI secret (e.g. WRAITH_SIGNING_PRIVATE_KEY).")
	fmt.Println("# Store the public key wherever the deploy step runs (WRAITH_SIGNING_PUBLIC_KEY).")
	fmt.Println("# This is printed ONCE. It is not saved anywhere by this tool.")
	fmt.Println()
	fmt.Println("WRAITH_SIGNING_PUBLIC_KEY=" + base64.StdEncoding.EncodeToString(pub))
	fmt.Println("WRAITH_SIGNING_PRIVATE_KEY=" + base64.StdEncoding.EncodeToString(priv))
}

type reportFile struct {
	RunID  string                     `json:"run_id"`
	RuleID string                     `json:"rule_id"`
	Stages map[string]json.RawMessage `json:"stages"`
	Passed bool                       `json:"passed"`
}

func runAttest() {
	reportPath := flagValue("--report")
	rulePath := flagValue("--rule")
	outPath := flagValue("--out")
	signingKeyB64 := flagValue("--signing-key")
	if signingKeyB64 == "" {
		signingKeyB64 = os.Getenv("WRAITH_SIGNING_PRIVATE_KEY")
	}
	if reportPath == "" || rulePath == "" || outPath == "" || signingKeyB64 == "" {
		fmt.Fprintln(os.Stderr, "usage: wraith attest --report r.json --rule rule.yml --out a.json --signing-key <base64|env WRAITH_SIGNING_PRIVATE_KEY>")
		os.Exit(1)
	}

	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading report:", err)
		os.Exit(1)
	}
	var report reportFile
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		fmt.Fprintln(os.Stderr, "parsing report:", err)
		os.Exit(1)
	}

	ruleBytes, err := os.ReadFile(rulePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading rule:", err)
		os.Exit(1)
	}

	privRaw, err := base64.StdEncoding.DecodeString(signingKeyB64)
	if err != nil || len(privRaw) != ed25519.PrivateKeySize {
		fmt.Fprintln(os.Stderr, "invalid signing key")
		os.Exit(1)
	}
	priv := ed25519.PrivateKey(privRaw)

	var validate, attackSim, robustness struct {
		FalsePositiveRate float64  `json:"false_positive_rate"`
		SimulatedChain    []string `json:"simulated_chain"`
		BaselineEventsN   int      `json:"events_indexed"`
		Score             float64  `json:"score"`
	}
	if raw, ok := report.Stages["validate"]; ok {
		_ = json.Unmarshal(raw, &validate)
	}
	if raw, ok := report.Stages["attack_simulation"]; ok {
		_ = json.Unmarshal(raw, &attackSim)
	}
	if raw, ok := report.Stages["baseline"]; ok {
		_ = json.Unmarshal(raw, &attackSim)
	}
	if raw, ok := report.Stages["robustness"]; ok {
		_ = json.Unmarshal(raw, &robustness)
	}

	att := provenance.Attestation{
		RunID:             report.RunID,
		RuleID:            report.RuleID,
		RuleContentSHA256: provenance.HashRuleContent(ruleBytes),
		TestedTechniques:  attackSim.SimulatedChain,
		BaselineEventsN:   attackSim.BaselineEventsN,
		FalsePositiveRate: validate.FalsePositiveRate,
		RobustnessScore:   robustness.Score,
		Passed:            report.Passed,
		Issuer:            "wraith-ci",
	}

	signed, err := provenance.Sign(att, priv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "signing error:", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(signed, "", "  ")
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "writing attestation:", err)
		os.Exit(1)
	}
	fmt.Println("wrote signed attestation to", outPath)
}

func runVerify() {
	attPath := flagValue("--attestation")
	rulePath := flagValue("--rule")
	pubKeyB64 := flagValue("--public-key")
	if pubKeyB64 == "" {
		pubKeyB64 = os.Getenv("WRAITH_SIGNING_PUBLIC_KEY")
	}
	if attPath == "" || pubKeyB64 == "" {
		fmt.Fprintln(os.Stderr, "usage: wraith verify --attestation a.json --rule rule.yml --public-key <base64|env WRAITH_SIGNING_PUBLIC_KEY>")
		os.Exit(1)
	}

	attBytes, err := os.ReadFile(attPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading attestation:", err)
		os.Exit(1)
	}
	var signed provenance.SignedAttestation
	if err := json.Unmarshal(attBytes, &signed); err != nil {
		fmt.Fprintln(os.Stderr, "parsing attestation:", err)
		os.Exit(1)
	}

	pubRaw, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		fmt.Fprintln(os.Stderr, "invalid public key")
		os.Exit(1)
	}
	pub := ed25519.PublicKey(pubRaw)

	if rulePath != "" {
		ruleBytes, err := os.ReadFile(rulePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "reading rule:", err)
			os.Exit(1)
		}
		if err := provenance.VerifyAgainstRule(&signed, pub, ruleBytes); err != nil {
			fmt.Fprintln(os.Stderr, "VERIFICATION FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("OK: attestation is valid and matches the current rule content")
		return
	}

	if err := provenance.Verify(&signed, pub); err != nil {
		fmt.Fprintln(os.Stderr, "VERIFICATION FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("OK: signature is valid")
}

func runAPIKey() {
	if len(os.Args) < 3 || os.Args[2] != "create" {
		fmt.Fprintln(os.Stderr, "usage: wraith apikey create --label <name> --role <viewer|analyst|lead|admin>")
		os.Exit(1)
	}
	label := flagValue("--label")
	role := flagValue("--role")
	if label == "" || role == "" {
		fmt.Fprintln(os.Stderr, "--label and --role are required")
		os.Exit(1)
	}
	dsn := os.Getenv("WRAITH_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "WRAITH_DATABASE_URL must be set to manage API keys (in-memory dev mode uses a fixed dev key)")
		os.Exit(1)
	}

	raw := generateRandomKey()
	hash := auth.HashKey(raw)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pg, err := store.NewPostgresStore(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connecting to postgres:", err)
		os.Exit(1)
	}
	defer pg.Close()

	if err := pg.CreateAPIKey(ctx, hash, label, role); err != nil {
		fmt.Fprintln(os.Stderr, "creating key:", err)
		os.Exit(1)
	}

	fmt.Println("# This is the ONLY time the raw key is shown. Store it securely now.")
	fmt.Println("WRAITH_API_KEY=" + raw)
}

func generateRandomKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintln(os.Stderr, "generating random key:", err)
		os.Exit(1)
	}
	return "wraith_" + base64.RawURLEncoding.EncodeToString(b)
}

func flagValue(name string) string {
	for i, a := range os.Args {
		if a == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}
