"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { Trash2 } from "lucide-react";

import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Button } from "@/components/ui/button";

export function ClearLogsButton() {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    setError(null);

    const response = await fetch("/api/admin/logs", {
      method: "DELETE"
    });

    if (!response.ok) {
      const message = await response.text();
      setError(message || "清空日志失败");
      return;
    }

    startTransition(() => {
      router.refresh();
    });
  }

  return (
    <div className="flex flex-col items-end gap-2">
      <ConfirmDialog
        title="清空请求日志"
        description="会立即删除当前全部请求日志记录。此操作不可撤销。"
        confirmLabel={isPending ? "刷新中..." : "确认清空"}
        onConfirm={handleConfirm}
        trigger={
          <Button variant="outline" className="gap-2" disabled={isPending}>
            <Trash2 className="h-4 w-4" />
            一键清空日志
          </Button>
        }
      />
      {error ? <p className="text-xs text-danger">{error}</p> : null}
    </div>
  );
}
