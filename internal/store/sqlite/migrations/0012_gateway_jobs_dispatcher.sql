ALTER TABLE gateway_jobs ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE gateway_jobs ADD COLUMN worker_id TEXT NOT NULL DEFAULT '';

ALTER TABLE gateway_jobs ADD COLUMN session_id TEXT NOT NULL DEFAULT '';

ALTER TABLE gateway_jobs ADD COLUMN lease_until TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_gateway_jobs_dispatch ON gateway_jobs(status, estimated_ready_at, lease_until, updated_at);
