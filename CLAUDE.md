# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

- 这是一个 **Go + React** 的个人 API 聚合/中转项目。
- 后端入口是 `main.go`，使用 **Gin** 提供两类能力：
  - `/api/*`：控制台与系统管理接口
  - `/v1/*`、`/v1beta/*`、`/pg/*`：面向上游/兼容协议的 relay 转发接口
- 前端位于 `web/`，由 **Vite + React 18 + react-router-dom** 构建，最终产物会被后端通过 `embed` 嵌入并直接提供。

## 常用命令

### 后端开发

- 启动后端开发服务：`go run main.go`
- 运行全部 Go 测试：`go test ./...`
- 运行单个包测试：`go test ./controller -v`
- 运行单个测试：`go test ./controller -run TestUserPinLogin -v`

### 前端开发（在 `web/` 目录）

- 安装依赖：`bun install`
- 启动开发服务器：`bun run dev`
- 构建前端：`bun run build`
- 代码格式检查：`bun run lint`
- 自动修复格式：`bun run lint:fix`
- ESLint 检查：`bun run eslint`
- ESLint 自动修复：`bun run eslint:fix`
- i18n 提取：`bun run i18n:extract`
- i18n 状态检查：`bun run i18n:status`
- i18n 同步：`bun run i18n:sync`
- i18n lint：`bun run i18n:lint`

### 整体构建 / 运行

- Docker 启动：`docker compose up -d --build`
- `make` 会先构建前端再启动后端：`make`

## 构建与开发注意点

- `main.go` 通过 `//go:embed web/dist` 嵌入前端静态资源；如果改了前端并希望由 Go 服务直接提供最新页面，需要先重新构建 `web/dist`。
- `web/vite.config.js` 将 `/api`、`/mj`、`/pg` 代理到 `http://localhost:5946`，所以前端本地开发通常需要同时启动后端。
- `makefile` 的 `build-frontend` 目标会读取根目录 `VERSION` 文件并注入 `VITE_REACT_APP_VERSION=$(cat VERSION)`；当前仓库里未发现该文件，若直接使用 `make` 失败，优先检查这一点。
- README 里的前端安装示例是 `npm install --legacy-peer-deps`，但仓库当前脚本与 `makefile` 都以 **bun** 为主。
- 当前仓库未看到独立的前端测试脚本；测试主要是 Go 侧测试。

## 后端架构

### 启动流程

- `main.go`
  - 调用 `InitResources()`：加载 `.env`、初始化环境变量、logger、HTTP client、token encoder、数据库。
  - 初始化缓存、配置热同步、看板统计、渠道自动测试、Codex credential 自动刷新等后台任务。
  - 创建 Gin server，挂载通用中间件、session、i18n、日志。
  - 调用 `router.SetRouter(server, buildFS, indexPage)` 组装 API、relay 和前端路由。

### 分层方式

- `router/`：路由装配层
  - `main.go` 负责组合 API / relay / web 路由。
  - `api-router.go` 是控制台与管理接口。
  - `relay-router.go` 是 OpenAI / Claude / Gemini 兼容转发入口。
- `controller/`：HTTP handler 层，处理参数、权限后的业务入口。
- `service/`：更偏可复用业务逻辑与外部服务集成，例如 HTTP client、token encoder、credential refresh 等。
- `model/`：GORM 模型、数据库初始化、迁移、缓存、配置持久化。
- `middleware/`：认证、限流、分发、性能检查、日志、CORS、请求体清理等横切逻辑。
- `relay/`：协议/渠道适配核心。这里是“中转能力”的关键实现，按不同 channel 或协议格式拆分。
- `common/`、`constant/`、`types/`、`dto/`：全局配置、常量、共享类型、请求/响应结构。

### 路由与业务边界

- `/api/*` 主要用于控制台后台：
  - setup/status
  - user / option / performance
  - channel / token / models
- `/v1/*`、`/v1beta/*`、`/pg/*` 主要用于 API relay：
  - `/v1/chat/completions`、`/v1/messages`、`/v1/responses` 等统一进入 `controller.Relay(...)`
  - 实际协议差异通过 `types.RelayFormat*` 和 `relay/` 内部适配器处理
- `router/main.go` 会根据 `FRONTEND_BASE_URL` 决定：
  - 直接由当前 Go 服务托管嵌入后的前端
  - 或把前端请求重定向到外部前端地址

### 数据与配置

- `model/main.go` 负责数据库选择与初始化：
  - 默认 SQLite
  - 也支持 MySQL / PostgreSQL（通过 `SQL_DSN`）
- `LOG_SQL_DSN` 可单独配置日志数据库；否则日志与主库共用。
- 启动时会执行 migration，并在首次启动时自动创建 root 账户/初始化 setup 状态。
- 配置热更新通过 `model.SyncOptions(...)` 后台同步，不要假设所有配置只在启动时加载一次。

### 认证与权限

- 控制台侧大量依赖 `middleware.UserAuth()` / `AdminAuth()` / `RootAuth()`。
- relay 侧主要依赖 `middleware.TokenAuth()`，与控制台 session 登录是两套入口。
- 登录相关接口可从 `router/api-router.go` 的 `/api/user/pin-login` 开始追踪。

### 与“渠道 / 模型 / 中转”最相关的代码

- 渠道管理接口：`router/api-router.go`
- relay 总入口：`router/relay-router.go`
- relay handler：`controller.Relay(...)`
- 各渠道/协议适配：`relay/`
- 渠道、模型、缓存、迁移等数据逻辑：`model/`

## 前端架构

### 启动与顶层结构

- 前端入口是 `web/src/index.jsx`。
- 顶层 Provider 顺序是：`StatusProvider` → `UserProvider` → `BrowserRouter` → `ThemeProvider` → `Semi LocaleProvider` → `PageLayout`。
- `PageLayout` 会在启动时拉取 `/api/status`，同步系统状态、标题、logo，并决定是否显示控制台侧边栏。
- 实际路由在 `web/src/App.jsx`，当前主流程集中在：
  - `/login`
  - `/setup`
  - `/console/channel`
  - `/console/models`
  - `/console/token`
  - `/console/setting`
  - `/console/personal`

### 前端分层

- `pages/`：路由页入口。
- `components/`：业务组件与布局组件。
  - `components/layout/` 是整体壳层。
  - `components/table/` 下是大量控制台表格与弹窗交互。
  - `components/settings/` 下是系统/个人设置模块。
- `context/` 与 `contexts/`：全局状态上下文。注意仓库里两种命名都存在，改动时不要想当然地只搜一个目录。
- `helpers/`：API 封装、鉴权、渲染与通用前端辅助逻辑。
- `hooks/`：通用与业务 hooks。
- `components/ui/`：通用 UI 基础组件。

### 关键实现特点

- UI 不是单一体系：当前同时混用了 **Semi UI**、`components/ui/` 下的基础组件，以及 `antd` 依赖。
- `web/vite.config.js` 明确把 `src/**/*.js` 按 JSX 处理；因此这个前端里 **`.js` 文件也可能写 JSX**，不要因为扩展名误判。
- API 请求主要通过 `web/src/helpers/api.js` 中的 Axios 实例统一发起：
  - 自动附带 `New-API-User`
  - 对重复 GET 做 in-flight 去重
  - 可通过 `disableDuplicate: true` 跳过去重
  - 有全局 response error interceptor，可通过 `skipErrorHandler: true` 跳过默认错误提示
- 主题层使用 `next-themes`，但样式变量强依赖 Semi 设计令牌；`web/tailwind.config.js` 也把大量颜色映射到了 `--semi-color-*` 变量。
- i18n 配置在 `web/i18next.config.js` 与 `web/src/i18n/`：提取配置声明了多语言 locale，但当前代码与现有翻译资产主要围绕 `zh-CN` 工作，新增语言时不要只改一处。

## 修改代码时的高价值提示

- 若改 relay 行为，通常需要同时检查 `router/relay-router.go`、`controller/`、`relay/`、`middleware/`，因为入口路由、鉴权/分发、协议转换是分散的。
- 若改控制台页面，通常需要同时检查 `pages/` 路由入口、`components/layout/PageLayout.jsx`、相关 `context/`、以及 `helpers/api.js` 的请求形态。
- 渠道、模型、token、设置页都高度依赖后端 `/api/*` 接口命名；前后端联调时优先从对应 router 路由反查 handler。
- 这个仓库近期已有未提交修改：`web/src/components/table/channels/modals/EditChannelModal.jsx`、`web/vite.config.js`。修改前先确认不要覆盖现有工作。
