import WraithShell from "../components/WraithShell";

const services = [["Frontend","Next.js control surface","Presentation / analyst workflow"],["Go control plane","Auth, policy, orchestration","API / governance boundary"],["Python engine","Simulation, validation, mutation","Detection intelligence"],["PostgreSQL","Runs, metadata, governance","System of record"],["Elasticsearch","Synthetic telemetry / queries","SIEM validation"],["Neo4j","Technique relationships","Attack graph"]];

export default function Architecture() {
 return <WraithShell eyebrow="Platform Architecture">
   <section className="page-hero"><div><div className="eyebrow">Platform architecture</div><h1>Wraith infrastructure map</h1><p>A security boundary between analyst workflows, orchestration, detection intelligence and evidence stores.</p></div></section>
   <section className="architecture-canvas">
     <div className="arch-lane"><span className="arch-label">Experience</span><div className="arch-node primary"><b>WRAITH UI</b><small>Next.js / analyst control surface</small></div></div>
     <div className="arch-connector">↓ authenticated API requests ↓</div>
     <div className="arch-lane"><span className="arch-label">Control plane</span><div className="arch-node primary"><b>GO CONTROL PLANE</b><small>Auth · RBAC · orchestration · provenance</small></div></div>
     <div className="arch-connector">↓ isolated engine jobs ↓</div>
     <div className="arch-lane"><span className="arch-label">Detection intelligence</span><div className="arch-node-grid">{services.slice(2,3).map(([a,b,c]) => <div className="arch-node" key={a}><b>{a}</b><small>{b}</small><em>{c}</em></div>)}</div></div>
     <div className="arch-connector">↓ evidence / telemetry ↓</div>
     <div className="arch-lane"><span className="arch-label">Data services</span><div className="arch-node-grid">{services.slice(3).map(([a,b,c]) => <div className="arch-node" key={a}><b>{a}</b><small>{b}</small><em>{c}</em></div>)}</div></div>
   </section>
   <section className="panel"><div className="panel-heading"><div><h2>Security boundaries</h2><p>Credentials stay server-side; the browser consumes control-plane APIs.</p></div></div><div className="boundary-list"><div>01 <span>Browser → Go API</span><b>Authenticated</b></div><div>02 <span>Go API → Engine</span><b>Orchestrated</b></div><div>03 <span>Engine → data services</span><b>Scoped</b></div><div>04 <span>Run → provenance</span><b>Attested</b></div></div></section>
 </WraithShell>
}
