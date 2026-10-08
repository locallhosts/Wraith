import { FormEvent, useState } from "react";
import Link from "next/link";
import WraithShell from "../components/WraithShell";
import Pipeline from "../components/security/Pipeline";
import { validatePlayground } from "../lib/api";

const example=`title: Suspicious PowerShell Execution
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

export default function Playground(){
 const [rule,setRule]=useState(example),[result,setResult]=useState<any>(null),[error,setError]=useState(""),[busy,setBusy]=useState(false);
 async function validate(e:FormEvent){e.preventDefault();setError("");setResult(null);setBusy(true);try{setResult(await validatePlayground(rule));}catch(err){setError(err instanceof Error?err.message:"Playground validation failed");}finally{setBusy(false);}}
 return <WraithShell eyebrow="Detection Playground">
   <section className="page-hero"><div><div className="eyebrow">Isolated detection laboratory</div><h1>Playground</h1><p>Test Sigma syntax and translation before committing a rule to the authenticated control plane. Public playground mode is deliberately isolated by the backend.</p></div><Link href="/detections" className="primary-button">Detection workspace →</Link></section>
   <div className="dashboard-grid">
     <form onSubmit={validate} className="panel" style={{marginBottom:0}}><div className="panel-heading"><div><h2>Rule editor</h2><p>Maximum 256 KiB. No production writes.</p></div><span className="status-badge success"><i/>isolated</span></div><div style={{padding:18}}><textarea value={rule} onChange={e=>setRule(e.target.value)} spellCheck={false} className="playground-editor"/><div style={{display:"flex",justifyContent:"space-between",alignItems:"center",marginTop:10}}><span style={{fontSize:9,color:"#536173"}}>{rule.length.toLocaleString()} bytes</span><button disabled={busy||!rule||rule.length>256*1024} className="primary-button" type="submit">{busy?"Analyzing…":"Run analysis"}</button></div></div></form>
     <section className="panel" style={{marginBottom:0}}><div className="panel-heading"><div><h2>Validation pipeline</h2><p>Rule → lint → translation → evidence</p></div></div><div style={{padding:18}}><Pipeline active={busy?2:result?4:0}/></div>{error&&<div className="error-box">{error}</div>}{!result&&!error&&<div className="empty-state">Run an isolated analysis to inspect the generated evidence.</div>}{result&&<div className="evidence-stack"><div className="evidence-card"><span>Execution</span><strong>{result.execution||"completed"}</strong></div><div className="evidence-card"><span>Lint findings</span><pre>{JSON.stringify(result.lint,null,2)}</pre></div>{result.query_dsl&&<div className="evidence-card"><span>Elasticsearch Query DSL</span><pre>{JSON.stringify(result.query_dsl,null,2)}</pre></div>}</div>}</section>
   </div>
 </WraithShell>;
}
