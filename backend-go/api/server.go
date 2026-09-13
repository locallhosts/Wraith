// Package api exposes the HTTP surface the Next.js dashboard, GitHub
// Actions runner, and SOC analysts talk to. Every state-changing route is
// behind API-key auth + role-based access control (backend-go/auth) and
// writes an audit_log entry (backend-go/store); read-only routes require
// at least `viewer`.
package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourname/wraith/backend-go/auth"
	"github.com/yourname/wraith/backend-go/deploy"
	"github.com/yourname/wraith/backend-go/linter"
	"github.com/yourname/wraith/backend-go/metrics"
	"github.com/yourname/wraith/backend-go/notify"
	"github.com/yourname/wraith/backend-go/provenance"
	"github.com/yourname/wraith/backend-go/store"
	"github.com/yourname/wraith/backend-go/webhook"
)

// Server bundles routes and dependencies. Nil optional fields (Slack,
// TrustedSigningKey) degrade gracefully — notifications no-op, deploy
// endpoint returns 503 explaining the missing key.
type Server struct {
	Store             store.Store
	WebhookSecret     string
	RulesDir          string
	OutputDir         string // where engine-python/run_pipeline.py writes report.json / attestation.json per run
	Slack             *notify.SlackNotifier
	TrustedSigningKey ed25519.PublicKey // nil disables the deploy endpoint
	Log               *slog.Logger
	PipelineTrigger   func(runID, rulePath string)
	RateLimit         gin.HandlerFunc // optional, applied globally if set
}

func (s *Server) outputDir() string {
	if s.OutputDir != "" {
		return s.OutputDir
	}
	return "engine-python/output"
}

func audit(s store.Store, c *gin.Context, action, resource, detail string) {
	id, _ := auth.GetIdentity(c)
	role := id.Role
	label := id.Label
	if label == "" {
		label = "anonymous"
		role = "none"
	}
	_ = s.AppendAudit(c.Request.Context(), &store.AuditEntry{
		Actor: label, ActorRole: role, Action: action, Resource: resource,
		Detail: detail, IPAddress: c.ClientIP(),
	})
}

func NewRouter(s *Server) *gin.Engine {
	if s.Log == nil {
		s.Log = slog.Default()
	}
	r := gin.New()
	r.Use(gin.Recovery(), slogMiddleware(s.Log))
	if s.RateLimit != nil {
		r.Use(s.RateLimit)
	}

	// --- Unauthenticated: liveness/readiness/metrics/webhook ---
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := s.Store.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/webhook/github", func(c *gin.Context) { handleWebhook(s, c) })

	// --- Authenticated API surface ---
	api := r.Group("/")
	api.Use(auth.Middleware(s.Store))
	{
		api.GET("/runs", auth.RequireRole("viewer"), func(c *gin.Context) { listRuns(s, c) })
		api.GET("/runs/:id", auth.RequireRole("viewer"), func(c *gin.Context) { getRun(s, c) })
		api.GET("/runs/:id/report", auth.RequireRole("viewer"), func(c *gin.Context) { getRunReport(s, c) })
		api.GET("/runs/:id/attestation", auth.RequireRole("viewer"), func(c *gin.Context) { getRunAttestation(s, c) })
		api.GET("/audit", auth.RequireRole("admin"), func(c *gin.Context) { listAudit(s, c) })
		api.POST("/lint", auth.RequireRole("analyst"), func(c *gin.Context) { runLint(s, c) })
		api.POST("/runs/:id/approve", auth.RequireRole("lead"), func(c *gin.Context) { approveRun(s, c) })
		api.POST("/runs/:id/deploy", auth.RequireRole("lead"), func(c *gin.Context) { deployRun(s, c) })
	}

	return r
}

func slogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		id, _ := auth.GetIdentity(c)
		log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"actor", id.Label,
			"remote_ip", c.ClientIP(),
		)
	}
}

func handleWebhook(s *Server, c *gin.Context) {
	evt, err := webhook.ParsePullRequestEvent(c.Request, s.WebhookSecret)
	if err != nil {
		metrics.WebhookRejectionsTotal.WithLabelValues("bad_signature").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if evt.Action != "opened" && evt.Action != "synchronize" && evt.Action != "reopened" {
		metrics.WebhookRejectionsTotal.WithLabelValues("unsupported_action").Inc()
		c.JSON(http.StatusOK, gin.H{"skipped": true, "action": evt.Action})
		return
	}

	results, err := linter.LintDir(s.RulesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var triggered []string
	for _, res := range results {
		if !res.Passed() {
			continue
		}
		runID := safeSlice(evt.PullRequest.Head.Sha, 8) + "-" + safeSlice(res.Rule.ID, 8)
		status := &store.RunStatus{
			RunID: runID, RulePath: res.Path, RuleID: res.Rule.ID, RuleTitle: res.Rule.Title,
			Repo: evt.Repository.FullName, PRNumber: evt.Number, Stage: "lint",
			StartedAt: time.Now(),
		}
		if err := s.Store.PutRun(c.Request.Context(), status); err != nil {
			s.Log.Error("failed to persist run", "run_id", runID, "error", err)
			continue
		}
		_ = s.Store.AppendAudit(c.Request.Context(), &store.AuditEntry{
			Actor: "github-webhook", ActorRole: "system", Action: "run.triggered",
			Resource: runID, Detail: fmt.Sprintf("PR #%d on %s", evt.Number, evt.Repository.FullName),
			IPAddress: c.ClientIP(),
		})
		if s.PipelineTrigger != nil {
			go s.PipelineTrigger(runID, res.Path)
		}
		triggered = append(triggered, runID)
	}

	c.JSON(http.StatusOK, gin.H{"triggered_runs": triggered, "lint_results": results})
}

func safeSlice(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func listRuns(s *Server, c *gin.Context) {
	runs, err := s.Store.ListRuns(c.Request.Context(), 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

func getRun(s *Server, c *gin.Context) {
	run, err := s.Store.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	c.JSON(http.StatusOK, run)
}

func listAudit(s *Server, c *gin.Context) {
	entries, err := s.Store.ListAudit(c.Request.Context(), 500)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

// getRunReport serves the full JSON report written by
// engine-python/run_pipeline.py for a given run — every stage's raw
// output (translate, baseline, attack_simulation, validate, robustness,
// soar_playbook) — so the dashboard can render it without the frontend
// needing filesystem access itself.
func getRunReport(s *Server, c *gin.Context) {
	runID := c.Param("id")
	reportPath := fmt.Sprintf("%s/%s/report.json", s.outputDir(), runID)
	data, err := os.ReadFile(reportPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no report found for this run yet — the pipeline may still be running"})
		return
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report.json is malformed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, parsed)
}

// getRunAttestation serves the signed provenance attestation for a run, if
// one was produced (only passing runs get attested — see `wraith attest`
// in the CI workflow). Note: this returns the attestation as-is; it does
// NOT re-verify the signature here (that only happens at deploy time,
// against the rule's current content) — this endpoint is for display.
func getRunAttestation(s *Server, c *gin.Context) {
	runID := c.Param("id")
	attPath := fmt.Sprintf("%s/%s/attestation.json", s.outputDir(), runID)
	data, err := os.ReadFile(attPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no attestation found for this run (rule may not have passed, or CI signing step hasn't run yet)"})
		return
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "attestation.json is malformed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, parsed)
}

func runLint(s *Server, c *gin.Context) {
	results, err := linter.LintDir(s.RulesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	audit(s.Store, c, "rules.lint", s.RulesDir, fmt.Sprintf("%d rules linted", len(results)))
	c.JSON(http.StatusOK, results)
}

// approveRun implements the human-in-the-loop gate: a `lead`+ operator
// signs off that a passing run is safe to promote, which is a
// precondition for /deploy (the other precondition being a valid
// provenance signature — see backend-go/deploy).
func approveRun(s *Server, c *gin.Context) {
	runID := c.Param("id")
	run, err := s.Store.GetRun(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	if run.Passed == nil || !*run.Passed {
		c.JSON(http.StatusConflict, gin.H{"error": "only a passing run can be approved"})
		return
	}

	id, _ := auth.GetIdentity(c)
	now := time.Now()
	run.ApprovedBy = id.Label
	run.ApprovedAt = &now
	if err := s.Store.PutRun(c.Request.Context(), run); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.ApprovalsTotal.WithLabelValues(id.Role).Inc()
	audit(s.Store, c, "run.approve", runID, "approved by "+id.Label)
	c.JSON(http.StatusOK, run)
}

// deployRun is the final gate: requires the run to be approved AND its
// provenance attestation to verify against the rule's CURRENT on-disk
// content, closing the "edited after CI passed" gap. Actual index write
// happens in backend-go/deploy; this handler is glue + audit + metrics.
func deployRun(s *Server, c *gin.Context) {
	runID := c.Param("id")
	run, err := s.Store.GetRun(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	if run.ApprovedBy == "" {
		metrics.DeploysTotal.WithLabelValues("not_approved").Inc()
		c.JSON(http.StatusConflict, gin.H{"error": deploy.ErrNotApproved.Error()})
		return
	}
	if s.TrustedSigningKey == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WRAITH_SIGNING_PUBLIC_KEY is not configured — production deploys are disabled",
		})
		return
	}

	attestationPath := fmt.Sprintf("%s/%s/attestation.json", s.outputDir(), runID)
	attBytes, err := os.ReadFile(attestationPath)
	if err != nil {
		metrics.DeploysTotal.WithLabelValues("error").Inc()
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no attestation found for this run: " + err.Error()})
		return
	}
	var signed provenance.SignedAttestation
	if err := json.Unmarshal(attBytes, &signed); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "malformed attestation: " + err.Error()})
		return
	}

	currentRule, err := os.ReadFile(run.RulePath)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "could not re-read current rule file: " + err.Error()})
		return
	}

	gate := deploy.Gate{TrustedPublicKey: s.TrustedSigningKey}
	if err := gate.Check(run.ApprovedBy, &signed, currentRule); err != nil {
		metrics.DeploysTotal.WithLabelValues("signature_invalid").Inc()
		audit(s.Store, c, "run.deploy.rejected", runID, err.Error())
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	run.DeployedAt = &now
	run.Stage = "done"
	_ = s.Store.PutRun(c.Request.Context(), run)

	metrics.DeploysTotal.WithLabelValues("success").Inc()
	audit(s.Store, c, "run.deploy", runID, "deployed by approval from "+run.ApprovedBy)
	c.JSON(http.StatusOK, gin.H{"deployed": true, "run": run})
}

// DefaultPythonPipelineTrigger shells out to the real Python engine, then
// sends Slack notifications and records metrics on completion.
func DefaultPythonPipelineTrigger(s *Server, esAddr, neo4jAddr string) func(runID, rulePath string) {
	return func(runID, rulePath string) {
		ctx := context.Background()
		update := func(stage string, passed *bool, reason string) {
			run, err := s.Store.GetRun(ctx, runID)
			if err != nil {
				return
			}
			run.Stage = stage
			run.Passed = passed
			run.Reason = reason
			_ = s.Store.PutRun(ctx, run)
		}

		start := time.Now()
		update("provision", nil, "")

		runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(runCtx, "python3", "engine-python/run_pipeline.py",
			"--rule", rulePath, "--run-id", runID,
			"--es-addr", esAddr, "--neo4j-addr", neo4jAddr, "--open-pr",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		update("simulate", nil, "")
		runErr := cmd.Run()
		passed := runErr == nil

		verdict := "passed"
		if !passed {
			verdict = "failed"
		}
		metrics.RunsTotal.WithLabelValues(verdict).Inc()
		metrics.RunDuration.WithLabelValues("total").Observe(time.Since(start).Seconds())

		reason := "pipeline completed"
		if runErr != nil {
			reason = runErr.Error()
		}
		update("done", &passed, reason)

		run, _ := s.Store.GetRun(ctx, runID)
		if run != nil && s.Slack != nil {
			_ = s.Slack.RunCompleted(ctx, runID, run.RuleTitle, run.Repo, run.PRNumber, passed, reason)
			if passed {
				_ = s.Slack.AwaitingApproval(ctx, runID, run.RuleTitle, 1.0)
			}
		}
	}
}
