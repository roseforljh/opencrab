package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"


	"opencrab/internal/domain"
)

const anthropicVersion = "2023-06-01"
const upstreamRequestTimeout = 15 * time.Second

type OpenAIExecutor struct {
	client *http.Client
}

type ClaudeExecutor struct {
	client *http.Client
}

type GeminiExecutor struct {
	client *http.Client
}

type executorPayload struct {
	body   []byte
	url    string
	stream bool
}

func NewOpenAIExecutor(client *http.Client) *OpenAIExecutor {
	return &OpenAIExecutor{client: client}
}

func NewClaudeExecutor(client *http.Client) *ClaudeExecutor {
	return &ClaudeExecutor{client: client}
}

func NewGeminiExecutor(client *http.Client) *GeminiExecutor {
	return &GeminiExecutor{client: client}
}

func (e *OpenAIExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	normalizedProvider := domain.NormalizeProvider(input.Channel.Provider)
	targetOperation := plannedOperationForExecutor(input, normalizedProvider)
	effectiveRequest := input.Request
	effectiveRequest.Stream = shouldUseNativeUpstreamStream(input.Request, normalizedProvider, targetOperation)
	payload, err := buildExecutorPayload(
		domain.ProtocolOpenAI,
		targetOperation,
		markResponsesBridgeRepair(toUnifiedRequest(domain.ExecutorRequest{
			Channel:       input.Channel,
			UpstreamModel: input.UpstreamModel,
			Request:       effectiveRequest,
		}, domain.ProtocolOpenAI, normalizedProvider), input.Request.Protocol, targetOperation),
		effectiveRequest.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 OpenAI 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, upstreamRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, payload.url, bytes.NewReader(payload.body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 OpenAI 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"authorization": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, payload.stream, nil)
}

func (e *ClaudeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	normalizedProvider := domain.NormalizeProvider(input.Channel.Provider)
	effectiveRequest := input.Request
	effectiveRequest.Stream = shouldUseNativeUpstreamStream(input.Request, normalizedProvider, domain.ProtocolOperationClaudeMessages)
	unifiedReq := toUnifiedRequest(domain.ExecutorRequest{
		Channel:       input.Channel,
		UpstreamModel: input.UpstreamModel,
		Request:       effectiveRequest,
	}, domain.ProtocolClaude, normalizedProvider)
	payload, err := buildExecutorPayload(
		domain.ProtocolClaude,
		domain.ProtocolOperationClaudeMessages,
		unifiedReq,
		effectiveRequest.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Claude 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, upstreamRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, payload.url, bytes.NewReader(payload.body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Claude 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-api-key", input.Channel.APIKey)
	version := anthropicVersion
	if custom := strings.TrimSpace(input.Request.RequestHeaders["anthropic-version"]); custom != "" {
		version = custom
	}
	req.Header.Set("anthropic-version", version)
	if beta := strings.TrimSpace(mergeAnthropicBetaHeader(input.Request.RequestHeaders["anthropic-beta"], unifiedReq.Metadata)); beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"x-api-key": {}, "content-type": {}, "accept": {}, "anthropic-version": {}, "anthropic-beta": {}})
	return doExecutorRequest(e.client, req, payload.stream, nil)
}

func (e *GeminiExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	targetOperation := domain.ProtocolOperationGeminiGenerateContent
	if input.Request.Stream {
		targetOperation = domain.ProtocolOperationGeminiStreamGenerate
	}
	normalizedProvider := domain.NormalizeProvider(input.Channel.Provider)
	effectiveRequest := input.Request
	effectiveRequest.Stream = shouldUseNativeUpstreamStream(input.Request, normalizedProvider, targetOperation)
	if effectiveRequest.Stream {
		targetOperation = domain.ProtocolOperationGeminiStreamGenerate
	} else {
		targetOperation = domain.ProtocolOperationGeminiGenerateContent
	}
	payload, err := buildExecutorPayload(
		domain.ProtocolGemini,
		targetOperation,
		toUnifiedRequest(domain.ExecutorRequest{
			Channel:       input.Channel,
			UpstreamModel: input.UpstreamModel,
			Request:       effectiveRequest,
		}, domain.ProtocolGemini, normalizedProvider),
		effectiveRequest.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Gemini 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, upstreamRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, payload.url, bytes.NewReader(payload.body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Gemini 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-goog-api-key", input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"x-goog-api-key": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, payload.stream, payloadDebugMetadata(normalizedProvider, targetOperation, payload))
}

func toUnifiedRequest(input domain.ExecutorRequest, protocol domain.Protocol, targetProvider string) domain.UnifiedChatRequest {
	messages := make([]domain.UnifiedMessage, 0, len(input.Request.Messages))
	for _, message := range input.Request.Messages {
		messages = append(messages, domain.UnifiedMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, InputItem: message.InputItem, Metadata: message.Metadata})
	}
	metadata := cloneRawMap(input.Request.Metadata)
	tools := cloneRawMessages(input.Request.Tools)
	if input.Request.Protocol == "" {
		input.Request.Protocol = protocol
	}
	messages = sanitizeMessagesForTarget(messages, input.Request.Protocol, protocol, targetProvider)
	metadata, tools = sanitizeRequestForTarget(metadata, tools, input.Request.Protocol, protocol, targetProvider)
	return domain.UnifiedChatRequest{
		Protocol: protocol,
		Model:    input.UpstreamModel,
		Stream:   input.Request.Stream,
		Messages: messages,
		Tools:    tools,
		Metadata: metadata,
	}
}

func plannedOperationForExecutor(input domain.ExecutorRequest, normalizedProvider string) domain.ProtocolOperation {
	if normalizedProvider == "openai" {
		switch input.Request.Operation {
		case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
			return domain.ProtocolOperationOpenAIResponses
		}
	}
	return domain.ProtocolOperationOpenAIChatCompletions
}

func buildExecutorPayload(protocol domain.Protocol, operation domain.ProtocolOperation, req domain.UnifiedChatRequest, session *domain.GatewaySessionState, endpoint string, upstreamModel string) (executorPayload, error) {
	switch operation {
	case domain.ProtocolOperationOpenAIResponses, domain.ProtocolOperationCodexResponses:
		req.Protocol = domain.ProtocolOpenAI
		body, err := EncodeOpenAIResponsesRequest(req, session)
		if err != nil {
			return executorPayload{}, err
		}
		return executorPayload{body: body, url: buildResponsesURL(endpoint), stream: req.Stream}, nil
	case domain.ProtocolOperationClaudeMessages:
		req.Protocol = domain.ProtocolClaude
		body, err := EncodeClaudeChatRequest(req)
		if err != nil {
			return executorPayload{}, err
		}
		return executorPayload{body: body, url: buildClaudeMessagesURL(endpoint), stream: req.Stream}, nil
	case domain.ProtocolOperationGeminiGenerateContent, domain.ProtocolOperationGeminiStreamGenerate:
		req.Protocol = domain.ProtocolGemini
		body, err := EncodeGeminiChatRequest(req)
		if err != nil {
			return executorPayload{}, err
		}
		url := buildGeminiGenerateContentURL(endpoint, upstreamModel)
		stream := req.Stream || operation == domain.ProtocolOperationGeminiStreamGenerate
		if stream {
			url = buildGeminiStreamGenerateContentURL(endpoint, upstreamModel)
		}
		return executorPayload{body: body, url: url, stream: stream}, nil
	default:
		req.Protocol = protocol
		if req.Protocol == domain.ProtocolCodex {
			req.Protocol = domain.ProtocolOpenAI
		}
		body, err := EncodeOpenAIChatRequest(req)
		if err != nil {
			return executorPayload{}, err
		}
		return executorPayload{body: body, url: buildChatCompletionsURL(endpoint), stream: req.Stream}, nil
	}
}

func sanitizeRequestMetadataForTarget(metadata map[string]json.RawMessage, sourceProtocol domain.Protocol, targetProtocol domain.Protocol, targetProvider string) map[string]json.RawMessage {
	if len(metadata) == 0 {
		return metadata
	}
	if sourceProtocol == "" {
		sourceProtocol = targetProtocol
	}
	cleaned := cloneRawMap(metadata)
	deleteKeys := func(keys ...string) {
		for _, key := range keys {
			delete(cleaned, key)
		}
	}

	switch {
	case sourceProtocol == domain.ProtocolOpenAI && targetProvider == "gemini":
		rewriteOpenAIReasoningToGemini(cleaned)
		rewriteOpenAIStructuredOutputsToGemini(cleaned)
		rewriteOpenAIToolChoiceToGemini(cleaned)
		rewriteOpenAIControlsToGemini(cleaned)
		deleteKeys("response_format", "text")
	case sourceProtocol == domain.ProtocolOpenAI && targetProvider == "claude":
		rewriteOpenAIReasoningToClaude(cleaned)
	case sourceProtocol == domain.ProtocolGemini && targetProvider == "openai":
		rewriteGeminiThinkingToOpenAI(cleaned)
		rewriteGeminiStructuredOutputsToOpenAI(cleaned)
		deleteKeys("generationConfig")
	case sourceProtocol == domain.ProtocolGemini && targetProvider == "claude":
		rewriteGeminiThinkingToClaude(cleaned)
	case sourceProtocol == domain.ProtocolClaude && targetProvider == "openai":
		rewriteClaudeThinkingToOpenAI(cleaned)
		rewriteClaudeToolChoiceToOpenAI(cleaned)
	case sourceProtocol == domain.ProtocolClaude && targetProvider == "gemini":
		rewriteClaudeThinkingToGemini(cleaned)
	}

	switch sourceProtocol {
	case domain.ProtocolOpenAI:
		deleteKeys("previous_response_id", "include", "reasoning", "store", "instructions", "metadata", "generate")
	case domain.ProtocolGemini:
		deleteKeys("previous_response_id", "include", "reasoning", "store", "instructions", "tool_choice", "parallel_tool_calls", "thinking", "cache_control")
	case domain.ProtocolClaude:
		deleteKeys("previous_response_id", "include", "store", "parallel_tool_calls")
	}
	_ = targetProtocol
	return cleaned
}

func sanitizeRequestForTarget(metadata map[string]json.RawMessage, tools []json.RawMessage, sourceProtocol domain.Protocol, targetProtocol domain.Protocol, targetProvider string) (map[string]json.RawMessage, []json.RawMessage) {
	metadata = sanitizeRequestMetadataForTarget(metadata, sourceProtocol, targetProtocol, targetProvider)
	if len(tools) == 0 {
		if sourceProtocol == domain.ProtocolClaude && targetProvider == "openai" {
			tools = rewriteClaudeToolsToOpenAI(tools)
			metadata, tools = rewriteClaudeContainerToOpenAI(metadata, tools)
			metadata, tools = rewriteClaudeMCPToOpenAITools(metadata, tools)
		}
		return metadata, tools
	}

	switch {
	case sourceProtocol == domain.ProtocolOpenAI && targetProvider == "gemini":
		tools = rewriteOpenAIToolsToGemini(tools)
	case sourceProtocol == domain.ProtocolGemini && targetProvider == "openai":
		tools = rewriteGeminiToolsToOpenAI(tools)
	case sourceProtocol == domain.ProtocolOpenAI && targetProvider == "claude":
		metadata, tools = rewriteOpenAICodeInterpreterToClaude(metadata, tools)
		metadata, tools = rewriteOpenAIMCPToolsToClaude(metadata, tools)
	case sourceProtocol == domain.ProtocolClaude && targetProvider == "openai":
		tools = rewriteClaudeToolsToOpenAI(tools)
		metadata, tools = rewriteClaudeContainerToOpenAI(metadata, tools)
		metadata, tools = rewriteClaudeMCPToOpenAITools(metadata, tools)
	}
	return metadata, tools
}

func sanitizeMessagesForTarget(messages []domain.UnifiedMessage, sourceProtocol domain.Protocol, targetProtocol domain.Protocol, targetProvider string) []domain.UnifiedMessage {
	if len(messages) == 0 {
		return messages
	}
	if sourceProtocol == "" {
		sourceProtocol = targetProtocol
	}
	if sourceProtocol == domain.ProtocolOpenAI && (targetProvider == "gemini" || (targetProvider == "openai" && targetProtocol == domain.ProtocolOpenAI)) {
		filtered := make([]domain.UnifiedMessage, 0, len(messages))
		for _, message := range messages {
			cleaned, keep := sanitizeOpenAIMessagesForGemini(message)
			if keep {
				filtered = append(filtered, cleaned)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}
	return messages
}

func sanitizeOpenAIMessagesForGemini(message domain.UnifiedMessage) (domain.UnifiedMessage, bool) {
	cleaned := domain.UnifiedMessage{
		Role:      message.Role,
		ToolCalls: message.ToolCalls,
		InputItem: message.InputItem,
		Metadata:  cloneRawMap(message.Metadata),
	}
	cleaned.Parts = make([]domain.UnifiedPart, 0, len(message.Parts))
	for _, part := range message.Parts {
		if shouldDropOpenAITextPartForGemini(part) {
			continue
		}
		cleaned.Parts = append(cleaned.Parts, cloneUnifiedPart(part))
	}
	if len(cleaned.Parts) == 0 && len(cleaned.ToolCalls) == 0 && len(cleaned.InputItem) == 0 {
		return domain.UnifiedMessage{}, false
	}
	return cleaned, true
}

func shouldDropOpenAITextPartForGemini(part domain.UnifiedPart) bool {
	if !strings.EqualFold(strings.TrimSpace(part.Type), "text") {
		return false
	}
	text := strings.TrimSpace(part.Text)
	if text == "" {
		return true
	}
	lowered := strings.ToLower(text)
	if lowered == "request interrupted by user" || lowered == "request cancelled by user" {
		return true
	}
	if strings.Contains(lowered, "<system-reminder>") {
		return true
	}
	return false
}

func cloneUnifiedPart(part domain.UnifiedPart) domain.UnifiedPart {
	cloned := part
	cloned.Metadata = cloneRawMap(part.Metadata)
	if len(part.InputItem) > 0 {
		cloned.InputItem = append(json.RawMessage(nil), part.InputItem...)
	}
	if len(part.OutputItem) > 0 {
		cloned.OutputItem = append(json.RawMessage(nil), part.OutputItem...)
	}
	if len(part.NativePayload) > 0 {
		cloned.NativePayload = append(json.RawMessage(nil), part.NativePayload...)
	}
	return cloned
}

func rewriteOpenAIStructuredOutputsToGemini(metadata map[string]json.RawMessage) {
	schema, jsonOnly := extractOpenAIStructuredOutput(metadata)
	if !jsonOnly && schema == nil {
		return
	}

	config := decodeJSONObject(metadata["generationConfig"])
	config["responseMimeType"] = "application/json"
	if schema != nil {
		config["responseSchema"] = schema
	}
	if encoded, err := json.Marshal(config); err == nil {
		metadata["generationConfig"] = encoded
	}
}

func rewriteGeminiStructuredOutputsToOpenAI(metadata map[string]json.RawMessage) {
	config := decodeJSONObject(metadata["generationConfig"])
	if len(config) == 0 {
		return
	}

	schema, hasSchema := config["responseSchema"]
	if !hasSchema {
		schema, hasSchema = config["responseJsonSchema"]
	}
	mimeType, _ := config["responseMimeType"].(string)
	if !hasSchema && strings.TrimSpace(strings.ToLower(mimeType)) != "application/json" {
		return
	}

	payload := map[string]any{}
	if hasSchema {
		payload["type"] = "json_schema"
		payload["json_schema"] = map[string]any{
			"name":   "structured_output",
			"schema": schema,
		}
	} else {
		payload["type"] = "json_object"
	}
	if encoded, err := json.Marshal(payload); err == nil {
		metadata["response_format"] = encoded
	}
}

func rewriteOpenAIReasoningToGemini(metadata map[string]json.RawMessage) {
	config := decodeJSONObject(metadata["generationConfig"])
	reasoning := decodeJSONObject(metadata["reasoning"])
	if len(reasoning) == 0 {
		return
	}
	thinkingConfig := decodeJSONObject(mustJSONRaw(config["thinkingConfig"]))
	if effort, _ := reasoning["effort"].(string); strings.TrimSpace(effort) != "" {
		thinkingConfig["thinkingBudget"] = openAIEffortToGeminiThinkingBudget(effort)
	}
	if summary, _ := reasoning["summary"].(string); strings.TrimSpace(summary) != "" {
		thinkingConfig["includeThoughts"] = true
	}
	if len(thinkingConfig) == 0 {
		return
	}
	config["thinkingConfig"] = thinkingConfig
	if encoded, err := json.Marshal(config); err == nil {
		metadata["generationConfig"] = encoded
	}
	delete(metadata, "reasoning")
}

func rewriteOpenAIControlsToGemini(metadata map[string]json.RawMessage) {
	if len(metadata) == 0 {
		return
	}
	config := decodeJSONObject(metadata["generationConfig"])
	if len(metadata["temperature"]) > 0 {
		var temperature float64
		if err := json.Unmarshal(metadata["temperature"], &temperature); err == nil {
			config["temperature"] = temperature
		}
		delete(metadata, "temperature")
	}
	delete(metadata, "max_tokens")
	delete(config, "maxOutputTokens")
	if len(config) == 0 {
		delete(metadata, "generationConfig")
		return
	}
	if encoded, err := json.Marshal(config); err == nil {
		metadata["generationConfig"] = encoded
	}
}

func rewriteGeminiThinkingToOpenAI(metadata map[string]json.RawMessage) {
	config := decodeJSONObject(metadata["generationConfig"])
	if len(config) == 0 {
		return
	}
	thinkingConfig := decodeJSONObject(mustJSONRaw(config["thinkingConfig"]))
	if len(thinkingConfig) == 0 {
		return
	}
	payload := map[string]any{}
	if budget, ok := asInt(thinkingConfig["thinkingBudget"]); ok {
		payload["effort"] = geminiThinkingBudgetToOpenAIEffort(budget)
	}
	if level, _ := thinkingConfig["thinkingLevel"].(string); strings.TrimSpace(level) != "" {
		payload["effort"] = geminiThinkingLevelToOpenAIEffort(level)
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); ok && includeThoughts {
		payload["summary"] = "auto"
	}
	if len(payload) == 0 {
		return
	}
	if encoded, err := json.Marshal(payload); err == nil {
		metadata["reasoning"] = encoded
	}
}

func rewriteOpenAIReasoningToClaude(metadata map[string]json.RawMessage) {
	reasoning := decodeJSONObject(metadata["reasoning"])
	if len(reasoning) == 0 {
		return
	}
	effort, _ := reasoning["effort"].(string)
	effort = strings.TrimSpace(strings.ToLower(effort))
	if effort == "" || effort == "none" {
		delete(metadata, "reasoning")
		return
	}
	payload := map[string]any{"type": "enabled", "budget_tokens": maxInt(1024, openAIEffortToClaudeBudget(effort))}
	if encoded, err := json.Marshal(payload); err == nil {
		metadata["thinking"] = encoded
	}
	ensureClaudeMaxTokensForThinking(metadata, payload["budget_tokens"])
	delete(metadata, "reasoning")
}

func rewriteClaudeThinkingToOpenAI(metadata map[string]json.RawMessage) {
	thinking := decodeJSONObject(metadata["thinking"])
	if len(thinking) == 0 {
		return
	}
	payload := map[string]any{}
	typeValue, _ := thinking["type"].(string)
	typeValue = strings.TrimSpace(strings.ToLower(typeValue))
	switch typeValue {
	case "adaptive":
		payload["effort"] = "medium"
	case "enabled":
		if budget, ok := asInt(thinking["budget_tokens"]); ok {
			payload["effort"] = claudeBudgetToOpenAIEffort(budget)
		} else {
			payload["effort"] = "medium"
		}
	default:
		payload["effort"] = "medium"
	}
	payload["summary"] = "auto"
	if encoded, err := json.Marshal(payload); err == nil {
		metadata["reasoning"] = encoded
	}
	delete(metadata, "thinking")
}

func rewriteClaudeToolChoiceToOpenAI(metadata map[string]json.RawMessage) {
	toolChoice := decodeJSONObject(metadata["tool_choice"])
	if len(toolChoice) == 0 {
		return
	}

	typeValue, _ := toolChoice["type"].(string)
	typeValue = strings.TrimSpace(strings.ToLower(typeValue))
	var rewritten any
	switch typeValue {
	case "", "auto":
		rewritten = "auto"
	case "any":
		rewritten = "required"
	case "tool":
		name := coalesceString(toolChoice["name"], toolChoice["tool_name"])
		if strings.TrimSpace(name) == "" {
			rewritten = "required"
		} else {
			rewritten = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": name,
				},
			}
		}
	case "none":
		rewritten = "none"
	default:
		return
	}
	if encoded, err := json.Marshal(rewritten); err == nil {
		metadata["tool_choice"] = encoded
	}
	if disabled, ok := toolChoice["disable_parallel_tool_use"].(bool); ok && disabled {
		if encoded, err := json.Marshal(false); err == nil {
			metadata["parallel_tool_calls"] = encoded
		}
	}
}

func rewriteGeminiThinkingToClaude(metadata map[string]json.RawMessage) {
	config := decodeJSONObject(metadata["generationConfig"])
	if len(config) == 0 {
		return
	}
	thinkingConfig := decodeJSONObject(mustJSONRaw(config["thinkingConfig"]))
	if len(thinkingConfig) == 0 {
		return
	}
	payload := map[string]any{}
	if budget, ok := asInt(thinkingConfig["thinkingBudget"]); ok {
		if budget <= 0 {
			delete(metadata, "generationConfig")
			return
		}
		payload["type"] = "enabled"
		payload["budget_tokens"] = maxInt(1024, budget)
	} else {
		payload["type"] = "adaptive"
	}
	if encoded, err := json.Marshal(payload); err == nil {
		metadata["thinking"] = encoded
	}
	ensureClaudeMaxTokensForThinking(metadata, payload["budget_tokens"])
}

func rewriteClaudeThinkingToGemini(metadata map[string]json.RawMessage) {
	thinking := decodeJSONObject(metadata["thinking"])
	if len(thinking) == 0 {
		return
	}
	config := decodeJSONObject(metadata["generationConfig"])
	thinkingConfig := decodeJSONObject(mustJSONRaw(config["thinkingConfig"]))
	typeValue, _ := thinking["type"].(string)
	typeValue = strings.TrimSpace(strings.ToLower(typeValue))
	switch typeValue {
	case "enabled":
		if budget, ok := asInt(thinking["budget_tokens"]); ok {
			thinkingConfig["thinkingBudget"] = budget
		}
		thinkingConfig["includeThoughts"] = true
	case "adaptive":
		thinkingConfig["includeThoughts"] = true
	}
	if len(thinkingConfig) == 0 {
		return
	}
	config["thinkingConfig"] = thinkingConfig
	if encoded, err := json.Marshal(config); err == nil {
		metadata["generationConfig"] = encoded
	}
	delete(metadata, "thinking")
}

func rewriteOpenAICodeInterpreterToClaude(metadata map[string]json.RawMessage, tools []json.RawMessage) (map[string]json.RawMessage, []json.RawMessage) {
	if len(tools) == 0 {
		return metadata, tools
	}
	rewritten := make([]json.RawMessage, 0, len(tools))
	containerAssigned := false
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		typeValue, _ := tool["type"].(string)
		if strings.TrimSpace(strings.ToLower(typeValue)) != "code_interpreter" {
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
			continue
		}
		if !containerAssigned {
			if container := tool["container"]; container != nil {
				if encoded, err := json.Marshal(container); err == nil {
					if metadata == nil {
						metadata = map[string]json.RawMessage{}
					}
					metadata["container"] = encoded
				}
			}
			containerAssigned = true
		}
		encoded, _ := json.Marshal(map[string]any{
			"type": "code_execution_20250825",
			"name": "code_execution",
		})
		rewritten = append(rewritten, encoded)
	}
	return metadata, rewritten
}

func rewriteClaudeContainerToOpenAI(metadata map[string]json.RawMessage, tools []json.RawMessage) (map[string]json.RawMessage, []json.RawMessage) {
	if len(metadata["container"]) == 0 {
		return metadata, tools
	}
	container := decodeJSONObject(metadata["container"])
	if len(container) == 0 {
		var containerID string
		if err := json.Unmarshal(metadata["container"], &containerID); err == nil && strings.TrimSpace(containerID) != "" {
			container = map[string]any{"id": containerID}
		}
	}
	if len(container) == 0 {
		return metadata, tools
	}
	rewritten := append([]json.RawMessage(nil), tools...)
	hasInterpreter := false
	for index, raw := range rewritten {
		tool := decodeJSONObject(raw)
		typeValue, _ := tool["type"].(string)
		if strings.TrimSpace(strings.ToLower(typeValue)) != "code_interpreter" {
			continue
		}
		tool["container"] = container
		if encoded, err := json.Marshal(tool); err == nil {
			rewritten[index] = encoded
		}
		hasInterpreter = true
	}
	if !hasInterpreter {
		encoded, _ := json.Marshal(map[string]any{
			"type":      "code_interpreter",
			"container": container,
		})
		rewritten = append(rewritten, encoded)
	}
	delete(metadata, "container")
	return metadata, rewritten
}

func rewriteOpenAIToolsToGemini(tools []json.RawMessage) []json.RawMessage {
	if len(tools) == 0 {
		return tools
	}
	rewritten := make([]json.RawMessage, 0, len(tools))
	functionDeclarations := make([]map[string]any, 0)
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		typeValue, _ := tool["type"].(string)
		switch strings.TrimSpace(strings.ToLower(typeValue)) {
		case "code_interpreter":
			encoded, _ := json.Marshal(map[string]any{"codeExecution": map[string]any{}})
			rewritten = append(rewritten, encoded)
		case "function":
			functionPayload, _ := tool["function"].(map[string]any)
			name, _ := functionPayload["name"].(string)
			name = sanitizeGeminiFunctionName(name)
			if strings.TrimSpace(name) == "" {
				rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
				continue
			}
			declaration := map[string]any{
				"name":        name,
				"description": coalesceString(functionPayload["description"]),
			}
			if parameters := functionPayload["parameters"]; parameters != nil {
				declaration["parameters"] = normalizeSchemaTypesForGemini(parameters)
			}
			functionDeclarations = append(functionDeclarations, declaration)
		default:
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
		}
	}
	if len(functionDeclarations) > 0 {
		encoded, _ := json.Marshal(map[string]any{"functionDeclarations": functionDeclarations})
		rewritten = append(rewritten, encoded)
	}
	return rewritten
}

func rewriteGeminiToolsToOpenAI(tools []json.RawMessage) []json.RawMessage {
	if len(tools) == 0 {
		return tools
	}
	rewritten := make([]json.RawMessage, 0, len(tools))
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		switch {
		case tool["codeExecution"] != nil || tool["code_execution"] != nil:
			encoded, _ := json.Marshal(map[string]any{"type": "code_interpreter", "container": map[string]any{"type": "auto"}})
			rewritten = append(rewritten, encoded)
		case tool["functionDeclarations"] != nil:
			if declarations, ok := tool["functionDeclarations"].([]any); ok {
				for _, item := range declarations {
					declaration, ok := item.(map[string]any)
					if !ok {
						continue
					}
					name, _ := declaration["name"].(string)
					if strings.TrimSpace(name) == "" {
						continue
					}
					openAITool := map[string]any{
						"type": "function",
						"function": map[string]any{
							"name":        name,
							"description": coalesceString(declaration["description"]),
							"parameters":  declaration["parameters"],
						},
					}
					if encoded, err := json.Marshal(openAITool); err == nil {
						rewritten = append(rewritten, encoded)
					}
				}
			}
		default:
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
		}
	}
	return rewritten
}

func rewriteOpenAIToolChoiceToGemini(metadata map[string]json.RawMessage) {
	if len(metadata) == 0 {
		return
	}
	if raw := strings.TrimSpace(decodeStringRaw(metadata["tool_choice"])); raw != "" {
		mode := "AUTO"
		switch strings.ToLower(raw) {
		case "required":
			mode = "ANY"
		case "none":
			mode = "NONE"
		}
		setGeminiFunctionCallingConfig(metadata, mode, nil)
		delete(metadata, "tool_choice")
		return
	}
	toolChoice := decodeJSONObject(metadata["tool_choice"])
	if len(toolChoice) == 0 {
		return
	}
	typeValue, _ := toolChoice["type"].(string)
	typeValue = strings.TrimSpace(strings.ToLower(typeValue))
	mode := "AUTO"
	var allowed []string
	switch typeValue {
	case "required", "any":
		mode = "ANY"
	case "none":
		mode = "NONE"
	case "function", "tool":
		mode = "ANY"
		if functionPayload, ok := toolChoice["function"].(map[string]any); ok {
			if name, _ := functionPayload["name"].(string); strings.TrimSpace(name) != "" {
				allowed = []string{name}
			}
		}
	}
	setGeminiFunctionCallingConfig(metadata, mode, allowed)
	delete(metadata, "tool_choice")
}

func setGeminiFunctionCallingConfig(metadata map[string]json.RawMessage, mode string, allowed []string) {
	config := decodeJSONObject(metadata["toolConfig"])
	functionCallingConfig := decodeJSONObject(mustJSONRaw(config["functionCallingConfig"]))
	functionCallingConfig["mode"] = mode
	if len(allowed) > 0 {
		functionCallingConfig["allowedFunctionNames"] = allowed
	} else {
		delete(functionCallingConfig, "allowedFunctionNames")
	}
	config["functionCallingConfig"] = functionCallingConfig
	if encoded, err := json.Marshal(config); err == nil {
		metadata["toolConfig"] = encoded
	}
}

func normalizeSchemaTypesForGemini(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		if flattened, ok := flattenGeminiCombinatorSchema(typed); ok {
			return normalizeSchemaTypesForGemini(flattened)
		}
		normalized := make(map[string]any, len(typed))
		for key, nested := range typed {
			if shouldDropGeminiSchemaKeyword(key) {
				continue
			}
			if key == "properties" {
				properties, ok := nested.(map[string]any)
				if !ok {
					continue
				}
				normalizedProperties := make(map[string]any, len(properties))
				for propertyName, propertyValue := range properties {
					normalizedProperties[propertyName] = normalizeSchemaTypesForGemini(propertyValue)
				}
				normalized[key] = normalizedProperties
				continue
			}
			if key == "type" {
				switch asType := nested.(type) {
				case string:
					normalized[key] = strings.ToUpper(strings.TrimSpace(asType))
					continue
				case []any:
					for _, candidate := range asType {
						if asString, ok := candidate.(string); ok && strings.TrimSpace(asString) != "" && !strings.EqualFold(strings.TrimSpace(asString), "null") {
							normalized[key] = strings.ToUpper(strings.TrimSpace(asString))
							break
						}
					}
					if _, ok := normalized[key]; ok {
						continue
					}
				}
			}
			normalized[key] = normalizeSchemaTypesForGemini(nested)
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, nested := range typed {
			normalized = append(normalized, normalizeSchemaTypesForGemini(nested))
		}
		return normalized
	default:
		return value
	}
}

func flattenGeminiCombinatorSchema(schema map[string]any) (map[string]any, bool) {
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		rawAlternatives, ok := schema[key].([]any)
		if !ok || len(rawAlternatives) == 0 {
			continue
		}
		flattened := make(map[string]any, len(schema))
		for existingKey, existingValue := range schema {
			if existingKey == key {
				continue
			}
			flattened[existingKey] = existingValue
		}
		for _, alternative := range rawAlternatives {
			alternativeMap, ok := alternative.(map[string]any)
			if !ok {
				continue
			}
			for nestedKey, nestedValue := range alternativeMap {
				if _, exists := flattened[nestedKey]; !exists {
					flattened[nestedKey] = nestedValue
				}
			}
			if _, exists := alternativeMap["type"]; exists {
				break
			}
		}
		return flattened, true
	}
	return nil, false
}

func shouldDropGeminiSchemaKeyword(key string) bool {
	switch key {
	case "$schema", "$id", "$defs", "definitions", "$ref", "const", "additionalProperties", "propertyNames", "patternProperties", "deprecated", "enumTitles", "prefill", "title", "nullable", "default", "examples", "example", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "minLength", "maxLength", "minItems", "maxItems", "uniqueItems", "format", "pattern", "propertyOrdering", "readOnly", "writeOnly", "contentMediaType", "contentEncoding", "unevaluatedProperties", "unevaluatedItems":
		return true
	default:
		return false
	}
}

var geminiFunctionNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.:-]`)

func sanitizeGeminiFunctionName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = geminiFunctionNameSanitizer.ReplaceAllString(name, "_")
	if name == "" {
		return ""
	}
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		name = "_" + name
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func rewriteClaudeToolsToOpenAI(tools []json.RawMessage) []json.RawMessage {
	if len(tools) == 0 {
		return tools
	}
	rewritten := make([]json.RawMessage, 0, len(tools))
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		if len(tool) == 0 {
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
			continue
		}
		typeValue, _ := tool["type"].(string)
		if strings.TrimSpace(strings.ToLower(typeValue)) != "" {
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
			continue
		}
		name, _ := tool["name"].(string)
		if strings.TrimSpace(name) == "" || tool["input_schema"] == nil {
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
			continue
		}
		functionTool := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":       name,
				"description": coalesceString(tool["description"]),
				"parameters": tool["input_schema"],
			},
		}
		if encoded, err := json.Marshal(functionTool); err == nil {
			rewritten = append(rewritten, encoded)
			continue
		}
		rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
	}
	return rewritten
}

func rewriteOpenAIMCPToolsToClaude(metadata map[string]json.RawMessage, tools []json.RawMessage) (map[string]json.RawMessage, []json.RawMessage) {
	if len(tools) == 0 {
		return metadata, tools
	}
	remaining := make([]json.RawMessage, 0, len(tools))
	servers := make([]map[string]any, 0)
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		typeValue, _ := tool["type"].(string)
		if strings.TrimSpace(strings.ToLower(typeValue)) != "mcp" {
			remaining = append(remaining, append(json.RawMessage(nil), raw...))
			continue
		}
		serverURL, _ := tool["server_url"].(string)
		if strings.TrimSpace(serverURL) == "" {
			remaining = append(remaining, append(json.RawMessage(nil), raw...))
			continue
		}
		server := map[string]any{
			"type": "url",
			"url":  serverURL,
			"name": coalesceString(tool["server_label"], tool["name"], "mcp-server"),
		}
		if auth, _ := tool["authorization"].(string); strings.TrimSpace(auth) != "" {
			server["authorization_token"] = auth
		}
		if allowedTools := tool["allowed_tools"]; allowedTools != nil {
			server["tool_configuration"] = map[string]any{"enabled": true, "allowed_tools": allowedTools}
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return metadata, remaining
	}
	encoded, err := json.Marshal(servers)
	if err == nil {
		if metadata == nil {
			metadata = map[string]json.RawMessage{}
		}
		metadata["mcp_servers"] = encoded
	}
	return metadata, remaining
}

func rewriteClaudeMCPToOpenAITools(metadata map[string]json.RawMessage, tools []json.RawMessage) (map[string]json.RawMessage, []json.RawMessage) {
	if len(metadata["mcp_servers"]) == 0 {
		return metadata, tools
	}
	var servers []map[string]any
	if err := json.Unmarshal(metadata["mcp_servers"], &servers); err != nil {
		return metadata, tools
	}
	rewritten := append([]json.RawMessage(nil), tools...)
	added := 0
	for _, server := range servers {
		serverURL, _ := server["url"].(string)
		if strings.TrimSpace(serverURL) == "" {
			continue
		}
		tool := map[string]any{
			"type":         "mcp",
			"server_label": coalesceString(server["name"], server["server_name"], "mcp-server"),
			"server_url":   serverURL,
		}
		if auth, _ := server["authorization_token"].(string); strings.TrimSpace(auth) != "" {
			tool["authorization"] = auth
		}
		if config, ok := server["tool_configuration"].(map[string]any); ok {
			if allowed := config["allowed_tools"]; allowed != nil {
				tool["allowed_tools"] = allowed
			}
		}
		if encoded, err := json.Marshal(tool); err == nil {
			rewritten = append(rewritten, encoded)
			added++
		}
	}
	if added > 0 {
		delete(metadata, "mcp_servers")
	}
	return metadata, rewritten
}

func extractOpenAIStructuredOutput(metadata map[string]json.RawMessage) (any, bool) {
	if schema, jsonOnly, ok := parseOpenAIStructuredOutputRaw(metadata["response_format"]); ok {
		return schema, jsonOnly
	}
	if len(metadata["text"]) > 0 {
		textConfig := decodeRawJSONObject(metadata["text"])
		if formatValue, ok := textConfig["format"]; ok {
			if formatRaw, err := json.Marshal(formatValue); err == nil {
				if schema, jsonOnly, ok := parseOpenAIStructuredOutputRaw(formatRaw); ok {
					return schema, jsonOnly
				}
			}
		}
	}
	return nil, false
}

func parseOpenAIStructuredOutputRaw(raw json.RawMessage) (any, bool, bool) {
	if len(raw) == 0 {
		return nil, false, false
	}
	payload := decodeRawJSONObject(raw)
	typeValue, _ := payload["type"].(string)
	switch strings.TrimSpace(strings.ToLower(typeValue)) {
	case "json_object":
		return nil, true, true
	case "json_schema":
		jsonSchema, _ := payload["json_schema"].(map[string]any)
		if jsonSchema == nil {
			return nil, false, false
		}
		if schema, ok := jsonSchema["schema"]; ok {
			return schema, true, true
		}
		return nil, true, true
	default:
		return nil, false, false
	}
}

func decodeJSONObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	return decodeRawJSONObject(raw)
}

func decodeRawJSONObject(raw json.RawMessage) map[string]any {
	payload := map[string]any{}
	_ = json.Unmarshal(raw, &payload)
	return payload
}

func cloneRawMessages(src []json.RawMessage) []json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	dst := make([]json.RawMessage, 0, len(src))
	for _, item := range src {
		dst = append(dst, append(json.RawMessage(nil), item...))
	}
	return dst
}

func mustJSONRaw(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	body, _ := json.Marshal(value)
	return body
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}

func openAIEffortToGeminiThinkingBudget(effort string) int {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "none":
		return 0
	case "minimal":
		return 1024
	case "low":
		return 2048
	case "medium":
		return 4096
	case "high":
		return 8192
	case "xhigh":
		return 16384
	default:
		return 4096
	}
}

func openAIEffortToClaudeBudget(effort string) int {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "minimal":
		return 1024
	case "low":
		return 2048
	case "medium":
		return 4096
	case "high":
		return 8192
	case "xhigh":
		return 16384
	default:
		return 4096
	}
}

func geminiThinkingBudgetToOpenAIEffort(budget int) string {
	switch {
	case budget <= 0:
		return "none"
	case budget <= 1024:
		return "minimal"
	case budget <= 2048:
		return "low"
	case budget <= 4096:
		return "medium"
	case budget <= 8192:
		return "high"
	default:
		return "xhigh"
	}
}

func geminiThinkingLevelToOpenAIEffort(level string) string {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case "none":
		return "none"
	case "minimal":
		return "minimal"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	default:
		return "medium"
	}
}

func claudeBudgetToOpenAIEffort(budget int) string {
	switch {
	case budget <= 1024:
		return "minimal"
	case budget <= 2048:
		return "low"
	case budget <= 4096:
		return "medium"
	case budget <= 8192:
		return "high"
	default:
		return "xhigh"
	}
}

func ensureClaudeMaxTokensForThinking(metadata map[string]json.RawMessage, budget any) {
	thinkingBudget, ok := asInt(budget)
	if !ok || thinkingBudget <= 0 {
		return
	}
	current := 0
	if len(metadata["max_tokens"]) > 0 {
		var raw int
		if err := json.Unmarshal(metadata["max_tokens"], &raw); err == nil {
			current = raw
		}
	}
	required := maxInt(thinkingBudget+1024, 2048)
	if current >= required {
		return
	}
	if encoded, err := json.Marshal(required); err == nil {
		metadata["max_tokens"] = encoded
	}
}

func coalesceString(values ...any) string {
	for _, value := range values {
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return str
		}
	}
	return ""
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func mergeAnthropicBetaHeader(current string, metadata map[string]json.RawMessage) string {
	current = strings.TrimSpace(current)
	required := make([]string, 0, 3)
	if len(metadata["mcp_servers"]) > 0 {
		required = append(required, "mcp-client-2025-11-20")
	}
	if len(metadata["context_management"]) > 0 {
		required = append(required, "context-management-2025-06-27")
	}
	if len(metadata["container"]) > 0 {
		required = append(required, "code-execution-2025-08-25")
	}
	if len(required) == 0 {
		return current
	}
	existing := map[string]struct{}{}
	parts := make([]string, 0)
	for _, item := range strings.Split(current, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := existing[item]; ok {
			continue
		}
		existing[item] = struct{}{}
		parts = append(parts, item)
	}
	for _, item := range required {
		if _, ok := existing[item]; ok {
			continue
		}
		existing[item] = struct{}{}
		parts = append(parts, item)
	}
	return strings.Join(parts, ",")
}

func doExecutorRequest(client *http.Client, req *http.Request, stream bool, debugMetadata map[string]string) (*domain.ExecutionResult, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.NewExecutionError(attachPayloadDebugMetadata(fmt.Errorf("请求上游失败: %w", err), debugMetadata), 0, true, false)
	}
	headers := cloneHeaders(resp.Header)
	if stream && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: resp.StatusCode, Headers: headers, Body: resp.Body}}, nil
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, domain.NewExecutionError(attachPayloadDebugMetadata(fmt.Errorf("读取上游响应失败: %w", readErr), debugMetadata), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := string(body)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, domain.NewExecutionError(attachPayloadDebugMetadata(fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message), debugMetadata), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}
	return &domain.ExecutionResult{Response: &domain.ProxyResponse{StatusCode: resp.StatusCode, Headers: headers, Body: body}}, nil
}

func cloneHeaders(header http.Header) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func payloadDebugMetadata(provider string, operation domain.ProtocolOperation, payload executorPayload) map[string]string {
	if len(payload.body) == 0 {
		return nil
	}
	digest := sha256.Sum256(payload.body)
	preview := buildFocusedPayloadPreview(payload.body)
	return map[string]string{
		"upstream_provider":        provider,
		"upstream_operation":       string(operation),
		"upstream_request_url":     payload.url,
		"upstream_request_stream":  fmt.Sprintf("%t", payload.stream),
		"upstream_payload_sha256":  hex.EncodeToString(digest[:]),
		"upstream_payload_preview": preview,
	}
}

func buildFocusedPayloadPreview(body []byte) string {
	preview := string(body)
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		if len(preview) > 4000 {
			return preview[:4000]
		}
		return preview
	}
	focused := map[string]any{}
	for _, key := range []string{"toolConfig", "tools", "generationConfig", "system_instruction"} {
		if value, ok := raw[key]; ok {
			focused[key] = value
		}
	}
	if focusedContentsCount, ok := raw["contents"].([]any); ok {
		focused["contents_count"] = len(focusedContentsCount)
		if extraKeys := collectGeminiContentExtraKeys(focusedContentsCount); len(extraKeys) > 0 {
			focused["contents_extra_keys"] = extraKeys
		}
	}
	if len(focused) == 0 {
		if len(preview) > 4000 {
			return preview[:4000]
		}
		return preview
	}
	encoded, err := json.Marshal(focused)
	if err != nil {
		if len(preview) > 4000 {
			return preview[:4000]
		}
		return preview
	}
	result := string(encoded)
	if len(result) > 4000 {
		return result[:4000]
	}
	return result
}

func collectGeminiContentExtraKeys(contents []any) map[string][]string {
	extraByIndex := map[string][]string{}
	for idx, item := range contents {
		content, ok := item.(map[string]any)
		if !ok {
			continue
		}
		extra := make([]string, 0)
		for key := range content {
			if key == "role" || key == "parts" {
				continue
			}
			extra = append(extra, key)
		}
		if len(extra) == 0 {
			continue
		}
		sort.Strings(extra)
		extraByIndex[fmt.Sprintf("contents[%d]", idx)] = extra
	}
	if len(extraByIndex) == 0 {
		return nil
	}
	return extraByIndex
}

func attachPayloadDebugMetadata(err error, metadata map[string]string) error {
	if err == nil || len(metadata) == 0 {
		return err
	}
	parts := []string{err.Error()}
	for _, key := range []string{"upstream_provider", "upstream_operation", "upstream_request_url", "upstream_request_stream", "upstream_payload_sha256", "upstream_payload_preview"} {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return errors.New(strings.Join(parts, " | "))
}

func applyRequestHeaders(req *http.Request, headers map[string]string, skip map[string]struct{}) {
	for key, value := range headers {
		if _, blocked := skip[strings.ToLower(strings.TrimSpace(key))]; blocked {
			continue
		}
		req.Header.Set(key, value)
	}
}

func shouldUseNativeUpstreamStream(req domain.GatewayRequest, targetProvider string, targetOperation domain.ProtocolOperation) bool {
	if !req.Stream {
		return false
	}
	if targetOperation == domain.ProtocolOperationOpenAIResponses {
		return false
	}
	if req.Protocol == "" {
		switch targetProvider {
		case "claude":
			return targetOperation == domain.ProtocolOperationClaudeMessages
		case "gemini":
			return targetOperation == domain.ProtocolOperationGeminiStreamGenerate
		case "openai":
			return true
		default:
			return false
		}
	}
	sourceOperation := inferGatewaySourceOperation(req)
	switch req.Protocol {
	case domain.ProtocolClaude:
		return targetProvider == "claude" && targetOperation == domain.ProtocolOperationClaudeMessages
	case domain.ProtocolGemini:
		return targetProvider == "gemini" && targetOperation == sourceOperation
	case domain.ProtocolCodex:
		return targetProvider == "openai" && targetOperation == domain.ProtocolOperationOpenAIResponses
	case domain.ProtocolOpenAI:
		return targetProvider == "openai" && targetOperation == sourceOperation
	default:
		return false
	}
}

func inferGatewaySourceOperation(req domain.GatewayRequest) domain.ProtocolOperation {
	if req.Operation != "" {
		return req.Operation
	}
	switch req.Protocol {
	case domain.ProtocolClaude:
		return domain.ProtocolOperationClaudeMessages
	case domain.ProtocolGemini:
		if req.Stream {
			return domain.ProtocolOperationGeminiStreamGenerate
		}
		return domain.ProtocolOperationGeminiGenerateContent
	case domain.ProtocolCodex:
		return domain.ProtocolOperationCodexResponses
	default:
		return domain.ProtocolOperationOpenAIChatCompletions
	}
}

func markResponsesBridgeRepair(req domain.UnifiedChatRequest, sourceProtocol domain.Protocol, targetOperation domain.ProtocolOperation) domain.UnifiedChatRequest {
	if targetOperation != domain.ProtocolOperationOpenAIResponses {
		return req
	}
	if sourceProtocol == "" || sourceProtocol == domain.ProtocolOpenAI {
		return req
	}
	return req
}
