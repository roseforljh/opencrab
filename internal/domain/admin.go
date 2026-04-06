package domain

type Channel struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Endpoint string `json:"endpoint"`
	Enabled  bool   `json:"enabled"`
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
	ID            int64  `json:"id"`
	ModelAlias    string `json:"model_alias"`
	ChannelName   string `json:"channel_name"`
	Priority      int    `json:"priority"`
	FallbackModel string `json:"fallback_model"`
}

type RequestLog struct {
	ID         int64  `json:"id"`
	RequestID  string `json:"request_id"`
	Model      string `json:"model"`
	Channel    string `json:"channel"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	CreatedAt  string `json:"created_at"`
}

type CreateChannelInput struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
	Enabled  bool   `json:"enabled"`
}

type UpdateChannelInput struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
	Enabled  bool   `json:"enabled"`
}

type CreateAPIKeyInput struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type UpdateAPIKeyInput struct {
	Enabled bool `json:"enabled"`
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
	ModelAlias    string `json:"model_alias"`
	ChannelName   string `json:"channel_name"`
	Priority      int    `json:"priority"`
	FallbackModel string `json:"fallback_model"`
}

type UpdateModelRouteInput struct {
	ModelAlias    string `json:"model_alias"`
	ChannelName   string `json:"channel_name"`
	Priority      int    `json:"priority"`
	FallbackModel string `json:"fallback_model"`
}

type CreatedAPIKey struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	RawKey  string `json:"raw_key"`
	Enabled bool   `json:"enabled"`
}
