package httpserver

import "testing"

func TestResolveRuntimeChannelsOrdersByWeightThenID(t *testing.T) {
	store := newAdminCompatChannelStore()
	store.channels[2] = &adminCompatChannel{ID: 2, Name: "low", Provider: "OpenAI", Endpoint: "https://low.example/v1", APIKey: "low-key", Enabled: true, ModelIDs: []string{"gpt-4.1"}, DispatchWeight: 50}
	store.channels[1] = &adminCompatChannel{ID: 1, Name: "high", Provider: "OpenAI", Endpoint: "https://high.example/v1", APIKey: "high-key", Enabled: true, ModelIDs: []string{"gpt-4.1"}, DispatchWeight: 100}

	items := store.resolveRuntimeChannels(runtimeRouteFamilyOpenAI, "gpt-4.1")
	if len(items) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(items))
	}
	if items[0].Name != "high" || items[1].Name != "low" {
		t.Fatalf("expected weight ordering, got %s then %s", items[0].Name, items[1].Name)
	}
}

func TestBuildRuntimeUpstreamURLForGemini(t *testing.T) {
	got := buildRuntimeUpstreamURL(runtimeRouteFamilyGemini, "", "https://generativelanguage.googleapis.com/v1beta", "gemini-2.5-pro")
	want := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveMessagesRoutesFiltersOpenAIByCapability(t *testing.T) {
	previous := compatChannels
	store := newAdminCompatChannelStore()
	store.channels[1] = &adminCompatChannel{ID: 1, Name: "claude", Provider: "Claude", Endpoint: "https://claude.example", APIKey: "claude-key", Enabled: true, ModelIDs: []string{"claude-sonnet-4-5"}, DispatchWeight: 100}
	store.channels[2] = &adminCompatChannel{ID: 2, Name: "openai", Provider: "OpenAI", Endpoint: "https://openai.example/v1", APIKey: "openai-key", Enabled: true, ModelIDs: []string{"claude-sonnet-4-5"}, DispatchWeight: 90}
	compatChannels = store
	defer func() { compatChannels = previous }()

	routes, err := resolveMessagesRoutes("claude-sonnet-4-5", []byte(`{"model":"claude-sonnet-4-5","max_tokens":128,"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("expected compatible request to resolve, got %v", err)
	}
	if len(routes) != 2 || routes[0].Family != runtimeRouteFamilyClaude || routes[1].Operation != runtimeRouteOperationChatCompletions {
		t.Fatalf("expected claude then openai chat routes, got %#v", routes)
	}

	routes, err = resolveMessagesRoutes("claude-sonnet-4-5", []byte(`{"model":"claude-sonnet-4-5","max_tokens":128,"metadata":{"source":"test"},"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("expected metadata request to resolve, got %v", err)
	}
	if len(routes) != 2 || routes[1].Operation != runtimeRouteOperationResponses {
		t.Fatalf("expected openai responses fallback, got %#v", routes)
	}

	routes, err = resolveMessagesRoutes("claude-sonnet-4-5", []byte(`{"model":"claude-sonnet-4-5","max_tokens":128,"top_k":3,"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("expected claude-only routing for incompatible openai request, got %v", err)
	}
	if len(routes) != 1 || routes[0].Family != runtimeRouteFamilyClaude {
		t.Fatalf("expected only claude route, got %#v", routes)
	}

	store = newAdminCompatChannelStore()
	store.channels[2] = &adminCompatChannel{ID: 2, Name: "openai", Provider: "OpenAI", Endpoint: "https://openai.example/v1", APIKey: "openai-key", Enabled: true, ModelIDs: []string{"claude-sonnet-4-5"}, DispatchWeight: 90}
	compatChannels = store
	_, err = resolveMessagesRoutes("claude-sonnet-4-5", []byte(`{"model":"claude-sonnet-4-5","max_tokens":128,"top_k":3,"messages":[{"role":"user","content":"ping"}]}`))
	if err == nil {
		t.Fatal("expected incompatible openai-only request to fail")
	}
}
