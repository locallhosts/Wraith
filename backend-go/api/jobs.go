package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/auth"
	"github.com/locallhosts/Wraith/backend-go/store"
)

func listJobs(s *Server, c *gin.Context) {
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	jobs, err := s.Store.ListPipelineJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func retryJob(s *Server, c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	job, err := s.Store.GetPipelineJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.Status != "failed" && job.Status != "cancelled" {
		c.JSON(http.StatusConflict, gin.H{"error": "only failed or cancelled jobs can be retried"})
		return
	}
	if err := s.Store.RetryPipelineJob(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	identity, _ := auth.GetIdentity(c)
	_ = s.Store.AppendAudit(c.Request.Context(), &store.AuditEntry{
		Actor: identity.Label, ActorRole: identity.Role, Action: "job.retry",
		Resource: strconv.FormatInt(id, 10), Detail: "pipeline job requeued by operator", IPAddress: c.ClientIP(),
	})
	c.JSON(http.StatusOK, gin.H{"queued": true, "job_id": id})
}

func cancelRun(s *Server, c *gin.Context) {
	runID := c.Param("id")
	run, err := s.Store.GetRun(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	if run.Stage == "done" || run.Stage == "failed" {
		c.JSON(http.StatusConflict, gin.H{"error": "terminal runs cannot be cancelled"})
		return
	}
	if err := s.Store.CancelPipelineJob(c.Request.Context(), runID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	identity, _ := auth.GetIdentity(c)
	_ = s.Store.AppendAudit(c.Request.Context(), &store.AuditEntry{
		Actor: identity.Label, ActorRole: identity.Role, Action: "run.cancel",
		Resource: runID, Detail: "pipeline job cancelled by operator", IPAddress: c.ClientIP(),
	})
	_ = s.Store.AppendRunEvent(c.Request.Context(), &store.RunEvent{
		RunID: runID, Stage: run.Stage, Level: "warn", Message: "pipeline job cancelled by operator", CreatedAt: time.Now(),
	})
	c.JSON(http.StatusOK, gin.H{"cancelled": true, "run_id": runID})
}
