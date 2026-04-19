package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type UpstreamChannel struct {
	Name           string
	Provider       string
	Endpoint       string
	APIKey         string
	RPMLimit       int
	MaxInflight    int
	SafetyFactor   float64
	DispatchWeight int
}

type ChatCompletionsMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionsRequest struct {
	Model    string                   `json:"model"`
	Stream   bool                     `json:"stream,omitempty"`
	Messages []ChatCompletionsMessage `json:"messages"`
}

func (r ChatCompletionsRequest) ToUnifiedChatRequest() UnifiedChatRequest {
	messages := make([]UnifiedMessage, 0, len(r.Messages))
	for _, message := range r.Messages {
		messages = append(messages, UnifiedMessage{
			Role: message.Role,
			Parts: []UnifiedPart{{
				Type: "text",
				Text: message.Content,
			}},
		})
	}

	return UnifiedChatRequest{
		Protocol: ProtocolOpenAI,
		Model:    r.Model,
		Stream:   r.Stream,
		Messages: messages,
	}
}

type ProxyResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type StreamResult struct {
	StatusCode int
	Headers    map[string][]string
	Body       io.ReadCloser
}

type ChatProvider interface {
	ForwardChatCompletions(ctx context.Context, channel UpstreamChannel, body []byte) (*ProxyResponse, error)
}

type GatewayMessage struct {
	Role      string                     `json:"role"`
	Parts     []UnifiedPart              `json:"parts"`
	ToolCalls []UnifiedToolCall          `json:"tool_calls,omitempty"`
	InputItem json.RawMessage            `json:"input_item,omitempty"`
	Metadata  map[string]json.RawMessage `json:"metadata,omitempty"`
}

type GatewayToolCallPolicy string

const (
	GatewayToolCallReject GatewayToolCallPolicy = "reject"
	GatewayToolCallAllow  GatewayToolCallPolicy = "allow"
)

type GatewayRequest struct {
	Protocol        Protocol                   `json:"protocol,omitempty"`
	Operation       ProtocolOperation          `json:"operation,omitempty"`
	Model           string                     `json:"model"`
	Stream          bool                       `json:"stream,omitempty"`
	Messages        []GatewayMessage           `json:"messages"`
	Tools           []json.RawMessage          `json:"tools,omitempty"`
	Metadata        map[string]json.RawMessage `json:"metadata,omitempty"`
	ToolCallPolicy  GatewayToolCallPolicy      `json:"tool_call_policy,omitempty"`
	RequestHeaders  map[string]string          `json:"-"`
	Session         *GatewaySessionState       `json:"-"`
	AffinityKey     string                     `json:"-"`
	RuntimeSettings *GatewayRuntimeSettings    `json:"-"`
	APIKeyScope     *APIKeyScope               `json:"-"`
}

type GatewaySessionState struct {
	SessionID          string            `json:"session_id,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Input              []GatewayMessage  `json:"input,omitempty"`
	History            []GatewayMessage  `json:"history,omitempty"`
	Output             []GatewayMessage  `json:"output,omitempty"`
	ResponseID         string            `json:"response_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	ToolResults        []UnifiedToolCall `json:"tool_results,omitempty"`
}

type GatewayRuntimeSettings struct {
	CooldownDuration time.Duration
	StickyEnabled    bool
	StickyKeySource  string
}

type RoutingStrategy string

const (
	RoutingStrategySequential RoutingStrategy = "sequential"
	RoutingStrategyRoundRobin RoutingStrategy = "round_robin"
)

func NormalizeRoutingStrategy(value string) RoutingStrategy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "round_robin", "round-robin", "roundrobin", "rr":
		return RoutingStrategyRoundRobin
	default:
		return RoutingStrategySequential
	}
}

type RoutingStrategyStore interface {
	GetRoutingStrategy(ctx context.Context) (RoutingStrategy, error)
}

type RoutingCursorStore interface {
	GetRoutingCursor(ctx context.Context, routeKey string) (int, error)
	AdvanceRoutingCursor(ctx context.Context, routeKey string, candidateCount int, selectedIndex int) error
}

type ExecutorRequest struct {
	Channel       UpstreamChannel
	UpstreamModel string
	Request       GatewayRequest
}

type ExecutionResult struct {
	Response *ProxyResponse
	Stream   *StreamResult
	Metadata *GatewayExecutionMetadata
}

type GatewayExecutionMetadata struct {
	RoutingStrategy string                 `json:"routing_strategy,omitempty"`
	DecisionReason  string                 `json:"decision_reason,omitempty"`
	FallbackStage   string                 `json:"fallback_stage,omitempty"`
	FallbackChain   []string               `json:"fallback_chain,omitempty"`
	VisitedAliases  []string               `json:"visited_aliases,omitempty"`
	AttemptCount    int                    `json:"attempt_count,omitempty"`
	StickyHit       bool                   `json:"sticky_hit,omitempty"`
	StickyRouteID   int64                  `json:"sticky_route_id,omitempty"`
	StickyChannel   string                 `json:"sticky_channel,omitempty"`
	StickyReason    string                 `json:"sticky_reason,omitempty"`
	AffinityKey     string                 `json:"affinity_key,omitempty"`
	WinningBucket   string                 `json:"winning_bucket,omitempty"`
	WinningPriority int                    `json:"winning_priority,omitempty"`
	SelectedChannel string                 `json:"selected_channel,omitempty"`
	DegradedSuccess bool                   `json:"degraded_success,omitempty"`
	AttemptedRoutes []GatewayAttemptTrace  `json:"attempted_routes,omitempty"`
	Skips           []GatewaySkip          `json:"skips,omitempty"`
}

type GatewayAttemptTrace struct {
	RouteID       int64  `json:"route_id,omitempty"`
	Channel       string `json:"channel,omitempty"`
	Provider      string `json:"provider,omitempty"`
	StatusCode    int    `json:"status_code,omitempty"`
	Retryable     bool   `json:"retryable,omitempty"`
	Success       bool   `json:"success,omitempty"`
	DecisionReason string `json:"decision_reason,omitempty"`
	LatencyMs     int64  `json:"latency_ms,omitempty"`
	ErrorSummary  string `json:"error_summary,omitempty"`
}

type GatewaySkip struct {
	RouteID        int64  `json:"route_id"`
	ModelAlias     string `json:"model_alias"`
	Channel        string `json:"channel"`
	Reason         string `json:"reason"`
	CooldownUntil  string `json:"cooldown_until,omitempty"`
	Provider       string `json:"provider,omitempty"`
	InvocationMode string `json:"invocation_mode,omitempty"`
	Priority       int    `json:"priority,omitempty"`
}

type ExecutionError struct {
	Cause         error
	StatusCode    int
	Retryable     bool
	StreamStarted bool
	Metadata      *GatewayExecutionMetadata
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return "执行失败"
	}
	return e.Cause.Error()
}

func (e *ExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewExecutionError(err error, statusCode int, retryable bool, streamStarted bool) *ExecutionError {
	if err == nil {
		err = errors.New("执行失败")
	}
	return &ExecutionError{Cause: err, StatusCode: statusCode, Retryable: retryable, StreamStarted: streamStarted}
}

func AsExecutionError(err error) *ExecutionError {
	if err == nil {
		return nil
	}
	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		return execErr
	}
	return &ExecutionError{Cause: err}
}

type Executor interface {
	Execute(ctx context.Context, input ExecutorRequest) (*ExecutionResult, error)
}

type GatewayRoute struct {
	ID             int64
	ModelAlias     string
	UpstreamModel  string
	Channel        UpstreamChannel
	InvocationMode string
	Priority       int
	FallbackModel  string
	CooldownUntil  string
	LastError      string
}

type GatewayRouteStore interface {
	ListEnabledRoutesByModel(ctx context.Context, model string) ([]GatewayRoute, error)
}

type GatewayRuntimeConfigStore interface {
	GetGatewayRuntimeSettings(ctx context.Context) (GatewayRuntimeSettings, error)
}

type GatewayJobStatus string

const (
	GatewayJobStatusAccepted   GatewayJobStatus = "accepted"
	GatewayJobStatusQueued     GatewayJobStatus = "queued"
	GatewayJobStatusProcessing GatewayJobStatus = "processing"
	GatewayJobStatusCompleted  GatewayJobStatus = "completed"
	GatewayJobStatusFailed     GatewayJobStatus = "failed"
)

type GatewayJob struct {
	ID                 int64            `json:"id"`
	RequestID          string           `json:"request_id"`
	IdempotencyKey     string           `json:"idempotency_key,omitempty"`
	OwnerKeyHash       string           `json:"-"`
	RequestHash        string           `json:"-"`
	Protocol           Protocol         `json:"protocol"`
	Model              string           `json:"model"`
	Status             GatewayJobStatus `json:"status"`
	Mode               string           `json:"mode"`
	RequestPath        string           `json:"request_path"`
	RequestBody        string           `json:"-"`
	RequestHeaders     string           `json:"-"`
	ResponseStatusCode int              `json:"response_status_code,omitempty"`
	ResponseBody       string           `json:"-"`
	ErrorMessage       string           `json:"error_message,omitempty"`
	AttemptCount       int              `json:"attempt_count,omitempty"`
	WorkerID           string           `json:"worker_id,omitempty"`
	SessionID          string           `json:"session_id,omitempty"`
	LeaseUntil         string           `json:"lease_until,omitempty"`
	DeliveryMode       string           `json:"delivery_mode,omitempty"`
	WebhookURL         string           `json:"webhook_url,omitempty"`
	WebhookDeliveredAt string           `json:"webhook_delivered_at,omitempty"`
	AcceptedAt         string           `json:"accepted_at"`
	CompletedAt        string           `json:"completed_at,omitempty"`
	UpdatedAt          string           `json:"updated_at"`
	EstimatedReadyAt   string           `json:"estimated_ready_at,omitempty"`
}

type GatewayAcceptedResponse struct {
	RequestID        string `json:"request_id"`
	Status           string `json:"status"`
	Mode             string `json:"mode"`
	DeliveryMode     string `json:"delivery_mode,omitempty"`
	AcceptedAt       string `json:"accepted_at"`
	EstimatedReadyAt string `json:"estimated_ready_at,omitempty"`
	StatusURL        string `json:"status_url"`
	EventsURL        string `json:"events_url,omitempty"`
	IdempotentReplay bool   `json:"idempotent_replay,omitempty"`
}

type GatewayJobStore interface {
	Create(ctx context.Context, item GatewayJob) (GatewayJob, error)
	GetByRequestID(ctx context.Context, requestID string) (GatewayJob, error)
	GetByIdempotencyKey(ctx context.Context, ownerKeyHash string, idempotencyKey string) (GatewayJob, error)
	UpdateStatus(ctx context.Context, requestID string, status GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error
	UpdateStatusIfClaimMatches(ctx context.Context, requestID string, workerID string, leaseUntil string, status GatewayJobStatus, responseStatusCode int, responseBody string, errorMessage string, completedAt string) error
	ClaimNextRunnable(ctx context.Context, workerID string, leaseUntil string) (GatewayJob, error)
	Requeue(ctx context.Context, requestID string, workerID string, leaseUntil string, errorMessage string, nextReadyAt string) error
	MarkWebhookDelivered(ctx context.Context, requestID string, deliveredAt string) error
}

type RoutingRuntimeStateStore interface {
	MarkCooldown(ctx context.Context, routeID int64, duration time.Duration, lastError string) (string, error)
	ClearCooldown(ctx context.Context, routeID int64) error
	CountActiveCooldowns(ctx context.Context) (int, error)
}

type StickyRoutingStore interface {
	GetStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol Protocol) (int64, bool, error)
	UpsertStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol Protocol, routeID int64) error
	CountStickyBindings(ctx context.Context) (int, error)
}

type GatewayAttemptLog struct {
	RouteID          int64
	RequestID        string
	Model            string
	UpstreamModel    string
	Channel          string
	Provider         string
	RoutingStrategy  string
	InvocationBucket string
	PriorityTier     int
	CandidateCount   int
	SelectedIndex    int
	Attempt          int
	StatusCode       int
	Retryable        bool
	StreamStarted    bool
	Success          bool
	ErrorMessage     string
	DecisionReason   string
	FallbackStage    string
	SkipReason       string
	CooldownApplied  bool
	CooldownUntil    string
	StickyHit        bool
	SelectedChannel  string
	AffinityKey      string
	FallbackChain    []string
	VisitedAliases   []string
	LatencyMs        int64
	RequestBody      string
	ResponseBody     string
}

type GatewayAttemptLogger interface {
	LogGatewayAttempt(ctx context.Context, item GatewayAttemptLog) error
}

func IsRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude", "anthropic":
		return "claude"
	case "gemini", "google":
		return "gemini"
	case "glm", "zhipu":
		return "glm"
	case "kimi", "moonshot":
		return "kimi"
	case "minimax", "mini_max", "mini max":
		return "minimax"
	case "openrouter":
		return "openrouter"
	default:
		return "openai"
	}
}

func ErrNoAvailableRoute(model string) error {
	return fmt.Errorf("模型 %s 没有可用执行路由", model)
}
