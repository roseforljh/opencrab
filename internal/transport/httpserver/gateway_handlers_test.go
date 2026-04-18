package httpserver

import (
	"encoding/json"
	"testing"

	"opencrab/internal/domain"
)

func TestRepairGatewayTranscriptMergesAssistantToolCalls(t *testing.T) {
	input := []domain.GatewayMessage{
		{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "a"}}},
		{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_2", Name: "b"}}},
	}
	repaired := repairGatewayTranscript(input)
	if len(repaired) != 1 || len(repaired[0].ToolCalls) != 2 {
		t.Fatalf("unexpected repaired transcript: %#v", repaired)
	}
}

func TestRepairGatewayTranscriptMergesToolResultsByCallID(t *testing.T) {
	input := []domain.GatewayMessage{
		{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "a"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
		{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "b"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
	}
	repaired := repairGatewayTranscript(input)
	if len(repaired) != 1 || len(repaired[0].Parts) != 2 {
		t.Fatalf("unexpected repaired transcript: %#v", repaired)
	}
}

func TestClearHistoricalToolUsesPreservesTrailingPendingToolExchange(t *testing.T) {
	input := []domain.GatewayMessage{
		{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "old_call", Name: "old"}}},
		{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "old"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"old_call"`)}},
		{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "done"}}},
		{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "lookup"}}},
		{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: ""}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
	}
	cleared := clearHistoricalToolUses(input)
	if len(cleared) != 3 {
		t.Fatalf("unexpected cleared transcript: %#v", cleared)
	}
	if len(cleared[1].ToolCalls) != 1 || cleared[1].ToolCalls[0].ID != "call_1" {
		t.Fatalf("latest assistant tool call should be preserved: %#v", cleared)
	}
	if cleared[2].Role != "tool" || string(cleared[2].Metadata["tool_call_id"]) != `"call_1"` {
		t.Fatalf("latest tool result should be preserved: %#v", cleared)
	}
}
