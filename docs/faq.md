# OpenCrab 常见问题

## 1. 为什么前端能打开，但请求代理还不能正常返回？

这通常说明前端已经跑起来了，但后端没有启动，或者后端里还没有可用的启用渠道。

排查顺序：

1. 先访问 `http://127.0.0.1:8080/healthz`
2. 再访问 `http://127.0.0.1:8080/readyz`
3. 确认数据库里已经有启用状态的渠道
4. 确认上游渠道的 `endpoint` 和 `api_key` 正确

## 2. 为什么 `/readyz` 返回 not_ready？

当前实现里，`/readyz` 已经会检查 SQLite 连接是否可用。

常见原因：

1. `OPENCRAB_DB_PATH` 配置错误
2. 数据目录没有写权限
3. SQLite 文件损坏或被占用

## 3. 为什么 `/v1/chat/completions`、`/v1/responses` 或 `/v1/messages` 返回“当前没有可用路由”？

这通常不是单纯“没有启用渠道”，而是运行时没有找到可执行 route。

当前真实选路会同时看：

1. `models`
2. `model_routes`
3. `channels.enabled`
4. API Key scope
5. cooldown / sticky / fallback 等运行时状态

排查顺序：

1. 确认目标模型 alias 已存在于 `models`
2. 确认该 alias 有对应 `model_routes`
3. 确认 route 指向的 channel 已启用
4. 确认当前 API Key 没有限制该模型或 channel
5. 确认目标 route 没有被 cooldown 暂时跳过

## 4. 为什么代理接口返回 401？

当前代理接口必须携带 Bearer Token：

```text
Authorization: Bearer <your-api-key>
```

如果缺少这个头，或者数据库里找不到匹配且启用的密钥，就会返回 401。

## 5. 为什么会收到 429？

当前已经接入基础限流，默认按 API Key 维度在内存里做限流保护。

如果你短时间内连续高频请求，就会收到 429。

## 6. 为什么生成的 API Key 在数据库里看不到明文？

这是设计要求。当前数据库只保存密钥哈希值，不保存明文，避免密钥直接泄露。

## 7. 为什么 `/v1/models` 返回的模型和上游真实模型列表不一样？

因为当前 `/v1/models` 暴露的是 OpenCrab 本地可见模型视图，不是上游 provider 的原始模型发现结果。

它会受本地 alias、route 和 API Key scope 影响。

## 8. 为什么前端里很多数据还是演示数据？

当前控制台主体已经接入真实后端数据，但部分协议与运行时细节仍在修复期。

优先以 `internal/app/app.go`、`internal/transport/httpserver/router.go`、`web/src/lib/admin-api.ts` 为真相来源，不要只看旧设计文档。

## 9. Docker 启动后数据库放在哪里？

当前 `docker-compose.yml` 会把 SQLite 文件挂载到宿主机：

```text
./data/opencrab.db
```

这样容器重启后数据不会丢失。
