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
	got := buildRuntimeUpstreamURL(runtimeRouteFamilyGemini, "https://generativelanguage.googleapis.com/v1beta", "gemini-2.5-pro")
	want := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
