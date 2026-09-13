// Package webhook verifies and parses inbound GitHub pull_request webhooks
// so the pipeline only runs against real, authenticated GitHub events.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PullRequestEvent is the subset of GitHub's pull_request webhook payload
// this pipeline cares about.
type PullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Head struct {
			Sha string `json:"sha"`
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		ChangedFiles int `json:"changed_files"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// VerifySignature validates the X-Hub-Signature-256 header GitHub sends,
// per https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
func VerifySignature(payload []byte, signatureHeader, secret string) error {
	if signatureHeader == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}
	const prefix = "sha256="
	if len(signatureHeader) <= len(prefix) || signatureHeader[:len(prefix)] != prefix {
		return fmt.Errorf("unexpected signature format")
	}
	expectedHex := signatureHeader[len(prefix):]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	computed := mac.Sum(nil)
	computedHex := hex.EncodeToString(computed)

	expectedBytes, err := hex.DecodeString(expectedHex)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}
	if !hmac.Equal(computed, expectedBytes) {
		_ = computedHex // kept for debug logging if needed
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// ParsePullRequestEvent reads and verifies the request body, returning a
// decoded PullRequestEvent only if the signature is valid.
func ParsePullRequestEvent(r *http.Request, secret string) (*PullRequestEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	if err := VerifySignature(body, r.Header.Get("X-Hub-Signature-256"), secret); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}
	var evt PullRequestEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}
	return &evt, nil
}
