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
	if len(req.Tools) > 0 {
		payload["tools"] = rawMessagesToAny(req.Tools)
	}
	systemInstruction, contents, err := encodeGeminiMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	if systemInstruction != nil {
		payload["system_instruction"] = systemInstruction
	}
	payload["contents"] = contents
	return json.Marshal(payload)
}

func DecodeGeminiChatRequest(body []byte, pathModel string) (domain.UnifiedChatRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.UnifiedChatRequest{}, fmt.Errorf("解析 Gemini 请求失败: %w", err)
	}

	req := domain.UnifiedChatRequest{Protocol: domain.ProtocolGemini}
	req.Metadata = collectUnknownFields(raw, "model", "stream", "contents", "generationConfig", "system_instruction", "systemInstruction", "tools")
	var bodyModel string
	_ = decodeRawString(raw, "model", &bodyModel, false)
	resolvedModel, err := resolveGeminiModel(pathModel, bodyModel)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Model = resolvedModel
	_ = decodeRawBool(raw, "stream", &req.Stream)

	if toolsRaw, ok := raw["tools"]; ok {
		var tools []json.RawMessage
		if err := json.Unmarshal(toolsRaw, &tools); err != nil {
			return domain.UnifiedChatRequest{}, fmt.Errorf("tools 格式非法: %w", err)
		}
		req.Tools = tools
	}
	if configRaw, ok := raw["generationConfig"]; ok {
		if req.Metadata == nil {
			req.Metadata = map[string]json.RawMessage{}
		}
		req.Metadata["generationConfig"] = append(json.RawMessage(nil), configRaw...)
	}

	for _, key := range []string{"system_instruction", "systemInstruction"} {
		if systemRaw, ok := raw[key]; ok {
			message, err := decodeGeminiSystemInstruction(systemRaw)
			if err != nil {
				return domain.UnifiedChatRequest{}, err
			}
			req.Messages = append(req.Messages, message)
			break
		}
	}

	var contentsRaw []map[string]json.RawMessage
	if err := decodeRaw(raw, "contents", &contentsRaw, true); err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	messages, err := decodeGeminiMessages(contentsRaw)
	if err != nil {
		return domain.UnifiedChatRequest{}, err
	}
	req.Messages = append(req.Messages, messages...)

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

func EncodeGeminiChatResponse(resp domain.UnifiedChatResponse) ([]byte, error) {
	payload := map[string]any{}
	mergeRawFields(payload, resp.Metadata)
	if resp.Model != "" {
		payload["modelVersion"] = resp.Model
	}
	parts, err := encodeGeminiParts(resp.Message.Parts)
	if err != nil {
		return nil, err
	}
	payload["candidates"] = []map[string]any{{
		"content": map[string]any{
			"role":  "model",
			"parts": parts,
		},
		"finishReason": resp.FinishReason,
	}}
	if len(resp.Usage) > 0 {
		payload["usageMetadata"] = map[string]any{
			"promptTokenCount":     resp.Usage["prompt_tokens"],
			"candidatesTokenCount": resp.Usage["completion_tokens"],
			"totalTokenCount":      resp.Usage["total_tokens"],
		}
	}
	return json.Marshal(payload)
}

func encodeGeminiMessages(messages []domain.UnifiedMessage) (map[string]any, []map[string]any, error) {
	var systemInstruction map[string]any
	out := make([]map[string]any, 0, len(messages))
	for i, message := range messages {
		parts, err := encodeGeminiParts(message.Parts)
		if err != nil {
			return nil, nil, fmt.Errorf("messages[%d]: %w", i, err)
		}
		if strings.EqualFold(strings.TrimSpace(message.Role), "system") {
			systemInstruction = map[string]any{"parts": parts}
			continue
		}
		item := map[string]any{}
		mergeRawFields(item, message.Metadata)
		item["role"] = geminiRoleFromUnified(message.Role)
		if strings.EqualFold(strings.TrimSpace(message.Role), "tool") {
			item["role"] = "user"
		}
		item["parts"] = parts
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, nil, fmt.Errorf("Gemini 请求至少需要一条非 system 消息")
	}
	return systemInstruction, out, nil
}

func encodeGeminiParts(parts []domain.UnifiedPart) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(parts))
	for i, part := range parts {
		item, err := encodeGeminiPart(part)
		if err != nil {
			return nil, fmt.Errorf("parts[%d]: %w", i, err)
		}
		out = append(out, item)
	}
	return out, nil
}

func encodeGeminiPart(part domain.UnifiedPart) (map[string]any, error) {
	item := map[string]any{}
	mergeRawFields(item, part.Metadata)
	desc := extractMediaDescriptor(part)
	switch part.Type {
	case "text":
		item["text"] = part.Text
	case "function_call":
		item["functionCall"] = map[string]any{"id": decodeStringRaw(part.Metadata["id"]), "name": decodeStringRaw(part.Metadata["name"]), "args": rawJSONToAny(part.Metadata["arguments"])}
	case "function_response":
		item["functionResponse"] = map[string]any{"id": decodeStringRaw(part.Metadata["id"]), "name": decodeStringRaw(part.Metadata["name"]), "response": rawJSONToAny(part.Metadata["response"])}
	case "image", "audio", "video", "document", "file":
		if _, ok := item["inlineData"]; !ok {
			if _, exists := item["fileData"]; !exists {
				switch {
				case desc.Data != "":
					item["inlineData"] = map[string]any{"mimeType": desc.MimeType, "data": desc.Data}
				case desc.FileURI != "":
					item["fileData"] = map[string]any{"mimeType": desc.MimeType, "fileUri": desc.FileURI}
				case desc.URL != "":
					item["fileData"] = map[string]any{"mimeType": desc.MimeType, "fileUri": desc.URL}
				default:
					return nil, fmt.Errorf("%s part 缺少 inlineData/fileData", part.Type)
				}
			}
		}
	default:
		if len(item) > 0 {
			return item, nil
		}
		return nil, fmt.Errorf("Gemini 暂不支持 part 类型: %s", part.Type)
	}
	return item, nil
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

	var partsRaw []map[string]json.RawMessage
	if err := decodeRaw(content, "parts", &partsRaw, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("parts: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(partsRaw))
	toolCalls := make([]domain.UnifiedToolCall, 0)
	toolResultMode := false
	for i, raw := range partsRaw {
		part, err := decodeGeminiPart(raw)
		if err != nil {
			return domain.UnifiedMessage{}, fmt.Errorf("parts[%d]: %w", i, err)
		}
		if part.Type == "function_call" {
			toolCalls = append(toolCalls, domain.UnifiedToolCall{ID: decodeStringRaw(part.Metadata["id"]), Name: decodeStringRaw(part.Metadata["name"]), Arguments: append(json.RawMessage(nil), part.Metadata["arguments"]...), Metadata: collectUnknownFields(part.Metadata, "id", "name", "arguments")})
			continue
		}
		if part.Type == "function_response" {
			toolResultMode = true
			part = domain.UnifiedPart{Type: "text", Text: string(part.Metadata["response"]), Metadata: map[string]json.RawMessage{"tool_call_id": append(json.RawMessage(nil), part.Metadata["id"]...), "tool_name": append(json.RawMessage(nil), part.Metadata["name"]...)}}
		}
		parts = append(parts, part)
	}
	if toolResultMode {
		unifiedRole = "tool"
	}
	return domain.UnifiedMessage{Role: unifiedRole, Parts: parts, ToolCalls: toolCalls, Metadata: collectUnknownFields(content, "role", "parts")}, nil
}

func decodeGeminiSystemInstruction(systemRaw json.RawMessage) (domain.UnifiedMessage, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(systemRaw, &payload); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("system_instruction 格式非法: %w", err)
	}
	var partsRaw []map[string]json.RawMessage
	if err := decodeRaw(payload, "parts", &partsRaw, true); err != nil {
		return domain.UnifiedMessage{}, fmt.Errorf("system_instruction.parts: %w", err)
	}
	parts := make([]domain.UnifiedPart, 0, len(partsRaw))
	for i, raw := range partsRaw {
		part, err := decodeGeminiPart(raw)
		if err != nil {
			return domain.UnifiedMessage{}, fmt.Errorf("system_instruction.parts[%d]: %w", i, err)
		}
		parts = append(parts, part)
	}
	return domain.UnifiedMessage{Role: "system", Parts: parts, Metadata: collectUnknownFields(payload, "parts")}, nil
}

func decodeGeminiPart(raw map[string]json.RawMessage) (domain.UnifiedPart, error) {
	if textRaw, ok := raw["text"]; ok {
		var text string
		if err := json.Unmarshal(textRaw, &text); err != nil {
			return domain.UnifiedPart{}, fmt.Errorf("text 格式非法: %w", err)
		}
		return domain.UnifiedPart{Type: "text", Text: text, Metadata: collectUnknownFields(raw, "text")}, nil
	}
	metadata := collectUnknownFields(raw)
	if functionCallRaw, ok := raw["functionCall"]; ok {
		metadata = cloneRawMap(metadata)
		metadata["functionCall"] = append(json.RawMessage(nil), functionCallRaw...)
		var call map[string]json.RawMessage
		if err := json.Unmarshal(functionCallRaw, &call); err != nil {
			return domain.UnifiedPart{}, fmt.Errorf("functionCall 格式非法: %w", err)
		}
		metadata["id"] = append(json.RawMessage(nil), call["id"]...)
		metadata["name"] = append(json.RawMessage(nil), call["name"]...)
		metadata["arguments"] = append(json.RawMessage(nil), call["args"]...)
		return domain.UnifiedPart{Type: "function_call", Metadata: metadata}, nil
	}
	if functionResponseRaw, ok := raw["functionResponse"]; ok {
		metadata = cloneRawMap(metadata)
		metadata["functionResponse"] = append(json.RawMessage(nil), functionResponseRaw...)
		var response map[string]json.RawMessage
		if err := json.Unmarshal(functionResponseRaw, &response); err != nil {
			return domain.UnifiedPart{}, fmt.Errorf("functionResponse 格式非法: %w", err)
		}
		metadata["id"] = append(json.RawMessage(nil), response["id"]...)
		metadata["name"] = append(json.RawMessage(nil), response["name"]...)
		metadata["response"] = append(json.RawMessage(nil), response["response"]...)
		return domain.UnifiedPart{Type: "function_response", Metadata: metadata}, nil
	}
	if inlineRaw, ok := raw["inlineData"]; ok {
		metadata = cloneRawMap(metadata)
		metadata["inlineData"] = append(json.RawMessage(nil), inlineRaw...)
		desc := extractMediaDescriptor(domain.UnifiedPart{Metadata: metadata})
		partType := partTypeFromMime(desc.MimeType)
		metadata = enrichPartMetadata(partType, metadata, desc)
		return domain.UnifiedPart{Type: partType, Metadata: metadata}, nil
	}
	if fileRaw, ok := raw["fileData"]; ok {
		metadata = cloneRawMap(metadata)
		metadata["fileData"] = append(json.RawMessage(nil), fileRaw...)
		desc := extractMediaDescriptor(domain.UnifiedPart{Metadata: metadata})
		partType := partTypeFromMime(desc.MimeType)
		metadata = enrichPartMetadata(partType, metadata, desc)
		return domain.UnifiedPart{Type: partType, Metadata: metadata}, nil
	}
	metadata = cloneRawMap(raw)
	return domain.UnifiedPart{Type: "file", Metadata: metadata}, nil
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
