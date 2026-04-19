package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
)

func EncodeClaudeChatRequest(req domain.UnifiedChatRequest) ([]byte, error) {
	if req.Protocol == "" {
		req.Protocol = domain.ProtocolClaude
	}
	if req.Protocol != domain.ProtocolClaude {
		return nil, fmt.Errorf("Claude codec 不支持协议: %s", req.Protocol)
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
	if _, ok := payload["max_tokens"]; !ok {
		payload["max_tokens"] = 1024
	}
	if len(req.Tools) > 0 {
		payload["tools"] = rawMessagesToAny(req.Tools)
	}
	systemBlocks, messages, err := encodeClaudeMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	if len(systemBlocks) > 0 {
		payload["system"] = systemBlocks
	}
	payload["messages"] = messages
	if err := normalizeClaudeCompatibilityPayload(payload); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

func DecodeClaudeChatRequest(body []byte) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 Claude 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolClaude}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "messages", "system", "tools")
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

	if systemRaw, ok := raw["system"]; ok {
		systemMessages, err := decodeClaudeSystem(systemRaw)
		if err != nil {
			return domain.UnifiedChatRequest{}, err
		}
		req.Messages = append(req.Messages, systemMessages...)
	}

	var messagesRaw []map[string]json.RawMessage
	if err := decodeRaw(raw, "messages", &messagesRaw, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	messages, err := decodeClaudeMessages(messagesRaw)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = append(req.Messages, messages...)

	if err := req.ValidateCore(); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	return req, nil
}

func DecodeClaudeChatResponse(body []byte) (domain.UnifiedChatResponse, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatResponse{}, fmt.Errorf("解析 Claude 响应失败: %w", err)
	}

	resp := domain.UnifiedChatResponse{Protocol: domain.ProtocolClaude}
	resp.Metadata = collectUnknownFields(raw, "id", "model", "content", "stop_reason", "usage", "role")
	_ = decodeRawString(raw, "id", &resp.ID, false)
	_ = decodeRawString(raw, "model", &resp.Model, false)
	_ = decodeRawString(raw, "stop_reason", &resp.FinishReason, false)
	_ = decodeUsage(raw, &resp.Usage)

	message, err := decodeClaudeMessage(raw)
	if err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	resp.Message = message
	return resp, nil
}

func EncodeClaudeChatResponse(resp domain.UnifiedChatResponse) ([]byte, error) {
	payload := map[string]any{}
	mergeRawFields(payload, resp.Metadata)
	if resp.ID != "" {
		payload["id"] = resp.ID
	}
	payload["type"] = "message"
	payload["role"] = "assistant"
	if resp.Model != "" {
		payload["model"] = resp.Model
	}
	content, err := encodeClaudeMessageBlocks(resp.Message)
	if err != nil {
		return nil, err
	}
	payload["content"] = content
	if resp.FinishReason != "" {
		payload["stop_reason"] = resp.FinishReason
	}
	if len(resp.Usage) > 0 {
		payload["usage"] = map[string]any{
			"input_tokens":  resp.Usage["prompt_tokens"],
			"output_tokens": resp.Usage["completion_tokens"],
		}
	}
	return json.Marshal(payload)
}

func encodeClaudeMessages(messages []domain.UnifiedMessage) ([]map[string]any, []map[string]any, error) {
	systemBlocks := make([]map[string]any, 0)
	out := make([]map[string]any, 0, len(messages))
	for i, message := range messages {
		content, err := encodeClaudeMessageBlocks(message)
		if err != nil {
			return nil, nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		if strings.EqualFold(strings.TrimSpace(message.Role), "system") {
			systemBlocks = append(systemBlocks, content...)
			continue
		}
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		role := message.Role
		if strings.EqualFold(strings.TrimSpace(role), "tool") {
			role = "user"
		}
		item["role"] = role
		item["content"] = content
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, nil, fmt.Errorf("Claude 请求至少需要一条非 system 消息")
	}
	return systemBlocks, out, nil
}

func encodeClaudeContentBlocks(parts []domain.UnifiedPart) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(parts))
	for i, part := range parts {
		block, err := encodeClaudeContentBlock(part)
		if err != nil {
			return nil, fmt.Errorf("parts[%d]: %w", i, err)
		}
		out = append(out, block)
	}
	return out, nil
}

func encodeClaudeMessageBlocks(message domain.UnifiedMessage) ([]map[string]any, error) {
	blocks, err := encodeClaudeContentBlocks(message.Parts)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(message.Role), "tool") {
		return buildClaudeToolResultBlocks(message)
	}
	for _, call := range message.ToolCalls {
		block := map[string]any{}
		mergeRawFields(block, call.Metadata)
		var input any = map[string]any{}
		if len(call.Arguments) > 0 {
			_ = json.Unmarshal(call.Arguments, &input)
		}
		block["type"] = "tool_use"
		block["id"] = call.ID
		block["name"] = call.Name
		block["input"] = input
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func buildClaudeToolResultBlocks(message domain.UnifiedMessage) ([]map[string]any, error) {
	toolBlocks := make([]map[string]any, 0, len(message.Parts))
	tailParts := make([]domain.UnifiedPart, 0, len(message.Parts))
	for _, part := range message.Parts {
		if len(part.NativePayload) > 0 {
			var block map[string]any
			if err := json.Unmarshal(part.NativePayload, &block); err != nil {
				return nil, err
			}
			toolBlocks = append(toolBlocks, block)
			continue
		}
		tailParts = append(tailParts, part)
	}
	if len(toolBlocks) == 0 {
		toolUseID := decodeStringRaw(message.Metadata["tool_call_id"])
		content, err := encodeClaudeToolResultContent(tailParts)
		if err != nil {
			return nil, err
		}
		toolBlocks = append(toolBlocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": toolUseID,
			"content":     content,
		})
		tailParts = nil
	}
	if len(tailParts) == 0 {
		return toolBlocks, nil
	}
	extraBlocks, err := encodeClaudeContentBlocks(tailParts)
	if err != nil {
		return nil, err
	}
	return append(toolBlocks, extraBlocks...), nil
}

func encodeClaudeToolResultContent(parts []domain.UnifiedPart) (any, error) {
	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) == 1 && parts[0].Type == "text" && len(parts[0].Metadata) == 0 {
		return parts[0].Text, nil
	}
	return encodeClaudeContentBlocks(parts)
}

func encodeClaudeContentBlock(part domain.UnifiedPart) (map[string]any, error) {
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
	case "image", "document", "file":
		blockType := part.Type
		if blockType == "file" {
			blockType = "document"
		}
		item["type"] = blockType
		if _, ok := item["source"]; !ok {
			source, err := buildClaudeSource(desc)
			if err != nil {
				return nil, err
			}
			item["source"] = source
		}
	case "audio", "video":
		return nil, fmt.Errorf("Claude 暂不支持 %s 输入", part.Type)
	default:
		if len(item) > 0 {
			if _, ok := item["type"]; !ok {
				item["type"] = part.Type
			}
			return item, nil
		}
		return nil, fmt.Errorf("Claude 暂不支持 part 类型: %s", part.Type)
	}
	return item, nil
}

func buildClaudeSource(desc mediaDescriptor) (map[string]any, error) {
	if desc.Data != "" {
		return map[string]any{"type": "base64", "media_type": desc.MimeType, "data": desc.Data}, nil
	}
	if desc.URL != "" {
		return map[string]any{"type": "url", "url": desc.URL}, nil
	}
	if desc.FileID != "" {
		return map[string]any{"type": "file", "file_id": desc.FileID}, nil
	}
	return nil, fmt.Errorf("Claude 媒体块缺少 source 数据")
}

func decodeClaudeMessages(items []map[string]json.RawMessage) ([]domain.UnifiedMessage, error) {
	messages := make([]domain.UnifiedMessage, 0, len(items))
	for i, item := range items {
		decoded, err := decodeClaudeRequestMessages(item)
		if err != nil {
			return nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		messages = append(messages, decoded...)
	}
	return messages, nil
}

func decodeClaudeRequestMessages(raw map[string]json.RawMessage) ([]domain.UnifiedMessage, error) {
	contentRaw, ok := raw["content"]
	if !ok {
		return nil, fmt.Errorf("content 缺失")
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &blocks); err != nil {
		message, err := decodeClaudeMessage(raw)
		if err != nil {
			return nil, err
		}
		return []domain.UnifiedMessage{message}, nil
	}
	toolResultCount := 0
	for _, block := range blocks {
		var blockType string
		if err := decodeRawString(block, "type", &blockType, false); err == nil && blockType == "tool_result" {
			toolResultCount++
		}
	}
	if toolResultCount <= 1 {
		message, err := decodeClaudeMessage(raw)
		if err != nil {
			return nil, err
		}
		return []domain.UnifiedMessage{message}, nil
	}

	var role string
	if err := decodeRawString(raw, "role", &role, true); err != nil {
		return nil, err
	}
	baseMetadata := collectUnknownFields(raw, "role", "content")
	messages := make([]domain.UnifiedMessage, 0, toolResultCount+1)
	current := domain.UnifiedMessage{Role: role, Metadata: cloneRawMap(baseMetadata)}
	flushCurrent := func() {
		if len(current.Parts) == 0 && len(current.ToolCalls) == 0 {
			return
		}
		messages = append(messages, current)
		current = domain.UnifiedMessage{Role: role, Metadata: cloneRawMap(baseMetadata)}
	}

	for i, block := range blocks {
		part, call, isToolResult, err := decodeClaudeContentBlock(block)
		if err != nil {
			return nil, fmt.Errorf("content[%d]: %w", i, err)
		}
		if isToolResult {
			flushCurrent()
			metadata := map[string]json.RawMessage{}
			if rawToolUseID, found := decodeClaudeToolResultID(part); found {
				metadata["tool_call_id"] = rawToolUseID
			}
			messages = append(messages, domain.UnifiedMessage{Role: "tool", Parts: []domain.UnifiedPart{part}, Metadata: metadata})
			continue
		}
		if call.Name != "" {
			current.ToolCalls = append(current.ToolCalls, call)
			continue
		}
		current.Parts = append(current.Parts, part)
	}
	flushCurrent()
	return messages, nil
}

func decodeClaudeSystem(systemRaw json.RawMessage) ([]domain.UnifiedMessage, error) {
	var systemString string
	if err := json.Unmarshal(systemRaw, &systemString); err == nil {
		return []domain.UnifiedMessage{{Role: "system", Parts: []domain.UnifiedPart{{Type: "text", Text: systemString}}}}, nil
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(systemRaw, &blocks); err != nil {
		return nil, fmt.Errorf("system 格式非法: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(blocks))
	for i, block := range blocks {
		part, _, _, err := decodeClaudeContentBlock(block)
		if err != nil {
			return nil, fmt.Errorf("system[%d]: %w", i, err)
		}
		parts = append(parts, part)
	}
	return []domain.UnifiedMessage{{Role: "system", Parts: parts}}, nil
}

func firstUnifiedText(message domain.UnifiedMessage) string {
	for _, part := range message.Parts {
		if part.Type == "text" {
			return part.Text
		}
	}
	return ""
}

func decodeClaudeMessage(raw map[string]json.RawMessage) (domain.UnifiedMessage, error) {
	var role string
	if err := decodeRawString(raw, "role", &role, true); err != nil {
		return domain.UnifiedMessage{}, err
	}
	contentRaw, ok := raw["content"]
	if !ok {
		return domain.UnifiedMessage{}, fmt.Errorf("content 缺失")
	}
	var contentString string
	if err := json.Unmarshal(contentRaw, &contentString); err == nil {
		return domain.UnifiedMessage{Role: role, Parts: []domain.UnifiedPart{{Type: "text", Text: contentString}}, Metadata: collectUnknownFields(raw, "role", "content")}, nil
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &blocks); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("content 格式非法: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(blocks))
	toolCalls := make([]domain.UnifiedToolCall, 0)
	toolResultMode := false
	messageMetadata := collectUnknownFields(raw, "role", "content")
	for i, block := range blocks {
		part, call, isToolResult, err := decodeClaudeContentBlock(block)
		if err != nil {
			return domain.UnifiedMessage{}, fmt.Errorf("content[%d]: %w", i, err)
		}
		if call.Name != "" {
			toolCalls = append(toolCalls, call)
			continue
		}
		if isToolResult {
			toolResultMode = true
			if rawToolUseID, found := decodeClaudeToolResultID(part); found {
				if messageMetadata == nil {
					messageMetadata = map[string]json.RawMessage{}
				}
				if _, exists := messageMetadata["tool_call_id"]; !exists {
					messageMetadata["tool_call_id"] = rawToolUseID
				}
			}
		}
		parts = append(parts, part)
	}
	if toolResultMode {
		role = "tool"
	}
	return domain.UnifiedMessage{Role: role, Parts: parts, ToolCalls: toolCalls, Metadata: messageMetadata}, nil
}

func decodeClaudeContentBlock(raw map[string]json.RawMessage) (domain.UnifiedPart, domain.UnifiedToolCall, bool, error) {
	var blockType string
	if err := decodeRawString(raw, "type", &blockType, true); err != nil {
		return domain.UnifiedPart{}, domain.UnifiedToolCall{}, false, err
	}
	metadata := collectUnknownFields(raw, "type", "text")
	switch blockType {
	case "text":
		var text string
		if err := decodeRawString(raw, "text", &text, true); err != nil {
			return domain.UnifiedPart{}, domain.UnifiedToolCall{}, false, err
		}
		return domain.UnifiedPart{Type: "text", Text: text, Metadata: metadata}, domain.UnifiedToolCall{}, false, nil
	case "image", "document":
		metadata = cloneRawMap(metadata)
		if sourceRaw, ok := raw["source"]; ok {
			metadata["source"] = append(json.RawMessage(nil), sourceRaw...)
			desc := extractMediaDescriptor(domain.UnifiedPart{Type: blockType, Metadata: metadata})
			metadata = enrichPartMetadata(blockType, metadata, desc)
		}
		return domain.UnifiedPart{Type: blockType, Metadata: metadata}, domain.UnifiedToolCall{}, false, nil
	case "tool_use":
		var id, name string
		_ = decodeRawString(raw, "id", &id, true)
		_ = decodeRawString(raw, "name", &name, true)
		arguments := json.RawMessage(`{}`)
		if inputRaw, ok := raw["input"]; ok {
			arguments = append(json.RawMessage(nil), inputRaw...)
		}
		return domain.UnifiedPart{}, domain.UnifiedToolCall{ID: id, Name: name, Arguments: arguments, Metadata: collectUnknownFields(raw, "type", "id", "name", "input")}, false, nil
	case "tool_result":
		rawBlock, err := json.Marshal(raw)
		if err != nil {
			return domain.UnifiedPart{}, domain.UnifiedToolCall{}, false, err
		}
		content := decodeClaudeToolResultText(raw["content"])
		partType := "tool_result"
		if strings.TrimSpace(content) != "" {
			partType = "text"
		}
		return domain.UnifiedPart{Type: partType, Text: content, NativePayload: rawBlock, Metadata: metadata}, domain.UnifiedToolCall{}, true, nil
	default:
		metadata = cloneRawMap(raw)
		return domain.UnifiedPart{Type: blockType, Metadata: metadata}, domain.UnifiedToolCall{}, false, nil
	}
}

func decodeClaudeToolResultID(part domain.UnifiedPart) (json.RawMessage, bool) {
	rawBlock := part.NativePayload
	if len(rawBlock) == 0 {
		return nil, false
	}
	var block map[string]json.RawMessage
	if err := json.Unmarshal(rawBlock, &block); err != nil {
		return nil, false
	}
	toolUseID, ok := block["tool_use_id"]
	if !ok || len(toolUseID) == 0 {
		return nil, false
	}
	return append(json.RawMessage(nil), toolUseID...), true
}

func decodeClaudeToolResultText(rawContent json.RawMessage) string {
	if len(rawContent) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(rawContent, &text); err == nil {
		return text
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(rawContent, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		var blockType string
		if err := decodeRawString(block, "type", &blockType, false); err != nil {
			continue
		}
		if blockType != "text" {
			continue
		}
		var text string
		if err := decodeRawString(block, "text", &text, false); err == nil && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func normalizeClaudeCompatibilityPayload(payload map[string]any) error {
	if payload == nil {
		return nil
	}
	if thinking, ok := payload["thinking"]; ok {
		if toolChoice, exists := payload["tool_choice"]; exists && claudeToolChoiceDisallowsThinking(toolChoice) {
			delete(payload, "thinking")
		} else if !claudeThinkingSupported(payload["model"], thinking) {
			delete(payload, "thinking")
		}
	}
	cacheControls := collectClaudeCacheControlTTLs(payload)
	if len(cacheControls) > 4 {
		return fmt.Errorf("Claude cache_control 断点不能超过 4 个")
	}
	seenFiveMinute := false
	for _, ttl := range cacheControls {
		if ttl == "5m" {
			seenFiveMinute = true
		}
		if ttl == "1h" && seenFiveMinute {
			return fmt.Errorf("Claude cache_control ttl 不能在 5m 之后再出现 1h")
		}
	}
	if topLevelTTL, nestedHasFive := claudeTopLevelAndNestedCacheTTL(payload); topLevelTTL == "1h" && nestedHasFive {
		return fmt.Errorf("Claude cache_control ttl 不能在 5m 之后再出现 1h")
	}
	return nil
}

func claudeToolChoiceDisallowsThinking(value any) bool {
	choice, ok := value.(map[string]any)
	if !ok {
		return false
	}
	typeValue, _ := choice["type"].(string)
	return typeValue == "any" || typeValue == "tool"
}

func claudeThinkingSupported(model any, value any) bool {
	thinking, ok := value.(map[string]any)
	if !ok {
		return false
	}
	modelName, _ := model.(string)
	typeValue, _ := thinking["type"].(string)
	switch typeValue {
	case "adaptive":
		return true
	case "enabled":
		if strings.Contains(strings.ToLower(strings.TrimSpace(modelName)), "opus-4.7") {
			return false
		}
		budget, ok := thinking["budget_tokens"].(float64)
		return ok && budget > 0
	default:
		return false
	}
}

func collectClaudeCacheControlTTLs(value any) []string {
	ttls := make([]string, 0, 4)
	collectClaudeCacheControlTTLsInto(value, &ttls)
	return ttls
}

func collectClaudeCacheControlTTLsInto(value any, ttls *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"tools", "system", "messages", "content"} {
			if child, ok := typed[key]; ok {
				collectClaudeCacheControlTTLsInto(child, ttls)
			}
		}
		if rawCache, ok := typed["cache_control"]; ok {
			if cache, ok := rawCache.(map[string]any); ok {
				ttl, _ := cache["ttl"].(string)
				if ttl == "" {
					ttl = "5m"
				}
				*ttls = append(*ttls, ttl)
			}
		}
		for key, item := range typed {
			switch key {
			case "tools", "system", "messages", "content", "cache_control":
				continue
			default:
				collectClaudeCacheControlTTLsInto(item, ttls)
			}
		}
	case []any:
		for _, item := range typed {
			collectClaudeCacheControlTTLsInto(item, ttls)
		}
	}
}

func claudeTopLevelAndNestedCacheTTL(payload map[string]any) (string, bool) {
	topLevelTTL := ""
	if rawCache, ok := payload["cache_control"]; ok {
		if cache, ok := rawCache.(map[string]any); ok {
			topLevelTTL, _ = cache["ttl"].(string)
			if topLevelTTL == "" {
				topLevelTTL = "5m"
			}
		}
	}
	var nestedHasFive bool
	for _, key := range []string{"tools", "system", "messages"} {
		if child, ok := payload[key]; ok {
			if containsCacheTTL(child, "5m") {
				nestedHasFive = true
				break
			}
		}
	}
	return topLevelTTL, nestedHasFive
}

func containsCacheTTL(value any, expected string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if rawCache, ok := typed["cache_control"]; ok {
			if cache, ok := rawCache.(map[string]any); ok {
				ttl, _ := cache["ttl"].(string)
				if ttl == "" {
					ttl = "5m"
				}
				if ttl == expected {
					return true
				}
			}
		}
		for _, item := range typed {
			if containsCacheTTL(item, expected) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if containsCacheTTL(item, expected) {
				return true
			}
		}
	}
	return false
}
