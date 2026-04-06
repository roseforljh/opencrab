package provider

import "testing"

func TestBuildChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expect   string
	}{
		{name: "base endpoint", endpoint: "https://api.example.com", expect: "https://api.example.com/v1/chat/completions"},
		{name: "v1 endpoint", endpoint: "https://api.example.com/v1", expect: "https://api.example.com/v1/chat/completions"},
		{name: "full endpoint", endpoint: "https://api.example.com/v1/chat/completions", expect: "https://api.example.com/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildChatCompletionsURL(tt.endpoint); got != tt.expect {
				t.Fatalf("expected %s, got %s", tt.expect, got)
			}
		})
	}
}
