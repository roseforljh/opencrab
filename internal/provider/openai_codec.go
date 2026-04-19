package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
)

func EncodeOpenAIChatRequest(req domain.UnifiedChatRequest) ([]byte, error) {
	if req.Protocol == "" {
		req.Protocol = domain.ProtocolOpenAI
	}
	if req.Protocol != domain.ProtocolOpenAI {
		return nil, fmt.Errorf("OpenAI codec 不支持协议: %s", req.Protocol)
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

	messages := make([]map[string]any, 0, len(req.Messages))
	for i, message := range req.Messages {
		content, err := encodeOpenAIMessageContent(message)
		if err != nil {
			return nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		item["role"] = message.Role
		if message.Role == "tool" {
			if toolCallID := decodeStringRaw(message.Metadata["tool_call_id"]); toolCallID != "" {
				item["tool_call_id"] = toolCallID
			}
		}
		if len(message.ToolCalls) > 0 {
			item["tool_calls"] = encodeOpenAIToolCalls(message.ToolCalls)
		}
		item["content"] = content
		messages = append(messages, item)
	}
	payload["messages"] = messages

	return json.Marshal(payload)
}

func DecodeOpenAIChatRequest(body []byte) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 OpenAI 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolOpenAI}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "messages", "tools")
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

	var messagesRaw []map[string]json.RawMessage
	if err := decodeRaw(raw, "messages", &messagesRaw, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = make([]domain.UnifiedMessage, 0, len(messagesRaw))
	for i, item := range messagesRaw {
		var role string
		if err := decodeRawString(item, "role", &role, true); err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("messages[%d].role: %w", i, err)
		}
		parts, err := decodeOpenAIMessageContent(item["content"])
		if err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("messages[%d].content: %w", i, err)
		}
		toolCalls, err := decodeOpenAIToolCalls(item["tool_calls"])
		if err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("messages[%d].tool_calls: %w", i, err)
		}
		req.Messages = append(req.Messages, domain.UnifiedMessage{
			Role:      role,
			Parts:     parts,
			ToolCalls: toolCalls,
			Metadata:  collectUnknownFields(item, "role", "content", "tool_calls", "tool_call_id"),
		})
		if role == "tool" {
			if toolCallIDRaw, ok := item["tool_call_id"]; ok {
				if req.Messages[len(req.Messages)-1].Metadata == nil {
					req.Messages[len(req.Messages)-1].Metadata = map[string]json.RawMessage{}
				}
				req.Messages[len(req.Messages)-1].Metadata["tool_call_id"] = append(json.RawMessage(nil), toolCallIDRaw...)
			}
		}
	}

	if err := req.ValidateCore(); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	return req, nil
}

func DecodeOpenAIChatResponse(body []byte) (domain.UnifiedChatResponse, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatResponse{}, fmt.Errorf("解析 OpenAI 响应失败: %w", err)
	}

	resp := domain.UnifiedChatResponse{Protocol: domain.ProtocolOpenAI}
	resp.Metadata = collectUnknownFields(raw, "id", "model", "choices", "usage")
	_ = decodeRawString(raw, "id", &resp.ID, false)
	_ = decodeRawString(raw, "model", &resp.Model, false)
	_ = decodeUsage(raw, &resp.Usage)

	var choices []map[string]json.RawMessage
	if err := decodeRaw(raw, "choices", &choices, true); err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	if len(choices) == 0 {
		return domain.UnifiedChatResponse{}, fmt.Errorf("choices 不能为空")
	}

	first := choices[0]
	_ = decodeRawString(first, "finish_reason", &resp.FinishReason, false)
	message, err := decodeOpenAIResponseMessage(first)
	if err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	resp.Message = message
	return resp, nil
}

func EncodeOpenAIChatResponse(resp domain.UnifiedChatResponse) ([]byte, error) {
	payload := map[string]any{}
	mergeRawFields(payload, resp.Metadata)
	if resp.ID != "" {
		payload["id"] = resp.ID
	}
	payload["object"] = "chat.completion"
	if resp.Model != "" {
		payload["model"] = resp.Model
	}
	content, err := encodeOpenAIMessageContent(resp.Message)
	if err != nil {
		return nil, err
	}
	payload["choices"] = []map[string]any{{
		"index": 0,
		"message": map[string]any{
			"role":       resp.Message.Role,
			"content":    content,
			"tool_calls": encodeOpenAIToolCalls(resp.Message.ToolCalls),
		},
		"finish_reason": resp.FinishReason,
	}}
	if len(resp.Usage) > 0 {
		payload["usage"] = resp.Usage
	}
	return json.Marshal(payload)
}

func decodeOpenAIResponseMessage(choice map[string]json.RawMessage) (domain.UnifiedMessage, error) {
	var messageRaw map[string]json.RawMessage
	if err := decodeRaw(choice, "message", &messageRaw, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message: %w", err)
	}
	var role string
	if err := decodeRawString(messageRaw, "role", &role, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message.role: %w", err)
	}
	parts, err := decodeOpenAIMessageContent(messageRaw["content"])
	if err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message.content: %w", err)
	}
	toolCalls, err := decodeOpenAIToolCalls(messageRaw["tool_calls"])
	if err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message.tool_calls: %w", err)
	}
	return domain.UnifiedMessage{
		Role:      role,
		Parts:     parts,
		ToolCalls: toolCalls,
		Metadata:  collectUnknownFields(messageRaw, "role", "content"),
	}, nil
}

func encodeOpenAIMessageContent(message domain.UnifiedMessage) (any, error) {
	if len(message.Parts) == 1 && message.Parts[0].Type == "text" && len(message.Parts[0].Metadata) == 0 {
		return message.Parts[0].Text, nil
	}
	parts := make([]map[string]any, 0, len(message.Parts))
	for i, part := range message.Parts {
		item, err := encodeOpenAIPart(part)
		if err != nil {
			return nil, fmt.Errorf("parts[%d]: %w", i, err)
		}
		parts = append(parts, item)
	}
	return parts, nil
}

func encodeOpenAIPart(part domain.UnifiedPart) (map[string]any, error) {
	if item, ok := decodePreservedMapItem(part.InputItem); ok {
		return item, nil
	}
	if item, ok := decodePreservedMapItem(part.NativePayload); ok {
		return item, nil
	}
	item := map[string]any{}
	mergeRawFields(item, part.Metadata)
	desc := extractMediaDescriptor(part)
	switch part.Type {
	case "text":
		item["type"] = "text"
		item["text"] = part.Text
	case "image":
		item["type"] = "image_url"
		if _, ok := item["image_url"]; !ok {
			if desc.URL != "" {
				item["image_url"] = map[string]any{"url": desc.URL}
			} else if desc.Data != "" {
				item["image_url"] = map[string]any{"url": buildDataURL(desc.MimeType, desc.Data)}
			} else {
				return nil, fmt.Errorf("image part 缺少 image_url/url/data")
			}
		}
	case "audio":
		item["type"] = "input_audio"
		if _, ok := item["input_audio"]; !ok {
			if desc.Data == "" {
				return nil, fmt.Errorf("audio part 缺少 data")
			}
			item["input_audio"] = map[string]any{"data": desc.Data, "format": mimeToAudioFormat(desc.MimeType)}
		}
	case "document", "file", "video":
		item["type"] = "input_file"
		if desc.FileID != "" {
			item["file_id"] = desc.FileID
		} else if desc.Data != "" {
			item["file_data"] = buildDataURL(desc.MimeType, desc.Data)
			if desc.Filename != "" {
				item["filename"] = desc.Filename
			}
		} else if desc.URL != "" {
			item["file_url"] = desc.URL
		} else if desc.FileURI != "" {
			item["file_url"] = desc.FileURI
		} else {
			return nil, fmt.Errorf("%s part 缺少 file 数据", part.Type)
		}
	default:
		if len(item) > 0 {
			if _, ok := item["type"]; !ok {
				item["type"] = part.Type
			}
			return item, nil
		}
		return nil, fmt.Errorf("OpenAI 暂不支持 part 类型: %s", part.Type)
	}
	return item, nil
}

func decodePreservedMapItem(raw json.RawMessage) (map[string]any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil || len(item) == 0 {
		return nil, false
	}
	return item, true
}

func decodeOpenAIMessageContent(contentRaw json.RawMessage) ([]domain.UnifiedPart, error) {
	if len(contentRaw) == 0 || string(contentRaw) == "null" {
		return nil, nil
	}
	var contentString string
	if err := json.Unmarshal(contentRaw, &contentString); err == nil {
		if strings.TrimSpace(contentString) == "" {
			return nil, nil
		}
		return []domain.UnifiedPart{{Type: "text", Text: contentString}}, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &items); err != nil {
		return nil, fmt.Errorf("content 格式非法: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(items))
	for i, item := range items {
		part, err := decodeOpenAIPart(item)
		if err != nil {
			return nil, fmt.Errorf("parts[%d]: %w", i, err)
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func encodeOpenAIToolCalls(calls []domain.UnifiedToolCall) []map[string]any {
	if len(calls) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		item := map[string]any{"id": call.ID, "type": "function", "function": map[string]any{"name": call.Name, "arguments": string(call.Arguments)}}
		if len(call.Metadata) > 0 {
			mergeRawFields(item, call.Metadata)
		}
		out = append(out, item)
	}
	return out
}

func decodeOpenAIToolCalls(raw json.RawMessage) ([]domain.UnifiedToolCall, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("tool_calls 格式非法: %w", err)
	}
	result := make([]domain.UnifiedToolCall, 0, len(items))
	for i, item := range items {
		var id string
		_ = decodeRawString(item, "id", &id, false)
		var functionRaw map[string]json.RawMessage
		if err := decodeRaw(item, "function", &functionRaw, true); err != nil {
			return nil, fmt.Errorf("tool_calls[%d].function: %w", i, err)
		}
		var name string
		if err := decodeRawString(functionRaw, "name", &name, true); err != nil {
			return nil, fmt.Errorf("tool_calls[%d].function.name: %w", i, err)
		}
		arguments := json.RawMessage(`{}`)
		if rawArgs, ok := functionRaw["arguments"]; ok {
			var argsString string
			if err := json.Unmarshal(rawArgs, &argsString); err == nil {
				arguments = json.RawMessage(argsString)
			} else {
				arguments = append(json.RawMessage(nil), rawArgs...)
			}
		}
		result = append(result, domain.UnifiedToolCall{ID: id, Name: name, Arguments: arguments, Metadata: collectUnknownFields(item, "id", "type", "function")})
	}
	return result, nil
}

func decodeOpenAIPart(raw map[string]json.RawMessage) (domain.UnifiedPart, error) {
	var blockType string
	if err := decodeRawString(raw, "type", &blockType, true); err != nil {
		return domain.UnifiedPart{}, err
	}
	metadata := collectUnknownFields(raw, "type", "text")
	switch blockType {
	case "text", "input_text":
		var text string
		if err := decodeRawString(raw, "text", &text, true); err != nil {
			return domain.UnifiedPart{}, err
		}
		return domain.UnifiedPart{Type: "text", Text: text, Metadata: metadata}, nil
	case "image_url", "input_image":
		metadata = cloneRawMap(metadata)
		if imageRaw, ok := raw["image_url"]; ok {
			metadata["image_url"] = append(json.RawMessage(nil), imageRaw...)
			desc := extractMediaDescriptor(domain.UnifiedPart{Type: "image", Metadata: metadata})
			metadata = enrichPartMetadata("image", metadata, desc)
		}
		if fileIDRaw, ok := raw["file_id"]; ok {
			metadata["file_id"] = append(json.RawMessage(nil), fileIDRaw...)
		}
		return domain.UnifiedPart{Type: "image", Metadata: metadata}, nil
	case "input_audio":
		metadata = cloneRawMap(metadata)
		if audioRaw, ok := raw["input_audio"]; ok {
			metadata["input_audio"] = append(json.RawMessage(nil), audioRaw...)
			desc := extractMediaDescriptor(domain.UnifiedPart{Type: "audio", Metadata: metadata})
			metadata = enrichPartMetadata("audio", metadata, desc)
		}
		return domain.UnifiedPart{Type: "audio", Metadata: metadata}, nil
	case "input_file", "file":
		metadata = cloneRawMap(metadata)
		for _, key := range []string{"file_id", "file_data", "file_url", "filename"} {
			if value, ok := raw[key]; ok {
				metadata[key] = append(json.RawMessage(nil), value...)
			}
		}
		partType := "file"
		if fileDataRaw, ok := raw["file_data"]; ok {
			var dataURL string
			_ = json.Unmarshal(fileDataRaw, &dataURL)
			if strings.HasPrefix(dataURL, "data:") {
				mime := strings.TrimPrefix(strings.SplitN(strings.TrimPrefix(dataURL, "data:"), ";", 2)[0], "")
				partType = partTypeFromMime(mime)
			}
		}
		return domain.UnifiedPart{Type: partType, Metadata: metadata}, nil
	default:
		metadata = cloneRawMap(raw)
		return domain.UnifiedPart{Type: blockType, Metadata: metadata}, nil
	}
}

func rawMessagesToAny(items []json.RawMessage) []any {
	result := make([]any, 0, len(items))
	for _, raw := range items {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			result = append(result, json.RawMessage(raw))
			continue
		}
		result = append(result, value)
	}
	return result
}

func mergeRawFields(dst map[string]any, metadata map[string]json.RawMessage) {
	for key, raw := range metadata {
		if len(raw) == 0 {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			dst[key] = json.RawMessage(raw)
			continue
		}
		dst[key] = value
	}
}

func collectUnknownFields(raw map[string]json.RawMessage, knownKeys ...string) map[string]json.RawMessage {
	known := make(map[string]struct{}, len(knownKeys))
	for _, key := range knownKeys {
		known[key] = struct{}{}
	}
	metadata := map[string]json.RawMessage{}
	for key, value := range raw {
		if _, ok := known[key]; ok {
			continue
		}
		metadata[key] = append(json.RawMessage(nil), value...)
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func decodeRaw(container map[string]json.RawMessage, key string, target any, required bool) error {
	raw, ok := container[key]
	if !ok {
		if required {
			return fmt.Errorf("%s 缺失", key)
		}
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s 格式非法: %w", key, err)
	}
	return nil
}

func decodeRawString(container map[string]json.RawMessage, key string, target *string, required bool) error {
	if err := decodeRaw(container, key, target, required); err != nil {
		return err
	}
	if required && strings.TrimSpace(*target) == "" {
		return fmt.Errorf("%s 不能为空", key)
	}
	return nil
}

func decodeRawBool(container map[string]json.RawMessage, key string, target *bool) error {
	return decodeRaw(container, key, target, false)
}

func decodeUsage(raw map[string]json.RawMessage, target *map[string]int64) error {
	usageRaw, ok := raw["usage"]
	if !ok {
		return nil
	}
	var usagePayload map[string]json.RawMessage
	if err := json.Unmarshal(usageRaw, &usagePayload); err != nil {
		return fmt.Errorf("usage 格式非法: %w", err)
	}
	usage := map[string]int64{}
	for key, value := range usagePayload {
		var number int64
		if err := json.Unmarshal(value, &number); err == nil {
			usage[key] = number
		}
	}
	if len(usage) == 0 {
		return nil
	}
	*target = usage
	return nil
}
