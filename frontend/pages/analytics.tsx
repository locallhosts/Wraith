import Link from "next/link";
import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import MetricCard from "../components/security/MetricCard";
import StatusBadge from "../components/security/StatusBadge";
import { fetcher, RunStatus, RuleRecord, JobRecord } from "../lib/api";

type RuleEnvelope = { count?: number; rules?: Array<RuleRecord & { id?: string; level?: string; passed?: boolean; issue_count?: number; error_count?: number; warning_count?: number }> };
type JobEnvelope = JobRecord[] | { jobs?: JobRecord[] };

const dayKey = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;

function ActivityChart({ runs }: { runs: RunStatus[] }) {
  const days = Array.from({ length: 7 }, (_, index) => {
    const date = new Date();
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() - (6 - index));
    return { key: dayKey(date), label: date.toLocaleDateString(undefined, { weekday: "short" }), count: 0, failed: 0 };
  });
  for (const run of runs) {
    const key = dayKey(new Date(run.started_at));
    const day = days.find((item) => item.key === key);
    if (day) {
      day.count += 1;
      if (run.stage === "failed" || run.passed === false) day.failed += 1;
    }
  }
  const max = Math.max(1, ...days.map((day) => day.count));
  const width = 560;
  const height = 164;
  const plotTop = 12;
  const plotHeight = 112;
  const step = width / days.length;
  const points = days.map((day, index) => `${(index * step) + step / 2},${plotTop + plotHeight - (day.count / max) * plotHeight}`).join(" ");
  return (
    <div className="analytics-chart">
      <div className="chart-legend"><span><i className="legend-cyan" />Validation runs</span><span><i className="legend-rose" />Failed runs</span></div>
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Validation runs over the last seven days" className="activity-svg">
        {[0, 1, 2, 3].map((line) => <line key={line} x1="0" x2={width} y1={plotTop + line * (plotHeight / 3)} y2={plotTop + line * (plotHeight / 3)} stroke="#1d2733" strokeDasharray="3 5" />)}
        <polyline points={points} fill="none" stroke="#38d9ff" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
        {days.map((day, index) => {
          const x = index * step + step / 2;
          const y = plotTop + plotHeight - (day.count / max) * plotHeight;
          return <g key={day.key}><circle cx={x} cy={y} r="4" fill="#070a0f" stroke="#38d9ff" strokeWidth="2" /><text x={x} y="151" fill="#718096" fontSize="10" textAnchor="middle">{day.label}</text><text x={x} y={Math.max(9, y - 10)} fill="#c8d3df" fontSize="10" textAnchor="middle">{day.count}</text></g>;
        })}
      </svg>
      <p className="chart-note">Counts are calculated from the runs returned by the control-plane API. Days with no recorded runs remain at zero.</p>
    </div>
  );
}

function SeverityBars({ rules }: { rules: NonNullable<RuleEnvelope["rules"]> }) {
  const levels = [
    { label: "Critical", match: ["critical"], color: "severity-critical" },
    { label: "High", match: ["high"], color: "severity-high" },
    { label: "Medium", match: ["medium", "moderate"], color: "severity-medium" },
    { label: "Low", match: ["low", "informational", "info"], color: "severity-low" },
    { label: "Unclassified", match: [], color: "severity-unknown" },
  ];
  const counts = levels.map((level) => ({
    ...level,
    count: rules.filter((rule) => {
      const value = String(rule.level || (rule as Record<string, unknown>).severity || "").toLowerCase();
      return level.match.length ? level.match.includes(value) : !["critical", "high", "medium", "moderate", "low", "informational", "info"].includes(value);
    }).length,
  }));
  const max = Math.max(1, ...counts.map((item) => item.count));
  return <div className="severity-list">{counts.map((item) => <div className="severity-row" key={item.label}><div className="severity-label"><span>{item.label}</span><strong>{item.count}</strong></div><div className="severity-track"><div className={item.color} style={{ width: `${(item.count / max) * 100}%` }} /></div></div>)}</div>;
}

export default function Analytics() {
  const runsQuery = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 10000 });
  const rulesQuery = useSWR<RuleEnvelope>("/rules", fetcher, { refreshInterval: 30000 });
  const jobsQuery = useSWR<JobEnvelope>("/jobs", fetcher, { refreshInterval: 10000 });
  const runs = runsQuery.data ?? [];
  const rules = rulesQuery.data?.rules ?? [];
  const jobs = Array.isArray(jobsQuery.data) ? jobsQuery.data : jobsQuery.data?.jobs ?? [];
  const failedRuns = runs.filter((run) => run.stage === "failed" || run.passed === false).length;
  const passedRuns = runs.filter((run) => run.stage === "done" && run.passed === true).length;
  const invalidRules = rules.filter((rule) => rule.passed === false).length;
  const queuedJobs = jobs.filter((job) => ["queued", "pending", "retry"].includes(String(job.status || "").toLowerCase())).length;
  const anyError = runsQuery.error || rulesQuery.error || jobsQuery.error;

  return (
    <WraithShell eyebrow="Security Analytics">
      <section className="page-hero">
        <div><div className="eyebrow">Evidence-led telemetry</div><h1>Security analytics</h1><p>Operational trends derived from Wraith validation runs, rule lint results, and pipeline jobs. This view deliberately avoids fabricated telemetry.</p></div>
        <Link href="/runs/active" className="primary-button">Inspect runs →</Link>
      </section>

      {anyError && <div className="error-box" role="alert">Some analytics sources could not be loaded. Check your API key and control-plane availability; available panels will still render.</div>}

      <div className="metric-grid">
        <MetricCard label="Rules inspected" value={rulesQuery.data?.count ?? rules.length} detail={rulesQuery.isLoading ? "Loading rule inventory…" : "Current control-plane inventory"} />
        <MetricCard label="Rules with findings" value={invalidRules} detail="Lint checks did not pass" tone={invalidRules ? "amber" : "green"} />
        <MetricCard label="Passed runs" value={passedRuns} detail="Completed with passing quality gate" tone="green" />
        <MetricCard label="Failed runs" value={failedRuns} detail="Review evidence and stage output" tone={failedRuns ? "rose" : "cyan"} />
      </div>

      <div className="dashboard-grid">
        <section className="panel">
          <div className="panel-heading"><div><h2>Validation activity</h2><p>Run starts across the last seven calendar days</p></div><StatusBadge status={runsQuery.isLoading ? "Loading" : runsQuery.error ? "Unavailable" : "Live data"} /></div>
          {runsQuery.isLoading ? <div className="empty-state">Loading run history…</div> : runsQuery.error ? <div className="empty-state">Run history is unavailable. Verify the API connection.</div> : <ActivityChart runs={runs} />}
        </section>
        <section className="panel">
          <div className="panel-heading"><div><h2>Rule severity profile</h2><p>Severity metadata from the current rule inventory</p></div><StatusBadge status={rulesQuery.error ? "Unavailable" : rulesQuery.isLoading ? "Loading" : "Live data"} /></div>
          {rulesQuery.isLoading ? <div className="empty-state">Loading rule inventory…</div> : rulesQuery.error ? <div className="empty-state">Rules could not be loaded.</div> : <SeverityBars rules={rules} />}
        </section>
      </div>

      <section className="panel">
        <div className="panel-heading"><div><h2>Pipeline queue overview</h2><p>Current job states returned by the backend</p></div><StatusBadge status={jobsQuery.error ? "Unavailable" : jobsQuery.isLoading ? "Loading" : "Live data"} /></div>
        <div className="metric-grid compact-metrics">
          <MetricCard label="Total jobs" value={jobs.length} detail="Visible queue records" />
          <MetricCard label="Queued / retrying" value={queuedJobs} detail="Waiting for worker capacity" tone={queuedJobs ? "amber" : "green"} />
          <MetricCard label="Running" value={jobs.filter((job) => ["running", "processing"].includes(String(job.status || "").toLowerCase())).length} detail="Currently being processed" tone="cyan" />
          <MetricCard label="Failed jobs" value={jobs.filter((job) => String(job.status || "").toLowerCase() === "failed").length} detail="Requires triage" tone={jobs.some((job) => String(job.status || "").toLowerCase() === "failed") ? "rose" : "green"} />
        </div>
        {jobsQuery.error && <div className="empty-state">Queue data unavailable. Confirm the API key has viewer access.</div>}
        {!jobsQuery.error && !jobsQuery.isLoading && jobs.length === 0 && <div className="empty-state">The API returned no pipeline jobs.</div>}
      </section>

      <p className="analytics-footnote">Refresh cadence: runs and jobs every 10 seconds; rules every 30 seconds. Counts reflect API responses, not external SIEM or vulnerability-scanner telemetry.</p>
    </WraithShell>
  );
}
