package domain

import "context"

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
