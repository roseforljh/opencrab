ALTER TABLE response_sessions ADD COLUMN input_items_json TEXT NOT NULL DEFAULT '';
ALTER TABLE response_sessions ADD COLUMN response_json TEXT NOT NULL DEFAULT '';
