package httpserver

import (
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// HandleGatewayChatCompletions 统一处理 OpenAI 兼容的代理请求。
//
// 该 Handler 负责：
// 1. API Key 校验。
// 2. 限流检查。
// 3. 读取并复用请求体。
// 4. 调用上游转发。
// 5. 记录请求日志（包含 Token 消耗）。
func HandleGatewayChatCompletions(deps Dependencies) http.HandlerFunc {
	return handleGatewayProxy(deps, "/v1/chat/completions", func(_ *http.Request, body []byte) string {
		return extractModel(body)
	}, func(req *http.Request, body []byte) (*domain.ProxyResponse, error) {
		return deps.ProxyChat(req.Context(), body)
	})
}

func HandleClaudeMessages(deps Dependencies) http.HandlerFunc {
	return handleGatewayProxy(deps, "/v1/messages", func(_ *http.Request, body []byte) string {
		return extractModel(body)
	}, func(req *http.Request, body []byte) (*domain.ProxyResponse, error) {
		return deps.ProxyClaude(req.Context(), body)
	})
}

func HandleGeminiGenerateContent(deps Dependencies, stream bool) http.HandlerFunc {
	requestPath := "/v1beta/models/{model}:generateContent"
	if stream {
		requestPath = "/v1beta/models/{model}:streamGenerateContent"
	}
	return handleGatewayProxy(deps, requestPath, extractGeminiModel, func(req *http.Request, body []byte) (*domain.ProxyResponse, error) {
		return deps.ProxyGemini(req.Context(), chi.URLParam(req, "model"), body, stream)
	})
}

func handleGatewayProxy(
	deps Dependencies,
	fallbackPath string,
	extractModelName func(req *http.Request, body []byte) string,
	proxy func(req *http.Request, body []byte) (*domain.ProxyResponse, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if proxy == nil || deps.CopyProxy == nil {
			http.Error(w, "proxy handler not configured", http.StatusNotImplemented)
			return
		}

		if deps.VerifyAPIKey == nil {
			http.Error(w, "api key verifier not configured", http.StatusNotImplemented)
			return
		}

		rawKey := extractGatewayAPIKey(req)
		if rawKey == "" {
			http.Error(w, "缺少 API Key", http.StatusUnauthorized)
			return
		}

		allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "API Key 无效或已禁用", http.StatusUnauthorized)
			return
		}

		if deps.CheckRateLimit != nil && !deps.CheckRateLimit(rawKey) {
			http.Error(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
			return
		}

		startedAt := time.Now()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "读取请求体失败", http.StatusBadRequest)
			return
		}

		resp, err := proxy(req, body)
		if err != nil {
			if deps.RenderProxyError != nil {
				deps.RenderProxyError(w, err)
			} else {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return
		}

		channelName := firstHeaderValue(resp.Headers, "X-Opencrab-Channel")
		if channelName == "" {
			channelName = "default-channel"
		}
		requestPath := req.URL.Path
		if requestPath == "" {
			requestPath = fallbackPath
		}
		modelName := extractModelName(req, body)
		usage := extractUsageMetrics(resp.Body)
		responseBody := truncateLogBody(string(resp.Body))

		if err := deps.CopyProxy(w, resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if deps.CreateRequestLog != nil {
			details := marshalLogDetails(map[string]any{
				"request_path":      requestPath,
				"channel":           channelName,
				"model":             modelName,
				"request_body":      truncateLogBody(string(body)),
				"response_body":     responseBody,
				"response_status":   resp.StatusCode,
				"prompt_tokens":     usage.PromptTokens,
				"completion_tokens": usage.CompletionTokens,
				"total_tokens":      usage.TotalTokens,
				"cache_hit":         usage.CacheHit,
				"test_mode":         false,
			})
			_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
				RequestID:        middleware.GetReqID(req.Context()),
				Model:            modelName,
				Channel:          channelName,
				StatusCode:       resp.StatusCode,
				LatencyMs:        time.Since(startedAt).Milliseconds(),
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
				CacheHit:         usage.CacheHit,
				RequestBody:      truncateLogBody(string(body)),
				ResponseBody:     responseBody,
				Details:          details,
				CreatedAt:        time.Now().Format(time.RFC3339),
			})
		}
	}
}

func extractGeminiModel(req *http.Request, body []byte) string {
	model := strings.TrimSpace(chi.URLParam(req, "model"))
	if model != "" {
		return model
	}
	return extractModel(body)
}
