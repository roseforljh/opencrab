# OpenCrab 后端计划

## 1. 后端目标

后端的目标不是做一个“大而全的平台内核”，而是先做一个稳定、可控、易扩展的聚合网关。

首版后端要解决的核心问题：

1. 统一接收请求。
2. 正确路由请求。
3. 正确代理到上游。
4. 正确处理流式返回。
5. 正确记录关键日志。
6. 给前端管理台提供管理 API。

## 2. 后端技术栈

1. 语言：Go
2. Web：优先 `chi`
3. HTTP：标准库 `net/http`
4. 数据库：SQLite
5. 配置：配置文件 + 环境变量覆盖
6. 日志：结构化日志

## 3. 后端目录职责

### `cmd/api`

程序入口，负责启动服务。

### `internal/app`

负责组装应用，把配置、日志、数据库、路由都串起来。

### `internal/config`

负责读取和校验配置。

### `internal/domain`

负责定义核心领域对象和接口。

### `internal/transport/http`

负责 HTTP 路由、中间件、请求与响应结构。

### `internal/usecase`

负责业务编排。

### `internal/provider`

负责不同上游渠道的协议适配。

### `internal/store/sqlite`

负责 SQLite 持久化实现。

### `internal/observability`

负责日志、request id、错误记录。

## 4. 首版后端模块

1. `system`
   - 健康检查
   - 就绪检查
   - 系统设置读取

2. `auth`
   - API Key 校验
   - 管理接口访问控制

3. `provider`
   - OpenAI 兼容 provider 适配
   - 请求与响应转换
   - SSE 处理

4. `model`
   - 模型别名映射
   - 模型到渠道的路由解析

5. `channel`
   - 渠道配置管理
   - 渠道启用/禁用

6. `proxy`
   - 普通代理
   - 流式代理
   - 错误转换

7. `log`
   - 请求日志摘要
   - 查询
   - 脱敏

## 5. 首版数据库对象

1. `channels`
2. `models`
3. `model_routes`
4. `api_keys`
5. `request_logs`
6. `system_settings`

## 6. 后端开发顺序

当前后端计划暂时后置。

只有在用户确认前端管理台满意后，才开始下面顺序：

1. 初始化项目骨架。
2. 做配置系统。
3. 做 SQLite 初始化和 migration。
4. 做健康检查。
5. 打通 OpenAI 兼容 `/v1/chat/completions`。
6. 做普通响应和 SSE 流式响应。
7. 做渠道、模型、路由管理。
8. 做 API Key、限流、日志。
9. 做管理 API。

## 7. 阶段 2 固定边界

阶段 2 只做这些：

1. 首个 provider 固定为 OpenAI 兼容渠道。
2. 只打通 `/v1/chat/completions`。
3. 同时支持普通响应和流式响应。
4. 不做复杂多渠道竞速。
5. 不做复杂多级回退。

## 8. 后端验证重点

1. 服务能正常启动。
2. 配置错误时能明确报错。
3. 数据库能初始化。
4. 普通代理能成功。
5. 流式代理能成功。
6. 请求中断后资源能释放。
7. 日志能记录关键摘要。
8. 密钥和敏感信息不会泄露到日志。
