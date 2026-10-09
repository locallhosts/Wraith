import { useMemo, useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import { fetcher, RunStatus } from "../lib/api";

export default function EvidencePage() {
  const { data, error, isLoading } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 5000 });
  const [query, setQuery] = useState("");
  const runs = data ?? [];
  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return runs.filter((run) => !normalized || [
      run.run_id, run.rule_id, run.rule_title, run.rule_path, run.repo, run.stage, run.reason,
    ].some((value) => String(value ?? "").toLowerCase().includes(normalized)))
      .slice().sort((a, b) => Date.parse(b.updated_at || b.started_at) - Date.parse(a.updated_at || a.started_at));
  }, [runs, query]);

  return <WraithShell eyebrow="Detection Evidence">
    <section className="page-hero">
      <div><div className="eyebrow">Detection / evidence</div><h1>Validation evidence</h1><p>Browse persisted validation runs and open the run detail to inspect stages, events, reports, and exportable evidence. Evidence comes from the authenticated API; this page does not fabricate results.</p></div>
      <span className="service-pill"><i /> Refresh · 5s</span>
    </section>
    <section className="metric-grid">
      <div className="metric-card"><div className="metric-label">Runs returned</div><div className="metric-value">{data ? runs.length : "—"}</div><div className="metric-detail">Records in current API response</div></div>
      <div className="metric-card metric-green"><div className="metric-label">Passed</div><div className="metric-value">{data ? runs.filter((run) => run.passed === true).length : "—"}</div><div className="metric-detail">Explicit passing verdicts</div></div>
      <div className="metric-card metric-rose"><div className="metric-label">Failed</div><div className="metric-value">{data ? runs.filter((run) => run.passed === false || run.stage === "failed").length : "—"}</div><div className="metric-detail">Failed verdict or stage</div></div>
      <div className="metric-card metric-cyan"><div className="metric-label">Active</div><div className="metric-value">{data ? runs.filter((run) => !["done", "failed"].includes(run.stage)).length : "—"}</div><div className="metric-detail">Runs not yet terminal</div></div>
    </section>
    <section className="panel">
      <div className="panel-heading"><div><h2>Evidence inventory</h2><p>Search by run, rule, repository, stage, or failure reason.</p></div><span className="count-badge">{filtered.length} shown</span></div>
      <div className="runs-toolbar">
        <label className="runs-search"><span aria-hidden="true">⌕</span><input aria-label="Search validation evidence" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search run ID, rule, repository…" /><kbd>LIVE</kbd></label>
        <Link className="audit-reset" href="/runs/active">All validation runs ↗</Link>
        <Link className="audit-reset" href="/jobs">Pipeline jobs ↗</Link>
      </div>
      {isLoading && <div className="empty-state">Loading persisted evidence inventory…</div>}
      {error && <div role="alert" className="empty-state runs-error">Unable to load evidence. Check the API base URL, API key, and viewer permissions. <span>{error.message}</span></div>}
      {!isLoading && !error && <div className="jobs-list">
        {filtered.map((run) => <article className="job-card" key={run.run_id}>
          <div className="job-card-main">
            <div className="job-title-row"><strong>{run.run_id}</strong><span className={`status-badge ${run.passed === true ? "success" : run.passed === false || run.stage === "failed" ? "danger" : "pending"}`}>{run.passed === true ? "passed" : run.passed === false || run.stage === "failed" ? "failed" : run.stage}</span></div>
            <p className="job-rule">{run.rule_title || run.rule_path || run.rule_id || "Rule metadata unavailable"}</p>
            <div className="job-meta"><span>Rule ID <b>{run.rule_id || "—"}</b></span><span>Repository <b>{run.repo || "—"}</b></span><span>Updated <b>{run.updated_at || run.started_at || "—"}</b></span></div>
            {run.reason && <p className="playground-result-note">{run.reason}</p>}
          </div>
          <div className="job-card-actions"><Link className="audit-reset" href={`/runs/${encodeURIComponent(run.run_id)}`}>Inspect evidence ↗</Link></div>
        </article>)}
        {filtered.length === 0 && <div className="empty-state">{runs.length ? "No runs match this search." : "No validation runs have been recorded yet."}</div>}
      </div>}
    </section>
  </WraithShell>;
}
