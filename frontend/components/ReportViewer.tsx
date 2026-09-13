import useSWR from "swr";
import { fetcher, RunReport, Attestation } from "../lib/api";

function ScoreBar({ value, label }: { value: number; label: string }) {
  const pct = Math.round(value * 100);
  const color = pct >= 90 ? "bg-emerald-600" : pct >= 70 ? "bg-amber-600" : "bg-rose-600";
  return (
    <div>
      <div className="mb-1 flex justify-between text-xs">
        <span className="text-zinc-500">{label}</span>
        <span className="font-mono text-zinc-300">{pct}%</span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded bg-zinc-800">
        <div className={`h-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

function KillChain({ chain }: { chain: string[] }) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {chain.map((technique, i) => (
        <div key={technique + i} className="flex items-center gap-1.5">
          <span className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 font-mono text-xs text-zinc-300">
            {technique}
          </span>
          {i < chain.length - 1 && <span className="text-zinc-700">→</span>}
        </div>
      ))}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded border border-zinc-800 p-5">
      <h3 className="mb-3 text-sm font-medium text-zinc-300">{title}</h3>
      {children}
    </div>
  );
}

export default function ReportViewer({ runId }: { runId: string }) {
  const { data: report, error: reportError } = useSWR<RunReport>(
    `/runs/${runId}/report`,
    fetcher,
    { refreshInterval: 5000 }
  );
  const { data: attestation } = useSWR<Attestation>(
    report?.passed ? `/runs/${runId}/attestation` : null,
    fetcher
  );

  if (reportError) {
    return (
      <div className="rounded border border-dashed border-zinc-800 p-5 text-sm text-zinc-500">
        No report yet — it appears once the pipeline finishes running (translate → baseline →
        attack simulation → validate → robustness fuzz → SOAR draft).
      </div>
    );
  }
  if (!report) {
    return <div className="rounded border border-zinc-800 p-5 text-sm text-zinc-500">Loading report…</div>;
  }

  const { validate, attack_simulation, robustness, soar_playbook, baseline } = report.stages;

  return (
    <div className="space-y-4">
      {attack_simulation?.simulated_chain && (
        <Section title="Simulated attack chain (MITRE ATT&CK)">
          <KillChain chain={attack_simulation.simulated_chain} />
          <p className="mt-3 text-xs text-zinc-600">
            {attack_simulation.simulated_user} on {attack_simulation.simulated_host} ·{" "}
            {attack_simulation.events_indexed} events injected
          </p>
        </Section>
      )}

      {validate && (
        <Section title="Detection validation">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-zinc-500">Fired on attack telemetry</p>
              <p className={validate.fired_on_attack ? "text-emerald-400" : "text-rose-400"}>
                {validate.fired_on_attack ? `Yes (${validate.attack_hit_count} hits)` : "No"}
              </p>
            </div>
            <div>
              <p className="text-zinc-500">Fired on baseline noise</p>
              <p className={validate.fired_on_baseline ? "text-rose-400" : "text-emerald-400"}>
                {validate.fired_on_baseline ? `Yes (${validate.baseline_hit_count} hits — false positive)` : "No"}
              </p>
            </div>
          </div>
          {baseline?.events_indexed && (
            <p className="mt-3 text-xs text-zinc-600">
              Tested against {baseline.events_indexed.toLocaleString()} synthetic baseline events
              ({validate.baseline_docs_scanned?.toLocaleString()} scanned by the query)
            </p>
          )}
          <div className="mt-4">
            <ScoreBar value={1 - (validate.false_positive_rate ?? 0)} label="Baseline precision (1 − false-positive rate)" />
          </div>
        </Section>
      )}

      {robustness && robustness.score !== undefined && (
        <Section title="Adversarial evasion robustness">
          <ScoreBar value={robustness.score} label="Evasion variants still detected" />
          <p className="mt-2 text-xs text-zinc-600">
            {robustness.variants_detected} / {robustness.variants_tested} mutated variants
            (case swaps, PowerShell parameter aliasing, env-var path substitution, whitespace
            variation) still caught by this rule&apos;s actual compiled query.
          </p>
          {robustness.undetected_examples && robustness.undetected_examples.length > 0 && (
            <div className="mt-4">
              <p className="mb-2 text-xs font-medium text-zinc-400">Variants that slipped through:</p>
              <ul className="space-y-2">
                {robustness.undetected_examples.map((ex, i) => (
                  <li key={i} className="rounded bg-zinc-900 p-2 font-mono text-xs text-zinc-400">
                    <span className="text-amber-500">[{ex.mutation_chain.join(", ")}]</span>{" "}
                    {ex.event.command_line}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </Section>
      )}
      {robustness?.status === "skipped" && (
        <Section title="Adversarial evasion robustness">
          <p className="text-sm text-zinc-500">Skipped — {robustness.reason}</p>
        </Section>
      )}

      {soar_playbook && (
        <Section title="Drafted SOAR playbook">
          {soar_playbook.status === "ok" ? (
            <div className="space-y-1 text-sm">
              <p className="font-mono text-zinc-300">{soar_playbook.path}</p>
              {soar_playbook.pr_url ? (
                <a href={soar_playbook.pr_url} target="_blank" rel="noreferrer" className="text-sky-400 hover:underline">
                  Review PR →
                </a>
              ) : (
                <p className="text-xs text-zinc-600">No PR opened (GITHUB_TOKEN/GITHUB_REPO not configured for this run).</p>
              )}
            </div>
          ) : (
            <p className="text-sm text-rose-400">{soar_playbook.error}</p>
          )}
        </Section>
      )}

      {attestation && (
        <Section title="Signed provenance attestation">
          <div className="grid grid-cols-2 gap-y-2 text-xs">
            <span className="text-zinc-500">Issuer</span>
            <span className="text-zinc-300">{attestation.attestation.issuer}</span>
            <span className="text-zinc-500">Issued at</span>
            <span className="text-zinc-300">{new Date(attestation.attestation.issued_at).toLocaleString()}</span>
            <span className="text-zinc-500">Rule content hash</span>
            <span className="truncate font-mono text-zinc-400" title={attestation.attestation.rule_content_sha256}>
              {attestation.attestation.rule_content_sha256}
            </span>
            <span className="text-zinc-500">Signature</span>
            <span className="truncate font-mono text-zinc-400" title={attestation.signature}>
              {attestation.signature.slice(0, 24)}…
            </span>
          </div>
          <p className="mt-3 text-xs text-zinc-600">
            This is a display copy only — the deploy gate independently re-verifies this signature
            against the pinned trusted key and re-hashes the rule&apos;s current on-disk content
            before promoting anything.
          </p>
        </Section>
      )}
    </div>
  );
}
