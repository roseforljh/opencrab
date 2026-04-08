package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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

	r.Post("/v1/chat/completions", func(w http.ResponseWriter, req *http.Request) {
		if deps.ProxyChat == nil || deps.CopyProxy == nil {
			http.Error(w, "proxy handler not configured", http.StatusNotImplemented)
			return
		}

		if deps.VerifyAPIKey == nil {
			http.Error(w, "api key verifier not configured", http.StatusNotImplemented)
			return
		}

		rawKey := strings.TrimSpace(strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "))
		if rawKey == "" {
			http.Error(w, "缺少 Authorization Bearer Token", http.StatusUnauthorized)
			return
		}

		allowed, err := deps.VerifyAPIKey(req.Context(), rawKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "API Key 无效或已禁用", http.StatusUnauthorized)
			return
		}

		if deps.CheckRateLimit != nil && !deps.CheckRateLimit(rawKey) {
			http.Error(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
			return
		}

		startedAt := time.Now()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "读取请求体失败", http.StatusBadRequest)
			return
		}

		resp, err := deps.ProxyChat(req.Context(), body)
		if err != nil {
			if deps.RenderProxyError != nil {
				deps.RenderProxyError(w, err)
			} else {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return
		}

		channelName := firstHeaderValue(resp.Headers, "X-Opencrab-Channel")
		if channelName == "" {
			channelName = "default-channel"
		}

		modelName := extractModel(body)
		usage := extractUsageMetrics(resp.Body)
		responseBody := truncateLogBody(string(resp.Body))

		if err := deps.CopyProxy(w, resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if deps.CreateRequestLog != nil {
			details := marshalLogDetails(map[string]any{
				"request_path":      req.URL.Path,
				"channel":           channelName,
				"model":             modelName,
				"request_body":      truncateLogBody(string(body)),
				"response_body":     responseBody,
				"response_status":   resp.StatusCode,
				"prompt_tokens":     usage.PromptTokens,
				"completion_tokens": usage.CompletionTokens,
				"total_tokens":      usage.TotalTokens,
				"cache_hit":         usage.CacheHit,
				"test_mode":         false,
			})
			_ = deps.CreateRequestLog(req.Context(), domain.RequestLog{
				RequestID:        middleware.GetReqID(req.Context()),
				Model:            modelName,
				Channel:          channelName,
				StatusCode:       resp.StatusCode,
				LatencyMs:        time.Since(startedAt).Milliseconds(),
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
				CacheHit:         usage.CacheHit,
				RequestBody:      truncateLogBody(string(body)),
				ResponseBody:     responseBody,
				Details:          details,
				CreatedAt:        time.Now().Format(time.RFC3339),
			})
		}
	})

	return r
}

// writeJSON 负责把结构化数据写成 JSON 响应。
//
// 这里统一封装的目的是让后续接口都走同一套输出方式，
// 避免每个 handler 自己设置响应头、自己编码，导致风格不统一。
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		http.Error(w, "写入 JSON 响应失败", http.StatusInternalServerError)
	}
}

func validateCreateChannelInput(input domain.CreateChannelInput) error {
	return validateChannelInput(input.Name, input.Provider, input.Endpoint, input.APIKey, input.ModelIDs, true)
}

func validateUpdateChannelInput(input domain.UpdateChannelInput) error {
	return validateChannelInput(input.Name, input.Provider, input.Endpoint, input.APIKey, input.ModelIDs, false)
}

func validateChannelInput(name string, provider string, endpoint string, apiKey string, modelIDs []string, requireAPIKey bool) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("渠道名称不能为空")
	}
	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf("渠道类型不能为空")
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("请求地址不能为空")
	}
	if requireAPIKey && strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if len(modelIDs) == 0 {
		return fmt.Errorf("至少添加一个模型 ID")
	}
	seen := make(map[string]struct{}, len(modelIDs))
	for _, modelID := range modelIDs {
		normalized := strings.TrimSpace(modelID)
		if normalized == "" {
			return fmt.Errorf("模型 ID 不能为空")
		}
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("模型 ID 不能重复")
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func validateCreateAPIKeyInput(input domain.CreateAPIKeyInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("密钥名称不能为空")
	}
	return nil
}

func extractModel(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "unknown-model"
	}
	model, _ := payload["model"].(string)
	if strings.TrimSpace(model) == "" {
		return "unknown-model"
	}
	return model
}

func fallbackLogModel(resultModel string, inputModel string) string {
	if strings.TrimSpace(resultModel) != "" {
		return resultModel
	}
	if strings.TrimSpace(inputModel) != "" {
		return strings.TrimSpace(inputModel)
	}
	return "unknown-model"
}

func fallbackLogChannel(resultChannel string, id int64) string {
	if strings.TrimSpace(resultChannel) != "" {
		return resultChannel
	}
	return fmt.Sprintf("channel-%d", id)
}

func fallbackStatusCode(statusCode int) int {
	if statusCode > 0 {
		return statusCode
	}
	return http.StatusBadGateway
}

func marshalLogDetails(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func truncateLogBody(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 1200 {
		return trimmed
	}
	return trimmed[:1200] + "..."
}

func firstHeaderValue(headers map[string][]string, key string) string {
	values := headers[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

type usageMetrics struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheHit         bool
}

func extractUsageMetrics(body []byte) usageMetrics {
	var payload struct {
		Usage struct {
			PromptTokens         int64 `json:"prompt_tokens"`
			CompletionTokens     int64 `json:"completion_tokens"`
			TotalTokens          int64 `json:"total_tokens"`
			PromptCacheHitTokens int64 `json:"prompt_cache_hit_tokens"`
			PromptTokensDetails  struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return usageMetrics{}
	}

	totalTokens := payload.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = payload.Usage.PromptTokens + payload.Usage.CompletionTokens
	}

	return usageMetrics{
		PromptTokens:     payload.Usage.PromptTokens,
		CompletionTokens: payload.Usage.CompletionTokens,
		TotalTokens:      totalTokens,
		CacheHit:         payload.Usage.PromptCacheHitTokens > 0 || payload.Usage.PromptTokensDetails.CachedTokens > 0,
	}
}

func parseInt64Param(req *http.Request, key string) (int64, error) {
	value := chi.URLParam(req, key)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的 %s 参数", key)
	}
	return id, nil
}
