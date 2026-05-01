package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opencrab/internal/gateway"
)

const maxGeminiRequestBodyBytes = 10 << 20

type geminiModelsHandler struct {
	service *gateway.Service
}

type geminiGenerateRequest struct {
	Contents []any `json:"contents"`
}

func newGeminiModelsHandler(service *gateway.Service) http.Handler {
	return &geminiModelsHandler{service: service}
}

func (h *geminiModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	statusCode := http.StatusOK
	defer func() {
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, statusCode, time.Since(startedAt).Round(time.Millisecond))
	}()

	if r.Method != http.MethodPost {
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", http.MethodPost)
		writeGeminiError(w, statusCode, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		statusCode = http.StatusUnsupportedMediaType
		writeGeminiError(w, statusCode, "Content-Type must be application/json", "INVALID_ARGUMENT")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxGeminiRequestBodyBytes+1))
	if err != nil {
		statusCode = http.StatusBadRequest
		writeGeminiError(w, statusCode, "Failed to read request body", "INVALID_ARGUMENT")
		return
	}
	if len(body) > maxGeminiRequestBodyBytes {
		statusCode = http.StatusRequestEntityTooLarge
		writeGeminiError(w, statusCode, "Request body is too large", "INVALID_ARGUMENT")
		return
	}
	request, err := normalizeGeminiGenerateContentRequest(r, body)
	if err != nil {
		routingError := &gateway.RoutingError{}
		if errors.As(err, &routingError) {
			statusCode = http.StatusBadGateway
			logGeminiRequest("", r.URL.Path, statusCode, startedAt, body, nil, err)
			writeGeminiError(w, statusCode, err.Error(), "BAD_GATEWAY")
			return
		}
		statusCode = http.StatusBadRequest
		logGeminiRequest("", r.URL.Path, statusCode, startedAt, body, nil, err)
		writeGeminiError(w, statusCode, err.Error(), "INVALID_ARGUMENT")
		return
	}
	if h.service == nil {
		statusCode = http.StatusInternalServerError
		logGeminiRequest(request.Model, r.URL.Path, statusCode, startedAt, body, nil, errors.New("Gateway service not configured"))
		writeGeminiError(w, statusCode, "Gateway service not configured", "INTERNAL")
		return
	}
	response, err := h.service.GenerateContent(r.Context(), request)
	if err != nil {
		statusCode = upstreamStatusCode(err)
		logGeminiRequest(request.Model, r.URL.Path, statusCode, startedAt, body, nil, err)
		writeGeminiError(w, statusCode, upstreamGeminiErrorMessage(err), upstreamGeminiErrorStatus(statusCode))
		return
	}
	defer drainAndClose(response.Body)

	statusCode = response.StatusCode
	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if response.Stream {
		captured, streamErr := streamResponse(r.Context(), w, response.Body)
		logGeminiRequest(request.Model, r.URL.Path, statusCode, startedAt, body, captured, streamErr)
		return
	}
	responseBody, readErr := readAndReplayResponse(w, response)
	logGeminiRequest(request.Model, r.URL.Path, statusCode, startedAt, body, responseBody, readErr)
}

func normalizeGeminiGenerateContentRequest(r *http.Request, body []byte) (gateway.GenerateContentRequest, error) {
	model, stream, err := parseGeminiModelPath(r.URL.Path)
	if err != nil {
		return gateway.GenerateContentRequest{}, err
	}
	var payload geminiGenerateRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return gateway.GenerateContentRequest{}, fmt.Errorf("Invalid JSON body")
	}
	if len(payload.Contents) == 0 {
		return gateway.GenerateContentRequest{}, fmt.Errorf("Field 'contents' must contain at least one item")
	}
	var routes []gateway.UpstreamRouteCandidate
	if stream {
		routes, err = resolveGeminiStreamGenerateContentRoutes(model)
	} else {
		routes, err = resolveGeminiGenerateContentRoutes(model)
	}
	if err != nil {
		return gateway.GenerateContentRequest{}, err
	}
	return gateway.GenerateContentRequest{
		Model:           model,
		Stream:          stream,
		Body:            body,
		ContentType:     r.Header.Get("Content-Type"),
		Accept:          r.Header.Get("Accept"),
		Headers:         r.Header.Clone(),
		RouteCandidates: routes,
	}, nil
}

func parseGeminiModelPath(path string) (string, bool, error) {
	prefix := "/v1beta/models/"
	if !strings.HasPrefix(path, prefix) {
		return "", false, fmt.Errorf("Unsupported Gemini path")
	}
	rest := strings.TrimPrefix(path, prefix)
	if model, ok := strings.CutSuffix(rest, ":generateContent"); ok && strings.TrimSpace(model) != "" {
		decoded, err := url.PathUnescape(model)
		if err != nil {
			return "", false, fmt.Errorf("Invalid model path")
		}
		return decoded, false, nil
	}
	if model, ok := strings.CutSuffix(rest, ":streamGenerateContent"); ok && strings.TrimSpace(model) != "" {
		decoded, err := url.PathUnescape(model)
		if err != nil {
			return "", false, fmt.Errorf("Invalid model path")
		}
		return decoded, true, nil
	}
	return "", false, fmt.Errorf("Unsupported Gemini operation")
}

func writeGeminiError(w http.ResponseWriter, statusCode int, message string, status string) {
	writeJSON(w, statusCode, map[string]any{
		"error": map[string]any{
			"code":    statusCode,
			"message": message,
			"status":  status,
		},
	})
}

func upstreamGeminiErrorMessage(err error) string {
	transportError := &gateway.TransportError{}
	if errors.As(err, &transportError) && transportError.Timeout {
		return "Upstream request timed out"
	}
	return err.Error()
}

func upstreamGeminiErrorStatus(statusCode int) string {
	if statusCode == http.StatusGatewayTimeout {
		return "DEADLINE_EXCEEDED"
	}
	return "BAD_GATEWAY"
}
