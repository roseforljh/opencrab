package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"opencrab/internal/config"
	"opencrab/internal/domain"
	"opencrab/internal/observability"
	"opencrab/internal/provider"
	store "opencrab/internal/store/sqlite"
	"opencrab/internal/transport/httpserver"
	"opencrab/internal/usecase"
)

// App 表示当前后端服务的应用实例。
//
// 这个结构体存在的目的是把“应用启动相关的信息”集中放在一起，
// 避免 main 函数后面不断膨胀，最终把配置、日志、数据库、路由全都写在入口文件里。
//
// 当前阶段只保留最小字段：
// 1. 服务监听地址。
// 2. HTTP Server。
//
// 后续会继续把配置对象、数据库连接、日志对象等逐步加进来。
type App struct {
	config config.Config
	logger *slog.Logger
	db     *sql.DB
	client *http.Client
	server *http.Server
}

// New 负责创建应用实例并准备好 HTTP 服务。
//
// 当前版本先使用固定地址和基础路由，目标是尽快把骨架立起来，
// 等配置系统落地后，再把地址、超时等参数改成配置驱动。
func New() (*App, error) {
	appConfig := config.Load()
	if err := config.Validate(appConfig); err != nil {
		return nil, err
	}

	logger := observability.NewLogger(appConfig)
	db, err := store.Open(appConfig.DB.Path)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.ApplyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: appConfig.TLS.UpstreamInsecureSkipVerify},
		},
	}
	rateLimiter := usecase.NewRateLimiter(5, 10)
	channelTester := provider.NewChannelTester(client)
	gatewayRouteStore := store.NewGatewayStore(db)
	attemptLogger := store.NewGatewayAttemptLogStore(db)
	gatewayService := usecase.NewGatewayService(gatewayRouteStore, map[string]domain.Executor{
		"openai": provider.NewOpenAIExecutor(client),
		"claude": provider.NewClaudeExecutor(client),
		"gemini": provider.NewGeminiExecutor(client),
	}, attemptLogger)

	router := httpserver.NewRouter(httpserver.Dependencies{
		Logger: logger,
		ReadinessCheck: func(ctx context.Context) error {
			return db.PingContext(ctx)
		},
		ListChannels: func(ctx context.Context) ([]domain.Channel, error) {
			return store.ListChannels(ctx, db)
		},
		ListAPIKeys: func(ctx context.Context) ([]domain.APIKey, error) {
			return store.ListAPIKeys(ctx, db)
		},
		ListModels: func(ctx context.Context) ([]domain.ModelMapping, error) {
			return store.ListModelMappings(ctx, db)
		},
		ListModelRoutes: func(ctx context.Context) ([]domain.ModelRoute, error) {
			return store.ListModelRoutes(ctx, db)
		},
		ListSettings: func(ctx context.Context) ([]domain.SystemSettingGroup, error) {
			items, err := store.ListSystemSettings(ctx, db)
			if err != nil {
				return nil, err
			}
			return buildSystemSettingGroups(appConfig, items), nil
		},
		CreateChannel: func(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
			return store.CreateChannel(ctx, db, input)
		},
		UpdateChannel: func(ctx context.Context, id int64, input domain.UpdateChannelInput) error {
			return store.UpdateChannel(ctx, db, id, input)
		},
		DeleteChannel: func(ctx context.Context, id int64) error {
			return store.DeleteChannel(ctx, db, id)
		},
		TestChannel: func(ctx context.Context, id int64, model string) (domain.ChannelTestResult, error) {
			channel, err := store.GetChannelByID(ctx, db, id)
			if err != nil {
				return domain.ChannelTestResult{}, err
			}
			return channelTester.TestChannel(ctx, channel, model)
		},
		CreateAPIKey: func(ctx context.Context, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error) {
			return store.CreateAPIKey(ctx, db, input)
		},
		UpdateAPIKey: func(ctx context.Context, id int64, input domain.UpdateAPIKeyInput) error {
			return store.UpdateAPIKey(ctx, db, id, input)
		},
		DeleteAPIKey: func(ctx context.Context, id int64) error {
			return store.DeleteAPIKey(ctx, db, id)
		},
		UpdateSetting: func(ctx context.Context, input domain.UpdateSystemSettingInput) (domain.SystemSetting, error) {
			return store.UpsertSystemSetting(ctx, db, input)
		},
		CreateModel: func(ctx context.Context, input domain.CreateModelMappingInput) (domain.ModelMapping, error) {
			return store.CreateModelMapping(ctx, db, input)
		},
		UpdateModel: func(ctx context.Context, id int64, input domain.UpdateModelMappingInput) error {
			return store.UpdateModelMapping(ctx, db, id, input)
		},
		DeleteModel: func(ctx context.Context, id int64) error {
			return store.DeleteModelMapping(ctx, db, id)
		},
		CreateModelRoute: func(ctx context.Context, input domain.CreateModelRouteInput) (domain.ModelRoute, error) {
			return store.CreateModelRoute(ctx, db, input)
		},
		UpdateModelRoute: func(ctx context.Context, id int64, input domain.UpdateModelRouteInput) error {
			return store.UpdateModelRoute(ctx, db, id, input)
		},
		DeleteModelRoute: func(ctx context.Context, id int64) error {
			return store.DeleteModelRoute(ctx, db, id)
		},
		ListRequestLogs: func(ctx context.Context) ([]domain.RequestLog, error) {
			return store.ListRequestLogs(ctx, db)
		},
		ClearRequestLogs: func(ctx context.Context) error {
			return store.ClearRequestLogs(ctx, db)
		},
		VerifyAPIKey: func(ctx context.Context, rawKey string) (bool, error) {
			return store.VerifyAPIKey(ctx, db, rawKey)
		},
		CreateRequestLog: func(ctx context.Context, item domain.RequestLog) error {
			return store.CreateRequestLog(ctx, db, item)
		},
		CheckRateLimit: func(key string) bool {
			return rateLimiter.Allow(key)
		},
		ProxyChat: func(ctx context.Context, body []byte) (*domain.ProxyResponse, error) {
			requestID := fmt.Sprintf("gw-%d", time.Now().UnixNano())
			gatewayReq, err := buildGatewayRequest(body)
			if err != nil {
				return nil, err
			}
			return gatewayService.Execute(ctx, requestID, gatewayReq)
		},
		CopyProxy:        provider.CopyResponse,
		RenderProxyError: provider.RenderProxyError,
	})

	server := &http.Server{
		Addr:              appConfig.HTTP.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		config: appConfig,
		logger: logger,
		db:     db,
		client: client,
		server: server,
	}, nil
}

// Run 负责真正启动 HTTP 服务。
//
// 这里单独拆成方法，而不是直接在 New 里启动，
// 是为了让“创建应用”和“运行应用”这两个阶段分开，
// 后续更方便补测试、补初始化逻辑、补优雅关闭。
func (a *App) Run() error {
	a.logger.Info("后端服务启动中",
		slog.String("address", a.config.HTTP.Address),
		slog.String("db_path", a.config.DB.Path),
	)
	fmt.Printf("%s 后端服务启动中，环境: %s，监听地址: %s\n", a.config.App.Name, a.config.App.Environment, a.config.HTTP.Address)
	return a.server.ListenAndServe()
}

func buildGatewayRequest(body []byte) (domain.GatewayRequest, error) {
	var payload domain.ChatCompletionsRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.GatewayRequest{}, fmt.Errorf("解析请求体失败: %w", err)
	}
	model := payload.Model
	if model == "" {
		return domain.GatewayRequest{}, fmt.Errorf("请求缺少 model")
	}
	messages := make([]domain.GatewayMessage, 0, len(payload.Messages))
	for _, message := range payload.Messages {
		messages = append(messages, domain.GatewayMessage{Role: message.Role, Text: message.Content})
	}
	return domain.GatewayRequest{Model: model, Stream: payload.Stream, Messages: messages, ToolCallPolicy: domain.GatewayToolCallReject}, nil
}

func buildSystemSettingGroups(appConfig config.Config, items []domain.SystemSetting) []domain.SystemSettingGroup {
	values := make(map[string]string, len(items))
	for _, item := range items {
		values[item.Key] = item.Value
	}

	readValue := func(key string, fallback string) string {
		if value, ok := values[key]; ok && value != "" {
			return value
		}
		return fallback
	}

	return []domain.SystemSettingGroup{
		{
			Title: "基础设置",
			Items: []domain.SystemSettingItem{
				{Key: "service_name", Label: "服务名称", Description: "控制台展示名称。", Value: readValue("service_name", appConfig.App.Name)},
				{Key: "runtime_environment", Label: "运行环境", Description: "当前后端运行环境标识。", Value: readValue("runtime_environment", appConfig.App.Environment)},
				{Key: "default_timeout", Label: "默认超时", Description: "上游请求默认超时。", Value: readValue("default_timeout", "60 秒")},
				{Key: "log_retention", Label: "默认日志保留", Description: "请求日志默认保留时长。", Value: readValue("log_retention", "7 天")},
			},
		},
		{
			Title: "运行策略",
			Items: []domain.SystemSettingItem{
				{Key: "max_concurrency", Label: "最大并发数", Description: "控制全局并发阈值。", Value: readValue("max_concurrency", "128")},
				{Key: "stream_release", Label: "流式中断释放", Description: "中断后是否立即回收资源。", Value: readValue("stream_release", "启用")},
				{Key: "error_redaction", Label: "错误脱敏", Description: "是否在日志中脱敏敏感错误信息。", Value: readValue("error_redaction", "启用")},
			},
		},
	}
}
