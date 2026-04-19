package provider

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestBuildChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://api.example.com", expect: "https://api.example.com/v1/chat/completions"},
		{name: "v1 endpoint", endpoint: "https://api.example.com/v1", expect: "https://api.example.com/v1/chat/completions"},
		{name: "full endpoint", endpoint: "https://api.example.com/v1/chat/completions", expect: "https://api.example.com/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildChatCompletionsURL(tt.endpoint); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestBuildClaudeMessagesURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://api.anthropic.com", expect: "https://api.anthropic.com/v1/messages"},
		{name: "v1 endpoint", endpoint: "https://api.anthropic.com/v1", expect: "https://api.anthropic.com/v1/messages"},
		{name: "full endpoint", endpoint: "https://api.anthropic.com/v1/messages", expect: "https://api.anthropic.com/v1/messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildClaudeMessagesURL(tt.endpoint); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestBuildClaudeCountTokensURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://api.anthropic.com", expect: "https://api.anthropic.com/v1/messages/count_tokens"},
		{name: "v1 endpoint", endpoint: "https://api.anthropic.com/v1", expect: "https://api.anthropic.com/v1/messages/count_tokens"},
		{name: "messages endpoint", endpoint: "https://api.anthropic.com/v1/messages", expect: "https://api.anthropic.com/v1/messages/count_tokens"},
		{name: "full endpoint", endpoint: "https://api.anthropic.com/v1/messages/count_tokens", expect: "https://api.anthropic.com/v1/messages/count_tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildClaudeCountTokensURL(tt.endpoint); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestBuildGeminiGenerateContentURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		model    string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://generativelanguage.googleapis.com", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"},
		{name: "version endpoint", endpoint: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"},
		{name: "model endpoint", endpoint: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildGeminiGenerateContentURL(tt.endpoint, tt.model); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestBuildGeminiStreamGenerateContentURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		model    string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://generativelanguage.googleapis.com", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse"},
		{name: "version endpoint", endpoint: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse"},
		{name: "model endpoint", endpoint: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash", model: "gemini-2.0-flash", expect: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildGeminiStreamGenerateContentURL(tt.endpoint, tt.model); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}

func TestCopyStreamResponsePrependsSSEBootstrapComment(t *testing.T) {
	rec := httptest.NewRecorder()
	err := CopyStreamResponse(rec, &domain.StreamResult{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("event: response.created\ndata: {}\n\n")),
	})
	if err != nil {
		t.Fatalf("copy stream response: %v", err)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, ": opencrab bootstrap\n\n") {
		t.Fatalf("expected bootstrap comment prefix, got: %q", body)
	}
	if !strings.Contains(body, "event: response.created") {
		t.Fatalf("expected upstream event in body, got: %q", body)
	}
}

func TestCopyStreamResponseDoesNotPrependBootstrapForJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	err := CopyStreamResponse(rec, &domain.StreamResult{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
	})
	if err != nil {
		t.Fatalf("copy stream response: %v", err)
	}
	if strings.HasPrefix(rec.Body.String(), ": opencrab bootstrap") {
		t.Fatalf("did not expect bootstrap comment for json stream: %q", rec.Body.String())
	}
}
