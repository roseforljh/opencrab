package usecase

import (
	"encoding/json"
	"strings"

	"opencrab/internal/domain"
)

func requestRequiresProtocolMatchedRoute(req domain.GatewayRequest) (bool, string) {
	if requestUsesTools(req) {
		return true, "tooling_requires_native_protocol"
	}
	if req.Protocol == domain.ProtocolClaude && requestUsesClaudeNativeFeatures(req) {
		return true, "claude_native_features_require_claude_route"
	}
	if req.Protocol == domain.ProtocolGemini && requestUsesGeminiNativeFeatures(req) {
		return true, "gemini_native_features_require_gemini_route"
	}
	return false, ""
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

func requestUsesClaudeNativeFeatures(req domain.GatewayRequest) bool {
	for key := range req.RequestHeaders {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "anthropic-version", "anthropic-beta", "anthropic-dangerous-direct-browser-access":
			return true
		}
	}
	return rawMapHasKeys(req.Metadata, "thinking", "tool_choice", "cache_control")
}

func requestUsesGeminiNativeFeatures(req domain.GatewayRequest) bool {
	return rawMapHasKeys(req.Metadata, "generationConfig", "safetySettings", "toolConfig")
}

func rawMapHasKeys(metadata map[string]json.RawMessage, keys ...string) bool {
	if len(metadata) == 0 {
		return false
	}
	for _, key := range keys {
		if _, ok := metadata[key]; ok {
			return true
		}
	}
	return false
}
