# Native Protocol Streaming Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 OpenCrab 补齐 Claude `/v1/messages`、Gemini `generateContent` 与 `streamGenerateContent` 原生入站接口，并把现有 OpenAI `/v1/chat/completions` 改成真流式。

**Architecture:** 在现有 codec 与 executor 基础上，新增协议边界 handler，把三套入站协议统一接入真实运行时执行链路。运行时执行结果显式区分整包响应与流式响应，流式场景禁止预读 body，并在 handler 层边写边 flush。

**Tech Stack:** Go, net/http, chi, SQLite, httptest

---

## 文件结构

### 需要修改

- `internal/domain/proxy.go`
  - 扩展执行结果结构，新增流式结果模型与必要接口定义。
- `internal/provider/executors.go`
  - 让 OpenAI、Claude、Gemini executor 显式区分流式与非流式结果。
- `internal/provider/openai_compatible.go`
  - 去掉当前假流式整包读取逻辑，补流式 copy/flush 支撑。
- `internal/provider/claude_codec.go`
  - 补 Claude 原生响应与流式事件的编码辅助。
- `internal/provider/gemini_codec.go`
  - 补 Gemini `generateContent` 与 `streamGenerateContent` 的编码辅助。
- `internal/transport/httpserver/gateway_handlers.go`
  - 拆协议 handler，新增 OpenAI、Claude、Gemini 的非流式与流式写回逻辑。
- `internal/transport/httpserver/router.go`
  - 注册 Claude / Gemini 原生入口。
- `internal/app/app.go`
  - 把代理入口从直接 `ProxyChat` 转发切换到统一执行链路。
- `internal/usecase/gateway_service.go`
  - 扩展统一执行层，支持流式结果与流式失败边界。
- `internal/provider/executors_test.go`
  - 增加三类 executor 的流式与非流式测试。
- `internal/transport/httpserver/router_test.go`
  - 增加三套原生入口的 handler 测试。
- `internal/usecase/gateway_service_test.go`
  - 增加流式成功与失败边界测试。

### 可能新增

- `internal/provider/stream_helpers.go`
  - 若 `executors.go` 过大，可拆最小流式辅助函数，但仅在必要时新增。
- `internal/transport/httpserver/gateway_handlers_test.go`
  - 若 router 测试难以承载协议细节，可拆专门 handler 测试文件。

### 参考但不修改或少改

- `internal/provider/openai_codec.go`
- `internal/domain/gateway.go`
- `docs/superpowers/specs/2026-04-14-native-protocol-streaming-design.md`

---

## Chunk 1: 统一执行结果与真实主链路接入

### Task 1: 扩展领域层执行结果模型

**Files:**
- Modify: `internal/domain/proxy.go`
- Test: `internal/usecase/gateway_service_test.go`

- [ ] **Step 1: 写失败测试，约束执行结果可承载流式返回**

在 `internal/usecase/gateway_service_test.go` 增加一个测试，断言 `GatewayService` 能处理“只返回流式结果、不返回整包响应”的 executor 结果。

```go
type fakeStreamReadCloser struct {
	io.Reader
}

func (f fakeStreamReadCloser) Close() error { return nil }

func TestGatewayServiceReturnsStreamResult(t *testing.T) {
	executor := &fakeExecutor{
		result: &domain.ExecutionResult{
			Stream: &domain.StreamResult{
				StatusCode: http.StatusOK,
				Headers:    map[string][]string{"Content-Type": {"text/event-stream"}},
				Body:       fakeStreamReadCloser{Reader: strings.NewReader("data: hi\n\n")},
			},
		},
	}
	// 断言 Execute 返回的结果带有 Stream 且不报错
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestGatewayServiceReturnsStreamResult`
Expected: FAIL，提示 `ExecutionResult` 或 `StreamResult` 不存在。

- [ ] **Step 3: 在 `internal/domain/proxy.go` 增加最小流式结果结构**

补以下最小结构：

```go
type StreamResult struct {
	StatusCode int
	Headers    map[string][]string
	Body       io.ReadCloser
}

type ExecutionResult struct {
	Response *ProxyResponse
	Stream   *StreamResult
}
```

只加最小必要字段，不做额外抽象。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/usecase -run TestGatewayServiceReturnsStreamResult`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/proxy.go internal/usecase/gateway_service_test.go
git commit -m "feat: add stream execution result model"
```

### Task 2: 让 GatewayService 同时支持整包与流式结果

**Files:**
- Modify: `internal/usecase/gateway_service.go`
- Test: `internal/usecase/gateway_service_test.go`

- [ ] **Step 1: 写失败测试，约束 GatewayService 成功返回流式时不再假设 `Response` 一定存在**

```go
func TestGatewayServiceReturnsStreamWithoutTouchingResponseBody(t *testing.T) {
	// executor 返回 StreamResult
	// 断言 Execute 成功，且结果中的 Stream 不为空
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestGatewayServiceReturnsStreamWithoutTouchingResponseBody`
Expected: FAIL，空指针或结果结构不匹配。

- [ ] **Step 3: 修改 `internal/usecase/gateway_service.go` 最小实现**

实现规则：

1. 把 `Execute` 返回类型改成 `(*domain.ExecutionResult, error)`。
2. `result.Response != nil` 时沿用现有整包成功分支。
3. `result.Stream != nil` 时走流式成功分支。
4. 流式成功时也补 `X-Opencrab-Channel` header。
5. 流式成功日志不读取响应 body，只记录 request 摘要与状态码。

- [ ] **Step 4: 运行包测试确认通过**

Run: `go test ./internal/usecase`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/gateway_service.go internal/usecase/gateway_service_test.go
git commit -m "feat: support stream results in gateway service"
```

### Task 3: 先把真实代理主链路切到 GatewayService

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Test: `internal/usecase/gateway_service_test.go`

- [ ] **Step 1: 写失败测试，证明统一执行层返回的流式结果能被主链路消费**

```go
func TestGatewayServiceAddsChannelHeaderForStreamResult(t *testing.T) {
	// 断言 stream 结果也附带 X-Opencrab-Channel
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestGatewayServiceAddsChannelHeaderForStreamResult`
Expected: FAIL

- [ ] **Step 3: 修改 `internal/app/app.go` 与 `internal/transport/httpserver/gateway_handlers.go`**

实现规则：

1. 在 `app.go` 中构造 `executors` 映射与真实 `GatewayService`。
2. 让当前代理主链路不再只走 `GetFirstEnabledChannel -> OpenAICompatibleProvider.ForwardChatCompletions`。
3. handler 共用执行入口改为调用 `GatewayService`。
4. 在这一阶段仍只要求 OpenAI 路径接入统一执行层，Claude / Gemini 路由后续再补。

- [ ] **Step 4: 运行后端相关测试确认通过**

Run: `go test ./internal/usecase ./internal/transport/httpserver`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/transport/httpserver/gateway_handlers.go internal/usecase/gateway_service_test.go
git commit -m "refactor: route gateway requests through gateway service"
```

### Task 4: 让三类 executor 显式区分流式与非流式路径

**Files:**
- Modify: `internal/provider/executors.go`
- Test: `internal/provider/executors_test.go`

- [ ] **Step 1: 为 OpenAI executor 写失败测试，流式时不允许 `ReadAll`**

```go
func TestOpenAIExecutorReturnsStreamResultWhenStreamEnabled(t *testing.T) {
	// 构造一个 transport，若代码尝试 ReadAll 整个 body 则测试无法满足 StreamResult 断言
}
```

- [ ] **Step 2: 为 Claude executor 写失败测试，流式时返回 `StreamResult`**

```go
func TestClaudeExecutorReturnsStreamResultWhenStreamEnabled(t *testing.T) {
	// 断言结果中 Stream 不为空，Response 为空
}
```

- [ ] **Step 3: 为 Gemini executor 写失败测试，流式时命中 `streamGenerateContent` URL**

```go
func TestGeminiExecutorUsesStreamGenerateContentURLWhenStreamEnabled(t *testing.T) {
	// 断言 URL 为 ...:streamGenerateContent
}
```

- [ ] **Step 4: 运行测试确认失败**

Run: `go test ./internal/provider -run "TestOpenAIExecutorReturnsStreamResultWhenStreamEnabled|TestClaudeExecutorReturnsStreamResultWhenStreamEnabled|TestGeminiExecutorUsesStreamGenerateContentURLWhenStreamEnabled"`
Expected: FAIL

- [ ] **Step 5: 修改 `internal/provider/executors.go` 最小实现**

实现规则：

1. 保留非流式 `ReadAll` 路径。
2. 新增流式执行分支，直接返回：

```go
return &domain.ExecutionResult{
	Stream: &domain.StreamResult{
		StatusCode: resp.StatusCode,
		Headers:    cloneHeaders(resp.Header),
		Body:       resp.Body,
	},
}, nil
```

3. Gemini 新增 URL helper：

```go
func buildGeminiStreamGenerateContentURL(endpoint string, model string) string
```

4. `input.Request.Stream` 为 true 时，Gemini 命中 `streamGenerateContent`。

- [ ] **Step 6: 运行 provider 包测试确认通过**

Run: `go test ./internal/provider`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/provider/executors.go internal/provider/executors_test.go
git commit -m "feat: add streaming executor paths"
```

---

## Chunk 2: OpenAI 真流式与统一 handler 基础设施

### Task 4: 拆掉 OpenAI 当前假流式实现

**Files:**
- Modify: `internal/provider/openai_compatible.go`
- Test: `internal/provider/executors_test.go`

- [ ] **Step 1: 写失败测试，证明当前 OpenAI 兼容转发在流式时返回整包**

```go
func TestCopyResponseFlushesStreamChunks(t *testing.T) {
	// 用自定义 ResponseWriter + Flusher，断言流式 copy 时发生 flush
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run TestCopyResponseFlushesStreamChunks`
Expected: FAIL

- [ ] **Step 3: 修改 `internal/provider/openai_compatible.go`**

最小改造：

1. 保留 `CopyResponse` 处理整包结果。
2. 新增：

```go
func CopyStreamResponse(w http.ResponseWriter, stream *domain.StreamResult) error {
	for key, values := range stream.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(stream.StatusCode)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 1024)
	for {
		n, err := stream.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
```

3. `stream.Body.Close()` 在 copy 结束后负责关闭。

- [ ] **Step 4: 运行 provider 包测试确认通过**

Run: `go test ./internal/provider -run TestCopyResponseFlushesStreamChunks`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/openai_compatible.go internal/provider/executors_test.go
git commit -m "feat: add stream copy response helper"
```

### Task 5: 重构 gateway handler 的统一入口

**Files:**
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Test: `internal/transport/httpserver/router_test.go`

- [ ] **Step 1: 为 OpenAI handler 写失败测试，流式时必须走 streaming copy**

```go
func TestOpenAIChatCompletionsStreamsThroughHandler(t *testing.T) {
	// fake deps 返回 StreamResult
	// 断言响应头是 text/event-stream，响应体包含 data chunk
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestOpenAIChatCompletionsStreamsThroughHandler`
Expected: FAIL

- [ ] **Step 3: 修改 `internal/transport/httpserver/gateway_handlers.go`**

改造规则：

1. 把当前 handler 的“鉴权、限流、读 body、调用 proxy、记日志”提炼成共用函数。
2. 共用函数返回 `*domain.ExecutionResult`。
3. `result.Response != nil` 时走 `CopyProxy`。
4. `result.Stream != nil` 时走 `CopyStreamResponse`。
5. 流式场景禁止在写流前调用 `extractUsageMetrics`。

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/router_test.go
git commit -m "refactor: unify gateway execution handling"
```

---

## Chunk 3: Claude `/v1/messages` 原生接口

### Task 6: 补 Claude 原生 handler 的请求解析与非流式返回

**Files:**
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Modify: `internal/transport/httpserver/router.go`
- Test: `internal/transport/httpserver/router_test.go`

- [ ] **Step 1: 写失败测试，`POST /v1/messages` 能命中路由并返回 Claude 格式 JSON**

```go
func TestClaudeMessagesReturnsNativePayload(t *testing.T) {
	// fake deps 返回 Unified/Execution 结果
	// 断言响应包含 Claude 风格字段，如 type / role / content
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestClaudeMessagesReturnsNativePayload`
Expected: FAIL，404 或 handler 不存在。

- [ ] **Step 3: 在 router 和 handler 中补最小实现**

实现步骤：

1. `router.go` 注册 `POST /v1/messages`。
2. `gateway_handlers.go` 新增 `HandleClaudeMessages`。
3. 用 `provider.DecodeClaudeChatRequest` 解析请求。
4. 把统一响应编码成 Claude 风格返回，优先复用现有 codec 辅助。

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver -run TestClaudeMessagesReturnsNativePayload`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/transport/httpserver/router.go internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/router_test.go
git commit -m "feat: add claude messages endpoint"
```

### Task 7: 补 Claude 原生流式写回

**Files:**
- Modify: `internal/provider/claude_codec.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Test: `internal/transport/httpserver/router_test.go`

- [ ] **Step 1: 写失败测试，Claude `stream=true` 时返回 event stream**

```go
func TestClaudeMessagesStreamsNativeEvents(t *testing.T) {
	// fake deps 返回 StreamResult，body 中是 Claude event stream
	// 断言 Content-Type 为 text/event-stream
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestClaudeMessagesStreamsNativeEvents`
Expected: FAIL

- [ ] **Step 3: 修改最小实现**

1. `HandleClaudeMessages` 识别 `stream=true`。
2. 直接走 `CopyStreamResponse`。
3. 如需补头，保证 `Content-Type: text/event-stream` 原样透传。
4. 在 `claude_codec.go` 只补最小必要的非流式编码辅助，不在本轮手搓复杂 event 重组。

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver -run TestClaudeMessagesStreamsNativeEvents`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/claude_codec.go internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/router_test.go
git commit -m "feat: add claude native streaming"
```

---

## Chunk 4: Gemini `generateContent` / `streamGenerateContent`

### Task 8: 补 Gemini `generateContent` 非流式入口

**Files:**
- Modify: `internal/transport/httpserver/router.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Test: `internal/transport/httpserver/router_test.go`

- [ ] **Step 1: 写失败测试，Gemini `generateContent` 能命中并返回 Gemini JSON**

```go
func TestGeminiGenerateContentReturnsNativePayload(t *testing.T) {
	// 断言 candidates 等字段存在
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestGeminiGenerateContentReturnsNativePayload`
Expected: FAIL

- [ ] **Step 3: 修改最小实现**

1. 注册 `POST /v1beta/models/{model}:generateContent`。
2. 用 `DecodeGeminiChatRequest` 解析请求。
3. 非流式响应 encode 成 Gemini `candidates` 结构。

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver -run TestGeminiGenerateContentReturnsNativePayload`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/transport/httpserver/router.go internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/router_test.go
git commit -m "feat: add gemini generate content endpoint"
```

### Task 9: 补 Gemini `streamGenerateContent` 流式入口

**Files:**
- Modify: `internal/provider/gemini_codec.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Modify: `internal/transport/httpserver/router.go`
- Test: `internal/transport/httpserver/router_test.go`

- [ ] **Step 1: 写失败测试，Gemini `streamGenerateContent` 返回流式 chunk**

```go
func TestGeminiStreamGenerateContentStreamsChunks(t *testing.T) {
	// fake deps 返回 StreamResult
	// 断言响应体能逐段读取 Gemini chunk
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestGeminiStreamGenerateContentStreamsChunks`
Expected: FAIL

- [ ] **Step 3: 修改最小实现**

1. 注册 `POST /v1beta/models/{model}:streamGenerateContent`。
2. handler 识别流式 Gemini 请求。
3. 流式场景直接 copy 上游 chunk，不预读 body。
4. 在 `gemini_codec.go` 仅补最小必要的非流式/路径模型辅助，不新增跨协议流式重编码。

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver -run TestGeminiStreamGenerateContentStreamsChunks`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/gemini_codec.go internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/router.go internal/transport/httpserver/router_test.go
git commit -m "feat: add gemini native streaming endpoint"
```

---

## Chunk 5: 接入真实代理主链路并完成验证

### Task 10: 清理重复接入步骤并补端到端流式集成测试

**Files:**
- Modify: `internal/transport/httpserver/router_test.go`
- Possibly create: `internal/transport/httpserver/gateway_handlers_test.go`

- [ ] **Step 1: 写 OpenAI 端到端流式测试**

```go
func TestOpenAIChatCompletionsStreamsEndToEnd(t *testing.T) {
	// httptest.Server 模拟上游 SSE
	// 断言客户端能读到多段 data chunk
}
```

- [ ] **Step 2: 写 Claude 端到端流式测试**

```go
func TestClaudeMessagesStreamsEndToEnd(t *testing.T) {
	// 模拟 message_start / content_block_delta / message_stop
}
```

- [ ] **Step 3: 写 Gemini 端到端流式测试**

```go
func TestGeminiStreamGenerateContentStreamsEndToEnd(t *testing.T) {
	// 模拟 Gemini stream chunk
}
```

- [ ] **Step 4: 运行 transport 包测试确认通过**

Run: `go test ./internal/transport/httpserver`
Expected: PASS

- [ ] **Step 5: 运行后端全量测试确认通过**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/transport/httpserver/router_test.go internal/transport/httpserver/gateway_handlers_test.go
git commit -m "test: cover native protocol streaming end to end"
```

---

## Chunk 6: 最终验证与收口

### Task 11: 做最终协议验收与结果核对

**Files:**
- Modify: `internal/transport/httpserver/router_test.go`
- Modify: `internal/provider/executors_test.go`

- [ ] **Step 1: 为三套入口补最终验收断言**

验收断言至少覆盖：

1. OpenAI `/v1/chat/completions` 非流式可用。
2. OpenAI `/v1/chat/completions` 真流式可用。
3. Claude `/v1/messages` 非流式可用。
4. Claude `/v1/messages` 真流式可用。
5. Gemini `generateContent` 可用。
6. Gemini `streamGenerateContent` 真流式可用。

- [ ] **Step 2: 运行关键测试集**

Run: `go test ./internal/provider ./internal/usecase ./internal/transport/httpserver`
Expected: PASS

- [ ] **Step 3: 运行全量测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: 回读关键文件确认实现已落地**

检查：

1. `internal/transport/httpserver/router.go`
2. `internal/transport/httpserver/gateway_handlers.go`
3. `internal/provider/executors.go`
4. `internal/app/app.go`

Expected: 能看到三套原生路由、流式 copy、统一执行链路接入。

- [ ] **Step 5: Commit**

```bash
git add internal/provider/executors_test.go internal/transport/httpserver/router_test.go
git commit -m "test: finalize native protocol streaming coverage"
```
