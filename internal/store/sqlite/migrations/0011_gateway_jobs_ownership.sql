ALTER TABLE gateway_jobs ADD COLUMN owner_key_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE gateway_jobs ADD COLUMN request_hash TEXT NOT NULL DEFAULT '';

DROP INDEX IF EXISTS idx_gateway_jobs_idempotency_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_gateway_jobs_owner_idempotency_key ON gateway_jobs(owner_key_hash, idempotency_key) WHERE idempotency_key <> '';

CREATE INDEX IF NOT EXISTS idx_gateway_jobs_owner_request_id ON gateway_jobs(owner_key_hash, request_id);
