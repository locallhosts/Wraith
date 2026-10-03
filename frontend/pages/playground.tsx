import { FormEvent, useState } from "react";
import Link from "next/link";

const example = `title: Suspicious PowerShell Execution
id: 11111111-1111-4111-8111-111111111111
status: test
description: Detects encoded PowerShell execution.
author: Wraith
logsource:
  product: windows
  service: powershell
detection:
  selection:
    CommandLine|contains: "-enc"
  condition: selection
falsepositives:
  - Administrative scripts
level: high
tags:
  - attack.t1059.001
`;

export default function Playground() {
  const [rule, setRule] = useState(example);
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function validate(e: FormEvent) {
    e.preventDefault();
    setError("");
    setResult(null);
    setBusy(true);
    try {
      const api = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";
      const key = typeof window === "undefined" ? "" : localStorage.getItem("wraith_api_key") || "";
      const response = await fetch(api + "/playground/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...(key ? { Authorization: "Bearer " + key } : {}) },
        body: JSON.stringify({ rule }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Playground validation failed");
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Playground validation failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="min-h-screen bg-[#0a0e13] text-zinc-200">
      <header className="border-b border-zinc-800 bg-[#0d1117] px-6 py-5">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div>
            <div className="text-xl font-semibold tracking-[0.18em]">WRAITH PLAYGROUND</div>
            <p className="mt-1 text-xs text-zinc-600">Offline Sigma linting and Elasticsearch translation — no production writes.</p>
          </div>
          <Link href="/" className="text-xs text-zinc-500 hover:text-zinc-200">← Control plane</Link>
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-5 px-6 py-6 lg:grid-cols-2">
        <form onSubmit={validate} className="rounded-lg border border-zinc-800 bg-[#0f141b] p-5">
          <div className="mb-3 flex items-center justify-between">
            <h1 className="text-sm font-semibold">Sigma rule</h1>
            <span className="rounded border border-emerald-900/60 px-2 py-1 text-[10px] uppercase tracking-widest text-emerald-400">isolated</span>
          </div>
          <textarea value={rule} onChange={(e) => setRule(e.target.value)} spellCheck={false}
            className="min-h-[560px] w-full rounded-md border border-zinc-800 bg-[#090c11] p-4 font-mono text-xs leading-5 text-zinc-300 outline-none focus:border-zinc-600" />
          <button disabled={busy || rule.length === 0 || rule.length > 256 * 1024} className="mt-4 rounded-md border border-zinc-700 bg-zinc-100 px-4 py-2 text-xs font-semibold text-zinc-900 hover:bg-white disabled:cursor-not-allowed disabled:opacity-40">{busy ? "Validating…" : "Validate & translate"}</button>
          <div className="mt-2 flex justify-between text-[10px] text-zinc-600"><span>Maximum 256 KiB</span><span>{rule.length.toLocaleString()} bytes</span></div>
        </form>

        <section className="rounded-lg border border-zinc-800 bg-[#0f141b] p-5">
          <h2 className="text-sm font-semibold">Evidence</h2>
          {error && <div className="mt-4 rounded border border-rose-900 bg-rose-950/30 p-3 text-xs text-rose-300">{error}</div>}
          {!result && !error && <p className="mt-4 text-xs text-zinc-600">Submit a rule to see lint findings and the generated Elasticsearch Query DSL.</p>}
          {result && (
            <div className="mt-4 space-y-4">
              <div className="rounded border border-zinc-800 bg-[#090c11] p-4">
                <div className="text-[10px] uppercase tracking-widest text-zinc-600">Execution</div>
                <div className="mt-1 text-sm text-emerald-400">{result.execution}</div>
              </div>
              <div className="rounded border border-zinc-800 bg-[#090c11] p-4">
                <div className="mb-2 text-[10px] uppercase tracking-widest text-zinc-600">Lint</div>
                <pre className="max-h-56 overflow-auto whitespace-pre-wrap text-xs text-zinc-400">{JSON.stringify(result.lint, null, 2)}</pre>
              </div>
              {result.query_dsl && (
                <div className="rounded border border-zinc-800 bg-[#090c11] p-4">
                  <div className="mb-2 text-[10px] uppercase tracking-widest text-zinc-600">Elasticsearch Query DSL</div>
                  <pre className="max-h-[520px] overflow-auto text-xs leading-5 text-zinc-400">{JSON.stringify(result.query_dsl, null, 2)}</pre>
                </div>
              )}
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
