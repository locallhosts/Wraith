package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/linter"
)

// listRules returns the real Sigma rules available to the Wraith pipeline.
// Rule content is intentionally not returned here; callers use GET /rules/:name
// for an individual rule so the list remains small and predictable.
func listRules(s *Server, c *gin.Context) {
	results, err := linter.LintDir(s.RulesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not inspect rules: " + err.Error(),
		})
		return
	}

	type ruleSummary struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Title       string `json:"title"`
		ID          string `json:"id"`
		Status      string `json:"status"`
		Level       string `json:"level"`
		Passed      bool   `json:"passed"`
		IssueCount  int    `json:"issue_count"`
		ErrorCount  int    `json:"error_count"`
		WarningCount int   `json:"warning_count"`
	}

	out := make([]ruleSummary, 0, len(results))
	for _, result := range results {
		name := filepath.Base(result.Path)
		summary := ruleSummary{
			Name:       name,
			Path:       result.Path,
			Passed:     result.Passed(),
			IssueCount: len(result.Issues),
		}
		if result.Rule != nil {
			summary.Title = result.Rule.Title
			summary.ID = result.Rule.ID
			summary.Status = result.Rule.Status
			summary.Level = result.Rule.Level
		}
		for _, issue := range result.Issues {
			switch strings.ToLower(issue.Severity) {
			case "error":
				summary.ErrorCount++
			case "warning":
				summary.WarningCount++
			}
		}
		out = append(out, summary)
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(out),
		"rules": out,
	})
}

// getRule returns one real rule's metadata, lint findings, and source YAML.
// The route accepts a filename only and rejects path traversal explicitly.
func getRule(s *Server, c *gin.Context) {
	name := c.Param("name")
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule name"})
		return
	}
	if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule name must end in .yml or .yaml"})
		return
	}

	path := filepath.Join(s.RulesDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read rule"})
		return
	}

	result, err := linter.LintBytes(path, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not lint rule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    name,
		"path":    path,
		"passed":  result.Passed(),
		"rule":    result.Rule,
		"issues":  result.Issues,
		"content": string(data),
	})
}
