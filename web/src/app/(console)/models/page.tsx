import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { modelRoutes } from "@/lib/mock/console-data";
import { Search, Plus, ArrowRight, Settings2 } from "lucide-react";

export default function ModelsPage() {
  return (
    <PageContainer>
      <PageHeader
        eyebrow="Models & Routing"
        title="模型映射与路由"
        description="配置对外暴露的模型别名，并设置它们如何路由到具体的上游渠道模型。"
      />

      <section className="grid gap-6 xl:grid-cols-[300px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-slate-500" />
              <Input placeholder="搜索别名..." className="pl-9" />
            </div>
            <Button size="icon" variant="outline" className="shrink-0">
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          
          <div className="flex flex-col gap-2">
            {modelRoutes.map((route, index) => (
              <button
                key={route.alias}
                className={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-colors ${
                  index === 0 
                    ? "border-blue-200 bg-blue-50/50 ring-1 ring-blue-500/20" 
                    : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50"
                }`}
              >
                <div className="flex w-full items-center justify-between">
                  <span className="font-medium text-slate-900">{route.alias}</span>
                  <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
                    {route.priority}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 text-xs text-slate-500">
                  <span>{route.target}</span>
                  <ArrowRight className="h-3 w-3" />
                  <span className="truncate">{route.channel}</span>
                </div>
              </button>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <SectionCard 
            title="路由配置" 
            description="当前选中别名的主路由规则。"
            action={
              <DetailDrawer title="编辑路由规则" description="修改模型别名的目标渠道和优先级。" triggerLabel="编辑">
                <div className="space-y-4 text-sm text-slate-600">
                  <div className="space-y-2">
                    <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">目标模型</label>
                    <Input defaultValue={modelRoutes[0].target} />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">目标渠道</label>
                    <Input defaultValue={modelRoutes[0].channel} />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">优先级</label>
                    <Input defaultValue={modelRoutes[0].priority} />
                  </div>
                  <Button className="w-full">保存更改</Button>
                </div>
              </DetailDrawer>
            }
          >
            <div className="grid gap-6 md:grid-cols-2">
              <div className="space-y-1">
                <span className="text-sm font-medium text-slate-500">对外别名 (Alias)</span>
                <p className="text-base font-medium text-slate-900">{modelRoutes[0].alias}</p>
              </div>
              <div className="space-y-1">
                <span className="text-sm font-medium text-slate-500">优先级</span>
                <p className="text-base font-medium text-slate-900">{modelRoutes[0].priority}</p>
              </div>
              <div className="space-y-1">
                <span className="text-sm font-medium text-slate-500">目标模型 (Target)</span>
                <p className="text-base font-medium text-slate-900">{modelRoutes[0].target}</p>
              </div>
              <div className="space-y-1">
                <span className="text-sm font-medium text-slate-500">目标渠道 (Channel)</span>
                <p className="text-base font-medium text-slate-900">{modelRoutes[0].channel}</p>
              </div>
            </div>
          </SectionCard>

          <SectionCard 
            title="回退策略 (Fallback)" 
            description="当主路由渠道不可用或触发限流时，将自动尝试回退策略。"
            action={
              <Button variant="outline" size="sm" className="gap-2">
                <Settings2 className="h-4 w-4" />
                配置回退
              </Button>
            }
          >
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-900">默认回退模型</p>
                  <p className="mt-1 text-sm text-slate-500">当前配置为 {modelRoutes[0].fallback}</p>
                </div>
                <span className="rounded-full bg-green-50 px-2.5 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">
                  已启用
                </span>
              </div>
            </div>
          </SectionCard>
        </div>
      </section>
    </PageContainer>
  );
}
