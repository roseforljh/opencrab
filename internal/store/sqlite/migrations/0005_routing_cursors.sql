CREATE TABLE IF NOT EXISTS routing_cursors (
  route_key TEXT PRIMARY KEY,
  next_index INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
