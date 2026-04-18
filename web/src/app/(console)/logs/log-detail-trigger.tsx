"use client";

import { useEffect, useRef, useState } from "react";

import { DetailDrawer } from "@/components/shared/detail-drawer";
import { Button } from "@/components/ui/button";
import { formatDateTime, formatNumber, type AdminRequestLogDetail, type AdminRequestLogSummary } from "@/lib/admin-api";

type LogDetailsSummary = {
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
  upstreamModel?: string;
  errorMessage?: string;
  fallbackChain?: string[];
  stickyHit?: boolean;
  stickyReason?: string;
  stickyChannel?: string;
  selectedChannel?: string;
  affinityKey?: string;
  skips?: { reason?: string; channel?: string; cooldown_until?: string }[];
};

function formatJsonBlock(value: string) {
  if (!value) {
    return "无";
  }

  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function parseLogDetails(value: string): LogDetailsSummary {
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
      upstreamModel: typeof payload.upstream_model === "string" ? payload.upstream_model : undefined,
      errorMessage: typeof payload.error_message === "string" ? payload.error_message : undefined,
      fallbackChain: Array.isArray(payload.fallback_chain) ? payload.fallback_chain.filter((item): item is string => typeof item === "string") : undefined,
      stickyHit: typeof payload.sticky_hit === "boolean" ? payload.sticky_hit : undefined,
      stickyReason: typeof payload.sticky_reason === "string" ? payload.sticky_reason : undefined,
      stickyChannel: typeof payload.sticky_channel === "string" ? payload.sticky_channel : undefined,
      selectedChannel: typeof payload.selected_channel === "string" ? payload.selected_channel : undefined,
      affinityKey: typeof payload.affinity_key === "string" ? payload.affinity_key : undefined,
      skips: Array.isArray(payload.skips)
        ? payload.skips.map((item) => (typeof item === "object" && item !== null ? item as { reason?: string; channel?: string; cooldown_until?: string } : {}))
        : undefined,
    };
  } catch {
    return {};
  }
}

function HorizontalScrollJson({ children }: { children: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const handleWheel = (event: globalThis.WheelEvent) => {
      if (!event.shiftKey) {
        return;
      }

      if (container.scrollWidth <= container.clientWidth) {
        return;
      }

      const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY;
      if (delta === 0) {
        return;
      }

      event.preventDefault();
      container.scrollLeft += delta;
    };

    container.addEventListener("wheel", handleWheel, { passive: false });
    return () => {
      container.removeEventListener("wheel", handleWheel);
    };
  }, []);

  return (
    <div
      ref={containerRef}
      className="overflow-x-auto rounded-xl border border-border/60 bg-background [scrollbar-gutter:stable]"
    >
      <pre className="min-w-full w-max p-4 font-mono text-xs leading-6 text-muted-foreground">
        {children}
      </pre>
    </div>
  );
}

export function LogDetailTrigger({ row }: { row: AdminRequestLogSummary }) {
  const [open, setOpen] = useState(false);
  const [detail, setDetail] = useState<AdminRequestLogDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || detail) {
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    fetch(`/api/admin/logs/${row.id}`, { cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.text()) || `请求失败: ${response.status}`);
        }
        return response.json() as Promise<AdminRequestLogDetail>;
      })
      .then((payload) => {
        if (!cancelled) {
          setDetail(payload);
        }
      })
      .catch((requestError: unknown) => {
        if (!cancelled) {
          setError(requestError instanceof Error ? requestError.message : "日志详情加载失败");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [detail, open, row.id]);

  const details = parseLogDetails(detail?.details ?? row.details);
  const selectedChannel = details.selectedChannel ?? detail?.channel ?? row.channel;
  const statusText = detail ? (detail.status_code >= 200 && detail.status_code < 400 ? "成功" : "异常") : (row.status_code >= 200 && row.status_code < 400 ? "成功" : "异常");

  return (
    <DetailDrawer
      title={`请求详情 · ${row.request_id}`}
      description={`${formatDateTime(row.created_at)} · ${row.model} · ${selectedChannel} · ${statusText}`}
      triggerLabel="查看详情"
      trigger={
        <Button
          variant="outline"
          size="sm"
          className="h-7 whitespace-nowrap rounded-md border-border/60 bg-muted/20 px-2.5 text-[11px] font-semibold text-foreground/88 shadow-none hover:border-primary/35 hover:bg-primary/10 hover:text-primary"
        >
          查看详情
        </Button>
      }
      open={open}
      onOpenChange={setOpen}
    >
      {loading && !detail ? <div className="text-sm text-muted-foreground">日志详情加载中...</div> : null}
      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 p-4 text-sm text-danger">{error}</div> : null}
      {detail ? (
        <div className="space-y-6 text-sm text-foreground">
          <div className="grid gap-3 rounded-xl border border-border/60 bg-muted/20 p-4 md:grid-cols-2">
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">请求 ID</p>
              <p className="mt-1 break-all font-mono text-xs">{detail.request_id}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">状态码</p>
              <p className="mt-1">{detail.status_code}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Prompt Tokens</p>
              <p className="mt-1">{formatNumber(detail.prompt_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Completion Tokens</p>
              <p className="mt-1">{formatNumber(detail.completion_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">总 Tokens</p>
              <p className="mt-1">{formatNumber(detail.total_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">缓存命中</p>
              <p className="mt-1">{detail.cache_hit ? "是" : "否"}</p>
            </div>
          </div>

          <div className="grid gap-3 rounded-xl border border-border/60 bg-muted/20 p-4 md:grid-cols-2">
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">日志类型</p>
              <p className="mt-1">{details.logType ?? "未知"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">选中渠道</p>
              <p className="mt-1">{selectedChannel}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">路由策略</p>
              <p className="mt-1">{details.routingStrategy ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">调用分段</p>
              <p className="mt-1">{details.invocationBucket ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">优先级层</p>
              <p className="mt-1">{details.priorityTier ? `P${details.priorityTier}` : "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">候选数 / 命中序号</p>
              <p className="mt-1">{details.candidateCount ?? 0} / {typeof details.selectedIndex === "number" ? details.selectedIndex : "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">上游模型</p>
              <p className="mt-1 break-all">{details.upstreamModel ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">执行器</p>
              <p className="mt-1">{details.provider ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Fallback 阶段</p>
              <p className="mt-1">{details.fallbackStage ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Sticky 命中</p>
              <p className="mt-1">{details.stickyHit ? "是" : "否"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Sticky 原因</p>
              <p className="mt-1">{details.stickyReason ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Sticky 渠道</p>
              <p className="mt-1">{details.stickyChannel ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Affinity Key</p>
              <p className="mt-1 break-all">{details.affinityKey ?? "无"}</p>
            </div>
          </div>

          {details.fallbackChain && details.fallbackChain.length > 0 ? (
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">Fallback 链路</h3>
              <div className="rounded-xl border border-border/60 bg-muted/20 p-4 text-sm text-foreground">
                {details.fallbackChain.join(" → ")}
              </div>
            </div>
          ) : null}

          {details.skips && details.skips.length > 0 ? (
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">跳过记录</h3>
              <div className="space-y-2 rounded-xl border border-border/60 bg-muted/20 p-4 text-sm text-foreground">
                {details.skips.map((skip, index) => (
                  <div key={`${skip.channel ?? "skip"}-${index}`} className="rounded-lg border border-border/50 bg-background/80 px-3 py-2">
                    <div>{skip.channel ?? "未知渠道"} · {skip.reason ?? "未知原因"}</div>
                    {skip.cooldown_until ? <div className="mt-1 text-xs text-muted-foreground">恢复时间: {formatDateTime(skip.cooldown_until)}</div> : null}
                  </div>
                ))}
              </div>
            </div>
          ) : null}

          {details.errorMessage ? (
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">路由错误</h3>
              <div className="rounded-xl border border-danger/20 bg-danger/5 p-4 text-sm text-danger">
                {details.errorMessage}
              </div>
            </div>
          ) : null}

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">附加详情</h3>
            <HorizontalScrollJson>{formatJsonBlock(detail.details)}</HorizontalScrollJson>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">请求体</h3>
            <HorizontalScrollJson>{formatJsonBlock(detail.request_body)}</HorizontalScrollJson>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">响应体</h3>
            <HorizontalScrollJson>{formatJsonBlock(detail.response_body)}</HorizontalScrollJson>
          </div>
        </div>
      ) : null}
    </DetailDrawer>
  );
}
