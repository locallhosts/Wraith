import { useRouter } from "next/router";
import { useMemo, useState } from "react";
import useSWR, { mutate } from "swr";
import Link from "next/link";
import { fetcher, RunStatus, PipelineStage, approveRun, deployRun, dryRunDeploy } from "../../lib/api";
import ReportViewer from "../../components/ReportViewer";

interface RunEvent {
  id: number;
  run_id: string;
  stage: string;
  level: string;
  message: string;
  created_at: string;
}

function ActionButton({ label, onClick, disabled, variant = "default" }: {
  label: string; onClick: () => void; disabled?: boolean; variant?: "default" | "primary";
}) {
  const base = "rounded px-3 py-1.5 text-sm font-medium transition disabled:opacity-40 disabled:cursor-not-allowed";
  const styles = variant === "primary"
    ? "bg-emerald-700 text-emerald-50 hover:bg-emerald-600"
    : "border border-zinc-700 text-zinc-200 hover:border-zinc-500";
  return <button type="button" onClick={onClick} disabled={disabled} className={`${base} ${styles}`}>{label}</button>;
}

function formatTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function tone(value?: string) {
  if (value === "passed" || value === "info" || value === "success") return "text-emerald-400";
  if (value === "failed" || value === "error" || value === "critical") return "text-rose-400";
  if (value === "running" || value === "warning") return "text-amber-300";
  return "text-zinc-400";
}

export default function RunDetail() {
  const router = useRouter();
  const { id } = router.query;
  const runId = typeof id === "string" ? id : "";
  const runKey = runId ? `/runs/${encodeURIComponent(runId)}` : null;
  const { data: run, error } = useSWR<RunStatus>(runKey, fetcher, { refreshInterval: 3000 });
  const { data: stages, error: stagesError } = useSWR<PipelineStage[]>(runKey ? `${runKey}/stages` : null, fetcher, { refreshInterval: 3000 });
  const { data: events, error: eventsError } = useSWR<RunEvent[]>(runKey ? `${runKey}/events?limit=500` : null, fetcher, { refreshInterval: 3000 });
  const { data: report } = useSWR<Record<string, unknown>>(runKey ? `${runKey}/report` : null, fetcher, { refreshInterval: 5000, shouldRetryOnError: false });
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [dryRun, setDryRun] = useState<unknown>(null);

  const orderedEvents = useMemo(() => [...(events ?? [])].sort((a, b) =>
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()), [events]);
  const stageSummary = useMemo(() => ({
    passed: (stages ?? []).filter(stage => stage.status === "passed").length,
    failed: (stages ?? []).filter(stage => stage.status === "failed").length,
    running: (stages ?? []).filter(stage => stage.status === "running").length,
  }), [stages]);

  async function runAction(action: (id: string) => Promise<{ ok: boolean; data: any }>, fallback: string) {
    if (!runId) return;
    setBusy(true);
    setActionError(null);
    try {
      const { ok, data } = await action(runId);
      if (!ok) { setActionError(data?.error || fallback); return; }
      await mutate(runKey);
      await mutate(`${runKey}/stages`);
      await mutate(`${runKey}/events?limit=500`);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : fallback);
    } finally {
      setBusy(false);
    }
  }

  async function handleDryRun() {
    if (!runId) return;
    setBusy(true); setActionError(null);
    try {
      const { ok, data } = await dryRunDeploy(runId);
      if (!ok) { setActionError(data?.error || "Dry-run failed."); return; }
      setDryRun(data);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Dry-run failed.");
    } finally { setBusy(false); }
  }

  function exportEvidence() {
    if (!run) return;
    const bundle = {
      schema_version: "wraith.run-evidence.v1",
      exported_at: new Date().toISOString(),
      run,
      stages: stages ?? [],
      events: events ?? [],
      report: report ?? null,
      report_available: Boolean(report),
    };
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `wraith-evidence-${run.run_id.replace(/[^a-zA-Z0-9._-]/g, "_")}.json`;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
  }

  const canApprove = run?.passed === true && !run.approved_by;
  const canDeploy = Boolean(run?.approved_by && !run.deployed_at);

  return (
    <main className="min-h-screen bg-[#0d1117] text-zinc-200">
      <header className="border-b border-zinc-800 px-5 py-5 sm:px-8">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-4">
          <div><Link href="/runs/active" className="text-xs text-zinc-500 hover:text-zinc-300">← all runs</Link><h1 className="mt-2 break-all font-mono text-lg text-zinc-100">{runId || "Loading run…"}</h1></div>
          {run && <button type="button" onClick={exportEvidence} className="rounded border border-cyan-900 px-3 py-2 text-xs text-cyan-300 hover:border-cyan-700">Export evidence JSON ↓</button>}
        </div>
      </header>

      <section className="mx-auto max-w-5xl space-y-5 px-5 py-8 sm:px-8">
        {error && <div role="alert" className="rounded border border-rose-900/60 p-4 text-sm text-rose-300">Run not found or access denied. Check the run ID and API credentials.</div>}
        {!run && !error && <p className="text-sm text-zinc-500">Loading run details…</p>}
        {run && <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded border border-zinc-800 bg-zinc-900/40 p-4"><p className="text-xs text-zinc-500">Current stage</p><p className="mt-2 text-lg font-medium">{run.stage}</p></div>
            <div className="rounded border border-zinc-800 bg-zinc-900/40 p-4"><p className="text-xs text-zinc-500">Verdict</p><p className={`mt-2 text-lg font-medium ${run.passed === true ? "text-emerald-400" : run.passed === false ? "text-rose-400" : "text-zinc-300"}`}>{run.passed === true ? "Passed" : run.passed === false ? "Failed" : "Pending"}</p></div>
            <div className="rounded border border-zinc-800 bg-zinc-900/40 p-4"><p className="text-xs text-zinc-500">Stages passed</p><p className="mt-2 text-lg font-medium">{stageSummary.passed}<span className="text-sm text-zinc-600"> / {(stages ?? []).length}</span></p></div>
            <div className="rounded border border-zinc-800 bg-zinc-900/40 p-4"><p className="text-xs text-zinc-500">Run events</p><p className="mt-2 text-lg font-medium">{(events ?? []).length}</p></div>
          </div>

          <section className="rounded border border-zinc-800 p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3"><h2 className="text-sm font-semibold">Run metadata</h2><span className={`text-xs uppercase ${tone(run.passed === true ? "passed" : run.passed === false ? "failed" : run.stage)}`}>{run.passed === true ? "passed" : run.passed === false ? "failed" : run.stage}</span></div>
            <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-[150px_1fr]">
              <dt className="text-zinc-500">Rule</dt><dd className="break-all font-mono text-zinc-200">{run.rule_title || run.rule_path || "—"}{run.rule_title && run.rule_path && <span className="mt-1 block text-xs text-zinc-500">{run.rule_path}</span>}</dd>
              <dt className="text-zinc-500">Rule ID</dt><dd className="break-all font-mono text-zinc-300">{run.rule_id || "—"}</dd>
              <dt className="text-zinc-500">Repository / PR</dt><dd className="text-zinc-200">{run.repo || "—"}{run.pr_number ? ` · #${run.pr_number}` : ""}</dd>
              <dt className="text-zinc-500">Started</dt><dd>{formatTime(run.started_at)}</dd>
              <dt className="text-zinc-500">Last updated</dt><dd>{formatTime(run.updated_at)}</dd>
              {run.reason && <><dt className="text-zinc-500">Reason</dt><dd className="text-zinc-300">{run.reason}</dd></>}
              {run.approved_by && <><dt className="text-zinc-500">Approved by</dt><dd>{run.approved_by}{run.approved_at ? ` · ${formatTime(run.approved_at)}` : ""}</dd></>}
              {run.deployed_at && <><dt className="text-zinc-500">Deployed at</dt><dd className="text-emerald-400">{formatTime(run.deployed_at)}</dd></>}
            </dl>
          </section>

          <section className="rounded border border-zinc-800 p-5">
            <div className="mb-3"><h2 className="text-sm font-semibold">Production deployment gate</h2><p className="mt-1 text-xs leading-5 text-zinc-500">Promotion requires a passing run, lead-or-higher approval, and valid cryptographic attestation verified by the backend.</p></div>
            <div className="flex flex-wrap gap-3">
              <ActionButton label={busy ? "Working…" : "Dry-run deployment"} onClick={handleDryRun} disabled={busy || !canDeploy} />
              <ActionButton label={busy ? "Working…" : "Approve run"} onClick={() => runAction(approveRun, "Approval failed — lead or admin role required.")} disabled={busy || !canApprove} />
              <ActionButton label={busy ? "Working…" : "Deploy to production"} onClick={() => runAction(deployRun, "Deployment failed.")} disabled={busy || !canDeploy} variant="primary" />
            </div>
            {actionError && <p role="alert" className="mt-3 text-sm text-rose-400">{actionError}</p>}
            {dryRun !== null && <div className="mt-4 rounded border border-zinc-800 bg-zinc-950 p-3"><div className="mb-2 flex items-center justify-between"><h3 className="text-xs font-medium text-zinc-300">Dry-run response</h3><button type="button" onClick={() => setDryRun(null)} className="text-xs text-zinc-500 hover:text-zinc-300">Dismiss</button></div><pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words text-xs text-zinc-400">{JSON.stringify(dryRun, null, 2)}</pre></div>}
          </section>

          <section className="rounded border border-zinc-800 p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h2 className="text-sm font-semibold">Pipeline stage history</h2><p className="mt-1 text-xs text-zinc-500">Persisted stage state refreshes automatically.</p></div><div className="flex gap-3 text-xs"><span className="text-emerald-400">{stageSummary.passed} passed</span><span className="text-rose-400">{stageSummary.failed} failed</span><span className="text-amber-300">{stageSummary.running} running</span></div></div>
            {stagesError && <p className="text-xs text-amber-300">Stage history is unavailable for this run.</p>}
            {!stagesError && (stages ?? []).length === 0 && <p className="text-sm text-zinc-500">No persisted stage records yet.</p>}
            <div className="space-y-2">{(stages ?? []).map(stage => <div key={stage.id} className="grid gap-2 rounded bg-zinc-950/80 p-3 sm:grid-cols-[minmax(0,1fr)_100px]"><div><div className="flex flex-wrap items-center gap-2"><span className="font-mono text-xs text-zinc-200">{stage.name}</span><span className={`text-[10px] uppercase ${tone(stage.status)}`}>{stage.status}</span></div>{stage.reason && <p className="mt-1 text-xs text-zinc-500">{stage.reason}</p>}<p className="mt-2 text-[10px] text-zinc-600">Started {formatTime(stage.started_at)}{stage.ended_at ? ` · Ended ${formatTime(stage.ended_at)}` : ""}</p></div></div>)}</div>
          </section>

          <section className="rounded border border-zinc-800 p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h2 className="text-sm font-semibold">Execution event timeline</h2><p className="mt-1 text-xs text-zinc-500">Chronological events recorded by the pipeline and operator actions.</p></div><span className="text-xs text-zinc-600">{orderedEvents.length} events · newest first</span></div>
            {eventsError && <p className="text-xs text-amber-300">Event history is unavailable. Verify API access and retry.</p>}
            {!eventsError && orderedEvents.length === 0 && <p className="text-sm text-zinc-500">No run events have been recorded yet.</p>}
            <ol className="space-y-0">{orderedEvents.map((event, index) => <li key={event.id || `${event.created_at}-${index}`} className="relative grid grid-cols-[14px_minmax(0,1fr)] gap-3 pb-5 last:pb-0"><div className="flex h-full flex-col items-center"><span className={`mt-1.5 h-2 w-2 rounded-full ${event.level === "error" || event.level === "critical" ? "bg-rose-400" : event.level === "warning" ? "bg-amber-300" : "bg-cyan-400"}`} />{index < orderedEvents.length - 1 && <span className="mt-1 w-px flex-1 bg-zinc-800" />}</div><div className="min-w-0"><div className="flex flex-wrap items-center gap-2"><span className="font-mono text-xs text-zinc-300">{event.stage || "pipeline"}</span><span className={`text-[9px] uppercase ${tone(event.level)}`}>{event.level || "info"}</span><time className="text-[10px] text-zinc-600">{formatTime(event.created_at)}</time></div><p className="mt-1 break-words text-sm leading-6 text-zinc-400">{event.message}</p></div></li>)}</ol>
          </section>

          <ReportViewer runId={runId} />
        </>}
      </section>
    </main>
  );
}
