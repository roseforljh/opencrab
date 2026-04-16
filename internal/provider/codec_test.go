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

	decoded, err := DecodeGeminiChatRequest(data, "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("decode gemini: %v", err)
	}
	if decoded.Model != "gemini-2.0-flash" || decoded.Messages[0].Role != "assistant" || !decoded.Stream {
		t.Fatalf("unexpected decoded gemini request: %+v", decoded)
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
	if !strings.Contains(string(data), `"tool_result"`) || !strings.Contains(string(data), `"tools"`) {
		t.Fatalf("unexpected claude payload: %s", string(data))
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

func TestGeminiCodecFunctionCallAndResponse(t *testing.T) {
	decoded, err := DecodeGeminiChatResponse([]byte(`{"modelVersion":"gemini-2.0-flash","candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[{"functionCall":{"id":"fc_1","name":"opencode","args":{"prompt":"ping"}}}]}}]}`))
	if err != nil {
		t.Fatalf("decode gemini functionCall response: %v", err)
	}
	if len(decoded.Message.ToolCalls) != 1 || decoded.Message.ToolCalls[0].Name != "opencode" {
		t.Fatalf("unexpected gemini tool calls: %+v", decoded.Message.ToolCalls)
	}
	data, err := EncodeGeminiChatRequest(domain.UnifiedChatRequest{Protocol: domain.ProtocolGemini, Model: "gemini-2.0-flash", Messages: []domain.UnifiedMessage{{Role: "tool", Parts: []domain.UnifiedPart{{Type: "function_response", Metadata: map[string]json.RawMessage{"id": json.RawMessage(`"fc_1"`), "name": json.RawMessage(`"opencode"`), "response": json.RawMessage(`{"ok":true}`)}}}}}, Tools: []json.RawMessage{json.RawMessage(`{"functionDeclarations":[{"name":"opencode"}]}`)}})
	if err != nil {
		t.Fatalf("encode gemini functionResponse request: %v", err)
	}
	if !strings.Contains(string(data), `"functionResponse"`) || !strings.Contains(string(data), `"tools"`) {
		t.Fatalf("unexpected gemini payload: %s", string(data))
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
