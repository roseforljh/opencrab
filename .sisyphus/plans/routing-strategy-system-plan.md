# OpenCrab 完整路由策略系统方案

## 1. 目标

为 OpenCrab 设计一套完整、可解释、可运营、可扩展的路由策略系统。

这套系统的设计原则来源于两类参考：

1. 基础调度语义参考 CLIProxyAPI
   - 明确的选择器语义
   - 清晰的优先级层
   - 调度、重试、粘性分层
2. 管理能力与运营暴露参考 NewAPI
   - 配置可见
   - 运维能力可见
   - 健康、禁用、日志、测试有后台入口

目标不是做“最复杂的调度器”，而是做一套：

1. 用户能理解
2. 行为可预测
3. 线上能解释
4. 后续能逐层增强

## 2. 系统边界

本方案覆盖 8 层：

1. 协议入口层
2. 模型解析层
3. 候选过滤层
4. 优先级分层
5. 层内分配层
6. 失败处置层
7. 状态持久化层
8. 管理与观测层

本方案不覆盖：

1. 商业化计费调度
2. 多租户隔离调度
3. 实时成本优化
4. 跨实例全局公平
5. 基于历史统计自动重排

## 3. 设计目标

路由策略系统必须满足以下 10 条：

1. 对同一模型多渠道有明确选路规则
2. 顺序与轮询是显式策略，不隐含在代码路径里
3. 优先级是严格分层，不跨层混轮
4. 协议偏好由 `invocation_mode` 表达
5. 失败处理和路由选择解耦
6. 粘性路由是独立层，不污染基础策略
7. 健康避让是独立层，不污染基础策略
8. 每次请求都能解释“为什么选中了这条渠道”
9. 设置改动可以立即影响运行时
10. 整套系统可以逐步落地，不要求一次做满所有高级能力

## 4. 核心概念

### 4.1 模型别名

对外请求只认模型别名，例如 `gpt-5.4`。

### 4.2 路由规则

每条 `model_route` 描述：

1. 这个别名可以走哪个渠道
2. 当前渠道上的 `upstream_model`
3. 优先级是多少
4. 更适合哪个协议入口

### 4.3 invocation_mode

`invocation_mode` 的职责只有一个：

**表达这条路由对某种外部协议入口的偏好。**

它不是：

1. 路由策略本身
2. 健康状态
3. 权重
4. 优先级

### 4.4 路由策略

`routing_strategy` 只定义“在同一候选层里如何挑第一条开始尝试”：

1. `sequential`
2. `round_robin`
3. 未来可扩展：`weighted_random`

### 4.5 健康状态

健康状态不是 route 字段，而是运行时状态，包括：

1. enabled / disabled
2. cooldown until
3. last failure
4. recent success/failure window

### 4.6 粘性状态

粘性状态单独存储，不写回 route 配置。它表示：

1. 某个 session / user / conversation
2. 最近绑定到哪个 route/channel

## 5. 路由决策总流程

每次请求的选路流程必须固定为：

1. 解析协议入口
2. 解析模型别名
3. 拉取全部启用候选路由
4. 根据 `invocation_mode` 拆成候选段
5. 在每个候选段内按 `priority` 分层
6. 对当前最高可用层应用基础策略
7. 若命中粘性绑定且仍可用，可短路基础策略
8. 发送上游请求
9. 根据结果决定：
   - 成功返回
   - 同层 fallback
   - 下一层 fallback
   - 下一候选段 fallback
   - 直接终止
10. 更新游标、粘性、健康、日志

## 6. 协议入口层

协议入口包括：

1. OpenAI 兼容
2. Claude 原生
3. Gemini 原生

每个请求必须带上 `GatewayRequest.Protocol`。

协议的作用：

1. 影响 `invocation_mode` 候选段排序
2. 影响轮询游标隔离维度
3. 影响日志解释

## 7. 候选过滤层

候选过滤必须在策略前执行。

过滤条件分四类：

1. 静态过滤
   - channel enabled
   - route 存在
   - model alias 匹配
2. 协议偏好过滤
   - matched
   - neutral
   - mismatched
3. 运行时可用性过滤
   - provider executor 存在
   - 渠道未被禁用
   - 不在 cooldown
4. 未来能力过滤
   - 是否支持流式
   - 是否支持 tools
   - 是否支持多模态

## 8. invocation 候选段语义

当前固定 3 段：

1. `matched`
2. `neutral`
3. `mismatched`

规则：

1. 精确匹配优先
2. 通用候选次之
3. 错配候选最后兜底

说明：

1. `matched` 不是唯一可用集
2. `mismatched` 不是绝对禁用，只是最后兜底

## 9. 优先级分层

`priority` 是严格分层语义，不是权重区间。

规则：

1. 只在当前最高可用优先级层内分配第一条路由
2. 高优先级层失败完，才进入下一层
3. 低优先级永远不与高优先级混轮

这是整个系统必须守住的硬规则。

## 10. 基础调度策略层

### 10.1 sequential

语义：

1. 保持稳定顺序
2. 当前层从第一条开始尝试
3. 可重试失败才继续下一条
4. 当前层耗尽后进下一层

适合场景：

1. 希望主渠道优先、备用渠道兜底
2. 不追求分流均衡

### 10.2 round_robin

语义：

1. 仅在当前层内轮换起始路由
2. 当前请求失败后仍按层内顺序 fallback
3. 当前层耗尽才进下一层

适合场景：

1. 同模型多个同级渠道分摊流量
2. 希望保持公平且可解释

### 10.3 weighted_random

这是下一阶段能力，不在当前实现里，但方案必须预留。

语义：

1. 仅在当前层内加权随机
2. 不跨优先级层
3. 权重只影响起始命中概率，不改 fallback 顺序

## 11. 粘性路由层

粘性路由是可选层，放在基础调度之前。

输入建议：

1. `X-Session-ID`
2. `conversation_id`
3. `user_id`

行为：

1. 若存在绑定且绑定 route 当前可用，则直接命中
2. 若绑定 route 已不可用，则回退到基础调度
3. 成功后更新绑定

注意：

1. 粘性只影响“先尝试谁”
2. 不改变 `priority` 语义
3. 不跨协议共享绑定

## 12. 失败处置层

失败分为四类：

1. 不可重试失败
2. 可重试失败
3. 流式已开始失败
4. 运行时不可用

规则：

### 12.1 不可重试失败

直接结束请求。

### 12.2 可重试失败

当前层内继续下一条。

### 12.3 当前层耗尽

进入下一优先级层。

### 12.4 当前候选段耗尽

进入下一 invocation 段。

### 12.5 流式已开始

不再切换。

### 12.6 fallback_model 语义

`fallback_model` 必须定义为：

1. **当前 route 全部正常选路路径耗尽后才触发的“模型别名重入”机制**
2. 它不是同层补链
3. 它不是直接跳过路由系统去请求某个固定 provider

具体规则：

1. 一条 route 若配置了 `fallback_model`，只有在以下条件全部满足时才触发：
   - 当前请求发生可重试失败
   - 当前 route 所在优先级层已耗尽
   - 当前 invocation bucket 已耗尽
   - 当前模型别名的其它普通候选都已无法命中成功结果
2. 触发后会以新的 `model_alias=fallback_model` 重新进入完整路由流程：
   - 重新做 `invocation_mode` 分段
   - 重新做 `priority` 分层
   - 重新应用当前 `routing_strategy`
3. `fallback_model` 仍受当前请求的 `protocol` 约束，不会绕过协议偏好逻辑。
4. `fallback_model` 只允许单向链式跳转，不允许环。

### 12.7 fallback 回环终止规则

必须写死以下保护：

1. 单次请求最多允许经过 3 次模型别名重入
2. 若出现别名环，例如 `A -> B -> A`，立即终止并返回错误
3. 请求上下文里必须保存 `visited_model_aliases`
4. 日志中必须记录 `fallback_chain`

## 13. 健康与避让层

健康不是策略本身，而是策略前置过滤。

第一阶段建议只做最小能力：

1. 渠道手动禁用
2. 基于失败状态进入 cooldown
3. cooldown 结束后自动恢复候选资格

第二阶段可以扩展：

1. 自动禁用
2. 定时探测
3. 响应时间阈值
4. 失败率窗口

## 14. 状态持久化层

### 14.1 轮询游标

使用 `routing_cursors` 表。

键格式：

`model_alias + protocol + invocation_bucket + priority`

### 14.2 粘性绑定

建议新增独立表，例如：

`routing_affinity_bindings`

字段：

1. affinity_key
2. model_alias
3. protocol
4. route_identity
5. updated_at

### 14.3 健康状态

建议新增独立表，例如：

`routing_runtime_states`

字段：

1. route_identity
2. cooldown_until
3. last_error
4. updated_at

## 15. 管理能力系统

完整路由策略系统必须有三类管理入口：

### 15.1 全局策略设置

包括：

1. `gateway.routing_strategy`
2. 是否启用粘性
3. 粘性 key 来源
4. cooldown 默认时长
5. 自动禁用阈值

### 15.2 模型路由管理

每条 route 至少要能编辑：

1. alias
2. channel
3. upstream_model
4. invocation_mode
5. priority
6. fallback_model

### 15.3 运维观察面板

至少展示：

1. 最近命中渠道分布
2. 最近 fallback 次数
3. 最近 cooldown 事件
4. 当前游标状态
5. 当前粘性绑定量

## 16. 日志与可观测性

每次 attempt 必须能解释原因。

至少记录：

1. `routing_strategy`
2. `invocation_bucket`
3. `priority_tier`
4. `candidate_count`
5. `selected_index`
6. `selected_channel`
7. `decision_reason`
8. `fallback_stage`

建议后续加指标：

1. route hit count
2. route fallback count
3. cooldown enter count
4. sticky hit ratio

## 17. OpenCrab 当前实现与完整系统差距

当前已经有：

1. `sequential`
2. `round_robin`
3. `gateway.routing_strategy`
4. `routing_cursors`
5. `invocation_mode` 候选段
6. attempt 日志核心字段

当前还缺：

1. 粘性路由层
2. 显式健康状态层
3. fallback_model 执行链接入
4. 运维面板级展示
5. 可配置的失败分类规则
6. 完整模型路由编辑流

## 18. 实施阶段建议

### Phase 1：基础策略收口

1. 把现有顺序/轮询实现收口成稳定契约
2. 补完整模型路由编辑
3. 补决策原因日志

### Phase 2：健康避让

1. 增加 runtime state 存储
2. 加 cooldown 过滤
3. 加恢复逻辑

### Phase 3：fallback_model 执行链

1. 按第 12.6 节定义，把 `fallback_model` 固定为“当前模型普通候选耗尽后的 alias 重入”
2. 重入后必须重新经过 `invocation_mode` 分段与 `priority` 分层
3. 按第 12.7 节补 `visited_model_aliases` 与最大重入次数保护
4. 接入运行时并补日志里的 `fallback_chain`

### Phase 4：粘性路由

1. 增加 affinity 存储
2. 读取 session key
3. 命中绑定短路基础策略

### Phase 5：运维可观测性

1. 路由面板
2. fallback 面板
3. cooldown 面板

## 19. 可执行 QA 清单

### Phase 1 QA：基础策略收口

工具：`go test`

1. 执行：`go test ./internal/usecase -run TestGatewayServicePrefersInvocationModeMatch`
2. 预期：Claude 请求优先命中 `invocation_mode=claude` 候选
3. 执行：`go test ./internal/usecase -run TestGatewayServiceRoundRobinRotatesWithinPriorityTier`
4. 预期：同层候选会从当前游标指定位置开始命中

### Phase 2 QA：健康避让

工具：`go test` + SQLite 查询 + 后台日志

1. 执行：`go test ./internal/usecase -run TestGatewayServiceSkipsCooldownRoute`
2. 若使用集成验证，先将某 route 的 `cooldown_until` 写入未来时间
3. 发起一次相同模型请求
4. 预期：响应头 `X-Opencrab-Channel` 不等于该 cooldown route 对应渠道
5. 查询 `request_logs.details`，应出现 `decision_reason` 或 `skip_reason=cooldown`

### Phase 3 QA：fallback_model 执行链

工具：`go test` + HTTP 请求验证

1. 构造主模型 `A` 的所有 route 都返回可重试失败
2. 配置 `A -> B` 的 `fallback_model`
3. 发起一次请求
4. 预期：请求最终以 `B` 的候选成功返回
5. 响应头中的 `X-Opencrab-Channel` 应来自 `B` 命中的渠道
6. 日志中应出现 `fallback_chain=[A,B]`

### Phase 4 QA：粘性路由

工具：HTTP 请求 + SQLite 查询 + 日志

1. 使用相同 `X-Session-ID` 连续请求两次
2. 第一次记录命中渠道
3. 第二次请求预期命中同一渠道
4. 查询粘性绑定表，确认存在该 session 与 route 的绑定记录
5. 若该渠道被禁用或进入 cooldown，下一次请求应自动回退到可用候选
6. 日志中应显示 `sticky_hit=true/false` 或等价决策字段

### Phase 5 QA：运维可观测性

工具：`go test` + `pnpm build` + 控制台页面

1. 执行：`go test ./...`
2. 执行：`pnpm build`（目录：`web/`）
3. 打开设置页，检查全局策略、粘性、健康参数是否可见
4. 打开日志页，检查是否能看到 `routing_strategy`、`invocation_bucket`、`priority_tier`、`fallback_chain`
5. 若有路由面板，检查命中分布、fallback 次数、cooldown 事件是否可见
6. 预期：控制台展示与 `request_logs.details` / 统计接口一致

### QA-1 基础策略

1. 运行 `go test ./internal/usecase`
2. 验证顺序模式和轮询模式都通过

### QA-2 设置生效

1. 修改 `gateway.routing_strategy`
2. 下一次请求立即生效
3. 无需重启服务

### QA-3 轮询游标

1. 连续请求同模型同协议
2. 观察命中渠道轮换
3. 重启服务后游标继续生效

### QA-4 协议偏好

1. 用 OpenAI / Claude / Gemini 三个入口分别请求同模型
2. 观察先命中的 invocation bucket 是否符合预期

### QA-5 fallback

1. 人为让第一条候选返回可重试失败
2. 观察是否进入下一条
3. 人为让第一条返回不可重试失败
4. 观察是否直接结束

### QA-6 管理台

1. `pnpm build`
2. 设置页能改策略
3. 模型页能显示 invocation_mode

## 20. 非目标

本方案当前不包括：

1. 多实例全局一致轮询
2. 实时延迟打分调度
3. 成本优化调度
4. 自动熔断与半开恢复
5. AI 自学习重排

## 21. 审查要点

你审查这份完整系统方案时，只看 6 件事：

1. 是否把“基础策略”“健康”“粘性”“失败”分层了
2. 是否守住 priority 先分层再调度
3. 是否让路由行为可解释
4. 是否避免把系统一次做成过重调度器
5. 是否给后续 Phase 留出清晰边界
6. 是否能和当前 OpenCrab 代码平滑衔接
