package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"object":"response"`) || !strings.Contains(rec.Body.String(), `"output_text":"pong"`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestProxyResponsesReturnsSyntheticStream(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			if req.Stream {
				t.Fatal("responses handler should use non-stream upstream execution")
			}
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
	for i := 0; i < 8; i++ {
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
	for i := 0; i < 8; i++ {
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
	for i := 0; i < 8; i++ {
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
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) { return true, nil },
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: http.StatusOK, Headers: map[string][]string{"Content-Type": {"application/json"}, "X-Opencrab-Provider": {"openai"}}, Body: []byte(`{"id":"chatcmpl-test","model":"gpt-4.1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)}}, nil
		},
		CopyProxy:  copyProxyForTest,
		CopyStream: copyStreamForTest,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"stream":true,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("x-api-key", "sk-opencrab-test")
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
