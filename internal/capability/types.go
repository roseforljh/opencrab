package capability

import "opencrab/internal/domain"

type Capability string

const (
	CapabilityFunctionTools           Capability = "function_tools"
	CapabilityCustomTools             Capability = "custom_tools"
	CapabilityBuiltinWebSearch        Capability = "builtin_web_search"
	CapabilityBuiltinFileSearch       Capability = "builtin_file_search"
	CapabilityBuiltinRemoteMCP        Capability = "builtin_remote_mcp"
	CapabilityBuiltinComputerUse      Capability = "builtin_computer_use"
	CapabilityBuiltinShell            Capability = "builtin_shell"
	CapabilityBuiltinApplyPatch       Capability = "builtin_apply_patch"
	CapabilityBuiltinCodeInterpreter  Capability = "builtin_code_interpreter"
	CapabilityBuiltinImageGeneration  Capability = "builtin_image_generation"
	CapabilityParallelToolCalls       Capability = "parallel_tool_calls"
	CapabilityStructuredOutputs       Capability = "structured_outputs"
	CapabilitySafetyIdentifier        Capability = "safety_identifier"
	CapabilityReasoning               Capability = "reasoning"
	CapabilityOpenAIResponsesSession  Capability = "openai_responses_session"
	CapabilityOpenAIResponsesInclude  Capability = "openai_responses_include"
	CapabilityOpenAIResponsesStore    Capability = "openai_responses_store"
	CapabilityClaudeBetaHeader        Capability = "claude_beta_header"
	CapabilityClaudeThinking          Capability = "claude_thinking"
	CapabilityClaudeToolChoiceForced  Capability = "claude_tool_choice_forced"
	CapabilityClaudePromptCaching     Capability = "claude_prompt_caching"
	CapabilityClaudeComputerUse       Capability = "claude_computer_use"
	CapabilityGeminiGenerationConfig  Capability = "gemini_generation_config"
	CapabilityGeminiSafetySettings    Capability = "gemini_safety_settings"
	CapabilityGeminiToolConfig        Capability = "gemini_tool_config"
	CapabilityGeminiThinking          Capability = "gemini_thinking"
	CapabilityGeminiStructuredOutputs Capability = "gemini_structured_outputs"
	CapabilityGeminiGoogleSearch      Capability = "gemini_google_search"
	CapabilityGeminiURLContext        Capability = "gemini_url_context"
	CapabilityGeminiCodeExecution     Capability = "gemini_code_execution"
	CapabilityGeminiThoughtSignatures Capability = "gemini_thought_signatures"
	CapabilityMultimodalImage         Capability = "multimodal_image"
	CapabilityMultimodalAudio         Capability = "multimodal_audio"
	CapabilityMultimodalFile          Capability = "multimodal_file"
)

type RequestProfile struct {
	SourceProtocol       domain.Protocol
	SourceOperation      domain.ProtocolOperation
	RequiredCapabilities map[Capability]struct{}
	PreferredTargetOp    domain.ProtocolOperation
}

type RouteCompatibility struct {
	Executable      bool
	Reason          string
	TargetOperation domain.ProtocolOperation
}
