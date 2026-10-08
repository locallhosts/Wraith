import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import StatusBadge from "../components/security/StatusBadge";
import { fetchDeployments, fetchHealth, fetchReady } from "../lib/api";

export default function Operations() {
 const { data: health } = useSWR("/healthz", fetchHealth, { refreshInterval: 10000 });
 const { data: ready } = useSWR("/readyz", fetchReady, { refreshInterval: 10000 });
 const { data: deployments } = useSWR("/deployments", fetchDeployments, { refreshInterval: 10000 });
 return <WraithShell eyebrow="Operations & Governance">
   <section className="page-hero"><div><div className="eyebrow">Operations / governance</div><h1>Deployment readiness</h1><p>Inspect service readiness and deployment evidence before promoting validated detections.</p></div></section>
   <div className="metric-grid"><div className="metric-card metric-green"><div className="metric-label">Health</div><div className="metric-value"><StatusBadge status={health?.status || "Checking"} /></div><div className="metric-detail">/healthz</div></div><div className="metric-card metric-cyan"><div className="metric-label">Readiness</div><div className="metric-value"><StatusBadge status={ready?.status || "Checking"} /></div><div className="metric-detail">/readyz</div></div><div className="metric-card metric-amber"><div className="metric-label">Deployments</div><div className="metric-value">{deployments?.length ?? 0}</div><div className="metric-detail">Recorded promotion events</div></div></div>
   <section className="panel"><div className="panel-heading"><div><h2>Deployment history</h2><p>Verification and rollback controls remain behind the existing API authorization boundary.</p></div></div><div className="run-table"><div className="run-head"><span>Deployment</span><span>Run</span><span>Status</span><span>Time</span></div>{(deployments ?? []).map((d:any)=><div className="run-row" key={d.id || d.run_id}><span><strong>{d.id || "deployment"}</strong></span><span>{d.run_id || "—"}</span><StatusBadge status={d.status || "Recorded"} /><span>{d.created_at ? new Date(d.created_at).toLocaleString() : "—"}</span></div>)}{!deployments?.length && <div className="empty-state">No deployment records returned by the control plane.</div>}</div></section>
 </WraithShell>
}
