package httpserver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyChatCompletionsCopiesResponse(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			return true, nil
		},
		ProxyChat: func(ctx context.Context, body []byte) (*http.Response, error) {
			upstream := httptest.NewRecorder()
			upstream.Header().Set("Content-Type", "application/json")
			upstream.WriteHeader(http.StatusOK)
			_, _ = upstream.WriteString(`{"id":"chatcmpl-test"}`)
			return upstream.Result(), nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *http.Response) error {
			defer resp.Body.Close()
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := io.Copy(w, resp.Body)
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
		ProxyChat: func(ctx context.Context, body []byte) (*http.Response, error) {
			upstream := httptest.NewRecorder()
			upstream.Header().Set("Content-Type", "text/event-stream")
			upstream.WriteHeader(http.StatusOK)
			_, _ = upstream.WriteString("data: {\"id\":\"chunk-1\"}\n\n")
			_, _ = upstream.WriteString("data: [DONE]\n\n")
			return upstream.Result(), nil
		},
		CopyProxy: func(w http.ResponseWriter, resp *http.Response) error {
			defer resp.Body.Close()
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err := io.Copy(w, resp.Body)
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
