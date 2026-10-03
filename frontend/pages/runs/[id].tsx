import { useRouter } from "next/router";
import { useState } from "react";
import useSWR, { mutate } from "swr";
import Link from "next/link";
import { fetcher, RunStatus, approveRun, deployRun, dryRunDeploy, fetchStages } from "../../lib/api";
import ReportViewer from "../../components/ReportViewer";

function ActionButton({
  label,
  onClick,
  disabled,
  variant = "default",
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  variant?: "default" | "primary";
}) {
  const base = "rounded px-3 py-1.5 text-sm font-medium transition disabled:opacity-40 disabled:cursor-not-allowed";
  const styles =
    variant === "primary"
      ? "bg-emerald-700 text-emerald-50 hover:bg-emerald-600"
      : "border border-zinc-700 text-zinc-200 hover:border-zinc-500";
  return (
    <button onClick={onClick} disabled={disabled} className={`${base} ${styles}`}>
      {label}
    </button>
  );
}

export default function RunDetail() {
  const router = useRouter();
  const { id } = router.query;
  const runKey = id ? `/runs/${id}` : null;
  const { data: run, error } = useSWR<RunStatus>(runKey, fetcher, { refreshInterval: 3000 });
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const { data: stages } = useSWR<any[]>(runKey ? `${runKey}/stages` : null, fetcher, { refreshInterval: 3000 });
  const [dryRun, setDryRun] = useState<any>(null);

  async function handleApprove() {
    if (!id) return;
    setBusy(true);
    setActionError(null);
    const { ok, data } = await approveRun(id as string);
    setBusy(false);
    if (!ok) {
      setActionError(data.error || "Approval failed — check your API key's role (requires lead or admin).");
      return;
    }
    mutate(runKey);
  }

  async function handleDryRun() { if (!id) return; setBusy(true); setActionError(null); const {ok,data}=await dryRunDeploy(id as string); setBusy(false); if(!ok){setActionError(data.error||"Dry-run failed.");return;} setDryRun(data); }

  async function handleDeploy() {
    if (!id) return;
    setBusy(true);
    setActionError(null);
    const { ok, data } = await deployRun(id as string);
    setBusy(false);
    if (!ok) {
      setActionError(data.error || "Deploy failed.");
      return;
    }
    mutate(runKey);
  }

  const canApprove = run && run.passed === true && !run.approved_by;
  const canDeploy = run && !!run.approved_by && !run.deployed_at;

  return (
    <main className="min-h-screen bg-[#0d1117] text-zinc-200">
      <header className="border-b border-zinc-800 px-8 py-6">
        <div className="mx-auto max-w-3xl">
          <Link href="/" className="text-xs text-zinc-500 hover:text-zinc-300">← all runs</Link>
          <h1 className="mt-2 font-mono text-lg text-zinc-100">{id}</h1>
        </div>
      </header>

      <section className="mx-auto max-w-3xl px-8 py-10">
        {error && <p className="text-sm text-rose-400">Run not found.</p>}
        {!run && !error && <p className="text-sm text-zinc-500">Loading…</p>}

        {run && (
          <div className="space-y-6">
            <div className="rounded border border-zinc-800 p-5">
              <dl className="grid grid-cols-2 gap-y-3 text-sm">
                <dt className="text-zinc-500">Rule</dt>
                <dd className="font-mono text-zinc-200">{run.rule_path}</dd>
                <dt className="text-zinc-500">Repository</dt>
                <dd className="text-zinc-200">{run.repo}</dd>
                <dt className="text-zinc-500">Pull request</dt>
                <dd className="text-zinc-200">#{run.pr_number}</dd>
                <dt className="text-zinc-500">Stage</dt>
                <dd className="text-zinc-200">{run.stage}</dd>
                {typeof run.passed === "boolean" && (
                  <>
                    <dt className="text-zinc-500">Verdict</dt>
                    <dd className={run.passed ? "text-emerald-400" : "text-rose-400"}>
                      {run.passed ? "Passed" : "Failed"}
                    </dd>
                  </>
                )}
                {run.reason && (
                  <>
                    <dt className="text-zinc-500">Reason</dt>
                    <dd className="text-zinc-300">{run.reason}</dd>
                  </>
                )}
                {run.approved_by && (
                  <>
                    <dt className="text-zinc-500">Approved by</dt>
                    <dd className="text-zinc-200">{run.approved_by}</dd>
                  </>
                )}
                {run.deployed_at && (
                  <>
                    <dt className="text-zinc-500">Deployed</dt>
                    <dd className="text-emerald-400">{new Date(run.deployed_at).toLocaleString()}</dd>
                  </>
                )}
              </dl>
            </div>

            <div className="rounded border border-zinc-800 p-5">
              <h2 className="mb-1 text-sm font-medium text-zinc-300">Production deploy gate</h2>
              <p className="mb-4 text-xs text-zinc-600">
                Requires a passing run, a <code className="font-mono">lead</code>+ approval, and a valid
                cryptographic attestation matching the rule&apos;s current content.
              </p>
              <div className="flex flex-wrap gap-3">
                <ActionButton label={busy ? "Working…" : "Dry-run"} onClick={handleDryRun} disabled={!!busy || !canDeploy} />
                <ActionButton
                  label={busy ? "Working…" : "Approve"}
                  onClick={handleApprove}
                  disabled={busy || !canApprove}
                />
                <ActionButton
                  label={busy ? "Working…" : "Deploy to production"}
                  onClick={handleDeploy}
                  disabled={busy || !canDeploy}
                  variant="primary"
                />
              </div>
              {actionError && <p className="mt-3 text-sm text-rose-400">{actionError}</p>}
              {!run.passed && (
                <p className="mt-3 text-xs text-zinc-600">This run hasn&apos;t passed yet — nothing to approve.</p>
              )}
            </div>

            {dryRun && <div className="rounded border border-zinc-800 p-5"><h2 className="text-sm font-medium text-zinc-300">Deployment dry-run</h2><pre className="mt-3 max-h-72 overflow-auto rounded bg-zinc-950 p-3 text-xs text-zinc-500">{JSON.stringify(dryRun, null, 2)}</pre></div>}
            <div className="rounded border border-zinc-800 p-5"><h2 className="mb-3 text-sm font-medium text-zinc-300">Live pipeline stages</h2><div className="space-y-2">{(stages ?? []).map((stage:any)=><div key={stage.id} className="flex items-center justify-between rounded bg-zinc-950 px-3 py-2"><div><span className="font-mono text-xs text-zinc-300">{stage.name}</span>{stage.reason&&<p className="mt-1 text-[11px] text-zinc-600">{stage.reason}</p>}</div><span className={`text-[11px] uppercase ${stage.status==="passed"?"text-emerald-400":stage.status==="failed"?"text-rose-400":stage.status==="running"?"text-amber-300":"text-zinc-500"}`}>{stage.status}</span></div>)}</div></div>
            <ReportViewer runId={id as string} />
          </div>
        )}
      </section>
    </main>
  );
}
