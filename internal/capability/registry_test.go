package capability

import (
	"context"
	"testing"

	"opencrab/internal/domain"
)

type fakeLoader struct {
	items []ProfileRecord
	err   error
}

func (l fakeLoader) ListCapabilityProfiles(ctx context.Context) ([]ProfileRecord, error) {
	return l.items, l.err
}

func TestRegistryMergesProviderChannelAndModelOverrides(t *testing.T) {
	registry := NewRegistry(fakeLoader{items: []ProfileRecord{
		{
			ScopeType:    ScopeTypeProviderDefault,
			ScopeKey:     "openai",
			Operation:    domain.ProtocolOperationOpenAIChatCompletions,
			Capabilities: []Capability{CapabilityFunctionTools},
		},
		{
			ScopeType: ScopeTypeChannelOverride,
			ScopeKey:  "openai-main",
			Operation: domain.ProtocolOperationOpenAIChatCompletions,
			Enabled:   boolPtr(false),
		},
		{
			ScopeType:    ScopeTypeModelProfile,
			ScopeKey:     "gpt-5.4",
			Operation:    domain.ProtocolOperationOpenAIChatCompletions,
			Capabilities: []Capability{CapabilityFunctionTools, CapabilityStructuredOutputs},
		},
	}})

	capabilities, enabled, err := registry.Surface(context.Background(), domain.GatewayRoute{
		ModelAlias: "gpt-5.4",
		Channel:    domain.UpstreamChannel{Name: "openai-main", Provider: "openai"},
	}, domain.ProtocolOperationOpenAIChatCompletions)
	if err != nil {
		t.Fatalf("surface: %v", err)
	}
	if enabled {
		t.Fatalf("expected channel override to disable operation")
	}
	if _, ok := capabilities[CapabilityStructuredOutputs]; !ok {
		t.Fatalf("expected model override capabilities to apply, got %#v", capabilities)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
