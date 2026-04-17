# Web Docker Image Optimization Implementation Plan

> **For Claude:** Use `${SUPERPOWERS_SKILLS_ROOT}/skills/collaboration/executing-plans/SKILL.md` to implement this plan task-by-task.

**Goal:** 在不改动 `docker compose` 对外使用方式的前提下，把前端生产镜像从当前约 `2.39GB` 降到几百 MB 量级，并保持页面可正常启动与访问。

**Architecture:** 保持 `docker-compose.yml` 不变，只改 `web/` 内的 Next.js 构建与运行镜像策略。先启用 Next.js `standalone` 产物，再让运行镜像只携带 `server.js`、trace 出来的运行时文件、`public` 和 `.next/static`，最后用 Docker Compose 做端到端冒烟验证。

**Tech Stack:** Next.js 15, React 19, pnpm, Docker, Docker Compose, PowerShell

---

## 范围约束

- 只改 `web/next.config.mjs` 和 `web/Dockerfile`
- 不改 `docker-compose.yml`
- 不改前端业务代码
- 第一轮不碰依赖升级，不换包管理器，不新增构建脚本
- 只有在 standalone 落地后体积仍明显超标时，才评估移除 `outputFileTracingRoot`

## 验收标准

- `cd web && pnpm build` 成功
- `docker compose build web` 成功
- `docker compose up -d --build api web` 后，`http://127.0.0.1:13000` 可访问
- `opencrab-web:latest` 明显小于当前约 `2.39GB`
- 运行镜像不再复制 builder 阶段的整份 `/app/node_modules`，只保留 standalone 产物自带的裁剪运行时依赖

---

### Task 1: 启用 Next standalone 产物

**Files:**
- Modify: `web/next.config.mjs`
- Verify: `web/.next/standalone/app/server.js`

**Step 1: 安装前端依赖，保证后续本地构建命令可执行**

Run:

```powershell
cd web
pnpm install --frozen-lockfile --config.node-linker=hoisted
```

Expected: PASS

**Step 2: 跑基线检查，确认当前没有 standalone 产物**

Run:

```powershell
cd web
pnpm build
Test-Path .next/standalone/app/server.js
```

Expected: 最后一行输出 `False`

**Step 3: 修改 `web/next.config.mjs`，只加最小配置**

把配置改成下面这个形态，保留现有 `reactStrictMode` 和 `outputFileTracingRoot`：

```js
import path from "node:path";

const nextConfig = {
  reactStrictMode: true,
  output: "standalone",
  outputFileTracingRoot: path.join(process.cwd(), "..")
};

export default nextConfig;
```

**Step 4: 重新构建，确认 standalone 产物已生成**

Run:

```powershell
cd web
pnpm build
Test-Path .next/standalone/app/server.js
```

Expected: 最后一行输出 `True`

**Step 5: 回读关键产物，确认运行入口存在**

Run:

```powershell
cd web
Get-ChildItem .next/standalone/app -Force
```

Expected: 输出里能看到 `server.js`

**Step 6: Commit**

```bash
git add web/next.config.mjs
git commit -m "build: enable next standalone output"
```

---

### Task 2: 改写前端 Dockerfile，只复制 standalone 运行集

**Files:**
- Modify: `web/Dockerfile`
- Verify: image `opencrab-web:latest`

**Step 1: 先复现当前运行镜像的冗余内容**

Run:

```powershell
docker build -t opencrab-web-before ./web
docker run --rm --entrypoint sh opencrab-web-before -lc "du -sh /app/node_modules /app/.next /app/public 2>/dev/null"
```

Expected: `node_modules` 约 `1.1G`，`.next` 约 `644M`

**Step 2: 修改 `web/Dockerfile` 为 standalone 运行方式**

把 runner 阶段改成下面这个最小形态：

```dockerfile
FROM node:24-bookworm-slim AS builder
WORKDIR /app
RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --config.node-linker=hoisted

COPY . .
RUN pnpm build

FROM node:24-bookworm-slim AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

COPY --from=builder /app/.next/standalone/app ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000

CMD ["node", "server.js"]
```

约束：

- 删除 `corepack enable`
- 删除 `COPY --from=builder /app/node_modules ./node_modules`
- 删除整份 `COPY --from=builder /app/.next ./.next`
- 删除 `pnpm start`

**Step 3: 构建新镜像**

Run:

```powershell
docker compose build web
```

Expected: PASS

说明：

- 当前项目保留 `outputFileTracingRoot: path.join(process.cwd(), "..")` 时，standalone 入口实际落在 `/app/.next/standalone/app/server.js`
- 所以 runner 需要复制 `/app/.next/standalone/app/` 到运行目录，才能保持 `CMD ["node", "server.js"]` 正常启动

**Step 4: 验证新镜像内容已收敛**

Run:

```powershell
docker run --rm --entrypoint sh opencrab-web:latest -lc "test -f /app/server.js && test -d /app/node_modules && du -sh /app /app/.next /app/public /app/node_modules 2>/dev/null"
```

Expected:

- 命令返回码为 0
- `/app/server.js` 存在
- `/app/node_modules` 存在，但它来自 standalone 裁剪产物，不是 builder 阶段的完整依赖目录
- `/app` 总体积显著小于旧镜像内容

**Step 5: Commit**

```bash
git add web/Dockerfile
git commit -m "build: ship next standalone runtime image"
```

---

### Task 3: 用 Docker Compose 做端到端冒烟验证

**Files:**
- Modify: none
- Verify: `docker-compose.yml`, `web/Dockerfile`, `web/next.config.mjs`

**Step 1: 启动完整栈**

Run:

```powershell
docker compose up -d --build api web
```

Expected: `api` 和 `web` 都启动成功

**Step 2: 检查前端首页可访问**

Run:

```powershell
(Invoke-WebRequest http://127.0.0.1:13000 -UseBasicParsing).StatusCode
```

Expected: `200`

**Step 3: 检查前端容器日志，没有启动入口错误**

Run:

```powershell
docker compose logs web --tail=100
```

Expected: 没有 `Cannot find module`、`MODULE_NOT_FOUND`、`ENOENT: no such file or directory, open '/app/server.js'`

**Step 4: 记录最终镜像体积**

Run:

```powershell
docker image inspect opencrab-web:latest --format '{{.Size}}'
docker image ls --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" | Select-String "opencrab-web"
```

Expected:

- 字节数显著低于当前基线 `2390000000` 左右
- 表格输出明显小于 `2.39GB`

**Step 5: 回读关键文件，确认最终实现已落地**

Run:

```powershell
Get-Content web\Dockerfile
Get-Content web\next.config.mjs
```

Expected:

- `web/Dockerfile` 使用 `COPY --from=builder /app/.next/standalone/app ./`
- `web/Dockerfile` 使用 `CMD ["node", "server.js"]`
- `web/next.config.mjs` 包含 `output: "standalone"`

---

### Task 4: 条件任务，只在镜像仍偏大时收缩 tracing 根

**Gate:** 只有在 Task 3 完成后，`opencrab-web:latest` 仍大于 `300MB`，才执行本任务。

**Files:**
- Modify: `web/next.config.mjs`
- Verify: `web/.next/required-server-files.json`

**Step 1: 先记录当前 tracing 配置和产物**

Run:

```powershell
Get-Content web\next.config.mjs
cd web
Get-Content .next/required-server-files.json
```

Expected: 能看到当前 `outputFileTracingRoot`

**Step 2: 移除 `outputFileTracingRoot`，只保留 `output: "standalone"`**

目标形态：

```js
const nextConfig = {
  reactStrictMode: true,
  output: "standalone"
};
```

**Step 3: 重新构建并验证**

Run:

```powershell
cd web
pnpm build
Get-ChildItem .next/standalone -Force
docker compose build web
```

Expected: 全部 PASS

**Step 4: 如 standalone 入口层级变化，同步改 `web/Dockerfile`**

规则：

- 如果 `.next/standalone/server.js` 出现在顶层，则把 `COPY --from=builder /app/.next/standalone/app ./` 改成 `COPY --from=builder /app/.next/standalone ./`
- 如果 `server.js` 仍在 `.next/standalone/app/`，则保持当前 Dockerfile 不变
- 无论哪种情况，都必须继续保持 `CMD ["node", "server.js"]`

**Step 5: 再跑一遍 Compose 冒烟**

Run:

```powershell
docker compose up -d --build api web
(Invoke-WebRequest http://127.0.0.1:13000 -UseBasicParsing).StatusCode
```

Expected: `200`

**Step 6: Commit**

```bash
git add web/next.config.mjs
git commit -m "build: narrow next tracing scope"
```

---

## 最终核对清单

- `web/next.config.mjs` 已启用 `output: "standalone"`
- `web/Dockerfile` runner 阶段不再复制 builder 的完整 `node_modules`
- `web/Dockerfile` runner 阶段不再复制整份 `.next`
- 前端容器通过 `node server.js` 启动
- `docker compose up -d --build api web` 后首页返回 `200`
- 最终镜像体积已记录，可与旧基线 `2.39GB` 对比
