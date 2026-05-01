package httpserver

import (
	"net/http"

	"opencrab/internal/gateway"
)

func NewRouter(service *gateway.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler("ok"))
	mux.HandleFunc("GET /readyz", healthHandler("ready"))
	mux.HandleFunc("GET /api/admin/auth/status", adminAuthStatusHandler)
	mux.HandleFunc("GET /api/admin/auth/security", adminSecondarySecurityHandler)
	mux.HandleFunc("GET /api/admin/dashboard/summary", adminDashboardSummaryHandler)
	mux.HandleFunc("GET /api/admin/channels", adminChannelsListHandler)
	mux.HandleFunc("POST /api/admin/channels", adminChannelsCreateHandler)
	mux.HandleFunc("PUT /api/admin/channels/{id}", adminChannelsUpdateHandler)
	mux.HandleFunc("DELETE /api/admin/channels/{id}", adminChannelsDeleteHandler)
	mux.HandleFunc("POST /api/admin/channels/{id}/test", adminChannelTestHandler)
	mux.HandleFunc("GET /api/admin/models", adminModelsListHandler)
	mux.HandleFunc("DELETE /api/admin/models/{id}", adminModelsDeleteHandler)
	mux.HandleFunc("GET /api/admin/model-routes", adminModelRoutesListHandler)
	mux.HandleFunc("GET /api/admin/api-keys", adminAPIKeysListHandler)
	mux.HandleFunc("POST /api/admin/api-keys", adminAPIKeysCreateHandler)
	mux.HandleFunc("PUT /api/admin/api-keys/{id}", adminAPIKeysUpdateHandler)
	mux.HandleFunc("DELETE /api/admin/api-keys/{id}", adminAPIKeysDeleteHandler)
	mux.HandleFunc("GET /api/admin/settings", adminEmptyListHandler)
	mux.HandleFunc("GET /api/admin/logs", adminLogsHandler)
	mux.HandleFunc("GET /api/admin/logs/{id}", adminLogDetailHandler)
	mux.HandleFunc("DELETE /api/admin/logs", clearRequestLogsHandler)
	mux.HandleFunc("GET /v1/models", modelsHandler)
	mux.Handle("/v1/chat/completions", newChatCompletionsHandler(service))
	mux.Handle("/v1/messages", newMessagesHandler(service))
	mux.Handle("/v1beta/models/", newGeminiModelsHandler(service))
	return mux
}
