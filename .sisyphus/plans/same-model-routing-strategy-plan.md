# 同模型多渠道路由策略实施方案

## 1. 目标

为 OpenCrab 增加两种同模型多渠道路由策略：

1. 顺序模式
2. 轮询模式

目标是让同一个对外模型别名在多个可用渠道下，既能保持当前的稳定顺序兜底，也能支持更均衡的请求分摊。

本次只做路由策略，不扩展计费、配额、熔断、复杂健康评分。

## 2. 当前现状

当前运行时行为：

1. `model_routes` 先按 `priority ASC, id ASC` 取候选。
2. 再按 `invocation_mode` 与请求协议做三段重排：精确匹配、通用、错配。
3. `GatewayService.Execute()` 固定从头到尾顺序尝试。
4. 命中第一条成功路由即返回。
5. 仅当错误满足 `Retryable=true` 且 `StreamStarted=false` 时才会尝试下一条。

结论：当前只有“顺序优先”语义，没有轮询。

## 3. 参考结论

参考 `router-for-me/CLIProxyAPI`，其核心行为不是简单随机，而是：

1. 先按最高优先级层筛选候选。
2. 再在该层内执行 `fill-first` 或 `round-robin`。
3. 低优先级只有在更高优先级层没有可用候选时才参与。

对 OpenCrab 来说，最小正确迁移方式是保留 `priority` 的“分层筛选”职责，再在层内引入两种策略。

## 4. 明确语义

### 4.1 顺序模式

定义：

1. **顺序模式保持现网语义，不做行为变更。**
2. 先按 `invocation_mode` 与请求协议做三段重排：
   - 精确匹配
   - 通用候选
   - 错配候选
3. 每段内部保持数据库返回顺序，即 `priority ASC, id ASC`。
4. 从重排后的第一条开始顺序尝试。
5. 第一条成功即返回。
6. 可重试失败才继续下一条；不可重试失败或已开始流式则直接结束。

### 4.2 轮询模式

定义：

1. 仍然先按 `invocation_mode` 与请求协议做三段重排。
2. **只在第一个非空候选段内启用轮询。**
3. 进入该候选段后，再按 `priority` 分层。
4. 只在当前最高优先级层内做轮询。
5. 本次请求的起始路由由轮询状态决定。
6. 若该起始路由失败且允许 fallback，则继续在当前层剩余候选中顺序尝试。
7. 若当前层全部失败，再进入该候选段的下一优先级层；下一层同样按其轮询状态确定起始位置。
8. 当前候选段全部失败后，才进入下一 `invocation_mode` 段继续按同样规则处理。

这意味着：

1. 轮询决定“谁先被尝试”。
2. fallback 决定“当前请求失败后还会不会继续试别的路由”。

## 5. 作用范围

策略作用范围必须明确为：

1. **按模型别名生效**
2. **按请求协议分组生效**
3. **按 invocation_mode 候选段生效**
4. **轮询只在优先级层内生效**

推荐最终轮询键：

`model_alias + protocol + invocation_bucket + priority`

原因：

1. 同一个模型在 OpenAI / Claude / Gemini 入口下，本来就可能命中不同候选段。
2. 不同 invocation 段不应共享一个轮询指针，否则会让错配渠道参与本不该进入的轮转。
3. 不同优先级层不应共享一个轮询指针，否则会破坏优先级语义。

## 6. 数据模型变更

### 6.1 新增系统级路由策略配置

新增系统设置项，例如：

- key: `gateway.routing_strategy`
- value: `sequential` 或 `round_robin`

这是最小改动方案。

原因：

1. 你当前诉求是全局支持两种模式。
2. 如果先把策略挂到每条 `model_route` 上，会把组级概念拆散，配置难理解。

### 6.2 运行时读取路径

必须新增两个运行时依赖接口：

1. `GatewayRoutingStrategyStore`
   - 请求时读取当前策略
   - 返回值：`sequential` 或 `round_robin`
2. `GatewayRoutingCursorStore`
   - 读写轮询游标
   - 仅在 `round_robin` 下使用

`GatewayService` 在每次请求时读取当前策略，而不是只在启动时固化。

原因：

1. 管理端改完设置后，下一次请求立即生效。
2. 不需要重启服务。

### 6.3 保留现有 `invocation_mode`

当前已存在的 `invocation_mode` 继续保留，用来表达：

1. 该路由更适合哪个外部原生入口协议
2. 请求进来后如何重排候选顺序

它不承担“顺序/轮询”策略职责。

## 7. 运行时算法

### 7.1 候选集构建顺序

运行时统一分三步：

1. 取模型全部启用路由
2. 按 `invocation_mode` 与请求协议拆成三段候选：精确匹配、通用、错配
3. 在候选段内部按 `priority` 分层

### 7.2 顺序模式算法

对于每一候选段：

1. 从该段第一条开始
2. 调用成功立即返回
3. 可重试失败则试下一条
4. 不可重试失败或已开始流式则直接结束
5. 当前段耗尽后进入下一段

### 7.3 轮询模式算法

对于每一候选段：

1. 先取该段中的最高优先级层
2. 根据轮询状态取当前层起始索引
3. 从该索引开始环形遍历本层候选
4. 第一条成功即返回
5. 成功后推进该层轮询指针到下一个位置
6. 若本层全部失败，再进入该段下一优先级层
7. 当前段全部失败，再进入下一候选段

### 7.4 轮询推进规则

建议规则：

1. **仅在当前层某条路由被实际尝试时推进起始位置**
2. **若本次请求在当前层成功，则指针推进到成功路由的下一条**
3. **若当前层全部失败，则也推进 1 步，避免下一次永远从同一条开始**

这样能避免热点永远集中在第一条失败路由上。

## 8. 状态持久化方案

轮询状态不能只存在内存里。

最小可落地方案：

1. 新增一张 SQLite 表，例如 `routing_cursors`
2. 字段建议：
   - `route_key`
   - `next_index`
   - `updated_at`

其中 `route_key` 建议格式：

`model_alias + protocol + invocation_bucket + priority`

原因：

1. 进程重启后继续轮询，不会突然全回到第一条。
2. 单机 SQLite 已符合当前项目定位。
3. 多实例一致性不作为本阶段目标。

## 9. 流式与失败语义

必须明确以下边界：

1. 轮询选择发生在首个上游请求发出前。
2. 若某条路由已开始流式返回，再出错，不允许切换到下一条。
3. provider 内部 retry 仍由 provider 自己处理；网关层只处理 provider 最终返回的成功/失败。
4. 网关层轮询与 provider 内部 retry 不耦合。

### 9.1 重试与推进规则

必须写死以下规则：

1. provider 内部 retry 完成后，才把结果返回给网关层。
2. 网关层只根据 provider 的最终成功/失败推进轮询状态。
3. 若某条路由返回不可重试错误，该请求直接结束，不推进到下一条。
4. 若某条路由返回可重试错误，当前请求继续尝试下一条，并在请求完成后按最终命中或层耗尽结果推进指针。

## 10. 管理端与接口变更

本阶段先不做复杂 UI 改造，但必须补最小管理能力：

1. 后端系统设置接口支持读取和更新 `gateway.routing_strategy`
2. 控制台设置页展示路由策略下拉：
   - 顺序
   - 轮询
3. 路由列表页无需新增复杂分组操作，只要保留现有 `priority` 和 `invocation_mode` 展示即可

设置页文案必须明确：

1. 顺序模式保持当前行为
2. 轮询模式只在协议匹配后的候选段内生效
3. 轮询仍然受 `priority` 约束，不会跨优先级平均分流

## 11. 日志与可观测性

需要补充 attempt 日志字段，至少包含：

1. `routing_strategy`
2. `invocation_bucket`
2. `priority_tier`
3. `candidate_count`
4. `selected_index`
5. `selected_channel`

否则后续很难解释“为什么命中了这条渠道”。

## 12. 测试计划

### 12.1 Store / 配置层

1. 新 migration 能正确创建轮询状态表
2. 系统设置可读写 `gateway.routing_strategy`
3. 游标 store 能按 `model + protocol + invocation_bucket + priority` 维度读写

### 12.2 Usecase 层

1. 顺序模式下固定先命中第一条可用路由
2. 顺序模式保持当前 invocation 重排语义，不做行为回归
2. 顺序模式下第一条可重试失败才会走下一条
3. 轮询模式下连续请求会轮流命中同层不同渠道
4. 不同 `protocol` 有各自独立轮询状态
5. 不同 `invocation_bucket` 有各自独立轮询状态
5. 不同 `priority` 层不共享轮询状态
6. 流式开始后失败不会切换路由

### 12.3 HTTP / 集成层

1. OpenAI 入口下相同模型多渠道轮询生效
2. Claude / Gemini 原生入口各自按协议优选后再轮询
3. 管理接口能更新策略并立即影响运行时行为
4. 设置切换后无需重启，下一请求生效

## 12.4 可执行 QA 清单

### QA-1 路由策略默认值

1. 执行：`go test ./internal/store/sqlite -run TestRoutingConfigStoreDefaultsToSequential`
2. 预期：通过，说明未配置时默认是 `sequential`

### QA-2 轮询游标读写

1. 执行：`go test ./internal/store/sqlite -run TestRoutingCursorStoreReadAndAdvance`
2. 预期：通过，说明游标能按 `model + protocol + invocation_bucket + priority` 维度读写

### QA-3 顺序模式保持现网语义

1. 执行：`go test ./internal/usecase -run TestGatewayServicePrefersInvocationModeMatch`
2. 预期：通过，Claude 请求仍优先命中 `invocation_mode=claude` 的候选

### QA-4 轮询模式同层轮转

1. 执行：`go test ./internal/usecase -run TestGatewayServiceRoundRobinRotatesWithinPriorityTier`
2. 预期：通过，连续请求会从当前游标指定的同优先级候选开始命中

### QA-5 轮询模式同层 fallback

1. 执行：`go test ./internal/usecase -run TestGatewayServiceRoundRobinFallsBackWithinTier`
2. 预期：通过，起始候选失败后会在当前层内继续尝试下一条

### QA-6 后端全量回归

1. 执行：`go test ./...`
2. 预期：全部通过，无回归

### QA-7 设置页展示

1. 执行：`pnpm build`（目录：`web/`）
2. 打开设置页，检查“模型路由策略”是否以下拉形式展示
3. 预期：只能选择 `sequential` 或 `round_robin`，不再是自由文本输入

### QA-8 模型页展示

1. 执行：`pnpm build`（目录：`web/`）
2. 打开模型页，检查路由详情区是否展示 `invocation_mode`
3. 预期：详情区能看到调用方式字段，和后端返回值一致

## 13. 实施顺序

建议实施步骤：

1. 原子提交 1：补系统设置项、默认值、读写路径，先加测试
2. 原子提交 2：补轮询状态表与 cursor store，先加测试
3. 原子提交 3：在 `GatewayService` 中接入策略读取，但先保持 `sequential` 现网语义不变，补回归测试
4. 原子提交 4：实现 `round_robin` 选择器与游标推进规则，补用例
5. 原子提交 5：补日志字段与设置页展示，补接口/前端测试
6. 最后全量验证

## 13.1 TDD 节奏

每一步都按以下顺序执行：

1. 先写失败测试
2. 再做最小实现使测试通过
3. 最后重构命名、抽函数、补日志

## 14. 非目标

本阶段明确不做：

1. 跨实例全局公平轮询
2. 按延迟/成功率动态打分
3. 熔断器、半开恢复
4. 按 token 用量做流量分配
5. 每个模型单独配置不同策略

## 15. 审查要点

你审查方案时，建议重点看这 6 件事：

1. 策略是否做成全局系统设置，而不是 route 级字段
2. `priority` 是否仍然保持“先分层，再选路”语义
3. 轮询键是否按 `model + protocol + priority` 维度隔离
4. 流式失败边界是否清晰
5. provider 内部 retry 和网关轮询是否解耦
6. 当前阶段是否坚持最小实现，没有顺手扩成复杂调度系统
