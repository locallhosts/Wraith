package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPipelineVerdict(t *testing.T) {
	dir := t.TempDir()
	runID := "run-test"
	runDir := filepath.Join(dir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "report.json"), []byte(`{"passed":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	passed, reason := readPipelineVerdict(dir, runID)
	if passed == nil || !*passed {
		t.Fatalf("expected passing verdict, got %#v", passed)
	}
	if reason != "" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestReadPipelineVerdictMalformedReport(t *testing.T) {
	dir := t.TempDir()
	runID := "run-malformed"
	runDir := filepath.Join(dir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "report.json"), []byte(`{"passed":`), 0644); err != nil {
		t.Fatal(err)
	}

	passed, reason := readPipelineVerdict(dir, runID)
	if passed != nil {
		t.Fatalf("expected nil verdict for malformed report, got %#v", *passed)
	}
	if reason == "" {
		t.Fatal("expected malformed report reason")
	}
}

func TestUpdateStageFromLine(t *testing.T) {
	var got string
	update := func(stage string, _ *bool, _ string) { got = stage }

	updateStageFromLine(update, "[run_pipeline] validating rule against attack + baseline indices")
	if got != "validate" {
		t.Fatalf("expected validate stage, got %q", got)
	}

	updateStageFromLine(update, "[run_pipeline] rule passed — drafting SOAR playbook")
	if got != "soar" {
		t.Fatalf("expected soar stage, got %q", got)
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("WRAITH_TEST_BOOL", "true")
	if !envBool("WRAITH_TEST_BOOL", false) {
		t.Fatal("expected true")
	}
	t.Setenv("WRAITH_TEST_BOOL", "off")
	if envBool("WRAITH_TEST_BOOL", true) {
		t.Fatal("expected false")
	}
	t.Setenv("WRAITH_TEST_BOOL", "not-a-bool")
	if !envBool("WRAITH_TEST_BOOL", true) {
		t.Fatal("expected fallback true")
	}
}
