# IR Hub Completion Implementation Plan

> **For Claude:** Use `${SUPERPOWERS_SKILLS_ROOT}/skills/collaboration/executing-plans/SKILL.md` to implement this plan task-by-task.

**Goal:** 补齐协议转换中枢缺失的 planner、reject engine、transform pipeline、codex adapter，并接入主链路。

**Architecture:** 以现有 `UnifiedChatRequest / UnifiedChatResponse` 为 Canonical IR，新增独立 `reject`、`planner`、`transform` 三层。`httpserver` 只做 surface 接线，`usecase.GatewayService` 负责调度与 fallback，`provider` 负责上游执行与协议 codec。

**Tech Stack:** Go, chi, current provider codecs, current gateway service

---

### Task 1: 扩展领域协议面

**Files:**
- Modify: `internal/domain/gateway.go`
- Modify: `internal/capability/analyzer.go`
- Modify: `internal/capability/types.go`

**Step 1: 增加 Codex source protocol 与 codex_responses operation**

**Step 2: 让 capability analyzer 能把 Codex surface 归并到 OpenAI Responses 目标面**

**Step 3: 运行相关测试**

Run: `go test ./internal/domain ./internal/capability`

### Task 2: 落独立 Reject Engine

**Files:**
- Create: `internal/reject/engine.go`

**Step 1: 建立结构化 reject decision**

**Step 2: 将现有 reason string 归一到稳定 code/message/status**

**Step 3: 运行测试**

Run: `go test ./internal/reject ./internal/planner`

### Task 3: 落独立 Planner

**Files:**
- Create: `internal/planner/execution_plan.go`
- Create: `internal/planner/execution_plan_test.go`
- Modify: `internal/usecase/gateway_service.go`

**Step 1: 增加 AttemptPlan / ExecutionPlan / TransformStep**

**Step 2: 生成 direct / single_hop / multi_hop / fallback plan**

**Step 3: GatewayService 改为实际使用 planner 结果**

**Step 4: 运行测试**

Run: `go test ./internal/planner ./internal/usecase`

### Task 4: 落独立 Transform Pipeline

**Files:**
- Create: `internal/transform/pipeline.go`
- Create: `internal/transform/pipeline_test.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Modify: `internal/transport/httpserver/responses_ws.go`
- Modify: `internal/provider/executors.go`
- Modify: `internal/provider/protocol_streams.go`

**Step 1: 抽统一 ingress normalizer**

**Step 2: 抽统一 reverse transform renderer**

**Step 3: 补 Gemini synthetic stream renderer**

**Step 4: executor 统一走 target payload builder**

**Step 5: 运行测试**

Run: `go test ./internal/transform ./internal/provider ./internal/transport/httpserver`

### Task 5: 补 Codex Adapter

**Files:**
- Modify: `internal/transport/httpserver/router.go`
- Modify: `internal/transport/httpserver/gateway_handlers.go`
- Modify: `internal/transport/httpserver/proxy_test.go`

**Step 1: 新增 `/v1/codex/responses` surface**

**Step 2: 让 Codex surface 走 responses-compatible ingress + renderer**

**Step 3: 增加回归测试**

**Step 4: 运行测试**

Run: `go test ./internal/transport/httpserver`

### Task 6: 全量验证

**Files:**
- Modify: `internal/provider/codec_test.go`

**Step 1: 补 IR 字段保真回归**

**Step 2: 全量运行**

Run: `go test ./...`
