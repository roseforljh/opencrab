package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"
	"opencrab/internal/transform"
)

type realtimeWebSocketEnvelope struct {
	Type     string          `json:"type"`
	Session  json.RawMessage `json:"session"`
	Item     json.RawMessage `json:"item"`
	Response json.RawMessage `json:"response"`
}

type realtimeConnectionState struct {
	SessionID              string
	DefaultModel           string
	SessionDefaults        map[string]json.RawMessage
	PendingItems           []json.RawMessage
	LastResponseID         string
	LastConversationItemID string
}

func HandleOpenAIRealtime(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			http.Error(w, err.Error(), gatewayErrorStatusCode(err))
			return
		}
		if maybeProxyOpenAIRealtimeWebSocket(deps, w, req, scope) {
			return
		}
		if deps.ExecuteGateway == nil || deps.ResponseSessions == nil {
			http.Error(w, "realtime websocket handler not configured", http.StatusNotImplemented)
			return
		}

		conn, err := responsesUpgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		state := newRealtimeConnectionState(strings.TrimSpace(req.URL.Query().Get("model")))
		if err := conn.WriteJSON(map[string]any{"type": "session.created", "session": state.sessionObject()}); err != nil {
			return
		}

		for {
			_, frame, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}

			var envelope realtimeWebSocketEnvelope
			if err := json.Unmarshal(frame, &envelope); err != nil {
				_ = conn.WriteJSON(realtimeErrorEvent(fmt.Errorf("解析 realtime 消息失败: %w", err)))
				continue
			}

			switch strings.TrimSpace(envelope.Type) {
			case "session.update":
				if err := state.applySessionUpdate(envelope.Session); err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}
				if err := conn.WriteJSON(map[string]any{"type": "session.updated", "session": state.sessionObject()}); err != nil {
					return
				}
			case "conversation.item.create":
				item, previousItemID, err := state.addConversationItem(envelope.Item)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}
				if err := conn.WriteJSON(buildRealtimeConversationEvent("conversation.item.added", previousItemID, item)); err != nil {
					return
				}
				if err := conn.WriteJSON(buildRealtimeConversationEvent("conversation.item.done", previousItemID, item)); err != nil {
					return
				}
			case "response.create", "":
				payload, err := state.buildResponsePayload(envelope.Response)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}

				gatewayReq, protocol, err := decodeOpenAIResponsesGatewayRequest(payload, req)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}
				gatewayReq.Operation = domain.ProtocolOperationOpenAIRealtime
				if err := applyAPIKeyScopeToGatewayRequest(&gatewayReq, scope); err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}

				result, err := executeGatewayRequestDirect(req.Context(), deps, req, gatewayReq, protocol)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}
				if result == nil || result.Response == nil {
					_ = conn.WriteJSON(realtimeErrorEvent(fmt.Errorf("empty gateway result")))
					continue
				}

				resp := encodeGatewayResponseForSurface(result.Response, transform.Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIResponses})
				providerName := normalizedHeaderProvider(resp.Headers)
				unified, err := decodeUnifiedByProvider(providerName, resp.Body)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}

				storeResponseSession(deps.ResponseSessions, req, payload, unified)
				events, lastItemID, err := provider.BuildOpenAIRealtimeEvents(unified, state.LastConversationItemID)
				if err != nil {
					_ = conn.WriteJSON(realtimeErrorEvent(err))
					continue
				}
				for _, event := range events {
					if err := conn.WriteJSON(event); err != nil {
						return
					}
				}

				state.PendingItems = nil
				state.LastConversationItemID = lastItemID
				state.LastResponseID = strings.TrimSpace(unified.ID)
			default:
				_ = conn.WriteJSON(realtimeErrorEvent(fmt.Errorf("暂不支持的 realtime 消息类型: %s", envelope.Type)))
			}
		}
	}
}

func newRealtimeConnectionState(model string) *realtimeConnectionState {
	state := &realtimeConnectionState{
		SessionID:       fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		DefaultModel:    strings.TrimSpace(model),
		SessionDefaults: map[string]json.RawMessage{},
	}
	if state.DefaultModel != "" {
		state.SessionDefaults["model"] = marshalRawString(state.DefaultModel)
	}
	state.SessionDefaults["output_modalities"] = json.RawMessage(`["text"]`)
	return state
}

func (s *realtimeConnectionState) sessionObject() map[string]any {
	session := map[string]any{
		"id":                s.SessionID,
		"object":            "realtime.session",
		"output_modalities": []string{"text"},
	}
	for key, raw := range s.SessionDefaults {
		var value any
		if err := json.Unmarshal(raw, &value); err == nil {
			session[key] = value
		}
	}
	if strings.TrimSpace(s.DefaultModel) != "" {
		session["model"] = s.DefaultModel
	}
	return session
}

func (s *realtimeConnectionState) applySessionUpdate(raw json.RawMessage) error {
	update, err := decodeRawMap(raw)
	if err != nil {
		return fmt.Errorf("session.update 格式不正确: %w", err)
	}
	if err := validateRealtimeOutputModalities(update); err != nil {
		return err
	}
	for key, value := range update {
		s.SessionDefaults[key] = append(json.RawMessage(nil), value...)
	}
	if model := strings.TrimSpace(decodeRawStringValue(update["model"])); model != "" {
		s.DefaultModel = model
	}
	return nil
}

func (s *realtimeConnectionState) addConversationItem(raw json.RawMessage) (map[string]any, string, error) {
	item, err := decodeRawMap(raw)
	if err != nil {
		return nil, "", fmt.Errorf("conversation.item.create 缺少 item")
	}
	previousItemID := s.LastConversationItemID
	if strings.TrimSpace(decodeRawStringValue(item["id"])) == "" {
		item["id"] = marshalRawString(fmt.Sprintf("item_%d", time.Now().UnixNano()))
	}
	if strings.TrimSpace(decodeRawStringValue(item["type"])) == "" {
		item["type"] = marshalRawString("message")
	}
	if strings.EqualFold(strings.TrimSpace(decodeRawStringValue(item["type"])), "message") && strings.TrimSpace(decodeRawStringValue(item["role"])) == "" {
		item["role"] = marshalRawString("user")
	}
	if _, ok := item["status"]; !ok {
		item["status"] = marshalRawString("completed")
	}

	encoded, err := json.Marshal(item)
	if err != nil {
		return nil, "", fmt.Errorf("编码 conversation item 失败: %w", err)
	}
	s.PendingItems = append(s.PendingItems, encoded)
	s.LastConversationItemID = strings.TrimSpace(decodeRawStringValue(item["id"]))

	normalized := map[string]any{}
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil, "", fmt.Errorf("解析 conversation item 失败: %w", err)
	}
	if _, ok := normalized["object"]; !ok {
		normalized["object"] = "realtime.item"
	}
	return normalized, previousItemID, nil
}

func (s *realtimeConnectionState) buildResponsePayload(raw json.RawMessage) ([]byte, error) {
	response, err := decodeOptionalRawMap(raw)
	if err != nil {
		return nil, fmt.Errorf("response.create 格式不正确: %w", err)
	}
	if err := validateRealtimeOutputModalities(response); err != nil {
		return nil, err
	}

	model := strings.TrimSpace(decodeRawStringValue(response["model"]))
	if model == "" {
		model = strings.TrimSpace(decodeRawStringValue(s.SessionDefaults["model"]))
	}
	if model == "" {
		model = s.DefaultModel
	}
	if model == "" {
		return nil, fmt.Errorf("realtime 请求缺少 model")
	}

	payload := map[string]json.RawMessage{
		"model": marshalRawString(model),
	}
	for _, key := range []string{"instructions", "tools", "reasoning", "include", "store", "text", "metadata", "parallel_tool_calls", "tool_choice", "temperature", "max_output_tokens"} {
		if value, ok := s.SessionDefaults[key]; ok {
			payload[key] = append(json.RawMessage(nil), value...)
		}
		if value, ok := response[key]; ok {
			payload[key] = append(json.RawMessage(nil), value...)
		}
	}

	if inputRaw, ok := response["input"]; ok && len(inputRaw) > 0 {
		payload["input"] = append(json.RawMessage(nil), inputRaw...)
	} else {
		encodedInput, err := marshalRawArray(s.PendingItems)
		if err != nil {
			return nil, err
		}
		payload["input"] = encodedInput
	}

	if previous := strings.TrimSpace(decodeRawStringValue(response["previous_response_id"])); previous != "" {
		payload["previous_response_id"] = marshalRawString(previous)
	} else if strings.TrimSpace(s.LastResponseID) != "" {
		payload["previous_response_id"] = marshalRawString(s.LastResponseID)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码 realtime 请求失败: %w", err)
	}
	return encoded, nil
}

func buildRealtimeConversationEvent(eventType string, previousItemID string, item map[string]any) map[string]any {
	event := map[string]any{
		"type": eventType,
		"item": item,
	}
	if strings.TrimSpace(previousItemID) != "" {
		event["previous_item_id"] = previousItemID
	}
	return event
}

func realtimeErrorEvent(err error) map[string]any {
	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "invalid_request_error",
			"message": err.Error(),
		},
	}
}

func validateRealtimeOutputModalities(raw map[string]json.RawMessage) error {
	for _, key := range []string{"output_modalities", "modalities"} {
		value, ok := raw[key]
		if !ok || len(value) == 0 {
			continue
		}
		var modalities []string
		if err := json.Unmarshal(value, &modalities); err != nil {
			return fmt.Errorf("%s 格式不正确", key)
		}
		for _, modality := range modalities {
			if strings.TrimSpace(strings.ToLower(modality)) != "text" {
				return fmt.Errorf("当前 realtime 仅支持 text modality")
			}
		}
	}
	return nil
}

func decodeRawMap(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing object")
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func decodeOptionalRawMap(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	return decodeRawMap(raw)
}

func marshalRawString(value string) json.RawMessage {
	body, _ := json.Marshal(value)
	return body
}

func marshalRawArray(items []json.RawMessage) (json.RawMessage, error) {
	if len(items) == 0 {
		return json.RawMessage(`[]`), nil
	}
	body := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		body = append(body, append(json.RawMessage(nil), item...))
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("编码 realtime input 失败: %w", err)
	}
	return encoded, nil
}

func decodeRawStringValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}
