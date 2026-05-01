package gemini

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
	timeout time.Duration
	http    *http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		timeout: timeout,
		http:    &http.Client{Timeout: 0},
	}
}

func (c *Client) GenerateContent(ctx context.Context, request gateway.GenerateContentRequest) (*gateway.ProxyResponse, error) {
	upstreamURL := strings.TrimSpace(request.UpstreamURL)
	if upstreamURL == "" {
		return nil, &gateway.RoutingError{Message: "No enabled gemini route configured for model"}
	}
	requestContext := ctx
	var cancel context.CancelFunc
	if !request.Stream && c.timeout > 0 {
		requestContext, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	upstreamRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, upstreamURL, bytes.NewReader(request.Body))
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini 上游请求失败: %w", err)
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

	if apiKey := strings.TrimSpace(request.UpstreamAPIKey); apiKey != "" {
		upstreamRequest.Header.Set("X-Goog-Api-Key", apiKey)
	}

	copyOptionalHeader(upstreamRequest.Header, request.Headers, "X-Goog-Api-Client")
	copyOptionalHeader(upstreamRequest.Header, request.Headers, "X-Request-Id")

	response, err := c.http.Do(upstreamRequest)
	if err != nil {
		return nil, classifyTransportError(err)
	}

	return &gateway.ProxyResponse{
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
		Body:       response.Body,
		Stream:     strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream"),
	}, nil
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
