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

func ForwardClaudeCountTokens(ctx context.Context, client *http.Client, channel domain.UpstreamChannel, body []byte, anthropicVersion string, anthropicBeta string) (*domain.ProxyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildClaudeCountTokensURL(channel.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 Claude count_tokens 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", channel.APIKey)
	if strings.TrimSpace(anthropicVersion) == "" {
		anthropicVersion = anthropicVersionFallback()
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	if strings.TrimSpace(anthropicBeta) != "" {
		req.Header.Set("anthropic-beta", anthropicBeta)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Claude count_tokens 失败: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Claude count_tokens 响应失败: %w", err)
	}
	headers := make(map[string][]string, len(resp.Header))
	for key, values := range resp.Header {
		headers[key] = append([]string(nil), values...)
	}
	return &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: bodyBytes}, nil
}

func anthropicVersionFallback() string {
	return anthropicVersion
}
