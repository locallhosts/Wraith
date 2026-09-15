import useSWR from "swr";
import Link from "next/link";
import { useMemo, useState } from "react";
import { fetcher, RunStatus } from "../lib/api";
import ApiKeyBar from "../components/ApiKeyBar";

const STAGE_LABEL: Record<string, string> = {
  lint: "Linting",
  provision: "Provisioning",
  simulate: "Simulation",
  validate: "Validation",
  soar: "SOAR",
  done: "Complete",
  failed: "Failed",
};

function StatusBadge({ run }: { run: RunStatus }) {
  const terminal = run.stage === "done" || run.stage === "failed";
  const failed = run.stage === "failed" || run.passed === false;
  const tone = terminal
    ? failed
      ? "border-rose-900 bg-rose-950/60 text-rose-300"
      : "border-emerald-900 bg-emerald-950/60 text-emerald-300"
    : "border-amber-900 bg-amber-950/60 text-amber-300";
  return (
    <span className={`rounded border px-2 py-1 text-[11px] font-medium ${tone}`}>
      {STAGE_LABEL[run.stage] ?? run.stage}
    </span>
  );
}

function Metric({ label, value, detail }: { label: string; value: string | number; detail?: string }) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-[#11161d] p-4">
      <p className="text-[11px] uppercase tracking-[0.16em] text-zinc-600">{label}</p>
      <p className="mt-2 text-2xl font-semibold tracking-tight text-zinc-100">{value}</p>
      {detail && <p className="mt-1 text-xs text-zinc-600">{detail}</p>}
    </div>
  );
}

export default function Home() {
  const { data: runs, error, isLoading } = useSWR<RunStatus[]>("/runs", fetcher, { refreshInterval: 4000 });
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");

  const filtered = useMemo(() => {
    if (!runs) return [];
    const q = query.trim().toLowerCase();
    return runs.filter((run) => {
      const matchesQuery = !q || [run.run_id, run.rule_path, run.rule_title, run.repo, run.rule_id]
        .some((value) => value?.toLowerCase().includes(q));
      const matchesStatus = status === "all"
        || (status === "passed" && run.stage === "done" && run.passed === true)
        || (status === "failed" && (run.stage === "failed" || run.passed === false))
        || (status === "active" && run.stage !== "done" && run.stage !== "failed");
      return matchesQuery && matchesStatus;
    });
  }, [runs, query, status]);

  const total = runs?.length ?? 0;
  const passed = runs?.filter((r) => r.stage === "done" && r.passed === true).length ?? 0;
  const failed = runs?.filter((r) => r.stage === "failed" || r.passed === false).length ?? 0;
  const active = total - passed - failed;

  return (
    <main className="min-h-screen bg-[#0a0e13] text-zinc-200">
      <header className="border-b border-zinc-800 bg-[#0d1117]/95 px-6 py-5 lg:px-10">
        <div className="mx-auto max-w-7xl">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div className="flex items-center gap-3">
                <span className="h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_12px_rgba(52,211,153,0.55)]" />
                <h1 className="text-xl font-semibold tracking-[0.18em] text-zinc-100">WRAITH</h1>
                <span className="rounded border border-zinc-800 px-2 py-0.5 text-[10px] uppercase tracking-widest text-zinc-600">Detection Control Plane</span>
              </div>
              <p className="mt-2 max-w-2xl text-sm text-zinc-500">Validate detection logic against synthetic attack telemetry, benign baselines, and adversarial mutations before deployment.</p>
            </div>
            <ApiKeyBar />
          </div>
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-6 px-6 py-6 lg:grid-cols-[190px_1fr] lg:px-10">
        <aside className="hidden lg:block">
          <nav className="sticky top-6 space-y-1 text-sm">
            <div className="mb-4 px-3 text-[10px] uppercase tracking-[0.18em] text-zinc-700">Operations</div>
            <div className="rounded-md border border-zinc-800 bg-zinc-900/70 px-3 py-2 text-zinc-100">Overview</div>
            <a href="#runs" className="block rounded-md px-3 py-2 text-zinc-500 transition hover:bg-zinc-900 hover:text-zinc-300">Runs</a>
            <a href="#capabilities" className="block rounded-md px-3 py-2 text-zinc-500 transition hover:bg-zinc-900 hover:text-zinc-300">Capabilities</a>
            <div className="mt-6 mb-2 px-3 text-[10px] uppercase tracking-[0.18em] text-zinc-700">Services</div>
            <div className="flex items-center justify-between rounded-md px-3 py-2 text-zinc-500"><span>API</span><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /></div>
            <div className="flex items-center justify-between rounded-md px-3 py-2 text-zinc-500"><span>PostgreSQL</span><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /></div>
            <div className="flex items-center justify-between rounded-md px-3 py-2 text-zinc-500"><span>Elastic</span><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /></div>
            <div className="flex items-center justify-between rounded-md px-3 py-2 text-zinc-500"><span>Neo4j</span><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /></div>
          </nav>
        </aside>

        <section className="min-w-0">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Metric label="Validation runs" value={total} detail="Persisted pipeline executions" />
            <Metric label="Passed" value={passed} detail="Completed successfully" />
            <Metric label="Failed" value={failed} detail="Failed or rejected" />
            <Metric label="Active" value={active} detail="Currently processing" />
          </div>

          <div id="runs" className="mt-6 rounded-lg border border-zinc-800 bg-[#0f141b]">
            <div className="flex flex-col gap-3 border-b border-zinc-800 p-4 md:flex-row md:items-center md:justify-between">
              <div>
                <h2 className="text-sm font-semibold text-zinc-200">Validation runs</h2>
                <p className="mt-1 text-xs text-zinc-600">Live pipeline state refreshes every four seconds.</p>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row">
                <input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search runs, rules, repositories…"
                  className="w-full rounded-md border border-zinc-800 bg-[#0a0e13] px-3 py-2 text-xs text-zinc-300 outline-none placeholder:text-zinc-700 focus:border-zinc-600 sm:w-64"
                />
                <select value={status} onChange={(e) => setStatus(e.target.value)} className="rounded-md border border-zinc-800 bg-[#0a0e13] px-3 py-2 text-xs text-zinc-400 outline-none">
                  <option value="all">All status</option>
                  <option value="active">Active</option>
                  <option value="passed">Passed</option>
                  <option value="failed">Failed</option>
                </select>
              </div>
            </div>

            {isLoading && <div className="p-8 text-sm text-zinc-600">Loading validation runs…</div>}
            {error && <div className="m-4 rounded-md border border-rose-900/80 bg-rose-950/30 p-4 text-sm text-rose-300">Couldn&apos;t reach the WRAITH API. Check the API key and backend at NEXT_PUBLIC_API_BASE.</div>}
            {runs && filtered.length === 0 && <div className="p-10 text-center text-sm text-zinc-600">No runs match the current filters.</div>}

            <div className="divide-y divide-zinc-800/80">
              {filtered.map((run) => (
                <Link key={run.run_id} href={`/runs/${run.run_id}`} className="block px-4 py-4 transition hover:bg-zinc-900/40">
                  <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="truncate font-mono text-sm text-zinc-200">{run.rule_title || run.rule_path}</span>
                        <StatusBadge run={run} />
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-zinc-600">
                        <span className="font-mono">{run.run_id}</span>
                        <span>{run.repo} · PR #{run.pr_number}</span>
                        <span>{new Date(run.updated_at).toLocaleString()}</span>
                      </div>
                    </div>
                    <span className="text-xs text-zinc-700">Open run →</span>
                  </div>
                </Link>
              ))}
            </div>
          </div>

          <div id="capabilities" className="mt-6 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            {[
              ["Attack simulation", "Synthetic multi-stage telemetry mapped to MITRE ATT&CK."],
              ["Detection validation", "Attack hit and benign baseline false-positive checks."],
              ["Adversarial robustness", "Mutation testing exposes fragile detection logic."],
              ["Provenance", "Signed attestations gate human approval and deployment."],
            ].map(([title, description]) => (
              <div key={title} className="rounded-lg border border-zinc-800 bg-[#0f141b] p-4">
                <p className="text-xs font-semibold text-zinc-300">{title}</p>
                <p className="mt-2 text-xs leading-5 text-zinc-600">{description}</p>
              </div>
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}
