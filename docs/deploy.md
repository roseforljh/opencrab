# OpenCrab 部署说明

## 1. 本地直接运行

### 后端

```powershell
go run ./cmd/api
```

默认监听：`http://127.0.0.1:8080`

健康检查：

```text
GET /healthz
GET /readyz
```

### 前端

```powershell
cd web
pnpm dev
```

默认访问：`http://127.0.0.1:3000`

## 2. Docker Compose 运行

```powershell
docker compose up --build
```

启动后：

- 前端：`http://127.0.0.1:13000`
- 后端：`http://127.0.0.1:18080`

## 3. 当前环境变量

- `OPENCRAB_HTTP_ADDR`
- `OPENCRAB_GATEWAY_PROVIDER`
- `OPENCRAB_UPSTREAM_BASE_URL`
- `OPENCRAB_UPSTREAM_API_KEY`
- `OPENCRAB_UPSTREAM_TIMEOUT`
- `OPENCRAB_ANTHROPIC_BASE_URL`
- `OPENCRAB_ANTHROPIC_API_KEY`
- `OPENCRAB_ANTHROPIC_VERSION`

参考：根目录 `.env.example`

## 4. 当前已可用后端接口

### 系统接口

- `GET /healthz`
- `GET /readyz`

### 管理接口

- `GET /api/admin/auth/status`
- `GET /api/admin/auth/security`
- `GET /api/admin/dashboard/summary`
- `GET /api/admin/channels`
- `GET /api/admin/api-keys`
- `GET /api/admin/models`
- `GET /api/admin/model-routes`
- `GET /api/admin/logs`
- `GET /api/admin/settings`

### 网关接口

- `POST /v1/chat/completions`
- `POST /v1/messages`

## 5. 当前限制

当前后端只实现了最小网关骨架，重点是先把两条主协议入口接通：

1. OpenAI 兼容目前只覆盖 `POST /v1/chat/completions`。
2. Claude 原生目前只覆盖 `POST /v1/messages`。
3. 其它协议面，比如 `responses`、`realtime`、`/v1/models`、多媒体专用接口，还没有接回。
4. 当前管理接口是为了现有 web SSR 不报错而补的兼容只读 stub，不是完整控制台后端。
