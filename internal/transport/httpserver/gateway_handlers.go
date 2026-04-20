package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/observability"
	"opencrab/internal/provider"
	"opencrab/internal/transform"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type openAIModel struct {
	ID            string   `json:"id"`
	Object        string   `json:"object"`
	Created       int64    `json:"created"`
	OwnedBy       string   `json:"owned_by"`
	UpstreamModel string   `json:"upstream_model,omitempty"`
	RouteCount    int      `json:"route_count,omitempty"`
	Providers     []string `json:"providers,omitempty"`
	Channels      []string `json:"channel_names,omitempty"`
}

type routeVisibility struct {
	channels  map[string]struct{}
	providers map[string]struct{}
}

func listVisibleOpenAIModels(deps Dependencies, req *http.Request, scope domain.APIKeyScope) ([]openAIModel, error) {
	models, err := deps.ListModels(req.Context())
	if err != nil {
		return nil, err
	}

	channelProviders := map[string]string{}
	enabledChannels := map[string]bool{}
	if deps.ListChannels != nil {
		channels, channelErr := deps.ListChannels(req.Context())
		if channelErr != nil {
			return nil, channelErr
		}
		for _, channel := range channels {
			name := strings.TrimSpace(channel.Name)
			channelProviders[name] = domain.NormalizeProvider(channel.Provider)
			enabledChannels[name] = channel.Enabled
		}
	}

	visibleRoutesByAlias := map[string]*routeVisibility{}
	if deps.ListModelRoutes != nil {
		routes, routeErr := deps.ListModelRoutes(req.Context())
		if routeErr != nil {
			return nil, routeErr
		}
		for _, route := range routes {
			alias := strings.TrimSpace(route.ModelAlias)
			channelName := strings.TrimSpace(route.ChannelName)
			if alias == "" || channelName == "" {
				continue
			}
			if len(scope.ChannelNames) > 0 && !scopeListContains(scope.ChannelNames, channelName) {
				continue
			}
			if len(enabledChannels) > 0 && !enabledChannels[channelName] {
				continue
			}
			item := visibleRoutesByAlias[alias]
			if item == nil {
				item = &routeVisibility{channels: map[string]struct{}{}, providers: map[string]struct{}{}}
				visibleRoutesByAlias[alias] = item
			}
			item.channels[channelName] = struct{}{}
			providerName := channelProviders[channelName]
			if providerName == "" {
				providerName = "opencrab"
			}
			item.providers[providerName] = struct{}{}
		}
	}

	data := make([]openAIModel, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		alias := strings.TrimSpace(model.Alias)
		if alias == "" {
			continue
		}
		if len(scope.ModelAliases) > 0 && !scopeListContains(scope.ModelAliases, alias) {
			continue
		}
		visibility := visibleRoutesByAlias[alias]
		if deps.ListModelRoutes != nil && visibility == nil {
			continue
		}
		if _, exists := seen[alias]; exists {
			continue
		}
		seen[alias] = struct{}{}
		ownedBy := "opencrab"
		providers := make([]string, 0)
		channels := make([]string, 0)
		routeCount := 0
		if visibility != nil {
			for providerName := range visibility.providers {
				providers = append(providers, providerName)
			}
			for channelName := range visibility.channels {
				channels = append(channels, channelName)
			}
			sort.Strings(providers)
			sort.Strings(channels)
			routeCount = len(channels)
			if len(providers) == 1 {
				ownedBy = providers[0]
			}
		}
		data = append(data, openAIModel{
			ID:            alias,
			Object:        "model",
			Created:       0,
			OwnedBy:       ownedBy,
			UpstreamModel: strings.TrimSpace(model.UpstreamModel),
			RouteCount:    routeCount,
			Providers:     providers,
			Channels:      channels,
		})
	}

	return data, nil
}

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
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("models handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}

		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}

		data, err := listVisibleOpenAIModels(deps, req, scope)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusInternalServerError, false, false), domain.ProtocolOpenAI)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   data,
		})
	}
}

func HandleOpenAIModel(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ListModels == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("models handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}

		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}

		modelID := strings.TrimSpace(chi.URLParam(req, "model"))
		if modelID == "" {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("model 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}

		data, err := listVisibleOpenAIModels(deps, req, scope)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusInternalServerError, false, false), domain.ProtocolOpenAI)
			return
		}
		for _, item := range data {
			if item.ID == modelID {
				writeJSON(w, http.StatusOK, item)
				return
			}
		}
		renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("model 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
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

func HandleOpenAIResponseRetrieve(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ResponseSessions == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("responses retrieve handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		if _, _, err := resolveGatewayAPIKey(deps, req); err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		responseID := strings.TrimSpace(chi.URLParam(req, "responseID"))
		if responseID == "" {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		session, ok := deps.ResponseSessions.Get(responseID)
		if !ok {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		if len(session.ResponseBody) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(session.ResponseBody)
			return
		}
		response := storedOpenAIResponseFromSession(session)
		body, err := provider.EncodeOpenAIResponsesResponse(response)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func HandleOpenAIResponseInputItems(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ResponseSessions == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("responses input items handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		if _, _, err := resolveGatewayAPIKey(deps, req); err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		responseID := strings.TrimSpace(chi.URLParam(req, "responseID"))
		if responseID == "" {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		session, ok := deps.ResponseSessions.Get(responseID)
		if !ok {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		if len(session.InputItems) > 0 {
			var items any
			if err := json.Unmarshal(session.InputItems, &items); err == nil {
				writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": items})
				return
			}
		}
		items := storedOpenAIInputItemsFromSession(session)
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": items})
	}
}

func HandleOpenAIResponseDelete(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ResponseSessions == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("responses delete handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		if _, _, err := resolveGatewayAPIKey(deps, req); err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		responseID := strings.TrimSpace(chi.URLParam(req, "responseID"))
		if responseID == "" {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		if _, ok := deps.ResponseSessions.Get(responseID); !ok {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("response 不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		deps.ResponseSessions.Delete(responseID)
		writeJSON(w, http.StatusOK, map[string]any{"id": responseID, "object": "response.deleted", "deleted": true})
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
		logGatewayRequestSummary(deps, req, body, resp.StatusCode, resp.Headers, resp.Body, startedAt, nil, nil)
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
	observability.MarkRequestExecuteStart(req.Context())
	defer observability.MarkRequestExecuteEnd(req.Context())
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
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("gateway request status handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		rawKey, _, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolOpenAI)
			return
		}
		item, err := deps.GetGatewayJobByRequestID(req.Context(), chi.URLParam(req, "requestID"))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if isGatewayJobNotFound(err) {
				statusCode = http.StatusNotFound
			}
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, statusCode, false, false), domain.ProtocolOpenAI)
			return
		}
		if item.OwnerKeyHash != gatewayOwnerKeyHash(rawKey) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("请求不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
			return
		}
		writeJSON(w, http.StatusOK, buildGatewayJobStatusResponse(item))
	}
}

func HandleGatewayRequestEvents(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.GetGatewayJobByRequestID == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("gateway request events handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		rawKey, _, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolOpenAI)
			return
		}
		item, err := deps.GetGatewayJobByRequestID(req.Context(), chi.URLParam(req, "requestID"))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if isGatewayJobNotFound(err) {
				statusCode = http.StatusNotFound
			}
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, statusCode, false, false), domain.ProtocolOpenAI)
			return
		}
		if item.OwnerKeyHash != gatewayOwnerKeyHash(rawKey) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("请求不存在"), http.StatusNotFound, false, false), domain.ProtocolOpenAI)
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
	observability.MarkRequestWriteStart(req.Context())
	defer observability.MarkRequestWriteEnd(req.Context())
	if result.Stream != nil {
		if err := deps.CopyStream(w, result.Stream); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logGatewayRequestSummary(deps, req, requestBody, result.Stream.StatusCode, result.Stream.Headers, nil, startedAt, result.Metadata, nil)
		return
	}
	if result.Response == nil {
		renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("empty gateway result"), http.StatusBadGateway, false, false), protocol)
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
				usage := usageMetricsFromUnified(unified.Usage)
				logGatewayRequestSummary(deps, req, requestBody, proxyResp.StatusCode, proxyResp.Headers, proxyResp.Body, startedAt, result.Metadata, &usage)
				return
			}
		}
	}
	resp := encodeGatewayResponseForSurface(result.Response, defaultSurfaceForProtocol(protocol, false))
	if err := deps.CopyProxy(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var structuredUsage *usageMetrics
	if unified, err := decodeUnifiedByProvider(providerName, result.Response.Body); err == nil {
		usage := usageMetricsFromUnified(unified.Usage)
		structuredUsage = &usage
	}
	logGatewayRequestSummary(deps, req, requestBody, resp.StatusCode, resp.Headers, resp.Body, startedAt, result.Metadata, structuredUsage)
}

func writeResponsesGatewayResult(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, result *domain.ExecutionResult, startedAt time.Time, surface transform.Surface) {
	observability.MarkRequestWriteStart(req.Context())
	defer observability.MarkRequestWriteEnd(req.Context())
	if result != nil && result.Stream != nil {
		if !requestWantsStreamForProtocol(surface.Protocol, req, requestBody) {
			if handled := tryWriteNormalizedResponsesStream(deps, w, req, requestBody, result, startedAt, surface); handled {
				return
			}
		}
		if err := deps.CopyStream(w, result.Stream); err != nil {
			logGatewayWriteFailure(req, deps.Logger, "stream", err)
			return
		}
		logGatewayRequestSummary(deps, req, requestBody, result.Stream.StatusCode, result.Stream.Headers, nil, startedAt, result.Metadata, nil)
		return
	}
	if result == nil || result.Response == nil {
		renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("empty gateway result"), http.StatusBadGateway, false, false), surface.Protocol)
		return
	}
	resp := encodeGatewayResponseForSurface(result.Response, surface)
	providerName := normalizedHeaderProvider(resp.Headers)
	unified, err := decodeUnifiedByProvider(providerName, resp.Body)
	if err != nil {
		if err := deps.CopyProxy(w, resp); err != nil {
			logGatewayWriteFailure(req, deps.Logger, "proxy_passthrough", err)
		}
		return
	}
	streamRequested := requestWantsStreamForProtocol(surface.Protocol, req, requestBody)
	if deps.ResponseSessions != nil {
		storeResponseSession(deps.ResponseSessions, req, requestBody, unified)
	}
	encoded, headers, err := transform.RenderClientResponse(surface, unified, streamRequested)
	if err != nil {
		renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusInternalServerError, false, false), surface.Protocol)
		return
	}
	proxyResp := &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: encoded}
	if err := deps.CopyProxy(w, proxyResp); err != nil {
		logGatewayWriteFailure(req, deps.Logger, "rendered_proxy", err)
		return
	}
	usage := usageMetricsFromUnified(unified.Usage)
	logGatewayRequestSummary(deps, req, requestBody, proxyResp.StatusCode, proxyResp.Headers, proxyResp.Body, startedAt, result.Metadata, &usage)
}

func tryWriteNormalizedResponsesStream(deps Dependencies, w http.ResponseWriter, req *http.Request, requestBody []byte, result *domain.ExecutionResult, startedAt time.Time, surface transform.Surface) bool {
	if result == nil || result.Stream == nil {
		return false
	}
	providerName := normalizedHeaderProvider(result.Stream.Headers)
	if !isOpenAIResponsesSurfaceProvider(providerName) {
		return false
	}
	body, unified, err := readUnifiedFromResponsesStream(result.Stream.Body)
	if err != nil {
		logGatewayWriteFailure(req, deps.Logger, "responses_stream_decode", err)
		proxyResp := &domain.ProxyResponse{StatusCode: result.Stream.StatusCode, Headers: cloneHeaderMap(result.Stream.Headers), Body: body}
		if copyErr := deps.CopyProxy(w, proxyResp); copyErr != nil {
			logGatewayWriteFailure(req, deps.Logger, "responses_stream_passthrough", copyErr)
		}
		return true
	}
	if deps.ResponseSessions != nil {
		storeResponseSession(deps.ResponseSessions, req, requestBody, unified)
	}
	encoded, headers, err := transform.RenderClientResponse(surface, unified, true)
	if err != nil {
		logGatewayWriteFailure(req, deps.Logger, "responses_stream_render", err)
		proxyResp := &domain.ProxyResponse{StatusCode: result.Stream.StatusCode, Headers: cloneHeaderMap(result.Stream.Headers), Body: body}
		if copyErr := deps.CopyProxy(w, proxyResp); copyErr != nil {
			logGatewayWriteFailure(req, deps.Logger, "responses_stream_passthrough", copyErr)
		}
		return true
	}
	proxyResp := &domain.ProxyResponse{StatusCode: result.Stream.StatusCode, Headers: headers, Body: encoded}
	if err := deps.CopyProxy(w, proxyResp); err != nil {
		logGatewayWriteFailure(req, deps.Logger, "responses_stream_normalized", err)
		return true
	}
	usage := usageMetricsFromUnified(unified.Usage)
	logGatewayRequestSummary(deps, req, requestBody, proxyResp.StatusCode, proxyResp.Headers, proxyResp.Body, startedAt, result.Metadata, &usage)
	return true
}

func isOpenAIResponsesSurfaceProvider(providerName string) bool {
	providerName = domain.NormalizeProvider(providerName)
	switch providerName {
	case "openai", "openrouter", "glm", "kimi", "minimax":
		return true
	default:
		return false
	}
}

func readUnifiedFromResponsesStream(body io.ReadCloser) ([]byte, domain.UnifiedChatResponse, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, domain.UnifiedChatResponse{}, err
	}
	unified, err := decodeUnifiedFromResponsesStream(data)
	return data, unified, err
}

func decodeUnifiedFromResponsesStream(body []byte) (domain.UnifiedChatResponse, error) {
	blocks := strings.Split(string(body), "\n\n")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		payload := decodeResponsesStreamPayload(block)
		if strings.TrimSpace(payload) == "" || strings.TrimSpace(payload) == "[DONE]" {
			continue
		}
		unified, ok := tryDecodeResponsesCompletedPayload([]byte(payload))
		if ok {
			return unified, nil
		}
	}
	return domain.UnifiedChatResponse{}, fmt.Errorf("responses stream missing completed payload")
}

func decodeResponsesStreamPayload(block string) string {
	lines := strings.Split(block, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		parts = append(parts, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
	}
	return strings.Join(parts, "\n")
}

func tryDecodeResponsesCompletedPayload(payload []byte) (domain.UnifiedChatResponse, bool) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return domain.UnifiedChatResponse{}, false
	}
	if rawType, ok := envelope["type"]; ok {
		var eventType string
		if err := json.Unmarshal(rawType, &eventType); err == nil && strings.TrimSpace(eventType) == "response.completed" {
			if responseRaw, exists := envelope["response"]; exists {
				unified, err := provider.DecodeOpenAIResponsesResponse(responseRaw)
				if err == nil {
					return unified, true
				}
			}
		}
	}
	if rawObject, ok := envelope["object"]; ok {
		var objectType string
		if err := json.Unmarshal(rawObject, &objectType); err == nil && strings.TrimSpace(objectType) == "response" {
			unified, err := provider.DecodeOpenAIResponsesResponse(payload)
			if err == nil {
				return unified, true
			}
		}
	}
	return domain.UnifiedChatResponse{}, false
}

func logGatewayWriteFailure(req *http.Request, logger *slog.Logger, stage string, err error) {
	if logger == nil || err == nil {
		return
	}
	requestID := ""
	path := ""
	method := ""
	if req != nil {
		requestID = middleware.GetReqID(req.Context())
		if req.URL != nil {
			path = req.URL.Path
		}
		method = req.Method
	}
	logger.Error("gateway_response_write_failed",
		slog.String("stage", stage),
		slog.String("method", method),
		slog.String("path", path),
		slog.String("request_id", requestID),
		slog.String("error", err.Error()),
	)
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

func logGatewayRequestSummary(deps Dependencies, req *http.Request, requestBody []byte, statusCode int, headers map[string][]string, responseBody []byte, startedAt time.Time, metadata *domain.GatewayExecutionMetadata, structuredUsage *usageMetrics) {
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
		loggedResponseBody = truncateLogBody(string(responseBody))
	}
	if structuredUsage != nil {
		usage = *structuredUsage
	} else if len(responseBody) > 0 {
		usage = extractUsageMetrics(responseBody)
	}
	totalDurationMs := time.Since(startedAt).Milliseconds()
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
		"duration_ms":       totalDurationMs,
	}
	mergeGatewayExecutionMetadata(detailPayload, metadata)
	details := marshalLogDetails(detailPayload)
	_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
		RequestID:        middleware.GetReqID(req.Context()),
		Model:            modelName,
		Channel:          channelName,
		StatusCode:       statusCode,
		LatencyMs:        totalDurationMs,
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
	totalDurationMs := time.Since(startedAt).Milliseconds()
	detailPayload := map[string]any{
		"log_type":        "gateway_request",
		"request_path":    req.URL.Path,
		"model":           modelName,
		"response_status": statusCode,
		"error_message":   err.Error(),
		"test_mode":       false,
		"duration_ms":     totalDurationMs,
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
		LatencyMs:    totalDurationMs,
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
	payload["degraded_success"] = metadata.DegradedSuccess
	payload["attempted_routes"] = metadata.AttemptedRoutes
	payload["skips"] = metadata.Skips
}

func renderGatewayError(deps Dependencies, w http.ResponseWriter, err error) {
	_ = deps
	statusCode := gatewayErrorStatusCode(err)
	renderOpenAIError(w, statusCode, err.Error())
}

func renderGatewayErrorForProtocol(deps Dependencies, w http.ResponseWriter, err error, protocol domain.Protocol) {
	if protocol == domain.ProtocolClaude {
		renderClaudeError(w, gatewayErrorStatusCode(err), err.Error())
		return
	}
	renderGatewayError(deps, w, err)
}

func gatewayErrorStatusCode(err error) int {
	if execErr := domain.AsExecutionError(err); execErr != nil && execErr.StatusCode > 0 {
		return execErr.StatusCode
	}
	message := err.Error()
	if strings.Contains(message, "请求不存在") || strings.Contains(message, "model 不存在") || strings.Contains(message, "response 不存在") || strings.Contains(message, "cached content not found") {
		return http.StatusNotFound
	}
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

func renderOpenAIError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    statusCode,
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

func storedOpenAIResponseFromSession(session ResponseSession) domain.UnifiedChatResponse {
	message := domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: ""}}}
	for index := len(session.Messages) - 1; index >= 0; index-- {
		candidate := session.Messages[index]
		if strings.EqualFold(candidate.Role, "assistant") {
			message = domain.UnifiedMessage{Role: candidate.Role, Parts: candidate.Parts, ToolCalls: candidate.ToolCalls, InputItem: candidate.InputItem, Metadata: candidate.Metadata}
			break
		}
	}
	return domain.UnifiedChatResponse{
		Protocol: domain.ProtocolOpenAI,
		ID:       session.ResponseID,
		Model:    session.Model,
		Message:  message,
	}
}

func storedOpenAIInputItemsFromSession(session ResponseSession) []map[string]any {
	items := make([]map[string]any, 0, len(session.Messages))
	for _, message := range session.Messages {
		if len(message.InputItem) > 0 {
			var item map[string]any
			if err := json.Unmarshal(message.InputItem, &item); err == nil && len(item) > 0 {
				items = append(items, item)
				continue
			}
		}
		content := make([]map[string]any, 0, len(message.Parts))
		for _, part := range message.Parts {
			switch part.Type {
			case "text":
				textType := "input_text"
				if strings.EqualFold(message.Role, "assistant") {
					textType = "output_text"
				}
				content = append(content, map[string]any{"type": textType, "text": part.Text})
			case "reasoning":
				content = append(content, map[string]any{"type": "reasoning", "text": part.Text})
			}
		}
		if len(content) == 0 && len(message.ToolCalls) == 0 {
			continue
		}
		items = append(items, map[string]any{"type": "message", "role": message.Role, "content": content})
		for _, call := range message.ToolCalls {
			items = append(items, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": string(call.Arguments)})
		}
		if strings.EqualFold(message.Role, "tool") {
			toolCallID := extractStringRawValue(message.Metadata["tool_call_id"])
			output := ""
			if len(message.Parts) > 0 {
				output = message.Parts[0].Text
			}
			items = append(items, map[string]any{"type": "function_call_output", "call_id": toolCallID, "output": output})
		}
	}
	return items
}

func requestWantsStreamForProtocol(protocol domain.Protocol, req *http.Request, body []byte) bool {
	if protocol == domain.ProtocolGemini && req != nil && strings.Contains(req.URL.Path, ":streamGenerateContent") {
		return true
	}
	if requestWantsStream(body) {
		return true
	}
	if protocol != domain.ProtocolOpenAI && protocol != domain.ProtocolCodex {
		return false
	}
	if req == nil {
		return false
	}
	accept := strings.ToLower(strings.TrimSpace(req.Header.Get("Accept")))
	return strings.Contains(accept, "text/event-stream")
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
	encodedResp, _ := provider.EncodeOpenAIResponsesResponse(resp)
	store.Put(ResponseSession{
		ResponseID:   resp.ID,
		SessionID:    sessionID,
		Model:        resp.Model,
		Messages:     transcript,
		InputItems:   extractStoredResponsesInputItems(requestBody),
		ResponseBody: json.RawMessage(encodedResp),
		UpdatedAt:    time.Now(),
	})
}

func extractStoredResponsesInputItems(requestBody []byte) json.RawMessage {
	if len(requestBody) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(requestBody, &raw); err != nil {
		return nil
	}
	inputRaw, ok := raw["input"]
	if !ok || len(inputRaw) == 0 {
		return nil
	}
	var list []any
	if err := json.Unmarshal(inputRaw, &list); err == nil {
		return append(json.RawMessage(nil), inputRaw...)
	}
	var text string
	if err := json.Unmarshal(inputRaw, &text); err == nil {
		encoded, marshalErr := json.Marshal([]map[string]any{{"type": "message", "role": "user", "content": []map[string]any{{"type": "input_text", "text": text}}}})
		if marshalErr == nil {
			return encoded
		}
	}
	var single map[string]any
	if err := json.Unmarshal(inputRaw, &single); err == nil {
		encoded, marshalErr := json.Marshal([]map[string]any{single})
		if marshalErr == nil {
			return encoded
		}
	}
	return nil
}

func mergePreviousResponse(store ResponseSessionStore, gatewayReq domain.GatewayRequest) domain.GatewayRequest {
	if store == nil || gatewayReq.Session == nil || strings.TrimSpace(gatewayReq.Session.PreviousResponseID) == "" {
		return gatewayReq
	}
	previous, ok := store.Get(gatewayReq.Session.PreviousResponseID)
	if !ok {
		return gatewayReq
	}
	if gatewayReq.Operation == domain.ProtocolOperationOpenAIResponses || gatewayReq.Operation == domain.ProtocolOperationOpenAIRealtime {
		// Native Responses continuation is upstream-authoritative. Replaying locally stored
		// transcript here would duplicate continuation state when previous_response_id is
		// also forwarded upstream. Some clients resend the full transcript anyway, so trim
		// any duplicated prefix before forwarding the incremental tail upstream.
		trimmed := trimDuplicatedContinuationPrefix(previous.Messages, gatewayReq.Messages)
		trimmed = collapseNativeContinuationMessages(trimmed)
		if len(trimmed) == len(gatewayReq.Messages) {
			return gatewayReq
		}
		normalized := gatewayReq
		normalized.Messages = trimmed
		return normalized
	}
	merged := gatewayReq
	history := append([]domain.GatewayMessage(nil), previous.Messages...)
	history = append(history, gatewayReq.Messages...)
	merged.Messages = history
	return merged
}

func trimDuplicatedContinuationPrefix(previous []domain.GatewayMessage, current []domain.GatewayMessage) []domain.GatewayMessage {
	if len(previous) == 0 || len(current) == 0 {
		return current
	}
	prefix := 0
	for prefix < len(previous) && prefix < len(current) {
		if !reflect.DeepEqual(previous[prefix], current[prefix]) {
			break
		}
		prefix++
	}
	if prefix == 0 || prefix >= len(current) {
		return current
	}
	trimmed := append([]domain.GatewayMessage(nil), current[prefix:]...)
	return trimmed
}

func collapseNativeContinuationMessages(messages []domain.GatewayMessage) []domain.GatewayMessage {
	if len(messages) <= 1 {
		return messages
	}
	leadingSystems := 0
	for leadingSystems < len(messages) && strings.EqualFold(messages[leadingSystems].Role, "system") {
		leadingSystems++
	}
	if preserveFrom := findTrailingPendingToolExchangeStart(messages); preserveFrom >= 0 {
		if preserveFrom < leadingSystems {
			preserveFrom = leadingSystems
		}
		collapsed := make([]domain.GatewayMessage, 0, leadingSystems+len(messages[preserveFrom:]))
		collapsed = append(collapsed, messages[:leadingSystems]...)
		collapsed = append(collapsed, messages[preserveFrom:]...)
		if len(collapsed) > 0 {
			return collapsed
		}
	}
	lastAssistant := -1
	for i := leadingSystems; i < len(messages); i++ {
		if strings.EqualFold(messages[i].Role, "assistant") {
			lastAssistant = i
		}
	}
	if lastAssistant < 0 || lastAssistant >= len(messages)-1 {
		return messages
	}
	collapsed := make([]domain.GatewayMessage, 0, leadingSystems+len(messages[lastAssistant+1:]))
	collapsed = append(collapsed, messages[:leadingSystems]...)
	collapsed = append(collapsed, messages[lastAssistant+1:]...)
	if len(collapsed) == 0 {
		return messages
	}
	return collapsed
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
		Role:  "system",
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
