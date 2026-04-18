"use client";

import { useState } from "react";
import { Copy } from "lucide-react";

import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type ApiKeyRow = {
  id: number;
  name: string;
  rawKey?: string;
  status: string;
};

export function ApiKeyManager({
  row,
  onStatusChange,
  onDelete,
  requiresSecondaryPassword,
}: {
  row: ApiKeyRow;
  onStatusChange: (status: string) => void;
  onDelete: (secondaryPassword: string) => Promise<void>;
  requiresSecondaryPassword: boolean;
}) {
  const [status, setStatus] = useState(row.status);
  const [copied, setCopied] = useState(false);
  const [secondaryPassword, setSecondaryPassword] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  const handleCopy = async () => {
    if (!row.rawKey) {
      return;
    }

    try {
      await navigator.clipboard.writeText(row.rawKey);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  const handleToggleStatus = () => {
    const next = status === "禁用" ? "启用" : "禁用";
    setStatus(next);
    onStatusChange(next);
  };

  const handleDelete = async () => {
    try {
      setDeleting(true);
      setError("");
      await onDelete(secondaryPassword);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "删除失败");
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-medium text-foreground">密钥详情</h3>
        <div className="mt-3 rounded-xl border border-border bg-card">
          <dl className="divide-y divide-border text-sm">
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">密钥编号</dt>
              <dd className="col-span-2 font-mono text-foreground">#{row.id}</dd>
            </div>
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">当前状态</dt>
              <dd className="col-span-2">
                <StatusBadge status={status} />
              </dd>
            </div>
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">密钥名称</dt>
              <dd className="col-span-2 text-foreground">{row.name}</dd>
            </div>
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">完整密钥</dt>
              <dd className="col-span-2 text-foreground">{row.rawKey ? <span className="font-mono">{row.rawKey}</span> : "仅创建时返回"}</dd>
            </div>
          </dl>
        </div>
      </div>

      <div className="flex justify-end gap-3">
        <Button
          variant={status === "禁用" ? "secondary" : "outline"}
          className={status === "禁用" ? "border border-success/20 bg-success/10 text-success hover:bg-success/20 hover:text-success" : "border border-danger/20 bg-danger/5 text-danger hover:bg-danger/10 hover:text-danger"}
          onClick={handleToggleStatus}
        >
          {status === "禁用" ? "重新启用" : "禁用密钥"}
        </Button>
        {row.rawKey ? (
          <Button className="gap-2" onClick={() => void handleCopy()}>
            <Copy className="h-4 w-4" />
            {copied ? "已复制" : "复制密钥"}
          </Button>
        ) : null}
      </div>

      <div className="rounded-2xl border border-danger/20 bg-danger/5 p-4">
        <div className="space-y-2">
          <div className="text-sm font-medium text-danger">删除访问密钥</div>
          <div className="text-sm text-danger/80">
            删除后无法恢复。{requiresSecondaryPassword ? "当前已开启二级密码，删除前必须校验。" : "当前未开启二级密码，将直接删除。"}
          </div>
        </div>
        {requiresSecondaryPassword ? (
          <div className="mt-4">
            <Input type="password" value={secondaryPassword} onChange={(event) => setSecondaryPassword(event.target.value)} placeholder="输入二级密码" />
          </div>
        ) : null}
        {error ? <div className="mt-3 rounded-xl border border-danger/20 bg-danger/10 px-3 py-2 text-xs text-danger">{error}</div> : null}
        <div className="mt-4 flex justify-end">
          <Button variant="danger" onClick={() => void handleDelete()} disabled={deleting}>
            {deleting ? "删除中..." : "删除密钥"}
          </Button>
        </div>
      </div>
    </div>
  );
}
