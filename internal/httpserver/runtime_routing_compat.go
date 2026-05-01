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
)

func resolveChatCompletionsRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	return resolveRuntimeRouteCandidates(runtimeRouteFamilyOpenAI, model)
}

func resolveMessagesRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	alias := strings.TrimSpace(model)
	if alias == "" {
		return nil, &gateway.RoutingError{Message: "Model is required"}
	}
	claudeChannels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyClaude, alias)
	openAIChannels := compatChannels.resolveRuntimeChannels(runtimeRouteFamilyOpenAI, alias)
	if len(claudeChannels) == 0 && len(openAIChannels) == 0 {
		return nil, &gateway.RoutingError{Message: fmt.Sprintf("No enabled claude route configured for model %s", alias)}
	}
	items := make([]gateway.UpstreamRouteCandidate, 0, len(claudeChannels)+len(openAIChannels))
	for _, channel := range claudeChannels {
		items = append(items, gateway.UpstreamRouteCandidate{
			Family: runtimeRouteFamilyClaude,
			URL:    buildRuntimeUpstreamURL(runtimeRouteFamilyClaude, channel.Endpoint, alias),
			APIKey: channel.APIKey,
		})
	}
	for _, channel := range openAIChannels {
		items = append(items, gateway.UpstreamRouteCandidate{
			Family: runtimeRouteFamilyOpenAI,
			URL:    buildRuntimeUpstreamURL(runtimeRouteFamilyOpenAI, channel.Endpoint, alias),
			APIKey: channel.APIKey,
		})
	}
	return items, nil
}

func resolveGeminiGenerateContentRoutes(model string) ([]gateway.UpstreamRouteCandidate, error) {
	return resolveRuntimeRouteCandidates(runtimeRouteFamilyGemini, model)
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

func resolveRuntimeRouteCandidates(family string, model string) ([]gateway.UpstreamRouteCandidate, error) {
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
			Family: family,
			URL:    buildRuntimeUpstreamURL(family, channel.Endpoint, alias),
			APIKey: channel.APIKey,
		})
	}
	return items, nil
}

func buildRuntimeUpstreamURL(family string, endpoint string, model string) string {
	switch family {
	case runtimeRouteFamilyClaude:
		return joinURL(endpoint, "/v1/messages")
	case runtimeRouteFamilyGemini:
		return joinURL(endpoint, fmt.Sprintf("/models/%s:generateContent", url.PathEscape(model)))
	default:
		return joinURL(endpoint, "/chat/completions")
	}
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
