package httpserver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opencrab/internal/domain"
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
