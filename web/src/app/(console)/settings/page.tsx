import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { settingsGroups } from "@/lib/mock/console-data";
import { AlertTriangle } from "lucide-react";

export default async function SettingsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <PageContainer>
      <PageHeader
        eyebrow={t("nav.settings")}
        title={t("settings.title")}
        description={t("settings.description")}
      />

      <FilterBar placeholder="搜索设置项..." chips={[{ label: "全部分组" }, { label: "基础设置" }, { label: "运行策略" }]} />

      <div className="space-y-8">
        {settingsGroups.map((group) => (
          <SectionCard key={group.title} title={group.title} description={`管理${group.title}相关的配置项。`}>
            <div className="divide-y divide-border rounded-xl border border-border bg-card">
              {group.items.map((item) => (
                <div key={item.label} className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <div className="text-sm font-medium text-foreground">{item.label}</div>
                    <div className="text-sm text-muted-foreground">配置说明或当前状态描述。</div>
                  </div>
                  <div className="flex items-center gap-3 sm:w-72">
                    <Input defaultValue={item.value} className="bg-muted/30" />
                    <Button variant="outline" className="shrink-0">保存</Button>
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        ))}

        <SectionCard 
          title="危险操作区" 
          description="这些操作可能会导致数据丢失或服务中断，请谨慎操作。"
          className="border-danger/20"
        >
          <div className="divide-y divide-danger/10 rounded-xl border border-danger/20 bg-danger/5">
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-danger">
                  <AlertTriangle className="h-4 w-4" />
                  清空系统日志
                </div>
                <div className="text-sm text-danger/80">永久删除所有请求日志和异常记录，此操作不可恢复。</div>
              </div>
              <ConfirmDialog
                trigger={<Button variant="danger" className="shrink-0">清空日志</Button>}
                title="确认清空系统日志"
                description="该操作会删除当前所有请求日志和异常记录，只建议在测试环境或明确需要时执行。"
                confirmLabel="确认清空"
              />
            </div>
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-danger">
                  <AlertTriangle className="h-4 w-4" />
                  重置所有配置
                </div>
                <div className="text-sm text-danger/80">将所有系统设置恢复为默认值，不影响渠道和模型数据。</div>
              </div>
              <ConfirmDialog
                trigger={<Button variant="danger" className="shrink-0">重置配置</Button>}
                title="确认重置系统配置"
                description="该操作会把系统设置恢复到默认值。高风险操作必须经过二次确认。"
                confirmLabel="确认重置"
              />
            </div>
          </div>
        </SectionCard>
      </div>
    </PageContainer>
  );
}
