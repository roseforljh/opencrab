package capability

import "opencrab/internal/domain"

type surfaceSupport struct {
	capabilities map[Capability]struct{}
}

type providerMatrix struct {
	operations map[domain.ProtocolOperation]surfaceSupport
}

func providerSupportMatrix(provider string) providerMatrix {
	switch provider {
	case "openai":
		return providerMatrix{operations: map[domain.ProtocolOperation]surfaceSupport{
			domain.ProtocolOperationOpenAIChatCompletions: {
				capabilities: capabilitySet(
					CapabilityFunctionTools,
					CapabilityStructuredOutputs,
					CapabilityParallelToolCalls,
					CapabilityReasoning,
					CapabilitySafetyIdentifier,
					CapabilityMultimodalImage,
					CapabilityMultimodalAudio,
					CapabilityMultimodalFile,
				),
			},
			domain.ProtocolOperationOpenAIResponses: {
				capabilities: capabilitySet(
					CapabilityFunctionTools,
					CapabilityCustomTools,
					CapabilityBuiltinWebSearch,
					CapabilityBuiltinFileSearch,
					CapabilityBuiltinRemoteMCP,
					CapabilityBuiltinComputerUse,
					CapabilityBuiltinShell,
					CapabilityBuiltinApplyPatch,
					CapabilityBuiltinCodeInterpreter,
					CapabilityBuiltinImageGeneration,
					CapabilityParallelToolCalls,
					CapabilityStructuredOutputs,
					CapabilityReasoning,
					CapabilitySafetyIdentifier,
					CapabilityOpenAIResponsesSession,
					CapabilityOpenAIResponsesInclude,
					CapabilityOpenAIResponsesStore,
					CapabilityMultimodalImage,
					CapabilityMultimodalAudio,
					CapabilityMultimodalFile,
				),
			},
		}}
	case "claude":
		return providerMatrix{operations: map[domain.ProtocolOperation]surfaceSupport{
			domain.ProtocolOperationClaudeMessages: {
				capabilities: capabilitySet(
					CapabilityFunctionTools,
					CapabilityClaudeBetaHeader,
					CapabilityClaudeThinking,
					CapabilityClaudeToolChoiceForced,
					CapabilityClaudePromptCaching,
					CapabilityClaudeComputerUse,
					CapabilityClaudeMCPServers,
					CapabilityClaudeContainer,
					CapabilityClaudeContextManagement,
					CapabilityMultimodalImage,
					CapabilityMultimodalFile,
				),
			},
			domain.ProtocolOperationClaudeCountTokens: {capabilities: capabilitySet()},
		}}
	case "gemini":
		return providerMatrix{operations: map[domain.ProtocolOperation]surfaceSupport{
			domain.ProtocolOperationGeminiGenerateContent: {
				capabilities: capabilitySet(
					CapabilityFunctionTools,
					CapabilityGeminiGenerationConfig,
					CapabilityGeminiSafetySettings,
					CapabilityGeminiToolConfig,
					CapabilityGeminiThinking,
					CapabilityGeminiStructuredOutputs,
					CapabilityGeminiGoogleSearch,
					CapabilityGeminiURLContext,
					CapabilityGeminiCodeExecution,
					CapabilityGeminiThoughtSignatures,
					CapabilityGeminiCachedContent,
					CapabilityMultimodalImage,
					CapabilityMultimodalAudio,
					CapabilityMultimodalFile,
				),
			},
			domain.ProtocolOperationGeminiStreamGenerate: {
				capabilities: capabilitySet(
					CapabilityFunctionTools,
					CapabilityGeminiGenerationConfig,
					CapabilityGeminiSafetySettings,
					CapabilityGeminiToolConfig,
					CapabilityGeminiThinking,
					CapabilityGeminiStructuredOutputs,
					CapabilityGeminiGoogleSearch,
					CapabilityGeminiURLContext,
					CapabilityGeminiCodeExecution,
					CapabilityGeminiThoughtSignatures,
					CapabilityGeminiCachedContent,
					CapabilityMultimodalImage,
					CapabilityMultimodalAudio,
					CapabilityMultimodalFile,
				),
			},
		}}
	default:
		return providerMatrix{operations: map[domain.ProtocolOperation]surfaceSupport{}}
	}
}

func capabilitySet(items ...Capability) map[Capability]struct{} {
	set := make(map[Capability]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}
