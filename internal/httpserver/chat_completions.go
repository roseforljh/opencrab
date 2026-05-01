package httpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

const maxChatCompletionsRequestBodyBytes = 10 << 20

type chatCompletionsHandler struct {
	service *gateway.Service
}

type chatCompletionsRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type requestValidationError struct {
	message string
	param   *string
}

func (e *requestValidationError) Error() string {
	return e.message
}

func newChatCompletionsHandler(service *gateway.Service) http.Handler {
	return &chatCompletionsHandler{service: service}
}

func (h *chatCompletionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	statusCode := http.StatusOK
	defer func() {
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, statusCode, time.Since(startedAt).Round(time.Millisecond))
	}()

	if r.Method != http.MethodPost {
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", http.MethodPost)
		writeOpenAIError(w, statusCode, "Method not allowed", "invalid_request_error", nil, nil)
		return
	}

	if !strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		statusCode = http.StatusUnsupportedMediaType
		writeOpenAIError(w, statusCode, "Content-Type must be application/json", "invalid_request_error", stringPointer("content-type"), nil)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxChatCompletionsRequestBodyBytes+1))
	if err != nil {
		statusCode = http.StatusBadRequest
		writeOpenAIError(w, statusCode, "Failed to read request body", "invalid_request_error", nil, nil)
		return
	}
	if len(body) > maxChatCompletionsRequestBodyBytes {
		statusCode = http.StatusRequestEntityTooLarge
		writeOpenAIError(w, statusCode, "Request body is too large", "invalid_request_error", nil, stringPointer("context_length_exceeded"))
		return
	}

	if err := validateChatCompletionsRequest(body); err != nil {
		statusCode = http.StatusBadRequest
		validationError := &requestValidationError{}
		if errors.As(err, &validationError) {
			logOpenAIRequest("", statusCode, startedAt, body, nil, err)
			writeOpenAIError(w, statusCode, validationError.message, "invalid_request_error", validationError.param, nil)
			return
		}
		logOpenAIRequest("", statusCode, startedAt, body, nil, err)
		writeOpenAIError(w, statusCode, err.Error(), "invalid_request_error", nil, nil)
		return
	}

	if h.service == nil {
		statusCode = http.StatusInternalServerError
		logOpenAIRequest("", statusCode, startedAt, body, nil, errors.New("Gateway service not configured"))
		writeOpenAIError(w, statusCode, "Gateway service not configured", "server_error", nil, nil)
		return
	}

	request, err := normalizeChatCompletionsRequest(r, body)
	if err != nil {
		statusCode = http.StatusBadRequest
		validationError := &requestValidationError{}
		if errors.As(err, &validationError) {
			writeOpenAIError(w, statusCode, validationError.message, "invalid_request_error", validationError.param, nil)
			return
		}
		writeOpenAIError(w, statusCode, err.Error(), "invalid_request_error", nil, nil)
		return
	}
	routes, err := resolveChatCompletionsRoutes(request.Model)
	if err != nil {
		statusCode = http.StatusBadGateway
		logOpenAIRequest(request.Model, statusCode, startedAt, body, nil, err)
		writeOpenAIError(w, statusCode, err.Error(), "server_error", nil, nil)
		return
	}
	request.RouteCandidates = routes

	response, err := h.service.ChatCompletions(r.Context(), request)
	if err != nil {
		statusCode = upstreamStatusCode(err)
		logOpenAIRequest(request.Model, statusCode, startedAt, body, nil, err)
		writeOpenAIError(w, statusCode, upstreamErrorMessage(err), upstreamErrorType(statusCode), nil, nil)
		return
	}
	defer drainAndClose(response.Body)

	statusCode = response.StatusCode
	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if response.Stream {
		captured, streamErr := streamResponse(r.Context(), w, response.Body)
		logOpenAIRequest(request.Model, statusCode, startedAt, body, captured, streamErr)
		return
	}

	responseBody, readErr := readAndReplayResponse(w, response)
	logOpenAIRequest(request.Model, statusCode, startedAt, body, responseBody, readErr)
}

func normalizeChatCompletionsRequest(r *http.Request, body []byte) (gateway.ChatCompletionsRequest, error) {
	stream, model, err := parseChatCompletionsEnvelope(body)
	if err != nil {
		return gateway.ChatCompletionsRequest{}, err
	}
	return gateway.ChatCompletionsRequest{
		Model:         model,
		Stream:        stream,
		Body:          body,
		ContentType:   r.Header.Get("Content-Type"),
		Accept:        r.Header.Get("Accept"),
		Authorization: r.Header.Get("Authorization"),
		Headers:       r.Header.Clone(),
	}, nil
}

func validateChatCompletionsRequest(body []byte) error {
	_, _, err := parseChatCompletionsEnvelope(body)
	return err
}

func parseChatCompletionsEnvelope(body []byte) (bool, string, error) {
	var payload chatCompletionsRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, "", &requestValidationError{message: "Invalid JSON body"}
	}
	if strings.TrimSpace(payload.Model) == "" {
		return false, "", &requestValidationError{message: "Field 'model' is required", param: stringPointer("model")}
	}
	if len(payload.Messages) == 0 {
		return false, "", &requestValidationError{message: "Field 'messages' must contain at least one message", param: stringPointer("messages")}
	}
	for index, message := range payload.Messages {
		if strings.TrimSpace(message.Role) == "" {
			return false, "", &requestValidationError{message: fmt.Sprintf("messages[%d].role is required", index), param: stringPointer("messages")}
		}
	}
	return requestsStream(body), payload.Model, nil
}

func requestsStream(body []byte) bool {
	var payload struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payload.Stream
}

func streamResponse(ctx context.Context, w http.ResponseWriter, body io.Reader) ([]byte, error) {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)
	var captured strings.Builder
	for {
		select {
		case <-ctx.Done():
			return []byte(captured.String()), ctx.Err()
		default:
		}

		chunk, err := reader.ReadBytes('\n')
		if len(chunk) > 0 {
			captured.Write(chunk)
			_, _ = w.Write(chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return []byte(captured.String()), nil
			}
			return []byte(captured.String()), err
		}
	}
}

func copyResponseHeaders(target http.Header, source http.Header) {
	for key, values := range source {
		if strings.EqualFold(key, "Content-Length") || strings.EqualFold(key, "Transfer-Encoding") || strings.EqualFold(key, "Connection") {
			continue
		}
		for _, value := range values {
			target.Add(key, value)
		}
	}
}

func writeOpenAIError(w http.ResponseWriter, statusCode int, message string, errorType string, param *string, code *string) {
	writeJSON(w, statusCode, map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errorType,
			"param":   param,
			"code":    code,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func stringPointer(value string) *string {
	return &value
}

func upstreamStatusCode(err error) int {
	transportError := &gateway.TransportError{}
	if errors.As(err, &transportError) && transportError.Timeout {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func upstreamErrorMessage(err error) string {
	transportError := &gateway.TransportError{}
	if errors.As(err, &transportError) && transportError.Timeout {
		return "Upstream request timed out"
	}
	return "Upstream request failed"
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

func upstreamErrorType(statusCode int) string {
	switch statusCode {
	case http.StatusGatewayTimeout:
		return "server_error"
	default:
		return "server_error"
	}
}
