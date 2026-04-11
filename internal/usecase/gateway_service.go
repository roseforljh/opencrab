package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"opencrab/internal/domain"
)

type GatewayService struct {
	routes    domain.GatewayRouteStore
	executors map[string]domain.Executor
	logger    domain.GatewayAttemptLogger
}

func NewGatewayService(routes domain.GatewayRouteStore, executors map[string]domain.Executor, logger domain.GatewayAttemptLogger) *GatewayService {
	return &GatewayService{routes: routes, executors: executors, logger: logger}
}

func (s *GatewayService) Execute(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ProxyResponse, error) {
	routeList, err := s.routes.ListEnabledRoutesByModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(routeList) == 0 {
		return nil, domain.ErrNoAvailableRoute(req.Model)
	}

	var lastErr error
	for idx, route := range routeList {
		executor := s.executors[domain.NormalizeProvider(route.Channel.Provider)]
		if executor == nil {
			lastErr = fmt.Errorf("provider %s 没有可用执行器", route.Channel.Provider)
			s.logAttempt(ctx, requestID, req, route, idx+1, 0, false, false, false, lastErr.Error(), "", "")
			continue
		}

		result, execErr := executor.Execute(ctx, domain.ExecutorRequest{
			Channel:       route.Channel,
			UpstreamModel: route.UpstreamModel,
			Request:       req,
		})
		if execErr == nil {
			s.logAttempt(ctx, requestID, req, route, idx+1, result.Response.StatusCode, false, false, true, "", marshalGatewayRequest(req), truncateGatewayBody(result.Response.Body))
			if result.Response.Headers == nil {
				result.Response.Headers = map[string][]string{}
			}
			result.Response.Headers["X-Opencrab-Channel"] = []string{route.Channel.Name}
			return result.Response, nil
		}

		detail := domain.AsExecutionError(execErr)
		s.logAttempt(ctx, requestID, req, route, idx+1, detail.StatusCode, detail.Retryable, detail.StreamStarted, false, detail.Error(), marshalGatewayRequest(req), "")
		lastErr = execErr
		if !detail.Retryable || detail.StreamStarted {
			return nil, execErr
		}
	}

	if lastErr == nil {
		lastErr = domain.ErrNoAvailableRoute(req.Model)
	}
	return nil, lastErr
}

func (s *GatewayService) logAttempt(ctx context.Context, requestID string, req domain.GatewayRequest, route domain.GatewayRoute, attempt int, statusCode int, retryable bool, streamStarted bool, success bool, errMessage string, requestBody string, responseBody string) {
	if s.logger == nil {
		return
	}
	_ = s.logger.LogGatewayAttempt(ctx, domain.GatewayAttemptLog{
		RequestID:     requestID,
		Model:         req.Model,
		UpstreamModel: route.UpstreamModel,
		Channel:       route.Channel.Name,
		Provider:      route.Channel.Provider,
		Attempt:       attempt,
		StatusCode:    statusCode,
		Retryable:     retryable,
		StreamStarted: streamStarted,
		Success:       success,
		ErrorMessage:  errMessage,
		RequestBody:   requestBody,
		ResponseBody:  responseBody,
	})
}

func marshalGatewayRequest(req domain.GatewayRequest) string {
	body, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(body)
}

func truncateGatewayBody(body []byte) string {
	const limit = 512
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit])
}
