"use client";

import { useMemo, useState } from "react";
import { Copy, Eye, EyeOff, RefreshCw } from "lucide-react";

import { StatusSelect } from "@/app/(console)/api-keys/status-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export type NewApiKeyDraft = {
  name: string;
  status: string;
  rawKey: string;
  preview: string;
};

function generateApiKey() {
  const randomPart = Array.from({ length: 24 }, () => Math.floor(Math.random() * 36).toString(36)).join("");
  return `sk-opencrab-${randomPart}`;
}

function toPreview(rawKey: string) {
  const head = rawKey.slice(0, 14);
  const tail = rawKey.slice(-4);
  return `${head}••••${tail}`;
}

export function NewApiKeyForm({
  onCreate,
  onCancel
}: {
  onCreate: (draft: NewApiKeyDraft) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("new-api-key");
  const [status, setStatus] = useState("启用");
  const [generatedKey, setGeneratedKey] = useState(generateApiKey());
  const [copied, setCopied] = useState(false);
  const [visible, setVisible] = useState(false);
  const [isRegenerating, setIsRegenerating] = useState(false);

  const preview = useMemo(() => toPreview(generatedKey), [generatedKey]);

  const handleGenerate = () => {
    const next = generateApiKey();
    setGeneratedKey(next);
    setCopied(false);
    onCreate({
      name: name.trim() || "new-api-key",
      status,
      rawKey: next,
      preview: toPreview(next)
    });
  };

  const handleRegeneratePreview = () => {
    setIsRegenerating(true);
    window.setTimeout(() => {
      setGeneratedKey(generateApiKey());
      setCopied(false);
      setIsRegenerating(false);
    }, 480);
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatedKey);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
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

      <div className="rounded-2xl border border-border bg-card/60 p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground">密钥预览</h3>
            <p className="mt-1 text-sm text-muted-foreground">可以重新生成、显示完整内容或复制到剪贴板。</p>
          </div>
          <span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">{status}</span>
        </div>

        <div className="mt-4 flex gap-2">
          <Input readOnly value={visible ? generatedKey : preview} className="font-mono" />
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => setVisible((current) => !current)}
            title={visible ? "隐藏 Key" : "显示 Key"}
            className={visible ? "border-primary/30 bg-primary/10 text-primary" : ""}
          >
            {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={handleRegeneratePreview}
            title="重新生成"
            className={isRegenerating ? "border-primary/30 bg-primary/10 text-primary" : ""}
          >
            <RefreshCw className={`h-4 w-4 ${isRegenerating ? "animate-smooth-spin" : ""}`} />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => void handleCopy()}
            title={copied ? "已复制" : "复制 Key"}
            className={copied ? "border-success/30 bg-success/10 text-success" : ""}
          >
            <Copy className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>取消</Button>
        <Button type="button" onClick={handleGenerate}>生成密钥</Button>
      </div>
    </div>
  );
}
