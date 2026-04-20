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

SQLite 数据文件会挂载到宿主机 `./data/opencrab.db`。

## 3. 当前环境变量

- `OPENCRAB_APP_NAME`
- `OPENCRAB_ENV`
- `OPENCRAB_HTTP_ADDR`
- `OPENCRAB_DB_PATH`

参考：根目录 `.env.example`

## 4. 当前已可用后端接口

### 系统接口

- `GET /healthz`
- `GET /readyz`

### 管理接口

- `GET /api/admin/channels`
- `POST /api/admin/channels`
- `PUT /api/admin/channels/{id}`
- `DELETE /api/admin/channels/{id}`
- `GET /api/admin/api-keys`
- `POST /api/admin/api-keys`
- `PUT /api/admin/api-keys/{id}`
- `DELETE /api/admin/api-keys/{id}`
- `GET /api/admin/models`
- `POST /api/admin/models`
- `PUT /api/admin/models/{id}`
- `DELETE /api/admin/models/{id}`
- `GET /api/admin/model-routes`
- `POST /api/admin/model-routes`
- `PUT /api/admin/model-routes/{id}`
- `DELETE /api/admin/model-routes/{id}`
- `GET /api/admin/logs`

### 代理接口

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `GET /v1/responses`
- `POST /v1/realtime/client_secrets`
- `POST /v1/realtime/calls`
- `GET /v1/realtime`
- `POST /v1/codex/responses`
- `GET /v1/requests/{requestID}`
- `GET /v1/requests/{requestID}/events`
- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `POST /v1beta/cachedContents`
- `GET /v1beta/cachedContents/{cacheID}`
- `POST /v1beta/models/{model}:generateContent`
- `POST /v1beta/models/{model}:streamGenerateContent`

## 5. 当前限制

当前后端已具备多协议网关基础能力，但仍处于修复与收敛阶段：

1. `responses` 与 `realtime` 仍有协议保真修复项，不能默认等同上游原生语义。
2. `/v1/models` 当前仍以本地可见模型为主，不是完整上游模型发现面。
3. 日志查询目前还是基础列表，没有复杂筛选。
4. Docker 运行在当前环境里还未完成真实 daemon 级实测。
