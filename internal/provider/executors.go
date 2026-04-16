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
	body, err := json.Marshal(map[string]any{
		"model":    input.UpstreamModel,
		"messages": toOpenAIMessages(input.Request.Messages),
		"stream":   input.Request.Stream,
	})
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
	return doExecutorRequest(e.client, req)
}

func (e *ClaudeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := buildClaudeExecutionBody(input)
	if err != nil {
		return nil, err
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
	return doExecutorRequest(e.client, req)
}

func (e *GeminiExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	body, err := buildGeminiExecutionBody(input)
	if err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, buildGeminiGenerateContentURL(input.Channel.Endpoint, input.UpstreamModel), bytes.NewReader(body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Gemini 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-goog-api-key", input.Channel.APIKey)
	return doExecutorRequest(e.client, req)
}

func buildClaudeExecutionBody(input domain.ExecutorRequest) ([]byte, error) {
	if len(input.Request.Messages) == 0 {
		return nil, domain.NewExecutionError(fmt.Errorf("请求缺少消息"), 0, false, false)
	}
	if input.Request.ToolCallPolicy != "" && input.Request.ToolCallPolicy != domain.GatewayToolCallReject {
		return nil, domain.NewExecutionError(fmt.Errorf("Claude 暂不支持工具调用转换"), 0, false, false)
	}

	body, err := json.Marshal(map[string]any{
		"model":      input.UpstreamModel,
		"max_tokens": 1024,
		"stream":     input.Request.Stream,
		"messages":   toOpenAIMessages(input.Request.Messages),
	})
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Claude 请求失败: %w", err), 0, false, false)
	}
	return body, nil
}

func buildGeminiExecutionBody(input domain.ExecutorRequest) ([]byte, error) {
	if len(input.Request.Messages) == 0 {
		return nil, domain.NewExecutionError(fmt.Errorf("请求缺少消息"), 0, false, false)
	}
	if input.Request.ToolCallPolicy != "" && input.Request.ToolCallPolicy != domain.GatewayToolCallReject {
		return nil, domain.NewExecutionError(fmt.Errorf("Gemini 暂不支持工具调用转换"), 0, false, false)
	}

	contents := make([]map[string]any, 0, len(input.Request.Messages))
	for _, message := range input.Request.Messages {
		role := "user"
		if strings.EqualFold(strings.TrimSpace(message.Role), "assistant") {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]string{{"text": message.Text}},
		})
	}

	body, err := json.Marshal(map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"candidateCount": 1,
		},
	})
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Gemini 请求失败: %w", err), 0, false, false)
	}
	return body, nil
}

func doExecutorRequest(client *http.Client, req *http.Request) (*domain.ExecutionResult, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("请求上游失败: %w", err), 0, true, false)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("读取上游响应失败: %w", readErr), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}

	headers := make(map[string][]string, len(resp.Header))
	for key, values := range resp.Header {
		headers[key] = append([]string(nil), values...)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, domain.NewExecutionError(fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}

	return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: body}}, nil
}

func toOpenAIMessages(messages []domain.GatewayMessage) []map[string]string {
	items := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		items = append(items, map[string]string{
			"role":    message.Role,
			"content": message.Text,
		})
	}
	return items
}
