import useSWR from "swr";
import Link from "next/link";
import WraithShell from "../components/WraithShell";
import StatusBadge from "../components/security/StatusBadge";
import { fetcher, RunStatus } from "../lib/api";

export default function Detections() {
  const { data: runs } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 5000 });
  const rules = Array.from(new Map((runs ?? []).map((r) => [r.rule_id || r.rule_path, r])).values());

  return <WraithShell eyebrow="Detection Engineering">
    <section className="page-hero"><div><div className="eyebrow">Detection engineering</div><h1>Rule Workspace</h1><p>Trace a detection from rule intake through validation, adversarial mutation and provenance.</p></div><Link href="/playground" className="primary-button">Analyze rule →</Link></section>
    <div className="workspace-grid">
      <section className="panel"><div className="panel-heading"><div><h2>Detection inventory</h2><p>Rules discovered from validation history.</p></div><span className="count-badge">{rules.length}</span></div>
        <div className="rule-list">{rules.map((rule) => <Link href={`/rules/${encodeURIComponent(rule.rule_path)}`} className="rule-card" key={rule.rule_id || rule.rule_path}><div><strong>{rule.rule_title || rule.rule_path}</strong><small>{rule.rule_id || "No rule ID"} · {rule.repo || "local"}</small></div><StatusBadge status={rule.passed === false ? "Failed" : rule.stage} /><span>→</span></Link>)}{!rules.length && <div className="empty-state">Run a rule through the playground to populate the workspace.</div>}</div>
      </section>
      <section className="panel"><div className="panel-heading"><div><h2>Quality model</h2><p>Evidence used by the deployment gate.</p></div></div>
        <div className="quality-stack">{[["Detection behavior","Attack telemetry must trigger the rule.","40 pts"],["False-positive control","Benign baselines should remain quiet.","20 pts"],["Robustness","Mutations measure evasion resistance.","20 pts"],["Evidence & provenance","Attestation binds the result to tested content.","20 pts"]].map(([a,b,c]) => <div className="quality-row" key={a}><div><strong>{a}</strong><span>{b}</span></div><b>{c}</b></div>)}</div>
      </section>
    </div>
  </WraithShell>;
}
