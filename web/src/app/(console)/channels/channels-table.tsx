"use client";

import { useMemo, useState } from "react";
import { Check, Search } from "lucide-react";

import { ChannelTestDialog } from "@/app/(console)/channels/channel-test-dialog";
import { DeleteChannelButton } from "@/app/(console)/channels/delete-channel-button";
import { EditChannelDrawer } from "@/app/(console)/channels/edit-channel-drawer";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { EmptyState } from "@/components/shared/empty-state";
import { NoticeDialog } from "@/components/shared/notice-dialog";
import { ProviderBrandIcon } from "@/components/shared/provider-brand-icon";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

type ChannelRow = {
  id: number;
  name: string;
  provider: string;
  status: string;
  endpoint: string;
  models: number;
  modelIds: string[];
  rpmLimit: number;
  maxInflight: number;
  safetyFactor: number;
  enabledForAsync: boolean;
  dispatchWeight: number;
  updatedAt: string;
};

export function ChannelsTable({ rows }: { rows: ChannelRow[] }) {
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [keyword, setKeyword] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [providerFilter, setProviderFilter] = useState("all");
  const [sortOrder, setSortOrder] = useState("updated_desc");

  const providers = useMemo(
    () => Array.from(new Set(rows.map((row) => row.provider))).sort((left, right) => left.localeCompare(right)),
    [rows],
  );

  const filteredRows = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    const nextRows = rows.filter((row) => {
      const matchesKeyword =
        normalizedKeyword.length === 0 ||
        row.name.toLowerCase().includes(normalizedKeyword) ||
        row.endpoint.toLowerCase().includes(normalizedKeyword) ||
        row.provider.toLowerCase().includes(normalizedKeyword) ||
        row.modelIds.some((modelId) => modelId.toLowerCase().includes(normalizedKeyword));
      const matchesStatus = statusFilter === "all" || row.status === statusFilter;
      const matchesProvider = providerFilter === "all" || row.provider === providerFilter;

      return matchesKeyword && matchesStatus && matchesProvider;
    });

    nextRows.sort((left, right) => {
      switch (sortOrder) {
        case "updated_asc":
          return left.updatedAt.localeCompare(right.updatedAt);
        case "name_asc":
          return left.name.localeCompare(right.name);
        case "models_desc":
          return right.models - left.models;
        default:
          return right.updatedAt.localeCompare(left.updatedAt);
      }
    });

    return nextRows;
  }, [keyword, providerFilter, rows, sortOrder, statusFilter]);

  const visibleIds = useMemo(() => filteredRows.map((row) => row.id), [filteredRows]);
  const visibleSelectedCount = useMemo(
    () => visibleIds.filter((id) => selectedIds.includes(id)).length,
    [selectedIds, visibleIds],
  );
  const allSelected = filteredRows.length > 0 && visibleSelectedCount === filteredRows.length;

  const toggleSelect = (channelId: number) => {
    setSelectedIds((current) =>
      current.includes(channelId) ? current.filter((id) => id !== channelId) : [...current, channelId],
    );
  };

  const handleToggleSelectAll = () => {
    setSelectedIds((current) => {
      if (allSelected) {
        return current.filter((id) => !visibleIds.includes(id));
      }

      return Array.from(new Set([...current, ...visibleIds]));
    });
  };

  const handleDeleteSelected = async () => {
    const idsToDelete = [...selectedIds];
    if (idsToDelete.length === 0) {
      return;
    }

    setIsDeleting(true);
    try {
      const results = await Promise.all(
        idsToDelete.map(async (id) => {
          const response = await fetch(`/api/admin/channels/${id}`, { method: "DELETE" });
          if (!response.ok) {
            throw new Error(await response.text());
          }
          return id;
        }),
      );

      if (results.length > 0) {
        window.location.reload();
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "批量删除失败");
    } finally {
      setIsDeleting(false);
    }
  };

  if (rows.length === 0) {
    return null;
  }

  return (
    <>
      <div className="flex flex-col gap-3 rounded-xl border border-border bg-muted/30 p-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-1 flex-col gap-3 lg:flex-row lg:items-center">
          <div className="relative w-full lg:max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索渠道名或地址" className="bg-background pl-9" />
          </div>
          <div className="grid gap-2 sm:grid-cols-3">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="min-w-[148px] bg-background/90 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]">
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent className="border-white/10 bg-[linear-gradient(180deg,rgba(20,20,20,0.96),rgba(8,8,8,0.98))] shadow-[0_20px_48px_rgba(0,0,0,0.34)] backdrop-blur-xl">
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="启用">启用</SelectItem>
                <SelectItem value="禁用">禁用</SelectItem>
              </SelectContent>
            </Select>

            <Select value={providerFilter} onValueChange={setProviderFilter}>
              <SelectTrigger className="min-w-[148px] bg-background/90 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]">
                <SelectValue placeholder="全部 Provider" />
              </SelectTrigger>
              <SelectContent className="border-white/10 bg-[linear-gradient(180deg,rgba(20,20,20,0.96),rgba(8,8,8,0.98))] shadow-[0_20px_48px_rgba(0,0,0,0.34)] backdrop-blur-xl">
                <SelectItem value="all">全部 Provider</SelectItem>
                {providers.map((provider) => (
                  <SelectItem key={provider} value={provider}>
                    {provider}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={sortOrder} onValueChange={setSortOrder}>
              <SelectTrigger className="min-w-[148px] bg-background/90 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]">
                <SelectValue placeholder="最近更新" />
              </SelectTrigger>
              <SelectContent className="border-white/10 bg-[linear-gradient(180deg,rgba(20,20,20,0.96),rgba(8,8,8,0.98))] shadow-[0_20px_48px_rgba(0,0,0,0.34)] backdrop-blur-xl">
                <SelectItem value="updated_desc">最近更新</SelectItem>
                <SelectItem value="updated_asc">最早更新</SelectItem>
                <SelectItem value="name_asc">按名称排序</SelectItem>
                <SelectItem value="models_desc">模型数最多</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between gap-3">
        <div className="text-sm text-muted-foreground">
          显示 {filteredRows.length} / {rows.length} 个渠道，已选 {selectedIds.length} 个
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleToggleSelectAll}>
            {allSelected ? "取消全选" : "全选"}
          </Button>
          <ConfirmDialog
            trigger={
              <Button variant="danger" disabled={selectedIds.length === 0 || isDeleting}>
                {isDeleting ? "删除中..." : allSelected ? "删除全部渠道" : "删除已选"}
              </Button>
            }
            title={allSelected ? "确认删除全部渠道" : "确认删除已选渠道"}
            description={
              allSelected
                ? "当前将删除全部渠道记录，此操作不可恢复。"
                : `当前将删除已选中的 ${selectedIds.length} 个渠道，此操作不可恢复。`
            }
            confirmLabel={allSelected ? "确认删除全部" : "确认删除"}
            onConfirm={handleDeleteSelected}
          />
        </div>
      </div>

      {filteredRows.length === 0 ? (
        <div className="mt-4">
          <EmptyState title="没有符合条件的渠道" description="调整搜索词、状态、Provider 或排序条件后再试。" />
        </div>
      ) : null}

      {filteredRows.length > 0 ? (
        <div className="mt-4 overflow-hidden rounded-xl border border-border bg-background shadow-sm">
          <table className="min-w-full divide-y divide-border/50 text-sm">
            <thead className="bg-secondary/30 text-left text-muted-foreground">
              <tr>
                <th className="w-16 px-3 py-3.5 text-center font-medium">选择</th>
                <th className="px-4 py-3.5 font-medium">渠道名</th>
                <th className="px-4 py-3.5 font-medium">类型</th>
                <th className="px-4 py-3.5 font-medium">状态</th>
                <th className="px-4 py-3.5 font-medium">地址</th>
                <th className="px-4 py-3.5 font-medium">模型数</th>
                <th className="px-4 py-3.5 font-medium">RPM / Inflight</th>
                <th className="px-4 py-3.5 font-medium">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/50 bg-background">
              {filteredRows.map((row) => {
                const checked = selectedIds.includes(row.id);
                return (
                  <tr key={row.id} className="transition-[background-color] duration-200 ease-[var(--ease-out-smooth)] hover:bg-secondary/30">
                    <td className="w-16 px-3 py-3.5 text-center align-middle text-foreground">
                      <button
                        type="button"
                        aria-label={checked ? `取消选择渠道 ${row.name}` : `选择渠道 ${row.name}`}
                        aria-pressed={checked}
                        onClick={() => toggleSelect(row.id)}
                        className={`inline-flex h-4.5 w-4.5 items-center justify-center rounded-[5px] border transition-[background-color,border-color,color,transform,box-shadow] duration-200 ease-[var(--ease-out-smooth)] ${
                          checked
                            ? "border-foreground bg-foreground text-background shadow-[0_0_0_1px_rgba(255,255,255,0.04)]"
                            : "border-border/80 bg-transparent text-transparent hover:border-foreground/60 hover:bg-foreground/5"
                        }`}
                      >
                        <Check className="h-3 w-3 stroke-[3]" />
                      </button>
                    </td>
                    <td className="px-4 py-3.5 text-foreground">
                      <span className="inline-flex items-center gap-3 font-medium text-foreground">
                        <ProviderBrandIcon provider={row.provider.replace(" Compatible", "").replace("Anthropic", "Claude").replace("Gemini", "Gemini")} />
                        <span>{row.name}</span>
                      </span>
                    </td>
                    <td className="px-4 py-3.5 text-foreground">{row.provider}</td>
                    <td className="px-4 py-3.5 text-foreground">
                      <StatusBadge status={row.status} />
                    </td>
                    <td className="px-4 py-3.5 text-foreground">
                      <span className="font-mono text-xs text-muted-foreground">{row.endpoint}</span>
                    </td>
                    <td className="px-4 py-3.5 text-foreground">{row.models}</td>
                    <td className="px-4 py-3.5 text-foreground">
                      <span className="font-mono text-xs text-muted-foreground">
                        {row.rpmLimit} / {row.maxInflight}
                      </span>
                    </td>
                    <td className="px-4 py-3.5 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <ChannelTestDialog row={row} />
                        <DetailDrawer title={row.name} description="这里会承载渠道编辑表单、密钥配置和测试连接操作。" triggerLabel="查看详情">
                          <div className="space-y-6">
                            <div>
                              <h3 className="text-sm font-medium text-foreground">基本信息</h3>
                              <div className="mt-3 rounded-xl border border-border bg-card">
                                <dl className="divide-y divide-border text-sm">
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">渠道类型</dt>
                                    <dd className="col-span-2 text-foreground">{row.provider}</dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">请求地址</dt>
                                    <dd className="col-span-2 font-mono text-foreground">{row.endpoint}</dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">当前状态</dt>
                                    <dd className="col-span-2">
                                      <StatusBadge status={row.status} />
                                    </dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">覆盖模型数</dt>
                                    <dd className="col-span-2 text-foreground">{row.models}</dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">每分钟额度</dt>
                                    <dd className="col-span-2 text-foreground">{row.rpmLimit}</dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">最大 Inflight</dt>
                                    <dd className="col-span-2 text-foreground">{row.maxInflight}</dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">安全系数 / 权重</dt>
                                    <dd className="col-span-2 text-foreground">
                                      {row.safetyFactor} / {row.dispatchWeight}
                                    </dd>
                                  </div>
                                  <div className="grid grid-cols-3 gap-4 px-4 py-3">
                                    <dt className="font-medium text-muted-foreground">异步受理</dt>
                                    <dd className="col-span-2 text-foreground">{row.enabledForAsync ? "启用" : "禁用"}</dd>
                                  </div>
                                </dl>
                              </div>
                            </div>
                            <div className="flex justify-end gap-3">
                              <EditChannelDrawer row={row} />
                              <DeleteChannelButton row={row} />
                            </div>
                          </div>
                        </DetailDrawer>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}

      <NoticeDialog
        open={error !== null}
        onOpenChange={(open) => {
          if (!open) {
            setError(null);
          }
        }}
        title="批量删除失败"
        description={error ?? ""}
      />
    </>
  );
}
