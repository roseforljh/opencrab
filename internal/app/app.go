package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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
	config           config.Config
	logger           *slog.Logger
	db               *sql.DB
	client           *http.Client
	server           *http.Server
	dispatcher       *usecase.GatewayJobDispatcher
	dispatchSettings domain.DispatchRuntimeConfigStore
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
	authRateLimiter := usecase.NewRateLimiter(1, 5)
	channelTester := provider.NewChannelTester(client)
	gatewayStore := store.NewGatewayStore(db)
	responseSessions := store.NewResponseSessionStore(db)
	routingConfigStore := store.NewRoutingConfigStore(db)
	runtimeConfigStore := store.NewGatewayRuntimeConfigStore(db)
	dispatchRuntimeStore := store.NewDispatchRuntimeConfigStore(db)
	routingCursorStore := store.NewRoutingCursorStore(db)
	runtimeStateStore := store.NewRoutingRuntimeStateStore(db)
	stickyRoutingStore := store.NewStickyRoutingStore(db)
	dispatchQuotaManager := usecase.NewRedisDispatchQuotaManager(dispatchRuntimeStore)
	jobStore := store.NewGatewayJobStore(db)
	gatewayService := usecase.NewGatewayService(
		gatewayStore,
		map[string]domain.Executor{
			"openai": provider.NewOpenAIExecutor(client),
			"claude": provider.NewClaudeExecutor(client),
			"gemini": provider.NewGeminiExecutor(client),
		},
		store.NewGatewayAttemptLogStore(db),
		dispatchQuotaManager,
		routingConfigStore,
		routingCursorStore,
		runtimeConfigStore,
		runtimeStateStore,
		stickyRoutingStore,
	)
	dispatcher := usecase.NewGatewayJobDispatcher(
		jobStore,
		dispatchRuntimeStore,
		func(job domain.GatewayJob) (domain.GatewayRequest, error) {
			return httpserver.DecodeStoredGatewayJobRequest(responseSessions, runtimeConfigStore.GetGatewayRuntimeSettings, job)
		},
		func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return gatewayService.Execute(ctx, requestID, req)
		},
		client,
	)

	router := httpserver.NewRouter(httpserver.Dependencies{
		Logger: logger,
		ReadinessCheck: func(ctx context.Context) error {
			return db.PingContext(ctx)
		},
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return store.GetAdminAuthState(ctx, db)
		},
		SetupAdminPassword: func(ctx context.Context, password string) (domain.AdminAuthState, error) {
			return store.SetupAdminPassword(ctx, db, password)
		},
		VerifyAdminPassword: func(ctx context.Context, password string) (domain.AdminAuthState, error) {
			return store.VerifyAdminPassword(ctx, db, password)
		},
		ChangeAdminPassword: func(ctx context.Context, input domain.AdminPasswordChangeInput) (domain.AdminAuthState, error) {
			return store.ChangeAdminPassword(ctx, db, input)
		},
		GetAdminSecondarySecurityState: func(ctx context.Context) (domain.AdminSecondarySecurityState, error) {
			return store.GetAdminSecondarySecurityState(ctx, db)
		},
		UpdateAdminSecondaryPassword: func(ctx context.Context, input domain.AdminSecondaryPasswordUpdateInput) (domain.AdminSecondarySecurityState, error) {
			return store.UpdateAdminSecondaryPassword(ctx, db, input)
		},
		VerifySecondaryPassword: func(ctx context.Context, password string) error {
			return store.VerifySecondaryPassword(ctx, db, password)
		},
		CheckAdminLoginRateLimit: func(key string) bool {
			return authRateLimiter.Allow(key)
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
		GetDashboardSummary: func(ctx context.Context) (domain.DashboardSummary, error) {
			return store.GetDashboardSummary(ctx, db)
		},
		ListSettings: func(ctx context.Context) ([]domain.SystemSettingGroup, error) {
			items, err := store.ListSystemSettings(ctx, db)
			if err != nil {
				return nil, err
			}
			return buildSystemSettingGroups(appConfig, items), nil
		},
		GetRoutingOverview: func(ctx context.Context) (domain.RoutingOverview, error) {
			return store.GetRoutingOverview(ctx, db)
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
		UpdateModelRouteBinding: func(ctx context.Context, id int64, input domain.UpdateModelRouteBindingInput) error {
			return store.UpdateModelRouteBinding(ctx, db, id, input)
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
		ListRequestLogs: func(ctx context.Context) ([]domain.RequestLogSummary, error) {
			return store.ListRequestLogSummaries(ctx, db)
		},
		GetRequestLogDetail: func(ctx context.Context, id int64) (domain.RequestLog, error) {
			return store.GetRequestLogDetail(ctx, db, id)
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
		GetGatewayRuntimeSettings: func(ctx context.Context) (domain.GatewayRuntimeSettings, error) {
			return runtimeConfigStore.GetGatewayRuntimeSettings(ctx)
		},
		GetDispatchRuntimeSettings: func(ctx context.Context) (domain.DispatchRuntimeSettings, error) {
			return dispatchRuntimeStore.GetDispatchRuntimeSettings(ctx)
		},
		CheckRateLimit: func(key string) bool {
			return rateLimiter.Allow(key)
		},
		CreateGatewayJob: func(ctx context.Context, item domain.GatewayJob) (domain.GatewayJob, error) {
			return jobStore.Create(ctx, item)
		},
		GetGatewayJobByRequestID: func(ctx context.Context, requestID string) (domain.GatewayJob, error) {
			return jobStore.GetByRequestID(ctx, requestID)
		},
		GetGatewayJobByIdempotencyKey: func(ctx context.Context, ownerKeyHash string, idempotencyKey string) (domain.GatewayJob, error) {
			return jobStore.GetByIdempotencyKey(ctx, ownerKeyHash, idempotencyKey)
		},
		UpdateGatewayJobStatus: func(ctx context.Context, requestID string, status domain.GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error {
			return jobStore.UpdateStatus(ctx, requestID, status, responseStatusCode, responseBody, errorMessage, completedAt)
		},
		ExecuteGateway: func(ctx context.Context, requestID string, req domain.GatewayRequest) (*domain.ExecutionResult, error) {
			return gatewayService.Execute(ctx, requestID, req)
		},
		CountClaudeTokens: func(ctx context.Context, req *http.Request, body []byte) (*domain.ProxyResponse, error) {
			unified, err := provider.DecodeClaudeChatRequest(body)
			if err != nil {
				return nil, err
			}
			gatewayReq := domain.GatewayRequest{Protocol: domain.ProtocolClaude, Model: unified.Model, ToolCallPolicy: domain.GatewayToolCallAllow}
			for _, message := range unified.Messages {
				gatewayReq.Messages = append(gatewayReq.Messages, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, Metadata: message.Metadata})
			}
			settings, settingsErr := runtimeConfigStore.GetGatewayRuntimeSettings(ctx)
			if settingsErr == nil {
				gatewayReq.AffinityKey = httpserver.ExtractSessionAffinityKey(req, gatewayReq, settings)
				gatewayReq.RuntimeSettings = &settings
			}
			selected, err := gatewayService.SelectRoute(ctx, gatewayReq)
			if err != nil {
				return nil, err
			}
			unified.Model = selected.UpstreamModel
			encoded, err := provider.EncodeClaudeChatRequest(unified)
			if err != nil {
				return nil, err
			}
			return provider.ForwardClaudeCountTokens(ctx, client, selected.Channel, encoded, req.Header.Get("anthropic-version"), req.Header.Get("anthropic-beta"))
		},
		ResponseSessions: responseSessions,
		CopyProxy:        provider.CopyResponse,
		CopyStream:       provider.CopyStreamResponse,
		RenderProxyError: provider.RenderProxyError,
	})

	server := &http.Server{
		Addr:              appConfig.HTTP.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		config:           appConfig,
		logger:           logger,
		db:               db,
		client:           client,
		server:           server,
		dispatcher:       dispatcher,
		dispatchSettings: dispatchRuntimeStore,
	}, nil
}

// Run 负责真正启动 HTTP 服务。
//
// 这里单独拆成方法，而不是直接在 New 里启动，
// 是为了让“创建应用”和“运行应用”这两个阶段分开，
// 后续更方便补测试、补初始化逻辑、补优雅关闭。
func (a *App) Run() error {
	if a.dispatcher != nil && a.dispatchSettings != nil {
		settings, err := a.dispatchSettings.GetDispatchRuntimeSettings(context.Background())
		if err == nil {
			workerCount := settings.WorkerConcurrency
			if workerCount <= 0 {
				workerCount = 1
			}
			ctx := context.Background()
			for index := 0; index < workerCount; index++ {
				go func(workerName string) {
					_ = a.dispatcher.Run(ctx, workerName)
				}(usecase.WorkerID("dispatcher", index+1))
			}
		}
	}
	a.logger.Info("后端服务启动中",
		slog.String("address", a.config.HTTP.Address),
		slog.String("db_path", a.config.DB.Path),
	)
	fmt.Printf("%s 后端服务启动中，环境: %s，监听地址: %s\n", a.config.App.Name, a.config.App.Environment, a.config.HTTP.Address)
	return a.server.ListenAndServe()
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
	makeItem := func(key string, label string, description string, fallback string) domain.SystemSettingItem {
		return domain.SystemSettingItem{Key: key, Label: label, Description: description, Value: readValue(key, fallback)}
	}
	makeSensitiveItem := func(key string, label string, description string) domain.SystemSettingItem {
		current, ok := values[key]
		return domain.SystemSettingItem{Key: key, Label: label, Description: description, Value: "", Sensitive: true, Configured: ok && strings.TrimSpace(current) != ""}
	}

	return []domain.SystemSettingGroup{
		{
			Title: "基础设置",
			Items: []domain.SystemSettingItem{
				makeItem("service_name", "服务名称", "控制台展示名称。", appConfig.App.Name),
				makeItem("runtime_environment", "运行环境", "当前后端运行环境标识。", appConfig.App.Environment),
				makeItem("default_timeout", "默认超时", "上游请求默认超时。", "60 秒"),
				makeItem("log_retention", "默认日志保留", "请求日志默认保留时长。", "7 天"),
			},
		},
		{
			Title: "运行策略",
			Items: []domain.SystemSettingItem{
				makeItem("gateway.routing_strategy", "模型路由策略", "同模型多渠道时使用顺序或轮询策略。可选值：sequential / round_robin。", string(domain.RoutingStrategySequential)),
				makeItem("gateway.cooldown_seconds", "Cooldown 秒数", "重试型失败后，路由进入冷却的秒数。", "45"),
				makeItem("gateway.sticky_enabled", "Sticky 路由", "是否启用会话粘性路由。可选值：true / false。", "true"),
				makeItem("gateway.sticky_key_source", "Sticky Key 来源", "会话 key 的提取来源。可选值：auto / header / metadata。", "auto"),
				makeItem("max_concurrency", "最大并发数", "控制全局并发阈值。", "128"),
				makeItem("stream_release", "流式中断释放", "中断后是否立即回收资源。", "启用"),
				makeItem("error_redaction", "错误脱敏", "是否在日志中脱敏敏感错误信息。", "启用"),
			},
		},
		{
			Title: "Runtime Redis",
			Items: []domain.SystemSettingItem{
				makeItem("dispatch.redis_enabled", "启用 Runtime Redis", "是否启用 Redis 作为调度运行时热状态层。可选值：true / false。", "false"),
				makeItem("dispatch.redis_address", "Redis 地址", "运行时 Redis 地址，格式示例：127.0.0.1:6379。", "127.0.0.1:6379"),
				makeItem("dispatch.redis_db", "Redis DB", "运行时 Redis 逻辑库编号。", "0"),
				makeSensitiveItem("dispatch.redis_password", "Redis 密码", "运行时 Redis 连接密码，已配置时不会明文回显。"),
				makeItem("dispatch.redis_tls_enabled", "Redis TLS", "是否启用 Redis TLS。可选值：true / false。", "false"),
				makeItem("dispatch.redis_key_prefix", "Key 前缀", "运行时 Redis key 前缀，便于隔离环境。", "opencrab"),
			},
		},
		{
			Title: "Dispatch",
			Items: []domain.SystemSettingItem{
				makeItem("dispatch.worker_concurrency", "Worker 并发", "调度 worker 总并发数。", "128"),
				makeItem("dispatch.queue_mode", "队列模式", "调度队列模式。可选值：single / priority。", "priority"),
				makeItem("dispatch.default_queue", "默认队列", "默认入队目标。", "model-default"),
				makeItem("dispatch.priority_queues", "优先级队列", "优先级队列名称列表，逗号分隔。", "p0,p1,p2"),
				makeItem("dispatch.pause_dispatch", "暂停调度", "是否暂停新任务分发。可选值：true / false。", "false"),
				makeItem("dispatch.sync_hold_ms", "同步等待预算 ms", "请求走同步桥接前允许占用的最大等待时间。", "3000"),
				makeItem("dispatch.backlog_cap", "积压上限", "允许同时受理的最大积压 job 数。", "20000"),
			},
		},
		{
			Title: "Retry & Recovery",
			Items: []domain.SystemSettingItem{
				makeItem("dispatch.max_attempts", "最大重试次数", "单个 job 的最大执行尝试次数。", "5"),
				makeItem("dispatch.backoff_mode", "退避模式", "失败后重试退避模式。可选值：fixed / exponential。", "exponential"),
				makeItem("dispatch.backoff_delay_ms", "退避延迟 ms", "基础退避延迟。", "500"),
				makeItem("dispatch.retry_reserve_ratio", "重试预算比例", "为失败重试预留的配额比例。", "0.10"),
				makeItem("dispatch.dead_letter_enabled", "死信队列", "是否启用死信队列。可选值：true / false。", "true"),
				makeItem("dispatch.queue_ttl_s", "队列 TTL 秒数", "job 在队列中的最长保留时间。", "1800"),
			},
		},
		{
			Title: "Observability",
			Items: []domain.SystemSettingItem{
				makeItem("dispatch.metrics_enabled", "启用调度指标", "是否开启调度观测指标。可选值：true / false。", "true"),
				makeItem("dispatch.long_wait_threshold_s", "长等待阈值秒数", "超过该阈值的 job 会被标记为长等待。", "15"),
				makeItem("dispatch.show_worker_status", "展示 Worker 状态", "dashboard 是否展示 worker 状态面板。可选值：true / false。", "true"),
				makeItem("dispatch.show_queue_depth", "展示队列深度", "dashboard 是否展示队列深度。可选值：true / false。", "true"),
				makeItem("dispatch.show_retry_rate", "展示重试率", "dashboard 是否展示重试率。可选值：true / false。", "true"),
			},
		},
	}
}
