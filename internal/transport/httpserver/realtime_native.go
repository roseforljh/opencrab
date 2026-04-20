package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"opencrab/internal/domain"

	"github.com/gorilla/websocket"
)

func HandleOpenAIRealtimeClientSecrets(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.SelectDirectRoute == nil || deps.ForwardOpenAIRealtimeClientSecret == nil || deps.CopyProxy == nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("realtime client secret handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolOpenAI)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("读取请求体失败"), http.StatusBadRequest, false, false), domain.ProtocolOpenAI)
			return
		}
		model := strings.TrimSpace(extractRealtimeModelFromJSON(body))
		if model == "" {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("realtime 请求缺少 model"), http.StatusBadRequest, false, false), domain.ProtocolOpenAI)
			return
		}
		route, err := deps.SelectDirectRoute(req.Context(), model, "openai", &scope)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		resp, err := deps.ForwardOpenAIRealtimeClientSecret(req.Context(), route, body)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		if err := deps.CopyProxy(w, resp); err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusInternalServerError, false, false), domain.ProtocolOpenAI)
		}
	}
}

func HandleOpenAIRealtimeCalls(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if deps.SelectDirectRoute == nil || deps.ForwardOpenAIRealtimeCall == nil || deps.CopyProxy == nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("realtime calls handler not configured"), http.StatusNotImplemented, false, false), domain.ProtocolOpenAI)
			return
		}
		_, scope, err := resolveGatewayAPIKey(deps, req)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, gatewayErrorStatusCode(err), false, false), domain.ProtocolOpenAI)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(fmt.Errorf("读取请求体失败"), http.StatusBadRequest, false, false), domain.ProtocolOpenAI)
			return
		}
		model, err := extractRealtimeModelFromBody(req.Header.Get("Content-Type"), body, req.URL.Query().Get("model"))
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusBadRequest, false, false), domain.ProtocolOpenAI)
			return
		}
		route, err := deps.SelectDirectRoute(req.Context(), model, "openai", &scope)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		resp, err := deps.ForwardOpenAIRealtimeCall(req.Context(), route, req.Header.Get("Content-Type"), body, req.URL.RawQuery)
		if err != nil {
			renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
			return
		}
		if err := deps.CopyProxy(w, resp); err != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusInternalServerError, false, false), domain.ProtocolOpenAI)
		}
	}
}

func maybeProxyOpenAIRealtimeWebSocket(deps Dependencies, w http.ResponseWriter, req *http.Request, scope domain.APIKeyScope) bool {
	if deps.SelectDirectRoute == nil || deps.DialOpenAIRealtime == nil {
		return false
	}
	model := strings.TrimSpace(req.URL.Query().Get("model"))
	if model == "" {
		return false
	}
	route, err := deps.SelectDirectRoute(req.Context(), model, "openai", &scope)
	if err != nil {
		renderGatewayErrorForProtocol(deps, w, err, domain.ProtocolOpenAI)
		return false
	}
	upstreamConn, resp, err := deps.DialOpenAIRealtime(req.Context(), route, req)
	if err != nil {
		if resp != nil {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, resp.StatusCode, false, false), domain.ProtocolOpenAI)
		} else {
			renderGatewayErrorForProtocol(deps, w, domain.NewExecutionError(err, http.StatusBadGateway, false, false), domain.ProtocolOpenAI)
		}
		return true
	}
	clientConn, err := responsesUpgrader.Upgrade(w, req, nil)
	if err != nil {
		_ = upstreamConn.Close()
		return true
	}
	proxyRealtimeSockets(clientConn, upstreamConn)
	return true
}

func proxyRealtimeSockets(clientConn *websocket.Conn, upstreamConn *websocket.Conn) {
	defer clientConn.Close()
	defer upstreamConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var once sync.Once
	stop := func() { once.Do(cancel) }
	copyLoop := func(src *websocket.Conn, dst *websocket.Conn) {
		defer stop()
		for {
			messageType, data, err := src.ReadMessage()
			if err != nil {
				return
			}
			if err := dst.WriteMessage(messageType, data); err != nil {
				return
			}
		}
	}

	go copyLoop(clientConn, upstreamConn)
	go copyLoop(upstreamConn, clientConn)
	<-ctx.Done()
}

func extractRealtimeModelFromBody(contentType string, body []byte, fallback string) (string, error) {
	if model := strings.TrimSpace(extractRealtimeModelFromJSON(body)); model != "" {
		return model, nil
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		model := extractRealtimeModelFromMultipart(body, params["boundary"])
		if strings.TrimSpace(model) != "" {
			return strings.TrimSpace(model), nil
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback), nil
	}
	return "", fmt.Errorf("realtime 请求缺少 model")
}

func extractRealtimeModelFromMultipart(body []byte, boundary string) string {
	if strings.TrimSpace(boundary) == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			return ""
		}
		if strings.TrimSpace(part.FormName()) != "session" {
			_, _ = io.Copy(io.Discard, part)
			continue
		}
		payload, readErr := io.ReadAll(part)
		if readErr != nil {
			return ""
		}
		return extractRealtimeModelFromJSON(payload)
	}
}

func extractRealtimeModelFromJSON(body []byte) string {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	for _, key := range []string{"model"} {
		if raw := payload[key]; len(raw) > 0 {
			var value string
			if err := json.Unmarshal(raw, &value); err == nil && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	if raw := payload["session"]; len(raw) > 0 {
		var session map[string]json.RawMessage
		if err := json.Unmarshal(raw, &session); err == nil {
			var value string
			if err := json.Unmarshal(session["model"], &value); err == nil && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}
