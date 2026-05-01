package httpserver

import "testing"

func TestParseClaudeUsageFromBridgedSSE(t *testing.T) {
	body := []byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":7,\"output_tokens\":0}}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":3}}\n\n")

	inputTokens, outputTokens, totalTokens := parseClaudeUsage(body)

	if inputTokens != 7 || outputTokens != 3 || totalTokens != 10 {
		t.Fatalf("expected bridged SSE usage 7/3/10, got %d/%d/%d", inputTokens, outputTokens, totalTokens)
	}
}
