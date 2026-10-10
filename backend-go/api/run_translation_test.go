package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetRunTranslation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	outputDir := t.TempDir()
	runDir := filepath.Join(outputDir, "run-123")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	wantQuery := map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	encoded, err := json.Marshal(wantQuery)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "query_dsl.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("serves query artifact", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Params = gin.Params{{Key: "id", Value: "run-123"}}
		ctx.Request = httptest.NewRequest(http.MethodGet, "/runs/run-123/translation", nil)

		getRunTranslation(&Server{OutputDir: outputDir}, ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		var got struct {
			RunID   string         `json:"run_id"`
			QueryDSL map[string]any `json:"query_dsl"`
			Path    string         `json:"path"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.RunID != "run-123" || got.Path != "query_dsl.json" {
			t.Fatalf("unexpected response metadata: %+v", got)
		}
		if got.QueryDSL["query"] == nil {
			t.Fatalf("query_dsl missing query: %+v", got.QueryDSL)
		}
	})

	t.Run("rejects traversal run ID", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Params = gin.Params{{Key: "id", Value: "../secret"}}
		ctx.Request = httptest.NewRequest(http.MethodGet, "/runs/../secret/translation", nil)

		getRunTranslation(&Server{OutputDir: outputDir}, ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("reports missing artifact", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Params = gin.Params{{Key: "id", Value: "missing-run"}}
		ctx.Request = httptest.NewRequest(http.MethodGet, "/runs/missing-run/translation", nil)

		getRunTranslation(&Server{OutputDir: outputDir}, ctx)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})
}
