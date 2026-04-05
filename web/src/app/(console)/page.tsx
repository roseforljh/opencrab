import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ChannelMix } from "@/components/shared/channel-mix";
import { DashboardChart } from "@/components/shared/dashboard-chart";
import { EmptyState } from "@/components/shared/empty-state";
import { ErrorState } from "@/components/shared/error-state";
import { SectionCard } from "@/components/shared/section-card";
import { StatCard } from "@/components/shared/stat-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import { StatusBadge } from "@/components/shared/status-badge";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import {
  dashboardChannelMix,
  dashboardMetrics,
  dashboardNetworkStatus,
  dashboardRanking,
  dashboardRecentLogs,
  dashboardSystemStatus,
  dashboardTrafficSeries,
  dashboardTrafficSummary,
  dashboardWeeklyTraffic
} from "@/lib/mock/console-data";

type RecentLog = typeof dashboardRecentLogs[0];

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
    cell: (row) => <StatusBadge status={row.status} />
  },
  {
    header: "耗时",
    cell: (row) => row.latency
  }
];

export default async function DashboardPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <PageContainer>
      <PageHeader
        eyebrow={t("nav.dashboard")}
        title={t("dashboard.title")}
        description={t("dashboard.description")}
      />

      <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
        {[
          ["路由模式", "按别名映射"],
          ["默认网关", "openai-main"],
          ["主模型池", "12 个模型"],
          ["系统状态", "Healthy"]
        ].map(([label, value]) => (
          <div key={label} className="rounded-full border border-border bg-card px-3 py-1.5">
            <span className="text-foreground">{label}</span>
            <span className="mx-2 text-muted-foreground/50">/</span>
            <span>{value}</span>
          </div>
        ))}
      </div>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {dashboardMetrics.map((metric, index) => (
          <StatCard
            key={metric.label}
            title={metric.label}
            description={metric.hint}
            value={metric.value}
            trend={metric.trend}
            accent={`var(--chart-${(index % 5) + 1})`}
          />
        ))}
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <SectionCard title="运行状态" description="集中展示 AI 网关的运行时长、并发请求、活跃模型与当前版本。">
          <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
            {dashboardSystemStatus.map((item) => (
              <div key={item.label} className="min-w-0 rounded-2xl border border-border/70 bg-background px-5 py-5">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.accent }} />
                  {item.label}
                </div>
                {item.label === "系统版本" ? (
                  <div className="mt-4 space-y-2">
                    <div className="text-xl font-semibold leading-tight tracking-tight text-foreground">OpenCrab</div>
                    <div className="text-sm text-muted-foreground">{item.value}</div>
                  </div>
                ) : (
                  <div className="mt-4 overflow-hidden text-ellipsis break-words text-[clamp(1.5rem,1.8vw,2.35rem)] font-semibold leading-[1.12] tracking-tight text-foreground">
                    {item.value}
                  </div>
                )}
              </div>
            ))}
          </div>
        </SectionCard>

        <SectionCard title="上游状态" description="展示主要模型供应商的响应延迟和当前默认网关状态。">
          <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
            {dashboardNetworkStatus.map((item) => (
              <div key={item.label} className="min-w-0 rounded-2xl border border-border/70 bg-background px-5 py-5">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.accent }} />
                  {item.label}
                </div>
                {item.label === "默认网关" ? (
                  <div className="mt-4 space-y-1">
                    <div className="text-xl font-semibold leading-tight tracking-tight text-foreground">{item.value}</div>
                    <div className="text-sm text-muted-foreground">主路由出口</div>
                  </div>
                ) : (
                  <div className="mt-4 overflow-hidden text-ellipsis break-words text-[clamp(1.5rem,1.8vw,2.35rem)] font-semibold leading-[1.12] tracking-tight text-foreground">
                    {item.value}
                  </div>
                )}
              </div>
            ))}
          </div>
        </SectionCard>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <SectionCard title="请求趋势" description="使用彩色线性图表达请求量、成功量和异常量的节奏变化。">
          <DashboardChart data={dashboardTrafficSeries} />
        </SectionCard>

        <SectionCard title="7 日请求量" description="按自然日查看调用高峰与低谷。">
          <div className="flex h-[320px] items-end gap-3 rounded-2xl border border-border/60 bg-background p-4">
            {dashboardWeeklyTraffic.map((item, index) => (
              <div key={item.label} className="flex flex-1 flex-col items-center gap-3">
                <div className="relative flex w-full flex-1 items-end">
                  <div
                    className="w-full rounded-t-xl bg-gradient-to-t from-[var(--chart-5)] via-[var(--chart-2)] to-[var(--chart-1)] opacity-90"
                    style={{ height: `${Math.max(item.value / 10, 10)}%` }}
                  />
                  {index === 3 ? <div className="absolute inset-x-0 top-1/3 h-0.5 rounded-full bg-[var(--chart-5)] shadow-[0_0_10px_var(--chart-5)]" /> : null}
                </div>
                <div className="text-xs text-muted-foreground">{item.label}</div>
              </div>
            ))}
          </div>
        </SectionCard>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard title="Token 总结" description="用总量圆环和渠道占比快速看清当前 token 消耗构成。">
          <div className="grid gap-6 lg:grid-cols-[260px_1fr] lg:items-center">
            <div className="flex items-center justify-center">
              <div
                className="relative flex h-52 w-52 items-center justify-center rounded-full"
                style={{
                  background:
                    "conic-gradient(var(--chart-1) 0 38%, var(--chart-2) 38% 71%, var(--chart-3) 71% 86%, var(--chart-4) 86% 100%)"
                }}
              >
                <div className="flex h-36 w-36 flex-col items-center justify-center rounded-full border border-border bg-background text-center">
                  <span className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Total</span>
                  <span className="mt-2 text-3xl font-semibold tracking-tight text-foreground">{dashboardTrafficSummary.total}</span>
                </div>
              </div>
            </div>
            <div className="space-y-5">
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                  <div className="text-xs text-muted-foreground">Prompt Tokens</div>
                  <div className="mt-2 text-2xl font-semibold text-foreground">{dashboardTrafficSummary.upload}</div>
                </div>
                <div className="rounded-xl border border-border/70 bg-background px-4 py-4">
                  <div className="text-xs text-muted-foreground">Completion Tokens</div>
                  <div className="mt-2 text-2xl font-semibold text-foreground">{dashboardTrafficSummary.download}</div>
                </div>
              </div>
              <ChannelMix items={dashboardChannelMix} />
            </div>
          </div>
        </SectionCard>

        <SectionCard title="模型排行" description="按当前 token 消耗展示主要模型的使用排名。">
          <div className="space-y-4">
            {dashboardRanking.map((item, index) => (
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
            ))}
          </div>
        </SectionCard>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard title="最近活动" description="首页最后一屏统一展示最近请求、异常和状态变化。">
          <StaticTable
            columns={columns}
            data={dashboardRecentLogs}
            emptyTitle="暂无活动"
            emptyDescription="当前系统还没有任何请求记录。"
          />
        </SectionCard>

        <SectionCard title="最近异常" description="当没有异常时，也要有稳定且统一的空状态表现。">
          <EmptyState title="暂无异常请求" description="当前系统没有新的错误请求或上游异常，这里后续会展示最近异常摘要。" />
        </SectionCard>
      </section>

      <SectionCard title="异常示例" description="这里预留统一错误状态组件，保证后续页面失败时也能维持一致表现。">
        <ErrorState title="模拟上游连接异常" description="当渠道验证失败、日志查询失败或图表接口返回异常时，页面统一使用这套错误展示方式。" />
      </SectionCard>
    </PageContainer>
  );
}
