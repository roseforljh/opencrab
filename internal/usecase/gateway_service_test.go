package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
)

type fakeRouteStore struct {
	routes []domain.GatewayRoute
	err    error
}

func (s fakeRouteStore) ListEnabledRoutesByModel(ctx context.Context, model string) ([]domain.GatewayRoute, error) {
	filtered := make([]domain.GatewayRoute, 0)
	for _, route := range s.routes {
		if route.ModelAlias == model {
			filtered = append(filtered, route)
		}
	}
	return filtered, s.err
}

type fakeExecutor struct {
	result *domain.ExecutionResult
	err    error
	calls  int
}

func (e *fakeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	e.calls++
	return e.result, e.err
}

type memoryAttemptLogger struct {
	items []domain.GatewayAttemptLog
}

func (l *memoryAttemptLogger) LogGatewayAttempt(ctx context.Context, item domain.GatewayAttemptLog) error {
	l.items = append(l.items, item)
	return nil
}

type fakeStrategyStore struct {
	strategy domain.RoutingStrategy
	err      error
}

func (s fakeStrategyStore) GetRoutingStrategy(ctx context.Context) (domain.RoutingStrategy, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.strategy == "" {
		return domain.RoutingStrategySequential, nil
	}
	return s.strategy, nil
}

type memoryCursorStore struct {
	values  map[string]int
	updates map[string]int
}

func (s *memoryCursorStore) GetRoutingCursor(ctx context.Context, routeKey string) (int, error) {
	if s.values == nil {
		return 0, nil
	}
	return s.values[routeKey], nil
}

func (s *memoryCursorStore) AdvanceRoutingCursor(ctx context.Context, routeKey string, candidateCount int, selectedIndex int) error {
	if s.values == nil {
		s.values = map[string]int{}
	}
	if s.updates == nil {
		s.updates = map[string]int{}
	}
	s.values[routeKey] = (selectedIndex + 1) % candidateCount
	s.updates[routeKey]++
	return nil
}

type fakeRuntimeConfigStore struct {
	settings domain.GatewayRuntimeSettings
	err      error
}

func (s fakeRuntimeConfigStore) GetGatewayRuntimeSettings(ctx context.Context) (domain.GatewayRuntimeSettings, error) {
	if s.err != nil {
		return domain.GatewayRuntimeSettings{}, s.err
	}
	if s.settings.CooldownDuration == 0 {
		s.settings.CooldownDuration = 45 * time.Second
	}
	if s.settings.StickyKeySource == "" {
		s.settings.StickyKeySource = "auto"
	}
	return s.settings, nil
}

type memoryRuntimeStateStore struct {
	cooldowns map[int64]string
}

func (s *memoryRuntimeStateStore) MarkCooldown(ctx context.Context, routeID int64, duration time.Duration, lastError string) (string, error) {
	if s.cooldowns == nil {
		s.cooldowns = map[int64]string{}
	}
	until := time.Now().Add(duration).Format(time.RFC3339)
	s.cooldowns[routeID] = until
	return until, nil
}

func (s *memoryRuntimeStateStore) ClearCooldown(ctx context.Context, routeID int64) error {
	if s.cooldowns != nil {
		delete(s.cooldowns, routeID)
	}
	return nil
}

func (s *memoryRuntimeStateStore) CountActiveCooldowns(ctx context.Context) (int, error) {
	return len(s.cooldowns), nil
}

type memoryStickyStore struct {
	bindings map[string]int64
}

type fakeQuotaManager struct {
	result   domain.DispatchReservationResult
	err      error
	calls    int
	releases []domain.DispatchReleaseInput
}

func (m *fakeQuotaManager) Reserve(ctx context.Context, input domain.DispatchReservationInput) (domain.DispatchReservationResult, error) {
	m.calls++
	if m.err != nil {
		return domain.DispatchReservationResult{}, m.err
	}
	if m.result.ChannelName == "" {
		m.result = domain.DispatchReservationResult{ChannelName: input.ChannelName, LeaseAcquired: true, Runtime: "test"}
	}
	return m.result, nil
}

func (m *fakeQuotaManager) Release(ctx context.Context, input domain.DispatchReleaseInput) error {
	m.releases = append(m.releases, input)
	return nil
}

func stickyKey(affinityKey string, modelAlias string, protocol domain.Protocol) string {
	return affinityKey + "|" + modelAlias + "|" + string(protocol)
}

func (s *memoryStickyStore) GetStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol domain.Protocol) (int64, bool, error) {
	if s.bindings == nil {
		return 0, false, nil
	}
	routeID, ok := s.bindings[stickyKey(affinityKey, modelAlias, protocol)]
	return routeID, ok, nil
}

func (s *memoryStickyStore) UpsertStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol domain.Protocol, routeID int64) error {
	if s.bindings == nil {
		s.bindings = map[string]int64{}
	}
	s.bindings[stickyKey(affinityKey, modelAlias, protocol)] = routeID
	return nil
}

func (s *memoryStickyStore) CountStickyBindings(ctx context.Context) (int, error) {
	return len(s.bindings), nil
}

func newGatewayServiceForTest(routes []domain.GatewayRoute, executors map[string]domain.Executor, logger domain.GatewayAttemptLogger, strategy domain.RoutingStrategyStore, cursors domain.RoutingCursorStore, runtimeStates domain.RoutingRuntimeStateStore, sticky domain.StickyRoutingStore) *GatewayService {
	return NewGatewayService(fakeRouteStore{routes: routes}, executors, logger, nil, strategy, cursors, fakeRuntimeConfigStore{settings: domain.GatewayRuntimeSettings{CooldownDuration: 45 * time.Second, StickyEnabled: true, StickyKeySource: "auto"}}, runtimeStates, sticky, capability.NewRegistry(nil))
}

func TestGatewayServiceRetryableFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("upstream 503"), 503, true, false)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok`)}}}
	logger := &memoryAttemptLogger{}
	runtimeStates := &memoryRuntimeStateStore{}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "gpt-4o", UpstreamModel: "claude-model", Channel: domain.UpstreamChannel{Name: "claude-a", Provider: "claude"}, Priority: 1},
		{ID: 2, ModelAlias: "gpt-4o", UpstreamModel: "gemini-model", Channel: domain.UpstreamChannel{Name: "gemini-b", Provider: "gemini"}, Priority: 2},
	}, map[string]domain.Executor{"claude": first, "gemini": second}, logger, nil, nil, runtimeStates, nil)

	resp, err := service.Execute(context.Background(), "req-1", domain.GatewayRequest{Model: "gpt-4o", Messages: []domain.GatewayMessage{testGatewayMessage("user", "hello")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Response == nil || resp.Response.Headers["X-Opencrab-Channel"][0] != "gemini-b" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if first.calls != 1 || second.calls != 1 {
		t.Fatalf("unexpected calls: first=%d second=%d", first.calls, second.calls)
	}
	if len(logger.items) != 2 || !logger.items[0].Retryable || !logger.items[1].Success {
		t.Fatalf("unexpected logs: %#v", logger.items)
	}
	if _, ok := runtimeStates.cooldowns[1]; !ok {
		t.Fatalf("expected cooldown to be written")
	}
}

func TestGatewayServiceNonRetryableDoesNotFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("bad request"), 400, false, false)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil, nil, nil)

	_, err := service.Execute(context.Background(), "req-2", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if first.calls != 1 || second.calls != 0 {
		t.Fatalf("unexpected calls: first=%d second=%d", first.calls, second.calls)
	}
}

func TestGatewayServiceAllAttemptsFailed(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("rate limit"), 429, true, false)}
	second := &fakeExecutor{err: domain.NewExecutionError(errors.New("server down"), 503, true, false)}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil, nil, nil)

	_, err := service.Execute(context.Background(), "req-3", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err == nil || err.Error() != "server down" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGatewayServiceStreamBoundaryDoesNotFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("stream interrupted"), 502, true, true)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil, nil, nil)

	_, err := service.Execute(context.Background(), "req-4", domain.GatewayRequest{Model: "m", Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err == nil || err.Error() != "stream interrupted" {
		t.Fatalf("unexpected err: %v", err)
	}
	if second.calls != 0 {
		t.Fatalf("expected no fallback after stream start")
	}
}

func TestGatewayServiceReturnsStreamResult(t *testing.T) {
	stream := io.NopCloser(strings.NewReader("data: hi\n\n"))
	executor := &fakeExecutor{result: &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: 200, Headers: map[string][]string{}, Body: stream}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1}}, map[string]domain.Executor{"claude": executor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-stream", domain.GatewayRequest{Model: "m", Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Stream == nil || result.Stream.Headers["X-Opencrab-Channel"][0] != "c1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGatewayServiceUsesPreloadedRuntimeSettings(t *testing.T) {
	executor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok`)}}}
	service := NewGatewayService(
		fakeRouteStore{routes: []domain.GatewayRoute{{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1}}},
		map[string]domain.Executor{"claude": executor},
		nil,
		nil,
		nil,
		nil,
		fakeRuntimeConfigStore{err: errors.New("runtime settings should not be read")},
		nil,
		nil,
		capability.NewRegistry(nil),
	)

	result, err := service.Execute(context.Background(), "req-runtime", domain.GatewayRequest{
		Model:           "m",
		Messages:        []domain.GatewayMessage{testGatewayMessage("user", "x")},
		RuntimeSettings: &domain.GatewayRuntimeSettings{CooldownDuration: time.Second, StickyEnabled: true, StickyKeySource: "auto"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Response == nil || executor.calls != 1 {
		t.Fatalf("unexpected result: %#v calls=%d", result, executor.calls)
	}
}

func TestGatewayServiceQuotaWaitSkipsExecutor(t *testing.T) {
	executor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok`)}}}
	quota := &fakeQuotaManager{result: domain.DispatchReservationResult{ChannelName: "c1", LeaseAcquired: false, WaitMs: 1500, Runtime: "redis"}}
	service := NewGatewayService(
		fakeRouteStore{routes: []domain.GatewayRoute{{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude", RPMLimit: 1000, MaxInflight: 16, SafetyFactor: 0.9}, Priority: 1}}},
		map[string]domain.Executor{"claude": executor},
		nil,
		quota,
		nil,
		nil,
		fakeRuntimeConfigStore{settings: domain.GatewayRuntimeSettings{CooldownDuration: time.Second, StickyEnabled: true, StickyKeySource: "auto"}},
		nil,
		nil,
		capability.NewRegistry(nil),
	)

	_, err := service.Execute(context.Background(), "req-quota", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "quota")}})
	if err == nil || !strings.Contains(err.Error(), "渠道额度预约中") {
		t.Fatalf("unexpected error: %v", err)
	}
	if executor.calls != 0 || quota.calls != 1 {
		t.Fatalf("unexpected calls: executor=%d quota=%d", executor.calls, quota.calls)
	}
}

func TestGatewayServiceReleasesQuotaAfterSuccess(t *testing.T) {
	executor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok`)}}}
	quota := &fakeQuotaManager{result: domain.DispatchReservationResult{ChannelName: "c1", ReservationKey: "lease-1", LeaseAcquired: true, Runtime: "redis"}}
	service := NewGatewayService(
		fakeRouteStore{routes: []domain.GatewayRoute{{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude", RPMLimit: 1000, MaxInflight: 16, SafetyFactor: 0.9}, Priority: 1}}},
		map[string]domain.Executor{"claude": executor},
		nil,
		quota,
		nil,
		nil,
		fakeRuntimeConfigStore{settings: domain.GatewayRuntimeSettings{CooldownDuration: time.Second, StickyEnabled: true, StickyKeySource: "auto"}},
		nil,
		nil,
		capability.NewRegistry(nil),
	)

	_, err := service.Execute(context.Background(), "req-release", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "ok")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(quota.releases) != 1 || quota.releases[0].ReservationKey != "lease-1" {
		t.Fatalf("unexpected releases: %#v", quota.releases)
	}
}

func TestGatewayServicePrefersInvocationModeMatch(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, InvocationMode: "claude", Priority: 2},
	}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-mode", domain.GatewayRequest{Protocol: domain.ProtocolClaude, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Response == nil || string(result.Response.Body) != "ok-claude" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if claudeExecutor.calls != 1 || openaiExecutor.calls != 0 {
		t.Fatalf("unexpected calls: openai=%d claude=%d", openaiExecutor.calls, claudeExecutor.calls)
	}
}

func TestGatewayServiceRoundRobinRotatesWithinPriorityTier(t *testing.T) {
	cursors := &memoryCursorStore{values: map[string]int{"m|openai|matched|1": 1}}
	callModels := make([]string, 0)
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
	}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		callModels = append(callModels, input.UpstreamModel)
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.UpstreamModel)}}, nil
	})}, nil, fakeStrategyStore{strategy: domain.RoutingStrategyRoundRobin}, cursors, nil, nil)

	result, err := service.Execute(context.Background(), "req-rr", domain.GatewayRequest{Protocol: domain.ProtocolOpenAI, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "u2" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if len(callModels) != 1 || callModels[0] != "u2" {
		t.Fatalf("unexpected call models: %#v", callModels)
	}
}

func TestGatewayServiceCooldownSkipsRoute(t *testing.T) {
	logger := &memoryAttemptLogger{}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, Priority: 1, CooldownUntil: time.Now().Add(time.Minute).Format(time.RFC3339), LastError: "retry later"},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, Priority: 2},
	}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.UpstreamModel)}}, nil
	})}, logger, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-cooldown", domain.GatewayRequest{Protocol: domain.ProtocolOpenAI, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "u2" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if len(logger.items) < 2 || logger.items[0].SkipReason != "cooldown" {
		t.Fatalf("expected cooldown skip log, got %#v", logger.items)
	}
}

func TestGatewayServiceFallbackAliasReentry(t *testing.T) {
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, Priority: 1, FallbackModel: "m-fallback"},
		{ID: 2, ModelAlias: "m-fallback", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, Priority: 1},
	}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		if input.Request.Model == "m" {
			return nil, domain.NewExecutionError(errors.New("retry"), 503, true, false)
		}
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.Request.Model)}}, nil
	})}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-fallback", domain.GatewayRequest{Protocol: domain.ProtocolOpenAI, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "m-fallback" {
		t.Fatalf("unexpected response body: %s", string(result.Response.Body))
	}
	if len(result.Metadata.FallbackChain) != 1 || result.Metadata.FallbackChain[0] != "m-fallback" {
		t.Fatalf("unexpected fallback chain: %#v", result.Metadata)
	}
}

func TestGatewayServiceStickyReordersFirstTier(t *testing.T) {
	sticky := &memoryStickyStore{bindings: map[string]int64{stickyKey("session-1", "m", domain.ProtocolOpenAI): 2}}
	callModels := make([]string, 0)
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, Priority: 1},
	}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		callModels = append(callModels, input.UpstreamModel)
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.UpstreamModel)}}, nil
	})}, nil, fakeStrategyStore{strategy: domain.RoutingStrategyRoundRobin}, &memoryCursorStore{}, nil, sticky)

	result, err := service.Execute(context.Background(), "req-sticky", domain.GatewayRequest{Protocol: domain.ProtocolOpenAI, Model: "m", AffinityKey: "session-1", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "u2" || len(callModels) == 0 || callModels[0] != "u2" {
		t.Fatalf("unexpected sticky execution: result=%#v calls=%#v", result, callModels)
	}
	if !result.Metadata.StickyHit {
		t.Fatalf("expected sticky hit metadata, got %#v", result.Metadata)
	}
}

func TestGatewayServiceToolsCanBridgeToCompatibleProvider(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, Priority: 2},
	}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-tools", domain.GatewayRequest{
		Protocol: domain.ProtocolClaude,
		Model:    "m",
		Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")},
		Tools:    []json.RawMessage{json.RawMessage(`{"name":"opencode","input_schema":{"type":"object"}}`)},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "ok-openai" {
		t.Fatalf("unexpected response body: %s metadata=%#v", string(result.Response.Body), result.Metadata)
	}
	if claudeExecutor.calls != 0 || openaiExecutor.calls != 1 {
		t.Fatalf("unexpected executor calls: openai=%d claude=%d", openaiExecutor.calls, claudeExecutor.calls)
	}
}

func TestGatewayServiceClaudeNativeHeadersRequireClaudeProvider(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, Priority: 2},
	}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-claude-native", domain.GatewayRequest{
		Protocol: domain.ProtocolClaude,
		Model:    "m",
		Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")},
		RequestHeaders: map[string]string{
			"anthropic-dangerous-direct-browser-access": "true",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "ok-claude" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if claudeExecutor.calls != 1 || openaiExecutor.calls != 0 {
		t.Fatalf("unexpected executor calls: openai=%d claude=%d", openaiExecutor.calls, claudeExecutor.calls)
	}
}

func TestGatewayServiceClaudeNativeMetadataRequireClaudeProvider(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, Priority: 2},
	}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-claude-thinking", domain.GatewayRequest{
		Protocol:  domain.ProtocolClaude,
		Model:     "m",
		Operation: domain.ProtocolOperationClaudeMessages,
		Messages:  []domain.GatewayMessage{testGatewayMessage("user", "x")},
		Metadata: map[string]json.RawMessage{
			"thinking": json.RawMessage(`{"type":"enabled","budget_tokens":1024}`),
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "ok-claude" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if claudeExecutor.calls != 1 || openaiExecutor.calls != 0 {
		t.Fatalf("unexpected executor calls: openai=%d claude=%d", openaiExecutor.calls, claudeExecutor.calls)
	}
}

func TestGatewayServiceResponsesSessionRequiresOpenAICompatibleRoute(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, Priority: 1},
		{ID: 2, ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, Priority: 2},
	}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-responses-session", domain.GatewayRequest{
		Protocol:  domain.ProtocolOpenAI,
		Operation: domain.ProtocolOperationOpenAIResponses,
		Model:     "m",
		Messages:  []domain.GatewayMessage{testGatewayMessage("user", "x")},
		Session: &domain.GatewaySessionState{
			PreviousResponseID: "resp_123",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "ok-openai" {
		t.Fatalf("unexpected response body: %s metadata=%#v", string(result.Response.Body), result.Metadata)
	}
	if claudeExecutor.calls != 0 || openaiExecutor.calls != 1 {
		t.Fatalf("unexpected executor calls: openai=%d claude=%d", openaiExecutor.calls, claudeExecutor.calls)
	}
}

func TestGatewayServiceBasicResponsesRequestCanBridgeToClaude(t *testing.T) {
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := newGatewayServiceForTest([]domain.GatewayRoute{
		{ID: 1, ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, Priority: 1},
	}, map[string]domain.Executor{"claude": claudeExecutor}, nil, nil, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-responses-bridge", domain.GatewayRequest{
		Protocol:  domain.ProtocolOpenAI,
		Operation: domain.ProtocolOperationOpenAIResponses,
		Model:     "m",
		Messages:  []domain.GatewayMessage{testGatewayMessage("user", "x")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "ok-claude" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if claudeExecutor.calls != 1 {
		t.Fatalf("unexpected executor calls: claude=%d", claudeExecutor.calls)
	}
}

func testGatewayMessage(role string, text string) domain.GatewayMessage {
	return domain.GatewayMessage{Role: role, Parts: []domain.UnifiedPart{{Type: "text", Text: text}}}
}

type executorFunc func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error)

func (f executorFunc) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	return f(ctx, input)
}
