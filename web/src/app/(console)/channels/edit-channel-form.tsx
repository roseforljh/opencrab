"use client";

import { useEffect, useRef, useState } from "react";
import { Plus, X } from "lucide-react";

import { ProviderSelect } from "@/app/(console)/channels/provider-select";
import { StatusSelect } from "@/app/(console)/api-keys/status-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getDefaultEndpointForProvider } from "@/lib/channel-provider";

type ChannelRow = {
  id: number;
  name: string;
  provider: string;
  endpoint: string;
  status: string;
  modelIds: string[];
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
  const [name, setName] = useState(row.name);
  const [provider, setProvider] = useState(row.provider.replace(" Compatible", ""));
  const [endpoint, setEndpoint] = useState(row.endpoint);
  const [apiKey, setApiKey] = useState("");
  const [status, setStatus] = useState(row.status);
  const [customModel, setCustomModel] = useState("");
  const [modelIds, setModelIds] = useState(row.modelIds);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const previousProviderRef = useRef(provider);

  useEffect(() => {
    const previousProvider = previousProviderRef.current;
    const previousDefaultEndpoint = getDefaultEndpointForProvider(previousProvider);
    const nextDefaultEndpoint = getDefaultEndpointForProvider(provider);

    setEndpoint((current) => {
      const trimmed = current.trim();
      if (trimmed === "" || trimmed === previousDefaultEndpoint) {
        return nextDefaultEndpoint;
      }
      return current;
    });

    previousProviderRef.current = provider;
  }, [provider]);

  const handleAddModel = () => {
	const normalized = customModel.trim();
	if (!normalized || modelIds.includes(normalized)) {
	  return;
	}
	setModelIds((current) => [...current, normalized]);
	setCustomModel("");
  };

  const handleRemoveModel = (modelId: string) => {
	setModelIds((current) => current.filter((item) => item !== modelId));
  };

  const handleSave = async () => {
    setError(null);
    if (modelIds.length === 0) {
      setError("至少添加一个模型 ID");
      return;
    }
    setIsSaving(true);
    try {
      const response = await fetch(`/api/admin/channels/${row.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          provider: provider.trim(),
          endpoint: endpoint.trim(),
          api_key: apiKey.trim(),
          enabled: status === "启用",
          model_ids: modelIds
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      onSave();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "保存失败");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-4 text-sm text-muted-foreground">
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">渠道名称</label>
        <Input value={name} onChange={(event) => setName(event.target.value)} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">兼容类型</label>
        <ProviderSelect value={provider} onValueChange={setProvider} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">请求地址</label>
        <Input value={endpoint} onChange={(event) => setEndpoint(event.target.value)} placeholder={getDefaultEndpointForProvider(provider)} className="font-mono" />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">状态</label>
        <StatusSelect value={status} onValueChange={setStatus} />
      </div>
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">默认密钥</label>
        <Input
          value={apiKey}
          onChange={(event) => setApiKey(event.target.value)}
          placeholder="留空表示沿用当前密钥"
          className="font-mono"
        />
      </div>
      <div className="rounded-2xl border border-border bg-card/60 p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground">模型 ID</h3>
            <p className="mt-1 text-sm text-muted-foreground">编辑渠道时同步调整该渠道承载的上游模型 ID。</p>
          </div>
          <span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
            {modelIds.length} 个模型
          </span>
        </div>

        <div className="mt-4 flex gap-2">
          <Input value={customModel} onChange={(event) => setCustomModel(event.target.value)} placeholder="输入模型 ID，例如：gpt-4.1" className="font-mono" />
          <Button type="button" variant="secondary" onClick={handleAddModel} disabled={!customModel.trim() || modelIds.includes(customModel.trim())} className="shrink-0 rounded-xl px-4">
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {modelIds.map((modelId) => (
            <span key={modelId} className="inline-flex items-center gap-2 rounded-xl border border-border bg-background px-3 py-2 font-mono text-xs text-foreground">
              <span>{modelId}</span>
              <button type="button" onClick={() => handleRemoveModel(modelId)} className="inline-flex h-4 w-4 items-center justify-center rounded-full text-danger hover:bg-danger/10" title="删除模型">
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      </div>
      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>取消</Button>
        <Button type="button" onClick={handleSave} className={isSaving ? "pointer-events-none" : ""}>
          {isSaving ? "保存中..." : "保存修改"}
        </Button>
      </div>
    </div>
  );
}
