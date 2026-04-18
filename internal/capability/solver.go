package capability

import (
	"context"

	"opencrab/internal/domain"
)

func EvaluateGatewayRoute(ctx context.Context, registry *Registry, req domain.GatewayRequest, route domain.GatewayRoute) RouteCompatibility {
	profile := AnalyzeGatewayRequest(req)
	provider := domain.NormalizeProvider(route.Channel.Provider)

	targetOp := resolveTargetOperation(profile, provider, req.Stream)
	supported, enabled, err := resolveOperationSurface(ctx, registry, route, targetOp)
	if err != nil {
		return RouteCompatibility{Executable: false, Reason: "capability_registry_unavailable"}
	}
	if !enabled || supported == nil {
		return RouteCompatibility{Executable: false, Reason: unsupportedOperationReason(profile, provider, targetOp)}
	}

	if reason := validateRequestLevelConstraints(profile, provider); reason != "" {
		return RouteCompatibility{Executable: false, Reason: reason}
	}
	if reason := validateSurfaceCapabilities(profile.RequiredCapabilities, supported, provider); reason != "" {
		return RouteCompatibility{Executable: false, Reason: reason}
	}

	return RouteCompatibility{
		Executable:      true,
		TargetOperation: targetOp,
	}
}

func resolveOperationSurface(ctx context.Context, registry *Registry, route domain.GatewayRoute, operation domain.ProtocolOperation) (map[Capability]struct{}, bool, error) {
	if registry != nil {
		return registry.Surface(ctx, route, operation)
	}
	matrix := providerSupportMatrix(domain.NormalizeProvider(route.Channel.Provider))
	surface, ok := matrix.operations[operation]
	if !ok {
		return nil, false, nil
	}
	return cloneCapabilitySet(surface.capabilities), true, nil
}

func resolveTargetOperation(profile RequestProfile, provider string, stream bool) domain.ProtocolOperation {
	switch provider {
	case "claude":
		return domain.ProtocolOperationClaudeMessages
	case "gemini":
		if stream {
			return domain.ProtocolOperationGeminiStreamGenerate
		}
		return domain.ProtocolOperationGeminiGenerateContent
	default:
		if profile.PreferredTargetOp != "" {
			return profile.PreferredTargetOp
		}
		if profile.SourceOperation != "" && profile.SourceProtocol == domain.ProtocolOpenAI {
			return profile.SourceOperation
		}
		if requiresAny(profile.RequiredCapabilities, CapabilityBuiltinWebSearch, CapabilityBuiltinFileSearch, CapabilityBuiltinRemoteMCP, CapabilityBuiltinComputerUse, CapabilityBuiltinShell, CapabilityBuiltinApplyPatch, CapabilityBuiltinCodeInterpreter, CapabilityBuiltinImageGeneration, CapabilityCustomTools, CapabilityOpenAIResponsesSession, CapabilityOpenAIResponsesInclude, CapabilityOpenAIResponsesStore) {
			return domain.ProtocolOperationOpenAIResponses
		}
		return domain.ProtocolOperationOpenAIChatCompletions
	}
}

func validateRequestLevelConstraints(profile RequestProfile, provider string) string {
	if provider == "gemini" && has(profile.RequiredCapabilities, CapabilityGeminiURLContext) && has(profile.RequiredCapabilities, CapabilityFunctionTools) {
		return "gemini_url_context_with_function_calling_unsupported"
	}
	if provider == "claude" && has(profile.RequiredCapabilities, CapabilityClaudeThinking) && has(profile.RequiredCapabilities, CapabilityClaudeToolChoiceForced) {
		return "claude_thinking_with_forced_tool_choice_unsupported"
	}
	return ""
}

func validateSurfaceCapabilities(required map[Capability]struct{}, supported map[Capability]struct{}, provider string) string {
	for capability := range required {
		if _, ok := supported[capability]; ok {
			continue
		}
		switch capability {
		case CapabilityClaudeBetaHeader, CapabilityClaudeThinking, CapabilityClaudeToolChoiceForced, CapabilityClaudePromptCaching, CapabilityClaudeComputerUse:
			return "claude_native_features_require_claude_route"
		case CapabilityGeminiGenerationConfig, CapabilityGeminiSafetySettings, CapabilityGeminiToolConfig, CapabilityGeminiThinking, CapabilityGeminiStructuredOutputs, CapabilityGeminiGoogleSearch, CapabilityGeminiURLContext, CapabilityGeminiCodeExecution, CapabilityGeminiThoughtSignatures:
			return "gemini_native_features_require_gemini_route"
		case CapabilityOpenAIResponsesSession, CapabilityOpenAIResponsesInclude, CapabilityOpenAIResponsesStore, CapabilityBuiltinWebSearch, CapabilityBuiltinFileSearch, CapabilityBuiltinRemoteMCP, CapabilityBuiltinComputerUse, CapabilityBuiltinShell, CapabilityBuiltinApplyPatch, CapabilityBuiltinCodeInterpreter, CapabilityBuiltinImageGeneration, CapabilityCustomTools:
			return "responses_native_features_require_openai_route"
		case CapabilityStructuredOutputs, CapabilityParallelToolCalls, CapabilityReasoning, CapabilitySafetyIdentifier:
			if provider != "openai" {
				return "openai_native_features_require_openai_route"
			}
		default:
			return "route_capability_not_supported"
		}
	}
	return ""
}

func has(required map[Capability]struct{}, capability Capability) bool {
	_, ok := required[capability]
	return ok
}

func unsupportedOperationReason(profile RequestProfile, provider string, targetOp domain.ProtocolOperation) string {
	switch targetOp {
	case domain.ProtocolOperationOpenAIResponses:
		return "responses_native_features_require_openai_route"
	case domain.ProtocolOperationClaudeMessages:
		return "claude_native_features_require_claude_route"
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		return "gemini_native_features_require_gemini_route"
	default:
		_ = profile
		_ = provider
		return "target_operation_not_supported"
	}
}
