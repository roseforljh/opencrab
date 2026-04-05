import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import { StatCard } from "@/components/shared/stat-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { apiKeys } from "@/lib/mock/console-data";

const columns: StaticTableColumn<(typeof apiKeys)[number]>[] = [
  {
    header: "名称",
    cell: (row) => <span className="font-medium text-foreground">{row.name}</span>
  },
  {
    header: "预览",
    cell: (row) => <span className="font-mono text-xs text-muted-foreground">{row.preview}</span>
  },
  {
    header: "状态",
    cell: (row) => <StatusBadge status={row.status} />
  },
  {
    header: "用量",
    cell: (row) => row.usage
  },
  {
    header: "最近使用",
    cell: (row) => row.lastUsed
  }
];

export default async function ApiKeysPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <PageContainer>
      <PageHeader
        eyebrow={t("nav.apikeys")}
        title={t("apikeys.title")}
        description={t("apikeys.description")}
        action={<Button>{t("common.create")}</Button>}
      />

      <section className="grid gap-4 md:grid-cols-3">
        <StatCard title="已创建密钥" description="当前系统内密钥总数" value="3" />
        <StatCard title="启用中" description="可正常调用的密钥数量" value="2" />
        <StatCard title="禁用中" description="已停用密钥数量" value="1" />
      </section>

      <SectionCard title="密钥列表" description="列表页负责管理状态，详情抽屉负责查看和操作。">
        <FilterBar
          placeholder="搜索密钥名称"
          chips={[{ label: "全部状态" }, { label: "最近使用" }]}
        />
        <div className="mt-4">
          <StaticTable
            columns={columns}
            data={apiKeys}
            emptyTitle="暂无访问密钥"
            emptyDescription="创建第一个密钥后，这里会展示调用状态和最近使用时间。"
            rowAction={(row) => (
              <DetailDrawer title={row.name} description="这里会承载复制、禁用、重置和查看权限等操作。" triggerLabel="管理">
                <div className="space-y-6">
                  <div>
                    <h3 className="text-sm font-medium text-foreground">密钥详情</h3>
                    <div className="mt-3 rounded-xl border border-border bg-card">
                      <dl className="divide-y divide-border text-sm">
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">密钥预览</dt>
                          <dd className="col-span-2 font-mono text-foreground">{row.preview}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">当前状态</dt>
                          <dd className="col-span-2"><StatusBadge status={row.status} /></dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">累计用量</dt>
                          <dd className="col-span-2 text-foreground">{row.usage}</dd>
                        </div>
                        <div className="grid grid-cols-3 gap-4 px-4 py-3">
                          <dt className="font-medium text-muted-foreground">最近使用</dt>
                          <dd className="col-span-2 text-foreground">{row.lastUsed}</dd>
                        </div>
                      </dl>
                    </div>
                  </div>
                  <div className="flex justify-end gap-3">
                    <Button variant="outline" className="text-danger hover:bg-danger/10 hover:text-danger">禁用密钥</Button>
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
