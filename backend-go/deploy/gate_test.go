package deploy

import (
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/locallhosts/Wraith/backend-go/provenance"
)

func signedTestAttestation(t *testing.T, content []byte, passed bool) (ed25519.PublicKey, ed25519.PrivateKey, *provenance.SignedAttestation) {
	t.Helper()
	pub, priv, err := provenance.GenerateKeypair()
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	signed, err := provenance.Sign(provenance.Attestation{
		RunID:             "run-123",
		RuleID:            "rule-123",
		RuleContentSHA256: provenance.HashRuleContent(content),
		Passed:            passed,
	}, priv)
	if err != nil {
		t.Fatalf("sign test attestation: %v", err)
	}
	return pub, priv, signed
}

func TestGateCheckRequiresApproval(t *testing.T) {
	content := []byte("title: Example\n")
	pub, _, signed := signedTestAttestation(t, content, true)

	err := (Gate{TrustedPublicKey: pub}).Check("", signed, content)
	if !errors.Is(err, ErrNotApproved) {
		t.Fatalf("expected ErrNotApproved, got %v", err)
	}
}

func TestGateCheckRejectsMissingOrIncompleteAttestation(t *testing.T) {
	pub, _, signed := signedTestAttestation(t, []byte("title: Example\n"), true)
	gate := Gate{TrustedPublicKey: pub}

	if err := gate.Check("security-lead", nil, []byte("title: Example\n")); !errors.Is(err, ErrAttestationBad) {
		t.Fatalf("expected missing attestation to fail closed, got %v", err)
	}

	incomplete := *signed
	incomplete.Attestation.RunID = ""
	if err := gate.Check("security-lead", &incomplete, []byte("title: Example\n")); !errors.Is(err, ErrAttestationBad) {
		t.Fatalf("expected incomplete attestation to fail closed, got %v", err)
	}
}

func TestGateCheckRejectsFailedAttestation(t *testing.T) {
	content := []byte("title: Example\n")
	pub, _, signed := signedTestAttestation(t, content, false)

	err := (Gate{TrustedPublicKey: pub}).Check("security-lead", signed, content)
	if !errors.Is(err, ErrAttestationBad) {
		t.Fatalf("expected failed verdict to be rejected, got %v", err)
	}
}

func TestGateCheckAcceptsTrustedPassingAttestation(t *testing.T) {
	content := []byte("title: Example\n")
	pub, _, signed := signedTestAttestation(t, content, true)

	if err := (Gate{TrustedPublicKey: pub}).Check("security-lead", signed, content); err != nil {
		t.Fatalf("valid trusted attestation should pass gate: %v", err)
	}
}

func TestGateCheckRejectsUntrustedSigningKey(t *testing.T) {
	content := []byte("title: Example\n")
	_, _, signed := signedTestAttestation(t, content, true)
	untrustedPub, _, err := provenance.GenerateKeypair()
	if err != nil {
		t.Fatalf("generate untrusted key: %v", err)
	}

	err = (Gate{TrustedPublicKey: untrustedPub}).Check("security-lead", signed, content)
	if !errors.Is(err, ErrAttestationBad) {
		t.Fatalf("expected untrusted key to be rejected, got %v", err)
	}
}

func TestGateCheckRejectsRuleContentDrift(t *testing.T) {
	original := []byte("title: Example\n")
	changed := []byte("title: Modified Example\n")
	pub, _, signed := signedTestAttestation(t, original, true)

	err := (Gate{TrustedPublicKey: pub}).Check("security-lead", signed, changed)
	if !errors.Is(err, ErrAttestationBad) {
		t.Fatalf("expected content drift to fail closed, got %v", err)
	}
}
