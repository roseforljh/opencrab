"use client";

import { useMemo, useState } from "react";
import { Check, Copy, CopyCheck } from "lucide-react";

import { ApiKeyManager } from "@/app/(console)/api-keys/api-key-manager";
import { NewApiKeyForm, type NewApiKeyDraft } from "@/app/(console)/api-keys/new-api-key-form";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { StatCard } from "@/components/shared/stat-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

type ApiKeyRow = {
  id: number;
  name: string;
  rawKey?: string;
  status: string;
  channelNames: string[];
  modelAliases: string[];
};

type StatusFilter = "all" | "enabled" | "disabled";
type SortOption = "id_desc" | "id_asc" | "name_asc" | "name_desc";

function toPreview(rawKey?: string) {
  if (!rawKey) {
    return "仅创建时显示";
  }

  const head = rawKey.slice(0, 14);
  const tail = rawKey.slice(-4);
  return `${head}...${tail}`;
}

function renderScope(values: string[]) {
  if (values.length === 0) {
    return "全部";
  }
  return values.join(" / ");
}

export function ApiKeysClient({
  eyebrow,
  title,
  description,
  initialRows,
  requiresSecondaryPassword,
  channelOptions,
  modelOptions,
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialRows: ApiKeyRow[];
  requiresSecondaryPassword: boolean;
  channelOptions: string[];
  modelOptions: string[];
}) {
  const [rows, setRows] = useState<ApiKeyRow[]>(initialRows);
  const [createOpen, setCreateOpen] = useState(false);
  const [copiedId, setCopiedId] = useState<number | null>(null);
  const [searchValue, setSearchValue] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [sortOption, setSortOption] = useState<SortOption>("id_desc");
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [batchDeleting, setBatchDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const stats = useMemo(() => {
    const enabled = rows.filter((row) => row.status === "启用").length;
    const disabled = rows.length - enabled;
    return { total: rows.length, enabled, disabled };
  }, [rows]);

  const filteredRows = useMemo(() => {
    const normalizedQuery = searchValue.trim().toLowerCase();
    const nextRows = rows.filter((row) => {
      if (statusFilter === "enabled" && row.status !== "启用") {
        return false;
      }
      if (statusFilter === "disabled" && row.status !== "禁用") {
        return false;
      }
      if (!normalizedQuery) {
        return true;
      }

      return [
        row.name,
        String(row.id),
        row.rawKey ?? "",
        row.channelNames.join(" "),
        row.modelAliases.join(" "),
      ].some((field) => field.toLowerCase().includes(normalizedQuery));
    });

    nextRows.sort((left, right) => {
      switch (sortOption) {
        case "id_asc":
          return left.id - right.id;
        case "name_asc":
          return left.name.localeCompare(right.name);
        case "name_desc":
          return right.name.localeCompare(left.name);
        case "id_desc":
        default:
          return right.id - left.id;
      }
    });

    return nextRows;
  }, [rows, searchValue, sortOption, statusFilter]);

  const selectedVisibleIds = filteredRows.filter((row) => selectedIds.includes(row.id)).map((row) => row.id);
  const allVisibleSelected = filteredRows.length > 0 && selectedVisibleIds.length === filteredRows.length;

  const handleCopy = async (row: ApiKeyRow) => {
    if (!row.rawKey) {
      return;
    }

    try {
      await navigator.clipboard.writeText(row.rawKey);
      setCopiedId(row.id);
      window.setTimeout(() => {
        setCopiedId((current) => (current === row.id ? null : current));
      }, 1200);
    } catch {
      setCopiedId(null);
    }
  };

  const handleCreate = async (draft: NewApiKeyDraft) => {
    setError(null);

    const response = await fetch("/api/admin/api-keys", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-OpenCrab-Secondary-Password": draft.secondaryPassword },
      body: JSON.stringify({
        name: draft.name.trim(),
        enabled: draft.enabled,
        channel_names: draft.channelNames,
        model_aliases: draft.modelAliases,
      }),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    const created = (await response.json()) as {
      id: number;
      name: string;
      raw_key: string;
      enabled: boolean;
      channel_names?: string[];
      model_aliases?: string[];
    };
    setRows((current) => [
      {
        id: created.id,
        name: created.name,
        rawKey: created.raw_key,
        status: created.enabled ? "启用" : "禁用",
        channelNames: created.channel_names ?? [],
        modelAliases: created.model_aliases ?? [],
      },
      ...current,
    ]);
    setCreateOpen(false);
  };

  const handleStatusChange = async (id: number, status: string) => {
    setError(null);

    const response = await fetch(`/api/admin/api-keys/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: status === "启用" }),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    setRows((current) => current.map((row) => (row.id === id ? { ...row, status } : row)));
  };

  const deleteOne = async (id: number, secondaryPassword: string) => {
    const response = await fetch(`/api/admin/api-keys/${id}`, {
      method: "DELETE",
      headers: { "X-OpenCrab-Secondary-Password": secondaryPassword },
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
  };

  const handleDelete = async (id: number, secondaryPassword: string) => {
    setError(null);
    await deleteOne(id, secondaryPassword);
    setRows((current) => current.filter((row) => row.id !== id));
    setSelectedIds((current) => current.filter((item) => item !== id));
  };

  const toggleSelect = (id: number) => {
    setSelectedIds((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  };

  const toggleSelectAllVisible = () => {
    const visibleIds = filteredRows.map((row) => row.id);
    if (allVisibleSelected) {
      setSelectedIds((current) => current.filter((id) => !visibleIds.includes(id)));
      return;
    }

    setSelectedIds((current) => Array.from(new Set([...current, ...visibleIds])));
  };

  const handleBatchDelete = async () => {
    if (selectedIds.length === 0) {
      return;
    }

    const confirmed = window.confirm(`确认删除已选中的 ${selectedIds.length} 个 API Key 吗？`);
    if (!confirmed) {
      return;
    }

    let secondaryPassword = "";
    if (requiresSecondaryPassword) {
      const prompted = window.prompt("请输入二级密码以删除所选 API Key");
      if (prompted === null) {
        return;
      }
      secondaryPassword = prompted;
    }

    try {
      setBatchDeleting(true);
      setError(null);
      await Promise.all(selectedIds.map((id) => deleteOne(id, secondaryPassword)));
      setRows((current) => current.filter((row) => !selectedIds.includes(row.id)));
      setSelectedIds([]);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "批量删除失败");
    } finally {
      setBatchDeleting(false);
    }
  };

  return (
    <PageContainer>
      <PageHeader
        eyebrow={eyebrow}
        title={title}
        description={description}
        action={
          <DetailDrawer
            title="新建密钥"
            description="创建一个新的访问密钥，并支持多选渠道和模型。"
            triggerLabel="新建"
            trigger={<Button>新建</Button>}
            open={createOpen}
            onOpenChange={setCreateOpen}
          >
            <NewApiKeyForm
              onCreate={handleCreate}
              onCancel={() => setCreateOpen(false)}
              requiresSecondaryPassword={requiresSecondaryPassword}
              channelOptions={channelOptions}
              modelOptions={modelOptions}
            />
          </DetailDrawer>
        }
      />

      <section className="grid gap-4 md:grid-cols-3">
        <StatCard title="已创建密钥" description="当前系统内密钥总数" value={String(stats.total)} />
        <StatCard title="启用中" description="可正常调用的密钥数量" value={String(stats.enabled)} />
        <StatCard title="禁用中" description="已停用密钥数量" value={String(stats.disabled)} />
      </section>

      <SectionCard title="密钥列表" description="支持多选限制、筛选、批量删除和查看详情。">
        <FilterBar
          placeholder="搜索密钥名称"
          searchValue={searchValue}
          onSearchValueChange={setSearchValue}
          chips={[]}
          trailingAction={
            <div className="flex flex-wrap gap-2">
              <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as StatusFilter)}>
                <SelectTrigger className="min-w-[132px] bg-background">
                  <SelectValue placeholder="状态筛选" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="enabled">仅启用</SelectItem>
                  <SelectItem value="disabled">仅禁用</SelectItem>
                </SelectContent>
              </Select>

              <Select value={sortOption} onValueChange={(value) => setSortOption(value as SortOption)}>
                <SelectTrigger className="min-w-[132px] bg-background">
                  <SelectValue placeholder="排序方式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="id_desc">编号从新到旧</SelectItem>
                  <SelectItem value="id_asc">编号从旧到新</SelectItem>
                  <SelectItem value="name_asc">名称 A-Z</SelectItem>
                  <SelectItem value="name_desc">名称 Z-A</SelectItem>
                </SelectContent>
              </Select>
            </div>
          }
        />

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <div className="text-sm text-muted-foreground">已选 {selectedIds.length} 项，共 {filteredRows.length} 项</div>
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={toggleSelectAllVisible} disabled={filteredRows.length === 0}>
              {allVisibleSelected ? "取消全选" : "全选"}
            </Button>
            <Button variant="danger" onClick={() => void handleBatchDelete()} disabled={selectedIds.length === 0 || batchDeleting}>
              {batchDeleting ? "删除中..." : "批量删除"}
            </Button>
          </div>
        </div>

        {error ? <div className="mt-4 rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

        {filteredRows.length > 0 ? (
          <div className="mt-4 overflow-hidden rounded-xl border border-border bg-background shadow-sm">
            <table className="min-w-full divide-y divide-border/50 text-sm">
              <thead className="bg-secondary/30 text-left text-muted-foreground">
                <tr>
                  <th className="w-16 px-3 py-3.5 text-center font-medium">选择</th>
                  <th className="px-4 py-3.5 font-medium">名称</th>
                  <th className="px-4 py-3.5 font-medium">预览</th>
                  <th className="px-4 py-3.5 font-medium">状态</th>
                  <th className="px-4 py-3.5 font-medium">渠道</th>
                  <th className="px-4 py-3.5 font-medium">模型</th>
                  <th className="px-4 py-3.5 font-medium">编号</th>
                  <th className="px-4 py-3.5 text-right font-medium">操作</th>
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
                          aria-label={checked ? `取消选中 ${row.name}` : `选中 ${row.name}`}
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
                      <td className="px-4 py-3.5">
                        <span className="font-medium text-foreground">{row.name}</span>
                      </td>
                      <td className="px-4 py-3.5">
                        {row.rawKey ? (
                          <button
                            type="button"
                            onClick={() => void handleCopy(row)}
                            className="inline-flex items-center gap-2 rounded-lg border border-transparent px-2 py-1 font-mono text-xs text-muted-foreground transition-[background-color,color,border-color] duration-200 ease-[var(--ease-out-smooth)] hover:border-border hover:bg-muted/60 hover:text-foreground"
                            title="单击复制 API Key"
                          >
                            {copiedId === row.id ? <CopyCheck className="h-3.5 w-3.5 text-success" /> : <Copy className="h-3.5 w-3.5" />}
                            <span>{toPreview(row.rawKey)}</span>
                          </button>
                        ) : (
                          <span className="font-mono text-xs text-muted-foreground">{toPreview(row.rawKey)}</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5">
                        <StatusBadge status={row.status} />
                      </td>
                      <td className="px-4 py-3.5 text-muted-foreground">{renderScope(row.channelNames)}</td>
                      <td className="px-4 py-3.5 text-muted-foreground">{renderScope(row.modelAliases)}</td>
                      <td className="px-4 py-3.5">{row.id}</td>
                      <td className="px-4 py-3.5 text-right">
                        <DetailDrawer title={row.name} description="可查看详情、启用禁用和删除。" triggerLabel="管理">
                          <ApiKeyManager
                            row={row}
                            onStatusChange={(status) => handleStatusChange(row.id, status)}
                            onDelete={(secondaryPassword) => handleDelete(row.id, secondaryPassword)}
                            requiresSecondaryPassword={requiresSecondaryPassword}
                          />
                        </DetailDrawer>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="mt-4 rounded-xl border border-dashed border-border bg-muted/20 px-4 py-6 text-sm text-muted-foreground">
            当前没有匹配的密钥。
          </div>
        )}
      </SectionCard>
    </PageContainer>
  );
}
