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

func TestResponsesProxyJSON(t *testing.T) {
	var receivedAuthorization string
	var receivedBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
		receivedAuthorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_test","object":"response","status":"completed","model":"gpt-4.1","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"pong"}]}],"usage":{"input_tokens":3,"output_tokens":1,"total_tokens":4}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-responses", upstream.URL+"/v1", "resp-key", "gpt-4.1")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/responses", apiAddr), `{"model":"gpt-4.1","input":"ping"}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected responses endpoint to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if receivedAuthorization != "Bearer resp-key" {
		t.Fatalf("expected bearer auth, got %q", receivedAuthorization)
	}
	if receivedBody["model"] != "gpt-4.1" {
		t.Fatalf("expected model passthrough, got %#v", receivedBody["model"])
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read responses body: %v", err)
	}
	if !strings.Contains(string(body), `"object":"response"`) || !strings.Contains(string(body), `"text":"pong"`) {
		t.Fatalf("expected response payload passthrough, got %s", string(body))
	}
}

func TestResponsesStreamPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_stream\",\"model\":\"gpt-4.1\",\"status\":\"in_progress\"}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"delta\":\"Hel\"}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_stream\",\"model\":\"gpt-4.1\",\"status\":\"completed\",\"usage\":{\"input_tokens\":3,\"output_tokens\":1}}}\n\n"))
		flusher.Flush()
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-responses", upstream.URL+"/v1", "resp-key", "gpt-4.1")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/responses", apiAddr), `{"model":"gpt-4.1","stream":true,"input":"ping"}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected responses stream endpoint to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "text/event-stream")
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `response.output_text.delta`) || !strings.Contains(text, `response.completed`) {
		t.Fatalf("expected raw responses stream passthrough, got %s", text)
	}
}
