package openai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

type Client struct {
	timeout time.Duration
	http    *http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		timeout: timeout,
		http: &http.Client{
			Timeout: 0,
		},
	}
}

func (c *Client) ChatCompletions(ctx context.Context, request gateway.ChatCompletionsRequest) (*gateway.ProxyResponse, error) {
	upstreamURL := strings.TrimSpace(request.UpstreamURL)
	if upstreamURL == "" {
		return nil, &gateway.RoutingError{Message: "No enabled OpenAI-compatible route configured for model"}
	}
	requestContext := ctx
	var cancel context.CancelFunc
	if !request.Stream && c.timeout > 0 {
		requestContext, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	upstreamRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, upstreamURL, bytes.NewReader(request.Body))
	if err != nil {
		return nil, fmt.Errorf("创建上游请求失败: %w", err)
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

	if auth := c.resolveAuthorization(request); auth != "" {
		upstreamRequest.Header.Set("Authorization", auth)
	}

	copyOptionalHeader(upstreamRequest.Header, request.Headers, "OpenAI-Beta")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "OpenAI-Organization")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "OpenAI-Project")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "Idempotency-Key")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "Prefer")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "User")

	response, err := c.http.Do(upstreamRequest)
	if err != nil {
		return nil, classifyTransportError(err)
	}
	return &gateway.ProxyResponse{
		StatusCode:     response.StatusCode,
		Header:         response.Header.Clone(),
		Body:           response.Body,
		Stream:         isEventStream(response.Header.Get("Content-Type")),
		UpstreamFamily: "openai",
	}, nil
}

func (c *Client) resolveAuthorization(request gateway.ChatCompletionsRequest) string {
	if key := strings.TrimSpace(request.UpstreamAPIKey); key != "" {
		return "Bearer " + key
	}
	if auth := strings.TrimSpace(request.Authorization); auth != "" {
		return auth
	}
	if key := strings.TrimSpace(request.Headers.Get("X-API-Key")); key != "" {
		return "Bearer " + key
	}
	return ""
}

func copyOptionalHeader(target http.Header, source http.Header, key string) {
	if value := strings.TrimSpace(source.Get(key)); value != "" {
		target.Set(key, value)
	}
}

func DrainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
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
