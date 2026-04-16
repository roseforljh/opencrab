package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"opencrab/internal/domain"
)

func TestProxyChatCompletionsCopiesResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			return true, nil
		},
		ProxyChat: func(ctx context.Context, body []byte) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}},
				Body:       []byte(`{"id":"chatcmpl-test"}`),
			}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write(resp.Body)
			return err
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4.1"}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if rec.Body.String() != `{"id":"chatcmpl-test"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestProxyChatCompletionsCopiesStreamResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			return true, nil
		},
		ProxyChat: func(ctx context.Context, body []byte) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"text/event-stream"}},
				Body:       []byte("data: {\"id\":\"chunk-1\"}\n\ndata: [DONE]\n\n"),
			}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write(resp.Body)
			return err
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4.1","stream":true}`))
	req.Header.Set("Authorization", "Bearer sk-opencrab-test")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	expected := "data: {\"id\":\"chunk-1\"}\n\ndata: [DONE]\n\n"
	if rec.Body.String() != expected {
		t.Fatalf("unexpected stream body: %s", rec.Body.String())
	}
}

func TestProxyClaudeMessagesCopiesResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("expected claude api key, got %s", rawKey)
			}
			return true, nil
		},
		ProxyClaude: func(ctx context.Context, body []byte) (*domain.ProxyResponse, error) {
			return &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}},
				Body:       []byte(`{"id":"msg_123","type":"message"}`),
			}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write(resp.Body)
			return err
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("x-api-key", "sk-opencrab-test")
	req.Header.Set("anthropic-version", "2023-06-01")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"id":"msg_123","type":"message"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestProxyGeminiGenerateContentCopiesResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("expected gemini api key, got %s", rawKey)
			}
			return true, nil
		},
		ProxyGemini: func(ctx context.Context, model string, body []byte, stream bool) (*domain.ProxyResponse, error) {
			if model != "gemini-2.0-flash" {
				t.Fatalf("expected model gemini-2.0-flash, got %s", model)
			}
			if stream {
				t.Fatal("expected non-stream request")
			}
			return &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"application/json"}},
				Body:       []byte(`{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}`),
			}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write(resp.Body)
			return err
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:generateContent", bytes.NewBufferString(`{"contents":[{"parts":[{"text":"ping"}]}]}`))
	req.Header.Set("x-goog-api-key", "sk-opencrab-test")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestProxyGeminiStreamGenerateContentCopiesStreamResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			if rawKey != "sk-opencrab-test" {
				t.Fatalf("expected gemini api key, got %s", rawKey)
			}
			return true, nil
		},
		ProxyGemini: func(ctx context.Context, model string, body []byte, stream bool) (*domain.ProxyResponse, error) {
			if model != "gemini-2.0-flash" {
				t.Fatalf("expected model gemini-2.0-flash, got %s", model)
			}
			if !stream {
				t.Fatal("expected stream request")
			}
			return &domain.ProxyResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"text/event-stream"}},
				Body:       []byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"pong\"}]}}]}\n\n"),
			}, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			for key, values := range resp.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := w.Write(resp.Body)
			return err
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:streamGenerateContent?alt=sse&key=sk-opencrab-test", bytes.NewBufferString(`{"contents":[{"parts":[{"text":"ping"}]}]}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestProxyRejectsRequestWithoutAnySupportedAPIKeyHeader(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			return true, nil
		},
		ProxyClaude: func(ctx context.Context, body []byte) (*domain.ProxyResponse, error) {
			t.Fatal("proxy should not be called without api key")
			return nil, nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *domain.ProxyResponse) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"ping"}]}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("缺少 API Key")) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
