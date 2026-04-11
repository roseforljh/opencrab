package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"opencrab/internal/domain"

	"github.com/go-chi/chi/v5"
)

// writeJSON 负责把结构化数据写成 JSON 响应。
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
