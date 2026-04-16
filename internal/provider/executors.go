package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
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
	body, err := EncodeOpenAIChatRequest(toUnifiedRequest(input, domain.ProtocolOpenAI))
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
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func (e *ClaudeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := EncodeClaudeChatRequest(toUnifiedRequest(input, domain.ProtocolClaude))
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
	req.Header.Set("anthropic-version", anthropicVersion)
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func (e *GeminiExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := EncodeGeminiChatRequest(toUnifiedRequest(input, domain.ProtocolGemini))
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
	return doExecutorRequest(e.client, req, input.Request.Stream)
}

func toUnifiedRequest(input domain.ExecutorRequest, protocol domain.Protocol) domain.UnifiedChatRequest {
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
		Metadata: cloneRawMap(input.Request.Metadata),
	}
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
