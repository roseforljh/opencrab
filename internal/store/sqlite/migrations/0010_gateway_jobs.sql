CREATE TABLE IF NOT EXISTS gateway_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  request_id TEXT NOT NULL UNIQUE,
  idempotency_key TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL,
  model TEXT NOT NULL,
  status TEXT NOT NULL,
  mode TEXT NOT NULL,
  request_path TEXT NOT NULL,
  request_body TEXT NOT NULL,
  request_headers TEXT NOT NULL DEFAULT '',
  response_status_code INTEGER NOT NULL DEFAULT 0,
  response_body TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  accepted_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  estimated_ready_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_gateway_jobs_idempotency_key ON gateway_jobs(idempotency_key) WHERE idempotency_key <> '';

CREATE INDEX IF NOT EXISTS idx_gateway_jobs_status_updated_at ON gateway_jobs(status, updated_at);
