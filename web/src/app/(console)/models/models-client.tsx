"use client";

import { type ReactNode, useEffect, useMemo, useState } from "react";
import { ArrowRight, Plus, Search, Settings2, Trash2 } from "lucide-react";

import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

type ModelRouteRow = {
  id: number;
  modelId?: number;
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

type RouteDraft = {
  alias: string;
  target: string;
  channel: string;
  invocationMode: string;
  priority: string;
  fallback: string;
};

const invocationModeOptions = [
  { value: "auto", label: "自动" },
  { value: "openai", label: "OpenAI" },
  { value: "claude", label: "Claude" },
  { value: "gemini", label: "Gemini" },
];

function createEmptyDraft(channelNames: string[]): RouteDraft {
  return {
    alias: "",
    target: "",
    channel: channelNames[0] ?? "",
    invocationMode: "auto",
    priority: "1",
    fallback: "",
  };
}

function routeToDraft(route: ModelRouteRow): RouteDraft {
  return {
    alias: route.alias,
    target: route.target,
    channel: route.channel,
    invocationMode: route.invocationMode || "auto",
    priority: String(route.priority),
    fallback: route.fallback,
  };
}

function normalizeInvocationMode(value: string) {
  const normalized = value.trim().toLowerCase();
  if (normalized === "" || normalized === "auto") {
    return "";
  }
  return normalized;
}

function buildRoutePayload(draft: RouteDraft) {
  const alias = draft.alias.trim();
  const target = draft.target.trim();
  const channel = draft.channel.trim();
  const fallback = draft.fallback.trim();
  const priority = Number.parseInt(draft.priority.trim(), 10);

  if (!alias || !target || !channel) {
    throw new Error("模型别名、目标模型和目标渠道不能为空");
  }
  if (!Number.isFinite(priority) || priority <= 0) {
    throw new Error("优先级必须是大于 0 的整数");
  }

  return {
    alias,
    target,
    route: {
      model_alias: alias,
      channel_name: channel,
      invocation_mode: normalizeInvocationMode(draft.invocationMode),
      priority,
      fallback_model: fallback,
    },
  };
}

async function ensureOk(response: Response, fallbackMessage: string) {
  if (!response.ok) {
    throw new Error((await response.text()) || fallbackMessage);
  }
}

function RouteForm({
  draft,
  onChange,
  channelNames,
  aliasShared,
  busy,
  submitLabel,
  onSubmit,
  onCancel,
  dangerAction,
}: {
  draft: RouteDraft;
  onChange: (draft: RouteDraft) => void;
  channelNames: string[];
  aliasShared: boolean;
  busy: boolean;
  submitLabel: string;
  onSubmit: () => void;
  onCancel: () => void;
  dangerAction?: ReactNode;
}) {
  return (
    <div className="space-y-4 text-sm text-muted-foreground">
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">模型别名</label>
        <Input value={draft.alias} onChange={(event) => onChange({ ...draft, alias: event.target.value })} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">目标模型</label>
        <Input value={draft.target} onChange={(event) => onChange({ ...draft, target: event.target.value })} />
        {aliasShared ? (
          <p className="text-xs text-muted-foreground">当前别名已绑定多条路由，修改别名或目标模型会同步影响同别名的全部路由。</p>
        ) : null}
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">目标渠道</label>
        <Select value={draft.channel} onValueChange={(value) => onChange({ ...draft, channel: value })}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择目标渠道" />
          </SelectTrigger>
          <SelectContent>
            {channelNames.map((channel) => (
              <SelectItem key={channel} value={channel}>
                {channel}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">优先级</label>
        <Input type="number" min={1} value={draft.priority} onChange={(event) => onChange({ ...draft, priority: event.target.value })} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">调用方式</label>
        <Select value={draft.invocationMode} onValueChange={(value) => onChange({ ...draft, invocationMode: value })}>
          <SelectTrigger className="bg-muted/30">
            <SelectValue placeholder="选择调用方式" />
          </SelectTrigger>
          <SelectContent>
            {invocationModeOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium leading-none text-foreground">回退模型</label>
        <Input value={draft.fallback} onChange={(event) => onChange({ ...draft, fallback: event.target.value })} placeholder="可留空" />
      </div>
      <div className="flex items-center justify-between gap-3 pt-2">
        <div>{dangerAction}</div>
        <div className="flex items-center gap-3">
          <Button variant="outline" onClick={onCancel} disabled={busy}>
            取消
          </Button>
          <Button onClick={onSubmit} disabled={busy}>
            {busy ? "提交中..." : submitLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

export function ModelsClient({
  eyebrow,
  title,
  description,
  initialRoutes,
  initialModels,
  channelNames,
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialRoutes: ModelRouteRow[];
  initialModels: ModelMappingSummary[];
  channelNames: string[];
}) {
  const [selectedRouteId, setSelectedRouteId] = useState<number | null>(initialRoutes[0]?.id ?? null);
  const [keyword, setKeyword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [busyAction, setBusyAction] = useState<"create" | "update" | "delete" | null>(null);
  const [createDraft, setCreateDraft] = useState<RouteDraft>(() => createEmptyDraft(channelNames));
  const [editDraft, setEditDraft] = useState<RouteDraft>(() => (initialRoutes[0] ? routeToDraft(initialRoutes[0]) : createEmptyDraft(channelNames)));

  const filteredRoutes = useMemo(() => {
    const query = keyword.toLowerCase();
    return initialRoutes.filter((route) => route.alias.toLowerCase().includes(query) || route.target.toLowerCase().includes(query) || route.channel.toLowerCase().includes(query));
  }, [initialRoutes, keyword]);

  const aliasRouteCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const route of initialRoutes) {
      counts.set(route.alias, (counts.get(route.alias) ?? 0) + 1);
    }
    return counts;
  }, [initialRoutes]);

  const selectedRoute = filteredRoutes.find((route) => route.id === selectedRouteId) ?? filteredRoutes[0] ?? null;
  const selectedAliasShared = selectedRoute ? (aliasRouteCounts.get(selectedRoute.alias) ?? 0) > 1 : false;

  useEffect(() => {
    if (!selectedRoute) {
      setSelectedRouteId(null);
      return;
    }
    if (selectedRoute.id !== selectedRouteId) {
      setSelectedRouteId(selectedRoute.id);
    }
  }, [selectedRoute, selectedRouteId]);

  useEffect(() => {
    if (selectedRoute) {
      setEditDraft(routeToDraft(selectedRoute));
    }
  }, [selectedRoute]);

  const handleCreate = async () => {
    setError(null);
    setBusyAction("create");

    let createdModelId: number | null = null;
    try {
      const payload = buildRoutePayload(createDraft);
      const existingModel = initialModels.find((item) => item.alias === payload.alias);

      if (existingModel && existingModel.upstreamModel !== payload.target) {
        throw new Error("当前数据模型下，同一别名只能映射一个目标模型");
      }

      if (!existingModel) {
        const createModelResponse = await fetch("/api/admin/models", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ alias: payload.alias, upstream_model: payload.target }),
        });
        await ensureOk(createModelResponse, "创建模型映射失败");
        const createdModel = (await createModelResponse.json()) as { id: number };
        createdModelId = createdModel.id;
      }

      const createRouteResponse = await fetch("/api/admin/model-routes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload.route),
      });
      await ensureOk(createRouteResponse, "创建路由失败");
      window.location.reload();
    } catch (requestError) {
      if (createdModelId !== null) {
        await fetch(`/api/admin/models/${createdModelId}`, { method: "DELETE" });
      }
      setError(requestError instanceof Error ? requestError.message : "创建路由失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleUpdate = async () => {
    if (!selectedRoute) {
      return;
    }

    setError(null);
    setBusyAction("update");

    try {
      const payload = buildRoutePayload(editDraft);
      const updateRouteResponse = await fetch(`/api/admin/model-route-bindings/${selectedRoute.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          alias: payload.alias,
          upstream_model: payload.target,
          channel_name: payload.route.channel_name,
          invocation_mode: payload.route.invocation_mode,
          priority: payload.route.priority,
          fallback_model: payload.route.fallback_model,
        }),
      });
      await ensureOk(updateRouteResponse, "更新模型路由失败");
      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "更新路由失败");
    } finally {
      setBusyAction(null);
    }
  };

  const handleDelete = async () => {
    if (!selectedRoute) {
      return;
    }

    if (!window.confirm(`确认删除路由 ${selectedRoute.alias} → ${selectedRoute.channel} 吗？`)) {
      return;
    }

    setError(null);
    setBusyAction("delete");

    try {
      const routeCount = aliasRouteCounts.get(selectedRoute.alias) ?? 0;
      if (routeCount <= 1 && selectedRoute.modelId) {
        const deleteModelResponse = await fetch(`/api/admin/models/${selectedRoute.modelId}`, { method: "DELETE" });
        await ensureOk(deleteModelResponse, "删除模型映射失败");
      } else {
        const deleteRouteResponse = await fetch(`/api/admin/model-routes/${selectedRoute.id}`, { method: "DELETE" });
        await ensureOk(deleteRouteResponse, "删除路由失败");
      }
      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "删除路由失败");
    } finally {
      setBusyAction(null);
    }
  };

  return (
    <PageContainer>
      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <section className="grid gap-6 xl:grid-cols-[300px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索别名、目标模型或渠道..." className="bg-card pl-9" />
            </div>
            <DetailDrawer
              title="新增路由规则"
              description="为一个模型别名新增路由规则。若别名不存在，会先创建模型映射。"
              triggerLabel="新增"
              trigger={
                <Button
                  size="icon"
                  variant="outline"
                  className="shrink-0"
                  onClick={() => {
                    setCreateDraft(createEmptyDraft(channelNames));
                    setCreateOpen(true);
                  }}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              }
              open={createOpen}
              onOpenChange={setCreateOpen}
            >
              <RouteForm
                draft={createDraft}
                onChange={setCreateDraft}
                channelNames={channelNames}
                aliasShared={false}
                busy={busyAction === "create"}
                submitLabel="创建路由"
                onSubmit={() => void handleCreate()}
                onCancel={() => setCreateOpen(false)}
              />
            </DetailDrawer>
          </div>

          <div className="flex flex-col gap-2">
            {filteredRoutes.map((route) => (
              <button
                key={route.id}
                onClick={() => setSelectedRouteId(route.id)}
                className={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-[background-color,border-color,transform,box-shadow] duration-200 ease-[var(--ease-out-smooth)] ${
                  route.id === selectedRoute?.id ? "border-primary/30 bg-primary/5 ring-1 ring-primary/20" : "border-border bg-card hover:border-border/80 hover:bg-muted/50"
                }`}
              >
                <div className="flex w-full items-center justify-between gap-3">
                  <span className="font-medium text-foreground">{route.alias}</span>
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">P{route.priority}</span>
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
            description="当前选中路由的可编辑配置。"
            action={
              selectedRoute ? (
                <DetailDrawer
                  title="编辑路由规则"
                  description="修改模型别名、目标模型、目标渠道和优先级。"
                  triggerLabel="编辑"
                  open={editOpen}
                  onOpenChange={setEditOpen}
                >
                  <RouteForm
                    draft={editDraft}
                    onChange={setEditDraft}
                    channelNames={channelNames}
                    aliasShared={selectedAliasShared}
                    busy={busyAction === "update" || busyAction === "delete"}
                    submitLabel="保存更改"
                    onSubmit={() => void handleUpdate()}
                    onCancel={() => setEditOpen(false)}
                    dangerAction={
                      <Button variant="danger" onClick={() => void handleDelete()} disabled={busyAction === "update" || busyAction === "delete"}>
                        <Trash2 className="mr-2 h-4 w-4" />
                        删除路由
                      </Button>
                    }
                  />
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
                  <p className="text-base font-medium text-foreground">P{selectedRoute.priority}</p>
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
                <div className="space-y-1">
                  <span className="text-sm font-medium text-muted-foreground">同别名路由数</span>
                  <p className="text-base font-medium text-foreground">{aliasRouteCounts.get(selectedRoute.alias) ?? 1}</p>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">暂无模型路由数据。</div>
            )}
          </SectionCard>

          <SectionCard
            title="回退策略 (Fallback)"
            description="当前仅支持配置与展示，运行时 fallback 链路将在后续阶段接入。"
            action={
              <Button variant="outline" size="sm" className="gap-2" onClick={() => selectedRoute && setEditOpen(true)} disabled={!selectedRoute}>
                <Settings2 className="h-4 w-4" />
                配置回退
              </Button>
            }
          >
            {selectedRoute ? (
              <div className="rounded-lg border border-border bg-muted/30 p-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="text-sm font-medium text-foreground">默认回退模型</p>
                    <p className="mt-1 text-sm text-muted-foreground">当前配置为 {selectedRoute.fallback || "未配置"}</p>
                  </div>
                  <span
                    className={`rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset ${
                      selectedRoute.fallback ? "bg-success/10 text-success ring-success/20" : "bg-muted text-muted-foreground ring-border"
                    }`}
                  >
                    {selectedRoute.fallback ? "已配置" : "未配置"}
                  </span>
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
