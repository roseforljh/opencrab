package provider

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestClaudeExecutorPassesThroughAnthropicBetaHeader(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("anthropic-beta") != "tools-2024-04-04" {
			t.Fatalf("unexpected anthropic-beta: %s", req.Header.Get("anthropic-beta"))
		}
		if req.Header.Get("anthropic-dangerous-direct-browser-access") != "true" {
			t.Fatalf("unexpected dangerous browser header: %s", req.Header.Get("anthropic-dangerous-direct-browser-access"))
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	_, err := NewClaudeExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request:       domain.GatewayRequest{Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}, RequestHeaders: map[string]string{"anthropic-beta": "tools-2024-04-04", "anthropic-dangerous-direct-browser-access": "true"}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestOpenAIExecutorPassesThroughCustomHeaders(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("OpenAI-Beta") != "responses=v1" {
			t.Fatalf("unexpected OpenAI-Beta: %s", req.Header.Get("OpenAI-Beta"))
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request:       domain.GatewayRequest{Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}, RequestHeaders: map[string]string{"OpenAI-Beta": "responses=v1"}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestOpenAIExecutorUsesResponsesEndpointForNativeResponses(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.openai.com/v1/responses" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5.4","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}]}`))}, nil
	})}
	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-5.4",
		Request: domain.GatewayRequest{
			Operation: domain.ProtocolOperationOpenAIResponses,
			Messages: []domain.GatewayMessage{
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"}),
			},
			Session: &domain.GatewaySessionState{PreviousResponseID: "resp_prev"},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if captured["model"] != "gpt-5.4" || captured["previous_response_id"] != "resp_prev" {
		t.Fatalf("unexpected responses payload: %#v", captured)
	}
	if got, ok := captured["stream"].(bool); ok && got {
		t.Fatalf("expected native responses request to stay non-stream, got %#v", captured)
	}
}

func TestOpenAIExecutorPreservesFunctionCallWhenBridgingClaudeToolUse(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.openai.com/v1/responses" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5.4","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}]}`))}, nil
	})}
	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-5.4",
		Request: domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationOpenAIResponses,
			Messages: []domain.GatewayMessage{
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "ping"}),
				{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "opencode", Arguments: json.RawMessage(`{"prompt":"ping"}`)}}},
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	input, ok := captured["input"].([]any)
	if !ok {
		t.Fatalf("unexpected input payload: %#v", captured["input"])
	}
	serialized, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if !strings.Contains(string(serialized), `"type":"function_call"`) || !strings.Contains(string(serialized), `"call_id":"call_1"`) {
		t.Fatalf("expected bridged payload to keep function_call, got %s", string(serialized))
	}
}

func TestOpenAIExecutorReturnsSyntheticResponsesStreamWhenRequested(t *testing.T) {
	var captured map[string]any
	body := &trackingReadCloser{reader: strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5.4","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}]}`)}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.openai.com/v1/responses" {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(payload, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: body}, nil
	})}

	result, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-5.4",
		Request: domain.GatewayRequest{
			Operation: domain.ProtocolOperationOpenAIResponses,
			Stream:    true,
			Messages:  []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Stream != nil {
		t.Fatalf("expected buffered json response for synthetic stream path: %#v", result)
	}
	if result.Response == nil {
		t.Fatalf("expected non-stream response result")
	}
	if body.readCalls == 0 {
		t.Fatalf("expected executor to fully read non-stream responses body")
	}
	if got, ok := captured["stream"].(bool); ok && got {
		t.Fatalf("expected upstream responses request to disable native stream, got %#v", captured)
	}
}

func TestOpenAIExecutorStripsResponsesOnlyFields(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})}, Metadata: map[string]json.RawMessage{
			"previous_response_id": json.RawMessage(`"resp_1"`),
			"include":              json.RawMessage(`["reasoning.encrypted_content"]`),
			"store":                json.RawMessage(`false`),
			"reasoning":            json.RawMessage(`{"effort":"medium"}`),
		}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, key := range []string{"previous_response_id", "include", "store", "reasoning"} {
		if _, exists := captured[key]; exists {
			t.Fatalf("unexpected leaked key %s in payload: %#v", key, captured)
		}
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
	if _, exists := captured["model"]; exists {
		t.Fatalf("gemini payload should not include model in body: %#v", captured)
	}
	if _, exists := captured["stream"]; exists {
		t.Fatalf("gemini payload should not include stream in body: %#v", captured)
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

func TestGeminiExecutorTransformsOpenAIStructuredOutputsToGenerationConfig(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"response_format": json.RawMessage(`{"type":"json_schema","json_schema":{"name":"answer","schema":{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["response_format"]; exists {
		t.Fatalf("unexpected response_format leak: %#v", captured)
	}
	config, ok := captured["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing generationConfig: %#v", captured)
	}
	if config["responseMimeType"] != "application/json" {
		t.Fatalf("unexpected responseMimeType: %#v", config)
	}
	if _, ok := config["responseSchema"].(map[string]any); !ok {
		t.Fatalf("missing responseSchema: %#v", config)
	}
}

func TestOpenAIExecutorTransformsGeminiStructuredOutputsToResponseFormat(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"generationConfig": json.RawMessage(`{"responseMimeType":"application/json","responseSchema":{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["generationConfig"]; exists {
		t.Fatalf("unexpected generationConfig leak: %#v", captured)
	}
	responseFormat, ok := captured["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("missing response_format: %#v", captured)
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("unexpected response_format: %#v", responseFormat)
	}
	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("missing json_schema: %#v", responseFormat)
	}
	if _, ok := jsonSchema["schema"].(map[string]any); !ok {
		t.Fatalf("missing response_format schema: %#v", jsonSchema)
	}
}

func TestGeminiExecutorTransformsOpenAIReasoningToThinkingConfig(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"reasoning": json.RawMessage(`{"effort":"high","summary":"auto"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	config, ok := captured["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing generationConfig: %#v", captured)
	}
	thinkingConfig, ok := config["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing thinkingConfig: %#v", config)
	}
	if thinkingConfig["thinkingBudget"] == nil || thinkingConfig["includeThoughts"] != true {
		t.Fatalf("unexpected thinkingConfig: %#v", thinkingConfig)
	}
	if _, exists := captured["reasoning"]; exists {
		t.Fatalf("unexpected reasoning leak: %#v", captured)
	}
}

func TestGeminiExecutorTransformsOpenAIToolsAndToolChoice(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools: []json.RawMessage{
				json.RawMessage(`{"type":"function","function":{"name":"1 bad tool/name with spaces and extremely-long-suffix-for-gemini-validation-over-sixty-four-characters","description":"Read file","parameters":{"type":["object","null"],"properties":{"file_path":{"type":"string","deprecated":true},"mode":{"anyOf":[{"type":"string","enum":["a","b"],"enumTitles":["A","B"]},{"type":"null"}]},"schema_ref":{"$ref":"#/$defs/sample"}},"required":["file_path"],"additionalProperties":false,"patternProperties":{"^x-":{"type":"string"}},"propertyNames":{"pattern":"^[a-z]+$"},"$defs":{"sample":{"type":"string"}},"$schema":"http://json-schema.org/draft-07/schema#","$id":"root","const":"x"}}}`),
			},
			Metadata: map[string]json.RawMessage{
				"tool_choice": json.RawMessage(`"required"`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected gemini tools in payload: %#v", captured)
	}
	tool0, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first tool payload: %#v", tools[0])
	}
	decls, ok := tool0["functionDeclarations"].([]any)
	if !ok || len(decls) != 1 {
		t.Fatalf("expected functionDeclarations payload: %#v", tool0)
	}
	decl, ok := decls[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected declaration payload: %#v", decls[0])
	}
	name, _ := decl["name"].(string)
	if name == "" || len(name) > 64 {
		t.Fatalf("expected sanitized gemini function name, got %#v", decl)
	}
	if strings.ContainsAny(name, " /") {
		t.Fatalf("expected sanitized gemini function name without spaces or slashes, got %#v", decl)
	}
	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected declaration parameters map: %#v", decl)
	}
	if params["type"] != "OBJECT" {
		t.Fatalf("expected gemini-style uppercase schema type, got %#v", params)
	}
	for _, forbidden := range []string{"$schema", "$id", "$defs", "$ref", "const", "additionalProperties", "patternProperties", "propertyNames"} {
		if _, exists := params[forbidden]; exists {
			t.Fatalf("unexpected %s leakage in gemini params: %#v", forbidden, params)
		}
	}
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map: %#v", params)
	}
	filePath, ok := props["file_path"].(map[string]any)
	if !ok || filePath["type"] != "STRING" {
		t.Fatalf("expected nested gemini-style schema type, got %#v", props)
	}
	if _, exists := filePath["deprecated"]; exists {
		t.Fatalf("unexpected deprecated leakage in gemini params: %#v", filePath)
	}
	mode, ok := props["mode"].(map[string]any)
	if !ok || mode["type"] != "STRING" {
		t.Fatalf("expected flattened anyOf schema to resolve to STRING, got %#v", props)
	}
	if _, exists := mode["enumTitles"]; exists {
		t.Fatalf("unexpected enumTitles leakage in gemini params: %#v", mode)
	}
	toolConfig, ok := captured["toolConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected toolConfig in gemini payload: %#v", captured)
	}
	functionCallingConfig, ok := toolConfig["functionCallingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected functionCallingConfig in gemini payload: %#v", toolConfig)
	}
	if functionCallingConfig["mode"] != "ANY" {
		t.Fatalf("expected required tool_choice to map to ANY, got %#v", functionCallingConfig)
	}
}

func TestGeminiExecutorMovesOpenAIControlsIntoGenerationConfig(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"max_tokens":  json.RawMessage(`256`),
				"temperature": json.RawMessage(`0.7`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, exists := captured["max_tokens"]; exists {
		t.Fatalf("unexpected max_tokens leak into gemini payload: %#v", captured)
	}
	if _, exists := captured["temperature"]; exists {
		t.Fatalf("unexpected temperature leak into gemini payload: %#v", captured)
	}
	config, ok := captured["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected generationConfig in gemini payload: %#v", captured)
	}
	if _, exists := config["maxOutputTokens"]; exists {
		t.Fatalf("expected maxOutputTokens to be removed for gemini payload: %#v", config)
	}
	if config["temperature"] != 0.7 {
		t.Fatalf("expected temperature=0.7, got %#v", config)
	}
}

func TestGeminiExecutorDropsInterruptedAndTodoReminderMessagesFromOpenAIHistory(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{
				testGatewayMessage("system", domain.UnifiedPart{Type: "text", Text: "You are Droid"}),
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "<system-reminder>IMPORTANT: TodoWrite was not called yet. You must call it for any non-trivial task requested by the user."}, domain.UnifiedPart{Type: "text", Text: "你好"}),
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "<system-reminder>\n\nUser system info (win32 10.0.26200)\nModel: gemini-3.1-pro-preview"}, domain.UnifiedPart{Type: "text", Text: "继续修复"}),
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "Request interrupted by user"}),
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "Request cancelled by user"}),
				testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "继续修复"}),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["system_instruction"]; !exists {
		t.Fatalf("expected system_instruction in gemini payload: %#v", captured)
	}
	contents, ok := captured["contents"].([]any)
	if !ok {
		t.Fatalf("expected contents array in gemini payload: %#v", captured)
	}
	serialized, err := json.Marshal(contents)
	if err != nil {
		t.Fatalf("marshal contents: %v", err)
	}
	body := string(serialized)
	if strings.Contains(body, "Request interrupted by user") {
		t.Fatalf("unexpected interrupted marker in gemini contents: %s", body)
	}
	if strings.Contains(body, "Request cancelled by user") {
		t.Fatalf("unexpected cancelled marker in gemini contents: %s", body)
	}
	if strings.Contains(body, "TodoWrite was not called yet") {
		t.Fatalf("unexpected TodoWrite reminder in gemini contents: %s", body)
	}
	if strings.Contains(body, "User system info") {
		t.Fatalf("unexpected generic system reminder leakage in gemini contents: %s", body)
	}
	if !strings.Contains(body, "你好") || !strings.Contains(body, "继续修复") {
		t.Fatalf("expected real user text to survive filtering: %s", body)
	}
}

func TestAttachPayloadDebugMetadata(t *testing.T) {
	err := attachPayloadDebugMetadata(fmt.Errorf("boom"), map[string]string{
		"upstream_provider":       "gemini",
		"upstream_operation":      "gemini_generate_content",
		"upstream_request_url":    "https://example.com/v1beta/models/gemini:generateContent",
		"upstream_request_stream": "true",
		"upstream_payload_sha256": "abc123",
		"upstream_payload_preview": `{"tools":[{"functionDeclarations":[]}]}`,
	})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	text := err.Error()
	for _, snippet := range []string{"boom", "upstream_provider=gemini", "upstream_operation=gemini_generate_content", "upstream_payload_sha256=abc123", `upstream_payload_preview={"tools":[{"functionDeclarations":[]}]}`} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected snippet %q in wrapped error: %s", snippet, text)
		}
	}
}

func TestBuildFocusedPayloadPreviewPrefersToolsAndToolConfig(t *testing.T) {
	payload := []byte(`{"contents":[{"parts":[{"text":"` + strings.Repeat("x", 5000) + `"}],"tool_call_id":"call_1"}],"tools":[{"functionDeclarations":[{"name":"Read","parameters":{"type":"OBJECT"}}]}],"toolConfig":{"functionCallingConfig":{"mode":"ANY"}},"generationConfig":{"temperature":1}}`)
	preview := buildFocusedPayloadPreview(payload)
	if !strings.Contains(preview, `"tools"`) {
		t.Fatalf("expected focused preview to include tools: %s", preview)
	}
	if !strings.Contains(preview, `"toolConfig"`) {
		t.Fatalf("expected focused preview to include toolConfig: %s", preview)
	}
	if strings.Contains(preview, strings.Repeat("x", 200)) {
		t.Fatalf("expected focused preview to avoid giant raw contents dump: %s", preview)
	}
	if !strings.Contains(preview, `"contents_count":1`) {
		t.Fatalf("expected focused preview to include contents_count: %s", preview)
	}
	if !strings.Contains(preview, `"contents_extra_keys":{"contents[0]":["tool_call_id"]}`) {
		t.Fatalf("expected focused preview to include message extra keys: %s", preview)
	}
}

func TestClaudeExecutorTransformsOpenAIReasoningToThinking(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	_, err := NewClaudeExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"reasoning": json.RawMessage(`{"effort":"medium"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["reasoning"]; exists {
		t.Fatalf("unexpected reasoning leak: %#v", captured)
	}
	thinking, ok := captured["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("missing thinking: %#v", captured)
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("unexpected thinking payload: %#v", thinking)
	}
}

func TestOpenAIExecutorTransformsGeminiCodeExecutionTool(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools:    []json.RawMessage{json.RawMessage(`{"codeExecution":{}}`)},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "code_interpreter" {
		t.Fatalf("unexpected tool payload: %#v", tool)
	}
}

func TestOpenAIExecutorTransformsGeminiFunctionDeclarationsToOpenAI(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolGemini,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools: []json.RawMessage{
				json.RawMessage(`{"functionDeclarations":[{"name":"lookup","description":"Find item","parameters":{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}}]}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" {
		t.Fatalf("unexpected tool payload: %#v", tool)
	}
	functionPayload, ok := tool["function"].(map[string]any)
	if !ok || functionPayload["name"] != "lookup" || functionPayload["description"] != "Find item" {
		t.Fatalf("unexpected function payload: %#v", tool)
	}
	parameters, ok := functionPayload["parameters"].(map[string]any)
	if !ok || parameters["type"] != "object" {
		t.Fatalf("unexpected function parameters: %#v", functionPayload)
	}
}

func TestGeminiExecutorTransformsOpenAICodeInterpreterTool(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools:    []json.RawMessage{json.RawMessage(`{"type":"code_interpreter","container":{"type":"auto"}}`)},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if _, ok := tool["codeExecution"].(map[string]any); !ok {
		t.Fatalf("unexpected tool payload: %#v", tool)
	}
}

func TestGeminiExecutorTransformsOpenAIFunctionTools(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[]}`))}, nil
	})}

	_, err := NewGeminiExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "gemini", Endpoint: "https://generativelanguage.googleapis.com", APIKey: "gemini-key"},
		UpstreamModel: "gemini-2.0-flash",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools: []json.RawMessage{
				json.RawMessage(`{"type":"function","function":{"name":"lookup","description":"Find item","parameters":{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	declarations, ok := tool["functionDeclarations"].([]any)
	if !ok || len(declarations) != 1 {
		t.Fatalf("unexpected functionDeclarations payload: %#v", tool)
	}
	declaration := declarations[0].(map[string]any)
	if declaration["name"] != "lookup" || declaration["description"] != "Find item" {
		t.Fatalf("unexpected declaration payload: %#v", declaration)
	}
	parameters, ok := declaration["parameters"].(map[string]any)
	if !ok || parameters["type"] != "OBJECT" {
		t.Fatalf("unexpected declaration parameters: %#v", declaration)
	}
	properties, ok := parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected declaration properties map: %#v", parameters)
	}
	queryProp, ok := properties["q"].(map[string]any)
	if !ok || queryProp["type"] != "STRING" {
		t.Fatalf("unexpected declaration properties: %#v", properties)
	}
}

func TestOpenAIExecutorTransformsClaudeMCPServersToResponsesTool(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}]}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-5.4",
		Request: domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationOpenAIResponses,
			Messages:  []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"mcp_servers": json.RawMessage(`[{"name":"repo","type":"url","url":"https://example.com/mcp"}]`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "mcp" || tool["server_url"] != "https://example.com/mcp" {
		t.Fatalf("unexpected mcp tool: %#v", tool)
	}
	if _, exists := captured["mcp_servers"]; exists {
		t.Fatalf("unexpected mcp_servers leak: %#v", captured)
	}
}

func TestClaudeExecutorTransformsOpenAIMCPToolAndAddsBetaHeader(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.Header.Get("anthropic-beta"), "mcp-client-2025-11-20") {
			t.Fatalf("unexpected anthropic-beta: %s", req.Header.Get("anthropic-beta"))
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

	_, err := NewClaudeExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools:    []json.RawMessage{json.RawMessage(`{"type":"mcp","server_label":"repo","server_url":"https://example.com/mcp"}`)},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["tools"]; exists {
		t.Fatalf("unexpected tools leak in Claude payload: %#v", captured)
	}
	servers, ok := captured["mcp_servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf("unexpected mcp_servers payload: %#v", captured)
	}
	server := servers[0].(map[string]any)
	if server["url"] != "https://example.com/mcp" {
		t.Fatalf("unexpected mcp server payload: %#v", server)
	}
}

func TestClaudeExecutorTransformsOpenAICodeInterpreterToolToContainer(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.Header.Get("anthropic-beta"), "code-execution-2025-08-25") {
			t.Fatalf("unexpected anthropic-beta: %s", req.Header.Get("anthropic-beta"))
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

	_, err := NewClaudeExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "claude", Endpoint: "https://api.anthropic.com", APIKey: "claude-key"},
		UpstreamModel: "claude-3-5-sonnet",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolOpenAI,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools:    []json.RawMessage{json.RawMessage(`{"type":"code_interpreter","container":{"type":"auto"}}`)},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, exists := captured["container"]; !exists {
		t.Fatalf("missing Claude container: %#v", captured)
	}
	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected Claude tools: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "code_execution_20250825" {
		t.Fatalf("unexpected Claude code execution tool: %#v", tool)
	}
}

func TestOpenAIExecutorTransformsClaudeContainerToCodeInterpreter(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}]}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-5.4",
		Request: domain.GatewayRequest{
			Protocol:  domain.ProtocolClaude,
			Operation: domain.ProtocolOperationOpenAIResponses,
			Messages:  []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Metadata: map[string]json.RawMessage{
				"container": json.RawMessage(`{"type":"auto"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected OpenAI tools: %#v", captured)
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "code_interpreter" {
		t.Fatalf("unexpected code interpreter tool: %#v", tool)
	}
	if _, exists := captured["container"]; exists {
		t.Fatalf("unexpected Claude container leak: %#v", captured)
	}
}

func TestOpenAIExecutorTransformsClaudeToolChoiceToOpenAI(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","choices":[{"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}]}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools:    []json.RawMessage{json.RawMessage(`{"name":"opencode","input_schema":{"type":"object"}}`)},
			Metadata: map[string]json.RawMessage{
				"tool_choice": json.RawMessage(`{"type":"tool","name":"opencode"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	toolChoice, ok := captured["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected tool_choice payload: %#v", captured)
	}
	if toolChoice["type"] != "function" {
		t.Fatalf("unexpected tool_choice type: %#v", toolChoice)
	}
	functionPayload, ok := toolChoice["function"].(map[string]any)
	if !ok || functionPayload["name"] != "opencode" {
		t.Fatalf("unexpected tool_choice function payload: %#v", toolChoice)
	}
}

func TestOpenAIExecutorTransformsClaudeFunctionToolsToOpenAI(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","choices":[{"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}]}`))}, nil
	})}

	_, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
			Tools: []json.RawMessage{
				json.RawMessage(`{"name":"opencode","description":"Run operation","input_schema":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", captured)
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["type"] != "function" {
		t.Fatalf("unexpected tool payload: %#v", tools[0])
	}
	functionPayload, ok := tool["function"].(map[string]any)
	if !ok || functionPayload["name"] != "opencode" || functionPayload["description"] != "Run operation" {
		t.Fatalf("unexpected function payload: %#v", tool)
	}
	parameters, ok := functionPayload["parameters"].(map[string]any)
	if !ok || parameters["type"] != "object" {
		t.Fatalf("unexpected function parameters: %#v", functionPayload)
	}
}

func TestOpenAIExecutorDisablesUpstreamStreamForClaudeBridge(t *testing.T) {
	var captured map[string]any
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","choices":[{"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}]}`))}, nil
	})}

	result, err := NewOpenAIExecutor(client).Execute(context.Background(), domain.ExecutorRequest{
		Channel:       domain.UpstreamChannel{Provider: "openai", Endpoint: "https://api.openai.com/v1", APIKey: "sk-test"},
		UpstreamModel: "gpt-4o-mini",
		Request: domain.GatewayRequest{
			Protocol: domain.ProtocolClaude,
			Stream:   true,
			Messages: []domain.GatewayMessage{testGatewayMessage("user", domain.UnifiedPart{Type: "text", Text: "hello"})},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Response == nil || result.Stream != nil {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, exists := captured["stream"]; exists {
		t.Fatalf("cross-protocol stream should be downgraded, got %#v", captured)
	}
}
