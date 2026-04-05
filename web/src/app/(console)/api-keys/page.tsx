"use client";

import type { ColumnDef } from "@tanstack/react-table";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DataTable } from "@/components/shared/data-table";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { apiKeys } from "@/lib/mock/console-data";

const columns: ColumnDef<(typeof apiKeys)[number]>[] = [
  {
    accessorKey: "name",
    header: "名称",
    cell: ({ row }) => <span className="font-medium text-slate-900">{row.original.name}</span>
  },
  {
    accessorKey: "preview",
    header: "预览",
    cell: ({ row }) => <span className="font-mono text-xs text-slate-600">{row.original.preview}</span>
  },
  {
    accessorKey: "status",
    header: "状态",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  },
  {
    accessorKey: "usage",
    header: "用量"
  },
  {
    accessorKey: "lastUsed",
    header: "最近使用"
  }
];

export default function ApiKeysPage() {
  return (
    <PageContainer>
      <PageHeader
        eyebrow="API Keys"
        title="访问密钥"
        description="这里优先展示密钥总览和状态列表，后续再把创建、复制、禁用等操作接成完整交互。"
        action={<Button>创建密钥</Button>}
      />

      <section className="grid gap-4 md:grid-cols-3">
        <SectionCard title="已创建密钥" description="当前系统内密钥总数" className="bg-slate-50"><p className="text-3xl font-semibold">3</p></SectionCard>
        <SectionCard title="启用中" description="可正常调用的密钥数量" className="bg-slate-50"><p className="text-3xl font-semibold">2</p></SectionCard>
        <SectionCard title="禁用中" description="已停用密钥数量" className="bg-slate-50"><p className="text-3xl font-semibold">1</p></SectionCard>
      </section>

      <SectionCard title="密钥列表" description="列表页负责管理状态，详情抽屉负责查看和操作。">
        <FilterBar
          placeholder="搜索密钥名称"
          chips={[{ label: "全部状态" }, { label: "最近使用" }]}
        />
        <div className="mt-4">
          <DataTable
            columns={columns}
            data={apiKeys}
            emptyTitle="暂无访问密钥"
            emptyDescription="创建第一个密钥后，这里会展示调用状态和最近使用时间。"
            rowAction={(row) => (
              <DetailDrawer title={row.name} description="这里会承载复制、禁用、重置和查看权限等操作。" triggerLabel="管理">
                <div className="space-y-6">
                  <div>
                    <h3 className="text-sm font-medium text-slate-900">密钥详情</h3>
                    <div className="mt-3 rounded-xl border border-slate-200 bg-white">
                      <dl className="divide-y divide-slate-200 text-sm">
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-slate-500">密钥预览</dt>
                          <dd className="col-span-2 font-mono text-slate-900">{row.preview}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-slate-500">当前状态</dt>
                          <dd className="col-span-2"><StatusBadge status={row.status} /></dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-slate-500">累计用量</dt>
                          <dd className="col-span-2 text-slate-900">{row.usage}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-slate-500">最近使用</dt>
                          <dd className="col-span-2 text-slate-900">{row.lastUsed}</dd>
                        </div>
                      </dl>
                    </div>
                  </div>
                  <div className="flex justify-end gap-3">
                    <Button variant="outline" className="text-rose-600 hover:bg-rose-50 hover:text-rose-700">禁用密钥</Button>
                    <Button>复制密钥</Button>
                  </div>
                </div>
              </DetailDrawer>
            )}
          />
        </div>
      </SectionCard>
    </PageContainer>
  );
}
