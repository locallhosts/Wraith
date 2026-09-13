// Package validator queries the ephemeral Elasticsearch instance to
// determine whether a Sigma rule's underlying query actually matched the
// injected attack telemetry, and whether it also (incorrectly) matched the
// injected baseline noise.
package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

// Verdict is the outcome of validating one rule against one test run.
type Verdict struct {
	RuleID              string  `json:"rule_id"`
	FiredOnAttack       bool    `json:"fired_on_attack"`
	AttackHitCount      int     `json:"attack_hit_count"`
	FiredOnBaseline     bool    `json:"fired_on_baseline"`
	BaselineHitCount    int     `json:"baseline_hit_count"`
	BaselineDocsScanned int     `json:"baseline_docs_scanned"`
	FalsePositiveRate   float64 `json:"false_positive_rate"`
	Passed              bool    `json:"passed"`
	Reason              string  `json:"reason"`
}

// NewClient builds an Elasticsearch client pointed at the ephemeral
// per-run instance provisioned by the orchestrator package.
func NewClient(addr string) (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{addr},
	})
}

// Validate runs the rule's Elasticsearch Query DSL (translated ahead of
// time from Sigma via sigma-cli / pySigma in the Python engine) against
// the "attack" index and the "baseline" index, and produces a pass/fail
// verdict. A rule passes only if it fires on the attack data and produces
// zero hits on 24h of benign baseline traffic.
func Validate(ctx context.Context, es *elasticsearch.Client, ruleID string, esQueryDSL map[string]interface{}) (*Verdict, error) {
	v := &Verdict{RuleID: ruleID}

	attackHits, err := countHits(ctx, es, "wraith-attack-*", esQueryDSL)
	if err != nil {
		return nil, fmt.Errorf("querying attack index: %w", err)
	}
	v.AttackHitCount = attackHits
	v.FiredOnAttack = attackHits > 0

	baselineHits, err := countHits(ctx, es, "wraith-baseline-*", esQueryDSL)
	if err != nil {
		return nil, fmt.Errorf("querying baseline index: %w", err)
	}
	v.BaselineHitCount = baselineHits
	v.FiredOnBaseline = baselineHits > 0

	totalBaseline, err := countHits(ctx, es, "wraith-baseline-*", map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}})
	if err != nil {
		return nil, fmt.Errorf("counting baseline corpus: %w", err)
	}
	v.BaselineDocsScanned = totalBaseline
	if totalBaseline > 0 {
		v.FalsePositiveRate = float64(baselineHits) / float64(totalBaseline)
	}

	switch {
	case !v.FiredOnAttack:
		v.Passed = false
		v.Reason = "rule did not fire on injected attack telemetry (false negative)"
	case v.FiredOnBaseline:
		v.Passed = false
		v.Reason = fmt.Sprintf("rule fired %d time(s) on benign baseline traffic (false positive)", baselineHits)
	default:
		v.Passed = true
		v.Reason = "rule fired on attack telemetry and produced zero false positives on baseline"
	}

	return v, nil
}

func countHits(ctx context.Context, es *elasticsearch.Client, index string, query map[string]interface{}) (int, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, err
	}
	res, err := es.Count(
		es.Count.WithContext(ctx),
		es.Count.WithIndex(index),
		es.Count.WithBody(&buf),
		es.Count.WithIgnoreUnavailable(true),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return 0, fmt.Errorf("elasticsearch count returned status %s", res.Status())
	}
	var parsed struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Count, nil
}

// PerformanceProfile measures indexer overhead by comparing query profile
// timing before and after the candidate rule's query is added, using
// Elasticsearch's built-in Profile API rather than an external benchmark.
type PerformanceProfile struct {
	RuleID          string  `json:"rule_id"`
	QueryTimeMillis float64 `json:"query_time_millis"`
	ShardsHit       int     `json:"shards_hit"`
	OverheadPercent float64 `json:"overhead_percent"`
	Rejected        bool    `json:"rejected"`
}

// ProfileQuery runs the rule query with Elasticsearch's profiling enabled
// against the baseline index (representative of production volume) and
// computes overhead relative to a supplied prior baseline query time.
func ProfileQuery(ctx context.Context, es *elasticsearch.Client, ruleID string, esQueryDSL map[string]interface{}, priorBaselineMillis float64, rejectThresholdPct float64) (*PerformanceProfile, error) {
	profiled := map[string]interface{}{
		"profile": true,
		"query":   esQueryDSL["query"],
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(profiled); err != nil {
		return nil, err
	}
	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex("wraith-baseline-*"),
		es.Search.WithBody(&buf),
		es.Search.WithIgnoreUnavailable(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch profile search returned status %s", res.Status())
	}

	var parsed struct {
		Took    float64 `json:"took"`
		Profile struct {
			Shards []struct {
				Searches []struct {
					Query []struct {
						TimeInNanos float64 `json:"time_in_nanos"`
					} `json:"query"`
				} `json:"searches"`
			} `json:"shards"`
		} `json:"profile"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	p := &PerformanceProfile{RuleID: ruleID, QueryTimeMillis: parsed.Took, ShardsHit: len(parsed.Profile.Shards)}
	if priorBaselineMillis > 0 {
		p.OverheadPercent = ((p.QueryTimeMillis - priorBaselineMillis) / priorBaselineMillis) * 100.0
		p.Rejected = p.OverheadPercent > rejectThresholdPct
	}
	return p, nil
}
