package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"
	"opencrab/internal/transform"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

var responsesUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type responsesWebSocketEnvelope struct {
	Type               string          `json:"type"`
	Response           json.RawMessage `json:"response"`
	ResponseID         string          `json:"response_id"`
	PreviousResponseID string          `json:"previous_response_id"`
	Model              string          `json:"model"`
	Input              json.RawMessage `json:"input"`
}

func HandleOpenAIResponsesWebSocket(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ExecuteGateway == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) || deps.ResponseSessions == nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("responses websocket handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolOpenAI)
			return
		}
		conn, err := responsesUpgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		lastResponseID := ""
		lastModel := ""
		for {
			_, message, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			payload, nextModel, nextPrev, buildErr := buildResponsesWebSocketPayload(message, lastModel, lastResponseID)
			if buildErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(buildErr, http.StatusBadRequest))
				continue
			}
			gatewayReq, protocol, decodeErr := decodeOpenAIResponsesGatewayRequest(payload, req)
			if decodeErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(decodeErr, http.StatusBadRequest))
				continue
			}
			if scopeErr := applyAPIKeyScopeToGatewayRequest(&gatewayReq, scope); scopeErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(scopeErr, gatewayErrorStatusCode(scopeErr)))
				continue
			}
			if responsesGenerateDisabled(payload) {
				emptyResponse := domain.UnifiedChatResponse{Protocol: domain.ProtocolOpenAI, ID: nextResponseID(lastResponseID), Model: gatewayReq.Model, Message: domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: ""}}}}
				storeResponseSession(deps.ResponseSessions, req, payload, emptyResponse)
				events, eventsErr := provider.BuildOpenAIResponsesEvents(emptyResponse)
				if eventsErr != nil {
					_ = conn.WriteJSON(openAIResponsesErrorEvent(eventsErr, http.StatusInternalServerError))
					continue
				}
				for _, event := range events {
					if err := conn.WriteJSON(event); err != nil {
						return
					}
				}
				lastModel = gatewayReq.Model
				lastResponseID = emptyResponse.ID
				continue
			}
			result, execErr := executeGatewayRequestDirect(req.Context(), deps, req, gatewayReq, protocol)
			if execErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(execErr, gatewayErrorStatusCode(execErr)))
				continue
			}
			if result == nil || result.Response == nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(fmt.Errorf("empty gateway result"), http.StatusBadGateway))
				continue
			}
			resp := encodeGatewayResponseForSurface(result.Response, transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIResponses})
			providerName := normalizedHeaderProvider(resp.Headers)
			unified, decodeRespErr := decodeUnifiedByProvider(providerName, resp.Body)
			if decodeRespErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(decodeRespErr, http.StatusBadGateway))
				continue
			}
			storeResponseSession(deps.ResponseSessions, req, payload, unified)
			events, eventsErr := provider.BuildOpenAIResponsesEvents(unified)
			if eventsErr != nil {
				_ = conn.WriteJSON(openAIResponsesErrorEvent(eventsErr, http.StatusInternalServerError))
				continue
			}
			for _, event := range events {
				if err := conn.WriteJSON(event); err != nil {
					return
				}
			}
			lastModel = nextModel
			if unified.Model != "" {
				lastModel = unified.Model
			}
			lastResponseID = unified.ID
			if lastResponseID == "" {
				lastResponseID = nextPrev
			}
		}
	}
}

func openAIResponsesErrorEvent(err error, statusCode int) map[string]any {
	message := "responses websocket error"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	}
	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"message": message,
			"type":    "invalid_request_error",
			"code":    statusCode,
		},
	}
}

func buildResponsesWebSocketPayload(frame []byte, lastModel string, lastResponseID string) ([]byte, string, string, error) {
	var envelope responsesWebSocketEnvelope
	if err := json.Unmarshal(frame, &envelope); err != nil {
		return nil, "", "", fmt.Errorf("解析 WebSocket 消息失败: %w", err)
	}
	switch strings.TrimSpace(envelope.Type) {
	case "response.create", "":
		payload := envelope.Response
		if len(payload) == 0 {
			payload = frame
		}
		model := lastModel
		if strings.TrimSpace(envelope.Model) != "" {
			model = envelope.Model
		}
		if model == "" {
			var raw map[string]json.RawMessage
			if json.Unmarshal(payload, &raw) == nil {
				_ = json.Unmarshal(raw["model"], &model)
			}
		}
		return payload, model, extractPreviousResponseIDValue(payload, lastResponseID), nil
	case "response.append":
		previous := strings.TrimSpace(envelope.ResponseID)
		if previous == "" {
			previous = strings.TrimSpace(envelope.PreviousResponseID)
		}
		if previous == "" {
			previous = strings.TrimSpace(lastResponseID)
		}
		model := strings.TrimSpace(envelope.Model)
		if model == "" {
			model = strings.TrimSpace(lastModel)
		}
		if model == "" {
			return nil, "", "", fmt.Errorf("response.append 缺少 model")
		}
		payload := map[string]any{"model": model, "previous_response_id": previous, "input": json.RawMessage(`[]`)}
		if len(envelope.Input) > 0 {
			payload["input"] = json.RawMessage(envelope.Input)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, "", "", fmt.Errorf("编码 response.append 失败: %w", err)
		}
		return encoded, model, previous, nil
	default:
		return nil, "", "", fmt.Errorf("暂不支持的 WebSocket 消息类型: %s", envelope.Type)
	}
}

func executeGatewayRequestDirect(ctx context.Context, deps Dependencies, req *http.Request, gatewayReq domain.GatewayRequest, protocol domain.Protocol) (*domain.ExecutionResult, error) {
	if deps.GetGatewayRuntimeSettings != nil {
		settings, err := deps.GetGatewayRuntimeSettings(ctx)
		if err != nil {
			return nil, err
		}
		gatewayReq.AffinityKey = extractSessionAffinityKey(req, gatewayReq, settings)
		gatewayReq.RuntimeSettings = &settings
	}
	if protocol == domain.ProtocolOpenAI {
		gatewayReq = mergePreviousResponse(deps.ResponseSessions, gatewayReq)
	}
	return deps.ExecuteGateway(ctx, middleware.GetReqID(req.Context()), gatewayReq)
}

func extractPreviousResponseIDValue(payload []byte, fallback string) string {
	if value, ok := extractPreviousResponseID(payload); ok {
		return value
	}
	return fallback
}

func responsesGenerateDisabled(payload []byte) bool {
	var body struct {
		Generate *bool `json:"generate"`
	}
	if err := json.Unmarshal(payload, &body); err != nil || body.Generate == nil {
		return false
	}
	return !*body.Generate
}

func nextResponseID(fallback string) string {
	if strings.TrimSpace(fallback) == "" {
		return fmt.Sprintf("resp_ws_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_next", fallback)
}
