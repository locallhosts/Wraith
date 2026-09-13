// Package linter performs fast, offline validation of Sigma detection rules
// before any infrastructure is provisioned. This is the cheap "fail fast"
// stage of the WRAITH pipeline.
package linter

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SigmaRule mirrors the subset of the official Sigma rule spec we validate.
// https://github.com/SigmaHQ/sigma-specification
type SigmaRule struct {
	Title         string                 `yaml:"title"`
	ID            string                 `yaml:"id"`
	Status        string                 `yaml:"status"`
	Description   string                 `yaml:"description"`
	References    []string               `yaml:"references"`
	Author        string                 `yaml:"author"`
	Date          string                 `yaml:"date"`
	Tags          []string               `yaml:"tags"`
	LogSource     map[string]string      `yaml:"logsource"`
	Detection     map[string]interface{} `yaml:"detection"`
	FalsePositive []string               `yaml:"falsepositives"`
	Level         string                 `yaml:"level"`
}

// Issue represents a single lint finding.
type Issue struct {
	Field    string
	Message  string
	Severity string // "error" blocks the pipeline, "warning" does not
}

// Result is the full outcome of linting a single rule file.
type Result struct {
	Path   string
	Rule   *SigmaRule
	Issues []Issue
}

// Passed returns true if there are no "error" severity issues.
func (r Result) Passed() bool {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return false
		}
	}
	return true
}

// PassedStrict returns true only if there are no issues at all — errors
// AND warnings. Used by `wraith lint --strict` / the GitHub Action's
// fail-on-warning input, for teams that want warnings to block merges too.
func (r Result) PassedStrict() bool {
	return len(r.Issues) == 0
}

var validLevels = map[string]bool{
	"informational": true, "low": true, "medium": true, "high": true, "critical": true,
}

var validStatuses = map[string]bool{
	"stable": true, "test": true, "experimental": true, "deprecated": true, "unsupported": true,
}

// LintFile reads a Sigma rule YAML file from disk and validates it.
func LintFile(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return LintBytes(path, data)
}

// LintBytes validates raw YAML bytes against Sigma schema requirements
// and a set of engineering best-practice checks used by this pipeline.
func LintBytes(path string, data []byte) (Result, error) {
	var rule SigmaRule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return Result{
			Path: path,
			Issues: []Issue{{
				Field:    "yaml",
				Message:  fmt.Sprintf("invalid YAML syntax: %v", err),
				Severity: "error",
			}},
		}, nil
	}

	res := Result{Path: path, Rule: &rule}
	add := func(field, msg, sev string) {
		res.Issues = append(res.Issues, Issue{Field: field, Message: msg, Severity: sev})
	}

	// --- Required fields (Sigma spec) ---
	if strings.TrimSpace(rule.Title) == "" {
		add("title", "title is required", "error")
	} else if len(rule.Title) > 256 {
		add("title", "title should be <= 256 characters", "warning")
	}

	if strings.TrimSpace(rule.ID) == "" {
		add("id", "id (UUIDv4) is required for stable rule tracking across CI runs", "error")
	}

	if rule.LogSource == nil || len(rule.LogSource) == 0 {
		add("logsource", "logsource is required (e.g. category, product, service)", "error")
	}

	if rule.Detection == nil || len(rule.Detection) == 0 {
		add("detection", "detection block is required", "error")
	} else if _, ok := rule.Detection["condition"]; !ok {
		add("detection.condition", "detection block must include a 'condition'", "error")
	}

	// --- Best-practice / pipeline-specific checks ---
	if rule.Level == "" {
		add("level", "level is required so alert routing/SOAR severity can be derived", "error")
	} else if !validLevels[strings.ToLower(rule.Level)] {
		add("level", fmt.Sprintf("unknown level %q", rule.Level), "error")
	}

	if rule.Status != "" && !validStatuses[strings.ToLower(rule.Status)] {
		add("status", fmt.Sprintf("unknown status %q", rule.Status), "warning")
	}

	if len(rule.Tags) == 0 {
		add("tags", "no MITRE ATT&CK tags found (expected e.g. attack.t1059) — required for graph-based attack simulation to map this rule to a technique", "error")
	} else {
		hasAttackTag := false
		for _, t := range rule.Tags {
			if strings.HasPrefix(strings.ToLower(t), "attack.t") {
				hasAttackTag = true
				break
			}
		}
		if !hasAttackTag {
			add("tags", "no tag of form attack.tXXXX found — MITRE technique mapping will be skipped for this rule", "warning")
		}
	}

	if len(rule.FalsePositive) == 0 {
		add("falsepositives", "no known false positives documented", "warning")
	}

	if strings.TrimSpace(rule.Description) == "" {
		add("description", "description is empty — required for SOAR playbook generation context", "warning")
	}

	return res, nil
}

// LintDir walks a directory of *.yml / *.yaml rule files and lints each one.
func LintDir(dir string) ([]Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		r, err := LintFile(dir + "/" + name)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
