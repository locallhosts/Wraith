package api

import (
	"strings"
	"testing"

	"github.com/locallhosts/Wraith/backend-go/webhook"
)

func TestBuildRunIDIsDeterministicAndRuleSpecific(t *testing.T) {
	sha := strings.Repeat("a", 40)
	first, err := buildRunID(sha, "rule-one")
	if err != nil {
		t.Fatalf("build run ID: %v", err)
	}
	again, err := buildRunID(strings.ToUpper(sha), "rule-one")
	if err != nil {
		t.Fatalf("build run ID with uppercase SHA: %v", err)
	}
	if first != again {
		t.Fatalf("run ID should be deterministic and case-normalized: %q vs %q", first, again)
	}
	other, err := buildRunID(sha, "rule-two")
	if err != nil {
		t.Fatalf("build run ID for second rule: %v", err)
	}
	if first == other {
		t.Fatal("different rule IDs must produce different run IDs")
	}
	if len(first) != 57 {
		t.Fatalf("expected full SHA plus 16-hex rule digest, got %q (%d chars)", first, len(first))
	}
}

func TestBuildRunIDRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		sha string
		ruleID string
	}{
		{name: "short SHA", sha: "deadbeef", ruleID: "rule"},
		{name: "path traversal SHA", sha: "../etc/passwd", ruleID: "rule"},
		{name: "missing rule ID", sha: strings.Repeat("b", 40), ruleID: "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := buildRunID(tt.sha, tt.ruleID); err == nil {
				t.Fatal("expected invalid input to be rejected")
			}
		})
	}
}

func TestValidateWebhookEventRequiresValidMetadata(t *testing.T) {
	valid := &webhook.PullRequestEvent{Action: "opened", Number: 7}
	valid.Repository.FullName = "example/rules"
	valid.PullRequest.Head.Sha = strings.Repeat("c", 40)
	if err := validateWebhookEvent(valid); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	tests := []struct {
		name string
		event *webhook.PullRequestEvent
	}{
		{name: "nil event", event: nil},
		{name: "missing PR number", event: &webhook.PullRequestEvent{Number: 0}},
		{name: "missing repository", event: &webhook.PullRequestEvent{Number: 1}},
		{name: "invalid commit SHA", event: func() *webhook.PullRequestEvent {
			e := *valid
			e.PullRequest.Head.Sha = "not-a-sha"
			return &e
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWebhookEvent(tt.event); err == nil {
				t.Fatal("expected invalid event to be rejected")
			}
		})
	}
}
