package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
	"opencrab/internal/provider"
)

type Surface struct {
	Protocol  domain.Protocol
	Operation domain.ProtocolOperation
}

type NormalizeOptions struct {
	PathModel   string
	Headers     map[string]string
	ForceStream bool
}

type ExecutorPayload struct {
	Body   []byte
	URL    string
	Stream bool
}

func NormalizeGatewayRequest(surface Surface, body []byte, options NormalizeOptions) (domain.GatewayRequest, error) {
	var (
		unified domain.UnifiedChatRequest
		session *domain.GatewaySessionState
		err     error
	)

	switch surface.Operation {
	case domain.ProtocolOperationOpenAIChatCompletions:
		unified, err = provider.DecodeOpenAIChatRequest(body)
	case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
		unified, err = provider.DecodeOpenAIResponsesRequest(body)
		if err == nil {
			session, err = provider.DecodeOpenAIResponsesSession(body)
		}
		if err == nil && surface.Protocol == domain.ProtocolCodex {
			unified.Protocol = domain.ProtocolCodex
		}
	case domain.ProtocolOperationClaudeMessages:
		unified, err = provider.DecodeClaudeChatRequest(body)
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		unified, err = provider.DecodeGeminiChatRequest(body, options.PathModel)
		if err == nil && (options.ForceStream || surface.Operation == domain.ProtocolOperationGeminiStreamGenerate) {
			unified.Stream = true
		}
	default:
		return domain.GatewayRequest{}, fmt.Errorf("unsupported ingress surface: %s", surface.Operation)
	}
	if err != nil {
		return domain.GatewayRequest{}, err
	}

	request := unifiedToGatewayRequest(unified, options.Headers, session)
	request.Operation = surface.Operation
	return request, nil
}

func BuildExecutorPayload(surface Surface, req domain.UnifiedChatRequest, session *domain.GatewaySessionState, endpoint string, upstreamModel string) (ExecutorPayload, error) {
	switch surface.Operation {
	case domain.ProtocolOperationOpenAIChatCompletions:
		payloadReq := req
		payloadReq.Protocol = domain.ProtocolOpenAI
		body, err := provider.EncodeOpenAIChatRequest(payloadReq)
		if err != nil {
			return ExecutorPayload{}, err
		}
		return ExecutorPayload{Body: body, URL: buildChatCompletionsURL(endpoint), Stream: req.Stream}, nil
	case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
		payloadReq := req
		payloadReq.Protocol = domain.ProtocolOpenAI
		body, err := provider.EncodeOpenAIResponsesRequest(payloadReq, session)
		if err != nil {
			return ExecutorPayload{}, err
		}
		return ExecutorPayload{Body: body, URL: buildResponsesURL(endpoint), Stream: req.Stream}, nil
	case domain.ProtocolOperationClaudeMessages:
		payloadReq := req
		payloadReq.Protocol = domain.ProtocolClaude
		body, err := provider.EncodeClaudeChatRequest(payloadReq)
		if err != nil {
			return ExecutorPayload{}, err
		}
		return ExecutorPayload{Body: body, URL: buildClaudeMessagesURL(endpoint), Stream: req.Stream}, nil
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		payloadReq := req
		payloadReq.Protocol = domain.ProtocolGemini
		body, err := provider.EncodeGeminiChatRequest(payloadReq)
		if err != nil {
			return ExecutorPayload{}, err
		}
		url := buildGeminiGenerateContentURL(endpoint, upstreamModel)
		stream := req.Stream || surface.Operation == domain.ProtocolOperationGeminiStreamGenerate
		if stream {
			url = buildGeminiStreamGenerateContentURL(endpoint, upstreamModel)
		}
		return ExecutorPayload{Body: body, URL: url, Stream: stream}, nil
	default:
		return ExecutorPayload{}, fmt.Errorf("unsupported target surface: %s", surface.Operation)
	}
}

func DecodeUpstreamResponse(providerName string, body []byte) (domain.UnifiedChatResponse, error) {
	return DecodeUpstreamResponseForOperation(providerName, "", body)
}

func DecodeUpstreamResponseForOperation(providerName string, operation domain.ProtocolOperation, body []byte) (domain.UnifiedChatResponse, error) {
	switch operation {
	case domain.ProtocolOperationOpenAIChatCompletions:
		return provider.DecodeOpenAIChatResponse(body)
	case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
		return provider.DecodeOpenAIResponsesResponse(body)
	case domain.ProtocolOperationClaudeMessages:
		return provider.DecodeClaudeChatResponse(body)
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		return provider.DecodeGeminiChatResponse(body)
	}
	switch domain.NormalizeProvider(providerName) {
	case "claude":
		return provider.DecodeClaudeChatResponse(body)
	case "gemini":
		return provider.DecodeGeminiChatResponse(body)
	default:
		if looksLikeResponsesPayload(body) {
			return provider.DecodeOpenAIResponsesResponse(body)
		}
		return provider.DecodeOpenAIChatResponse(body)
	}
}

func RenderClientResponse(surface Surface, resp domain.UnifiedChatResponse, stream bool) ([]byte, map[string][]string, error) {
	switch surface.Operation {
	case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
		if stream {
			body, err := provider.EncodeOpenAIResponsesStream(toOpenAIProtocolResponse(resp))
			if err != nil {
				return nil, nil, err
			}
			return body, streamHeaders(), nil
		}
		body, err := provider.EncodeOpenAIResponsesResponse(toOpenAIProtocolResponse(resp))
		if err != nil {
			return nil, nil, err
		}
		return body, jsonHeaders(), nil
	case domain.ProtocolOperationClaudeMessages:
		if stream {
			body, err := provider.EncodeClaudeChatStream(toClaudeProtocolResponse(resp))
			if err != nil {
				return nil, nil, err
			}
			return body, streamHeaders(), nil
		}
		body, err := provider.EncodeClaudeChatResponse(toClaudeProtocolResponse(resp))
		if err != nil {
			return nil, nil, err
		}
		return body, jsonHeaders(), nil
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		if stream {
			body, err := provider.EncodeGeminiChatStream(toGeminiProtocolResponse(resp))
			if err != nil {
				return nil, nil, err
			}
			return body, streamHeaders(), nil
		}
		body, err := provider.EncodeGeminiChatResponse(toGeminiProtocolResponse(resp))
		if err != nil {
			return nil, nil, err
		}
		return body, jsonHeaders(), nil
	default:
		if stream {
			body, err := provider.EncodeOpenAIChatStream(toOpenAIProtocolResponse(resp))
			if err != nil {
				return nil, nil, err
			}
			return body, streamHeaders(), nil
		}
		body, err := provider.EncodeOpenAIChatResponse(toOpenAIProtocolResponse(resp))
		if err != nil {
			return nil, nil, err
		}
		return body, jsonHeaders(), nil
	}
}

func unifiedToGatewayRequest(unified domain.UnifiedChatRequest, headers map[string]string, session *domain.GatewaySessionState) domain.GatewayRequest {
	messages := make([]domain.GatewayMessage, 0, len(unified.Messages))
	for _, message := range unified.Messages {
		messages = append(messages, domain.GatewayMessage{
			Role:      message.Role,
			Parts:     message.Parts,
			ToolCalls: message.ToolCalls,
			InputItem: message.InputItem,
			Metadata:  message.Metadata,
		})
	}
	policy := domain.GatewayToolCallReject
	if len(unified.Tools) > 0 || hasToolMessages(messages) || session != nil && len(session.ToolResults) > 0 {
		policy = domain.GatewayToolCallAllow
	}
	return domain.GatewayRequest{
		Protocol:       unified.Protocol,
		Model:          unified.Model,
		Stream:         unified.Stream,
		Messages:       messages,
		Tools:          unified.Tools,
		Metadata:       unified.Metadata,
		ToolCallPolicy: policy,
		RequestHeaders: headers,
		Session:        session,
	}
}

func hasToolMessages(messages []domain.GatewayMessage) bool {
	for _, message := range messages {
		if len(message.ToolCalls) > 0 || strings.EqualFold(message.Role, "tool") {
			return true
		}
	}
	return false
}

func looksLikeResponsesPayload(body []byte) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	var objectType string
	if err := json.Unmarshal(raw["object"], &objectType); err == nil && strings.TrimSpace(objectType) == "response" {
		return true
	}
	_, hasOutput := raw["output"]
	return hasOutput
}

func toOpenAIProtocolResponse(resp domain.UnifiedChatResponse) domain.UnifiedChatResponse {
	resp.Protocol = domain.ProtocolOpenAI
	return resp
}

func toClaudeProtocolResponse(resp domain.UnifiedChatResponse) domain.UnifiedChatResponse {
	resp.Protocol = domain.ProtocolClaude
	return resp
}

func toGeminiProtocolResponse(resp domain.UnifiedChatResponse) domain.UnifiedChatResponse {
	resp.Protocol = domain.ProtocolGemini
	return resp
}

func jsonHeaders() map[string][]string {
	return map[string][]string{"Content-Type": {"application/json"}}
}

func streamHeaders() map[string][]string {
	return map[string][]string{"Content-Type": {"text/event-stream"}, "Cache-Control": {"no-cache"}, "Connection": {"keep-alive"}}
}

func buildChatCompletionsURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") || strings.HasSuffix(trimmed, "/v4") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}

func buildResponsesURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/responses") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") || strings.HasSuffix(trimmed, "/v4") {
		return trimmed + "/responses"
	}
	return trimmed + "/v1/responses"
}

func buildClaudeMessagesURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/messages") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/messages"
	}
	return trimmed + "/v1/messages"
}

func buildGeminiGenerateContentURL(endpoint string, model string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.Contains(trimmed, ":generateContent") {
		return trimmed
	}
	if strings.Contains(trimmed, "/models/") {
		return trimmed + ":generateContent"
	}
	if strings.HasSuffix(trimmed, "/v1beta") {
		return trimmed + "/models/" + model + ":generateContent"
	}
	return trimmed + "/v1beta/models/" + model + ":generateContent"
}

func buildGeminiStreamGenerateContentURL(endpoint string, model string) string {
	base := buildGeminiGenerateContentURL(endpoint, model)
	base = strings.Replace(base, ":generateContent", ":streamGenerateContent", 1)
	if strings.Contains(base, "?") {
		return base + "&alt=sse"
	}
	return base + "?alt=sse"
}
