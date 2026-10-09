import useSWR from "swr";
import Link from "next/link";
import WraithShell from "../components/WraithShell";
import MetricCard from "../components/security/MetricCard";
import Pipeline from "../components/security/Pipeline";
import StatusBadge from "../components/security/StatusBadge";
import { fetcher, RunStatus, fetchHealth } from "../lib/api";

export default function Dashboard() {
  const { data: runs } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 5000 });
  const { data: health } = useSWR("/healthz", fetchHealth, { refreshInterval: 10000 });
  const total = runs?.length ?? 0;
  const passed = runs?.filter((r) => r.stage === "done" && r.passed === true).length ?? 0;
  const failed = runs?.filter((r) => r.stage === "failed" || r.passed === false).length ?? 0;
  const active = total - passed - failed;

  return (
    <WraithShell>
      <section className="page-hero">
        <div><div className="eyebrow">Security engineering / operations</div><h1>Command Center</h1><p>Observe validation, adversarial testing, provenance and deployment readiness from one control surface.</p></div>
        <Link href="/playground" className="primary-button">Open playground →</Link>
      </section>

      <div className="metric-grid">
        <MetricCard label="Validation runs" value={total} detail="Persisted pipeline executions" />
        <MetricCard label="Passed" value={passed} detail="Quality gates satisfied" tone="green" />
        <MetricCard label="Active" value={active} detail="Currently processing" tone="amber" />
        <MetricCard label="Failed" value={failed} detail="Requires investigation" tone="rose" />
      </div>

      <div className="dashboard-grid">
        <section className="panel panel-large">
          <div className="panel-heading"><div><h2>Detection validation pipeline</h2><p>Every rule moves through an evidence-producing security workflow.</p></div><StatusBadge status="Live" /></div>
          <Pipeline active={4} />
        </section>
        <section className="panel">
          <div className="panel-heading"><div><h2>Service health</h2><p>Control-plane readiness</p></div></div>
          <div className="health-list">
            {["API / Control plane", "Detection engine", "PostgreSQL", "Elasticsearch", "Neo4j"].map((name) => <div className="health-row" key={name}><span>{name}</span><StatusBadge status={health?.status ? "Online" : "Checking"} /></div>)}
          </div>
        </section>
      </div>

      <section className="panel">
        <div className="panel-heading"><div><h2>Recent validation activity</h2><p>Live runs are refreshed every five seconds.</p></div><Link href="/runs/active" className="text-link">View all →</Link></div>
        <div className="run-table">
          <div className="run-head"><span>Rule</span><span>Repository</span><span>Stage</span><span>Updated</span></div>
          {(runs ?? []).slice(0, 8).map((run) => <Link className="run-row" href={`/runs/${run.run_id}`} key={run.run_id}><span><strong>{run.rule_title || run.rule_path}</strong><small>{run.run_id}</small></span><span>{run.repo || "—"}</span><StatusBadge status={run.stage} /><span>{new Date(run.updated_at).toLocaleString()}</span></Link>)}
          {!runs?.length && <div className="empty-state">No validation runs are available yet.</div>}
        </div>
      </section>
    </WraithShell>
  );
}
