package planner

import (
	"context"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
	"opencrab/internal/reject"
)

type HopMode string

const (
	HopModeDirect    HopMode = "direct"
	HopModeSingleHop HopMode = "single_hop"
	HopModeMultiHop  HopMode = "multi_hop"
)

type TransformStage string

const (
	TransformStageRequest  TransformStage = "request"
	TransformStageResponse TransformStage = "response"
	TransformStageStream   TransformStage = "stream"
)

type TransformStep struct {
	Stage         TransformStage
	FromProtocol  domain.Protocol
	FromOperation domain.ProtocolOperation
	ToProtocol    domain.Protocol
	ToOperation   domain.ProtocolOperation
	Description   string
}

type AttemptPlan struct {
	Route              domain.GatewayRoute
	Executable         bool
	Reason             string
	TargetProvider     string
	TargetOperation    domain.ProtocolOperation
	Mode               HopMode
	RequestTransforms  []TransformStep
	ResponseTransforms []TransformStep
	StreamTransforms   []TransformStep
}

type ExecutionPlan struct {
	SourceProtocol  domain.Protocol
	SourceOperation domain.ProtocolOperation
	Attempts        []AttemptPlan
	FallbackAlias   string
	Rejection       *reject.Decision
}

func PlanRoute(ctx context.Context, registry *capability.Registry, req domain.GatewayRequest, route domain.GatewayRoute) AttemptPlan {
	compatibility := capability.EvaluateGatewayRoute(ctx, registry, req, route)
	attempt := AttemptPlan{
		Route:           route,
		Executable:      compatibility.Executable,
		Reason:          compatibility.Reason,
		TargetProvider:  domain.NormalizeProvider(route.Channel.Provider),
		TargetOperation: compatibility.TargetOperation,
	}
	if !compatibility.Executable {
		return attempt
	}
	requestSteps, responseSteps, streamSteps, mode := buildTransformPlan(req, attempt.TargetProvider, attempt.TargetOperation)
	attempt.Mode = mode
	attempt.RequestTransforms = requestSteps
	attempt.ResponseTransforms = responseSteps
	attempt.StreamTransforms = streamSteps
	return attempt
}

func BuildExecutionPlan(ctx context.Context, registry *capability.Registry, req domain.GatewayRequest, routes []domain.GatewayRoute, fallbackAlias string, engine *reject.Engine) ExecutionPlan {
	plan := ExecutionPlan{
		SourceProtocol:  req.Protocol,
		SourceOperation: normalizeSourceOperation(req),
		FallbackAlias:   fallbackAlias,
		Attempts:        make([]AttemptPlan, 0, len(routes)),
	}
	lastReason := ""
	for _, route := range routes {
		attempt := PlanRoute(ctx, registry, req, route)
		if !attempt.Executable {
			if attempt.Reason != "" {
				lastReason = attempt.Reason
			}
			continue
		}
		plan.Attempts = append(plan.Attempts, attempt)
	}
	if len(plan.Attempts) == 0 && engine != nil {
		if lastReason == "" {
			lastReason = "no_viable_route"
		}
		plan.Rejection = engine.Decide(req, lastReason)
	}
	return plan
}

func normalizeSourceOperation(req domain.GatewayRequest) domain.ProtocolOperation {
	if req.Operation != "" {
		return req.Operation
	}
	switch req.Protocol {
	case domain.ProtocolClaude:
		return domain.ProtocolOperationClaudeMessages
	case domain.ProtocolGemini:
		if req.Stream {
			return domain.ProtocolOperationGeminiStreamGenerate
		}
		return domain.ProtocolOperationGeminiGenerateContent
	case domain.ProtocolCodex:
		return domain.ProtocolOperationCodexResponses
	default:
		return domain.ProtocolOperationOpenAIChatCompletions
	}
}

func buildTransformPlan(req domain.GatewayRequest, targetProvider string, targetOperation domain.ProtocolOperation) ([]TransformStep, []TransformStep, []TransformStep, HopMode) {
	sourceProtocol := req.Protocol
	sourceOperation := normalizeSourceOperation(req)
	requestSteps := make([]TransformStep, 0, 2)

	if sourceProtocol == domain.ProtocolCodex {
		requestSteps = append(requestSteps, TransformStep{
			Stage:         TransformStageRequest,
			FromProtocol:  domain.ProtocolCodex,
			FromOperation: domain.ProtocolOperationCodexResponses,
			ToProtocol:    domain.ProtocolOpenAI,
			ToOperation:   domain.ProtocolOperationOpenAIResponses,
			Description:   "codex surface normalized into responses surface",
		})
		sourceProtocol = domain.ProtocolOpenAI
		sourceOperation = domain.ProtocolOperationOpenAIResponses
	}

	targetProtocol := targetProtocolForProvider(targetProvider)
	if sourceProtocol != targetProtocol || sourceOperation != targetOperation {
		requestSteps = append(requestSteps, TransformStep{
			Stage:         TransformStageRequest,
			FromProtocol:  sourceProtocol,
			FromOperation: sourceOperation,
			ToProtocol:    targetProtocol,
			ToOperation:   targetOperation,
			Description:   "canonical IR transformed into target execution surface",
		})
	}

	mode := HopModeDirect
	switch len(requestSteps) {
	case 0:
		mode = HopModeDirect
	case 1:
		mode = HopModeSingleHop
	default:
		mode = HopModeMultiHop
	}

	responseStep := TransformStep{
		Stage:         TransformStageResponse,
		FromProtocol:  targetProtocol,
		FromOperation: targetOperation,
		ToProtocol:    req.Protocol,
		ToOperation:   normalizeSourceOperation(req),
		Description:   "upstream response rendered back to client surface",
	}
	streamStep := TransformStep{
		Stage:         TransformStageStream,
		FromProtocol:  targetProtocol,
		FromOperation: targetOperation,
		ToProtocol:    req.Protocol,
		ToOperation:   normalizeSourceOperation(req),
		Description:   "upstream stream rendered back to client stream surface",
	}
	return requestSteps, []TransformStep{responseStep}, []TransformStep{streamStep}, mode
}

func targetProtocolForProvider(provider string) domain.Protocol {
	switch provider {
	case "claude":
		return domain.ProtocolClaude
	case "gemini":
		return domain.ProtocolGemini
	default:
		return domain.ProtocolOpenAI
	}
}
