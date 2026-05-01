package httpserver

import "net/http"

func adminAuthStatusHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"initialized":   true,
		"authenticated": true,
	})
}

func adminSecondarySecurityHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":    false,
		"configured": false,
	})
}

func adminEmptyListHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
	})
}

func adminDashboardSummaryHandler(w http.ResponseWriter, r *http.Request) {
	channels := compatChannels.listChannels()
	models := compatChannels.listModels()
	routes := compatChannels.listModelRoutes()
	logs, _, _ := compatRequestLogs.list("", "all")
	enabledChannelsCount := 0
	defaultChannel := ""
	providerNames := make(map[string]struct{})
	for index, item := range channels {
		if enabled, _ := item["enabled"].(bool); enabled {
			enabledChannelsCount++
			if defaultChannel == "" {
				if name, _ := item["name"].(string); name != "" {
					defaultChannel = name
				}
			}
		}
		if provider, _ := item["provider"].(string); provider != "" {
			providerNames[provider] = struct{}{}
		}
		if index == 0 && defaultChannel == "" {
			if name, _ := item["name"].(string); name != "" {
				defaultChannel = name
			}
		}
	}
	successCount := 0
	errorCount := 0
	totalLatency := int64(0)
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0
	recentLogs := make([]any, 0, minInt(len(logs), 10))
	channelMixBuckets := make(map[string]int)
	modelBuckets := make(map[string]int)
	for index, item := range logs {
		statusCode, _ := item["status_code"].(int)
		if statusCode >= 200 && statusCode < 400 {
			successCount++
		} else if statusCode >= 400 {
			errorCount++
		}
		if latency, ok := item["latency_ms"].(int64); ok {
			totalLatency += latency
		}
		if value, ok := item["prompt_tokens"].(int); ok { promptTokens += value }
		if value, ok := item["completion_tokens"].(int); ok { completionTokens += value }
		if value, ok := item["total_tokens"].(int); ok { totalTokens += value }
		channelName, _ := item["channel"].(string)
		if channelName != "" { channelMixBuckets[channelName]++ }
		modelName, _ := item["model"].(string)
		if modelName != "" { modelBuckets[modelName]++ }
		if index < 10 {
			recentLogs = append(recentLogs, map[string]any{
				"time": item["created_at"],
				"model": item["model"],
				"channel": item["channel"],
				"status": map[bool]string{true: "成功", false: "异常"}[statusCode >= 200 && statusCode < 400],
				"latency_ms": item["latency_ms"],
			})
		}
	}
	averageLatency := 0.0
	if len(logs) > 0 {
		averageLatency = float64(totalLatency) / float64(len(logs))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels_count":                     len(channels),
		"models_count":                       len(models),
		"routes_count":                       len(routes),
		"api_keys_count":                     len(compatAPIKeys.list()),
		"enabled_channels_count":             enabledChannelsCount,
		"default_channel":                    defaultChannel,
		"provider_count":                     len(providerNames),
		"routing_overview": map[string]any{
			"active_cooldowns": 0,
			"sticky_bindings":  0,
			"sticky_hits_24h":  0,
			"fallbacks_24h":    0,
			"skipped_24h":      0,
			"request_count_24h": 0,
			"healthy_routes":   0,
			"total_routes":     0,
			"pressure_score":   0,
			"recent_errors":    []any{},
			"cursor_states":    []any{},
		},
		"today_requests":                      len(logs),
		"today_success_count":                 successCount,
		"today_error_count":                   errorCount,
		"total_requests":                      len(logs),
		"success_count":                       successCount,
		"error_count":                         errorCount,
		"average_latency":                     averageLatency,
		"prompt_tokens":                       promptTokens,
		"completion_tokens":                   completionTokens,
		"total_tokens":                        totalTokens,
		"total_metered_requests":              len(logs),
		"cache_hit_count":                     0,
		"cache_hit_rate":                      0,
		"requests_per_minute":                 0,
		"requests_per_minute_success":         0,
		"requests_per_minute_error":           0,
		"tokens_per_minute":                   0,
		"tokens_per_minute_metered_requests":  0,
		"daily_counts":                        []any{},
		"traffic_series":                      []any{},
		"recent_logs":                         recentLogs,
		"channel_mix":                         topCountList(channelMixBuckets),
		"model_ranking":                       topRankingList(modelBuckets),
		"runtime_redis_enabled":               false,
		"runtime_redis_address":               "",
		"runtime_redis_db":                    0,
		"runtime_redis_tls_enabled":           false,
		"runtime_redis_key_prefix":            "",
		"dispatch_pause":                      false,
		"dispatcher_workers":                  0,
		"queue_mode":                          "disabled",
		"default_queue":                       "",
		"priority_queues":                     "",
		"queue_ttl_s":                         0,
		"sync_hold_ms":                        0,
		"retry_reserve_ratio":                 0,
		"backlog_cap":                         0,
		"max_attempts":                        0,
		"backoff_mode":                        "disabled",
		"backoff_delay_ms":                    0,
		"dead_letter_enabled":                 false,
		"metrics_enabled":                     false,
		"long_wait_threshold_s":               0,
		"show_worker_status":                  false,
		"show_queue_depth":                    false,
		"show_retry_rate":                     false,
		"async_enabled_channels":              0,
		"total_rpm_limit":                     0,
		"total_max_inflight":                  0,
	})
}

func minInt(left int, right int) int {
	if left < right { return left }
	return right
}

type countItem struct { label string; value int }

func sortCountItems(items []countItem) []countItem {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].value > items[i].value || (items[j].value == items[i].value && items[j].label < items[i].label) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}

func topCountList(values map[string]int) []any {
	items := make([]countItem, 0, len(values))
	for label, value := range values { items = append(items, countItem{label: label, value: value}) }
	items = sortCountItems(items)
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{"label": item.label, "value": item.value})
	}
	return result
}

func topRankingList(values map[string]int) []any {
	items := make([]countItem, 0, len(values))
	maxValue := 1
	for label, value := range values {
		items = append(items, countItem{label: label, value: value})
		if value > maxValue { maxValue = value }
	}
	items = sortCountItems(items)
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{"label": item.label, "value": item.value, "width": float64(item.value) / float64(maxValue) * 100})
	}
	return result
}
