"use client";

import { useState } from "react";

import { StatusSelect } from "@/app/(console)/api-keys/status-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export type NewApiKeyDraft = {
  name: string;
  enabled: boolean;
  secondaryPassword: string;
};

export function NewApiKeyForm({
  onCreate,
  onCancel,
  requiresSecondaryPassword
}: {
  onCreate: (draft: NewApiKeyDraft) => Promise<void>;
  onCancel: () => void;
  requiresSecondaryPassword: boolean;
}) {
  const [name, setName] = useState("new-api-key");
  const [status, setStatus] = useState("启用");
  const [secondaryPassword, setSecondaryPassword] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    setError(null);
    setIsCreating(true);
    try {
      await onCreate({
        name: name.trim() || "new-api-key",
			enabled: status === "启用",
        secondaryPassword
      });
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建失败");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="space-y-6 text-sm text-muted-foreground">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">密钥名称</label>
        <Input value={name} onChange={(event) => setName(event.target.value)} />
      </div>

      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">默认状态</label>
        <StatusSelect value={status} onValueChange={setStatus} />
      </div>

      <div className="rounded-2xl border border-border bg-card/60 p-4 text-sm leading-6 text-muted-foreground">
        后端会在创建成功时返回一次完整密钥，控制台只会保留本次创建拿到的原文，刷新页面后不再展示。
      </div>

      {requiresSecondaryPassword ? (
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">二级密码</label>
          <Input type="password" value={secondaryPassword} onChange={(event) => setSecondaryPassword(event.target.value)} placeholder="创建密钥前需输入二级密码" />
        </div>
      ) : null}

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>取消</Button>
        <Button type="button" onClick={handleCreate} className={isCreating ? "pointer-events-none" : ""}>
          {isCreating ? "生成中..." : "生成密钥"}
        </Button>
      </div>
    </div>
  );
}
