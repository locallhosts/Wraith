// Package notify sends real HTTPS webhook notifications to Slack (or any
// Slack-compatible incoming webhook, e.g. Mattermost) when a pipeline run
// completes, and separately when a run is awaiting SOC lead approval.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackNotifier struct {
	WebhookURL string
	httpClient *http.Client
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type slackMessage struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type string     `json:"type"`
	Text *slackText `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (n *SlackNotifier) send(ctx context.Context, msg slackMessage) error {
	if n == nil || n.WebhookURL == "" {
		return nil // notifications are optional; no-op if unconfigured
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting to slack: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// RunCompleted notifies on a terminal pipeline state.
func (n *SlackNotifier) RunCompleted(ctx context.Context, runID, ruleTitle, repo string, prNumber int, passed bool, reason string) error {
	emoji, verdict := ":white_check_mark:", "PASSED"
	if !passed {
		emoji, verdict = ":x:", "FAILED"
	}
	text := fmt.Sprintf("%s *WRAITH %s* — `%s`\n%s/pull/%d · run `%s`\n%s",
		emoji, verdict, ruleTitle, repo, prNumber, runID, reason)
	return n.send(ctx, slackMessage{Text: text})
}

// AwaitingApproval notifies the SOC lead channel that a passing rule is
// ready for the human-in-the-loop production approval gate.
func (n *SlackNotifier) AwaitingApproval(ctx context.Context, runID, ruleTitle string, robustnessScore float64) error {
	text := fmt.Sprintf(
		":hourglass_flowing_sand: *Rule ready for review* — `%s`\nRobustness score: %.0f%% of evasion variants still detected\nApprove: `POST /runs/%s/approve`",
		ruleTitle, robustnessScore*100, runID,
	)
	return n.send(ctx, slackMessage{Text: text})
}
