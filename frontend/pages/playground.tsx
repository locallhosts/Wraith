import Link from "next/link";

const planned = [
  ["Detection Lab", "Select a repository rule and run the real Wraith validation pipeline."],
  ["Attack Simulation", "Choose a supported target OS and inspect generated multi-stage telemetry."],
  ["Mutation Lab", "Explore evasion variants and see which representations bypass the compiled query."],
  ["Query Translation", "Inspect the backend-specific query produced from Sigma detection content."],
];

export default function Playground() {
  return (
    <main className="min-h-screen bg-[#0a0e13] text-zinc-200">
      <header className="border-b border-zinc-800 bg-[#0d1117]/95 px-6 py-5 lg:px-10">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-3">
            <Link href="/" className="text-xl font-semibold tracking-[0.18em] text-zinc-100">WRAITH</Link>
            <span className="rounded border border-zinc-800 px-2 py-0.5 text-[10px] uppercase tracking-widest text-zinc-600">Playground</span>
          </div>
          <Link href="/" className="text-xs text-zinc-500 hover:text-zinc-300">Back to operations →</Link>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-6 py-10 lg:px-10">
        <section className="max-w-3xl">
          <p className="text-[10px] uppercase tracking-[0.2em] text-amber-400">Controlled validation lab</p>
          <h1 className="mt-3 text-4xl font-semibold tracking-tight text-zinc-100">Experience the Wraith pipeline.</h1>
          <p className="mt-5 text-sm leading-7 text-zinc-500">The playground is the interactive entry point for running real Wraith detection workflows. The surface is intentionally marked as planned until the backend workflow is exposed safely through the API.</p>
        </section>

        <div className="mt-10 grid gap-4 md:grid-cols-2">
          {planned.map(([title, description]) => (
            <div key={title} className="rounded-xl border border-zinc-800 bg-[#0f141b] p-6">
              <div className="flex items-center justify-between gap-4">
                <h2 className="text-sm font-semibold text-zinc-200">{title}</h2>
                <span className="rounded border border-amber-900/70 bg-amber-950/30 px-2 py-1 text-[10px] uppercase tracking-wider text-amber-400">Planned</span>
              </div>
              <p className="mt-3 text-xs leading-6 text-zinc-600">{description}</p>
            </div>
          ))}
        </div>

        <section className="mt-6 rounded-xl border border-zinc-800 bg-[#0f141b] p-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="text-sm font-semibold text-zinc-200">Why this is not a mock console</h2>
              <p className="mt-2 max-w-2xl text-xs leading-6 text-zinc-600">When enabled, Playground will call the same Wraith control plane used by validation runs. Results will come from the simulator, detection compiler, Elasticsearch baseline, Neo4j attack graph, robustness engine, and persisted run report.</p>
            </div>
            <Link href="/" className="shrink-0 rounded-md border border-zinc-700 px-4 py-2 text-xs text-zinc-300 hover:bg-zinc-900">View verified run</Link>
          </div>
        </section>
      </div>
    </main>
  );
}
