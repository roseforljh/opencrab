package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
)

type GatewayService struct {
	routes    domain.GatewayRouteStore
	executors map[string]domain.Executor
	logger    domain.GatewayAttemptLogger
	strategy  domain.RoutingStrategyStore
	cursors   domain.RoutingCursorStore
}

type routePlan struct {
	route         domain.GatewayRoute
	cursorKey     string
	groupSize     int
	originalIndex int
	startIndex    int
	groupID       string
	bucketName    string
}

func NewGatewayService(routes domain.GatewayRouteStore, executors map[string]domain.Executor, logger domain.GatewayAttemptLogger, strategy domain.RoutingStrategyStore, cursors domain.RoutingCursorStore) *GatewayService {
	return &GatewayService{routes: routes, executors: executors, logger: logger, strategy: strategy, cursors: cursors}
}

func (s *GatewayService) Execute(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
	routeList, err := s.routes.ListEnabledRoutesByModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(routeList) == 0 {
		return nil, domain.ErrNoAvailableRoute(req.Model)
	}
	plans, strategy, err := s.arrangeRoutes(ctx, req, routeList)
	if err != nil {
		return nil, err
	}

	var lastErr error
	var activeGroup *routePlan
	advanceGroup := func(plan *routePlan, selectedIndex int) {
		if plan == nil || plan.cursorKey == "" || s.cursors == nil || plan.groupSize <= 1 {
			return
		}
		_ = s.cursors.AdvanceRoutingCursor(ctx, plan.cursorKey, plan.groupSize, selectedIndex)
	}
	for idx, plan := range plans {
		if activeGroup != nil && activeGroup.groupID != "" && plan.groupID != activeGroup.groupID {
			advanceGroup(activeGroup, activeGroup.startIndex)
		}
		activeGroup = &plan
		route := plan.route
		executor := s.executors[domain.NormalizeProvider(route.Channel.Provider)]
		if executor == nil {
			lastErr = fmt.Errorf("provider %s 没有可用执行器", route.Channel.Provider)
			s.logAttempt(ctx, requestID, req, plan, strategy, idx+1, 0, false, false, false, lastErr.Error(), "", "")
			continue
		}

		result, execErr := executor.Execute(ctx, domain.ExecutorRequest{
			Channel:       route.Channel,
			UpstreamModel: route.UpstreamModel,
			Request:       req,
		})
		if execErr == nil {
			if result.Stream != nil {
				if result.Stream.Headers == nil {
					result.Stream.Headers = map[string][]string{}
				}
				result.Stream.Headers["X-Opencrab-Channel"] = []string{route.Channel.Name}
				result.Stream.Headers["X-Opencrab-Provider"] = []string{domain.NormalizeProvider(route.Channel.Provider)}
				s.logAttempt(ctx, requestID, req, plan, strategy, idx+1, result.Stream.StatusCode, false, false, true, "", marshalGatewayRequest(req), "")
				advanceGroup(&plan, plan.originalIndex)
				return result, nil
			}
			if result.Response != nil {
				s.logAttempt(ctx, requestID, req, plan, strategy, idx+1, result.Response.StatusCode, false, false, true, "", marshalGatewayRequest(req), truncateGatewayBody(result.Response.Body))
				if result.Response.Headers == nil {
					result.Response.Headers = map[string][]string{}
				}
				result.Response.Headers["X-Opencrab-Channel"] = []string{route.Channel.Name}
				result.Response.Headers["X-Opencrab-Provider"] = []string{domain.NormalizeProvider(route.Channel.Provider)}
				advanceGroup(&plan, plan.originalIndex)
				return result, nil
			}
			lastErr = fmt.Errorf("执行结果既无 Response 也无 Stream")
			s.logAttempt(ctx, requestID, req, plan, strategy, idx+1, 0, false, false, false, lastErr.Error(), marshalGatewayRequest(req), "")
			continue
		}

		detail := domain.AsExecutionError(execErr)
		s.logAttempt(ctx, requestID, req, plan, strategy, idx+1, detail.StatusCode, detail.Retryable, detail.StreamStarted, false, detail.Error(), marshalGatewayRequest(req), "")
		lastErr = execErr
		if !detail.Retryable || detail.StreamStarted {
			advanceGroup(&plan, plan.originalIndex)
			return nil, execErr
		}
	}
	advanceGroup(activeGroup, activeGroup.startIndex)

	if lastErr == nil {
		lastErr = domain.ErrNoAvailableRoute(req.Model)
	}
	return nil, lastErr
}

func (s *GatewayService) arrangeRoutes(ctx context.Context, req domain.GatewayRequest, routes []domain.GatewayRoute) ([]routePlan, domain.RoutingStrategy, error) {
	strategy := domain.RoutingStrategySequential
	if s.strategy != nil {
		value, err := s.strategy.GetRoutingStrategy(ctx)
		if err != nil {
			return nil, "", err
		}
		strategy = value
	}
	ordered := prioritizeRoutesByInvocationMode(routes, req.Protocol)
	if strategy != domain.RoutingStrategyRoundRobin || s.cursors == nil {
		return wrapSequentialPlans(ordered, req.Protocol), strategy, nil
	}
	plans, err := s.rotateRoutes(ctx, req, ordered)
	return plans, strategy, err
}

func wrapSequentialPlans(routes []domain.GatewayRoute, protocol domain.Protocol) []routePlan {
	plans := make([]routePlan, 0, len(routes))
	for _, route := range routes {
		plans = append(plans, routePlan{route: route, bucketName: invocationBucketName(route, protocol), groupSize: 1, originalIndex: 0, startIndex: 0})
	}
	return plans
}

func (s *GatewayService) rotateRoutes(ctx context.Context, req domain.GatewayRequest, routes []domain.GatewayRoute) ([]routePlan, error) {
	buckets := splitInvocationBuckets(routes, req.Protocol)
	ordered := make([]routePlan, 0, len(routes))
	for _, bucketName := range []string{"matched", "neutral", "mismatched"} {
		bucketRoutes := buckets[bucketName]
		if len(bucketRoutes) == 0 {
			continue
		}
		priorityGroups := groupRoutesByPriority(bucketRoutes)
		for _, group := range priorityGroups {
			rotated, err := s.rotatePriorityGroup(ctx, req, bucketName, group)
			if err != nil {
				return nil, err
			}
			ordered = append(ordered, rotated...)
		}
	}
	return ordered, nil
}

func splitInvocationBuckets(routes []domain.GatewayRoute, protocol domain.Protocol) map[string][]domain.GatewayRoute {
	buckets := map[string][]domain.GatewayRoute{
		"matched":    {},
		"neutral":    {},
		"mismatched": {},
	}
	for _, route := range routes {
		switch routeInvocationMatch(route.InvocationMode, protocol) {
		case 2:
			buckets["matched"] = append(buckets["matched"], route)
		case 1:
			buckets["neutral"] = append(buckets["neutral"], route)
		default:
			buckets["mismatched"] = append(buckets["mismatched"], route)
		}
	}
	return buckets
}

func groupRoutesByPriority(routes []domain.GatewayRoute) [][]domain.GatewayRoute {
	if len(routes) == 0 {
		return nil
	}
	groups := make([][]domain.GatewayRoute, 0)
	current := []domain.GatewayRoute{routes[0]}
	currentPriority := routes[0].Priority
	for _, route := range routes[1:] {
		if route.Priority != currentPriority {
			groups = append(groups, current)
			current = []domain.GatewayRoute{route}
			currentPriority = route.Priority
			continue
		}
		current = append(current, route)
	}
	groups = append(groups, current)
	return groups
}

func (s *GatewayService) rotatePriorityGroup(ctx context.Context, req domain.GatewayRequest, bucketName string, routes []domain.GatewayRoute) ([]routePlan, error) {
	if len(routes) <= 1 {
		groupID := buildRoutingCursorKey(req.Model, req.Protocol, bucketName, routes[0].Priority)
		return []routePlan{{route: routes[0], cursorKey: groupID, groupSize: 1, originalIndex: 0, startIndex: 0, groupID: groupID}}, nil
	}
	routeKey := buildRoutingCursorKey(req.Model, req.Protocol, bucketName, routes[0].Priority)
	startIndex, err := s.cursors.GetRoutingCursor(ctx, routeKey)
	if err != nil {
		return nil, err
	}
	startIndex = startIndex % len(routes)
	plans := make([]routePlan, 0, len(routes))
	for offset := 0; offset < len(routes); offset++ {
		originalIndex := (startIndex + offset) % len(routes)
		plans = append(plans, routePlan{route: routes[originalIndex], cursorKey: routeKey, groupSize: len(routes), originalIndex: originalIndex, startIndex: startIndex, groupID: routeKey, bucketName: bucketName})
	}
	return plans, nil
}

func invocationBucketName(route domain.GatewayRoute, protocol domain.Protocol) string {
	switch routeInvocationMatch(route.InvocationMode, protocol) {
	case 2:
		return "matched"
	case 1:
		return "neutral"
	default:
		return "mismatched"
	}
}

func buildRoutingCursorKey(model string, protocol domain.Protocol, bucketName string, priority int) string {
	return fmt.Sprintf("%s|%s|%s|%d", strings.TrimSpace(model), strings.TrimSpace(string(protocol)), bucketName, priority)
}

func prioritizeRoutesByInvocationMode(routes []domain.GatewayRoute, protocol domain.Protocol) []domain.GatewayRoute {
	if len(routes) <= 1 {
		return routes
	}
	matched := make([]domain.GatewayRoute, 0, len(routes))
	neutral := make([]domain.GatewayRoute, 0, len(routes))
	mismatched := make([]domain.GatewayRoute, 0, len(routes))
	for _, route := range routes {
		switch routeInvocationMatch(route.InvocationMode, protocol) {
		case 2:
			matched = append(matched, route)
		case 1:
			neutral = append(neutral, route)
		default:
			mismatched = append(mismatched, route)
		}
	}
	ordered := make([]domain.GatewayRoute, 0, len(routes))
	ordered = append(ordered, matched...)
	ordered = append(ordered, neutral...)
	ordered = append(ordered, mismatched...)
	return ordered
}

func routeInvocationMatch(mode string, protocol domain.Protocol) int {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" || normalizedMode == "auto" || normalizedMode == "any" {
		return 1
	}
	normalizedProtocol := strings.ToLower(strings.TrimSpace(string(protocol)))
	if normalizedProtocol == "" {
		return 1
	}
	if normalizedMode == normalizedProtocol {
		return 2
	}
	return 0
}

func (s *GatewayService) logAttempt(ctx context.Context, requestID string, req domain.GatewayRequest, plan routePlan, strategy domain.RoutingStrategy, attempt int, statusCode int, retryable bool, streamStarted bool, success bool, errMessage string, requestBody string, responseBody string) {
	if s.logger == nil {
		return
	}
	route := plan.route
	_ = s.logger.LogGatewayAttempt(ctx, domain.GatewayAttemptLog{
		RequestID:        requestID,
		Model:            req.Model,
		UpstreamModel:    route.UpstreamModel,
		Channel:          route.Channel.Name,
		Provider:         route.Channel.Provider,
		RoutingStrategy:  string(strategy),
		InvocationBucket: plan.bucketName,
		PriorityTier:     route.Priority,
		CandidateCount:   plan.groupSize,
		SelectedIndex:    plan.originalIndex,
		Attempt:          attempt,
		StatusCode:       statusCode,
		Retryable:        retryable,
		StreamStarted:    streamStarted,
		Success:          success,
		ErrorMessage:     errMessage,
		RequestBody:      requestBody,
		ResponseBody:     responseBody,
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
