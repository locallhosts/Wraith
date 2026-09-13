-- WRAITH production schema.
-- Applied automatically by backend-go/store/postgres.go on startup
-- (idempotent — safe to run repeatedly), or manually via:
--   psql "$WRAITH_DATABASE_URL" -f schema.sql

CREATE TABLE IF NOT EXISTS runs (
    run_id       TEXT PRIMARY KEY,
    rule_path    TEXT NOT NULL,
    rule_id      TEXT NOT NULL,
    rule_title   TEXT NOT NULL DEFAULT '',
    repo         TEXT NOT NULL,
    pr_number    INTEGER NOT NULL DEFAULT 0,
    stage        TEXT NOT NULL,
    passed       BOOLEAN,
    reason       TEXT NOT NULL DEFAULT '',
    approved_by  TEXT NOT NULL DEFAULT '',
    approved_at  TIMESTAMPTZ,
    deployed_at  TIMESTAMPTZ,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_rule_id ON runs (rule_id);
CREATE INDEX IF NOT EXISTS idx_runs_repo ON runs (repo);

-- Append-only audit trail. Every state-changing API call writes here —
-- see backend-go/audit/audit.go. Rows are never updated or deleted by the
-- application; retention/archival is an operator concern (e.g. partition
-- by month and ship old partitions to cold storage for compliance).
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    actor       TEXT NOT NULL,
    actor_role  TEXT NOT NULL,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    ip_address  TEXT NOT NULL DEFAULT '',
    ts          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log (ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log (resource);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log (actor);

-- API keys are stored as SHA-256 hashes only — the raw key is shown to the
-- operator exactly once at creation time (see `wraith apikey create`).
CREATE TABLE IF NOT EXISTS api_keys (
    key_hash    TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('viewer', 'analyst', 'lead', 'admin')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked     BOOLEAN NOT NULL DEFAULT false
);
