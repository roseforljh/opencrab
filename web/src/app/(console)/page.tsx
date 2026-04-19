import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ChannelMix } from "@/components/shared/channel-mix";
import { DashboardChart } from "@/components/shared/dashboard-chart";
import { EmptyState } from "@/components/shared/empty-state";
import { ErrorState } from "@/components/shared/error-state";
import { RuntimePressureGauge } from "@/components/shared/runtime-pressure-gauge";
import { SectionCard } from "@/components/shared/section-card";
import { StatCard } from "@/components/shared/stat-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import { formatCompactNumber, formatDateTime, formatLatency, formatNumber, formatPercent } from "@/lib/admin-api";
import { getAdminDashboardSummary } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

type RecentLog = {
  time: string;
  model: string;
  channel: string;
  status: string;
  latency: string;
};

function DashboardStatusPill({ status }: { status: string }) {
  const success = status === "成功";
  return (
    <span
      className={`inline-flex min-w-[72px] items-center justify-center rounded-md border px-2.5 py-1.5 text-xs font-semibold tracking-[0.08em] ${
        success ? "border-success/25 bg-success/10 text-success" : "border-danger/25 bg-danger/10 text-danger"
      }`}
    >
      {status}
    </span>
  );
}

const columns: StaticTableColumn<RecentLog>[] = [
  { header: "时间", cell: (row) => row.time },
  { header: "模型", cell: (row) => <span className="font-medium text-foreground">{row.model}</span> },
  { header: "渠道", cell: (row) => row.channel },
  { header: "状态", cell: (row) => <DashboardStatusPill status={row.status} /> },
  { header: "耗时", cell: (row) => row.latency },
];

export default async function DashboardPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  try {
    const summary = await getAdminDashboardSummary();
    const routingOverview = summary.routing_overview;

    const recentLogs: RecentLog[] = summary.recent_logs.map((log) => ({
      time: formatDateTime(log.time),
      model: log.model,
      channel: log.channel,
      status: log.status,
      latency: formatLatency(log.latency_ms),
    }));

    const channelMix = summary.channel_mix.map((item, index) => ({
      label: item.label,
      value: item.value,
      color: `var(--chart-${(index % 5) + 1})`,
    }));

    const ranking = summary.model_ranking.map((item) => ({
      label: item.label,
      value: `${formatNumber(item.value)} 次请求`,
      width: item.width,
    }));

    const statusChips = [
      ["默认渠道", summary.default_channel || "未配置"],
      ["启用渠道", `${summary.enabled_channels_count} 个`],
      ["模型映射", `${summary.models_count} 条`],
      ["缓存命中", `${formatNumber(summary.cache_hit_count)} 次`],
    ];

    const topMetrics = [
      {
        label: "今日请求",
        value: formatNumber(summary.today_requests),
        hint: `成功 ${formatNumber(summary.today_success_count)} / 失败 ${formatNumber(summary.today_error_count)}`,
        trend: summary.daily_counts.map((item) => item.requests),
      },
      {
        label: "累计 Tokens",
        value: formatCompactNumber(summary.total_tokens),
        hint: "",
        valueClassName: "text-4xl",
        trend: summary.daily_counts.map((item) => item.total_tokens),
      },
      {
        label: "近 60 秒 RPM",
        value: formatNumber(summary.requests_per_minute),
        hint: `成功 ${formatNumber(summary.requests_per_minute_success)} / 失败 ${formatNumber(summary.requests_per_minute_error)}`,
        trend: summary.traffic_series.map((item) => item.requests),
      },
      {
        label: "近 60 秒 TPM",
        value: formatNumber(summary.tokens_per_minute),
        hint: `近 60 秒有 usage 的请求 ${formatNumber(summary.tokens_per_minute_metered_requests)} 条`,
        trend: summary.daily_counts.map((item) => item.total_tokens),
      },
    ];

    return (
      <PageContainer>
        <PageHeader eyebrow={t("nav.dashboard")} title={t("dashboard.title")} description={t("dashboard.description")} />

        <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
          {statusChips.map(([label, value]) => (
            <div key={label} className="rounded-full border border-border bg-card px-3 py-1.5">
              <span className="text-foreground">{label}</span>
              <span className="mx-2 text-muted-foreground/50">/</span>
              <span>{value}</span>
            </div>
          ))}
        </div>

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {topMetrics.map((metric, index) => (
            <StatCard
              key={metric.label}
              title={metric.label}
              description={metric.hint}
              value={metric.value}
              trend={metric.trend}
              accent={`var(--chart-${(index % 5) + 1})`}
              valueClassName={"valueClassName" in metric ? metric.valueClassName : undefined}
            />
          ))}
        </section>

        <RuntimePressureGauge
          pressureScore={routingOverview.pressure_score}
          headline="Routing Runtime"
          summary={`活跃 cooldown ${formatNumber(routingOverview.active_cooldowns)} 条，Sticky 绑定 ${formatNumber(routingOverview.sticky_bindings)} 条`}
          environment="控制台"
          readiness={routingOverview.healthy_routes === routingOverview.total_routes ? "ready" : "degraded"}
          health={routingOverview.active_cooldowns > 0 ? "波动" : "稳定"}
          enabledChannels={summary.enabled_channels_count}
          requestsPerMinute={summary.requests_per_minute}
          tokensPerMinute={summary.tokens_per_minute}
          averageLatency={formatLatency(summary.average_latency)}
          errorRate={formatPercent(summary.total_requests > 0 ? (summary.error_count / summary.total_requests) * 100 : 0)}
          cacheHitRate={formatPercent(summary.cache_hit_rate)}
        />

        <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <SectionCard title="请求趋势" description="按最近 24 小时真实请求日志聚合。">
            <DashboardChart data={summary.traffic_series} />
          </SectionCard>

          <SectionCard title="7 日 Token 消耗" description="按自然日统计最近一周真实 usage 汇总。">
            <div className="flex h-[320px] items-end gap-3 rounded-2xl border border-border/60 bg-background p-4">
              {summary.daily_counts.map((item) => {
                const max = Math.max(...summary.daily_counts.map((entry) => entry.total_tokens), 1);
                return (
                  <div key={item.label} className="flex flex-1 flex-col items-center gap-3">
                    <div className="relative flex w-full flex-1 items-end">
                      <div
                        className="w-full rounded-t-xl bg-gradient-to-t from-[var(--chart-5)] via-[var(--chart-2)] to-[var(--chart-1)] opacity-90"
                        style={{ height: `${Math.max((item.total_tokens / max) * 100, 10)}%` }}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground">{item.label}</div>
                  </div>
                );
              })}
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <SectionCard title="Token 结果" description="只基于真实 usage 回传统计。">
            <div className="grid gap-6 lg:grid-cols-[260px_1fr] lg:items-center">
              <div className="flex items-center justify-center">
                <div
                  className="relative flex h-52 w-52 items-center justify-center rounded-full"
                  style={{
                    background: `conic-gradient(var(--chart-1) 0 ${(summary.prompt_tokens / Math.max(summary.total_tokens, 1)) * 100}%, var(--chart-2) ${(summary.prompt_tokens / Math.max(summary.total_tokens, 1)) * 100}% 100%)`,
                  }}
                >
                  <div className="flex h-36 w-36 flex-col items-center justify-center rounded-full border border-border bg-background text-center">
                    <span className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Tokens</span>
                    <span className="mt-2 text-3xl font-semibold tracking-tight text-foreground">{formatNumber(summary.total_tokens)}</span>
                  </div>
                </div>
              </div>

              <div className="space-y-5">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">输入 Tokens</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(summary.prompt_tokens)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">输出 Tokens</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(summary.completion_tokens)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">缓存命中率</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatPercent(summary.cache_hit_rate)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">异常请求</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(summary.error_count)}</div>
                  </div>
                </div>
                <ChannelMix items={channelMix.length > 0 ? channelMix : [{ label: "暂无数据", value: 0, color: "var(--chart-1)" }]} />
              </div>
            </div>
          </SectionCard>

          <SectionCard title="模型排行" description="按真实请求次数统计。">
            <div className="space-y-4">
              {ranking.length > 0 ? (
                ranking.map((item, index) => (
                  <div key={item.label} className="grid gap-2 rounded-xl border border-border/60 bg-background px-4 py-3">
                    <div className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-3">
                        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-secondary text-xs text-foreground">{index + 1}</span>
                        <span className="font-medium text-foreground">{item.label}</span>
                      </div>
                      <span className="text-muted-foreground">{item.value}</span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-gradient-to-r from-[var(--chart-2)] via-[var(--chart-3)] to-[var(--chart-1)] transition-[width] duration-500 ease-[var(--ease-emphasized)]"
                        style={{ width: `${item.width}%` }}
                      />
                    </div>
                  </div>
                ))
              ) : (
                <EmptyState title="暂无模型请求" description="当前还没有进入网关日志的真实模型请求。" />
              )}
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6">
          <SectionCard title="最近活动" description="展示最近的真实请求日志。">
            <StaticTable columns={columns} data={recentLogs} emptyTitle="暂无活动" emptyDescription="当前系统还没有任何真实请求记录。" />
          </SectionCard>
        </section>
      </PageContainer>
    );
  } catch (error) {
    const message = error instanceof Error ? error.message : "Dashboard 数据加载失败";
    return (
      <PageContainer>
        <PageHeader eyebrow={t("nav.dashboard")} title={t("dashboard.title")} description={t("dashboard.description")} />
        <ErrorState title="Dashboard 数据加载失败" description={message} />
      </PageContainer>
    );
  }
}
