// Package provenance applies software supply-chain attestation concepts
// (think SLSA / in-toto / cosign) to detection engineering, where they are
// almost never used today. Every rule that passes the WRAITH pipeline gets
// a signed, tamper-evident attestation binding:
//
//   - the exact rule content (by hash)
//   - what it was tested against (attack chain + baseline size)
//   - the verdict and false-positive rate
//   - the robustness score against adversarial evasion variants
//   - when, and which pipeline run produced it
//
// The production deploy step (backend-go/deploy) refuses to promote a rule
// without a valid signature from the pipeline's key — closing the gap
// where someone with SIEM/API access could hand-edit a rule after it
// passed CI but before it ships, which is invisible to a plain "did CI
// pass" checkbox.
//
// Uses only Go's standard library crypto/ed25519 — no external dependency,
// which also means this package is fully verifiable offline.
package provenance

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Attestation is the signed statement produced for a rule that passed the
// pipeline. Field order/naming is stable because it's part of what gets
// signed (via canonical JSON below) — do not reorder without bumping
// SchemaVersion.
type Attestation struct {
	SchemaVersion     string    `json:"schema_version"`
	RunID             string    `json:"run_id"`
	RuleID            string    `json:"rule_id"`
	RuleContentSHA256 string    `json:"rule_content_sha256"`
	TestedTechniques  []string  `json:"tested_techniques"`
	BaselineEventsN   int       `json:"baseline_events_n"`
	FalsePositiveRate float64   `json:"false_positive_rate"`
	RobustnessScore   float64   `json:"robustness_score"`
	Passed            bool      `json:"passed"`
	IssuedAt          time.Time `json:"issued_at"`
	Issuer            string    `json:"issuer"` // e.g. "wraith-ci@github-actions"
}

// SignedAttestation bundles the attestation with its signature and the
// public key needed to verify it, so it's fully self-contained (e.g. to
// hand to an auditor without also handing them the key management system).
type SignedAttestation struct {
	Attestation Attestation `json:"attestation"`
	Signature   string      `json:"signature"`  // base64-encoded ed25519 signature
	PublicKey   string      `json:"public_key"` // base64-encoded ed25519 public key
}

const SchemaVersion = "wraith.provenance/v1"

// GenerateKeypair creates a new ed25519 signing key. In production this is
// generated once and the private key stored in a secrets manager (Vault,
// AWS KMS asymmetric signing key, etc.) — never committed to the repo. See
// docs/ENTERPRISE.md "Secrets & key management".
func GenerateKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// canonicalBytes produces a deterministic byte representation of an
// Attestation for signing. Using struct field order + json.Marshal is
// deterministic in Go for a fixed struct (map ordering is the usual
// culprit for non-determinism, and there are no maps here).
func canonicalBytes(a Attestation) ([]byte, error) {
	return json.Marshal(a)
}

// HashRuleContent returns the hex-free base64 SHA-256 of raw Sigma rule
// bytes, used to bind the attestation to the exact rule text that was
// tested — if a single byte changes after signing, verification fails.
func HashRuleContent(ruleYAML []byte) string {
	sum := sha256.Sum256(ruleYAML)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// Sign produces a SignedAttestation for the given Attestation using the
// pipeline's private key.
func Sign(a Attestation, priv ed25519.PrivateKey) (*SignedAttestation, error) {
	a.SchemaVersion = SchemaVersion
	if a.IssuedAt.IsZero() {
		a.IssuedAt = time.Now().UTC()
	}
	msg, err := canonicalBytes(a)
	if err != nil {
		return nil, fmt.Errorf("canonicalizing attestation: %w", err)
	}
	sig := ed25519.Sign(priv, msg)
	pub := priv.Public().(ed25519.PublicKey)

	return &SignedAttestation{
		Attestation: a,
		Signature:   base64.StdEncoding.EncodeToString(sig),
		PublicKey:   base64.StdEncoding.EncodeToString(pub),
	}, nil
}

// Verify checks that:
//  1. the signature is valid for the embedded attestation content, and
//  2. the embedded public key matches the trusted key supplied by the
//     caller (the deploy step should NEVER trust the public key embedded
//     in the attestation alone — that would let anyone self-sign — it
//     must compare against a pinned, out-of-band-distributed key).
func Verify(sa *SignedAttestation, trustedPub ed25519.PublicKey) error {
	embeddedPub, err := base64.StdEncoding.DecodeString(sa.PublicKey)
	if err != nil {
		return fmt.Errorf("decoding embedded public key: %w", err)
	}
	if !ed25519.PublicKey(embeddedPub).Equal(trustedPub) {
		return fmt.Errorf("attestation was signed by an untrusted key")
	}

	sig, err := base64.StdEncoding.DecodeString(sa.Signature)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}
	msg, err := canonicalBytes(sa.Attestation)
	if err != nil {
		return fmt.Errorf("canonicalizing attestation: %w", err)
	}
	if !ed25519.Verify(trustedPub, msg, sig) {
		return fmt.Errorf("signature verification failed — attestation may have been tampered with")
	}
	return nil
}

// VerifyAgainstRule additionally checks that the rule content presented at
// deploy time still matches what was actually tested — this is what
// catches a hand-edit after CI passed but before deploy.
func VerifyAgainstRule(sa *SignedAttestation, trustedPub ed25519.PublicKey, currentRuleYAML []byte) error {
	if err := Verify(sa, trustedPub); err != nil {
		return err
	}
	currentHash := HashRuleContent(currentRuleYAML)
	if currentHash != sa.Attestation.RuleContentSHA256 {
		return fmt.Errorf("rule content has changed since it was attested (expected sha256 %s, got %s) — re-run the pipeline",
			sa.Attestation.RuleContentSHA256, currentHash)
	}
	return nil
}
