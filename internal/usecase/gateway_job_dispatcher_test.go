package usecase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opencrab/internal/domain"
)

type memoryJobStore struct {
	job domain.GatewayJob
}

func (s *memoryJobStore) Create(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
	s.job = item
	return item, nil
}
func (s *memoryJobStore) GetByRequestID(ctx context.Context, requestID string) (domain.GatewayJob, error) {
	return s.job, nil
}
func (s *memoryJobStore) GetByIdempotencyKey(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error) {
	return s.job, nil
}
func (s *memoryJobStore) UpdateStatus(ctx context.Context, requestID string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
	s.job.Status = status
	s.job.ResponseStatusCode = responseStatusCode
	s.job.ResponseBody = responseBody
	s.job.ErrorMessage = errorMessage
	s.job.CompletedAt = completedAt
	s.job.WorkerID = ""
	s.job.LeaseUntil = ""
	return nil
}
func (s *memoryJobStore) UpdateStatusIfClaimMatches(ctx context.Context, requestID string, workerID string, leaseUntil string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
	if s.job.WorkerID != workerID || s.job.LeaseUntil != leaseUntil {
		return errors.New("gateway job claim 已失效")
	}
	return s.UpdateStatus(ctx, requestID, status, responseStatusCode, responseBody, errorMessage, completedAt)
}
func (s *memoryJobStore) ClaimNextRunnable(ctx context.Context, workerID string, leaseUntil string) (domain.GatewayJob, error) {
	if s.job.RequestID == "" || (s.job.Status != domain.GatewayJobStatusAccepted && s.job.Status != domain.GatewayJobStatusQueued) {
		return domain.GatewayJob{}, errors.New("请求不存在")
	}
	s.job.Status = domain.GatewayJobStatusProcessing
	s.job.WorkerID = workerID
	s.job.LeaseUntil = leaseUntil
	s.job.AttemptCount++
	return s.job, nil
}
func (s *memoryJobStore) Requeue(ctx context.Context, requestID string, workerID string, leaseUntil string, errorMessage string, nextReadyAt string) error {
	if s.job.WorkerID != workerID || s.job.LeaseUntil != leaseUntil {
		return errors.New("gateway job claim 已失效")
	}
	s.job.Status = domain.GatewayJobStatusQueued
	s.job.ErrorMessage = errorMessage
	s.job.EstimatedReadyAt = nextReadyAt
	s.job.WorkerID = ""
	s.job.LeaseUntil = ""
	return nil
}
func (s *memoryJobStore) MarkWebhookDelivered(ctx context.Context, requestID string, deliveredAt string) error {
	s.job.WebhookDeliveredAt = deliveredAt
	return nil
}

func TestGatewayJobDispatcherRunOnceCompletesJob(t *testing.T) {
	jobs := &memoryJobStore{job: domain.GatewayJob{RequestID: "req-1", Status: domain.GatewayJobStatusAccepted, Protocol: domain.ProtocolOpenAI, RequestPath: "/v1/responses", RequestBody: `{}`, AcceptedAt: time.Now().Format(time.RFC3339)}}
	dispatcher := NewGatewayJobDispatcher(
		jobs,
		fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{WorkerConcurrency: 1, MaxAttempts: 5, BackoffDelayMs: 500}},
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return domain.GatewayRequest{Model: "m"}, nil
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Body: []byte(`ok`)}}, nil
		},
		nil,
	)
	worked, err := dispatcher.RunOnce(context.Background(), "worker-1")
	if err != nil || !worked {
		t.Fatalf("run once: worked=%v err=%v", worked, err)
	}
	if jobs.job.Status != domain.GatewayJobStatusCompleted || jobs.job.ResponseStatusCode != 200 {
		t.Fatalf("unexpected job after complete: %#v", jobs.job)
	}
}

func TestGatewayJobDispatcherRunOnceRequeuesRetryable(t *testing.T) {
	jobs := &memoryJobStore{job: domain.GatewayJob{RequestID: "req-2", Status: domain.GatewayJobStatusAccepted, Protocol: domain.ProtocolOpenAI, RequestPath: "/v1/responses", RequestBody: `{}`, AcceptedAt: time.Now().Format(time.RFC3339)}}
	dispatcher := NewGatewayJobDispatcher(
		jobs,
		fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{WorkerConcurrency: 1, MaxAttempts: 5, BackoffDelayMs: 500}},
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return domain.GatewayRequest{Model: "m"}, nil
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return nil, domain.NewExecutionError(errors.New("upstream 503"), 503, true, false)
		},
		nil,
	)
	worked, err := dispatcher.RunOnce(context.Background(), "worker-1")
	if err != nil || !worked {
		t.Fatalf("run once: worked=%v err=%v", worked, err)
	}
	if jobs.job.Status != domain.GatewayJobStatusQueued || jobs.job.ErrorMessage == "" {
		t.Fatalf("unexpected job after requeue: %#v", jobs.job)
	}
}

func TestGatewayJobDispatcherRejectsStaleWorkerWriteback(t *testing.T) {
	jobs := &memoryJobStore{job: domain.GatewayJob{RequestID: "req-stale", Status: domain.GatewayJobStatusAccepted, Protocol: domain.ProtocolOpenAI, RequestPath: "/v1/responses", RequestBody: `{}`, AcceptedAt: time.Now().Format(time.RFC3339)}}
	dispatcher := NewGatewayJobDispatcher(
		jobs,
		fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{WorkerConcurrency: 1, MaxAttempts: 5, BackoffDelayMs: 500}},
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return domain.GatewayRequest{Model: "m"}, nil
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			jobs.job.WorkerID = "other-worker"
			jobs.job.LeaseUntil = time.Now().Add(time.Minute).Format(time.RFC3339)
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Body: []byte(`ok`)}}, nil
		},
		nil,
	)
	worked, err := dispatcher.RunOnce(context.Background(), "worker-1")
	if !worked || err == nil || !strings.Contains(err.Error(), "claim 已失效") {
		t.Fatalf("unexpected stale worker result: worked=%v err=%v", worked, err)
	}
}

func TestGatewayJobDispatcherDeliversWebhook(t *testing.T) {
	var deliveredBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		deliveredBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	jobs := &memoryJobStore{job: domain.GatewayJob{RequestID: "req-webhook", Status: domain.GatewayJobStatusAccepted, Protocol: domain.ProtocolOpenAI, RequestPath: "/v1/responses", RequestBody: `{}`, DeliveryMode: "webhook", WebhookURL: server.URL, AcceptedAt: time.Now().Format(time.RFC3339)}}
	dispatcher := NewGatewayJobDispatcher(
		jobs,
		fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{WorkerConcurrency: 1, MaxAttempts: 5, BackoffDelayMs: 500}},
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return domain.GatewayRequest{Model: "m"}, nil
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: 200, Body: []byte(`{"id":"resp_1"}`)}}, nil
		},
		server.Client(),
	)
	worked, err := dispatcher.RunOnce(context.Background(), "worker-1")
	if err != nil || !worked {
		t.Fatalf("run once: worked=%v err=%v", worked, err)
	}
	if jobs.job.WebhookDeliveredAt == "" || !strings.Contains(deliveredBody, `"request_id":"req-webhook"`) {
		t.Fatalf("unexpected webhook delivery: deliveredAt=%q body=%s", jobs.job.WebhookDeliveredAt, deliveredBody)
	}
}

func TestGatewayJobDispatcherDecodeFailureStillDeliversWebhook(t *testing.T) {
	var deliveredBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		deliveredBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	jobs := &memoryJobStore{job: domain.GatewayJob{RequestID: "req-webhook-failed", Status: domain.GatewayJobStatusAccepted, Protocol: domain.ProtocolOpenAI, RequestPath: "/v1/responses", RequestBody: `{}`, DeliveryMode: "webhook", WebhookURL: server.URL, AcceptedAt: time.Now().Format(time.RFC3339)}}
	dispatcher := NewGatewayJobDispatcher(
		jobs,
		fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{WorkerConcurrency: 1, MaxAttempts: 5, BackoffDelayMs: 500}},
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return domain.GatewayRequest{}, errors.New("decode failed")
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return nil, nil
		},
		server.Client(),
	)
	worked, err := dispatcher.RunOnce(context.Background(), "worker-1")
	if err != nil || !worked {
		t.Fatalf("run once: worked=%v err=%v", worked, err)
	}
	if jobs.job.Status != domain.GatewayJobStatusFailed || jobs.job.WebhookDeliveredAt == "" || !strings.Contains(deliveredBody, `"status":"failed"`) {
		t.Fatalf("unexpected failed webhook delivery: job=%#v body=%s", jobs.job, deliveredBody)
	}
}
