import Link from "next/link";

const groups = [
 {label:"Introduction",items:[["Overview","/"],["Architecture","/introduction"]]},
 {label:"Operations",items:[["Validation runs","/#runs"],["Pipeline jobs","/jobs"],["Rules workspace","/rules"],["Capabilities","/#capabilities"]]},
 {label:"Detection Engineering",items:[["Rules","/rules"],["Mutation & robustness","/#evidence"],["Quality scoring","/#evidence"]]},
 {label:"Threat Intelligence",items:[["MITRE ATT&CK","/#evidence"],["Attack chains","/#evidence"]]},
 {label:"Validation",items:[["Attack simulation","/#evidence"],["Detection validation","/#evidence"],["Baseline testing","/#evidence"],["Adversarial testing","/#evidence"]]},
 {label:"Integrations",items:[["Integration health","/integrations"],["SIEM / Elasticsearch","/integrations"],["Neo4j attack graph","/integrations"],["GitHub webhook","/integrations"]]},
 {label:"Automation",items:[["SOAR","/#evidence"],["CI/CD","/integrations"]]},
 {label:"Governance",items:[["Audit trail","/audit"],["Provenance","/#evidence"],["Approval & deployment","/#evidence"]]},
 {label:"System",items:[["Service health","/#system"],["Prometheus telemetry","/metrics"],["Playground","/playground"]]},
];
export default function PlatformNav(){return <nav className="sticky top-6 space-y-5">{groups.map(g=><div key={g.label}><div className="mb-1 px-3 text-[10px] uppercase tracking-[0.18em] text-zinc-700">{g.label}</div><div className="space-y-0.5">{g.items.map(([label,href])=><Link key={label} href={href} className="block rounded-md px-3 py-1.5 text-xs text-zinc-500 transition hover:bg-zinc-900 hover:text-zinc-300">{label}</Link>)}</div></div>)}</nav>}
