package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"opencrab/internal/domain"

	"github.com/go-chi/chi/v5"
)

// writeJSON 把结构化数据写成 JSON 响应。
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		http.Error(w, "写入 JSON 响应失败", http.StatusInternalServerError)
	}
}

func parseInt64Param(req *http.Request, key string) (int64, error) {
	value := chi.URLParam(req, key)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的 %s 参数", key)
	}
	return id, nil
}

func validateCreateChannelInput(input domain.CreateChannelInput) error {
	return validateChannelInput(input.Name, input.Provider, input.Endpoint, input.APIKey, input.ModelIDs, true)
}

func validateUpdateChannelInput(input domain.UpdateChannelInput) error {
	return validateChannelInput(input.Name, input.Provider, input.Endpoint, input.APIKey, input.ModelIDs, false)
}

func validateChannelInput(name string, provider string, endpoint string, apiKey string, modelIDs []string, requireAPIKey bool) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("渠道名称不能为空")
	}
	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf("渠道类型不能为空")
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("请求地址不能为空")
	}
	if requireAPIKey && strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if len(modelIDs) == 0 {
		return fmt.Errorf("至少添加一个模型 ID")
	}
	seen := make(map[string]struct{}, len(modelIDs))
	for _, modelID := range modelIDs {
		normalized := strings.TrimSpace(modelID)
		if normalized == "" {
			return fmt.Errorf("模型 ID 不能为空")
		}
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("模型 ID 不能重复")
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func validateCreateAPIKeyInput(input domain.CreateAPIKeyInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("密钥名称不能为空")
	}
	return nil
}

func validateUpdateSystemSettingInput(input *domain.UpdateSystemSettingInput) error {
	input.Key = strings.TrimSpace(input.Key)
	input.Value = strings.TrimSpace(input.Value)
	if input.Key == "" {
		return fmt.Errorf("设置项 key 不能为空")
	}
	if strings.HasPrefix(input.Key, "admin.") {
		return fmt.Errorf("认证安全相关配置不能通过系统设置接口修改")
	}
	switch input.Key {
	case "dispatch.redis_enabled", "dispatch.redis_tls_enabled", "dispatch.pause_dispatch", "dispatch.dead_letter_enabled", "dispatch.metrics_enabled", "dispatch.show_worker_status", "dispatch.show_queue_depth", "dispatch.show_retry_rate", "gateway.sticky_enabled":
		lower := strings.ToLower(input.Value)
		if lower != "true" && lower != "false" {
			return fmt.Errorf("%s 只能是 true 或 false", input.Key)
		}
		input.Value = lower
	case "dispatch.queue_mode":
		if input.Value != "single" && input.Value != "priority" {
			return fmt.Errorf("dispatch.queue_mode 只能是 single 或 priority")
		}
	case "dispatch.backoff_mode":
		if input.Value != "fixed" && input.Value != "exponential" {
			return fmt.Errorf("dispatch.backoff_mode 只能是 fixed 或 exponential")
		}
	case "gateway.routing_strategy":
		if input.Value != "sequential" && input.Value != "round_robin" {
			return fmt.Errorf("gateway.routing_strategy 只能是 sequential 或 round_robin")
		}
	case "gateway.sticky_key_source":
		if input.Value != "auto" && input.Value != "header" && input.Value != "metadata" {
			return fmt.Errorf("gateway.sticky_key_source 只能是 auto、header 或 metadata")
		}
	case "dispatch.redis_db", "dispatch.worker_concurrency", "dispatch.sync_hold_ms", "dispatch.backlog_cap", "dispatch.max_attempts", "dispatch.backoff_delay_ms", "dispatch.queue_ttl_s", "dispatch.long_wait_threshold_s", "gateway.cooldown_seconds":
		parsed, err := strconv.Atoi(input.Value)
		if err != nil || parsed < 0 {
			return fmt.Errorf("%s 必须是非负整数", input.Key)
		}
		input.Value = strconv.Itoa(parsed)
	case "dispatch.retry_reserve_ratio":
		parsed, err := strconv.ParseFloat(input.Value, 64)
		if err != nil || math.IsNaN(parsed) || parsed < 0 || parsed > 1 {
			return fmt.Errorf("dispatch.retry_reserve_ratio 必须在 0 到 1 之间")
		}
		input.Value = strconv.FormatFloat(parsed, 'f', 2, 64)
	}
	return nil
}

func validateCapabilityProfileInput(scopeType string, scopeKey string, operation string) error {
	if strings.TrimSpace(scopeType) == "" {
		return fmt.Errorf("scope_type 不能为空")
	}
	if strings.TrimSpace(scopeKey) == "" {
		return fmt.Errorf("scope_key 不能为空")
	}
	if strings.TrimSpace(operation) == "" {
		return fmt.Errorf("operation 不能为空")
	}
	return nil
}

func extractModel(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "unknown-model"
	}
	model, _ := payload["model"].(string)
	if strings.TrimSpace(model) == "" {
		return "unknown-model"
	}
	return model
}

func extractModelFromRequest(path string, body []byte) string {
	if model := strings.TrimSpace(extractModel(body)); model != "" && model != "unknown-model" {
		return model
	}

	const geminiPrefix = "/v1beta/models/"
	if strings.HasPrefix(path, geminiPrefix) {
		modelPart := strings.TrimPrefix(path, geminiPrefix)
		if idx := strings.Index(modelPart, ":"); idx >= 0 {
			modelPart = modelPart[:idx]
		}
		if decoded, err := url.PathUnescape(strings.TrimSpace(modelPart)); err == nil && decoded != "" {
			return decoded
		}
		if strings.TrimSpace(modelPart) != "" {
			return strings.TrimSpace(modelPart)
		}
	}

	return "unknown-model"
}

func fallbackLogModel(resultModel string, inputModel string) string {
	if strings.TrimSpace(resultModel) != "" {
		return resultModel
	}
	if strings.TrimSpace(inputModel) != "" {
		return strings.TrimSpace(inputModel)
	}
	return "unknown-model"
}

func fallbackLogChannel(resultChannel string, id int64) string {
	if strings.TrimSpace(resultChannel) != "" {
		return resultChannel
	}
	return fmt.Sprintf("channel-%d", id)
}

func fallbackStatusCode(statusCode int) int {
	if statusCode > 0 {
		return statusCode
	}
	return http.StatusBadGateway
}

func marshalLogDetails(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func truncateLogBody(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 1200 {
		return trimmed
	}
	return trimmed[:1200] + "..."
}

func firstHeaderValue(headers map[string][]string, key string) string {
	if headers == nil {
		return ""
	}
	values := headers[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func extractGatewayAPIKey(req *http.Request) string {
	if req == nil {
		return ""
	}

	if value := strings.TrimSpace(req.Header.Get("Authorization")); value != "" {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(strings.ToLower(value), strings.ToLower(bearerPrefix)) {
			return strings.TrimSpace(value[len(bearerPrefix):])
		}
	}

	if value := strings.TrimSpace(req.Header.Get("x-api-key")); value != "" {
		return value
	}

	if value := strings.TrimSpace(req.Header.Get("x-goog-api-key")); value != "" {
		return value
	}

	if value := strings.TrimSpace(req.URL.Query().Get("key")); value != "" {
		return value
	}

	return ""
}

func ExtractGatewayAPIKey(req *http.Request) string {
	return extractGatewayAPIKey(req)
}

func resolveGatewayAPIKey(deps Dependencies, req *http.Request) (string, domain.APIKeyScope, error) {
	rawKey := extractGatewayAPIKey(req)
	if rawKey == "" {
		return "", domain.APIKeyScope{}, fmt.Errorf("缺少 API Key")
	}
	if deps.ResolveAPIKey != nil {
		scope, found, err := deps.ResolveAPIKey(req.Context(), rawKey)
		if err != nil {
			return "", domain.APIKeyScope{}, err
		}
		if !found {
			return "", domain.APIKeyScope{}, fmt.Errorf("API Key 无效或已禁用")
		}
		return rawKey, scope, nil
	}
	if deps.VerifyAPIKey == nil {
		return "", domain.APIKeyScope{}, fmt.Errorf("api key verifier not configured")
	}
	allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
	if err != nil {
		return "", domain.APIKeyScope{}, err
	}
	if !allowed {
		return "", domain.APIKeyScope{}, fmt.Errorf("API Key 无效或已禁用")
	}
	return rawKey, domain.APIKeyScope{}, nil
}

func applyAPIKeyScopeToGatewayRequest(gatewayReq *domain.GatewayRequest, scope domain.APIKeyScope) error {
	if gatewayReq == nil {
		return nil
	}
	if len(scope.ModelAliases) > 0 && !scopeListContains(scope.ModelAliases, gatewayReq.Model) {
		return fmt.Errorf("API Key 不允许访问模型 %s", gatewayReq.Model)
	}
	if len(scope.ChannelNames) == 0 && len(scope.ModelAliases) == 0 {
		return nil
	}
	gatewayReq.APIKeyScope = &scope
	return nil
}

func scopeListContains(values []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == normalizedTarget {
			return true
		}
	}
	return false
}

func extractGatewaySessionID(req *http.Request) string {
	if req == nil {
		return ""
	}
	for _, key := range []string{"X-Claude-Code-Session-Id", "X-Session-ID"} {
		if value := strings.TrimSpace(req.Header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func extractStringRawValue(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	var direct string
	if err := json.Unmarshal(value, &direct); err == nil {
		return strings.TrimSpace(direct)
	}
	var wrapped struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(value, &wrapped); err == nil {
		return strings.TrimSpace(wrapped.ID)
	}
	return ""
}

func extractSessionAffinityKey(req *http.Request, gatewayReq domain.GatewayRequest, settings domain.GatewayRuntimeSettings) string {
	if req == nil || !settings.StickyEnabled {
		return ""
	}
	readHeader := func() string {
		return extractGatewaySessionID(req)
	}
	readMetadata := func() string {
		for _, key := range []string{"session_id", "conversation_id", "user_id"} {
			if gatewayReq.Metadata == nil {
				continue
			}
			if value, ok := gatewayReq.Metadata[key]; ok {
				if extracted := extractStringRawValue(value); extracted != "" {
					return extracted
				}
			}
		}
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(settings.StickyKeySource)) {
	case "header":
		return readHeader()
	case "metadata":
		return readMetadata()
	default:
		if value := readHeader(); value != "" {
			return value
		}
		return readMetadata()
	}
}

func ExtractSessionAffinityKey(req *http.Request, gatewayReq domain.GatewayRequest, settings domain.GatewayRuntimeSettings) string {
	return extractSessionAffinityKey(req, gatewayReq, settings)
}

type usageMetrics struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheHit         bool
}

func extractUsageMetrics(body []byte) usageMetrics {
	var payload struct {
		Usage struct {
			PromptTokens         int64 `json:"prompt_tokens"`
			CompletionTokens     int64 `json:"completion_tokens"`
			TotalTokens          int64 `json:"total_tokens"`
			PromptCacheHitTokens int64 `json:"prompt_cache_hit_tokens"`
			PromptTokensDetails  struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return usageMetrics{}
	}

	totalTokens := payload.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = payload.Usage.PromptTokens + payload.Usage.CompletionTokens
	}

	return usageMetrics{
		PromptTokens:     payload.Usage.PromptTokens,
		CompletionTokens: payload.Usage.CompletionTokens,
		TotalTokens:      totalTokens,
		CacheHit:         payload.Usage.PromptCacheHitTokens > 0 || payload.Usage.PromptTokensDetails.CachedTokens > 0,
	}
}
