import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { settingsGroups } from "@/lib/mock/console-data";
import { AlertTriangle } from "lucide-react";

export default function SettingsPage() {
  return (
    <PageContainer>
      <PageHeader
        eyebrow="Settings"
        title="系统设置"
        description="管理 OpenCrab 的全局配置、运行策略和高风险操作。"
      />

      <FilterBar placeholder="搜索设置项..." chips={[{ label: "全部分组" }, { label: "基础设置" }, { label: "运行策略" }]} />

      <div className="space-y-8">
        {settingsGroups.map((group) => (
          <SectionCard key={group.title} title={group.title} description={`管理${group.title}相关的配置项。`}>
            <div className="divide-y divide-slate-200 rounded-xl border border-slate-200 bg-white">
              {group.items.map((item) => (
                <div key={item.label} className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <div className="text-sm font-medium text-slate-900">{item.label}</div>
                    <div className="text-sm text-slate-500">配置说明或当前状态描述。</div>
                  </div>
                  <div className="flex items-center gap-3 sm:w-72">
                    <Input defaultValue={item.value} className="bg-slate-50" />
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
          className="border-rose-200"
        >
          <div className="divide-y divide-rose-100 rounded-xl border border-rose-200 bg-rose-50/50">
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-rose-900">
                  <AlertTriangle className="h-4 w-4" />
                  清空系统日志
                </div>
                <div className="text-sm text-rose-700/80">永久删除所有请求日志和异常记录，此操作不可恢复。</div>
              </div>
              <Button variant="danger" className="shrink-0">清空日志</Button>
            </div>
            <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-sm font-medium text-rose-900">
                  <AlertTriangle className="h-4 w-4" />
                  重置所有配置
                </div>
                <div className="text-sm text-rose-700/80">将所有系统设置恢复为默认值，不影响渠道和模型数据。</div>
              </div>
              <Button variant="danger" className="shrink-0">重置配置</Button>
            </div>
          </div>
        </SectionCard>
      </div>
    </PageContainer>
  );
}
