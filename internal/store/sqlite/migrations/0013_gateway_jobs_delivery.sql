ALTER TABLE gateway_jobs ADD COLUMN delivery_mode TEXT NOT NULL DEFAULT '';

ALTER TABLE gateway_jobs ADD COLUMN webhook_url TEXT NOT NULL DEFAULT '';

ALTER TABLE gateway_jobs ADD COLUMN webhook_delivered_at TEXT NOT NULL DEFAULT '';
