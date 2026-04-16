# OpenCrab 10k 并发强保证调度方案

## 1. 目标定义

目标不是“10k 个 HTTP 连接同时挂着并立刻全部 200 返回”。

目标定义为：

1. 当 10k 个请求同时到达时，系统先完成 **durable accepted**。
2. 对所有 **accepted** 请求，系统保证最终得到成功结果，或在受理前明确拒绝，不出现“受理了但丢了”。
3. 成功交付可以是：
   - 同步返回
   - 异步轮询
   - SSE
   - webhook

因此，“保证成立”的对象是 **已被系统正式受理的请求**，不是任意一个瞬时连进来的 socket。

## 2. 保证成立的前提

只有同时满足以下前提，才能对“10k 同时到达最终全部成功”做强承诺：

1. **50 个渠道的 1000 rpm 限额彼此独立**。
2. 上游限额不只是 rpm，还要明确：
   - 是否存在 max inflight
   - 是否存在 token/minute 限制
   - 是否存在流式特殊限制
3. 客户端接受异步交付，或同步等待预算足够长。
4. 请求必须具备幂等键，避免重试导致副作用重复执行。
5. 运行时热状态不能继续放在 SQLite 上，要有支持原子预约和高并发 lease 的组件。

## 3. 为什么当前实现做不到

当前仓库只能做到“到达即选路并执行”，不能做到“全局受理后排队调度”：

1. `internal/transport/httpserver/gateway_handlers.go`
   - 当前入口是 API Key 校验、限流、解码后直接 `deps.ExecuteGateway(...)`
   - 没有入队、排队、202 受理、结果查询、回调能力
2. `internal/usecase/gateway_service.go`
   - 当前只有 `sequential / round_robin / sticky / cooldown / fallback`
   - 没有全局配额预约，没有每渠道未来时间槽管理
3. `internal/store/sqlite/routing_store.go`
   - 当前状态只有 cursor / cooldown / sticky
   - 不具备高并发下的原子配额扣减与 lease 管理能力
4. `internal/app/app.go`
   - 当前还有本地内存 rate limiter
   - 它适合简单保护，不适合 10k 突发下的全局调度

## 4. 总体架构

方案采用 **双平面调度**：

### 4.1 控制面

继续使用当前 Go + SQLite：

1. `channels`
2. `models`
3. `model_routes`
4. `system_settings`
5. `request_logs`
6. dashboard / settings / routing overview

控制面负责：

1. 渠道配置
2. 配额配置
3. 路由优先级
4. 审计和观测

### 4.2 运行时热平面

新增 Redis，负责：

1. 请求队列
2. job 状态
3. 渠道配额虚拟时钟
4. inflight 计数
5. worker lease
6. 调度结果缓存

SQLite 不再承担热路径一致性。

## 5. 必需新增组件

### 5.1 Admission Service

职责：

1. 校验 API Key
2. 生成 `request_id`
3. 生成 `idempotency_key`
4. 判定是同步桥接还是异步受理
5. durable accepted 后入队

接入位置：

- `internal/transport/httpserver/gateway_handlers.go`

### 5.2 Job Queue

职责：

1. 存储待执行 job
2. 按 `dispatch_at` 或优先级出队
3. 支持重试和回退重排

推荐实现：

- Redis Sorted Set + Hash

### 5.3 Quota Manager

职责：

1. 为每个渠道维护：
   - `rpm_limit`
   - `max_inflight`
   - `cooldown_until`
   - `health_penalty`
   - `tat`，Theoretical Arrival Time
2. 原子计算最早可发时间
3. 预约未来时间槽
4. 在失败时返还或延期处罚

推荐实现：

- Redis Lua 脚本

### 5.4 Dispatcher Workers

职责：

1. 从队列取出 ready job
2. 调用现有 `GatewayService` 或其拆分后的执行层
3. 更新 job 状态
4. 写回结果或继续重排

### 5.5 Result Delivery

职责：

1. 同步桥接
2. 轮询查询
3. SSE 推送
4. webhook 回调

### 5.6 Observability Extension

在现有 dashboard 基础上增加：

1. `queue_depth`
2. `oldest_job_age_ms`
3. `accepted_rps`
4. `dispatch_rps`
5. `channel_quota_utilization`
6. `channel_inflight`
7. `requeue_count`
8. `expired_jobs`

## 6. 配额与调度模型

### 6.1 核心思想

不是让请求“立刻找一个渠道试试”，而是先回答两个问题：

1. 哪个渠道最早还能合法发出一个请求
2. 这个请求现在被受理后，能否在 SLA 内完成

### 6.2 渠道状态

每个渠道维护以下运行时状态：

1. `rpm_limit`
2. `max_inflight`
3. `tat`
4. `cooldown_until`
5. `health_penalty_ms`
6. `last_success_at`
7. `last_error_at`

### 6.3 最早可执行时间

每个候选渠道计算：

`ready_at = max(gcra_next_time, cooldown_until, inflight_gate_time) + health_penalty - sticky_bias`

解释：

1. `gcra_next_time` 表示 rpm 额度允许的最早发送时间
2. `cooldown_until` 表示渠道故障冷却截止
3. `inflight_gate_time` 表示并发槽位释放前的最早时间
4. `health_penalty` 表示近期错误率高时的惩罚
5. `sticky_bias` 只是轻量偏置，不能压过额度约束

### 6.4 选路顺序

1. 先按 `model_routes.priority` 分层
2. 在当前最高可用层里选 `ready_at` 最小的渠道
3. 通过 Redis Lua 一次性完成：
   - 读取状态
   - 计算 `dispatch_at`
   - 更新 `tat`
   - 占用 inflight lease
   - 返回调度结果

### 6.5 为什么必须用预约，不是 round robin

因为 `1000 rpm` 的真实含义是约每 `60ms` 只能发 1 个请求。

如果 10k 人同时到达，系统不能把它们立刻轮到 50 个渠道上“碰碰运气”。
必须把未来时间槽先预约出来，再按预约时间平滑出队。

## 7. 交付模型

### 7.1 同步桥接

仅在满足以下条件时启用：

1. `eta <= sync_hold_ms`
2. 客户端协议允许同步等待
3. 请求不是长流式高成本任务

此时 Admission 可以阻塞等待 worker 结果，再返回标准 OpenAI / Claude / Gemini 响应。

### 7.2 异步受理

如果 `eta > sync_hold_ms`，立即返回：

- `202 Accepted`
- `request_id`
- `estimated_dispatch_at`
- `estimated_complete_at`

再通过：

1. `GET /v1/requests/{id}`
2. `GET /v1/requests/{id}/events`
3. webhook

交付最终结果。

### 7.3 强保证的关键点

只要 job 已被 durable accepted：

1. 不丢
2. 可查
3. 可重试
4. 可超时
5. 可最终完成或被系统明确标记失败原因

## 8. 失败与重试策略

### 8.1 可重试失败

包括：

1. 429
2. 5xx
3. 网络超时
4. 连接失败

处理方式：

1. 增加 `health_penalty`
2. 必要时进入 `cooldown`
3. 将 job 重新放回队列
4. 若当前 alias 全部不可行，再进入现有 `fallback_model`

### 8.2 不可重试失败

包括：

1. 请求格式错误
2. 权限错误
3. 模型或工具参数非法
4. 已发生外部副作用且不可重放

直接结束 job，不进入重试。

### 8.3 流式请求

一旦 stream 已开始输出，透明重试边界就消失。

处理规则：

1. 流式请求单独队列
2. 单独配额池
3. 默认 async-only
4. 不与短请求共享同一保证模型

## 9. 数据模型扩展

### 9.1 channels 新增字段

1. `rpm_limit`
2. `max_inflight`
3. `safety_factor`
4. `enabled_for_async`
5. `dispatch_weight`

### 9.2 system_settings 新增字段

1. `gateway.queue_ttl_s`
2. `gateway.sync_hold_ms`
3. `gateway.retry_reserve_ratio`
4. `gateway.backlog_cap`
5. `gateway.dispatcher_workers`
6. `gateway.stream_queue_enabled`

### 9.3 新增运行时对象

Redis：

1. `job:{id}`
2. `queue:model:{alias}`
3. `quota:channel:{name}`
4. `lease:job:{id}`
5. `result:{id}`

## 10. 对当前仓库的接入方案

### 10.1 请求入队

位置：`internal/transport/httpserver/gateway_handlers.go`

改造：

1. `executeGatewayRequest()` 不再默认直调 `deps.ExecuteGateway`
2. 改成：
   - decode
   - build job
   - admission
   - sync wait 或 202 accepted

### 10.2 调度执行

位置：`internal/usecase/gateway_service.go`

改造：

1. 现有 `GatewayService` 退化成“已分配渠道后的执行器”
2. 新增 `DispatchService`
3. `DispatchService` 负责：
   - 候选渠道计算
   - Quota Manager 预约
   - 选择最终 route
   - 调用 `GatewayService` 执行

### 10.3 运行时状态

位置：`internal/store/sqlite/routing_store.go`

改造：

1. SQLite 只保留低频控制面配置
2. Redis 承担热状态
3. 当前 sticky / cooldown / cursor 语义迁移为调度器状态的一部分

### 10.4 配置面

位置：

1. `internal/store/sqlite/admin_store.go`
2. `internal/app/app.go`
3. `web/src/app/(console)/settings/settings-client.tsx`

改造：

1. 暴露渠道配额参数
2. 暴露队列参数
3. 暴露调度器开关

### 10.5 结果回传

位置：

1. `router.go` 新增 `/v1/requests/{id}`
2. `gateway_handlers.go` 新增状态查询和事件流处理

## 11. SLA 计算模型

总能力：

- 50 渠道 × 1000 rpm = 50000 rpm ≈ 833 rps

10k 突发排空时间近似：

- `10000 / 833 ≈ 12s`

若预留 10% 给重试预算：

- 有效 rps ≈ `750`
- 排空时间 ≈ `13.3s`

因此：

1. 若同步等待预算小于 13 到 20 秒，不能承诺全同步成功
2. 若允许异步 accepted，再最终回传，可以对 accepted 请求做强承诺

## 12. 边界与风险

1. 如果上游还有 TPM 限制，而请求没有可预估 token 上限，则只能做“高概率”而不是强保证
2. 如果客户端必须完全兼容现有同步 OpenAI SDK 且不可接受 202，则这个需求本身无法成立
3. 如果坚持不新增 Redis，只靠 SQLite，可以做单机排队版，但不能对 10k 突发做强保证
4. 如果请求存在工具调用外部副作用，必须引入幂等执行令牌，否则重试会重复副作用

## 13. 实施阶段

### 阶段 A，控制面补齐

1. channels / settings 扩字段
2. 管理台补配置表单
3. dashboard 补排队和配额视图

#### 阶段 A QA

1. 工具：`go test ./...`
2. 操作：运行后端测试，确认 schema 变更、settings 读写、dashboard 聚合未破坏现有后端行为。
3. 预期结果：测试全部通过。
4. 工具：`pnpm build`，目录 `web/`
5. 操作：构建前端，确认 settings 和 dashboard 新字段类型、页面渲染均通过。
6. 预期结果：构建成功。
7. 工具：HTTP 请求或浏览器手动验证。
8. 操作：
   - 读取 `/api/admin/settings`
   - 修改新增调度参数
   - 打开 dashboard
9. 预期结果：
   - 新增字段可以读写
   - dashboard 能展示配额与排队相关指标占位或真实值

### 阶段 B，Admission + 202 受理

1. 建立 job 模型
2. 建立状态查询接口
3. 打通 durable accepted

#### 阶段 B QA

1. 工具：`go test ./...`
2. 操作：运行后端测试，覆盖 admission、幂等键、状态查询、重复提交等路径。
3. 预期结果：测试全部通过。
4. 工具：HTTP 请求。
5. 操作：
   - 提交同一幂等键请求两次
   - 查询 `/v1/requests/{id}`
   - 提交一个超出同步预算的请求
6. 预期结果：
   - 第一次返回 accepted
   - 第二次命中幂等，不产生重复 job
   - 状态查询可用
   - 超预算请求返回 `202 + request_id`

### 阶段 C，Redis Quota Manager

1. 建立配额状态
2. 建立 Lua 原子预约
3. 打通 dispatch_at 计算

#### 阶段 C QA

1. 工具：`go test ./...`
2. 操作：运行后端测试，覆盖 quota 预约、并发竞争、cooldown、健康惩罚、sticky bias。
3. 预期结果：测试全部通过。
4. 工具：Redis 集成测试或脚本。
5. 操作：并发模拟多个 worker 同时预约同一渠道未来槽位。
6. 预期结果：
   - 不出现同一时间槽被重复预约
   - `rpm_limit` 与 `max_inflight` 不被突破
   - `dispatch_at` 单调递增或符合回退逻辑

### 阶段 D，Dispatcher Workers

1. 出队
2. 调用执行器
3. 写回结果
4. 重试 / fallback

#### 阶段 D QA

1. 工具：`go test ./...`
2. 操作：运行后端测试，覆盖 worker lease、崩溃恢复、重试、fallback、结果写回。
3. 预期结果：测试全部通过。
4. 工具：集成测试或本地多 worker 启动脚本。
5. 操作：
   - 启动多个 worker
   - 人为制造一个 worker 中途退出
   - 模拟 429/5xx 与成功回退
6. 预期结果：
   - job 不丢失
   - lease 过期后可被重新接管
   - 可重试 job 能重排并最终完成或明确失败

### 阶段 E，同步桥接与异步交付

1. 短任务同步桥接
2. 长任务 202 + polling/SSE/webhook

#### 阶段 E QA

1. 工具：`go test ./...`
2. 操作：运行后端测试，覆盖同步桥接、异步轮询、SSE、webhook 交付。
3. 预期结果：测试全部通过。
4. 工具：HTTP 请求、SSE 客户端、webhook mock。
5. 操作：
   - 提交一个可在 `sync_hold_ms` 内完成的请求
   - 提交一个超预算请求
   - 轮询或订阅 SSE 获取结果
   - 检查 webhook 回调
6. 预期结果：
   - 短任务走同步返回
   - 长任务返回 `202`
   - 最终结果可通过 polling/SSE/webhook 获取
   - 返回内容与原协议兼容

## 14. 验证标准

### 功能验证

1. 10k 请求同时 accepted 后，无丢 job
2. 所有 accepted job 都可查状态
3. 失败后可重试并最终完成或明确失败

### 调度验证

1. 单渠道不会突破 `rpm_limit`
2. 单渠道不会突破 `max_inflight`
3. priority 生效
4. sticky 只做 bias，不会压垮配额

### 压测验证

1. 50 渠道、总 50k rpm 条件下，10k burst 可全部 accepted
2. accepted job 在预算窗口内全部完成
3. 系统恢复后不会重复执行已完成 job
