# 2026-04-14 Claude 与 Gemini 原生接口及真流式改造设计

## 1. 背景

当前 OpenCrab 已具备三类上游执行能力，但对外只完整暴露了 OpenAI 风格入口。

已确认现状：

1. 对外真实入口只有 `POST /v1/chat/completions`，位置在 `internal/transport/httpserver/router.go`。
2. Claude codec 与 Gemini codec 已存在，但主要停留在协议转换能力，尚未形成对外原生入站闭环。
3. OpenAI、Claude、Gemini 三类 executor 已存在，说明上游执行抓手已具备。
4. OpenAI 兼容入口当前并非真流式，因为代理层会先 `io.ReadAll(resp.Body)` 再整包返回。
5. Claude 原生 `/v1/messages` 与 Gemini 原生 `generateContent` / `streamGenerateContent` 还没有对外路由入口。

本次改造目标很明确：把网关从“仅 OpenAI 外壳”升级成“OpenAI、Claude、Gemini 三套原生入站协议 + 真流式 + 统一执行闭环”。

## 2. 目标

本次采用统一运行时方案，验收口径为完整 C：

1. 保留现有 OpenAI 兼容入站接口。
2. 新增 Claude 原生入站接口 `POST /v1/messages`。
3. 新增 Gemini 原生入站接口：
   - `POST /v1beta/models/{model}:generateContent`
   - `POST /v1beta/models/{model}:streamGenerateContent`
4. OpenAI 兼容接口补齐真流式能力。
5. Claude 原生接口支持非流式与真流式。
6. Gemini 原生接口支持非流式与真流式。
7. 三套入站协议统一走鉴权、限流、路由、上游执行、错误映射、日志闭环。
8. 不扩大业务范围，不顺手重做产品策略，只完成协议层与执行层打通。

## 3. 范围

### 3.1 本次必须完成

1. OpenAI `/v1/chat/completions` 真流式。
2. Claude `/v1/messages` 非流式。
3. Claude `/v1/messages` 流式。
4. Gemini `generateContent` 非流式。
5. Gemini `streamGenerateContent` 流式。
6. 统一入站执行链路。
7. 统一错误映射。
8. 流式日志摘要记录。
9. 针对三套协议的单测与集成测试。

### 3.2 本次明确不做

1. 多模态完整支持。
2. 跨 provider 工具调用转换。
3. 跨 route 的流中 fallback。
4. 新增复杂负载均衡策略。
5. 熔断、自动摘除、自动健康探测。
6. 复杂 reasoning 字段统一抽象。

本次只保证 text-only 主链路稳定可用，并保证流式真正可被客户端逐段消费。

## 4. 总体架构

### 4.1 入站协议层

新增四个协议入口：

1. OpenAI `POST /v1/chat/completions`
2. Claude `POST /v1/messages`
3. Gemini `POST /v1beta/models/{model}:generateContent`
4. Gemini `POST /v1beta/models/{model}:streamGenerateContent`

各 handler 只负责：

1. 解析协议请求。
2. 判断是否流式。
3. 调用统一执行层。
4. 按原协议格式写回成功响应或错误响应。

协议差异只留在边界层，不渗透到选路、执行与日志主链路。

### 4.2 协议转换层

继续复用现有 codec：

1. `internal/provider/openai_codec.go`
2. `internal/provider/claude_codec.go`
3. `internal/provider/gemini_codec.go`

需要补的能力：

1. Claude 流式事件解析与回写。
2. Gemini 流式 chunk 解析与回写。
3. OpenAI 流式链路的真流式透传支持。

原则：能原样透传就原样透传，避免网关层重组协议事件导致漂移。

### 4.3 统一运行时内核

统一执行层不是当前已经接入代理入口的现成闭环，而是本次要完成的主改造点。

已确认当前现状：

1. `internal/app/app.go` 里的 `ProxyChat` 仍直接调用 `GetFirstEnabledChannel` 加 `OpenAICompatibleProvider.ForwardChatCompletions`。
2. 现有代理入口没有接入 `GatewayService`。
3. 现有 `GatewayService` 只服务于路由与 fallback 逻辑雏形，本次需要把它纳入真实代理主链路，或抽出与其等价的新统一执行入口。

因此本次设计里的“统一执行层”本质上是：

1. 替换当前 `ProxyChat` 的直接转发路径。
2. 把协议入站统一接到真实运行时选路与 executor 执行闭环。
3. 让 Claude / Gemini / OpenAI 三套入站协议共享同一条执行主链路。

统一执行层最终负责：

1. API Key 校验。
2. 限流检查。
3. 请求体转统一模型。
4. 路由选择。
5. 上游 executor 调用。
6. 错误归一化。
7. 日志落库。

这样可以保证三套协议共享一套业务执行闭环。

## 5. 统一运行时设计

### 5.1 handler 分层

当前 `gateway_handlers.go` 需要拆成两层：

#### 协议边界层

- `HandleOpenAIChatCompletions`
- `HandleClaudeMessages`
- `HandleGeminiGenerateContent`
- `HandleGeminiStreamGenerateContent`

职责：

1. decode 协议请求。
2. 构造统一请求。
3. 选择非流式或流式写回方式。

#### 统一执行层

统一执行层负责：

1. 鉴权。
2. 限流。
3. 执行路由。
4. 触发 executor。
5. 记录日志。
6. 返回统一执行结果。

### 5.2 请求统一模型

继续复用现有统一模型：

- `UnifiedChatRequest`
- `UnifiedChatResponse`
- `UnifiedStreamEvent`

本次不大改现有请求模型，只补最小必要字段与流式返回结构。

### 5.3 统一执行结果模型

当前现状需要先对齐：

1. `domain.ProxyResponse` 现在的 `Body` 是 `[]byte`。
2. `domain.ExecutionResult` 现在只承载整包响应。
3. executor 统一通过 `doExecutorRequest` 读取完整 body，流式链路还没有独立返回结构。

所以本次不是“补个标志位”就够，而是要新增最小结果分层。

建议落地成两类结果：

1. 非流式结果，继续使用整包 `ProxyResponse`。
2. 流式结果，新增最小 `StreamResult`，至少包含：
   - `StatusCode`
   - `Headers`
   - `BodyReader`
   - `Protocol`
   - `UsageExtractor` 或等价摘要钩子

再由统一执行层返回统一结果包，显式区分：

1. `Response *ProxyResponse`
2. `Stream *StreamResult`

这样 handler 才能明确判断：

1. 非流式走整包 encode / write。
2. 流式走 copy + flush。

### 5.4 executor 复用策略

保留现有三种 executor：

1. `OpenAIExecutor`
2. `ClaudeExecutor`
3. `GeminiExecutor`

当前现状需要正面承认：

1. 三个 executor 现在都通过 `doExecutorRequest` 统一 `io.ReadAll(resp.Body)`。
2. 它们当前没有流式与非流式分叉返回。
3. `GeminiExecutor` 当前只构造 `generateContent` URL，没有 `streamGenerateContent` URL 分支。

因此本次改造要求是：

1. executor 显式区分流式与非流式路径。
2. 非流式继续返回完整 `ProxyResponse`。
3. 流式返回 `StreamResult`，把上游 body reader 保留给 handler。
4. Gemini executor 新增流式 URL 构造逻辑，分别支持：
   - `generateContent`
   - `streamGenerateContent`
5. OpenAI 与 Claude executor 在流式模式下不允许读完整 body。

也就是说，本次会对 executor 返回模型和执行分支做实质改造，而不是在现有实现上做轻量修补。

### 5.5 路由策略

本次不扩产品范围，不重做调度策略。

原则：

1. 明确当前代理入口还未接入 `GatewayService`，现状仍是 `ProxyChat -> GetFirstEnabledChannel -> OpenAICompatibleProvider`。
2. 本次要把真实代理入口改接到统一执行层，不能继续停留在直接转发路径。
3. 现有 `GatewayService` 是最接近统一执行层的现成抓手，但需要扩展为同时承载非流式与流式结果。
4. 不顺手增加新业务配置项。

换句话说，本次不是“沿用已有统一执行主链路”，而是“把现有未接入主链路的统一执行能力正式拉到代理入口上”。

### 5.6 日志策略

继续复用现有请求日志表。

非流式：

1. 记录请求摘要。
2. 记录响应摘要。
3. 提取 usage。

流式：

1. 记录请求摘要。
2. 记录状态码。
3. 记录耗时。
4. usage 只允许通过流式结束后的摘要钩子或尾段提取，禁止在写流前预读整个 body。
5. 不要求保存全量响应 body。

当前现状里，handler 会在 `CopyProxy` 之前直接对响应 body 提取 usage，这一逻辑必须在流式场景下拆除，否则会把流重新读成整包，破坏真流式。

### 5.7 错误映射

统一执行层只返回统一错误语义：

1. 状态码。
2. 错误信息。
3. 是否可重试。
4. 是否已开始流式输出。

最终由协议边界层映射成：

1. OpenAI error JSON。
2. Claude error JSON。
3. Gemini error JSON。

## 6. 流式设计

### 6.1 OpenAI 真流式

当前问题是代理层先 `io.ReadAll(resp.Body)`，导致客户端拿到的是整包结果。

改造后：

1. 非流式继续完整读取响应。
2. 流式单独走 stream proxy。
3. handler 边读上游 body，边写下游 `ResponseWriter`。
4. 每个 chunk 写出后立刻 `Flush()`。
5. 保持 OpenAI SSE 格式原样透传，不在网关层重组事件。

### 6.2 Claude 原生流式

入口统一为 `POST /v1/messages`。

非流式：

1. decode Claude request。
2. 转统一请求。
3. 调用执行层。
4. encode 成 Claude messages response。

流式：

1. 请求中 `stream=true` 时走流式通路。
2. 优先原样透传 Anthropic event stream。
3. 保留 Claude 事件序列：
   - `message_start`
   - `content_block_start`
   - `content_block_delta`
   - `content_block_stop`
   - `message_delta`
   - `message_stop`

### 6.3 Gemini 原生流式

新增两个入口：

1. `generateContent`
2. `streamGenerateContent`

非流式：

1. decode Gemini request。
2. 转统一请求。
3. 执行并 encode 回 Gemini response。

流式：

1. `streamGenerateContent` 明确走流式执行。
2. executor 需要显式区分 Gemini 非流式 URL 与流式 URL，分别命中 `generateContent` 与 `streamGenerateContent`。
3. 路由到 Gemini 上游时优先透传原生流式返回。
4. 本次范围只承诺 Gemini 原生入口对接 Gemini 上游时的原生真流式，不扩展到跨协议流式重编码。

这样可以保证这次验收闭环与当前仓库能力对齐，避免把跨协议事件重组带进本轮实现范围。

### 6.4 统一流式结果抽象

新增最小流式结果结构，至少包含：

1. `StatusCode`
2. `Headers`
3. `BodyReader`
4. `Protocol`
5. `UsageExtractor` 或等价摘要钩子

这样 handler 可以统一处理：

1. 非流式整包写回。
2. 流式 copy + flush。

### 6.5 流式错误边界

流式错误必须区分两种场景：

1. 首包前失败
   - 可以返回标准协议错误响应。
2. 首包后失败
   - 不能再切换成普通 JSON 错误。
   - 只能按当前协议结束流，或直接中断连接。

因此统一执行层必须保留 `streamStarted` 语义，禁止首包后 fallback。

## 7. 代码改造落点

### 7.1 需要修改的核心文件

1. `internal/app/app.go`
   - 注入新的 gateway 依赖
   - 不再只绑定 OpenAI chat completions

2. `internal/transport/httpserver/router.go`
   - 注册 Claude / Gemini 原生路由
   - 保留 OpenAI 路由

3. `internal/transport/httpserver/gateway_handlers.go`
   - 拆分协议 handler 与统一执行入口
   - 增加流式写回逻辑

4. `internal/provider/executors.go`
   - 补流式执行返回能力

5. `internal/provider/openai_compatible.go`
   - 移除当前假流式整包读取逻辑

6. `internal/provider/claude_codec.go`
   - 补流式事件处理能力

7. `internal/provider/gemini_codec.go`
   - 补 `streamGenerateContent` 处理能力

8. `internal/domain/gateway.go`
   - 补最小必要的流式结果结构或相关字段

### 7.2 不做的无关扩展

1. 不新增无关配置项。
2. 不重构前端。
3. 不改后台管理页交互。
4. 不顺手重做模型路由产品逻辑。

## 8. 测试与验收

### 8.1 单元测试

必须补四组测试：

#### 一，router / handler

覆盖：

1. OpenAI `/v1/chat/completions`
2. Claude `/v1/messages`
3. Gemini `generateContent`
4. Gemini `streamGenerateContent`

验证点：

1. 路由命中正确。
2. 缺 key 返回正确错误。
3. key 无效返回正确错误。
4. 非流式返回正确协议格式。
5. 流式返回正确 `Content-Type`。

#### 二，codec

覆盖：

1. Claude request decode / response encode
2. Gemini request decode / response encode
3. Claude stream event 处理
4. Gemini stream chunk 处理

验证点：

1. model 解析正确。
2. messages / contents 转换正确。
3. stream 标志识别正确。
4. Gemini path model 与 body model 冲突能报错。

#### 三，executor

覆盖：

1. OpenAI 非流式
2. OpenAI 流式
3. Claude 非流式
4. Claude 流式
5. Gemini 非流式
6. Gemini 流式

验证点：

1. 请求 URL 正确。
2. Header 正确。
3. 流式时不允许 `ReadAll`。
4. 流式时保留上游 body 给 handler 消费。

#### 四，统一执行层

覆盖：

1. 正常选路成功。
2. 首个上游失败后 fallback。
3. 流式首包前失败允许 fallback。
4. 流式首包后失败禁止 fallback。

### 8.2 集成测试

使用 `httptest.Server` 模拟上游流式输出，验证客户端能逐段收到内容。

必须覆盖：

#### OpenAI

1. 客户端逐段读到 `data:` chunk。
2. 不是等待服务端结束后一次性收到。

#### Claude

1. 顺序收到 `message_start`。
2. 顺序收到 `content_block_delta`。
3. 顺序收到 `message_stop`。

#### Gemini

1. `streamGenerateContent` 逐段收到 chunk。
2. `generateContent` 返回完整 JSON。

### 8.3 最终验收口径

必须同时满足：

1. OpenAI `/v1/chat/completions` 非流式可用。
2. OpenAI `/v1/chat/completions` 真流式可用。
3. Claude `/v1/messages` 非流式可用。
4. Claude `/v1/messages` 真流式可用。
5. Gemini `generateContent` 可用。
6. Gemini `streamGenerateContent` 真流式可用。
7. 三套接口都能通过统一鉴权、限流、路由、日志闭环。
8. `go test ./...` 通过。

## 9. 结论

本次改造的底层逻辑是：保留现有三类 executor 和现有 codec 基础，新增 Claude 与 Gemini 原生入站路由，抽统一执行层，补齐 OpenAI 真流式，同时让 Claude `/v1/messages` 与 Gemini `generateContent` / `streamGenerateContent` 全部走统一鉴权、限流、路由、日志闭环。
