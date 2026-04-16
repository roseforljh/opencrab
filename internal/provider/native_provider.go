package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
)

type ClaudeNativeProvider struct {
	client *http.Client
}

type GeminiNativeProvider struct {
	client *http.Client
}

func NewClaudeNativeProvider(client *http.Client) *ClaudeNativeProvider {
	return &ClaudeNativeProvider{client: client}
}

func NewGeminiNativeProvider(client *http.Client) *GeminiNativeProvider {
	return &GeminiNativeProvider{client: client}
}

func (p *ClaudeNativeProvider) ForwardMessages(ctx context.Context, channel domain.UpstreamChannel, body []byte) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildClaudeMessagesURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Claude 上游请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-api-key", channel.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	return doProxyRequest(p.client, req)
}

func (p *GeminiNativeProvider) ForwardGenerateContent(ctx context.Context, channel domain.UpstreamChannel, model string, body []byte, stream bool) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("Gemini model 不能为空")
	}

	url := buildGeminiGenerateContentURL(channel.Endpoint, model)
	if stream {
		url = buildGeminiStreamGenerateContentURL(channel.Endpoint, model)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini 上游请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-goog-api-key", channel.APIKey)
	return doProxyRequest(p.client, req)
}

func doProxyRequest(client *http.Client, req *http.Request) (*domain.ProxyResponse, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取上游响应失败: %w", err)
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
		return nil, fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message)
	}

	return &domain.ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}

func buildGeminiStreamGenerateContentURL(endpoint string, model string) string {
	base := buildGeminiGenerateContentURL(endpoint, model)
	base = strings.Replace(base, ":generateContent", ":streamGenerateContent", 1)
	if strings.Contains(base, "?") {
		return base + "&alt=sse"
	}
	return base + "?alt=sse"
}
