package provider

import (
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"
)

func BuildOpenAIRealtimeEvents(resp domain.UnifiedChatResponse, previousItemID string) ([]map[string]any, string, error) {
	responseID := strings.TrimSpace(resp.ID)
	if responseID == "" {
		responseID = fmt.Sprintf("resp_rt_%d", time.Now().UnixNano())
	}

	output, _, err := encodeResponsesOutput(resp.Message)
	if err != nil {
		return nil, previousItemID, err
	}

	responseObject := map[string]any{
		"id":     responseID,
		"object": "realtime.response",
		"status": "completed",
		"output": output,
	}
	if strings.TrimSpace(resp.Model) != "" {
		responseObject["model"] = resp.Model
	}
	if usage := encodeResponsesUsage(resp.Usage); usage != nil {
		responseObject["usage"] = usage
	}

	events := []map[string]any{
		{
			"type": "response.created",
			"response": map[string]any{
				"id":     responseID,
				"object": "realtime.response",
				"status": "in_progress",
				"output": []any{},
				"model":  resp.Model,
			},
		},
	}

	lastItemID := previousItemID
	for index, item := range output {
		itemID := resolveResponsesEventItemID(item, index)
		item["id"] = itemID
		if _, ok := item["object"]; !ok {
			item["object"] = "realtime.item"
		}
		addedItem := cloneRealtimeItem(item)
		addedItem["status"] = "in_progress"

		events = append(events,
			buildRealtimeConversationItemEvent("conversation.item.added", lastItemID, addedItem),
			buildRealtimeOutputItemEvent("response.output_item.added", responseID, index, addedItem),
		)

		appendRealtimeOutputEvents(&events, responseID, index, itemID, item)

		doneItem := cloneRealtimeItem(item)
		doneItem["status"] = "completed"
		events = append(events,
			buildRealtimeConversationItemEvent("conversation.item.done", lastItemID, doneItem),
			buildRealtimeOutputItemEvent("response.output_item.done", responseID, index, doneItem),
		)

		lastItemID = itemID
	}

	events = append(events, map[string]any{
		"type":     "response.done",
		"response": responseObject,
	})
	return events, lastItemID, nil
}

func appendRealtimeOutputEvents(events *[]map[string]any, responseID string, outputIndex int, itemID string, item map[string]any) {
	itemType, _ := item["type"].(string)
	switch itemType {
	case "message":
		for contentIndex, part := range responsesAnySliceToMaps(item["content"]) {
			*events = append(*events, map[string]any{
				"type":          "response.content_part.added",
				"response_id":   responseID,
				"output_index":  outputIndex,
				"item_id":       itemID,
				"content_index": contentIndex,
				"part":          part,
			})
			partType, _ := part["type"].(string)
			if partType == "output_text" {
				text, _ := part["text"].(string)
				*events = append(*events,
					map[string]any{
						"type":          "response.output_text.delta",
						"response_id":   responseID,
						"output_index":  outputIndex,
						"item_id":       itemID,
						"content_index": contentIndex,
						"delta":         text,
					},
					map[string]any{
						"type":          "response.output_text.done",
						"response_id":   responseID,
						"output_index":  outputIndex,
						"item_id":       itemID,
						"content_index": contentIndex,
						"text":          text,
					},
				)
			}
			*events = append(*events, map[string]any{
				"type":          "response.content_part.done",
				"response_id":   responseID,
				"output_index":  outputIndex,
				"item_id":       itemID,
				"content_index": contentIndex,
				"part":          part,
			})
		}
	case "function_call":
		arguments, _ := item["arguments"].(string)
		if strings.TrimSpace(arguments) != "" {
			*events = append(*events,
				map[string]any{
					"type":         "response.function_call_arguments.delta",
					"response_id":  responseID,
					"output_index": outputIndex,
					"item_id":      itemID,
					"delta":        arguments,
				},
				map[string]any{
					"type":         "response.function_call_arguments.done",
					"response_id":  responseID,
					"output_index": outputIndex,
					"item_id":      itemID,
					"arguments":    arguments,
				},
			)
		}
	}
}

func buildRealtimeConversationItemEvent(eventType string, previousItemID string, item map[string]any) map[string]any {
	event := map[string]any{
		"type": eventType,
		"item": item,
	}
	if strings.TrimSpace(previousItemID) != "" {
		event["previous_item_id"] = previousItemID
	}
	return event
}

func buildRealtimeOutputItemEvent(eventType string, responseID string, outputIndex int, item map[string]any) map[string]any {
	return map[string]any{
		"type":         eventType,
		"response_id":  responseID,
		"output_index": outputIndex,
		"item":         item,
	}
}

func cloneRealtimeItem(item map[string]any) map[string]any {
	cloned := make(map[string]any, len(item))
	for key, value := range item {
		cloned[key] = value
	}
	return cloned
}
