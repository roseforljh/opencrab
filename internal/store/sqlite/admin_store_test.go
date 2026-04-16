package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"opencrab/internal/domain"
)

func TestUpdateModelMappingPropagatesAliasToRoutes(t *testing.T) {
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
('claude-a', 'claude', 'https://api.anthropic.com', 'k1', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now');
INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-a', 'claude', 1, '', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	if err := UpdateModelMapping(context.Background(), db, 1, domain.UpdateModelMappingInput{
		Alias:         "gpt-4.1",
		UpstreamModel: "claude-3-7-sonnet",
	}); err != nil {
		t.Fatalf("update model mapping: %v", err)
	}

	var alias string
	var upstreamModel string
	if err := db.QueryRowContext(context.Background(), `SELECT alias, upstream_model FROM models WHERE id = 1`).Scan(&alias, &upstreamModel); err != nil {
		t.Fatalf("query models: %v", err)
	}
	if alias != "gpt-4.1" || upstreamModel != "claude-3-7-sonnet" {
		t.Fatalf("unexpected model mapping: alias=%s upstream=%s", alias, upstreamModel)
	}

	var routeAlias string
	if err := db.QueryRowContext(context.Background(), `SELECT model_alias FROM model_routes WHERE id = 1`).Scan(&routeAlias); err != nil {
		t.Fatalf("query model_routes: %v", err)
	}
	if routeAlias != "gpt-4.1" {
		t.Fatalf("unexpected route alias: %s", routeAlias)
	}
}

func TestCreateChannelPersistsDispatchControlFields(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	created, err := CreateChannel(context.Background(), db, domain.CreateChannelInput{
		Name:            "openai-main",
		Provider:        "openai",
		Endpoint:        "https://api.openai.com/v1",
		APIKey:          "sk-test",
		Enabled:         true,
		ModelIDs:        []string{"gpt-4.1"},
		RPMLimit:        1200,
		MaxInflight:     48,
		SafetyFactor:    0.85,
		EnabledForAsync: true,
		DispatchWeight:  140,
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if created.RPMLimit != 1200 || created.MaxInflight != 48 || created.SafetyFactor != 0.85 || !created.EnabledForAsync || created.DispatchWeight != 140 {
		t.Fatalf("unexpected created channel: %#v", created)
	}

	items, err := ListChannels(context.Background(), db)
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(items))
	}
	item := items[0]
	if item.RPMLimit != 1200 || item.MaxInflight != 48 || item.SafetyFactor != 0.85 || !item.EnabledForAsync || item.DispatchWeight != 140 {
		t.Fatalf("unexpected listed channel: %#v", item)
	}
}

func TestGatewayJobStoreCreateAndGetByIdempotencyKey(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewGatewayJobStore(db)
	created, err := store.Create(context.Background(), domain.GatewayJob{
		RequestID:        "req_async_1",
		IdempotencyKey:   "idem-1",
		OwnerKeyHash:     "owner-1",
		RequestHash:      "hash-1",
		Protocol:         domain.ProtocolOpenAI,
		Model:            "gpt-5.4",
		Status:           domain.GatewayJobStatusAccepted,
		Mode:             "async",
		RequestPath:      "/v1/responses",
		RequestBody:      `{"model":"gpt-5.4"}`,
		RequestHeaders:   `{"Prefer":"respond-async"}`,
		AcceptedAt:       time.Now().Format(time.RFC3339),
		EstimatedReadyAt: time.Now().Format(time.RFC3339),
		UpdatedAt:        time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected non-zero id: %#v", created)
	}

	byKey, err := store.GetByIdempotencyKey(context.Background(), "owner-1", "idem-1")
	if err != nil {
		t.Fatalf("get by idempotency key: %v", err)
	}
	if byKey.RequestID != "req_async_1" || byKey.Status != domain.GatewayJobStatusAccepted || byKey.RequestHash != "hash-1" {
		t.Fatalf("unexpected job: %#v", byKey)
	}
}

func TestGatewayJobStoreClaimAndRequeue(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewGatewayJobStore(db)
	_, err = store.Create(context.Background(), domain.GatewayJob{
		RequestID:   "req-dispatch-1",
		Protocol:    domain.ProtocolOpenAI,
		Model:       "gpt-5.4",
		Status:      domain.GatewayJobStatusAccepted,
		Mode:        "async",
		RequestPath: "/v1/responses",
		RequestBody: `{"model":"gpt-5.4","input":[]}`,
		AcceptedAt:  time.Now().Add(-time.Second).Format(time.RFC3339),
		UpdatedAt:   time.Now().Add(-time.Second).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	claimed, err := store.ClaimNextRunnable(context.Background(), "worker-1", time.Now().Add(time.Minute).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed.Status != domain.GatewayJobStatusProcessing || claimed.WorkerID != "worker-1" || claimed.AttemptCount != 1 {
		t.Fatalf("unexpected claimed job: %#v", claimed)
	}
	if err := store.Requeue(context.Background(), "req-dispatch-1", "worker-1", claimed.LeaseUntil, "retry", time.Now().Add(time.Second).Format(time.RFC3339)); err != nil {
		t.Fatalf("requeue job: %v", err)
	}
	reloaded, err := store.GetByRequestID(context.Background(), "req-dispatch-1")
	if err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if reloaded.Status != domain.GatewayJobStatusQueued || reloaded.WorkerID != "" || reloaded.LeaseUntil != "" {
		t.Fatalf("unexpected requeued job: %#v", reloaded)
	}
}

func TestGatewayJobStoreCanTakeOverExpiredLease(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewGatewayJobStore(db)
	_, err = store.Create(context.Background(), domain.GatewayJob{
		RequestID:   "req-dispatch-lease",
		Protocol:    domain.ProtocolOpenAI,
		Model:       "gpt-5.4",
		Status:      domain.GatewayJobStatusAccepted,
		Mode:        "async",
		RequestPath: "/v1/responses",
		RequestBody: `{"model":"gpt-5.4","input":[]}`,
		AcceptedAt:  time.Now().Add(-2 * time.Second).Format(time.RFC3339),
		UpdatedAt:   time.Now().Add(-2 * time.Second).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	claimed, err := store.ClaimNextRunnable(context.Background(), "worker-old", time.Now().Add(-time.Second).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("claim old job: %v", err)
	}
	if claimed.Status != domain.GatewayJobStatusProcessing {
		t.Fatalf("unexpected old claim: %#v", claimed)
	}
	taken, err := store.ClaimNextRunnable(context.Background(), "worker-new", time.Now().Add(time.Minute).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("take over expired lease: %v", err)
	}
	if taken.WorkerID != "worker-new" || taken.AttemptCount != 2 {
		t.Fatalf("unexpected takeover: %#v", taken)
	}
}

func TestDeleteModelMappingRemovesRoutes(t *testing.T) {
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
('claude-a', 'claude', 'https://api.anthropic.com', 'k1', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now');
INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-a', 'claude', 1, '', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	if err := DeleteModelMapping(context.Background(), db, 1); err != nil {
		t.Fatalf("delete model mapping: %v", err)
	}

	var modelCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM models`).Scan(&modelCount); err != nil {
		t.Fatalf("count models: %v", err)
	}
	if modelCount != 0 {
		t.Fatalf("expected 0 models, got %d", modelCount)
	}

	var routeCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM model_routes`).Scan(&routeCount); err != nil {
		t.Fatalf("count model_routes: %v", err)
	}
	if routeCount != 0 {
		t.Fatalf("expected 0 model routes, got %d", routeCount)
	}
}

func TestCreateModelRouteValidatesReferences(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	if _, err := CreateModelRoute(context.Background(), db, domain.CreateModelRouteInput{
		ModelAlias:     "gpt-4o",
		ChannelName:    "claude-a",
		InvocationMode: "claude",
		Priority:       1,
	}); err == nil || !strings.Contains(err.Error(), "模型别名不存在") {
		t.Fatalf("expected missing model alias error, got %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
INSERT INTO channels(name, provider, endpoint, api_key, enabled, created_at, updated_at) VALUES
('claude-a', 'claude', 'https://api.anthropic.com', 'k1', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed refs: %v", err)
	}

	if _, err := CreateModelRoute(context.Background(), db, domain.CreateModelRouteInput{
		ModelAlias:     "gpt-4o",
		ChannelName:    "claude-a",
		InvocationMode: "claude",
		Priority:       1,
	}); err != nil {
		t.Fatalf("create model route: %v", err)
	}

	if _, err := CreateModelRoute(context.Background(), db, domain.CreateModelRouteInput{
		ModelAlias:     "gpt-4o",
		ChannelName:    "claude-a",
		InvocationMode: "claude",
		Priority:       2,
	}); err == nil || !strings.Contains(err.Error(), "路由已存在") {
		t.Fatalf("expected duplicate route error, got %v", err)
	}
}

func TestCreateModelRouteValidatesFallbackConsistency(t *testing.T) {
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
('claude-b', 'claude', 'https://api.anthropic.com', 'k2', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now'),
('gpt-4o-fallback', 'claude-3-7-sonnet', 'now', 'now'),
('gpt-4o-fallback-2', 'claude-3-7-opus', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed refs: %v", err)
	}

	if _, err := CreateModelRoute(context.Background(), db, domain.CreateModelRouteInput{
		ModelAlias:     "gpt-4o",
		ChannelName:    "claude-a",
		InvocationMode: "claude",
		Priority:       1,
		FallbackModel:  "gpt-4o-fallback",
	}); err != nil {
		t.Fatalf("create route with fallback: %v", err)
	}

	if _, err := CreateModelRoute(context.Background(), db, domain.CreateModelRouteInput{
		ModelAlias:     "gpt-4o",
		ChannelName:    "claude-b",
		InvocationMode: "claude",
		Priority:       2,
		FallbackModel:  "gpt-4o-fallback-2",
	}); err == nil || !strings.Contains(err.Error(), "必须保持一致") {
		t.Fatalf("expected fallback consistency error, got %v", err)
	}
}

func TestUpdateModelRouteBindingUpdatesModelAndRouteAtomically(t *testing.T) {
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
('claude-b', 'claude', 'https://api.anthropic.com', 'k2', 1, 'now', 'now');
INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-3-5-sonnet', 'now', 'now'),
('gpt-4o-fallback', 'claude-3-7-sonnet', 'now', 'now');
INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES
('gpt-4o', 'claude-a', 'claude', 1, 'gpt-4o-fallback', 'now', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	if err := UpdateModelRouteBinding(context.Background(), db, 1, domain.UpdateModelRouteBindingInput{
		Alias:          "gpt-4.1",
		UpstreamModel:  "claude-3-7-opus",
		ChannelName:    "claude-b",
		InvocationMode: "claude",
		Priority:       2,
		FallbackModel:  "gpt-4o-fallback",
	}); err != nil {
		t.Fatalf("update model route binding: %v", err)
	}

	var alias string
	var upstreamModel string
	if err := db.QueryRowContext(context.Background(), `SELECT alias, upstream_model FROM models WHERE id = 1`).Scan(&alias, &upstreamModel); err != nil {
		t.Fatalf("query model: %v", err)
	}
	if alias != "gpt-4.1" || upstreamModel != "claude-3-7-opus" {
		t.Fatalf("unexpected model state: alias=%s upstream=%s", alias, upstreamModel)
	}

	var routeAlias string
	var routeChannel string
	var routePriority int
	if err := db.QueryRowContext(context.Background(), `SELECT model_alias, channel_name, priority FROM model_routes WHERE id = 1`).Scan(&routeAlias, &routeChannel, &routePriority); err != nil {
		t.Fatalf("query route: %v", err)
	}
	if routeAlias != "gpt-4.1" || routeChannel != "claude-b" || routePriority != 2 {
		t.Fatalf("unexpected route state: alias=%s channel=%s priority=%d", routeAlias, routeChannel, routePriority)
	}
}

func TestGetRoutingOverviewIncludesRecentCursorStates(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(context.Background(), `
INSERT INTO routing_cursors(route_key, next_index, updated_at) VALUES
('gpt-4o|openai|matched|1', 2, ?);
INSERT INTO request_logs(request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at) VALUES
('req-1', 'gpt-4o', 'claude-a', 200, 10, 1, 2, 3, 0, '{}', '{}', '{"log_type":"gateway_request"}', ?);`, now, now)
	if err != nil {
		t.Fatalf("seed overview: %v", err)
	}

	overview, err := GetRoutingOverview(context.Background(), db)
	if err != nil {
		t.Fatalf("get routing overview: %v", err)
	}
	if len(overview.CursorStates) != 1 {
		t.Fatalf("expected 1 cursor state, got %d", len(overview.CursorStates))
	}
	if overview.CursorStates[0].RouteKey != "gpt-4o|openai|matched|1" || overview.CursorStates[0].NextIndex != 2 {
		t.Fatalf("unexpected cursor state: %#v", overview.CursorStates[0])
	}
}
