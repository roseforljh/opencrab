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

func TestEvaluateGatewayRouteRejectsClaudeNativeThinkingOnOpenAI(t *testing.T) {
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
	if result.Executable || result.Reason != "claude_native_features_require_claude_route" {
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
