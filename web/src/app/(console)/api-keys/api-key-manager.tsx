"use client";

import { useState } from "react";
import { Copy } from "lucide-react";

import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";

type ApiKeyRow = {
	id: number;
  name: string;
  rawKey?: string;
  status: string;
};

export function ApiKeyManager({
  row,
  onStatusChange
}: {
  row: ApiKeyRow;
  onStatusChange: (status: string) => void;
}) {
  const [status, setStatus] = useState(row.status);
  const [copied, setCopied] = useState(false);

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
              <dd className="col-span-2"><StatusBadge status={status} /></dd>
            </div>
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">密钥名称</dt>
              <dd className="col-span-2 text-foreground">{row.name}</dd>
            </div>
            <div className="grid grid-cols-3 gap-4 px-4 py-3">
              <dt className="font-medium text-muted-foreground">完整密钥</dt>
              <dd className="col-span-2 text-foreground">{row.rawKey ? <span className="font-mono">{row.rawKey}</span> : "仅在创建时返回，刷新后不再显示。"}</dd>
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
    </div>
  );
}
