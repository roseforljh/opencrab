# OpenCrab 性能优化后续阶段实施计划

## 1. 当前基线

当前仓库已经完成第一阶段：

1. dashboard 改为走 `GET /api/admin/dashboard/summary`
2. `GET /api/admin/logs` 已改为轻量列表
3. `GET /api/admin/logs/{id}` 已支持单条详情
4. 日志页已改为摘要列表加按 id 拉详情

后续实施直接从第二阶段开始，不重复第一阶段。

## 2. 第二阶段，日志写放大治理

### 2.1 目标

减少成功路径日志的重复宽字段写入，控制 `request_logs` 增长速度和单次请求写放大。

### 2.2 代码证据

1. `internal/transport/httpserver/gateway_handlers.go`
   - `logGatewayRequestSummary()` 会把 `request_body`、`response_body` 同时写入列，又重复写入 `details`。
2. `internal/store/sqlite/request_log_store.go`
   - `LogGatewayAttempt()` 当前无论成功还是失败，都会把 `RequestBody`、`ResponseBody` 落进 `request_logs`。
3. 当前第一阶段已经把日志读取链路变轻，下一步收益最高的是继续压缩写入体积。

### 2.3 实施项

1. `gateway_request` 成功摘要从 `details` 中移除 `request_body`、`response_body`。
2. 成功态 `gateway_attempt` 不再写 `RequestBody`、`ResponseBody`。
3. 失败态、fallback、cooldown skip 保留必要上下文，保证排障可用。

### 2.4 QA

1. 工具：`go test ./...`
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `gateway_handlers.go` 与 `request_log_store.go`。
6. 预期结果：
   - 成功态日志不再重复写入宽字段。
   - 失败态必要排障字段仍保留。

## 3. 第三阶段，网关热路径压缩

### 3.1 目标

减少单次网关请求中的重复读和无意义写入。

### 3.2 代码证据

1. `internal/transport/httpserver/gateway_handlers.go`
   - handler 会先读一次 `GetGatewayRuntimeSettings()`。
2. `internal/usecase/gateway_service.go`
   - `Execute()` 里又会再读一次同一组 runtime settings。
3. `internal/store/sqlite/routing_store.go`
   - `ClearCooldown()` 当前每次成功都会写库。
   - `UpsertStickyBinding()` 和 `AdvanceRoutingCursor()` 没有针对无变化场景做短路。

### 3.3 实施项

1. 去掉 runtime settings 双读，只保留一次。
2. `ClearCooldown()` 仅在原本存在 cooldown 时写库。
3. sticky 绑定与游标推进只在必要时写库。

### 3.4 QA

1. 工具：`go test ./...`
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `gateway_handlers.go`、`gateway_service.go`、`routing_store.go`。
6. 预期结果：
   - `GetGatewayRuntimeSettings()` 不再双读。
   - `ClearCooldown()` 存在状态短路。
   - sticky/cursor 写入路径比之前更收敛。

## 4. 第四阶段，SQLite 护栏

### 4.1 目标

补齐 SQLite 的并发写保护和基础运行参数，降低锁冲突放大效应。

### 4.2 代码证据

1. `internal/store/sqlite/db.go` 当前只启用 `foreign_keys`。
2. 尚未显式设置 WAL、busy timeout、连接池策略。

### 4.3 实施项

1. 开启 WAL。
2. 设置 busy timeout。
3. 设置连接池策略。

### 4.4 QA

1. 工具：`go test ./...`
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查 `db.go`。
6. 预期结果：
   - 已显式启用 WAL 与 busy timeout。
   - 已设置连接池策略。

## 5. 第五阶段，索引优化

### 5.1 目标

基于当前真实查询形态补充热路径索引。

### 5.2 代码证据

1. `api_keys` 校验走 `key_hash + enabled`。
2. `ListEnabledRoutesByModel()` 热点依赖 `model_routes.model_alias` 与优先级排序。
3. 当前迁移缺少这些热索引。

### 5.3 实施项

1. 新增迁移，为 `api_keys(key_hash, enabled)` 建索引。
2. 新增迁移，为 `model_routes(model_alias, priority)` 建索引。
3. 基于当前日志查询形态，为 `request_logs(created_at)` 或等价字段补索引。

### 5.4 QA

1. 工具：`go test ./...`
2. 操作：运行后端测试。
3. 预期结果：测试全部通过。
4. 工具：代码回读。
5. 操作：检查新迁移文件与当前热查询 SQL。
6. 预期结果：
   - 索引字段与当前查询条件一致。
   - 没有对已不用的查询形态盲目加索引。
