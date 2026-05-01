package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	openAIOperationChatCompletions = "chat_completions"
	openAIOperationResponses       = "responses"
)

var supportedAnthropicBridgeFields = map[string]struct{}{
	"max_tokens":     {},
	"messages":       {},
	"metadata":       {},
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

func ResolveClaudeMessagesOpenAIOperation(body []byte) (string, error) {
	fields, err := parseAnthropicBridgeRequest(body)
	if err != nil {
		return "", err
	}
	for key := range fields.Raw {
		if _, ok := supportedAnthropicBridgeFields[key]; !ok {
			return "", unsupportedOpenAIFieldError(key)
		}
	}
	if hasMeaningfulRawJSON(fields.Raw["thinking"]) {
		return "", unsupportedOpenAIFieldError("thinking")
	}
	if hasMeaningfulRawJSON(fields.Raw["top_k"]) {
		return "", unsupportedOpenAIFieldError("top_k")
	}
	if _, err := bridgeSystemToOpenAIMessages(fields.Request.System); err != nil {
		return "", err
	}
	usesResponses := hasMeaningfulRawJSON(fields.Request.Metadata)
	if usesResponses {
		var metadata map[string]any
		if err := json.Unmarshal(fields.Request.Metadata, &metadata); err != nil {
			return "", &RequestError{StatusCode: 400, Message: "metadata must be a JSON object when routing Claude Messages through OpenAI Responses", UpstreamFamily: "openai"}
		}
	}
	for index, message := range fields.Request.Messages {
		blocks, err := decodeAnthropicContent(message.Content)
		if err != nil {
			return "", &RequestError{StatusCode: 400, Message: fmt.Sprintf("messages[%d].content must be a string or content block array", index), UpstreamFamily: "openai"}
		}
		for _, block := range blocks {
			switch block.Type {
			case "", "text", "tool_use", "tool_result":
			case "image":
				usesResponses = true
			case "thinking", "redacted_thinking":
				return "", unsupportedOpenAIFieldError(block.Type)
			default:
				return "", &RequestError{StatusCode: 400, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through OpenAI", index, block.Type), UpstreamFamily: "openai"}
			}
		}
	}
	if usesResponses && len(fields.Request.StopSequences) > 0 {
		return "", unsupportedOpenAIFieldError("stop_sequences")
	}
	if usesResponses {
		return openAIOperationResponses, nil
	}
	return openAIOperationChatCompletions, nil
}

type anthropicBridgeFields struct {
	Request anthropicBridgeRequest
	Raw     map[string]json.RawMessage
}

func parseAnthropicBridgeRequest(body []byte) (anthropicBridgeFields, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return anthropicBridgeFields{}, &RequestError{StatusCode: 400, Message: "Invalid JSON body", UpstreamFamily: "openai"}
	}
	var request anthropicBridgeRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return anthropicBridgeFields{}, &RequestError{StatusCode: 400, Message: "Invalid JSON body", UpstreamFamily: "openai"}
	}
	return anthropicBridgeFields{Request: request, Raw: raw}, nil
}

func hasMeaningfulRawJSON(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && string(trimmed) != "null"
}

func unsupportedOpenAIFieldError(field string) error {
	field = strings.TrimSpace(field)
	if field == "" {
		field = "field"
	}
	return &RequestError{StatusCode: 400, Message: fmt.Sprintf("%s is not supported when routing Claude Messages through OpenAI", field), UpstreamFamily: "openai"}
}
