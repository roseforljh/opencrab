package domain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type UpstreamChannel struct {
	Name     string
	Provider string
	Endpoint string
	APIKey   string
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

type ChatProvider interface {
	ForwardChatCompletions(ctx context.Context, channel UpstreamChannel, body []byte) (*ProxyResponse, error)
}

type GatewayMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type GatewayToolCallPolicy string

const (
	GatewayToolCallReject GatewayToolCallPolicy = "reject"
)

type GatewayRequest struct {
	Model          string                `json:"model"`
	Stream         bool                  `json:"stream,omitempty"`
	Messages       []GatewayMessage      `json:"messages"`
	ToolCallPolicy GatewayToolCallPolicy `json:"tool_call_policy,omitempty"`
}

type ExecutorRequest struct {
	Channel       UpstreamChannel
	UpstreamModel string
	Request       GatewayRequest
}

type ExecutionResult struct {
	Response *ProxyResponse
}

type ExecutionError struct {
	Cause         error
	StatusCode    int
	Retryable     bool
	StreamStarted bool
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
	ModelAlias    string
	UpstreamModel string
	Channel       UpstreamChannel
	Priority      int
}

type GatewayRouteStore interface {
	ListEnabledRoutesByModel(ctx context.Context, model string) ([]GatewayRoute, error)
}

type GatewayAttemptLog struct {
	RequestID     string
	Model         string
	UpstreamModel string
	Channel       string
	Provider      string
	Attempt       int
	StatusCode    int
	Retryable     bool
	StreamStarted bool
	Success       bool
	ErrorMessage  string
	RequestBody   string
	ResponseBody  string
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
