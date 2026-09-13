const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";
const API_KEY_STORAGE_KEY = "wraith_api_key";

export interface RunStatus {
  run_id: string;
  rule_path: string;
  rule_id: string;
  rule_title: string;
  repo: string;
  pr_number: number;
  stage: "lint" | "provision" | "simulate" | "validate" | "soar" | "done" | "failed";
  passed?: boolean;
  reason?: string;
  approved_by?: string;
  approved_at?: string;
  deployed_at?: string;
  started_at: string;
  updated_at: string;
}

export function getApiKey(): string {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(API_KEY_STORAGE_KEY) || "";
}

export function setApiKey(key: string) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(API_KEY_STORAGE_KEY, key);
}

function authHeaders(): HeadersInit {
  const key = getApiKey();
  return key ? { Authorization: `Bearer ${key}` } : {};
}

export const fetcher = (path: string) =>
  fetch(`${API_BASE}${path}`, { headers: authHeaders() }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `API error ${res.status}`);
    }
    return res.json();
  });

async function post(path: string): Promise<{ ok: boolean; data: any }> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: authHeaders(),
  });
  const data = await res.json().catch(() => ({}));
  return { ok: res.ok, data };
}

export async function fetchRuns(): Promise<RunStatus[]> {
  return fetcher("/runs");
}

export async function fetchRun(id: string): Promise<RunStatus> {
  return fetcher(`/runs/${id}`);
}

export interface StageValidate {
  fired_on_attack?: boolean;
  attack_hit_count?: number;
  fired_on_baseline?: boolean;
  baseline_hit_count?: number;
  baseline_docs_scanned?: number;
  false_positive_rate?: number;
  passed?: boolean;
  reason?: string;
}

export interface StageAttackSimulation {
  status?: string;
  rule_tagged_techniques?: string[];
  simulated_chain?: string[];
  simulated_user?: string;
  simulated_host?: string;
  events_indexed?: number;
}

export interface StageRobustness {
  status?: string;
  reason?: string;
  rule_id?: string;
  variants_tested?: number;
  variants_detected?: number;
  score?: number;
  undetected_examples?: { mutation_chain: string[]; event: Record<string, any> }[];
}

export interface StageSoarPlaybook {
  status?: string;
  path?: string;
  pr_url?: string;
  error?: string;
}

export interface RunReport {
  run_id: string;
  rule_path: string;
  rule_id: string;
  passed: boolean;
  duration_seconds: number;
  stages: {
    translate?: { status?: string; query_dsl_path?: string };
    baseline?: { status?: string; events_indexed?: number };
    attack_simulation?: StageAttackSimulation;
    validate?: StageValidate;
    robustness?: StageRobustness;
    soar_playbook?: StageSoarPlaybook;
  };
}

export interface Attestation {
  attestation: {
    schema_version: string;
    run_id: string;
    rule_id: string;
    rule_content_sha256: string;
    tested_techniques: string[];
    baseline_events_n: number;
    false_positive_rate: number;
    robustness_score: number;
    passed: boolean;
    issued_at: string;
    issuer: string;
  };
  signature: string;
  public_key: string;
}

export async function fetchReport(id: string): Promise<RunReport> {
  return fetcher(`/runs/${id}/report`);
}

export async function fetchAttestation(id: string): Promise<Attestation> {
  return fetcher(`/runs/${id}/attestation`);
}

/** Requires an API key with role >= lead. */
export async function approveRun(runId: string) {
  return post(`/runs/${runId}/approve`);
}

/** Requires an API key with role >= lead, plus the run must already be
 * approved and have a valid provenance attestation on disk. */
export async function deployRun(runId: string) {
  return post(`/runs/${runId}/deploy`);
}
