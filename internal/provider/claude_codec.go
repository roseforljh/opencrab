package provider

import (
	"encoding/json"
	"fmt"

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
	payload["messages"] = encodeClaudeMessages(req.Messages)
	return json.Marshal(payload)
}

func DecodeClaudeChatRequest(body []byte) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 Claude 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolClaude}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "messages")
	if err := decodeRawString(raw, "model", &req.Model, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	_ = decodeRawBool(raw, "stream", &req.Stream)

	var messagesRaw []map[string]json.RawMessage
	if err := decodeRaw(raw, "messages", &messagesRaw, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	messages, err := decodeClaudeMessages(messagesRaw)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = messages

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

func encodeClaudeMessages(messages []domain.UnifiedMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		item["role"] = message.Role
		item["content"] = []map[string]any{{
			"type": "text",
			"text": message.Parts[0].Text,
		}}
		out = append(out, item)
	}
	return out
}

func decodeClaudeMessages(items []map[string]json.RawMessage) ([]domain.UnifiedMessage, error) {
	messages := make([]domain.UnifiedMessage, 0, len(items))
	for i, item := range items {
		message, err := decodeClaudeMessage(item)
		if err != nil {
			return nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		messages = append(messages, message)
	}
	return messages, nil
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
		return domain.UnifiedMessage{
			Role:     role,
			Parts:    []domain.UnifiedPart{{Type: "text", Text: contentString}},
			Metadata: collectUnknownFields(raw, "role", "content"),
		}, nil
	}

	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &blocks); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("content 格式非法: %w", err)
	}
	if len(blocks) != 1 {
		return domain.UnifiedMessage{}, fmt.Errorf("当前仅支持单个 text content block")
	}

	var blockType string
	if err := decodeRawString(blocks[0], "type", &blockType, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("content[0].type: %w", err)
	}
	if blockType != "text" {
		return domain.UnifiedMessage{}, fmt.Errorf("当前仅支持 Claude text content block")
	}

	var text string
	if err := decodeRawString(blocks[0], "text", &text, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("content[0].text: %w", err)
	}
	if len(collectUnknownFields(blocks[0], "type", "text")) > 0 {
		return domain.UnifiedMessage{}, fmt.Errorf("当前仅支持 text-only 主链路，Claude content metadata 暂不支持")
	}

	return domain.UnifiedMessage{
		Role:     role,
		Parts:    []domain.UnifiedPart{{Type: "text", Text: text}},
		Metadata: collectUnknownFields(raw, "role", "content"),
	}, nil
}
