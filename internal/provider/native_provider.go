package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opencrab/internal/domain"

	"github.com/gorilla/websocket"
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

func ForwardGeminiCachedContentCreate(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, body []byte) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildGeminiCachedContentCreateURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini cachedContent 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", channel.APIKey)
	return doProxyRequest(client, req)
}

func ForwardGeminiCachedContentGet(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, name string) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildGeminiCachedContentGetURL(channel.Endpoint, name), nil)
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini cachedContent get 请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", channel.APIKey)
	return doProxyRequest(client, req)
}

func ForwardOpenAIRealtimeClientSecret(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, body []byte) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildOpenAIRealtimeClientSecretsURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 OpenAI realtime client secret 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	return doProxyRequest(client, req)
}

func ForwardOpenAIRealtimeCall(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, contentType string, body []byte, rawQuery string) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	targetURL := buildOpenAIRealtimeCallsURL(channel.Endpoint)
	if strings.TrimSpace(rawQuery) != "" {
		targetURL += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 OpenAI realtime calls 请求失败: %w", err)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/sdp, application/json")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	return doProxyRequest(client, req)
}

func DialOpenAIRealtime(ctx context.Context, channel domain.UpstreamChannel, req *http.Request) (*websocket.Conn, *http.Response, error) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+channel.APIKey)
	if req != nil {
		if beta := strings.TrimSpace(req.Header.Get("OpenAI-Beta")); beta != "" {
			headers.Set("OpenAI-Beta", beta)
		}
		if origin := strings.TrimSpace(req.Header.Get("Origin")); origin != "" {
			headers.Set("Origin", origin)
		}
	}
	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
	}
	targetURL := buildOpenAIRealtimeWebSocketURL(channel.Endpoint, "")
	if req != nil && req.URL != nil {
		targetURL = buildOpenAIRealtimeWebSocketURL(channel.Endpoint, req.URL.RawQuery)
	}
	return dialer.DialContext(ctx, targetURL, headers)
}

func buildGeminiCachedContentCreateURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/v1beta") {
		return trimmed + "/cachedContents"
	}
	return trimmed + "/v1beta/cachedContents"
}

func buildGeminiCachedContentGetURL(endpoint string, name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.TrimPrefix(trimmed, "/")
	base := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(base, "/v1beta") {
		return base + "/" + trimmed
	}
	return base + "/v1beta/" + trimmed
}

func buildOpenAIRealtimeClientSecretsURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/realtime/client_secrets"
	}
	return trimmed + "/v1/realtime/client_secrets"
}

func buildOpenAIRealtimeCallsURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/realtime/calls"
	}
	return trimmed + "/v1/realtime/calls"
}

func buildOpenAIRealtimeWebSocketURL(endpoint string, rawQuery string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		trimmed += "/realtime"
	} else {
		trimmed += "/v1/realtime"
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	}
	if strings.TrimSpace(rawQuery) != "" {
		parsed.RawQuery = strings.TrimPrefix(rawQuery, "?")
	}
	return parsed.String()
}
