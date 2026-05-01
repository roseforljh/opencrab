package gateway

import (
	"context"
	"net/http"
)

type MessagesRequest struct {
	Model           string
	Stream          bool
	Body            []byte
	ContentType     string
	Accept          string
	Authorization   string
	Headers         http.Header
	MaxTokens       int
	UpstreamFamily  string
	UpstreamURL     string
	UpstreamAPIKey  string
	RouteCandidates []UpstreamRouteCandidate
}

func (s *Service) Messages(ctx context.Context, request MessagesRequest) (*ProxyResponse, error) {
	if len(request.RouteCandidates) > 0 {
		var lastErr error
		for index, candidate := range request.RouteCandidates {
			attempt := request
			attempt.UpstreamFamily = candidate.Family
			attempt.UpstreamURL = candidate.URL
			attempt.UpstreamAPIKey = candidate.APIKey
			response, err := s.provider.Messages(ctx, attempt)
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
	return s.provider.Messages(ctx, request)
}
