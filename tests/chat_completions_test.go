package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHealthzEndpoint(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := waitForGET(t, fmt.Sprintf("http://%s/healthz", apiAddr), 10*time.Second)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected /healthz to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "application/json")
}

func TestAdminCompatibilityReadEndpoints(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	testCases := []struct {
		path            string
		statusCode      int
		bodyMustContain string
	}{
		{path: "/api/admin/auth/status", statusCode: http.StatusOK, bodyMustContain: `"authenticated":true`},
		{path: "/api/admin/auth/security", statusCode: http.StatusOK, bodyMustContain: `"configured":false`},
		{path: "/api/admin/dashboard/summary", statusCode: http.StatusOK, bodyMustContain: `"routing_overview"`},
		{path: "/api/admin/channels", statusCode: http.StatusOK, bodyMustContain: `"items":[]`},
		{path: "/api/admin/models", statusCode: http.StatusOK, bodyMustContain: `"items":[]`},
		{path: "/api/admin/model-routes", statusCode: http.StatusOK, bodyMustContain: `"items":[]`},
		{path: "/api/admin/api-keys", statusCode: http.StatusOK, bodyMustContain: `"items":[]`},
		{path: "/api/admin/settings", statusCode: http.StatusOK, bodyMustContain: `"items":[]`},
		{path: "/api/admin/logs", statusCode: http.StatusOK, bodyMustContain: `"filtered":0`},
	}

	for _, testCase := range testCases {
		response := waitForGET(t, fmt.Sprintf("http://%s%s", apiAddr, testCase.path), 10*time.Second)
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatalf("read body for %s: %v", testCase.path, err)
		}
		if response.StatusCode != testCase.statusCode {
			t.Fatalf("expected %s to return %d, got %d: %s", testCase.path, testCase.statusCode, response.StatusCode, string(body))
		}
		if !strings.Contains(string(body), testCase.bodyMustContain) {
			t.Fatalf("expected %s body to contain %q, got %s", testCase.path, testCase.bodyMustContain, string(body))
		}
	}
}

func TestAdminChannelsCreateAndList(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createPayload := `{"name":"openai-main","provider":"OpenAI","endpoint":"https://api.openai.com/v1","api_key":"sk-test","enabled":true,"model_ids":["gpt-4.1","gpt-4o-mini"],"rpm_limit":1000,"max_inflight":32,"safety_factor":0.9,"enabled_for_async":true,"dispatch_weight":100}`
	createResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), createPayload, map[string]string{"Content-Type": "application/json"})
	defer createResponse.Body.Close()

	if createResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResponse.Body)
		t.Fatalf("expected create channel to return 201, got %d: %s", createResponse.StatusCode, string(body))
	}

	channelsResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), 10*time.Second)
	defer channelsResponse.Body.Close()
	channelsBody, err := io.ReadAll(channelsResponse.Body)
	if err != nil {
		t.Fatalf("read channels body: %v", err)
	}
	if channelsResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected channels list to return 200, got %d: %s", channelsResponse.StatusCode, string(channelsBody))
	}
	if !strings.Contains(string(channelsBody), `"name":"openai-main"`) {
		t.Fatalf("expected created channel in list, got %s", string(channelsBody))
	}

	modelsResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/models", apiAddr), 10*time.Second)
	defer modelsResponse.Body.Close()
	modelsBody, err := io.ReadAll(modelsResponse.Body)
	if err != nil {
		t.Fatalf("read models body: %v", err)
	}
	if !strings.Contains(string(modelsBody), `"alias":"gpt-4.1"`) || !strings.Contains(string(modelsBody), `"alias":"gpt-4o-mini"`) {
		t.Fatalf("expected created model aliases in models list, got %s", string(modelsBody))
	}

	routesResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/model-routes", apiAddr), 10*time.Second)
	defer routesResponse.Body.Close()
	routesBody, err := io.ReadAll(routesResponse.Body)
	if err != nil {
		t.Fatalf("read model routes body: %v", err)
	}
	if !strings.Contains(string(routesBody), `"channel_name":"openai-main"`) {
		t.Fatalf("expected created routes in route list, got %s", string(routesBody))
	}
}

func TestAdminModelsDelete(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createPayload := `{"name":"openai-main","provider":"OpenAI","endpoint":"https://api.openai.com/v1","api_key":"sk-test","enabled":true,"model_ids":["gpt-4.1","gpt-4o-mini"],"rpm_limit":1000,"max_inflight":32,"safety_factor":0.9,"enabled_for_async":true,"dispatch_weight":100}`
	createResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), createPayload, map[string]string{"Content-Type": "application/json"})
	createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResponse.Body)
		t.Fatalf("expected create channel 201, got %d: %s", createResponse.StatusCode, string(body))
	}

	listResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/models", apiAddr), 10*time.Second)
	listBody, err := io.ReadAll(listResponse.Body)
	listResponse.Body.Close()
	if err != nil {
		t.Fatalf("read models list body: %v", err)
	}
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected models list 200, got %d: %s", listResponse.StatusCode, string(listBody))
	}
	if !strings.Contains(string(listBody), `"alias":"gpt-4.1"`) {
		t.Fatalf("expected model alias in list, got %s", string(listBody))
	}

	deleteResponse := doRequest(t, http.MethodDelete, fmt.Sprintf("http://%s/api/admin/models/1", apiAddr), "", nil)
	deleteBody, err := io.ReadAll(deleteResponse.Body)
	deleteResponse.Body.Close()
	if err != nil {
		t.Fatalf("read model delete body: %v", err)
	}
	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("expected model delete 204, got %d: %s", deleteResponse.StatusCode, string(deleteBody))
	}

	finalModelsResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/models", apiAddr), 10*time.Second)
	finalModelsBody, err := io.ReadAll(finalModelsResponse.Body)
	finalModelsResponse.Body.Close()
	if err != nil {
		t.Fatalf("read final models body: %v", err)
	}
	if strings.Contains(string(finalModelsBody), `"alias":"gpt-4.1"`) {
		t.Fatalf("expected deleted model alias to disappear, got %s", string(finalModelsBody))
	}
	if !strings.Contains(string(finalModelsBody), `"alias":"gpt-4o-mini"`) {
		t.Fatalf("expected remaining model alias to stay, got %s", string(finalModelsBody))
	}

	routesResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/model-routes", apiAddr), 10*time.Second)
	routesBody, err := io.ReadAll(routesResponse.Body)
	routesResponse.Body.Close()
	if err != nil {
		t.Fatalf("read routes body: %v", err)
	}
	if strings.Contains(string(routesBody), `"model_alias":"gpt-4.1"`) {
		t.Fatalf("expected deleted model route to disappear, got %s", string(routesBody))
	}
}

func TestAdminChannelTestUsesOpenAICompatibleUpstream(t *testing.T) {
	var receivedAuthorization string
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthorization = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createPayload := fmt.Sprintf(`{"name":"openai-main","provider":"OpenAI","endpoint":%q,"api_key":"sk-test","enabled":true,"model_ids":["gpt-4.1"],"rpm_limit":1000,"max_inflight":32,"safety_factor":0.9,"enabled_for_async":true,"dispatch_weight":100}`, upstream.URL+"/v1")
	createResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), createPayload, map[string]string{"Content-Type": "application/json"})
	createBody, _ := io.ReadAll(createResponse.Body)
	createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected create channel 201, got %d: %s", createResponse.StatusCode, string(createBody))
	}

	testResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels/1/test", apiAddr), `{"model":"gpt-4.1"}`, map[string]string{"Content-Type": "application/json"})
	defer testResponse.Body.Close()
	if testResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(testResponse.Body)
		t.Fatalf("expected channel test 200, got %d: %s", testResponse.StatusCode, string(body))
	}
	if receivedPath != "/v1/chat/completions" {
		t.Fatalf("expected openai-compatible probe to hit /v1/chat/completions, got %q", receivedPath)
	}
	if receivedAuthorization != "Bearer sk-test" {
		t.Fatalf("expected bearer auth on probe, got %q", receivedAuthorization)
	}
}

func TestAdminChannelTestUsesClaudeUpstream(t *testing.T) {
	var receivedAPIKey string
	var receivedVersion string
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedVersion = r.Header.Get("Anthropic-Version")
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","content":[{"type":"text","text":"pong"}],"model":"claude-sonnet-4-5","stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createPayload := fmt.Sprintf(`{"name":"claude-main","provider":"Claude","endpoint":%q,"api_key":"claude-secret","enabled":true,"model_ids":["claude-sonnet-4-5"],"rpm_limit":1000,"max_inflight":32,"safety_factor":0.9,"enabled_for_async":true,"dispatch_weight":100}`, upstream.URL)
	createResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), createPayload, map[string]string{"Content-Type": "application/json"})
	createBody, _ := io.ReadAll(createResponse.Body)
	createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected create channel 201, got %d: %s", createResponse.StatusCode, string(createBody))
	}

	testResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels/1/test", apiAddr), `{"model":"claude-sonnet-4-5"}`, map[string]string{"Content-Type": "application/json"})
	defer testResponse.Body.Close()
	if testResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(testResponse.Body)
		t.Fatalf("expected claude channel test 200, got %d: %s", testResponse.StatusCode, string(body))
	}
	if receivedPath != "/v1/messages" {
		t.Fatalf("expected claude probe to hit /v1/messages, got %q", receivedPath)
	}
	if receivedAPIKey != "claude-secret" {
		t.Fatalf("expected claude api key on probe, got %q", receivedAPIKey)
	}
	if receivedVersion != "2023-06-01" {
		t.Fatalf("expected anthropic version header, got %q", receivedVersion)
	}
}

func TestAdminAPIKeysCreateListUpdateDelete(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	createResponse := doPOST(t, fmt.Sprintf("http://%s/api/admin/api-keys", apiAddr), `{"name":"gateway-key","enabled":true,"channel_names":["openai-main"],"model_aliases":["gpt-4.1"]}`, map[string]string{"Content-Type": "application/json"})
	createBody, err := io.ReadAll(createResponse.Body)
	createResponse.Body.Close()
	if err != nil {
		t.Fatalf("read create api key body: %v", err)
	}
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected api key create 201, got %d: %s", createResponse.StatusCode, string(createBody))
	}
	if !strings.Contains(string(createBody), `"raw_key":"oc_`) {
		t.Fatalf("expected created api key raw_key, got %s", string(createBody))
	}

	listResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/api-keys", apiAddr), 10*time.Second)
	listBody, err := io.ReadAll(listResponse.Body)
	listResponse.Body.Close()
	if err != nil {
		t.Fatalf("read api key list body: %v", err)
	}
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected api key list 200, got %d: %s", listResponse.StatusCode, string(listBody))
	}
	if !strings.Contains(string(listBody), `"name":"gateway-key"`) {
		t.Fatalf("expected created api key in list, got %s", string(listBody))
	}

	updateResponse := doRequest(t, http.MethodPut, fmt.Sprintf("http://%s/api/admin/api-keys/1", apiAddr), `{"enabled":false}`, map[string]string{"Content-Type": "application/json"})
	updateBody, err := io.ReadAll(updateResponse.Body)
	updateResponse.Body.Close()
	if err != nil {
		t.Fatalf("read api key update body: %v", err)
	}
	if updateResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected api key update 200, got %d: %s", updateResponse.StatusCode, string(updateBody))
	}
	if !strings.Contains(string(updateBody), `"enabled":false`) {
		t.Fatalf("expected disabled api key after update, got %s", string(updateBody))
	}

	deleteResponse := doRequest(t, http.MethodDelete, fmt.Sprintf("http://%s/api/admin/api-keys/1", apiAddr), "", nil)
	deleteBody, err := io.ReadAll(deleteResponse.Body)
	deleteResponse.Body.Close()
	if err != nil {
		t.Fatalf("read api key delete body: %v", err)
	}
	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("expected api key delete 204, got %d: %s", deleteResponse.StatusCode, string(deleteBody))
	}

	finalListResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/api-keys", apiAddr), 10*time.Second)
	finalListBody, err := io.ReadAll(finalListResponse.Body)
	finalListResponse.Body.Close()
	if err != nil {
		t.Fatalf("read final api key list body: %v", err)
	}
	if strings.Contains(string(finalListBody), `"name":"gateway-key"`) {
		t.Fatalf("expected deleted api key to disappear, got %s", string(finalListBody))
	}
}

func TestRequestLogsCaptureAndPersist(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "opencrab-state.json")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{
		"OPENCRAB_DB_PATH": statePath,
	})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "logs-channel", upstream.URL+"/v1", "test-upstream-key", "gpt-4o-mini")

	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	_, _ = io.ReadAll(response.Body)
	response.Body.Close()

	logsResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/logs", apiAddr), 10*time.Second)
	logsBody, err := io.ReadAll(logsResponse.Body)
	logsResponse.Body.Close()
	if err != nil {
		t.Fatalf("read logs body: %v", err)
	}
	if !strings.Contains(string(logsBody), `/v1/chat/completions`) {
		t.Fatalf("expected request log to include chat path, got %s", string(logsBody))
	}
	if !strings.Contains(string(logsBody), `"total_tokens":4`) {
		t.Fatalf("expected usage in request logs, got %s", string(logsBody))
	}

	detailResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/logs/1", apiAddr), 10*time.Second)
	detailBody, err := io.ReadAll(detailResponse.Body)
	detailResponse.Body.Close()
	if err != nil {
		t.Fatalf("read log detail body: %v", err)
	}
	if !strings.Contains(string(detailBody), `"request_body":"{\"model\":`) {
		t.Fatalf("expected request body in log detail, got %s", string(detailBody))
	}

	stopProcess(t, cmd)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	cmd2 := startAPIServer(t, ctx2, apiAddr, map[string]string{"OPENCRAB_DB_PATH": statePath})
	defer stopProcess(t, cmd2)

	restartedLogs := waitForGET(t, fmt.Sprintf("http://%s/api/admin/logs", apiAddr), 10*time.Second)
	restartedLogsBody, err := io.ReadAll(restartedLogs.Body)
	restartedLogs.Body.Close()
	if err != nil {
		t.Fatalf("read restarted logs body: %v", err)
	}
	if !strings.Contains(string(restartedLogsBody), `/v1/chat/completions`) {
		t.Fatalf("expected logs to persist across restart, got %s", string(restartedLogsBody))
	}

	clearResponse := doRequest(t, http.MethodDelete, fmt.Sprintf("http://%s/api/admin/logs", apiAddr), "", nil)
	clearResponse.Body.Close()
	if clearResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("expected clear logs 204, got %d", clearResponse.StatusCode)
	}
}

func TestCompatStatePersistsAcrossRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "opencrab-state.json")
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{"OPENCRAB_DB_PATH": statePath})
	defer stopProcess(t, cmd)

	createChannel := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), `{"name":"persisted-channel","provider":"OpenAI","endpoint":"https://api.openai.com/v1","api_key":"sk-test","enabled":true,"model_ids":["gpt-4.1"],"rpm_limit":1000,"max_inflight":32,"safety_factor":0.9,"enabled_for_async":true,"dispatch_weight":100}`, map[string]string{"Content-Type": "application/json"})
	createChannel.Body.Close()
	createKey := doPOST(t, fmt.Sprintf("http://%s/api/admin/api-keys", apiAddr), `{"name":"persisted-key","enabled":true,"channel_names":["persisted-channel"],"model_aliases":["gpt-4.1"]}`, map[string]string{"Content-Type": "application/json"})
	createKey.Body.Close()

	stopProcess(t, cmd)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	cmd2 := startAPIServer(t, ctx2, apiAddr, map[string]string{"OPENCRAB_DB_PATH": statePath})
	defer stopProcess(t, cmd2)

	channelsResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), 10*time.Second)
	channelsBody, _ := io.ReadAll(channelsResponse.Body)
	channelsResponse.Body.Close()
	if !strings.Contains(string(channelsBody), `"name":"persisted-channel"`) {
		t.Fatalf("expected persisted channel after restart, got %s", string(channelsBody))
	}

	apiKeysResponse := waitForGET(t, fmt.Sprintf("http://%s/api/admin/api-keys", apiAddr), 10*time.Second)
	apiKeysBody, _ := io.ReadAll(apiKeysResponse.Body)
	apiKeysResponse.Body.Close()
	if !strings.Contains(string(apiKeysBody), `"name":"persisted-key"`) {
		t.Fatalf("expected persisted api key after restart, got %s", string(apiKeysBody))
	}
}

func TestChatCompletionsProxyJSON(t *testing.T) {
	var receivedAuthorization string
	var receivedOrganization string
	var receivedProject string
	var receivedBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		receivedAuthorization = r.Header.Get("Authorization")
		receivedOrganization = r.Header.Get("OpenAI-Organization")
		receivedProject = r.Header.Get("OpenAI-Project")
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("OpenAI-Model", "gpt-4o-mini")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-json", upstream.URL+"/v1", "test-upstream-key", "gpt-4o-mini")

	payload := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{
		"Content-Type":        "application/json",
		"OpenAI-Organization": "org-test",
		"OpenAI-Project":      "proj-test",
	})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected chat completions to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "application/json")

	if receivedAuthorization != "Bearer test-upstream-key" {
		t.Fatalf("expected upstream auth header to use configured key, got %q", receivedAuthorization)
	}
	if receivedOrganization != "org-test" {
		t.Fatalf("expected organization header to be forwarded, got %q", receivedOrganization)
	}
	if receivedProject != "proj-test" {
		t.Fatalf("expected project header to be forwarded, got %q", receivedProject)
	}
	if receivedBody["model"] != "gpt-4o-mini" {
		t.Fatalf("expected upstream to receive model, got %#v", receivedBody["model"])
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(string(body), `"object":"chat.completion"`) {
		t.Fatalf("expected completion response body, got %s", string(body))
	}
	if !strings.Contains(string(body), `"content":"pong"`) {
		t.Fatalf("expected assistant message in body, got %s", string(body))
	}
}

func TestChatCompletionsStreamPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_test\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hel\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-stream", upstream.URL+"/v1", "stream-key", "gpt-4o-mini")

	payload := `{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected streaming request to return 200, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "text/event-stream")

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"chat.completion.chunk"`) {
		t.Fatalf("expected chunk payload, got %s", text)
	}
	if !strings.Contains(text, `[DONE]`) {
		t.Fatalf("expected done sentinel, got %s", text)
	}
}

func TestChatCompletionsValidationError(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), `{"messages":[]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected validation error 400, got %d: %s", response.StatusCode, string(body))
	}
	assertContentTypeContains(t, response, "application/json")

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read validation body: %v", err)
	}
	if !strings.Contains(string(body), `"invalid_request_error"`) {
		t.Fatalf("expected OpenAI style validation error, got %s", string(body))
	}
}

func TestChatCompletionsAllowsAssistantToolCallHistory(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_tool","object":"chat.completion","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`))
	}))
	defer upstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-tools", upstream.URL+"/v1", "tool-key", "gpt-4o-mini")

	payload := `{"model":"gpt-4o-mini","messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"done"},{"role":"user","content":"continue"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected assistant tool call history to pass through, got %d: %s", response.StatusCode, string(body))
	}
}

func TestChatCompletionsStreamNotCutByTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl_test\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hel\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		time.Sleep(400 * time.Millisecond)
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
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
	createOpenAIChannel(t, apiAddr, "openai-timeout", upstream.URL+"/v1", "stream-key", "gpt-4o-mini")

	payload := `{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected streaming request to return 200, got %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	if !strings.Contains(string(body), `[DONE]`) {
		t.Fatalf("expected stream to survive timeout window, got %s", string(body))
	}
}

func TestChatCompletionsTransportErrorIsStable(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{
		"OPENCRAB_UPSTREAM_TIMEOUT": "300ms",
	})
	defer stopProcess(t, cmd)
	createOpenAIChannel(t, apiAddr, "openai-broken", "http://127.0.0.1:1/v1", "", "gpt-4o-mini")

	payload := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected bad gateway, got %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read transport error body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"Upstream request failed"`) {
		t.Fatalf("expected stable error message, got %s", text)
	}
	if strings.Contains(text, `dial tcp`) || strings.Contains(text, `127.0.0.1:1`) {
		t.Fatalf("expected internal transport details to be hidden, got %s", text)
	}
}

func TestChatCompletionsRequestTooLarge(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	largeContent := strings.Repeat("a", (10<<20)+1024)
	payload := fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}]}`, largeContent)
	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), payload, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected request too large, got %d: %s", response.StatusCode, string(body))
	}
}

func TestChatCompletionsRouteMissingIsStable(t *testing.T) {
	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)

	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected missing route to return 502, got %d: %s", response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read missing route body: %v", err)
	}
	if !strings.Contains(string(body), `No enabled openai route configured for model gpt-4o-mini`) {
		t.Fatalf("expected stable route error, got %s", string(body))
	}
}

func TestChatCompletionsFailoverAcrossSameAliasChannels(t *testing.T) {
	var firstHits int
	firstUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstHits++
		http.Error(w, "try next", http.StatusServiceUnavailable)
	}))
	defer firstUpstream.Close()

	var secondHits int
	secondUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"fallback-ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`))
	}))
	defer secondUpstream.Close()

	apiAddr := reserveLocalAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := startAPIServer(t, ctx, apiAddr, map[string]string{})
	defer stopProcess(t, cmd)
	createCompatChannel(t, apiAddr, map[string]any{"name": "openai-primary", "provider": "OpenAI", "endpoint": firstUpstream.URL + "/v1", "api_key": "primary-key", "enabled": true, "model_ids": []string{"gpt-4o-mini"}, "rpm_limit": 1000, "max_inflight": 32, "safety_factor": 0.9, "enabled_for_async": true, "dispatch_weight": 200})
	createCompatChannel(t, apiAddr, map[string]any{"name": "openai-fallback", "provider": "OpenAI", "endpoint": secondUpstream.URL + "/v1", "api_key": "fallback-key", "enabled": true, "model_ids": []string{"gpt-4o-mini"}, "rpm_limit": 1000, "max_inflight": 32, "safety_factor": 0.9, "enabled_for_async": true, "dispatch_weight": 100})

	response := doPOST(t, fmt.Sprintf("http://%s/v1/chat/completions", apiAddr), `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`, map[string]string{"Content-Type": "application/json"})
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected fallback request to return 200, got %d: %s", response.StatusCode, string(body))
	}
	if firstHits != 1 || secondHits != 1 {
		t.Fatalf("expected one hit on each upstream, got first=%d second=%d", firstHits, secondHits)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fallback response body: %v", err)
	}
	if !strings.Contains(string(body), `fallback-ok`) {
		t.Fatalf("expected fallback response body, got %s", string(body))
	}
}

func startAPIServer(t *testing.T, ctx context.Context, addr string, extraEnv map[string]string) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/api")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "OPENCRAB_HTTP_ADDR="+addr)
	if _, ok := extraEnv["OPENCRAB_STATE_PATH"]; !ok {
		if _, hasDBPath := extraEnv["OPENCRAB_DB_PATH"]; !hasDBPath {
			cmd.Env = append(cmd.Env, "OPENCRAB_STATE_PATH="+filepath.Join(t.TempDir(), "opencrab-state.json"))
		}
	}
	for key, value := range extraEnv {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start api server: %v", err)
	}
	return cmd
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func doPOST(t *testing.T, url string, body string, headers map[string]string) *http.Response {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, headers)
}

func doRequest(t *testing.T, method string, url string, body string, headers map[string]string) *http.Response {
	t.Helper()
	bodyBytes := []byte(body)
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return doRequestWithRetry(t, req, bodyBytes, 10*time.Second)
}

func waitForGET(t *testing.T, url string, timeout time.Duration) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new get request: %v", err)
	}
	return doRequestWithRetry(t, req, nil, timeout)
}

func doRequestWithRetry(t *testing.T, req *http.Request, body []byte, timeout time.Duration) *http.Response {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		attemptReq := req.Clone(context.Background())
		if body != nil {
			attemptReq.Body = io.NopCloser(bytes.NewReader(body))
			attemptReq.ContentLength = int64(len(body))
		}
		response, err := client.Do(attemptReq)
		if err == nil {
			return response
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("request %s %s failed before timeout: %v", req.Method, req.URL.String(), lastErr)
	return nil
}

func assertContentTypeContains(t *testing.T, response *http.Response, expected string) {
	t.Helper()
	if !strings.Contains(response.Header.Get("Content-Type"), expected) {
		t.Fatalf("expected content type %q, got %q", expected, response.Header.Get("Content-Type"))
	}
}

func reserveLocalAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local port: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

func repoRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(directory, ".."))
}

func createCompatChannel(t *testing.T, apiAddr string, payload map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal channel payload: %v", err)
	}
	response := doPOST(t, fmt.Sprintf("http://%s/api/admin/channels", apiAddr), string(body), map[string]string{"Content-Type": "application/json"})
	responseBody, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		t.Fatalf("read create channel body: %v", readErr)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected create channel 201, got %d: %s", response.StatusCode, string(responseBody))
	}
}

func createOpenAIChannel(t *testing.T, apiAddr string, name string, endpoint string, apiKey string, modelIDs ...string) {
	t.Helper()
	createCompatChannel(t, apiAddr, map[string]any{
		"name":              name,
		"provider":          "OpenAI",
		"endpoint":          endpoint,
		"api_key":           apiKey,
		"enabled":           true,
		"model_ids":         modelIDs,
		"rpm_limit":         1000,
		"max_inflight":      32,
		"safety_factor":     0.9,
		"enabled_for_async": true,
		"dispatch_weight":   100,
	})
}

func createClaudeChannel(t *testing.T, apiAddr string, name string, endpoint string, apiKey string, modelIDs ...string) {
	t.Helper()
	createCompatChannel(t, apiAddr, map[string]any{
		"name":              name,
		"provider":          "Claude",
		"endpoint":          endpoint,
		"api_key":           apiKey,
		"enabled":           true,
		"model_ids":         modelIDs,
		"rpm_limit":         1000,
		"max_inflight":      32,
		"safety_factor":     0.9,
		"enabled_for_async": true,
		"dispatch_weight":   100,
	})
}
