package capability

import (
	"encoding/json"
	"strings"

	"opencrab/internal/domain"
)

func hasRawMetadataKey(metadata map[string]json.RawMessage, key string) bool {
	if len(metadata) == 0 {
		return false
	}
	_, ok := metadata[key]
	return ok
}

func rawJSONContains(raw json.RawMessage, patterns ...string) bool {
	if len(raw) == 0 {
		return false
	}
	text := strings.ToLower(string(raw))
	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func addCapability(set map[Capability]struct{}, capability Capability) {
	set[capability] = struct{}{}
}

func firstPartMetadataKey(parts []domain.UnifiedPart, keys ...string) bool {
	for _, part := range parts {
		for _, key := range keys {
			if hasRawMetadataKey(part.Metadata, key) {
				return true
			}
		}
	}
	return false
}
