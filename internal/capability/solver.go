package capability

import (
	"context"
	"encoding/json"
	"strings"

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

	if reason := validateRequestLevelConstraints(profile, provider, req); reason != "" {
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
		if requiresAny(profile.RequiredCapabilities, CapabilityBuiltinWebSearch, CapabilityBuiltinFileSearch, CapabilityBuiltinRemoteMCP, CapabilityBuiltinComputerUse, CapabilityBuiltinShell, CapabilityBuiltinApplyPatch, CapabilityBuiltinCodeInterpreter, CapabilityBuiltinImageGeneration, CapabilityCustomTools, CapabilityOpenAIResponsesSession, CapabilityOpenAIResponsesInclude, CapabilityOpenAIResponsesStore, CapabilityGeminiCodeExecution, CapabilityClaudeMCPServers, CapabilityClaudeContainer) {
			return domain.ProtocolOperationOpenAIResponses
		}
		return domain.ProtocolOperationOpenAIChatCompletions
	}
}

func validateRequestLevelConstraints(profile RequestProfile, provider string, req domain.GatewayRequest) string {
	if provider == "gemini" && has(profile.RequiredCapabilities, CapabilityGeminiURLContext) && has(profile.RequiredCapabilities, CapabilityFunctionTools) {
		return "gemini_url_context_with_function_calling_unsupported"
	}
	if provider == "claude" && has(profile.RequiredCapabilities, CapabilityClaudeThinking) && has(profile.RequiredCapabilities, CapabilityClaudeToolChoiceForced) {
		return "claude_thinking_with_forced_tool_choice_unsupported"
	}
	if provider == "claude" && has(profile.RequiredCapabilities, CapabilityBuiltinRemoteMCP) && !openAIMCPToolsCanBridgeToClaude(req.Tools) {
		return "route_capability_not_supported"
	}
	if provider == "openai" && has(profile.RequiredCapabilities, CapabilityClaudeMCPServers) && !claudeMCPServersCanBridgeToOpenAI(req.Metadata["mcp_servers"]) {
		return "route_capability_not_supported"
	}
	return ""
}

func validateSurfaceCapabilities(required map[Capability]struct{}, supported map[Capability]struct{}, provider string) string {
	for capability := range required {
		if capabilitySatisfiedByProvider(capability, supported, provider) {
			continue
		}
		switch capability {
		case CapabilityClaudeBetaHeader, CapabilityClaudeThinking, CapabilityClaudeToolChoiceForced, CapabilityClaudePromptCaching, CapabilityClaudeComputerUse, CapabilityClaudeMCPServers, CapabilityClaudeContainer, CapabilityClaudeContextManagement:
			return "claude_native_features_require_claude_route"
		case CapabilityGeminiGenerationConfig, CapabilityGeminiSafetySettings, CapabilityGeminiToolConfig, CapabilityGeminiThinking, CapabilityGeminiStructuredOutputs, CapabilityGeminiGoogleSearch, CapabilityGeminiURLContext, CapabilityGeminiCodeExecution, CapabilityGeminiThoughtSignatures, CapabilityGeminiCachedContent:
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

func capabilitySatisfiedByProvider(required Capability, supported map[Capability]struct{}, provider string) bool {
	if _, ok := supported[required]; ok {
		return true
	}
	switch required {
	case CapabilityClaudeBetaHeader:
		return provider != "claude"
	case CapabilityStructuredOutputs:
		return provider == "gemini" && has(supported, CapabilityGeminiStructuredOutputs)
	case CapabilityGeminiStructuredOutputs:
		return provider == "openai" && has(supported, CapabilityStructuredOutputs)
	case CapabilityReasoning:
		return (provider == "gemini" && has(supported, CapabilityGeminiThinking)) || (provider == "claude" && has(supported, CapabilityClaudeThinking))
	case CapabilityGeminiThinking:
		return (provider == "openai" && has(supported, CapabilityReasoning)) || (provider == "claude" && has(supported, CapabilityClaudeThinking))
	case CapabilityClaudeThinking:
		return (provider == "openai" && has(supported, CapabilityReasoning)) || (provider == "gemini" && has(supported, CapabilityGeminiThinking))
	case CapabilityClaudeToolChoiceForced:
		return provider == "openai" && has(supported, CapabilityFunctionTools)
	case CapabilityBuiltinCodeInterpreter:
		return (provider == "gemini" && has(supported, CapabilityGeminiCodeExecution)) || provider == "claude"
	case CapabilityGeminiCodeExecution:
		return provider == "openai" && has(supported, CapabilityBuiltinCodeInterpreter)
	case CapabilityBuiltinRemoteMCP:
		return provider == "claude" && has(supported, CapabilityClaudeMCPServers)
	case CapabilityClaudeMCPServers:
		return provider == "openai" && has(supported, CapabilityBuiltinRemoteMCP)
	case CapabilityClaudeContainer:
		return provider == "openai" && has(supported, CapabilityBuiltinCodeInterpreter)
	case CapabilityClaudeContextManagement:
		return provider == "openai" || provider == "gemini"
	default:
		return false
	}
}

func openAIMCPToolsCanBridgeToClaude(tools []json.RawMessage) bool {
	if len(tools) == 0 {
		return true
	}
	for _, raw := range tools {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(decodeRawToolType(payload))) != "mcp" {
			continue
		}
		if strings.TrimSpace(decodeCapabilityRawString(payload["server_url"])) == "" {
			return false
		}
	}
	return true
}

func claudeMCPServersCanBridgeToOpenAI(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var servers []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &servers); err != nil {
		return false
	}
	for _, server := range servers {
		if strings.TrimSpace(decodeCapabilityRawString(server["url"])) == "" {
			return false
		}
	}
	return true
}

func decodeCapabilityRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
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
