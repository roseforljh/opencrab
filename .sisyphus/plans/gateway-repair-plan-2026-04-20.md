# OpenCrab 网关修复总方案

## 1. 背景与目标

当前 OpenCrab 已经具备多协议入口和部分多 provider 能力，但对外行为仍存在三类核心问题：

1. 对外接口面不完整，`responses`、`realtime`、`models` 等 surface 仍缺少关键对象与生命周期接口。
2. 协议保真不足，`responses` 与 `realtime` 中仍存在大量本地合成、事件重建、字段删改和语义漂移。
3. 控制面和运行时真相不一致，模型发现、路由选择、部分后台字段与真实执行行为存在偏差。

本次修复目标不是继续叠功能，而是把当前网关从“部分可用的协议翻译层”修到“协议面更完整、语义更稳定、控制面与运行时一致”的状态。

## 2. 范围

### 2.1 本次必须完成

1. 写入并固化新的总修复方案，替换仓库中过期叙述。
2. 收敛共享运行时底座，统一模型可见性、route 选择和错误映射。
3. 修正 `/v1/models` 语义，使其反映真实可执行模型，而不是本地 alias 壳。
4. 补全 `responses` 的最小生命周期接口。
5. 修正 `responses` 的 continuation、stream、retrieve、websocket 语义漂移。
6. 修正 `realtime` native 与 fallback 两条路径的选路、错误和事件一致性。
7. 统一 OpenAI / Claude / Gemini / Codex 的错误形状策略。
8. 同步更新部署、FAQ、架构与仓库说明文档。

### 2.2 本次明确不做

1. 不顺手扩成 NewAPI 那种平台型产品。
2. 不引入计费、钱包、多租户、工作流、外部队列等新系统。
3. 不做与当前问题无关的前端重构。
4. 不做大规模 UI 改版。

## 3. 现状问题清单

### 3.1 接口面问题

1. 缺少 `responses` retrieve/delete/input_items 等生命周期接口。
2. 缺少更完整的模型详情接口。
3. 常见 OpenAI family 端点仍有较大缺口。

### 3.2 协议保真问题

1. `responses` 存在本地合成事件、状态重建、usage 补写。
2. `realtime` fallback 仍是 `responses` 投影，不是真实时协议执行。
3. 流式链路仍有 synthetic stream 行为。
4. 多 provider 桥接时存在字段删改与默认值注入。

### 3.3 控制面与运行时错位

1. `/v1/models` 只返回 alias，缺少真实可执行语义。
2. realtime native 直连和 gateway 主链路的选路策略不一致。
3. `dispatch_weight`、`enabled_for_async` 等控制面字段未完整兑现到运行时。
4. 文档仍有大量“只有 `/v1/chat/completions` 真入口”的过时描述。

## 4. 实施原则

1. 先统一底座，再补 surface，最后修 fidelity 和文档。
2. 优先最小改动复用现有结构，不额外引入实体，除非现有存储无法承载目标语义。
3. 新接口优先复用现有 `GatewayJob`、`ResponseSessionStore`、`GatewayService`、`codec` 体系。
4. 所有阶段必须带测试和回归验证。
5. 文档与代码同步推进，禁止代码修完后再整体补文档。

## 5. 分阶段计划

### Phase 0：方案与文档真相收敛

目标：先把单一真相写进项目，消除过期文档对后续开发的干扰。

涉及文件：

1. `.sisyphus/plans/gateway-repair-plan-2026-04-20.md`
2. `docs/deploy.md`
3. `docs/faq.md`
4. `docs/architecture.md`
5. `AGENTS.md`

动作：

1. 记录当前真实公开路由矩阵。
2. 更新运行时事实，移除“只接 chat/completions”“只取第一个 enabled channel”这类旧描述。
3. 补入本次修复阶段和验收规则。

验收：

1. 文档描述与 `internal/app/app.go`、`internal/transport/httpserver/router.go` 一致。
2. 仓库内不再保留明显冲突的运行时描述。

### Phase 1：共享运行时底座收敛

目标：统一模型可见性、route 选择、provider 过滤、cooldown 过滤、错误映射出口。

涉及文件：

1. `internal/app/app.go`
2. `internal/transport/httpserver/gateway_handlers.go`
3. `internal/transport/httpserver/helpers.go`
4. `internal/usecase/gateway_service.go`
5. `internal/store/sqlite/gateway_store.go`
6. `internal/domain/proxy.go`

动作：

1. 抽统一 route selection helper，避免 realtime/native 走旁路。
2. 统一 API key scope、cooldown、provider 和 alias 解析。
3. 统一 gateway error 结构和状态码来源。
4. 明确 public model id 与 upstream model 的职责边界。

验收：

1. chat、responses、realtime native 至少共享同一套 route 选择事实。
2. 错误出口不再依赖大量字符串 contains 判定。

### Phase 2：模型发现与控制面语义修正

目标：让 `/v1/models` 和管理面模型视图反映真实可调用状态。

涉及文件：

1. `internal/transport/httpserver/gateway_handlers.go`
2. `internal/store/sqlite/gateway_store.go`
3. `internal/store/sqlite/admin_store.go`
4. `internal/domain/admin.go`
5. `web/src/lib/admin-api.ts`
6. `web/src/app/(console)/models/models-client.tsx`

动作：

1. `/v1/models` 改为按 scope、已启用 route、provider 可执行性输出。
2. 模型列表包含 route 执行语义，而不只是 alias。
3. 后台模型页与运行时实际映射保持一致。

验收：

1. 不再出现“列表可见但实际不可路由”的模型。
2. 前台与后台对同一模型的解释一致。

### Phase 3：Responses 生命周期接口补齐

目标：把 `responses` 从 create-only 补成最小可管理对象。

涉及文件：

1. `internal/transport/httpserver/router.go`
2. `internal/transport/httpserver/gateway_handlers.go`
3. `internal/store/sqlite/response_session_store.go`
4. `internal/domain/proxy.go`
5. 必要时 `internal/store/sqlite/migrations/*.sql`

动作：

1. 新增 `GET /v1/responses/{responseID}`。
2. 新增 `DELETE /v1/responses/{responseID}`。
3. 新增 `GET /v1/responses/{responseID}/input_items`。
4. 视需要新增 `GET /v1/models/{model}`。

验收：

1. create → retrieve → input_items → delete 可闭环。
2. session/object 查询语义一致。

### Phase 4：Responses fidelity 修复

目标：对齐 REST、SSE、WebSocket 三条 responses 路径的状态、stream、continuation 语义。

涉及文件：

1. `internal/transport/httpserver/responses_ws.go`
2. `internal/transport/httpserver/gateway_handlers.go`
3. `internal/provider/responses_codec.go`
4. `internal/provider/responses_projector.go`
5. `internal/store/sqlite/response_session_store.go`
6. `internal/provider/executors.go`

动作：

1. 统一 `response.create`、`response.append`、`generate=false` 语义。
2. 修正 `previous_response_id` 和 transcript 裁剪行为。
3. 修复 retrieve 与 websocket 结果不一致问题。
4. 修正 stream 完成态恢复逻辑。
5. 修掉乱码错误文案。

验收：

1. REST、SSE、WebSocket 的 response object 能互相对齐。
2. continuation 和 context 清理逻辑可测试、可预测。

### Phase 5：Realtime fidelity 修复

目标：让 native passthrough 与 fallback synthesized realtime 使用统一运行时事实，并尽量减少协议漂移。

涉及文件：

1. `internal/transport/httpserver/realtime_native.go`
2. `internal/transport/httpserver/realtime_ws.go`
3. `internal/provider/realtime_codec.go`
4. `internal/provider/native_provider.go`
5. `internal/app/app.go`

动作：

1. native realtime 走统一 route 选择。
2. 错误输出统一改成协议化 JSON。
3. 补齐 session/update/error/output 事件一致性。
4. 尽量放开当前仅 text modality 的限制，至少明确降级行为。
5. 统一 passthrough 与 fallback 的日志和 scope 行为。

验收：

1. realtime native 与 fallback 不再使用两套路由真相。
2. 事件、错误、状态对象的行为可预测。

### Phase 6：错误形状与剩余协议漂移收口

目标：把 OpenAI、Claude、Gemini、Codex 各 surface 的错误对象和剩余协议漂移收拢。

涉及文件：

1. `internal/transport/httpserver/gateway_handlers.go`
2. `internal/provider/openai_compatible.go`
3. `internal/provider/claude_codec.go`
4. 相关 Gemini codec/renderer

动作：

1. OpenAI error 至少补齐 `message/type/code/param` 策略。
2. Claude error 统一 status 和 body。
3. Gemini/Codex 避免直接泄露不匹配协议的错误形状。

验收：

1. 各 surface 的错误对象与对应协议预期更接近。
2. 401/403/404/409/429/5xx 都有回归测试。

## 6. 依赖顺序

严格顺序如下：

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. Phase 5
7. Phase 6

原因：

1. 不先修文档真相，后续实现会继续被旧设计误导。
2. 不先收敛底座，后续每个新 endpoint 都会复制不一致的路由和错误逻辑。
3. 不先修模型发现与控制面语义，responses/realtime 的新增接口也会建立在错误模型视图上。
4. 先补 surface，再修 fidelity，可以降低接口和语义同时变动的冲突风险。

## 7. 风险清单

1. `response_sessions` 当前更偏 transcript store，若要承载完整对象，可能需要最小 migration。
2. `/v1/models` 语义一旦调整，前端 models 页面与现有测试会联动变化。
3. realtime native 与 fallback 双路径并存，最容易产生行为分叉。
4. `executors.go` 当前已存在大量字段删改，修 fidelity 时必须配套回归测试。
5. 文档同步若落后，会再次把旧错误认知写回仓库。

## 8. 验证计划

每个 phase 都必须至少执行以下验证：

1. `go test ./internal/transport/httpserver ./internal/provider ./internal/usecase ./internal/store/sqlite`
2. 变更 migration 时，加跑相关 store 包测试。
3. 变更 admin API shape 或 models 页契约时，在 `web/` 下运行 `pnpm build`。
4. 最终全量回归运行 `go test ./...`。

关键回归矩阵：

1. `/v1/models`
2. `/v1/chat/completions`
3. `/v1/responses`
4. `/v1/realtime/*`
5. `/v1/messages`
6. Gemini `generateContent/streamGenerateContent/cachedContents`
7. `/v1/codex/responses`
8. `/v1/requests/{id}` 和 `/events`

### Phase 0 QA 场景

工具：`Read`、`grep`、必要时 `go test ./...`

步骤：

1. 对照 `internal/transport/httpserver/router.go`、`internal/app/app.go`、`AGENTS.md`、`docs/deploy.md`、`docs/faq.md`、`docs/architecture.md`。
2. 核对每份文档是否都写出当前真实公开路由面。
3. 核对是否仍存在“只有 `/v1/chat/completions` 真入口”“只取第一个 enabled channel”这类过时描述。

预期结果：

1. 以上文档中的运行时描述与源码一致。
2. 仓库内不再出现上述过时叙述。

### Phase 1 QA 场景

工具：`go test`、`curl`

步骤：

1. 运行 `go test ./internal/usecase ./internal/store/sqlite ./internal/transport/httpserver`。
2. 构造至少两条 route，其中一条处于 cooldown，一条可执行，分别走普通 gateway 请求与 realtime native 选路路径。
3. 使用 `curl` 访问 `/v1/chat/completions`、`/v1/realtime/client_secrets`，观察 route 选择和错误输出。

预期结果：

1. 普通 gateway 与 realtime native 不再出现不同 route 真相。
2. cooldown、scope、provider 过滤行为一致。
3. 错误状态码和错误对象来源一致，不再出现一条链返回 `http.Error`、另一条链返回私有 JSON 的分裂行为。

### Phase 2 QA 场景

工具：`go test`、`curl`、`pnpm build`

步骤：

1. 运行 `go test ./internal/transport/httpserver ./internal/store/sqlite`。
2. 准备包含不同 alias、不同 route、不同 API key scope 的测试数据。
3. 用 `curl -H "Authorization: Bearer ..." http://127.0.0.1:8080/v1/models` 分别请求受限与不受限 key。
4. 在 `web/` 下运行 `pnpm build`。

预期结果：

1. `/v1/models` 仅返回真正可执行的模型。
2. 不同 scope 下模型可见性正确。
3. 前端 models 页面构建通过，且契约未断裂。

### Phase 3 QA 场景

工具：`go test`、`curl`

步骤：

1. 运行 `go test ./internal/transport/httpserver ./internal/store/sqlite`。
2. 发送 `POST /v1/responses` 创建 response，保存返回的 `responseID`。
3. 依次请求：
   - `GET /v1/responses/{responseID}`
   - `GET /v1/responses/{responseID}/input_items`
   - `DELETE /v1/responses/{responseID}`
   - 再次 `GET /v1/responses/{responseID}`

预期结果：

1. create 返回 200/201 且对象可检索。
2. `input_items` 可返回对应输入项。
3. delete 成功后再次 retrieve 返回 404 或约定的删除态。

### Phase 4 QA 场景

工具：`go test`、`curl`、WebSocket 客户端

步骤：

1. 运行 `go test ./internal/provider ./internal/transport/httpserver`。
2. 同一组输入分别走：
   - `POST /v1/responses`
   - `POST /v1/responses` with stream
   - WebSocket `GET /v1/responses`
3. 对比 `response.create`、`response.append`、`previous_response_id`、`generate=false` 的对象字段和事件序列。

预期结果：

1. REST、SSE、WebSocket 返回的 response object 关键字段一致。
2. continuation 行为一致，不出现某一路径能续写、另一路径丢上下文的分裂。
3. 错误文案无乱码。

### Phase 5 QA 场景

工具：`go test`、`curl`、WebSocket 客户端

步骤：

1. 运行 `go test ./internal/transport/httpserver ./internal/provider`。
2. 分别验证：
   - `POST /v1/realtime/client_secrets`
   - `POST /v1/realtime/calls`
   - native realtime WebSocket
   - fallback realtime WebSocket
3. 在 route 受限、cooldown、provider 不匹配的情况下重复测试。

预期结果：

1. native 和 fallback 两条路径遵守相同 scope 与 route 规则。
2. 错误对象形状一致。
3. session/update/output/error 事件序列与计划一致。

### Phase 6 QA 场景

工具：`go test`、`curl`

步骤：

1. 运行 `go test ./internal/transport/httpserver ./internal/provider`。
2. 分别对 OpenAI、Claude、Gemini、Codex surface 构造以下错误：
   - 缺少 key
   - key 禁用
   - 模型不可见
   - route 不可用
   - provider 上游失败
3. 记录状态码和错误对象字段。

预期结果：

1. 各 surface 的错误对象与对应协议预期一致。
2. 401/403/404/409/429/5xx 行为稳定。
3. 不再出现某些 surface 只有裸文本错误、某些 surface 才有 JSON 错误的问题。

## 9. 当前执行起点

从现在开始，按以下顺序直接执行：

1. 落盘本方案。
2. 评审本方案并修正缺口。
3. 完成 Phase 0 文档同步。
4. 立即进入 Phase 1 后端修复。
