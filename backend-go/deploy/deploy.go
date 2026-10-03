// Package deploy contains the production promotion lifecycle for Wraith.
package deploy

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/locallhosts/Wraith/backend-go/provenance"
)

const (
	ProductionIndex = "wraith-rules-production"
	HistoryIndex = "wraith-deployment-history"
)

var (
	ErrNotApproved = fmt.Errorf("run has not been approved by a lead/admin")
	ErrAttestationBad = fmt.Errorf("attestation failed verification")
	ErrRuleContentDrift = fmt.Errorf("rule content has drifted since it was attested")
	ErrAlreadyDeployed = fmt.Errorf("this exact run and rule content are already deployed")
	ErrConcurrentChange = fmt.Errorf("production rule changed since this deployment was recorded")
)

type Gate struct { TrustedPublicKey ed25519.PublicKey }

func (g Gate) Check(approvedBy string, signed *provenance.SignedAttestation, currentRuleYAML []byte) error {
	if approvedBy == "" { return ErrNotApproved }
	if signed == nil || signed.Attestation.RunID == "" || signed.Attestation.RuleID == "" {
		return fmt.Errorf("%w: incomplete attestation", ErrAttestationBad)
	}
	if !signed.Attestation.Passed { return fmt.Errorf("%w: attestation verdict is not passing", ErrAttestationBad) }
	if err := provenance.VerifyAgainstRule(signed, g.TrustedPublicKey, currentRuleYAML); err != nil {
		return fmt.Errorf("%w: %v", ErrAttestationBad, err)
	}
	return nil
}

type DeployedRule struct {
	RuleID string `json:"rule_id"`
	RunID string `json:"run_id"`
	QueryDSL any `json:"query_dsl"`
	ApprovedBy string `json:"approved_by"`
	AttestedAt time.Time `json:"attested_at"`
	DeployedAt time.Time `json:"deployed_at"`
	AttestationSig string `json:"attestation_signature"`
	ContentSHA256 string `json:"content_sha256"`
}

type DeploymentRecord struct {
	DeploymentID string `json:"deployment_id"`
	RuleID string `json:"rule_id"`
	RunID string `json:"run_id"`
	Actor string `json:"actor"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	Error string `json:"error,omitempty"`
	PreviousSource json.RawMessage `json:"previous_source,omitempty"`
	CurrentSource json.RawMessage `json:"current_source"`
}

func hashRule(content []byte) string { sum := sha256.Sum256(content); return hex.EncodeToString(sum[:]) }

func readCurrent(ctx context.Context, es *elasticsearch.Client, ruleID string) ([]byte, error) {
	res, err := es.Get(ProductionIndex, ruleID, es.Get.WithContext(ctx))
	if err != nil { return nil, fmt.Errorf("reading production rule: %w", err) }
	defer res.Body.Close()
	if res.StatusCode == 404 { return nil, nil }
	if res.IsError() {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<10))
		return nil, fmt.Errorf("reading production rule: HTTP %s: %s", res.Status(), string(body))
	}
	var envelope struct { Source json.RawMessage `json:"_source"` }
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil { return nil, fmt.Errorf("decoding production rule: %w", err) }
	return envelope.Source, nil
}

func indexDocument(ctx context.Context, es *elasticsearch.Client, index, id string, value any) error {
	res, err := es.Index(index, jsonReader(value), es.Index.WithDocumentID(id), es.Index.WithContext(ctx), es.Index.WithRefresh("true"))
	if err != nil { return err }
	defer res.Body.Close()
	if res.IsError() {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<10))
		return fmt.Errorf("elasticsearch write rejected: HTTP %s: %s", res.Status(), string(body))
	}
	return nil
}

func updateHistory(ctx context.Context, es *elasticsearch.Client, id string, fields map[string]any) error {
	res, err := es.Update(HistoryIndex, id, jsonReader(map[string]any{"doc": fields}), es.Update.WithContext(ctx), es.Update.WithRefresh("true"))
	if err != nil { return err }
	defer res.Body.Close()
	if res.IsError() {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 8<<10))
		return fmt.Errorf("deployment history update rejected: HTTP %s: %s", res.Status(), string(data))
	}
	return nil
}

func Deploy(ctx context.Context, es *elasticsearch.Client, gate Gate, approvedBy string,
	signed *provenance.SignedAttestation, queryDSL any, currentRuleYAML []byte) (*DeployedRule, error) {
	if err := gate.Check(approvedBy, signed, currentRuleYAML); err != nil { return nil, err }
	rule := DeployedRule{
		RuleID: signed.Attestation.RuleID, RunID: signed.Attestation.RunID,
		QueryDSL: queryDSL, ApprovedBy: approvedBy, AttestedAt: signed.Attestation.IssuedAt,
		DeployedAt: time.Now().UTC(), AttestationSig: signed.Signature,
		ContentSHA256: hashRule(currentRuleYAML),
	}
	previous, err := readCurrent(ctx, es, rule.RuleID)
	if err != nil { return nil, err }
	if previous != nil {
		var prev DeployedRule
		if json.Unmarshal(previous, &prev) == nil && prev.RunID == rule.RunID && prev.ContentSHA256 == rule.ContentSHA256 {
			return nil, ErrAlreadyDeployed
		}
	}
	current, _ := json.Marshal(rule)
	deploymentID := hashRule([]byte(rule.RunID+"|"+rule.ContentSHA256+"|"+rule.DeployedAt.Format(time.RFC3339Nano)))[:24]
	record := DeploymentRecord{DeploymentID: deploymentID, RuleID: rule.RuleID, RunID: rule.RunID, Actor: approvedBy, Status: "pending", CreatedAt: rule.DeployedAt, PreviousSource: previous, CurrentSource: current}
	if err := indexDocument(ctx, es, HistoryIndex, deploymentID, record); err != nil { return nil, fmt.Errorf("creating deployment record: %w", err) }
	if err := indexDocument(ctx, es, ProductionIndex, rule.RuleID, rule); err != nil {
		_ = updateHistory(ctx, es, deploymentID, map[string]any{"status":"failed","error":err.Error()})
		return nil, fmt.Errorf("indexing deployed rule: %w", err)
	}
	if err := updateHistory(ctx, es, deploymentID, map[string]any{"status":"verified","verified_at":time.Now().UTC()}); err != nil {
		return nil, fmt.Errorf("deployment succeeded but history verification failed: %w", err)
	}
	return &rule, nil
}

func DryRun(ctx context.Context, es *elasticsearch.Client, gate Gate, approvedBy string, signed *provenance.SignedAttestation, currentRuleYAML []byte) (map[string]any, error) {
	if err := gate.Check(approvedBy, signed, currentRuleYAML); err != nil { return nil, err }
	current, err := readCurrent(ctx, es, signed.Attestation.RuleID)
	if err != nil { return nil, err }
	result := map[string]any{"would_deploy":true,"mutated":false,"rule_id":signed.Attestation.RuleID,"run_id":signed.Attestation.RunID,"content_sha256":hashRule(currentRuleYAML),"already_deployed":false}
	if current != nil {
		var prev DeployedRule
		if json.Unmarshal(current, &prev) == nil && prev.RunID == signed.Attestation.RunID && prev.ContentSHA256 == hashRule(currentRuleYAML) {
			result["would_deploy"] = false
			result["already_deployed"] = true
		}
	}
	return result, nil
}

func Verify(ctx context.Context, es *elasticsearch.Client, ruleID, expectedRunID, expectedHash string) (*DeployedRule, error) {
	source, err := readCurrent(ctx, es, ruleID)
	if err != nil { return nil, err }
	if source == nil { return nil, fmt.Errorf("production rule %q is not deployed", ruleID) }
	var rule DeployedRule
	if err := json.Unmarshal(source, &rule); err != nil { return nil, fmt.Errorf("decoding deployed rule: %w", err) }
	if expectedRunID != "" && rule.RunID != expectedRunID { return nil, fmt.Errorf("deployment verification failed: run id mismatch") }
	if expectedHash != "" && rule.ContentSHA256 != expectedHash { return nil, fmt.Errorf("deployment verification failed: content hash mismatch") }
	return &rule, nil
}

func GetHistory(ctx context.Context, es *elasticsearch.Client, deploymentID string) (*DeploymentRecord, error) {
	res, err := es.Get(HistoryIndex, deploymentID, es.Get.WithContext(ctx))
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode == 404 { return nil, fmt.Errorf("deployment record %q not found", deploymentID) }
	if res.IsError() { body,_:=io.ReadAll(io.LimitReader(res.Body,8<<10)); return nil, fmt.Errorf("reading deployment record: HTTP %s: %s",res.Status(),string(body)) }
	var envelope struct { Source DeploymentRecord `json:"_source"` }
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil { return nil, err }
	return &envelope.Source, nil
}

func ListHistory(ctx context.Context, es *elasticsearch.Client, limit int) ([]DeploymentRecord, error) {
	if limit <= 0 || limit > 200 { limit = 50 }
	res, err := es.Search(es.Search.WithContext(ctx), es.Search.WithIndex(HistoryIndex), es.Search.WithSize(limit), es.Search.WithSort("created_at:desc"))
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.IsError() { body,_:=io.ReadAll(io.LimitReader(res.Body,8<<10)); return nil, fmt.Errorf("listing deployment history: HTTP %s: %s",res.Status(),string(body)) }
	var response struct { Hits struct { Hits []struct { Source DeploymentRecord `json:"_source"` } `json:"hits"` } `json:"hits"` }
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil { return nil, err }
	out:=make([]DeploymentRecord,0,len(response.Hits.Hits)); for _,hit:=range response.Hits.Hits { out=append(out,hit.Source) }; return out,nil
}

func Rollback(ctx context.Context, es *elasticsearch.Client, deploymentID string) (*DeploymentRecord, error) {
	record, err := GetHistory(ctx, es, deploymentID)
	if err != nil { return nil, err }
	if record.Status != "verified" { return nil, fmt.Errorf("deployment %s is not in a rollback-safe state", deploymentID) }
	current, err := readCurrent(ctx, es, record.RuleID)
	if err != nil { return nil, err }
	if current == nil { return nil, fmt.Errorf("production rule %q is already absent", record.RuleID) }
	var currentRule, deployed DeployedRule
	if json.Unmarshal(current,&currentRule)!=nil || json.Unmarshal(record.CurrentSource,&deployed)!=nil { return nil, fmt.Errorf("deployment record or current production rule is malformed") }
	if currentRule.RunID != deployed.RunID || currentRule.ContentSHA256 != deployed.ContentSHA256 { return nil, ErrConcurrentChange }
	if len(record.PreviousSource)==0 || string(record.PreviousSource)=="null" {
		res,err:=es.Delete(ProductionIndex,record.RuleID,es.Delete.WithContext(ctx),es.Delete.WithRefresh("true")); if err!=nil{return nil,err}; defer res.Body.Close()
		if res.IsError(){body,_:=io.ReadAll(io.LimitReader(res.Body,8<<10));return nil,fmt.Errorf("rollback delete rejected: HTTP %s: %s",res.Status(),string(body))}
	} else if err:=indexDocument(ctx,es,ProductionIndex,record.RuleID,json.RawMessage(record.PreviousSource)); err!=nil { return nil,fmt.Errorf("rollback restore rejected: %w",err) }
	if err:=updateHistory(ctx,es,deploymentID,map[string]any{"status":"rolled_back","verified_at":time.Now().UTC()});err!=nil{return nil,fmt.Errorf("rollback succeeded but history update failed: %w",err)}
	record.Status="rolled_back"
	return record,nil
}

func LoadTrustedPublicKey(envVar string) (ed25519.PublicKey,error) {
	b64:=os.Getenv(envVar); if b64=="" { return nil,fmt.Errorf("%s is not set — production deploys are disabled until a trusted signing key is configured",envVar) }
	pub,err:=decodeBase64PublicKey(b64); if err!=nil{return nil,fmt.Errorf("decoding %s: %w",envVar,err)}; return pub,nil
}
