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
	Logger                         *slog.Logger
	ReadinessCheck                 func(ctx context.Context) error
	GetAdminAuthState              func(ctx context.Context) (domain.AdminAuthState, error)
	SetupAdminPassword             func(ctx context.Context, password string) (domain.AdminAuthState, error)
	VerifyAdminPassword            func(ctx context.Context, password string) (domain.AdminAuthState, error)
	ChangeAdminPassword            func(ctx context.Context, input domain.AdminPasswordChangeInput) (domain.AdminAuthState, error)
	GetAdminSecondarySecurityState func(ctx context.Context) (domain.AdminSecondarySecurityState, error)
	UpdateAdminSecondaryPassword   func(ctx context.Context, input domain.AdminSecondaryPasswordUpdateInput) (domain.AdminSecondarySecurityState, error)
	VerifySecondaryPassword        func(ctx context.Context, password string) error
	CheckAdminLoginRateLimit       func(key string) bool
	ListChannels                   func(ctx context.Context) ([]domain.Channel, error)
	ListAPIKeys                    func(ctx context.Context) ([]domain.APIKey, error)
	ListModels                     func(ctx context.Context) ([]domain.ModelMapping, error)
	ListModelRoutes                func(ctx context.Context) ([]domain.ModelRoute, error)
	ListRequestLogs                func(ctx context.Context) ([]domain.RequestLogSummary, error)
	GetRequestLogDetail            func(ctx context.Context, id int64) (domain.RequestLog, error)
	ClearRequestLogs               func(ctx context.Context) error
	GetRoutingOverview             func(ctx context.Context) (domain.RoutingOverview, error)
	GetDashboardSummary            func(ctx context.Context) (domain.DashboardSummary, error)
	ListSettings                   func(ctx context.Context) ([]domain.SystemSettingGroup, error)
	CreateChannel                  func(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error)
	UpdateChannel                  func(ctx context.Context, id int64, input domain.UpdateChannelInput) error
	DeleteChannel                  func(ctx context.Context, id int64) error
	TestChannel                    func(ctx context.Context, id int64, model string) (domain.ChannelTestResult, error)
	CreateAPIKey                   func(ctx context.Context, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error)
	UpdateAPIKey                   func(ctx context.Context, id int64, input domain.UpdateAPIKeyInput) error
	DeleteAPIKey                   func(ctx context.Context, id int64) error
	UpdateSetting                  func(ctx context.Context, input domain.UpdateSystemSettingInput) (domain.SystemSetting, error)
	CreateModel                    func(ctx context.Context, input domain.CreateModelMappingInput) (domain.ModelMapping, error)
	UpdateModel                    func(ctx context.Context, id int64, input domain.UpdateModelMappingInput) error
	UpdateModelRouteBinding        func(ctx context.Context, id int64, input domain.UpdateModelRouteBindingInput) error
	DeleteModel                    func(ctx context.Context, id int64) error
	CreateModelRoute               func(ctx context.Context, input domain.CreateModelRouteInput) (domain.ModelRoute, error)
	UpdateModelRoute               func(ctx context.Context, id int64, input domain.UpdateModelRouteInput) error
	DeleteModelRoute               func(ctx context.Context, id int64) error
	VerifyAPIKey                   func(ctx context.Context, rawKey string) (bool, error)
	CreateRequestLog               func(ctx context.Context, item domain.RequestLog) error
	GetGatewayRuntimeSettings      func(ctx context.Context) (domain.GatewayRuntimeSettings, error)
	GetDispatchRuntimeSettings     func(ctx context.Context) (domain.DispatchRuntimeSettings, error)
	CheckRateLimit                 func(key string) bool
	CreateGatewayJob               func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error)
	GetGatewayJobByRequestID       func(ctx context.Context, requestID string) (domain.GatewayJob, error)
	GetGatewayJobByIdempotencyKey  func(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error)
	UpdateGatewayJobStatus         func(ctx context.Context, requestID string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error
	ExecuteGateway                 func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error)
	CountClaudeTokens              func(ctx context.Context, req *http.Request, body []byte) (*domain.ProxyResponse, error)
	ResponseSessions               ResponseSessionStore
	CopyProxy                      func(w http.ResponseWriter, resp *domain.ProxyResponse) error
	CopyStream                     func(w http.ResponseWriter, stream *domain.StreamResult) error
	RenderProxyError               func(w http.ResponseWriter, err error)
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
		admin.Get("/auth/status", func(w http.ResponseWriter, req *http.Request) {
			if deps.GetAdminAuthState == nil {
				http.Error(w, "admin auth status handler not configured", http.StatusNotImplemented)
				return
			}
			state, err := deps.GetAdminAuthState(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, buildAdminAuthStatus(req, state))
		})

		admin.Post("/auth/setup", func(w http.ResponseWriter, req *http.Request) {
			if deps.SetupAdminPassword == nil {
				http.Error(w, "admin auth setup handler not configured", http.StatusNotImplemented)
				return
			}
			var input domain.AdminPasswordInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := validateAdminPasswordInput(input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			state, err := deps.SetupAdminPassword(req.Context(), input.Password)
			if err != nil {
				statusCode := http.StatusInternalServerError
				if strings.Contains(err.Error(), "已初始化") {
					statusCode = http.StatusConflict
				}
				http.Error(w, err.Error(), statusCode)
				return
			}
			writeAdminSessionCookie(w, req, state.SessionSecret)
			writeJSON(w, http.StatusCreated, buildAuthenticatedAdminStatus(state))
		})

		admin.Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
			if deps.VerifyAdminPassword == nil {
				http.Error(w, "admin auth login handler not configured", http.StatusNotImplemented)
				return
			}
			var input domain.AdminPasswordInput
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				http.Error(w, "请求体格式不正确", http.StatusBadRequest)
				return
			}
			if err := validateAdminPasswordInput(input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if deps.CheckAdminLoginRateLimit != nil && !deps.CheckAdminLoginRateLimit(adminAuthRateLimitKey(req)) {
				http.Error(w, "登录尝试过于频繁，请稍后再试", http.StatusTooManyRequests)
				return
			}
			state, err := deps.VerifyAdminPassword(req.Context(), input.Password)
			if err != nil {
				statusCode := http.StatusInternalServerError
				if strings.Contains(err.Error(), "尚未初始化") {
					statusCode = http.StatusPreconditionRequired
				}
				if strings.Contains(err.Error(), "密码错误") {
					statusCode = http.StatusUnauthorized
				}
				http.Error(w, err.Error(), statusCode)
				return
			}
			writeAdminSessionCookie(w, req, state.SessionSecret)
			writeJSON(w, http.StatusOK, buildAuthenticatedAdminStatus(state))
		})

		admin.Post("/auth/logout", func(w http.ResponseWriter, req *http.Request) {
			clearAdminSessionCookie(w, req)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})

		admin.Group(func(protected chi.Router) {
			protected.Use(requireAdminSession(deps))

			protected.Get("/auth/security", func(w http.ResponseWriter, req *http.Request) {
				if deps.GetAdminSecondarySecurityState == nil {
					http.Error(w, "admin security handler not configured", http.StatusNotImplemented)
					return
				}
				state, err := deps.GetAdminSecondarySecurityState(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, state)
			})

			protected.Put("/auth/password", func(w http.ResponseWriter, req *http.Request) {
				if deps.ChangeAdminPassword == nil {
					http.Error(w, "admin password change handler not configured", http.StatusNotImplemented)
					return
				}
				var input domain.AdminPasswordChangeInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					http.Error(w, "请求体格式不正确", http.StatusBadRequest)
					return
				}
				if err := validateAdminPasswordChangeInput(input); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				state, err := deps.ChangeAdminPassword(req.Context(), input)
				if err != nil {
					statusCode := http.StatusInternalServerError
					if strings.Contains(err.Error(), "密码错误") {
						statusCode = http.StatusUnauthorized
					}
					if strings.Contains(err.Error(), "不一致") || strings.Contains(err.Error(), "至少需要") {
						statusCode = http.StatusBadRequest
					}
					http.Error(w, err.Error(), statusCode)
					return
				}
				writeAdminSessionCookie(w, req, state.SessionSecret)
				writeJSON(w, http.StatusOK, buildAuthenticatedAdminStatus(state))
			})

			protected.Put("/auth/secondary", func(w http.ResponseWriter, req *http.Request) {
				if deps.UpdateAdminSecondaryPassword == nil {
					http.Error(w, "admin secondary password handler not configured", http.StatusNotImplemented)
					return
				}
				var input domain.AdminSecondaryPasswordUpdateInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					http.Error(w, "请求体格式不正确", http.StatusBadRequest)
					return
				}
				if err := validateSecondaryPasswordUpdateInput(input); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				state, err := deps.UpdateAdminSecondaryPassword(req.Context(), input)
				if err != nil {
					statusCode := http.StatusInternalServerError
					if strings.Contains(err.Error(), "密码错误") || strings.Contains(err.Error(), "未通过校验") {
						statusCode = http.StatusUnauthorized
					}
					if strings.Contains(err.Error(), "不一致") || strings.Contains(err.Error(), "至少需要") || strings.Contains(err.Error(), "尚未设置") {
						statusCode = http.StatusBadRequest
					}
					http.Error(w, err.Error(), statusCode)
					return
				}
				writeJSON(w, http.StatusOK, state)
			})

			protected.Get("/channels", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Post("/channels", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Put("/channels/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Delete("/channels/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Post("/channels/{id}/test", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Get("/api-keys", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Post("/api-keys", func(w http.ResponseWriter, req *http.Request) {
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
				if deps.VerifySecondaryPassword != nil {
					if err := deps.VerifySecondaryPassword(req.Context(), extractSecondaryPassword(req)); err != nil {
						statusCode := http.StatusInternalServerError
						if strings.Contains(err.Error(), "未通过校验") || strings.Contains(err.Error(), "尚未设置") {
							statusCode = http.StatusUnauthorized
						}
						http.Error(w, err.Error(), statusCode)
						return
					}
				}

				created, err := deps.CreateAPIKey(req.Context(), input)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				writeJSON(w, http.StatusCreated, created)
			})

			protected.Put("/api-keys/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Delete("/api-keys/{id}", func(w http.ResponseWriter, req *http.Request) {
				if deps.DeleteAPIKey == nil {
					http.Error(w, "api key delete handler not configured", http.StatusNotImplemented)
					return
				}
				id, err := parseInt64Param(req, "id")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if deps.VerifySecondaryPassword != nil {
					if err := deps.VerifySecondaryPassword(req.Context(), extractSecondaryPassword(req)); err != nil {
						statusCode := http.StatusInternalServerError
						if strings.Contains(err.Error(), "未通过校验") || strings.Contains(err.Error(), "尚未设置") {
							statusCode = http.StatusUnauthorized
						}
						http.Error(w, err.Error(), statusCode)
						return
					}
				}
				if err := deps.DeleteAPIKey(req.Context(), id); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			})

			protected.Get("/models", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Post("/models", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Put("/models/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Put("/model-route-bindings/{id}", func(w http.ResponseWriter, req *http.Request) {
				if deps.UpdateModelRouteBinding == nil {
					http.Error(w, "model route binding update handler not configured", http.StatusNotImplemented)
					return
				}
				id, err := parseInt64Param(req, "id")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				var input domain.UpdateModelRouteBindingInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					http.Error(w, "请求体格式不正确", http.StatusBadRequest)
					return
				}
				if strings.TrimSpace(input.Alias) == "" || strings.TrimSpace(input.UpstreamModel) == "" || strings.TrimSpace(input.ChannelName) == "" {
					http.Error(w, "模型别名、上游模型和渠道名称不能为空", http.StatusBadRequest)
					return
				}
				if err := deps.UpdateModelRouteBinding(req.Context(), id, input); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			})

			protected.Delete("/models/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Get("/model-routes", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Post("/model-routes", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Put("/model-routes/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Delete("/model-routes/{id}", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Get("/logs", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Get("/logs/{id}", func(w http.ResponseWriter, req *http.Request) {
				if deps.GetRequestLogDetail == nil {
					http.Error(w, "log detail handler not configured", http.StatusNotImplemented)
					return
				}
				id, err := parseInt64Param(req, "id")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				item, err := deps.GetRequestLogDetail(req.Context(), id)
				if err != nil {
					statusCode := http.StatusInternalServerError
					if strings.Contains(err.Error(), "不存在") {
						statusCode = http.StatusNotFound
					}
					http.Error(w, err.Error(), statusCode)
					return
				}
				writeJSON(w, http.StatusOK, item)
			})

			protected.Get("/routing/overview", func(w http.ResponseWriter, req *http.Request) {
				if deps.GetRoutingOverview == nil {
					http.Error(w, "routing overview handler not configured", http.StatusNotImplemented)
					return
				}
				item, err := deps.GetRoutingOverview(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, item)
			})

			protected.Get("/dashboard/summary", func(w http.ResponseWriter, req *http.Request) {
				if deps.GetDashboardSummary == nil {
					http.Error(w, "dashboard summary handler not configured", http.StatusNotImplemented)
					return
				}
				item, err := deps.GetDashboardSummary(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, item)
			})

			protected.Delete("/logs", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
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

			protected.Put("/settings", func(w http.ResponseWriter, req *http.Request) {
				if deps.UpdateSetting == nil {
					http.Error(w, "settings update handler not configured", http.StatusNotImplemented)
					return
				}

				var input domain.UpdateSystemSettingInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					http.Error(w, "请求体格式不正确", http.StatusBadRequest)
					return
				}
				if err := validateUpdateSystemSettingInput(&input); err != nil {
					statusCode := http.StatusBadRequest
					if strings.Contains(err.Error(), "不能通过系统设置接口修改") {
						statusCode = http.StatusForbidden
					}
					http.Error(w, err.Error(), statusCode)
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
	})

	r.Get("/v1/models", HandleOpenAIModels(deps))
	r.Post("/v1/chat/completions", HandleGatewayChatCompletions(deps))
	r.Post("/v1/responses", HandleOpenAIResponses(deps))
	r.Get("/v1/responses", HandleOpenAIResponsesWebSocket(deps))
	r.Get("/v1/requests/{requestID}", HandleGatewayRequestStatus(deps))
	r.Get("/v1/requests/{requestID}/events", HandleGatewayRequestEvents(deps))
	r.Post("/v1/messages", HandleClaudeMessages(deps))
	r.Post("/v1/messages/count_tokens", HandleClaudeCountTokens(deps))
	r.Post("/v1beta/models/{model}:generateContent", HandleGeminiGenerateContent(deps))
	r.Post("/v1beta/models/{model}:streamGenerateContent", HandleGeminiStreamGenerateContent(deps))
	return r
}
