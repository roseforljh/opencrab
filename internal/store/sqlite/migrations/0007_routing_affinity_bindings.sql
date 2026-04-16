CREATE TABLE IF NOT EXISTS routing_affinity_bindings (
  affinity_key TEXT NOT NULL,
  model_alias TEXT NOT NULL,
  protocol TEXT NOT NULL,
  route_id INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (affinity_key, model_alias, protocol)
);
