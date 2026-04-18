package planner

import (
	"context"
	"encoding/json"
	"testing"

	"opencrab/internal/domain"
)

func TestEvaluateGatewayRouteAllowsToolBridgeToOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Model:    "m",
			Tools:    []json.RawMessage{json.RawMessage(`{"type":"function","name":"opencode"}`)},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable {
		t.Fatalf("expected bridgeable route, got %#v", result)
	}
}

func TestEvaluateGatewayRouteMapsRealtimeToResponses(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol:  domain.ProtocolOpenAI,
			Operation: domain.ProtocolOperationOpenAIRealtime,
			Model:     "gpt-realtime",
			Messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
		},
		domain.GatewayRoute{
			ModelAlias: "gpt-realtime",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable || result.TargetOperation != domain.ProtocolOperationOpenAIResponses {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsClaudeThinkingOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationClaudeMessages,
			Model:     "m",
			Metadata: map[string]json.RawMessage{
				"thinking": json.RawMessage(`{"type":"enabled","budget_tokens":1024}`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteRejectsResponsesSessionOnNonOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol:  domain.ProtocolOpenAI,
			Operation: domain.ProtocolOperationOpenAIResponses,
			Model:     "m",
			Session:   &domain.GatewaySessionState{PreviousResponseID: "resp_123"},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "kimi-a", Provider: "kimi"},
		},
	)
	if result.Executable || result.Reason != "responses_native_features_require_openai_route" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsClaudeComputerUseOnClaude(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Model:    "m",
			Tools:    []json.RawMessage{json.RawMessage(`{"type":"computer_use"}`)},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "claude-a", Provider: "claude"},
		},
	)
	if !result.Executable {
		t.Fatalf("expected claude computer use route, got %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsOpenAIStructuredOutputsOnGemini(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"response_format": json.RawMessage(`{"type":"json_schema","json_schema":{"name":"answer","schema":{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}}}`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "gemini-a", Provider: "gemini"},
		},
	)
	if !result.Executable || result.TargetOperation != domain.ProtocolOperationGeminiGenerateContent {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsGeminiStructuredOutputsOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"generationConfig": json.RawMessage(`{"responseMimeType":"application/json","responseSchema":{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}}`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsOpenAIReasoningOnGemini(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"reasoning": json.RawMessage(`{"effort":"medium"}`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "gemini-a", Provider: "gemini"},
		},
	)
	if !result.Executable {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsGeminiCodeExecutionOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Model:    "m",
			Tools:    []json.RawMessage{json.RawMessage(`{"codeExecution":{}}`)},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteAllowsClaudeMCPServersOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"mcp_servers": json.RawMessage(`[{"name":"repo","type":"url","url":"https://example.com/mcp"}]`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if !result.Executable || result.TargetOperation != domain.ProtocolOperationOpenAIResponses {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteRejectsMalformedClaudeMCPServersOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"mcp_servers": json.RawMessage(`[{"name":"repo"}]`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if result.Executable || result.Reason != "route_capability_not_supported" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateGatewayRouteRejectsGeminiCachedContentOnOpenAI(t *testing.T) {
	result := EvaluateGatewayRoute(
		context.Background(),
		nil,
		domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Model:    "m",
			Metadata: map[string]json.RawMessage{
				"cachedContent": json.RawMessage(`"cachedContents/123"`),
			},
		},
		domain.GatewayRoute{
			ModelAlias: "m",
			Channel:    domain.UpstreamChannel{Name: "openai-a", Provider: "openai"},
		},
	)
	if result.Executable || result.Reason != "gemini_native_features_require_gemini_route" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
