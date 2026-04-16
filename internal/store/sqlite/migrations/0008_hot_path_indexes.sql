CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash_enabled ON api_keys(key_hash, enabled);

CREATE INDEX IF NOT EXISTS idx_model_routes_model_alias_priority ON model_routes(model_alias, priority);

CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
