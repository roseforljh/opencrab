package reject

import (
	"fmt"

	"opencrab/internal/domain"
)

type Decision struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Retryable  bool   `json:"retryable"`
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Decide(req domain.GatewayRequest, reason string) *Decision {
	code := normalizeRejectCode(reason)
	return &Decision{
		Code:       code,
		Message:    rejectMessage(req, code),
		StatusCode: rejectStatusCode(code),
		Retryable:  false,
	}
}

func normalizeRejectCode(reason string) string {
	switch reason {
	case "claude_native_features_require_claude_route",
		"gemini_native_features_require_gemini_route",
		"responses_native_features_require_openai_route",
		"openai_native_features_require_openai_route",
		"route_capability_not_supported",
		"target_operation_not_supported",
		"gemini_url_context_with_function_calling_unsupported",
		"claude_thinking_with_forced_tool_choice_unsupported":
		return reason
	case "no_viable_route", "no_execution_attempts":
		return "no_viable_route"
	default:
		if reason == "" {
			return "request_rejected"
		}
		return reason
	}
}

func rejectStatusCode(code string) int {
	switch code {
	case "no_viable_route":
		return 503
	default:
		return 400
	}
}

func rejectMessage(req domain.GatewayRequest, code string) string {
	switch code {
	case "claude_native_features_require_claude_route":
		return "当前请求包含 Claude 原生能力，未找到可安全执行的 Claude 路由"
	case "gemini_native_features_require_gemini_route":
		return "当前请求包含 Gemini 原生能力，未找到可安全执行的 Gemini 路由"
	case "responses_native_features_require_openai_route":
		return "当前请求包含 Responses 原生能力，未找到可安全执行的 OpenAI Responses 路由"
	case "openai_native_features_require_openai_route":
		return "当前请求包含 OpenAI 原生能力，未找到可安全执行的 OpenAI 路由"
	case "route_capability_not_supported":
		return "当前请求能力超出候选路由支持范围"
	case "target_operation_not_supported":
		return "当前请求无法规划到受支持的目标协议面"
	case "gemini_url_context_with_function_calling_unsupported":
		return "Gemini URL Context 与函数调用当前不存在安全转换路径"
	case "claude_thinking_with_forced_tool_choice_unsupported":
		return "Claude thinking 与强制 tool choice 当前不存在安全转换路径"
	case "no_viable_route":
		return fmt.Sprintf("模型 %s 没有可执行的安全转换路径", req.Model)
	default:
		return "当前请求不存在可安全执行的转换路径"
	}
}
