package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"
)

func ListChannels(ctx context.Context, db *sql.DB) ([]domain.Channel, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, provider, endpoint, enabled, updated_at FROM channels ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 channels 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Channel, 0)
	for rows.Next() {
		var item domain.Channel
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Provider, &item.Endpoint, &enabled, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取 channel 失败: %w", err)
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 channels 失败: %w", err)
	}

	return items, nil
}

func ListAPIKeys(ctx context.Context, db *sql.DB) ([]domain.APIKey, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, enabled FROM api_keys ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 api_keys 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.APIKey, 0)
	for rows.Next() {
		var item domain.APIKey
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &enabled); err != nil {
			return nil, fmt.Errorf("读取 api_key 失败: %w", err)
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 api_keys 失败: %w", err)
	}

	return items, nil
}

func ListModelMappings(ctx context.Context, db *sql.DB) ([]domain.ModelMapping, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, alias, upstream_model FROM models ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 models 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ModelMapping, 0)
	for rows.Next() {
		var item domain.ModelMapping
		if err := rows.Scan(&item.ID, &item.Alias, &item.UpstreamModel); err != nil {
			return nil, fmt.Errorf("读取 model 失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 models 失败: %w", err)
	}

	return items, nil
}

func ListModelRoutes(ctx context.Context, db *sql.DB) ([]domain.ModelRoute, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, model_alias, channel_name, priority, fallback_model FROM model_routes ORDER BY priority ASC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 model_routes 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ModelRoute, 0)
	for rows.Next() {
		var item domain.ModelRoute
		if err := rows.Scan(&item.ID, &item.ModelAlias, &item.ChannelName, &item.Priority, &item.FallbackModel); err != nil {
			return nil, fmt.Errorf("读取 model_route 失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 model_routes 失败: %w", err)
	}

	return items, nil
}

func ListRequestLogs(ctx context.Context, db *sql.DB) ([]domain.RequestLog, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at FROM request_logs ORDER BY id DESC`)
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

func ClearRequestLogs(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM request_logs`); err != nil {
		return fmt.Errorf("清空 request_logs 失败: %w", err)
	}

	return nil
}

func ListSystemSettings(ctx context.Context, db *sql.DB) ([]domain.SystemSetting, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value, updated_at FROM system_settings ORDER BY key ASC`)
	if err != nil {
		return nil, fmt.Errorf("查询 system_settings 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.SystemSetting, 0)
	for rows.Next() {
		var item domain.SystemSetting
		if err := rows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取 system_setting 失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 system_settings 失败: %w", err)
	}

	return items, nil
}

func UpsertSystemSetting(ctx context.Context, db *sql.DB, input domain.UpdateSystemSettingInput) (domain.SystemSetting, error) {
	now := time.Now().Format(time.RFC3339)
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		input.Key,
		input.Value,
		now,
	)
	if err != nil {
		return domain.SystemSetting{}, fmt.Errorf("写入 system_setting 失败: %w", err)
	}

	return domain.SystemSetting{
		Key:       input.Key,
		Value:     input.Value,
		UpdatedAt: now,
	}, nil
}

func GetFirstEnabledChannel(ctx context.Context, db *sql.DB) (domain.UpstreamChannel, error) {
	var item domain.UpstreamChannel
	if err := db.QueryRowContext(
		ctx,
		`SELECT name, provider, endpoint, api_key FROM channels WHERE enabled = 1 ORDER BY id ASC LIMIT 1`,
	).Scan(&item.Name, &item.Provider, &item.Endpoint, &item.APIKey); err != nil {
		if err == sql.ErrNoRows {
			return domain.UpstreamChannel{}, fmt.Errorf("当前没有可用的启用渠道")
		}
		return domain.UpstreamChannel{}, fmt.Errorf("查询启用渠道失败: %w", err)
	}

	return item, nil
}

func GetChannelByID(ctx context.Context, db *sql.DB, id int64) (domain.UpstreamChannel, error) {
	var item domain.UpstreamChannel
	if err := db.QueryRowContext(
		ctx,
		`SELECT name, provider, endpoint, api_key FROM channels WHERE id = ? LIMIT 1`,
		id,
	).Scan(&item.Name, &item.Provider, &item.Endpoint, &item.APIKey); err != nil {
		if err == sql.ErrNoRows {
			return domain.UpstreamChannel{}, fmt.Errorf("渠道不存在")
		}
		return domain.UpstreamChannel{}, fmt.Errorf("查询渠道失败: %w", err)
	}

	return item, nil
}

func CreateChannel(ctx context.Context, db *sql.DB, input domain.CreateChannelInput) (domain.Channel, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO channels(name, provider, endpoint, api_key, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Name,
		input.Provider,
		input.Endpoint,
		input.APIKey,
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("创建 channel 失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.Channel{}, fmt.Errorf("读取 channel id 失败: %w", err)
	}

	for index, modelID := range input.ModelIDs {
		normalized := strings.TrimSpace(modelID)
		if normalized == "" {
			continue
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			normalized,
			normalized,
			now,
			now,
		); err != nil {
			return domain.Channel{}, fmt.Errorf("创建 model 失败: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO model_routes(model_alias, channel_name, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			normalized,
			input.Name,
			index+1,
			"",
			now,
			now,
		); err != nil {
			return domain.Channel{}, fmt.Errorf("创建 model_route 失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Channel{}, fmt.Errorf("提交事务失败: %w", err)
	}

	return domain.Channel{
		ID:        id,
		Name:      input.Name,
		Provider:  input.Provider,
		Endpoint:  input.Endpoint,
		Enabled:   input.Enabled,
		UpdatedAt: now,
	}, nil
}

func UpdateChannel(ctx context.Context, db *sql.DB, id int64, input domain.UpdateChannelInput) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	var originalName string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM channels WHERE id = ? LIMIT 1`, id).Scan(&originalName); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("渠道不存在")
		}
		return fmt.Errorf("查询 channel 失败: %w", err)
	}

	query := `UPDATE channels SET name = ?, provider = ?, endpoint = ?, enabled = ?, updated_at = ? WHERE id = ?`
	args := []any{input.Name, input.Provider, input.Endpoint, boolToInt(input.Enabled), now, id}
	if input.APIKey != "" {
		query = `UPDATE channels SET name = ?, provider = ?, endpoint = ?, api_key = ?, enabled = ?, updated_at = ? WHERE id = ?`
		args = []any{input.Name, input.Provider, input.Endpoint, input.APIKey, boolToInt(input.Enabled), now, id}
	}
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("更新 channel 失败: %w", err)
	}

	if input.Name != "" && originalName != input.Name {
		if _, err := tx.ExecContext(ctx, `UPDATE model_routes SET channel_name = ?, updated_at = ? WHERE channel_name = ?`, input.Name, now, originalName); err != nil {
			return fmt.Errorf("更新 model_route 渠道名失败: %w", err)
		}
	}

	if input.ModelIDs != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM model_routes WHERE channel_name = ?`, input.Name); err != nil {
			return fmt.Errorf("清理 model_route 失败: %w", err)
		}

		for index, modelID := range input.ModelIDs {
			normalized := strings.TrimSpace(modelID)
			if normalized == "" {
				continue
			}

			if _, err := tx.ExecContext(ctx, `INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(alias) DO UPDATE SET upstream_model = excluded.upstream_model, updated_at = excluded.updated_at`, normalized, normalized, now, now); err != nil {
				return fmt.Errorf("写入 model 失败: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `INSERT INTO model_routes(model_alias, channel_name, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, normalized, input.Name, index+1, "", now, now); err != nil {
				return fmt.Errorf("写入 model_route 失败: %w", err)
			}
		}

		if err := cleanupOrphanModels(ctx, tx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func DeleteChannel(ctx context.Context, db *sql.DB, id int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var channelName string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM channels WHERE id = ? LIMIT 1`, id).Scan(&channelName); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("渠道不存在")
		}
		return fmt.Errorf("查询 channel 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM model_routes WHERE channel_name = ?`, channelName); err != nil {
		return fmt.Errorf("删除 model_route 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM channels WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除 channel 失败: %w", err)
	}

	if err := cleanupOrphanModels(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func cleanupOrphanModels(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM models WHERE alias NOT IN (SELECT DISTINCT model_alias FROM model_routes)`); err != nil {
		return fmt.Errorf("清理孤立 model 失败: %w", err)
	}
	return nil
}

func CreateAPIKey(ctx context.Context, db *sql.DB, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error) {
	rawKey, err := generateAPIKey()
	if err != nil {
		return domain.CreatedAPIKey{}, fmt.Errorf("生成 api key 失败: %w", err)
	}

	keyHash := sha256.Sum256([]byte(rawKey))
	now := time.Now().Format(time.RFC3339)
	result, err := db.ExecContext(
		ctx,
		`INSERT INTO api_keys(name, key_hash, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		input.Name,
		hex.EncodeToString(keyHash[:]),
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return domain.CreatedAPIKey{}, fmt.Errorf("创建 api key 失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.CreatedAPIKey{}, fmt.Errorf("读取 api key id 失败: %w", err)
	}

	return domain.CreatedAPIKey{
		ID:      id,
		Name:    input.Name,
		RawKey:  rawKey,
		Enabled: input.Enabled,
	}, nil
}

func UpdateAPIKey(ctx context.Context, db *sql.DB, id int64, input domain.UpdateAPIKeyInput) error {
	_, err := db.ExecContext(
		ctx,
		`UPDATE api_keys SET enabled = ?, updated_at = ? WHERE id = ?`,
		boolToInt(input.Enabled),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return fmt.Errorf("更新 api key 失败: %w", err)
	}
	return nil
}

func DeleteAPIKey(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除 api key 失败: %w", err)
	}
	return nil
}

func CreateModelMapping(ctx context.Context, db *sql.DB, input domain.CreateModelMappingInput) (domain.ModelMapping, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := db.ExecContext(ctx, `INSERT INTO models(alias, upstream_model, created_at, updated_at) VALUES (?, ?, ?, ?)`, input.Alias, input.UpstreamModel, now, now)
	if err != nil {
		return domain.ModelMapping{}, fmt.Errorf("创建 model 失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.ModelMapping{}, fmt.Errorf("读取 model id 失败: %w", err)
	}
	return domain.ModelMapping{ID: id, Alias: input.Alias, UpstreamModel: input.UpstreamModel}, nil
}

func UpdateModelMapping(ctx context.Context, db *sql.DB, id int64, input domain.UpdateModelMappingInput) error {
	_, err := db.ExecContext(ctx, `UPDATE models SET alias = ?, upstream_model = ?, updated_at = ? WHERE id = ?`, input.Alias, input.UpstreamModel, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("更新 model 失败: %w", err)
	}
	return nil
}

func DeleteModelMapping(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM models WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除 model 失败: %w", err)
	}
	return nil
}

func CreateModelRoute(ctx context.Context, db *sql.DB, input domain.CreateModelRouteInput) (domain.ModelRoute, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := db.ExecContext(ctx, `INSERT INTO model_routes(model_alias, channel_name, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, input.ModelAlias, input.ChannelName, input.Priority, input.FallbackModel, now, now)
	if err != nil {
		return domain.ModelRoute{}, fmt.Errorf("创建 model_route 失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.ModelRoute{}, fmt.Errorf("读取 model_route id 失败: %w", err)
	}
	return domain.ModelRoute{ID: id, ModelAlias: input.ModelAlias, ChannelName: input.ChannelName, Priority: input.Priority, FallbackModel: input.FallbackModel}, nil
}

func UpdateModelRoute(ctx context.Context, db *sql.DB, id int64, input domain.UpdateModelRouteInput) error {
	_, err := db.ExecContext(ctx, `UPDATE model_routes SET model_alias = ?, channel_name = ?, priority = ?, fallback_model = ?, updated_at = ? WHERE id = ?`, input.ModelAlias, input.ChannelName, input.Priority, input.FallbackModel, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("更新 model_route 失败: %w", err)
	}
	return nil
}

func DeleteModelRoute(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM model_routes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除 model_route 失败: %w", err)
	}
	return nil
}

func VerifyAPIKey(ctx context.Context, db *sql.DB, rawKey string) (bool, error) {
	keyHash := sha256.Sum256([]byte(rawKey))
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT 1 FROM api_keys WHERE key_hash = ? AND enabled = 1 LIMIT 1`, hex.EncodeToString(keyHash[:])).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("校验 api key 失败: %w", err)
	}

	return true, nil
}

func CreateRequestLog(ctx context.Context, db *sql.DB, item domain.RequestLog) error {
	_, err := db.ExecContext(
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

func generateAPIKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return "sk-opencrab-" + hex.EncodeToString(buf), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
