package gateway

import (
	"context"
	"net/http"
)

type GenerateContentRequest struct {
	Model           string
	Stream          bool
	Body            []byte
	ContentType     string
	Accept          string
	Headers         http.Header
	UpstreamURL     string
	UpstreamAPIKey  string
	RouteCandidates []UpstreamRouteCandidate
}

type GenerateContentProvider interface {
	GenerateContent(ctx context.Context, request GenerateContentRequest) (*ProxyResponse, error)
}

func (s *Service) GenerateContent(ctx context.Context, request GenerateContentRequest) (*ProxyResponse, error) {
	if len(request.RouteCandidates) > 0 {
		var lastErr error
		for index, candidate := range request.RouteCandidates {
			attempt := request
			attempt.UpstreamURL = candidate.URL
			attempt.UpstreamAPIKey = candidate.APIKey
			response, err := s.provider.GenerateContent(ctx, attempt)
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
	return s.provider.GenerateContent(ctx, request)
}
