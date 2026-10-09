import { FormEvent, useMemo, useState } from "react";
import WraithShell from "../components/WraithShell";
import Pipeline from "../components/security/Pipeline";
import { validatePlayground } from "../lib/api";

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

type ResultTab = "overview" | "lint" | "query" | "raw";

function pretty(value: unknown) {
  return JSON.stringify(value ?? null, null, 2);
}

function ResultTabButton({ tab, active, onClick, children }: {
  tab: ResultTab; active: ResultTab; onClick: (tab: ResultTab) => void; children: string;
}) {
  return <button type="button" onClick={() => onClick(tab)} className={`rounded-md px-3 py-2 text-[10px] transition ${active === tab ? "border border-cyan-800 bg-cyan-950/40 text-cyan-200" : "border border-transparent text-zinc-500 hover:text-zinc-200"}`}>{children}</button>;
}

export default function Playground() {
  const [rule, setRule] = useState(example);
  const [result, setResult] = useState<Record<string, any> | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<ResultTab>("overview");
  const [copied, setCopied] = useState(false);

  const lintItems = useMemo(() => {
    const lint = result?.lint;
    if (Array.isArray(lint)) return lint;
    if (lint && Array.isArray(lint.issues)) return lint.issues;
    if (lint && Array.isArray(lint.findings)) return lint.findings;
    return [];
  }, [result]);
  const verdict = result && (result.passed === true || result.valid === true || result.lint?.passed === true)
    ? "passed"
    : result && (result.passed === false || result.valid === false || result.lint?.passed === false)
      ? "failed" : result ? "completed" : "pending";
  const queryDSL = result?.query_dsl ?? result?.translation?.query_dsl ?? result?.translate?.query_dsl;

  async function validate(e: FormEvent) {
    e.preventDefault();
    setError(""); setResult(null); setTab("overview"); setCopied(false); setBusy(true);
    try {
      const response = await validatePlayground(rule);
      setResult(response && typeof response === "object" ? response : { response });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Playground validation failed");
    } finally { setBusy(false); }
  }

  async function copyOutput() {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(pretty(result));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setError("Clipboard access was blocked by the browser. Use the Raw JSON tab to select and copy the output.");
    }
  }

  function downloadOutput() {
    if (!result) return;
    const blob = new Blob([pretty(result)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "wraith-playground-result.json";
    document.body.appendChild(anchor); anchor.click(); anchor.remove(); URL.revokeObjectURL(url);
  }

  return <WraithShell eyebrow="Detection Playground">
    <section className="page-hero">
      <div><div className="eyebrow">Isolated detection laboratory</div><h1>Playground</h1><p>Inspect Sigma validation output before committing a rule. Public playground mode is isolated from production writes; analysis results are not persisted as production runs.</p></div>
      <button type="button" className="primary-button" onClick={() => { setRule(example); setResult(null); setError(""); setTab("overview"); }}>Load example rule ↺</button>
    </section>

    <div className="dashboard-grid">
      <form onSubmit={validate} className="panel" style={{ marginBottom: 0 }}>
        <div className="panel-heading"><div><h2>Rule editor</h2><p>Maximum 256 KiB · YAML / Sigma rule</p></div><span className="status-badge success"><i />isolated</span></div>
        <div style={{ padding: 18 }}>
          <textarea aria-label="Sigma rule editor" value={rule} onChange={e => setRule(e.target.value)} spellCheck={false} className="playground-editor" />
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12, marginTop: 10 }}>
            <span style={{ fontSize: 9, color: rule.length > 256 * 1024 ? "var(--rose)" : "#536173" }}>{rule.length.toLocaleString()} / 262,144 characters</span>
            <button disabled={busy || !rule.trim() || rule.length > 256 * 1024} className="primary-button" type="submit">{busy ? "Analyzing…" : "Run analysis →"}</button>
          </div>
          {rule.length > 256 * 1024 && <p className="mt-2 text-xs text-rose-400">The rule exceeds the playground size limit.</p>}
        </div>
      </form>

      <section className="panel" style={{ marginBottom: 0 }}>
        <div className="panel-heading"><div><h2>Analysis results</h2><p>Rule → lint → translation → inspect evidence</p></div>{result && <span className={`status-badge ${verdict === "passed" ? "success" : verdict === "failed" ? "danger" : "warning"}`}><i />{verdict}</span>}</div>
        <div style={{ padding: 18 }}><Pipeline active={busy ? 2 : result ? 5 : 0} /></div>
        {error && <div role="alert" className="error-box">{error}</div>}
        {busy && <div className="empty-state">Running isolated validation…</div>}
        {!result && !error && !busy && <div className="empty-state">Submit a rule to inspect validation, lint findings, translation output, and the full response.</div>}
        {result && <div className="evidence-stack">
          <div className="playground-result-toolbar">
            <div className="playground-result-tabs" role="tablist" aria-label="Analysis result views">
              <ResultTabButton tab="overview" active={tab} onClick={setTab}>Overview</ResultTabButton>
              <ResultTabButton tab="lint" active={tab} onClick={setTab}>Lint {lintItems.length ? `(${lintItems.length})` : ""}</ResultTabButton>
              <ResultTabButton tab="query" active={tab} onClick={setTab}>Query DSL</ResultTabButton>
              <ResultTabButton tab="raw" active={tab} onClick={setTab}>Raw JSON</ResultTabButton>
            </div>
            <div className="playground-result-actions"><button type="button" onClick={copyOutput}>{copied ? "Copied ✓" : "Copy JSON"}</button><button type="button" onClick={downloadOutput}>Export ↓</button></div>
          </div>

          {tab === "overview" && <div className="playground-result-grid">
            <div className="evidence-card"><span>Execution</span><strong>{String(result.execution || result.status || "completed")}</strong><p className="playground-result-note">A successful request means the analyzer returned a response; inspect the verdict and findings below for the rule outcome.</p></div>
            <div className="evidence-card"><span>Rule verdict</span><strong className={verdict === "failed" ? "text-rose-300" : verdict === "passed" ? "text-emerald-300" : ""}>{verdict === "passed" ? "Passed" : verdict === "failed" ? "Failed" : "Returned"}</strong><p className="playground-result-note">{result.reason || result.message || "The response is shown as returned by the backend."}</p></div>
            <div className="evidence-card"><span>Lint findings</span><strong>{lintItems.length}</strong><p className="playground-result-note">{lintItems.length ? "Review each finding in the Lint tab." : "No findings were exposed in the response's recognized lint fields."}</p></div>
            <div className="evidence-card"><span>Translation</span><strong>{queryDSL ? "Available" : "Not returned"}</strong><p className="playground-result-note">{queryDSL ? "Inspect generated query output in the Query DSL tab." : "This response did not include a recognized query_dsl field."}</p></div>
          </div>}

          {tab === "lint" && <div className="evidence-card"><span>Lint output</span>{lintItems.length ? <div className="playground-findings">{lintItems.map((item: any, index: number) => <div className="playground-finding" key={item.id || item.code || index}><div><strong>{item.severity || item.level || item.code || `Finding ${index + 1}`}</strong><small>{item.field || item.path || item.line ? [item.field || item.path, item.line ? `line ${item.line}` : ""].filter(Boolean).join(" · ") : "Location not supplied"}</small></div><p>{item.message || item.description || pretty(item)}</p></div>)}</div> : <><p className="playground-result-note">No structured findings array was present in the response.</p><pre>{pretty(result.lint ?? { note: "The backend did not return a lint field." })}</pre></>}</div>}

          {tab === "query" && <div className="evidence-card"><span>Generated query</span>{queryDSL ? <pre>{pretty(queryDSL)}</pre> : <p className="playground-result-note">No query DSL was returned by this backend response. This is not treated as a successful translation; inspect Raw JSON for the exact payload.</p>}</div>}

          {tab === "raw" && <div className="evidence-card"><span>Raw API response · read-only</span><pre className="playground-raw-json">{pretty(result)}</pre></div>}
        </div>}
      </section>
    </div>
  </WraithShell>;
}
