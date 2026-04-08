package provider

import (
	"context"
	"net/http"
	"testing"

	"opencrab/internal/domain"
)

func TestBuildTestRequestForOpenAICompatibleProviders(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		endpoint    string
		model       string
		expectURL   string
		expectModel string
	}{
		{name: "openai", provider: "OpenAI", endpoint: "https://api.openai.com/v1", model: "", expectURL: "https://api.openai.com/v1/chat/completions", expectModel: "gpt-4o-mini"},
		{name: "glm", provider: "GLM", endpoint: "https://open.bigmodel.cn/api/paas/v4", model: "", expectURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions", expectModel: "glm-4-flash"},
		{name: "kimi", provider: "KIMI", endpoint: "https://api.moonshot.ai/v1", model: "", expectURL: "https://api.moonshot.ai/v1/chat/completions", expectModel: "moonshot-v1-8k"},
		{name: "minimax", provider: "MiniMAX", endpoint: "https://api.minimax.chat", model: "", expectURL: "https://api.minimax.chat/v1/chat/completions", expectModel: "MiniMax-Text-01"},
		{name: "openrouter", provider: "OpenRouter", endpoint: "https://openrouter.ai/api/v1", model: "", expectURL: "https://openrouter.ai/api/v1/chat/completions", expectModel: "openai/gpt-4o-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := domain.UpstreamChannel{Provider: tt.provider, Endpoint: tt.endpoint, APIKey: "sk-test"}
			usedModel := tt.model
			if usedModel == "" {
				usedModel = defaultTestModel(normalizeProvider(tt.provider))
			}

			req, err := buildTestRequest(context.Background(), channel, normalizeProvider(tt.provider), usedModel)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}

			if req.URL.String() != tt.expectURL {
				t.Fatalf("expected url %s, got %s", tt.expectURL, req.URL.String())
			}
			if req.Header.Get("Authorization") != "Bearer sk-test" {
				t.Fatalf("expected bearer auth header, got %q", req.Header.Get("Authorization"))
			}
			if req.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", req.Method)
			}
			if usedModel != tt.expectModel {
				t.Fatalf("expected model %s, got %s", tt.expectModel, usedModel)
			}
		})
	}
}

func TestBuildTestRequestForClaude(t *testing.T) {
	channel := domain.UpstreamChannel{Provider: "Claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"}
	req, err := buildTestRequest(context.Background(), channel, normalizeProvider(channel.Provider), defaultTestModel(normalizeProvider(channel.Provider)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if req.URL.String() != "https://api.anthropic.com/v1/messages" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}
	if req.Header.Get("x-api-key") != "claude-key" {
		t.Fatalf("expected x-api-key header, got %q", req.Header.Get("x-api-key"))
	}
	if req.Header.Get("anthropic-version") != "2023-06-01" {
		t.Fatalf("unexpected anthropic version: %s", req.Header.Get("anthropic-version"))
	}
}

func TestBuildTestRequestForGemini(t *testing.T) {
	channel := domain.UpstreamChannel{Provider: "Gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"}
	req, err := buildTestRequest(context.Background(), channel, normalizeProvider(channel.Provider), defaultTestModel(normalizeProvider(channel.Provider)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if req.URL.String() != "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}
	if req.Header.Get("x-goog-api-key") != "gemini-key" {
		t.Fatalf("expected x-goog-api-key header, got %q", req.Header.Get("x-goog-api-key"))
	}
}

func TestNormalizeProvider(t *testing.T) {
	tests := map[string]string{
		"OpenAI":     "openai",
		"GLM":        "glm",
		"zhipu":      "glm",
		"KIMI":       "kimi",
		"moonshot":   "kimi",
		"MiniMAX":    "minimax",
		"OpenRouter": "openrouter",
		"Claude":     "claude",
		"Gemini":     "gemini",
	}

	for input, expect := range tests {
		if got := normalizeProvider(input); got != expect {
			t.Fatalf("normalizeProvider(%q) expected %q, got %q", input, expect, got)
		}
	}
}
