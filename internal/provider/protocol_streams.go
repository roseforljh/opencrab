package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"
)

func EncodeOpenAIChatStream(resp domain.UnifiedChatResponse) ([]byte, error) {
	chunkID := resp.ID
	if chunkID == "" {
		chunkID = fmt.Sprintf("chatcmpl_%d", time.Now().UnixNano())
	}
	created := time.Now().Unix()
	chunks := make([]string, 0, 4)
	chunks = append(chunks, mustOpenAISSE(map[string]any{"id": chunkID, "object": "chat.completion.chunk", "created": created, "model": resp.Model, "choices": []map[string]any{{"index": 0, "delta": map[string]any{"role": "assistant"}, "finish_reason": nil}}}))
	if text := firstUnifiedText(resp.Message); text != "" {
		chunks = append(chunks, mustOpenAISSE(map[string]any{"id": chunkID, "object": "chat.completion.chunk", "created": created, "model": resp.Model, "choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": text}, "finish_reason": nil}}}))
	}
	if len(resp.Message.ToolCalls) > 0 {
		for _, call := range resp.Message.ToolCalls {
			chunks = append(chunks, mustOpenAISSE(map[string]any{"id": chunkID, "object": "chat.completion.chunk", "created": created, "model": resp.Model, "choices": []map[string]any{{"index": 0, "delta": map[string]any{"tool_calls": []map[string]any{{"id": call.ID, "type": "function", "function": map[string]any{"name": call.Name, "arguments": string(call.Arguments)}}}}, "finish_reason": nil}}}))
		}
	}
	finish := resp.FinishReason
	if finish == "" {
		if len(resp.Message.ToolCalls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	chunks = append(chunks, mustOpenAISSE(map[string]any{"id": chunkID, "object": "chat.completion.chunk", "created": created, "model": resp.Model, "choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}}))
	chunks = append(chunks, "data: [DONE]\n\n")
	return []byte(strings.Join(chunks, "")), nil
}

func EncodeClaudeChatStream(resp domain.UnifiedChatResponse) ([]byte, error) {
	messageID := resp.ID
	if messageID == "" {
		messageID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	inputTokens := firstUsageValue(resp.Usage, "input_tokens", "prompt_tokens")
	outputTokens := firstUsageValue(resp.Usage, "output_tokens", "completion_tokens")
	cacheReadTokens := firstUsageValue(resp.Usage, "cache_read_input_tokens", "prompt_cache_hit_tokens")
	cacheCreationTokens := firstUsageValue(resp.Usage, "cache_creation_input_tokens")
	events := make([]string, 0, 8)
	messageStartUsage := map[string]any{"input_tokens": inputTokens, "output_tokens": 0}
	if cacheReadTokens > 0 {
		messageStartUsage["cache_read_input_tokens"] = cacheReadTokens
	}
	if cacheCreationTokens > 0 {
		messageStartUsage["cache_creation_input_tokens"] = cacheCreationTokens
	}
	messageStart := map[string]any{"type": "message_start", "message": map[string]any{"id": messageID, "type": "message", "role": "assistant", "model": resp.Model, "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": messageStartUsage}}
	events = append(events, mustClaudeSSE("message_start", messageStart))
	blockIndex := 0
	text := firstUnifiedText(resp.Message)
	if text != "" || len(resp.Message.ToolCalls) == 0 {
		events = append(events, mustClaudeSSE("content_block_start", map[string]any{"type": "content_block_start", "index": blockIndex, "content_block": map[string]any{"type": "text", "text": ""}}))
		if text != "" {
			events = append(events, mustClaudeSSE("content_block_delta", map[string]any{"type": "content_block_delta", "index": blockIndex, "delta": map[string]any{"type": "text_delta", "text": text}}))
		}
		events = append(events, mustClaudeSSE("content_block_stop", map[string]any{"type": "content_block_stop", "index": blockIndex}))
		blockIndex++
	}
	for _, call := range resp.Message.ToolCalls {
		events = append(events, mustClaudeSSE("content_block_start", map[string]any{"type": "content_block_start", "index": blockIndex, "content_block": map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": map[string]any{}}}))
		events = append(events, mustClaudeSSE("content_block_delta", map[string]any{"type": "content_block_delta", "index": blockIndex, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(call.Arguments)}}))
		events = append(events, mustClaudeSSE("content_block_stop", map[string]any{"type": "content_block_stop", "index": blockIndex}))
		blockIndex++
	}
	stopReason := resp.FinishReason
	if stopReason == "" {
		stopReason = "end_turn"
	}
	messageDeltaUsage := map[string]any{"output_tokens": outputTokens}
	if cacheReadTokens > 0 {
		messageDeltaUsage["cache_read_input_tokens"] = cacheReadTokens
	}
	if cacheCreationTokens > 0 {
		messageDeltaUsage["cache_creation_input_tokens"] = cacheCreationTokens
	}
	events = append(events, mustClaudeSSE("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil}, "usage": messageDeltaUsage}))
	events = append(events, mustClaudeSSE("message_stop", map[string]any{"type": "message_stop"}))
	return []byte(strings.Join(events, "")), nil
}

func EncodeGeminiChatStream(resp domain.UnifiedChatResponse) ([]byte, error) {
	body, err := EncodeGeminiChatResponse(resp)
	if err != nil {
		return nil, err
	}
	return []byte("data: " + string(body) + "\n\n"), nil
}

func mustOpenAISSE(payload map[string]any) string {
	body, _ := json.Marshal(payload)
	return fmt.Sprintf("data: %s\n\n", string(body))
}

func mustClaudeSSE(event string, payload map[string]any) string {
	body, _ := json.Marshal(payload)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(body))
}

func firstUsageValue(usage map[string]int64, keys ...string) int64 {
	for _, key := range keys {
		if usage[key] > 0 {
			return usage[key]
		}
	}
	return 0
}
