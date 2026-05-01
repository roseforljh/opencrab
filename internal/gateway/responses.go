package gateway

import "context"

func (s *Service) Responses(ctx context.Context, request ResponsesRequest) (*ProxyResponse, error) {
	if len(request.RouteCandidates) > 0 {
		var lastErr error
		for index, candidate := range request.RouteCandidates {
			attempt := request
			attempt.UpstreamFamily = candidate.Family
			attempt.UpstreamOperation = candidate.Operation
			attempt.UpstreamURL = candidate.URL
			attempt.UpstreamAPIKey = candidate.APIKey
			response, err := s.provider.Responses(ctx, attempt)
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
	return s.provider.Responses(ctx, request)
}
