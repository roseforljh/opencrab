"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type ChannelRow = {
  name: string;
  provider: string;
  endpoint: string;
};

export function EditChannelForm({
  row,
  onCancel,
  onSave
}: {
  row: ChannelRow;
  onCancel: () => void;
  onSave: () => void;
}) {
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = () => {
    setIsSaving(true);
    window.setTimeout(() => {
      setIsSaving(false);
      onSave();
    }, 650);
  };

  return (
    <div className="space-y-4 text-sm text-muted-foreground">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">渠道名称</label>
        <Input defaultValue={row.name} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">兼容类型</label>
        <Input defaultValue={row.provider} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">请求地址</label>
        <Input defaultValue={row.endpoint} className="font-mono" />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">默认密钥</label>
        <Input defaultValue="sk-example-key" className="font-mono" />
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>取消</Button>
        <Button type="button" onClick={handleSave} className={isSaving ? "pointer-events-none" : ""}>
          {isSaving ? "保存中..." : "保存修改"}
        </Button>
      </div>
    </div>
  );
}
