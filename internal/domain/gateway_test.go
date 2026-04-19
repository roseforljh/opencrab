package domain

import (
	"encoding/json"
	"testing"
)

func TestUnifiedChatRequestValidateCore(t *testing.T) {
	toolRaw, _ := json.Marshal(map[string]any{"type": "function"})

	tests := []struct {
		name    string
		request UnifiedChatRequest
		wantErr bool
	}{
		{
			name: "valid text only request",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:  "user",
					Parts: []UnifiedPart{{Type: "text", Text: "ping"}},
				}},
			},
		},
		{
			name: "reject empty model",
			request: UnifiedChatRequest{
				Messages: []UnifiedMessage{{
					Role:  "user",
					Parts: []UnifiedPart{{Type: "text", Text: "ping"}},
				}},
			},
			wantErr: true,
		},
		{
			name: "reject invalid role",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:  "developer",
					Parts: []UnifiedPart{{Type: "text", Text: "ping"}},
				}},
			},
			wantErr: true,
		},
		{
			name: "allow non text part",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:  "user",
					Parts: []UnifiedPart{{Type: "image_url", Text: "ignored"}},
				}},
			},
		},
		{
			name: "reject empty text",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:  "user",
					Parts: []UnifiedPart{{Type: "text", Text: "  "}},
				}},
			},
			wantErr: true,
		},
		{
			name: "allow assistant tool calls without content parts",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:      "assistant",
					ToolCalls: []UnifiedToolCall{{Name: "Read"}},
				}},
			},
		},
		{
			name: "allow tools payload",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role:  "user",
					Parts: []UnifiedPart{{Type: "text", Text: "ping"}},
				}},
				Tools: []json.RawMessage{toolRaw},
			},
		},
		{
			name: "allow part metadata",
			request: UnifiedChatRequest{
				Model: "gpt-4o-mini",
				Messages: []UnifiedMessage{{
					Role: "user",
					Parts: []UnifiedPart{{
						Type:     "text",
						Text:     "ping",
						Metadata: map[string]json.RawMessage{"mime_type": json.RawMessage(`"text/plain"`)},
					}},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.ValidateCore()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestChatCompletionsRequestToUnifiedChatRequest(t *testing.T) {
	request := ChatCompletionsRequest{
		Model:  "gpt-4o-mini",
		Stream: true,
		Messages: []ChatCompletionsMessage{{
			Role:    "user",
			Content: "hello",
		}},
	}

	unified := request.ToUnifiedChatRequest()
	if unified.Protocol != ProtocolOpenAI {
		t.Fatalf("expected protocol %q, got %q", ProtocolOpenAI, unified.Protocol)
	}
	if unified.Model != request.Model || !unified.Stream {
		t.Fatalf("unexpected unified request header: %+v", unified)
	}
	if len(unified.Messages) != 1 || len(unified.Messages[0].Parts) != 1 {
		t.Fatalf("unexpected unified messages: %+v", unified.Messages)
	}
	if unified.Messages[0].Parts[0].Text != "hello" {
		t.Fatalf("expected text hello, got %q", unified.Messages[0].Parts[0].Text)
	}
}
