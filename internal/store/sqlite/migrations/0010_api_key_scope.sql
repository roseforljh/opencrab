ALTER TABLE api_keys ADD COLUMN channel_name TEXT NOT NULL DEFAULT '';

ALTER TABLE api_keys ADD COLUMN model_alias TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_api_keys_channel_name ON api_keys(channel_name);

CREATE INDEX IF NOT EXISTS idx_api_keys_model_alias ON api_keys(model_alias);
