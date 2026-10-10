// Package api exposes the HTTP surface the Next.js dashboard, GitHub
// Actions runner, and SOC analysts talk to. Every state-changing route is
// behind API-key auth + role-based access control (backend-go/auth) and
// writes an audit_log entry (backend-go/store); read-only routes require
// at least `viewer`.
package api

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/auth"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/locallhosts/Wraith/backend-go/deploy"
	"github.com/locallhosts/Wraith/backend-go/linter"
	"github.com/locallhosts/Wraith/backend-go/metrics"
	"github.com/locallhosts/Wraith/backend-go/notify"
	"github.com/locallhosts/Wraith/backend-go/provenance"
	"github.com/locallhosts/Wraith/backend-go/store"
	"github.com/locallhosts/Wraith/backend-go/webhook"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	ESAddr            string
	Log               *slog.Logger
	PipelineTrigger   func(runID, rulePath string)
	PipelineJobsEnabled bool
	RateLimit         gin.HandlerFunc // optional, applied globally if set
	PublicPlayground  bool
	PublicOrigins     map[string]bool
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

// localhostCORSMiddleware allows the local Next.js dashboard to call the
// Go API during local development.
//
// The browser frontend runs on http://localhost:3000 while the API runs on
// http://localhost:8080, so the browser requires explicit CORS headers.
// Authorization requests trigger an OPTIONS preflight request, which must
// be handled before the authenticated API middleware.
func corsMiddleware(origins map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && origins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			if origins[origin] {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}

			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

func NewRouter(s *Server) *gin.Engine {
	if s.Log == nil {
		s.Log = slog.Default()
	}

	r := gin.New()

	r.Use(gin.Recovery(), slogMiddleware(s.Log), securityHeadersMiddleware(), requestTimeoutMiddleware(30*time.Second))
	r.Use(corsMiddleware(s.PublicOrigins))

	if s.RateLimit != nil {
		r.Use(s.RateLimit)
	}

	// --- Unauthenticated: liveness/readiness/metrics/webhook ---

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := s.Store.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ready": false,
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/webhook/github", func(c *gin.Context) {
		handleWebhook(s, c)
	})

	// --- Authenticated API surface ---

	// Public Playground: deliberately isolated from the authenticated control plane.
	if s.PublicPlayground {
		r.POST("/playground/validate", func(c *gin.Context) { playgroundValidate(s, c) })
	}

	api := r.Group("/")
	api.Use(auth.Middleware(s.Store))

	{
		api.GET("/runs", auth.RequireRole("viewer"), func(c *gin.Context) {
			listRuns(s, c)
		})

		api.GET("/runs/:id", auth.RequireRole("viewer"), func(c *gin.Context) {
			getRun(s, c)
		})

		api.GET("/runs/:id/report", auth.RequireRole("viewer"), func(c *gin.Context) {
			getRunReport(s, c)
		})

		api.GET("/runs/:id/stages", auth.RequireRole("viewer"), func(c *gin.Context) {
			listRunStages(s, c)
		})

		api.GET("/runs/:id/events", auth.RequireRole("viewer"), func(c *gin.Context) { listRunEvents(s, c) })
		api.GET("/jobs", auth.RequireRole("viewer"), func(c *gin.Context) { listJobs(s, c) })
		api.POST("/jobs/:id/retry", auth.RequireRole("analyst"), func(c *gin.Context) { retryJob(s, c) })
		api.GET("/session", auth.RequireRole("viewer"), func(c *gin.Context) { getSession(c) })
		api.GET("/rules", auth.RequireRole("viewer"), func(c *gin.Context) { listRules(s, c) })
		api.GET("/rules/:name", auth.RequireRole("viewer"), func(c *gin.Context) { getRule(s, c) })

		api.GET("/runs/:id/attestation", auth.RequireRole("viewer"), func(c *gin.Context) {
			getRunAttestation(s, c)
		})

		api.GET("/audit", auth.RequireRole("admin"), func(c *gin.Context) {
			listAudit(s, c)
		})

		api.GET("/api-keys", auth.RequireRole("admin"), func(c *gin.Context) {
			listAPIKeys(s, c)
		})

		api.POST("/api-keys", auth.RequireRole("admin"), func(c *gin.Context) {
			createAPIKey(s, c)
		})

		api.POST("/api-keys/:id/revoke", auth.RequireRole("admin"), func(c *gin.Context) {
			revokeAPIKey(s, c)
		})

		api.POST("/lint", auth.RequireRole("analyst"), func(c *gin.Context) {
			runLint(s, c)
		})


		api.POST("/runs/:id/approve", auth.RequireRole("lead"), func(c *gin.Context) {
			approveRun(s, c)
		})

		api.POST("/runs/:id/deploy/dry-run", auth.RequireRole("lead"), func(c *gin.Context) { dryRunDeploy(s, c) })
		api.POST("/runs/:id/deploy", auth.RequireRole("lead"), func(c *gin.Context) { deployRun(s, c) })
		api.GET("/deployments", auth.RequireRole("viewer"), func(c *gin.Context) { listDeployments(s, c) })
		api.GET("/deployments/:id/verify", auth.RequireRole("viewer"), func(c *gin.Context) { verifyDeployment(s, c) })
		api.POST("/deployments/:id/rollback", auth.RequireRole("lead"), func(c *gin.Context) { rollbackDeployment(s, c) })
	}

	return r
}

func requestTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

func slogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		id, _ := auth.GetIdentity(c)

		log.Info(
			"request",
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
		if errors.Is(err, webhook.ErrPayloadTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "webhook payload exceeds size limit"})
			return
		}
		metrics.WebhookRejectionsTotal.WithLabelValues("bad_signature").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook signature verification failed"})
		return
	}

	if evt.Action != "opened" &&
		evt.Action != "synchronize" &&
		evt.Action != "reopened" {

		metrics.WebhookRejectionsTotal.WithLabelValues("unsupported_action").Inc()

		c.JSON(http.StatusOK, gin.H{
			"skipped": true,
			"action":  evt.Action,
		})
		return
	}

	results, err := linter.LintDir(s.RulesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var triggered []string

	for _, res := range results {
		if !res.Passed() {
			continue
		}

		runID := safeSlice(evt.PullRequest.Head.Sha, 8) +
			"-" +
			safeSlice(res.Rule.ID, 8)

		status := &store.RunStatus{
			RunID:     runID,
			RulePath:  res.Path,
			RuleID:    res.Rule.ID,
			RuleTitle: res.Rule.Title,
			Repo:      evt.Repository.FullName,
			PRNumber:  evt.Number,
			Stage:     "lint",
			StartedAt: time.Now(),
		}

		if err := s.Store.PutRun(c.Request.Context(), status); err != nil {
			s.Log.Error(
				"failed to persist run",
				"run_id",
				runID,
				"error",
				err,
			)
			continue
		}

		_ = s.Store.AppendAudit(
			c.Request.Context(),
			&store.AuditEntry{
				Actor:     "github-webhook",
				ActorRole: "system",
				Action:    "run.triggered",
				Resource:  runID,
				Detail:    fmt.Sprintf("PR #%d on %s", evt.Number, evt.Repository.FullName),
				IPAddress: c.ClientIP(),
			},
		)

		if s.PipelineJobsEnabled {
			if err := s.Store.EnqueuePipelineJob(c.Request.Context(), &store.PipelineJob{RunID: runID, RulePath: res.Path, Status: "queued", MaxAttempts: 3}); err != nil {
				s.Log.Error("failed to enqueue pipeline job", "run_id", runID, "error", err)
				continue
			}
		} else if s.PipelineTrigger != nil {
			go s.PipelineTrigger(runID, res.Path)
		}

		triggered = append(triggered, runID)
	}

	c.JSON(http.StatusOK, gin.H{
		"triggered_runs": triggered,
		"lint_results":   results,
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func getRun(s *Server, c *gin.Context) {
	run, err := s.Store.GetRun(
		c.Request.Context(),
		c.Param("id"),
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "run not found",
		})
		return
	}

	c.JSON(http.StatusOK, run)
}

func listAudit(s *Server, c *gin.Context) {
	entries, err := s.Store.ListAudit(
		c.Request.Context(),
		500,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
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

	reportPath, pathErr := runArtifactPath(s.outputDir(), runID, "report.json")
	if pathErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no report found for this run yet — the pipeline may still be running",
		})
		return
	}

	var parsed map[string]any

	if err := json.Unmarshal(data, &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "report.json is malformed: " + err.Error(),
		})
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

	attPath, pathErr := runArtifactPath(s.outputDir(), runID, "attestation.json")
	if pathErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}

	data, err := os.ReadFile(attPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no attestation found for this run (rule may not have passed, or CI signing step hasn't run yet)",
		})
		return
	}

	var parsed map[string]any

	if err := json.Unmarshal(data, &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "attestation.json is malformed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, parsed)
}

func runLint(s *Server, c *gin.Context) {
	results, err := linter.LintDir(s.RulesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	audit(
		s.Store,
		c,
		"rules.lint",
		s.RulesDir,
		fmt.Sprintf("%d rules linted", len(results)),
	)

	c.JSON(http.StatusOK, results)
}

// approveRun implements the human-in-the-loop gate: a `lead`+ operator
// signs off that a passing run is safe to promote, which is a
// precondition for /deploy (the other precondition being a valid
// provenance signature — see backend-go/deploy).
func approveRun(s *Server, c *gin.Context) {
	runID := c.Param("id")

	run, err := s.Store.GetRun(
		c.Request.Context(),
		runID,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "run not found",
		})
		return
	}

	if run.Passed == nil || !*run.Passed {
		c.JSON(http.StatusConflict, gin.H{
			"error": "only a passing run can be approved",
		})
		return
	}

	id, _ := auth.GetIdentity(c)

	now := time.Now()

	run.ApprovedBy = id.Label
	run.ApprovedAt = &now

	if err := s.Store.PutRun(c.Request.Context(), run); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	metrics.ApprovalsTotal.WithLabelValues(id.Role).Inc()

	audit(
		s.Store,
		c,
		"run.approve",
		runID,
		"approved by "+id.Label,
	)

	c.JSON(http.StatusOK, run)
}

// deployRun is the final gate: requires the run to be approved AND its
// provenance attestation to verify against the rule's CURRENT on-disk
// content, closing the "edited after CI passed" gap. Actual index write
// happens in backend-go/deploy; this handler is glue + audit + metrics.
func prepareDeployment(s *Server, c *gin.Context) (*store.RunStatus, *provenance.SignedAttestation, []byte, any, *elasticsearch.Client, deploy.Gate, error) {
	run, err := s.Store.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("run not found") }
	if run.Passed == nil || !*run.Passed { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("only a passing run can be deployed") }
	if run.ApprovedBy == "" { return nil,nil,nil,nil,nil,deploy.Gate{},deploy.ErrNotApproved }
	if s.TrustedSigningKey == nil { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("production signing key is not configured") }
	if s.ESAddr == "" { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("production deployment target is not configured") }
	attBytes, err := os.ReadFile(fmt.Sprintf("%s/%s/attestation.json", s.outputDir(), run.RunID)); if err != nil { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("no attestation found for this run: %w",err) }
	var signed provenance.SignedAttestation; if err=json.Unmarshal(attBytes,&signed); err!=nil { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("malformed attestation: %w",err) }
	if signed.Attestation.RunID != run.RunID || signed.Attestation.RuleID != run.RuleID { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("attestation does not belong to this run/rule") }
	rule, err := os.ReadFile(run.RulePath); if err != nil { return nil,nil,nil,nil,nil,deploy.Gate{},fmt.Errorf("could not read rule: %w",err) }
	gate:=deploy.Gate{TrustedPublicKey:s.TrustedSigningKey}; if err=gate.Check(run.ApprovedBy,&signed,rule); err!=nil{return nil,nil,nil,nil,nil,gate,err}
	qb,err:=os.ReadFile(fmt.Sprintf("%s/%s/query_dsl.json",s.outputDir(),run.RunID)); if err!=nil{return nil,nil,nil,nil,nil,gate,fmt.Errorf("no translated query found: %w",err)}
	var q any; if err=json.Unmarshal(qb,&q);err!=nil{return nil,nil,nil,nil,nil,gate,fmt.Errorf("translated query is malformed: %w",err)}
	es,err:=elasticsearch.NewClient(elasticsearch.Config{Addresses:[]string{s.ESAddr}});if err!=nil{return nil,nil,nil,nil,nil,gate,err}
	return run,&signed,rule,q,es,gate,nil
}

func deployRun(s *Server, c *gin.Context) {
	run,signed,rule,q,es,gate,err:=prepareDeployment(s,c); if err!=nil { c.JSON(http.StatusConflict,gin.H{"error":err.Error()}); return }
	deployed,err:=deploy.Deploy(c.Request.Context(),es,gate,run.ApprovedBy,signed,q,rule)
	if err!=nil { metrics.DeploysTotal.WithLabelValues("error").Inc(); audit(s.Store,c,"run.deploy.rejected",run.RunID,err.Error()); c.JSON(http.StatusUnprocessableEntity,gin.H{"error":err.Error()}); return }
	if _,err=deploy.Verify(c.Request.Context(),es,run.RuleID,run.RunID,deployed.ContentSHA256);err!=nil { c.JSON(http.StatusBadGateway,gin.H{"error":"deployment verification failed: "+err.Error()}); return }
	now:=time.Now().UTC(); run.DeployedAt=&now; run.Stage="done"; if err=s.Store.PutRun(c.Request.Context(),run);err!=nil { c.JSON(http.StatusInternalServerError,gin.H{"error":"persisting deployment state: "+err.Error()});return }
	metrics.DeploysTotal.WithLabelValues("success").Inc(); audit(s.Store,c,"run.deploy",run.RunID,"deployed by approval from "+run.ApprovedBy)
	c.JSON(http.StatusOK,gin.H{"deployed":true,"run":run,"rule":deployed})
}

func dryRunDeploy(s *Server,c *gin.Context) {
	run,signed,rule,q,es,gate,err:=prepareDeployment(s,c);if err!=nil{c.JSON(http.StatusConflict,gin.H{"error":err.Error(),"mutated":false});return}
	result,err:=deploy.DryRun(c.Request.Context(),es,gate,run.ApprovedBy,signed,q,rule);if err!=nil{audit(s.Store,c,"run.deploy.dry_run.failed",run.RunID,err.Error());c.JSON(http.StatusUnprocessableEntity,gin.H{"error":err.Error(),"mutated":false});return}
	audit(s.Store,c,"run.deploy.dry_run",run.RunID,"dry-run completed without production mutation");c.JSON(http.StatusOK,result)
}

func listDeployments(s *Server,c *gin.Context){if s.ESAddr==""{c.JSON(http.StatusServiceUnavailable,gin.H{"error":"deployment target not configured"});return};es,err:=elasticsearch.NewClient(elasticsearch.Config{Addresses:[]string{s.ESAddr}});if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()});return};items,err:=deploy.ListHistory(c.Request.Context(),es,100);if err!=nil{c.JSON(http.StatusBadGateway,gin.H{"error":err.Error()});return};c.JSON(http.StatusOK,items)}

func verifyDeployment(s *Server,c *gin.Context){if s.ESAddr==""{c.JSON(http.StatusServiceUnavailable,gin.H{"error":"deployment target not configured"});return};es,err:=elasticsearch.NewClient(elasticsearch.Config{Addresses:[]string{s.ESAddr}});if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()});return};record,err:=deploy.GetHistory(c.Request.Context(),es,c.Param("id"));if err!=nil{c.JSON(http.StatusNotFound,gin.H{"error":err.Error()});return};var expected deploy.DeployedRule;if err=json.Unmarshal(record.CurrentSource,&expected);err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"deployment record is malformed"});return};rule,err:=deploy.Verify(c.Request.Context(),es,record.RuleID,record.RunID,expected.ContentSHA256);if err!=nil{c.JSON(http.StatusConflict,gin.H{"verified":false,"error":err.Error(),"deployment":record});return};c.JSON(http.StatusOK,gin.H{"verified":true,"deployment":record,"rule":rule})}

func rollbackDeployment(s *Server,c *gin.Context){if s.ESAddr==""{c.JSON(http.StatusServiceUnavailable,gin.H{"error":"deployment target not configured"});return};es,err:=elasticsearch.NewClient(elasticsearch.Config{Addresses:[]string{s.ESAddr}});if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()});return};record,err:=deploy.Rollback(c.Request.Context(),es,c.Param("id"));if err!=nil{audit(s.Store,c,"run.rollback.rejected",c.Param("id"),err.Error());c.JSON(http.StatusConflict,gin.H{"error":err.Error()});return};audit(s.Store,c,"run.rollback",record.RuleID,"rolled back deployment "+record.DeploymentID);c.JSON(http.StatusOK,gin.H{"rolled_back":true,"deployment":record})}

// DefaultPythonPipelineTrigger shells out to the real Python engine, then
// sends Slack notifications and records metrics on completion.
func DefaultPythonPipelineTrigger(
	s *Server,
	esAddr,
	neo4jAddr string,
) func(runID, rulePath string) {
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

		runCtx, cancel := context.WithTimeout(
			ctx,
			10*time.Minute,
		)
		defer cancel()

		cmd := exec.CommandContext(
			runCtx,
			"python3",
			"engine-python/run_pipeline.py",
			"--rule",
			rulePath,
			"--run-id",
			runID,
			"--es-addr",
			esAddr,
			"--neo4j-addr",
			neo4jAddr,
			"--open-pr",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		update("simulate", nil, "")

		reportPath := fmt.Sprintf("%s/%s/report.json", s.outputDir(), runID)
		stageIDs := map[string]int64{}
		syncStages := func() {
			reportBytes, readErr := os.ReadFile(reportPath)
			if readErr != nil { return }
			var report struct { Stages map[string]map[string]any `json:"stages"` }
			if json.Unmarshal(reportBytes, &report) != nil { return }
			for name, raw := range report.Stages {
				status := "running"
				if v, ok := raw["status"].(string); ok {
					switch v { case "passed", "ok": status = "passed"; case "failed", "error": status = "failed"; case "skipped": status = "skipped" }
				}
				started := time.Now()
				if v, ok := raw["started_at"].(float64); ok { started = time.Unix(0, int64(v*float64(time.Second))) }
				var ended *time.Time
				if v, ok := raw["ended_at"].(float64); ok { t := time.Unix(0, int64(v*float64(time.Second))); ended = &t }
				reason, _ := raw["reason"].(string)
				stage := &store.RunStage{RunID: runID, Name: name, Status: status, Reason: reason, StartedAt: started, EndedAt: ended}
				if id, ok := stageIDs[name]; ok { stage.ID = id; _ = s.Store.UpdateRunStage(ctx, stage) } else if s.Store.AppendRunStage(ctx, stage) == nil { stageIDs[name] = stage.ID }
			}
		}
		stopSync := make(chan struct{})
		var syncWG sync.WaitGroup
		syncWG.Add(1)
		go func() {
			defer syncWG.Done()
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select { case <-ticker.C: syncStages(); case <-stopSync: syncStages(); return }
			}
		}()
		runErr := cmd.Run()
		close(stopSync)
		syncWG.Wait()

		// If the engine exits before it can close a stage itself (panic,
		// dependency failure, timeout, or process error), do not leave
		// persisted stages permanently stuck in "running". The backend is
		// the authoritative observer of process failure and records the
		// actual command error as the stage reason.
		if runErr != nil {
			if stages, stageErr := s.Store.ListRunStages(ctx, runID); stageErr == nil {
				ended := time.Now().UTC()
				for _, stage := range stages {
					if stage.Status != "running" {
						continue
					}
					stage.Status = "failed"
					stage.Reason = runErr.Error()
					stage.EndedAt = &ended
					_ = s.Store.UpdateRunStage(ctx, stage)
				}
			}
		}

		passed := runErr == nil

		verdict := "passed"
		if !passed {
			verdict = "failed"
		}

		metrics.RunsTotal.WithLabelValues(verdict).Inc()
		metrics.RunDuration.
			WithLabelValues("total").
			Observe(time.Since(start).Seconds())

		reason := "pipeline completed"

		if runErr != nil {
			reason = runErr.Error()
		}

		update("done", &passed, reason)

		run, _ := s.Store.GetRun(ctx, runID)

		if run != nil && s.Slack != nil {
			_ = s.Slack.RunCompleted(
				ctx,
				runID,
				run.RuleTitle,
				run.Repo,
				run.PRNumber,
				passed,
				reason,
			)

			if passed {
				_ = s.Slack.AwaitingApproval(
					ctx,
					runID,
					run.RuleTitle,
					1.0,
				)
			}
		}
	}
}


type apiKeyCreateRequest struct {
	Label string `json:"label"`
	Role  string `json:"role"`
}

func listAPIKeys(s *Server, c *gin.Context) {
	keys, err := s.Store.ListAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
}

func newAPIKeyID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil { return base64.RawURLEncoding.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano))) }
	return base64.RawURLEncoding.EncodeToString(buf)
}

func createAPIKey(s *Server, c *gin.Context) {
	var req apiKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Label == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label and role are required"})
		return
	}
	if req.Role != "viewer" && req.Role != "analyst" && req.Role != "lead" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generating API key"})
		return
	}
	raw := "wraith_" + base64.RawURLEncoding.EncodeToString(buf)
	hash := auth.HashKey(raw)
	keyID := newAPIKeyID()

	if err := s.Store.CreateAPIKey(c.Request.Context(), keyID, hash, req.Label, req.Role); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "could not create API key: " + err.Error()})
		return
	}
	audit(s.Store, c, "apikey.create", req.Label, "created role="+req.Role)
	c.JSON(http.StatusCreated, gin.H{
		"label": req.Label,
		"id": keyID,
		"role": req.Role,
		"api_key": raw,
		"warning": "The raw API key is returned once. Store it securely; Wraith never stores or returns it again.",
	})
}

func revokeAPIKey(s *Server, c *gin.Context) {
	id := c.Param("id")
	if len(id) < 16 || len(id) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key identifier"})
		return
	}
	if err := s.Store.RevokeAPIKey(c.Request.Context(), id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	audit(s.Store, c, "apikey.revoke", id, "API key revoked")
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}


func listRunStages(s *Server, c *gin.Context) {
	stages, err := s.Store.ListRunStages(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stages)
}


type playgroundRequest struct {
	Rule string `json:"rule"`
}

func playgroundValidate(s *Server, c *gin.Context) {
	const maxRuleBytes = 256 * 1024
	if c.GetHeader("Content-Type") != "application/json" && !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json;") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRuleBytes+4096)
	var req playgroundRequest
	if err := c.ShouldBindJSON(&req); err != nil || len([]byte(req.Rule)) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule is required"})
		return
	}
	if len([]byte(req.Rule)) > 256*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "rule exceeds 256 KiB limit"})
		return
	}

	linted, err := linter.LintBytes("playground.yml", []byte(req.Rule))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	if !linted.Passed() {
		audit(s.Store, c, "playground.validate", "playground", "lint rejected rule")
		c.JSON(http.StatusUnprocessableEntity, gin.H{"lint": linted, "translated": false})
		return
	}

	tmpDir, err := os.MkdirTemp("", "wraith-playground-")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creating isolated playground workspace"})
		return
	}
	defer os.RemoveAll(tmpDir)

	rulePath := tmpDir + "/rule.yml"
	if err := os.WriteFile(rulePath, []byte(req.Rule), 0600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "writing isolated playground rule"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "engine-python/sigma_to_es.py", rulePath)
	cmd.Dir = "."
	// Do not expose API keys, database credentials, signing material, or other
	// server environment variables to the untrusted translation subprocess.
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "PYTHONUNBUFFERED=1"}
	output, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"lint": linted,
			"translated": false,
			"error": "Sigma translation failed: " + err.Error(),
		})
		return
	}

	var dsl any
	if err := json.Unmarshal(output, &dsl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "translator returned malformed JSON"})
		return
	}

	audit(s.Store, c, "playground.validate", "playground", "offline lint + Sigma-to-ES translation; no production writes")
	c.JSON(http.StatusOK, gin.H{
		"lint": linted,
		"translated": true,
		"query_dsl": dsl,
		"execution": "offline",
	})
}
