package usecase

import (
	"context"
	"errors"
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
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, logger)

	resp, err := service.Execute(context.Background(), "req-1", domain.GatewayRequest{Model: "gpt-4o", Messages: []domain.GatewayMessage{{Role: "user", Text: "hello"}}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Headers["X-Opencrab-Channel"][0] != "gemini-b" {
		t.Fatalf("unexpected channel header: %#v", resp.Headers)
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
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil)

	_, err := service.Execute(context.Background(), "req-2", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{{Role: "user", Text: "x"}}})
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
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil)

	_, err := service.Execute(context.Background(), "req-3", domain.GatewayRequest{Model: "m", Messages: []domain.GatewayMessage{{Role: "user", Text: "x"}}})
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
	}}, map[string]domain.Executor{"claude": first, "gemini": second}, nil)

	_, err := service.Execute(context.Background(), "req-4", domain.GatewayRequest{Model: "m", Stream: true, Messages: []domain.GatewayMessage{{Role: "user", Text: "x"}}})
	if err == nil || err.Error() != "stream interrupted" {
		t.Fatalf("unexpected err: %v", err)
	}
	if second.calls != 0 {
		t.Fatalf("expected no fallback after stream start")
	}
}
