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

func TestCodecBoundaryErrors(t *testing.T) {
	_, err := DecodeOpenAIChatRequest([]byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":123}]}`))
	if err == nil {
		t.Fatalf("expected openai content type error")
	}

	_, err = DecodeClaudeChatRequest([]byte(`{"model":"claude-3-5-haiku-latest","messages":[{"role":"user","content":[{"type":"image","source":{}}]}]}`))
	if err == nil {
		t.Fatalf("expected claude non-text error")
	}

	_, err = DecodeGeminiChatRequest([]byte(`{"model":"gemini-2.0-flash","contents":[{"parts":[{"text":"ping","inlineData":{}}]}]}`), "")
	if err == nil {
		t.Fatalf("expected gemini metadata boundary error")
	}
}
