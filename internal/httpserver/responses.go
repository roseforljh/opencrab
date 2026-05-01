package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

const maxResponsesRequestBodyBytes = 10 << 20

type responsesHandler struct {
	service *gateway.Service
}

type openAIResponsesEnvelope struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

func newResponsesHandler(service *gateway.Service) http.Handler {
	return &responsesHandler{service: service}
}

func (h *responsesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	body, err := io.ReadAll(io.LimitReader(r.Body, maxResponsesRequestBodyBytes+1))
	if err != nil {
		statusCode = http.StatusBadRequest
		writeOpenAIError(w, statusCode, "Failed to read request body", "invalid_request_error", nil, nil)
		return
	}
	if len(body) > maxResponsesRequestBodyBytes {
		statusCode = http.StatusRequestEntityTooLarge
		writeOpenAIError(w, statusCode, "Request body is too large", "invalid_request_error", nil, stringPointer("context_length_exceeded"))
		return
	}
	request, err := normalizeResponsesRequest(r, body)
	if err != nil {
		statusCode = http.StatusBadRequest
		validationError := &requestValidationError{}
		if errors.As(err, &validationError) {
			logOpenAIRequest("", statusCode, startedAt, body, nil, err, "/v1/responses")
			writeOpenAIError(w, statusCode, validationError.message, "invalid_request_error", validationError.param, nil)
			return
		}
		logOpenAIRequest("", statusCode, startedAt, body, nil, err, "/v1/responses")
		writeOpenAIError(w, statusCode, err.Error(), "invalid_request_error", nil, nil)
		return
	}
	routes, err := resolveResponsesRoutes(request.Model)
	if err != nil {
		statusCode = http.StatusBadGateway
		logOpenAIRequest(request.Model, statusCode, startedAt, body, nil, err, "/v1/responses")
		writeOpenAIError(w, statusCode, err.Error(), "server_error", nil, nil)
		return
	}
	request.RouteCandidates = routes
	if h.service == nil {
		statusCode = http.StatusInternalServerError
		logOpenAIRequest("", statusCode, startedAt, body, nil, errors.New("Gateway service not configured"), "/v1/responses")
		writeOpenAIError(w, statusCode, "Gateway service not configured", "server_error", nil, nil)
		return
	}
	response, err := h.service.Responses(r.Context(), request)
	if err != nil {
		statusCode = upstreamStatusCode(err)
		logOpenAIRequest(request.Model, statusCode, startedAt, body, nil, err, "/v1/responses")
		writeOpenAIError(w, statusCode, upstreamErrorMessage(err), upstreamErrorType(statusCode), nil, nil)
		return
	}
	defer drainAndClose(response.Body)
	statusCode = response.StatusCode
	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)
	if response.Stream {
		captured, streamErr := streamResponse(r.Context(), w, response.Body)
		logOpenAIRequest(request.Model, statusCode, startedAt, body, captured, streamErr, "/v1/responses")
		return
	}
	responseBody, readErr := readAndReplayResponse(w, response)
	logOpenAIRequest(request.Model, statusCode, startedAt, body, responseBody, readErr, "/v1/responses")
}

func normalizeResponsesRequest(r *http.Request, body []byte) (gateway.ResponsesRequest, error) {
	stream, model, err := parseResponsesEnvelope(body)
	if err != nil {
		return gateway.ResponsesRequest{}, err
	}
	return gateway.ResponsesRequest{
		Model:         model,
		Stream:        stream,
		Body:          body,
		ContentType:   r.Header.Get("Content-Type"),
		Accept:        r.Header.Get("Accept"),
		Authorization: r.Header.Get("Authorization"),
		Headers:       r.Header.Clone(),
	}, nil
}

func parseResponsesEnvelope(body []byte) (bool, string, error) {
	var payload openAIResponsesEnvelope
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, "", &requestValidationError{message: "Invalid JSON body"}
	}
	if strings.TrimSpace(payload.Model) == "" {
		return false, "", &requestValidationError{message: "Field 'model' is required", param: stringPointer("model")}
	}
	return payload.Stream, payload.Model, nil
}
