const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";
// Keep credentials in memory only: browser storage is readable by injected JavaScript.\nlet inMemoryApiKey = "";

export interface RunStatus { run_id:string; rule_path:string; rule_id:string; rule_title:string; repo:string; pr_number:number; stage:"lint"|"provision"|"simulate"|"validate"|"soar"|"done"|"failed"; passed?:boolean; reason?:string; approved_by?:string; approved_at?:string; deployed_at?:string; started_at:string; updated_at:string; }
export interface HealthStatus { status?:string; ready?:boolean; error?:string; [key:string]:unknown }
export interface RuleRecord { name?:string; path?:string; rule_id?:string; title?:string; [key:string]:unknown }
export interface RuleDetail { passed?:boolean; content?:string; rule?:{id?:string; level?:string; [key:string]:unknown}; issues?:Array<{severity?:string; field?:string; message?:string; [key:string]:unknown}>; [key:string]:unknown }
export interface JobRecord { id?:number|string; run_id?:string; rule_path?:string; status?:string; stage?:string; attempts?:number; max_attempts?:number; available_at?:string; last_error?:string; created_at?:string; updated_at?:string; [key:string]:unknown }
export interface SessionInfo { authenticated?:boolean; label?:string; role?:string; [key:string]:unknown }
export interface AuditEvent { [key:string]:unknown }
export function getApiKey():string { if(typeof window==="undefined")return ""; return window.localStorage.getItem(API_KEY_STORAGE_KEY)||""; }
export function setApiKey(key:string){ if(typeof window==="undefined")return; if(key)window.localStorage.setItem(API_KEY_STORAGE_KEY,key); else window.localStorage.removeItem(API_KEY_STORAGE_KEY); }
function authHeaders():HeadersInit{const key=getApiKey();return key?{Authorization:`Bearer ${key}`}:{};}
export const fetcher=(path:string)=>fetch(`${API_BASE}${path}`,{headers:authHeaders()}).then(async res=>{if(!res.ok){const body=await res.json().catch(()=>({}));throw new Error(body.error||`API error ${res.status}`);}return res.json();});
async function post(path:string,body?:unknown):Promise<{ok:boolean;data:any}>{const res=await fetch(`${API_BASE}${path}`,{method:"POST",headers:{...authHeaders(),"Content-Type":"application/json"},body:body===undefined?undefined:JSON.stringify(body)});const data=await res.json().catch(()=>({}));return{ok:res.ok,data};}

export const fetchRuns=():Promise<RunStatus[]>=>fetcher("/runs");
export const fetchRun=(id:string):Promise<RunStatus>=>fetcher(`/runs/${encodeURIComponent(id)}`);
export const fetchReport=(id:string):Promise<RunReport>=>fetcher(`/runs/${encodeURIComponent(id)}/report`);
export const fetchAttestation=(id:string):Promise<Attestation>=>fetcher(`/runs/${encodeURIComponent(id)}/attestation`);
export const fetchStages=(id:string):Promise<PipelineStage[]>=>fetcher(`/runs/${encodeURIComponent(id)}/stages`);
export const fetchEvents=(id:string):Promise<any[]>=>fetcher(`/runs/${encodeURIComponent(id)}/events`);
export const fetchRules=():Promise<RuleRecord[]>=>fetcher("/rules");
export const fetchRule=(name:string):Promise<RuleRecord>=>fetcher(`/rules/${encodeURIComponent(name)}`);
export const fetchMetrics=():Promise<string>=>fetch(`${API_BASE}/metrics`,{headers:authHeaders()}).then(async res=>{if(!res.ok)throw new Error(`Metrics request failed (${res.status})`);return res.text();});
export const fetchJobs=():Promise<JobRecord[]>=>fetcher("/jobs");
export const retryPipelineJob=(id:string)=>post(`/jobs/${encodeURIComponent(id)}/retry`);
export const fetchSession=():Promise<SessionInfo>=>fetcher("/session");
export const fetchAudit=():Promise<AuditEvent[]>=>fetcher("/audit");
export const fetchHealth=():Promise<HealthStatus>=>fetcher("/healthz");
export const fetchReady=():Promise<HealthStatus>=>fetcher("/readyz");
export async function lintRules(body:unknown){const r=await post("/lint",body);if(!r.ok)throw new Error(r.data?.error||"Lint request failed");return r.data;}
export async function validatePlayground(rule:string){const r=await post("/playground/validate",{rule});if(!r.ok)throw new Error(r.data?.error||"Playground validation failed");return r.data;}
export const approveRun=(id:string)=>post(`/runs/${encodeURIComponent(id)}/approve`);
export const deployRun=(id:string)=>post(`/runs/${encodeURIComponent(id)}/deploy`);
export const dryRunDeploy=(id:string)=>post(`/runs/${encodeURIComponent(id)}/deploy/dry-run`);
export const fetchDeployments=():Promise<any[]>=>fetcher("/deployments");
export const verifyDeployment=(id:string)=>fetcher(`/deployments/${encodeURIComponent(id)}/verify`);
export const rollbackDeployment=(id:string)=>post(`/deployments/${encodeURIComponent(id)}/rollback`);
export const fetchAPIKeys=():Promise<APIKeyRecord[]>=>fetcher("/api-keys");
export const createAPIKey=(label:string,role:string)=>post("/api-keys",{label,role});
export const revokeAPIKey=(id:string)=>post(`/api-keys/${encodeURIComponent(id)}/revoke`);

export interface StageValidate{fired_on_attack?:boolean;attack_hit_count?:number;fired_on_baseline?:boolean;baseline_hit_count?:number;baseline_docs_scanned?:number;false_positive_rate?:number;passed?:boolean;reason?:string}
export interface StageAttackSimulation{status?:string;target_os?:string;target_os_source?:string;rule_tagged_techniques?:string[];simulated_chain?:string[];simulated_user?:string;simulated_host?:string;events_indexed?:number}
export interface StageRobustness{status?:string;reason?:string;rule_id?:string;variants_tested?:number;variants_detected?:number;score?:number;undetected_examples?:{mutation_chain:string[];event:Record<string,any>}[]}
export interface StageSoarPlaybook{status?:string;path?:string;pr_url?:string;error?:string}
export interface StageQualityScore{available_weight?:number;max?:number;score?:number;rating?:string;components?:Record<string,number>}
export interface RunReport{run_id:string;rule_path:string;rule_id:string;passed:boolean;duration_seconds:number;environment?:{target_os?:string;target_os_source?:string};stages:{translate?:{status?:string;query_dsl_path?:string};baseline?:{status?:string;events_indexed?:number};attack_simulation?:StageAttackSimulation;validate?:StageValidate;robustness?:StageRobustness;quality_score?:StageQualityScore;soar_playbook?:StageSoarPlaybook}}
export interface Attestation{attestation:{schema_version:string;run_id:string;rule_id:string;rule_content_sha256:string;tested_techniques:string[];baseline_events_n:number;false_positive_rate:number;robustness_score:number;passed:boolean;issued_at:string;issuer:string};signature:string;public_key:string}
export interface PipelineStage{id:number;run_id:string;name:string;status:"running"|"passed"|"failed"|"skipped";reason?:string;started_at:string;ended_at?:string}
export interface APIKeyRecord{id:string;label:string;role:string;created_at:string;revoked:boolean}
