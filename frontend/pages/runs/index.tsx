import { useMemo, useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import WraithShell from "../../components/WraithShell";
import StatusBadge from "../../components/security/StatusBadge";
import { fetcher, RunStatus } from "../../lib/api";

type RunFilter = "all" | "active" | "passed" | "failed";

function formatTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export default function RunsIndex() {
  const { data: runs, error, isLoading } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 4000 });
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<RunFilter>("all");
  const [sort, setSort] = useState<"newest" | "oldest">("newest");

  const list = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return [...(runs ?? [])]
      .filter(run => {
        const matchesQuery = !normalized || [run.run_id, run.rule_title, run.rule_path, run.rule_id, run.repo, String(run.pr_number ?? "")]
          .some(value => String(value ?? "").toLowerCase().includes(normalized));
        const matchesFilter = filter === "all"
          || (filter === "active" && !["done", "failed"].includes(run.stage))
          || (filter === "passed" && run.passed === true)
          || (filter === "failed" && (run.passed === false || run.stage === "failed"));
        return matchesQuery && matchesFilter;
      })
      .sort((a, b) => {
        const delta = new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
        return sort === "newest" ? -delta : delta;
      });
  }, [runs, query, filter, sort]);

  const summary = useMemo(() => ({
    total: (runs ?? []).length,
    active: (runs ?? []).filter(run => !["done", "failed"].includes(run.stage)).length,
    passed: (runs ?? []).filter(run => run.passed === true).length,
    failed: (runs ?? []).filter(run => run.passed === false || run.stage === "failed").length,
  }), [runs]);

  const filters: { label: string; value: RunFilter; count: number }[] = [
    { label: "All runs", value: "all", count: summary.total },
    { label: "Active", value: "active", count: summary.active },
    { label: "Passed", value: "passed", count: summary.passed },
    { label: "Failed", value: "failed", count: summary.failed },
  ];

  return <WraithShell eyebrow="Validation Runs">
    <section className="page-hero"><div><div className="eyebrow">Execution history</div><h1>Validation Runs</h1><p>Follow pipeline executions from rule intake through validation evidence, approval, and deployment readiness.</p></div><span className="service-pill"><i /> Auto-refresh · 4s</span></section>

    <section className="metric-grid run-summary-grid">
      <div className="metric-card"><div className="metric-label">Total runs</div><div className="metric-value">{summary.total}</div><div className="metric-detail">Persisted validation executions</div></div>
      <div className="metric-card metric-cyan"><div className="metric-label">Active</div><div className="metric-value">{summary.active}</div><div className="metric-detail">Not yet in a terminal stage</div></div>
      <div className="metric-card metric-green"><div className="metric-label">Passed</div><div className="metric-value">{summary.passed}</div><div className="metric-detail">Backend verdict is passing</div></div>
      <div className="metric-card metric-rose"><div className="metric-label">Failed</div><div className="metric-value">{summary.failed}</div><div className="metric-detail">Failed stage or verdict</div></div>
    </section>

    <section className="panel">
      <div className="panel-heading"><div><h2>Pipeline executions</h2><p>Search by run, rule, repository, or pull request. Select a run for stage history and evidence.</p></div><span className="count-badge">{list.length} shown</span></div>
      <div className="runs-toolbar">
        <label className="runs-search"><span aria-hidden="true">⌕</span><input aria-label="Search validation runs" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search run ID, rule, repository, PR…" /><kbd>LIVE</kbd></label>
        <label className="runs-sort">Sort <select aria-label="Sort runs" value={sort} onChange={e => setSort(e.target.value as "newest" | "oldest")}><option value="newest">Newest first</option><option value="oldest">Oldest first</option></select></label>
      </div>
      <div className="runs-filters" role="group" aria-label="Filter validation runs">{filters.map(item => <button type="button" key={item.value} aria-pressed={filter === item.value} onClick={() => setFilter(item.value)} className={filter === item.value ? "runs-filter active" : "runs-filter"}>{item.label}<span>{item.count}</span></button>)}</div>

      {isLoading && <div className="empty-state">Loading validation runs…</div>}
      {error && <div role="alert" className="empty-state runs-error">Unable to reach the control plane. Check the API configuration and credentials.</div>}
      {!isLoading && !error && <div className="run-table">
        <div className="run-head"><span>Rule / run ID</span><span>Repository</span><span>Stage</span><span>Updated</span></div>
        {list.map(run => <Link className="run-row" href={`/runs/${encodeURIComponent(run.run_id)}`} key={run.run_id}>
          <span><strong>{run.rule_title || run.rule_path || "Untitled rule"}</strong><small>{run.run_id}{run.rule_id ? ` · ${run.rule_id}` : ""}</small></span>
          <span>{run.repo || "—"}{run.pr_number ? <small>PR #{run.pr_number}</small> : null}</span>
          <span className="run-stage-cell"><StatusBadge status={run.stage} />{typeof run.passed === "boolean" && <small className={run.passed ? "run-verdict passed" : "run-verdict failed"}>{run.passed ? "Verdict passed" : "Verdict failed"}</small>}</span>
          <span>{formatTime(run.updated_at)}</span>
        </Link>)}
        {list.length === 0 && <div className="empty-state">{runs?.length ? "No runs match these filters. Try a different search or status." : "No runs recorded yet. Runs appear here when the pipeline receives work."}</div>}
      </div>}
    </section>
  </WraithShell>;
}
