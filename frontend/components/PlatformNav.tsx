import Link from "next/link";

const groups = [
  { label: "Introduction", items: [["Overview", "/"], ["Architecture", "/introduction"]] },
  { label: "Operations", items: [["Validation runs", "/#runs"], ["Capabilities", "/#capabilities"]] },
  { label: "Detection Engineering", items: [["Rules", "#planned-rules"], ["Sigma translation", "#planned-translation"], ["Mutation testing", "#planned-mutations"], ["Robustness", "#planned-robustness"], ["Quality", "#planned-quality"]] },
  { label: "Threat Intelligence", items: [["MITRE ATT&CK", "#planned-attack"], ["Attack chains", "#planned-attack"]] },
  { label: "Validation", items: [["Attack simulation", "#capabilities"], ["Detection validation", "#capabilities"], ["Baseline testing", "#capabilities"], ["Adversarial testing", "#capabilities"]] },
  { label: "Integrations", items: [["SIEM backends", "#planned-integrations"], ["GitHub", "#planned-integrations"]] },
  { label: "Automation", items: [["SOAR", "#planned-automation"], ["CI/CD", "#planned-automation"]] },
  { label: "Governance", items: [["Audit", "#planned-governance"], ["Provenance", "#planned-governance"], ["Approval & deployment", "#planned-governance"]] },
  { label: "System", items: [["Services", "#system"], ["Playground", "/playground"]] },
];

export default function PlatformNav() {
  return (
    <nav className="sticky top-6 space-y-5 text-sm">
      {groups.map((group) => (
        <div key={group.label}>
          <div className="mb-1 px-3 text-[10px] uppercase tracking-[0.18em] text-zinc-700">{group.label}</div>
          <div className="space-y-0.5">
            {group.items.map(([label, href], index) => (
              <Link key={label} href={href} className={`block rounded-md px-3 py-1.5 text-xs transition ${index === 0 && group.label === "Introduction" ? "bg-zinc-900/70 text-zinc-100" : "text-zinc-500 hover:bg-zinc-900 hover:text-zinc-300"}`}>
                {label}
              </Link>
            ))}
          </div>
        </div>
      ))}
    </nav>
  );
}
