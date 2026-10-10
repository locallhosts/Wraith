package api

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var safeRunIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// runArtifactPath resolves an artifact below the configured output directory.
// Run IDs are intentionally constrained to a single path component so request
// input cannot escape the per-run directory with traversal sequences.
func runArtifactPath(outputDir, runID, artifact string) (string, error) {
	if !safeRunIDPattern.MatchString(runID) || runID == "." || runID == ".." {
		return "", fmt.Errorf("invalid run id")
	}
	if artifact != "report.json" && artifact != "attestation.json" {
		return "", fmt.Errorf("invalid run artifact")
	}
	return filepath.Join(outputDir, runID, artifact), nil
}
