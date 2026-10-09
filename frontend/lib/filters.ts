export interface AuditFilterRecord {
  id?: string | number;
  actor?: string;
  actor_role?: string;
  action?: string;
  resource?: string;
  detail?: string;
  ip_address?: string;
  timestamp?: string;
}

export interface AuditFilterOptions {
  query?: string;
  action?: string;
  role?: string;
}

function timestampValue(value?: string): number {
  if (!value) return 0;
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function filterAuditRecords<T extends AuditFilterRecord>(
  records: readonly T[],
  options: AuditFilterOptions = {},
): T[] {
  const query = (options.query ?? "").trim().toLocaleLowerCase();
  return records.filter((item) => {
    const fields = [item.id, item.actor, item.actor_role, item.action, item.resource, item.detail, item.ip_address, item.timestamp];
    const matchesQuery = !query || fields.some((value) => String(value ?? "").toLocaleLowerCase().includes(query));
    return matchesQuery
      && (!options.action || options.action === "all" || item.action === options.action)
      && (!options.role || options.role === "all" || item.actor_role === options.role);
  }).slice().sort((a, b) => timestampValue(b.timestamp) - timestampValue(a.timestamp));
}

export interface JobFilterRecord {
  id?: string | number;
  run_id?: string;
  rule_path?: string;
  status?: string;
  last_error?: string;
  updated_at?: string;
  created_at?: string;
}

export type JobFilter = "all" | "queued" | "running" | "failed" | "complete";

export function filterJobs<T extends JobFilterRecord>(
  jobs: readonly T[],
  query = "",
  filter: JobFilter = "all",
): T[] {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  return jobs.filter((job) => {
    const fields = [job.id, job.run_id, job.rule_path, job.status, job.last_error];
    const matchesText = !normalizedQuery || fields.some((value) => String(value ?? "").toLocaleLowerCase().includes(normalizedQuery));
    const matchesStatus = filter === "all"
      || (filter === "queued" && job.status === "queued")
      || (filter === "running" && job.status === "running")
      || (filter === "failed" && (job.status === "failed" || job.status === "cancelled"))
      || (filter === "complete" && (job.status === "completed" || job.status === "done"));
    return matchesText && matchesStatus;
  }).slice().sort((a, b) => timestampValue(b.updated_at || b.created_at) - timestampValue(a.updated_at || a.created_at));
}
