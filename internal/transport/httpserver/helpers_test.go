package httpserver

import (
	"net/http/httptest"
	"testing"

	"opencrab/internal/domain"
)

func TestExtractModelFromRequestPrefersBodyModel(t *testing.T) {
	model := extractModelFromRequest("/v1/chat/completions", []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hi"}]}`))
	if model != "gpt-5.4" {
		t.Fatalf("expected body model, got %q", model)
	}
}

func TestExtractModelFromRequestFallsBackToGeminiPathModel(t *testing.T) {
	model := extractModelFromRequest("/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent", []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`))
	if model != "gemini-3.1-pro-preview" {
		t.Fatalf("expected path model, got %q", model)
	}
}

func TestExtractGatewaySessionIDPrefersClaudeCodeHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("X-Session-ID", "legacy-session")
	req.Header.Set("X-Claude-Code-Session-Id", "claude-code-session")

	sessionID := extractGatewaySessionID(req)
	if sessionID != "claude-code-session" {
		t.Fatalf("expected claude code session header, got %q", sessionID)
	}
}

func TestExtractSessionAffinityKeyReadsClaudeCodeHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("X-Claude-Code-Session-Id", "claude-code-session")

	affinityKey := ExtractSessionAffinityKey(req, domain.GatewayRequest{}, domain.GatewayRuntimeSettings{
		StickyEnabled:   true,
		StickyKeySource: "auto",
	})
	if affinityKey != "claude-code-session" {
		t.Fatalf("expected affinity key from claude code session header, got %q", affinityKey)
	}
}
