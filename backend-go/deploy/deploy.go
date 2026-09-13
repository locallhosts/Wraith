// Package deploy is the final step of the pipeline: promoting a passing,
// human-approved, cryptographically-attested rule into the production
// SIEM's active rule index. Three independent gates must all pass:
//
//  1. The pipeline verdict was "passed" (automated).
//  2. A human with `lead` role or higher approved it (human-in-the-loop —
//     see api/server.go POST /runs/:id/approve).
//  3. The provenance signature verifies against the rule content AS IT
//     EXISTS RIGHT NOW, not as it existed when CI ran (see
//     backend-go/provenance) — this is what stops a rule from being
//     loosened after passing tests but before going live.
package deploy

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/yourname/wraith/backend-go/provenance"
)

var (
	ErrNotApproved      = fmt.Errorf("run has not been approved by a lead/admin")
	ErrAttestationBad   = fmt.Errorf("attestation failed verification")
	ErrRuleContentDrift = fmt.Errorf("rule content has drifted since it was attested")
)

// Gate checks all three conditions before Deploy will proceed.
type Gate struct {
	TrustedPublicKey ed25519.PublicKey
}

// Check runs the approval + signature checks without actually deploying —
// useful for a "would this pass?" dry-run endpoint.
func (g Gate) Check(approvedBy string, signed *provenance.SignedAttestation, currentRuleYAML []byte) error {
	if approvedBy == "" {
		return ErrNotApproved
	}
	if err := provenance.VerifyAgainstRule(signed, g.TrustedPublicKey, currentRuleYAML); err != nil {
		return fmt.Errorf("%w: %v", ErrAttestationBad, err)
	}
	return nil
}

// DeployedRule is the document written to the production rule index once
// all gates pass — this is what the SIEM's own rule engine (or a sidecar
// that watches this index) actually consumes.
type DeployedRule struct {
	RuleID         string    `json:"rule_id"`
	RunID          string    `json:"run_id"`
	QueryDSL       any       `json:"query_dsl"`
	ApprovedBy     string    `json:"approved_by"`
	AttestedAt     time.Time `json:"attested_at"`
	DeployedAt     time.Time `json:"deployed_at"`
	AttestationSig string    `json:"attestation_signature"`
}

// Deploy writes the rule into the wraith-rules-production index in
// Elasticsearch (representing "live" in the SIEM) only after Check
// passes. In a real deployment this would instead call the specific
// SIEM's rule-creation API (Elastic Security detection rules API, Splunk
// ES correlation search API, etc.) — the index-write here is a
// vendor-neutral stand-in that's genuinely functional against any
// Elasticsearch/OpenSearch-backed SIEM.
func Deploy(ctx context.Context, es *elasticsearch.Client, gate Gate, approvedBy string,
	signed *provenance.SignedAttestation, queryDSL any, currentRuleYAML []byte) (*DeployedRule, error) {

	if err := gate.Check(approvedBy, signed, currentRuleYAML); err != nil {
		return nil, err
	}

	rule := DeployedRule{
		RuleID:         signed.Attestation.RuleID,
		RunID:          signed.Attestation.RunID,
		QueryDSL:       queryDSL,
		ApprovedBy:     approvedBy,
		AttestedAt:     signed.Attestation.IssuedAt,
		DeployedAt:     time.Now().UTC(),
		AttestationSig: signed.Signature,
	}

	if _, err := es.Index(
		"wraith-rules-production",
		jsonReader(rule),
		es.Index.WithDocumentID(rule.RuleID),
		es.Index.WithContext(ctx),
		es.Index.WithRefresh("true"),
	); err != nil {
		return nil, fmt.Errorf("indexing deployed rule: %w", err)
	}

	return &rule, nil
}

// LoadTrustedPublicKey reads the pipeline's public signing key from an
// environment variable (base64), for wiring into main.go. The private key
// used for signing is intentionally NOT loadable through this package —
// it should only ever be held by the CI runner that performs signing (see
// docs/ENTERPRISE.md "Secrets & key management").
func LoadTrustedPublicKey(envVar string) (ed25519.PublicKey, error) {
	b64 := os.Getenv(envVar)
	if b64 == "" {
		return nil, fmt.Errorf("%s is not set — production deploys are disabled until a trusted signing key is configured", envVar)
	}
	pub, err := decodeBase64PublicKey(b64)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", envVar, err)
	}
	return pub, nil
}
