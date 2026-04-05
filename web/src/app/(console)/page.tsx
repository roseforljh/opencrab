"use client";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { ErrorState } from "@/components/shared/error-state";
import { LoadingState } from "@/components/shared/loading-state";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { DataTable } from "@/components/shared/data-table";
import { dashboardMetrics, dashboardRecentLogs } from "@/lib/mock/console-data";
import type { ColumnDef } from "@tanstack/react-table";

type RecentLog = typeof dashboardRecentLogs[0];

const columns: ColumnDef<RecentLog>[] = [
  {
    accessorKey: "time",
    header: "时间",
  },
  {
    accessorKey: "model",
    header: "模型",
    cell: ({ row }) => <span className="font-medium text-slate-900">{row.original.model}</span>,
  },
  {
    accessorKey: "channel",
    header: "渠道",
  },
  {
    accessorKey: "status",
    header: "状态",
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
  },
  {
    accessorKey: "latency",
    header: "耗时",
  },
];

export default function DashboardPage() {
  return (
    <PageContainer>
      <PageHeader
        eyebrow="Dashboard"
        title="系统概览"
        description="优先展示最重要的健康信息、关键指标和最近活动，让控制台首页承担真正的系统总览职责。"
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {dashboardMetrics.map((metric) => (
          <SectionCard key={metric.label} title={metric.label} description={metric.hint} className="bg-slate-50">
            <p className="text-3xl font-semibold tracking-tight text-slate-950">{metric.value}</p>
          </SectionCard>
        ))}
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.3fr_0.7fr]">
        <SectionCard title="请求趋势" description="后续会接入真实图表组件，这里先固定信息层级和占位区块。">
          <LoadingState label="图表数据准备中" />
        </SectionCard>

        <div className="grid gap-6">
          <SectionCard title="运行摘要" description="让首页同时承担状态总览和近期变更提醒。">
            <ul className="space-y-3 text-sm leading-6 text-slate-600">
              <li>已固定整体布局、导航、页面骨架和组件栈。</li>
              <li>当前阶段继续优先完善前端页面，不进入后端开发。</li>
              <li>后续图表、筛选和抽屉会先基于假数据补齐完整体验。</li>
            </ul>
          </SectionCard>

          <SectionCard title="最近异常" description="当没有异常时，也要有稳定且统一的空状态表现。">
            <EmptyState title="暂无异常请求" description="当前系统没有新的错误请求或上游异常，这里后续会展示最近异常摘要。" />
          </SectionCard>
        </div>
      </section>

      <SectionCard title="最近活动" description="首页最后一屏统一展示最近请求、异常和状态变化。">
        <DataTable
          columns={columns}
          data={dashboardRecentLogs}
          emptyTitle="暂无活动"
          emptyDescription="当前系统还没有任何请求记录。"
        />
      </SectionCard>

      <SectionCard title="异常示例" description="这里预留统一错误状态组件，保证后续页面失败时也能维持一致表现。">
        <ErrorState title="模拟上游连接异常" description="当渠道验证失败、日志查询失败或图表接口返回异常时，页面统一使用这套错误展示方式。" />
      </SectionCard>
    </PageContainer>
  );
}
