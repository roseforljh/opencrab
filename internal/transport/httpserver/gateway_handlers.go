package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func HandleGatewayChatCompletions(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, err := executeGatewayRequest(deps, req, decodeOpenAIGatewayRequest)
		if err != nil {
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result)
	}
}

func HandleClaudeMessages(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, err := executeGatewayRequest(deps, req, decodeClaudeGatewayRequest)
		if err != nil {
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result)
	}
}

func HandleGeminiGenerateContent(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, err := executeGatewayRequest(deps, req, decodeGeminiGatewayRequest)
		if err != nil {
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result)
	}
}

func HandleGeminiStreamGenerateContent(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, err := executeGatewayRequest(deps, req, decodeGeminiGatewayRequest)
		if err != nil {
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result)
	}
}

type gatewayDecoder func(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error)

func executeGatewayRequest(deps Dependencies, req *http.Request, decode gatewayDecoder) ([]byte, *domain.ExecutionResult, domain.Protocol, error) {
	if deps.ExecuteGateway == nil || deps.CopyProxy == nil || deps.CopyStream == nil {
		return nil, nil, "", fmt.Errorf("gateway handler not configured")
	}
	if deps.VerifyAPIKey == nil {
		return nil, nil, "", fmt.Errorf("api key verifier not configured")
	}

	rawKey := extractGatewayAPIKey(req)
	if rawKey == "" {
		return nil, nil, "", fmt.Errorf("缺少 API Key")
	}
	allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
	if err != nil {
		return nil, nil, "", err
	}
	if !allowed {
		return nil, nil, "", fmt.Errorf("API Key 无效或已禁用")
	}
	if deps.CheckRateLimit != nil && !deps.CheckRateLimit(rawKey) {
		return nil, nil, "", fmt.Errorf("请求过于频繁，请稍后再试")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, "", fmt.Errorf("读取请求体失败")
	}
	gatewayReq, protocol, err := decode(body, req)
	if err != nil {
		return nil, nil, "", err
	}
	result, err := deps.ExecuteGateway(req.Context(), middleware.GetReqID(req.Context()), gatewayReq)
	if err != nil {
		return body, nil, protocol, err
	}
	return body, result, protocol, nil
}

func decodeOpenAIGatewayRequest(body []byte, _ *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeOpenAIChatRequest(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return unifiedToGatewayRequest(unified), domain.ProtocolOpenAI, nil
}

func decodeClaudeGatewayRequest(body []byte, _ *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeClaudeChatRequest(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return unifiedToGatewayRequest(unified), domain.ProtocolClaude, nil
}

func decodeGeminiGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeGeminiChatRequest(body, chi.URLParam(req, "model"))
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	if strings.Contains(req.URL.Path, ":streamGenerateContent") {
		unified.Stream = true
	}
	return unifiedToGatewayRequest(unified), domain.ProtocolGemini, nil
}

func unifiedToGatewayRequest(unified domain.UnifiedChatRequest) domain.GatewayRequest {
	messages := make([]domain.GatewayMessage, 0, len(unified.Messages))
	for _, message := range unified.Messages {
		messages = append(messages, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, Metadata: message.Metadata})
	}
	return domain.GatewayRequest{Protocol: unified.Protocol, Model: unified.Model, Stream: unified.Stream, Messages: messages, Tools: unified.Tools, Metadata: unified.Metadata, ToolCallPolicy: domain.GatewayToolCallReject}
}

func writeGatewayResult(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, protocol domain.Protocol, result *domain.ExecutionResult) {
	startedAt := time.Now()
	if result.Stream != nil {
		if err := deps.CopyStream(w, result.Stream); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logGatewayRequestSummary(deps, req, requestBody, result.Stream.StatusCode, result.Stream.Headers, nil, startedAt)
		return
	}
	if result.Response == nil {
		http.Error(w, "empty gateway result", http.StatusBadGateway)
		return
	}
	resp := encodeGatewayResponseForProtocol(result.Response, protocol)
	if err := deps.CopyProxy(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logGatewayRequestSummary(deps, req, requestBody, resp.StatusCode, resp.Headers, resp.Body, startedAt)
}

func encodeGatewayResponseForProtocol(resp *domain.ProxyResponse, protocol domain.Protocol) *domain.ProxyResponse {
	providerName := normalizedHeaderProvider(resp.Headers)
	if protocolMatchesProvider(protocol, providerName) {
		return resp
	}
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		return resp
	}
	encoded, err := encodeUnifiedByProtocol(protocol, unified)
	if err != nil {
		return resp
	}
	headers := cloneHeaderMap(resp.Headers)
	headers["Content-Type"] = []string{"application/json"}
	return &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: encoded}
}

func decodeUnifiedByProvider(providerName string, body []byte) (domain.UnifiedChatResponse, error) {
	switch providerName {
	case "claude":
		return provider.DecodeClaudeChatResponse(body)
	case "gemini":
		return provider.DecodeGeminiChatResponse(body)
	default:
		return provider.DecodeOpenAIChatResponse(body)
	}
}

func encodeUnifiedByProtocol(protocol domain.Protocol, resp domain.UnifiedChatResponse) ([]byte, error) {
	switch protocol {
	case domain.ProtocolClaude:
		return provider.EncodeClaudeChatResponse(resp)
	case domain.ProtocolGemini:
		return provider.EncodeGeminiChatResponse(resp)
	default:
		return provider.EncodeOpenAIChatResponse(resp)
	}
}

func normalizedHeaderProvider(headers map[string][]string) string {
	return domain.NormalizeProvider(firstHeaderValue(headers, "X-Opencrab-Provider"))
}

func protocolMatchesProvider(protocol domain.Protocol, providerName string) bool {
	switch protocol {
	case domain.ProtocolClaude:
		return providerName == "claude"
	case domain.ProtocolGemini:
		return providerName == "gemini"
	default:
		return providerName == "" || providerName == "openai" || providerName == "openrouter" || providerName == "glm" || providerName == "kimi" || providerName == "minimax"
	}
}

func cloneHeaderMap(headers map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func logGatewayRequestSummary(deps Dependencies, req *http.Request, requestBody []byte, statusCode int, headers map[string][]string, responseBody []byte, startedAt time.Time) {
	if deps.CreateRequestLog == nil {
		return
	}
	channelName := firstHeaderValue(headers, "X-Opencrab-Channel")
	if channelName == "" {
		channelName = "default-channel"
	}
	modelName := extractModel(requestBody)
	usage := usageMetrics{}
	loggedResponseBody := ""
	if len(responseBody) > 0 {
		usage = extractUsageMetrics(responseBody)
		loggedResponseBody = truncateLogBody(string(responseBody))
	}
	details := marshalLogDetails(map[string]any{
		"request_path":      req.URL.Path,
		"channel":           channelName,
		"model":             modelName,
		"request_body":      truncateLogBody(string(requestBody)),
		"response_body":     loggedResponseBody,
		"response_status":   statusCode,
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
		StatusCode:       statusCode,
		LatencyMs:        time.Since(startedAt).Milliseconds(),
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CacheHit:         usage.CacheHit,
		RequestBody:      truncateLogBody(string(requestBody)),
		ResponseBody:     loggedResponseBody,
		Details:          details,
		CreatedAt:        time.Now().Format(time.RFC3339),
	})
}

func renderGatewayError(deps Dependencies, w http.ResponseWriter, err error) {
	message := err.Error()
	if strings.Contains(message, "缺少 API Key") || strings.Contains(message, "无效或已禁用") {
		http.Error(w, message, http.StatusUnauthorized)
		return
	}
	if strings.Contains(message, "请求过于频繁") {
		http.Error(w, message, http.StatusTooManyRequests)
		return
	}
	if deps.RenderProxyError != nil {
		deps.RenderProxyError(w, err)
		return
	}
	http.Error(w, message, http.StatusBadGateway)
}
