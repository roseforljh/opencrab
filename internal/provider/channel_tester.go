package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opencrab/internal/domain"
)

type ChannelTester struct {
	client *http.Client
}

func NewChannelTester(client *http.Client) *ChannelTester {
	return &ChannelTester{client: client}
}

func (t *ChannelTester) TestChannel(ctx context.Context, channel domain.UpstreamChannel, model string) (domain.ChannelTestResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	providerName := normalizeProvider(channel.Provider)
	usedModel := strings.TrimSpace(model)
	if usedModel == "" {
		usedModel = defaultTestModel(providerName)
	}

	req, err := buildTestRequest(ctx, channel, providerName, usedModel)
	if err != nil {
		return domain.ChannelTestResult{}, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return domain.ChannelTestResult{}, fmt.Errorf("测试请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	result := domain.ChannelTestResult{
		Channel:    channel.Name,
		Provider:   channel.Provider,
		Model:      usedModel,
		StatusCode: resp.StatusCode,
		Details: map[string]any{
			"request_url": req.URL.String(),
		},
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		result.Message = message
		result.Details["upstream_status"] = resp.StatusCode
		result.Details["error_summary"] = message
		return result, fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message)
	}

	result.Message = "连接成功"
	result.Details["upstream_status"] = resp.StatusCode
	return result, nil
}

func buildTestRequest(ctx context.Context, channel domain.UpstreamChannel, providerName string, model string) (*http.Request, error) {
	switch providerName {
	case "claude":
		return buildClaudeTestRequest(ctx, channel, model)
	case "gemini":
		return buildGeminiTestRequest(ctx, channel, model)
	default:
		return buildOpenAICompatibleTestRequest(ctx, channel, model)
	}
}

func buildOpenAICompatibleTestRequest(ctx context.Context, channel domain.UpstreamChannel, model string) (*http.Request, error) {
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "ping",
		}},
		"max_tokens": 8,
		"stream":     false,
	})
	if err != nil {
		return nil, fmt.Errorf("构造测试请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildChatCompletionsURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建测试请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	return req, nil
}

func buildClaudeTestRequest(ctx context.Context, channel domain.UpstreamChannel, model string) (*http.Request, error) {
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 8,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "ping",
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("构造 Claude 测试请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildClaudeMessagesURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Claude 测试请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", channel.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return req, nil
}

func buildGeminiTestRequest(ctx context.Context, channel domain.UpstreamChannel, model string) (*http.Request, error) {
	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{{
			"parts": []map[string]string{{
				"text": "ping",
			}},
		}},
		"generationConfig": map[string]any{
			"maxOutputTokens": 8,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("构造 Gemini 测试请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildGeminiGenerateContentURL(channel.Endpoint, model), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini 测试请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", channel.APIKey)
	return req, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude", "anthropic":
		return "claude"
	case "gemini", "google":
		return "gemini"
	case "glm", "zhipu":
		return "glm"
	case "kimi", "moonshot":
		return "kimi"
	case "minimax", "mini_max", "mini max":
		return "minimax"
	case "openrouter":
		return "openrouter"
	default:
		return "openai"
	}
}

func defaultTestModel(providerName string) string {
	switch providerName {
	case "claude":
		return "claude-3-5-haiku-latest"
	case "gemini":
		return "gemini-2.0-flash"
	case "glm":
		return "glm-4-flash"
	case "kimi":
		return "moonshot-v1-8k"
	case "minimax":
		return "MiniMax-Text-01"
	case "openrouter":
		return "openai/gpt-4o-mini"
	default:
		return "gpt-4o-mini"
	}
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

func buildClaudeCountTokensURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/messages/count_tokens") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/messages") {
		return trimmed + "/count_tokens"
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/messages/count_tokens"
	}
	return trimmed + "/v1/messages/count_tokens"
}

func buildGeminiGenerateContentURL(endpoint string, model string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	escapedModel := url.PathEscape(model)
	if strings.Contains(trimmed, ":generateContent") {
		return trimmed
	}
	if strings.Contains(trimmed, "/models/") {
		return trimmed + ":generateContent"
	}
	if strings.HasSuffix(trimmed, "/v1beta") || strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/models/" + escapedModel + ":generateContent"
	}
	return trimmed + "/v1beta/models/" + escapedModel + ":generateContent"
}
