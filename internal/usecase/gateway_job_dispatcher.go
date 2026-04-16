package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
)

type GatewayJobDispatcher struct {
	jobs     domain.GatewayJobStore
	settings domain.DispatchRuntimeConfigStore
	decode   func(job domain.GatewayJob) (domain.GatewayRequest, error)
	execute  func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error)
	client   *http.Client
}

func NewGatewayJobDispatcher(jobs domain.GatewayJobStore, settings domain.DispatchRuntimeConfigStore, decode func(job domain.GatewayJob) (domain.GatewayRequest, error), execute func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error), client *http.Client) *GatewayJobDispatcher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GatewayJobDispatcher{jobs: jobs, settings: settings, decode: decode, execute: execute, client: client}
}

func (d *GatewayJobDispatcher) Run(ctx context.Context, workerID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		worked, err := d.RunOnce(ctx, workerID)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if !worked {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
}

func (d *GatewayJobDispatcher) RunOnce(ctx context.Context, workerID string) (bool, error) {
	if d == nil || d.jobs == nil || d.settings == nil || d.decode == nil || d.execute == nil {
		return false, nil
	}
	settings, err := d.settings.GetDispatchRuntimeSettings(ctx)
	if err != nil {
		return false, err
	}
	if settings.PauseDispatch {
		return false, nil
	}
	leaseUntil := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	job, err := d.jobs.ClaimNextRunnable(ctx, workerID, leaseUntil)
	if err != nil {
		if strings.Contains(err.Error(), "请求不存在") {
			return false, nil
		}
		return false, err
	}
	req, err := d.decode(job)
	if err != nil {
		completedAt := time.Now().Format(time.RFC3339)
		if updateErr := d.jobs.UpdateStatusIfClaimMatches(ctx, job.RequestID, job.WorkerID, job.LeaseUntil, domain.GatewayJobStatusFailed, 0, "", err.Error(), completedAt); updateErr != nil {
			return true, updateErr
		}
		if webhookErr := d.deliverWebhook(ctx, job, domain.GatewayJobStatusFailed, 0, "", err.Error(), completedAt); webhookErr != nil {
			return true, webhookErr
		}
		return true, nil
	}
	result, execErr := d.execute(ctx, job.RequestID, req)
	if execErr == nil {
		statusCode := 0
		responseBody := ""
		if result != nil {
			if result.Response != nil {
				statusCode = result.Response.StatusCode
				responseBody = string(result.Response.Body)
			}
			if result.Stream != nil {
				statusCode = result.Stream.StatusCode
			}
		}
		completedAt := time.Now().Format(time.RFC3339)
		if err := d.jobs.UpdateStatusIfClaimMatches(ctx, job.RequestID, job.WorkerID, job.LeaseUntil, domain.GatewayJobStatusCompleted, statusCode, responseBody, "", completedAt); err != nil {
			return true, err
		}
		if err := d.deliverWebhook(ctx, job, domain.GatewayJobStatusCompleted, statusCode, responseBody, "", completedAt); err != nil {
			return true, err
		}
		return true, nil
	}
	execDetail := domain.AsExecutionError(execErr)
	if execDetail.Retryable && !execDetail.StreamStarted && job.AttemptCount < max(1, settings.MaxAttempts) {
		nextReadyAt := time.Now().Add(time.Duration(max(250, settings.BackoffDelayMs)) * time.Millisecond).Format(time.RFC3339)
		return true, d.jobs.Requeue(ctx, job.RequestID, job.WorkerID, job.LeaseUntil, execDetail.Error(), nextReadyAt)
	}
	statusCode := execDetail.StatusCode
	if statusCode == 0 {
		statusCode = 502
	}
	completedAt := time.Now().Format(time.RFC3339)
	if err := d.jobs.UpdateStatusIfClaimMatches(ctx, job.RequestID, job.WorkerID, job.LeaseUntil, domain.GatewayJobStatusFailed, statusCode, "", execDetail.Error(), completedAt); err != nil {
		return true, err
	}
	if err := d.deliverWebhook(ctx, job, domain.GatewayJobStatusFailed, statusCode, "", execDetail.Error(), completedAt); err != nil {
		return true, err
	}
	return true, nil
}
func WorkerID(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}

func (d *GatewayJobDispatcher) deliverWebhook(ctx context.Context, job domain.GatewayJob, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
	if d == nil || d.jobs == nil || strings.ToLower(strings.TrimSpace(job.DeliveryMode)) != "webhook" || strings.TrimSpace(job.WebhookURL) == "" {
		return nil
	}
	payload := map[string]any{
		"request_id":           job.RequestID,
		"status":               status,
		"response_status_code": responseStatusCode,
		"error_message":        errorMessage,
		"completed_at":         completedAt,
	}
	if strings.TrimSpace(responseBody) != "" {
		payload["result"] = json.RawMessage(responseBody)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码 webhook payload 失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.WebhookURL, strings.NewReader(string(encoded)))
	if err != nil {
		return fmt.Errorf("创建 webhook 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 webhook 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 返回异常状态: %d", resp.StatusCode)
	}
	return d.jobs.MarkWebhookDelivered(ctx, job.RequestID, time.Now().Format(time.RFC3339))
}
