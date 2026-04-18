package planner

import (
	"encoding/json"
	"strings"

	"opencrab/internal/domain"
)

type Capability string

const (
	CapabilityToolUse                 Capability = "tool_use"
	CapabilityClaudeHeaders           Capability = "claude_headers"
	CapabilityClaudeThinking          Capability = "claude_thinking"
	CapabilityClaudeToolChoice        Capability = "claude_tool_choice"
	CapabilityClaudeCacheControl      Capability = "claude_cache_control"
	CapabilityGeminiGenerationConfig  Capability = "gemini_generation_config"
	CapabilityGeminiSafetySettings    Capability = "gemini_safety_settings"
	CapabilityGeminiToolConfig        Capability = "gemini_tool_config"
	CapabilityOpenAIResponsesSession  Capability = "openai_responses_session"
	CapabilityOpenAIResponsesReasoner Capability = "openai_responses_reasoning"
	CapabilityOpenAIResponsesInclude  Capability = "openai_responses_include"
	CapabilityOpenAIResponsesStore    Capability = "openai_responses_store"
)

type RequestProfile struct {
	Protocol             domain.Protocol
	Operation            domain.ProtocolOperation
	RequiredCapabilities map[Capability]struct{}
}

type RouteCompatibility struct {
	Executable bool
	Reason     string
}

func AnalyzeGatewayRequest(req domain.GatewayRequest) RequestProfile {
	required := map[Capability]struct{}{}
	if requestUsesTools(req) {
		required[CapabilityToolUse] = struct{}{}
	}
	if requestUsesClaudeHeaders(req) {
		required[CapabilityClaudeHeaders] = struct{}{}
	}
	if requestUsesClaudeThinking(req) {
		required[CapabilityClaudeThinking] = struct{}{}
	}
	if hasRawMetadataKey(req.Metadata, "tool_choice") {
		required[CapabilityClaudeToolChoice] = struct{}{}
	}
	if requestUsesClaudeCacheControl(req) {
		required[CapabilityClaudeCacheControl] = struct{}{}
	}
	if hasRawMetadataKey(req.Metadata, "generationConfig") {
		required[CapabilityGeminiGenerationConfig] = struct{}{}
	}
	if hasRawMetadataKey(req.Metadata, "safetySettings") {
		required[CapabilityGeminiSafetySettings] = struct{}{}
	}
	if hasRawMetadataKey(req.Metadata, "toolConfig") {
		required[CapabilityGeminiToolConfig] = struct{}{}
	}
	if requestUsesResponsesSession(req) {
		required[CapabilityOpenAIResponsesSession] = struct{}{}
	}
	if requestUsesResponsesReasoning(req) {
		required[CapabilityOpenAIResponsesReasoner] = struct{}{}
	}
	if requestUsesResponsesInclude(req) {
		required[CapabilityOpenAIResponsesInclude] = struct{}{}
	}
	if requestUsesResponsesStore(req) {
		required[CapabilityOpenAIResponsesStore] = struct{}{}
	}
	return RequestProfile{
		Protocol:             req.Protocol,
		Operation:            req.Operation,
		RequiredCapabilities: required,
	}
}

func EvaluateGatewayRoute(req domain.GatewayRequest, route domain.GatewayRoute) RouteCompatibility {
	profile := AnalyzeGatewayRequest(req)
	provider := domain.NormalizeProvider(route.Channel.Provider)

	if requiresAny(profile, CapabilityClaudeHeaders, CapabilityClaudeThinking, CapabilityClaudeToolChoice, CapabilityClaudeCacheControl) && provider != "claude" {
		return RouteCompatibility{Executable: false, Reason: "claude_native_features_require_claude_route"}
	}
	if requiresAny(profile, CapabilityGeminiGenerationConfig, CapabilityGeminiSafetySettings, CapabilityGeminiToolConfig) && provider != "gemini" {
		return RouteCompatibility{Executable: false, Reason: "gemini_native_features_require_gemini_route"}
	}
	if requiresAny(profile, CapabilityOpenAIResponsesSession, CapabilityOpenAIResponsesReasoner, CapabilityOpenAIResponsesInclude, CapabilityOpenAIResponsesStore) && !providerSupportsOpenAINative(provider) {
		return RouteCompatibility{Executable: false, Reason: "responses_native_features_require_openai_route"}
	}

	return RouteCompatibility{Executable: true}
}

func providerSupportsOpenAINative(provider string) bool {
	switch provider {
	case "openai":
		return true
	default:
		return false
	}
}

func requiresAny(profile RequestProfile, capabilities ...Capability) bool {
	for _, capability := range capabilities {
		if _, ok := profile.RequiredCapabilities[capability]; ok {
			return true
		}
	}
	return false
}

func requestUsesTools(req domain.GatewayRequest) bool {
	if len(req.Tools) > 0 {
		return true
	}
	if req.Session != nil && len(req.Session.ToolResults) > 0 {
		return true
	}
	for _, message := range req.Messages {
		if strings.EqualFold(message.Role, "tool") || len(message.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

func requestUsesClaudeHeaders(req domain.GatewayRequest) bool {
	for key := range req.RequestHeaders {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "anthropic-version", "anthropic-beta", "anthropic-dangerous-direct-browser-access":
			return true
		}
	}
	return false
}

func requestUsesClaudeThinking(req domain.GatewayRequest) bool {
	return hasRawMetadataKey(req.Metadata, "thinking")
}

func requestUsesClaudeCacheControl(req domain.GatewayRequest) bool {
	if hasRawMetadataKey(req.Metadata, "cache_control") {
		return true
	}
	for _, message := range req.Messages {
		if hasRawMetadataKey(message.Metadata, "cache_control") {
			return true
		}
		for _, part := range message.Parts {
			if hasRawMetadataKey(part.Metadata, "cache_control") {
				return true
			}
		}
	}
	return false
}

func requestUsesResponsesSession(req domain.GatewayRequest) bool {
	return req.Session != nil && strings.TrimSpace(req.Session.PreviousResponseID) != ""
}

func requestUsesResponsesReasoning(req domain.GatewayRequest) bool {
	return req.Session != nil && strings.TrimSpace(req.Session.Metadata["reasoning"]) != ""
}

func requestUsesResponsesInclude(req domain.GatewayRequest) bool {
	return req.Session != nil && strings.TrimSpace(req.Session.Metadata["include"]) != ""
}

func requestUsesResponsesStore(req domain.GatewayRequest) bool {
	return req.Session != nil && strings.TrimSpace(req.Session.Metadata["store"]) != ""
}

func hasRawMetadataKey(metadata map[string]json.RawMessage, key string) bool {
	if len(metadata) == 0 {
		return false
	}
	_, ok := metadata[key]
	return ok
}
