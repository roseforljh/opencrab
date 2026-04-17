package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
)

func ListChannels(ctx context.Context, db *sql.DB) ([]domain.Channel, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, provider, endpoint, enabled, rpm_limit, max_inflight, safety_factor, enabled_for_async, dispatch_weight, updated_at FROM channels ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 channels 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Channel, 0)
	for rows.Next() {
		var item domain.Channel
		var enabled int
		var enabledForAsync int
		if err := rows.Scan(&item.ID, &item.Name, &item.Provider, &item.Endpoint, &enabled, &item.RPMLimit, &item.MaxInflight, &item.SafetyFactor, &enabledForAsync, &item.DispatchWeight, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取 channel 失败: %w", err)
		}
		item.Enabled = enabled == 1
		item.EnabledForAsync = enabledForAsync == 1
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
	rows, err := db.QueryContext(ctx, `SELECT mr.id, mr.model_alias, mr.channel_name, mr.invocation_mode, mr.priority, mr.fallback_model, COALESCE(rrs.cooldown_until, ''), COALESCE(rrs.last_error, '') FROM model_routes mr LEFT JOIN routing_runtime_states rrs ON rrs.route_id = mr.id ORDER BY mr.priority ASC, mr.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询 model_routes 失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ModelRoute, 0)
	for rows.Next() {
		var item domain.ModelRoute
		if err := rows.Scan(&item.ID, &item.ModelAlias, &item.ChannelName, &item.InvocationMode, &item.Priority, &item.FallbackModel, &item.CooldownUntil, &item.LastError); err != nil {
			return nil, fmt.Errorf("读取 model_route 失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 model_routes 失败: %w", err)
	}

	return items, nil
}

func GetRoutingOverview(ctx context.Context, db *sql.DB) (domain.RoutingOverview, error) {
	logs, err := ListRequestLogSummariesSince(ctx, db, time.Now().Add(-24*time.Hour), 1000)
	if err != nil {
		return domain.RoutingOverview{}, err
	}
	return buildRoutingOverview(ctx, db, logs)
}

func buildRoutingOverview(ctx context.Context, db *sql.DB, logs []domain.RequestLogSummary) (domain.RoutingOverview, error) {
	overview := domain.RoutingOverview{RecentErrors: []string{}, CursorStates: []domain.RoutingCursorState{}}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM routing_runtime_states WHERE cooldown_until <> '' AND datetime(cooldown_until) > datetime('now')`).Scan(&overview.ActiveCooldowns); err != nil {
		return overview, fmt.Errorf("统计 cooldown 失败: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM routing_affinity_bindings`).Scan(&overview.StickyBindings); err != nil {
		return overview, fmt.Errorf("统计 sticky 绑定失败: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_routes`).Scan(&overview.TotalRoutes); err != nil {
		return overview, fmt.Errorf("统计总路由失败: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_routes mr LEFT JOIN routing_runtime_states rrs ON rrs.route_id = mr.id WHERE COALESCE(rrs.cooldown_until, '') = '' OR datetime(rrs.cooldown_until) <= datetime('now')`).Scan(&overview.HealthyRoutes); err != nil {
		return overview, fmt.Errorf("统计健康路由失败: %w", err)
	}

	since := time.Now().Add(-24 * time.Hour)
	for _, logItem := range logs {
		createdAt, parseErr := time.Parse(time.RFC3339, logItem.CreatedAt)
		if parseErr != nil || createdAt.Before(since) {
			continue
		}
		details := parseJSONMap(logItem.Details)
		logType, _ := details["log_type"].(string)
		if logType == "gateway_request" {
			overview.RequestCount24h++
			if stickyHit, ok := details["sticky_hit"].(bool); ok && stickyHit {
				overview.StickyHits24h++
			}
			if chain, ok := details["fallback_chain"].([]any); ok && len(chain) > 0 {
				overview.Fallbacks24h++
			}
			if skips, ok := details["skips"].([]any); ok {
				overview.Skipped24h += len(skips)
			}
		}
		if logType == "gateway_attempt" && logItem.StatusCode >= 500 && len(overview.RecentErrors) < 3 {
			if errorMessage, ok := details["error_message"].(string); ok && strings.TrimSpace(errorMessage) != "" {
				overview.RecentErrors = append(overview.RecentErrors, errorMessage)
			}
		}
	}

	denominator := overview.TotalRoutes + overview.ActiveCooldowns + overview.Fallbacks24h
	if denominator > 0 {
		overview.PressureScore = min(100, (overview.ActiveCooldowns*45)+(overview.Fallbacks24h*10)+(overview.Skipped24h*2))
	}

	cursorRows, err := db.QueryContext(ctx, `SELECT route_key, next_index, updated_at FROM routing_cursors ORDER BY updated_at DESC LIMIT 5`)
	if err != nil {
		return overview, fmt.Errorf("查询 routing_cursors 失败: %w", err)
	}
	defer cursorRows.Close()
	for cursorRows.Next() {
		var item domain.RoutingCursorState
		if err := cursorRows.Scan(&item.RouteKey, &item.NextIndex, &item.UpdatedAt); err != nil {
			return overview, fmt.Errorf("读取 routing_cursor 失败: %w", err)
		}
		overview.CursorStates = append(overview.CursorStates, item)
	}
	if err := cursorRows.Err(); err != nil {
		return overview, fmt.Errorf("遍历 routing_cursors 失败: %w", err)
	}

	return overview, nil
}

func ListRequestLogSummaries(ctx context.Context, db *sql.DB) ([]domain.RequestLogSummary, error) {
	return listRequestLogSummaries(ctx, db, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, details, created_at FROM request_logs ORDER BY created_at DESC LIMIT 200`)
}

func ListRequestLogSummariesSince(ctx context.Context, db *sql.DB, since time.Time, limit int) ([]domain.RequestLogSummary, error) {
	if limit <= 0 {
		limit = 1000
	}
	return listRequestLogSummaries(ctx, db, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, details, created_at FROM request_logs WHERE created_at >= ? ORDER BY created_at DESC LIMIT ?`, since.UTC().Format(time.RFC3339), limit)
}

func listRequestLogSummaries(ctx context.Context, db *sql.DB, query string, args ...any) ([]domain.RequestLogSummary, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询 request_logs 摘要失败: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RequestLogSummary, 0)
	for rows.Next() {
		var item domain.RequestLogSummary
		var cacheHit int
		if err := rows.Scan(&item.ID, &item.RequestID, &item.Model, &item.Channel, &item.StatusCode, &item.LatencyMs, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &cacheHit, &item.Details, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("读取 request_log 摘要失败: %w", err)
		}
		item.CacheHit = cacheHit == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 request_log 摘要失败: %w", err)
	}

	return items, nil
}

func GetRequestLogDetail(ctx context.Context, db *sql.DB, id int64) (domain.RequestLog, error) {
	var item domain.RequestLog
	var cacheHit int
	err := db.QueryRowContext(ctx, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at FROM request_logs WHERE id = ? LIMIT 1`, id).
		Scan(&item.ID, &item.RequestID, &item.Model, &item.Channel, &item.StatusCode, &item.LatencyMs, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &cacheHit, &item.RequestBody, &item.ResponseBody, &item.Details, &item.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.RequestLog{}, fmt.Errorf("请求日志不存在")
		}
		return domain.RequestLog{}, fmt.Errorf("读取 request_log 详情失败: %w", err)
	}
	item.CacheHit = cacheHit == 1
	return item, nil
}

func GetDashboardSummary(ctx context.Context, db *sql.DB) (domain.DashboardSummary, error) {
	channels, err := ListChannels(ctx, db)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	models, err := ListModelMappings(ctx, db)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	routes, err := ListModelRoutes(ctx, db)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	apiKeys, err := ListAPIKeys(ctx, db)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	settings, err := ListSystemSettings(ctx, db)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	logs, err := ListRequestLogSummariesSince(ctx, db, startOfDay(time.Now()).AddDate(0, 0, -6), 5000)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	overview, err := buildRoutingOverview(ctx, db, logs)
	if err != nil {
		return domain.DashboardSummary{}, err
	}

	summary := domain.DashboardSummary{
		ChannelsCount:   len(channels),
		ModelsCount:     len(models),
		RoutesCount:     len(routes),
		APIKeysCount:    len(apiKeys),
		RoutingOverview: overview,
		DailyCounts:     buildDashboardDailyCounts(logs),
		TrafficSeries:   buildDashboardTrafficSeries(logs),
		RecentLogs:      buildDashboardRecentLogs(logs),
	}

	providerSet := map[string]struct{}{}
	var defaultChannelID int64
	for _, channel := range channels {
		providerSet[channel.Provider] = struct{}{}
		if channel.Enabled {
			summary.EnabledChannelsCount++
			summary.TotalRPMLimit += channel.RPMLimit
			summary.TotalMaxInflight += channel.MaxInflight
			if channel.EnabledForAsync {
				summary.AsyncEnabledChannels++
			}
			if summary.DefaultChannel == "" || channel.ID < defaultChannelID {
				summary.DefaultChannel = channel.Name
				defaultChannelID = channel.ID
			}
		}
	}
	summary.ProviderCount = len(providerSet)

	requestLogs := filterGatewayRequestLogs(logs)
	summary.TodayRequests = countTodayRequests(requestLogs)
	summary.TotalRequests = len(requestLogs)
	todayStart := startOfDay(time.Now())
	for _, logItem := range requestLogs {
		createdAt, parseErr := time.Parse(time.RFC3339, logItem.CreatedAt)
		if isSuccessStatus(logItem.StatusCode) {
			summary.SuccessCount++
		}
		if parseErr == nil && !createdAt.Before(todayStart) {
			if isSuccessStatus(logItem.StatusCode) {
				summary.TodaySuccessCount++
			} else {
				summary.TodayErrorCount++
			}
		}
		summary.AverageLatency += logItem.LatencyMs
		summary.PromptTokens += logItem.PromptTokens
		summary.CompletionTokens += logItem.CompletionTokens
		summary.TotalTokens += logItem.TotalTokens
		if logItem.TotalTokens > 0 {
			summary.TotalMeteredRequests++
		}
		if logItem.CacheHit {
			summary.CacheHitCount++
		}
	}
	if summary.TotalRequests > 0 {
		summary.ErrorCount = summary.TotalRequests - summary.SuccessCount
		summary.AverageLatency = summary.AverageLatency / int64(summary.TotalRequests)
		summary.CacheHitRate = (float64(summary.CacheHitCount) / float64(summary.TotalRequests)) * 100
	} else {
		summary.ErrorCount = 0
		summary.AverageLatency = 0
		summary.CacheHitRate = 0
	}

	lastMinute := time.Now().Add(-time.Minute)
	for _, logItem := range requestLogs {
		createdAt, err := time.Parse(time.RFC3339, logItem.CreatedAt)
		if err != nil || createdAt.Before(lastMinute) {
			continue
		}
		summary.RequestsPerMinute++
		if isSuccessStatus(logItem.StatusCode) {
			summary.RequestsPerMinuteSuccess++
		} else {
			summary.RequestsPerMinuteError++
		}
		summary.TokensPerMinute += logItem.TotalTokens
		if logItem.TotalTokens > 0 {
			summary.TokensPerMinuteMeteredRequests++
		}
	}

	summary.ChannelMix = buildDashboardChannelMix(requestLogs)
	summary.ModelRanking = buildDashboardModelRanking(requestLogs)
	applyDispatchSettingsSummary(&summary, settings)

	return summary, nil
}

func parseJSONMap(value string) map[string]any {
	if strings.TrimSpace(value) == "" {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func filterGatewayRequestLogs(logs []domain.RequestLogSummary) []domain.RequestLogSummary {
	items := make([]domain.RequestLogSummary, 0, len(logs))
	for _, logItem := range logs {
		details := parseJSONMap(logItem.Details)
		logType, _ := details["log_type"].(string)
		if logType != "gateway_request" {
			continue
		}
		if testMode, ok := details["test_mode"].(bool); ok && testMode {
			continue
		}
		items = append(items, logItem)
	}
	return items
}

func countTodayRequests(logs []domain.RequestLogSummary) int {
	start := startOfDay(time.Now())
	count := 0
	for _, logItem := range logs {
		createdAt, err := time.Parse(time.RFC3339, logItem.CreatedAt)
		if err != nil || createdAt.Before(start) {
			continue
		}
		count++
	}
	return count
}

func buildDashboardDailyCounts(logs []domain.RequestLogSummary) []domain.DashboardDailyCount {
	requestLogs := filterGatewayRequestLogs(logs)
	items := make([]domain.DashboardDailyCount, 0, 7)
	for index := 0; index < 7; index++ {
		date := time.Now().AddDate(0, 0, -(6 - index))
		start := startOfDay(date)
		end := start.Add(24 * time.Hour)
		item := domain.DashboardDailyCount{Label: start.Format("01-02")}
		var successCount int
		for _, logItem := range requestLogs {
			createdAt, err := time.Parse(time.RFC3339, logItem.CreatedAt)
			if err != nil || createdAt.Before(start) || !createdAt.Before(end) {
				continue
			}
			item.Requests++
			if isSuccessStatus(logItem.StatusCode) {
				successCount++
			}
			item.AverageLatency += logItem.LatencyMs
			item.TotalTokens += logItem.TotalTokens
		}
		if item.Requests > 0 {
			item.SuccessRate = (float64(successCount) / float64(item.Requests)) * 100
			item.AverageLatency = item.AverageLatency / int64(item.Requests)
		}
		items = append(items, item)
	}
	return items
}

func buildDashboardTrafficSeries(logs []domain.RequestLogSummary) []domain.DashboardTrafficPoint {
	requestLogs := filterGatewayRequestLogs(logs)
	items := make([]domain.DashboardTrafficPoint, 0, 6)
	now := time.Now()
	for index := 0; index < 6; index++ {
		start := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).Add(time.Duration((index-5)*4) * time.Hour)
		end := start.Add(4 * time.Hour)
		item := domain.DashboardTrafficPoint{Label: start.Format("15:04")}
		for _, logItem := range requestLogs {
			createdAt, err := time.Parse(time.RFC3339, logItem.CreatedAt)
			if err != nil || createdAt.Before(start) || !createdAt.Before(end) {
				continue
			}
			item.Requests++
			if isSuccessStatus(logItem.StatusCode) {
				item.Success++
			} else {
				item.Errors++
			}
		}
		items = append(items, item)
	}
	return items
}

func buildDashboardRecentLogs(logs []domain.RequestLogSummary) []domain.DashboardRecentLog {
	requestLogs := filterGatewayRequestLogs(logs)
	items := make([]domain.DashboardRecentLog, 0, min(len(requestLogs), 5))
	for index, logItem := range requestLogs {
		if index >= 5 {
			break
		}
		details := parseJSONMap(logItem.Details)
		channel := logItem.Channel
		if selectedChannel, ok := details["selected_channel"].(string); ok && strings.TrimSpace(selectedChannel) != "" {
			channel = selectedChannel
		}
		status := "异常"
		if isSuccessStatus(logItem.StatusCode) {
			status = "成功"
		}
		items = append(items, domain.DashboardRecentLog{Time: logItem.CreatedAt, Model: logItem.Model, Channel: channel, Status: status, LatencyMs: logItem.LatencyMs})
	}
	return items
}

func buildDashboardChannelMix(logs []domain.RequestLogSummary) []domain.DashboardShareItem {
	requestLogs := filterGatewayRequestLogs(logs)
	counts := map[string]int{}
	for _, logItem := range requestLogs {
		counts[logItem.Channel]++
	}
	return buildDashboardShareItems(counts, len(requestLogs), 4)
}

func buildDashboardModelRanking(logs []domain.RequestLogSummary) []domain.DashboardRankingItem {
	requestLogs := filterGatewayRequestLogs(logs)
	counts := map[string]int{}
	for _, logItem := range requestLogs {
		counts[logItem.Model]++
	}
	type pair struct {
		Label string
		Value int
	}
	pairs := make([]pair, 0, len(counts))
	for label, value := range counts {
		pairs = append(pairs, pair{Label: label, Value: value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Value == pairs[j].Value {
			return pairs[i].Label < pairs[j].Label
		}
		return pairs[i].Value > pairs[j].Value
	})
	if len(pairs) > 5 {
		pairs = pairs[:5]
	}
	maxValue := 1
	if len(pairs) > 0 && pairs[0].Value > 0 {
		maxValue = pairs[0].Value
	}
	items := make([]domain.DashboardRankingItem, 0, len(pairs))
	for _, item := range pairs {
		width := (item.Value * 100) / maxValue
		if width < 10 {
			width = 10
		}
		items = append(items, domain.DashboardRankingItem{Label: item.Label, Value: item.Value, Width: width})
	}
	return items
}

func buildDashboardShareItems(counts map[string]int, total int, limit int) []domain.DashboardShareItem {
	type pair struct {
		Label string
		Value int
	}
	pairs := make([]pair, 0, len(counts))
	for label, value := range counts {
		pairs = append(pairs, pair{Label: label, Value: value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Value == pairs[j].Value {
			return pairs[i].Label < pairs[j].Label
		}
		return pairs[i].Value > pairs[j].Value
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	items := make([]domain.DashboardShareItem, 0, len(pairs))
	for _, item := range pairs {
		share := 0
		if total > 0 {
			share = int(float64(item.Value) / float64(total) * 100)
		}
		items = append(items, domain.DashboardShareItem{Label: item.Label, Value: share})
	}
	return items
}

func applyDispatchSettingsSummary(summary *domain.DashboardSummary, items []domain.SystemSetting) {
	values := make(map[string]string, len(items))
	for _, item := range items {
		values[item.Key] = item.Value
	}
	readValue := func(key string, fallback string) string {
		if value, ok := values[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		return fallback
	}
	summary.RuntimeRedisEnabled = parseBoolSetting(readValue("dispatch.redis_enabled", "false"))
	summary.RuntimeRedisAddress = readValue("dispatch.redis_address", "127.0.0.1:6379")
	summary.RuntimeRedisDB = parseIntSetting(readValue("dispatch.redis_db", "0"), 0)
	summary.RuntimeRedisTLSEnabled = parseBoolSetting(readValue("dispatch.redis_tls_enabled", "false"))
	summary.RuntimeRedisKeyPrefix = readValue("dispatch.redis_key_prefix", "opencrab")
	summary.DispatchPause = parseBoolSetting(readValue("dispatch.pause_dispatch", "false"))
	summary.DispatcherWorkers = parseIntSetting(readValue("dispatch.worker_concurrency", "128"), 128)
	summary.QueueMode = readValue("dispatch.queue_mode", "priority")
	summary.DefaultQueue = readValue("dispatch.default_queue", "model-default")
	summary.PriorityQueues = readValue("dispatch.priority_queues", "p0,p1,p2")
	summary.QueueTTLSec = parseIntSetting(readValue("dispatch.queue_ttl_s", "1800"), 1800)
	summary.SyncHoldMs = parseIntSetting(readValue("dispatch.sync_hold_ms", "3000"), 3000)
	summary.RetryReserveRatio = parseFloatSetting(readValue("dispatch.retry_reserve_ratio", "0.10"), 0.10)
	summary.BacklogCap = parseIntSetting(readValue("dispatch.backlog_cap", "20000"), 20000)
	summary.MaxAttempts = parseIntSetting(readValue("dispatch.max_attempts", "5"), 5)
	summary.BackoffMode = readValue("dispatch.backoff_mode", "exponential")
	summary.BackoffDelayMs = parseIntSetting(readValue("dispatch.backoff_delay_ms", "500"), 500)
	summary.DeadLetterEnabled = parseBoolSetting(readValue("dispatch.dead_letter_enabled", "true"))
	summary.MetricsEnabled = parseBoolSetting(readValue("dispatch.metrics_enabled", "true"))
	summary.LongWaitThresholdSec = parseIntSetting(readValue("dispatch.long_wait_threshold_s", "15"), 15)
	summary.ShowWorkerStatus = parseBoolSetting(readValue("dispatch.show_worker_status", "true"))
	summary.ShowQueueDepth = parseBoolSetting(readValue("dispatch.show_queue_depth", "true"))
	summary.ShowRetryRate = parseBoolSetting(readValue("dispatch.show_retry_rate", "true"))
}

func parseBoolSetting(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "true" || normalized == "1" || normalized == "enabled" || normalized == "启用"
}

func parseIntSetting(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseFloatSetting(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func normalizeChannelRPMLimit(value int) int {
	if value <= 0 {
		return 1000
	}
	return value
}

func normalizeChannelMaxInflight(value int) int {
	if value <= 0 {
		return 32
	}
	return value
}

func normalizeChannelSafetyFactor(value float64) float64 {
	if math.IsNaN(value) || value <= 0 || value > 1 {
		return 0.9
	}
	return value
}

func normalizeChannelDispatchWeight(value int) int {
	if value <= 0 {
		return 100
	}
	return value
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func isSuccessStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 400
}

func ListRequestLogs(ctx context.Context, db *sql.DB) ([]domain.RequestLog, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, request_id, model, channel, status_code, latency_ms, prompt_tokens, completion_tokens, total_tokens, cache_hit, request_body, response_body, details, created_at FROM request_logs ORDER BY created_at DESC LIMIT 200`)
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
		`INSERT INTO channels(name, provider, endpoint, api_key, enabled, rpm_limit, max_inflight, safety_factor, enabled_for_async, dispatch_weight, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.Name,
		input.Provider,
		input.Endpoint,
		input.APIKey,
		boolToInt(input.Enabled),
		normalizeChannelRPMLimit(input.RPMLimit),
		normalizeChannelMaxInflight(input.MaxInflight),
		normalizeChannelSafetyFactor(input.SafetyFactor),
		boolToInt(input.EnabledForAsync),
		normalizeChannelDispatchWeight(input.DispatchWeight),
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
			`INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			normalized,
			input.Name,
			"",
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
		ID:              id,
		Name:            input.Name,
		Provider:        input.Provider,
		Endpoint:        input.Endpoint,
		Enabled:         input.Enabled,
		RPMLimit:        normalizeChannelRPMLimit(input.RPMLimit),
		MaxInflight:     normalizeChannelMaxInflight(input.MaxInflight),
		SafetyFactor:    normalizeChannelSafetyFactor(input.SafetyFactor),
		EnabledForAsync: input.EnabledForAsync,
		DispatchWeight:  normalizeChannelDispatchWeight(input.DispatchWeight),
		UpdatedAt:       now,
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

	query := `UPDATE channels SET name = ?, provider = ?, endpoint = ?, enabled = ?, rpm_limit = ?, max_inflight = ?, safety_factor = ?, enabled_for_async = ?, dispatch_weight = ?, updated_at = ? WHERE id = ?`
	args := []any{input.Name, input.Provider, input.Endpoint, boolToInt(input.Enabled), normalizeChannelRPMLimit(input.RPMLimit), normalizeChannelMaxInflight(input.MaxInflight), normalizeChannelSafetyFactor(input.SafetyFactor), boolToInt(input.EnabledForAsync), normalizeChannelDispatchWeight(input.DispatchWeight), now, id}
	if input.APIKey != "" {
		query = `UPDATE channels SET name = ?, provider = ?, endpoint = ?, api_key = ?, enabled = ?, rpm_limit = ?, max_inflight = ?, safety_factor = ?, enabled_for_async = ?, dispatch_weight = ?, updated_at = ? WHERE id = ?`
		args = []any{input.Name, input.Provider, input.Endpoint, input.APIKey, boolToInt(input.Enabled), normalizeChannelRPMLimit(input.RPMLimit), normalizeChannelMaxInflight(input.MaxInflight), normalizeChannelSafetyFactor(input.SafetyFactor), boolToInt(input.EnabledForAsync), normalizeChannelDispatchWeight(input.DispatchWeight), now, id}
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

			if _, err := tx.ExecContext(ctx, `INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, normalized, input.Name, "", index+1, "", now, now); err != nil {
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
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	var originalAlias string
	if err := tx.QueryRowContext(ctx, `SELECT alias FROM models WHERE id = ? LIMIT 1`, id).Scan(&originalAlias); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("模型不存在")
		}
		return fmt.Errorf("查询 model 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE models SET alias = ?, upstream_model = ?, updated_at = ? WHERE id = ?`, input.Alias, input.UpstreamModel, now, id); err != nil {
		return fmt.Errorf("更新 model 失败: %w", err)
	}

	if strings.TrimSpace(originalAlias) != strings.TrimSpace(input.Alias) {
		if _, err := tx.ExecContext(ctx, `UPDATE model_routes SET model_alias = ?, updated_at = ? WHERE model_alias = ?`, input.Alias, now, originalAlias); err != nil {
			return fmt.Errorf("同步 model_route 别名失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func DeleteModelMapping(ctx context.Context, db *sql.DB, id int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var alias string
	if err := tx.QueryRowContext(ctx, `SELECT alias FROM models WHERE id = ? LIMIT 1`, id).Scan(&alias); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("模型不存在")
		}
		return fmt.Errorf("查询 model 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM model_routes WHERE model_alias = ?`, alias); err != nil {
		return fmt.Errorf("删除 model_route 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM models WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除 model 失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func CreateModelRoute(ctx context.Context, db *sql.DB, input domain.CreateModelRouteInput) (domain.ModelRoute, error) {
	if err := ensureModelRouteReferences(ctx, db, input.ModelAlias, input.ChannelName, nil); err != nil {
		return domain.ModelRoute{}, err
	}
	if err := validateFallbackConsistency(ctx, db, input.ModelAlias, input.FallbackModel, nil); err != nil {
		return domain.ModelRoute{}, err
	}
	now := time.Now().Format(time.RFC3339)
	result, err := db.ExecContext(ctx, `INSERT INTO model_routes(model_alias, channel_name, invocation_mode, priority, fallback_model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, input.ModelAlias, input.ChannelName, input.InvocationMode, input.Priority, input.FallbackModel, now, now)
	if err != nil {
		return domain.ModelRoute{}, fmt.Errorf("创建 model_route 失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.ModelRoute{}, fmt.Errorf("读取 model_route id 失败: %w", err)
	}
	return domain.ModelRoute{ID: id, ModelAlias: input.ModelAlias, ChannelName: input.ChannelName, InvocationMode: input.InvocationMode, Priority: input.Priority, FallbackModel: input.FallbackModel}, nil
}

func UpdateModelRoute(ctx context.Context, db *sql.DB, id int64, input domain.UpdateModelRouteInput) error {
	if err := ensureModelRouteReferences(ctx, db, input.ModelAlias, input.ChannelName, &id); err != nil {
		return err
	}
	if err := validateFallbackConsistency(ctx, db, input.ModelAlias, input.FallbackModel, &id); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `UPDATE model_routes SET model_alias = ?, channel_name = ?, invocation_mode = ?, priority = ?, fallback_model = ?, updated_at = ? WHERE id = ?`, input.ModelAlias, input.ChannelName, input.InvocationMode, input.Priority, input.FallbackModel, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("更新 model_route 失败: %w", err)
	}
	return nil
}

func UpdateModelRouteBinding(ctx context.Context, db *sql.DB, id int64, input domain.UpdateModelRouteBindingInput) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var currentAlias string
	var modelID int64
	if err := tx.QueryRowContext(ctx, `SELECT mr.model_alias, m.id FROM model_routes mr JOIN models m ON m.alias = mr.model_alias WHERE mr.id = ? LIMIT 1`, id).Scan(&currentAlias, &modelID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("模型路由不存在")
		}
		return fmt.Errorf("查询模型路由失败: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	normalizedAlias := strings.TrimSpace(input.Alias)
	normalizedChannel := strings.TrimSpace(input.ChannelName)
	if normalizedAlias == "" || normalizedChannel == "" || strings.TrimSpace(input.UpstreamModel) == "" {
		return fmt.Errorf("模型别名、上游模型和渠道名称不能为空")
	}
	if !recordExistsTx(ctx, tx, `SELECT 1 FROM channels WHERE name = ? LIMIT 1`, normalizedChannel) {
		return fmt.Errorf("渠道不存在")
	}
	if recordExistsTx(ctx, tx, `SELECT id FROM model_routes WHERE model_alias = ? AND channel_name = ? AND id <> ? LIMIT 1`, normalizedAlias, normalizedChannel, id) {
		return fmt.Errorf("相同模型别名和渠道的路由已存在")
	}
	if err := validateFallbackConsistencyTx(ctx, tx, normalizedAlias, input.FallbackModel, &id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE models SET alias = ?, upstream_model = ?, updated_at = ? WHERE id = ?`, normalizedAlias, input.UpstreamModel, now, modelID); err != nil {
		return fmt.Errorf("更新 model 失败: %w", err)
	}
	if strings.TrimSpace(currentAlias) != normalizedAlias {
		if _, err := tx.ExecContext(ctx, `UPDATE model_routes SET model_alias = ?, updated_at = ? WHERE model_alias = ?`, normalizedAlias, now, currentAlias); err != nil {
			return fmt.Errorf("同步 model_route 别名失败: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE model_routes SET model_alias = ?, channel_name = ?, invocation_mode = ?, priority = ?, fallback_model = ?, updated_at = ? WHERE id = ?`, normalizedAlias, normalizedChannel, input.InvocationMode, input.Priority, input.FallbackModel, now, id); err != nil {
		return fmt.Errorf("更新 model_route 失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func DeleteModelRoute(ctx context.Context, db *sql.DB, id int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM model_routes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除 model_route 失败: %w", err)
	}

	if err := cleanupOrphanModels(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func ensureModelRouteReferences(ctx context.Context, db *sql.DB, modelAlias string, channelName string, excludeID *int64) error {
	normalizedAlias := strings.TrimSpace(modelAlias)
	normalizedChannel := strings.TrimSpace(channelName)
	if normalizedAlias == "" || normalizedChannel == "" {
		return fmt.Errorf("模型别名和渠道名称不能为空")
	}

	if !recordExists(ctx, db, `SELECT 1 FROM models WHERE alias = ? LIMIT 1`, normalizedAlias) {
		return fmt.Errorf("模型别名不存在")
	}
	if !recordExists(ctx, db, `SELECT 1 FROM channels WHERE name = ? LIMIT 1`, normalizedChannel) {
		return fmt.Errorf("渠道不存在")
	}

	query := `SELECT id FROM model_routes WHERE model_alias = ? AND channel_name = ? LIMIT 1`
	args := []any{normalizedAlias, normalizedChannel}
	if excludeID != nil {
		query = `SELECT id FROM model_routes WHERE model_alias = ? AND channel_name = ? AND id <> ? LIMIT 1`
		args = append(args, *excludeID)
	}
	if recordExists(ctx, db, query, args...) {
		return fmt.Errorf("相同模型别名和渠道的路由已存在")
	}

	return nil
}

func ensureModelRouteReferencesTx(ctx context.Context, tx *sql.Tx, modelAlias string, channelName string, excludeID *int64) error {
	normalizedAlias := strings.TrimSpace(modelAlias)
	normalizedChannel := strings.TrimSpace(channelName)
	if normalizedAlias == "" || normalizedChannel == "" {
		return fmt.Errorf("模型别名和渠道名称不能为空")
	}
	if !recordExistsTx(ctx, tx, `SELECT 1 FROM models WHERE alias = ? LIMIT 1`, normalizedAlias) {
		return fmt.Errorf("模型别名不存在")
	}
	if !recordExistsTx(ctx, tx, `SELECT 1 FROM channels WHERE name = ? LIMIT 1`, normalizedChannel) {
		return fmt.Errorf("渠道不存在")
	}
	query := `SELECT id FROM model_routes WHERE model_alias = ? AND channel_name = ? LIMIT 1`
	args := []any{normalizedAlias, normalizedChannel}
	if excludeID != nil {
		query = `SELECT id FROM model_routes WHERE model_alias = ? AND channel_name = ? AND id <> ? LIMIT 1`
		args = append(args, *excludeID)
	}
	if recordExistsTx(ctx, tx, query, args...) {
		return fmt.Errorf("相同模型别名和渠道的路由已存在")
	}
	return nil
}

func validateFallbackConsistency(ctx context.Context, db *sql.DB, modelAlias string, fallbackModel string, excludeID *int64) error {
	normalizedAlias := strings.TrimSpace(modelAlias)
	normalizedFallback := strings.TrimSpace(fallbackModel)
	if normalizedFallback == "" {
		return nil
	}
	if normalizedFallback == normalizedAlias {
		return fmt.Errorf("fallback 模型不能指向自己")
	}
	if !recordExists(ctx, db, `SELECT 1 FROM models WHERE alias = ? LIMIT 1`, normalizedFallback) {
		return fmt.Errorf("fallback 模型不存在")
	}
	query := `SELECT fallback_model FROM model_routes WHERE model_alias = ? AND fallback_model <> '' LIMIT 1`
	args := []any{normalizedAlias}
	if excludeID != nil {
		query = `SELECT fallback_model FROM model_routes WHERE model_alias = ? AND fallback_model <> '' AND id <> ? LIMIT 1`
		args = append(args, *excludeID)
	}
	var existing string
	err := db.QueryRowContext(ctx, query, args...).Scan(&existing)
	if err == nil && strings.TrimSpace(existing) != normalizedFallback {
		return fmt.Errorf("同一模型别名下 fallback 模型必须保持一致")
	}
	return nil
}

func validateFallbackConsistencyTx(ctx context.Context, tx *sql.Tx, modelAlias string, fallbackModel string, excludeID *int64) error {
	normalizedAlias := strings.TrimSpace(modelAlias)
	normalizedFallback := strings.TrimSpace(fallbackModel)
	if normalizedFallback == "" {
		return nil
	}
	if normalizedFallback == normalizedAlias {
		return fmt.Errorf("fallback 模型不能指向自己")
	}
	if !recordExistsTx(ctx, tx, `SELECT 1 FROM models WHERE alias = ? LIMIT 1`, normalizedFallback) {
		return fmt.Errorf("fallback 模型不存在")
	}
	query := `SELECT fallback_model FROM model_routes WHERE model_alias = ? AND fallback_model <> '' LIMIT 1`
	args := []any{normalizedAlias}
	if excludeID != nil {
		query = `SELECT fallback_model FROM model_routes WHERE model_alias = ? AND fallback_model <> '' AND id <> ? LIMIT 1`
		args = append(args, *excludeID)
	}
	var existing string
	err := tx.QueryRowContext(ctx, query, args...).Scan(&existing)
	if err == nil && strings.TrimSpace(existing) != normalizedFallback {
		return fmt.Errorf("同一模型别名下 fallback 模型必须保持一致")
	}
	return nil
}

func recordExists(ctx context.Context, db *sql.DB, query string, args ...any) bool {
	var id int64
	err := db.QueryRowContext(ctx, query, args...).Scan(&id)
	return err == nil
}

func recordExistsTx(ctx context.Context, tx *sql.Tx, query string, args ...any) bool {
	var id int64
	err := tx.QueryRowContext(ctx, query, args...).Scan(&id)
	return err == nil
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
