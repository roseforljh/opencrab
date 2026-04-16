"use client";

import { useMemo, useState } from "react";
import { Search, Plus, ArrowRight, Settings2 } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type ModelRouteRow = {
  id: number;
  modelId?: number;
  alias: string;
  target: string;
  channel: string;
  invocationMode: string;
  priority: string;
  fallback: string;
};

export function ModelsClient({
  eyebrow,
  title,
  description,
  initialRoutes,
  channelNames
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialRoutes: ModelRouteRow[];
  channelNames: string[];
}) {
  const [selectedAlias, setSelectedAlias] = useState(initialRoutes[0]?.alias ?? "");
  const [keyword, setKeyword] = useState("");

  const filteredRoutes = useMemo(
    () => initialRoutes.filter((route) => route.alias.toLowerCase().includes(keyword.toLowerCase()) || route.target.toLowerCase().includes(keyword.toLowerCase())),
    [initialRoutes, keyword]
  );
  const selectedRoute = filteredRoutes.find((route) => route.alias === selectedAlias) ?? filteredRoutes[0];

  return (
    <PageContainer>
      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      <section className="grid gap-6 xl:grid-cols-[300px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索别名..." className="bg-card pl-9" />
            </div>
            <Button size="icon" variant="outline" className="shrink-0">
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex flex-col gap-2">
            {filteredRoutes.map((route) => (
              <button
                key={route.alias}
                onClick={() => setSelectedAlias(route.alias)}
                className={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-[background-color,border-color,transform,box-shadow] duration-200 ease-[var(--ease-out-smooth)] ${
                  route.alias === selectedAlias
                    ? "border-primary/30 bg-primary/5 ring-1 ring-primary/20"
                    : "border-border bg-card hover:border-border/80 hover:bg-muted/50"
                }`}
              >
                <div className="flex w-full items-center justify-between">
                  <span className="font-medium text-foreground">{route.alias}</span>
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{route.priority}</span>
                </div>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
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
              selectedRoute ? (
                <DetailDrawer title="编辑路由规则" description="修改模型别名的目标渠道和优先级。" triggerLabel="编辑">
                  <div className="space-y-4 text-sm text-muted-foreground">
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">目标模型</label>
                      <Input defaultValue={selectedRoute.target} />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">目标渠道</label>
                      <Input defaultValue={selectedRoute.channel} list="channel-options" />
                      <datalist id="channel-options">
                        {channelNames.map((channel) => <option key={channel} value={channel} />)}
                      </datalist>
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">优先级</label>
                      <Input defaultValue={selectedRoute.priority} />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">调用方式</label>
                      <Input defaultValue={selectedRoute.invocationMode} />
                    </div>
                    <Button className="w-full">保存更改</Button>
                  </div>
                </DetailDrawer>
              ) : null
            }
          >
            {selectedRoute ? (
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">对外别名 (Alias)</span>
                  <p className="text-base font-medium text-foreground">{selectedRoute.alias}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">优先级</span>
                  <p className="text-base font-medium text-foreground">{selectedRoute.priority}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">目标模型 (Target)</span>
                  <p className="text-base font-medium text-foreground">{selectedRoute.target}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">目标渠道 (Channel)</span>
                  <p className="text-base font-medium text-foreground">{selectedRoute.channel}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">调用方式 (Invocation Mode)</span>
                  <p className="text-base font-medium text-foreground">{selectedRoute.invocationMode}</p>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">暂无模型路由数据。</div>
            )}
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
            {selectedRoute ? (
              <div className="rounded-lg border border-border bg-muted/30 p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-foreground">默认回退模型</p>
                    <p className="mt-1 text-sm text-muted-foreground">当前配置为 {selectedRoute.fallback || "未配置"}</p>
                  </div>
                  <span className="rounded-full bg-success/10 px-2.5 py-1 text-xs font-medium text-success ring-1 ring-inset ring-success/20">已配置</span>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">暂无回退策略。</div>
            )}
          </SectionCard>
        </div>
      </section>
    </PageContainer>
  );
}
