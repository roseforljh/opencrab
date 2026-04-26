package transform

import (
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestNormalizeGatewayRequestCodexResponses(t *testing.T) {
	req, err := NormalizeGatewayRequest(
		Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses},
		[]byte(`{"model":"gpt-5.4","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`),
		NormalizeOptions{},
	)
	if err != nil {
		t.Fatalf("normalize codex request: %v", err)
	}
	if req.Protocol != domain.ProtocolCodex || req.Operation != domain.ProtocolOperationCodexResponses {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestRenderClientResponseCodexResponses(t *testing.T) {
	body, headers, err := RenderClientResponse(
		Surface{Protocol: domain.ProtocolCodex, Operation: domain.ProtocolOperationCodexResponses},
		domain.UnifiedChatResponse{
			Protocol: domain.ProtocolCodex,
			ID:       "resp_1",
			Model:    "gpt-5.4",
			Message:  domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
		},
		false,
	)
	if err != nil {
		t.Fatalf("render codex response: %v", err)
	}
	if headers["Content-Type"][0] != "application/json" || !strings.Contains(string(body), `"object":"response"`) {
		t.Fatalf("unexpected rendered codex response: headers=%#v body=%s", headers, string(body))
	}
}

func TestRenderClientResponseGeminiStream(t *testing.T) {
	body, headers, err := RenderClientResponse(
		Surface{Protocol: domain.ProtocolGemini, Operation: domain.ProtocolOperationGeminiStreamGenerate},
		domain.UnifiedChatResponse{
			Protocol: domain.ProtocolGemini,
			Model:    "gemini-2.5-pro",
			Message:  domain.UnifiedMessage{Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}},
		},
		true,
	)
	if err != nil {
		t.Fatalf("render gemini stream: %v", err)
	}
	if headers["Content-Type"][0] != "text/event-stream" || !strings.Contains(string(body), `"candidates"`) {
		t.Fatalf("unexpected gemini stream: headers=%#v body=%s", headers, string(body))
	}
}

func TestBuildExecutorPayloadBridgesClaudeToolResultToResponsesInputText(t *testing.T) {
	gatewayReq, err := NormalizeGatewayRequest(
		Surface{Protocol: domain.ProtocolClaude, Operation: domain.ProtocolOperationClaudeMessages},
		[]byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"tools":[{"name":"Read","input_schema":{"type":"object"}}],"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":[{"type":"text","text":"Launching skill: pua"}]}]}]}`),
		NormalizeOptions{},
	)
	if err != nil {
		t.Fatalf("normalize claude request: %v", err)
	}
	unified := domain.UnifiedChatRequest{
		Protocol: gatewayReq.Protocol,
		Model:    "gpt-5.4",
		Stream:   gatewayReq.Stream,
		Tools:    gatewayReq.Tools,
		Metadata: gatewayReq.Metadata,
	}
	for _, message := range gatewayReq.Messages {
		unified.Messages = append(unified.Messages, domain.UnifiedMessage{
			Role:      message.Role,
			Parts:     message.Parts,
			ToolCalls: message.ToolCalls,
			InputItem: message.InputItem,
			Metadata:  message.Metadata,
		})
	}
	payload, err := BuildExecutorPayload(
		Surface{Protocol: domain.ProtocolOpenAI, Operation: domain.ProtocolOperationOpenAIResponses},
		unified,
		gatewayReq.Session,
		"https://api.example.com/v1",
		"gpt-5.4",
	)
	if err != nil {
		t.Fatalf("build responses payload: %v", err)
	}
	body := string(payload.Body)
	if !strings.Contains(body, `"type":"function_call_output"`) || !strings.Contains(body, `"call_id":"call_1"`) {
		t.Fatalf("expected bridged function_call_output in responses payload: %s", body)
	}
	if strings.Contains(body, `"output":[{"text":"Launching skill: pua","type":"text"}]`) || strings.Contains(body, `"type":"text","text":"Launching skill: pua"`) {
		t.Fatalf("claude tool_result text block should not leak as responses text type: %s", body)
	}
	if strings.Contains(body, `"output":[{"text":"Launching skill: pua","type":"output_text"}]`) {
		t.Fatalf("claude tool_result text block must not become output_text: %s", body)
	}
	if !strings.Contains(body, `"output":[{"text":"Launching skill: pua","type":"input_text"}]`) {
		t.Fatalf("expected claude tool_result text block to become input_text: %s", body)
	}
}
