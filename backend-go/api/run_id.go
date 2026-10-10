package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/locallhosts/Wraith/backend-go/webhook"
)

var commitSHARegex = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

func validateWebhookEvent(evt *webhook.PullRequestEvent) error {
	if evt == nil {
		return fmt.Errorf("missing webhook event")
	}
	if evt.Number <= 0 {
		return fmt.Errorf("invalid pull request number")
	}
	if strings.TrimSpace(evt.Repository.FullName) == "" {
		return fmt.Errorf("missing repository full name")
	}
	if !commitSHARegex.MatchString(evt.PullRequest.Head.Sha) {
		return fmt.Errorf("invalid pull request head commit SHA")
	}
	return nil
}

// buildRunID binds the full commit digest to a truncated cryptographic digest
// of the rule ID. This keeps IDs path-safe while avoiding the collision-prone
// 8-character truncation previously used for both values.
func buildRunID(commitSHA, ruleID string) (string, error) {
	if !commitSHARegex.MatchString(commitSHA) {
		return "", fmt.Errorf("invalid commit SHA")
	}
	ruleID = strings.TrimSpace(ruleID)
	if ruleID == "" {
		return "", fmt.Errorf("missing rule ID")
	}
	sum := sha256.Sum256([]byte(ruleID))
	return strings.ToLower(commitSHA) + "-" + hex.EncodeToString(sum[:8]), nil
}
