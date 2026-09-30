const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";
const API_KEY_STORAGE_KEY = "wraith_api_key";

export interface RunStatus {
  run_id: string; rule_path: string; rule_id: string; rule_title: string; repo: string; pr_number: number;
  stage: "lint" | "provision" | "simulate" | "validate" | "soar" | "done" | "failed";
  passed?: boolean; reason?: string; approved_by?: string; approved_at?: string; deployed_at?: string;
  started_at: string; updated_at: string;
}
export interface RunEvent { id:number; run_id:string; stage:string; level:string; message:string; created_at:string; }
export interface RuleSummary { name:string; path:string; title:string; id:string; status:string; level:string; passed:boolean; issue_count:number; error_count:number; warning_count:number; }
export interface RuleDetail { name:string; path:string; passed:boolean; rule:any; issues:any[]; content:string; }
export interface AuditEvent { id:number; actor:string; actor_role:string; action:string; resource:string; detail:string; ip_address?:string; timestamp:string; }
export interface Session { label:string; role:"viewer"|"analyst"|"lead"|"admin"; }
export interface HealthStatus { status?:string; ready?:boolean; error?:string; [key:string]:unknown; }

export function getApiKey(){ if(typeof window==="undefined") return ""; return window.localStorage.getItem(API_KEY_STORAGE_KEY)||""; }
export function setApiKey(key:string){ if(typeof window==="undefined") return; if(key) window.localStorage.setItem(API_KEY_STORAGE_KEY,key); else window.localStorage.removeItem(API_KEY_STORAGE_KEY); }
function authHeaders():HeadersInit { const key=getApiKey(); return key?{Authorization:`Bearer ${key}`}:{}; }

export const fetcher=(path:string)=>fetch(`${API_BASE}${path}`,{headers:authHeaders()}).then(async res=>{if(!res.ok){const body=await res.json().catch(()=>({}));throw new Error(body.error||`API error ${res.status}`)}return res.json()});
async function post(path:string,body?:unknown):Promise<{ok:boolean;data:any}>{const headers:HeadersInit={...authHeaders(),"Content-Type":"application/json"};const res=await fetch(`${API_BASE}${path}`,{method:"POST",headers,body:body===undefined?undefined:JSON.stringify(body)});const data=await res.json().catch(()=>({}));return{ok:res.ok,data};}
export function fetchRuns(){return fetcher("/runs") as Promise<RunStatus[]>}
export function fetchRun(id:string){return fetcher(`/runs/${id}`) as Promise<RunStatus>}
export function fetchRunEvents(id:string){return fetcher(`/runs/${id}/events?limit=1000`) as Promise<RunEvent[]>}
export function fetchReport(id:string){return fetcher(`/runs/${id}/report`)}
export function fetchAttestation(id:string){return fetcher(`/runs/${id}/attestation`)}
export function fetchRules(){return fetcher("/rules") as Promise<{count:number;rules:RuleSummary[]}>}
export function fetchRule(name:string){return fetcher(`/rules/${encodeURIComponent(name)}`) as Promise<RuleDetail>}
export function fetchAudit(){return fetcher("/audit") as Promise<AuditEvent[]>}
export function fetchSession(){return fetcher("/session") as Promise<Session>}
export function fetchHealth(){return fetcher("/healthz") as Promise<HealthStatus>}
export function fetchReady(){return fetcher("/readyz") as Promise<HealthStatus>}
export async function lintRules(){const result=await post("/lint",{});if(!result.ok)throw new Error(result.data?.error||"Lint request failed");return result.data}
export function approveRun(id:string){return post(`/runs/${id}/approve`)}
export function deployRun(id:string){return post(`/runs/${id}/deploy`)}
