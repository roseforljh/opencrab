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

type trackingReadCloser struct {
	reader    *strings.Reader
	readCalls int
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	r.readCalls++
	return r.reader.Read(p)
}

func (r *trackingReadCloser) Close() error { return nil }

func testGatewayMessage(role string, parts ...domain.UnifiedPart) domain.GatewayMessage {
	return domain.GatewayMessage{Role: role, Parts: parts}
}

func TestOpenAIExecutorReturnsStreamResultWhenStreamEnabled(t *testing.T) {
	body := &trackingReadCloser{reader: strings.NewReader("data: hello\n\n")}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body}, nil
	})}

	result, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request:       domain.GatewayRequest{Model: "gpt-4o", Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Stream == nil || result.Response != nil {
		t.Fatalf("unexpected result: %#v", result)
	}
	if body.readCalls != 0 {
		t.Fatalf("expected no pre-read for stream body, got %d", body.readCalls)
	}
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
		Request:       domain.GatewayRequest{Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}},
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
	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages: %#v", captured["messages"])
	}
	first := messages[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("unexpected message payload: %#v", first)
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
		Request:       domain.GatewayRequest{Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"}), testGatewayMessage("assistant", domain.UnifiedPart{Type: "text", Text: "world"})}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
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

func TestClaudeExecutorBuildsMultimodalRequest(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	_, err := NewClaudeExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request: domain.GatewayRequest{Messages: []domain.GatewayMessage{testGatewayMessage("user",
			domain.UnifiedPart{Type: "image", Metadata: map[string]json.RawMessage{"mime_type": json.RawMessage(`"image/png"`), "data": json.RawMessage(`"abc"`)}},
			domain.UnifiedPart{Type: "text", Text: "describe"},
		)}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	messages := captured["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["type"] != "image" {
		t.Fatalf("unexpected multimodal payload: %#v", content)
	}
}

func TestGeminiExecutorUsesStreamGenerateContentURLWhenStreamEnabled(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})}

	result, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request:       domain.GatewayRequest{Model: "gpt-4o", Stream: true, Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Stream == nil {
		t.Fatalf("expected stream result: %#v", result)
	}
}
