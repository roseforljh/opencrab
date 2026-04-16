# Gateway Native Protocol Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 OpenCrab 从仅支持 OpenAI compatible 真实转发，升级为支持 OpenAI compatible、Claude native、Gemini native 入站，并通过统一内部抽象、真实路由和 provider executor 完成多协议网关主链路。

**Architecture:** 保留现有 `chi` 路由、SQLite、管理 API 和日志体系，在 `internal/domain` 定义统一请求响应模型，在 `internal/provider` 拆出协议转换与 executor，在 `internal/usecase` 增加 runtime router 与 gateway service，在 `internal/transport/httpserver` 新增三类入站 handler 与对应 encoder。第一版仅保证 text-only 主链路、基础流式、同一 requested model 下的 route fallback，不启用 `fallback_model` 真实执行。

**Tech Stack:** Go, net/http, chi, SQLite, 现有 request log 与 rate limiter 体系

---

## File Structure

### Existing files to modify
- `internal/domain/proxy.go`
  - 现有 `UpstreamChannel`、`ProxyResponse`、`ChatProvider` 定义过于单薄，需要升级为统一网关模型与 executor 接口的核心定义。
- `internal/provider/openai_compatible.go`
  - 从单一 provider 实现改造成 `OpenAICompatibleExecutor` 的一部分，保留 URL/header/body 组装能力。
- `internal/provider/channel_tester.go`
  - 复用现有 Claude/Gemini 原生请求组装逻辑，抽出给 executor 共用的 URL/header 构造能力。
- `internal/store/sqlite/admin_store.go`
  - 新增按 model alias 读取 route 候选、按 channel name 读取 channel、读取 enabled route 集合等运行时查询。
- `internal/transport/httpserver/router.go`
  - 增加多协议入站路由与统一 gateway 依赖注入，保留现有管理 API。
- `internal/app/app.go`
  - 从“第一个 enabled channel + OpenAICompatibleProvider”切换为 gateway service + runtime router + executor registry。
- `internal/transport/httpserver/proxy_test.go`
  - 扩充 handler 层集成测试，覆盖 OpenAI/Claude/Gemini 入站。

### New files to create
- `internal/domain/gateway.go`
  - 定义 `UnifiedChatRequest`、`UnifiedChatResponse`、`UnifiedStreamEvent`、`GatewayError`、`GatewayAttemptLog`、executor/router 接口。
- `internal/usecase/gateway_service.go`
  - 统一编排入口。接收统一请求，调用 runtime router，执行 fallback，产出统一响应与 attempt 日志。
- `internal/usecase/runtime_router.go`
  - 路由候选查询、provider 能力校验、executor 选择。
- `internal/usecase/runtime_router_store_test.go`
  - route 候选查询与 channel 运行时查询测试，避免与 router/service 混在同一测试文件。
- `internal/usecase/gateway_service_test.go`
  - gateway service 的 fallback、attempt 日志与 request 汇总测试。
- `internal/provider/openai_codec.go`
  - OpenAI compatible 入站 decode 与出站 encode。
- `internal/provider/claude_codec.go`
  - Claude native 入站 decode 与出站 encode。
- `internal/provider/gemini_codec.go`
  - Gemini native 入站 decode 与出站 encode。
- `internal/provider/openai_executor.go`
  - OpenAI compatible executor 具体执行。
- `internal/provider/claude_executor.go`
  - Claude native executor。
- `internal/provider/gemini_executor.go`
  - Gemini native executor。
- `internal/provider/executor_test.go`
  - 三类 executor 的请求构造与错误处理测试。
- `internal/provider/codec_test.go`
  - 三类 codec 与统一模型互转测试。
- `internal/transport/httpserver/gateway_handlers.go`
  - 三类入站 handler，避免继续膨胀 `router.go`。
- `internal/store/sqlite/request_logs_runtime.go`
  - request 级与 attempt 级日志写入辅助，承接新增日志字段。
- `internal/store/sqlite/request_logs_runtime_test.go`
  - 校验 attempt 级与 request 级日志落库行为。
- `docs/architecture.md`
  - 实现完成后同步“首版只提供 OpenAI compatible”旧表述。
- `docs/backend-plan.md`
  - 实现完成后同步阶段边界变化。

## Chunk 1: 统一模型与协议编解码

### Task 1: 定义统一网关领域模型

**Files:**
- Create: `internal/domain/gateway.go`
- Modify: `internal/domain/proxy.go`
- Test: `internal/provider/codec_test.go`

- [ ] **Step 1: 写统一模型相关失败测试骨架**

```go
func TestUnifiedChatRequestRequiresCoreFields(t *testing.T) {
    req := domain.UnifiedChatRequest{}
    if err := req.ValidateCore(); err == nil {
        t.Fatal("expected validation error")
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run TestUnifiedChatRequestRequiresCoreFields`
Expected: FAIL with undefined `UnifiedChatRequest` or `ValidateCore`

- [ ] **Step 3: 在 `internal/domain/gateway.go` 定义最小可用模型**

```go
type Protocol string
const (
    ProtocolOpenAI Protocol = "openai"
    ProtocolClaude Protocol = "claude"
    ProtocolGemini Protocol = "gemini"
)

type UnifiedChatRequest struct {
    Protocol Protocol
    Model string
    Messages []UnifiedMessage
    System []string
    Tools []UnifiedTool
    ToolChoice string
    Stream bool
    Temperature *float64
    TopP *float64
    MaxTokens *int
    Stop []string
    Metadata map[string]any
}
```

同时补：`UnifiedMessage`、`UnifiedPart`、`UnifiedChatResponse`、`UnifiedStreamEvent`、`GatewayError`、`GatewayAttemptLog`、`ValidateCore()`。

- [ ] **Step 4: 收敛旧 `proxy.go`**

把 `ProxyResponse` 迁移或复用到新网关模型中，补 `Executor` / `Codec` / `RuntimeRouter` 等接口定义，避免 domain 分散。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/provider -run TestUnifiedChatRequestRequiresCoreFields`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/domain/proxy.go internal/domain/gateway.go internal/provider/codec_test.go
git commit -m "feat: add unified gateway domain models"
```

### Task 2: 实现 OpenAI / Claude / Gemini codec

**Files:**
- Create: `internal/provider/openai_codec.go`
- Create: `internal/provider/claude_codec.go`
- Create: `internal/provider/gemini_codec.go`
- Test: `internal/provider/codec_test.go`

- [ ] **Step 1: 为三类入站 decode 写失败测试**

```go
func TestDecodeOpenAIRequest(t *testing.T) {}
func TestDecodeClaudeRequest(t *testing.T) {}
func TestDecodeGeminiRequest(t *testing.T) {}
```

覆盖：核心字段映射、`Metadata` 兜底、非法字段报错、不允许静默丢失。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run 'TestDecode(OpenAIRequest|ClaudeRequest|GeminiRequest)'`
Expected: FAIL with missing codec symbols

- [ ] **Step 3: 实现三类 decode**

要求：
- OpenAI `messages[].content` 支持 string 和 array 两种形态
- Claude `system`、`messages[].content[]` 正确映射
- Gemini 支持从路径 model 与 body 读取 model，并对冲突报错
- 无法稳定映射的字段显式进入 `Metadata`

- [ ] **Step 4: 为三类出站 encode 写失败测试**

```go
func TestEncodeOpenAIResponse(t *testing.T) {}
func TestEncodeClaudeResponse(t *testing.T) {}
func TestEncodeGeminiResponse(t *testing.T) {}
```

覆盖：普通响应、错误响应、最小流式事件编码。

- [ ] **Step 5: 运行测试确认失败**

Run: `go test ./internal/provider -run 'TestEncode(OpenAIResponse|ClaudeResponse|GeminiResponse)'`
Expected: FAIL with missing encoder symbols

- [ ] **Step 6: 实现三类 encode**

要求：
- 普通响应按原协议输出
- 错误响应按各协议形状输出
- 流式仅支持 text delta 主链路
- tool 调用相关事件第一版仅协议内透传或明确拒绝

- [ ] **Step 7: 运行 provider codec 测试**

Run: `go test ./internal/provider -run 'Test(Decode|Encode)'`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/provider/openai_codec.go internal/provider/claude_codec.go internal/provider/gemini_codec.go internal/provider/codec_test.go
git commit -m "feat: add gateway protocol codecs"
```

## Chunk 2: 路由层、错误模型与运行时查询

### Task 3: 增加 route 候选查询与 channel 读取能力

**Files:**
- Modify: `internal/store/sqlite/admin_store.go`
- Test: `internal/usecase/runtime_router_store_test.go`

- [ ] **Step 1: 为 route 候选查询写失败测试**

```go
func TestListRouteCandidatesByModelAlias(t *testing.T) {}
```

覆盖：priority 排序、disabled channel 过滤、provider/name/endpoint/api_key 一并返回。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestListRouteCandidatesByModelAlias`
Expected: FAIL with missing store helper

- [ ] **Step 3: 在 `admin_store.go` 增加最小查询函数**

建议新增：
- `ListRouteCandidatesByModelAlias(ctx, db, modelAlias)`
- `GetEnabledChannelByName(ctx, db, channelName)` 如需要

返回结构要包含 runtime router 所需的 provider、channel、priority、fallback_model。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/usecase -run TestListRouteCandidatesByModelAlias`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/sqlite/admin_store.go internal/usecase/runtime_router_test.go
git commit -m "feat: add runtime route candidate queries"
```

### Task 4: 先落统一错误模型与 capability 判定

**Files:**
- Modify: `internal/domain/gateway.go`
- Create: `internal/usecase/runtime_capabilities.go`
- Test: `internal/usecase/runtime_router_test.go`

- [ ] **Step 1: 写 capability 与错误模型失败测试**

```go
func TestGatewayErrorRetryableByLayer(t *testing.T) {}
func TestProviderCapabilitiesRejectUnsupportedTools(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run 'Test(GatewayErrorRetryableByLayer|ProviderCapabilitiesRejectUnsupportedTools)'`
Expected: FAIL with missing capability helpers or gateway error behavior

- [ ] **Step 3: 实现最小能力矩阵与错误判断**

要求：
- 明确 text、stream、tools、system、max_tokens 的 provider 支持判断
- 明确 `GatewayError` 的 retryable 判定
- 不让 router 在能力和错误语义未稳定前直接编码逻辑

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/usecase -run 'Test(GatewayErrorRetryableByLayer|ProviderCapabilitiesRejectUnsupportedTools)'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/gateway.go internal/usecase/runtime_capabilities.go internal/usecase/runtime_router_test.go
git commit -m "feat: add gateway capability matrix"
```

### Task 5: 实现 runtime router

**Files:**
- Create: `internal/usecase/runtime_router.go`
- Test: `internal/usecase/runtime_router_test.go`

- [ ] **Step 1: 写 router 行为失败测试**

```go
func TestRuntimeRouterSelectsLowestPriorityCandidate(t *testing.T) {}
func TestRuntimeRouterRejectsUnsupportedCapabilities(t *testing.T) {}
func TestRuntimeRouterIgnoresFallbackModelInV1(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestRuntimeRouter`
Expected: FAIL with missing router

- [ ] **Step 3: 实现最小 router**

要求：
- 根据 `Model` 查 route candidates
- 校验 provider 能力矩阵，至少处理 text、stream、tools、system、max_tokens
- 选择最低 priority 候选
- 第一版明确不把 `fallback_model` 放进真实执行链路

- [ ] **Step 4: 补 retryable fallback 决策测试**

```go
func TestRuntimeRouterAllowsRetryableFallback(t *testing.T) {}
func TestRuntimeRouterBlocksFallbackAfterOutputStarted(t *testing.T) {}
```

- [ ] **Step 5: 实现 fallback 前置判断**

要求：
- 只有 retryable error 才尝试下一候选
- 已写出正文后禁止 fallback
- stream 过程中禁止跨 route fallback

- [ ] **Step 6: 运行 usecase 测试**

Run: `go test ./internal/usecase`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/usecase/runtime_router.go internal/usecase/runtime_router_test.go
.gitignore
```

实际 commit：
```bash
git add internal/usecase/runtime_router.go internal/usecase/runtime_router_test.go
git commit -m "feat: add runtime router for protocol gateway"
```

## Chunk 3: Provider executor

### Task 5: 拆出 OpenAI compatible executor

**Files:**
- Create: `internal/provider/openai_executor.go`
- Modify: `internal/provider/openai_compatible.go`
- Test: `internal/provider/executor_test.go`

- [ ] **Step 1: 写 OpenAI executor 失败测试**

```go
func TestOpenAIExecutorBuildsChatCompletionsRequest(t *testing.T) {}
func TestOpenAIExecutorReturnsUnifiedResponse(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run TestOpenAIExecutor`
Expected: FAIL with missing executor

- [ ] **Step 3: 实现最小 executor**

要求：
- 封装现有 `/v1/chat/completions` URL 拼接与 `Authorization` header
- 输入为 `UnifiedChatRequest`
- 输出为 `UnifiedChatResponse` 或 `UnifiedStreamEvent`
- usage 尽可能从响应中提取

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/provider -run TestOpenAIExecutor`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/openai_executor.go internal/provider/openai_compatible.go internal/provider/executor_test.go
git commit -m "feat: add openai compatible executor"
```

### Task 6: 实现 Claude executor

**Files:**
- Create: `internal/provider/claude_executor.go`
- Modify: `internal/provider/channel_tester.go`
- Test: `internal/provider/executor_test.go`

- [ ] **Step 1: 写 Claude executor 失败测试**

```go
func TestClaudeExecutorBuildsMessagesRequest(t *testing.T) {}
func TestClaudeExecutorRejectsCrossProviderToolsInV1(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run TestClaudeExecutor`
Expected: FAIL with missing executor

- [ ] **Step 3: 抽复用 URL/header 组装**

从 `channel_tester.go` 抽出：
- Claude messages URL 构造
- `x-api-key`
- `anthropic-version`

避免测试器和 executor 各维护一套。

- [ ] **Step 4: 实现 Claude executor**

要求：
- 请求走 `/v1/messages`
- 正确处理 `system`
- 普通响应转为统一响应
- 流式先支持 text delta
- 非协议内可透传的工具调用返回明确错误

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/provider -run TestClaudeExecutor`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/provider/claude_executor.go internal/provider/channel_tester.go internal/provider/executor_test.go
git commit -m "feat: add claude native executor"
```

### Task 7: 实现 Gemini executor

**Files:**
- Create: `internal/provider/gemini_executor.go`
- Modify: `internal/provider/channel_tester.go`
- Test: `internal/provider/executor_test.go`

- [ ] **Step 1: 写 Gemini executor 失败测试**

```go
func TestGeminiExecutorBuildsGenerateContentRequest(t *testing.T) {}
func TestGeminiExecutorRejectsConflictingModelSources(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/provider -run TestGeminiExecutor`
Expected: FAIL with missing executor

- [ ] **Step 3: 抽复用 Gemini URL/header 组装**

复用：
- `.../models/{model}:generateContent`
- `x-goog-api-key`

- [ ] **Step 4: 实现 Gemini executor**

要求：
- 支持 text-only generateContent 主链路
- 普通响应转为统一响应
- 流式按第一版最小文本增量支持
- 对不支持字段明确报错，不静默丢失

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/provider -run TestGeminiExecutor`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/provider/gemini_executor.go internal/provider/channel_tester.go internal/provider/executor_test.go
git commit -m "feat: add gemini native executor"
```

## Chunk 4: Gateway service 与 HTTP 接线

### Task 8: 实现 gateway service

**Files:**
- Create: `internal/usecase/gateway_service.go`
- Test: `internal/usecase/runtime_router_test.go`

- [ ] **Step 1: 写 gateway service 失败测试**

```go
func TestGatewayServiceRoutesAndExecutes(t *testing.T) {}
func TestGatewayServiceRecordsAttempts(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/usecase -run TestGatewayService`
Expected: FAIL with missing service

- [ ] **Step 3: 实现最小 service**

职责：
- 调 router 取候选
- 调 executor 执行
- 管理 retryable fallback
- 产出 request 级与 attempt 级日志数据

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/usecase -run TestGatewayService`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/gateway_service.go internal/usecase/runtime_router_test.go
git commit -m "feat: add gateway runtime service"
```

### Task 9: 新增多协议 handler

**Files:**
- Create: `internal/transport/httpserver/gateway_handlers.go`
- Modify: `internal/transport/httpserver/router.go`
- Test: `internal/transport/httpserver/gateway_handlers_test.go`

- [ ] **Step 1: 写 OpenAI / Claude / Gemini handler 失败测试**

```go
func TestOpenAIHandlerReturnsProtocolShapedResponse(t *testing.T) {}
func TestClaudeHandlerReturnsProtocolShapedResponse(t *testing.T) {}
func TestGeminiHandlerReturnsProtocolShapedResponse(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run 'Test(OpenAIHandler|ClaudeHandler|GeminiHandler)'`
Expected: FAIL with missing handlers

- [ ] **Step 3: 实现新 handlers**

要求：
- OpenAI 继续承接 `/v1/chat/completions`
- Claude 新增原生 endpoint
- Gemini 新增原生 endpoint
- handler 只做 decode、调用 gateway service、encode
- API key 校验与 rate limit 复用现有逻辑

- [ ] **Step 4: 调整 `router.go` 接线**

要求：
- 保持管理 API 不动
- 把代理主链路从直接 `ProxyChat` 改为新 handler 注册
- 避免继续膨胀单文件，路由定义最小化

- [ ] **Step 5: 运行 handler 测试确认通过**

Run: `go test ./internal/transport/httpserver -run 'Test(OpenAIHandler|ClaudeHandler|GeminiHandler)'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/transport/httpserver/router.go internal/transport/httpserver/gateway_handlers.go internal/transport/httpserver/gateway_handlers_test.go
git commit -m "feat: add native protocol gateway handlers"
```

### Task 10: 在 app 层接入 gateway runtime

**Files:**
- Modify: `internal/app/app.go`
- Test: `internal/transport/httpserver/proxy_test.go`

- [ ] **Step 1: 写 app 接线回归测试或补现有集成测试**

```go
func TestProxyPathUsesGatewayRuntime(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/transport/httpserver -run TestProxyPathUsesGatewayRuntime`
Expected: FAIL with old wiring assumptions

- [ ] **Step 3: 在 `app.go` 替换旧接线**

要求：
- 不再直接实例化 `OpenAICompatibleProvider` 作为唯一真实执行器
- 初始化 executor registry、runtime router、gateway service
- 复用现有 `VerifyAPIKey`、`CreateRequestLog`、`CheckRateLimit`
- 保留 `ChannelTester`

- [ ] **Step 4: 运行 httpserver 集成测试**

Run: `go test ./internal/transport/httpserver`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/transport/httpserver/proxy_test.go
git commit -m "feat: wire gateway runtime into app"
```

## Chunk 5: 全量验证与文档收尾

### Task 11: 补全 provider / usecase / httpserver 测试并回归

**Files:**
- Modify: `internal/provider/executor_test.go`
- Modify: `internal/provider/codec_test.go`
- Modify: `internal/usecase/runtime_router_test.go`
- Modify: `internal/transport/httpserver/gateway_handlers_test.go`
- Modify: `internal/transport/httpserver/proxy_test.go`

- [ ] **Step 1: 补全缺失测试案例**

至少覆盖：
- retryable 与 non-retryable 分支
- stream 首 token 后禁止 fallback
- Gemini model 冲突
- Claude/Gemini 原生错误格式
- attempt 级日志记录

- [ ] **Step 2: 运行后端全量测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: 如失败，最小修复后重跑**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider internal/usecase internal/transport/httpserver
git commit -m "test: cover gateway native protocol flows"
```

### Task 12: 同步文档口径

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/backend-plan.md`
- Reference: `docs/superpowers/specs/2026-04-11-gateway-native-protocol-design.md`

- [ ] **Step 1: 写 docs 变更**

要求：
- 去掉“只提供 OpenAI compatible 主链路”的过时表述
- 补多协议入站、统一内部抽象、runtime router、executor 新口径
- 不额外扩写无关内容

- [ ] **Step 2: 运行最小验证并回读 docs**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add docs/architecture.md docs/backend-plan.md
git commit -m "docs: update gateway architecture for native protocols"
```

## Final Verification Checklist

- [ ] `go test ./internal/provider`
- [ ] `go test ./internal/usecase`
- [ ] `go test ./internal/transport/httpserver`
- [ ] `go test ./...`
- [ ] 回读关键实现文件，确认未残留 `GetFirstEnabledChannel()` 作为真实主链路
- [ ] 回读关键实现文件，确认 Claude/Gemini 原生 handler、executor、codec 已接线
- [ ] 回读关键实现文件，确认日志字段包含 attempt 级信息

## Review Notes for Implementer

1. 第一版不要把 `fallback_model` 做成真实执行逻辑。
2. 第一版不要做跨 provider 工具调用转换。
3. 第一版不要支持多模态完整链路。
4. 任何无法稳定映射的字段都必须显式放进 `Metadata` 或返回错误，禁止静默丢失。
5. 一旦写出正文 token 或正文 chunk，禁止 route fallback。
6. `channel_tester.go` 的 URL/header 组装要尽量复用，避免测试器与 executor 两套逻辑漂移。
