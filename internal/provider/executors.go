package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opencrab/internal/domain"
)

const anthropicVersion = "2023-06-01"

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
	payload, err := buildExecutorPayload(
		domain.ProtocolOpenAI,
		plannedOperationForExecutor(input, normalizedProvider),
		toUnifiedRequest(input, domain.ProtocolOpenAI, normalizedProvider),
		input.Request.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 OpenAI 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, payload.url, bytes.NewReader(payload.body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 OpenAI 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"authorization": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, payload.stream)
}

func (e *ClaudeExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	unifiedReq := toUnifiedRequest(input, domain.ProtocolClaude, domain.NormalizeProvider(input.Channel.Provider))
	payload, err := buildExecutorPayload(
		domain.ProtocolClaude,
		domain.ProtocolOperationClaudeMessages,
		unifiedReq,
		input.Request.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Claude 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
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
	return doExecutorRequest(e.client, req, payload.stream)
}

func (e *GeminiExecutor) Execute(ctx context.Context, input domain.ExecutorRequest) (*domain.ExecutionResult, error) {
	targetOperation := domain.ProtocolOperationGeminiGenerateContent
	if input.Request.Stream {
		targetOperation = domain.ProtocolOperationGeminiStreamGenerate
	}
	payload, err := buildExecutorPayload(
		domain.ProtocolGemini,
		targetOperation,
		toUnifiedRequest(input, domain.ProtocolGemini, domain.NormalizeProvider(input.Channel.Provider)),
		input.Request.Session,
		input.Channel.Endpoint,
		input.UpstreamModel,
	)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("构造 Gemini 请求失败: %w", err), 0, false, false)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, payload.url, bytes.NewReader(payload.body))
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("创建 Gemini 请求失败: %w", err), 0, false, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("x-goog-api-key", input.Channel.APIKey)
	applyRequestHeaders(req, input.Request.RequestHeaders, map[string]struct{}{"x-goog-api-key": {}, "content-type": {}, "accept": {}})
	return doExecutorRequest(e.client, req, payload.stream)
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
		metadata, tools = rewriteClaudeContainerToOpenAI(metadata, tools)
		metadata, tools = rewriteClaudeMCPToOpenAITools(metadata, tools)
	}
	return metadata, tools
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
	for _, raw := range tools {
		tool := decodeJSONObject(raw)
		typeValue, _ := tool["type"].(string)
		switch strings.TrimSpace(strings.ToLower(typeValue)) {
		case "code_interpreter":
			encoded, _ := json.Marshal(map[string]any{"codeExecution": map[string]any{}})
			rewritten = append(rewritten, encoded)
		default:
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
		}
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
		default:
			rewritten = append(rewritten, append(json.RawMessage(nil), raw...))
		}
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

func doExecutorRequest(client *http.Client, req *http.Request, stream bool) (*domain.ExecutionResult, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("请求上游失败: %w", err), 0, true, false)
	}
	headers := cloneHeaders(resp.Header)
	if stream && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &domain.ExecutionResult{Stream: &domain.StreamResult{StatusCode: resp.StatusCode, Headers: headers, Body: resp.Body}}, nil
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, domain.NewExecutionError(fmt.Errorf("读取上游响应失败: %w", readErr), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := string(body)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, domain.NewExecutionError(fmt.Errorf("上游返回 %d: %s", resp.StatusCode, message), resp.StatusCode, domain.IsRetryableStatusCode(resp.StatusCode), false)
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

func applyRequestHeaders(req *http.Request, headers map[string]string, skip map[string]struct{}) {
	for key, value := range headers {
		if _, blocked := skip[strings.ToLower(strings.TrimSpace(key))]; blocked {
			continue
		}
		req.Header.Set(key, value)
	}
}
