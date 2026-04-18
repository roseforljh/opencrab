# Protocol Bridge Audit

日期：2026-04-18

范围：
- OpenAI
- Claude
- Gemini
- Codex
- 请求工具定义
- `tool_choice`
- 流式响应桥接

结论：
- 已确认并修复 3 处同类协议桥接缺口。
- 当前未继续发现与本次同等级、同触发面的明显漏转问题。

已修复：

1. `Claude -> OpenAI`
- 问题：Claude function tools 未改写成 OpenAI `type:function`。
- 影响：请求会被放行，但上游拿不到可执行工具定义。
- 修复：在执行层增加 `rewriteClaudeToolsToOpenAI(...)`。

2. `OpenAI -> Gemini`
- 问题：OpenAI function tools 未改写成 Gemini `functionDeclarations`。
- 影响：能力层允许，但 Gemini 上游收到错误工具结构。
- 修复：在 `rewriteOpenAIToolsToGemini(...)` 中补 function tool 转换。

3. `Gemini -> OpenAI`
- 问题：Gemini `functionDeclarations` 未改写成 OpenAI `type:function`。
- 影响：能力层允许，但 OpenAI 上游收到 Gemini 原生工具结构。
- 修复：在 `rewriteGeminiToolsToOpenAI(...)` 中补 function declaration 转换。

已复核：

- `Claude -> OpenAI` 的 `tool_choice` 映射已存在并可用。
- `Claude -> OpenAI` 的 `container` / `mcp_servers` 映射已存在并可用。
- 跨协议流式降级已存在，避免把不兼容原生 SSE 直接透传给下游协议。
- `OpenAI builtin tools -> Gemini/Claude` 仍按能力层拒绝，不属于本次“允许但未改写”的漏洞。
- `Gemini URL Context + function calling` 已被显式拒绝，不属于漏转。

验证：

- `go test ./internal/provider`
- `go test ./...`
