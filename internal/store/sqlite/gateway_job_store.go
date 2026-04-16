package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"
)

type GatewayJobStore struct {
	db *sql.DB
}

var ErrGatewayJobConflict = errors.New("gateway job 冲突")

func NewGatewayJobStore(db *sql.DB) *GatewayJobStore {
	return &GatewayJobStore{db: db}
}

func (s *GatewayJobStore) Create(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO gateway_jobs(request_id, idempotency_key, owner_key_hash, request_hash, protocol, model, status, mode, request_path, request_body, request_headers, response_status_code, response_body, error_message, attempt_count, worker_id, session_id, lease_until, delivery_mode, webhook_url, webhook_delivered_at, accepted_at, completed_at, estimated_ready_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.RequestID,
		strings.TrimSpace(item.IdempotencyKey),
		strings.TrimSpace(item.OwnerKeyHash),
		strings.TrimSpace(item.RequestHash),
		string(item.Protocol),
		item.Model,
		string(item.Status),
		item.Mode,
		item.RequestPath,
		item.RequestBody,
		item.RequestHeaders,
		item.ResponseStatusCode,
		item.ResponseBody,
		item.ErrorMessage,
		item.AttemptCount,
		item.WorkerID,
		item.SessionID,
		item.LeaseUntil,
		item.DeliveryMode,
		item.WebhookURL,
		item.WebhookDeliveredAt,
		item.AcceptedAt,
		item.CompletedAt,
		item.EstimatedReadyAt,
		item.UpdatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.GatewayJob{}, ErrGatewayJobConflict
		}
		return domain.GatewayJob{}, fmt.Errorf("创建 gateway job 失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.GatewayJob{}, fmt.Errorf("读取 gateway job id 失败: %w", err)
	}
	item.ID = id
	return item, nil
}

func (s *GatewayJobStore) GetByRequestID(ctx context.Context, requestID string) (domain.GatewayJob, error) {
	return s.getOne(ctx, `SELECT id, request_id, idempotency_key, owner_key_hash, request_hash, protocol, model, status, mode, request_path, request_body, request_headers, response_status_code, response_body, error_message, attempt_count, worker_id, session_id, lease_until, delivery_mode, webhook_url, webhook_delivered_at, accepted_at, completed_at, estimated_ready_at, updated_at FROM gateway_jobs WHERE request_id = ? LIMIT 1`, requestID)
}

func (s *GatewayJobStore) GetByIdempotencyKey(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error) {
	return s.getOneWithArgs(ctx, `SELECT id, request_id, idempotency_key, owner_key_hash, request_hash, protocol, model, status, mode, request_path, request_body, request_headers, response_status_code, response_body, error_message, attempt_count, worker_id, session_id, lease_until, delivery_mode, webhook_url, webhook_delivered_at, accepted_at, completed_at, estimated_ready_at, updated_at FROM gateway_jobs WHERE owner_key_hash = ? AND idempotency_key = ? LIMIT 1`, strings.TrimSpace(ownerKeyHash), strings.TrimSpace(idempotencyKey))
}

func (s *GatewayJobStore) UpdateStatus(ctx context.Context, requestID string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE gateway_jobs SET status = ?, response_status_code = ?, response_body = ?, error_message = ?, completed_at = ?, worker_id = '', lease_until = '', updated_at = ? WHERE request_id = ?`, string(status), responseStatusCode, responseBody, errorMessage, completedAt, nowRFC3339(), requestID)
	if err != nil {
		return fmt.Errorf("更新 gateway job 状态失败: %w", err)
	}
	return nil
}

func (s *GatewayJobStore) UpdateStatusIfClaimMatches(ctx context.Context, requestID string, workerID string, leaseUntil string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE gateway_jobs SET status = ?, response_status_code = ?, response_body = ?, error_message = ?, completed_at = ?, worker_id = '', lease_until = '', updated_at = ? WHERE request_id = ? AND worker_id = ? AND lease_until = ?`, string(status), responseStatusCode, responseBody, errorMessage, completedAt, nowRFC3339(), requestID, workerID, leaseUntil)
	if err != nil {
		return fmt.Errorf("按 claim 更新 gateway job 状态失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取按 claim 更新结果失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("gateway job claim 已失效")
	}
	return nil
}

func (s *GatewayJobStore) ClaimNextRunnable(ctx context.Context, workerID string, leaseUntil string) (domain.GatewayJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.GatewayJob{}, fmt.Errorf("开启 gateway job claim 事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	var id int64
	now := nowRFC3339()
	err = tx.QueryRowContext(ctx, `SELECT id FROM gateway_jobs WHERE ((status IN (?, ?)) OR (status = ? AND lease_until <> '' AND lease_until <= ?)) AND (estimated_ready_at = '' OR estimated_ready_at <= ?) ORDER BY accepted_at ASC, id ASC LIMIT 1`, string(domain.GatewayJobStatusAccepted), string(domain.GatewayJobStatusQueued), string(domain.GatewayJobStatusProcessing), now, now).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.GatewayJob{}, fmt.Errorf("请求不存在")
		}
		return domain.GatewayJob{}, fmt.Errorf("查询可领取 gateway job 失败: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE gateway_jobs SET status = ?, worker_id = ?, lease_until = ?, attempt_count = attempt_count + 1, updated_at = ? WHERE id = ? AND ((status IN (?, ?)) OR (status = ? AND lease_until <> '' AND lease_until <= ?))`, string(domain.GatewayJobStatusProcessing), workerID, leaseUntil, now, id, string(domain.GatewayJobStatusAccepted), string(domain.GatewayJobStatusQueued), string(domain.GatewayJobStatusProcessing), now)
	if err != nil {
		return domain.GatewayJob{}, fmt.Errorf("领取 gateway job 失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.GatewayJob{}, fmt.Errorf("请求不存在")
	}
	var item domain.GatewayJob
	var protocol string
	var status string
	err = tx.QueryRowContext(ctx, `SELECT id, request_id, idempotency_key, owner_key_hash, request_hash, protocol, model, status, mode, request_path, request_body, request_headers, response_status_code, response_body, error_message, attempt_count, worker_id, session_id, lease_until, delivery_mode, webhook_url, webhook_delivered_at, accepted_at, completed_at, estimated_ready_at, updated_at FROM gateway_jobs WHERE id = ? LIMIT 1`, id).Scan(&item.ID, &item.RequestID, &item.IdempotencyKey, &item.OwnerKeyHash, &item.RequestHash, &protocol, &item.Model, &status, &item.Mode, &item.RequestPath, &item.RequestBody, &item.RequestHeaders, &item.ResponseStatusCode, &item.ResponseBody, &item.ErrorMessage, &item.AttemptCount, &item.WorkerID, &item.SessionID, &item.LeaseUntil, &item.DeliveryMode, &item.WebhookURL, &item.WebhookDeliveredAt, &item.AcceptedAt, &item.CompletedAt, &item.EstimatedReadyAt, &item.UpdatedAt)
	if err != nil {
		return domain.GatewayJob{}, fmt.Errorf("读取已领取 gateway job 失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.GatewayJob{}, fmt.Errorf("提交 gateway job claim 事务失败: %w", err)
	}
	item.Protocol = domain.Protocol(protocol)
	item.Status = domain.GatewayJobStatus(status)
	return item, nil
}

func (s *GatewayJobStore) Requeue(ctx context.Context, requestID string, workerID string, leaseUntil string, errorMessage string, nextReadyAt string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE gateway_jobs SET status = ?, error_message = ?, estimated_ready_at = ?, worker_id = '', lease_until = '', updated_at = ? WHERE request_id = ? AND worker_id = ? AND lease_until = ?`, string(domain.GatewayJobStatusQueued), errorMessage, nextReadyAt, nowRFC3339(), requestID, workerID, leaseUntil)
	if err != nil {
		return fmt.Errorf("重排 gateway job 失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取重排结果失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("gateway job claim 已失效")
	}
	return nil
}

func (s *GatewayJobStore) SetAcceptedReadyAt(ctx context.Context, requestID string, readyAt string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE gateway_jobs SET estimated_ready_at = ?, updated_at = ? WHERE request_id = ?`, readyAt, nowRFC3339(), requestID)
	if err != nil {
		return fmt.Errorf("更新 gateway job ready_at 失败: %w", err)
	}
	return nil
}

func (s *GatewayJobStore) MarkWebhookDelivered(ctx context.Context, requestID string, deliveredAt string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE gateway_jobs SET webhook_delivered_at = ?, updated_at = ? WHERE request_id = ?`, deliveredAt, nowRFC3339(), requestID)
	if err != nil {
		return fmt.Errorf("更新 webhook 交付时间失败: %w", err)
	}
	return nil
}

func (s *GatewayJobStore) getOne(ctx context.Context, query string, arg string) (domain.GatewayJob, error) {
	return s.getOneWithArgs(ctx, query, arg)
}

func (s *GatewayJobStore) getOneWithArgs(ctx context.Context, query string, args ...any) (domain.GatewayJob, error) {
	var item domain.GatewayJob
	var protocol string
	var status string
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&item.ID, &item.RequestID, &item.IdempotencyKey, &item.OwnerKeyHash, &item.RequestHash, &protocol, &item.Model, &status, &item.Mode, &item.RequestPath, &item.RequestBody, &item.RequestHeaders, &item.ResponseStatusCode, &item.ResponseBody, &item.ErrorMessage, &item.AttemptCount, &item.WorkerID, &item.SessionID, &item.LeaseUntil, &item.DeliveryMode, &item.WebhookURL, &item.WebhookDeliveredAt, &item.AcceptedAt, &item.CompletedAt, &item.EstimatedReadyAt, &item.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.GatewayJob{}, fmt.Errorf("请求不存在")
		}
		return domain.GatewayJob{}, fmt.Errorf("查询 gateway job 失败: %w", err)
	}
	item.Protocol = domain.Protocol(protocol)
	item.Status = domain.GatewayJobStatus(status)
	return item, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "constraint failed")
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
