package httpserver

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"opencrab/internal/gateway"
)

const (
	runtimeRouteFamilyOpenAI = "openai"
	runtimeRouteFamilyClaude = "claude"
	runtimeRouteFamilyGemini = "gemini"
	runtimeRouteOperationChatCompletions = "chat_completions"
	runtimeRouteOperationResponses       = "responses"
)

func resolveChatCompletionsRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	return resolveRuntimeRouteCandidates(runtimeRouteFamilyOpenAI, runtimeRouteOperationChatCompletions, model)
}

func resolveResponsesRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	return resolveRuntimeRouteCandidates(runtimeRouteFamilyOpenAI, runtimeRouteOperationResponses, model)
}

func resolveMessagesRoutes(model string, body []byte) ([]gateway.UpstreamRouteCandidate, error) {
	alias := strings.TrimSpace(model)
	if alias == "" {
		return nil, &gateway.RoutingError{Message: "Model is required"}
	}
	claudeChannels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyClaude, alias)
	openAIChannels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyOpenAI, alias)
	geminiChannels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyGemini, alias)
	openAIOperation, openAICompatibilityErr := gateway.ResolveClaudeMessagesOpenAIOperation(body)
	geminiCompatibilityErr := gateway.ResolveClaudeMessagesGeminiCompatibility(body)
	if len(claudeChannels) == 0 && len(openAIChannels) == 0 && len(geminiChannels) == 0 {
		return nil, &gateway.RoutingError{Message: fmt.Sprintf("No enabled claude route configured for model %s", alias)}
	}
	stream := requestsStream(body)
	items := make([]gateway.UpstreamRouteCandidate, 0, len(claudeChannels)+len(openAIChannels)+len(geminiChannels))
	for _, channel := range claudeChannels {
		items = append(items, gateway.UpstreamRouteCandidate{
			Family: runtimeRouteFamilyClaude,
			URL:    buildRuntimeUpstreamURL(runtimeRouteFamilyClaude, "", channel.Endpoint, alias),
			APIKey: channel.APIKey,
		})
	}
	if openAICompatibilityErr == nil {
		for _, channel := range openAIChannels {
			items = append(items, gateway.UpstreamRouteCandidate{
				Family:    runtimeRouteFamilyOpenAI,
				Operation: openAIOperation,
				URL:       buildRuntimeUpstreamURL(runtimeRouteFamilyOpenAI, openAIOperation, channel.Endpoint, alias),
				APIKey:    channel.APIKey,
			})
		}
	}
	if geminiCompatibilityErr == nil {
		for _, channel := range geminiChannels {
			url := buildRuntimeUpstreamURL(runtimeRouteFamilyGemini, "", channel.Endpoint, alias)
			if stream {
				url = buildRuntimeGeminiStreamURL(channel.Endpoint, alias)
			}
			items = append(items, gateway.UpstreamRouteCandidate{
				Family: runtimeRouteFamilyGemini,
				URL:    url,
				APIKey: channel.APIKey,
			})
		}
	}
	if len(items) > 0 {
		return items, nil
	}
	if len(openAIChannels) > 0 && openAICompatibilityErr != nil {
		return nil, openAICompatibilityErr
	}
	if len(geminiChannels) > 0 && geminiCompatibilityErr != nil {
		return nil, geminiCompatibilityErr
	}
	return items, nil
}

func resolveGeminiGenerateContentRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	return resolveRuntimeRouteCandidates(runtimeRouteFamilyGemini, "", model)
}

func resolveGeminiStreamGenerateContentRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	alias := strings.TrimSpace(model)
	if alias == "" {
		return nil, &gateway.RoutingError{Message: "Model is required"}
	}
	channels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyGemini, alias)
	if len(channels) == 0 {
		return nil, &gateway.RoutingError{Message: fmt.Sprintf("No enabled gemini route configured for model %s", alias)}
	}
	items := make([]gateway.UpstreamRouteCandidate, 0, len(channels))
	for _, channel := range channels {
		items = append(items, gateway.UpstreamRouteCandidate{
			URL:    buildRuntimeGeminiStreamURL(channel.Endpoint, alias),
			APIKey: channel.APIKey,
		})
	}
	return items, nil
}

func resolveRuntimeRouteCandidates(family string, operation string, model string) ([]gateway.UpstreamRouteCandidate, error) {
	alias := strings.TrimSpace(model)
	if alias == "" {
		return nil, &gateway.RoutingError{Message: "Model is required"}
	}
	channels := compatChannels.resolveRuntimeChannels(family, alias)
	if len(channels) == 0 {
		return nil, &gateway.RoutingError{Message: fmt.Sprintf("No enabled %s route configured for model %s", family, alias)}
	}
	items := make([]gateway.UpstreamRouteCandidate, 0, len(channels))
	for _, channel := range channels {
		items = append(items, gateway.UpstreamRouteCandidate{
			Family:    family,
			Operation: operation,
			URL:       buildRuntimeUpstreamURL(family, operation, channel.Endpoint, alias),
			APIKey:    channel.APIKey,
		})
	}
	return items, nil
}

func buildRuntimeUpstreamURL(family string, operation string, endpoint string, model string) string {
	switch family {
	case runtimeRouteFamilyClaude:
		return joinURL(endpoint, "/v1/messages")
	case runtimeRouteFamilyGemini:
		return joinURL(endpoint, fmt.Sprintf("/models/%s:generateContent", url.PathEscape(model)))
	default:
		if operation == runtimeRouteOperationResponses {
			return joinURL(normalizeOpenAICompatibleEndpoint(endpoint), "/responses")
		}
		return joinURL(normalizeOpenAICompatibleEndpoint(endpoint), "/chat/completions")
	}
}

func normalizeOpenAICompatibleEndpoint(endpoint string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if trimmed == "" {
		return trimmed
	}
	if strings.HasSuffix(strings.ToLower(trimmed), "/v1") {
		return trimmed
	}
	return trimmed + "/v1"
}

func buildRuntimeGeminiStreamURL(endpoint string, model string) string {
	return joinURL(endpoint, fmt.Sprintf("/models/%s:streamGenerateContent", url.PathEscape(model))) + "?alt=sse"
}

func normalizeRuntimeRouteFamily(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case runtimeRouteFamilyClaude:
		return runtimeRouteFamilyClaude
	case runtimeRouteFamilyGemini:
		return runtimeRouteFamilyGemini
	default:
		return runtimeRouteFamilyOpenAI
	}
}

func (s *adminCompatChannelStore) resolveRuntimeChannels(family string, model string) []*adminCompatChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	matched := make([]*adminCompatChannel, 0)
	for _, channel := range s.channels {
		if !channel.Enabled || normalizeRuntimeRouteFamily(channel.Provider) != family {
			continue
		}
		for _, modelID := range channel.ModelIDs {
			if strings.TrimSpace(modelID) != model {
				continue
			}
			clone := *channel
			clone.ModelIDs = append([]string(nil), channel.ModelIDs...)
			matched = append(matched, &clone)
			break
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].DispatchWeight != matched[j].DispatchWeight {
			return matched[i].DispatchWeight > matched[j].DispatchWeight
		}
		return matched[i].ID < matched[j].ID
	})
	return matched
}
