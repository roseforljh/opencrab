package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClaudeExecutorBuildsNativeTextRequest(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.anthropic.com/v1/messages" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		if req.Header.Get("x-api-key") != "claude-key" {
			t.Fatalf("unexpected x-api-key: %s", req.Header.Get("x-api-key"))
		}
		if req.Header.Get("anthropic-version") != anthropicVersion {
			t.Fatalf("unexpected anthropic-version: %s", req.Header.Get("anthropic-version"))
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	executor := NewClaudeExecutor(client)
	_, err := executor.Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request: domain.GatewayRequest{Stream: true, Messages: []domain.GatewayMessage{
			{Role: "system", Text: "be precise"},
			{Role: "user", Text: "hello"},
		}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if captured["model"] != "claude-3-5-sonnet" {
		t.Fatalf("unexpected model: %v", captured["model"])
	}
	if captured["stream"] != true {
		t.Fatalf("expected stream true, got %v", captured["stream"])
	}
	if captured["system"] != "be precise" {
		t.Fatalf("unexpected system: %v", captured["system"])
	}
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages: %#v", captured["messages"])
	}
	first := messages[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("unexpected role: %#v", first)
	}
	content := first["content"].([]any)
	block := content[0].(map[string]any)
	if block["type"] != "text" || block["text"] != "hello" {
		t.Fatalf("unexpected content block: %#v", block)
	}
}

func TestGeminiExecutorBuildsNativeTextRequest(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		if req.Header.Get("x-goog-api-key") != "gemini-key" {
			t.Fatalf("unexpected x-goog-api-key: %s", req.Header.Get("x-goog-api-key"))
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	executor := NewGeminiExecutor(client)
	_, err := executor.Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{Messages: []domain.GatewayMessage{
			{Role: "system", Text: "be precise"},
			{Role: "user", Text: "hello"},
			{Role: "assistant", Text: "world"},
		}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	systemInstruction, ok := captured["system_instruction"].(map[string]any)
	if !ok {
		t.Fatalf("missing system_instruction: %#v", captured)
	}
	systemParts := systemInstruction["parts"].([]any)
	if systemParts[0].(map[string]any)["text"] != "be precise" {
		t.Fatalf("unexpected system parts: %#v", systemInstruction)
	}
	contents, ok := captured["contents"].([]any)
	if !ok || len(contents) != 2 {
		t.Fatalf("unexpected contents: %#v", captured["contents"])
	}
	first := contents[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("unexpected first role: %v", first["role"])
	}
	second := contents[1].(map[string]any)
	if second["role"] != "model" {
		t.Fatalf("unexpected second role: %v", second["role"])
	}
	parts := first["parts"].([]any)
	if parts[0].(map[string]any)["text"] != "hello" {
		t.Fatalf("unexpected first text: %#v", parts)
	}
}

func TestGeminiExecutorRejectsUnsupportedRole(t *testing.T) {
	executor := NewGeminiExecutor(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("request should not be sent")
		return nil, nil
	})})
	_, err := executor.Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request:       domain.GatewayRequest{Messages: []domain.GatewayMessage{{Role: "tool", Text: "x"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "暂不支持") {
		t.Fatalf("unexpected err: %v", err)
	}
}
