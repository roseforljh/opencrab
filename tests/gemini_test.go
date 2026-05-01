package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiGenerateContentProxyJSON(t *testing.T) {
	var receivedPath string
	var receivedAPIKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAPIKey = r.Header.Get("X-Goog-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"pong"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1,"totalTokenCount":4}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createCompatChannel(t, apiAddr, map[string]any{"name": "gemini-main", "provider": "Gemini", "endpoint": upstream.URL + "/v1beta", "api_key": "gemini-key", "enabled": true, "model_ids": []string{"gemini-2.5-pro"}, "rpm_limit": 1000, "max_inflight": 32, "safety_factor": 0.9, "enabled_for_async": true, "dispatch_weight": 100})

	payload := `{"contents":[{"parts":[{"text":"ping"}]}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1beta/models/gemini-2.5-pro:generateContent", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected generateContent to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "application/json")
	if receivedPath != "/v1beta/models/gemini-2.5-pro:generateContent" {
		t.Fatalf("expected upstream path to be generateContent, got %q", receivedPath)
	}
	if receivedAPIKey != "gemini-key" {
		t.Fatalf("expected upstream API key to use channel key, got %q", receivedAPIKey)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read gemini response body: %v", err)
	}
	if !strings.Contains(string(body), `"text":"pong"`) {
		t.Fatalf("expected generated text in body, got %s", string(body))
	}
}

func TestGeminiStreamGenerateContentPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hel\"}]}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"lo\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2,\"totalTokenCount\":5}}\n\n"))
		flusher.Flush()
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createCompatChannel(t, apiAddr, map[string]any{"name": "gemini-stream", "provider": "Gemini", "endpoint": upstream.URL + "/v1beta", "api_key": "gemini-stream-key", "enabled": true, "model_ids": []string{"gemini-2.5-flash"}, "rpm_limit": 1000, "max_inflight": 32, "safety_factor": 0.9, "enabled_for_async": true, "dispatch_weight": 100})

	payload := `{"contents":[{"parts":[{"text":"ping"}]}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse", apiAddr), payload, map[string]string{"Content-Type": "application/json", "Accept": "text/event-stream"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected streamGenerateContent to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "text/event-stream")
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read gemini stream body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"text":"Hel"`) || !strings.Contains(text, `"text":"lo"`) {
		t.Fatalf("expected SSE chunks in body, got %s", text)
	}
}

func TestGeminiRouteMissingIsStable(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := doPOST(t, fmt.Sprintf("http://%s/v1beta/models/gemini-2.5-pro:generateContent", apiAddr), `{"contents":[{"parts":[{"text":"ping"}]}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected missing route to return 502, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing gemini route body: %v", err)
	}
	if !strings.Contains(string(body), `No enabled gemini route configured for model gemini-2.5-pro`) {
		t.Fatalf("expected stable route error, got %s", string(body))
	}
}
