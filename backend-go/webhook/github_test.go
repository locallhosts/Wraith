package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func signatureFor(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignatureRejectsEmptySecret(t *testing.T) {
	payload := []byte(`{"action":"opened"}`)
	if err := VerifySignature(payload, signatureFor(payload, ""), ""); err == nil {
		t.Fatal("expected empty webhook secret to be rejected")
	}
}

func TestParsePullRequestEventRejectsOversizedBodyBeforeVerification(t *testing.T) {
	body := bytes.NewReader([]byte(strings.Repeat("x", int(MaxWebhookBodyBytes+1))))
	req, err := http.NewRequest(http.MethodPost, "/webhook/github", body)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParsePullRequestEvent(req, "configured-secret")
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestParsePullRequestEventAcceptsValidSignature(t *testing.T) {
	payload := []byte(`{"action":"opened","number":12,"repository":{"full_name":"example/rules"}}`)
	req, err := http.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Hub-Signature-256", signatureFor(payload, "configured-secret"))
	evt, err := ParsePullRequestEvent(req, "configured-secret")
	if err != nil {
		t.Fatalf("valid signed payload rejected: %v", err)
	}
	if evt.Action != "opened" || evt.Number != 12 || evt.Repository.FullName != "example/rules" {
		t.Fatalf("unexpected decoded event: %#v", evt)
	}
}

func TestParsePullRequestEventRejectsInvalidSignature(t *testing.T) {
	payload := []byte(`{"action":"opened"}`)
	req, err := http.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Hub-Signature-256", signatureFor(payload, "wrong-secret"))
	if _, err := ParsePullRequestEvent(req, "configured-secret"); err == nil {
		t.Fatal("expected invalid signature to be rejected")
	}
}
