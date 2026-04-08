import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ChannelMix } from "@/components/shared/channel-mix";
import { DashboardChart } from "@/components/shared/dashboard-chart";
import { EmptyState } from "@/components/shared/empty-state";
import { ErrorState } from "@/components/shared/error-state";
import { SectionCard } from "@/components/shared/section-card";
import { StatCard } from "@/components/shared/stat-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import {
  getAdminApiKeys,
  getAdminChannels,
  getAdminLogs,
  getAdminModels,
  getAdminModelRoutes,
  formatDateTime,
  formatLatency,
  formatNumber,
  formatPercent,
  type AdminRequestLog
} from "@/lib/admin-api";
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
  const isSuccessStatus = status === "成功";
  return (
    <span
      className={`inline-flex min-w-[72px] items-center justify-center rounded-md border px-2.5 py-1.5 text-xs font-semibold tracking-[0.08em] ${
        isSuccessStatus
          ? "border-success/25 bg-success/10 text-success"
          : "border-danger/25 bg-danger/10 text-danger"
      }`}
    >
      {status}
    </span>
  );
}

const columns: StaticTableColumn<RecentLog>[] = [
  {
    header: "时间",
    cell: (row) => row.time
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
    header: "状态",
    cell: (row) => <DashboardStatusPill status={row.status} />
  },
  {
    header: "耗时",
    cell: (row) => row.latency
  }
];

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function isSuccess(statusCode: number) {
  return statusCode >= 200 && statusCode < 400;
}

function buildDailyCounts(logs: AdminRequestLog[]) {
  const days = Array.from({ length: 7 }, (_, index) => {
    const date = new Date();
    date.setDate(date.getDate() - (6 - index));
    return date;
  });

  return days.map((date) => {
    const start = startOfDay(date);
    const end = new Date(start);
    end.setDate(end.getDate() + 1);

    const dayLogs = logs.filter((log) => {
      const createdAt = new Date(log.created_at);
      return createdAt >= start && createdAt < end;
    });

    return {
      label: `${String(start.getMonth() + 1).padStart(2, "0")}-${String(start.getDate()).padStart(2, "0")}`,
      requests: dayLogs.length,
      successRate: dayLogs.length > 0 ? (dayLogs.filter((log) => isSuccess(log.status_code)).length / dayLogs.length) * 100 : 0,
      averageLatency: dayLogs.length > 0 ? Math.round(dayLogs.reduce((sum, log) => sum + log.latency_ms, 0) / dayLogs.length) : 0,
      totalTokens: dayLogs.reduce((sum, log) => sum + log.total_tokens, 0)
    };
  });
}

function buildTrafficSeries(logs: AdminRequestLog[]) {
  return Array.from({ length: 6 }, (_, index) => {
    const now = new Date();
    const bucketStart = new Date(now);
    bucketStart.setHours(now.getHours() - (5 - index) * 4, 0, 0, 0);
    const bucketEnd = new Date(bucketStart);
    bucketEnd.setHours(bucketStart.getHours() + 4);

    const bucketLogs = logs.filter((log) => {
      const createdAt = new Date(log.created_at);
      return createdAt >= bucketStart && createdAt < bucketEnd;
    });

    return {
      label: `${String(bucketStart.getHours()).padStart(2, "0")}:00`,
      requests: bucketLogs.length,
      success: bucketLogs.filter((log) => isSuccess(log.status_code)).length,
      errors: bucketLogs.filter((log) => !isSuccess(log.status_code)).length
    };
  });
}

export default async function DashboardPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  try {
    const [channels, models, routes, apiKeys, logs] = await Promise.all([
      getAdminChannels(),
      getAdminModels(),
      getAdminModelRoutes(),
      getAdminApiKeys(),
      getAdminLogs()
    ]);

    const today = startOfDay(new Date());
    const todayLogs = logs.filter((log) => new Date(log.created_at) >= today);
    const totalRequests = logs.length;
    const successCount = logs.filter((log) => isSuccess(log.status_code)).length;
    const errorCount = totalRequests - successCount;
    const enabledChannels = channels.filter((channel) => channel.enabled);
    const defaultChannel = [...enabledChannels].sort((left, right) => left.id - right.id)[0];
    const averageLatency = totalRequests > 0 ? Math.round(logs.reduce((sum, log) => sum + log.latency_ms, 0) / totalRequests) : 0;
    const promptTokenTotal = logs.reduce((sum, log) => sum + log.prompt_tokens, 0);
    const completionTokenTotal = logs.reduce((sum, log) => sum + log.completion_tokens, 0);
    const totalTokenTotal = logs.reduce((sum, log) => sum + log.total_tokens, 0);
    const cacheHitCount = logs.filter((log) => log.cache_hit).length;
    const cacheHitRate = totalRequests > 0 ? (cacheHitCount / totalRequests) * 100 : 0;
    const lastMinute = Date.now() - 60_000;
    const lastMinuteLogs = logs.filter((log) => new Date(log.created_at).getTime() >= lastMinute);
    const requestsPerMinute = lastMinuteLogs.length;
    const tokensPerMinute = lastMinuteLogs.reduce((sum, log) => sum + log.total_tokens, 0);
    const dailyCounts = buildDailyCounts(logs);
    const trafficSeries = buildTrafficSeries(logs);
    const recentLogs: RecentLog[] = logs.slice(0, 5).map((log) => ({
      time: formatDateTime(log.created_at),
      model: log.model,
      channel: log.channel,
      status: isSuccess(log.status_code) ? "成功" : "异常",
      latency: formatLatency(log.latency_ms)
    }));

    const providerStats = Array.from(
      channels.reduce((map, channel) => {
        const key = channel.provider;
        map.set(key, (map.get(key) ?? 0) + 1);
        return map;
      }, new Map<string, number>())
    );
    const channelUsageStats = Array.from(
      logs.reduce((map, log) => {
        map.set(log.channel, (map.get(log.channel) ?? 0) + 1);
        return map;
      }, new Map<string, number>())
    );
    const modelUsageStats = Array.from(
      logs.reduce((map, log) => {
        map.set(log.model, (map.get(log.model) ?? 0) + 1);
        return map;
      }, new Map<string, number>())
    ).sort((left, right) => right[1] - left[1]);

    const channelMix = channelUsageStats.slice(0, 4).map(([label, value], index) => ({
      label,
      value: totalRequests > 0 ? Math.round((value / totalRequests) * 100) : 0,
      color: `var(--chart-${(index % 5) + 1})`
    }));
    const rankingBase = modelUsageStats[0]?.[1] ?? 1;
    const ranking = modelUsageStats.slice(0, 5).map(([label, value]) => ({
      label,
      value: `${formatNumber(value)} 次请求`,
      width: Math.max(10, Math.round((value / rankingBase) * 100))
    }));
    const statusChips = [
      ["默认渠道", defaultChannel?.name ?? "未配置"],
      ["启用渠道", `${enabledChannels.length} 个`],
      ["模型映射", `${models.length} 条`],
      ["缓存命中", `${formatNumber(cacheHitCount)} 次`]
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
          {[
            { label: "今日请求", value: formatNumber(todayLogs.length), hint: `今日累计 ${formatNumber(todayLogs.length)} 条`, trend: dailyCounts.map((item) => item.requests) },
            { label: "累计 Tokens", value: formatNumber(totalTokenTotal), hint: `输入 ${formatNumber(promptTokenTotal)} / 输出 ${formatNumber(completionTokenTotal)}`, trend: dailyCounts.map((item) => item.totalTokens) },
            { label: "近 60 秒 RPM", value: formatNumber(requestsPerMinute), hint: `${formatNumber(lastMinuteLogs.filter((log) => isSuccess(log.status_code)).length)} 条成功请求`, trend: trafficSeries.map((item) => item.requests) },
            { label: "近 60 秒 TPM", value: formatNumber(tokensPerMinute), hint: `${formatNumber(cacheHitCount)} 次缓存命中`, trend: dailyCounts.map((item) => item.totalTokens) }
          ].map((metric, index) => (
            <StatCard key={metric.label} title={metric.label} description={metric.hint} value={metric.value} trend={metric.trend} accent={`var(--chart-${(index % 5) + 1})`} />
          ))}
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
          <SectionCard title="资源概览" description="真实展示当前后端已落库的渠道、模型、路由和密钥规模。">
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
              {[
                { label: "渠道总数", value: formatNumber(channels.length), accent: "var(--chart-1)" },
                { label: "模型映射", value: formatNumber(models.length), accent: "var(--chart-2)" },
                { label: "路由规则", value: formatNumber(routes.length), accent: "var(--chart-3)" },
                { label: "访问密钥", value: formatNumber(apiKeys.length), accent: "var(--chart-4)" }
              ].map((item) => (
                <div key={item.label} className="min-w-0 rounded-2xl border border-border/70 bg-background px-5 py-5">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.accent }} />
                    {item.label}
                  </div>
                  <div className="mt-4 overflow-hidden text-ellipsis break-words text-[clamp(1.5rem,1.8vw,2.35rem)] font-semibold leading-[1.12] tracking-tight text-foreground">
                    {item.value}
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>

          <SectionCard title="网络与路由状态" description="从真实渠道配置和调用结果汇总当前路由健康度。">
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
              {[
                { label: "默认渠道", value: defaultChannel?.name ?? "未配置", accent: "var(--chart-1)" },
                { label: "Provider 数量", value: formatNumber(providerStats.length), accent: "var(--chart-2)" },
                { label: "成功率", value: formatPercent(totalRequests > 0 ? (successCount / totalRequests) * 100 : 0), accent: "var(--chart-4)" },
                { label: "平均耗时", value: formatLatency(averageLatency), accent: "var(--chart-3)" }
              ].map((item) => (
                <div key={item.label} className="min-w-0 rounded-2xl border border-border/70 bg-background px-5 py-5">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.accent }} />
                    {item.label}
                  </div>
                  <div className="mt-4 overflow-hidden text-ellipsis break-words text-[clamp(1.5rem,1.8vw,2.35rem)] font-semibold leading-[1.12] tracking-tight text-foreground">
                    {item.value}
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <SectionCard title="请求趋势" description="按最近 24 小时每 4 小时一个桶聚合真实调用数据。">
            <DashboardChart data={trafficSeries} />
          </SectionCard>

          <SectionCard title="7 日 Token 消耗" description="按自然日统计最近一周的输入与输出 Token 总量。">
            <div className="flex h-[320px] items-end gap-3 rounded-2xl border border-border/60 bg-background p-4">
              {dailyCounts.map((item) => {
                const max = Math.max(...dailyCounts.map((entry) => entry.totalTokens), 1);
                return (
                  <div key={item.label} className="flex flex-1 flex-col items-center gap-3">
                    <div className="relative flex w-full flex-1 items-end">
                      <div className="w-full rounded-t-xl bg-gradient-to-t from-[var(--chart-5)] via-[var(--chart-2)] to-[var(--chart-1)] opacity-90" style={{ height: `${Math.max((item.totalTokens / max) * 100, 10)}%` }} />
                    </div>
                    <div className="text-xs text-muted-foreground">{item.label}</div>
                  </div>
                );
              })}
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <SectionCard title="Token 消耗结构" description="用输入、输出和缓存命中来观察真实调用成本。">
            <div className="grid gap-6 lg:grid-cols-[260px_1fr] lg:items-center">
              <div className="flex items-center justify-center">
                <div className="relative flex h-52 w-52 items-center justify-center rounded-full" style={{ background: `conic-gradient(var(--chart-1) 0 ${(promptTokenTotal / Math.max(totalTokenTotal, 1)) * 100}%, var(--chart-2) ${(promptTokenTotal / Math.max(totalTokenTotal, 1)) * 100}% 100%)` }}>
                  <div className="flex h-36 w-36 flex-col items-center justify-center rounded-full border border-border bg-background text-center">
                    <span className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Tokens</span>
                    <span className="mt-2 text-3xl font-semibold tracking-tight text-foreground">{formatNumber(totalTokenTotal)}</span>
                  </div>
                </div>
              </div>
              <div className="space-y-5">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">输入 Tokens</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(promptTokenTotal)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">输出 Tokens</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(completionTokenTotal)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">缓存命中率</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatPercent(cacheHitRate)}</div>
                  </div>
                  <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">异常请求</div>
                    <div className="mt-2 text-2xl font-semibold text-foreground">{formatNumber(errorCount)}</div>
                  </div>
                </div>
                <ChannelMix items={channelMix.length > 0 ? channelMix : [{ label: "暂无数据", value: 0, color: "var(--chart-1)" }]} />
              </div>
            </div>
          </SectionCard>

          <SectionCard title="模型排行" description="按真实请求次数展示当前最常用的模型。">
            <div className="space-y-4">
              {ranking.length > 0 ? ranking.map((item, index) => (
                <div key={item.label} className="grid gap-2 rounded-xl border border-border/60 bg-background px-4 py-3">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-3">
                      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-secondary text-xs text-foreground">{index + 1}</span>
                      <span className="font-medium text-foreground">{item.label}</span>
                    </div>
                    <span className="text-muted-foreground">{item.value}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-muted">
                    <div className="h-full rounded-full bg-gradient-to-r from-[var(--chart-2)] via-[var(--chart-3)] to-[var(--chart-1)] transition-[width] duration-500 ease-[var(--ease-emphasized)]" style={{ width: `${item.width}%` }} />
                  </div>
                </div>
              )) : <EmptyState title="暂无模型请求" description="当请求日志进入数据库后，这里会自动展示模型排行。" />}
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6">
          <SectionCard title="最近活动" description="统一展示最近请求的真实日志记录。">
            <StaticTable columns={columns} data={recentLogs} emptyTitle="暂无活动" emptyDescription="当前系统还没有任何请求记录。" />
          </SectionCard>
        </section>
      </PageContainer>
    );
  } catch (error) {
    const message = error instanceof Error ? error.message : "仪表盘数据加载失败";

    return (
      <PageContainer>
        <PageHeader eyebrow={t("nav.dashboard")} title={t("dashboard.title")} description={t("dashboard.description")} />
        <ErrorState title="仪表盘数据加载失败" description={message} />
      </PageContainer>
    );
  }
}
