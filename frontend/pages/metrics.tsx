import {useEffect,useMemo,useState} from "react";
import PlatformNav from "../components/PlatformNav";
import {fetchMetrics} from "../lib/api";

type Sample={name:string,value:string,labels:string};
function parseMetrics(raw:string):Sample[]{return raw.split("\n").filter(line=>line&&!line.startsWith("#")).map(line=>{const m=line.match(/^([^\s{]+)(?:\{([^}]*)\})?\s+(.+)$/);return m?{name:m[1],labels:m[2]||"",value:m[3]}:null}).filter(Boolean) as Sample[]}
const groups=["wraith_pipeline_runs_total","wraith_pipeline_run_duration_seconds","wraith_rule_false_positive_rate","wraith_rule_robustness_score","wraith_approvals_total","wraith_deploys_total","wraith_webhook_rejections_total"];
export default function MetricsPage(){
 const [raw,setRaw]=useState("");const [error,setError]=useState("");const [showRaw,setShowRaw]=useState(false);
 useEffect(()=>{const load=()=>fetchMetrics().then(setRaw).catch((e: unknown)=>setError(e instanceof Error ? e.message : "Unable to load metrics"));load();const t=setInterval(load,5000);return()=>clearInterval(t)},[]);
 const samples=useMemo(()=>parseMetrics(raw),[raw]);
 const by=useMemo(()=>Object.fromEntries(groups.map(g=>[g,samples.filter(s=>s.name===g)])),[samples]);
 return <div className="min-h-screen bg-[#070707] text-zinc-200"><div className="mx-auto grid max-w-[1500px] grid-cols-[220px_1fr] gap-10 px-8 py-8"><PlatformNav/><main>
 <div className="mb-8"><div className="text-[10px] uppercase tracking-[0.22em] text-zinc-600">System / Observability</div><h1 className="mt-2 text-3xl font-semibold tracking-tight">Prometheus telemetry</h1><p className="mt-2 max-w-2xl text-sm leading-6 text-zinc-500">Operational telemetry emitted by the Wraith backend. Values refresh automatically from the live <span className="font-mono text-zinc-400">/metrics</span> endpoint.</p></div>
 {error&&<div className="mb-5 rounded-md border border-red-950 bg-red-950/20 px-4 py-3 text-xs text-red-300">{error}</div>}
 <div className="grid gap-4 md:grid-cols-2">{groups.map(name=><section key={name} className="rounded-xl border border-zinc-900 bg-zinc-950 p-5"><div className="font-mono text-xs text-zinc-300">{name}</div><div className="mt-4 space-y-2">{(by[name]||[]).length===0?<div className="text-xs text-zinc-700">No samples emitted yet.</div>:(by[name]||[]).map((s,i)=><div key={i} className="flex items-center justify-between gap-4 rounded-md bg-zinc-900/40 px-3 py-2"><span className="truncate text-[11px] text-zinc-600">{s.labels||"aggregate"}</span><span className="font-mono text-sm text-zinc-300">{s.value}</span></div>)}</div></section>)}</div>
 <div className="mt-6 rounded-xl border border-zinc-900 bg-zinc-950 p-5"><button onClick={()=>setShowRaw(!showRaw)} className="text-xs text-zinc-400 hover:text-white">{showRaw?"Hide":"Show"} raw exposition</button>{showRaw&&<pre className="mt-4 max-h-[500px] overflow-auto whitespace-pre-wrap font-mono text-[10px] leading-5 text-zinc-600">{raw||"No telemetry returned."}</pre>}</div>
 </main></div></div>
}