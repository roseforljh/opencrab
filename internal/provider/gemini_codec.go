package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
)

func EncodeGeminiChatRequest(req domain.UnifiedChatRequest) ([]byte, error) {
	if req.Protocol == "" {
		req.Protocol = domain.ProtocolGemini
	}
	if req.Protocol != domain.ProtocolGemini {
		return nil, fmt.Errorf("Gemini codec 不支持协议: %s", req.Protocol)
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
	payload["contents"] = encodeGeminiMessages(req.Messages)
	return json.Marshal(payload)
}

func DecodeGeminiChatRequest(body []byte, pathModel string) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 Gemini 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolGemini}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "contents", "generationConfig")

	var bodyModel string
	_ = decodeRawString(raw, "model", &bodyModel, false)
	resolvedModel, err := resolveGeminiModel(pathModel, bodyModel)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Model = resolvedModel
	_ = decodeRawBool(raw, "stream", &req.Stream)

	var contentsRaw []map[string]json.RawMessage
	if err := decodeRaw(raw, "contents", &contentsRaw, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	messages, err := decodeGeminiMessages(contentsRaw)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = messages

	if configRaw, ok := raw["generationConfig"]; ok {
		if req.Metadata == nil {
			req.Metadata = map[string]json.RawMessage{}
		}
		req.Metadata["generationConfig"] = append(json.RawMessage(nil), configRaw...)
	}

	if err := req.ValidateCore(); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	return req, nil
}

func DecodeGeminiChatResponse(body []byte) (domain.UnifiedChatResponse, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatResponse{}, fmt.Errorf("解析 Gemini 响应失败: %w", err)
	}

	resp := domain.UnifiedChatResponse{Protocol: domain.ProtocolGemini}
	resp.Metadata = collectUnknownFields(raw, "modelVersion", "candidates", "usageMetadata")
	_ = decodeRawString(raw, "modelVersion", &resp.Model, false)

	if usageRaw, ok := raw["usageMetadata"]; ok {
		usage := map[string]int64{}
		var source map[string]json.RawMessage
		if err := json.Unmarshal(usageRaw, &source); err == nil {
			extractInt64(source, "promptTokenCount", usage, "prompt_tokens")
			extractInt64(source, "candidatesTokenCount", usage, "completion_tokens")
			extractInt64(source, "totalTokenCount", usage, "total_tokens")
		}
		if len(usage) > 0 {
			resp.Usage = usage
		}
	}

	var candidates []map[string]json.RawMessage
	if err := decodeRaw(raw, "candidates", &candidates, true); err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	if len(candidates) == 0 {
		return domain.UnifiedChatResponse{}, fmt.Errorf("candidates 不能为空")
	}
	_ = decodeRawString(candidates[0], "finishReason", &resp.FinishReason, false)

	message, err := decodeGeminiCandidate(candidates[0])
	if err != nil {
		return domain.UnifiedChatResponse{}, err
	}
	resp.Message = message
	return resp, nil
}

func encodeGeminiMessages(messages []domain.UnifiedMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		item["role"] = geminiRoleFromUnified(message.Role)
		item["parts"] = []map[string]any{{"text": message.Parts[0].Text}}
		out = append(out, item)
	}
	return out
}

func decodeGeminiMessages(items []map[string]json.RawMessage) ([]domain.UnifiedMessage, error) {
	messages := make([]domain.UnifiedMessage, 0, len(items))
	for i, item := range items {
		message, err := decodeGeminiContent(item)
		if err != nil {
			return nil, fmt.Errorf("contents[%d]: %w", i, err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func decodeGeminiCandidate(raw map[string]json.RawMessage) (domain.UnifiedMessage, error) {
	var content map[string]json.RawMessage
	if err := decodeRaw(raw, "content", &content, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("content: %w", err)
	}
	return decodeGeminiContent(content)
}

func decodeGeminiContent(content map[string]json.RawMessage) (domain.UnifiedMessage, error) {
	var role string
	_ = decodeRawString(content, "role", &role, false)
	unifiedRole := unifiedRoleFromGemini(role)

	var parts []map[string]json.RawMessage
	if err := decodeRaw(content, "parts", &parts, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("parts: %w", err)
	}
	if len(parts) != 1 {
		return domain.UnifiedMessage{}, fmt.Errorf("当前仅支持单个 Gemini text part")
	}

	var text string
	if err := decodeRawString(parts[0], "text", &text, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("parts[0].text: %w", err)
	}
	if len(collectUnknownFields(parts[0], "text")) > 0 {
		return domain.UnifiedMessage{}, fmt.Errorf("当前仅支持 text-only 主链路，Gemini part metadata 暂不支持")
	}

	return domain.UnifiedMessage{
		Role:     unifiedRole,
		Parts:    []domain.UnifiedPart{{Type: "text", Text: text}},
		Metadata: collectUnknownFields(content, "role", "parts"),
	}, nil
}

func resolveGeminiModel(pathModel string, bodyModel string) (string, error) {
	pathModel = strings.TrimSpace(pathModel)
	bodyModel = strings.TrimSpace(bodyModel)
	switch {
	case pathModel == "" && bodyModel == "":
		return "", fmt.Errorf("model 不能为空")
	case pathModel == "":
		return bodyModel, nil
	case bodyModel == "":
		return pathModel, nil
	case pathModel != bodyModel:
		return "", fmt.Errorf("Gemini path model 与 body model 冲突: %s != %s", pathModel, bodyModel)
	default:
		return pathModel, nil
	}
}

func geminiRoleFromUnified(role string) string {
	if role == "assistant" {
		return "model"
	}
	return role
}

func unifiedRoleFromGemini(role string) string {
	role = strings.TrimSpace(role)
	if role == "model" || role == "" {
		return "assistant"
	}
	return role
}

func extractInt64(raw map[string]json.RawMessage, key string, dst map[string]int64, targetKey string) {
	var value int64
	if err := decodeRaw(raw, key, &value, false); err == nil {
		dst[targetKey] = value
	}
}
