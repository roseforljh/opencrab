package sqlite

import (
	"context"
	"testing"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
)

func TestCapabilityProfileStoreListCapabilityProfiles(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
INSERT INTO capability_profiles(scope_type, scope_key, operation, config_json, updated_at) VALUES
('provider_default', 'openai', 'chat_completions', '{"enabled":true,"capabilities":["function_tools","structured_outputs"]}', 'now'),
('channel_override', 'openai-main', 'responses', '{"enabled":false}', 'now');`)
	if err != nil {
		t.Fatalf("seed db: %v", err)
	}

	store := NewCapabilityProfileStore(db)
	items, err := store.ListCapabilityProfiles(context.Background())
	if err != nil {
		t.Fatalf("list capability profiles: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ScopeType != capability.ScopeTypeChannelOverride && items[1].ScopeType != capability.ScopeTypeChannelOverride {
		t.Fatalf("expected channel override item, got %#v", items)
	}
	foundProvider := false
	for _, item := range items {
		if item.ScopeType == capability.ScopeTypeProviderDefault && item.Operation == domain.ProtocolOperationOpenAIChatCompletions {
			foundProvider = true
			if len(item.Capabilities) != 2 {
				t.Fatalf("unexpected provider capabilities: %#v", item)
			}
		}
	}
	if !foundProvider {
		t.Fatalf("expected provider default item, got %#v", items)
	}
}
