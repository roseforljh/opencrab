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
	"sort"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"

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
		writeResponsesGatewayResult(deps, w, req, body, result, startedAt)
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
		if deps.CountClaudeTokens == nil || deps.CopyProxy == nil || deps.VerifyAPIKey == nil {
			renderClaudeError(w, http.StatusNotImplemented, "count tokens handler not configured")
			return
		}
		rawKey := extractGatewayAPIKey(req)
		if rawKey == "" {
			renderClaudeError(w, http.StatusUnauthorized, "缺少 API Key")
			return
		}
		allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
		if err != nil {
			renderClaudeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		if !allowed {
			renderClaudeError(w, http.StatusUnauthorized, "API Key 无效或已禁用")
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
	if deps.VerifyAPIKey == nil {
		return nil, nil, "", startedAt, fmt.Errorf("api key verifier not configured")
	}

	rawKey := extractGatewayAPIKey(req)
	if rawKey == "" {
		return nil, nil, "", startedAt, fmt.Errorf("缺少 API Key")
	}
	allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
	if err != nil {
		return nil, nil, "", startedAt, err
	}
	if !allowed {
		return nil, nil, "", startedAt, fmt.Errorf("API Key 无效或已禁用")
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
	if deps.GetGatewayRuntimeSettings != nil {
		settings, settingsErr := deps.GetGatewayRuntimeSettings(req.Context())
		if settingsErr != nil {
			return body, nil, protocol, startedAt, settingsErr
		}
		gatewayReq.AffinityKey = extractSessionAffinityKey(req, gatewayReq, settings)
		gatewayReq.RuntimeSettings = &settings
	}
	if protocol == domain.ProtocolOpenAI {
		gatewayReq = mergePreviousResponse(deps.ResponseSessions, gatewayReq)
	}
	result, err := deps.ExecuteGateway(req.Context(), middleware.GetReqID(req.Context()), gatewayReq)
	if err != nil {
		return body, nil, protocol, startedAt, err
	}
	return body, result, protocol, startedAt, nil
}

func acceptGatewayRequest(deps Dependencies, req *http.Request, decode gatewayDecoder) (domain.GatewayAcceptedResponse, error) {
	if deps.CreateGatewayJob == nil || deps.GetGatewayJobByRequestID == nil || deps.VerifyAPIKey == nil {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("gateway admission handler not configured")
	}
	rawKey := extractGatewayAPIKey(req)
	if rawKey == "" {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("缺少 API Key")
	}
	allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
	if err != nil {
		return domain.GatewayAcceptedResponse{}, err
	}
	if !allowed {
		return domain.GatewayAcceptedResponse{}, fmt.Errorf("API Key 无效或已禁用")
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
		SessionID:        strings.TrimSpace(req.Header.Get("X-Session-ID")),
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
		if deps.GetGatewayJobByRequestID == nil || deps.VerifyAPIKey == nil {
			http.Error(w, "gateway request status handler not configured", http.StatusNotImplemented)
			return
		}
		rawKey := extractGatewayAPIKey(req)
		if rawKey == "" {
			http.Error(w, "缺少 API Key", http.StatusUnauthorized)
			return
		}
		allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
		if err != nil || !allowed {
			http.Error(w, "API Key 无效或已禁用", http.StatusUnauthorized)
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
		if deps.GetGatewayJobByRequestID == nil || deps.VerifyAPIKey == nil {
			http.Error(w, "gateway request events handler not configured", http.StatusNotImplemented)
			return
		}
		rawKey := extractGatewayAPIKey(req)
		if rawKey == "" {
			http.Error(w, "缺少 API Key", http.StatusUnauthorized)
			return
		}
		allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
		if err != nil || !allowed {
			http.Error(w, "API Key 无效或已禁用", http.StatusUnauthorized)
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
		"x-opencrab-async":  {},
		"idempotency-key":   {},
		"x-requested-with":  {},
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
	unified, err := provider.DecodeOpenAIChatRequest(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return unifiedToGatewayRequest(unified, nil, nil), domain.ProtocolOpenAI, nil
}

func decodeOpenAIResponsesGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeOpenAIResponsesRequest(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	unified.Stream = false
	session, err := provider.DecodeOpenAIResponsesSession(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return unifiedToGatewayRequest(unified, enrichRequestHeaders(req, []string{"OpenAI-Beta", "X-Stainless-Helper-Method", "X-Stainless-Retry-Count", "X-Stainless-Timeout"}), session), domain.ProtocolOpenAI, nil
}

func decodeClaudeGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeClaudeChatRequest(body)
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	return unifiedToGatewayRequest(unified, enrichRequestHeaders(req, []string{"anthropic-version", "anthropic-beta", "anthropic-dangerous-direct-browser-access"}), nil), domain.ProtocolClaude, nil
}

func decodeGeminiGatewayRequest(body []byte, req *http.Request) (domain.GatewayRequest, domain.Protocol, error) {
	unified, err := provider.DecodeGeminiChatRequest(body, chi.URLParam(req, "model"))
	if err != nil {
		return domain.GatewayRequest{}, "", err
	}
	if strings.Contains(req.URL.Path, ":streamGenerateContent") {
		unified.Stream = true
	}
	return unifiedToGatewayRequest(unified, enrichRequestHeaders(req, nil), nil), domain.ProtocolGemini, nil
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
	}
	if settingsStore != nil {
		settings, settingsErr := settingsStore(context.Background())
		if settingsErr == nil {
			req.AffinityKey = extractSessionAffinityKey(httpReq, req, settings)
			req.RuntimeSettings = &settings
		}
	}
	if job.Protocol == domain.ProtocolOpenAI {
		req = mergePreviousResponse(store, req)
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
		messages = append(messages, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, Metadata: message.Metadata})
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
	if requestWantsStream(requestBody) && shouldSynthesizeProtocolStream(protocol, providerName) {
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
	resp := encodeGatewayResponseForProtocol(result.Response, protocol)
	if err := deps.CopyProxy(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logGatewayRequestSummary(deps, req, requestBody, resp.StatusCode, resp.Headers, resp.Body, startedAt, result.Metadata)
}

func writeResponsesGatewayResult(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, result *domain.ExecutionResult, startedAt time.Time) {
	if result == nil || result.Response == nil {
		http.Error(w, "empty gateway result", http.StatusBadGateway)
		return
	}
	resp := encodeGatewayResponseForProtocol(result.Response, domain.ProtocolOpenAI)
	providerName := normalizedHeaderProvider(resp.Headers)
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		if err := deps.CopyProxy(w, resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	streamRequested := requestWantsStream(requestBody)
	if deps.ResponseSessions != nil {
		storeResponseSession(deps.ResponseSessions, req, requestBody, unified)
	}
	encoded, headers, err := encodeResponsesProxyResponse(unified, streamRequested)
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

func encodeGatewayResponseForProtocol(resp *domain.ProxyResponse, protocol domain.Protocol) *domain.ProxyResponse {
	providerName := normalizedHeaderProvider(resp.Headers)
	if protocolMatchesProvider(protocol, providerName) {
		return resp
	}
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		return resp
	}
	encoded, err := encodeUnifiedByProtocol(protocol, unified)
	if err != nil {
		return resp
	}
	headers := cloneHeaderMap(resp.Headers)
	headers["Content-Type"] = []string{"application/json"}
	return &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: encoded}
}

func decodeUnifiedByProvider(providerName string, body []byte) (domain.UnifiedChatResponse, error) {
	switch providerName {
	case "claude":
		return provider.DecodeClaudeChatResponse(body)
	case "gemini":
		return provider.DecodeGeminiChatResponse(body)
	default:
		return provider.DecodeOpenAIChatResponse(body)
	}
}

func encodeUnifiedByProtocol(protocol domain.Protocol, resp domain.UnifiedChatResponse) ([]byte, error) {
	switch protocol {
	case domain.ProtocolClaude:
		return provider.EncodeClaudeChatResponse(resp)
	case domain.ProtocolGemini:
		return provider.EncodeGeminiChatResponse(resp)
	default:
		return provider.EncodeOpenAIChatResponse(resp)
	}
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
	modelName := extractModel(requestBody)
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
	modelName := extractModel(requestBody)
	metadata := execErr.Metadata
	if strings.TrimSpace(modelName) == "unknown-model" && metadata != nil && len(metadata.FallbackChain) > 0 {
		modelName = metadata.FallbackChain[0]
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
	case domain.ProtocolOpenAI:
		return providerName != "openai" && providerName != "openrouter" && providerName != "glm" && providerName != "kimi" && providerName != "minimax"
	default:
		return false
	}
}

func encodeProtocolStream(protocol domain.Protocol, resp domain.UnifiedChatResponse) ([]byte, map[string][]string, error) {
	switch protocol {
	case domain.ProtocolClaude:
		body, err := provider.EncodeClaudeChatStream(resp)
		if err != nil {
			return nil, nil, err
		}
		return body, map[string][]string{"Content-Type": {"text/event-stream"}, "Cache-Control": {"no-cache"}, "Connection": {"keep-alive"}}, nil
	default:
		body, err := provider.EncodeOpenAIChatStream(resp)
		if err != nil {
			return nil, nil, err
		}
		return body, map[string][]string{"Content-Type": {"text/event-stream"}, "Cache-Control": {"no-cache"}, "Connection": {"keep-alive"}}, nil
	}
}

func enrichRequestHeaders(req *http.Request, extra []string) map[string]string {
	if req == nil {
		return nil
	}
	keys := append([]string{"Authorization", "x-api-key", "x-goog-api-key", "X-Session-ID"}, extra...)
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
		sessionID = strings.TrimSpace(req.Header.Get("X-Session-ID"))
	}
	transcript := make([]domain.GatewayMessage, 0, 4)
	if previous, ok := extractPreviousResponseID(requestBody); ok {
		if prior, found := store.Get(previous); found {
			transcript = append(transcript, prior.Messages...)
		}
	}
	if unifiedReq, err := provider.DecodeOpenAIResponsesRequest(requestBody); err == nil {
		for _, message := range unifiedReq.Messages {
			transcript = append(transcript, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, Metadata: message.Metadata})
		}
	}
	transcript = append(transcript, domain.GatewayMessage{Role: resp.Message.Role, Parts: resp.Message.Parts, ToolCalls: resp.Message.ToolCalls, Metadata: resp.Message.Metadata})
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
