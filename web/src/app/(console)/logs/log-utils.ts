import type { AdminRequestLogSummary } from "@/lib/admin-api";

export const LOG_CATEGORY_OPTIONS = [
  { value: "all", label: "全部日志" },
  { value: "failed", label: "失败请求" },
  { value: "success", label: "成功请求" },
  { value: "cached", label: "缓存命中" },
  { value: "bridged", label: "协议桥接" }
] as const;

export type LogCategory = (typeof LOG_CATEGORY_OPTIONS)[number]["value"];

export type LogDetailsSummary = {
  logType?: string;
  provider?: string;
  routingStrategy?: string;
  decisionReason?: string;
  fallbackStage?: string;
  invocationBucket?: string;
  priorityTier?: number;
  candidateCount?: number;
  selectedIndex?: number;
  attempt?: number;
  attemptCount?: number;
  upstreamModel?: string;
  errorMessage?: string;
  fallbackChain?: string[];
  stickyHit?: boolean;
  stickyReason?: string;
  stickyChannel?: string;
  selectedChannel?: string;
  affinityKey?: string;
  requestPath?: string;
  responseStatus?: number;
  cachedTokens?: number;
  cacheCreationTokens?: number;
  visitedAliases?: string[];
  skips?: { reason?: string; channel?: string; cooldown_until?: string }[];
};

export function parseLogDetails(value: string): LogDetailsSummary {
  if (!value) {
    return {};
  }

  try {
    const payload = JSON.parse(value) as Record<string, unknown>;
    return {
      logType: typeof payload.log_type === "string" ? payload.log_type : undefined,
      provider: typeof payload.provider === "string" ? payload.provider : undefined,
      routingStrategy: typeof payload.routing_strategy === "string" ? payload.routing_strategy : undefined,
      decisionReason: typeof payload.decision_reason === "string" ? payload.decision_reason : undefined,
      fallbackStage: typeof payload.fallback_stage === "string" ? payload.fallback_stage : undefined,
      invocationBucket: typeof payload.invocation_bucket === "string" ? payload.invocation_bucket : undefined,
      priorityTier: typeof payload.priority_tier === "number" ? payload.priority_tier : undefined,
      candidateCount: typeof payload.candidate_count === "number" ? payload.candidate_count : undefined,
      selectedIndex: typeof payload.selected_index === "number" ? payload.selected_index : undefined,
      attempt: typeof payload.attempt === "number" ? payload.attempt : undefined,
      attemptCount: typeof payload.attempt_count === "number" ? payload.attempt_count : undefined,
      upstreamModel: typeof payload.upstream_model === "string" ? payload.upstream_model : undefined,
      errorMessage: typeof payload.error_message === "string" ? payload.error_message : undefined,
      fallbackChain: Array.isArray(payload.fallback_chain) ? payload.fallback_chain.filter((item): item is string => typeof item === "string") : undefined,
      stickyHit: typeof payload.sticky_hit === "boolean" ? payload.sticky_hit : undefined,
      stickyReason: typeof payload.sticky_reason === "string" ? payload.sticky_reason : undefined,
      stickyChannel: typeof payload.sticky_channel === "string" ? payload.sticky_channel : undefined,
      selectedChannel: typeof payload.selected_channel === "string" ? payload.selected_channel : undefined,
      affinityKey: typeof payload.affinity_key === "string" ? payload.affinity_key : undefined,
      requestPath: typeof payload.request_path === "string" ? payload.request_path : undefined,
      responseStatus: typeof payload.response_status === "number" ? payload.response_status : undefined,
      cachedTokens: typeof payload.cached_tokens === "number" ? payload.cached_tokens : undefined,
      cacheCreationTokens: typeof payload.cache_creation_tokens === "number" ? payload.cache_creation_tokens : undefined,
      visitedAliases: Array.isArray(payload.visited_aliases) ? payload.visited_aliases.filter((item): item is string => typeof item === "string") : undefined,
      skips: Array.isArray(payload.skips)
        ? payload.skips.map((item) => (typeof item === "object" && item !== null ? item as { reason?: string; channel?: string; cooldown_until?: string } : {}))
        : undefined,
    };
  } catch {
    return {};
  }
}

function scoreLog(summary: AdminRequestLogSummary, details: LogDetailsSummary) {
  let score = 0;
  if (details.logType === "gateway_request") score += 100;
  if (summary.total_tokens > 0) score += 50;
  if ((details.responseStatus ?? summary.status_code) >= 200 && (details.responseStatus ?? summary.status_code) < 400) score += 10;
  score += Math.min(summary.latency_ms, 9);
  return score;
}

export function selectDisplayLogs(logs: AdminRequestLogSummary[]) {
  const bestByRequestId = new Map<string, AdminRequestLogSummary>();

  for (const log of logs) {
    const details = parseLogDetails(log.details);
    const key = log.request_id || `log-${log.id}`;
    const existing = bestByRequestId.get(key);
    if (!existing) {
      bestByRequestId.set(key, log);
      continue;
    }

    const existingDetails = parseLogDetails(existing.details);
    if (scoreLog(log, details) > scoreLog(existing, existingDetails)) {
      bestByRequestId.set(key, log);
    }
  }

  return Array.from(bestByRequestId.values())
    .filter((log) => {
      const hasUsage = log.total_tokens > 0 || log.prompt_tokens > 0 || log.completion_tokens > 0;
      const isFailure = log.status_code >= 400;
      return hasUsage || isFailure;
    })
    .sort((left, right) => right.id - left.id);
}

export function isNativeDirectLog(details: LogDetailsSummary) {
  if (!details.provider || !details.requestPath) {
    return false;
  }

  const providerName = details.provider.toLowerCase();
  const requestPath = details.requestPath;
  return (requestPath.includes("/v1/messages") && providerName === "claude")
    || (requestPath.includes("/v1/chat/completions") && providerName === "openai")
    || (requestPath.includes("/v1/responses") && providerName === "openai")
    || (requestPath.includes("/v1beta/models") && providerName === "gemini");
}

function includesQuery(value: string | undefined, query: string) {
  if (!value) {
    return false;
  }

  return value.toLowerCase().includes(query);
}

export function matchesLogSearch(log: AdminRequestLogSummary, details: LogDetailsSummary, query: string) {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return true;
  }

  const haystacks = [
    log.request_id,
    log.model,
    log.channel,
    details.selectedChannel,
    details.provider,
    details.upstreamModel,
    details.errorMessage,
    details.routingStrategy,
    details.decisionReason,
    details.logType,
    String(log.status_code)
  ];

  return haystacks.some((value) => includesQuery(value, normalized));
}

export function matchesLogCategory(log: AdminRequestLogSummary, details: LogDetailsSummary, category: LogCategory) {
  switch (category) {
    case "failed":
      return log.status_code >= 400;
    case "success":
      return log.status_code >= 200 && log.status_code < 400;
    case "cached":
      return log.cache_hit;
    case "bridged":
      return !isNativeDirectLog(details);
    case "all":
    default:
      return true;
  }
}

export function filterDisplayLogs(logs: AdminRequestLogSummary[], options: { query?: string; category?: LogCategory }) {
  const query = options.query ?? "";
  const category = options.category ?? "all";

  return logs.filter((log) => {
    const details = parseLogDetails(log.details);
    return matchesLogSearch(log, details, query) && matchesLogCategory(log, details, category);
  });
}

export function buildRoutingNarrative(details: LogDetailsSummary, summary: { model: string; channel: string; statusCode: number }) {
  const lines: string[] = [];
  const requestPath = details.requestPath ?? "未知入口";
  const targetChannel = details.selectedChannel ?? summary.channel;
  const provider = details.provider ?? "未知执行器";
  const upstreamModel = details.upstreamModel ?? summary.model;

  lines.push(`1. 请求从 ${requestPath} 进入网关，目标模型别名是 ${summary.model}。`);

  if (details.stickyHit) {
    lines.push(`2. 本次命中了 Sticky 绑定，系统优先选择已绑定渠道 ${details.stickyChannel ?? targetChannel}${details.stickyReason ? `（原因：${details.stickyReason}）` : ""}。`);
  } else if (details.routingStrategy) {
    lines.push(`2. 网关按 ${details.routingStrategy} 路由策略挑选候选渠道。`);
  }

  if (details.invocationBucket || details.priorityTier) {
    lines.push(`3. 命中的执行桶为 ${details.invocationBucket ?? "默认桶"}${details.priorityTier ? `，优先级 P${details.priorityTier}` : ""}。`);
  }

  lines.push(`4. 最终由渠道 ${targetChannel} 发起转发，执行器是 ${provider}，上游模型是 ${upstreamModel}。`);

  if (details.provider) {
    const nativeDirect = isNativeDirectLog(details);
    lines.push(nativeDirect
      ? "5. 这次属于原生直连，请求协议与上游执行器一致，没有额外协议桥接。"
      : "5. 这次不是原生直连，网关在转发前做了协议桥接/格式转换后再发往上游。"
    );
  }

  if (details.fallbackChain && details.fallbackChain.length > 0) {
    lines.push(`6. 期间触发过 fallback，链路为：${details.fallbackChain.join(" → ")}。`);
  }

  if (details.skips && details.skips.length > 0) {
    lines.push(`7. 有 ${details.skips.length} 个候选渠道在调度阶段被跳过，详细原因见下方“跳过记录”。`);
  }

  if (details.errorMessage) {
    lines.push(`8. 最终返回 ${details.responseStatus ?? summary.statusCode}，并记录错误：${details.errorMessage}`);
  } else {
    lines.push(`8. 最终返回状态码 ${details.responseStatus ?? summary.statusCode}。`);
  }

  return lines;
}
