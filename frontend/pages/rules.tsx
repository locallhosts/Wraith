import { useMemo, useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import WraithShell from "../components/WraithShell";
import MetricCard from "../components/security/MetricCard";
import StatusBadge from "../components/security/StatusBadge";
import { fetcher } from "../lib/api";

interface RuleSummary {
  name: string;
  path?: string;
  title?: string;
  id?: string;
  status?: string;
  level?: string;
  passed?: boolean;
  issue_count?: number;
  error_count?: number;
  warning_count?: number;
}
interface RulesResponse { count?: number; rules?: RuleSummary[] }
type RuleFilter = "all" | "passing" | "findings";

export default function Rules() {
  const { data, error, isLoading, mutate } = useSWR<RulesResponse>("/rules", fetcher, { refreshInterval: 30000 });
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<RuleFilter>("all");
  const rules = data?.rules ?? [];
  const passing = rules.filter((rule) => rule.passed === true).length;
  const findings = rules.filter((rule) => rule.passed === false).length;
  const visibleRules = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return rules.filter((rule) => {
      const matchesFilter = filter === "all" || (filter === "passing" ? rule.passed === true : rule.passed === false);
      const haystack = [rule.name, rule.title, rule.id, rule.level, rule.status].filter(Boolean).join(" ").toLowerCase();
      return matchesFilter && (!normalized || haystack.includes(normalized));
    }).sort((a, b) => {
      if (a.passed !== b.passed) return a.passed ? 1 : -1;
      return (a.name || "").localeCompare(b.name || "");
    });
  }, [rules, query, filter]);

  return (
    <WraithShell eyebrow="Detection Engineering">
      <section className="page-hero">
        <div><div className="eyebrow">Detection engineering / rule quality</div><h1>Rules workspace</h1><p>Inspect the current Sigma rule inventory, lint quality, severity metadata, and findings returned by the Wraith control plane.</p></div>
        <button type="button" className="primary-button" onClick={() => void mutate()}>Refresh inventory ↻</button>
      </section>

      {error && <div className="error-box" role="alert">Unable to load the rule inventory: {error.message}. Confirm your API key and viewer access.</div>}

      <div className="metric-grid">
        <MetricCard label="Rules discovered" value={data?.count ?? rules.length} detail={isLoading ? "Loading inventory…" : "Returned by the control plane"} />
        <MetricCard label="Passing lint" value={passing} detail="No blocking rule findings" tone="green" />
        <MetricCard label="Needs review" value={findings} detail="Lint checks reported findings" tone={findings ? "amber" : "green"} />
        <MetricCard label="Visible results" value={visibleRules.length} detail="After search and filters" />
      </div>

      <section className="panel">
        <div className="panel-heading rule-toolbar">
          <div><h2>Rule inventory</h2><p>Search by title, rule ID, filename, severity, or status.</p></div>
          <label className="rule-search"><span className="sr-only">Search rules</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search rule inventory…" /></label>
        </div>
        <div className="rule-filters" role="group" aria-label="Filter rules">
          {([{ value: "all", label: "All rules" }, { value: "passing", label: "Passing" }, { value: "findings", label: "Needs review" }] as const).map((item) => (
            <button type="button" key={item.value} className={filter === item.value ? "rule-filter active" : "rule-filter"} onClick={() => setFilter(item.value)} aria-pressed={filter === item.value}>{item.label}</button>
          ))}
          <span className="rule-result-count">{visibleRules.length} result{visibleRules.length === 1 ? "" : "s"}</span>
        </div>
        {isLoading && <div className="empty-state">Loading rules from the control plane…</div>}
        {!isLoading && !error && visibleRules.length === 0 && <div className="empty-state">{rules.length ? "No rules match the current search and filter." : "No rule records were returned by the API."}</div>}
        <div className="rule-inventory">
          {visibleRules.map((rule) => (
            <Link key={rule.name} href={`/rules/${encodeURIComponent(rule.name)}`} className="rule-inventory-row">
              <span className="rule-status-glyph" aria-hidden="true">{rule.passed ? "✓" : "!"}</span>
              <span className="rule-main"><strong>{rule.title || rule.name || "Untitled rule"}</strong><small>{rule.name}{rule.id ? ` · ${rule.id}` : ""}</small><span className="rule-meta">{rule.level || "Unclassified"} severity <span>·</span> {rule.status || "status not set"}</span></span>
              <span className="rule-findings"><strong>{rule.issue_count ?? 0}</strong><small>findings</small></span>
              <StatusBadge status={rule.passed ? "Passing" : "Review"} />
              <span className="rule-chevron" aria-hidden="true">→</span>
            </Link>
          ))}
        </div>
      </section>
      <p className="analytics-footnote">Rule metadata and lint findings are read from the existing authenticated API. This workspace does not modify rule files.</p>
    </WraithShell>
  );
}
