import { buildRoutingNarrative, filterDisplayLogs, isNativeDirectLog, parseLogDetails, selectDisplayLogs } from "./log-utils";
import type { AdminRequestLogSummary } from "@/lib/admin-api";

function assert(condition: unknown, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

const attemptLog: AdminRequestLogSummary = {
  id: 10,
  request_id: "req-1",
  model: "gpt-5.4",
  channel: "Codex-Fuck",
  status_code: 200,
  latency_ms: 0,
  prompt_tokens: 0,
  completion_tokens: 0,
  total_tokens: 0,
  cache_hit: false,
  details: JSON.stringify({
    log_type: "gateway_attempt",
    invocation_bucket: "neutral",
    priority_tier: 1,
    selected_channel: "Codex-Fuck",
    provider: "OpenAI"
  }),
  created_at: "2026-04-18T17:19:48Z"
};

const requestLog: AdminRequestLogSummary = {
  id: 11,
  request_id: "req-1",
  model: "gpt-5.4",
  channel: "default-channel",
  status_code: 200,
  latency_ms: 120,
  prompt_tokens: 100,
  completion_tokens: 20,
  total_tokens: 120,
  cache_hit: false,
  details: JSON.stringify({
    log_type: "gateway_request",
    decision_reason: "sticky_hit",
    selected_channel: "Codex-Fuck",
    request_path: "/v1/chat/completions",
    provider: "OpenAI",
    upstream_model: "gpt-5.4",
    response_status: 200
  }),
  created_at: "2026-04-18T17:19:48Z"
};

const selected = selectDisplayLogs([attemptLog, requestLog]);
assert(selected.length === 1, "should collapse duplicate request_id rows into one visible row");
assert(selected[0].id === 11, "should prefer gateway_request row with tokens over gateway_attempt row");

const zeroTokenRequest: AdminRequestLogSummary = {
  ...requestLog,
  id: 12,
  request_id: "req-2",
  prompt_tokens: 0,
  completion_tokens: 0,
  total_tokens: 0,
  details: JSON.stringify({
    log_type: "gateway_request",
    decision_reason: "route_success",
    selected_channel: "Codex-Fuck",
    request_path: "/v1/chat/completions",
    provider: "OpenAI",
    upstream_model: "gpt-5.4",
    response_status: 200
  })
};

const filtered = selectDisplayLogs([zeroTokenRequest]);
assert(filtered.length === 0, "should hide successful zero-token requests from logs list by default");

const failedZeroTokenRequest: AdminRequestLogSummary = {
  ...zeroTokenRequest,
  id: 13,
  request_id: "req-3",
  status_code: 502,
  details: JSON.stringify({
    log_type: "gateway_request",
    decision_reason: "route_failed",
    selected_channel: "gateway-error",
    request_path: "/v1/chat/completions",
    provider: "OpenAI",
    upstream_model: "gpt-5.4",
    response_status: 502,
    error_message: "upstream failed"
  })
};

const failedVisible = selectDisplayLogs([failedZeroTokenRequest]);
assert(failedVisible.length === 1, "should keep failed zero-token requests visible for debugging");

const filteredBySearch = filterDisplayLogs([requestLog, failedZeroTokenRequest], { query: "gateway-error", category: "all" });
assert(filteredBySearch.length === 1 && filteredBySearch[0].id === 13, "search should match selected channel / error context");

const filteredByRequestId = filterDisplayLogs([requestLog, failedZeroTokenRequest], { query: "req-3", category: "all" });
assert(filteredByRequestId.length === 1 && filteredByRequestId[0].id === 13, "search should match request id");

const filteredByCategory = filterDisplayLogs([requestLog, failedZeroTokenRequest], { query: "", category: "failed" });
assert(filteredByCategory.length === 1 && filteredByCategory[0].id === 13, "failed category should only keep failed requests");

const nativeDetails = parseLogDetails(requestLog.details);
assert(isNativeDirectLog(nativeDetails), "openai chat completions should be treated as native direct");
assert(filterDisplayLogs([requestLog], { query: "", category: "bridged" }).length === 0, "native direct requests should be excluded from bridged category");

const details = parseLogDetails(requestLog.details);
const narrative = buildRoutingNarrative(details, {
  model: requestLog.model,
  channel: requestLog.channel,
  statusCode: requestLog.status_code
});
assert(narrative.some((line) => line.includes("原生直连") || line.includes("协议桥接")), "narrative should explain forwarding path");
assert(narrative.some((line) => line.includes("渠道 Codex-Fuck")), "narrative should mention final channel");

console.log("log-utils tests passed");
