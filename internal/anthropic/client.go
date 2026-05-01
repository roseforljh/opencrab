package anthropic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

type Client struct {
	version string
	timeout time.Duration
	http    *http.Client
}

func NewClient(version string, timeout time.Duration) *Client {
	return &Client{
		version: strings.TrimSpace(version),
		timeout: timeout,
		http: &http.Client{
			Timeout: 0,
		},
	}
}

func (c *Client) Messages(ctx context.Context, request gateway.MessagesRequest) (*gateway.ProxyResponse, error) {
	upstreamURL := strings.TrimSpace(request.UpstreamURL)
	if upstreamURL == "" {
		return nil, &gateway.RoutingError{Message: "No enabled Claude route configured for model"}
	}
	requestContext := ctx
	var cancel context.CancelFunc
	if !request.Stream && c.timeout > 0 {
		requestContext, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	upstreamRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, upstreamURL, bytes.NewReader(request.Body))
	if err != nil {
		return nil, fmt.Errorf("创建 Anthropic 上游请求失败: %w", err)
	}

	contentType := strings.TrimSpace(request.ContentType)
	if contentType == "" {
		contentType = "application/json"
	}
	upstreamRequest.Header.Set("Content-Type", contentType)

	accept := strings.TrimSpace(request.Accept)
	if accept == "" {
		accept = "application/json"
	}
	upstreamRequest.Header.Set("Accept", accept)

	if apiKey := c.resolveAPIKey(request); apiKey != "" {
		upstreamRequest.Header.Set("X-API-Key", apiKey)
	}
	if c.version != "" {
		upstreamRequest.Header.Set("Anthropic-Version", c.version)
	}

	copyOptionalHeader(upstreamRequest.Header, request.Headers, "Anthropic-Beta")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "Idempotency-Key")

	response, err := c.http.Do(upstreamRequest)
	if err != nil {
		return nil, classifyTransportError(err)
	}

	return &gateway.ProxyResponse{
		StatusCode:     response.StatusCode,
		Header:         response.Header.Clone(),
		Body:           response.Body,
		Stream:         isEventStream(response.Header.Get("Content-Type")),
		UpstreamFamily: "claude",
	}, nil
}

func (c *Client) resolveAPIKey(request gateway.MessagesRequest) string {
	if key := strings.TrimSpace(request.UpstreamAPIKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(request.Headers.Get("X-API-Key")); key != "" {
		return key
	}
	if auth := strings.TrimSpace(request.Authorization); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func copyOptionalHeader(target http.Header, source http.Header, key string) {
	if value := strings.TrimSpace(source.Get(key)); value != "" {
		target.Set(key, value)
	}
}

func classifyTransportError(err error) error {
	if err == nil {
		return nil
	}
	transportError := &gateway.TransportError{Cause: err}
	if errors.Is(err, context.DeadlineExceeded) {
		transportError.Timeout = true
		return transportError
	}
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Timeout() {
		transportError.Timeout = true
	}
	return transportError
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}
