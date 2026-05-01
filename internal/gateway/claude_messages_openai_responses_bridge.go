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

func bridgeClaudeMessagesToOpenAIResponses(ctx context.Context, responsesProvider ResponsesProvider, request MessagesRequest) (*ProxyResponse, error) {
	body, err := buildOpenAIResponsesBody(request.Body)
	if err != nil {
		return nil, err
	}
	response, err := responsesProvider.Responses(ctx, ResponsesRequest{
		Model:             request.Model,
		Stream:            request.Stream,
		Body:              body,
		ContentType:       "application/json",
		Accept:            request.Accept,
		Authorization:     request.Authorization,
		Headers:           request.Headers,
		UpstreamFamily:    "openai",
		UpstreamOperation: openAIOperationResponses,
		UpstreamURL:       request.UpstreamURL,
		UpstreamAPIKey:    request.UpstreamAPIKey,
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
		response.Body = convertOpenAIResponsesStreamToClaudeStream(response.Body)
		return response, nil
	}
	convertedBody, header, err := convertOpenAIResponsesToClaudeResponse(response.Body, response.StatusCode, response.Header)
	if err != nil {
		return nil, err
	}
	response.Header = header
	response.Body = io.NopCloser(bytes.NewReader(convertedBody))
	return response, nil
}

func buildOpenAIResponsesBody(body []byte) ([]byte, error) {
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
	input := make([]map[string]any, 0, len(request.Messages)+1)
	if instructions, err := bridgeSystemToOpenAIMessages(request.System); err != nil {
		return nil, err
	} else if instructions != "" {
		input = append(input, map[string]any{"type": "message", "role": "system", "content": []map[string]any{{"type": "input_text", "text": instructions}}})
	}
	for index, message := range request.Messages {
		mapped, err := bridgeMessageToOpenAIResponsesInput(message, index)
		if err != nil {
			return nil, err
		}
		input = append(input, mapped...)
	}
	payload := map[string]any{
		"model":             request.Model,
		"input":             input,
		"max_output_tokens": request.MaxTokens,
		"stream":            request.Stream,
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
				"name": tool.Name,
			}
			if strings.TrimSpace(tool.Description) != "" {
				item["description"] = tool.Description
			}
			if len(bytes.TrimSpace(tool.InputSchema)) > 0 {
				var schema any
				if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
					return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tools.input_schema must be valid JSON", UpstreamFamily: "openai"}
				}
				item["parameters"] = schema
			}
			tools = append(tools, item)
		}
		payload["tools"] = tools
	}
	if hasMeaningfulRawJSON(request.Metadata) {
		var metadata map[string]any
		if err := json.Unmarshal(request.Metadata, &metadata); err != nil {
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "metadata must be a JSON object when routing Claude Messages through OpenAI Responses", UpstreamFamily: "openai"}
		}
		payload["metadata"] = metadata
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

func bridgeMessageToOpenAIResponsesInput(message anthropicBridgeMessage, index int) ([]map[string]any, error) {
	role := strings.TrimSpace(message.Role)
	blocks, err := decodeAnthropicContent(message.Content)
	if err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content must be a string or content block array", index), UpstreamFamily: "openai"}
	}
	switch role {
	case "user":
		return bridgeUserMessageToOpenAIResponsesInput(blocks, index)
	case "assistant":
		return bridgeAssistantMessageToOpenAIResponsesInput(blocks, index)
	default:
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].role %q is not supported when routing Claude Messages through OpenAI Responses", index, role), UpstreamFamily: "openai"}
	}
}

func bridgeUserMessageToOpenAIResponsesInput(blocks []anthropicContentBlock, index int) ([]map[string]any, error) {
	items := make([]map[string]any, 0, 1)
	content := make([]map[string]any, 0, len(blocks))
	flush := func() {
		if len(content) == 0 {
			return
		}
		items = append(items, map[string]any{"type": "message", "role": "user", "content": append([]map[string]any(nil), content...)})
		content = nil
	}
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			content = append(content, map[string]any{"type": "input_text", "text": block.Text})
		case "image":
			imagePart, err := bridgeAnthropicImageToOpenAIResponsesPart(block, index)
			if err != nil {
				return nil, err
			}
			content = append(content, imagePart)
		case "tool_result":
			flush()
			if strings.TrimSpace(block.ToolUseID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_result.tool_use_id is required when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
			}
			text, err := bridgeContentText(block.Content, fmt.Sprintf("messages[%d].content.tool_result", index))
			if err != nil {
				return nil, err
			}
			items = append(items, map[string]any{"type": "function_call_output", "call_id": block.ToolUseID, "output": text})
		case "thinking", "redacted_thinking":
			return nil, unsupportedOpenAIFieldError(block.Type)
		default:
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through OpenAI Responses", index, block.Type), UpstreamFamily: "openai"}
		}
	}
	flush()
	if len(items) == 0 {
		items = append(items, map[string]any{"type": "message", "role": "user", "content": []map[string]any{{"type": "input_text", "text": ""}}})
	}
	return items, nil
}

func bridgeAssistantMessageToOpenAIResponsesInput(blocks []anthropicContentBlock, index int) ([]map[string]any, error) {
	items := make([]map[string]any, 0, 1)
	content := make([]map[string]any, 0, len(blocks))
	flush := func() {
		if len(content) == 0 {
			return
		}
		items = append(items, map[string]any{"type": "message", "role": "assistant", "content": append([]map[string]any(nil), content...)})
		content = nil
	}
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			content = append(content, map[string]any{"type": "input_text", "text": block.Text})
		case "tool_use":
			flush()
			if strings.TrimSpace(block.ID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.id is required when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
			}
			if strings.TrimSpace(block.Name) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.name is required when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
			}
			arguments := "{}"
			if len(bytes.TrimSpace(block.Input)) > 0 {
				arguments = string(bytes.TrimSpace(block.Input))
			}
			items = append(items, map[string]any{"type": "function_call", "call_id": block.ID, "name": block.Name, "arguments": arguments})
		case "thinking", "redacted_thinking":
			return nil, unsupportedOpenAIFieldError(block.Type)
		default:
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through OpenAI Responses", index, block.Type), UpstreamFamily: "openai"}
		}
	}
	flush()
	if len(items) == 0 {
		items = append(items, map[string]any{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "input_text", "text": ""}}})
	}
	return items, nil
}

func bridgeAnthropicImageToOpenAIResponsesPart(block anthropicContentBlock, index int) (map[string]any, error) {
	var source struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
		URL       string `json:"url"`
	}
	if err := json.Unmarshal(block.Source, &source); err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source must be valid JSON when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
	}
	switch strings.TrimSpace(source.Type) {
	case "base64":
		if strings.TrimSpace(source.MediaType) == "" || strings.TrimSpace(source.Data) == "" {
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source requires media_type and data when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
		}
		return map[string]any{"type": "input_image", "image_url": fmt.Sprintf("data:%s;base64,%s", source.MediaType, source.Data)}, nil
	case "url":
		if strings.TrimSpace(source.URL) == "" {
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source.url is required when routing Claude Messages through OpenAI Responses", index), UpstreamFamily: "openai"}
		}
		return map[string]any{"type": "input_image", "image_url": source.URL}, nil
	default:
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source.type %q is not supported when routing Claude Messages through OpenAI Responses", index, source.Type), UpstreamFamily: "openai"}
	}
}

func convertOpenAIResponsesToClaudeResponse(body io.ReadCloser, statusCode int, header http.Header) ([]byte, http.Header, error) {
	defer drainProxyBody(body)
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	convertedHeader := cloneHeaderWithJSONContentType(header)
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return marshalAnthropicErrorFromOpenAI(payload, statusCode), convertedHeader, nil
	}
	var response map[string]any
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, nil, err
	}
	content, sawTool := bridgeOpenAIResponsesOutputToClaude(response)
	usage := mapValue(response["usage"])
	converted := map[string]any{
		"id":            stringValue(response["id"]),
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         stringValue(response["model"]),
		"stop_reason":   mapOpenAIResponsesStopReason(response, sawTool),
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  intValue(usage["input_tokens"]),
			"output_tokens": intValue(usage["output_tokens"]),
		},
	}
	convertedBody, err := json.Marshal(converted)
	if err != nil {
		return nil, nil, err
	}
	return convertedBody, convertedHeader, nil
}

func bridgeOpenAIResponsesOutputToClaude(response map[string]any) ([]map[string]any, bool) {
	output := sliceValue(response["output"])
	content := make([]map[string]any, 0)
	sawTool := false
	for _, rawItem := range output {
		item := mapValue(rawItem)
		switch stringValue(item["type"]) {
		case "message":
			if stringValue(item["role"]) != "assistant" {
				continue
			}
			for _, rawPart := range sliceValue(item["content"]) {
				part := mapValue(rawPart)
				switch stringValue(part["type"]) {
				case "output_text", "text":
					if text := stringValue(part["text"]); text != "" {
						content = append(content, map[string]any{"type": "text", "text": text})
					}
				case "refusal":
					if text := stringValue(part["refusal"]); text != "" {
						content = append(content, map[string]any{"type": "text", "text": text})
					}
				}
			}
		case "function_call":
			sawTool = true
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    coalesceString(item["call_id"], item["id"]),
				"name":  stringValue(item["name"]),
				"input": decodeOpenAIToolArguments(stringValue(item["arguments"])),
			})
		}
	}
	return content, sawTool
}

func mapOpenAIResponsesStopReason(response map[string]any, sawTool bool) string {
	if sawTool {
		return "tool_use"
	}
	if strings.TrimSpace(stringValue(response["status"])) == "incomplete" {
		reason := strings.TrimSpace(stringValue(mapValue(response["incomplete_details"])["reason"]))
		if strings.Contains(reason, "max") {
			return "max_tokens"
		}
	}
	return "end_turn"
}

func convertOpenAIResponsesStreamToClaudeStream(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer drainProxyBody(body)
		defer writer.Close()
		if err := writeClaudeStreamFromOpenAIResponses(body, writer); err != nil {
			_ = writer.CloseWithError(err)
		}
	}()
	return reader
}

type claudeResponsesStreamState struct {
	messageStarted bool
	messageStopped bool
	blockStarted   bool
	blockType      string
	blockIndex     int
	messageID      string
	model          string
	sawTool        bool
	toolBlocks     map[int]claudeToolBlockState
}

func writeClaudeStreamFromOpenAIResponses(body io.Reader, writer *io.PipeWriter) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var eventLines []string
	state := claudeResponsesStreamState{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := handleOpenAIResponsesStreamEvent(eventLines, writer, &state); err != nil {
				return err
			}
			eventLines = eventLines[:0]
			continue
		}
		eventLines = append(eventLines, line)
	}
	if len(eventLines) > 0 {
		if err := handleOpenAIResponsesStreamEvent(eventLines, writer, &state); err != nil {
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
			state.blockStarted = false
		}
		if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
			return err
		}
		state.messageStopped = true
	}
	return nil
}

func handleOpenAIResponsesStreamEvent(lines []string, writer *io.PipeWriter, state *claudeResponsesStreamState) error {
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
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return err
	}
	eventType := stringValue(payload["type"])
	switch eventType {
	case "response.created", "response.in_progress":
		return ensureClaudeResponsesMessageStarted(writer, state, mapValue(payload["response"]))
	case "response.output_item.added":
		if err := ensureClaudeResponsesMessageStarted(writer, state, mapValue(payload["response"])); err != nil {
			return err
		}
		item := mapValue(payload["item"])
		if stringValue(item["type"]) != "function_call" {
			return nil
		}
		index := intValue(payload["output_index"])
		if state.toolBlocks == nil {
			state.toolBlocks = map[int]claudeToolBlockState{}
		}
		state.toolBlocks[index] = claudeToolBlockState{id: coalesceString(item["call_id"], item["id"]), name: stringValue(item["name"])}
		state.sawTool = true
		if strings.TrimSpace(stringValue(item["arguments"])) != "" {
			if err := emitClaudeResponsesToolArguments(writer, state, index, stringValue(item["arguments"])); err != nil {
				return err
			}
		}
		return nil
	case "response.output_item.done":
		item := mapValue(payload["item"])
		if stringValue(item["type"]) != "function_call" {
			return nil
		}
		index := intValue(payload["output_index"])
		if state.toolBlocks == nil {
			state.toolBlocks = map[int]claudeToolBlockState{}
		}
		toolState := state.toolBlocks[index]
		toolState.id = coalesceString(item["call_id"], item["id"], toolState.id)
		toolState.name = coalesceString(item["name"], toolState.name)
		state.toolBlocks[index] = toolState
		state.sawTool = true
		if strings.TrimSpace(stringValue(item["arguments"])) != "" {
			if err := emitClaudeResponsesToolArguments(writer, state, index, stringValue(item["arguments"])); err != nil {
				return err
			}
		}
		return nil
	case "response.output_text.delta":
		if err := ensureClaudeResponsesMessageStarted(writer, state, mapValue(payload["response"])); err != nil {
			return err
		}
		index := intValue(payload["output_index"])
		if state.blockStarted && (state.blockType != "text" || state.blockIndex != index) {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		if !state.blockStarted {
			state.blockStarted = true
			state.blockType = "text"
			state.blockIndex = index
			if err := writeAnthropicStreamEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": map[string]any{"type": "text", "text": ""}}); err != nil {
				return err
			}
		}
		return writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "text_delta", "text": stringValue(payload["delta"])}})
	case "response.function_call_arguments.delta":
		if err := ensureClaudeResponsesMessageStarted(writer, state, mapValue(payload["response"])); err != nil {
			return err
		}
		index := intValue(payload["output_index"])
		return emitClaudeResponsesToolArguments(writer, state, index, stringValue(payload["delta"]))
	case "response.completed":
		response := mapValue(payload["response"])
		if err := ensureClaudeResponsesMessageStarted(writer, state, response); err != nil {
			return err
		}
		if state.blockStarted {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		usage := mapValue(response["usage"])
		if err := writeAnthropicStreamEvent(writer, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": mapOpenAIResponsesStopReason(response, state.sawTool), "stop_sequence": nil}, "usage": map[string]any{"input_tokens": intValue(usage["input_tokens"]), "output_tokens": intValue(usage["output_tokens"])} }); err != nil {
			return err
		}
		if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
			return err
		}
		state.messageStopped = true
		return nil
	default:
		return nil
	}
}

func emitClaudeResponsesToolArguments(writer *io.PipeWriter, state *claudeResponsesStreamState, index int, arguments string) error {
	toolState := state.toolBlocks[index]
	state.sawTool = true
	if state.blockStarted && (state.blockType != "tool_use" || state.blockIndex != index) {
		if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
			return err
		}
		state.blockStarted = false
	}
	if !state.blockStarted {
		state.blockStarted = true
		state.blockType = "tool_use"
		state.blockIndex = index
		if err := writeAnthropicStreamEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": map[string]any{"type": "tool_use", "id": toolState.id, "name": toolState.name, "input": map[string]any{}}}); err != nil {
			return err
		}
	}
	if strings.TrimSpace(arguments) == "" {
		return nil
	}
	return writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": arguments}})
}

func ensureClaudeResponsesMessageStarted(writer *io.PipeWriter, state *claudeResponsesStreamState, response map[string]any) error {
	if state.messageStarted {
		return nil
	}
	state.messageStarted = true
	state.messageID = stringValue(response["id"])
	state.model = stringValue(response["model"])
	return writeAnthropicStreamEvent(writer, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            state.messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []any{},
			"model":         state.model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})
}

func mapValue(value any) map[string]any {
	item, _ := value.(map[string]any)
	if item == nil {
		return map[string]any{}
	}
	return item
}

func sliceValue(value any) []any {
	items, _ := value.([]any)
	if items == nil {
		return []any{}
	}
	return items
}

func intValue(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		result, _ := v.Int64()
		return int(result)
	default:
		return 0
	}
}

func coalesceString(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(stringValue(value)); text != "" {
			return text
		}
	}
	return ""
}
