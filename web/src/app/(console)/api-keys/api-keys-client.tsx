"use client";

import { useMemo, useState } from "react";
import { Copy, CopyCheck } from "lucide-react";

import { ApiKeyManager } from "@/app/(console)/api-keys/api-key-manager";
import { NewApiKeyForm, type NewApiKeyDraft } from "@/app/(console)/api-keys/new-api-key-form";
import { PageContainer } from "@/components/layout/page-container";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { FilterBar } from "@/components/shared/filter-bar";
import { SectionCard } from "@/components/shared/section-card";
import { StaticTable, type StaticTableColumn } from "@/components/shared/static-table";
import { StatCard } from "@/components/shared/stat-card";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";

type ApiKeyRow = {
  id: number;
  name: string;
  rawKey?: string;
  status: string;
};

function toPreview(rawKey?: string) {
  if (!rawKey) {
    return "仅创建时展示";
  }

  const head = rawKey.slice(0, 14);
  const tail = rawKey.slice(-4);
  return `${head}••••${tail}`;
}

export function ApiKeysClient({
  eyebrow,
  title,
  description,
  initialRows,
  requiresSecondaryPassword
}: {
  eyebrow: string;
  title: string;
  description: string;
  initialRows: ApiKeyRow[];
  requiresSecondaryPassword: boolean;
}) {
  const [rows, setRows] = useState<ApiKeyRow[]>(initialRows);
  const [createOpen, setCreateOpen] = useState(false);
  const [copiedId, setCopiedId] = useState<number | null>(null);

  const stats = useMemo(() => {
    const enabled = rows.filter((row) => row.status === "启用").length;
    const disabled = rows.length - enabled;
    return { total: rows.length, enabled, disabled };
  }, [rows]);

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

  const columns: StaticTableColumn<ApiKeyRow>[] = [
    {
      header: "名称",
      cell: (row) => <span className="font-medium text-foreground">{row.name}</span>
    },
    {
      header: "预览",
      cell: (row) =>
        row.rawKey ? (
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
        )
    },
    {
      header: "状态",
      cell: (row) => <StatusBadge status={row.status} />
    },
    {
      header: "编号",
      cell: (row) => row.id
    }
  ];

  const handleCreate = async (draft: NewApiKeyDraft) => {
    const response = await fetch("/api/admin/api-keys", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-OpenCrab-Secondary-Password": draft.secondaryPassword },
      body: JSON.stringify({ name: draft.name.trim(), enabled: draft.enabled })
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    const created = (await response.json()) as { id: number; name: string; raw_key: string; enabled: boolean };
    setRows((current) => [
      { id: created.id, name: created.name, rawKey: created.raw_key, status: created.enabled ? "启用" : "禁用" },
      ...current
    ]);
    setCreateOpen(false);
  };

  const handleStatusChange = async (id: number, status: string) => {
    const response = await fetch(`/api/admin/api-keys/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: status === "启用" })
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    setRows((current) => current.map((row) => (row.id === id ? { ...row, status } : row)));
  };

  const handleDelete = async (id: number, secondaryPassword: string) => {
    const response = await fetch(`/api/admin/api-keys/${id}`, {
      method: "DELETE",
      headers: { "X-OpenCrab-Secondary-Password": secondaryPassword }
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    setRows((current) => current.filter((row) => row.id !== id));
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
            description="创建一个新的访问密钥，并设置名称与默认状态。"
            triggerLabel="新建"
            trigger={<Button>新建</Button>}
            open={createOpen}
            onOpenChange={setCreateOpen}
          >
            <NewApiKeyForm onCreate={handleCreate} onCancel={() => setCreateOpen(false)} requiresSecondaryPassword={requiresSecondaryPassword} />
          </DetailDrawer>
        }
      />

      <section className="grid gap-4 md:grid-cols-3">
        <StatCard title="已创建密钥" description="当前系统内密钥总数" value={String(stats.total)} />
        <StatCard title="启用中" description="可正常调用的密钥数量" value={String(stats.enabled)} />
        <StatCard title="禁用中" description="已停用密钥数量" value={String(stats.disabled)} />
      </section>

      <SectionCard title="密钥列表" description="列表页负责管理状态，详情抽屉负责查看和操作。">
        <FilterBar placeholder="搜索密钥名称" chips={[{ label: "全部状态" }, { label: "最近使用" }]} />
        <div className="mt-4">
          <StaticTable
            columns={columns}
            data={rows}
            emptyTitle="暂无访问密钥"
            emptyDescription="创建第一个密钥后，这里会展示调用状态和最近使用时间。"
            rowAction={(row) => (
              <DetailDrawer title={row.name} description="这里会承载复制、禁用、重置和查看权限等操作。" triggerLabel="管理">
                <ApiKeyManager
                  row={row}
                  onStatusChange={(status) => handleStatusChange(row.id, status)}
                  onDelete={(secondaryPassword) => handleDelete(row.id, secondaryPassword)}
                  requiresSecondaryPassword={requiresSecondaryPassword}
                />
              </DetailDrawer>
            )}
          />
        </div>
      </SectionCard>
    </PageContainer>
  );
}
