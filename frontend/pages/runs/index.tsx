import useSWR from "swr";
import Link from "next/link";
import WraithShell from "../../components/WraithShell";
import StatusBadge from "../../components/security/StatusBadge";
import { fetcher, RunStatus } from "../../lib/api";

export default function RunsIndex() {
  const { data: runs, error, isLoading } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 4000 });
  return <WraithShell eyebrow="Validation Runs">
    <section className="page-hero"><div><div className="eyebrow">Execution history</div><h1>Validation Runs</h1><p>Follow every pipeline execution from rule intake to evidence and deployment readiness.</p></div></section>
    <section className="panel"><div className="panel-heading"><div><h2>Pipeline executions</h2><p>Live state is refreshed automatically.</p></div></div>
      {isLoading && <div className="empty-state">Loading runs…</div>}
      {error && <div className="empty-state">Unable to reach the control plane. Check the API configuration and credentials.</div>}
      <div className="run-table"><div className="run-head"><span>Rule</span><span>Repository</span><span>Stage</span><span>Updated</span></div>{(runs ?? []).map((run) => <Link className="run-row" href={`/runs/${run.run_id}`} key={run.run_id}><span><strong>{run.rule_title || run.rule_path}</strong><small>{run.run_id}</small></span><span>{run.repo || "—"}</span><StatusBadge status={run.stage} /><span>{new Date(run.updated_at).toLocaleString()}</span></Link>)}{runs && runs.length === 0 && <div className="empty-state">No runs recorded.</div>}</div>
    </section>
  </WraithShell>;
}
