package provider

import (
	"strings"

	"opencrab/internal/domain"
)

func projectOpenAIResponsesRequest(req domain.UnifiedChatRequest, session *domain.GatewaySessionState) domain.UnifiedChatRequest {
	projected := req
	projected.Messages = sanitizeOpenAIResponsesMessages(req.Messages)
	if session != nil && strings.TrimSpace(session.PreviousResponseID) != "" {
		projected.Messages = projectResponsesContinueMessages(projected.Messages)
	}
	return projected
}

func sanitizeOpenAIResponsesMessages(messages []domain.UnifiedMessage) []domain.UnifiedMessage {
	if len(messages) == 0 {
		return messages
	}
	filtered := make([]domain.UnifiedMessage, 0, len(messages))
	for _, message := range messages {
		next := domain.UnifiedMessage{
			Role:      message.Role,
			ToolCalls: cloneUnifiedToolCalls(message.ToolCalls),
			InputItem: appendJSON(message.InputItem),
			Metadata:  cloneRawMap(message.Metadata),
		}
		if len(message.Parts) > 0 {
			next.Parts = make([]domain.UnifiedPart, 0, len(message.Parts))
			for _, part := range message.Parts {
				cleaned, keep := normalizeResponsesTextPart(part)
				if keep {
					next.Parts = append(next.Parts, cleaned)
				}
			}
		}
		if len(next.Parts) == 0 && len(next.ToolCalls) == 0 && len(next.InputItem) == 0 {
			continue
		}
		filtered = append(filtered, next)
	}
	if len(filtered) == 0 {
		return messages
	}
	return filtered
}

func normalizeResponsesTextPart(part domain.UnifiedPart) (domain.UnifiedPart, bool) {
	cloned := cloneUnifiedPart(part)
	if !strings.EqualFold(strings.TrimSpace(cloned.Type), "text") {
		return cloned, true
	}
	text := sanitizeResponsesText(cloned.Text)
	if strings.TrimSpace(text) == "" {
		return domain.UnifiedPart{}, false
	}
	cloned.Text = text
	return cloned, true
}

func sanitizeResponsesText(text string) string {
	text = strings.ReplaceAll(text, "\r", "")
	if strings.TrimSpace(text) == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lowered := strings.ToLower(trimmed)
		switch {
		case lowered == "request interrupted by user", lowered == "request cancelled by user":
			continue
		case strings.Contains(lowered, "<system-reminder>"):
			continue
		case strings.HasPrefix(lowered, "user system info"), strings.HasPrefix(lowered, "today's date:"), strings.HasPrefix(lowered, "model:"), strings.HasPrefix(lowered, "execute calls:"), strings.HasPrefix(lowered, "tool usage guidelines:"), strings.HasPrefix(lowered, "working directory management:"), strings.HasPrefix(lowered, "directory verification:"), strings.HasPrefix(lowered, "path quoting:"), strings.HasPrefix(lowered, "correct:"), strings.HasPrefix(lowered, "incorrect"), strings.HasPrefix(lowered, "how can i help?"), strings.HasPrefix(lowered, "skill activated"), strings.HasPrefix(lowered, "plan ."), strings.Contains(lowered, "? for help | mcp"):
			continue
		case strings.HasPrefix(trimmed, "% "), strings.HasPrefix(trimmed, "# The commands below"):
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return strings.Join(filtered, "\n")
}

func projectResponsesContinueMessages(messages []domain.UnifiedMessage) []domain.UnifiedMessage {
	if len(messages) <= 1 {
		return messages
	}
	leadingSystems := leadingSystemMessageCount(messages)
	if preserveFrom := findTrailingPendingToolExchangeStartUnified(messages); preserveFrom >= 0 {
		if preserveFrom < leadingSystems {
			preserveFrom = leadingSystems
		}
		return appendMessagesWindow(messages[:leadingSystems], messages[preserveFrom:])
	}
	lastAssistant := -1
	for i := leadingSystems; i < len(messages); i++ {
		if strings.EqualFold(messages[i].Role, "assistant") {
			lastAssistant = i
		}
	}
	if lastAssistant >= 0 && lastAssistant < len(messages)-1 {
		return appendMessagesWindow(messages[:leadingSystems], messages[lastAssistant+1:])
	}
	return messages
}

func leadingSystemMessageCount(messages []domain.UnifiedMessage) int {
	count := 0
	for count < len(messages) && strings.EqualFold(messages[count].Role, "system") {
		count++
	}
	return count
}

func findTrailingPendingToolExchangeStartUnified(messages []domain.UnifiedMessage) int {
	if len(messages) == 0 {
		return -1
	}
	last := messages[len(messages)-1]
	if !strings.EqualFold(last.Role, "tool") {
		return -1
	}
	start := len(messages) - 1
	for start > 0 {
		prev := messages[start-1]
		if strings.EqualFold(prev.Role, "tool") || (strings.EqualFold(prev.Role, "assistant") && len(prev.ToolCalls) > 0) {
			start--
			continue
		}
		break
	}
	return start
}

func appendMessagesWindow(head []domain.UnifiedMessage, tail []domain.UnifiedMessage) []domain.UnifiedMessage {
	combined := make([]domain.UnifiedMessage, 0, len(head)+len(tail))
	for _, message := range head {
		combined = append(combined, cloneUnifiedMessage(message))
	}
	for _, message := range tail {
		combined = append(combined, cloneUnifiedMessage(message))
	}
	return combined
}

func cloneUnifiedMessage(message domain.UnifiedMessage) domain.UnifiedMessage {
	cloned := domain.UnifiedMessage{
		Role:      message.Role,
		InputItem: appendJSON(message.InputItem),
		Metadata:  cloneRawMap(message.Metadata),
	}
	if len(message.Parts) > 0 {
		cloned.Parts = make([]domain.UnifiedPart, 0, len(message.Parts))
		for _, part := range message.Parts {
			cloned.Parts = append(cloned.Parts, cloneUnifiedPart(part))
		}
	}
	cloned.ToolCalls = cloneUnifiedToolCalls(message.ToolCalls)
	return cloned
}

func cloneUnifiedToolCalls(calls []domain.UnifiedToolCall) []domain.UnifiedToolCall {
	if len(calls) == 0 {
		return nil
	}
	cloned := make([]domain.UnifiedToolCall, 0, len(calls))
	for _, call := range calls {
		next := call
		next.Arguments = appendJSON(call.Arguments)
		next.InputItem = appendJSON(call.InputItem)
		next.OutputItem = appendJSON(call.OutputItem)
		next.NativePayload = appendJSON(call.NativePayload)
		next.Metadata = cloneRawMap(call.Metadata)
		cloned = append(cloned, next)
	}
	return cloned
}

func appendJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	return append([]byte(nil), raw...)
}
