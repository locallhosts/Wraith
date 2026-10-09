import { useMemo, useState } from "react";
import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import { fetcher } from "../lib/api";

interface AuditRecord {
  id: number | string;
  actor?: string;
  actor_role?: string;
  action?: string;
  resource?: string;
  detail?: string;
  ip_address?: string;
  timestamp?: string;
}

function formatTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export default function AuditPage() {
  const { data, error, isLoading } = useSWR<AuditRecord[]>("/audit", fetcher, { refreshInterval: 10000 });
  const [query, setQuery] = useState("");
  const [action, setAction] = useState("all");
  const [role, setRole] = useState("all");

  const actions = useMemo(() => Array.from(new Set((data ?? []).map(item => item.action).filter((value): value is string => Boolean(value)))).sort(), [data]);
  const roles = useMemo(() => Array.from(new Set((data ?? []).map(item => item.actor_role).filter((value): value is string => Boolean(value)))).sort(), [data]);
  const entries = useMemo(() => {
    const q = query.trim().toLowerCase();
    return [...(data ?? [])].filter(item => {
      const searchable = [item.id, item.actor, item.actor_role, item.action, item.resource, item.detail, item.ip_address, item.timestamp];
      return (!q || searchable.some(value => String(value ?? "").toLowerCase().includes(q)))
        && (action === "all" || item.action === action)
        && (role === "all" || item.actor_role === role);
    }).sort((a, b) => new Date(b.timestamp ?? 0).getTime() - new Date(a.timestamp ?? 0).getTime());
  }, [data, query, action, role]);

  const actors = new Set((data ?? []).map(item => item.actor).filter(Boolean)).size;
  const actionCount = new Set((data ?? []).map(item => item.action).filter(Boolean)).size;

  return <WraithShell eyebrow="Governance Audit">
    <section className="page-hero"><div><div className="eyebrow">Governance / accountability</div><h1>Audit trail</h1><p>Review persisted control-plane actions, operator identity, role, resource, and source IP. The audit endpoint is restricted to administrators.</p></div><span className="service-pill"><i /> Refresh · 10s</span></section>

    {error && <div role="alert" className="audit-access-error"><strong>Audit data unavailable</strong><p>This endpoint requires an authenticated administrator API key. Check the configured key and its role; the page does not bypass server-side authorization.</p><code>{error.message}</code></div>}

    <section className="metric-grid">
      <div className="metric-card"><div className="metric-label">Recorded events</div><div className="metric-value">{data?.length ?? "—"}</div><div className="metric-detail">Most recent records returned by API</div></div>
      <div className="metric-card metric-cyan"><div className="metric-label">Distinct actors</div><div className="metric-value">{data ? actors : "—"}</div><div className="metric-detail">Actor labels in current result set</div></div>
      <div className="metric-card metric-green"><div className="metric-label">Action types</div><div className="metric-value">{data ? actionCount : "—"}</div><div className="metric-detail">Unique persisted action names</div></div>
      <div className="metric-card"><div className="metric-label">Matching events</div><div className="metric-value">{data ? entries.length : "—"}</div><div className="metric-detail">After local filters</div></div>
    </section>

    <section className="panel">
      <div className="panel-heading"><div><h2>Operator activity</h2><p>Search actor, resource, action, detail, role, or IP address.</p></div><span className="count-badge">{entries.length} shown</span></div>
      <div className="runs-toolbar">
        <label className="runs-search"><span aria-hidden="true">⌕</span><input aria-label="Search audit trail" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search audit events…" /><kbd>LIVE</kbd></label>
        <label className="runs-sort">Action <select aria-label="Filter by audit action" value={action} onChange={e => setAction(e.target.value)}><option value="all">All actions</option>{actions.map(value => <option key={value} value={value}>{value}</option>)}</select></label>
        <label className="runs-sort">Role <select aria-label="Filter by actor role" value={role} onChange={e => setRole(e.target.value)}><option value="all">All roles</option>{roles.map(value => <option key={value} value={value}>{value}</option>)}</select></label>
        <button type="button" className="audit-reset" onClick={() => { setQuery(""); setAction("all"); setRole("all"); }}>Reset</button>
      </div>
      {isLoading && <div className="empty-state">Loading protected audit records…</div>}
      {!isLoading && !error && <div className="audit-table-wrap"><table className="audit-table"><thead><tr><th>Timestamp</th><th>Actor / role</th><th>Action / resource</th><th>Details</th><th>Source IP</th></tr></thead><tbody>
        {entries.map((item, index) => <tr key={item.id ?? `${item.timestamp}-${index}`}>
          <td className="audit-time">{formatTime(item.timestamp)}</td>
          <td><strong>{item.actor || "system"}</strong><small>{item.actor_role || "role unavailable"}</small></td>
          <td><strong className="audit-action">{item.action || "unknown action"}</strong><small className="audit-resource">{item.resource || "—"}</small></td>
          <td className="audit-detail">{item.detail || "No detail supplied"}</td>
          <td className="audit-ip">{item.ip_address || "—"}</td>
        </tr>)}
      </tbody></table>{entries.length === 0 && <div className="empty-state">{data?.length ? "No events match these filters." : "No audit events were returned."}</div>}</div>}
    </section>
  </WraithShell>;
}
