import Link from "next/link";

const sections = [
  { title: "Detection Engineering", items: ["Rule analysis", "Sigma translation", "Rule diff & fingerprinting", "Mutation testing", "Robustness analysis"] },
  { title: "Validation", items: ["Attack simulation", "Benign baseline testing", "Detection validation", "Adversarial testing", "Quality scoring"] },
  { title: "Threat & Operations", items: ["MITRE ATT&CK mapping", "Attack-chain construction", "CI/CD integration", "SOAR integration", "Provenance and deployment controls"] },
];

export default function Introduction() {
  return (
    <main className="min-h-screen bg-[#0a0e13] text-zinc-200">
      <header className="border-b border-zinc-800 bg-[#0d1117]/95 px-6 py-5 lg:px-10">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-3">
            <Link href="/" className="text-xl font-semibold tracking-[0.18em] text-zinc-100">WRAITH</Link>
            <span className="rounded border border-zinc-800 px-2 py-0.5 text-[10px] uppercase tracking-widest text-zinc-600">Introduction</span>
          </div>
          <Link href="/" className="text-xs text-zinc-500 hover:text-zinc-300">Back to operations →</Link>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-6 py-10 lg:px-10">
        <section className="max-w-4xl">
          <p className="text-[10px] uppercase tracking-[0.2em] text-emerald-400">Detection engineering platform</p>
          <h1 className="mt-3 text-4xl font-semibold tracking-tight text-zinc-100 md:text-5xl">Build detections. Break them. Measure them. Validate them.</h1>
          <p className="mt-5 max-w-3xl text-base leading-7 text-zinc-500">Wraith is a security engineering platform for developing, translating, testing, validating, and operationalizing detection rules across security environments.</p>
        </section>

        <section className="mt-12 grid gap-4 lg:grid-cols-3">
          {sections.map((section) => (
            <div key={section.title} className="rounded-xl border border-zinc-800 bg-[#0f141b] p-5">
              <h2 className="text-sm font-semibold text-zinc-200">{section.title}</h2>
              <ul className="mt-4 space-y-3 text-xs text-zinc-500">
                {section.items.map((item) => <li key={item} className="flex gap-2"><span className="text-emerald-500">•</span>{item}</li>)}
              </ul>
            </div>
          ))}
        </section>

        <section className="mt-12 rounded-xl border border-zinc-800 bg-[#0f141b] p-6">
          <h2 className="text-sm font-semibold text-zinc-200">Detection lifecycle</h2>
          <div className="mt-6 grid gap-3 md:grid-cols-4 lg:grid-cols-8">
            {["Author", "Analyze", "Translate", "Simulate", "Validate", "Mutate", "Attest", "Deploy"].map((step, i) => (
              <div key={step} className="flex items-center gap-2">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-zinc-700 bg-[#0a0e13] text-xs font-mono text-zinc-300">{String(i + 1).padStart(2, "0")}</div>
                <span className="text-xs text-zinc-500">{step}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="mt-6 grid gap-6 lg:grid-cols-2">
          <div className="rounded-xl border border-zinc-800 bg-[#0f141b] p-6">
            <h2 className="text-sm font-semibold text-zinc-200">Architecture</h2>
            <pre className="mt-5 overflow-x-auto rounded-lg border border-zinc-800 bg-[#0a0e13] p-5 text-[11px] leading-6 text-zinc-500">{`GitHub / CI
     │
     ▼
Go API / Control Plane
     │
 ┌───┼───────────┐
 ▼   ▼           ▼
Postgres     Python Engine     Neo4j
                 │
        ┌────────┼────────┐
        ▼        ▼        ▼
      Sigma   Simulation Mutations
        │        │        │
        └────────┼────────┘
                 ▼
             Validation
                 │
                 ▼
          Attestation / Approval
                 │
                 ▼
             Deployment`}</pre>
          </div>
          <div className="rounded-xl border border-zinc-800 bg-[#0f141b] p-6">
            <h2 className="text-sm font-semibold text-zinc-200">Engineering principles</h2>
            <div className="mt-5 space-y-4 text-xs leading-6 text-zinc-500">
              <p><strong className="text-zinc-300">Evidence over claims.</strong> Validation results expose the underlying attack, baseline, mutation, and provenance evidence.</p>
              <p><strong className="text-zinc-300">Real integrations.</strong> The application surfaces actual Wraith services and persisted pipeline results rather than simulated dashboard state.</p>
              <p><strong className="text-zinc-300">Honest capability status.</strong> Features that are not yet exposed or verified remain clearly identified as planned instead of being presented as complete.</p>
            </div>
            <div className="mt-7 flex flex-wrap gap-3">
              <Link href="/" className="rounded-md border border-zinc-700 px-4 py-2 text-xs text-zinc-300 hover:bg-zinc-900">Open operations</Link>
              <Link href="/playground" className="rounded-md bg-zinc-100 px-4 py-2 text-xs font-medium text-zinc-900 hover:bg-white">Open playground</Link>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
