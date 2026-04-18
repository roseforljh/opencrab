package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Protocol string

const (
	ProtocolOpenAI Protocol = "openai"
	ProtocolClaude Protocol = "claude"
	ProtocolGemini Protocol = "gemini"
)

type ProtocolOperation string

const (
	ProtocolOperationOpenAIChatCompletions ProtocolOperation = "chat_completions"
	ProtocolOperationOpenAIResponses       ProtocolOperation = "responses"
	ProtocolOperationOpenAIRealtime        ProtocolOperation = "realtime"
	ProtocolOperationClaudeMessages        ProtocolOperation = "messages"
	ProtocolOperationClaudeCountTokens     ProtocolOperation = "count_tokens"
	ProtocolOperationGeminiGenerateContent ProtocolOperation = "generate_content"
	ProtocolOperationGeminiStreamGenerate  ProtocolOperation = "stream_generate_content"
)

type UnifiedChatRequest struct {
	Protocol Protocol `json:"protocol"`
	Model    string   `json:"model"`
	Stream   bool     `json:"stream,omitempty"`

	Messages []UnifiedMessage           `json:"messages"`
	Tools    []json.RawMessage          `json:"tools,omitempty"`
	Metadata map[string]json.RawMessage `json:"metadata,omitempty"`
}

type UnifiedMessage struct {
	Role      string                     `json:"role"`
	Parts     []UnifiedPart              `json:"parts"`
	ToolCalls []UnifiedToolCall          `json:"tool_calls,omitempty"`
	Metadata  map[string]json.RawMessage `json:"metadata,omitempty"`
}

type UnifiedPart struct {
	Type     string                     `json:"type"`
	Text     string                     `json:"text,omitempty"`
	Metadata map[string]json.RawMessage `json:"metadata,omitempty"`
}

type UnifiedChatResponse struct {
	Protocol     Protocol                   `json:"protocol"`
	ID           string                     `json:"id,omitempty"`
	Model        string                     `json:"model,omitempty"`
	FinishReason string                     `json:"finish_reason,omitempty"`
	Message      UnifiedMessage             `json:"message"`
	Usage        map[string]int64           `json:"usage,omitempty"`
	Metadata     map[string]json.RawMessage `json:"metadata,omitempty"`
}

type UnifiedToolCall struct {
	ID        string                     `json:"id,omitempty"`
	Name      string                     `json:"name"`
	Arguments json.RawMessage            `json:"arguments,omitempty"`
	Metadata  map[string]json.RawMessage `json:"metadata,omitempty"`
}

type UnifiedStreamEvent struct {
	Protocol     Protocol                   `json:"protocol"`
	Type         string                     `json:"type"`
	Delta        *UnifiedMessage            `json:"delta,omitempty"`
	FinishReason string                     `json:"finish_reason,omitempty"`
	Metadata     map[string]json.RawMessage `json:"metadata,omitempty"`
}

type GatewayError struct {
	Protocol   Protocol                   `json:"protocol"`
	StatusCode int                        `json:"status_code,omitempty"`
	Code       string                     `json:"code,omitempty"`
	Message    string                     `json:"message"`
	Metadata   map[string]json.RawMessage `json:"metadata,omitempty"`
}

func (e *GatewayError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (r UnifiedChatRequest) ValidateCore() error {
	if strings.TrimSpace(r.Model) == "" {
		return fmt.Errorf("model 不能为空")
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}

	for i, message := range r.Messages {
		switch strings.TrimSpace(message.Role) {
		case "system", "user", "assistant", "tool":
		default:
			return fmt.Errorf("messages[%d].role 非法: %s", i, message.Role)
		}

		if len(message.Parts) == 0 {
			if len(message.ToolCalls) == 0 {
				return fmt.Errorf("messages[%d].parts 不能为空", i)
			}
		}

		for j, part := range message.Parts {
			if strings.TrimSpace(part.Type) == "" {
				return fmt.Errorf("messages[%d].parts[%d].type 不能为空", i, j)
			}
			if strings.TrimSpace(part.Type) == "text" && strings.TrimSpace(part.Text) == "" {
				return fmt.Errorf("messages[%d].parts[%d].text 不能为空", i, j)
			}
		}

		for j, toolCall := range message.ToolCalls {
			if strings.TrimSpace(toolCall.Name) == "" {
				return fmt.Errorf("messages[%d].tool_calls[%d].name 不能为空", i, j)
			}
		}
	}

	return nil
}
