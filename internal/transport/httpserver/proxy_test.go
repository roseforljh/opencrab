package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"strings"
	"testing"
	"time"

	"opencrab/internal/domain"

	"github.com/gorilla/websocket"
)

func TestProxyChatCompletionsCopiesResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test"}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4.1","messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"id":"chatcmpl-test"}` {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIModelsReturnsConfiguredAliases(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("unexpected api key: %s", rawKey)
			}
			return true, nil
		},
		ListModels: func(ctx context.Context) ([]domain.ModelMapping, error) {
			return []domain.ModelMapping{
				{ID: 1, Alias: "gpt-4.1", UpstreamModel: "gpt-4.1"},
				{ID: 2, Alias: "gpt-4o-mini", UpstreamModel: "gpt-4o-mini"},
			}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"object":"list"`) || !strings.Contains(body, `"id":"gpt-4.1"`) || !strings.Contains(body, `"id":"gpt-4o-mini"`) {
		t.Fatalf("unexpected models response: %s", body)
	}
}

func TestProxyResponsesConvertsChatCompletionResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping"}`))
	req.Header.Set("Authorization", "Bearer ***")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"object":"response"`) || !strings.Contains(rec.Body.String(), `"output_text":"pong"`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestProxyResponsesJSONRequestWithStreamHeaderReturnsJSONBody(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping"}`))
	req.Header.Set("Authorization", "Bearer ***")
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected json content-type, got %q", got)
	}
	if strings.Contains(rec.Body.String(), "event: response.created") {
		t.Fatalf("expected json body, got synthetic stream: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"output_text":"pong"`) {
		t.Fatalf("expected assistant output_text in json body: %s", rec.Body.String())
	}
}

func TestProxyResponsesLogsRenderedProxyWriteFailureWithoutOverwritingStatus(t *testing.T) {
	logger, records := newCaptureLogger()
	router := NewRouter(Dependencies{
		Logger: logger,
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)}}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write([]byte(`partial`))
			return fmt.Errorf("boom write failure")
		},
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping"}`))
	req.Header.Set("Authorization", "Bearer ***")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected original 200 status to remain, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "partial") {
		t.Fatalf("expected partial body to be preserved, got %q", rec.Body.String())
	}
	if !captureLogsContain(records, "gateway_response_write_failed") {
		t.Fatalf("expected gateway_response_write_failed log, got %#v", *records)
	}
	if !captureLogsContain(records, "rendered_proxy") {
		t.Fatalf("expected rendered_proxy stage in logs, got %#v", *records)
	}
}

func TestProxyResponsesReturnsSyntheticStream(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping","stream":true}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: response.created") || !strings.Contains(body, "event: response.completed") || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("unexpected stream body: %s", body)
	}
}

func TestProxyResponsesPassesThroughNativeResponsesStream(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			if !req.Stream {
				t.Fatal("responses handler should preserve native stream requests")
			}
			return &domain.ExecutionResult{Stream: &domain.StreamResult{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"text/event-stream"}, "X-Opencrab-Provider": {"openai"}},
				Body:       io.NopCloser(strings.NewReader("event: response.created\ndata: {}\n\ndata: [DONE]\n\n")),
			}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping","stream":true}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: response.created") || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("unexpected native stream body: %s", body)
	}
}

func TestProxyCodexResponsesRendersResponsesShape(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			if req.Protocol != domain.ProtocolCodex || req.Operation != domain.ProtocolOperationCodexResponses {
				t.Fatalf("unexpected codex request: %+v", req)
			}
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"claude"}},
				Body:       []byte(`{"id":"msg_1","model":"claude-sonnet","role":"assistant","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn"}`),
			}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/codex/responses", bytes.NewBufferString(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"object":"response"`) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyResponsesMergesPreviousResponseTranscript(t *testing.T) {
	var callCount int
	var captured []domain.GatewayMessage
	store := NewMemoryResponseSessionStore(16)
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			callCount++
			captured = append([]domain.GatewayMessage(nil), req.Messages...)
			body := `{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
			if callCount == 2 {
				body = `{"id":"chatcmpl-test-2","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"again"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
			}
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(body)}}, nil
		},
		ResponseSessions: store,
		CopyProxy:        copyProxyForTest,
		CopyStream:       copyStreamForTest,
	})
	first := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":"ping"}`))
	first.Header.Set("Authorization", "Bearer sk-opencrab-test")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("unexpected first response: %d %s", firstRec.Code, firstRec.Body.String())
	}
	second := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","previous_response_id":"chatcmpl-test","input":"next"}`))
	second.Header.Set("Authorization", "Bearer sk-opencrab-test")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("unexpected second response: %d %s", secondRec.Code, secondRec.Body.String())
	}
	if len(captured) < 3 {
		t.Fatalf("expected merged transcript, got %#v", captured)
	}
}

func TestResponsesWebSocketCreateAndAppend(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	server := httptest.NewServer(NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			text := "pong"
			id := "resp_1"
			if len(req.Messages) > 2 {
				text = "again"
				id = "resp_2"
			}
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"` + id + `","model":"gpt-5.4","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"` + text + `"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)}}, nil
		},
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/responses"
	headers := http.Header{"Authorization": {"Bearer sk-opencrab-test"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]any{"type": "response.create", "response": map[string]any{"model": "gpt-5.4", "input": []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": "ping"}}}}}}); err != nil {
		t.Fatalf("write create: %v", err)
	}
	var event map[string]any
	seenCompleted := false
	for i := 0; i < 16; i++ {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read create event: %v", err)
		}
		if event["type"] == "response.completed" {
			seenCompleted = true
			break
		}
	}
	if !seenCompleted {
		t.Fatalf("did not receive response.completed")
	}
	if err := conn.WriteJSON(map[string]any{"type": "response.append", "response_id": "resp_1", "input": []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": "next"}}}}}); err != nil {
		t.Fatalf("write append: %v", err)
	}
	seenSecondCompleted := false
	for i := 0; i < 16; i++ {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read append event: %v", err)
		}
		if event["type"] == "response.completed" {
			seenSecondCompleted = true
			break
		}
	}
	if !seenSecondCompleted {
		t.Fatalf("did not receive append response.completed")
	}
}

func TestResponsesWebSocketGenerateFalseWarmup(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	server := httptest.NewServer(NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			t.Fatal("generate=false should not hit upstream")
			return nil, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/responses"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer sk-opencrab-test"}})
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]any{"type": "response.create", "response": map[string]any{"model": "gpt-5.4", "generate": false, "input": []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": "warmup"}}}}}}); err != nil {
		t.Fatalf("write warmup: %v", err)
	}
	var event map[string]any
	seenCompleted := false
	for i := 0; i < 16; i++ {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read warmup event: %v", err)
		}
		if event["type"] == "response.completed" {
			seenCompleted = true
			break
		}
	}
	if !seenCompleted {
		t.Fatalf("did not receive warmup completion")
	}
}

func TestRealtimeWebSocketConversationAndResponse(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	var captured domain.GatewayRequest
	server := httptest.NewServer(NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			captured = req
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}},
				Body:       []byte(`{"id":"resp_rt_1","model":"gpt-realtime","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`),
			}}, nil
		},
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/realtime?model=gpt-realtime"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer sk-opencrab-test"}})
	if err != nil {
		t.Fatalf("dial realtime ws: %v", err)
	}
	defer conn.Close()

	var event map[string]any
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read session.created: %v", err)
	}
	if event["type"] != "session.created" {
		t.Fatalf("unexpected first event: %#v", event)
	}

	if err := conn.WriteJSON(map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{{"type": "input_text", "text": "ping"}},
		},
	}); err != nil {
		t.Fatalf("write conversation item: %v", err)
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read conversation.item.added: %v", err)
	}
	if event["type"] != "conversation.item.added" {
		t.Fatalf("unexpected conversation event: %#v", event)
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read conversation.item.done: %v", err)
	}
	if event["type"] != "conversation.item.done" {
		t.Fatalf("unexpected conversation done event: %#v", event)
	}

	if err := conn.WriteJSON(map[string]any{"type": "response.create", "response": map[string]any{}}); err != nil {
		t.Fatalf("write response.create: %v", err)
	}

	seenDone := false
	seenDelta := false
	for i := 0; i < 16; i++ {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read realtime response event: %v", err)
		}
		switch event["type"] {
		case "response.output_text.delta":
			seenDelta = true
		case "response.done":
			seenDone = true
		}
		if seenDelta && seenDone {
			break
		}
	}
	if !seenDelta || !seenDone {
		t.Fatalf("missing realtime events, delta=%v done=%v", seenDelta, seenDone)
	}
	if captured.Operation != domain.ProtocolOperationOpenAIRealtime || captured.Model != "gpt-realtime" || len(captured.Messages) == 0 {
		t.Fatalf("unexpected captured gateway request: %#v", captured)
	}
}

func TestClaudeContextManagementClearsToolHistoryBeforeExecution(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	store.Put(ResponseSession{
		ResponseID: "resp_ctx_1",
		Model:      "claude-sonnet",
		Messages: []domain.GatewayMessage{
			{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{{ID: "call_1", Name: "lookup", Arguments: json.RawMessage(`{"q":"ping"}`)}}},
			{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: `{"ok":true}`}}, Metadata: map[string]json.RawMessage{"tool_call_id": json.RawMessage(`"call_1"`)}},
			{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "reasoning", Text: "hidden"}, {Type: "text", Text: "visible"}}},
		},
		UpdatedAt: time.Now(),
	})

	var captured domain.GatewayRequest
	router := NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			captured = req
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test"}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-4.1","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}],"previous_response_id":"resp_ctx_1","context_management":{"clear_function_results":true,"clear_thinking":true}}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("unexpected merged messages: %#v", captured.Messages)
	}
	if captured.Messages[0].Role != "assistant" || len(captured.Messages[0].ToolCalls) != 0 || len(captured.Messages[0].Parts) != 1 || captured.Messages[0].Parts[0].Text != "visible" {
		t.Fatalf("context management did not clear history as expected: %#v", captured.Messages)
	}
}

func TestGeminiCachedContentCreateAndUse(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	var captured domain.GatewayRequest
	router := NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			captured = req
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test"}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})

	createReq := httptest.NewRequest(http.MethodPost, "/v1beta/cachedContents", bytes.NewBufferString(`{"model":"gemini-2.5-pro","contents":[{"role":"user","parts":[{"text":"cached prompt"}]}]}`))
	createReq.Header.Set("Authorization", "Bearer sk-opencrab-test")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("unexpected create response: %d %s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	cacheName, _ := created["name"].(string)
	if !strings.HasPrefix(cacheName, "cachedContents/opencrab-") {
		t.Fatalf("unexpected cache name: %#v", created)
	}

	useReq := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", bytes.NewBufferString(fmt.Sprintf(`{"cachedContent":%q,"contents":[{"role":"user","parts":[{"text":"next"}]}]}`, cacheName)))
	useReq.Header.Set("Authorization", "Bearer sk-opencrab-test")
	useRec := httptest.NewRecorder()
	router.ServeHTTP(useRec, useReq)
	if useRec.Code != http.StatusOK {
		t.Fatalf("unexpected use response: %d %s", useRec.Code, useRec.Body.String())
	}
	if len(captured.Messages) < 2 || captured.Messages[0].Parts[0].Text != "cached prompt" {
		t.Fatalf("cached content was not merged: %#v", captured.Messages)
	}
}

func TestGeminiURLContextExpandsIntoSystemMessage(t *testing.T) {
	contextServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>OpenCrab</h1><p>Gateway context page</p></body></html>`))
	}))
	defer contextServer.Close()

	var captured domain.GatewayRequest
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			captured = req
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test"}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})

	body := fmt.Sprintf(`{"contents":[{"role":"user","parts":[{"text":"read %s"}]}],"tools":[{"urlContext":{}}]}`, contextServer.URL)
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if len(captured.Messages) < 2 || captured.Messages[0].Role != "system" || !strings.Contains(captured.Messages[0].Parts[0].Text, "Gateway context page") {
		t.Fatalf("url context not expanded: %#v", captured.Messages)
	}
	if len(captured.Tools) != 0 {
		t.Fatalf("urlContext tool should be removed after expansion: %#v", captured.Tools)
	}
}

func TestGeminiCachedContentCreateUsesNativeForwarderWhenAvailable(t *testing.T) {
	store := NewMemoryResponseSessionStore(16)
	router := NewRouter(Dependencies{
		VerifyAPIKey:     func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ResponseSessions: store,
		SelectDirectRoute: func(ctx context.Context, model string, provider string, scope *domain.APIKeyScope) (domain.GatewayRoute, error) {
			return domain.GatewayRoute{ModelAlias: model, Channel: domain.UpstreamChannel{Name: "gemini-a", Provider: "gemini"}}, nil
		},
		ForwardGeminiCachedContentCreate: func(ctx context.Context, route domain.GatewayRoute, body []byte) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}}, Body: []byte(`{"name":"cachedContents/native-1","model":"gemini-2.5-pro"}`)}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1beta/cachedContents", bytes.NewBufferString(`{"model":"gemini-2.5-pro","contents":[{"role":"user","parts":[{"text":"cached prompt"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"cachedContents/native-1"`) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := store.Get("cachedContents/native-1"); !ok {
		t.Fatalf("expected local mirror for native cache")
	}
}

func TestOpenAIRealtimeClientSecretsForwardsNativeResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		SelectDirectRoute: func(ctx context.Context, model string, provider string, scope *domain.APIKeyScope) (domain.GatewayRoute, error) {
			return domain.GatewayRoute{ModelAlias: model, Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}}, nil
		},
		ForwardOpenAIRealtimeClientSecret: func(ctx context.Context, route domain.GatewayRoute, body []byte) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}}, Body: []byte(`{"client_secret":{"value":"rt_123"}}`)}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/client_secrets", bytes.NewBufferString(`{"model":"gpt-realtime"}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"rt_123"`) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIRealtimeCallsForwardsNativeResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		SelectDirectRoute: func(ctx context.Context, model string, provider string, scope *domain.APIKeyScope) (domain.GatewayRoute, error) {
			return domain.GatewayRoute{ModelAlias: model, Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}}, nil
		},
		ForwardOpenAIRealtimeCall: func(ctx context.Context, route domain.GatewayRoute, contentType string, body []byte, rawQuery string) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/sdp"}}, Body: []byte("v=0")}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/realtime/calls?model=gpt-realtime", bytes.NewBufferString("offer-sdp"))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	req.Header.Set("Content-Type", "application/sdp")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "v=0" {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIRealtimeWebSocketUsesNativeProxyWhenAvailable(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade upstream: %v", err)
		}
		defer conn.Close()
		if err := conn.WriteJSON(map[string]any{"type": "session.created", "session": map[string]any{"id": "native"}}); err != nil {
			t.Fatalf("write upstream: %v", err)
		}
		for {
			var payload map[string]any
			if err := conn.ReadJSON(&payload); err != nil {
				return
			}
			if payload["type"] == "ping" {
				_ = conn.WriteJSON(map[string]any{"type": "pong"})
				return
			}
		}
	}))
	defer upstream.Close()

	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		SelectDirectRoute: func(ctx context.Context, model string, provider string, scope *domain.APIKeyScope) (domain.GatewayRoute, error) {
			return domain.GatewayRoute{ModelAlias: model, Channel: domain.UpstreamChannel{Name: "openai-a", Provider: "openai"}}, nil
		},
		DialOpenAIRealtime: func(ctx context.Context, route domain.GatewayRoute, req *http.Request) (*websocket.Conn, *http.Response, error) {
			target := "ws" + strings.TrimPrefix(upstream.URL, "http")
			return websocket.DefaultDialer.DialContext(ctx, target, nil)
		},
		ResponseSessions: NewMemoryResponseSessionStore(16),
		CopyProxy:        copyProxyForTest,
		CopyStream:       copyStreamForTest,
	})
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/realtime?model=gpt-realtime"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer sk-opencrab-test"}})
	if err != nil {
		t.Fatalf("dial realtime ws: %v", err)
	}
	defer conn.Close()

	var event map[string]any
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read native session event: %v", err)
	}
	if event["type"] != "session.created" {
		t.Fatalf("unexpected first event: %#v", event)
	}
	if err := conn.WriteJSON(map[string]any{"type": "ping"}); err != nil {
		t.Fatalf("write proxy message: %v", err)
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read proxied response: %v", err)
	}
	if event["type"] != "pong" {
		t.Fatalf("unexpected proxied event: %#v", event)
	}
}

func TestOpenAIResponsesAsyncAdmissionReturnsAccepted(t *testing.T) {
	var createdJob domain.GatewayJob
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		CreateGatewayJob: func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
			createdJob = item
			createdJob.ID = 1
			return createdJob, nil
		},
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) {
			return createdJob, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	req.Header.Set("Prefer", "respond-async")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected code: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"accepted"`) || !strings.Contains(rec.Body.String(), `"status_url":"/v1/requests/`) || !strings.Contains(rec.Body.String(), `"events_url":"/v1/requests/`) {
		t.Fatalf("unexpected accepted body: %s", rec.Body.String())
	}
	if createdJob.RequestPath != "/v1/responses" || createdJob.Mode != "async" {
		t.Fatalf("unexpected job: %#v", createdJob)
	}
}

func TestOpenAIResponsesAsyncAdmissionSyncBridgeReturnsCompletedBody(t *testing.T) {
	completed := domain.GatewayJob{RequestID: "req_bridge", OwnerKeyHash: gatewayOwnerKeyHash("sk-opencrab-test"), Status: domain.GatewayJobStatusCompleted, ResponseStatusCode: 200, ResponseBody: `{"id":"resp_bridge"}`, AcceptedAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey:             func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		CreateGatewayJob:         func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) { return completed, nil },
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) { return completed, nil },
		GetDispatchRuntimeSettings: func(ctx context.Context) (domain.DispatchRuntimeSettings, error) {
			return domain.DispatchRuntimeSettings{SyncHoldMs: 3000}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	req.Header.Set("Prefer", "respond-async, wait=2")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"id":"resp_bridge"}` {
		t.Fatalf("unexpected sync bridge response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIResponsesAsyncAdmissionIdempotencyReplay(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","input":[]}`)
	existing := domain.GatewayJob{RequestID: "req_existing", IdempotencyKey: "idem-1", OwnerKeyHash: gatewayOwnerKeyHash("sk-opencrab-test"), RequestHash: gatewayAdmissionRequestHash("/v1/responses", body), RequestPath: "/v1/responses", Status: domain.GatewayJobStatusAccepted, Mode: "async", AcceptedAt: time.Now().Format(time.RFC3339), EstimatedReadyAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		GetGatewayJobByIdempotencyKey: func(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error) {
			if ownerKeyHash == gatewayOwnerKeyHash("sk-opencrab-test") && idempotencyKey == "idem-1" {
				return existing, nil
			}
			return domain.GatewayJob{}, fmt.Errorf("请求不存在")
		},
		CreateGatewayJob: func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
			t.Fatal("should not create duplicate job")
			return domain.GatewayJob{}, nil
		},
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) { return existing, nil },
		CopyProxy:                copyProxyForTest,
		CopyStream:               copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	req.Header.Set("Prefer", "respond-async")
	req.Header.Set("Idempotency-Key", "idem-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"idempotent_replay":true`) {
		t.Fatalf("unexpected replay response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIResponsesAsyncAdmissionRejectsMismatchedReplay(t *testing.T) {
	existing := domain.GatewayJob{RequestID: "req_existing", IdempotencyKey: "idem-1", OwnerKeyHash: gatewayOwnerKeyHash("sk-opencrab-test"), RequestHash: gatewayAdmissionRequestHash("/v1/responses", []byte(`{"model":"gpt-5.4","input":[]}`)), RequestPath: "/v1/responses", Status: domain.GatewayJobStatusAccepted, Mode: "async", AcceptedAt: time.Now().Format(time.RFC3339), EstimatedReadyAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		GetGatewayJobByIdempotencyKey: func(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error) {
			return existing, nil
		},
		CreateGatewayJob: func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
			t.Fatal("should not create conflicting replay")
			return domain.GatewayJob{}, nil
		},
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) { return existing, nil },
		CopyProxy:                copyProxyForTest,
		CopyStream:               copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"different"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	req.Header.Set("Prefer", "respond-async")
	req.Header.Set("Idempotency-Key", "idem-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "Idempotency-Key 已被不同请求占用") {
		t.Fatalf("unexpected conflict response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayRequestStatusReturnsStoredJob(t *testing.T) {
	existing := domain.GatewayJob{RequestID: "req_status", OwnerKeyHash: gatewayOwnerKeyHash("sk-opencrab-test"), Status: domain.GatewayJobStatusCompleted, Mode: "async", DeliveryMode: "poll", ResponseStatusCode: 200, ResponseBody: `{"id":"resp_1"}`, AttemptCount: 1, AcceptedAt: time.Now().Format(time.RFC3339), CompletedAt: time.Now().Format(time.RFC3339), UpdatedAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) {
			if requestID == "req_status" {
				return existing, nil
			}
			return domain.GatewayJob{}, fmt.Errorf("请求不存在")
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/requests/req_status", nil)
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"request_id":"req_status"`) || !strings.Contains(rec.Body.String(), `"result":{"id":"resp_1"}`) {
		t.Fatalf("unexpected status response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayRequestEventsReturnsCompletedSSE(t *testing.T) {
	existing := domain.GatewayJob{RequestID: "req_events", OwnerKeyHash: gatewayOwnerKeyHash("sk-opencrab-test"), Status: domain.GatewayJobStatusCompleted, ResponseBody: `{"id":"resp_1"}`, AcceptedAt: time.Now().Format(time.RFC3339), CompletedAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey:             func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) { return existing, nil },
		CopyProxy:                copyProxyForTest,
		CopyStream:               copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/requests/req_events/events", nil)
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "event: response.created") || !strings.Contains(rec.Body.String(), "event: response.completed") || !strings.Contains(rec.Body.String(), "data: [DONE]") {
		t.Fatalf("unexpected events response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayRequestStatusRejectsDifferentOwner(t *testing.T) {
	existing := domain.GatewayJob{RequestID: "req_status", OwnerKeyHash: gatewayOwnerKeyHash("another-key"), Status: domain.GatewayJobStatusAccepted, Mode: "async", AcceptedAt: time.Now().Format(time.RFC3339), UpdatedAt: time.Now().Format(time.RFC3339)}
	router := NewRouter(Dependencies{
		VerifyAPIKey:             func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) { return existing, nil },
		CopyProxy:                copyProxyForTest,
		CopyStream:               copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/requests/req_status", nil)
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProxyChatCompletionsCopiesStreamResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"text/event-stream"}, "X-Opencrab-Provider": {"openai"}}, Body: io.NopCloser(strings.NewReader("data: {\"id\":\"chunk-1\"}\n\ndata: [DONE]\n\n"))}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4.1","stream":true,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "data: {\"id\":\"chunk-1\"}\n\ndata: [DONE]\n\n" {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyClaudeMessagesAcceptsNativeHeader(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("unexpected api key: %s", rawKey)
			}
			return true, nil
		},
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"claude"}}, Body: []byte(`{"id":"msg_1","type":"message"}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("x-api-key", "sk-opencrab-test")
	req.Header.Set("anthropic-version", "2023-06-01")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected code: %d", rec.Code)
	}
}

func TestProxyClaudeMessagesSynthesizesClaudeStreamFromOpenAIResponse(t *testing.T) {
	var logged domain.RequestLog
	logger, records := newCaptureLogger()
	router := NewRouter(Dependencies{
		Logger: logger,
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)}, Metadata: &domain.GatewayExecutionMetadata{DegradedSuccess: false, AttemptCount: 1, SelectedChannel: "openai-upstream"}}, nil
		},
		CreateRequestLog: func(ctx context.Context, item domain.RequestLog) error {
			logged = item
			return nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"stream":true,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("x-api-key", "***")
	req.Header.Set("anthropic-version", "2023-06-01")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected code: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: message_start") || !strings.Contains(body, "event: content_block_delta") || !strings.Contains(body, "event: message_stop") {
		t.Fatalf("unexpected body: %s", body)
	}
	if !strings.Contains(body, `"output_tokens":2`) {
		t.Fatalf("expected usage tokens in body: %s", body)
	}
	if logged.TotalTokens != 3 || logged.PromptTokens != 1 || logged.CompletionTokens != 2 {
		t.Fatalf("expected persisted usage tokens, got %+v", logged)
	}
	if !strings.Contains(logged.Details, `"degraded_success":false`) {
		t.Fatalf("expected degraded_success field in gateway log details, got %s", logged.Details)
	}
	if !captureLogsContain(records, "decode_and_preprocess_duration") || !captureLogsContain(records, "write_response_duration") {
		t.Fatalf("expected request logger timing fields, got %#v", *records)
	}
}

func TestExtractUsageMetricsFromClaudeSSE(t *testing.T) {
	body := []byte("event: message_start\ndata: {\"message\":{\"usage\":{\"input_tokens\":123,\"output_tokens\":0}}}\n\n" +
		"event: message_delta\ndata: {\"usage\":{\"output_tokens\":45}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	usage := extractUsageMetrics(body)
	if usage.PromptTokens != 123 || usage.CompletionTokens != 45 || usage.TotalTokens != 168 {
		t.Fatalf("unexpected sse usage: %+v", usage)
	}
}

func TestProxyGeminiStreamSynthesizesGeminiStreamFromClaudeResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"claude"}},
				Body:       []byte(`{"id":"msg_1","model":"claude-sonnet","role":"assistant","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn"}`),
			}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:streamGenerateContent", bytes.NewBufferString(`{"contents":[{"role":"user","parts":[{"text":"ping"}]}]}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"candidates"`) || !strings.Contains(rec.Body.String(), `data:`) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyClaudeCountTokensAcceptsNativeHeader(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("unexpected api key: %s", rawKey)
			}
			return true, nil
		},
		CountClaudeTokens: func(ctx context.Context, req *http.Request, body []byte) (*domain.ProxyResponse, error) {
			if req.Header.Get("anthropic-version") != "2023-06-01" {
				t.Fatalf("unexpected anthropic-version: %s", req.Header.Get("anthropic-version"))
			}
			return &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}}, Body: []byte(`{"input_tokens":12}`)}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("x-api-key", "sk-opencrab-test")
	req.Header.Set("anthropic-version", "2023-06-01")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"input_tokens":12}` {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyGeminiStreamAcceptsQueryKey(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("unexpected api key: %s", rawKey)
			}
			return true, nil
		},
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"text/event-stream"}, "X-Opencrab-Provider": {"gemini"}}, Body: io.NopCloser(strings.NewReader("data: {}\n\n"))}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse&key=sk-opencrab-test", bytes.NewBufferString(`{"contents":[{"parts":[{"text":"ping"}]}]}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "data: {}\n\n" {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyRejectsRequestWithoutAnySupportedAPIKeyHeader(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			t.Fatal("gateway should not be called without api key")
			return nil, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"ping"}]}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !bytes.Contains(rec.Body.Bytes(), []byte("缺少 API Key")) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func copyProxyForTest(w http.ResponseWriter, resp *domain.ProxyResponse) error {
	for key, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err := w.Write(resp.Body)
	return err
}

func copyStreamForTest(w http.ResponseWriter, stream *domain.StreamResult) error {
	for key, values := range stream.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(stream.StatusCode)
	_, err := io.Copy(w, stream.Body)
	return err
}

type captureHandler struct {
	mu      sync.Mutex
	records []string
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	parts := []string{record.Message}
	record.Attrs(func(attr slog.Attr) bool {
		parts = append(parts, attr.Key+"="+attr.Value.String())
		return true
	})
	h.records = append(h.records, strings.Join(parts, " "))
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func newCaptureLogger() (*slog.Logger, *[]string) {
	handler := &captureHandler{}
	return slog.New(handler), &handler.records
}

func captureLogsContain(records *[]string, needle string) bool {
	for _, record := range *records {
		if strings.Contains(record, needle) {
			return true
		}
	}
	return false
}
