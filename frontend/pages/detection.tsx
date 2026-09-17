import Link from "next/link";
import useSWR from "swr";
import { useMemo, useState } from "react";
import { fetcher } from "../lib/api";
import PlatformNav from "../components/PlatformNav";
import ThemeToggle from "../components/ThemeToggle";
import ApiKeyBar from "../components/ApiKeyBar";

type Rule = { name: string; path: string; title?: string; id?: string; status?: string; level?: string; passed: boolean; issue_count: number; error_count: number; warning_count: number };
type RulesResponse = { count: number; rules: Rule[] };

export default function DetectionEngineering() {
  const { data, error, isLoading } = useSWR<RulesResponse>("/rules", fetcher, { refreshInterval: 10000 });
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data?.rules ?? []).filter((r) => !q || [r.name, r.path, r.title, r.id, r.level].some(v => v?.toLowerCase().includes(q)));
  }, [data, query]);

  return <main className="min-h-screen bg-[#0a0e13] text-zinc-200">
    <header className="border-b border-zinc-800 bg-[#0d1117]/95 px-6 py-5 lg:px-10"><div className="mx-auto flex max-w-7xl items-center justify-between gap-4"><div><Link href="/" className="text-xl font-semibold tracking-[0.18em] text-zinc-100">WRAITH</Link><p className="mt-1 text-xs text-zinc-600">Detection Engineering</p></div><div className="flex items-center gap-2"><ThemeToggle /><ApiKeyBar /></div></div></header>
    <div className="mx-auto grid max-w-7xl gap-6 px-6 py-6 lg:grid-cols-[210px_1fr] lg:px-10"><aside className="hidden lg:block"><PlatformNav /></aside><section className="min-w-0">
      <div className="rounded-xl border border-zinc-800 bg-[#0f141b] p-6"><p className="text-[10px] uppercase tracking-[0.2em] text-emerald-400">Detection workspace</p><h1 className="mt-2 text-3xl font-semibold tracking-tight text-zinc-100">Rules</h1><p className="mt-3 max-w-3xl text-sm leading-6 text-zinc-500">Browse the real Sigma content available to the Wraith pipeline, inspect lint state, and open a rule before running validation.</p></div>
      <div className="mt-6 rounded-lg border border-zinc-800 bg-[#0f141b]"><div className="flex flex-col gap-3 border-b border-zinc-800 p-4 md:flex-row md:items-center md:justify-between"><div><h2 className="text-sm font-semibold">Repository rules</h2><p className="mt-1 text-xs text-zinc-600">Loaded from the backend rule directory.</p></div><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search rules…" className="rounded-md border border-zinc-800 bg-[#0a0e13] px-3 py-2 text-xs text-zinc-300 outline-none placeholder:text-zinc-700" /></div>
      {isLoading && <div className="p-8 text-sm text-zinc-600">Loading rules…</div>}{error && <div className="m-4 rounded-md border border-rose-900 bg-rose-950/20 p-4 text-sm text-rose-300">Couldn&apos;t load rules. Check API authentication and the backend.</div>}
      <div className="divide-y divide-zinc-800/80">{filtered.map(rule => <Link key={rule.name} href={`/rules/${encodeURIComponent(rule.name)}`} className="block p-4 hover:bg-zinc-900/40"><div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between"><div><div className="flex flex-wrap items-center gap-2"><span className="font-mono text-sm text-zinc-200">{rule.title || rule.name}</span><span className={`rounded border px-2 py-0.5 text-[10px] ${rule.passed ? "border-emerald-900 text-emerald-400" : "border-rose-900 text-rose-400"}`}>{rule.passed ? "Lint pass" : "Lint issues"}</span></div><p className="mt-2 text-[11px] text-zinc-600">{rule.name} · {rule.id || "no id"} · {rule.level || "no level"}</p></div><div className="text-right text-[11px] text-zinc-600">{rule.issue_count} issues<br />Open →</div></div></Link>)}</div>
      {data && filtered.length === 0 && <div className="p-10 text-center text-sm text-zinc-600">No rules match the search.</div>}
      </div>
    </section></div>
  </main>;
}
