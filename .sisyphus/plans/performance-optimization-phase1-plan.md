# OpenCrab 性能优化第一阶段实施计划

## 1. 目标

第一阶段只解决当前最确定、收益最高的两类问题：

1. 管理台首页和日志页对 `request_logs` 的全量重复读取。
2. 日志接口一次返回宽字段，导致 SSR 体积、JSON 解析和内存聚合成本过高。

本阶段不处理 Redis、异步日志、拆库拆表、分布式缓存。

> 状态说明：第一阶段已经完成并落地到当前仓库。
> 本文件第 2 节到第 8 节保留为第一阶段归档记录。
> 后续继续执行时，直接从第 9 节开始。

## 2. 第一阶段归档，代码证据与根因

### 2.1 首页双重读取日志

- `web/src/app/(console)/page.tsx` 当前会同时请求 `getAdminLogs()` 和 `getAdminRoutingOverview()`。
- `internal/store/sqlite/admin_store.go` 中：
  - `ListRequestLogs()` 会全量读取 `request_logs`。
  - `GetRoutingOverview()` 内部又会调用 `ListRequestLogs()`，再次全量读取 `request_logs`，再在 Go 内存里做 24h 聚合和 JSON 解析。

### 2.2 日志接口返回宽字段

- `internal/domain/admin.go` 的 `RequestLog` 同时包含 `request_body`、`response_body`、`details`。
- `internal/store/sqlite/request_log_store.go` 与 `internal/store/sqlite/admin_store.go` 的日志查询当前直接返回这三个大字段。
- `web/src/app/(console)/logs/page.tsx` 的列表页和详情抽屉共享同一份全量数据，导致列表加载时就把详情大字段一起 SSR。

### 2.3 首页重复聚合

- `web/src/app/(console)/page.tsx` 会在拿到全量日志后多轮 `filter/reduce/map` 做日趋势、4 小时桶、最近活动、Token 汇总、缓存命中率等计算。
- 当前 `force-dynamic` 和 `no-store` 会让这些计算在每次请求时重新执行。

## 3. 第一阶段归档，实施范围

### 3.1 后端新增或调整接口

#### A. 日志轻量列表接口

保留路径：`GET /api/admin/logs`

改为返回轻量日志列表，不再返回：

- `request_body`
- `response_body`

保留：

- `id`
- `request_id`
- `model`
- `channel`
- `status_code`
- `latency_ms`
- `prompt_tokens`
- `completion_tokens`
- `total_tokens`
- `cache_hit`
- `details`
- `created_at`

说明：

1. 第一阶段保留 `details`，因为首页和日志页当前都依赖其中的 `log_type`、`selected_channel`、`sticky_hit`、`fallback_chain`、`skips` 等摘要信息。
2. 先去掉 `request_body` 和 `response_body` 这两个最重的大字段，控制风险。

#### B. 单条日志详情接口

新增路径：`GET /api/admin/logs/{id}`

返回完整日志详情，包含：

- 轻量列表已有字段
- `request_body`
- `response_body`

说明：

1. 日志详情抽屉改为按 id 获取。
2. 列表页不再在首次 SSR 时携带大字段。

#### C. Dashboard 聚合接口

新增路径：`GET /api/admin/dashboard/summary`

返回首页真正需要的聚合结果，至少包含：

1. 资源规模：`channels`、`models`、`routes`、`api_keys` 数量。
2. 路由运行态：`routing_overview` 当前已有字段。
3. 请求统计：
   - 今日请求数
   - 成功数 / 错误数
   - 平均延迟
   - prompt/completion/total token 汇总
   - 缓存命中数 / 命中率
   - 最近 60 秒 RPM / TPM
4. 图表数据：
   - 最近 7 天 daily counts
   - 最近 24 小时 4 小时桶 traffic series
5. 最近活动：最近 5 条 `gateway_request` 摘要。
6. 常用模型排行与渠道占比。

说明：

1. 该接口在后端完成聚合，首页不再自行拉全量日志做重复运算。
2. 该接口内部可以继续复用现有 `GetRoutingOverview()`，但不能再额外全量读取两遍 `request_logs`。

## 4. 第一阶段归档，修改文件

### 后端

1. `internal/domain/admin.go`
   - 新增轻量日志类型。
   - 新增单条日志详情类型。
   - 新增 dashboard summary 返回类型。

2. `internal/store/sqlite/admin_store.go`
   - 将现有全量日志查询拆分为：
     - 轻量日志列表查询
     - 单条日志详情查询
   - 新增 dashboard summary 聚合查询与聚合逻辑。
   - 调整 `GetRoutingOverview()`，避免内部依赖全量宽日志对象。

3. `internal/store/sqlite/request_log_store.go`
   - 仅在需要时复用写入逻辑。
   - 不强求第一阶段修改写入结构。

4. `internal/transport/httpserver/router.go`
   - 为 logs 增加 `GET /logs/{id}`。
   - 新增 `GET /dashboard/summary`。
   - 补充依赖注入字段。

5. `internal/app/app.go`
   - 注入新增的日志详情查询和 dashboard summary 逻辑。

### 前端

6. `web/src/lib/admin-api.ts`
   - 新增轻量日志类型。
   - 新增日志详情类型。
   - 新增 dashboard summary 类型。

7. `web/src/lib/admin-api-server.ts`
   - `getAdminLogs()` 改为请求轻量列表。
   - 新增 `getAdminLogDetail(id)`。
   - 新增 `getAdminDashboardSummary()`。

8. `web/src/app/(console)/page.tsx`
   - 改为使用 `getAdminDashboardSummary()`。
   - 移除页面内对全量日志的重复聚合。
   - 不再直接请求 `/api/admin/logs`。

9. `web/src/app/(console)/logs/page.tsx`
   - 列表只消费轻量日志。
   - 详情抽屉改为按 id 拉取单条详情。
   - 避免列表 SSR 时携带 `request_body` / `response_body`。

## 5. 第一阶段归档，实施顺序

### 步骤 1

先定义后端与前端的新类型，明确：

1. 轻量日志列表返回结构。
2. 单条日志详情返回结构。
3. Dashboard summary 返回结构。

#### 步骤 1 QA

1. 工具：代码回读。
2. 操作：检查 `internal/domain/admin.go` 与 `web/src/lib/admin-api.ts`。
3. 预期结果：
   - 后端与前端都存在轻量日志类型、日志详情类型、dashboard summary 类型。
   - 轻量日志类型不包含 `request_body` 与 `response_body`。
   - dashboard summary 类型字段足以覆盖首页当前展示。

### 步骤 2

改后端 store 和 router：

1. 拆出轻量日志查询。
2. 增加按 id 详情查询。
3. 增加 dashboard summary 聚合接口。

#### 步骤 2 QA

1. 工具：`go test ./...`。
2. 操作：运行后端测试，确认新增 store 与 router 逻辑未破坏现有后端行为。
3. 预期结果：测试全部通过。
4. 工具：`curl` 或等价 HTTP 请求。
5. 操作：分别请求：
   - `GET /api/admin/logs`
   - `GET /api/admin/logs/{id}`
   - `GET /api/admin/dashboard/summary`
6. 预期结果：
   - `/logs` 列表结果不再返回 `request_body` 与 `response_body`。
   - `/logs/{id}` 返回完整详情。
   - `/dashboard/summary` 返回首页渲染所需字段。

### 步骤 3

改前端数据访问层：

1. 更新 `admin-api.ts` 类型。
2. 更新 `admin-api-server.ts` 请求函数。

#### 步骤 3 QA

1. 工具：代码回读。
2. 操作：检查 `web/src/lib/admin-api.ts` 与 `web/src/lib/admin-api-server.ts`。
3. 预期结果：
   - `getAdminLogs()` 对应轻量列表。
   - 存在 `getAdminLogDetail(id)`。
   - 存在 `getAdminDashboardSummary()`。
   - 旧的 dashboard 页面调用链不再依赖全量日志类型。

### 步骤 4

改 dashboard 页面：

1. 首页改为单次请求 summary 接口。
2. 删除页面内部重复日志聚合逻辑。

#### 步骤 4 QA

1. 工具：`pnpm build`，目录 `web/`。
2. 操作：构建前端，确认类型和 SSR 均通过。
3. 预期结果：构建成功。
4. 工具：代码回读。
5. 操作：检查 `web/src/app/(console)/page.tsx`。
6. 预期结果：
   - 页面不再直接调用 `getAdminLogs()`。
   - 页面不再保留基于全量日志的多轮 `filter/reduce/map` 聚合主逻辑。
   - 页面改为消费 `getAdminDashboardSummary()` 返回的数据。

### 步骤 5

改 logs 页面：

1. 列表使用轻量日志。
2. 详情按 id 获取。
3. 确保详情交互仍完整。

#### 步骤 5 QA

1. 工具：`pnpm build`，目录 `web/`。
2. 操作：再次构建前端，确认日志页改造未引入类型或 SSR 问题。
3. 预期结果：构建成功。
4. 工具：页面手动验证。
5. 操作：打开日志页，检查列表与详情抽屉。
6. 预期结果：
   - 列表正常显示。
   - 打开详情时可以看到 `request_body`、`response_body`、`details`。
   - 列表首屏数据不再携带完整大字段。

## 6. 第一阶段归档，验证要求

### 后端

1. `go test ./...`
2. 手动核对：
   - `/api/admin/logs` 结果不再包含 `request_body` 与 `response_body`
   - `/api/admin/logs/{id}` 能返回完整详情
   - `/api/admin/dashboard/summary` 能返回首页所需字段

### 前端

1. `pnpm build` in `web/`
2. 核对首页：
   - 不再直接依赖 `getAdminLogs()`
   - 渲染数据来自 summary 接口
3. 核对日志页：
   - 列表渲染正常
   - 详情抽屉仍可看到 request/response/details

## 7. 第一阶段归档，当时暂不做

1. Redis
2. 异步日志落盘
3. 拆独立日志表或独立日志库文件
4. `GetGatewayRuntimeSettings` 双读优化
5. `gateway_request` / `gateway_attempt` 写入结构瘦身
6. SQLite WAL、busy timeout、连接池策略
7. 新索引迁移

说明：这些会放到第二阶段，因为第一阶段先处理日志读放大，收益最直接，回归范围也最可控。

## 8. 第一阶段归档，风险与约束

1. `details` 当前仍要保留在轻量列表里，否则首页和日志页会丢摘要能力。
2. dashboard summary 的统计口径必须与当前页面展示一致，避免前后端切换后数据突变。
3. 日志详情改为按 id 请求后，要确保 SSR/客户端交互不破坏现有 UI 行为。
4. 第一阶段追求最小改动，不引入新的基础设施或大规模重构。

## 9. 第二阶段实施计划，日志写放大治理

### 9.1 目标

减少成功路径日志的重复宽字段写入，控制 `request_logs` 增长速度和单次请求写放大。

### 9.2 改动范围

1. `internal/transport/httpserver/gateway_handlers.go`
   - `gateway_request` 成功摘要不再把 `request_body` 和 `response_body` 重复写进 `details`。
   - 失败摘要保留必要错误信息，避免丢失排障抓手。
2. `internal/store/sqlite/request_log_store.go`
   - 成功态 `gateway_attempt` 不再写入 `request_body` 与 `response_body`。
   - 失败态、fallback、cooldown skip 继续保留必要上下文。
3. `internal/store/sqlite/admin_store.go`
   - 与日志详情展示保持兼容。

### 9.3 实施步骤

1. 区分成功态和失败态 attempt 日志写入内容。
2. 精简 `gateway_request` 的 `details` 载荷，去掉重复体字段。
3. 回读首页、日志页和 overview 的详情依赖，确保不破坏现有摘要展示。

### 9.4 第二阶段 QA

1. 工具：`go test ./...`。
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `gateway_handlers.go` 与 `request_log_store.go`。
6. 预期结果：
   - 成功态日志不再重复记录大字段。
   - 失败态和排障必要字段仍保留。

## 10. 第三阶段实施计划，网关热路径压缩

### 10.1 目标

减少单次网关请求中的重复读和无意义写入。

### 10.2 改动范围

1. `internal/transport/httpserver/gateway_handlers.go`
   - 不再与 `GatewayService.Execute` 重复读取运行时设置。
2. `internal/usecase/gateway_service.go`
   - 支持复用预读取的运行时设置。
   - 仅在 cooldown、sticky、cursor 状态发生变化时写库。
3. `internal/store/sqlite/routing_store.go`
   - 提供必要的无变更短路逻辑或状态检查支持。

### 10.3 实施步骤

1. 把 runtime settings 读取下沉到单点，避免双读。
2. 对 `ClearCooldown` 做状态短路，只在原本处于 cooldown 时写库。
3. 对 sticky 绑定和 round robin cursor 推进补充必要条件，避免明显无意义写入。

### 10.4 第三阶段 QA

1. 工具：`go test ./...`。
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `gateway_handlers.go`、`gateway_service.go`、`routing_store.go`。
6. 预期结果：
   - `GetGatewayRuntimeSettings` 不再在 handler 和 service 双读。
   - `ClearCooldown` 存在状态短路。
   - sticky/cursor 写入路径比之前更收敛。

## 11. 第四阶段实施计划，SQLite 护栏

### 11.1 目标

补齐 SQLite 的并发写保护和基础运行参数，降低锁冲突放大效应。

### 11.2 改动范围

1. `internal/store/sqlite/db.go`
   - 开启 WAL。
   - 设置 busy timeout。
   - 明确连接池策略。

### 11.3 实施步骤

1. 更新 DSN 或连接初始化逻辑。
2. 设置合理的 `SetMaxOpenConns`、`SetMaxIdleConns`、`SetConnMaxLifetime`。
3. 确保不影响当前测试和启动路径。

### 11.4 第四阶段 QA

1. 工具：`go test ./...`。
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `db.go`。
6. 预期结果：
   - 已显式启用 WAL 与 busy timeout。
   - 已设置连接池策略。

## 12. 第五阶段实施计划，索引优化

### 12.1 目标

基于前四个阶段稳定后的真实查询形态，补充热路径索引。

### 12.2 改动范围

1. `internal/store/sqlite/migrations/` 新增迁移。
2. 针对以下查询补索引：
   - `api_keys(key_hash, enabled)`
   - `model_routes(model_alias, priority)`
   - `request_logs(created_at)` 或与最新查询形态匹配的索引

### 12.3 实施步骤

1. 新增迁移文件。
2. 保证迁移可重复执行。
3. 回读热查询 SQL，确认索引和当前查询一致。

### 12.4 第五阶段 QA

1. 工具：`go test ./...`。
2. 操作：运行后端测试，确认迁移与查询行为正常。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查新迁移文件与热查询 SQL。
6. 预期结果：
   - 索引字段和当前查询条件一致。
   - 没有为已不用的旧查询形态盲目加索引。
