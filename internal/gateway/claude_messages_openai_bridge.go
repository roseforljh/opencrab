package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func bridgeClaudeMessagesToOpenAI(ctx context.Context, chatProvider ChatCompletionsProvider, request MessagesRequest) (*ProxyResponse, error) {
	body, err := buildOpenAIChatCompletionsBody(request.Body)
	if err != nil {
		return nil, err
	}
	response, err := chatProvider.ChatCompletions(ctx, ChatCompletionsRequest{
		Model:          request.Model,
		Stream:         request.Stream,
		Body:           body,
		ContentType:    "application/json",
		Accept:         request.Accept,
		Authorization:  request.Authorization,
		Headers:        request.Headers,
		UpstreamFamily: "openai",
		UpstreamURL:    request.UpstreamURL,
		UpstreamAPIKey: request.UpstreamAPIKey,
	})
	if err != nil {
		requestError := &RequestError{}
		if errorsAsRequestError(err, requestError) {
			requestError.UpstreamFamily = "openai"
		}
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	response.UpstreamFamily = "openai"
	if response.Stream {
		response.Header = cloneHeaderWithSSEContentType(response.Header)
		response.Body = convertOpenAIStreamToClaudeStream(response.Body)
		return response, nil
	}
	convertedBody, header, err := convertOpenAIResponseToClaudeResponse(response.Body, response.StatusCode, response.Header)
	if err != nil {
		return nil, err
	}
	response.Header = header
	response.Body = io.NopCloser(bytes.NewReader(convertedBody))
	return response, nil
}

type anthropicBridgeRequest struct {
	Model         string                   `json:"model"`
	MaxTokens     int                      `json:"max_tokens"`
	Stream        bool                     `json:"stream,omitempty"`
	System        json.RawMessage          `json:"system,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
	Temperature   *float64                 `json:"temperature,omitempty"`
	TopP          *float64                 `json:"top_p,omitempty"`
	Messages      []anthropicBridgeMessage `json:"messages"`
	Metadata      json.RawMessage          `json:"metadata,omitempty"`
	Tools         []anthropicBridgeTool    `json:"tools,omitempty"`
	ToolChoice    json.RawMessage          `json:"tool_choice,omitempty"`
	Thinking      json.RawMessage          `json:"thinking,omitempty"`
}

type anthropicBridgeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicBridgeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Source    json.RawMessage `json:"source,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

func buildOpenAIChatCompletionsBody(body []byte) ([]byte, error) {
	fields, err := parseAnthropicBridgeRequest(body)
	if err != nil {
		return nil, err
	}
	request := fields.Request
	for key := range fields.Raw {
		if _, ok := supportedAnthropicBridgeFields[key]; !ok {
			return nil, unsupportedOpenAIFieldError(key)
		}
	}
	if hasMeaningfulRawJSON(fields.Raw["thinking"]) {
		return nil, unsupportedOpenAIFieldError("thinking")
	}
	if hasMeaningfulRawJSON(fields.Raw["top_k"]) {
		return nil, unsupportedOpenAIFieldError("top_k")
	}
	if hasMeaningfulRawJSON(fields.Raw["metadata"]) {
		return nil, unsupportedOpenAIFieldError("metadata")
	}
	messages := make([]map[string]any, 0, len(request.Messages)+1)
	if systemText, err := bridgeSystemToOpenAIMessages(request.System); err != nil {
		return nil, err
	} else if systemText != "" {
		messages = append(messages, map[string]any{"role": "system", "content": systemText})
	}
	for index, message := range request.Messages {
		mapped, err := bridgeMessageToOpenAIMessages(message, index)
		if err != nil {
			return nil, err
		}
		messages = append(messages, mapped...)
	}
	payload := map[string]any{
		"model":      request.Model,
		"max_tokens": request.MaxTokens,
		"stream":     request.Stream,
		"messages":   messages,
	}
	if len(request.StopSequences) > 0 {
		payload["stop"] = request.StopSequences
	}
	if request.Temperature != nil {
		payload["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		payload["top_p"] = *request.TopP
	}
	if len(request.Tools) > 0 {
		tools := make([]map[string]any, 0, len(request.Tools))
		for _, tool := range request.Tools {
			item := map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": tool.Name,
				},
			}
			if strings.TrimSpace(tool.Description) != "" {
				item["function"].(map[string]any)["description"] = tool.Description
			}
			if len(bytes.TrimSpace(tool.InputSchema)) > 0 {
				var schema any
				if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
					return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tools.input_schema must be valid JSON", UpstreamFamily: "openai"}
				}
				item["function"].(map[string]any)["parameters"] = schema
			}
			tools = append(tools, item)
		}
		payload["tools"] = tools
	}
	if len(bytes.TrimSpace(request.ToolChoice)) > 0 && string(bytes.TrimSpace(request.ToolChoice)) != "null" {
		choice, err := bridgeToolChoiceToOpenAI(request.ToolChoice)
		if err != nil {
			return nil, err
		}
		payload["tool_choice"] = choice
	}
	return json.Marshal(payload)
}

func bridgeSystemToOpenAIMessages(raw json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return "", nil
	}
	return bridgeContentText(raw, "system")
}

func bridgeMessageToOpenAIMessages(message anthropicBridgeMessage, index int) ([]map[string]any, error) {
	role := strings.TrimSpace(message.Role)
	blocks, err := decodeAnthropicContent(message.Content)
	if err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content must be a string or content block array", index), UpstreamFamily: "openai"}
	}
	switch role {
	case "user":
		return bridgeUserMessageToOpenAI(blocks, index)
	case "assistant":
		return bridgeAssistantMessageToOpenAI(blocks, index)
	default:
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].role %q is not supported when routing Claude Messages through OpenAI chat completions", index, role), UpstreamFamily: "openai"}
	}
}

func bridgeUserMessageToOpenAI(blocks []anthropicContentBlock, index int) ([]map[string]any, error) {
	texts := make([]string, 0)
	messages := make([]map[string]any, 0, 1)
	flushText := func() {
		if len(texts) == 0 {
			return
		}
		messages = append(messages, map[string]any{"role": "user", "content": strings.Join(texts, "\n")})
		texts = nil
	}
	for _, block := range blocks {
		switch block.Type {
		case "text":
			texts = append(texts, block.Text)
		case "tool_result":
			flushText()
			if strings.TrimSpace(block.ToolUseID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_result.tool_use_id is required when routing Claude Messages through OpenAI chat completions", index), UpstreamFamily: "openai"}
			}
			content, err := bridgeContentText(block.Content, fmt.Sprintf("messages[%d].content.tool_result", index))
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": block.ToolUseID, "content": content})
		case "thinking", "redacted_thinking":
			return nil, newUnsupportedFallbackFieldError(block.Type)
		default:
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through OpenAI chat completions", index, block.Type), UpstreamFamily: "openai"}
		}
	}
	flushText()
	if len(messages) == 0 {
		messages = append(messages, map[string]any{"role": "user", "content": ""})
	}
	return messages, nil
}

func bridgeAssistantMessageToOpenAI(blocks []anthropicContentBlock, index int) ([]map[string]any, error) {
	texts := make([]string, 0)
	toolCalls := make([]map[string]any, 0)
	for _, block := range blocks {
		switch block.Type {
		case "text":
			texts = append(texts, block.Text)
		case "tool_use":
			if strings.TrimSpace(block.ID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.id is required when routing Claude Messages through OpenAI chat completions", index), UpstreamFamily: "openai"}
			}
			if strings.TrimSpace(block.Name) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.name is required when routing Claude Messages through OpenAI chat completions", index), UpstreamFamily: "openai"}
			}
			arguments := "{}"
			if len(bytes.TrimSpace(block.Input)) > 0 {
				arguments = string(bytes.TrimSpace(block.Input))
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   block.ID,
				"type": "function",
				"function": map[string]any{
					"name":      block.Name,
					"arguments": arguments,
				},
			})
		case "thinking", "redacted_thinking":
			return nil, newUnsupportedFallbackFieldError(block.Type)
		default:
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through OpenAI chat completions", index, block.Type), UpstreamFamily: "openai"}
		}
	}
	message := map[string]any{"role": "assistant"}
	if len(texts) > 0 {
		message["content"] = strings.Join(texts, "\n")
	} else {
		message["content"] = ""
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}
	return []map[string]any{message}, nil
}

func bridgeToolChoiceToOpenAI(raw json.RawMessage) (any, error) {
	var payload struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tool_choice must be valid JSON", UpstreamFamily: "openai"}
	}
	switch strings.TrimSpace(payload.Type) {
	case "auto", "none":
		return payload.Type, nil
	case "any":
		return "required", nil
	case "tool":
		return map[string]any{"type": "function", "function": map[string]any{"name": payload.Name}}, nil
	default:
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("tool_choice type %q is not supported when routing Claude Messages through OpenAI chat completions", payload.Type), UpstreamFamily: "openai"}
	}
}

func decodeAnthropicContent(raw json.RawMessage) ([]anthropicContentBlock, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return nil, err
		}
		return []anthropicContentBlock{{Type: "text", Text: text}}, nil
	}
	var blocks []anthropicContentBlock
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func bridgeContentText(raw json.RawMessage, field string) (string, error) {
	blocks, err := decodeAnthropicContent(raw)
	if err == nil {
		return bridgeBlocksToText(blocks, field)
	}
	var text string
	if unmarshalErr := json.Unmarshal(raw, &text); unmarshalErr == nil {
		return text, nil
	}
	return "", &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("%s is not supported when routing Claude Messages through OpenAI chat completions", field), UpstreamFamily: "openai"}
}

func bridgeBlocksToText(blocks []anthropicContentBlock, field string) (string, error) {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			parts = append(parts, block.Text)
		case "thinking", "redacted_thinking":
			return "", newUnsupportedFallbackFieldError(block.Type)
		default:
			return "", &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("%s block type %q is not supported when routing Claude Messages through OpenAI chat completions", field, block.Type), UpstreamFamily: "openai"}
		}
	}
	return strings.Join(parts, "\n"), nil
}

func newUnsupportedFallbackFieldError(field string) error {
	return &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("%s is not supported when routing Claude Messages through OpenAI chat completions", field), UpstreamFamily: "openai"}
}

type openAIResponseEnvelope struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content   any `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func convertOpenAIResponseToClaudeResponse(body io.ReadCloser, statusCode int, header http.Header) ([]byte, http.Header, error) {
	defer drainProxyBody(body)
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	convertedHeader := cloneHeaderWithJSONContentType(header)
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return marshalAnthropicErrorFromOpenAI(payload, statusCode), convertedHeader, nil
	}
	var response openAIResponseEnvelope
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, nil, err
	}
	content := make([]map[string]any, 0)
	if len(response.Choices) > 0 {
		choice := response.Choices[0]
		content = append(content, bridgeOpenAIContentToClaude(choice.Message.Content)...)
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    toolCall.ID,
				"name":  toolCall.Function.Name,
				"input": decodeOpenAIToolArguments(toolCall.Function.Arguments),
			})
		}
	}
	converted := map[string]any{
		"id":            response.ID,
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         response.Model,
		"stop_reason":   bridgeStopReason(response.Choices),
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  response.Usage.PromptTokens,
			"output_tokens": response.Usage.CompletionTokens,
		},
	}
	convertedBody, err := json.Marshal(converted)
	if err != nil {
		return nil, nil, err
	}
	return convertedBody, convertedHeader, nil
}

func bridgeOpenAIContentToClaude(content any) []map[string]any {
	blocks := make([]map[string]any, 0)
	switch value := content.(type) {
	case string:
		if value != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": value})
		}
	case []any:
		for _, item := range value {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(stringValue(part["type"])) != "text" {
				continue
			}
			text := stringValue(part["text"])
			if text == "" {
				continue
			}
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
		}
	}
	return blocks
}

func bridgeStopReason(choices []struct {
	Message struct {
		Content   any `json:"content"`
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}) string {
	if len(choices) == 0 {
		return "end_turn"
	}
	return mapOpenAIFinishReason(choices[0].FinishReason)
}

func mapOpenAIFinishReason(reason string) string {
	switch reason {
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "stop", "content_filter", "":
		return "end_turn"
	default:
		return "end_turn"
	}
}

func decodeOpenAIToolArguments(arguments string) any {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return map[string]any{}
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return map[string]any{"raw": arguments}
	}
	return payload
}

func marshalAnthropicErrorFromOpenAI(body []byte, statusCode int) []byte {
	message := http.StatusText(statusCode)
	errorType := "api_error"
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			message = payload.Error.Message
		}
		errorType = mapOpenAIErrorType(statusCode, payload.Error.Type)
	}
	converted, _ := json.Marshal(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errorType,
			"message": message,
		},
	})
	return converted
}

func mapOpenAIErrorType(statusCode int, openAIType string) string {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusGatewayTimeout:
		return "timeout_error"
	case 529:
		return "overloaded_error"
	}
	switch strings.ToLower(strings.TrimSpace(openAIType)) {
	case "invalid_request_error":
		return "invalid_request_error"
	case "authentication_error":
		return "authentication_error"
	case "permission_error":
		return "permission_error"
	case "not_found_error":
		return "not_found_error"
	case "rate_limit_error":
		return "rate_limit_error"
	case "overloaded_error":
		return "overloaded_error"
	}
	return "api_error"
}

type openAIStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func convertOpenAIStreamToClaudeStream(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer drainProxyBody(body)
		defer writer.Close()
		if err := writeClaudeStreamFromOpenAI(body, writer); err != nil {
			_ = writer.CloseWithError(err)
		}
	}()
	return reader
}

func writeClaudeStreamFromOpenAI(body io.Reader, writer *io.PipeWriter) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var eventLines []string
	state := claudeStreamState{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := handleOpenAIStreamEvent(eventLines, writer, &state); err != nil {
				return err
			}
			eventLines = eventLines[:0]
			continue
		}
		eventLines = append(eventLines, line)
	}
	if len(eventLines) > 0 {
		if err := handleOpenAIStreamEvent(eventLines, writer, &state); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if state.messageStarted && !state.messageStopped {
		if state.blockStarted {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
		}
		return writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"})
	}
	return nil
}

type claudeStreamState struct {
	messageStarted bool
	blockStarted   bool
	messageStopped bool
	blockType      string
	blockIndex     int
	messageID      string
	model          string
	toolBlocks     map[int]claudeToolBlockState
}

type claudeToolBlockState struct {
	id   string
	name string
}

func handleOpenAIStreamEvent(lines []string, writer *io.PipeWriter, state *claudeStreamState) error {
	if len(lines) == 0 {
		return nil
	}
	dataLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
	}
	if len(dataLines) == 0 {
		return nil
	}
	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		if state.messageStarted && !state.messageStopped {
			if state.blockStarted {
				if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
					return err
				}
				state.blockStarted = false
			}
			if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
				return err
			}
			state.messageStopped = true
		}
		return nil
	}
	var chunk openAIStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return err
	}
	if !state.messageStarted {
		state.messageStarted = true
		state.messageID = chunk.ID
		state.model = chunk.Model
		if err := writeAnthropicStreamEvent(writer, "message_start", map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":            chunk.ID,
				"type":          "message",
				"role":          "assistant",
				"content":       []any{},
				"model":         chunk.Model,
				"stop_reason":   nil,
				"stop_sequence": nil,
				"usage": map[string]any{
					"input_tokens":  0,
					"output_tokens": 0,
				},
			},
		}); err != nil {
			return err
		}
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]
	if state.toolBlocks == nil {
		state.toolBlocks = map[int]claudeToolBlockState{}
	}
	if strings.TrimSpace(choice.Delta.Content) != "" {
		if state.blockStarted && state.blockType != "text" {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		if !state.blockStarted {
			state.blockStarted = true
			state.blockType = "text"
			state.blockIndex = 0
			if err := writeAnthropicStreamEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": state.blockIndex, "content_block": map[string]any{"type": "text", "text": ""}}); err != nil {
				return err
			}
		}
		if err := writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": state.blockIndex, "delta": map[string]any{"type": "text_delta", "text": choice.Delta.Content}}); err != nil {
			return err
		}
	}
	for _, toolCall := range choice.Delta.ToolCalls {
		toolState := state.toolBlocks[toolCall.Index]
		if strings.TrimSpace(toolCall.ID) != "" {
			toolState.id = toolCall.ID
		}
		if strings.TrimSpace(toolCall.Function.Name) != "" {
			toolState.name = toolCall.Function.Name
		}
		state.toolBlocks[toolCall.Index] = toolState
		if state.blockStarted && (state.blockType != "tool_use" || state.blockIndex != toolCall.Index) {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		if !state.blockStarted {
			state.blockStarted = true
			state.blockType = "tool_use"
			state.blockIndex = toolCall.Index
			if err := writeAnthropicStreamEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": state.blockIndex, "content_block": map[string]any{"type": "tool_use", "id": toolState.id, "name": toolState.name, "input": map[string]any{}}}); err != nil {
				return err
			}
		}
		if strings.TrimSpace(toolCall.Function.Arguments) != "" {
			if err := writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": state.blockIndex, "delta": map[string]any{"type": "input_json_delta", "partial_json": toolCall.Function.Arguments}}); err != nil {
				return err
			}
		}
	}
	if choice.FinishReason != nil {
		if state.blockStarted {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		if err := writeAnthropicStreamEvent(writer, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": mapOpenAIFinishReason(*choice.FinishReason), "stop_sequence": nil}, "usage": map[string]any{"input_tokens": chunk.Usage.PromptTokens, "output_tokens": chunk.Usage.CompletionTokens}}); err != nil {
			return err
		}
		if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
			return err
		}
		state.messageStopped = true
	}
	return nil
}

func writeAnthropicStreamEvent(writer *io.PipeWriter, event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("event: " + event + "\ndata: " + string(body) + "\n\n"))
	return err
}

func cloneHeaderWithJSONContentType(header http.Header) http.Header {
	clone := header.Clone()
	clone.Set("Content-Type", "application/json")
	clone.Del("Content-Length")
	return clone
}

func cloneHeaderWithSSEContentType(header http.Header) http.Header {
	clone := header.Clone()
	clone.Set("Content-Type", "text/event-stream")
	clone.Del("Content-Length")
	return clone
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func errorsAsRequestError(err error, target *RequestError) bool {
	if err == nil || target == nil {
		return false
	}
	requestError, ok := err.(*RequestError)
	if !ok {
		return false
	}
	*target = *requestError
	return true
}
