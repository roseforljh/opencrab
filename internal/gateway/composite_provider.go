package gateway

import "context"

type compositeProvider struct {
	chatProvider            ChatCompletionsProvider
	responsesProvider       ResponsesProvider
	messagesProvider        MessagesProvider
	generateContentProvider GenerateContentProvider
}

func NewCompositeProvider(chatProvider ChatCompletionsProvider, responsesProvider ResponsesProvider, messagesProvider MessagesProvider, generateContentProvider GenerateContentProvider) Provider {
	return &compositeProvider{
		chatProvider:            chatProvider,
		responsesProvider:       responsesProvider,
		messagesProvider:        messagesProvider,
		generateContentProvider: generateContentProvider,
	}
}

func (p *compositeProvider) ChatCompletions(ctx context.Context, request ChatCompletionsRequest) (*ProxyResponse, error) {
	return p.chatProvider.ChatCompletions(ctx, request)
}

func (p *compositeProvider) Responses(ctx context.Context, request ResponsesRequest) (*ProxyResponse, error) {
	return p.responsesProvider.Responses(ctx, request)
}

func (p *compositeProvider) Messages(ctx context.Context, request MessagesRequest) (*ProxyResponse, error) {
	if request.UpstreamFamily == "openai" && request.UpstreamOperation == openAIOperationResponses {
		return bridgeClaudeMessagesToOpenAIResponses(ctx, p.responsesProvider, request)
	}
	if request.UpstreamFamily == "openai" {
		return bridgeClaudeMessagesToOpenAI(ctx, p.chatProvider, request)
	}
	return p.messagesProvider.Messages(ctx, request)
}

func (p *compositeProvider) GenerateContent(ctx context.Context, request GenerateContentRequest) (*ProxyResponse, error) {
	return p.generateContentProvider.GenerateContent(ctx, request)
}
