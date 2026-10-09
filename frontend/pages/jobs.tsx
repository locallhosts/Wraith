import { useMemo, useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import StatusBadge from "../components/security/StatusBadge";
import { fetcher, JobRecord, retryPipelineJob } from "../lib/api";

type JobFilter = "all" | "queued" | "running" | "failed" | "complete";
function formatTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export default function JobsPage() {
  const { data: jobs, error, isLoading, mutate } = useSWR<JobRecord[]>("/jobs?limit=500", fetcher, { refreshInterval: 4000 });
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<JobFilter>("all");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ kind: "success" | "error"; text: string } | null>(null);

  const summary = useMemo(() => ({
    total: (jobs ?? []).length,
    queued: (jobs ?? []).filter(job => job.status === "queued").length,
    running: (jobs ?? []).filter(job => job.status === "running").length,
    failed: (jobs ?? []).filter(job => job.status === "failed").length,
    complete: (jobs ?? []).filter(job => job.status === "completed" || job.status === "done").length,
  }), [jobs]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return [...(jobs ?? [])].filter(job => {
      const matchesText = !q || [job.id, job.run_id, job.rule_path, job.status, job.last_error]
        .some(value => String(value ?? "").toLowerCase().includes(q));
      const matchesFilter = filter === "all"
        || (filter === "queued" && job.status === "queued")
        || (filter === "running" && job.status === "running")
        || (filter === "failed" && (job.status === "failed" || job.status === "cancelled"))
        || (filter === "complete" && (job.status === "completed" || job.status === "done"));
      return matchesText && matchesFilter;
    }).sort((a, b) => new Date(b.updated_at || b.created_at || 0).getTime() - new Date(a.updated_at || a.created_at || 0).getTime());
  }, [jobs, query, filter]);

  async function retry(job: JobRecord) {
    if (job.id === undefined || job.id === null) return;
    const id = String(job.id);
    if (!window.confirm(`Retry failed job #${id}? The backend will requeue it and reset its run to lint.`)) return;
    setBusyId(id); setNotice(null);
    try {
      const result = await retryPipelineJob(id);
      if (!result.ok) {
        setNotice({ kind: "error", text: result.data?.error || "Retry rejected. Analyst role or higher is required." });
        return;
      }
      setNotice({ kind: "success", text: `Job #${id} was queued for retry.` });
      await mutate();
    } catch (err) {
      setNotice({ kind: "error", text: err instanceof Error ? err.message : "Could not retry this job." });
    } finally { setBusyId(null); }
  }

  const filters: { label: string; value: JobFilter; count: number }[] = [
    { label: "All jobs", value: "all", count: summary.total },
    { label: "Queued", value: "queued", count: summary.queued },
    { label: "Running", value: "running", count: summary.running },
    { label: "Failed", value: "failed", count: summary.failed },
    { label: "Completed", value: "complete", count: summary.complete },
  ];

  return <WraithShell eyebrow="Pipeline Operations">
    <section className="page-hero"><div><div className="eyebrow">Operations / queue</div><h1>Pipeline jobs</h1><p>Monitor worker execution, inspect retry history, and requeue recoverable failures without bypassing the API's analyst-role authorization.</p></div><span className="service-pill"><i /> Auto-refresh · 4s</span></section>
    <section className="metric-grid">
      <div className="metric-card"><div className="metric-label">Total jobs</div><div className="metric-value">{summary.total}</div><div className="metric-detail">Persisted queue records</div></div>
      <div className="metric-card metric-cyan"><div className="metric-label">Queued</div><div className="metric-value">{summary.queued}</div><div className="metric-detail">Waiting for a worker</div></div>
      <div className="metric-card metric-amber"><div className="metric-label">Running</div><div className="metric-value">{summary.running}</div><div className="metric-detail">Claimed by a worker</div></div>
      <div className="metric-card metric-rose"><div className="metric-label">Failed</div><div className="metric-value">{summary.failed}</div><div className="metric-detail">Failed jobs need review</div></div>
    </section>
    {notice && <div role="status" className={notice.kind === "success" ? "jobs-notice success" : "jobs-notice error"}>{notice.text}<button type="button" onClick={() => setNotice(null)} aria-label="Dismiss message">×</button></div>}
    <section className="panel">
      <div className="panel-heading"><div><h2>Queue inventory</h2><p>Retry is enabled only for failed or cancelled jobs. Each retry is audited by the backend.</p></div><span className="count-badge">{filtered.length} shown</span></div>
      <div className="runs-toolbar"><label className="runs-search"><span aria-hidden="true">⌕</span><input aria-label="Search pipeline jobs" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search job ID, run ID, rule, error…" /><kbd>LIVE</kbd></label></div>
      <div className="runs-filters" role="group" aria-label="Filter pipeline jobs">{filters.map(item => <button type="button" key={item.value} aria-pressed={filter === item.value} onClick={() => setFilter(item.value)} className={filter === item.value ? "runs-filter active" : "runs-filter"}>{item.label}<span>{item.count}</span></button>)}</div>
      {isLoading && <div className="empty-state">Loading pipeline queue…</div>}
      {error && <div role="alert" className="empty-state runs-error">Unable to load jobs. Check the API base URL, API key, and viewer permissions.</div>}
      {!isLoading && !error && <div className="jobs-list">
        {filtered.map(job => {
          const id = job.id === undefined || job.id === null ? "" : String(job.id);
          const canRetry = job.status === "failed" || job.status === "cancelled";
          return <article className="job-card" key={id || `${job.run_id}-${job.created_at}`}>
            <div className="job-card-main">
              <div className="job-title-row"><strong>Job #{id || "—"}</strong><StatusBadge status={job.status || "unknown"} /><span className="job-stage">{job.stage || "pipeline"}</span></div>
              <p className="job-rule">{job.rule_path || "Rule path not supplied"}</p>
              <div className="job-meta"><span>Run <Link href={job.run_id ? `/runs/${encodeURIComponent(job.run_id)}` : "/runs/active"}>{job.run_id || "—"}</Link></span><span>Attempts <b>{job.attempts ?? 0} / {job.max_attempts ?? "—"}</b></span><span>Updated <b>{formatTime(job.updated_at || job.created_at)}</b></span></div>
              {job.last_error && <details className="job-error-details"><summary>Last error / worker response</summary><pre>{job.last_error}</pre></details>}
            </div>
            <div className="job-card-actions">{canRetry ? <button type="button" disabled={busyId === id || !id} onClick={() => retry(job)} className="job-retry-button">{busyId === id ? "Requeuing…" : "Retry job ↻"}</button> : <span className="job-action-hint">{job.status === "queued" ? "Waiting for worker" : job.status === "running" ? "Worker owns job" : "No action available"}</span>}</div>
          </article>;
        })}
        {filtered.length === 0 && <div className="empty-state">{jobs?.length ? "No jobs match this filter." : "No pipeline jobs have been recorded yet."}</div>}
      </div>}
    </section>
  </WraithShell>;
}
