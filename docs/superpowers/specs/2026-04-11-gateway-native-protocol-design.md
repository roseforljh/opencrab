# 2026-04-11 网关多协议原生接口改造设计

## 1. 背景

当前 OpenCrab 的真实代理链路只接入了 OpenAI compatible 转发。

已确认现状：

1. 对外真实入口只有 `POST /v1/chat/completions`。
2. 运行时只实例化了 `OpenAICompatibleProvider`。
3. Claude 原生与 Gemini 原生目前只用于后台渠道测试，不参与真实请求转发。
4. `model_routes.priority` 和 `fallback_model` 还没有进入真实运行时路由。
5. 真实运行时仍是“取第一个启用的 channel 直接转发”。

本次改造目标是参考 CLIProxyAPI 的核心思路，把网关升级成多协议入站、多 provider 执行、统一内部抽象、真实模型路由的结构。

## 2. 目标

本次改造采用方案 1。

目标如下：

1. 保留现有 OpenAI compatible 入站能力。
2. 新增 Claude 原生入站接口。
3. 新增 Gemini 原生入站接口。
4. 引入统一内部请求与响应模型，作为多协议之间的中台抽象。
5. 引入 runtime router，真实使用 `model_routes`、`channels`、`priority`、`fallback_model`。
6. 引入 provider executor 层，把 OpenAI compatible、Claude、Gemini 三类上游协议拆开执行。
7. 出站按原始入站协议回写响应与错误。
8. 统一日志与可观测字段。

## 3. 第一版范围

### 3.1 第一版要做的内容

1. OpenAI compatible 入站。
2. Claude 原生入站。
3. Gemini 原生入站。
4. 统一内部请求与响应模型。
5. runtime router 真实命中 `model_routes`。
6. OpenAI compatible executor。
7. Claude executor。
8. Gemini executor。
9. 按原始协议回写普通响应。
10. 基础流式响应支持。
11. 路由日志、错误日志、usage 记录。
12. fallback 基础能力。
13. provider 能力约束校验。

### 3.2 第一版明确不做的内容

1. 多模态完整支持。
2. 跨 provider 的复杂工具调用转换。
3. 权重负载均衡。
4. 自动健康探测与摘除。
5. 熔断器。
6. provider capability 自动协商。
7. 复杂 reasoning 字段统一抽象。
8. 流式过程中跨 route fallback。
9. 已产生正文 token 后的 fallback。

第一版只保证 text-only 主链路稳定可用。工具调用默认仅保留结构和协议内透传约束，不承诺跨 provider 执行链路。图片等多模态字段先不打通。

## 4. 运行时架构

### 4.1 入站协议层

新增三类 handler：

1. OpenAI compatible handler
2. Claude native handler
3. Gemini native handler

这些 handler 只负责：

1. 解析各自协议请求。
2. 转换成统一内部请求结构。
3. 调用统一 gateway service。
4. 使用对应 encoder 把统一响应转回原始协议。

handler 不直接处理上游 provider 协议，不直接做 channel 选择。

### 4.2 统一内部模型层

新增统一请求与响应结构：

1. `UnifiedChatRequest`
2. `UnifiedChatResponse`
3. `UnifiedStreamEvent`

这层是所有协议适配、路由、executor、日志复用的中台对象。

### 4.3 路由层

新增 runtime router，职责如下：

1. 根据统一请求中的目标模型查找 `model_routes`。
2. 在路由前执行 provider 能力约束校验，至少校验文本、stream、tools、system、`max_tokens` 等能力是否满足。
3. 按 `priority ASC` 排序候选路由。
4. 过滤未启用 channel。
5. 根据 channel.provider 选择对应 executor。
6. 在可重试错误且未产生正文 token 时执行 fallback。
7. 输出最终命中结果与路由日志元信息。

当前 `GetFirstEnabledChannel()` 将退出真实主链路。

### 4.4 Provider Executor 层

新增统一 executor 接口，并提供三类实现：

1. `OpenAICompatibleExecutor`
2. `ClaudeExecutor`
3. `GeminiExecutor`

每个 executor 负责：

1. 把统一内部请求翻译为上游原生协议。
2. 发送上游请求。
3. 把上游普通响应或流式事件转换为统一内部响应。
4. 返回 usage、header、provider 元信息。

### 4.5 出站协议层

按原始入站协议回写：

1. OpenAI compatible response encoder
2. Claude native response encoder
3. Gemini native response encoder

目标是做到：

1. OpenAI 进，OpenAI 出。
2. Claude 进，Claude 出。
3. Gemini 进，Gemini 出。

## 5. 统一内部数据模型

### 5.1 `UnifiedChatRequest`

建议包含以下字段：

- `Protocol`
- `Model`
- `Messages`
- `System`
- `Tools`
- `ToolChoice`
- `Stream`
- `Temperature`
- `TopP`
- `MaxTokens`
- `Stop`
- `Metadata`

其中以下字段在第一版中视为统一主链路必填语义：

1. `Protocol`
2. `Model`
3. `Messages`
4. `Stream`

其余字段允许为空。跨协议转换时必须遵守两条规则：

1. 不能稳定映射的字段，必须显式落入 `Metadata` 或直接返回明确错误，禁止静默丢失。
2. 若字段会影响上游能力选择或输出语义，必须在路由前完成校验，禁止带着不兼容字段进入 executor。

`Metadata` 用于承接第一版不强抽象的 provider 特有字段。

### 5.2 `UnifiedMessage`

建议统一为 role + parts 结构：

- `Role`
- `Parts`

`Role` 取值至少支持：

- `system`
- `user`
- `assistant`
- `tool`

### 5.3 `UnifiedPart`

建议支持以下类型：

- `text`
- `image`
- `tool_call`
- `tool_result`

字段包括：

- `Type`
- `Text`
- `MimeType`
- `Data`
- `ToolName`
- `ToolCallID`
- `Arguments`
- `Result`

第一版真实主链路只要求 `text` 稳定可用，其他类型先保留结构。

### 5.4 协议到统一模型的映射

#### OpenAI -> Unified

- `model` -> `Model`
- `stream` -> `Stream`
- `messages[].role` -> `Role`
- `messages[].content` -> `Parts`
- `tools` -> `Tools`
- `tool_choice` -> `ToolChoice`

#### Claude -> Unified

- `model` -> `Model`
- 顶层 `system` -> `System`
- `messages[].role` -> `Role`
- `messages[].content[]` -> `Parts`
- `tools` -> `Tools`
- `tool_choice` -> `ToolChoice`
- `max_tokens` -> `MaxTokens`

#### Gemini -> Unified

- 路径或请求中的 model -> `Model`
- `contents[]` -> `Messages`
- `parts[]` -> `Parts`
- `system_instruction` -> `System`
- `tools` -> `Tools`
- `generationConfig.maxOutputTokens` -> `MaxTokens`
- `generationConfig.temperature` -> `Temperature`
- `generationConfig.topP` -> `TopP`

### 5.5 `UnifiedChatResponse`

建议包含：

- `Model`
- `Message`
- `FinishReason`
- `ToolCalls`
- `Usage`
- `Raw`

`Usage` 至少包含：

- `PromptTokens`
- `CompletionTokens`
- `TotalTokens`
- `CacheHit`

### 5.6 `UnifiedStreamEvent`

建议流式内部事件包含：

- `Type`
- `Delta`
- `Usage`
- `Raw`

`Type` 至少支持：

- `message_start`
- `content_delta`
- `tool_call_delta`
- `message_end`
- `error`

## 6. 路由命中与 fallback 规则

### 6.1 命中规则

统一请求进入 runtime router 后，按以下规则执行：

1. 抽取 `requestedModel`。
2. 以 `requestedModel` 查找 `model_routes.model_alias`。
3. 命中结果按 `priority ASC` 排序。
4. 关联 `channels`，只保留 `enabled = 1` 的 channel。
5. 根据 channel.provider 选择 executor。
6. 依次尝试候选 route，直到成功或全部失败。

### 6.2 provider 与 executor 映射

第一版建议：

1. OpenAI / OpenRouter / GLM / Kimi / MiniMax 归到 `OpenAICompatibleExecutor`
2. Claude / Anthropic 归到 `ClaudeExecutor`
3. Gemini / Google 归到 `GeminiExecutor`

### 6.3 fallback 规则

第一版采用最小可用策略：

1. 优先尝试 priority 最小的 route。
2. 当主 route 失败时，按 priority 顺序尝试下一个 route。
3. `fallback_model` 第一版不进入真实执行链路，只保留字段与后续扩展位。
4. 全部失败后返回最后一个错误。

### 6.4 fallback 幂等边界

1. 仅在未向客户端写出任何正文 token 或正文 chunk 时允许 fallback。
2. stream 模式下，一旦发出首个正文增量事件，禁止切换 route。
3. 普通响应模式下，一旦确认上游已经开始返回可见正文，禁止切换 route。
4. usage 与日志必须按 attempt 独立记录，再汇总到 request 级别。

### 6.5 允许 fallback 的错误

只对以下情况执行 fallback：

1. 网络错误
2. 上游超时
3. 429
4. 5xx

其中 429 需要区分 provider 级限流与账号级配额耗尽。只有仍存在其他可用 route 时才允许继续尝试。

### 6.6 不允许 fallback 的错误

以下情况直接返回：

1. 入站请求体不合法
2. 模型字段缺失
3. 协议转换失败
4. 明确的 4xx 参数错误

## 7. 错误处理

新增统一错误结构 `GatewayError`：

- `Layer`
- `Code`
- `Message`
- `StatusCode`
- `Retryable`
- `Provider`
- `Channel`
- `Cause`

### 7.1 错误分层

错误按四层处理：

1. `ingress`
   - 入站协议解析和校验错误
2. `routing`
   - 路由未命中、channel 不可用、executor 不存在
3. `upstream`
   - 网络、超时、429、5xx、TLS 等执行错误
4. `egress`
   - 出站编码错误、流式事件编码错误

router 依赖 `Retryable` 判断是否 fallback。最终错误由对应协议 encoder 输出成目标协议格式。

## 8. 流式响应策略

第一版不统一外部协议形状，只统一内部流式语义。

策略如下：

1. OpenAI 入站 stream，返回 OpenAI SSE。
2. Claude 入站 stream，返回 Claude 原生流式事件。
3. Gemini 入站 stream，返回 Gemini 原生流式格式。
4. executor 输出统一内部流式事件。
5. encoder 根据原协议把内部事件写回客户端。

### 8.1 流式状态机要求

每条流式请求都必须显式覆盖以下阶段：

1. 首包建立
2. 内容增量
3. 结束事件
4. 错误事件
5. 客户端断连
6. 上游中断

### 8.2 流式边界

1. 先支持文本增量。
2. usage 只允许在尾事件或流结束后落账。
3. tool call streaming 先保留结构，不承诺完整打通。
4. 流式过程中禁止跨 route fallback。
5. 一旦发出首个正文 token 或正文 chunk，禁止切换 route。

## 9. 日志与可观测性

请求日志建议统一记录以下字段：

- request id
- inbound protocol
- requested model
- matched model alias
- selected provider
- selected channel
- upstream endpoint
- fallback attempts
- final upstream model
- final status code
- latency
- usage
- stream flag
- route id
- attempt index
- fallback reason
- retryable
- upstream status
- stream end reason

错误日志必须能区分：

1. 协议解析错误
2. 路由命中错误
3. 上游执行错误
4. 出站编码错误

日志与 usage 记录分两层：

1. attempt 级，记录每次真实上游尝试
2. request 级，记录汇总结果

## 10. 测试策略

### 10.1 协议转换单元测试

验证：

1. OpenAI -> Unified
2. Claude -> Unified
3. Gemini -> Unified
4. Unified -> OpenAI
5. Unified -> Claude
6. Unified -> Gemini

### 10.2 Router 单元测试

验证：

1. 单 route 命中
2. 多 route priority 顺序
3. channel disabled 跳过
4. 第一版不使用 `fallback_model`，相关字段仅作为配置保留且不会参与执行
5. 只有 retryable error 才 fallback

### 10.3 Executor 单元测试

验证：

1. Claude executor URL、header、body 正确
2. Gemini executor URL、header、body 正确
3. OpenAI compatible executor 保持现有行为

### 10.4 Handler 集成测试

验证：

1. OpenAI 入站 -> Claude 上游
2. OpenAI 入站 -> Gemini 上游
3. Claude 入站 -> Claude 上游
4. Gemini 入站 -> Gemini 上游
5. 普通响应回写正确
6. stream 回写正确
7. 错误格式符合各自协议

### 10.5 回归测试

验证以下现有能力不回退：

1. `/v1/chat/completions`
2. API Key 校验
3. rate limit
4. request logs
5. channel test

## 11. 实施顺序

为降低改造风险，建议按以下顺序落地：

1. 引入 `UnifiedChatRequest`、`UnifiedChatResponse`、`UnifiedStreamEvent`
2. 抽 runtime router
3. 把现有 OpenAI compatible 转发接入 executor 接口
4. 落地 Claude executor
5. 落地 Gemini executor
6. 新增 Claude 原生入站 handler
7. 新增 Gemini 原生入站 handler
8. 统一日志与错误编码
9. 补齐 stream 与回归测试

## 12. 风险与第一版控制策略

### 12.1 高风险点

1. 三套协议的消息结构差异较大
2. 流式事件格式差异大
3. 工具调用字段语义不完全一致
4. Gemini model 可能在路径与 body 中同时出现
5. fallback 后的 usage 和日志汇总容易混乱

### 12.2 控制策略

1. 第一版先收敛到 text-only 主链路
2. 工具调用只保留结构，并限定为协议内透传或明确拒绝，跨 provider 一律返回清晰错误
3. 多模态输入先不打通
4. provider 特有字段先落 `Metadata`
5. 日志优先记录统一字段，不追求一次抽象所有 provider 特性

## 13. 与当前文档的关系

现有 `docs/architecture.md` 与 `docs/backend-plan.md` 记录的是 OpenCrab 首版只提供 OpenAI compatible 主链路的架构边界。

本设计文档是下一阶段的网关升级方案，目标是把首版已有的 channels、models、model_routes、request_logs 等基础能力真正拉通到多协议运行时网关中。
