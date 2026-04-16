package domain

type Channel struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Endpoint  string `json:"endpoint"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updated_at"`
}

type ChannelTestResult struct {
	Channel    string `json:"channel,omitempty"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type APIKey struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
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
}

type SystemSettingGroup struct {
	Title string              `json:"title"`
	Items []SystemSettingItem `json:"items"`
}

type CreateChannelInput struct {
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Endpoint string   `json:"endpoint"`
	APIKey   string   `json:"api_key"`
	Enabled  bool     `json:"enabled"`
	ModelIDs []string `json:"model_ids"`
}

type UpdateChannelInput struct {
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Endpoint string   `json:"endpoint"`
	APIKey   string   `json:"api_key"`
	Enabled  bool     `json:"enabled"`
	ModelIDs []string `json:"model_ids"`
}

type CreateAPIKeyInput struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
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

type CreatedAPIKey struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	RawKey  string `json:"raw_key"`
	Enabled bool   `json:"enabled"`
}
