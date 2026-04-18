package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"
)

func DecodeOpenAIResponsesRequest(body []byte) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 Responses 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolOpenAI}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "input", "tools", "store", "parallel_tool_calls", "previous_response_id", "include", "reasoning", "instructions")
	if err := decodeRawString(raw, "model", &req.Model, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	_ = decodeRawBool(raw, "stream", &req.Stream)
	if toolsRaw, ok := raw["tools"]; ok {
		var tools []json.RawMessage
		if err := json.Unmarshal(toolsRaw, &tools); err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("tools 格式非法: %w", err)
		}
		req.Tools = tools
	}
	if instructionsRaw, ok := raw["instructions"]; ok {
		var instructions string
		if err := json.Unmarshal(instructionsRaw, &instructions); err == nil && strings.TrimSpace(instructions) != "" {
			req.Messages = append(req.Messages, domain.UnifiedMessage{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: instructions}}})
		}
	}
	input, ok := raw["input"]
	if !ok {
		return domain.UnifiedChatRequest{}, fmt.Errorf("input 缺失")
	}
	messages, err := decodeResponsesInput(input)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = append(req.Messages, messages...)
	if err := req.ValidateCore(); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	return req, nil
}

func DecodeOpenAIResponsesSession(body []byte) (*domain.GatewaySessionState, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("解析 Responses 会话失败: %w", err)
	}
	session := &domain.GatewaySessionState{Metadata: map[string]string{}}
	if previousRaw, ok := raw["previous_response_id"]; ok {
		_ = json.Unmarshal(previousRaw, &session.PreviousResponseID)
	}
	if includeRaw, ok := raw["include"]; ok {
		session.Metadata["include"] = string(includeRaw)
	}
	if reasoningRaw, ok := raw["reasoning"]; ok {
		session.Metadata["reasoning"] = string(reasoningRaw)
	}
	if storeRaw, ok := raw["store"]; ok {
		session.Metadata["store"] = string(storeRaw)
	}
	if len(session.Metadata) == 0 {
		session.Metadata = nil
	}
	if session.PreviousResponseID == "" {
		return nil, nil
	}
	return session, nil
}

func EncodeOpenAIResponsesRequest(req domain.UnifiedChatRequest, session *domain.GatewaySessionState) ([]byte, error) {
	if req.Protocol == "" {
		req.Protocol = domain.ProtocolOpenAI
	}
	if req.Protocol != domain.ProtocolOpenAI {
		return nil, fmt.Errorf("Responses codec 不支持协议: %s", req.Protocol)
	}
	if err := req.ValidateCore(); err != nil {
		return nil, err
	}

	payload := map[string]any{}
	mergeRawFields(payload, req.Metadata)
	payload["model"] = req.Model
	if req.Stream {
		payload["stream"] = true
	}
	if len(req.Tools) > 0 {
		payload["tools"] = rawMessagesToAny(req.Tools)
	}
	instructions, input, err := encodeResponsesInputFromMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(instructions) != "" {
		payload["instructions"] = instructions
	}
	payload["input"] = input
	if session != nil {
		if strings.TrimSpace(session.PreviousResponseID) != "" {
			payload["previous_response_id"] = session.PreviousResponseID
		}
		if session.Metadata != nil {
			if include := strings.TrimSpace(session.Metadata["include"]); include != "" {
				var raw any
				if err := json.Unmarshal([]byte(include), &raw); err == nil {
					payload["include"] = raw
				}
			}
			if reasoning := strings.TrimSpace(session.Metadata["reasoning"]); reasoning != "" {
				var raw any
				if err := json.Unmarshal([]byte(reasoning), &raw); err == nil {
					payload["reasoning"] = raw
				}
			}
			if store := strings.TrimSpace(session.Metadata["store"]); store != "" {
				var raw any
				if err := json.Unmarshal([]byte(store), &raw); err == nil {
					payload["store"] = raw
				}
			}
		}
	}
	return json.Marshal(payload)
}

func DecodeOpenAIResponsesResponse(body []byte) (domain.UnifiedChatResponse, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatResponse{}, fmt.Errorf("解析 Responses 响应失败: %w", err)
	}

	resp := domain.UnifiedChatResponse{Protocol: domain.ProtocolOpenAI}
	resp.Metadata = collectUnknownFields(raw, "id", "model", "status", "output", "usage", "output_text")
	_ = decodeRawString(raw, "id", &resp.ID, false)
	_ = decodeRawString(raw, "model", &resp.Model, false)
	_ = decodeRawString(raw, "status", &resp.FinishReason, false)
	if usageRaw, ok := raw["usage"]; ok {
		usage, err := decodeResponsesUsage(usageRaw)
		if err != nil {
			return domain.UnifiedChatResponse{}, err
		}
		resp.Usage = usage
	}
	outputRaw, ok := raw["output"]
	if !ok {
		return domain.UnifiedChatResponse{}, fmt.Errorf("output 缺失")
	}
	message, err := decodeResponsesOutput(outputRaw)
	if err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	resp.Message = message
	return resp, nil
}

func EncodeOpenAIResponsesResponse(resp domain.UnifiedChatResponse) ([]byte, error) {
	payload := map[string]any{}
	mergeRawFields(payload, resp.Metadata)
	if resp.ID != "" {
		payload["id"] = resp.ID
	} else {
		payload["id"] = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	payload["object"] = "response"
	payload["created_at"] = time.Now().Unix()
	payload["status"] = "completed"
	if resp.Model != "" {
		payload["model"] = resp.Model
	}
	output, outputText, err := encodeResponsesOutput(resp.Message)
	if err != nil {
		return nil, err
	}
	payload["output"] = output
	if outputText != "" {
		payload["output_text"] = outputText
	}
	usage := encodeResponsesUsage(resp.Usage)
	if usage != nil {
		payload["usage"] = usage
	}
	return json.Marshal(payload)
}

func EncodeOpenAIResponsesStream(resp domain.UnifiedChatResponse) ([]byte, error) {
	events, err := BuildOpenAIResponsesEvents(resp)
	if err != nil {
		return nil, err
	}
	encoded := make([]string, 0, len(events)+1)
	for _, event := range events {
		eventType, _ := event["type"].(string)
		encoded = append(encoded, mustSSEEvent(eventType, event))
	}
	encoded = append(encoded, "data: [DONE]\n\n")
	return []byte(strings.Join(encoded, "")), nil
}

func BuildOpenAIResponsesEvents(resp domain.UnifiedChatResponse) ([]map[string]any, error) {
	responseID := resp.ID
	if responseID == "" {
		responseID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	createdAt := time.Now().Unix()
	output, outputText, err := encodeResponsesOutput(resp.Message)
	if err != nil {
		return nil, err
	}
	usage := encodeResponsesUsage(resp.Usage)
	responseObject := map[string]any{
		"id":         responseID,
		"object":     "response",
		"created_at": createdAt,
		"status":     "completed",
		"model":      resp.Model,
		"output":     output,
	}
	if outputText != "" {
		responseObject["output_text"] = outputText
	}
	if usage != nil {
		responseObject["usage"] = usage
	}
	events := make([]map[string]any, 0, 8)
	events = append(events, map[string]any{"type": "response.created", "response": map[string]any{"id": responseID, "object": "response", "created_at": createdAt, "status": "in_progress", "model": resp.Model}})
	events = append(events, map[string]any{"type": "response.in_progress", "response": map[string]any{"id": responseID, "object": "response", "created_at": createdAt, "status": "in_progress", "model": resp.Model}})
	if len(resp.Message.ToolCalls) > 0 {
		for idx, call := range resp.Message.ToolCalls {
			itemID := call.ID
			if itemID == "" {
				itemID = fmt.Sprintf("fc_%d", idx+1)
			}
			item := map[string]any{"id": itemID, "type": "function_call", "call_id": itemID, "name": call.Name, "arguments": string(call.Arguments)}
			events = append(events, map[string]any{"type": "response.output_item.added", "output_index": idx, "item": item})
			events = append(events, map[string]any{"type": "response.function_call_arguments.delta", "output_index": idx, "item_id": itemID, "delta": string(call.Arguments)})
			events = append(events, map[string]any{"type": "response.function_call_arguments.done", "output_index": idx, "item_id": itemID, "arguments": string(call.Arguments)})
			events = append(events, map[string]any{"type": "response.output_item.done", "output_index": idx, "item": item})
		}
	} else {
		itemID := "msg_1"
		item := map[string]any{"id": itemID, "type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": outputText}}}
		if outputText == "" {
			item["content"] = []map[string]any{}
		}
		events = append(events, map[string]any{"type": "response.output_item.added", "output_index": 0, "item": item})
		if outputText != "" {
			events = append(events, map[string]any{"type": "response.output_text.delta", "output_index": 0, "item_id": itemID, "content_index": 0, "delta": outputText})
			events = append(events, map[string]any{"type": "response.output_text.done", "output_index": 0, "item_id": itemID, "content_index": 0, "text": outputText})
		}
		events = append(events, map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})
	}
	events = append(events, map[string]any{"type": "response.completed", "response": responseObject})
	return events, nil
}

func encodeResponsesInputFromMessages(messages []domain.UnifiedMessage) (string, []any, error) {
	instructions := make([]string, 0)
	items := make([]any, 0, len(messages))
	for _, message := range messages {
		if strings.EqualFold(strings.TrimSpace(message.Role), "system") {
			if text := firstUnifiedText(message); strings.TrimSpace(text) != "" {
				instructions = append(instructions, text)
			}
			continue
		}
		encoded, err := encodeResponsesInputItems(message)
		if err != nil {
			return "", nil, err
		}
		items = append(items, encoded...)
	}
	if len(items) == 0 {
		return "", nil, fmt.Errorf("Responses 请求至少需要一条非 system 消息")
	}
	return strings.Join(instructions, "\n\n"), items, nil
}

func encodeResponsesInputItems(message domain.UnifiedMessage) ([]any, error) {
	if strings.EqualFold(strings.TrimSpace(message.Role), "tool") {
		return []any{map[string]any{
			"type":    "function_call_output",
			"call_id": decodeStringRaw(message.Metadata["tool_call_id"]),
			"output":  firstUnifiedText(message),
		}}, nil
	}

	items := make([]any, 0, 1+len(message.ToolCalls))
	if len(message.Parts) > 0 {
		content, err := encodeResponsesContent(message.Parts)
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"type":    "message",
			"role":    message.Role,
			"content": content,
		})
	}
	for idx, call := range message.ToolCalls {
		callID := call.ID
		if strings.TrimSpace(callID) == "" {
			callID = fmt.Sprintf("fc_%d", idx+1)
		}
		items = append(items, map[string]any{
			"type":      "function_call",
			"call_id":   callID,
			"name":      call.Name,
			"arguments": string(call.Arguments),
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("Responses 消息不能为空")
	}
	return items, nil
}

func encodeResponsesContent(parts []domain.UnifiedPart) ([]map[string]any, error) {
	items := make([]map[string]any, 0, len(parts))
	for _, part := range parts {
		item, err := encodeOpenAIPart(part)
		if err != nil {
			return nil, err
		}
		switch item["type"] {
		case "text":
			item["type"] = "input_text"
		case "image_url":
			item["type"] = "input_image"
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeResponsesOutput(raw json.RawMessage) (domain.UnifiedMessage, error) {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("output 格式非法: %w", err)
	}
	message := domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{}, ToolCalls: []domain.UnifiedToolCall{}}
	for i, item := range items {
		var itemType string
		if err := decodeRawString(item, "type", &itemType, true); err != nil {
			return domain.UnifiedMessage{}, fmt.Errorf("output[%d].type: %w", i, err)
		}
		switch itemType {
		case "message":
			var role string
			_ = decodeRawString(item, "role", &role, false)
			if strings.TrimSpace(role) != "" {
				message.Role = role
			}
			if contentRaw, ok := item["content"]; ok {
				parts, err := decodeResponsesContent(contentRaw)
				if err != nil {
					return domain.UnifiedMessage{}, fmt.Errorf("output[%d].content: %w", i, err)
				}
				message.Parts = append(message.Parts, parts...)
			}
		case "function_call":
			var name string
			if err := decodeRawString(item, "name", &name, true); err != nil {
				return domain.UnifiedMessage{}, fmt.Errorf("output[%d].name: %w", i, err)
			}
			arguments := json.RawMessage(`{}`)
			if rawArgs, ok := item["arguments"]; ok {
				var argsString string
				if err := json.Unmarshal(rawArgs, &argsString); err == nil {
					arguments = json.RawMessage(argsString)
				} else {
					arguments = append(json.RawMessage(nil), rawArgs...)
				}
			}
			call := domain.UnifiedToolCall{Name: name, Arguments: arguments}
			if callIDRaw, ok := item["call_id"]; ok {
				_ = json.Unmarshal(callIDRaw, &call.ID)
			}
			message.ToolCalls = append(message.ToolCalls, call)
		}
	}
	if len(message.Parts) == 0 && len(message.ToolCalls) == 0 {
		message.Parts = append(message.Parts, domain.UnifiedPart{Type: "text", Text: ""})
	}
	return message, nil
}

func decodeResponsesUsage(raw json.RawMessage) (map[string]int64, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("usage 格式非法: %w", err)
	}
	usage := map[string]int64{}
	extractInt64(payload, "input_tokens", usage, "prompt_tokens")
	extractInt64(payload, "output_tokens", usage, "completion_tokens")
	extractInt64(payload, "total_tokens", usage, "total_tokens")
	return usage, nil
}

func decodeResponsesInput(input json.RawMessage) ([]domain.UnifiedMessage, error) {
	var inputString string
	if err := json.Unmarshal(input, &inputString); err == nil {
		return []domain.UnifiedMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: inputString}}}}, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(input, &items); err != nil {
		return nil, fmt.Errorf("input 格式非法: %w", err)
	}
	messages := make([]domain.UnifiedMessage, 0, len(items))
	for i, raw := range items {
		message, err := decodeResponsesInputItem(raw)
		if err != nil {
			return nil, fmt.Errorf("input[%d]: %w", i, err)
		}
		messages = append(messages, message...)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("input 不能为空")
	}
	return messages, nil
}

func decodeResponsesInputItem(raw json.RawMessage) ([]domain.UnifiedMessage, error) {
	var item map[string]json.RawMessage
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("item 格式非法: %w", err)
	}
	var itemType string
	_ = decodeRawString(item, "type", &itemType, false)
	var role string
	_ = decodeRawString(item, "role", &role, false)
	switch itemType {
	case "function_call_output":
		var output string
		_ = decodeRawString(item, "output", &output, false)
		metadata := map[string]json.RawMessage{}
		if callID, ok := item["call_id"]; ok {
			metadata["tool_call_id"] = append(json.RawMessage(nil), callID...)
		}
		return []domain.UnifiedMessage{{Role: "tool", Parts: []domain.UnifiedPart{{Type: "text", Text: output}}, Metadata: metadata}}, nil
	case "function_call":
		var name string
		_ = decodeRawString(item, "name", &name, true)
		args := json.RawMessage(`{}`)
		if argsRaw, ok := item["arguments"]; ok {
			var argsString string
			if err := json.Unmarshal(argsRaw, &argsString); err == nil {
				args = json.RawMessage(argsString)
			} else {
				args = append(json.RawMessage(nil), argsRaw...)
			}
		}
		toolCall := domain.UnifiedToolCall{Name: name, Arguments: args}
		if callID, ok := item["call_id"]; ok {
			_ = json.Unmarshal(callID, &toolCall.ID)
		}
		return []domain.UnifiedMessage{{Role: "assistant", ToolCalls: []domain.UnifiedToolCall{toolCall}, Parts: []domain.UnifiedPart{{Type: "text", Text: "tool_call"}}}}, nil
	case "message", "":
		if role == "" {
			role = "user"
		}
		if contentRaw, ok := item["content"]; ok {
			parts, err := decodeResponsesContent(contentRaw)
			if err != nil {
				return nil, err
			}
			return []domain.UnifiedMessage{{Role: role, Parts: parts, Metadata: collectUnknownFields(item, "type", "role", "content")}}, nil
		}
		if textRaw, ok := item["text"]; ok {
			var text string
			if err := json.Unmarshal(textRaw, &text); err == nil {
				return []domain.UnifiedMessage{{Role: role, Parts: []domain.UnifiedPart{{Type: "text", Text: text}}}}, nil
			}
		}
	}
	return nil, fmt.Errorf("暂不支持的 Responses input item")
}

func decodeResponsesContent(raw json.RawMessage) ([]domain.UnifiedPart, error) {
	var contentString string
	if err := json.Unmarshal(raw, &contentString); err == nil {
		return []domain.UnifiedPart{{Type: "text", Text: contentString}}, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("content 格式非法: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(items))
	for i, item := range items {
		var itemType string
		if err := decodeRawString(item, "type", &itemType, true); err != nil {
			return nil, fmt.Errorf("content[%d].type: %w", i, err)
		}
		switch itemType {
		case "input_text", "output_text", "text":
			var text string
			if err := decodeRawString(item, "text", &text, true); err != nil {
				return nil, fmt.Errorf("content[%d].text: %w", i, err)
			}
			parts = append(parts, domain.UnifiedPart{Type: "text", Text: text, Metadata: collectUnknownFields(item, "type", "text")})
		case "input_image", "image_url":
			part, err := decodeOpenAIPart(item)
			if err != nil {
				return nil, fmt.Errorf("content[%d]: %w", i, err)
			}
			parts = append(parts, part)
		case "input_file", "file", "input_audio":
			part, err := decodeOpenAIPart(item)
			if err != nil {
				return nil, fmt.Errorf("content[%d]: %w", i, err)
			}
			parts = append(parts, part)
		default:
			parts = append(parts, domain.UnifiedPart{Type: itemType, Metadata: cloneRawMap(item)})
		}
	}
	return parts, nil
}

func encodeResponsesOutput(message domain.UnifiedMessage) ([]map[string]any, string, error) {
	if len(message.ToolCalls) > 0 {
		items := make([]map[string]any, 0, len(message.ToolCalls))
		for idx, call := range message.ToolCalls {
			callID := call.ID
			if callID == "" {
				callID = fmt.Sprintf("fc_%d", idx+1)
			}
			items = append(items, map[string]any{
				"id":        callID,
				"type":      "function_call",
				"call_id":   callID,
				"name":      call.Name,
				"arguments": string(call.Arguments),
			})
		}
		return items, "", nil
	}
	content, err := encodeOpenAIMessageContent(message)
	if err != nil {
		return nil, "", err
	}
	outputText := firstUnifiedText(message)
	contentItems := []map[string]any{}
	switch typed := content.(type) {
	case string:
		contentItems = append(contentItems, map[string]any{"type": "output_text", "text": typed})
	case []map[string]any:
		for _, item := range typed {
			mapped := map[string]any{"type": "output_text"}
			for key, value := range item {
				mapped[key] = value
			}
			if mapped["type"] == "text" || mapped["type"] == "input_text" {
				mapped["type"] = "output_text"
			}
			contentItems = append(contentItems, mapped)
		}
	default:
		contentItems = append(contentItems, map[string]any{"type": "output_text", "text": outputText})
	}
	return []map[string]any{{"id": "msg_1", "type": "message", "role": "assistant", "content": contentItems}}, outputText, nil
}

func encodeResponsesUsage(usage map[string]int64) map[string]any {
	if len(usage) == 0 {
		return nil
	}
	promptTokens := usage["prompt_tokens"]
	completionTokens := usage["completion_tokens"]
	totalTokens := usage["total_tokens"]
	if totalTokens == 0 {
		totalTokens = promptTokens + completionTokens
	}
	return map[string]any{
		"input_tokens":  promptTokens,
		"output_tokens": completionTokens,
		"total_tokens":  totalTokens,
		"output_tokens_details": map[string]any{
			"reasoning_tokens": 0,
		},
	}
}

func mustSSEEvent(event string, payload map[string]any) string {
	body, _ := json.Marshal(payload)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(body))
}
