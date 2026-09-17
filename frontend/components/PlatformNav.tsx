import Link from "next/link";

const groups = [
  { label: "Introduction", items: [["Overview", "/"], ["Architecture", "/introduction"]] },
  { label: "Operations", items: [["Validation runs", "/#runs"], ["Capabilities", "/#capabilities"]] },
  { label: "Detection Engineering", items: [["Rules", "/detection"], ["Validation", "/validation"]] },
  { label: "Threat Intelligence", items: [["ATT&CK", "/validation#attack"], ["Attack chains", "/validation#attack"]] },
  { label: "Integrations", items: [["SIEM", "/integrations"], ["GitHub", "/integrations#github"]] },
  { label: "Automation", items: [["SOAR", "/automation"], ["CI/CD", "/automation#cicd"]] },
  { label: "Governance", items: [["Audit", "/governance"], ["Provenance", "/governance#provenance"], ["Approval", "/governance#approval"]] },
];

export default function PlatformNav() {
  return <nav className="sticky top-6 space-y-5 text-sm">
    {groups.map(group => <div key={group.label}>
      <div className="mb-1 px-3 text-[10px] uppercase tracking-[0.18em] text-zinc-700">{group.label}</div>
      <div className="space-y-0.5">{group.items.map(([label, href]) => <Link key={label} href={href} className="block rounded-md px-3 py-1.5 text-xs text-zinc-500 transition hover:bg-zinc-900 hover:text-zinc-300">{label}</Link>)}</div>
    </div>)}
  </nav>;
}
