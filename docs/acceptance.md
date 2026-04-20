# OpenCrab 当前验收清单

## 前端

- [x] 控制台壳层可访问
- [x] Dashboard / Channels / Models / API Keys / Logs / Settings 可访问
- [x] 主要抽屉、弹窗、下拉可打开
- [x] 默认黑夜模式与中文已接入

## 后端

- [x] 服务可启动
- [x] 配置校验可工作
- [x] SQLite 可初始化
- [x] migration 可执行
- [x] `/healthz` 可访问
- [x] `/readyz` 可访问
- [x] 管理接口基础列表/创建可用
- [x] `/v1/chat/completions` 代理入口已打通
- [x] `/v1/models` 列表接口已打通
- [x] `/v1/models/{model}` 详情接口已打通
- [x] `/v1/responses/{responseID}` 查询接口已打通
- [x] `/v1/responses/{responseID}/input_items` 查询接口已打通
- [x] `/v1/responses/{responseID}` 删除接口已打通
- [x] 普通响应透传可测试
- [x] SSE 流式透传可测试
- [x] 基础 API Key 校验已接入
- [x] 基础限流已接入

## 待继续补齐

- [ ] 模型映射 CRUD
- [ ] 路由规则 CRUD
- [ ] 更完整的 API Key 管理接口
- [ ] 日志筛选与更完整查询
- [ ] 更严格的错误转换
- [ ] `responses` / `realtime` 更高保真语义对齐
- [ ] native realtime 与统一运行时选路完全收敛
- [ ] Docker 运行实测
