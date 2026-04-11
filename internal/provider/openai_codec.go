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

	messages := make([]map[string]any, 0, len(req.Messages))
	for i, message := range req.Messages {
		if len(message.Parts) != 1 {
			return nil, fmt.Errorf("messages[%d] 当前仅支持单个 text part", i)
		}
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		item["role"] = message.Role
		item["content"] = message.Parts[0].Text
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
	req.Metadata = collectUnknownFields(raw, "model", "stream", "messages")
	if err := decodeRawString(raw, "model", &req.Model, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	_ = decodeRawBool(raw, "stream", &req.Stream)

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
		var content string
		if err := decodeRawString(item, "content", &content, true); err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("messages[%d].content: %w", i, err)
		}
		req.Messages = append(req.Messages, domain.UnifiedMessage{
			Role:     role,
			Parts:    []domain.UnifiedPart{{Type: "text", Text: content}},
			Metadata: collectUnknownFields(item, "role", "content"),
		})
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

func decodeOpenAIResponseMessage(choice map[string]json.RawMessage) (domain.UnifiedMessage, error) {
	var messageRaw map[string]json.RawMessage
	if err := decodeRaw(choice, "message", &messageRaw, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message: %w", err)
	}
	var role string
	if err := decodeRawString(messageRaw, "role", &role, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message.role: %w", err)
	}
	var content string
	if err := decodeRawString(messageRaw, "content", &content, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("message.content: %w", err)
	}
	return domain.UnifiedMessage{
		Role:     role,
		Parts:    []domain.UnifiedPart{{Type: "text", Text: content}},
		Metadata: collectUnknownFields(messageRaw, "role", "content"),
	}, nil
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
	usage := map[string]int64{}
	if err := json.Unmarshal(usageRaw, &usage); err != nil {
		return fmt.Errorf("usage 格式非法: %w", err)
	}
	*target = usage
	return nil
}
