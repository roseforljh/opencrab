package sqlite

import (
	"context"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestGatewayStoreListEnabledRoutesByModel(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
INSERT INTO channels(name, provider, endpoint, api_key, enabled, created_at, updated_at) VALUES
('claude-a', 'claude', 'https://api.anthropic.com', 'k1', 1, 'now', 'now'),
('gemini-b', 'gemini', 'https://generativelanguage.googleapis.com', 'k2', 0, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now');
INSERT INTO model_routes(model_alias, channel_name, priority, fallback_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-a', 1, 'ignored-model', 'now', 'now'),
('gpt-4o', 'gemini-b', 2, 'also-ignored', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	store := NewGatewayStore(db)
	routes, err := store.ListEnabledRoutesByModel(context.Background(), "gpt-4o")
	if err != nil {
		t.Fatalf("list routes: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 enabled route, got %d", len(routes))
	}
	if routes[0].Channel.Name != "claude-a" || routes[0].UpstreamModel != "claude-3-5-sonnet" {
		t.Fatalf("unexpected route: %#v", routes[0])
	}
}

func TestGatewayAttemptLogStoreLogGatewayAttempt(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewGatewayAttemptLogStore(db)
	err = store.LogGatewayAttempt(context.Background(), domain.GatewayAttemptLog{
		RequestID:     "req-1",
		Model:         "gpt-4o",
		UpstreamModel: "claude-3-5-sonnet",
		Channel:       "claude-a",
		Provider:      "claude",
		Attempt:       1,
		StatusCode:    503,
		Retryable:     true,
		ErrorMessage:  "upstream 503",
		RequestBody:   `{"model":"gpt-4o"}`,
	})
	if err != nil {
		t.Fatalf("log attempt: %v", err)
	}

	items, err := NewRequestLogStore(db).List(context.Background())
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 log, got %d", len(items))
	}
	if items[0].Channel != "claude-a" || items[0].StatusCode != 503 {
		t.Fatalf("unexpected log item: %#v", items[0])
	}
	if !strings.Contains(items[0].Details, `"log_type":"gateway_attempt"`) || !strings.Contains(items[0].Details, `"attempt":1`) {
		t.Fatalf("unexpected details: %s", items[0].Details)
	}
}

func TestRequestLogStoreCreateListAndClear(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewRequestLogStore(db)
	err = store.Create(context.Background(), domain.RequestLog{
		RequestID:    "req-raw",
		Model:        "gpt-4o",
		Channel:      "claude-a",
		StatusCode:   200,
		RequestBody:  `{"model":"gpt-4o"}`,
		ResponseBody: `{"ok":true}`,
		Details:      `{"log_type":"request"}`,
		CreatedAt:    "2026-04-11T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("create request log: %v", err)
	}

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list request logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 log, got %d", len(items))
	}
	if items[0].RequestID != "req-raw" || !strings.Contains(items[0].Details, `"log_type":"request"`) {
		t.Fatalf("unexpected request log: %#v", items[0])
	}

	if err := store.Clear(context.Background()); err != nil {
		t.Fatalf("clear request logs: %v", err)
	}
	items, err = store.List(context.Background())
	if err != nil {
		t.Fatalf("list request logs after clear: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 log after clear, got %d", len(items))
	}
}
