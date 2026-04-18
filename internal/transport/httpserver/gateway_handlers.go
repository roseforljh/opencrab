package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"
	"opencrab/internal/transform"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func HandleGatewayChatCompletions(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeOpenAIGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result, startedAt)
	}
}

func HandleOpenAIModels(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ListModels == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			http.Error(w, "models handler not configured", http.StatusNotImplemented)
			return
		}

		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}

		models, err := deps.ListModels(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type openAIModel struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}

		data := make([]openAIModel, 0, len(models))
		seen := make(map[string]struct{}, len(models))
		allowedByChannel := map[string]struct{}{}
		if len(scope.ChannelNames) > 0 && deps.ListModelRoutes != nil {
			routes, routeErr := deps.ListModelRoutes(req.Context())
			if routeErr != nil {
				http.Error(w, routeErr.Error(), http.StatusInternalServerError)
				return
			}
			for _, route := range routes {
				if scopeListContains(scope.ChannelNames, route.ChannelName) {
					allowedByChannel[strings.TrimSpace(route.ModelAlias)] = struct{}{}
				}
			}
		}
		for _, model := range models {
			alias := strings.TrimSpace(model.Alias)
			if alias == "" {
				continue
			}
			if len(scope.ModelAliases) > 0 && !scopeListContains(scope.ModelAliases, alias) {
				continue
			}
			if len(scope.ChannelNames) > 0 && deps.ListModelRoutes != nil {
				if _, ok := allowedByChannel[alias]; !ok {
					continue
				}
			}
			if _, exists := seen[alias]; exists {
				continue
			}
			seen[alias] = struct{}{}
			data = append(data, openAIModel{
				ID:      alias,
				Object:  "model",
				Created: 0,
				OwnedBy: "opencrab",
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   data,
		})
	}
}

func HandleOpenAIResponses(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if wantsAsyncAdmission(req) {
			accepted, err := acceptGatewayRequest(deps, req, decodeOpenAIResponsesGatewayRequest)
			if err != nil {
				renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
				return
			}
			if tryWriteSyncBridgeResponse(deps, w, req, accepted) {
				return
			}
			writeJSON(w, http.StatusAccepted, accepted)
			return
		}
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeOpenAIResponsesGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		writeResponsesGatewayResult(deps, w, req, body, result, startedAt, transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIResponses})
	}
}

func HandleCodexResponses(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if wantsAsyncAdmission(req) {
			accepted, err := acceptGatewayRequest(deps, req, decodeCodexGatewayRequest)
			if err != nil {
				renderGatewayError(deps, w, err)
				return
			}
			if tryWriteSyncBridgeResponse(deps, w, req, accepted) {
				return
			}
			writeJSON(w, http.StatusAccepted, accepted)
			return
		}
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeCodexGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayError(deps, w, err)
			return
		}
		writeResponsesGatewayResult(deps, w, req, body, result, startedAt, transform.Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses})
	}
}

func tryWriteSyncBridgeResponse(deps Dependencies, w http.ResponseWriter, req *http.Request, accepted domain.GatewayAcceptedResponse) bool {
	if deps.GetDispatchRuntimeSettings == nil || deps.GetGatewayJobByRequestID == nil {
		return false
	}
	waitBudget := preferredWaitMs(req)
	if waitBudget <= 0 {
		return false
	}
	settings, err := deps.GetDispatchRuntimeSettings(req.Context())
	if err != nil {
		return false
	}
	if settings.SyncHoldMs <= 0 {
		return false
	}
	if waitBudget > settings.SyncHoldMs {
		waitBudget = settings.SyncHoldMs
	}
	deadline := time.Now().Add(time.Duration(waitBudget) * time.Millisecond)
	for time.Now().Before(deadline) {
		job, err := deps.GetGatewayJobByRequestID(req.Context(), accepted.RequestID)
		if err == nil {
			switch job.Status {
			case domain.GatewayJobStatusCompleted:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(job.ResponseBody))
				return true
			case domain.GatewayJobStatusFailed:
				renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(errors.New(job.ErrorMessage), max(job.ResponseStatusCode, 502), false, false), domain.ProtocolOpenAI)
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func HandleClaudeMessages(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeClaudeGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolClaude)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result, startedAt)
	}
}

func HandleClaudeCountTokens(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		startedAt := time.Now()
		if deps.CountClaudeTokens == nil || deps.CopyProxy == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderClaudeError(w, http.StatusNotImplemented, "count tokens handler not configured")
			return
		}
		_, _, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderClaudeError(w, gatewayErrorStatusCode(err), err.Error())
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			renderClaudeError(w, http.StatusBadRequest, "读取请求体失败")
			return
		}
		resp, err := deps.CountClaudeTokens(req.Context(), req, body)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, domain.ProtocolClaude, err, startedAt)
			renderClaudeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if err := deps.CopyProxy(w, resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logGatewayRequestSummary(deps, req, body, resp.StatusCode, resp.Headers, resp.Body, startedAt, nil)
	}
}

func HandleGeminiGenerateContent(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeGeminiGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result, startedAt)
	}
}

func HandleGeminiStreamGenerateContent(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, result, protocol, startedAt, err := executeGatewayRequest(deps, req, decodeGeminiGatewayRequest)
		if err != nil {
			logGatewayFailureSummary(deps, req, body, protocol, err, startedAt)
			renderGatewayError(deps, w, err)
			return
		}
		writeGatewayResult(deps, w, req, body, protocol, result, startedAt)
	}
}

type gatewayDecoder func(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error)

func executeGatewayRequest(deps Dependencies, req *http.Request, decode gatewayDecoder) ([]byte, *domain.ExecutionResult, domain.Protocol, time.Time, error) {
	startedAt := time.Now()
	if deps.ExecuteGateway == nil || deps.CopyProxy == nil || deps.CopyStream == nil {
		return nil, nil, "", startedAt, fmt.Errorf("gateway handler not configured")
	}
	if deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil {
		return nil, nil, "", startedAt, fmt.Errorf("api key verifier not configured")
	}

	rawKey, scope, err := resolveGatewayAPIKey(deps, req)
	if err != nil {
		return nil, nil, "", startedAt, err
	}
	if deps.CheckRateLimit != nil && !deps.CheckRateLimit(rawKey) {
		return nil, nil, "", startedAt, fmt.Errorf("请求过于频繁，请稍后再试")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, "", startedAt, fmt.Errorf("读取请求体失败")
	}
	gatewayReq, protocol, err := decode(body, req)
	if err != nil {
		return body, nil, protocol, startedAt, err
	}
	if err := applyAPIKeyScopeToGatewayRequest(&gatewayReq, scope); err != nil {
		return body, nil, protocol, startedAt, err
	}
	if deps.GetGatewayRuntimeSettings != nil {
		settings, settingsErr := deps.GetGatewayRuntimeSettings(req.Context())
		if settingsErr != nil {
			return body, nil, protocol, startedAt, settingsErr
		}
		gatewayReq.AffinityKey = extractSessionAffinityKey(req, gatewayReq, settings)
		gatewayReq.RuntimeSettings = &settings
	}
	gatewayReq, err = preprocessGatewayRequest(deps.ResponseSessions, gatewayReq)
	if err != nil {
		return body, nil, protocol, startedAt, err
	}
	result, err := deps.ExecuteGateway(req.Context(), middleware.GetReqID(req.Context()), gatewayReq)
	if err != nil {
		return body, nil, protocol, startedAt, err
	}
	return body, result, protocol, startedAt, nil
}

func acceptGatewayRequest(deps Dependencies, req *http.Request, decode gatewayDecoder) (domain.GatewayAcceptedResponse, error) {
	if deps.CreateGatewayJob == nil || deps.GetGatewayJobByRequestID == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("gateway admission handler not configured")
	}
	rawKey, scope, err := resolveGatewayAPIKey(deps, req)
	if err != nil {
		return domain.GatewayAcceptedResponse{}, err
	}
	if deps.CheckRateLimit != nil && !deps.CheckRateLimit(rawKey) {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("请求过于频繁，请稍后再试")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("读取请求体失败")
	}
	idempotencyKey := strings.TrimSpace(req.Header.Get("Idempotency-Key"))
	ownerKeyHash := gatewayOwnerKeyHash(rawKey)
	requestHash := gatewayAdmissionRequestHash(req.URL.Path, body)
	if idempotencyKey != "" && deps.GetGatewayJobByIdempotencyKey != nil {
		existing, getErr := deps.GetGatewayJobByIdempotencyKey(req.Context(), ownerKeyHash, idempotencyKey)
		if getErr == nil {
			if existing.RequestHash != requestHash || existing.RequestPath != req.URL.Path {
				return domain.GatewayAcceptedResponse{}, fmt.Errorf("Idempotency-Key 已被不同请求占用")
			}
			return buildAcceptedResponse(existing, true), nil
		}
		if !isGatewayJobNotFound(getErr) {
			return domain.GatewayAcceptedResponse{}, getErr
		}
	}
	gatewayReq, protocol, err := decode(body, req)
	if err != nil {
		return domain.GatewayAcceptedResponse{}, err
	}
	if err := applyAPIKeyScopeToGatewayRequest(&gatewayReq, scope); err != nil {
		return domain.GatewayAcceptedResponse{}, err
	}
	requestID := middleware.GetReqID(req.Context())
	if strings.TrimSpace(requestID) == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	acceptedAt := time.Now().Format(time.RFC3339)
	job, err := deps.CreateGatewayJob(req.Context(), domain.GatewayJob{
		RequestID:        requestID,
		IdempotencyKey:   idempotencyKey,
		OwnerKeyHash:     ownerKeyHash,
		RequestHash:      requestHash,
		Protocol:         protocol,
		Model:            gatewayReq.Model,
		Status:           domain.GatewayJobStatusAccepted,
		Mode:             "async",
		RequestPath:      req.URL.Path,
		RequestBody:      string(body),
		RequestHeaders:   marshalGatewayAdmissionHeaders(req.Header),
		SessionID:        extractGatewaySessionID(req),
		DeliveryMode:     extractDeliveryMode(req, body),
		WebhookURL:       extractWebhookURL(body),
		AcceptedAt:       acceptedAt,
		EstimatedReadyAt: acceptedAt,
		UpdatedAt:        acceptedAt,
	})
	if err != nil {
		if idempotencyKey != "" && isGatewayJobConflict(err) && deps.GetGatewayJobByIdempotencyKey != nil {
			existing, getErr := deps.GetGatewayJobByIdempotencyKey(req.Context(), ownerKeyHash, idempotencyKey)
			if getErr != nil {
				return domain.GatewayAcceptedResponse{}, getErr
			}
			if existing.RequestHash != requestHash || existing.RequestPath != req.URL.Path {
				return domain.GatewayAcceptedResponse{}, fmt.Errorf("Idempotency-Key 已被不同请求占用")
			}
			return buildAcceptedResponse(existing, true), nil
		}
		return domain.GatewayAcceptedResponse{}, err
	}
	return buildAcceptedResponse(job, false), nil
}

func HandleGatewayRequestStatus(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.GetGatewayJobByRequestID == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			http.Error(w, "gateway request status handler not configured", http.StatusNotImplemented)
			return
		}
		rawKey, _, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			http.Error(w, err.Error(), gatewayErrorStatusCode(err))
			return
		}
		item, err := deps.GetGatewayJobByRequestID(req.Context(), chi.URLParam(req, "requestID"))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if isGatewayJobNotFound(err) {
				statusCode = http.StatusNotFound
			}
			http.Error(w, err.Error(), statusCode)
			return
		}
		if item.OwnerKeyHash != gatewayOwnerKeyHash(rawKey) {
			http.Error(w, "请求不存在", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, buildGatewayJobStatusResponse(item))
	}
}

func HandleGatewayRequestEvents(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.GetGatewayJobByRequestID == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			http.Error(w, "gateway request events handler not configured", http.StatusNotImplemented)
			return
		}
		rawKey, _, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			http.Error(w, err.Error(), gatewayErrorStatusCode(err))
			return
		}
		item, err := deps.GetGatewayJobByRequestID(req.Context(), chi.URLParam(req, "requestID"))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if isGatewayJobNotFound(err) {
				statusCode = http.StatusNotFound
			}
			http.Error(w, err.Error(), statusCode)
			return
		}
		if item.OwnerKeyHash != gatewayOwnerKeyHash(rawKey) {
			http.Error(w, "请求不存在", http.StatusNotFound)
			return
		}
		writeGatewayJobEvents(w, item)
	}
}

func wantsAsyncAdmission(req *http.Request) bool {
	prefer := strings.ToLower(strings.TrimSpace(req.Header.Get("Prefer")))
	if strings.Contains(prefer, "respond-async") {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(req.Header.Get("X-OpenCrab-Async")))
	return value == "true" || value == "1"
}

func preferredWaitMs(req *http.Request) int {
	prefer := strings.ToLower(strings.TrimSpace(req.Header.Get("Prefer")))
	for _, part := range strings.Split(prefer, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "wait=") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(part, "wait="))
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return 0
		}
		return seconds * 1000
	}
	return 0
}

func marshalGatewayAdmissionHeaders(headers http.Header) string {
	allowedHeaders := map[string]struct{}{
		"accept":            {},
		"content-type":      {},
		"openai-beta":       {},
		"anthropic-version": {},
		"anthropic-beta":    {},
		"anthropic-dangerous-direct-browser-access": {},
		"x-opencrab-async":                          {},
		"idempotency-key":                           {},
		"x-requested-with":                          {},
		"x-session-id":                              {},
		"x-claude-code-session-id":                  {},
	}
	payload := map[string]string{}
	for key, values := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if _, ok := allowedHeaders[lowerKey]; !ok {
			continue
		}
		if len(values) == 0 {
			continue
		}
		payload[key] = values[0]
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func gatewayOwnerKeyHash(rawKey string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(rawKey)))
	return hex.EncodeToString(digest[:])
}

func gatewayAdmissionRequestHash(path string, body []byte) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(path) + "\n" + strings.TrimSpace(string(body))))
	return hex.EncodeToString(digest[:])
}

func buildAcceptedResponse(item domain.GatewayJob, replay bool) domain.GatewayAcceptedResponse {
	return domain.GatewayAcceptedResponse{
		RequestID:        item.RequestID,
		Status:           string(item.Status),
		Mode:             item.Mode,
		DeliveryMode:     item.DeliveryMode,
		AcceptedAt:       item.AcceptedAt,
		EstimatedReadyAt: item.EstimatedReadyAt,
		StatusURL:        fmt.Sprintf("/v1/requests/%s", item.RequestID),
		EventsURL:        fmt.Sprintf("/v1/requests/%s/events", item.RequestID),
		IdempotentReplay: replay,
	}
}

func isGatewayJobNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "请求不存在")
}

func isGatewayJobConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "gateway job 冲突")
}

func extractDeliveryMode(req *http.Request, body []byte) string {
	var payload struct {
		DeliveryMode string `json:"delivery_mode"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		mode := strings.ToLower(strings.TrimSpace(payload.DeliveryMode))
		if mode == "sse" || mode == "webhook" {
			return mode
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(req.Header.Get("Accept"))), "text/event-stream") {
		return "sse"
	}
	return "poll"
}

func extractWebhookURL(body []byte) string {
	var payload struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.WebhookURL)
}

func buildGatewayJobStatusResponse(item domain.GatewayJob) map[string]any {
	response := map[string]any{
		"request_id":           item.RequestID,
		"status":               item.Status,
		"mode":                 item.Mode,
		"delivery_mode":        item.DeliveryMode,
		"response_status_code": item.ResponseStatusCode,
		"error_message":        item.ErrorMessage,
		"attempt_count":        item.AttemptCount,
		"accepted_at":          item.AcceptedAt,
		"completed_at":         item.CompletedAt,
		"estimated_ready_at":   item.EstimatedReadyAt,
		"webhook_delivered_at": item.WebhookDeliveredAt,
	}
	if item.Status == domain.GatewayJobStatusCompleted && strings.TrimSpace(item.ResponseBody) != "" {
		response["result"] = json.RawMessage(item.ResponseBody)
	}
	return response
}

func writeGatewayJobEvents(w http.ResponseWriter, item domain.GatewayJob) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	created := map[string]any{"type": "response.created", "request_id": item.RequestID, "status": string(item.Status), "accepted_at": item.AcceptedAt}
	_, _ = w.Write([]byte(mustGatewaySSEEvent("response.created", created)))
	if item.Status == domain.GatewayJobStatusCompleted {
		completed := map[string]any{"type": "response.completed", "request_id": item.RequestID, "status": string(item.Status), "completed_at": item.CompletedAt}
		if strings.TrimSpace(item.ResponseBody) != "" {
			completed["result"] = json.RawMessage(item.ResponseBody)
		}
		_, _ = w.Write([]byte(mustGatewaySSEEvent("response.completed", completed)))
	} else if item.Status == domain.GatewayJobStatusFailed {
		failed := map[string]any{"type": "response.failed", "request_id": item.RequestID, "status": string(item.Status), "error_message": item.ErrorMessage}
		_, _ = w.Write([]byte(mustGatewaySSEEvent("response.failed", failed)))
	} else {
		progress := map[string]any{"type": "response.in_progress", "request_id": item.RequestID, "status": string(item.Status), "estimated_ready_at": item.EstimatedReadyAt}
		_, _ = w.Write([]byte(mustGatewaySSEEvent("response.in_progress", progress)))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func mustGatewaySSEEvent(event string, payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(`{"type":"error","message":"encode failed"}`)
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(encoded))
}

func decodeOpenAIGatewayRequest(body []byte, _ *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	request, err := transform.NormalizeGatewayRequest(transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIChatCompletions}, body, transform.NormalizeOptions{})
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return request, domain.ProtocolOpenAI, nil
}

func decodeOpenAIResponsesGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	request, err := transform.NormalizeGatewayRequest(
		transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIResponses},
		body,
		transform.NormalizeOptions{Headers: enrichRequestHeaders(req, []string{"OpenAI-Beta", "X-Stainless-Helper-Method", "X-Stainless-Retry-Count", "X-Stainless-Timeout"})},
	)
	return request, domain.ProtocolOpenAI, err
}

func decodeClaudeGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	request, err := transform.NormalizeGatewayRequest(
		transform.Surface{Protocol: domain.ProtocolClaude, Operation: domain.ProtocolOperationClaudeMessages},
		body,
		transform.NormalizeOptions{Headers: enrichRequestHeaders(req, []string{"anthropic-version", "anthropic-beta", "anthropic-dangerous-direct-browser-access"})},
	)
	return request, domain.ProtocolClaude, err
}

func decodeGeminiGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	operation := domain.ProtocolOperationGeminiGenerateContent
	forceStream := false
	if strings.Contains(req.URL.Path, ":streamGenerateContent") {
		forceStream = true
		operation = domain.ProtocolOperationGeminiStreamGenerate
	}
	request, err := transform.NormalizeGatewayRequest(
		transform.Surface{Protocol: domain.ProtocolGemini, Operation: operation},
		body,
		transform.NormalizeOptions{PathModel: chi.URLParam(req, "model"), Headers: enrichRequestHeaders(req, nil), ForceStream: forceStream},
	)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return request, domain.ProtocolGemini, nil
}

func decodeCodexGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	request, err := transform.NormalizeGatewayRequest(
		transform.Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses},
		body,
		transform.NormalizeOptions{Headers: enrichRequestHeaders(req, []string{"OpenAI-Beta", "X-Stainless-Helper-Method", "X-Stainless-Retry-Count", "X-Stainless-Timeout"})},
	)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return request, domain.ProtocolCodex, nil
}

func DecodeStoredGatewayRequest(protocol domain.Protocol, path string, body []byte, headerJSON string) (domain.GatewayRequest, error) {
	req, err := buildStoredGatewayRequest(path, headerJSON)
	if err != nil {
		return domain.GatewayRequest{}, err
	}
	switch protocol {
	case domain.ProtocolClaude:
		request, _, err := decodeClaudeGatewayRequest(body, req)
		return request, err
	case domain.ProtocolGemini:
		request, _, err := decodeGeminiGatewayRequest(body, req)
		return request, err
	case domain.ProtocolCodex:
		request, _, err := decodeCodexGatewayRequest(body, req)
		return request, err
	case domain.ProtocolOpenAI:
		if strings.Contains(strings.TrimSpace(path), "/responses") {
			request, _, err := decodeOpenAIResponsesGatewayRequest(body, req)
			return request, err
		}
		request, _, err := decodeOpenAIGatewayRequest(body, req)
		return request, err
	default:
		return domain.GatewayRequest{}, fmt.Errorf("不支持的协议: %s", protocol)
	}
}

func DecodeStoredGatewayJobRequest(store ResponseSessionStore, settingsStore func(context.Context) (domain.GatewayRuntimeSettings, error), job domain.GatewayJob) (domain.GatewayRequest, error) {
	req, err := DecodeStoredGatewayRequest(job.Protocol, job.RequestPath, []byte(job.RequestBody), job.RequestHeaders)
	if err != nil {
		return domain.GatewayRequest{}, err
	}
	httpReq, err := buildStoredGatewayRequest(job.RequestPath, job.RequestHeaders)
	if err != nil {
		return domain.GatewayRequest{}, err
	}
	if job.SessionID != "" {
		httpReq.Header.Set("X-Session-ID", job.SessionID)
		httpReq.Header.Set("X-Claude-Code-Session-Id", job.SessionID)
	}
	if settingsStore != nil {
		settings, settingsErr := settingsStore(context.Background())
		if settingsErr == nil {
			req.AffinityKey = extractSessionAffinityKey(httpReq, req, settings)
			req.RuntimeSettings = &settings
		}
	}
	req, err = preprocessGatewayRequest(store, req)
	if err != nil {
		return domain.GatewayRequest{}, err
	}
	return req, nil
}

func buildStoredGatewayRequest(path string, headerJSON string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(headerJSON) == "" {
		return req, nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(headerJSON), &headers); err != nil {
		return nil, fmt.Errorf("解析存储请求头失败: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil
}

func unifiedToGatewayRequest(unified domain.UnifiedChatRequest, headers map[string]string, session *domain.GatewaySessionState) domain.GatewayRequest {
	messages := make([]domain.GatewayMessage, 0, len(unified.Messages))
	for _, message := range unified.Messages {
		messages = append(messages, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, InputItem: message.InputItem, Metadata: message.Metadata})
	}
	policy := domain.GatewayToolCallReject
	if len(unified.Tools) > 0 || hasToolMessages(messages) || session != nil && len(session.ToolResults) > 0 {
		policy = domain.GatewayToolCallAllow
	}
	return domain.GatewayRequest{Protocol: unified.Protocol, Model: unified.Model, Stream: unified.Stream, Messages: messages, Tools: unified.Tools, Metadata: unified.Metadata, ToolCallPolicy: policy, RequestHeaders: headers, Session: session}
}

func writeGatewayResult(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, protocol domain.Protocol, result *domain.ExecutionResult, startedAt time.Time) {
	if result.Stream != nil {
		if err := deps.CopyStream(w, result.Stream); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logGatewayRequestSummary(deps, req, requestBody, result.Stream.StatusCode, result.Stream.Headers, nil, startedAt, result.Metadata)
		return
	}
	if result.Response == nil {
		http.Error(w, "empty gateway result", http.StatusBadGateway)
		return
	}
	providerName := normalizedHeaderProvider(result.Response.Headers)
	if requestWantsStreamForProtocol(protocol, req, requestBody) && shouldSynthesizeProtocolStream(protocol, providerName) {
		unified, err := decodeUnifiedByProvider(providerName, result.Response.Body)
		if err == nil {
			encoded, headers, streamErr := encodeProtocolStream(protocol, unified)
			if streamErr == nil {
				proxyResp := &domain.ProxyResponse{StatusCode: result.Response.StatusCode, Headers: headers, Body: encoded}
				if err := deps.CopyProxy(w, proxyResp); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				logGatewayRequestSummary(deps, req, requestBody, proxyResp.StatusCode, proxyResp.Headers, proxyResp.Body, startedAt, result.Metadata)
				return
			}
		}
	}
	resp := encodeGatewayResponseForSurface(result.Response, defaultSurfaceForProtocol(protocol, false))
	if err := deps.CopyProxy(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logGatewayRequestSummary(deps, req, requestBody, resp.StatusCode, resp.Headers, resp.Body, startedAt, result.Metadata)
}

func writeResponsesGatewayResult(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, result *domain.ExecutionResult, startedAt time.Time, surface transform.Surface) {
	if result != nil && result.Stream != nil {
		if err := deps.CopyStream(w, result.Stream); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logGatewayRequestSummary(deps, req, requestBody, result.Stream.StatusCode, result.Stream.Headers, nil, startedAt, result.Metadata)
		return
	}
	if result == nil || result.Response == nil {
		http.Error(w, "empty gateway result", http.StatusBadGateway)
		return
	}
	resp := encodeGatewayResponseForSurface(result.Response, surface)
	providerName := normalizedHeaderProvider(resp.Headers)
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		if err := deps.CopyProxy(w, resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	streamRequested := requestWantsStreamForProtocol(surface.Protocol, req, requestBody)
	if deps.ResponseSessions != nil {
		storeResponseSession(deps.ResponseSessions, req, requestBody, unified)
	}
	encoded, headers, err := transform.RenderClientResponse(surface, unified, streamRequested)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	proxyResp := &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: encoded}
	if err := deps.CopyProxy(w, proxyResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logGatewayRequestSummary(deps, req, requestBody, proxyResp.StatusCode, proxyResp.Headers, proxyResp.Body, startedAt, result.Metadata)
}

func encodeGatewayResponseForSurface(resp *domain.ProxyResponse, surface transform.Surface) *domain.ProxyResponse {
	providerName := normalizedHeaderProvider(resp.Headers)
	if protocolMatchesProvider(surface.Protocol, providerName) {
		return resp
	}
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		return resp
	}
	encoded, headers, err := transform.RenderClientResponse(surface, unified, false)
	if err != nil {
		return resp
	}
	return &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: encoded}
}

func decodeUnifiedByProvider(providerName string, body []byte) (domain.UnifiedChatResponse, error) {
	return transform.DecodeUpstreamResponse(providerName, body)
}

func normalizedHeaderProvider(headers map[string][]string) string {
	return domain.NormalizeProvider(firstHeaderValue(headers, "X-Opencrab-Provider"))
}

func protocolMatchesProvider(protocol domain.Protocol, providerName string) bool {
	switch protocol {
	case domain.ProtocolClaude:
		return providerName == "claude"
	case domain.ProtocolGemini:
		return providerName == "gemini"
	case domain.ProtocolCodex:
		return providerName == "" || providerName == "openai" || providerName == "openrouter" || providerName == "glm" || providerName == "kimi" || providerName == "minimax"
	default:
		return providerName == "" || providerName == "openai" || providerName == "openrouter" || providerName == "glm" || providerName == "kimi" || providerName == "minimax"
	}
}

func cloneHeaderMap(headers map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func logGatewayRequestSummary(deps Dependencies, req *http.Request, requestBody []byte, statusCode int, headers map[string][]string, responseBody []byte, startedAt time.Time, metadata *domain.GatewayExecutionMetadata) {
	if deps.CreateRequestLog == nil {
		return
	}
	channelName := firstHeaderValue(headers, "X-Opencrab-Channel")
	if channelName == "" {
		channelName = "default-channel"
	}
	modelName := extractModelFromRequest(req.URL.Path, requestBody)
	if strings.TrimSpace(modelName) == "unknown-model" && metadata != nil {
		switch {
		case len(metadata.VisitedAliases) > 0:
			modelName = metadata.VisitedAliases[0]
		case len(metadata.FallbackChain) > 0:
			modelName = metadata.FallbackChain[0]
		}
	}
	usage := usageMetrics{}
	loggedResponseBody := ""
	if len(responseBody) > 0 {
		usage = extractUsageMetrics(responseBody)
		loggedResponseBody = truncateLogBody(string(responseBody))
	}
	detailPayload := map[string]any{
		"log_type":          "gateway_request",
		"request_path":      req.URL.Path,
		"channel":           channelName,
		"model":             modelName,
		"response_status":   statusCode,
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":      usage.TotalTokens,
		"cache_hit":         usage.CacheHit,
		"test_mode":         false,
	}
	mergeGatewayExecutionMetadata(detailPayload, metadata)
	details := marshalLogDetails(detailPayload)
	_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
		RequestID:        middleware.GetReqID(req.Context()),
		Model:            modelName,
		Channel:          channelName,
		StatusCode:       statusCode,
		LatencyMs:        time.Since(startedAt).Milliseconds(),
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CacheHit:         usage.CacheHit,
		RequestBody:      truncateLogBody(string(requestBody)),
		ResponseBody:     loggedResponseBody,
		Details:          details,
		CreatedAt:        time.Now().Format(time.RFC3339),
	})
}

func logGatewayFailureSummary(deps Dependencies, req *http.Request, requestBody []byte, protocol domain.Protocol, err error, startedAt time.Time) {
	if deps.CreateRequestLog == nil {
		return
	}
	execErr := domain.AsExecutionError(err)
	statusCode := fallbackStatusCode(execErr.StatusCode)
	modelName := extractModelFromRequest(req.URL.Path, requestBody)
	metadata := execErr.Metadata
	if strings.TrimSpace(modelName) == "unknown-model" && metadata != nil {
		switch {
		case len(metadata.VisitedAliases) > 0:
			modelName = metadata.VisitedAliases[0]
		case len(metadata.FallbackChain) > 0:
			modelName = metadata.FallbackChain[0]
		}
	}
	detailPayload := map[string]any{
		"log_type":        "gateway_request",
		"request_path":    req.URL.Path,
		"model":           modelName,
		"response_status": statusCode,
		"error_message":   err.Error(),
		"test_mode":       false,
	}
	mergeGatewayExecutionMetadata(detailPayload, metadata)
	channelName := "gateway-error"
	if metadata != nil && strings.TrimSpace(metadata.StickyChannel) != "" {
		channelName = metadata.StickyChannel
	}
	_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
		RequestID:    middleware.GetReqID(req.Context()),
		Model:        modelName,
		Channel:      channelName,
		StatusCode:   statusCode,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		RequestBody:  truncateLogBody(string(requestBody)),
		ResponseBody: "",
		Details:      marshalLogDetails(detailPayload),
		CreatedAt:    time.Now().Format(time.RFC3339),
	})
	_ = protocol
}

func mergeGatewayExecutionMetadata(payload map[string]any, metadata *domain.GatewayExecutionMetadata) {
	if metadata == nil {
		return
	}
	payload["routing_strategy"] = metadata.RoutingStrategy
	payload["decision_reason"] = metadata.DecisionReason
	payload["fallback_stage"] = metadata.FallbackStage
	payload["fallback_chain"] = metadata.FallbackChain
	payload["visited_aliases"] = metadata.VisitedAliases
	payload["attempt_count"] = metadata.AttemptCount
	payload["sticky_hit"] = metadata.StickyHit
	payload["sticky_route_id"] = metadata.StickyRouteID
	payload["sticky_channel"] = metadata.StickyChannel
	payload["sticky_reason"] = metadata.StickyReason
	payload["affinity_key"] = metadata.AffinityKey
	payload["winning_bucket"] = metadata.WinningBucket
	payload["winning_priority"] = metadata.WinningPriority
	payload["selected_channel"] = metadata.SelectedChannel
	payload["skips"] = metadata.Skips
}

func renderGatewayError(deps Dependencies, w http.ResponseWriter, err error) {
	message := err.Error()
	if strings.Contains(message, "缺少 API Key") || strings.Contains(message, "无效或已禁用") {
		http.Error(w, message, http.StatusUnauthorized)
		return
	}
	if strings.Contains(message, "API Key 不允许") {
		http.Error(w, message, http.StatusForbidden)
		return
	}
	if strings.Contains(message, "请求过于频繁") {
		http.Error(w, message, http.StatusTooManyRequests)
		return
	}
	if deps.RenderProxyError != nil {
		deps.RenderProxyError(w, err)
		return
	}
	http.Error(w, message, http.StatusBadGateway)
}

func renderGatewayErrorForProtocol(deps Dependencies, w http.ResponseWriter, err error, protocol domain.Protocol) {
	if protocol == domain.ProtocolClaude {
		renderClaudeError(w, gatewayErrorStatusCode(err), err.Error())
		return
	}
	renderGatewayError(deps, w, err)
}

func gatewayErrorStatusCode(err error) int {
	message := err.Error()
	if strings.Contains(message, "缺少 API Key") || strings.Contains(message, "无效或已禁用") {
		return http.StatusUnauthorized
	}
	if strings.Contains(message, "API Key 不允许") {
		return http.StatusForbidden
	}
	if strings.Contains(message, "请求过于频繁") {
		return http.StatusTooManyRequests
	}
	return http.StatusBadGateway
}

func renderClaudeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "invalid_request_error",
			"message": message,
		},
	})
}

func requestWantsStream(body []byte) bool {
	var payload struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payload.Stream
}

func requestWantsStreamForProtocol(protocol domain.Protocol, req *http.Request, body []byte) bool {
	if protocol == domain.ProtocolGemini && req != nil && strings.Contains(req.URL.Path, ":streamGenerateContent") {
		return true
	}
	return requestWantsStream(body)
}

func encodeResponsesProxyResponse(resp domain.UnifiedChatResponse, stream bool) ([]byte, map[string][]string, error) {
	if stream {
		body, err := provider.EncodeOpenAIResponsesStream(resp)
		if err != nil {
			return nil, nil, err
		}
		return body, map[string][]string{"Content-Type": {"text/event-stream"}, "Cache-Control": {"no-cache"}, "Connection": {"keep-alive"}}, nil
	}
	body, err := provider.EncodeOpenAIResponsesResponse(resp)
	if err != nil {
		return nil, nil, err
	}
	return body, map[string][]string{"Content-Type": {"application/json"}}, nil
}

func shouldSynthesizeProtocolStream(protocol domain.Protocol, providerName string) bool {
	providerName = domain.NormalizeProvider(providerName)
	switch protocol {
	case domain.ProtocolClaude:
		return providerName != "claude"
	case domain.ProtocolGemini:
		return providerName != "gemini"
	case domain.ProtocolCodex:
		return providerName != "openai" && providerName != "openrouter" && providerName != "glm" && providerName != "kimi" && providerName != "minimax"
	case domain.ProtocolOpenAI:
		return providerName != "openai" && providerName != "openrouter" && providerName != "glm" && providerName != "kimi" && providerName != "minimax"
	default:
		return false
	}
}

func encodeProtocolStream(protocol domain.Protocol, resp domain.UnifiedChatResponse) ([]byte, map[string][]string, error) {
	return transform.RenderClientResponse(defaultSurfaceForProtocol(protocol, true), resp, true)
}

func defaultSurfaceForProtocol(protocol domain.Protocol, stream bool) transform.Surface {
	switch protocol {
	case domain.ProtocolClaude:
		return transform.Surface{Protocol: domain.ProtocolClaude, Operation: domain.ProtocolOperationClaudeMessages}
	case domain.ProtocolGemini:
		if stream {
			return transform.Surface{Protocol: domain.ProtocolGemini, Operation: domain.ProtocolOperationGeminiStreamGenerate}
		}
		return transform.Surface{Protocol: domain.ProtocolGemini, Operation: domain.ProtocolOperationGeminiGenerateContent}
	case domain.ProtocolCodex:
		return transform.Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses}
	default:
		return transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIChatCompletions}
	}
}

func enrichRequestHeaders(req *http.Request, extra []string) map[string]string {
	if req == nil {
		return nil
	}
	keys := append([]string{"Authorization", "x-api-key", "x-goog-api-key", "X-Session-ID", "X-Claude-Code-Session-Id"}, extra...)
	result := map[string]string{}
	for _, key := range keys {
		if value := strings.TrimSpace(req.Header.Get(key)); value != "" {
			result[key] = value
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func hasToolMessages(messages []domain.GatewayMessage) bool {
	for _, message := range messages {
		if len(message.ToolCalls) > 0 || strings.EqualFold(message.Role, "tool") {
			return true
		}
	}
	return false
}

func storeResponseSession(store ResponseSessionStore, req *http.Request, requestBody []byte, resp domain.UnifiedChatResponse) {
	if store == nil || resp.ID == "" {
		return
	}
	sessionID := ""
	if req != nil {
		sessionID = extractGatewaySessionID(req)
	}
	transcript := make([]domain.GatewayMessage, 0, 4)
	if previous, ok := extractPreviousResponseID(requestBody); ok {
		if prior, found := store.Get(previous); found {
			transcript = append(transcript, prior.Messages...)
		}
	}
	if unifiedReq, err := provider.DecodeOpenAIResponsesRequest(requestBody); err == nil {
		for _, message := range unifiedReq.Messages {
			transcript = append(transcript, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, InputItem: message.InputItem, Metadata: message.Metadata})
		}
	}
	transcript = append(transcript, domain.GatewayMessage{Role: resp.Message.Role, Parts: resp.Message.Parts, ToolCalls: resp.Message.ToolCalls, InputItem: resp.Message.InputItem, Metadata: resp.Message.Metadata})
	transcript = repairGatewayTranscript(transcript)
	store.Put(ResponseSession{ResponseID: resp.ID, SessionID: sessionID, Model: resp.Model, Messages: transcript, UpdatedAt: time.Now()})
}

func mergePreviousResponse(store ResponseSessionStore, gatewayReq domain.GatewayRequest) domain.GatewayRequest {
	if store == nil || gatewayReq.Session == nil || strings.TrimSpace(gatewayReq.Session.PreviousResponseID) == "" {
		return gatewayReq
	}
	previous, ok := store.Get(gatewayReq.Session.PreviousResponseID)
	if !ok {
		return gatewayReq
	}
	merged := gatewayReq
	history := append([]domain.GatewayMessage(nil), previous.Messages...)
	history = append(history, gatewayReq.Messages...)
	merged.Messages = history
	return merged
}

func preprocessGatewayRequest(store ResponseSessionStore, gatewayReq domain.GatewayRequest) (domain.GatewayRequest, error) {
	if gatewayReq.Protocol == domain.ProtocolOpenAI {
		gatewayReq = mergePreviousResponse(store, gatewayReq)
	}
	if gatewayReq.Protocol == domain.ProtocolGemini {
		gatewayReq = mergeGeminiCachedContent(store, gatewayReq)
		var err error
		gatewayReq, err = expandGeminiURLContext(gatewayReq)
		if err != nil {
			return gatewayReq, err
		}
	}
	gatewayReq = applyClaudeContextManagement(gatewayReq)
	return gatewayReq, nil
}

func mergeGeminiCachedContent(store ResponseSessionStore, gatewayReq domain.GatewayRequest) domain.GatewayRequest {
	if store == nil {
		return gatewayReq
	}
	cacheName := cachedContentNameFromMetadata(gatewayReq.Metadata)
	if strings.TrimSpace(cacheName) == "" {
		return gatewayReq
	}
	previous, ok := store.Get(cacheName)
	if !ok {
		return gatewayReq
	}
	merged := gatewayReq
	history := append([]domain.GatewayMessage(nil), previous.Messages...)
	history = append(history, gatewayReq.Messages...)
	merged.Messages = history
	if merged.Metadata != nil {
		delete(merged.Metadata, "cachedContent")
		delete(merged.Metadata, "cached_content")
		if len(merged.Metadata) == 0 {
			merged.Metadata = nil
		}
	}
	return merged
}

var urlContextPattern = regexp.MustCompile(`https?://[^\s<>"')]+`)
var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)

func expandGeminiURLContext(gatewayReq domain.GatewayRequest) (domain.GatewayRequest, error) {
	if !requestHasGeminiURLContextTool(gatewayReq.Tools) {
		return gatewayReq, nil
	}
	urls := collectRequestURLs(gatewayReq.Messages)
	if len(urls) == 0 {
		return gatewayReq, nil
	}
	contextMessages := make([]domain.GatewayMessage, 0, len(urls))
	for _, targetURL := range urls {
		message, err := fetchURLContextMessage(targetURL)
		if err != nil {
			return gatewayReq, err
		}
		if len(message.Parts) == 0 {
			continue
		}
		contextMessages = append(contextMessages, message)
	}
	if len(contextMessages) == 0 {
		return gatewayReq, nil
	}
	expanded := gatewayReq
	expanded.Messages = append(contextMessages, expanded.Messages...)
	expanded.Tools = stripGeminiURLContextTools(expanded.Tools)
	return expanded, nil
}

func requestHasGeminiURLContextTool(tools []json.RawMessage) bool {
	for _, raw := range tools {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if payload["urlContext"] != nil || payload["url_context"] != nil {
			return true
		}
	}
	return false
}

func stripGeminiURLContextTools(tools []json.RawMessage) []json.RawMessage {
	if len(tools) == 0 {
		return nil
	}
	filtered := make([]json.RawMessage, 0, len(tools))
	for _, raw := range tools {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err != nil {
			filtered = append(filtered, raw)
			continue
		}
		if payload["urlContext"] != nil || payload["url_context"] != nil {
			continue
		}
		filtered = append(filtered, raw)
	}
	return filtered
}

func collectRequestURLs(messages []domain.GatewayMessage) []string {
	seen := map[string]struct{}{}
	urls := make([]string, 0)
	appendURL := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		urls = append(urls, value)
	}
	for _, message := range messages {
		for _, part := range message.Parts {
			for _, match := range urlContextPattern.FindAllString(part.Text, -1) {
				appendURL(match)
			}
			for _, key := range []string{"url", "file_uri"} {
				if raw, ok := part.Metadata[key]; ok {
					var value string
					if err := json.Unmarshal(raw, &value); err == nil {
						appendURL(value)
					}
				}
			}
		}
	}
	return urls
}

func fetchURLContextMessage(targetURL string) (domain.GatewayMessage, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return domain.GatewayMessage{}, fmt.Errorf("构建 URL Context 请求失败: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return domain.GatewayMessage{}, fmt.Errorf("抓取 URL Context 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.GatewayMessage{}, fmt.Errorf("抓取 URL Context 失败: upstream returned %d", resp.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "image/") {
		return domain.GatewayMessage{
			Role: "user",
			Parts: []domain.UnifiedPart{
				{Type: "text", Text: "Referenced URL context"},
				{Type: "image", Metadata: map[string]json.RawMessage{"url": marshalGatewayRawString(targetURL), "mime_type": marshalGatewayRawString(strings.Split(contentType, ";")[0])}},
			},
		}, nil
	}
	if strings.Contains(contentType, "pdf") {
		return domain.GatewayMessage{
			Role: "user",
			Parts: []domain.UnifiedPart{
				{Type: "text", Text: "Referenced URL context"},
				{Type: "document", Metadata: map[string]json.RawMessage{"url": marshalGatewayRawString(targetURL), "mime_type": marshalGatewayRawString("application/pdf")}},
			},
		}, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return domain.GatewayMessage{}, fmt.Errorf("读取 URL Context 失败: %w", err)
	}
	text := sanitizeFetchedURLText(string(body), contentType)
	if strings.TrimSpace(text) == "" {
		return domain.GatewayMessage{}, nil
	}
	return domain.GatewayMessage{
		Role: "system",
		Parts: []domain.UnifiedPart{{Type: "text", Text: "URL context from " + targetURL + ":\n" + text}},
	}, nil
}

func sanitizeFetchedURLText(body string, contentType string) string {
	if strings.Contains(contentType, "html") {
		replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n", "</div>", "\n", "</li>", "\n")
		body = replacer.Replace(body)
		body = htmlTagPattern.ReplaceAllString(body, " ")
	}
	body = strings.ReplaceAll(body, "\r", "")
	lines := strings.Split(body, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	body = strings.Join(filtered, "\n")
	if len(body) > 12000 {
		body = body[:12000]
	}
	return body
}

func marshalGatewayRawString(value string) json.RawMessage {
	body, _ := json.Marshal(value)
	return body
}

func applyClaudeContextManagement(gatewayReq domain.GatewayRequest) domain.GatewayRequest {
	raw := gatewayReq.Metadata["context_management"]
	if len(raw) == 0 {
		return gatewayReq
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return gatewayReq
	}
	edited := false
	if clear, ok := decodeContextManagementBoolean(payload, "clear_function_results", "clear_tool_uses"); ok && clear {
		gatewayReq.Messages = clearHistoricalToolUses(gatewayReq.Messages)
		edited = true
	}
	if clear, ok := decodeContextManagementBoolean(payload, "clear_thinking"); ok && clear {
		gatewayReq.Messages = clearHistoricalThinking(gatewayReq.Messages)
		edited = true
	}
	if edited {
		gatewayReq.Messages = repairGatewayTranscript(gatewayReq.Messages)
	}
	return gatewayReq
}

func decodeContextManagementBoolean(payload map[string]json.RawMessage, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		var flag bool
		if err := json.Unmarshal(value, &flag); err == nil {
			return flag, true
		}
	}
	return false, false
}

func clearHistoricalToolUses(messages []domain.GatewayMessage) []domain.GatewayMessage {
	preserveFrom := findTrailingPendingToolExchangeStart(messages)
	cleared := make([]domain.GatewayMessage, 0, len(messages))
	for index, message := range messages {
		if preserveFrom >= 0 && index >= preserveFrom {
			cleared = append(cleared, message)
			continue
		}
		if strings.EqualFold(message.Role, "tool") {
			continue
		}
		next := message
		next.ToolCalls = nil
		if len(next.Parts) > 0 {
			filteredParts := make([]domain.UnifiedPart, 0, len(next.Parts))
			for _, part := range next.Parts {
				switch part.Type {
				case "tool_result", "function_response", "computer_call", "computer_call_output", "mcp_call", "mcp_list_tools", "mcp_approval_request", "custom_tool_call", "code_interpreter_call", "image_generation_call", "local_shell_call", "local_shell_call_output", "shell_call_output", "apply_patch_call", "apply_patch_call_output":
					continue
				default:
					filteredParts = append(filteredParts, part)
				}
			}
			next.Parts = filteredParts
		}
		if len(next.Parts) == 0 && len(next.ToolCalls) == 0 && strings.EqualFold(next.Role, "assistant") {
			continue
		}
		cleared = append(cleared, next)
	}
	return cleared
}

func findTrailingPendingToolExchangeStart(messages []domain.GatewayMessage) int {
	if len(messages) == 0 {
		return -1
	}
	last := messages[len(messages)-1]
	if !strings.EqualFold(last.Role, "tool") {
		return -1
	}
	start := len(messages) - 1
	for start > 0 {
		prev := messages[start-1]
		if strings.EqualFold(prev.Role, "tool") || (strings.EqualFold(prev.Role, "assistant") && len(prev.ToolCalls) > 0) {
			start--
			continue
		}
		break
	}
	return start
}

func clearHistoricalThinking(messages []domain.GatewayMessage) []domain.GatewayMessage {
	cleared := make([]domain.GatewayMessage, 0, len(messages))
	for _, message := range messages {
		next := message
		if len(next.Parts) > 0 {
			filteredParts := make([]domain.UnifiedPart, 0, len(next.Parts))
			for _, part := range next.Parts {
				if part.Type == "reasoning" {
					continue
				}
				filteredParts = append(filteredParts, part)
			}
			next.Parts = filteredParts
		}
		cleared = append(cleared, next)
	}
	return cleared
}

func extractPreviousResponseID(body []byte) (string, bool) {
	var payload struct {
		PreviousResponseID string `json:"previous_response_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || strings.TrimSpace(payload.PreviousResponseID) == "" {
		return "", false
	}
	return payload.PreviousResponseID, true
}

func sortedHeaderKeys(headers map[string]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func repairGatewayTranscript(messages []domain.GatewayMessage) []domain.GatewayMessage {
	if len(messages) <= 1 {
		return messages
	}
	repaired := make([]domain.GatewayMessage, 0, len(messages))
	for _, message := range messages {
		if len(repaired) == 0 {
			repaired = append(repaired, message)
			continue
		}
		last := &repaired[len(repaired)-1]
		if strings.EqualFold(last.Role, message.Role) && strings.EqualFold(message.Role, "assistant") && len(last.Parts) == 0 && len(message.Parts) == 0 {
			last.ToolCalls = append(last.ToolCalls, message.ToolCalls...)
			continue
		}
		if strings.EqualFold(last.Role, message.Role) && strings.EqualFold(message.Role, "tool") && sameToolCallID(last.Metadata, message.Metadata) {
			last.Parts = append(last.Parts, message.Parts...)
			continue
		}
		repaired = append(repaired, message)
	}
	return repaired
}

func sameToolCallID(left map[string]json.RawMessage, right map[string]json.RawMessage) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	return string(left["tool_call_id"]) != "" && string(left["tool_call_id"]) == string(right["tool_call_id"])
}
