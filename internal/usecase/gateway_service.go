package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
	"opencrab/internal/planner"
	"opencrab/internal/reject"
)

const (
	maxFallbackDepth      = 3
	maxRouteRetryAttempts = 3
)

type GatewayService struct {
	routes        domain.GatewayRouteStore
	executors     map[string]domain.Executor
	logger        domain.GatewayAttemptLogger
	quota         domain.DispatchQuotaManager
	strategy      domain.RoutingStrategyStore
	cursors       domain.RoutingCursorStore
	runtimeConfig domain.GatewayRuntimeConfigStore
	runtimeStates domain.RoutingRuntimeStateStore
	sticky        domain.StickyRoutingStore
	capabilities  *capability.Registry
}

func (s *GatewayService) SelectRoute(ctx context.Context, req domain.GatewayRequest) (domain.GatewayRoute, error) {
	settings := domain.GatewayRuntimeSettings{CooldownDuration: 45 * time.Second, StickyEnabled: true, StickyKeySource: "auto"}
	if req.RuntimeSettings != nil {
		settings = *req.RuntimeSettings
	} else if s.runtimeConfig != nil {
		loaded, err := s.runtimeConfig.GetGatewayRuntimeSettings(ctx)
		if err != nil {
			return domain.GatewayRoute{}, err
		}
		settings = loaded
	}
	state := &gatewayExecutionState{settings: settings, fallbackStage: "none", visitedSet: map[string]struct{}{}, plannedOps: map[int64]domain.ProtocolOperation{}}
	return s.selectRouteForAlias(ctx, req, req.Model, 0, state)
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

type gatewayExecutionState struct {
	settings        domain.GatewayRuntimeSettings
	strategy        domain.RoutingStrategy
	fallbackStage   string
	fallbackChain   []string
	visitedAliases  []string
	visitedSet      map[string]struct{}
	skips           []domain.GatewaySkip
	attemptCount    int
	stickyHit       bool
	stickyRouteID   int64
	stickyChannel   string
	stickyReason    string
	winningBucket   string
	winningPriority int
	decisionReason  string
	selectedChannel string
	attemptedRoutes []domain.GatewayAttemptTrace
	plannedOps      map[int64]domain.ProtocolOperation
}

func NewGatewayService(
	routes domain.GatewayRouteStore,
	executors map[string]domain.Executor,
	logger domain.GatewayAttemptLogger,
	quota domain.DispatchQuotaManager,
	strategy domain.RoutingStrategyStore,
	cursors domain.RoutingCursorStore,
	runtimeConfig domain.GatewayRuntimeConfigStore,
	runtimeStates domain.RoutingRuntimeStateStore,
	sticky domain.StickyRoutingStore,
	capabilities *capability.Registry,
) *GatewayService {
	return &GatewayService{
		routes:        routes,
		executors:     executors,
		logger:        logger,
		quota:         quota,
		strategy:      strategy,
		cursors:       cursors,
		runtimeConfig: runtimeConfig,
		runtimeStates: runtimeStates,
		sticky:        sticky,
		capabilities:  capabilities,
	}
}

func (s *GatewayService) Execute(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
	settings := domain.GatewayRuntimeSettings{CooldownDuration: 45 * time.Second, StickyEnabled: true, StickyKeySource: "auto"}
	if req.RuntimeSettings != nil {
		settings = *req.RuntimeSettings
	} else if s.runtimeConfig != nil {
		loaded, err := s.runtimeConfig.GetGatewayRuntimeSettings(ctx)
		if err != nil {
			return nil, err
		}
		settings = loaded
	}

	state := &gatewayExecutionState{
		settings:      settings,
		fallbackStage: "none",
		visitedSet:    map[string]struct{}{},
		plannedOps:    map[int64]domain.ProtocolOperation{},
	}

	result, err := s.executeAlias(ctx, requestID, req, req.Model, 0, state)
	if err != nil {
		execErr := domain.AsExecutionError(err)
		execErr.Metadata = state.metadata(req)
		return nil, execErr
	}

	if result.Metadata == nil {
		result.Metadata = state.metadata(req)
	}
	return result, nil
}

func (s *GatewayService) executeAlias(ctx context.Context, requestID string, req domain.GatewayRequest, alias string, depth int, state *gatewayExecutionState) (*domain.ExecutionResult, error) {
	normalizedAlias := strings.TrimSpace(alias)
	if normalizedAlias == "" {
		return nil, domain.ErrNoAvailableRoute(req.Model)
	}
	if depth > maxFallbackDepth {
		return nil, fmt.Errorf("fallback 重入次数超过限制")
	}
	if _, exists := state.visitedSet[normalizedAlias]; exists {
		return nil, fmt.Errorf("fallback 链路出现循环: %s", normalizedAlias)
	}
	state.visitedSet[normalizedAlias] = struct{}{}
	state.visitedAliases = append(state.visitedAliases, normalizedAlias)
	if depth > 0 {
		state.fallbackChain = append(state.fallbackChain, normalizedAlias)
	}

	routingReq := req
	routingReq.Model = normalizedAlias

	routeList, err := s.routes.ListEnabledRoutesByModel(ctx, normalizedAlias)
	if err != nil {
		return nil, err
	}
	if len(routeList) == 0 {
		return nil, domain.ErrNoAvailableRoute(normalizedAlias)
	}

	available := s.filterAvailableRoutes(ctx, requestID, req, routingReq, routeList, state)
	if len(available) == 0 {
		fallbackAlias := selectFallbackAlias(routeList)
		if fallbackAlias != "" {
			state.decisionReason = "cooldown_fallback"
			state.fallbackStage = "model_alias_reentry"
			return s.executeAlias(ctx, requestID, req, fallbackAlias, depth+1, state)
		}
		reason := "no_viable_route"
		if count := len(state.skips); count > 0 {
			reason = state.skips[count-1].Reason
		}
		rejection := reject.NewEngine().Decide(routingReq, reason)
		return nil, domain.NewExecutionError(fmt.Errorf("%s", rejection.Message), rejection.StatusCode, rejection.Retryable, false)
	}

	plans, strategy, err := s.arrangeRoutes(ctx, routingReq, available)
	if err != nil {
		return nil, err
	}
	state.strategy = strategy
	plans = s.applyStickyPreference(ctx, req, routingReq, plans, state)
	executionPlan := planner.BuildExecutionPlan(ctx, s.capabilities, routingReq, routesFromPlans(plans), selectFallbackAlias(routeList), reject.NewEngine())
	if executionPlan.Rejection != nil && len(executionPlan.Attempts) == 0 {
		return nil, domain.NewExecutionError(fmt.Errorf("%s", executionPlan.Rejection.Message), executionPlan.Rejection.StatusCode, executionPlan.Rejection.Retryable, false)
	}
	attemptPlans := make(map[int64]planner.AttemptPlan, len(executionPlan.Attempts))
	for _, attemptPlan := range executionPlan.Attempts {
		attemptPlans[attemptPlan.Route.ID] = attemptPlan
	}

	var lastErr error
	var activeGroup *routePlan
	advanceGroup := func(plan *routePlan, selectedIndex int) {
		if plan == nil || plan.cursorKey == "" || s.cursors == nil || plan.groupSize <= 1 {
			return
		}
		_ = s.cursors.AdvanceRoutingCursor(ctx, plan.cursorKey, plan.groupSize, selectedIndex)
	}

	for _, plan := range plans {
		if activeGroup != nil && activeGroup.groupID != "" && plan.groupID != activeGroup.groupID {
			advanceGroup(activeGroup, activeGroup.startIndex)
		}
		activeGroup = &plan
		route := plan.route
		executor := s.executors[domain.NormalizeProvider(route.Channel.Provider)]
		if executor == nil {
			lastErr = fmt.Errorf("provider %s 没有可用执行器", route.Channel.Provider)
			state.attemptCount++
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
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
				Attempt:          state.attemptCount,
				StatusCode:       0,
				Retryable:        false,
				StreamStarted:    false,
				Success:          false,
				ErrorMessage:     lastErr.Error(),
				DecisionReason:   "missing_executor",
				FallbackStage:    state.fallbackStage,
				StickyHit:        state.stickyHit,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
			}, state)
			continue
		}

		reservation, reserveErr := s.reserveQuota(ctx, requestID, route)
		if reserveErr != nil {
			return nil, reserveErr
		}
		if !reservation.LeaseAcquired {
			lastErr = domain.NewExecutionError(fmt.Errorf("渠道额度预约中，请稍后重试"), 429, true, false)
			state.attemptCount++
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
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
				Attempt:          state.attemptCount,
				StatusCode:       429,
				Retryable:        true,
				StreamStarted:    false,
				Success:          false,
				ErrorMessage:     lastErr.Error(),
				DecisionReason:   "quota_wait",
				FallbackStage:    state.fallbackStage,
				StickyHit:        state.stickyHit,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
			}, state)
			continue
		}
		releaseQuota := func() {
			if s.quota == nil || !reservation.LeaseAcquired || reservation.ReservationKey == "" {
				return
			}
			_ = s.quota.Release(ctx, domain.DispatchReleaseInput{ChannelName: route.Channel.Name, ReservationKey: reservation.ReservationKey})
		}

		plannedReq := adaptRequestForProvider(routingReq, route.Channel.Provider)
		if attemptPlan, ok := attemptPlans[route.ID]; ok && attemptPlan.TargetOperation != "" {
			plannedReq.Operation = attemptPlan.TargetOperation
		} else if plannedOp, ok := state.plannedOps[route.ID]; ok && plannedOp != "" {
			plannedReq.Operation = plannedOp
		}

		for routeAttempt := 0; routeAttempt < maxRouteRetryAttempts; routeAttempt++ {
			attemptStartedAt := time.Now()
			result, execErr := executor.Execute(ctx, domain.ExecutorRequest{
				Channel:       route.Channel,
				UpstreamModel: route.UpstreamModel,
				Request:       plannedReq,
			})

			if execErr == nil {
				releaseQuota()
				state.attemptCount++
				state.winningBucket = plan.bucketName
				state.winningPriority = route.Priority
				state.selectedChannel = route.Channel.Name
				if depth > 0 {
					state.decisionReason = "fallback_success"
					state.fallbackStage = "model_alias_reentry"
				} else if state.stickyHit {
					state.decisionReason = "sticky_hit"
					state.fallbackStage = "none"
				} else {
					state.decisionReason = "route_success"
					state.fallbackStage = "none"
				}
				if req.AffinityKey != "" && state.settings.StickyEnabled && s.sticky != nil && (!state.stickyHit || state.stickyRouteID != route.ID) {
					_ = s.sticky.UpsertStickyBinding(ctx, req.AffinityKey, normalizedAlias, req.Protocol, route.ID)
				}

				responseBody := ""
				statusCode := 0
				latencyMs := time.Since(attemptStartedAt).Milliseconds()
				if result.Stream != nil {
					statusCode = result.Stream.StatusCode
					if result.Stream.Headers == nil {
						result.Stream.Headers = map[string][]string{}
					}
					result.Stream.Headers["X-Opencrab-Channel"] = []string{route.Channel.Name}
					result.Stream.Headers["X-Opencrab-Provider"] = []string{domain.NormalizeProvider(route.Channel.Provider)}
					result.Stream.Headers["X-Opencrab-Operation"] = []string{string(plannedReq.Operation)}
				}
				if result.Response != nil {
					statusCode = result.Response.StatusCode
					responseBody = truncateGatewayBody(result.Response.Body)
					if result.Response.Headers == nil {
						result.Response.Headers = map[string][]string{}
					}
					result.Response.Headers["X-Opencrab-Channel"] = []string{route.Channel.Name}
					result.Response.Headers["X-Opencrab-Provider"] = []string{domain.NormalizeProvider(route.Channel.Provider)}
					result.Response.Headers["X-Opencrab-Operation"] = []string{string(plannedReq.Operation)}
				}
				s.logAttempt(ctx, domain.GatewayAttemptLog{
					RouteID:          route.ID,
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
					Attempt:          state.attemptCount,
					StatusCode:       statusCode,
					Retryable:        false,
					StreamStarted:    false,
					Success:          true,
					DecisionReason:   state.decisionReason,
					FallbackStage:    state.fallbackStage,
					StickyHit:        state.stickyHit,
					SelectedChannel:  route.Channel.Name,
					AffinityKey:      req.AffinityKey,
					FallbackChain:    append([]string(nil), state.fallbackChain...),
					VisitedAliases:   append([]string(nil), state.visitedAliases...),
					RequestBody:      marshalGatewayRequest(req),
					ResponseBody:     responseBody,
					LatencyMs:        latencyMs,
				}, state)
				result.Metadata = state.metadata(req)
				advanceGroup(&plan, plan.originalIndex)
				return result, nil
			}

			detail := domain.AsExecutionError(execErr)
			releaseQuota()
			state.attemptCount++
			latencyMs := time.Since(attemptStartedAt).Milliseconds()
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
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
				Attempt:          state.attemptCount,
				StatusCode:       detail.StatusCode,
				Retryable:        detail.Retryable,
				StreamStarted:    detail.StreamStarted,
				Success:          false,
				ErrorMessage:     detail.Error(),
				DecisionReason:   "attempt_failed",
				FallbackStage:    state.fallbackStage,
				StickyHit:        state.stickyHit,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
				LatencyMs:        latencyMs,
			}, state)
			lastErr = detail
			if !detail.Retryable || detail.StreamStarted {
				advanceGroup(&plan, plan.originalIndex)
				return nil, detail
			}
			if shouldAvoidSameRouteRetryForOpenAIResponses(detail, plannedReq) {
				break
			}
		}
	}

	advanceGroup(activeGroup, activeGroup.startIndex)
	fallbackAlias := executionPlan.FallbackAlias
	if fallbackAlias != "" {
		state.decisionReason = "fallback_reentry"
		state.fallbackStage = "model_alias_reentry"
		return s.executeAlias(ctx, requestID, req, fallbackAlias, depth+1, state)
	}

	if lastErr == nil {
		lastErr = domain.ErrNoAvailableRoute(normalizedAlias)
	}
	return nil, lastErr
}

func (s *GatewayService) selectRouteForAlias(ctx context.Context, req domain.GatewayRequest, alias string, depth int, state *gatewayExecutionState) (domain.GatewayRoute, error) {
	normalizedAlias := strings.TrimSpace(alias)
	if normalizedAlias == "" {
		return domain.GatewayRoute{}, domain.ErrNoAvailableRoute(req.Model)
	}
	if depth > maxFallbackDepth {
		return domain.GatewayRoute{}, fmt.Errorf("fallback 重入次数超过限制")
	}
	if _, exists := state.visitedSet[normalizedAlias]; exists {
		return domain.GatewayRoute{}, fmt.Errorf("fallback 链路出现循环: %s", normalizedAlias)
	}
	state.visitedSet[normalizedAlias] = struct{}{}
	state.visitedAliases = append(state.visitedAliases, normalizedAlias)
	routingReq := req
	routingReq.Model = normalizedAlias
	routeList, err := s.routes.ListEnabledRoutesByModel(ctx, normalizedAlias)
	if err != nil {
		return domain.GatewayRoute{}, err
	}
	if len(routeList) == 0 {
		return domain.GatewayRoute{}, domain.ErrNoAvailableRoute(normalizedAlias)
	}
	available := s.filterAvailableRoutes(ctx, "route-select", req, routingReq, routeList, state)
	if len(available) == 0 {
		fallbackAlias := selectFallbackAlias(routeList)
		if fallbackAlias != "" {
			return s.selectRouteForAlias(ctx, req, fallbackAlias, depth+1, state)
		}
		return domain.GatewayRoute{}, domain.ErrNoAvailableRoute(normalizedAlias)
	}
	plans, strategy, err := s.arrangeRoutes(ctx, routingReq, available)
	if err != nil {
		return domain.GatewayRoute{}, err
	}
	state.strategy = strategy
	plans = s.applyStickyPreference(ctx, req, routingReq, plans, state)
	if len(plans) == 0 {
		return domain.GatewayRoute{}, domain.ErrNoAvailableRoute(normalizedAlias)
	}
	return plans[0].route, nil
}

func adaptRequestForProvider(req domain.GatewayRequest, providerName string) domain.GatewayRequest {
	adapted := req
	if req.Stream && !protocolMatchesProviderForExecution(req.Protocol, providerName) {
		adapted.Stream = false
	}
	return adapted
}

func shouldAvoidSameRouteRetryForOpenAIResponses(detail *domain.ExecutionError, req domain.GatewayRequest) bool {
	if detail == nil {
		return false
	}
	if req.Protocol != domain.ProtocolOpenAI {
		return false
	}
	if req.Operation != domain.ProtocolOperationOpenAIResponses {
		return false
	}
	if detail.StreamStarted || !detail.Retryable {
		return false
	}
	return true
}

func protocolMatchesProviderForExecution(protocol domain.Protocol, providerName string) bool {
	providerName = domain.NormalizeProvider(providerName)
	switch protocol {
	case domain.ProtocolClaude:
		return providerName == "claude"
	case domain.ProtocolGemini:
		return providerName == "gemini"
	case domain.ProtocolCodex:
		return providerName == "openai" || providerName == "openrouter" || providerName == "glm" || providerName == "kimi" || providerName == "minimax"
	default:
		return providerName == "openai" || providerName == "openrouter" || providerName == "glm" || providerName == "kimi" || providerName == "minimax"
	}
}

func (s *GatewayService) reserveQuota(ctx context.Context, requestID string, route domain.GatewayRoute) (domain.DispatchReservationResult, error) {
	if s.quota == nil {
		return domain.DispatchReservationResult{ChannelName: route.Channel.Name, LeaseAcquired: true, Runtime: "disabled"}, nil
	}
	return s.quota.Reserve(ctx, domain.DispatchReservationInput{
		ChannelName:    route.Channel.Name,
		RPMLimit:       route.Channel.RPMLimit,
		MaxInflight:    route.Channel.MaxInflight,
		SafetyFactor:   route.Channel.SafetyFactor,
		LeaseMs:        30000,
		ReservationKey: requestID,
	})
}

func (s *GatewayService) filterAvailableRoutes(ctx context.Context, requestID string, req domain.GatewayRequest, routingReq domain.GatewayRequest, routes []domain.GatewayRoute, state *gatewayExecutionState) []domain.GatewayRoute {
	available := make([]domain.GatewayRoute, 0, len(routes))
	for _, route := range routes {
		if preferredProvider := strings.TrimSpace(req.PreferredProvider); preferredProvider != "" && domain.NormalizeProvider(route.Channel.Provider) != domain.NormalizeProvider(preferredProvider) {
			skip := domain.GatewaySkip{
				RouteID:        route.ID,
				ModelAlias:     route.ModelAlias,
				Channel:        route.Channel.Name,
				Reason:         "preferred_provider_mismatch",
				Provider:       route.Channel.Provider,
				InvocationMode: route.InvocationMode,
				Priority:       route.Priority,
			}
			state.skips = append(state.skips, skip)
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
				RequestID:        requestID,
				Model:            req.Model,
				UpstreamModel:    route.UpstreamModel,
				Channel:          route.Channel.Name,
				Provider:         route.Channel.Provider,
				RoutingStrategy:  string(state.strategy),
				InvocationBucket: invocationBucketName(route, routingReq.Protocol),
				PriorityTier:     route.Priority,
				CandidateCount:   0,
				SelectedIndex:    0,
				Attempt:          0,
				StatusCode:       0,
				Retryable:        false,
				StreamStarted:    false,
				Success:          false,
				DecisionReason:   "route_skipped",
				FallbackStage:    state.fallbackStage,
				SkipReason:       "preferred_provider_mismatch",
				StickyHit:        false,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
			}, state)
			continue
		}
		if req.APIKeyScope != nil && len(req.APIKeyScope.ChannelNames) > 0 && !scopeListContains(req.APIKeyScope.ChannelNames, route.Channel.Name) {
			skip := domain.GatewaySkip{
				RouteID:        route.ID,
				ModelAlias:     route.ModelAlias,
				Channel:        route.Channel.Name,
				Reason:         "api_key_channel_restricted",
				Provider:       route.Channel.Provider,
				InvocationMode: route.InvocationMode,
				Priority:       route.Priority,
			}
			state.skips = append(state.skips, skip)
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
				RequestID:        requestID,
				Model:            req.Model,
				UpstreamModel:    route.UpstreamModel,
				Channel:          route.Channel.Name,
				Provider:         route.Channel.Provider,
				RoutingStrategy:  string(state.strategy),
				InvocationBucket: invocationBucketName(route, routingReq.Protocol),
				PriorityTier:     route.Priority,
				CandidateCount:   0,
				SelectedIndex:    0,
				Attempt:          0,
				StatusCode:       0,
				Retryable:        false,
				StreamStarted:    false,
				Success:          false,
				DecisionReason:   "route_skipped",
				FallbackStage:    state.fallbackStage,
				SkipReason:       "api_key_channel_restricted",
				StickyHit:        false,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
			}, state)
			continue
		}
		routePlan := planner.PlanRoute(ctx, s.capabilities, routingReq, route)
		if !routePlan.Executable {
			skip := domain.GatewaySkip{
				RouteID:        route.ID,
				ModelAlias:     route.ModelAlias,
				Channel:        route.Channel.Name,
				Reason:         routePlan.Reason,
				Provider:       route.Channel.Provider,
				InvocationMode: route.InvocationMode,
				Priority:       route.Priority,
			}
			state.skips = append(state.skips, skip)
			s.logAttempt(ctx, domain.GatewayAttemptLog{
				RouteID:          route.ID,
				RequestID:        requestID,
				Model:            req.Model,
				UpstreamModel:    route.UpstreamModel,
				Channel:          route.Channel.Name,
				Provider:         route.Channel.Provider,
				RoutingStrategy:  string(state.strategy),
				InvocationBucket: invocationBucketName(route, routingReq.Protocol),
				PriorityTier:     route.Priority,
				CandidateCount:   0,
				SelectedIndex:    0,
				Attempt:          0,
				StatusCode:       0,
				Retryable:        false,
				StreamStarted:    false,
				Success:          false,
				DecisionReason:   "route_skipped",
				FallbackStage:    state.fallbackStage,
				SkipReason:       routePlan.Reason,
				StickyHit:        false,
				SelectedChannel:  route.Channel.Name,
				AffinityKey:      req.AffinityKey,
				FallbackChain:    append([]string(nil), state.fallbackChain...),
				VisitedAliases:   append([]string(nil), state.visitedAliases...),
				RequestBody:      marshalGatewayRequest(req),
			}, state)
			continue
		}
		if routePlan.TargetOperation != "" {
			state.plannedOps[route.ID] = routePlan.TargetOperation
		}
		available = append(available, route)
	}
	return available
}

func (s *GatewayService) applyStickyPreference(ctx context.Context, req domain.GatewayRequest, routingReq domain.GatewayRequest, plans []routePlan, state *gatewayExecutionState) []routePlan {
	state.stickyHit = false
	state.stickyRouteID = 0
	state.stickyChannel = ""
	state.stickyReason = ""

	if !state.settings.StickyEnabled || req.AffinityKey == "" || s.sticky == nil || len(plans) == 0 {
		if !state.settings.StickyEnabled {
			state.stickyReason = "sticky_disabled"
		}
		return plans
	}

	routeID, found, err := s.sticky.GetStickyBinding(ctx, req.AffinityKey, routingReq.Model, routingReq.Protocol)
	if err != nil || !found {
		if err != nil {
			state.stickyReason = "sticky_lookup_failed"
		} else {
			state.stickyReason = "sticky_binding_miss"
		}
		return plans
	}

	first := plans[0]
	for index, plan := range plans {
		if plan.route.ID != routeID {
			continue
		}
		if plan.bucketName != first.bucketName || plan.route.Priority != first.route.Priority {
			state.stickyReason = "sticky_binding_out_of_tier"
			return plans
		}
		if index == 0 {
			state.stickyHit = true
			state.stickyRouteID = plan.route.ID
			state.stickyChannel = plan.route.Channel.Name
			state.stickyReason = "sticky_binding_hit"
			return plans
		}
		reordered := append([]routePlan{plan}, append(plans[:index], plans[index+1:]...)...)
		state.stickyHit = true
		state.stickyRouteID = plan.route.ID
		state.stickyChannel = plan.route.Channel.Name
		state.stickyReason = "sticky_binding_hit"
		return reordered
	}

	state.stickyReason = "sticky_binding_missing_route"
	return plans
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
	routes = preferNativeProviderRoutes(req, routes)
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

func routesFromPlans(plans []routePlan) []domain.GatewayRoute {
	routes := make([]domain.GatewayRoute, 0, len(plans))
	for _, plan := range plans {
		routes = append(routes, plan.route)
	}
	return routes
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
	buckets := map[string][]domain.GatewayRoute{"matched": {}, "neutral": {}, "mismatched": {}}
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
		return []routePlan{{route: routes[0], cursorKey: groupID, groupSize: 1, originalIndex: 0, startIndex: 0, groupID: groupID, bucketName: bucketName}}, nil
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

func selectFallbackAlias(routes []domain.GatewayRoute) string {
	fallback := ""
	for _, route := range routes {
		candidate := strings.TrimSpace(route.FallbackModel)
		if candidate == "" {
			continue
		}
		if fallback == "" {
			fallback = candidate
			continue
		}
		if fallback != candidate {
			return ""
		}
	}
	return fallback
}

func (s *GatewayService) logAttempt(ctx context.Context, item domain.GatewayAttemptLog, state *gatewayExecutionState) {
	if state != nil {
		state.recordAttemptTrace(item)
	}
	if s.logger == nil {
		return
	}
	_ = s.logger.LogGatewayAttempt(ctx, item)
}

func (s *gatewayExecutionState) recordAttemptTrace(item domain.GatewayAttemptLog) {
	trace := domain.GatewayAttemptTrace{
		RouteID:        item.RouteID,
		Channel:        item.Channel,
		Provider:       item.Provider,
		StatusCode:     item.StatusCode,
		Retryable:      item.Retryable,
		Success:        item.Success,
		DecisionReason: item.DecisionReason,
		LatencyMs:      item.LatencyMs,
		ErrorSummary:   truncateGatewayAttemptError(item.ErrorMessage),
	}
	s.attemptedRoutes = append(s.attemptedRoutes, trace)
}

func truncateGatewayAttemptError(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 240 {
		return trimmed
	}
	return trimmed[:240]
}

func (s *gatewayExecutionState) metadata(req domain.GatewayRequest) *domain.GatewayExecutionMetadata {
	return &domain.GatewayExecutionMetadata{
		RoutingStrategy: string(s.strategy),
		DecisionReason:  s.decisionReason,
		FallbackStage:   s.fallbackStage,
		FallbackChain:   append([]string(nil), s.fallbackChain...),
		VisitedAliases:  append([]string(nil), s.visitedAliases...),
		AttemptCount:    s.attemptCount,
		StickyHit:       s.stickyHit,
		StickyRouteID:   s.stickyRouteID,
		StickyChannel:   s.stickyChannel,
		StickyReason:    s.stickyReason,
		AffinityKey:     req.AffinityKey,
		WinningBucket:   s.winningBucket,
		WinningPriority: s.winningPriority,
		SelectedChannel: s.selectedChannel,
		DegradedSuccess: s.selectedChannel != "" && s.attemptCount > 1,
		AttemptedRoutes: append([]domain.GatewayAttemptTrace(nil), s.attemptedRoutes...),
		Skips:           append([]domain.GatewaySkip(nil), s.skips...),
	}
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

func scopeListContains(items []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	if normalizedTarget == "" {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item) == normalizedTarget {
			return true
		}
	}
	return false
}

func preferNativeProviderRoutes(req domain.GatewayRequest, routes []domain.GatewayRoute) []domain.GatewayRoute {
	preferredProvider := nativePreferredProvider(req)
	if preferredProvider == "" || len(routes) <= 1 {
		return routes
	}
	preferred := make([]domain.GatewayRoute, 0, len(routes))
	fallback := make([]domain.GatewayRoute, 0, len(routes))
	for _, route := range routes {
		if domain.NormalizeProvider(route.Channel.Provider) == preferredProvider {
			preferred = append(preferred, route)
			continue
		}
		fallback = append(fallback, route)
	}
	if len(preferred) == 0 {
		return routes
	}
	return append(preferred, fallback...)
}

func nativePreferredProvider(req domain.GatewayRequest) string {
	profile := capability.AnalyzeGatewayRequest(req)
	switch req.Protocol {
	case domain.ProtocolClaude:
		if requestHasAnyCapability(profile.RequiredCapabilities,
			capability.CapabilityFunctionTools,
			capability.CapabilityClaudeBetaHeader,
			capability.CapabilityClaudeThinking,
			capability.CapabilityClaudeToolChoiceForced,
			capability.CapabilityClaudePromptCaching,
			capability.CapabilityClaudeComputerUse,
			capability.CapabilityClaudeMCPServers,
			capability.CapabilityClaudeContainer,
			capability.CapabilityClaudeContextManagement,
		) {
			return "claude"
		}
	case domain.ProtocolGemini:
		if requestHasAnyCapability(profile.RequiredCapabilities,
			capability.CapabilityFunctionTools,
			capability.CapabilityGeminiGenerationConfig,
			capability.CapabilityGeminiSafetySettings,
			capability.CapabilityGeminiToolConfig,
			capability.CapabilityGeminiThinking,
			capability.CapabilityGeminiStructuredOutputs,
			capability.CapabilityGeminiGoogleSearch,
			capability.CapabilityGeminiURLContext,
			capability.CapabilityGeminiCodeExecution,
			capability.CapabilityGeminiThoughtSignatures,
			capability.CapabilityGeminiCachedContent,
		) {
			return "gemini"
		}
	case domain.ProtocolOpenAI, domain.ProtocolCodex:
		if requestHasAnyCapability(profile.RequiredCapabilities,
			capability.CapabilityBuiltinWebSearch,
			capability.CapabilityBuiltinFileSearch,
			capability.CapabilityBuiltinRemoteMCP,
			capability.CapabilityBuiltinComputerUse,
			capability.CapabilityBuiltinShell,
			capability.CapabilityBuiltinApplyPatch,
			capability.CapabilityBuiltinCodeInterpreter,
			capability.CapabilityBuiltinImageGeneration,
			capability.CapabilityCustomTools,
			capability.CapabilityParallelToolCalls,
			capability.CapabilityStructuredOutputs,
			capability.CapabilityReasoning,
			capability.CapabilitySafetyIdentifier,
			capability.CapabilityOpenAIResponsesSession,
			capability.CapabilityOpenAIResponsesInclude,
			capability.CapabilityOpenAIResponsesStore,
		) {
			return "openai"
		}
	}
	return ""
}

func requestHasAnyCapability(required map[capability.Capability]struct{}, items ...capability.Capability) bool {
	for _, item := range items {
		if _, ok := required[item]; ok {
			return true
		}
	}
	return false
}
