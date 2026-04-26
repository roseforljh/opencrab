import { ErrorState } from "@/components/shared/error-state";
import { LocalDateTime } from "@/components/shared/local-date-time";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { SectionCard } from "@/components/shared/section-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import {
  formatLatency,
  formatNumber,
  type AdminRequestLogSummary
} from "@/lib/admin-api";
import { getAdminLogs } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { ClearLogsButton } from "@/app/(console)/logs/clear-logs-button";
import { LogDetailTrigger } from "@/app/(console)/logs/log-detail-trigger";
import { filterDisplayLogs, parseLogDetails, type LogCategory, type LogDetailsSummary } from "@/app/(console)/logs/log-utils";

export const dynamic = "force-dynamic";

type LogRow = {
  id: number;
  createdAt: string;
  model: string;
  channel: string;
  statusCode: number;
  statusText: string;
  latency: string;
  totalTokens: string;
  cacheHit: string;
  detailSummary: LogDetailsSummary;
  raw: AdminRequestLogSummary;
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

const columns: StaticTableColumn<LogRow>[] = [
  {
    header: "时间",
    cell: (row) => <LocalDateTime value={row.createdAt} />,
    className: "whitespace-nowrap"
  },
  {
    header: "模型",
    cell: (row) => <span className="whitespace-nowrap font-medium text-foreground">{row.model}</span>,
    className: "whitespace-nowrap"
  },
  {
    header: "渠道",
    cell: (row) => <span className="whitespace-nowrap">{row.channel}</span>,
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

export default async function LogsPage({
  searchParams
}: {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}) {
  const resolvedSearchParams = searchParams ? await searchParams : {};
  const query = typeof resolvedSearchParams.q === "string" ? resolvedSearchParams.q : "";
  const category = typeof resolvedSearchParams.category === "string"
    ? resolvedSearchParams.category
    : "all";
  const normalizedCategory: LogCategory = ["all", "failed", "success", "cached", "bridged"].includes(category)
    ? category as LogCategory
    : "all";
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  let rows: LogRow[] = [];
  let totalVisibleLogs = 0;
  let totalDisplayLogs = 0;
  let loadError: string | null = null;

  try {
    const response = await getAdminLogs({ q: query, category: normalizedCategory });
    totalDisplayLogs = response.total;
    totalVisibleLogs = response.filtered;
    const logs = filterDisplayLogs(response.items, { query: "", category: "all" });
    rows = logs.map((log) => {
      const detailSummary = parseLogDetails(log.details);
      return {
        id: log.id,
        createdAt: log.created_at,
        model: log.model,
        channel: detailSummary.selectedChannel ?? log.channel,
        statusCode: log.status_code,
        statusText: isSuccessStatus(log.status_code) ? "成功" : "异常",
        latency: formatLatency(log.latency_ms),
        totalTokens: formatNumber(log.total_tokens),
        cacheHit: detailSummary.cachedTokens && detailSummary.cachedTokens > 0
          ? `命中 · 读 ${formatNumber(detailSummary.cachedTokens)}${detailSummary.cacheCreationTokens && detailSummary.cacheCreationTokens > 0 ? ` · 写 ${formatNumber(detailSummary.cacheCreationTokens)}` : ""}`
          : detailSummary.cacheCreationTokens && detailSummary.cacheCreationTokens > 0
            ? `未命中 · 写 ${formatNumber(detailSummary.cacheCreationTokens)}`
            : log.cache_hit ? "命中" : "未命中",
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

      <SectionCard
        title="请求明细"
        description="支持搜索、分类筛选和一键清空；失败日志会保留，便于直接 debug。"
        action={
          <div className="text-xs text-muted-foreground">
            当前显示 {totalVisibleLogs} / {totalDisplayLogs}
          </div>
        }
      >
        <ClearLogsButton
          category={normalizedCategory}
          totalCount={totalDisplayLogs}
          visibleCount={totalVisibleLogs}
          query={query}
        />
        <div className="mt-5">
          {loadError ? <ErrorState title="请求日志加载失败" description={loadError} /> : null}
          {!loadError ? (
            <StaticTable
              columns={columns}
              data={rows}
              emptyTitle="暂无匹配日志"
              emptyDescription="当前筛选条件下没有匹配结果。你可以清空搜索词、切换分类，或先发起一次模型请求。"
              rowAction={(row) => (
                <div className="flex justify-end whitespace-nowrap">
                  <LogDetailTrigger row={row.raw} />
                </div>
              )}
            />
          ) : null}
        </div>
      </SectionCard>
    </PageContainer>
  );
}
