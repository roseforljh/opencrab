"use client";

import { useState } from "react";

import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { NoticeDialog } from "@/components/shared/notice-dialog";
import { Button } from "@/components/ui/button";

export function DeleteChannelButton({ row }: { row: { id: number; name: string } }) {
  const [error, setError] = useState<string | null>(null);

  const handleDelete = async () => {
    const response = await fetch(`/api/admin/channels/${row.id}`, { method: "DELETE" });
    if (!response.ok) {
      const message = await response.text();
      setError(message || "删除失败");
      return;
    }

    window.location.reload();
  };

  return (
    <>
      <ConfirmDialog
        trigger={<Button variant="danger">删除渠道</Button>}
        title="确认删除渠道"
        description={`删除后渠道“${row.name}”及其当前配置将不可恢复。`}
        confirmLabel="确认删除"
        onConfirm={handleDelete}
      />
      <NoticeDialog
        open={error !== null}
        onOpenChange={(open) => {
          if (!open) {
            setError(null);
          }
        }}
        title="删除失败"
        description={error ?? ""}
      />
    </>
  );
}
