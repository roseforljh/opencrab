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
import {
  formatDateTime,
  formatLatency,
  formatNumber,
  formatPercent
} from "@/lib/admin-api";
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
      latency: formatLatency(log.latency_ms)
    }));
    const channelMix = summary.channel_mix.map((item, index) => ({
      label: item.label,
      value: item.value,
      color: `var(--chart-${(index % 5) + 1})`
    }));
    const ranking = summary.model_ranking.map((item) => ({
      label: item.label,
      value: `${formatNumber(item.value)} 次请求`,
      width: item.width
    }));
    const statusChips = [
      ["默认渠道", summary.default_channel || "未配置"],
      ["启用渠道", `${summary.enabled_channels_count} 个`],
      ["模型映射", `${summary.models_count} 条`],
      ["缓存命中", `${formatNumber(summary.cache_hit_count)} 次`]
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
            { label: "今日请求", value: formatNumber(summary.today_requests), hint: `今日累计 ${formatNumber(summary.today_requests)} 条`, trend: summary.daily_counts.map((item) => item.requests) },
            { label: "累计 Tokens", value: formatNumber(summary.total_tokens), hint: `输入 ${formatNumber(summary.prompt_tokens)} / 输出 ${formatNumber(summary.completion_tokens)}`, trend: summary.daily_counts.map((item) => item.total_tokens) },
            { label: "近 60 秒 RPM", value: formatNumber(summary.requests_per_minute), hint: `${formatNumber(summary.success_count)} 条成功请求`, trend: summary.traffic_series.map((item) => item.requests) },
            { label: "近 60 秒 TPM", value: formatNumber(summary.tokens_per_minute), hint: `${formatNumber(summary.cache_hit_count)} 次缓存命中`, trend: summary.daily_counts.map((item) => item.total_tokens) }
          ].map((metric, index) => (
            <StatCard key={metric.label} title={metric.label} description={metric.hint} value={metric.value} trend={metric.trend} accent={`var(--chart-${(index % 5) + 1})`} />
          ))}
        </section>

        <RuntimePressureGauge
          pressureScore={routingOverview.pressure_score}
          headline="Routing Runtime"
          summary={`活跃 cooldown ${formatNumber(routingOverview.active_cooldowns)} 条，Sticky 绑定 ${formatNumber(routingOverview.sticky_bindings)} 条。`}
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

        <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
          <SectionCard title="资源概览" description="真实展示当前后端已落库的渠道、模型、路由和密钥规模。">
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
              {[
                { label: "渠道总数", value: formatNumber(summary.channels_count), accent: "var(--chart-1)" },
                { label: "模型映射", value: formatNumber(summary.models_count), accent: "var(--chart-2)" },
                { label: "路由规则", value: formatNumber(summary.routes_count), accent: "var(--chart-3)" },
                { label: "访问密钥", value: formatNumber(summary.api_keys_count), accent: "var(--chart-4)" }
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
                { label: "默认渠道", value: summary.default_channel || "未配置", accent: "var(--chart-1)" },
                { label: "Provider 数量", value: formatNumber(summary.provider_count), accent: "var(--chart-2)" },
                { label: "活跃 Cooldown", value: formatNumber(routingOverview.active_cooldowns), accent: "var(--chart-4)" },
                { label: "Sticky 绑定", value: formatNumber(routingOverview.sticky_bindings), accent: "var(--chart-3)" }
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

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <SectionCard title="调度控制面" description="阶段 A 先展示已落库的调度配置，不提前伪造 Redis 或 dispatcher 运行时数据。">
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-4">
              {[
                { label: "Runtime Redis", value: summary.runtime_redis_enabled ? "已启用" : "未启用", accent: "var(--chart-1)" },
                { label: "Worker 并发", value: formatNumber(summary.dispatcher_workers), accent: "var(--chart-2)" },
                { label: "积压上限", value: formatNumber(summary.backlog_cap), accent: "var(--chart-3)" },
                { label: "同步等待预算", value: `${formatNumber(summary.sync_hold_ms)} ms`, accent: "var(--chart-4)" }
              ].map((item) => (
                <div key={item.label} className="min-w-0 rounded-2xl border border-border/70 bg-background px-5 py-5">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="h-2 w-2 rounded-full" style={{ backgroundColor: item.accent }} />
                    {item.label}
                  </div>
                  <div className="mt-4 overflow-hidden text-ellipsis break-words text-[clamp(1.35rem,1.6vw,2rem)] font-semibold leading-[1.12] tracking-tight text-foreground">
                    {item.value}
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-4 grid gap-3 rounded-xl border border-border/60 bg-muted/20 p-4 md:grid-cols-2 xl:grid-cols-3">
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Redis 地址</div>
                <div className="mt-1 break-all font-mono text-xs text-foreground">{summary.runtime_redis_address}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Redis DB / TLS</div>
                <div className="mt-1 text-sm text-foreground">{summary.runtime_redis_db} / {summary.runtime_redis_tls_enabled ? "开启" : "关闭"}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Key 前缀</div>
                <div className="mt-1 text-sm text-foreground">{summary.runtime_redis_key_prefix}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">队列模式</div>
                <div className="mt-1 text-sm text-foreground">{summary.queue_mode}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">默认队列</div>
                <div className="mt-1 text-sm text-foreground">{summary.default_queue}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">调度开关</div>
                <div className="mt-1 text-sm text-foreground">{summary.dispatch_pause ? "已暂停" : "运行中"}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">优先级队列</div>
                <div className="mt-1 text-sm text-foreground">{summary.priority_queues}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">重试策略</div>
                <div className="mt-1 text-sm text-foreground">{summary.backoff_mode} / {formatNumber(summary.max_attempts)} 次</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">退避延迟 / 重试预算</div>
                <div className="mt-1 text-sm text-foreground">{formatNumber(summary.backoff_delay_ms)} ms / {formatPercent(summary.retry_reserve_ratio * 100)}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">Queue TTL / 死信</div>
                <div className="mt-1 text-sm text-foreground">{formatNumber(summary.queue_ttl_s)} 秒 / {summary.dead_letter_enabled ? "开启" : "关闭"}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">观测开关</div>
                <div className="mt-1 text-sm text-foreground">{summary.metrics_enabled ? "指标开启" : "指标关闭"}</div>
              </div>
              <div>
                <div className="text-xs uppercase tracking-[0.08em] text-muted-foreground">展示项</div>
                <div className="mt-1 text-sm text-foreground">
                  {summary.show_worker_status ? "Worker" : "-"} / {summary.show_queue_depth ? "Queue" : "-"} / {summary.show_retry_rate ? "Retry" : "-"}
                </div>
              </div>
            </div>
          </SectionCard>

          <SectionCard title="渠道调度容量基线" description="展示当前渠道控制面配置出的理论容量，并区分是否可异步受理。">
            <div className="grid gap-4 sm:grid-cols-2">
              <StatCard title="总 RPM 配额" description="按渠道控制面配置累加的每分钟理论配额" value={formatNumber(summary.total_rpm_limit)} />
              <StatCard title="总 Inflight" description="按渠道控制面配置累加的最大并发槽位" value={formatNumber(summary.total_max_inflight)} />
              <StatCard title="支持异步的渠道" description="控制面中启用异步受理的渠道数量" value={formatNumber(summary.async_enabled_channels)} />
              <StatCard title="长等待阈值" description="超过该阈值的 job 将被视为长等待" value={`${formatNumber(summary.long_wait_threshold_s)} 秒`} />
            </div>
          </SectionCard>
        </section>

        <section className="grid gap-6 xl:grid-cols-4">
          <StatCard title="24h Sticky 命中" description="最终请求日志中的 sticky 命中次数" value={formatNumber(routingOverview.sticky_hits_24h)} />
          <StatCard title="24h Fallback 次数" description="发生 alias fallback 的请求数" value={formatNumber(routingOverview.fallbacks_24h)} />
          <StatCard title="24h 跳过次数" description="运行时被 cooldown 跳过的路由数" value={formatNumber(routingOverview.skipped_24h)} />
          <StatCard title="健康路由" description="当前未处于 cooldown 的路由数" value={`${formatNumber(routingOverview.healthy_routes)} / ${formatNumber(routingOverview.total_routes)}`} />
        </section>

        <section className="grid gap-6">
          <SectionCard title="当前游标状态" description="展示最近活跃的路由游标分组，用于确认轮询起始位点是否持续推进。">
            {routingOverview.cursor_states.length > 0 ? (
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {routingOverview.cursor_states.map((cursor) => (
                  <div key={cursor.route_key} className="rounded-xl border border-border/70 bg-background px-4 py-4">
                    <div className="text-xs text-muted-foreground">{cursor.route_key}</div>
                    <div className="mt-3 text-2xl font-semibold text-foreground">{formatNumber(cursor.next_index)}</div>
                    <div className="mt-2 text-xs text-muted-foreground">更新时间 {formatDateTime(cursor.updated_at)}</div>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState title="暂无游标状态" description="当前还没有轮询游标写入数据库，触发 round_robin 请求后这里会出现最近状态。" />
            )}
          </SectionCard>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <SectionCard title="请求趋势" description="按最近 24 小时每 4 小时一个桶聚合真实调用数据。">
            <DashboardChart data={summary.traffic_series} />
          </SectionCard>

          <SectionCard title="7 日 Token 消耗" description="按自然日统计最近一周的输入与输出 Token 总量。">
            <div className="flex h-[320px] items-end gap-3 rounded-2xl border border-border/60 bg-background p-4">
              {summary.daily_counts.map((item) => {
                const max = Math.max(...summary.daily_counts.map((entry) => entry.total_tokens), 1);
                return (
                  <div key={item.label} className="flex flex-1 flex-col items-center gap-3">
                    <div className="relative flex w-full flex-1 items-end">
                      <div className="w-full rounded-t-xl bg-gradient-to-t from-[var(--chart-5)] via-[var(--chart-2)] to-[var(--chart-1)] opacity-90" style={{ height: `${Math.max((item.total_tokens / max) * 100, 10)}%` }} />
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
                <div className="relative flex h-52 w-52 items-center justify-center rounded-full" style={{ background: `conic-gradient(var(--chart-1) 0 ${(summary.prompt_tokens / Math.max(summary.total_tokens, 1)) * 100}%, var(--chart-2) ${(summary.prompt_tokens / Math.max(summary.total_tokens, 1)) * 100}% 100%)` }}>
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
    const message = error instanceof Error ? error.message : "Dashboard 数据加载失败";
    return (
      <PageContainer>
        <PageHeader eyebrow={t("nav.dashboard")} title={t("dashboard.title")} description={t("dashboard.description")} />
        <ErrorState title="Dashboard 数据加载失败" description={message} />
      </PageContainer>
    );
  }
}
