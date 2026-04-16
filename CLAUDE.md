# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## UI 与设计约束
- 做任何 UI 或视觉相关改动前，先读 `DESIGN.md`。
- 该仓库的视觉方向、字体、颜色、间距、布局密度都以 `DESIGN.md` 为准。

## 仓库结构
- 这是一个双端仓库，不是共享工具链的 monorepo。
- 仓库根目录是 Go 后端，入口在 `cmd/api/main.go`。
- 后端真实装配入口在 `internal/app/app.go`，负责加载配置、初始化日志、打开 SQLite、执行迁移、构造 provider 与路由依赖。
- 管理后台前端在 `web/`，是 Next.js App Router 应用。控制台页面集中在 `web/src/app/(console)`。
- `api.exe`、`data/opencrab.db`、`web/.next/` 都是产物或本地运行数据，不是源码真相。

## 常用命令
### 后端
- 安装依赖后直接运行后端：`go run ./cmd/api`
- 运行全部后端测试：`go test ./...`
- 运行单个包测试：`go test ./internal/config`
- 运行单个测试用例：`go test ./internal/config -run TestValidate`

### 前端
在 `web/` 目录下执行：
- 开发模式：`pnpm dev`
- 生产构建：`pnpm build`
- 本地启动生产包：`pnpm start`
- Lint：`pnpm lint`

### Docker
- 整体启动：`docker compose up -d --build`
- 查看服务日志：`docker compose logs api`
- 查看前端日志：`docker compose logs web`

## 配置与环境变量
### 后端
后端配置完全来自环境变量，关键项：
- `OPENCRAB_APP_NAME`
- `OPENCRAB_ENV`
- `OPENCRAB_HTTP_ADDR`
- `OPENCRAB_DB_PATH`
- `OPENCRAB_UPSTREAM_TLS_INSECURE_SKIP_VERIFY`

`OPENCRAB_ENV` 仅支持：`development`、`test`、`production`。

### 前端
- 前端服务端拉管理接口使用 `OPENCRAB_ADMIN_API_BASE`。
- 本地默认值是 `http://127.0.0.1:8080`。
- Docker 中必须设为 `http://api:8080`，容器内使用 `localhost` 会导致服务端渲染请求失败。

## 后端架构
### 启动链路
- `cmd/api/main.go` 只负责创建 `app.App` 并启动服务。
- `internal/app/app.go` 是后端装配中心：
  - 读取并校验配置
  - 初始化结构化日志
  - 打开 SQLite 数据库
  - 自动执行 `internal/store/sqlite/migrations/*.sql`
  - 构造 `http.Client`
	- 创建 rate limiter、channel tester 与多 provider executor
  - 通过 `httpserver.Dependencies` 注入路由层

### 路由与接口
- 路由定义集中在 `internal/transport/httpserver/router.go`。
- 健康检查接口：`/healthz`、`/readyz`
- 管理接口统一挂在 `/api/admin/*`，覆盖 channels、models、model-routes、api-keys、logs、settings。
- 代理接口覆盖 `POST /v1/chat/completions`、`POST /v1/messages`、`POST /v1beta/models/{model}:generateContent`、`POST /v1beta/models/{model}:streamGenerateContent`。

### 数据与分层
- 持久化只有 SQLite。
- SQLite 相关实现集中在 `internal/store/sqlite`。
- migration 文件当前在：
  - `0001_initial.sql`
  - `0002_request_logs_details.sql`
  - `0003_request_logs_usage.sql`
- `internal/domain` 放领域对象和输入输出结构。
- `internal/usecase/usecase.go` 目前基本为空，不要假设仓库已经形成完整 usecase 层。当前实际在用的 usecase 逻辑包括 rate limiter 与 `GatewayService` 的运行时路由。

### 当前运行时事实
- 实际代理转发按 `models`、`model_routes`、`gateway.routing_strategy`、`routing_cursors` 做真实运行时路由。
- `POST /v1/chat/completions` 会先做 invocation bucket 分段，再做 priority 分层，并在层内应用顺序或轮询策略。
- 当前实现已接入 cooldown 过滤、`fallback_model` alias 重入、sticky routing 与运行时决策日志。
- `model_routes.priority` 是线上真实请求路由策略的一部分，不只是后台展示顺序。
- API Key 只在创建时返回一次明文，数据库里保存的是哈希值。

## 前端架构
### 页面壳层
- 根布局在 `web/src/app/layout.tsx`，统一挂载主题与 i18n provider，并加载 Geist Sans / Geist Mono。
- 控制台布局在 `web/src/app/(console)/layout.tsx`，统一套左侧导航、顶部栏和主内容滚动容器。
- 导航定义在 `web/src/lib/navigation.ts`，当前控制台入口包括：`/`、`/channels`、`/models`、`/api-keys`、`/logs`、`/settings`。

### 数据流
- 服务端页面直接通过 `web/src/lib/admin-api-server.ts` 调后端管理接口，默认 `cache: "no-store"`。
- 浏览器端的增删改请求走 `web/src/app/api/admin/[...path]/route.ts`，由 Next.js Route Handler 反向代理到后端。
- 控制台页面普遍依赖真实后端数据，`web/src/lib/mock/console-data.ts` 不应再回填业务 mock 数据。

### 页面组织方式
- `web/src/app/(console)` 下每个页面通常是 Server Component 负责取数。
- 交互复杂的部分下沉到 client component，例如 models、channels、api-keys、settings 下的 `*-client.tsx` 或表单组件。
- 页面大量使用共享布局与展示组件，主要集中在：
  - `web/src/components/layout`
  - `web/src/components/shared`
  - `web/src/components/ui`

### 动态渲染
- 多个控制台页面显式导出 `dynamic = "force-dynamic"`。
- 除非同步重构后端取数与缓存策略，否则不要随意改回静态或默认缓存模式。

## 开发与验证约定
- 后端改动优先跑 `go test ./...`。
- 前端或全栈改动优先在 `web/` 跑 `pnpm build`，它比 `pnpm lint` 更容易暴露类型错误和服务端渲染失败。
- Docker 相关问题优先检查：
  - `OPENCRAB_ADMIN_API_BASE` 是否正确
  - `docker compose logs api`
  - `docker compose logs web`
- 做 UI 改动时，既要遵守 `DESIGN.md`，也要注意控制台是高密度、黑白高对比、左侧导航加顶部状态栏的固定设计方向。
