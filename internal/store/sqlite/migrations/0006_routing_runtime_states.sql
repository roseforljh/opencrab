CREATE TABLE IF NOT EXISTS routing_runtime_states (
  route_id INTEGER PRIMARY KEY,
  cooldown_until TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
