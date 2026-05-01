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

func bridgeClaudeMessagesToGemini(ctx context.Context, generateContentProvider GenerateContentProvider, request MessagesRequest) (*ProxyResponse, error) {
	body, err := buildGeminiGenerateContentBody(request.Body)
	if err != nil {
		return nil, err
	}
	response, err := generateContentProvider.GenerateContent(ctx, GenerateContentRequest{
		Model:          request.Model,
		Stream:         request.Stream,
		Body:           body,
		ContentType:    "application/json",
		Accept:         request.Accept,
		Headers:        request.Headers,
		UpstreamURL:    request.UpstreamURL,
		UpstreamAPIKey: request.UpstreamAPIKey,
	})
	if err != nil {
		requestError := &RequestError{}
		if errorsAsRequestError(err, requestError) {
			requestError.UpstreamFamily = "gemini"
		}
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	response.UpstreamFamily = "gemini"
	if response.Stream {
		response.Header = cloneHeaderWithSSEContentType(response.Header)
		response.Body = convertGeminiStreamToClaudeStream(response.Body, extractStopSequencesFromBody(request.Body))
		return response, nil
	}
	convertedBody, header, err := convertGeminiResponseToClaudeResponse(response.Body, response.StatusCode, response.Header, extractStopSequencesFromBody(request.Body))
	if err != nil {
		return nil, err
	}
	response.Header = header
	response.Body = io.NopCloser(bytes.NewReader(convertedBody))
	return response, nil
}

func buildGeminiGenerateContentBody(body []byte) ([]byte, error) {
	fields, err := parseAnthropicBridgeRequest(body)
	if err != nil {
		return nil, &RequestError{StatusCode: 400, Message: "Invalid JSON body", UpstreamFamily: "gemini"}
	}
	request := fields.Request
	for key := range fields.Raw {
		if _, ok := supportedAnthropicGeminiBridgeFields[key]; !ok {
			return nil, unsupportedGeminiFieldError(key)
		}
	}
	if hasMeaningfulRawJSON(fields.Raw["thinking"]) {
		return nil, unsupportedGeminiFieldError("thinking")
	}
	if hasMeaningfulRawJSON(fields.Raw["top_k"]) {
		return nil, unsupportedGeminiFieldError("top_k")
	}
	if hasMeaningfulRawJSON(fields.Raw["metadata"]) {
		return nil, unsupportedGeminiFieldError("metadata")
	}
	if len(request.StopSequences) > 0 {
		return nil, unsupportedGeminiFieldError("stop_sequences")
	}
	payload := map[string]any{}
	if systemText, err := bridgeSystemToGeminiInstruction(request.System); err != nil {
		return nil, err
	} else if systemText != "" {
		payload["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": systemText}}}
	}
	toolNames := map[string]string{}
	contents := make([]map[string]any, 0, len(request.Messages))
	for index, message := range request.Messages {
		content, err := bridgeAnthropicMessageToGeminiContent(message, index, toolNames)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}
	payload["contents"] = contents
	generationConfig := map[string]any{"maxOutputTokens": request.MaxTokens}
	if request.Temperature != nil {
		generationConfig["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		generationConfig["topP"] = *request.TopP
	}
	payload["generationConfig"] = generationConfig
	if len(request.Tools) > 0 {
		declarations := make([]map[string]any, 0, len(request.Tools))
		for _, tool := range request.Tools {
			item := map[string]any{"name": tool.Name}
			if strings.TrimSpace(tool.Description) != "" {
				item["description"] = tool.Description
			}
			if len(bytes.TrimSpace(tool.InputSchema)) > 0 {
				var schema any
				if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
					return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tools.input_schema must be valid JSON", UpstreamFamily: "gemini"}
				}
				item["parameters"] = schema
			}
			declarations = append(declarations, item)
		}
		payload["tools"] = []map[string]any{{"functionDeclarations": declarations}}
		if len(bytes.TrimSpace(request.ToolChoice)) > 0 && string(bytes.TrimSpace(request.ToolChoice)) != "null" {
			toolConfig, err := bridgeToolChoiceToGemini(request.ToolChoice)
			if err != nil {
				return nil, err
			}
			payload["toolConfig"] = toolConfig
		}
	}
	return json.Marshal(payload)
}

func bridgeAnthropicMessageToGeminiContent(message anthropicBridgeMessage, index int, toolNames map[string]string) (map[string]any, error) {
	role := strings.TrimSpace(message.Role)
	blocks, err := decodeAnthropicContent(message.Content)
	if err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content must be a string or content block array", index), UpstreamFamily: "gemini"}
	}
	parts := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			parts = append(parts, map[string]any{"text": block.Text})
		case "image":
			part, err := bridgeAnthropicImageToGeminiPart(block, index)
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case "tool_use":
			if role != "assistant" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use is only supported for assistant messages when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
			}
			if strings.TrimSpace(block.ID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.id is required when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
			}
			if strings.TrimSpace(block.Name) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.name is required when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
			}
			args := map[string]any{}
			if len(bytes.TrimSpace(block.Input)) > 0 {
				if err := json.Unmarshal(block.Input, &args); err != nil {
					var anyArgs any
					if err := json.Unmarshal(block.Input, &anyArgs); err != nil {
						return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_use.input must be valid JSON when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
					}
					args = map[string]any{"value": anyArgs}
				}
			}
			parts = append(parts, map[string]any{"functionCall": map[string]any{"id": block.ID, "name": block.Name, "args": args}})
			toolNames[block.ID] = block.Name
		case "tool_result":
			if role != "user" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_result is only supported for user messages when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
			}
			if strings.TrimSpace(block.ToolUseID) == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_result.tool_use_id is required when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
			}
			toolName := strings.TrimSpace(toolNames[block.ToolUseID])
			if toolName == "" {
				return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.tool_result.tool_use_id %q has no matching tool_use when routing Claude Messages through Gemini", index, block.ToolUseID), UpstreamFamily: "gemini"}
			}
			responsePayload, err := bridgeAnthropicToolResultToGeminiResponse(block)
			if err != nil {
				return nil, err
			}
			parts = append(parts, map[string]any{"functionResponse": map[string]any{"id": block.ToolUseID, "name": toolName, "response": responsePayload}})
		case "thinking", "redacted_thinking":
			return nil, unsupportedGeminiFieldError(block.Type)
		default:
			return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content block type %q is not supported when routing Claude Messages through Gemini", index, block.Type), UpstreamFamily: "gemini"}
		}
	}
	if len(parts) == 0 {
		parts = append(parts, map[string]any{"text": ""})
	}
	geminiRole := "user"
	if role == "assistant" {
		geminiRole = "model"
	}
	return map[string]any{"role": geminiRole, "parts": parts}, nil
}

func bridgeAnthropicToolResultToGeminiResponse(block anthropicContentBlock) (map[string]any, error) {
	responsePayload := map[string]any{}
	trimmed := bytes.TrimSpace(block.Content)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		responsePayload["content"] = ""
	} else {
		var text string
		if err := json.Unmarshal(trimmed, &text); err == nil {
			responsePayload["content"] = text
		} else {
			blocks, blockErr := decodeAnthropicContent(block.Content)
			if blockErr == nil {
				parts := make([]string, 0, len(blocks))
				for _, item := range blocks {
					switch item.Type {
					case "", "text":
						parts = append(parts, item.Text)
					case "thinking", "redacted_thinking":
						return nil, unsupportedGeminiFieldError(item.Type)
					default:
						return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("tool_result content block type %q is not supported when routing Claude Messages through Gemini", item.Type), UpstreamFamily: "gemini"}
					}
				}
				responsePayload["content"] = strings.Join(parts, "\n")
			} else {
				var anyPayload any
				if err := json.Unmarshal(trimmed, &anyPayload); err != nil {
					return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tool_result content must be valid JSON when routing Claude Messages through Gemini", UpstreamFamily: "gemini"}
				}
				responsePayload["content"] = anyPayload
			}
		}
	}
	if block.IsError {
		responsePayload["is_error"] = true
	}
	return responsePayload, nil
}

func bridgeAnthropicImageToGeminiPart(block anthropicContentBlock, index int) (map[string]any, error) {
	var source struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
		URL       string `json:"url"`
	}
	if err := json.Unmarshal(block.Source, &source); err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source must be valid JSON when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
	}
	if strings.TrimSpace(source.Type) != "base64" {
		return nil, unsupportedGeminiFieldError("image.source.type")
	}
	if strings.TrimSpace(source.MediaType) == "" || strings.TrimSpace(source.Data) == "" {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("messages[%d].content.image.source requires media_type and data when routing Claude Messages through Gemini", index), UpstreamFamily: "gemini"}
	}
	return map[string]any{"inlineData": map[string]any{"mimeType": source.MediaType, "data": source.Data}}, nil
}

func bridgeToolChoiceToGemini(raw json.RawMessage) (map[string]any, error) {
	var payload struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: "tool_choice must be valid JSON", UpstreamFamily: "gemini"}
	}
	config := map[string]any{"functionCallingConfig": map[string]any{}}
	callingConfig := config["functionCallingConfig"].(map[string]any)
	switch strings.TrimSpace(payload.Type) {
	case "auto", "":
		callingConfig["mode"] = "AUTO"
	case "none":
		callingConfig["mode"] = "NONE"
	case "any":
		callingConfig["mode"] = "ANY"
	case "tool":
		callingConfig["mode"] = "ANY"
		callingConfig["allowedFunctionNames"] = []string{payload.Name}
	default:
		return nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("tool_choice type %q is not supported when routing Claude Messages through Gemini", payload.Type), UpstreamFamily: "gemini"}
	}
	return config, nil
}

func extractStopSequencesFromBody(body []byte) []string {
	fields, err := parseAnthropicBridgeRequest(body)
	if err != nil {
		return nil
	}
	return append([]string(nil), fields.Request.StopSequences...)
}

func convertGeminiResponseToClaudeResponse(body io.ReadCloser, statusCode int, header http.Header, stopSequences []string) ([]byte, http.Header, error) {
	defer drainProxyBody(body)
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	convertedHeader := cloneHeaderWithJSONContentType(header)
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return marshalAnthropicErrorFromGemini(payload, statusCode), convertedHeader, nil
	}
	var response map[string]any
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, nil, err
	}
	if blockReason := strings.TrimSpace(stringValue(mapValue(response["promptFeedback"])["blockReason"])); blockReason != "" {
		return nil, nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("Gemini prompt was blocked: %s", blockReason), UpstreamFamily: "gemini"}
	}
	content, sawTool, _ := bridgeGeminiContentToClaude(response)
	finishReason := strings.TrimSpace(stringValue(firstGeminiCandidate(response)["finishReason"]))
	if geminiFinishReasonBlocked(finishReason) {
		return nil, nil, &RequestError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("Gemini finish reason %s is not supported when routing Claude Messages through Gemini", finishReason), UpstreamFamily: "gemini"}
	}
	usage := mapValue(response["usageMetadata"])
	stopReason, stopSequence := mapGeminiStopReason(finishReason, sawTool)
	converted := map[string]any{
		"id":            coalesceString(response["responseId"], response["id"], "msg_gemini"),
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         coalesceString(response["modelVersion"], response["model"], "gemini"),
		"stop_reason":   stopReason,
		"stop_sequence": stopSequence,
		"usage": map[string]any{
			"input_tokens":  geminiPromptTokens(usage),
			"output_tokens": geminiResponseTokens(usage),
		},
	}
	convertedBody, err := json.Marshal(converted)
	if err != nil {
		return nil, nil, err
	}
	return convertedBody, convertedHeader, nil
}

func bridgeGeminiContentToClaude(response map[string]any) ([]map[string]any, bool, string) {
	candidate := firstGeminiCandidate(response)
	content := mapValue(candidate["content"])
	parts := sliceValue(content["parts"])
	blocks := make([]map[string]any, 0, len(parts))
	var sawTool bool
	var textBuilder strings.Builder
	for index, rawPart := range parts {
		part := mapValue(rawPart)
		if text := strings.TrimSpace(stringValue(part["text"])); text != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
			textBuilder.WriteString(text)
			continue
		}
		functionCall := mapValue(part["functionCall"])
		if len(functionCall) == 0 {
			continue
		}
		sawTool = true
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    coalesceString(functionCall["id"], fmt.Sprintf("toolu_gemini_%d", index)),
			"name":  stringValue(functionCall["name"]),
			"input": mapValue(functionCall["args"]),
		})
	}
	return blocks, sawTool, textBuilder.String()
}

func firstGeminiCandidate(response map[string]any) map[string]any {
	candidates := sliceValue(response["candidates"])
	if len(candidates) == 0 {
		return map[string]any{}
	}
	return mapValue(candidates[0])
}

func geminiPromptTokens(usage map[string]any) int {
	if value := intValue(usage["promptTokenCount"]); value > 0 {
		return value
	}
	return intValue(usage["prompt_tokens"])
}

func geminiResponseTokens(usage map[string]any) int {
	if value := intValue(usage["responseTokenCount"]); value > 0 {
		return value
	}
	if value := intValue(usage["candidatesTokenCount"]); value > 0 {
		return value
	}
	return intValue(usage["output_tokens"])
}

func mapGeminiStopReason(finishReason string, sawTool bool) (string, any) {
	if sawTool {
		return "tool_use", nil
	}
	if strings.EqualFold(strings.TrimSpace(finishReason), "MAX_TOKENS") {
		return "max_tokens", nil
	}
	return "end_turn", nil
}

func geminiFinishReasonBlocked(finishReason string) bool {
	switch strings.ToUpper(strings.TrimSpace(finishReason)) {
	case "SAFETY", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII", "RECITATION", "LANGUAGE", "IMAGE_SAFETY", "IMAGE_PROHIBITED_CONTENT", "IMAGE_RECITATION", "IMAGE_OTHER", "NO_IMAGE", "MALFORMED_FUNCTION_CALL", "UNEXPECTED_TOOL_CALL", "TOO_MANY_TOOL_CALLS", "MALFORMED_RESPONSE", "MISSING_THOUGHT_SIGNATURE":
		return true
	default:
		return false
	}
}

func marshalAnthropicErrorFromGemini(body []byte, statusCode int) []byte {
	message := http.StatusText(statusCode)
	errorType := "api_error"
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			message = payload.Error.Message
		}
		switch strings.ToUpper(strings.TrimSpace(payload.Error.Status)) {
		case "INVALID_ARGUMENT":
			errorType = "invalid_request_error"
		case "DEADLINE_EXCEEDED":
			errorType = "timeout_error"
		default:
			errorType = "api_error"
		}
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

func convertGeminiStreamToClaudeStream(body io.ReadCloser, stopSequences []string) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer drainProxyBody(body)
		defer writer.Close()
		if err := writeClaudeStreamFromGemini(body, writer, stopSequences); err != nil {
			_ = writer.CloseWithError(err)
		}
	}()
	return reader
}

type claudeGeminiStreamState struct {
	messageStarted bool
	messageStopped bool
	blockStarted   bool
	blockType      string
	blockIndex     int
	messageID      string
	model          string
	generatedText  string
	sawTool        bool
	toolStates     map[int]claudeToolBlockState
	toolArgs       map[int]string
	stopSequences  []string
	inputTokens    int
	outputTokens   int
	textParts      map[int]string
}

func writeClaudeStreamFromGemini(body io.Reader, writer *io.PipeWriter, stopSequences []string) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var eventLines []string
	state := claudeGeminiStreamState{stopSequences: append([]string(nil), stopSequences...), toolStates: map[int]claudeToolBlockState{}, toolArgs: map[int]string{}, textParts: map[int]string{}}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := handleGeminiStreamEvent(eventLines, writer, &state); err != nil {
				return err
			}
			eventLines = eventLines[:0]
			continue
		}
		eventLines = append(eventLines, line)
	}
	if len(eventLines) > 0 {
		if err := handleGeminiStreamEvent(eventLines, writer, &state); err != nil {
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
		if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
			return err
		}
	}
	return nil
}

func handleGeminiStreamEvent(lines []string, writer *io.PipeWriter, state *claudeGeminiStreamState) error {
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
	var payload map[string]any
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return err
	}
	if blockReason := strings.TrimSpace(stringValue(mapValue(payload["promptFeedback"])["blockReason"])); blockReason != "" {
		return &RequestError{StatusCode: 400, Message: fmt.Sprintf("Gemini prompt was blocked: %s", blockReason), UpstreamFamily: "gemini"}
	}
	usage := mapValue(payload["usageMetadata"])
	if value := geminiPromptTokens(usage); value > 0 {
		state.inputTokens = value
	}
	if value := geminiResponseTokens(usage); value > 0 {
		state.outputTokens = value
	}
	if !state.messageStarted {
		state.messageStarted = true
		state.messageID = "msg_gemini_stream"
		state.model = "gemini"
		if err := writeAnthropicStreamEvent(writer, "message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": state.messageID, "type": "message", "role": "assistant", "content": []any{}, "model": state.model, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]any{"input_tokens": state.inputTokens, "output_tokens": 0}}}); err != nil {
			return err
		}
	}
	candidate := firstGeminiCandidate(payload)
	parts := sliceValue(mapValue(candidate["content"])["parts"])
	for index, rawPart := range parts {
		part := mapValue(rawPart)
		if text, ok := part["text"].(string); ok {
			delta := text
			if previous, exists := state.textParts[index]; exists && strings.HasPrefix(text, previous) {
				delta = strings.TrimPrefix(text, previous)
			}
			state.textParts[index] = text
			if delta == "" {
				continue
			}
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
			state.generatedText += delta
			if err := writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "text_delta", "text": delta}}); err != nil {
				return err
			}
			continue
		}
		functionCall := mapValue(part["functionCall"])
		if len(functionCall) == 0 {
			continue
		}
		state.sawTool = true
		toolState := state.toolStates[index]
		toolState.id = coalesceString(functionCall["id"], toolState.id, fmt.Sprintf("toolu_gemini_%d", index))
		toolState.name = coalesceString(functionCall["name"], toolState.name)
		state.toolStates[index] = toolState
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
		arguments, _ := json.Marshal(mapValue(functionCall["args"]))
		argumentString := string(arguments)
		delta := argumentString
		if previousArgs := state.toolArgs[index]; previousArgs != "" {
			prefixLength := commonPrefixLength(previousArgs, argumentString)
			delta = argumentString[prefixLength:]
		}
		state.toolArgs[index] = argumentString
		if delta != "" {
			if err := writeAnthropicStreamEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": delta}}); err != nil {
				return err
			}
		}
	}
	finishReason := strings.TrimSpace(stringValue(candidate["finishReason"]))
	if finishReason != "" {
		if state.blockStarted {
			if err := writeAnthropicStreamEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": state.blockIndex}); err != nil {
				return err
			}
			state.blockStarted = false
		}
		if geminiFinishReasonBlocked(finishReason) {
			return &RequestError{StatusCode: 400, Message: fmt.Sprintf("Gemini finish reason %s is not supported when routing Claude Messages through Gemini", finishReason), UpstreamFamily: "gemini"}
		}
		stopReason, stopSequence := mapGeminiStopReason(finishReason, state.sawTool)
		if err := writeAnthropicStreamEvent(writer, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stopReason, "stop_sequence": stopSequence}, "usage": map[string]any{"input_tokens": state.inputTokens, "output_tokens": state.outputTokens}}); err != nil {
			return err
		}
		if err := writeAnthropicStreamEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
			return err
		}
		state.messageStopped = true
	}
	return nil
}

func commonPrefixLength(a string, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for index := 0; index < limit; index++ {
		if a[index] != b[index] {
			return index
		}
	}
	return limit
}
