package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesProxyJSON(t *testing.T) {
	var receivedAPIKey string
	var receivedVersion string
	var receivedBeta string
	var receivedBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedVersion = r.Header.Get("Anthropic-Version")
		receivedBeta = r.Header.Get("Anthropic-Beta")
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("request-id", "req_test")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","content":[{"type":"text","text":"pong"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-json", upstream.URL, "anthropic-configured-key", "claude-sonnet-4-5")

	payload := `{"model":"claude-sonnet-4-5","max_tokens":256,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), payload, map[string]string{
		"Content-Type":      "application/json",
		"Anthropic-Beta":    "tools-2024-04-04",
		"X-API-Key":         "caller-key-should-lose",
		"Anthropic-Version": "2023-01-01",
	})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages endpoint to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "application/json")
	if receivedAPIKey != "anthropic-configured-key" {
		t.Fatalf("expected configured anthropic key to win, got %q", receivedAPIKey)
	}
	if receivedVersion != "2023-06-01" {
		t.Fatalf("expected configured anthropic version, got %q", receivedVersion)
	}
	if receivedBeta != "tools-2024-04-04" {
		t.Fatalf("expected anthropic beta header to be forwarded, got %q", receivedBeta)
	}
	if receivedBody["model"] != "claude-sonnet-4-5" {
		t.Fatalf("expected model to be forwarded, got %#v", receivedBody["model"])
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read messages response body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"message"`) || !strings.Contains(text, `"text":"pong"`) {
		t.Fatalf("expected native claude response body, got %s", text)
	}
}

func TestMessagesStreamPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
		flusher.Flush()
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{
		"OPENCRAB_UPSTREAM_TIMEOUT": "200ms",
	})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-stream", upstream.URL, "anthropic-stream-key", "claude-sonnet-4-5")

	payload := `{"model":"claude-sonnet-4-5","max_tokens":256,"stream":true,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages stream to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "text/event-stream")
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read messages stream: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "event: message_start") || !strings.Contains(text, "event: content_block_delta") || !strings.Contains(text, "event: message_stop") {
		t.Fatalf("expected native anthropic stream events, got %s", text)
	}
}

func TestMessagesUsesCompatibleClaudeChannelRoute(t *testing.T) {
	var compatHits int
	var receivedPath string
	var receivedAPIKey string
	var receivedVersion string
	compatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		compatHits++
		receivedPath = r.URL.Path
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedVersion = r.Header.Get("Anthropic-Version")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_compat","type":"message","role":"assistant","content":[{"type":"text","text":"pong"}],"model":"mimo-v2.5-pro","stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}`))
	}))
	defer compatUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createClaudeChannel(t, apiAddr, "claude-main", compatUpstream.URL+"/anthropic", "compat-key", "mimo-v2.5-pro")

	payload := `{"model":"mimo-v2.5-pro","max_tokens":256,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages endpoint to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if compatHits != 1 {
		t.Fatalf("expected compat upstream to be hit once, got %d", compatHits)
	}
	if receivedPath != "/anthropic/v1/messages" {
		t.Fatalf("expected compat upstream path /anthropic/v1/messages, got %q", receivedPath)
	}
	if receivedAPIKey != "compat-key" {
		t.Fatalf("expected compat api key, got %q", receivedAPIKey)
	}
	if receivedVersion != "2023-06-01" {
		t.Fatalf("expected anthropic version header, got %q", receivedVersion)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read messages response body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"id":"msg_compat"`) || !strings.Contains(text, `"text":"pong"`) {
		t.Fatalf("expected compat claude response body, got %s", text)
	}
}

func TestMessagesPrefersClaudeRouteBeforeOpenAIFallback(t *testing.T) {
	var claudeHits int
	claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_claude","type":"message","role":"assistant","content":[{"type":"text","text":"claude-first"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":2}}`))
	}))
	defer claudeUpstream.Close()

	var openAIHits int
	openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openAIHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_unused","object":"chat.completion","model":"claude-sonnet-4-5","choices":[{"index":0,"message":{"role":"assistant","content":"should-not-run"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`))
	}))
	defer openAIUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
	createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages endpoint to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if claudeHits != 1 {
		t.Fatalf("expected claude upstream to be used once, got %d", claudeHits)
	}
	if openAIHits != 0 {
		t.Fatalf("expected openai fallback to stay unused, got %d hits", openAIHits)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(string(body), `claude-first`) {
		t.Fatalf("expected claude response body, got %s", string(body))
	}
}

func TestMessagesFallsBackToOpenAIJSON(t *testing.T) {
	var claudeHits int
	claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeHits++
		http.Error(w, "try next", http.StatusServiceUnavailable)
	}))
	defer claudeUpstream.Close()

	var receivedAuthorization string
	var receivedBody map[string]any
	openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_bridge","object":"chat.completion","created":1710000000,"model":"claude-sonnet-4-5","choices":[{"index":0,"message":{"role":"assistant","content":"fallback-json-ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":9,"completion_tokens":4,"total_tokens":13}}`))
	}))
	defer openAIUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
	createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

	payload := `{"model":"claude-sonnet-4-5","max_tokens":128,"system":"You are helpful","stop_sequences":["END","DONE"],"temperature":0.25,"top_p":0.8,"tools":[{"name":"lookup","description":"Look up data","input_schema":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}],"tool_choice":{"type":"tool","name":"lookup"},"messages":[{"role":"user","content":[{"type":"text","text":"Need weather"}]},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"lookup","input":{"city":"Paris"}}]},{"role":"user","content":[{"type":"text","text":"before result"},{"type":"tool_result","tool_use_id":"toolu_1","content":"Paris is sunny"},{"type":"text","text":"after result"}]}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected fallback request to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if claudeHits != 1 {
		t.Fatalf("expected claude upstream to be attempted once, got %d", claudeHits)
	}
	if receivedAuthorization != "Bearer openai-key" {
		t.Fatalf("expected openai bearer auth, got %q", receivedAuthorization)
	}
	if receivedBody["model"] != "claude-sonnet-4-5" {
		t.Fatalf("expected model in fallback body, got %#v", receivedBody["model"])
	}
	if receivedBody["max_tokens"] == nil {
		t.Fatalf("expected max_tokens in fallback body")
	}
	if receivedBody["temperature"] != 0.25 {
		t.Fatalf("expected mapped temperature, got %#v", receivedBody["temperature"])
	}
	if receivedBody["top_p"] != 0.8 {
		t.Fatalf("expected mapped top_p, got %#v", receivedBody["top_p"])
	}
	stop, ok := receivedBody["stop"].([]any)
	if !ok || len(stop) != 2 || stop[0] != "END" || stop[1] != "DONE" {
		t.Fatalf("expected mapped stop sequences, got %#v", receivedBody["stop"])
	}
	tools, ok := receivedBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one mapped tool, got %#v", receivedBody["tools"])
	}
	toolChoice, ok := receivedBody["tool_choice"].(map[string]any)
	if !ok || toolChoice["type"] != "function" {
		t.Fatalf("expected mapped tool_choice, got %#v", receivedBody["tool_choice"])
	}
	messages, ok := receivedBody["messages"].([]any)
	if !ok || len(messages) != 6 {
		t.Fatalf("expected six mapped openai messages, got %#v", receivedBody["messages"])
	}
	if first, ok := messages[0].(map[string]any); !ok || first["role"] != "system" {
		t.Fatalf("expected first mapped message to be system, got %#v", messages[0])
	}
	if second, ok := messages[1].(map[string]any); !ok || second["role"] != "user" || second["content"] != "Need weather" {
		t.Fatalf("expected initial user text to stay ordered, got %#v", messages[1])
	}
	if third, ok := messages[2].(map[string]any); !ok || third["role"] != "assistant" {
		t.Fatalf("expected assistant tool_use message, got %#v", messages[2])
	}
	if fourth, ok := messages[3].(map[string]any); !ok || fourth["role"] != "user" || fourth["content"] != "before result" {
		t.Fatalf("expected user text before tool_result to stay before tool role, got %#v", messages[3])
	}
	if fifth, ok := messages[4].(map[string]any); !ok || fifth["role"] != "tool" {
		t.Fatalf("expected tool_result to map to tool role, got %#v", messages[4])
	}
	if sixth, ok := messages[5].(map[string]any); !ok || sixth["role"] != "user" || sixth["content"] != "after result" {
		t.Fatalf("expected user text after tool_result to stay after tool role, got %#v", messages[5])
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fallback body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"message"`) || !strings.Contains(text, `"fallback-json-ok"`) {
		t.Fatalf("expected bridged claude response body, got %s", text)
	}
	if !strings.Contains(text, `"stop_reason":"end_turn"`) {
		t.Fatalf("expected claude stop_reason mapping, got %s", text)
	}
	if !strings.Contains(text, `"input_tokens":9`) || !strings.Contains(text, `"output_tokens":4`) {
		t.Fatalf("expected claude usage mapping, got %s", text)
	}
}

func TestMessagesFallsBackToOpenAIStream(t *testing.T) {
	var claudeHits int
	claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeHits++
		http.Error(w, "try next", http.StatusServiceUnavailable)
	}))
	defer claudeUpstream.Close()

	openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hel\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":2,\"total_tokens\":7}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer openAIUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{"OPENCRAB_UPSTREAM_TIMEOUT": "200ms"})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
	createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected fallback stream to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if claudeHits != 1 {
		t.Fatalf("expected claude upstream to be attempted once, got %d", claudeHits)
	}
	assertContentTypeContains(t, response, "text/event-stream")
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fallback stream: %v", err)
	}
	text := string(body)
	for _, marker := range []string{"event: message_start", "event: content_block_start", "event: content_block_delta", "event: content_block_stop", "event: message_delta", "event: message_stop", `"text":"Hel"`, `"text":"lo"`, `"output_tokens":2`} {
		if !strings.Contains(text, marker) {
			t.Fatalf("expected bridged anthropic stream marker %q, got %s", marker, text)
		}
	}
	if strings.Contains(text, "[DONE]") {
		t.Fatalf("expected raw openai done marker to be consumed, got %s", text)
	}
}

func TestMessagesFallbackMapsOpenAIErrorTypes(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		expectedType string
	}{
		{name: "forbidden", statusCode: http.StatusForbidden, expectedType: "permission_error"},
		{name: "not found", statusCode: http.StatusNotFound, expectedType: "not_found_error"},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, expectedType: "rate_limit_error"},
		{name: "overloaded", statusCode: 529, expectedType: "overloaded_error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "try next", http.StatusServiceUnavailable)
			}))
			defer claudeUpstream.Close()

			openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(`{"error":{"type":"upstream_error","message":"mapped error"}}`))
			}))
			defer openAIUpstream.Close()

			apiAddr := reserveLocalAddress(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
			defer stopProcess(t, cmd)
			createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
			createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

			response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
			defer response.Body.Close()

			if response.StatusCode != tc.statusCode {
				body, _ := io.ReadAll(response.Body)
				t.Fatalf("expected status %d, got %d: %s", tc.statusCode, response.StatusCode, string(body))
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), `"type":"`+tc.expectedType+`"`) {
				t.Fatalf("expected anthropic error type %q, got %s", tc.expectedType, string(body))
			}
		})
	}
}

func TestMessagesFallsBackToOpenAIStreamToolUse(t *testing.T) {
	claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "try next", http.StatusServiceUnavailable)
	}))
	defer claudeUpstream.Close()

	openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream_tool\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"city\\\":\\\"Pa\"}}]},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream_tool\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"ris\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":3,\"total_tokens\":10}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer openAIUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
	createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected stream tool fallback to return 200, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read stream tool fallback body: %v", err)
	}
	text := string(body)
	for _, marker := range []string{"event: content_block_start", `"type":"tool_use"`, `"name":"lookup"`, `"type":"input_json_delta"`, `"partial_json":"{\"city\":\"Pa"`, `"partial_json":"ris\"}"`, `"stop_reason":"tool_use"`} {
		if !strings.Contains(text, marker) {
			t.Fatalf("expected bridged tool_use stream marker %q, got %s", marker, text)
		}
	}
}

func TestMessagesFallsBackToOpenAIMultiToolUseStream(t *testing.T) {
	claudeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "try next", http.StatusServiceUnavailable)
	}))
	defer claudeUpstream.Close()

	openAIUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream_multi_tool\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\"}}]},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream_multi_tool\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":1,\"id\":\"call_2\",\"type\":\"function\",\"function\":{\"name\":\"weather\",\"arguments\":\"{}\"}}]},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_stream_multi_tool\",\"object\":\"chat.completion.chunk\",\"model\":\"claude-sonnet-4-5\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"}\"}}]},\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2,\"total_tokens\":6}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer openAIUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-primary", claudeUpstream.URL, "claude-key", "claude-sonnet-4-5")
	createOpenAIChannel(t, apiAddr, "openai-fallback", openAIUpstream.URL+"/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read multi tool stream body: %v", err)
	}
	text := string(body)
	for _, marker := range []string{`"index":0`, `"index":1`, `"name":"lookup"`, `"name":"weather"`} {
		if !strings.Contains(text, marker) {
			t.Fatalf("expected multi tool-use stream marker %q, got %s", marker, text)
		}
	}
	if strings.Count(text, `"name":"lookup"`) < 2 {
		t.Fatalf("expected lookup tool block to reopen with preserved identity, got %s", text)
	}
}

func TestMessagesFallbackRejectsUnsupportedRole(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-only", "https://example.com/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"system","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read unsupported role body: %v", err)
	}
	text := string(body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(text, `"invalid_request_error"`) || !strings.Contains(text, `messages[0].role`) {
		t.Fatalf("expected stable invalid role error, got %d: %s", response.StatusCode, text)
	}
}

func TestMessagesFallbackRejectsMissingToolIdentifiers(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-only", "https://example.com/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"assistant","content":[{"type":"tool_use","name":"lookup","input":{}}]}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing tool identifier body: %v", err)
	}
	text := string(body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(text, `"invalid_request_error"`) || !strings.Contains(text, `tool_use.id is required`) {
		t.Fatalf("expected stable missing tool id error, got %d: %s", response.StatusCode, text)
	}

	response = doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","input":{}}]}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing tool name body: %v", err)
	}
	text = string(body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(text, `tool_use.name is required`) {
		t.Fatalf("expected stable missing tool name error, got %d: %s", response.StatusCode, text)
	}

	response = doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":[{"type":"tool_result","content":"done"}]}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing tool_result id body: %v", err)
	}
	text = string(body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(text, `tool_result.tool_use_id is required`) {
		t.Fatalf("expected stable missing tool_result id error, got %d: %s", response.StatusCode, text)
	}
}

func TestMessagesFallbackRejectsUnsupportedClaudeFields(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-only", "https://example.com/v1", "openai-key", "claude-sonnet-4-5")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"thinking":{"type":"enabled","budget_tokens":32},"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected unsupported fallback field to return 400, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read unsupported field body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"error"`) || !strings.Contains(text, `"invalid_request_error"`) {
		t.Fatalf("expected anthropic-style invalid request error, got %s", text)
	}
	if !strings.Contains(text, `thinking is not supported when routing Claude Messages through OpenAI chat completions`) {
		t.Fatalf("expected stable unsupported field error, got %s", text)
	}
}

func TestMessagesValidationError(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","messages":[]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages validation error 400, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read validation body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"error"`) || !strings.Contains(text, `"invalid_request_error"`) {
		t.Fatalf("expected anthropic-style validation error, got %s", text)
	}
}

func TestMessagesTransportErrorIsStable(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{
		"OPENCRAB_UPSTREAM_TIMEOUT": "300ms",
	})
	defer stopProcess(t, cmd)
	createClaudeChannel(t, apiAddr, "claude-broken", "http://127.0.0.1:1", "", "claude-sonnet-4-5")

	payload := `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected messages bad gateway, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read transport body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"type":"api_error"`) || !strings.Contains(text, `"Upstream request failed"`) {
		t.Fatalf("expected stable anthropic transport error, got %s", text)
	}
	if strings.Contains(text, `dial tcp`) || strings.Contains(text, `127.0.0.1:1`) {
		t.Fatalf("expected transport internals to be hidden, got %s", text)
	}
}

func TestMessagesRouteMissingIsStable(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := doPOST(t, fmt.Sprintf("http://%s/v1/messages", apiAddr), `{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected missing route to return 502, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing route body: %v", err)
	}
	if !strings.Contains(string(body), `No enabled claude route configured for model claude-sonnet-4-5`) {
		t.Fatalf("expected stable route error, got %s", string(body))
	}
}
