import { ErrorState } from "@/components/shared/error-state";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { SectionCard } from "@/components/shared/section-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import {
  formatDateTime,
  formatLatency,
  formatNumber,
  type AdminRequestLogSummary
} from "@/lib/admin-api";
import { getAdminLogs } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { LogDetailTrigger } from "@/app/(console)/logs/log-detail-trigger";

export const dynamic = "force-dynamic";

type LogRow = {
  id: number;
  requestId: string;
  time: string;
  model: string;
  channel: string;
  routing: string;
  statusCode: number;
  statusText: string;
  latency: string;
  totalTokens: string;
  cacheHit: string;
  detailSummary: LogDetailsSummary;
  raw: AdminRequestLogSummary;
};

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

function isSuccessStatus(statusCode: number) {
  return statusCode >= 200 && statusCode < 400;
}

function StatusPill({ statusCode }: { statusCode: number }) {
  const isSuccess = isSuccessStatus(statusCode);

  return (
    <span
      className={`inline-flex min-w-[84px] items-center justify-center rounded-md border px-2.5 py-1.5 text-xs font-semibold tracking-[0.08em] ${
        isSuccess
          ? "border-success/25 bg-success/10 text-success"
          : "border-danger/25 bg-danger/10 text-danger"
      }`}
    >
      {statusCode}
    </span>
  );
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

const columns: StaticTableColumn<LogRow>[] = [
  {
    header: "时间",
    cell: (row) => row.time,
    className: "whitespace-nowrap"
  },
  {
    header: "请求 ID",
    cell: (row) => <span className="font-mono text-xs text-muted-foreground">{row.requestId}</span>
  },
  {
    header: "模型",
    cell: (row) => <span className="font-medium text-foreground">{row.model}</span>
  },
  {
    header: "渠道",
    cell: (row) => row.channel
  },
  {
    header: "路由",
    cell: (row) => row.routing,
    className: "whitespace-nowrap"
  },
  {
    header: "状态",
    cell: (row) => <StatusPill statusCode={row.statusCode} />,
    className: "whitespace-nowrap"
  },
  {
    header: "耗时",
    cell: (row) => row.latency,
    className: "whitespace-nowrap"
  },
  {
    header: "Tokens",
    cell: (row) => row.totalTokens,
    className: "whitespace-nowrap"
  },
  {
    header: "缓存",
    cell: (row) => row.cacheHit,
    className: "whitespace-nowrap"
  }
];

export default async function LogsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  let rows: LogRow[] = [];
  let loadError: string | null = null;

  try {
    const logs = await getAdminLogs();
    rows = logs.map((log) => {
      const detailSummary = parseLogDetails(log.details);
      return {
        id: log.id,
        requestId: log.request_id,
        time: formatDateTime(log.created_at),
        model: log.model,
        channel: detailSummary.selectedChannel ?? log.channel,
        routing: detailSummary.logType === "gateway_attempt"
          ? `${detailSummary.invocationBucket ?? "attempt"} · P${detailSummary.priorityTier ?? 0}`
          : detailSummary.decisionReason ?? "普通请求",
        statusCode: log.status_code,
        statusText: isSuccessStatus(log.status_code) ? "成功" : "异常",
        latency: formatLatency(log.latency_ms),
        totalTokens: formatNumber(log.total_tokens),
        cacheHit: log.cache_hit ? "命中" : "未命中",
        detailSummary,
        raw: log
      };
    });
  } catch (error) {
    loadError = error instanceof Error ? error.message : "请求日志加载失败";
  }

  return (
    <PageContainer>
      <PageHeader eyebrow={t("nav.logs")} title={t("logs.title")} description={t("logs.description")} />

      <SectionCard title="请求明细" description="展示后端真实落库的请求结果，支持查看请求体、响应体和附加详情。">
        {loadError ? <ErrorState title="请求日志加载失败" description={loadError} /> : null}
        {!loadError ? (
          <StaticTable
            columns={columns}
            data={rows}
            emptyTitle="暂无请求日志"
            emptyDescription="当前还没有请求进入日志表，发起一次模型请求后这里会出现最新记录。"
            rowAction={(row) => <LogDetailTrigger row={row.raw} />}
          />
        ) : null}
      </SectionCard>
    </PageContainer>
  );
}
