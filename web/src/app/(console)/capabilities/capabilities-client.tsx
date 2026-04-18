"use client";

import { useMemo, useState } from "react";
import { Plus, Save, Search, Trash2 } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { AdminCapabilityCatalog, AdminCapabilityProfile } from "@/lib/admin-api";

type CapabilityDraft = {
  scopeType: string;
  scopeKey: string;
  operation: string;
  enabled: "inherit" | "enabled" | "disabled";
  capabilities: string[];
};

function createEmptyDraft(catalog: AdminCapabilityCatalog): CapabilityDraft {
  return {
    scopeType: catalog.scope_types[0] ?? "provider_default",
    scopeKey: "",
    operation: catalog.operations[0] ?? "chat_completions",
    enabled: "inherit",
    capabilities: [],
  };
}

function draftFromItem(item: AdminCapabilityProfile): CapabilityDraft {
  return {
    scopeType: item.scope_type,
    scopeKey: item.scope_key,
    operation: item.operation,
    enabled: item.enabled === undefined ? "inherit" : item.enabled ? "enabled" : "disabled",
    capabilities: item.capabilities ?? [],
  };
}

function keyOf(item: { scope_type: string; scope_key: string; operation: string }) {
  return `${item.scope_type}::${item.scope_key}::${item.operation}`;
}

export function CapabilitiesClient({
  eyebrow,
  title,
  description,
  initialItems,
  catalog,
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialItems: AdminCapabilityProfile[];
  catalog: AdminCapabilityCatalog;
}) {
  const [items, setItems] = useState(initialItems);
  const [keyword, setKeyword] = useState("");
  const [scopeFilter, setScopeFilter] = useState("all");
  const [draft, setDraft] = useState<CapabilityDraft>(() => createEmptyDraft(catalog));
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const filteredItems = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    return items.filter((item) => {
      if (scopeFilter !== "all" && item.scope_type !== scopeFilter) {
        return false;
      }
      if (!query) {
        return true;
      }
      return [item.scope_type, item.scope_key, item.operation, ...(item.capabilities ?? [])].some((value) =>
        value.toLowerCase().includes(query),
      );
    });
  }, [items, keyword, scopeFilter]);

  const handleToggleCapability = (capability: string) => {
    setDraft((current) => ({
      ...current,
      capabilities: current.capabilities.includes(capability)
        ? current.capabilities.filter((item) => item !== capability)
        : [...current.capabilities, capability].sort((left, right) => left.localeCompare(right)),
    }));
  };

  const resetDraft = () => {
    setEditingKey(null);
    setDraft(createEmptyDraft(catalog));
  };

  const handleEdit = (item: AdminCapabilityProfile) => {
    setEditingKey(keyOf(item));
    setDraft(draftFromItem(item));
    setError(null);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const payload = {
        scope_type: draft.scopeType.trim(),
        scope_key: draft.scopeKey.trim(),
        operation: draft.operation.trim(),
        enabled: draft.enabled === "inherit" ? undefined : draft.enabled === "enabled",
        capabilities: draft.capabilities,
      };
      const response = await fetch("/api/admin/capability-profiles", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "保存能力配置失败");
      }

      const nextItem: AdminCapabilityProfile = {
        scope_type: payload.scope_type,
        scope_key: payload.scope_key,
        operation: payload.operation,
        enabled: payload.enabled,
        capabilities: payload.capabilities,
      };

      setItems((current) => {
        const next = current.filter((item) => keyOf(item) !== keyOf(nextItem));
        next.push(nextItem);
        return next.sort((left, right) => keyOf(left).localeCompare(keyOf(right)));
      });
      resetDraft();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "保存能力配置失败");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!editingKey) {
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const response = await fetch("/api/admin/capability-profiles", {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          scope_type: draft.scopeType.trim(),
          scope_key: draft.scopeKey.trim(),
          operation: draft.operation.trim(),
        }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "删除能力配置失败");
      }
      setItems((current) => current.filter((item) => keyOf(item) !== editingKey));
      resetDraft();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "删除能力配置失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <PageContainer>
      <PageHeader
        eyebrow={eyebrow}
        title={title}
        description={description}
        action={
          <Button variant="outline" onClick={resetDraft}>
            <Plus className="mr-2 h-4 w-4" />
            新建规则
          </Button>
        }
      />

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
        <SectionCard
          title={editingKey ? "编辑能力规则" : "新建能力规则"}
          description="一条规则只对应一个作用域和一个操作面。保存后会覆盖默认能力矩阵。"
        >
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">作用域类型</label>
              <Select value={draft.scopeType} onValueChange={(value) => setDraft((current) => ({ ...current, scopeType: value }))}>
                <SelectTrigger className="bg-muted/30">
                  <SelectValue placeholder="选择作用域类型" />
                </SelectTrigger>
                <SelectContent>
                  {catalog.scope_types.map((scopeType) => (
                    <SelectItem key={scopeType} value={scopeType}>
                      {scopeType}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">作用域键</label>
              <Input value={draft.scopeKey} onChange={(event) => setDraft((current) => ({ ...current, scopeKey: event.target.value }))} placeholder="例如 openai / codex-main / gpt-5.4" />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">操作面</label>
              <Select value={draft.operation} onValueChange={(value) => setDraft((current) => ({ ...current, operation: value }))}>
                <SelectTrigger className="bg-muted/30">
                  <SelectValue placeholder="选择操作面" />
                </SelectTrigger>
                <SelectContent>
                  {catalog.operations.map((operation) => (
                    <SelectItem key={operation} value={operation}>
                      {operation}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">启用状态</label>
              <Select value={draft.enabled} onValueChange={(value) => setDraft((current) => ({ ...current, enabled: value as CapabilityDraft["enabled"] }))}>
                <SelectTrigger className="bg-muted/30">
                  <SelectValue placeholder="选择启用状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="inherit">继承默认</SelectItem>
                  <SelectItem value="enabled">启用</SelectItem>
                  <SelectItem value="disabled">禁用</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-3">
              <div className="text-sm font-medium text-foreground">能力集合</div>
              <div className="grid max-h-[360px] gap-2 overflow-y-auto rounded-xl border border-border/60 bg-muted/20 p-3">
                {catalog.items.map((item) => {
                  const checked = draft.capabilities.includes(item);
                  return (
                    <label key={item} className="flex items-center gap-3 rounded-lg border border-border/50 bg-background/80 px-3 py-2 text-sm text-foreground">
                      <input type="checkbox" checked={checked} onChange={() => handleToggleCapability(item)} className="h-4 w-4 accent-white" />
                      <span className="font-mono text-xs">{item}</span>
                    </label>
                  );
                })}
              </div>
            </div>

            <div className="flex items-center justify-between gap-3 pt-2">
              {editingKey ? (
                <Button variant="danger" onClick={() => void handleDelete()} disabled={saving}>
                  <Trash2 className="mr-2 h-4 w-4" />
                  删除
                </Button>
              ) : (
                <div />
              )}
              <div className="flex items-center gap-3">
                <Button variant="outline" onClick={resetDraft} disabled={saving}>
                  取消
                </Button>
                <Button onClick={() => void handleSave()} disabled={saving}>
                  <Save className="mr-2 h-4 w-4" />
                  {saving ? "保存中..." : "保存规则"}
                </Button>
              </div>
            </div>
          </div>
        </SectionCard>

        <SectionCard
          title="已生效规则"
          description="按作用域和操作面查看当前覆盖项。作用域越具体，优先级越高。"
          action={
            <div className="flex items-center gap-2">
              <div className="relative w-56">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索 scope_key 或 capability" className="bg-background pl-9" />
              </div>
              <Select value={scopeFilter} onValueChange={setScopeFilter}>
                <SelectTrigger className="w-44 bg-background">
                  <SelectValue placeholder="全部作用域" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部作用域</SelectItem>
                  {catalog.scope_types.map((scopeType) => (
                    <SelectItem key={scopeType} value={scopeType}>
                      {scopeType}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          }
        >
          {filteredItems.length === 0 ? (
            <div className="rounded-xl border border-dashed border-border bg-muted/20 px-4 py-8 text-sm text-muted-foreground">当前没有匹配的能力规则。</div>
          ) : (
            <div className="overflow-hidden rounded-xl border border-border/60">
              <table className="min-w-full divide-y divide-border/50 text-sm">
                <thead className="bg-muted/20 text-left text-muted-foreground">
                  <tr>
                    <th className="px-4 py-3 font-medium">作用域类型</th>
                    <th className="px-4 py-3 font-medium">作用域键</th>
                    <th className="px-4 py-3 font-medium">操作面</th>
                    <th className="px-4 py-3 font-medium">启用</th>
                    <th className="px-4 py-3 font-medium">能力</th>
                    <th className="px-4 py-3 font-medium text-right">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/50 bg-background">
                  {filteredItems.map((item) => (
                    <tr key={keyOf(item)} className="hover:bg-muted/20">
                      <td className="px-4 py-3 font-mono text-xs text-foreground">{item.scope_type}</td>
                      <td className="px-4 py-3 font-mono text-xs text-foreground">{item.scope_key}</td>
                      <td className="px-4 py-3 font-mono text-xs text-foreground">{item.operation}</td>
                      <td className="px-4 py-3 text-foreground">
                        {item.enabled === undefined ? "继承" : item.enabled ? "启用" : "禁用"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-2">
                          {(item.capabilities ?? []).map((capability) => (
                            <span key={capability} className="rounded-full border border-border/60 bg-muted/20 px-2 py-0.5 font-mono text-[11px] text-muted-foreground">
                              {capability}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Button variant="outline" size="sm" onClick={() => handleEdit(item)}>
                          编辑
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </SectionCard>
      </div>
    </PageContainer>
  );
}
