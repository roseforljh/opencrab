# 2026-04-20 网关修复计划

正式修复方案以 `.sisyphus/plans/gateway-repair-plan-2026-04-20.md` 为准。

这份文档只保留项目内公共入口，避免执行期继续引用过期设计：

1. 当前公开协议面不再只有 `POST /v1/chat/completions`。
2. 当前已存在 `responses`、`realtime`、Claude、Gemini、Codex、async request status/events` 等路由。
3. 当前运行时已使用 `models`、`model_routes`、`priority`、`sticky`、`cooldown`、`fallback_model` 参与真实路由。
4. 当前修复顺序固定为：文档真相收敛 → 共享运行时底座 → 模型发现语义 → responses surface → responses fidelity → realtime fidelity → 错误形状收口。

实施与验收细节见总方案文件，不在这里重复抄写。
