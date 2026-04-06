package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"opencrab/internal/domain"
)

type OpenAICompatibleProvider struct {
	client *http.Client
}

func NewOpenAICompatibleProvider(client *http.Client) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{client: client}
}

func (p *OpenAICompatibleProvider) ForwardChatCompletions(ctx context.Context, channel domain.UpstreamChannel, body []byte) (*domain.ProxyResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := ForwardChatCompletions(ctx, p.client, channel, body)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取上游响应失败: %w", err)
		}

		headers := make(map[string][]string, len(resp.Header))
		for key, values := range resp.Header {
			headers[key] = append([]string(nil), values...)
		}

		return &domain.ProxyResponse{
			StatusCode: resp.StatusCode,
			Headers:    headers,
			Body:       bodyBytes,
		}, nil
	}

	return nil, fmt.Errorf("请求上游失败: %w", lastErr)
}

func ForwardChatCompletions(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, body []byte) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	url := buildChatCompletionsURL(channel.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建上游请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}

	return resp, nil
}

func RenderProxyError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = io.WriteString(w, fmt.Sprintf(`{"error":{"message":%q}}`, err.Error()))
}

func DumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return ""
	}
	return string(dump)
}

func CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("写入代理响应失败: %w", err)
	}

	return nil
}

func buildChatCompletionsURL(endpoint string) string {
	trimmed := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}
