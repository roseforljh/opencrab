package transform

import (
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestNormalizeGatewayRequestCodexResponses(t *testing.T) {
	req, err := NormalizeGatewayRequest(
		Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses},
		[]byte(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`),
		NormalizeOptions{},
	)
	if err != nil {
		t.Fatalf("normalize codex request: %v", err)
	}
	if req.Protocol != domain.ProtocolCodex || req.Operation != domain.ProtocolOperationCodexResponses {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestRenderClientResponseCodexResponses(t *testing.T) {
	body, headers, err := RenderClientResponse(
		Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses},
		domain.UnifiedChatResponse{
			Protocol: domain.ProtocolCodex,
			ID:       "resp_1",
			Model:    "gpt-5.4",
			Message:  domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
		},
		false,
	)
	if err != nil {
		t.Fatalf("render codex response: %v", err)
	}
	if headers["Content-Type"][0] != "application/json" || !strings.Contains(string(body), `"object":"response"`) {
		t.Fatalf("unexpected rendered codex response: headers=%#v body=%s", headers, string(body))
	}
}

func TestRenderClientResponseGeminiStream(t *testing.T) {
	body, headers, err := RenderClientResponse(
		Surface{Protocol: domain.ProtocolGemini, Operation: domain.ProtocolOperationGeminiStreamGenerate},
		domain.UnifiedChatResponse{
			Protocol: domain.ProtocolGemini,
			Model:    "gemini-2.5-pro",
			Message:  domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
		},
		true,
	)
	if err != nil {
		t.Fatalf("render gemini stream: %v", err)
	}
	if headers["Content-Type"][0] != "text/event-stream" || !strings.Contains(string(body), `"candidates"`) {
		t.Fatalf("unexpected gemini stream: headers=%#v body=%s", headers, string(body))
	}
}
