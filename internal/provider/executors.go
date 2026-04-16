package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
)

const anthropicVersion = "2023-06-01"

type OpenAIExecutor struct {
	client *http.Client
}

type ClaudeExecutor struct {
	client *http.Client
}

type GeminiExecutor struct {
	client *http.Client
}

func NewOpenAIExecutor(client *http.Client) *OpenAIExecutor {
	return &OpenAIExecutor{client: client}
}

func NewClaudeExecutor(client *http.Client) *ClaudeExecutor {
	return &ClaudeExecutor{client: client}
}

func NewGeminiExecutor(client *http.Client) *GeminiExecutor {
	return &GeminiExecutor{client: client}
}

func (e *OpenAIExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := EncodeOpenAIChatRequest(toUnifiedRequest(input, domain.ProtocolOpenAI, domain.NormalizeProvider(input.Channel.Provider)))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 OpenAI 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, buildChatCompletionsURL(input.Channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 OpenAI 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"authorization": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func (e *ClaudeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := EncodeClaudeChatRequest(toUnifiedRequest(input, domain.ProtocolClaude, domain.NormalizeProvider(input.Channel.Provider)))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Claude 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, buildClaudeMessagesURL(input.Channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Claude 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-api-key", input.Channel.APIKey)
	version := anthropicVersion
	if custom := strings.TrimSpace(input.Request.RequestHeaders["anthropic-version"]); custom != "" {
		version = custom
	}
	req.Header.Set("anthropic-version", version)
	if beta := strings.TrimSpace(input.Request.RequestHeaders["anthropic-beta"]); beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"x-api-key": {}, "content-type": {}, "accept": {}, "anthropic-version": {}, "anthropic-beta": {}})
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func (e *GeminiExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := EncodeGeminiChatRequest(toUnifiedRequest(input, domain.ProtocolGemini, domain.NormalizeProvider(input.Channel.Provider)))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Gemini 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	url := buildGeminiGenerateContentURL(input.Channel.Endpoint, input.UpstreamModel)
	if input.Request.Stream {
		url = buildGeminiStreamGenerateContentURL(input.Channel.Endpoint, input.UpstreamModel)
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Gemini 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-goog-api-key", input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"x-goog-api-key": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func toUnifiedRequest(input domain.ExecutorRequest, protocol domain.Protocol, targetProvider string) domain.UnifiedChatRequest {
	messages := make([]domain.UnifiedMessage, 0, len(input.Request.Messages))
	for _, message := range input.Request.Messages {
		messages = append(messages, domain.UnifiedMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, Metadata: message.Metadata})
	}
	return domain.UnifiedChatRequest{
		Protocol: protocol,
		Model:    input.UpstreamModel,
		Stream:   input.Request.Stream,
		Messages: messages,
		Tools:    input.Request.Tools,
		Metadata: sanitizeRequestMetadataForTarget(cloneRawMap(input.Request.Metadata), protocol, targetProvider),
	}
}

func sanitizeRequestMetadataForTarget(metadata map[string]json.RawMessage, protocol domain.Protocol, targetProvider string) map[string]json.RawMessage {
	if len(metadata) == 0 {
		return metadata
	}
	cleaned := cloneRawMap(metadata)
	deleteKeys := func(keys ...string) {
		for _, key := range keys {
			delete(cleaned, key)
		}
	}
	switch protocol {
	case domain.ProtocolOpenAI:
		deleteKeys("previous_response_id", "include", "reasoning", "store", "instructions", "metadata", "generate")
	case domain.ProtocolGemini:
		deleteKeys("previous_response_id", "include", "reasoning", "store", "instructions", "tool_choice", "parallel_tool_calls", "thinking", "cache_control")
	case domain.ProtocolClaude:
		deleteKeys("previous_response_id", "include", "store", "parallel_tool_calls")
	}
	_ = targetProvider
	return cleaned
}

func doExecutorRequest(client *http.Client, req *http.Request, stream bool) (*domain.ExecutionResult, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("请求上游失败: %w", err), 0, true, false)
	}
	headers := cloneHeaders(resp.Header)
	if stream && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: resp.StatusCode, Headers: headers, Body: resp.Body}}, nil
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("读取上游响应失败: %w", readErr), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := string(body)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, domain.NewExecutionError(fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}
	return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: body}}, nil
}

func cloneHeaders(header http.Header) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func applyRequestHeaders(req *http.Request, headers map[string]string, skip map[string]struct{}) {
	for key, value := range headers {
		if _, blocked := skip[strings.ToLower(strings.TrimSpace(key))]; blocked {
			continue
		}
		req.Header.Set(key, value)
	}
}
