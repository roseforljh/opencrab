package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"opencrab/internal/domain"
)

type DispatchRuntimeConfigStore struct {
	db *sql.DB
}

func NewDispatchRuntimeConfigStore(db *sql.DB) *DispatchRuntimeConfigStore {
	return &DispatchRuntimeConfigStore{db: db}
}

func (s *DispatchRuntimeConfigStore) GetDispatchRuntimeSettings(ctx context.Context) (domain.DispatchRuntimeSettings, error) {
	settings := domain.DispatchRuntimeSettings{
		RedisEnabled:      false,
		RedisAddress:      "127.0.0.1:6379",
		RedisDB:           0,
		RedisTLSEnabled:   false,
		RedisKeyPrefix:    "opencrab",
		RetryReserveRatio: 0.10,
		WorkerConcurrency: 1,
		PauseDispatch:     false,
		MaxAttempts:       5,
		BackoffDelayMs:    500,
		SyncHoldMs:        3000,
	}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM system_settings WHERE key IN (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"dispatch.redis_enabled",
		"dispatch.redis_address",
		"dispatch.redis_password",
		"dispatch.redis_db",
		"dispatch.redis_tls_enabled",
		"dispatch.redis_key_prefix",
		"dispatch.retry_reserve_ratio",
		"dispatch.worker_concurrency",
		"dispatch.pause_dispatch",
		"dispatch.max_attempts",
		"dispatch.backoff_delay_ms",
		"dispatch.sync_hold_ms",
	)
	if err != nil {
		return settings, fmt.Errorf("读取 dispatch runtime settings 失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, fmt.Errorf("读取 dispatch runtime setting 失败: %w", err)
		}
		switch key {
		case "dispatch.redis_enabled":
			settings.RedisEnabled = parseDispatchBool(value)
		case "dispatch.redis_address":
			if strings.TrimSpace(value) != "" {
				settings.RedisAddress = strings.TrimSpace(value)
			}
		case "dispatch.redis_password":
			settings.RedisPassword = value
		case "dispatch.redis_db":
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
				settings.RedisDB = parsed
			}
		case "dispatch.redis_tls_enabled":
			settings.RedisTLSEnabled = parseDispatchBool(value)
		case "dispatch.redis_key_prefix":
			if strings.TrimSpace(value) != "" {
				settings.RedisKeyPrefix = strings.TrimSpace(value)
			}
		case "dispatch.retry_reserve_ratio":
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed >= 0 && parsed <= 1 {
				settings.RetryReserveRatio = parsed
			}
		case "dispatch.worker_concurrency":
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
				settings.WorkerConcurrency = parsed
			}
		case "dispatch.pause_dispatch":
			settings.PauseDispatch = parseDispatchBool(value)
		case "dispatch.max_attempts":
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
				settings.MaxAttempts = parsed
			}
		case "dispatch.backoff_delay_ms":
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
				settings.BackoffDelayMs = parsed
			}
		case "dispatch.sync_hold_ms":
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
				settings.SyncHoldMs = parsed
			}
		}
	}
	if err := rows.Err(); err != nil {
		return settings, fmt.Errorf("遍历 dispatch runtime settings 失败: %w", err)
	}
	return settings, nil
}

func parseDispatchBool(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "true" || normalized == "1" || normalized == "enabled" || normalized == "启用"
}
