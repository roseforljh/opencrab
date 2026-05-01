package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
)

var supportedAnthropicGeminiBridgeFields = map[string]struct{}{
	"max_tokens":     {},
	"messages":       {},
	"model":          {},
	"stop_sequences": {},
	"stream":         {},
	"system":         {},
	"temperature":    {},
	"thinking":       {},
	"tool_choice":    {},
	"tools":          {},
	"top_k":          {},
	"top_p":          {},
}

func ResolveClaudeMessagesGeminiCompatibility(body []byte) error {
	_, err := buildGeminiGenerateContentBody(body)
	return err
}

func unsupportedGeminiFieldError(field string) error {
	field = strings.TrimSpace(field)
	if field == "" {
		field = "field"
	}
	return &RequestError{StatusCode: 400, Message: fmt.Sprintf("%s is not supported when routing Claude Messages through Gemini", field), UpstreamFamily: "gemini"}
}

func bridgeSystemToGeminiInstruction(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}
	return bridgeAnthropicTextContentForGemini(raw, "system")
}

func bridgeAnthropicTextContentForGemini(raw json.RawMessage, field string) (string, error) {
	blocks, err := decodeAnthropicContent(raw)
	if err == nil {
		parts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			switch block.Type {
			case "", "text":
				parts = append(parts, block.Text)
			case "thinking", "redacted_thinking":
				return "", unsupportedGeminiFieldError(block.Type)
			default:
				return "", &RequestError{StatusCode: 400, Message: fmt.Sprintf("%s block type %q is not supported when routing Claude Messages through Gemini", field, block.Type), UpstreamFamily: "gemini"}
			}
		}
		return strings.Join(parts, "\n"), nil
	}
	var text string
	if unmarshalErr := json.Unmarshal(raw, &text); unmarshalErr == nil {
		return text, nil
	}
	return "", &RequestError{StatusCode: 400, Message: fmt.Sprintf("%s is not supported when routing Claude Messages through Gemini", field), UpstreamFamily: "gemini"}
}
