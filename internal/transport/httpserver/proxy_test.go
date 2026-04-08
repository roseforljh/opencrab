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
