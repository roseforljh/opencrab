package planner

import (
	"context"
	"encoding/json"
	"testing"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
	"opencrab/internal/reject"
)

func TestBuildExecutionPlanDirectClaude(t *testing.T) {
	plan := BuildExecutionPlan(
		context.Background(),
		capability.NewRegistry(nil),
		domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationClaudeMessages,
			Model:     "m",
			Messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
		},
		[]domain.GatewayRoute{{ModelAlias: "m", Channel: domain.UpstreamChannel{Name: "claude-a", Provider: "claude"}}},
		"",
		reject.NewEngine(),
	)
	if len(plan.Attempts) != 1 || plan.Attempts[0].Mode != HopModeDirect {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestBuildExecutionPlanCodexToClaudeUsesMultiHop(t *testing.T) {
	plan := BuildExecutionPlan(
		context.Background(),
		capability.NewRegistry(nil),
		domain.GatewayRequest{
			Protocol:  domain.ProtocolCodex,
			Operation: domain.ProtocolOperationCodexResponses,
			Model:     "m",
			Messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
		},
		[]domain.GatewayRoute{{ModelAlias: "m", Channel: domain.UpstreamChannel{Name: "claude-a", Provider: "claude"}}},
		"m-fallback",
		reject.NewEngine(),
	)
	if len(plan.Attempts) != 1 || plan.Attempts[0].Mode != HopModeMultiHop {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if len(plan.Attempts[0].RequestTransforms) != 2 {
		t.Fatalf("expected 2 request transforms, got %#v", plan.Attempts[0].RequestTransforms)
	}
	if plan.FallbackAlias != "m-fallback" {
		t.Fatalf("expected fallback alias, got %#v", plan)
	}
}

func TestBuildExecutionPlanAllowsTranslatedClaudeThinkingOnOpenAI(t *testing.T) {
	plan := BuildExecutionPlan(
		context.Background(),
		capability.NewRegistry(nil),
		domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationClaudeMessages,
			Model:     "m",
			Messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
			Metadata: map[string]json.RawMessage{
				"thinking": json.RawMessage(`{"type":"enabled","budget_tokens":1024}`),
			},
		},
		[]domain.GatewayRoute{{ModelAlias: "m", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}}},
		"",
		reject.NewEngine(),
	)
	if len(plan.Attempts) != 1 || plan.Rejection != nil || plan.Attempts[0].TargetProvider != "openai" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestBuildExecutionPlanRealtimeUsesResponsesSurface(t *testing.T) {
	plan := BuildExecutionPlan(
		context.Background(),
		capability.NewRegistry(nil),
		domain.GatewayRequest{
			Protocol:  domain.ProtocolOpenAI,
			Operation: domain.ProtocolOperationOpenAIRealtime,
			Model:     "gpt-realtime",
			Messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
		},
		[]domain.GatewayRoute{{ModelAlias: "gpt-realtime", Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}}},
		"",
		reject.NewEngine(),
	)
	if len(plan.Attempts) != 1 || plan.Attempts[0].TargetOperation != domain.ProtocolOperationOpenAIResponses || plan.Attempts[0].Mode != HopModeSingleHop {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}
