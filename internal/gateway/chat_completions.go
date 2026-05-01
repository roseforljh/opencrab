package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
)

type ChatCompletionsRequest struct {
	Model           string
	Stream          bool
	Body            []byte
	ContentType     string
	Accept          string
	Authorization   string
	Headers         http.Header
	UpstreamFamily  string
	UpstreamURL     string
	UpstreamAPIKey  string
	RouteCandidates []UpstreamRouteCandidate
}

type UpstreamRouteCandidate struct {
	Family string
	URL    string
	APIKey string
}

type ProxyResponse struct {
	StatusCode     int
	Header         http.Header
	Body           io.ReadCloser
	Stream         bool
	UpstreamFamily string
}

type ChatCompletionsProvider interface {
	ChatCompletions(ctx context.Context, request ChatCompletionsRequest) (*ProxyResponse, error)
}

type MessagesProvider interface {
	Messages(ctx context.Context, request MessagesRequest) (*ProxyResponse, error)
}

type Provider interface {
	ChatCompletionsProvider
	MessagesProvider
	GenerateContentProvider
}

type Service struct {
	provider Provider
}

type TransportError struct {
	Timeout bool
	Cause   error
}

type RoutingError struct {
	Message string
}

type RequestError struct {
	StatusCode     int
	Message        string
	UpstreamFamily string
}

func (e *RoutingError) Error() string {
	if e == nil || e.Message == "" {
		return "upstream route not configured"
	}
	return e.Message
}

func (e *RequestError) Error() string {
	if e == nil || e.Message == "" {
		return "invalid request"
	}
	return e.Message
}

func (e *TransportError) Error() string {
	if e == nil || e.Cause == nil {
		return "gateway transport error"
	}
	return e.Cause.Error()
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ChatCompletions(ctx context.Context, request ChatCompletionsRequest) (*ProxyResponse, error) {
	if len(request.RouteCandidates) > 0 {
		var lastErr error
		for index, candidate := range request.RouteCandidates {
			attempt := request
			attempt.UpstreamFamily = candidate.Family
			attempt.UpstreamURL = candidate.URL
			attempt.UpstreamAPIKey = candidate.APIKey
			response, err := s.provider.ChatCompletions(ctx, attempt)
			if err != nil {
				lastErr = err
				if index < len(request.RouteCandidates)-1 && shouldRetryCandidate(err, nil) {
					continue
				}
				return nil, err
			}
			if index < len(request.RouteCandidates)-1 && shouldRetryCandidate(nil, response) {
				drainProxyBody(response.Body)
				continue
			}
			return response, nil
		}
		if lastErr != nil {
			return nil, lastErr
		}
	}
	return s.provider.ChatCompletions(ctx, request)
}

func shouldRetryCandidate(err error, response *ProxyResponse) bool {
	if err != nil {
		transportError := &TransportError{}
		return errors.As(err, &transportError)
	}
	if response == nil {
		return false
	}
	return response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
}

func drainProxyBody(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
