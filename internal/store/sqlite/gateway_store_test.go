package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

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
INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-a', 'claude', 1, 'ignored-model', 'now', 'now'),
('gpt-4o', 'gemini-b', 'gemini', 2, 'also-ignored', 'now', 'now');`)
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
	if routes[0].InvocationMode != "claude" {
		t.Fatalf("unexpected invocation mode: %#v", routes[0])
	}
	if routes[0].FallbackModel != "ignored-model" {
		t.Fatalf("expected fallback model to be loaded, got %#v", routes[0])
	}
}

func TestGatewayStoreListEnabledRoutesByAliasTarget(t *testing.T) {
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
('codex', 'openai', 'https://api.openai.com/v1', 'k1', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-5.4', 'gpt-5.4', 'now', 'now'),
('aaa', 'gpt-5.4', 'now', 'now');
INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES
('gpt-5.4', 'codex', 'openai', 1, '', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	store := NewGatewayStore(db)
	routes, err := store.ListEnabledRoutesByModel(context.Background(), "aaa")
	if err != nil {
		t.Fatalf("list routes by alias target: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 resolved route, got %d", len(routes))
	}
	if routes[0].Channel.Name != "codex" || routes[0].UpstreamModel != "gpt-5.4" {
		t.Fatalf("unexpected resolved route: %#v", routes[0])
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
	if !strings.Contains(items[0].Details, `"log_type":"gateway_attempt"`) || !strings.Contains(items[0].Details, `"attempt":1`) || !strings.Contains(items[0].Details, `"routing_strategy"`) {
		t.Fatalf("unexpected details: %s", items[0].Details)
	}
}

func TestGatewayAttemptLogStoreOmitsBodiesOnSuccess(t *testing.T) {
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
		RequestID:    "req-success",
		Model:        "gpt-4o",
		Channel:      "claude-a",
		Provider:     "claude",
		Attempt:      1,
		StatusCode:   200,
		Success:      true,
		RequestBody:  `{"model":"gpt-4o"}`,
		ResponseBody: `{"ok":true}`,
	})
	if err != nil {
		t.Fatalf("log success attempt: %v", err)
	}

	items, err := NewRequestLogStore(db).List(context.Background())
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 log, got %d", len(items))
	}
	if items[0].RequestBody != "" || items[0].ResponseBody != "" {
		t.Fatalf("expected success attempt bodies to be omitted, got %#v", items[0])
	}
}

func TestRoutingRuntimeAndStickyStores(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	runtimeStore := NewRoutingRuntimeStateStore(db)
	until, err := runtimeStore.MarkCooldown(context.Background(), 12, time.Minute, "upstream 503")
	if err != nil || until == "" {
		t.Fatalf("mark cooldown: until=%s err=%v", until, err)
	}
	count, err := runtimeStore.CountActiveCooldowns(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("count cooldowns: count=%d err=%v", count, err)
	}
	if err := runtimeStore.ClearCooldown(context.Background(), 12); err != nil {
		t.Fatalf("clear cooldown: %v", err)
	}

	stickyStore := NewStickyRoutingStore(db)
	if err := stickyStore.UpsertStickyBinding(context.Background(), "session-1", "gpt-4o", domain.ProtocolOpenAI, 12); err != nil {
		t.Fatalf("upsert sticky binding: %v", err)
	}
	routeID, found, err := stickyStore.GetStickyBinding(context.Background(), "session-1", "gpt-4o", domain.ProtocolOpenAI)
	if err != nil || !found || routeID != 12 {
		t.Fatalf("get sticky binding: routeID=%d found=%v err=%v", routeID, found, err)
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

func TestRoutingConfigStoreDefaultsToSequential(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	strategy, err := NewRoutingConfigStore(db).GetRoutingStrategy(context.Background())
	if err != nil {
		t.Fatalf("get routing strategy: %v", err)
	}
	if strategy != domain.RoutingStrategySequential {
		t.Fatalf("unexpected strategy: %s", strategy)
	}
}

func TestRoutingCursorStoreReadAndAdvance(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewRoutingCursorStore(db)
	value, err := store.GetRoutingCursor(context.Background(), "gpt-4o|openai|matched|1")
	if err != nil {
		t.Fatalf("get cursor: %v", err)
	}
	if value != 0 {
		t.Fatalf("expected zero default cursor, got %d", value)
	}
	if err := store.AdvanceRoutingCursor(context.Background(), "gpt-4o|openai|matched|1", 3, 1); err != nil {
		t.Fatalf("advance cursor: %v", err)
	}
	value, err = store.GetRoutingCursor(context.Background(), "gpt-4o|openai|matched|1")
	if err != nil {
		t.Fatalf("get cursor after advance: %v", err)
	}
	if value != 2 {
		t.Fatalf("expected cursor 2, got %d", value)
	}
}
