package domain

type Channel struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Provider        string  `json:"provider"`
	Endpoint        string  `json:"endpoint"`
	Enabled         bool    `json:"enabled"`
	RPMLimit        int     `json:"rpm_limit"`
	MaxInflight     int     `json:"max_inflight"`
	SafetyFactor    float64 `json:"safety_factor"`
	EnabledForAsync bool    `json:"enabled_for_async"`
	DispatchWeight  int     `json:"dispatch_weight"`
	UpdatedAt       string  `json:"updated_at"`
}

type ChannelTestResult struct {
	Channel    string `json:"channel,omitempty"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type APIKey struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	ChannelNames []string `json:"channel_names,omitempty"`
	ModelAliases []string `json:"model_aliases,omitempty"`
}

type ModelMapping struct {
	ID            int64  `json:"id"`
	Alias         string `json:"alias"`
	UpstreamModel string `json:"upstream_model"`
}

type ModelRoute struct {
	ID             int64  `json:"id"`
	ModelAlias     string `json:"model_alias"`
	ChannelName    string `json:"channel_name"`
	InvocationMode string `json:"invocation_mode,omitempty"`
	Priority       int    `json:"priority"`
	FallbackModel  string `json:"fallback_model"`
	CooldownUntil  string `json:"cooldown_until,omitempty"`
	LastError      string `json:"last_error,omitempty"`
}

type RequestLog struct {
	ID               int64  `json:"id"`
	RequestID        string `json:"request_id"`
	Model            string `json:"model"`
	Channel          string `json:"channel"`
	StatusCode       int    `json:"status_code"`
	LatencyMs        int64  `json:"latency_ms"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	CacheHit         bool   `json:"cache_hit"`
	RequestBody      string `json:"request_body"`
	ResponseBody     string `json:"response_body"`
	Details          string `json:"details"`
	CreatedAt        string `json:"created_at"`
}

type RequestLogSummary struct {
	ID               int64  `json:"id"`
	RequestID        string `json:"request_id"`
	Model            string `json:"model"`
	Channel          string `json:"channel"`
	StatusCode       int    `json:"status_code"`
	LatencyMs        int64  `json:"latency_ms"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	CacheHit         bool   `json:"cache_hit"`
	Details          string `json:"details"`
	CreatedAt        string `json:"created_at"`
}

type DashboardSummary struct {
	ChannelsCount                  int                     `json:"channels_count"`
	ModelsCount                    int                     `json:"models_count"`
	RoutesCount                    int                     `json:"routes_count"`
	APIKeysCount                   int                     `json:"api_keys_count"`
	EnabledChannelsCount           int                     `json:"enabled_channels_count"`
	DefaultChannel                 string                  `json:"default_channel"`
	ProviderCount                  int                     `json:"provider_count"`
	RoutingOverview                RoutingOverview         `json:"routing_overview"`
	TodayRequests                  int                     `json:"today_requests"`
	TodaySuccessCount              int                     `json:"today_success_count"`
	TodayErrorCount                int                     `json:"today_error_count"`
	TotalRequests                  int                     `json:"total_requests"`
	SuccessCount                   int                     `json:"success_count"`
	ErrorCount                     int                     `json:"error_count"`
	AverageLatency                 int64                   `json:"average_latency"`
	PromptTokens                   int64                   `json:"prompt_tokens"`
	CompletionTokens               int64                   `json:"completion_tokens"`
	TotalTokens                    int64                   `json:"total_tokens"`
	TotalMeteredRequests           int                     `json:"total_metered_requests"`
	CacheHitCount                  int                     `json:"cache_hit_count"`
	CacheHitRate                   float64                 `json:"cache_hit_rate"`
	RequestsPerMinute              int                     `json:"requests_per_minute"`
	RequestsPerMinuteSuccess       int                     `json:"requests_per_minute_success"`
	RequestsPerMinuteError         int                     `json:"requests_per_minute_error"`
	TokensPerMinute                int64                   `json:"tokens_per_minute"`
	TokensPerMinuteMeteredRequests int                     `json:"tokens_per_minute_metered_requests"`
	DailyCounts                    []DashboardDailyCount   `json:"daily_counts"`
	TrafficSeries                  []DashboardTrafficPoint `json:"traffic_series"`
	RecentLogs                     []DashboardRecentLog    `json:"recent_logs"`
	ChannelMix                     []DashboardShareItem    `json:"channel_mix"`
	ModelRanking                   []DashboardRankingItem  `json:"model_ranking"`
	RuntimeRedisEnabled            bool                    `json:"runtime_redis_enabled"`
	RuntimeRedisAddress            string                  `json:"runtime_redis_address"`
	RuntimeRedisDB                 int                     `json:"runtime_redis_db"`
	RuntimeRedisTLSEnabled         bool                    `json:"runtime_redis_tls_enabled"`
	RuntimeRedisKeyPrefix          string                  `json:"runtime_redis_key_prefix"`
	DispatchPause                  bool                    `json:"dispatch_pause"`
	DispatcherWorkers              int                     `json:"dispatcher_workers"`
	QueueMode                      string                  `json:"queue_mode"`
	DefaultQueue                   string                  `json:"default_queue"`
	PriorityQueues                 string                  `json:"priority_queues"`
	QueueTTLSec                    int                     `json:"queue_ttl_s"`
	SyncHoldMs                     int                     `json:"sync_hold_ms"`
	RetryReserveRatio              float64                 `json:"retry_reserve_ratio"`
	BacklogCap                     int                     `json:"backlog_cap"`
	MaxAttempts                    int                     `json:"max_attempts"`
	BackoffMode                    string                  `json:"backoff_mode"`
	BackoffDelayMs                 int                     `json:"backoff_delay_ms"`
	DeadLetterEnabled              bool                    `json:"dead_letter_enabled"`
	MetricsEnabled                 bool                    `json:"metrics_enabled"`
	LongWaitThresholdSec           int                     `json:"long_wait_threshold_s"`
	ShowWorkerStatus               bool                    `json:"show_worker_status"`
	ShowQueueDepth                 bool                    `json:"show_queue_depth"`
	ShowRetryRate                  bool                    `json:"show_retry_rate"`
	AsyncEnabledChannels           int                     `json:"async_enabled_channels"`
	TotalRPMLimit                  int                     `json:"total_rpm_limit"`
	TotalMaxInflight               int                     `json:"total_max_inflight"`
}

type DashboardDailyCount struct {
	Label          string  `json:"label"`
	Requests       int     `json:"requests"`
	SuccessRate    float64 `json:"success_rate"`
	AverageLatency int64   `json:"average_latency"`
	TotalTokens    int64   `json:"total_tokens"`
}

type DashboardTrafficPoint struct {
	Label    string `json:"label"`
	Requests int    `json:"requests"`
	Success  int    `json:"success"`
	Errors   int    `json:"errors"`
}

type DashboardRecentLog struct {
	Time      string `json:"time"`
	Model     string `json:"model"`
	Channel   string `json:"channel"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

type DashboardShareItem struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type DashboardRankingItem struct {
	Label string `json:"label"`
	Value int    `json:"value"`
	Width int    `json:"width"`
}

type SystemSetting struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

type SystemSettingItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Sensitive   bool   `json:"sensitive"`
	Configured  bool   `json:"configured"`
}

type SystemSettingGroup struct {
	Title string              `json:"title"`
	Items []SystemSettingItem `json:"items"`
}

type RoutingOverview struct {
	ActiveCooldowns int                  `json:"active_cooldowns"`
	StickyBindings  int                  `json:"sticky_bindings"`
	StickyHits24h   int                  `json:"sticky_hits_24h"`
	Fallbacks24h    int                  `json:"fallbacks_24h"`
	Skipped24h      int                  `json:"skipped_24h"`
	RequestCount24h int                  `json:"request_count_24h"`
	HealthyRoutes   int                  `json:"healthy_routes"`
	TotalRoutes     int                  `json:"total_routes"`
	PressureScore   int                  `json:"pressure_score"`
	RecentErrors    []string             `json:"recent_errors"`
	CursorStates    []RoutingCursorState `json:"cursor_states"`
}

type RoutingCursorState struct {
	RouteKey  string `json:"route_key"`
	NextIndex int    `json:"next_index"`
	UpdatedAt string `json:"updated_at"`
}

type AdminAuthState struct {
	Initialized   bool   `json:"initialized"`
	PasswordHash  string `json:"-"`
	SessionSecret string `json:"-"`
	InitializedAt string `json:"password_initialized_at,omitempty"`
}

type AdminAuthStatus struct {
	Initialized   bool `json:"initialized"`
	Authenticated bool `json:"authenticated"`
}

type AdminPasswordInput struct {
	Password string `json:"password"`
}

type AdminPasswordChangeInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type AdminSecondaryPasswordUpdateInput struct {
	Enabled                  bool   `json:"enabled"`
	CurrentAdminPassword     string `json:"current_admin_password"`
	CurrentSecondaryPassword string `json:"current_secondary_password,omitempty"`
	NewPassword              string `json:"new_password,omitempty"`
	ConfirmPassword          string `json:"confirm_password,omitempty"`
}

type AdminSecondarySecurityState struct {
	Enabled    bool `json:"enabled"`
	Configured bool `json:"configured"`
}

type CreateChannelInput struct {
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	Endpoint        string   `json:"endpoint"`
	APIKey          string   `json:"api_key"`
	Enabled         bool     `json:"enabled"`
	ModelIDs        []string `json:"model_ids"`
	RPMLimit        int      `json:"rpm_limit"`
	MaxInflight     int      `json:"max_inflight"`
	SafetyFactor    float64  `json:"safety_factor"`
	EnabledForAsync bool     `json:"enabled_for_async"`
	DispatchWeight  int      `json:"dispatch_weight"`
}

type UpdateChannelInput struct {
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	Endpoint        string   `json:"endpoint"`
	APIKey          string   `json:"api_key"`
	Enabled         bool     `json:"enabled"`
	ModelIDs        []string `json:"model_ids"`
	RPMLimit        int      `json:"rpm_limit"`
	MaxInflight     int      `json:"max_inflight"`
	SafetyFactor    float64  `json:"safety_factor"`
	EnabledForAsync bool     `json:"enabled_for_async"`
	DispatchWeight  int      `json:"dispatch_weight"`
}

type CreateAPIKeyInput struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	ChannelNames []string `json:"channel_names,omitempty"`
	ModelAliases []string `json:"model_aliases,omitempty"`
}

type UpdateAPIKeyInput struct {
	Enabled bool `json:"enabled"`
}

type UpdateSystemSettingInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateModelMappingInput struct {
	Alias         string `json:"alias"`
	UpstreamModel string `json:"upstream_model"`
}

type UpdateModelMappingInput struct {
	Alias         string `json:"alias"`
	UpstreamModel string `json:"upstream_model"`
}

type CreateModelRouteInput struct {
	ModelAlias     string `json:"model_alias"`
	ChannelName    string `json:"channel_name"`
	InvocationMode string `json:"invocation_mode,omitempty"`
	Priority       int    `json:"priority"`
	FallbackModel  string `json:"fallback_model"`
}

type UpdateModelRouteInput struct {
	ModelAlias     string `json:"model_alias"`
	ChannelName    string `json:"channel_name"`
	InvocationMode string `json:"invocation_mode,omitempty"`
	Priority       int    `json:"priority"`
	FallbackModel  string `json:"fallback_model"`
}

type UpdateModelRouteBindingInput struct {
	Alias          string `json:"alias"`
	UpstreamModel  string `json:"upstream_model"`
	ChannelName    string `json:"channel_name"`
	InvocationMode string `json:"invocation_mode,omitempty"`
	Priority       int    `json:"priority"`
	FallbackModel  string `json:"fallback_model"`
}

type CreatedAPIKey struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	RawKey       string   `json:"raw_key"`
	Enabled      bool     `json:"enabled"`
	ChannelNames []string `json:"channel_names,omitempty"`
	ModelAliases []string `json:"model_aliases,omitempty"`
}

type APIKeyScope struct {
	ID           int64
	Name         string
	ChannelNames []string
	ModelAliases []string
}

type CapabilityProfile struct {
	ScopeType    string   `json:"scope_type"`
	ScopeKey     string   `json:"scope_key"`
	Operation    string   `json:"operation"`
	Enabled      *bool    `json:"enabled,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type CapabilityCatalog struct {
	ScopeTypes []string `json:"scope_types"`
	Operations []string `json:"operations"`
	Items      []string `json:"items"`
}

type CapabilityProfileListResponse struct {
	Items   []CapabilityProfile `json:"items"`
	Catalog CapabilityCatalog   `json:"catalog"`
}

type UpsertCapabilityProfileInput struct {
	ScopeType    string   `json:"scope_type"`
	ScopeKey     string   `json:"scope_key"`
	Operation    string   `json:"operation"`
	Enabled      *bool    `json:"enabled,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type DeleteCapabilityProfileInput struct {
	ScopeType string `json:"scope_type"`
	ScopeKey  string `json:"scope_key"`
	Operation string `json:"operation"`
}
