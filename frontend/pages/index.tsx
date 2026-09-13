import useSWR from "swr";
import Link from "next/link";
import { fetcher, RunStatus } from "../lib/api";
import ApiKeyBar from "../components/ApiKeyBar";

const STAGE_LABEL: Record<string, string> = {
  lint: "Linting rule",
  provision: "Provisioning test range",
  simulate: "Simulating attack chain",
  validate: "Validating detections",
  soar: "Drafting SOAR playbook",
  done: "Complete",
  failed: "Failed",
};

function StageBadge({ run }: { run: RunStatus }) {
  const isTerminal = run.stage === "done" || run.stage === "failed";
  const passed = run.passed;
  let color = "bg-zinc-700 text-zinc-200";
  if (isTerminal) {
    color = passed === false || run.stage === "failed"
      ? "bg-rose-950 text-rose-300 border border-rose-800"
      : "bg-emerald-950 text-emerald-300 border border-emerald-800";
  } else {
    color = "bg-amber-950 text-amber-300 border border-amber-800";
  }
  return (
    <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-mono tracking-tight ${color}`}>
      {STAGE_LABEL[run.stage] ?? run.stage}
    </span>
  );
}

export default function Home() {
  const { data: runs, error, isLoading } = useSWR<RunStatus[]>("/runs", fetcher, {
    refreshInterval: 4000,
  });

  return (
    <main className="min-h-screen bg-[#0d1117] text-zinc-200">
      <header className="border-b border-zinc-800 px-8 py-6">
        <div className="mx-auto flex max-w-5xl items-baseline justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-zinc-100">WRAITH</h1>
            <p className="mt-1 text-sm text-zinc-500">Predictive detection engineering — every rule fights a synthetic attacker before it reaches your SIEM.</p>
          </div>
          <span className="font-mono text-xs text-zinc-600">{runs ? `${runs.length} runs` : ""}</span>
        </div>
        <div className="mx-auto mt-4 flex max-w-5xl justify-end">
          <ApiKeyBar />
        </div>
      </header>

      <section className="mx-auto max-w-5xl px-8 py-10">
        {isLoading && <p className="text-sm text-zinc-500">Loading runs…</p>}
        {error && (
          <p className="text-sm text-rose-400">
            Couldn&apos;t reach the WRAITH API. Is the backend running? (expected at NEXT_PUBLIC_API_BASE)
          </p>
        )}
        {runs && runs.length === 0 && (
          <div className="rounded border border-dashed border-zinc-800 px-6 py-14 text-center">
            <p className="text-zinc-400">No pipeline runs yet.</p>
            <p className="mt-1 text-sm text-zinc-600">Open a pull request that touches a rule in /rules to trigger one.</p>
          </div>
        )}

        <ul className="divide-y divide-zinc-800">
          {runs?.map((run) => (
            <li key={run.run_id}>
              <Link
                href={`/runs/${run.run_id}`}
                className="flex items-center justify-between gap-4 py-4 transition hover:bg-zinc-900/60 -mx-3 px-3 rounded"
              >
                <div className="min-w-0">
                  <p className="truncate font-mono text-sm text-zinc-200">{run.rule_path}</p>
                  <p className="mt-0.5 text-xs text-zinc-600">
                    {run.repo} · PR #{run.pr_number} · run {run.run_id}
                  </p>
                </div>
                <StageBadge run={run} />
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
