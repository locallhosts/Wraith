package linter

import "testing"

func TestLintBytes_ValidRule(t *testing.T) {
	good := []byte(`
title: Suspicious PowerShell EncodedCommand
id: 8f1a2b3c-1111-4a4a-9a9a-abcdefabcdef
status: test
description: Detects PowerShell invoked with -EncodedCommand, common in stage-2 payload delivery.
author: wraith
date: 2024/01/01
level: high
tags:
  - attack.execution
  - attack.t1059.001
logsource:
  category: process_creation
  product: windows
detection:
  selection:
    Image|endswith: '\powershell.exe'
    CommandLine|contains: '-EncodedCommand'
  condition: selection
falsepositives:
  - Legitimate admin scripting (rare)
`)
	res, err := LintBytes("test.yml", good)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed() {
		t.Fatalf("expected rule to pass, got issues: %+v", res.Issues)
	}
}

func TestLintBytes_MissingRequiredFields(t *testing.T) {
	bad := []byte(`
title: ""
detection:
  selection:
    Image|endswith: '\cmd.exe'
`)
	res, err := LintBytes("bad.yml", bad)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed() {
		t.Fatalf("expected rule to fail linting")
	}
	found := map[string]bool{}
	for _, iss := range res.Issues {
		found[iss.Field] = true
	}
	for _, want := range []string{"title", "id", "logsource", "detection.condition", "level", "tags"} {
		if !found[want] {
			t.Errorf("expected an issue for field %q, got issues: %+v", want, res.Issues)
		}
	}
}

func TestLintBytes_InvalidYAML(t *testing.T) {
	res, err := LintBytes("broken.yml", []byte("title: [unclosed"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed() {
		t.Fatal("expected invalid YAML to fail")
	}
}

func TestPassedStrict_FailsOnWarningsOnly(t *testing.T) {
	// A rule with zero errors but at least one warning (missing
	// falsepositives, in this case) should pass the normal (lenient)
	// check but fail the strict one — proving --strict / the GitHub
	// Action's fail-on-warning input actually changes behavior.
	rule := []byte(`
title: Warning-only rule
id: 22222222-2222-2222-2222-222222222222
level: medium
tags:
  - attack.t1059.001
logsource:
  category: process_creation
detection:
  selection:
    x: y
  condition: selection
`)
	res, err := LintBytes("warn.yml", rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed() {
		t.Fatalf("expected lenient Passed() to succeed despite warnings, got issues: %+v", res.Issues)
	}
	hasWarning := false
	for _, i := range res.Issues {
		if i.Severity == "warning" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Fatalf("test setup expected at least one warning-severity issue, got: %+v", res.Issues)
	}
	if res.PassedStrict() {
		t.Fatal("expected PassedStrict() to fail when warnings are present")
	}
}
