package httpserver

import (
	"bufio"
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

const maxMessagesRequestBodyBytes = 10 << 20

type messagesHandler struct {
	service *gateway.Service
}

type anthropicMessagesRequest struct {
	Model     string                     `json:"model"`
	MaxTokens *int                       `json:"max_tokens"`
	Stream    bool                       `json:"stream"`
	Messages  []anthropicMessageEnvelope `json:"messages"`
}

type anthropicMessageEnvelope struct {
	Role string `json:"role"`
}

type anthropicValidationError struct {
	message string
}

func (e *anthropicValidationError) Error() string {
	return e.message
}

func newMessagesHandler(service *gateway.Service) http.Handler {
	return &messagesHandler{service: service}
}

func (h *messagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	statusCode := http.StatusOK
	defer func() {
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, statusCode, time.Since(startedAt).Round(time.Millisecond))
	}()

	if r.Method != http.MethodPost {
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", http.MethodPost)
		writeAnthropicError(w, statusCode, "Method not allowed", "invalid_request_error")
		return
	}

	if !strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		statusCode = http.StatusUnsupportedMediaType
		writeAnthropicError(w, statusCode, "Content-Type must be application/json", "invalid_request_error")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxMessagesRequestBodyBytes+1))
	if err != nil {
		statusCode = http.StatusBadRequest
		writeAnthropicError(w, statusCode, "Failed to read request body", "invalid_request_error")
		return
	}
	if len(body) > maxMessagesRequestBodyBytes {
		statusCode = http.StatusRequestEntityTooLarge
		writeAnthropicError(w, statusCode, "Request body is too large", "request_too_large")
		return
	}

	request, err := normalizeMessagesRequest(r, body)
	if err != nil {
		statusCode = http.StatusBadRequest
		validationError := &anthropicValidationError{}
		if errors.As(err, &validationError) {
			logClaudeRequest("", "", statusCode, startedAt, body, nil, err)
			writeAnthropicError(w, statusCode, validationError.message, "invalid_request_error")
			return
		}
		logClaudeRequest("", "", statusCode, startedAt, body, nil, err)
		writeAnthropicError(w, statusCode, err.Error(), "invalid_request_error")
		return
	}
	routes, err := resolveMessagesRoutes(request.Model, body)
	if err != nil {
		requestError := &gateway.RequestError{}
		if errors.As(err, &requestError) {
			statusCode = requestError.StatusCode
			logClaudeRequest(request.Model, "openai", statusCode, startedAt, body, nil, err)
			writeAnthropicError(w, statusCode, requestError.Message, "invalid_request_error")
			return
		}
		statusCode = http.StatusBadGateway
		logClaudeRequest(request.Model, "", statusCode, startedAt, body, nil, err)
		writeAnthropicError(w, statusCode, err.Error(), "api_error")
		return
	}
	request.RouteCandidates = routes

	if h.service == nil {
		statusCode = http.StatusInternalServerError
		logClaudeRequest("", "", statusCode, startedAt, body, nil, errors.New("Gateway service not configured"))
		writeAnthropicError(w, statusCode, "Gateway service not configured", "api_error")
		return
	}

	response, err := h.service.Messages(r.Context(), request)
	if err != nil {
		requestError := &gateway.RequestError{}
		if errors.As(err, &requestError) {
			statusCode = requestError.StatusCode
			logClaudeRequest(request.Model, requestError.UpstreamFamily, statusCode, startedAt, body, nil, err)
			writeAnthropicError(w, statusCode, requestError.Message, "invalid_request_error")
			return
		}
		statusCode = upstreamStatusCode(err)
		logClaudeRequest(request.Model, request.UpstreamFamily, statusCode, startedAt, body, nil, err)
		writeAnthropicError(w, statusCode, upstreamAnthropicErrorMessage(err), upstreamAnthropicErrorType(statusCode))
		return
	}
	defer drainAndClose(response.Body)

	statusCode = response.StatusCode
	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if response.Stream {
		captured, streamErr := streamAnthropicResponse(w, response.Body)
		logClaudeRequest(request.Model, response.UpstreamFamily, statusCode, startedAt, body, captured, streamErr)
		return
	}

	responseBody, readErr := readAndReplayResponse(w, response)
	logClaudeRequest(request.Model, response.UpstreamFamily, statusCode, startedAt, body, responseBody, readErr)
}

func normalizeMessagesRequest(r *http.Request, body []byte) (gateway.MessagesRequest, error) {
	stream, model, maxTokens, err := parseMessagesEnvelope(body)
	if err != nil {
		return gateway.MessagesRequest{}, err
	}
	request := gateway.MessagesRequest{
		Model:         model,
		Stream:        stream,
		Body:          body,
		ContentType:   r.Header.Get("Content-Type"),
		Accept:        r.Header.Get("Accept"),
		Authorization: r.Header.Get("Authorization"),
		Headers:       r.Header.Clone(),
		MaxTokens:     maxTokens,
	}
	return request, nil
}

func parseMessagesEnvelope(body []byte) (bool, string, int, error) {
	var payload anthropicMessagesRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, "", 0, &anthropicValidationError{message: "Invalid JSON body"}
	}
	if strings.TrimSpace(payload.Model) == "" {
		return false, "", 0, &anthropicValidationError{message: "Field 'model' is required"}
	}
	if payload.MaxTokens == nil {
		return false, "", 0, &anthropicValidationError{message: "Field 'max_tokens' is required"}
	}
	if *payload.MaxTokens < 0 {
		return false, "", 0, &anthropicValidationError{message: "Field 'max_tokens' must be greater than or equal to 0"}
	}
	if len(payload.Messages) == 0 {
		return false, "", 0, &anthropicValidationError{message: "Field 'messages' must contain at least one message"}
	}
	for index, message := range payload.Messages {
		if strings.TrimSpace(message.Role) == "" {
			return false, "", 0, &anthropicValidationError{message: fmt.Sprintf("messages[%d].role is required", index)}
		}
	}
	return payload.Stream, payload.Model, *payload.MaxTokens, nil
}

func writeAnthropicError(w http.ResponseWriter, statusCode int, message string, errorType string) {
	writeJSON(w, statusCode, map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errorType,
			"message": message,
		},
	})
}

func streamAnthropicResponse(w http.ResponseWriter, body io.Reader) ([]byte, error) {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)
	var captured strings.Builder
	for {
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

func upstreamAnthropicErrorMessage(err error) string {
	transportError := &gateway.TransportError{}
	if errors.As(err, &transportError) && transportError.Timeout {
		return "Upstream request timed out"
	}
	return "Upstream request failed"
}

func upstreamAnthropicErrorType(statusCode int) string {
	switch statusCode {
	case http.StatusGatewayTimeout:
		return "timeout_error"
	default:
		return "api_error"
	}
}
