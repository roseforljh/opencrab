"use client";

import { useEffect, useRef, useState } from "react";

import { LocalDateTime } from "@/components/shared/local-date-time";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { Button } from "@/components/ui/button";
import { formatNumber, type AdminRequestLogDetail, type AdminRequestLogSummary } from "@/lib/admin-api";
import { buildRoutingNarrative, parseLogDetails } from "@/app/(console)/logs/log-utils";

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

  const detailRow = detail ?? {
    ...row,
    request_body: "",
    response_body: "",
  };
  const details = parseLogDetails(detailRow.details);
  const selectedChannel = details.selectedChannel ?? detailRow.channel;
  const statusText = detailRow.status_code >= 200 && detailRow.status_code < 400 ? "成功" : "异常";
  const routingNarrative = buildRoutingNarrative(details, {
    model: detailRow.model,
    channel: detailRow.channel,
    statusCode: detailRow.status_code
  });

  return (
    <DetailDrawer
      title={`请求详情 · ${row.request_id}`}
      description={
        <>
          <LocalDateTime value={row.created_at} /> · {row.model} · {selectedChannel} · {statusText}
        </>
      }
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
      {detailRow ? (
        <div className="space-y-6 text-sm text-foreground">
          <div className="grid gap-3 rounded-xl border border-border/60 bg-muted/20 p-4 md:grid-cols-2">
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">请求 ID</p>
              <p className="mt-1 break-all font-mono text-xs">{detailRow.request_id}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">状态码</p>
              <p className="mt-1">{detailRow.status_code}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Prompt Tokens</p>
              <p className="mt-1">{formatNumber(detailRow.prompt_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Completion Tokens</p>
              <p className="mt-1">{formatNumber(detailRow.completion_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">总 Tokens</p>
              <p className="mt-1">{formatNumber(detailRow.total_tokens)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">缓存命中</p>
              <p className="mt-1">{detailRow.cache_hit ? "是" : "否"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">缓存读取 Tokens</p>
              <p className="mt-1">{formatNumber(details.cachedTokens ?? 0)}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">缓存创建 Tokens</p>
              <p className="mt-1">{formatNumber(details.cacheCreationTokens ?? 0)}</p>
            </div>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">路由与转发过程</h3>
            <div className="space-y-2 rounded-xl border border-border/60 bg-muted/20 p-4 text-sm text-foreground">
              {routingNarrative.map((line) => (
                <div key={line}>{line}</div>
              ))}
            </div>
          </div>

          <div className="grid gap-3 rounded-xl border border-border/60 bg-muted/20 p-4 md:grid-cols-2">
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">日志类型</p>
              <p className="mt-1">{details.logType ?? "未知"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">最终渠道</p>
              <p className="mt-1">{selectedChannel}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">路由策略</p>
              <p className="mt-1">{details.routingStrategy ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">决策原因</p>
              <p className="mt-1">{details.decisionReason ?? "无"}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">执行桶 / 优先级</p>
              <p className="mt-1">{details.invocationBucket ?? "无"}{details.priorityTier ? ` · P${details.priorityTier}` : ""}</p>
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
                    {skip.cooldown_until ? <div className="mt-1 text-xs text-muted-foreground">恢复时间: <LocalDateTime value={skip.cooldown_until} /></div> : null}
                  </div>
                ))}
              </div>
            </div>
          ) : null}

          {details.errorMessage ? (
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">错误信息</h3>
              <div className="rounded-xl border border-danger/20 bg-danger/5 p-4 text-sm text-danger">
                {details.errorMessage}
              </div>
            </div>
          ) : null}

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">附加详情</h3>
            <HorizontalScrollJson>{formatJsonBlock(detailRow.details)}</HorizontalScrollJson>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">请求体</h3>
            <HorizontalScrollJson>{formatJsonBlock(detailRow.request_body)}</HorizontalScrollJson>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-semibold">响应体</h3>
            <HorizontalScrollJson>{formatJsonBlock(detailRow.response_body)}</HorizontalScrollJson>
          </div>
        </div>
      ) : null}
    </DetailDrawer>
  );
}
