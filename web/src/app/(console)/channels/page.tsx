import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { channels } from "@/lib/mock/console-data";

const columns: StaticTableColumn<(typeof channels)[number]>[] = [
  {
    header: "渠道名",
    cell: (row) => <span className="font-medium text-foreground">{row.name}</span>
  },
  {
    header: "类型",
    cell: (row) => row.provider
  },
  {
    header: "状态",
    cell: (row) => <StatusBadge status={row.status} />
  },
  {
    header: "地址",
    cell: (row) => <span className="font-mono text-xs text-muted-foreground">{row.endpoint}</span>
  },
  {
    header: "模型数",
    cell: (row) => row.models
  }
];

export default async function ChannelsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <PageContainer>
      <PageHeader
        eyebrow={t("nav.channels")}
        title={t("channels.title")}
        description={t("channels.description")}
        action={
          <DetailDrawer
            title="新建渠道"
            description="新增一个上游模型渠道，填写兼容类型、请求地址和默认认证信息。"
            triggerLabel={t("common.create")}
            trigger={<Button>{t("common.create")}</Button>}
          >
            <div className="space-y-4 text-sm text-muted-foreground">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">渠道名称</label>
                <input className="h-10 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none transition-[border-color,box-shadow] duration-200 ease-[var(--ease-out-smooth)] focus:border-ring focus:ring-2 focus:ring-ring/15" defaultValue="new-channel" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">兼容类型</label>
                <input className="h-10 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none transition-[border-color,box-shadow] duration-200 ease-[var(--ease-out-smooth)] focus:border-ring focus:ring-2 focus:ring-ring/15" defaultValue="OpenAI Compatible" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">请求地址</label>
                <input className="h-10 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground outline-none transition-[border-color,box-shadow] duration-200 ease-[var(--ease-out-smooth)] focus:border-ring focus:ring-2 focus:ring-ring/15" defaultValue="https://api.example.com/v1" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <Button variant="outline">测试连接</Button>
                <Button>保存渠道</Button>
              </div>
            </div>
          </DetailDrawer>
        }
      />

      <SectionCard title="渠道列表" description="这一页采用筛选条 + 表格区 + 右侧编辑抽屉的标准模式。">
        <FilterBar
          placeholder="搜索渠道名或地址"
          chips={[{ label: "全部状态" }, { label: "全部 Provider" }, { label: "最近更新" }]}
          trailingAction={<Button variant="secondary">测试连通性</Button>}
        />
        <div className="mt-4">
          <StaticTable
            columns={columns}
            data={channels}
            emptyTitle="暂无渠道"
            emptyDescription="添加第一个渠道后，这里会展示可接入的上游 provider。"
            rowAction={(row) => (
              <DetailDrawer title={row.name} description="这里会承载渠道编辑表单、密钥配置和测试连接操作。" triggerLabel="查看详情">
                <div className="space-y-6">
                  <div>
                    <h3 className="text-sm font-medium text-foreground">基本信息</h3>
                    <div className="mt-3 rounded-xl border border-border bg-card">
                      <dl className="divide-y divide-border text-sm">
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">渠道类型</dt>
                          <dd className="col-span-2 text-foreground">{row.provider}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">请求地址</dt>
                          <dd className="col-span-2 font-mono text-foreground">{row.endpoint}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">当前状态</dt>
                          <dd className="col-span-2"><StatusBadge status={row.status} /></dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">覆盖模型数</dt>
                          <dd className="col-span-2 text-foreground">{row.models}</dd>
                        </div>
                      </dl>
                    </div>
                  </div>
                  <div className="flex justify-end gap-3">
                    <Button variant="outline">测试连接</Button>
                    <Button>编辑配置</Button>
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
