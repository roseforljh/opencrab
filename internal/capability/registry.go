package capability

import (
	"context"

	"opencrab/internal/domain"
)

type ScopeType string

const (
	ScopeTypeProviderDefault ScopeType = "provider_default"
	ScopeTypeChannelOverride ScopeType = "channel_override"
	ScopeTypeModelProfile    ScopeType = "model_profile"
)

type ProfileRecord struct {
	ScopeType    ScopeType
	ScopeKey     string
	Operation    domain.ProtocolOperation
	Enabled      *bool
	Capabilities []Capability
}

type Loader interface {
	ListCapabilityProfiles(ctx context.Context) ([]ProfileRecord, error)
}

type Registry struct {
	loader Loader
}

func NewRegistry(loader Loader) *Registry {
	return &Registry{loader: loader}
}

func (r *Registry) Surface(ctx context.Context, route domain.GatewayRoute, operation domain.ProtocolOperation) (map[Capability]struct{}, bool, error) {
	matrix := providerSupportMatrix(domain.NormalizeProvider(route.Channel.Provider))
	surface, ok := matrix.operations[operation]
	if !ok {
		return nil, false, nil
	}

	enabled := true
	capabilities := cloneCapabilitySet(surface.capabilities)
	if r == nil || r.loader == nil {
		return capabilities, enabled, nil
	}

	records, err := r.loader.ListCapabilityProfiles(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, record := range records {
		if record.Operation != operation {
			continue
		}
		switch record.ScopeType {
		case ScopeTypeProviderDefault:
			if record.ScopeKey == domain.NormalizeProvider(route.Channel.Provider) {
				enabled, capabilities = applyProfileRecord(enabled, capabilities, record)
			}
		case ScopeTypeChannelOverride:
			if record.ScopeKey == route.Channel.Name {
				enabled, capabilities = applyProfileRecord(enabled, capabilities, record)
			}
		case ScopeTypeModelProfile:
			if record.ScopeKey == route.ModelAlias {
				enabled, capabilities = applyProfileRecord(enabled, capabilities, record)
			}
		}
	}

	return capabilities, enabled, nil
}

func applyProfileRecord(enabled bool, capabilities map[Capability]struct{}, record ProfileRecord) (bool, map[Capability]struct{}) {
	if record.Enabled != nil {
		enabled = *record.Enabled
	}
	if record.Capabilities != nil {
		capabilities = capabilitySet(record.Capabilities...)
	}
	return enabled, capabilities
}

func cloneCapabilitySet(input map[Capability]struct{}) map[Capability]struct{} {
	output := make(map[Capability]struct{}, len(input))
	for key := range input {
		output[key] = struct{}{}
	}
	return output
}
