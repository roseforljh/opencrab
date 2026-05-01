package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

func logOpenAIRequest(model string, statusCode int, startedAt time.Time, requestBody []byte, responseBody []byte, err error) {
	promptTokens, completionTokens, totalTokens := parseOpenAIUsage(responseBody)
	details := map[string]any{
		"log_type":         "gateway_request",
		"provider":         "openai",
		"selected_channel": "openai",
		"request_path":     "/v1/chat/completions",
		"response_status":  statusCode,
		"upstream_model":   model,
	}
	if err != nil {
		details["error_message"] = err.Error()
	}
	_, _ = compatRequestLogs.append(adminCompatRequestLogInput{Model: model, Channel: "openai", StatusCode: statusCode, LatencyMS: time.Since(startedAt).Milliseconds(), PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTokens: totalTokens, RequestBody: string(requestBody), ResponseBody: string(responseBody), Details: details})
}

func logClaudeRequest(model string, providerFamily string, statusCode int, startedAt time.Time, requestBody []byte, responseBody []byte, err error) {
	inputTokens, outputTokens, totalTokens := parseClaudeUsage(responseBody)
	provider := "claude"
	channel := "claude"
	if strings.EqualFold(strings.TrimSpace(providerFamily), "openai") {
		provider = "openai"
		channel = "openai"
	}
	details := map[string]any{
		"log_type":         "gateway_request",
		"provider":         provider,
		"selected_channel": channel,
		"request_path":     "/v1/messages",
		"response_status":  statusCode,
		"upstream_model":   model,
	}
	if err != nil {
		details["error_message"] = err.Error()
	}
	_, _ = compatRequestLogs.append(adminCompatRequestLogInput{Model: model, Channel: channel, StatusCode: statusCode, LatencyMS: time.Since(startedAt).Milliseconds(), PromptTokens: inputTokens, CompletionTokens: outputTokens, TotalTokens: totalTokens, RequestBody: string(requestBody), ResponseBody: string(responseBody), Details: details})
}

func logGeminiRequest(model string, requestPath string, statusCode int, startedAt time.Time, requestBody []byte, responseBody []byte, err error) {
	promptTokens, completionTokens, totalTokens := parseGeminiUsage(responseBody)
	details := map[string]any{
		"log_type":         "gateway_request",
		"provider":         "gemini",
		"selected_channel": "gemini",
		"request_path":     requestPath,
		"response_status":  statusCode,
		"upstream_model":   model,
	}
	if err != nil {
		details["error_message"] = err.Error()
	}
	_, _ = compatRequestLogs.append(adminCompatRequestLogInput{Model: model, Channel: "gemini", StatusCode: statusCode, LatencyMS: time.Since(startedAt).Milliseconds(), PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTokens: totalTokens, RequestBody: string(requestBody), ResponseBody: string(responseBody), Details: details})
}

func parseOpenAIUsage(body []byte) (int, int, int) {
	var payload struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, 0
	}
	return payload.Usage.PromptTokens, payload.Usage.CompletionTokens, payload.Usage.TotalTokens
}

func parseClaudeUsage(body []byte) (int, int, int) {
	var payload struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		var inputTokens, outputTokens int
		for _, line := range strings.Split(string(body), "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var eventPayload struct {
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &eventPayload); err != nil {
				continue
			}
			if eventPayload.Message.Usage.InputTokens > 0 {
				inputTokens = eventPayload.Message.Usage.InputTokens
			}
			if eventPayload.Usage.InputTokens > 0 {
				inputTokens = eventPayload.Usage.InputTokens
			}
			if eventPayload.Usage.OutputTokens > 0 {
				outputTokens = eventPayload.Usage.OutputTokens
			}
		}
		return inputTokens, outputTokens, inputTokens + outputTokens
	}
	return payload.Usage.InputTokens, payload.Usage.OutputTokens, payload.Usage.InputTokens + payload.Usage.OutputTokens
}

func parseGeminiUsage(body []byte) (int, int, int) {
	var payload struct {
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		return payload.UsageMetadata.PromptTokenCount, payload.UsageMetadata.CandidatesTokenCount, payload.UsageMetadata.TotalTokenCount
	}
	var promptTokens, completionTokens, totalTokens int
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
			continue
		}
		promptTokens = payload.UsageMetadata.PromptTokenCount
		completionTokens = payload.UsageMetadata.CandidatesTokenCount
		totalTokens = payload.UsageMetadata.TotalTokenCount
	}
	return promptTokens, completionTokens, totalTokens
}

func readAndReplayResponse(w http.ResponseWriter, response *gateway.ProxyResponse) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	_, writeErr := w.Write(body)
	if writeErr != nil {
		return body, writeErr
	}
	return body, nil
}

func clearRequestLogsHandler(w http.ResponseWriter, r *http.Request) {
	if err := compatRequestLogs.clear(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func adminLogsHandler(w http.ResponseWriter, r *http.Request) {
	items, total, filtered := compatRequestLogs.list(r.URL.Query().Get("q"), r.URL.Query().Get("category"))
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "filtered": filtered})
}

func adminLogDetailHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminEntityID(r.PathValue("id"), "无效日志编号")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, ok := compatRequestLogs.detail(id)
	if !ok {
		http.Error(w, "日志不存在", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, requestLogToDetailAPI(item))
}

func upstreamStatusFromError(err error) int {
	transportError := &gateway.TransportError{}
	if errors.As(err, &transportError) && transportError.Timeout {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func safeResponseBody(body []byte) []byte {
	if strings.Contains(string(body), "[DONE]") {
		return body
	}
	return body
}
