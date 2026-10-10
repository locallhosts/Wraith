package provenance

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
)

func TestSignAndVerify_Valid(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	ruleContent := []byte("title: test rule\nid: abc123\n")
	att := Attestation{
		RunID:             "run-1",
		RuleID:            "abc123",
		RuleContentSHA256: HashRuleContent(ruleContent),
		TestedTechniques:  []string{"T1059.001", "T1053.005"},
		BaselineEventsN:   12000,
		FalsePositiveRate: 0.0,
		RobustnessScore:   0.9,
		Passed:            true,
		Issuer:            "wraith-ci@test",
	}

	signed, err := Sign(att, priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := Verify(signed, pub); err != nil {
		t.Fatalf("expected valid signature to verify, got error: %v", err)
	}
	if err := VerifyAgainstRule(signed, pub, ruleContent); err != nil {
		t.Fatalf("expected rule content match to verify, got error: %v", err)
	}
}

func TestVerify_RejectsTamperedContent(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	att := Attestation{RunID: "run-1", RuleID: "abc123", Passed: true}
	signed, _ := Sign(att, priv)

	// Tamper with the signed payload after the fact.
	signed.Attestation.Passed = false // flip a fail to a pass (or vice versa) post-signing

	if err := Verify(signed, pub); err == nil {
		t.Fatal("expected tampered attestation to fail verification, but it passed")
	}
}

func TestVerify_RejectsUntrustedKey(t *testing.T) {
	_, priv, _ := GenerateKeypair()
	otherPub, _, _ := GenerateKeypair() // a different, "untrusted" key

	att := Attestation{RunID: "run-1", RuleID: "abc123", Passed: true}
	signed, _ := Sign(att, priv)

	err := Verify(signed, otherPub)
	if err == nil {
		t.Fatal("expected verification against a different trusted key to fail")
	}
	if !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("expected 'untrusted key' error, got: %v", err)
	}
}

func TestVerifyAgainstRule_DetectsPostSigningEdit(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	originalRule := []byte("title: original rule\ndetection: strict\n")
	att := Attestation{
		RunID:             "run-1",
		RuleID:            "abc123",
		RuleContentSHA256: HashRuleContent(originalRule),
		Passed:            true,
	}
	signed, _ := Sign(att, priv)

	// Simulate someone loosening the rule after it was tested and signed,
	// but before it reaches the deploy step.
	editedRule := []byte("title: original rule\ndetection: loosened-to-avoid-noise\n")

	err := VerifyAgainstRule(signed, pub, editedRule)
	if err == nil {
		t.Fatal("expected post-signing edit to be detected and rejected")
	}
	if !strings.Contains(err.Error(), "changed since it was attested") {
		t.Fatalf("expected drift-detection error, got: %v", err)
	}
}

func TestHashRuleContent_Deterministic(t *testing.T) {
	content := []byte("some rule content")
	if HashRuleContent(content) != HashRuleContent(content) {
		t.Fatal("expected deterministic hashing")
	}
}


func TestVerify_RejectsNilAttestation(t *testing.T) {
	pub, _, err := GenerateKeypair()
	if err != nil { t.Fatalf("keygen: %v", err) }
	if err := Verify(nil, pub); err == nil {
		t.Fatal("expected nil signed attestation to be rejected")
	}
}

func TestVerify_RejectsMalformedKeyAndSignatureLengths(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil { t.Fatalf("keygen: %v", err) }
	signed, err := Sign(Attestation{RunID: "run-1", RuleID: "rule-1", Passed: true}, priv)
	if err != nil { t.Fatalf("sign: %v", err) }

	badPublic := *signed
	badPublic.PublicKey = base64.StdEncoding.EncodeToString([]byte("short"))
	if err := Verify(&badPublic, pub); err == nil {
		t.Fatal("expected malformed embedded public key to be rejected")
	}

	badSignature := *signed
	badSignature.Signature = base64.StdEncoding.EncodeToString([]byte("short"))
	if err := Verify(&badSignature, pub); err == nil {
		t.Fatal("expected malformed signature to be rejected")
	}

	if err := Verify(signed, ed25519.PublicKey([]byte("short"))); err == nil {
		t.Fatal("expected malformed trusted public key to be rejected")
	}
}

func TestVerify_RejectsUnsupportedSchemaVersion(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil { t.Fatalf("keygen: %v", err) }
	signed, err := Sign(Attestation{RunID: "run-1", RuleID: "rule-1", Passed: true}, priv)
	if err != nil { t.Fatalf("sign: %v", err) }
	signed.Attestation.SchemaVersion = "wraith.provenance/v999"
	if err := Verify(signed, pub); err == nil {
		t.Fatal("expected unsupported schema version to be rejected")
	}
}

func TestSign_RejectsMalformedPrivateKey(t *testing.T) {
	if _, err := Sign(Attestation{}, ed25519.PrivateKey([]byte("short"))); err == nil {
		t.Fatal("expected malformed private key to be rejected")
	}
}
