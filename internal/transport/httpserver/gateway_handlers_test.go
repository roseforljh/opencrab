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
