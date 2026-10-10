package api

import (
	"path/filepath"
	"testing"
)

func TestRunArtifactPathConstrainsRunID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "output")
	tests := []struct {
		name    string
		runID   string
		artifact string
		wantErr bool
	}{
		{name: "normal run id", runID: "a1b2c3d4-rule_123", artifact: "report.json"},
		{name: "hyphenated run id", runID: "run-123", artifact: "attestation.json"},
		{name: "parent traversal", runID: "../secret", artifact: "report.json", wantErr: true},
		{name: "absolute path", runID: "/tmp/secret", artifact: "report.json", wantErr: true},
		{name: "path separator", runID: "run/../../secret", artifact: "attestation.json", wantErr: true},
		{name: "empty run id", runID: "", artifact: "report.json", wantErr: true},
		{name: "dot path", runID: ".", artifact: "report.json", wantErr: true},
		{name: "overlong run id", runID: string(make([]byte, 129)), artifact: "report.json", wantErr: true},
		{name: "unexpected artifact", runID: "run-123", artifact: "../../etc/passwd", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runArtifactPath(root, tt.runID, tt.artifact)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected rejection, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := filepath.Join(root, tt.runID, tt.artifact)
			if got != want {
				t.Fatalf("path mismatch: got %q want %q", got, want)
			}
		})
	}
}
