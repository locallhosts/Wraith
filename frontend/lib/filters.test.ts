import { describe, expect, it } from "vitest";
import { filterAuditRecords, filterJobs } from "./filters";

describe("filterAuditRecords", () => {
  const records: Array<{ id: number; actor?: string; actor_role?: string; action?: string; detail?: string; ip_address?: string; timestamp?: string }> = [
    { id: 1, actor: "alice", actor_role: "admin", action: "key.create", timestamp: "2026-01-01T10:00:00Z" },
    { id: 2, actor: "bob", actor_role: "analyst", action: "run.retry", detail: "retry failed job", timestamp: "2026-01-02T10:00:00Z" },
    { id: 3, actor: "carol", actor_role: "admin", action: "key.revoke", ip_address: "192.0.2.10", timestamp: "invalid-date" },
  ];

  it("matches case-insensitive text across audit fields and sorts newest first", () => {
    expect(filterAuditRecords(records, { query: "RETRY" }).map((item) => item.id)).toEqual([2]);
    expect(filterAuditRecords(records).map((item) => item.id)).toEqual([2, 1, 3]);
  });

  it("combines action and role filters without mutating input", () => {
    const originalOrder = records.map((item) => item.id);
    expect(filterAuditRecords(records, { action: "key.create", role: "admin" }).map((item) => item.id)).toEqual([1]);
    expect(records.map((item) => item.id)).toEqual(originalOrder);
  });

  it("treats an empty query as no text restriction", () => {
    expect(filterAuditRecords(records, { query: "   " })).toHaveLength(3);
  });
});

describe("filterJobs", () => {
  const jobs: Array<{ id: number; status: string; last_error?: string; run_id?: string; rule_path?: string; updated_at?: string; created_at?: string }> = [
    { id: 1, status: "failed", last_error: "timeout", updated_at: "2026-02-01T00:00:00Z" },
    { id: 2, status: "cancelled", run_id: "run-abc", updated_at: "2026-02-02T00:00:00Z" },
    { id: 3, status: "completed", rule_path: "rules/example.yml", created_at: "2026-01-01T00:00:00Z" },
    { id: 4, status: "running", updated_at: "invalid-date" },
  ];

  it("groups cancelled jobs with failures and sorts safely", () => {
    expect(filterJobs(jobs, "", "failed").map((job) => job.id)).toEqual([2, 1]);
  });

  it("searches identifiers, rule paths and error details", () => {
    expect(filterJobs(jobs, "RUN-ABC").map((job) => job.id)).toEqual([2]);
    expect(filterJobs(jobs, "timeout", "failed").map((job) => job.id)).toEqual([1]);
  });

  it("supports completed status aliases and preserves the source list", () => {
    const originalOrder = jobs.map((job) => job.id);
    expect(filterJobs(jobs, "", "complete").map((job) => job.id)).toEqual([3]);
    expect(jobs.map((job) => job.id)).toEqual(originalOrder);
  });
});
