package httpserver

import (
	"encoding/json"
	"reflect"
	"strings"
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

func TestMergePreviousResponseNormalizesNativeResponsesContinuation(t *testing.T) {
	tests := []struct {
		name      string
		operation domain.ProtocolOperation
		messages  []domain.GatewayMessage
		expected  []domain.GatewayMessage
	}{
		{
			name:      "delta only remains unchanged",
			operation: domain.ProtocolOperationOpenAIResponses,
			messages:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}}},
			expected:  []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}}},
		},
		{
			name:      "duplicated prefix trimmed for responses",
			operation: domain.ProtocolOperationOpenAIResponses,
			messages: []domain.GatewayMessage{
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
				{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}},
			},
			expected: []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}}},
		},
		{
			name:      "duplicated prefix trimmed for realtime",
			operation: domain.ProtocolOperationOpenAIRealtime,
			messages: []domain.GatewayMessage{
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
				{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}},
			},
			expected: []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "next"}}}},
		},
		{
			name:      "full transcript collapses to latest tail while preserving system",
			operation: domain.ProtocolOperationOpenAIResponses,
			messages: []domain.GatewayMessage{
				{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: "rules"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
				{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
				{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "tool-result"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
				{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "done"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "continue"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "continue 2"}}},
			},
			expected: []domain.GatewayMessage{
				{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: "rules"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "continue"}}},
				{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "continue 2"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryResponseSessionStore(8)
			store.Put(ResponseSession{
				ResponseID: "resp_1",
				Model:      "gpt-5.4",
				Messages: []domain.GatewayMessage{
					{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
					{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
				},
			})

			req := domain.GatewayRequest{
				Protocol:  domain.ProtocolOpenAI,
				Operation: tt.operation,
				Messages:  tt.messages,
				Session:   &domain.GatewaySessionState{PreviousResponseID: "resp_1"},
			}

			merged := mergePreviousResponse(store, req)
			if !reflect.DeepEqual(merged.Messages, tt.expected) {
				t.Fatalf("unexpected normalized messages: %#v", merged.Messages)
			}
			if merged.Session == nil || merged.Session.PreviousResponseID != "resp_1" {
				t.Fatalf("previous_response_id should be preserved: %#v", merged.Session)
			}
		})
	}
}

func TestPreprocessGatewayRequestDoesNotCompactOpenAIResponsesByDefault(t *testing.T) {
	req := domain.GatewayRequest{
		Protocol:  domain.ProtocolOpenAI,
		Operation: domain.ProtocolOperationOpenAIResponses,
		Messages: []domain.GatewayMessage{
			{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: strings.Repeat("rules ", 5000)}}},
			{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "old question"}}},
			{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "Read"}}},
			{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "tool output"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
			{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "new question"}}},
		},
	}
	got, err := preprocessGatewayRequest(nil, req)
	if err != nil {
		t.Fatalf("preprocess gateway request: %v", err)
	}
	if !reflect.DeepEqual(got.Messages, req.Messages) {
		t.Fatalf("preprocess should no longer compact responses hot path: %#v", got.Messages)
	}
}

func TestCollapseNativeContinuationMessagesPreservesTrailingPendingToolExchange(t *testing.T) {
	messages := []domain.GatewayMessage{
		{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: "rules"}}},
		{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "old"}}},
		{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "done"}}},
		{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "Read"}}},
		{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "tool output"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
	}

	collapsed := collapseNativeContinuationMessages(messages)
	if len(collapsed) != 3 {
		t.Fatalf("expected system + pending tool exchange, got %#v", collapsed)
	}
	if collapsed[0].Role != "system" || collapsed[1].Role != "assistant" || len(collapsed[1].ToolCalls) != 1 || collapsed[2].Role != "tool" {
		t.Fatalf("unexpected collapsed pending tool exchange: %#v", collapsed)
	}
}
