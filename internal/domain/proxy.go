package domain

import "context"

type UpstreamChannel struct {
	Name     string
	Provider string
	Endpoint string
	APIKey   string
}

type ChatCompletionsRequest struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream,omitempty"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type ProxyResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type ChatProvider interface {
	ForwardChatCompletions(ctx context.Context, channel UpstreamChannel, body []byte) (*ProxyResponse, error)
}
