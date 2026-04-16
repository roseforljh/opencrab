package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// healthResponse 是健康检查接口的统一返回结构。
//
// 这里先保持字段极少，目的是让第一阶段先打通“服务活着”和“服务可对外响应”的验证链路。
// 后续会继续增加版本号、数据库状态、依赖状态等信息。
type healthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type Dependencies struct {
	Logger           *slog.Logger
	ReadinessCheck   func(ctx context.Context) error
	ListChannels     func(ctx context.Context) ([]domain.Channel, error)
	ListAPIKeys      func(ctx context.Context) ([]domain.APIKey, error)
	ListModels       func(ctx context.Context) ([]domain.ModelMapping, error)
	ListModelRoutes  func(ctx context.Context) ([]domain.ModelRoute, error)
	ListRequestLogs  func(ctx context.Context) ([]domain.RequestLog, error)
	ClearRequestLogs func(ctx context.Context) error
	ListSettings     func(ctx context.Context) ([]domain.SystemSettingGroup, error)
	CreateChannel    func(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error)
	UpdateChannel    func(ctx context.Context, id int64, input domain.UpdateChannelInput) error
	DeleteChannel    func(ctx context.Context, id int64) error
	TestChannel      func(ctx context.Context, id int64, model string) (domain.ChannelTestResult, error)
	CreateAPIKey     func(ctx context.Context, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error)
	UpdateAPIKey     func(ctx context.Context, id int64, input domain.UpdateAPIKeyInput) error
	DeleteAPIKey     func(ctx context.Context, id int64) error
	UpdateSetting    func(ctx context.Context, input domain.UpdateSystemSettingInput) (domain.SystemSetting, error)
	CreateModel      func(ctx context.Context, input domain.CreateModelMappingInput) (domain.ModelMapping, error)
	UpdateModel      func(ctx context.Context, id int64, input domain.UpdateModelMappingInput) error
	DeleteModel      func(ctx context.Context, id int64) error
	CreateModelRoute func(ctx context.Context, input domain.CreateModelRouteInput) (domain.ModelRoute, error)
	UpdateModelRoute func(ctx context.Context, id int64, input domain.UpdateModelRouteInput) error
	DeleteModelRoute func(ctx context.Context, id int64) error
	VerifyAPIKey     func(ctx context.Context, rawKey string) (bool, error)
	CreateRequestLog func(ctx context.Context, item domain.RequestLog) error
	CheckRateLimit   func(key string) bool
	ProxyChat        func(ctx context.Context, body []byte) (*domain.ProxyResponse, error)
	ProxyClaude      func(ctx context.Context, body []byte) (*domain.ProxyResponse, error)
	ProxyGemini      func(ctx context.Context, model string, body []byte, stream bool) (*domain.ProxyResponse, error)
	CopyProxy        func(w http.ResponseWriter, resp *domain.ProxyResponse) error
	RenderProxyError func(w http.ResponseWriter, err error)
}

// NewRouter 负责创建整个 HTTP 路由树。
//
// 当前阶段先只注册最基础的中间件与健康检查路由：
// 1. 统一 request id，方便后面串日志。
// 2. 统一恢复 panic，避免服务直接崩掉。
// 3. 提供 /healthz 和 /readyz 作为首批验证接口。
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	if deps.Logger != nil {
		r.Use(observability.RequestLogger(deps.Logger))
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if deps.ReadinessCheck != nil {
			ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()
			if err := deps.ReadinessCheck(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, healthResponse{
					Status:    "not_ready",
					Timestamp: time.Now().Format(time.RFC3339),
				})
				return
			}
		}

		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ready",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})

	r.Route("/api/admin", func(admin chi.Router) {
		admin.Get("/channels", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListChannels == nil {
				http.Error(w, "channels handler not configured", http.StatusNotImplemented)
				return
			}

			items, err := deps.ListChannels(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Post("/channels", func(w http.ResponseWriter, req *http.Request) {
			if deps.CreateChannel == nil {
				http.Error(w, "channel create handler not configured", http.StatusNotImplemented)
				return
			}

			var input domain.CreateChannelInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := validateCreateChannelInput(input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			created, err := deps.CreateChannel(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusCreated, created)
		})

		admin.Put("/channels/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.UpdateChannel == nil {
				http.Error(w, "channel update handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var input domain.UpdateChannelInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := validateUpdateChannelInput(input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.UpdateChannel(req.Context(), id, input); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Delete("/channels/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.DeleteChannel == nil {
				http.Error(w, "channel delete handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.DeleteChannel(req.Context(), id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Post("/channels/{id}/test", func(w http.ResponseWriter, req *http.Request) {
			if deps.TestChannel == nil {
				http.Error(w, "channel test handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var input struct {
				Model string `json:"model"`
			}
			if req.Body != nil {
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
					http.Error(w, "请求体格式不正确", http.StatusBadRequest)
					return
				}
			}

			startedAt := time.Now()
			result, err := deps.TestChannel(req.Context(), id, strings.TrimSpace(input.Model))
			if err != nil {
				if deps.CreateRequestLog != nil {
					details := marshalLogDetails(map[string]any{
						"request_path": req.URL.Path,
						"provider":     result.Provider,
						"channel":      fallbackLogChannel(result.Channel, id),
						"model":        fallbackLogModel(result.Model, input.Model),
						"message":      result.Message,
						"test_mode":    true,
					})
					_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
						RequestID:    middleware.GetReqID(req.Context()),
						Model:        fallbackLogModel(result.Model, input.Model),
						Channel:      fallbackLogChannel(result.Channel, id),
						StatusCode:   fallbackStatusCode(result.StatusCode),
						LatencyMs:    time.Since(startedAt).Milliseconds(),
						RequestBody:  truncateLogBody(marshalLogDetails(map[string]any{"model": strings.TrimSpace(input.Model), "test_mode": true})),
						ResponseBody: truncateLogBody(result.Message),
						Details:      details,
						CreatedAt:    time.Now().Format(time.RFC3339),
					})
				}
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			if deps.CreateRequestLog != nil {
				details := marshalLogDetails(map[string]any{
					"request_path": req.URL.Path,
					"provider":     result.Provider,
					"channel":      fallbackLogChannel(result.Channel, id),
					"model":        fallbackLogModel(result.Model, input.Model),
					"message":      result.Message,
					"test_mode":    true,
				})
				_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
					RequestID:    middleware.GetReqID(req.Context()),
					Model:        fallbackLogModel(result.Model, input.Model),
					Channel:      fallbackLogChannel(result.Channel, id),
					StatusCode:   result.StatusCode,
					LatencyMs:    time.Since(startedAt).Milliseconds(),
					RequestBody:  truncateLogBody(marshalLogDetails(map[string]any{"model": strings.TrimSpace(input.Model), "test_mode": true})),
					ResponseBody: truncateLogBody(result.Message),
					Details:      details,
					CreatedAt:    time.Now().Format(time.RFC3339),
				})
			}

			writeJSON(w, http.StatusOK, result)
		})

		admin.Get("/api-keys", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListAPIKeys == nil {
				http.Error(w, "api keys handler not configured", http.StatusNotImplemented)
				return
			}

			items, err := deps.ListAPIKeys(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Post("/api-keys", func(w http.ResponseWriter, req *http.Request) {
			if deps.CreateAPIKey == nil {
				http.Error(w, "api key create handler not configured", http.StatusNotImplemented)
				return
			}

			var input domain.CreateAPIKeyInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := validateCreateAPIKeyInput(input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			created, err := deps.CreateAPIKey(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusCreated, created)
		})

		admin.Put("/api-keys/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.UpdateAPIKey == nil {
				http.Error(w, "api key update handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var input domain.UpdateAPIKeyInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := deps.UpdateAPIKey(req.Context(), id, input); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Delete("/api-keys/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.DeleteAPIKey == nil {
				http.Error(w, "api key delete handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.DeleteAPIKey(req.Context(), id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Get("/models", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListModels == nil {
				http.Error(w, "models handler not configured", http.StatusNotImplemented)
				return
			}
			items, err := deps.ListModels(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Post("/models", func(w http.ResponseWriter, req *http.Request) {
			if deps.CreateModel == nil {
				http.Error(w, "model create handler not configured", http.StatusNotImplemented)
				return
			}
			var input domain.CreateModelMappingInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(input.Alias) == "" || strings.TrimSpace(input.UpstreamModel) == "" {
				http.Error(w, "模型别名和上游模型不能为空", http.StatusBadRequest)
				return
			}
			created, err := deps.CreateModel(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusCreated, created)
		})

		admin.Put("/models/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.UpdateModel == nil {
				http.Error(w, "model update handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var input domain.UpdateModelMappingInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(input.Alias) == "" || strings.TrimSpace(input.UpstreamModel) == "" {
				http.Error(w, "模型别名和上游模型不能为空", http.StatusBadRequest)
				return
			}
			if err := deps.UpdateModel(req.Context(), id, input); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Delete("/models/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.DeleteModel == nil {
				http.Error(w, "model delete handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.DeleteModel(req.Context(), id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Get("/model-routes", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListModelRoutes == nil {
				http.Error(w, "model routes handler not configured", http.StatusNotImplemented)
				return
			}
			items, err := deps.ListModelRoutes(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Post("/model-routes", func(w http.ResponseWriter, req *http.Request) {
			if deps.CreateModelRoute == nil {
				http.Error(w, "model route create handler not configured", http.StatusNotImplemented)
				return
			}
			var input domain.CreateModelRouteInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(input.ModelAlias) == "" || strings.TrimSpace(input.ChannelName) == "" {
				http.Error(w, "模型别名和渠道名称不能为空", http.StatusBadRequest)
				return
			}
			created, err := deps.CreateModelRoute(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusCreated, created)
		})

		admin.Put("/model-routes/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.UpdateModelRoute == nil {
				http.Error(w, "model route update handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var input domain.UpdateModelRouteInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(input.ModelAlias) == "" || strings.TrimSpace(input.ChannelName) == "" {
				http.Error(w, "模型别名和渠道名称不能为空", http.StatusBadRequest)
				return
			}
			if err := deps.UpdateModelRoute(req.Context(), id, input); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Delete("/model-routes/{id}", func(w http.ResponseWriter, req *http.Request) {
			if deps.DeleteModelRoute == nil {
				http.Error(w, "model route delete handler not configured", http.StatusNotImplemented)
				return
			}
			id, err := parseInt64Param(req, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := deps.DeleteModelRoute(req.Context(), id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Get("/logs", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListRequestLogs == nil {
				http.Error(w, "logs handler not configured", http.StatusNotImplemented)
				return
			}

			items, err := deps.ListRequestLogs(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Delete("/logs", func(w http.ResponseWriter, req *http.Request) {
			if deps.ClearRequestLogs == nil {
				http.Error(w, "logs clear handler not configured", http.StatusNotImplemented)
				return
			}

			if err := deps.ClearRequestLogs(req.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
			if deps.ListSettings == nil {
				http.Error(w, "settings handler not configured", http.StatusNotImplemented)
				return
			}

			items, err := deps.ListSettings(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		})

		admin.Put("/settings", func(w http.ResponseWriter, req *http.Request) {
			if deps.UpdateSetting == nil {
				http.Error(w, "settings update handler not configured", http.StatusNotImplemented)
				return
			}

			var input domain.UpdateSystemSettingInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(input.Key) == "" {
				http.Error(w, "设置项 key 不能为空", http.StatusBadRequest)
				return
			}

			updated, err := deps.UpdateSetting(req.Context(), input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, updated)
		})
	})

	r.Post("/v1/chat/completions", HandleGatewayChatCompletions(deps))
	r.Post("/v1/messages", HandleClaudeMessages(deps))
	r.Post("/v1beta/models/{model}:generateContent", HandleGeminiGenerateContent(deps, false))
	r.Post("/v1beta/models/{model}:streamGenerateContent", HandleGeminiGenerateContent(deps, true))
	return r
}
