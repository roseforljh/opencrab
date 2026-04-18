CREATE TABLE IF NOT EXISTS capability_profiles (
  scope_type TEXT NOT NULL,
  scope_key TEXT NOT NULL,
  operation TEXT NOT NULL,
  config_json TEXT NOT NULL DEFAULT '{}',
  updated_at TEXT NOT NULL,
  PRIMARY KEY (scope_type, scope_key, operation)
);
