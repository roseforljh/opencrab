package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"opencrab/internal/domain"
)

type RequestLogStore struct {
	db *sql.DB
}

type GatewayAttemptLogStore struct {
	requestLogs *RequestLogStore
}

func NewRequestLogStore(db *sql.DB) *RequestLogStore {
	return &RequestLogStore{db: db}
}

func NewGatewayAttemptLogStore(db *sql.DB) *GatewayAttemptLogStore {
	return &GatewayAttemptLogStore{requestLogs: NewRequestLogStore(db)}
}

func (s *RequestLogStore) List(ctx context.Context) ([]domain.RequestLog, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at FROM request_logs ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 request_logs 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RequestLog, 0)
	for rows.Next() {
		var item domain.RequestLog
		var cacheHit int
		if err := rows.Scan(&item.ID, &item.RequestID, &item.Model, &item.Channel, &item.StatusCode, &item.LatencyMs, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &cacheHit, &item.RequestBody, &item.ResponseBody, &item.Details, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("读取 request_log 失败: %w", err)
		}
		item.CacheHit = cacheHit == 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 request_logs 失败: %w", err)
	}

	return items, nil
}

func (s *RequestLogStore) Create(ctx context.Context, item domain.RequestLog) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO request_logs(request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.RequestID,
		item.Model,
		item.Channel,
		item.StatusCode,
		item.LatencyMs,
		item.PromptTokens,
		item.CompletionTokens,
		item.TotalTokens,
		boolToInt(item.CacheHit),
		item.RequestBody,
		item.ResponseBody,
		item.Details,
		item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("写入 request_log 失败: %w", err)
	}
	return nil
}

func (s *RequestLogStore) Clear(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM request_logs`); err != nil {
		return fmt.Errorf("清空 request_logs 失败: %w", err)
	}
	return nil
}

func (s *GatewayAttemptLogStore) LogGatewayAttempt(ctx context.Context, item domain.GatewayAttemptLog) error {
	details, err := json.Marshal(map[string]any{
		"log_type":       "gateway_attempt",
		"provider":       item.Provider,
		"attempt":        item.Attempt,
		"retryable":      item.Retryable,
		"stream_started": item.StreamStarted,
		"success":        item.Success,
		"error_message":  item.ErrorMessage,
		"upstream_model": item.UpstreamModel,
	})
	if err != nil {
		return fmt.Errorf("序列化 attempt 日志失败: %w", err)
	}

	return s.requestLogs.Create(ctx, domain.RequestLog{
		RequestID:    item.RequestID,
		Model:        item.Model,
		Channel:      item.Channel,
		StatusCode:   item.StatusCode,
		RequestBody:  item.RequestBody,
		ResponseBody: item.ResponseBody,
		Details:      string(details),
		CreatedAt:    time.Now().Format(time.RFC3339),
	})
}
