"use client";

import { useState } from "react";
import { AlertTriangle } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { AdminSettingGroup, AdminSettingItem } from "@/lib/admin-api";

const routingStrategyOptions = [
  { value: "sequential", label: "顺序" },
  { value: "round_robin", label: "轮询" }
];

export function SettingsClient({
  eyebrow,
  title,
  description,
  initialGroups
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialGroups: AdminSettingGroup[];
}) {
  const [groups, setGroups] = useState(initialGroups);
  const [savingKey, setSavingKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleChange = (groupTitle: string, key: string, value: string) => {
    setGroups((current) =>
      current.map((group) =>
        group.title === groupTitle
          ? { ...group, items: group.items.map((item) => (item.key === key ? { ...item, value } : item)) }
          : group
      )
    );
  };

  const handleSave = async (key: string, value: string) => {
    setError(null);
    setSavingKey(key);
    try {
      const response = await fetch("/api/admin/settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, value })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "设置保存失败");
    } finally {
      setSavingKey(null);
    }
  };

  const renderField = (groupTitle: string, item: AdminSettingItem) => {
    if (item.key === "gateway.routing_strategy") {
      return (
        <Select value={item.value} onValueChange={(value) => handleChange(groupTitle, item.key, value)}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择路由策略" />
          </SelectTrigger>
          <SelectContent>
            {routingStrategyOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    return <Input value={item.value} onChange={(event) => handleChange(groupTitle, item.key, event.target.value)} className="bg-muted/30" />;
  };

  return (
    <PageContainer>
      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      <FilterBar placeholder="搜索设置项..." chips={groups.map((group) => ({ label: group.title }))} />

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="space-y-8">
        {groups.map((group) => (
          <SectionCard key={group.title} title={group.title} description={`管理${group.title}相关的配置项。`}>
            <div className="divide-y divide-border rounded-xl border border-border bg-card">
              {group.items.map((item) => (
                <div key={item.key} className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <div className="text-sm font-medium text-foreground">{item.label}</div>
                    <div className="text-sm text-muted-foreground">{item.description}</div>
                  </div>
                  <div className="flex items-center gap-3 sm:w-72">
                    {renderField(group.title, item)}
                    <Button variant="outline" className="shrink-0" onClick={() => void handleSave(item.key, item.value)}>
                      {savingKey === item.key ? "保存中..." : "保存"}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </SectionCard>
        ))}

        <SectionCard title="危险操作区" description="这些操作可能会导致数据丢失或服务中断，请谨慎操作。" className="border-danger/20">
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
