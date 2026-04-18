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

type ModelTargetGroup = {
  key: string;
  target: string;
  routes: ModelRouteRow[];
  models: ModelMappingSummary[];
  customAliases: ModelMappingSummary[];
  systemModel: ModelMappingSummary | null;
};

type AliasDraft = {
  alias: string;
  target: string;
};

type TargetDraft = {
  modelId: string;
};

function createEmptyAliasDraft(): AliasDraft {
  return {
    alias: "",
    target: "",
  };
}

function createEmptyTargetDraft(): TargetDraft {
  return {
    modelId: "",
  };
}

function mappingToDraft(mapping: ModelMappingSummary): AliasDraft {
  return {
    alias: mapping.alias,
    target: mapping.upstreamModel,
  };
}

function normalizeAliasDraft(draft: AliasDraft) {
  const alias = draft.alias.trim();
  const target = draft.target.trim();

  if (!alias || !target) {
    throw new Error("对外模型别名和目标模型不能为空");
  }

  return { alias, target };
}

function buildTargetGroups(
  models: ModelMappingSummary[],
  directRoutesByAlias: Map<string, ModelRouteRow[]>,
): ModelTargetGroup[] {
  const groups = new Map<string, ModelTargetGroup>();

  for (const model of models) {
    const hasDirectRoutes = (directRoutesByAlias.get(model.alias)?.length ?? 0) > 0;
    const groupKey = hasDirectRoutes ? model.alias : model.upstreamModel;
    const current = groups.get(groupKey) ?? {
      key: groupKey,
      target: groupKey,
      routes: directRoutesByAlias.get(groupKey) ?? [],
      models: [],
      customAliases: [],
      systemModel: null,
    };

    current.models.push(model);
    if (model.alias === groupKey) {
      current.systemModel = model;
    } else {
      current.customAliases.push(model);
    }

    groups.set(groupKey, current);
  }

  return Array.from(groups.values())
    .map((group) => ({
      ...group,
      models: [...group.models].sort((left, right) => left.alias.localeCompare(right.alias)),
      customAliases: [...group.customAliases].sort((left, right) => left.alias.localeCompare(right.alias)),
      routes: [...group.routes].sort((left, right) => left.priority - right.priority || left.channel.localeCompare(right.channel)),
    }))
    .sort((left, right) => left.target.localeCompare(right.target));
}

function getAliasTagClassName(index: number) {
  const variants = [
    "border-info/30 bg-info/10 text-info",
    "border-success/30 bg-success/10 text-success",
    "border-warning/30 bg-warning/10 text-warning",
    "border-primary/30 bg-primary/10 text-primary",
  ];

  return variants[index % variants.length];
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
  const [selectedGroupKey, setSelectedGroupKey] = useState<string | null>(null);
  const [editingModelId, setEditingModelId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [createTargetOpen, setCreateTargetOpen] = useState(false);
  const [createAliasOpen, setCreateAliasOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [busyAction, setBusyAction] = useState<"createTarget" | "createAlias" | "update" | "delete" | null>(null);
  const [createTargetDraft, setCreateTargetDraft] = useState<TargetDraft>(() => createEmptyTargetDraft());
  const [createAliasDraft, setCreateAliasDraft] = useState<AliasDraft>(() => createEmptyAliasDraft());
  const [editDraft, setEditDraft] = useState<AliasDraft>(() => createEmptyAliasDraft());

  const directRoutesByAlias = useMemo(() => {
    const map = new Map<string, ModelRouteRow[]>();
    for (const route of initialRoutes) {
      const current = map.get(route.alias) ?? [];
      current.push(route);
      map.set(route.alias, current);
    }
    return map;
  }, [initialRoutes]);

  const targetGroups = useMemo(() => buildTargetGroups(initialModels, directRoutesByAlias), [directRoutesByAlias, initialModels]);

  const filteredGroups = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) {
      return targetGroups;
    }

    return targetGroups.filter((group) => {
      if (group.target.toLowerCase().includes(query)) {
        return true;
      }

      return group.models.some(
        (item) => item.alias.toLowerCase().includes(query) || item.upstreamModel.toLowerCase().includes(query),
      );
    });
  }, [keyword, targetGroups]);

  const selectedGroup = filteredGroups.find((group) => group.key === selectedGroupKey) ?? filteredGroups[0] ?? null;
  const editingModel = initialModels.find((item) => item.id === editingModelId) ?? null;

  useEffect(() => {
    if (!selectedGroup) {
      setSelectedGroupKey(null);
      return;
    }

    if (selectedGroup.key !== selectedGroupKey) {
      setSelectedGroupKey(selectedGroup.key);
    }
  }, [selectedGroup, selectedGroupKey]);

  useEffect(() => {
    if (editingModel) {
      setEditDraft(mappingToDraft(editingModel));
    }
  }, [editingModel]);

  const availableTargetAliases = useMemo(
    () =>
      initialModels
        .filter((item) => (directRoutesByAlias.get(item.alias)?.length ?? 0) > 0)
        .map((item) => item.alias)
        .sort((left, right) => left.localeCompare(right)),
    [directRoutesByAlias, initialModels],
  );

  const handleCreateTarget = async () => {
    setError(null);
    setBusyAction("createTarget");

    try {
      const modelId = createTargetDraft.modelId.trim();
      if (!modelId) {
        throw new Error("目标模型不能为空");
      }

      const response = await fetch("/api/admin/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alias: modelId, upstream_model: modelId }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "创建目标模型失败");
      }

      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建目标模型失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateAlias = async () => {
    if (!selectedGroup) {
      return;
    }

    setError(null);
    setBusyAction("createAlias");

    try {
      const payload = normalizeAliasDraft({ ...createAliasDraft, target: selectedGroup.target });
      if (payload.alias === selectedGroup.target) {
        throw new Error("转发模型不能与目标模型相同");
      }

      const response = await fetch("/api/admin/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alias: payload.alias, upstream_model: payload.target }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || "创建转发模型失败");
      }

      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建转发模型失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleUpdate = async () => {
    if (!editingModel) {
      return;
    }

    setError(null);
    setBusyAction("update");

    try {
      const payload = normalizeAliasDraft(editDraft);
      if (!availableTargetAliases.includes(payload.target)) {
        throw new Error("目标模型必须是已接入渠道的内部模型");
      }

      const response = await fetch(`/api/admin/models/${editingModel.id}`, {
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
    if (!editingModel) {
      return;
    }

    if (!window.confirm(`确认删除对外模型 ${editingModel.alias} 吗？`)) {
      return;
    }

    setError(null);
    setBusyAction("delete");

    try {
      const response = await fetch(`/api/admin/models/${editingModel.id}`, { method: "DELETE" });
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

      <section className="grid gap-6 xl:grid-cols-[360px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="搜索目标模型或转发别名..."
                className="bg-card pl-9"
              />
            </div>
            <DetailDrawer
              title="新增目标模型"
              description="这里只添加当前站点内可用的目标模型本体，不创建转发别名。"
              triggerLabel="新增"
              trigger={
                <Button
                  size="icon"
                  variant="outline"
                  className="shrink-0"
                  onClick={() => {
                    setCreateTargetDraft(createEmptyTargetDraft());
                    setCreateTargetOpen(true);
                  }}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              }
              open={createTargetOpen}
              onOpenChange={setCreateTargetOpen}
            >
              <div className="space-y-4 text-sm text-muted-foreground">
                <div className="space-y-2">
                  <label className="text-sm font-medium leading-none text-foreground">目标模型 ID</label>
                  <Input
                    value={createTargetDraft.modelId}
                    onChange={(event) => setCreateTargetDraft({ modelId: event.target.value })}
                    placeholder="例如 gpt-5.4"
                  />
                  <p className="text-xs text-muted-foreground">当前站点可用模型：{availableTargetAliases.join(" , ") || "暂无"}</p>
                </div>
                <div className="flex items-center justify-end gap-3 pt-2">
                  <Button variant="outline" onClick={() => setCreateTargetOpen(false)} disabled={busyAction === "createTarget"}>
                    取消
                  </Button>
                  <Button onClick={() => void handleCreateTarget()} disabled={busyAction === "createTarget"}>
                    {busyAction === "createTarget" ? "提交中..." : "创建目标模型"}
                  </Button>
                </div>
              </div>
            </DetailDrawer>
          </div>

          <div className="flex flex-col gap-2">
            {filteredGroups.map((group) => {
              const isSelected = group.key === selectedGroup?.key;
              const visibleAliases = group.customAliases.slice(0, 3);
              const hiddenAliasCount = Math.max(group.customAliases.length - visibleAliases.length, 0);

              return (
                <button
                  key={group.key}
                  onClick={() => setSelectedGroupKey(group.key)}
                  className={`flex flex-col items-start gap-3 rounded-lg border p-3 text-left transition-[background-color,border-color,transform,box-shadow] duration-200 ease-[var(--ease-out-smooth)] ${
                    isSelected ? "border-primary/30 bg-primary/5 ring-1 ring-primary/20" : "border-border bg-card hover:border-border/80 hover:bg-muted/50"
                  }`}
                >
                  <div className="flex w-full items-center justify-between gap-3">
                    <span className="font-medium text-foreground">{group.target}</span>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{group.routes.length} 条路由</span>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <span>{group.customAliases.length} 个转发别名</span>
                    <span>•</span>
                    <span>{group.systemModel ? "渠道目标模型" : "复用目标模型"}</span>
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {visibleAliases.length > 0 ? (
                      <>
                        {visibleAliases.map((item, index) => (
                          <span
                            key={item.id}
                            className={`rounded-full border px-2 py-0.5 text-xs font-medium ${getAliasTagClassName(index)}`}
                          >
                            {item.alias}
                          </span>
                        ))}
                        {hiddenAliasCount > 0 ? (
                          <span className="rounded-full border border-border/60 bg-background/70 px-2 py-0.5 text-xs font-medium text-muted-foreground">
                            +{hiddenAliasCount}
                          </span>
                        ) : null}
                      </>
                    ) : (
                      <span className="text-xs text-muted-foreground">当前没有额外转发别名</span>
                    )}
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <SectionCard
            title="转发别名"
            description="这里展示当前目标模型下，哪些对外模型会复用这组渠道路由。"
            action={
              selectedGroup ? (
                <DetailDrawer
                  title="添加转发模型"
                  description="为当前目标模型新增一个对外转发别名。"
                  triggerLabel="添加转发模型"
                  trigger={
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setCreateAliasDraft({ alias: "", target: selectedGroup.target });
                        setCreateAliasOpen(true);
                      }}
                    >
                      添加转发模型
                    </Button>
                  }
                  open={createAliasOpen}
                  onOpenChange={setCreateAliasOpen}
                >
                  <div className="space-y-4 text-sm text-muted-foreground">
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none text-foreground">目标模型</label>
                      <Input value={selectedGroup.target} readOnly />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium leading-none text-foreground">转发模型 ID</label>
                      <Input
                        value={createAliasDraft.alias}
                        onChange={(event) => setCreateAliasDraft({ ...createAliasDraft, alias: event.target.value })}
                        placeholder="例如 my-gemini"
                      />
                    </div>
                    <div className="flex items-center justify-end gap-3 pt-2">
                      <Button variant="outline" onClick={() => setCreateAliasOpen(false)} disabled={busyAction === "createAlias"}>
                        取消
                      </Button>
                      <Button onClick={() => void handleCreateAlias()} disabled={busyAction === "createAlias"}>
                        {busyAction === "createAlias" ? "提交中..." : "创建转发模型"}
                      </Button>
                    </div>
                  </div>
                </DetailDrawer>
              ) : null
            }
          >
            {selectedGroup ? (
              <div className="space-y-5">
                <div className="grid gap-4 md:grid-cols-3">
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-4">
                    <div className="text-sm font-medium text-muted-foreground">目标模型</div>
                    <div className="mt-2 text-base font-medium text-foreground">{selectedGroup.target}</div>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-4">
                    <div className="text-sm font-medium text-muted-foreground">转发别名数</div>
                    <div className="mt-2 text-base font-medium text-foreground">{selectedGroup.customAliases.length}</div>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-4">
                    <div className="text-sm font-medium text-muted-foreground">可用渠道数</div>
                    <div className="mt-2 text-base font-medium text-foreground">{selectedGroup.routes.length}</div>
                  </div>
                </div>

                {selectedGroup.customAliases.length > 0 ? (
                  <div className="space-y-3">
                    {selectedGroup.customAliases.map((item) => (
                      <div key={item.id} className="rounded-lg border border-border/60 bg-muted/30 p-4">
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div className="space-y-2">
                            <div className="text-sm font-medium text-foreground">{item.alias}</div>
                            <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                              <span>{item.alias}</span>
                              <ArrowRight className="h-3 w-3" />
                              <span>{selectedGroup.target}</span>
                            </div>
                          </div>
                          <DetailDrawer
                            title="编辑别名映射"
                            description="只编辑对外别名和目标模型，不直接编辑渠道。"
                            triggerLabel="编辑"
                            trigger={
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                  setEditingModelId(item.id);
                                  setEditOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                            }
                            open={editOpen && editingModelId === item.id}
                            onOpenChange={(open) => {
                              setEditOpen(open);
                              setEditingModelId(open ? item.id : null);
                            }}
                          >
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
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-4 text-sm text-muted-foreground">
                    当前没有额外转发别名，请求会直接使用目标模型名进入这组渠道路由。
                  </div>
                )}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">暂无模型映射数据。</div>
            )}
          </SectionCard>
        </div>
      </section>
    </PageContainer>
  );
}
