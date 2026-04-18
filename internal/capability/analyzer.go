package capability

import (
	"encoding/json"
	"strings"

	"opencrab/internal/domain"
)

func AnalyzeGatewayRequest(req domain.GatewayRequest) RequestProfile {
	required := map[Capability]struct{}{}

	analyzeMessageParts(required, req.Messages)
	analyzeOpenAIMetadata(required, req)
	analyzeClaudeMetadata(required, req)
	analyzeGeminiMetadata(required, req)
	analyzeTools(required, req.Tools)

	if req.Session != nil {
		if strings.TrimSpace(req.Session.PreviousResponseID) != "" {
			addCapability(required, CapabilityOpenAIResponsesSession)
		}
		if strings.TrimSpace(req.Session.Metadata["include"]) != "" {
			addCapability(required, CapabilityOpenAIResponsesInclude)
		}
		if strings.TrimSpace(req.Session.Metadata["reasoning"]) != "" {
			addCapability(required, CapabilityReasoning)
		}
		if strings.TrimSpace(req.Session.Metadata["store"]) != "" {
			addCapability(required, CapabilityOpenAIResponsesStore)
		}
		if len(req.Session.ToolResults) > 0 {
			addCapability(required, CapabilityFunctionTools)
		}
	}

	if req.Operation == domain.ProtocolOperationOpenAIResponses {
		if requiresAny(required, CapabilityBuiltinWebSearch, CapabilityBuiltinFileSearch, CapabilityBuiltinRemoteMCP, CapabilityBuiltinComputerUse, CapabilityBuiltinShell, CapabilityBuiltinApplyPatch, CapabilityBuiltinCodeInterpreter, CapabilityBuiltinImageGeneration) {
			return RequestProfile{SourceProtocol: req.Protocol, SourceOperation: req.Operation, RequiredCapabilities: required, PreferredTargetOp: domain.ProtocolOperationOpenAIResponses}
		}
		return RequestProfile{SourceProtocol: req.Protocol, SourceOperation: req.Operation, RequiredCapabilities: required, PreferredTargetOp: domain.ProtocolOperationOpenAIResponses}
	}

	var preferred domain.ProtocolOperation
	if req.Protocol == domain.ProtocolOpenAI {
		preferred = req.Operation
	}
	if req.Operation == domain.ProtocolOperationOpenAIChatCompletions && requiresAny(required, CapabilityBuiltinWebSearch, CapabilityBuiltinFileSearch, CapabilityBuiltinRemoteMCP, CapabilityBuiltinComputerUse, CapabilityBuiltinShell, CapabilityBuiltinApplyPatch, CapabilityBuiltinCodeInterpreter, CapabilityBuiltinImageGeneration, CapabilityOpenAIResponsesSession, CapabilityOpenAIResponsesInclude, CapabilityOpenAIResponsesStore) {
		preferred = domain.ProtocolOperationOpenAIResponses
	}
	if preferred == "" && req.Protocol == domain.ProtocolOpenAI {
		preferred = defaultOperationForProtocol(req.Protocol, req.Stream)
	}

	return RequestProfile{
		SourceProtocol:       req.Protocol,
		SourceOperation:      req.Operation,
		RequiredCapabilities: required,
		PreferredTargetOp:    preferred,
	}
}

func analyzeMessageParts(required map[Capability]struct{}, messages []domain.GatewayMessage) {
	for _, message := range messages {
		if len(message.ToolCalls) > 0 {
			addCapability(required, CapabilityFunctionTools)
		}
		if strings.EqualFold(message.Role, "tool") {
			addCapability(required, CapabilityFunctionTools)
		}
		for _, part := range message.Parts {
			switch part.Type {
			case "image":
				addCapability(required, CapabilityMultimodalImage)
			case "audio":
				addCapability(required, CapabilityMultimodalAudio)
			case "document", "file", "video":
				addCapability(required, CapabilityMultimodalFile)
			}
		}
		if firstPartMetadataKey(message.Parts, "thoughtSignature", "thought_signature") {
			addCapability(required, CapabilityGeminiThoughtSignatures)
		}
		if hasRawMetadataKey(message.Metadata, "cache_control") {
			addCapability(required, CapabilityClaudePromptCaching)
		}
	}
}

func analyzeOpenAIMetadata(required map[Capability]struct{}, req domain.GatewayRequest) {
	if hasRawMetadataKey(req.Metadata, "response_format") || hasRawMetadataKey(req.Metadata, "text") {
		addCapability(required, CapabilityStructuredOutputs)
	}
	if hasRawMetadataKey(req.Metadata, "parallel_tool_calls") {
		addCapability(required, CapabilityParallelToolCalls)
	}
	if hasRawMetadataKey(req.Metadata, "reasoning") {
		addCapability(required, CapabilityReasoning)
	}
	if hasRawMetadataKey(req.Metadata, "safety_identifier") {
		addCapability(required, CapabilitySafetyIdentifier)
	}
	if hasRawMetadataKey(req.Metadata, "include") {
		addCapability(required, CapabilityOpenAIResponsesInclude)
	}
	if hasRawMetadataKey(req.Metadata, "store") {
		addCapability(required, CapabilityOpenAIResponsesStore)
	}
}

func analyzeClaudeMetadata(required map[Capability]struct{}, req domain.GatewayRequest) {
	for key := range req.RequestHeaders {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "anthropic-version", "anthropic-beta", "anthropic-dangerous-direct-browser-access":
			addCapability(required, CapabilityClaudeBetaHeader)
		}
	}
	if hasRawMetadataKey(req.Metadata, "thinking") {
		addCapability(required, CapabilityClaudeThinking)
	}
	if hasRawMetadataKey(req.Metadata, "tool_choice") {
		addCapability(required, CapabilityClaudeToolChoiceForced)
	}
	if hasRawMetadataKey(req.Metadata, "cache_control") {
		addCapability(required, CapabilityClaudePromptCaching)
	}
	if hasRawMetadataKey(req.Metadata, "container") || hasRawMetadataKey(req.Metadata, "context_management") || hasRawMetadataKey(req.Metadata, "mcp_servers") {
		addCapability(required, CapabilityClaudeBetaHeader)
	}
}

func analyzeGeminiMetadata(required map[Capability]struct{}, req domain.GatewayRequest) {
	if config, ok := req.Metadata["generationConfig"]; ok {
		addCapability(required, CapabilityGeminiGenerationConfig)
		if rawJSONContains(config, "responseSchema", "response_schema", "responseJsonSchema", "response_mime_type", "responseMimeType") {
			addCapability(required, CapabilityGeminiStructuredOutputs)
		}
		if rawJSONContains(config, "thinkingConfig", "thinking_config") {
			addCapability(required, CapabilityGeminiThinking)
		}
	}
	if hasRawMetadataKey(req.Metadata, "safetySettings") {
		addCapability(required, CapabilityGeminiSafetySettings)
	}
	if hasRawMetadataKey(req.Metadata, "toolConfig") {
		addCapability(required, CapabilityGeminiToolConfig)
	}
}

func analyzeTools(required map[Capability]struct{}, tools []json.RawMessage) {
	for _, raw := range tools {
		analyzeRawTool(required, raw)
	}
}

func analyzeRawTool(required map[Capability]struct{}, raw json.RawMessage) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err == nil {
		if toolType := decodeRawToolType(payload); toolType != "" {
			switch toolType {
			case "function":
				addCapability(required, CapabilityFunctionTools)
			case "custom":
				addCapability(required, CapabilityCustomTools)
			case "web_search", "web_search_preview", "web-search":
				addCapability(required, CapabilityBuiltinWebSearch)
			case "file_search", "file-search":
				addCapability(required, CapabilityBuiltinFileSearch)
			case "mcp", "remote_mcp":
				addCapability(required, CapabilityBuiltinRemoteMCP)
			case "computer_use", "computer_use_preview", "computer-preview", "computer":
				addCapability(required, CapabilityBuiltinComputerUse)
				addCapability(required, CapabilityClaudeComputerUse)
			case "shell":
				addCapability(required, CapabilityBuiltinShell)
			case "apply_patch":
				addCapability(required, CapabilityBuiltinApplyPatch)
			case "code_interpreter", "code_execution":
				addCapability(required, CapabilityBuiltinCodeInterpreter)
				addCapability(required, CapabilityGeminiCodeExecution)
			case "image_generation":
				addCapability(required, CapabilityBuiltinImageGeneration)
			}
		}

		switch {
		case payload["functionDeclarations"] != nil:
			addCapability(required, CapabilityFunctionTools)
		case payload["googleSearch"] != nil || payload["google_search"] != nil || payload["googleSearchRetrieval"] != nil:
			addCapability(required, CapabilityGeminiGoogleSearch)
		case payload["urlContext"] != nil || payload["url_context"] != nil:
			addCapability(required, CapabilityGeminiURLContext)
		case payload["codeExecution"] != nil || payload["code_execution"] != nil:
			addCapability(required, CapabilityGeminiCodeExecution)
		}
		return
	}

	text := strings.ToLower(string(raw))
	switch {
	case strings.Contains(text, "functiondeclarations"):
		addCapability(required, CapabilityFunctionTools)
	case strings.Contains(text, "googlesearch"), strings.Contains(text, "google_search"):
		addCapability(required, CapabilityGeminiGoogleSearch)
	case strings.Contains(text, "urlcontext"), strings.Contains(text, "url_context"):
		addCapability(required, CapabilityGeminiURLContext)
	case strings.Contains(text, "codeexecution"), strings.Contains(text, "code_execution"):
		addCapability(required, CapabilityGeminiCodeExecution)
	}
}

func decodeRawToolType(payload map[string]json.RawMessage) string {
	var value string
	if raw, ok := payload["type"]; ok {
		_ = json.Unmarshal(raw, &value)
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func defaultOperationForProtocol(protocol domain.Protocol, stream bool) domain.ProtocolOperation {
	switch protocol {
	case domain.ProtocolClaude:
		return domain.ProtocolOperationClaudeMessages
	case domain.ProtocolGemini:
		if stream {
			return domain.ProtocolOperationGeminiStreamGenerate
		}
		return domain.ProtocolOperationGeminiGenerateContent
	default:
		return domain.ProtocolOperationOpenAIChatCompletions
	}
}

func requiresAny(required map[Capability]struct{}, capabilities ...Capability) bool {
	for _, capability := range capabilities {
		if _, ok := required[capability]; ok {
			return true
		}
	}
	return false
}
