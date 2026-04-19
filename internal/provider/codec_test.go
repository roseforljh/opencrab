package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestOpenAICodecRoundTripAndMetadata(t *testing.T) {
	req := domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-4o-mini",
		Stream:   true,
		Messages: []domain.UnifiedMessage{{
			Role:     "user",
			Parts:    []domain.UnifiedPart{{Type: "text", Text: "ping"}},
			Metadata: map[string]json.RawMessage{"name": json.RawMessage(`"alice"`)},
		}},
		Metadata: map[string]json.RawMessage{"temperature": json.RawMessage(`0.7`)},
	}

	data, err := EncodeOpenAIChatRequest(req)
	if err != nil {
		t.Fatalf("encode openai: %v", err)
	}

	decoded, err := DecodeOpenAIChatRequest(data)
	if err != nil {
		t.Fatalf("decode openai: %v", err)
	}
	if decoded.Model != req.Model || !decoded.Stream {
		t.Fatalf("unexpected decoded request: %+v", decoded)
	}
	if string(decoded.Metadata["temperature"]) != "0.7" {
		t.Fatalf("expected metadata temperature, got %+v", decoded.Metadata)
	}
	if string(decoded.Messages[0].Metadata["name"]) != `"alice"` {
		t.Fatalf("expected message metadata, got %+v", decoded.Messages[0].Metadata)
	}

	resp, err := DecodeOpenAIChatResponse([]byte(`{"id":"chatcmpl-1","model":"gpt-4o-mini","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong","refusal":null}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2},"system_fingerprint":"fp"}`))
	if err != nil {
		t.Fatalf("decode openai response: %v", err)
	}
	if resp.Message.Parts[0].Text != "pong" || resp.FinishReason != "stop" {
		t.Fatalf("unexpected openai response: %+v", resp)
	}
	if string(resp.Metadata["system_fingerprint"]) != `"fp"` {
		t.Fatalf("expected response metadata, got %+v", resp.Metadata)
	}
	if string(resp.Message.Metadata["refusal"]) != `null` {
		t.Fatalf("expected message metadata refusal, got %+v", resp.Message.Metadata)
	}

	respWithNestedUsage, err := DecodeOpenAIChatResponse([]byte(`{"id":"chatcmpl-2","model":"gpt-4o-mini","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}},"system_fingerprint":"fp"}`))
	if err != nil {
		t.Fatalf("decode openai response with nested usage: %v", err)
	}
	if respWithNestedUsage.Usage["prompt_tokens"] != 11 || respWithNestedUsage.Usage["completion_tokens"] != 7 || respWithNestedUsage.Usage["total_tokens"] != 18 {
		t.Fatalf("expected usage tokens to survive nested details, got %+v", respWithNestedUsage.Usage)
	}
}

func TestClaudeCodecRoundTripAndMetadata(t *testing.T) {
	req := domain.UnifiedChatRequest{
		Protocol: domain.ProtocolClaude,
		Model:    "claude-3-5-haiku-latest",
		Stream:   true,
		Messages: []domain.UnifiedMessage{{
			Role:     "user",
			Parts:    []domain.UnifiedPart{{Type: "text", Text: "ping"}},
			Metadata: map[string]json.RawMessage{"cache_control": json.RawMessage(`{"type":"ephemeral"}`)},
		}},
		Metadata: map[string]json.RawMessage{"max_tokens": json.RawMessage(`8`)},
	}

	data, err := EncodeClaudeChatRequest(req)
	if err != nil {
		t.Fatalf("encode claude: %v", err)
	}

	decoded, err := DecodeClaudeChatRequest(data)
	if err != nil {
		t.Fatalf("decode claude: %v", err)
	}
	if decoded.Model != req.Model || !decoded.Stream {
		t.Fatalf("unexpected decoded claude request: %+v", decoded)
	}
	if string(decoded.Metadata["max_tokens"]) != "8" {
		t.Fatalf("expected top metadata, got %+v", decoded.Metadata)
	}
	if string(decoded.Messages[0].Metadata["cache_control"]) != `{"type":"ephemeral"}` {
		t.Fatalf("expected message metadata, got %+v", decoded.Messages[0].Metadata)
	}

	resp, err := DecodeClaudeChatResponse([]byte(`{"id":"msg_1","model":"claude-3-5-haiku-latest","role":"assistant","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1},"stop_sequence":null}`))
	if err != nil {
		t.Fatalf("decode claude response: %v", err)
	}
	if resp.Message.Parts[0].Text != "pong" || resp.FinishReason != "end_turn" {
		t.Fatalf("unexpected claude response: %+v", resp)
	}
	if string(resp.Metadata["stop_sequence"]) != `null` {
		t.Fatalf("expected response metadata, got %+v", resp.Metadata)
	}
}

func TestGeminiCodecRoundTripConflictAndMetadata(t *testing.T) {
	req := domain.UnifiedChatRequest{
		Protocol: domain.ProtocolGemini,
		Model:    "gemini-2.0-flash",
		Stream:   true,
		Messages: []domain.UnifiedMessage{{
			Role:     "assistant",
			Parts:    []domain.UnifiedPart{{Type: "text", Text: "pong"}},
			Metadata: map[string]json.RawMessage{"thought": json.RawMessage(`false`)},
		}},
		Metadata: map[string]json.RawMessage{"generationConfig": json.RawMessage(`{"maxOutputTokens":8}`)},
	}

	data, err := EncodeGeminiChatRequest(req)
	if err != nil {
		t.Fatalf("encode gemini: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("unmarshal encoded gemini payload: %v", err)
	}

	decoded, err := DecodeGeminiChatRequest(data, "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("decode gemini: %v", err)
	}
	if decoded.Model != "gemini-2.0-flash" || decoded.Messages[0].Role != "assistant" {
		t.Fatalf("unexpected decoded gemini request: %+v", decoded)
	}
	if _, exists := encoded["model"]; exists {
		t.Fatalf("unexpected gemini body model field: %s", string(data))
	}
	if _, exists := encoded["stream"]; exists {
		t.Fatalf("unexpected gemini body stream field: %s", string(data))
	}
	if string(decoded.Metadata["generationConfig"]) != `{"maxOutputTokens":8}` {
		t.Fatalf("expected generationConfig metadata, got %+v", decoded.Metadata)
	}
	if string(decoded.Messages[0].Metadata["thought"]) != `false` {
		t.Fatalf("expected message metadata, got %+v", decoded.Messages[0].Metadata)
	}

	_, err = DecodeGeminiChatRequest([]byte(`{"model":"gemini-2.0-pro","contents":[{"parts":[{"text":"ping"}]}]}`), "gemini-2.0-flash")
	if err == nil || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("expected model conflict error, got %v", err)
	}

	resp, err := DecodeGeminiChatResponse([]byte(`{"modelVersion":"gemini-2.0-flash","candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[{"text":"pong"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"totalTokenCount":2},"promptFeedback":{"blockReason":"NONE"}}`))
	if err != nil {
		t.Fatalf("decode gemini response: %v", err)
	}
	if resp.Message.Role != "assistant" || resp.Message.Parts[0].Text != "pong" {
		t.Fatalf("unexpected gemini response: %+v", resp)
	}
	if string(resp.Metadata["promptFeedback"]) == "" {
		t.Fatalf("expected response metadata, got %+v", resp.Metadata)
	}
}

func TestOpenAICodecToolCalls(t *testing.T) {
	decoded, err := DecodeOpenAIChatResponse([]byte(`{"id":"chatcmpl-tool","choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"opencode","arguments":"{\"prompt\":\"ping\"}"}}]}}]}`))
	if err != nil {
		t.Fatalf("decode openai tool call response: %v", err)
	}
	if len(decoded.Message.ToolCalls) != 1 || decoded.Message.ToolCalls[0].Name != "opencode" {
		t.Fatalf("unexpected tool calls: %+v", decoded.Message.ToolCalls)
	}
	data, err := EncodeOpenAIChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolOpenAI, Model: "gpt-4o-mini", Messages: []domain.UnifiedMessage{{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "opencode", Arguments: json.RawMessage(`{"prompt":"ping"}`)}}}}})
	if err != nil {
		t.Fatalf("encode openai tool request: %v", err)
	}
	if !strings.Contains(string(data), `"tool_calls"`) {
		t.Fatalf("expected tool_calls in payload: %s", string(data))
	}
}

func TestResponsesCodecRoundTrip(t *testing.T) {
	decoded, err := DecodeOpenAIResponsesRequest([]byte(`{"model":"gpt-5.4","stream":true,"input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}],"tools":[{"type":"function","name":"opencode"}]}`))
	if err != nil {
		t.Fatalf("decode responses request: %v", err)
	}
	if decoded.Model != "gpt-5.4" || !decoded.Stream || decoded.Messages[0].Parts[0].Text != "ping" {
		t.Fatalf("unexpected decoded request: %+v", decoded)
	}
	respBody, err := EncodeOpenAIResponsesResponse(domain.UnifiedChatResponse{Protocol: domain.ProtocolOpenAI, ID: "resp_1", Model: "gpt-5.4", FinishReason: "stop", Message: domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}}, Usage: map[string]int64{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}})
	if err != nil {
		t.Fatalf("encode responses response: %v", err)
	}
	if !strings.Contains(string(respBody), `"object":"response"`) || !strings.Contains(string(respBody), `"output_text":"pong"`) {
		t.Fatalf("unexpected encoded response: %s", string(respBody))
	}
	streamBody, err := EncodeOpenAIResponsesStream(domain.UnifiedChatResponse{Protocol: domain.ProtocolOpenAI, ID: "resp_1", Model: "gpt-5.4", Message: domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}}})
	if err != nil {
		t.Fatalf("encode responses stream: %v", err)
	}
	if !strings.Contains(string(streamBody), "event: response.created") || !strings.Contains(string(streamBody), "event: response.completed") {
		t.Fatalf("unexpected stream encoding: %s", string(streamBody))
	}
}

func TestResponsesCodecRequestEncodeAndResponseDecode(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{
			{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: "be precise"}}},
			{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
			{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "fc_1", Name: "opencode", Arguments: json.RawMessage(`{"prompt":"ping"}`)}}},
			{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: `{"ok":true}`}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"fc_1"`)}},
		},
	}, &domain.GatewaySessionState{PreviousResponseID: "resp_prev", Metadata: map[string]string{"store": "false"}})
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(data), `"instructions":"be precise"`) || !strings.Contains(string(data), `"previous_response_id":"resp_prev"`) || !strings.Contains(string(data), `"function_call_output"`) {
		t.Fatalf("unexpected encoded responses request: %s", string(data))
	}

	decodedResp, err := DecodeOpenAIResponsesResponse([]byte(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5.4","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]},{"id":"fc_1","type":"function_call","call_id":"fc_1","name":"opencode","arguments":"{\"prompt\":\"ping\"}"}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
	if err != nil {
		t.Fatalf("decode responses response: %v", err)
	}
	if decodedResp.ID != "resp_1" || decodedResp.Message.Parts[0].Text != "pong" || len(decodedResp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected decoded responses response: %+v", decodedResp)
	}
	if len(decodedResp.Message.ToolCalls[0].OutputItem) == 0 {
		t.Fatalf("expected responses output item in IR field, got %+v", decodedResp.Message.ToolCalls[0])
	}
}

func TestResponsesCodecEncodesAssistantHistoryAsOutputText(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{
			{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
			{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(data), `"role":"assistant"`) {
		t.Fatalf("expected assistant message in encoded request: %s", string(data))
	}
	if !strings.Contains(string(data), `"text":"pong","type":"output_text"`) && !strings.Contains(string(data), `"type":"output_text","text":"pong"`) {
		t.Fatalf("expected assistant history to encode as output_text: %s", string(data))
	}
	if strings.Contains(string(data), `"role":"assistant","content":[{"type":"input_text"`) || strings.Contains(string(data), `"role":"assistant","type":"message","content":[{"type":"input_text"`) || strings.Contains(string(data), `"role":"assistant","content":[{"text":"pong","type":"input_text"`) {
		t.Fatalf("assistant history must not encode as input_text: %s", string(data))
	}
}

func TestResponsesCodecFiltersUnsupportedMetadataFields(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}},
		Metadata: map[string]json.RawMessage{
			"context_window":     json.RawMessage(`4096`),
			"max_context_tokens": json.RawMessage(`8192`),
			"service_tier":       json.RawMessage(`"auto"`),
		},
	}, nil)
	if err != nil {
		t.Fatalf("encode responses request with filtered metadata: %v", err)
	}
	if strings.Contains(string(data), `"context_window"`) {
		t.Fatalf("unsupported metadata field should be dropped: %s", string(data))
	}
	if strings.Contains(string(data), `"max_context_tokens"`) {
		t.Fatalf("unsupported metadata field should be dropped: %s", string(data))
	}
	if !strings.Contains(string(data), `"service_tier":"auto"`) {
		t.Fatalf("supported metadata field should be preserved: %s", string(data))
	}
}

func TestResponsesCodecPreservesReasoningAndBuiltInToolOutputItems(t *testing.T) {
	decodedResp, err := DecodeOpenAIResponsesResponse([]byte(`{
		"id":"resp_2",
		"object":"response",
		"status":"completed",
		"model":"gpt-5.4",
		"output":[
			{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"first step"},{"type":"summary_text","text":"second step"}]},
			{"id":"ws_1","type":"web_search_call","status":"completed","action":{"query":"OpenCrab"}}
		],
		"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}
	}`))
	if err != nil {
		t.Fatalf("decode responses response: %v", err)
	}
	if len(decodedResp.Message.Parts) != 2 {
		t.Fatalf("unexpected parts: %+v", decodedResp.Message.Parts)
	}
	if decodedResp.Message.Parts[0].Type != "reasoning" || !strings.Contains(decodedResp.Message.Parts[0].Text, "first step") {
		t.Fatalf("unexpected reasoning part: %+v", decodedResp.Message.Parts[0])
	}
	if decodedResp.Message.Parts[1].Type != "web_search_call" {
		t.Fatalf("unexpected built-in tool part: %+v", decodedResp.Message.Parts[1])
	}
	if len(decodedResp.Message.Parts[0].OutputItem) == 0 || len(decodedResp.Message.Parts[1].OutputItem) == 0 {
		t.Fatalf("expected responses output items in IR fields: %+v", decodedResp.Message.Parts)
	}

	encoded, err := EncodeOpenAIResponsesResponse(decodedResp)
	if err != nil {
		t.Fatalf("encode responses response: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"reasoning"`) || !strings.Contains(string(encoded), `"type":"web_search_call"`) {
		t.Fatalf("unexpected encoded responses response: %s", string(encoded))
	}
}

func TestResponsesCodecPreservesBuiltInToolInputItems(t *testing.T) {
	decoded, err := DecodeOpenAIResponsesRequest([]byte(`{
		"model":"gpt-5.4",
		"input":[
			{"type":"reasoning","summary":[{"type":"summary_text","text":"keep this"}]},
			{"type":"web_search_call","id":"ws_1","status":"completed","action":{"query":"OpenCrab"}},
			{"role":"user","content":[{"type":"input_text","text":"ping"}]}
		]
	}`))
	if err != nil {
		t.Fatalf("decode responses request: %v", err)
	}
	if len(decoded.Messages) != 3 {
		t.Fatalf("unexpected decoded messages: %+v", decoded.Messages)
	}
	if decoded.Messages[0].Parts[0].Type != "reasoning" || decoded.Messages[1].Parts[0].Type != "web_search_call" {
		t.Fatalf("unexpected preserved input parts: %+v", decoded.Messages)
	}
	if len(decoded.Messages[0].Parts[0].InputItem) == 0 || len(decoded.Messages[1].Parts[0].InputItem) == 0 {
		t.Fatalf("expected responses input items in IR fields: %+v", decoded.Messages)
	}

	encoded, err := EncodeOpenAIResponsesRequest(decoded, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"reasoning"`) || !strings.Contains(string(encoded), `"type":"web_search_call"`) {
		t.Fatalf("unexpected re-encoded responses request: %s", string(encoded))
	}
}

func TestResponsesCodecPreservesFunctionCallInputItemShape(t *testing.T) {
	decoded, err := DecodeOpenAIResponsesRequest([]byte(`{
		"model":"gpt-5.4",
		"input":[
			{"type":"function_call","call_id":"fc_1","name":"opencode","arguments":"{\"prompt\":\"ping\"}","status":"completed"}
		]
	}`))
	if err != nil {
		t.Fatalf("decode responses request: %v", err)
	}
	if len(decoded.Messages) != 1 || len(decoded.Messages[0].ToolCalls) != 1 {
		t.Fatalf("unexpected decoded request: %+v", decoded)
	}
	if len(decoded.Messages[0].ToolCalls[0].InputItem) == 0 {
		t.Fatalf("expected function call input item in IR field: %+v", decoded.Messages[0].ToolCalls[0])
	}
	encoded, err := EncodeOpenAIResponsesRequest(decoded, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"function_call"`) || !strings.Contains(string(encoded), `"status":"completed"`) {
		t.Fatalf("unexpected re-encoded function call item: %s", string(encoded))
	}
}

func TestResponsesCodecPreservesFunctionCallOutputItemShape(t *testing.T) {
	decoded, err := DecodeOpenAIResponsesRequest([]byte(`{
		"model":"gpt-5.4",
		"input":[
			{"type":"function_call_output","call_id":"fc_1","output":"done","status":"completed"}
		]
	}`))
	if err != nil {
		t.Fatalf("decode responses request: %v", err)
	}
	if len(decoded.Messages) != 1 || decoded.Messages[0].Role != "tool" {
		t.Fatalf("unexpected decoded request: %+v", decoded)
	}
	if len(decoded.Messages[0].InputItem) == 0 {
		t.Fatalf("expected function call output item in IR field: %+v", decoded.Messages[0])
	}
	encoded, err := EncodeOpenAIResponsesRequest(decoded, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(encoded), `"type":"function_call_output"`) || !strings.Contains(string(encoded), `"status":"completed"`) {
		t.Fatalf("unexpected re-encoded function call output item: %s", string(encoded))
	}
}

func TestResponsesCodecEncodesClaudeToolResultStructuredOutput(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{
			{
				Role: "tool",
				Parts: []domain.UnifiedPart{{
					Type:        "tool_result",
					NativePayload: json.RawMessage(`{"type":"tool_result","tool_use_id":"call_1","content":[]}`),
				}},
				Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(data), `"function_call_output"`) || !strings.Contains(string(data), `"call_id":"call_1"`) || !strings.Contains(string(data), `"output":[]`) {
		t.Fatalf("unexpected encoded structured tool result: %s", string(data))
	}
}

func TestResponsesCodecKeepsOrphanFunctionCallWithoutOutput(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{
			{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}},
			{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "opencode", Arguments: json.RawMessage(`{"prompt":"ping"}`)}}},
		},
		Metadata: map[string]json.RawMessage{"__opencrab_repair_tool_pairs": json.RawMessage(`true`)},
	}, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(data), `"type":"function_call"`) || !strings.Contains(string(data), `"call_id":"call_1"`) {
		t.Fatalf("orphan function_call should be preserved: %s", string(data))
	}
}

func TestResponsesCodecKeepsOrphanOutputWhenPreviousResponseIDPresent(t *testing.T) {
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{
		Protocol: domain.ProtocolOpenAI,
		Model:    "gpt-5.4",
		Messages: []domain.UnifiedMessage{
			{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: "done"}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
		},
	}, &domain.GatewaySessionState{PreviousResponseID: "resp_prev"})
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	if !strings.Contains(string(data), `"previous_response_id":"resp_prev"`) || !strings.Contains(string(data), `"function_call_output"`) {
		t.Fatalf("orphan function_call_output should be kept with previous_response_id: %s", string(data))
	}
}

func TestClaudeCodecToolUseAndResult(t *testing.T) {
	decoded, err := DecodeClaudeChatResponse([]byte(`{"id":"msg_tool","model":"claude-sonnet","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"opencode","input":{"prompt":"ping"}}],"stop_reason":"tool_use"}`))
	if err != nil {
		t.Fatalf("decode claude tool_use response: %v", err)
	}
	if len(decoded.Message.ToolCalls) != 1 || decoded.Message.ToolCalls[0].Name != "opencode" {
		t.Fatalf("unexpected claude tool calls: %+v", decoded.Message.ToolCalls)
	}
	data, err := EncodeClaudeChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolClaude, Model: "claude-sonnet", Messages: []domain.UnifiedMessage{{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: `{"ok":true}`}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"toolu_1"`)}}}, Tools: []json.RawMessage{json.RawMessage(`{"name":"opencode","input_schema":{"type":"object"}}`)}})
	if err != nil {
		t.Fatalf("encode claude tool_result request: %v", err)
	}
	if !strings.Contains(string(data), `"tool_result"`) || !strings.Contains(string(data), `"tools"`) || !strings.Contains(string(data), `"role":"user"`) || strings.Contains(string(data), `"role":"tool"`) {
		t.Fatalf("unexpected claude payload: %s", string(data))
	}
}

func TestClaudeCodecPreservesStructuredToolResultBlocks(t *testing.T) {
	req, err := DecodeClaudeChatRequest([]byte(`{
		"model":"claude-sonnet",
		"messages":[
			{
				"role":"user",
				"content":[
					{"type":"tool_result","tool_use_id":"toolu_1","is_error":true,"content":[{"type":"text","text":"broken"}]},
					{"type":"text","text":"after"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("decode claude request: %v", err)
	}
	if req.Messages[0].Role != "tool" || req.Messages[0].Parts[0].Text != "broken" {
		t.Fatalf("unexpected decoded tool result message: %+v", req.Messages[0])
	}
	if len(req.Messages[0].Parts[0].NativePayload) == 0 {
		t.Fatalf("expected claude native payload, got %+v", req.Messages[0].Parts[0])
	}

	encoded, err := EncodeClaudeChatRequest(req)
	if err != nil {
		t.Fatalf("encode claude request: %v", err)
	}
	for _, snippet := range []string{`"role":"user"`, `"tool_result"`, `"is_error":true`, `"after"`} {
		if !strings.Contains(string(encoded), snippet) {
			t.Fatalf("expected %s in encoded payload: %s", snippet, string(encoded))
		}
	}
	if strings.Contains(string(encoded), `"role":"tool"`) {
		t.Fatalf("unexpected tool role leak: %s", string(encoded))
	}
}

func TestClaudeCodecSplitsMultipleToolResultsIntoDistinctMessages(t *testing.T) {
	req, err := DecodeClaudeChatRequest([]byte(`{
		"model":"claude-sonnet",
		"messages":[
			{
				"role":"user",
				"content":[
					{"type":"tool_result","tool_use_id":"call_read","content":[{"type":"text","text":"read failed"}]},
					{"type":"tool_result","tool_use_id":"call_bash","content":"bash ok"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("decode claude request: %v", err)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("expected split tool results into 2 messages, got %d: %+v", len(req.Messages), req.Messages)
	}
	for i, expectedID := range []string{"call_read", "call_bash"} {
		msg := req.Messages[i]
		if msg.Role != "tool" {
			t.Fatalf("message %d expected tool role, got %+v", i, msg)
		}
		if got := decodeStringRaw(msg.Metadata["tool_call_id"]); got != expectedID {
			t.Fatalf("message %d expected tool_call_id %s, got %s", i, expectedID, got)
		}
		if len(msg.Parts) == 0 || len(msg.Parts[0].NativePayload) == 0 {
			t.Fatalf("message %d expected structured native payload, got %+v", i, msg)
		}
	}

	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolOpenAI, Model: "gpt-5.4", Messages: req.Messages}, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	for _, snippet := range []string{`"function_call_output"`, `"call_id":"call_read"`, `"call_id":"call_bash"`} {
		if !strings.Contains(string(data), snippet) {
			t.Fatalf("expected %s in encoded responses payload: %s", snippet, string(data))
		}
	}
}

func TestClaudeCodecDecodeMessageSingleToolResultPreservesBinding(t *testing.T) {
	msg, err := decodeClaudeMessage(map[string]json.RawMessage{
		"role":    json.RawMessage(`"user"`),
		"content": json.RawMessage(`[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]`),
	})
	if err != nil {
		t.Fatalf("decode claude message: %v", err)
	}
	if msg.Role != "tool" {
		t.Fatalf("expected tool role, got %+v", msg)
	}
	if got := decodeStringRaw(msg.Metadata["tool_call_id"]); got != "toolu_1" {
		t.Fatalf("expected tool_call_id toolu_1, got %+v", msg.Metadata)
	}
}

func TestClaudeCodecRemovesThinkingWhenToolChoiceRequiresTool(t *testing.T) {
	data, err := EncodeClaudeChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolClaude, Model: "claude-sonnet", Messages: []domain.UnifiedMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}}, Metadata: map[string]json.RawMessage{
		"thinking":    json.RawMessage(`{"type":"enabled","budget_tokens":1024}`),
		"tool_choice": json.RawMessage(`{"type":"any"}`),
	}})
	if err != nil {
		t.Fatalf("encode claude request: %v", err)
	}
	if strings.Contains(string(data), `"thinking"`) {
		t.Fatalf("expected thinking to be stripped, got %s", string(data))
	}
}

func TestClaudeCodecRejectsInvalidCacheControlOrdering(t *testing.T) {
	err := normalizeClaudeCompatibilityPayload(map[string]any{
		"messages":      []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "ping", "cache_control": map[string]any{"type": "ephemeral", "ttl": "5m"}}}}},
		"cache_control": map[string]any{"type": "ephemeral", "ttl": "1h"},
	})
	if err == nil || !strings.Contains(err.Error(), "ttl") {
		t.Fatalf("expected ttl ordering error, got %v", err)
	}
}

func TestClaudeCodecDisablesManualThinkingForOpus47(t *testing.T) {
	data, err := EncodeClaudeChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolClaude, Model: "claude-opus-4.7", Messages: []domain.UnifiedMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}}, Metadata: map[string]json.RawMessage{
		"thinking": json.RawMessage(`{"type":"enabled","budget_tokens":1024}`),
	}})
	if err != nil {
		t.Fatalf("encode claude request: %v", err)
	}
	if strings.Contains(string(data), `"thinking"`) {
		t.Fatalf("expected manual thinking to be stripped for opus 4.7, got %s", string(data))
	}
}

func TestClaudeCodecPreservesAdvancedTopLevelMetadata(t *testing.T) {
	req, err := DecodeClaudeChatRequest([]byte(`{
		"model":"claude-sonnet-4-5",
		"messages":[{"role":"user","content":[{"type":"text","text":"ping"}]}],
		"mcp_servers":[{"name":"repo","type":"url","url":"https://example.com/mcp"}],
		"container":{"type":"auto"},
		"context_management":{"clear_function_results":false}
	}`))
	if err != nil {
		t.Fatalf("decode claude request: %v", err)
	}
	for _, key := range []string{"mcp_servers", "container", "context_management"} {
		if len(req.Metadata[key]) == 0 {
			t.Fatalf("expected metadata %s to be preserved, got %+v", key, req.Metadata)
		}
	}

	encoded, err := EncodeClaudeChatRequest(req)
	if err != nil {
		t.Fatalf("encode claude request: %v", err)
	}
	for _, snippet := range []string{`"mcp_servers"`, `"container"`, `"context_management"`} {
		if !strings.Contains(string(encoded), snippet) {
			t.Fatalf("expected %s in encoded payload: %s", snippet, string(encoded))
		}
	}
}

func TestGeminiCodecFunctionCallAndResponse(t *testing.T) {
	decoded, err := DecodeGeminiChatResponse([]byte(`{"modelVersion":"gemini-2.0-flash","candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[{"functionCall":{"id":"fc_1","name":"opencode","args":{"prompt":"ping"}}}]}}]}`))
	if err != nil {
		t.Fatalf("decode gemini functionCall response: %v", err)
	}
	if len(decoded.Message.ToolCalls) != 1 || decoded.Message.ToolCalls[0].Name != "opencode" {
		t.Fatalf("unexpected gemini tool calls: %+v", decoded.Message.ToolCalls)
	}
	if decoded.Message.ToolCalls[0].Order == nil || len(decoded.Message.ToolCalls[0].NativePayload) == 0 {
		t.Fatalf("expected gemini tool call IR fields, got %+v", decoded.Message.ToolCalls[0])
	}
	data, err := EncodeGeminiChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolGemini, Model: "gemini-2.0-flash", Messages: []domain.UnifiedMessage{{Role: "tool", Parts: []domain.UnifiedPart{{Type: "function_response", Metadata: map[string]json.RawMessage{"id": json.RawMessage(`"fc_1"`), "name": json.RawMessage(`"opencode"`), "response": json.RawMessage(`{"ok":true}`)}}}}}, Tools: []json.RawMessage{json.RawMessage(`{"functionDeclarations":[{"name":"opencode"}]}`)}})
	if err != nil {
		t.Fatalf("encode gemini functionResponse request: %v", err)
	}
	if !strings.Contains(string(data), `"functionResponse"`) || !strings.Contains(string(data), `"tools"`) {
		t.Fatalf("unexpected gemini payload: %s", string(data))
	}

	encodedResp, err := EncodeGeminiChatResponse(decoded)
	if err != nil {
		t.Fatalf("encode gemini functionCall response: %v", err)
	}
	if !strings.Contains(string(encodedResp), `"functionCall"`) {
		t.Fatalf("expected functionCall in encoded gemini response: %s", string(encodedResp))
	}
}

func TestGeminiCodecSplitsMultipleFunctionResponsesIntoDistinctMessages(t *testing.T) {
	req, err := DecodeGeminiChatRequest([]byte(`{
		"model":"gemini-2.0-flash",
		"contents":[
			{
				"role":"user",
				"parts":[
					{"functionResponse":{"id":"call_read","name":"Read","response":{"error":"bad pages"}}},
					{"functionResponse":{"id":"call_bash","name":"Bash","response":{"ok":true}}}
				]
			}
		]
	}`), "")
	if err != nil {
		t.Fatalf("decode gemini request: %v", err)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 split tool messages, got %d: %+v", len(req.Messages), req.Messages)
	}
	for i, expectedID := range []string{"call_read", "call_bash"} {
		msg := req.Messages[i]
		if msg.Role != "tool" {
			t.Fatalf("message %d expected tool role, got %+v", i, msg)
		}
		if got := decodeStringRaw(msg.Metadata["tool_call_id"]); got != expectedID {
			t.Fatalf("message %d expected tool_call_id %s, got %s", i, expectedID, got)
		}
		if len(msg.Parts) == 0 || len(msg.Parts[0].NativePayload) == 0 {
			t.Fatalf("message %d expected native payload, got %+v", i, msg)
		}
	}
	data, err := EncodeOpenAIResponsesRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolOpenAI, Model: "gpt-5.4", Messages: req.Messages}, nil)
	if err != nil {
		t.Fatalf("encode responses request: %v", err)
	}
	for _, snippet := range []string{`"function_call_output"`, `"call_id":"call_read"`, `"call_id":"call_bash"`} {
		if !strings.Contains(string(data), snippet) {
			t.Fatalf("expected %s in encoded responses payload: %s", snippet, string(data))
		}
	}
}

func TestGeminiCodecPreservesFunctionResponsePayloadAndPartOrder(t *testing.T) {
	req, err := DecodeGeminiChatRequest([]byte(`{
		"model":"gemini-2.5-pro",
		"contents":[
			{
				"role":"user",
				"parts":[
					{"functionResponse":{"id":"fc_1","name":"lookup","response":{"ok":true},"parts":[{"text":"raw"}]}}
				]
			}
		]
	}`), "")
	if err != nil {
		t.Fatalf("decode gemini request: %v", err)
	}
	encodedReq, err := EncodeGeminiChatRequest(req)
	if err != nil {
		t.Fatalf("encode gemini request: %v", err)
	}
	for _, snippet := range []string{`"functionResponse"`, `"parts":[{"text":"raw"}]`, `"name":"lookup"`} {
		if !strings.Contains(string(encodedReq), snippet) {
			t.Fatalf("expected %s in encoded gemini request: %s", snippet, string(encodedReq))
		}
	}
	if len(req.Messages[0].Parts[0].NativePayload) == 0 {
		t.Fatalf("expected gemini request native payload, got %+v", req.Messages[0].Parts[0])
	}

	resp, err := DecodeGeminiChatResponse([]byte(`{
		"modelVersion":"gemini-2.5-pro",
		"candidates":[{
			"finishReason":"STOP",
			"content":{
				"role":"model",
				"parts":[
					{"text":"before"},
					{"functionCall":{"id":"fc_1","name":"lookup","args":{"q":"ping"}},"thoughtSignature":"sig_1"}
				]
			}
		}]
	}`))
	if err != nil {
		t.Fatalf("decode gemini response: %v", err)
	}
	encodedResp, err := EncodeGeminiChatResponse(resp)
	if err != nil {
		t.Fatalf("encode gemini response: %v", err)
	}
	textIndex := strings.Index(string(encodedResp), `"text":"before"`)
	callIndex := strings.Index(string(encodedResp), `"functionCall"`)
	if textIndex < 0 || callIndex < 0 || textIndex > callIndex {
		t.Fatalf("expected text part before functionCall: %s", string(encodedResp))
	}
	if !strings.Contains(string(encodedResp), `"thoughtSignature":"sig_1"`) {
		t.Fatalf("expected thoughtSignature in encoded gemini response: %s", string(encodedResp))
	}
	if resp.Message.Parts[0].Order == nil || resp.Message.ToolCalls[0].Order == nil || len(resp.Message.ToolCalls[0].NativePayload) == 0 {
		t.Fatalf("expected gemini response IR ordering/native fields, got parts=%+v calls=%+v", resp.Message.Parts, resp.Message.ToolCalls)
	}
}

func TestResponsesStreamBuildsDetailedBuiltInToolEvents(t *testing.T) {
	resp, err := DecodeOpenAIResponsesResponse([]byte(`{
		"id":"resp_stream",
		"object":"response",
		"status":"completed",
		"model":"gpt-5.4",
		"output":[
			{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"step one"}]},
			{"id":"ws_1","type":"web_search_call","status":"completed","action":{"query":"OpenCrab"}},
			{"id":"mcp_1","type":"mcp_call","status":"completed","arguments":"{\"path\":\"README.md\"}"},
			{"id":"ct_1","type":"custom_tool_call","status":"completed","input":"echo hi"},
			{"id":"ci_1","type":"code_interpreter_call","status":"completed","code":"print(1)"}
		]
	}`))
	if err != nil {
		t.Fatalf("decode responses response: %v", err)
	}

	events, err := BuildOpenAIResponsesEvents(resp)
	if err != nil {
		t.Fatalf("build responses events: %v", err)
	}
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal responses events: %v", err)
	}
	for _, snippet := range []string{
		`"response.reasoning_summary_part.added"`,
		`"response.web_search_call.completed"`,
		`"response.mcp_call_arguments.done"`,
		`"response.custom_tool_call_input.done"`,
		`"response.code_interpreter_call.code.done"`,
	} {
		if !strings.Contains(string(encoded), snippet) {
			t.Fatalf("expected %s in responses events: %s", snippet, string(encoded))
		}
	}
}

func TestGeminiCodecPreservesAdvancedTopLevelMetadata(t *testing.T) {
	req, err := DecodeGeminiChatRequest([]byte(`{
		"model":"gemini-2.5-pro",
		"cachedContent":"cachedContents/123",
		"contents":[{"role":"user","parts":[{"text":"ping"}]}],
		"tools":[{"googleSearch":{}},{"urlContext":{}},{"codeExecution":{}}],
		"generationConfig":{"thinkingConfig":{"includeThoughts":true}}
	}`), "")
	if err != nil {
		t.Fatalf("decode gemini request: %v", err)
	}
	if len(req.Metadata["cachedContent"]) == 0 || len(req.Metadata["generationConfig"]) == 0 {
		t.Fatalf("expected gemini metadata to be preserved, got %+v", req.Metadata)
	}

	encoded, err := EncodeGeminiChatRequest(req)
	if err != nil {
		t.Fatalf("encode gemini request: %v", err)
	}
	for _, snippet := range []string{`"cachedContent"`, `"googleSearch"`, `"urlContext"`, `"codeExecution"`, `"thinkingConfig"`} {
		if !strings.Contains(string(encoded), snippet) {
			t.Fatalf("expected %s in encoded payload: %s", snippet, string(encoded))
		}
	}
}

func TestGeminiCodecPreservesThoughtSignatureAndUnknownPartBlocks(t *testing.T) {
	resp, err := DecodeGeminiChatResponse([]byte(`{
		"modelVersion":"gemini-2.5-pro",
		"candidates":[{
			"finishReason":"STOP",
			"content":{
				"role":"model",
				"parts":[
					{"text":"pong","thoughtSignature":"abc123"},
					{"executableCode":{"language":"python","code":"print(1)"}}
				]
			}
		}]
	}`))
	if err != nil {
		t.Fatalf("decode gemini response: %v", err)
	}
	if len(resp.Message.Parts) != 2 {
		t.Fatalf("unexpected parts: %+v", resp.Message.Parts)
	}
	if string(resp.Message.Parts[0].Metadata["thoughtSignature"]) != `"abc123"` {
		t.Fatalf("expected thought signature metadata, got %+v", resp.Message.Parts[0].Metadata)
	}
	if resp.Message.Parts[1].Type == "text" {
		t.Fatalf("expected unknown part block to be preserved, got %+v", resp.Message.Parts[1])
	}

	encoded, err := EncodeGeminiChatResponse(resp)
	if err != nil {
		t.Fatalf("encode gemini response: %v", err)
	}
	if !strings.Contains(string(encoded), `"thoughtSignature":"abc123"`) || !strings.Contains(string(encoded), `"executableCode"`) {
		t.Fatalf("unexpected encoded gemini response: %s", string(encoded))
	}
}

func TestCodecBoundaryErrors(t *testing.T) {
	_, err := DecodeOpenAIChatRequest([]byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":123}]}`))
	if err == nil {
		t.Fatalf("expected openai content type error")
	}

	_, err = DecodeClaudeChatRequest([]byte(`{"model":"claude-3-5-haiku-latest","messages":[{"role":"user","content":[{"type":"image","source":{}}]}]}`))
	if err != nil {
		t.Fatalf("unexpected claude image decode error: %v", err)
	}

	_, err = DecodeGeminiChatRequest([]byte(`{"model":"gemini-2.0-flash","contents":[{"parts":[{"text":"ping","inlineData":{}}]}]}`), "")
	if err != nil {
		t.Fatalf("unexpected gemini inlineData decode error: %v", err)
	}
}
