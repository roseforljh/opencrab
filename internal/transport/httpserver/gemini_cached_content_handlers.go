package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/provider"

	"github.com/go-chi/chi/v5"
)

func HandleGeminiCachedContentCreate(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ResponseSessions == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("cached content handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolGemini)
			return
		}
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolGemini)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("读取请求体失败"), http.StatusBadRequest, false, false), domain.ProtocolGemini)
			return
		}
		unified, err := provider.DecodeGeminiChatRequest(body, "")
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusBadRequest, false, false), domain.ProtocolGemini)
			return
		}
		if deps.SelectDirectRoute != nil && deps.ForwardGeminiCachedContentCreate != nil {
			route, routeErr := deps.SelectDirectRoute(req.Context(), unified.Model, "gemini", &scope)
			if routeErr == nil {
				resp, forwardErr := deps.ForwardGeminiCachedContentCreate(req.Context(), route, body)
				if forwardErr == nil {
					if name, ok := decodeGeminiCachedContentName(resp.Body); ok {
						deps.ResponseSessions.Put(ResponseSession{
							ResponseID: name,
							SessionID:  "gemini-cache",
							Model:      unified.Model,
							Messages:   transcriptFromUnifiedMessages(unified.Messages),
							ResponseBody: json.RawMessage(append([]byte(nil), resp.Body...)),
							UpdatedAt:  time.Now(),
						})
					}
					if deps.CopyProxy != nil {
						_ = deps.CopyProxy(w, resp)
						return
					}
				}
			}
		}
		cacheName := fmt.Sprintf("cachedContents/opencrab-%d", time.Now().UnixNano())
		now := time.Now()
		deps.ResponseSessions.Put(ResponseSession{
			ResponseID: cacheName,
			SessionID:  "gemini-cache",
			Model:      unified.Model,
			Messages:   transcriptFromUnifiedMessages(unified.Messages),
			ResponseBody: mustMarshalJSON(buildGeminiCachedContentResponse(cacheName, unified.Model, now)),
			UpdatedAt:  now,
		})
		writeJSON(w, http.StatusOK, buildGeminiCachedContentResponse(cacheName, unified.Model, now))
	}
}

func HandleGeminiCachedContentGet(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.ResponseSessions == nil || (deps.ResolveAPIKey == nil && deps.VerifyAPIKey == nil) {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("cached content handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolGemini)
			return
		}
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolGemini)
			return
		}
		cacheName := "cachedContents/" + strings.TrimSpace(chi.URLParam(req, "cacheID"))
		if deps.SelectDirectRoute != nil && deps.ForwardGeminiCachedContentGet != nil {
			model := strings.TrimSpace(req.URL.Query().Get("model"))
			if model != "" {
				route, routeErr := deps.SelectDirectRoute(req.Context(), model, "gemini", &scope)
				if routeErr == nil {
					resp, forwardErr := deps.ForwardGeminiCachedContentGet(req.Context(), route, cacheName)
					if forwardErr == nil {
						if deps.CopyProxy != nil {
							_ = deps.CopyProxy(w, resp)
							return
						}
					}
				}
			}
		}
		item, ok := deps.ResponseSessions.Get(cacheName)
		if !ok {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("cached content not found"), http.StatusNotFound, false, false), domain.ProtocolGemini)
			return
		}
		if len(item.ResponseBody) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(item.ResponseBody)
			return
		}
		writeJSON(w, http.StatusOK, buildGeminiCachedContentResponse(cacheName, item.Model, item.UpdatedAt))
	}
}

func buildGeminiCachedContentResponse(name string, model string, updatedAt time.Time) map[string]any {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return map[string]any{
		"name":       name,
		"model":      model,
		"createTime": updatedAt.Format(time.RFC3339),
		"updateTime": updatedAt.Format(time.RFC3339),
		"expireTime": updatedAt.Add(24 * time.Hour).Format(time.RFC3339),
	}
}

func cachedContentNameFromMetadata(metadata map[string]json.RawMessage) string {
	for _, key := range []string{"cachedContent", "cached_content"} {
		if raw, ok := metadata[key]; ok {
			var value string
			if err := json.Unmarshal(raw, &value); err == nil && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func decodeGeminiCachedContentName(body []byte) (string, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	var name string
	if err := json.Unmarshal(payload["name"], &name); err != nil || strings.TrimSpace(name) == "" {
		return "", false
	}
	return strings.TrimSpace(name), true
}

func mustMarshalJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func transcriptFromUnifiedMessages(messages []domain.UnifiedMessage) []domain.GatewayMessage {
	result := make([]domain.GatewayMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, InputItem: message.InputItem, Metadata: message.Metadata})
	}
	return result
}
