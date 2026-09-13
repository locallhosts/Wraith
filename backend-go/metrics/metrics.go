// Package metrics exposes Prometheus counters/histograms for the WRAITH
// pipeline so it can be scraped by a real Prometheus server and alerted on
// (e.g. "page on-call if pipeline failure rate > 20% over 1h").
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wraith",
		Name:      "pipeline_runs_total",
		Help:      "Total number of detection pipeline runs, labeled by final verdict.",
	}, []string{"verdict"}) // "passed" | "failed" | "error"

	RunDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "wraith",
		Name:      "pipeline_run_duration_seconds",
		Help:      "End-to-end duration of a single rule's pipeline run.",
		Buckets:   []float64{5, 15, 30, 60, 120, 300, 600},
	}, []string{"stage"})

	FalsePositiveRate = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "wraith",
		Name:      "rule_false_positive_rate",
		Help:      "False-positive rate observed for a rule against the synthetic baseline corpus.",
		Buckets:   []float64{0, 0.0001, 0.001, 0.01, 0.05, 0.1},
	}, []string{"rule_id"})

	RobustnessScore = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "wraith",
		Name:      "rule_robustness_score",
		Help:      "Fraction of adversarial evasion variants a rule still detected (1.0 = fully robust).",
		Buckets:   []float64{0, 0.25, 0.5, 0.75, 0.9, 1.0},
	}, []string{"rule_id"})

	ApprovalsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wraith",
		Name:      "approvals_total",
		Help:      "Total number of run approvals, labeled by approving role.",
	}, []string{"role"})

	DeploysTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wraith",
		Name:      "deploys_total",
		Help:      "Total number of rules deployed to production, labeled by outcome.",
	}, []string{"outcome"}) // "success" | "signature_invalid" | "error"

	WebhookRejectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "wraith",
		Name:      "webhook_rejections_total",
		Help:      "GitHub webhook deliveries rejected, labeled by reason.",
	}, []string{"reason"}) // "bad_signature" | "unsupported_action"
)
