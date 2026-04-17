"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowRight, Plus, Search, Trash2 } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type ModelRouteRow = {
  id: number;
  alias: string;
  target: string;
  channel: string;
  invocationMode: string;
  priority: number;
  fallback: string;
};

type ModelMappingSummary = {
  id: number;
  alias: string;
  upstreamModel: string;
};

type AliasDraft = {
  alias: string;
  target: string;
};

function createEmptyDraft(): AliasDraft {
  return {
    alias: "",
    target: "",
  };
}

function mappingToDraft(mapping: ModelMappingSummary): AliasDraft {
  return {
    alias: mapping.alias,
    target: mapping.upstreamModel,
  };
}

function normalizeDraft(draft: AliasDraft) {
  const alias = draft.alias.trim();
  const target = draft.target.trim();

  if (!alias || !target) {
    throw new Error("对外模型别名和目标模型都不能为空");
  }

  return { alias, target };
}

export function ModelsClient({
  eyebrow,
  title,
  description,
  initialRoutes,
  initialModels,
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialRoutes: ModelRouteRow[];
  initialModels: ModelMappingSummary[];
}) {
  const [keyword, setKeyword] = useState("");
  const [selectedModelId, setSelectedModelId] = useState<number | null>(initialModels[0]?.id ?? null);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [busyAction, setBusyAction] = useState<"create" | "update" | "delete" | null>(null);
  const [createDraft, setCreateDraft] = useState<AliasDraft>(() => createEmptyDraft());
  const [editDraft, setEditDraft] = useState<AliasDraft>(() => (initialModels[0] ? mappingToDraft(initialModels[0]) : createEmptyDraft()));

  const directRoutesByAlias = useMemo(() => {
    const map = new Map<string, ModelRouteRow[]>();
    for (const route of initialRoutes) {
      const current = map.get(route.alias) ?? [];
      current.push(route);
      map.set(route.alias, current);
    }
    return map;
  }, [initialRoutes]);

  const filteredModels = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) {
      return initialModels;
    }
    return initialModels.filter((item) => item.alias.toLowerCase().includes(query) || item.upstreamModel.toLowerCase().includes(query));
  }, [initialModels, keyword]);

  const selectedModel = filteredModels.find((item) => item.id === selectedModelId) ?? filteredModels[0] ?? null;

  useEffect(() => {
    if (!selectedModel) {
      setSelectedModelId(null);
      return;
    }
    if (selectedModel.id !== selectedModelId) {
      setSelectedModelId(selectedModel.id);
    }
  }, [selectedModel, selectedModelId]);

  useEffect(() => {
    if (selectedModel) {
      setEditDraft(mappingToDraft(selectedModel));
    }
  }, [selectedModel]);

  const selectedResolvedRoutes = useMemo(() => {
    if (!selectedModel) {
      return [] as ModelRouteRow[];
    }
    const direct = directRoutesByAlias.get(selectedModel.alias);
    if (direct && direct.length > 0) {
      return direct;
    }
    return directRoutesByAlias.get(selectedModel.upstreamModel) ?? [];
  }, [directRoutesByAlias, selectedModel]);

  const selectedManagedByChannel = selectedModel ? (directRoutesByAlias.get(selectedModel.alias)?.length ?? 0) > 0 : false;

  const availableTargetAliases = useMemo(
    () =>
      initialModels
        .filter((item) => (directRoutesByAlias.get(item.alias)?.length ?? 0) > 0)
        .map((item) => item.alias)
        .sort((left, right) => left.localeCompare(right)),
    [directRoutesByAlias, initialModels],
  );

  const handleCreate = async () => {
    setError(null);
    setBusyAction("create");

    try {
      const payload = normalizeDraft(createDraft);
      if (!availableTargetAliases.includes(payload.target)) {
        throw new Error("目标模型必须是已接入渠道的内部模型");
      }

      const response = await fetch("/api/admin/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alias: payload.alias, upstream_model: payload.target }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "创建别名映射失败");
      }

      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建别名映射失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleUpdate = async () => {
    if (!selectedModel) {
      return;
    }

    setError(null);
    setBusyAction("update");

    try {
      const payload = normalizeDraft(editDraft);
      if (!availableTargetAliases.includes(payload.target)) {
        throw new Error("目标模型必须是已接入渠道的内部模型");
      }

      const response = await fetch(`/api/admin/models/${selectedModel.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alias: payload.alias, upstream_model: payload.target }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "更新别名映射失败");
      }

      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "更新别名映射失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleDelete = async () => {
    if (!selectedModel) {
      return;
    }

    if (!window.confirm(`确认删除对外模型 ${selectedModel.alias} 吗？`)) {
      return;
    }

    setError(null);
    setBusyAction("delete");

    try {
      const response = await fetch(`/api/admin/models/${selectedModel.id}`, { method: "DELETE" });
      if (!response.ok) {
        throw new Error((await response.text()) || "删除别名映射失败");
      }

      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "删除别名映射失败");
    } finally {
      setBusyAction(null);
    }
  };

  return (
    <PageContainer>
      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <section className="grid gap-6 xl:grid-cols-[320px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索对外别名或目标模型..." className="bg-card pl-9" />
            </div>
            <DetailDrawer
              title="新增对外模型别名"
              description="这里只定义对外模型名和内部目标模型，不在这里指定渠道。实际请求会自动复用目标模型当前可用的渠道路由。"
              triggerLabel="新增"
              trigger={
                <Button
                  size="icon"
                  variant="outline"
                  className="shrink-0"
                  onClick={() => {
                    setCreateDraft(createEmptyDraft());
                    setCreateOpen(true);
                  }}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              }
              open={createOpen}
              onOpenChange={setCreateOpen}
            >
              <div className="space-y-4 text-sm text-muted-foreground">
                <div className="space-y-2">
                  <label className="text-sm font-medium leading-none text-foreground">对外模型别名</label>
                  <Input value={createDraft.alias} onChange={(event) => setCreateDraft({ ...createDraft, alias: event.target.value })} placeholder="例如 aaa" />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium leading-none text-foreground">内部目标模型</label>
                  <Input value={createDraft.target} onChange={(event) => setCreateDraft({ ...createDraft, target: event.target.value })} placeholder="例如 gpt-5.4" />
                  <p className="text-xs text-muted-foreground">可用内部模型：{availableTargetAliases.join(" , ") || "暂无"}</p>
                </div>
                <div className="flex items-center justify-end gap-3 pt-2">
                  <Button variant="outline" onClick={() => setCreateOpen(false)} disabled={busyAction === "create"}>
                    取消
                  </Button>
                  <Button onClick={() => void handleCreate()} disabled={busyAction === "create"}>
                    {busyAction === "create" ? "提交中..." : "创建映射"}
                  </Button>
                </div>
              </div>
            </DetailDrawer>
          </div>

          <div className="flex flex-col gap-2">
            {filteredModels.map((item) => {
              const directCount = directRoutesByAlias.get(item.alias)?.length ?? 0;
              const resolvedCount = directCount > 0 ? directCount : directRoutesByAlias.get(item.upstreamModel)?.length ?? 0;
              return (
                <button
                  key={item.id}
                  onClick={() => setSelectedModelId(item.id)}
                  className={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-[background-color,border-color,transform,box-shadow] duration-200 ease-[var(--ease-out-smooth)] ${
                    item.id === selectedModel?.id ? "border-primary/30 bg-primary/5 ring-1 ring-primary/20" : "border-border bg-card hover:border-border/80 hover:bg-muted/50"
                  }`}
                >
                  <div className="flex w-full items-center justify-between gap-3">
                    <span className="font-medium text-foreground">{item.alias}</span>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{resolvedCount} 条路由</span>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <span>{item.alias}</span>
                    <ArrowRight className="h-3 w-3" />
                    <span className="truncate">{item.upstreamModel}</span>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <SectionCard
            title="别名映射"
            description="用户对外请求的模型名会先映射到内部目标模型，再复用该目标模型已有的渠道路由。"
            action={
              selectedModel && !selectedManagedByChannel ? (
                <DetailDrawer title="编辑别名映射" description="只编辑对外别名和目标模型，不直接编辑渠道。" triggerLabel="编辑" open={editOpen} onOpenChange={setEditOpen}>
                  <div className="space-y-4 text-sm text-muted-foreground">
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none text-foreground">对外模型别名</label>
                      <Input value={editDraft.alias} onChange={(event) => setEditDraft({ ...editDraft, alias: event.target.value })} />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none text-foreground">内部目标模型</label>
                      <Input value={editDraft.target} onChange={(event) => setEditDraft({ ...editDraft, target: event.target.value })} />
                      <p className="text-xs text-muted-foreground">可用内部模型：{availableTargetAliases.join(" , ") || "暂无"}</p>
                    </div>
                    <div className="flex items-center justify-between gap-3 pt-2">
                      <Button variant="danger" onClick={() => void handleDelete()} disabled={busyAction === "update" || busyAction === "delete"}>
                        <Trash2 className="mr-2 h-4 w-4" />
                        删除映射
                      </Button>
                      <div className="flex items-center gap-3">
                        <Button variant="outline" onClick={() => setEditOpen(false)} disabled={busyAction === "update" || busyAction === "delete"}>
                          取消
                        </Button>
                        <Button onClick={() => void handleUpdate()} disabled={busyAction === "update" || busyAction === "delete"}>
                          {busyAction === "update" ? "提交中..." : "保存映射"}
                        </Button>
                      </div>
                    </div>
                  </div>
                </DetailDrawer>
              ) : null
            }
          >
            {selectedModel ? (
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">对外模型别名</span>
                  <p className="text-base font-medium text-foreground">{selectedModel.alias}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">内部目标模型</span>
                  <p className="text-base font-medium text-foreground">{selectedModel.upstreamModel}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">当前可用渠道数</span>
                  <p className="text-base font-medium text-foreground">{selectedResolvedRoutes.length}</p>
                </div>
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">映射类型</span>
                  <p className="text-base font-medium text-foreground">{selectedManagedByChannel ? "渠道内置模型" : "自定义对外别名"}</p>
                </div>
                {selectedManagedByChannel ? (
                  <div className="md:col-span-2 rounded-lg border border-border/60 bg-muted/30 p-4 text-sm text-muted-foreground">
                    这条模型由渠道配置自动生成，当前页面只展示，不建议直接修改。若要扩展多个对外模型，请新增自定义别名并指向这个内部模型。
                  </div>
                ) : null}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">暂无模型映射数据。</div>
            )}
          </SectionCard>

          <SectionCard
            title="实际转发路由"
            description="这里展示当前别名最终会复用到哪些渠道路由。你不需要在这里手动指定渠道。"
          >
            {selectedModel ? (
              selectedResolvedRoutes.length > 0 ? (
                <div className="space-y-3">
                  {selectedResolvedRoutes.map((route) => (
                    <div key={route.id} className="rounded-lg border border-border/60 bg-muted/30 p-4">
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div className="text-sm font-medium text-foreground">{route.channel}</div>
                        <div className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">P{route.priority}</div>
                      </div>
                      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                        <span>调用方式：{route.invocationMode || "auto"}</span>
                        <span>回退：{route.fallback || "无"}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground">目标模型当前还没有可复用的渠道路由。</div>
              )
            ) : (
              <div className="text-sm text-muted-foreground">暂无路由数据。</div>
            )}
          </SectionCard>
        </div>
      </section>
    </PageContainer>
  );
}
