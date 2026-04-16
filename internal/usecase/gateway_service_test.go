package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

type fakeRouteStore struct {
	routes []domain.GatewayRoute
	err    error
}

func (s fakeRouteStore) ListEnabledRoutesByModel(ctx context.Context, model string) ([]domain.GatewayRoute, error) {
	return s.routes, s.err
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

func (l *memoryAttemptLogger) LogGatewayAttempt(ctx context.Context, item domain.GatewayAttemptLog) error {
	l.items = append(l.items, item)
	return nil
}

func TestGatewayServiceRetryableFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("upstream 503"), 503, true, false)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok`)}}}
	logger := &memoryAttemptLogger{}
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "gpt-4o", UpstreamModel: "claude-model", Channel: domain.UpstreamChannel{Name: "claude-a", Provider: "claude"}, Priority: 1},
		{ModelAlias: "gpt-4o", UpstreamModel: "gemini-model", Channel: domain.UpstreamChannel{Name: "gemini-b", Provider: "gemini"}, Priority: 2},
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, logger, nil, nil)

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
	if len(logger.items) != 2 || logger.items[0].Retryable != true || logger.items[1].Success != true {
		t.Fatalf("unexpected logs: %#v", logger.items)
	}
}

func TestGatewayServiceNonRetryableDoesNotFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("bad request"), 400, false, false)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200}}}
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil)

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
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil)

	_, err := service.Execute(context.Background(), "req-3", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err == nil || err.Error() != "server down" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGatewayServiceStreamBoundaryDoesNotFallback(t *testing.T) {
	first := &fakeExecutor{err: domain.NewExecutionError(errors.New("stream interrupted"), 502, true, true)}
	second := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200}}}
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "gemini"}, Priority: 2},
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil, nil, nil)

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
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "claude"}, Priority: 1}}}, map[string]domain.Executor{"claude": executor}, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-stream", domain.GatewayRequest{Model: "m", Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Stream == nil || result.Stream.Headers["X-Opencrab-Channel"][0] != "c1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGatewayServicePrefersInvocationModeMatch(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	claudeExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-claude`)}}}
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u-openai", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u-claude", Channel: domain.UpstreamChannel{Name: "claude-b", Provider: "claude"}, InvocationMode: "claude", Priority: 2},
	}}, map[string]domain.Executor{"openai": openaiExecutor, "claude": claudeExecutor}, nil, nil, nil)

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

func TestGatewayServiceFallsBackToNeutralInvocationMode(t *testing.T) {
	openaiExecutor := &fakeExecutor{result: &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(`ok-openai`)}}}
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u-any", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}, InvocationMode: "", Priority: 1},
	}}, map[string]domain.Executor{"openai": openaiExecutor}, nil, nil, nil)

	result, err := service.Execute(context.Background(), "req-mode-neutral", domain.GatewayRequest{Protocol: domain.ProtocolClaude, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Response == nil || string(result.Response.Body) != "ok-openai" {
		t.Fatalf("unexpected response: %#v", result)
	}
}

func testGatewayMessage(role string, text string) domain.GatewayMessage {
	return domain.GatewayMessage{Role: role, Parts: []domain.UnifiedPart{{Type: "text", Text: text}}}
}

func TestGatewayServiceRoundRobinRotatesWithinPriorityTier(t *testing.T) {
	cursors := &memoryCursorStore{values: map[string]int{"m|openai|matched|1": 1}}
	callModels := make([]string, 0)
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
	}}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		callModels = append(callModels, input.UpstreamModel)
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.UpstreamModel)}}, nil
	})}, nil, fakeStrategyStore{strategy: domain.RoutingStrategyRoundRobin}, cursors)

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
	if cursors.values["m|openai|matched|1"] != 0 {
		t.Fatalf("unexpected cursor: %#v", cursors.values)
	}
}

func TestGatewayServiceRoundRobinFallsBackWithinTier(t *testing.T) {
	cursors := &memoryCursorStore{values: map[string]int{"m|openai|matched|1": 0}}
	callModels := make([]string, 0)
	service := NewGatewayService(fakeRouteStore{routes: []domain.GatewayRoute{
		{ModelAlias: "m", UpstreamModel: "u1", Channel: domain.UpstreamChannel{Name: "c1", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
		{ModelAlias: "m", UpstreamModel: "u2", Channel: domain.UpstreamChannel{Name: "c2", Provider: "openai"}, InvocationMode: "openai", Priority: 1},
	}}, map[string]domain.Executor{"openai": executorFunc(func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
		callModels = append(callModels, input.UpstreamModel)
		if input.UpstreamModel == "u1" {
			return nil, domain.NewExecutionError(errors.New("retry"), 503, true, false)
		}
		return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Headers: map[string][]string{}, Body: []byte(input.UpstreamModel)}}, nil
	})}, nil, fakeStrategyStore{strategy: domain.RoutingStrategyRoundRobin}, cursors)

	result, err := service.Execute(context.Background(), "req-rr-fallback", domain.GatewayRequest{Protocol: domain.ProtocolOpenAI, Model: "m", Messages: []domain.GatewayMessage{testGatewayMessage("user", "x")}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(result.Response.Body) != "u2" {
		t.Fatalf("unexpected response: %#v", result)
	}
	if len(callModels) != 2 || callModels[0] != "u1" || callModels[1] != "u2" {
		t.Fatalf("unexpected call models: %#v", callModels)
	}
	if cursors.values["m|openai|matched|1"] != 0 {
		t.Fatalf("unexpected cursor: %#v", cursors.values)
	}
}

type executorFunc func(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error)

func (f executorFunc) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	return f(ctx, input)
}
