"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Eye, EyeOff, Plus, X } from "lucide-react";

import { ProviderSelect } from "@/app/(console)/channels/provider-select";
import { StatusSelect } from "@/app/(console)/api-keys/status-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getDefaultEndpointForProvider } from "@/lib/channel-provider";

export function NewChannelForm() {
  const [keyVisible, setKeyVisible] = useState(false);
  const [provider, setProvider] = useState("OpenAI");
  const [status, setStatus] = useState("启用");
  const [name, setName] = useState("");
  const [endpoint, setEndpoint] = useState(getDefaultEndpointForProvider("OpenAI"));
  const [apiKey, setApiKey] = useState("");
  const [customModel, setCustomModel] = useState("");
  const [modelIds, setModelIds] = useState<string[]>([]);
  const [rpmLimit, setRpmLimit] = useState("1000");
  const [maxInflight, setMaxInflight] = useState("32");
  const [safetyFactor, setSafetyFactor] = useState("0.9");
  const [enabledForAsync, setEnabledForAsync] = useState("true");
  const [dispatchWeight, setDispatchWeight] = useState("100");
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

  const canAddModel = useMemo(() => {
    const normalized = customModel.trim();
    return normalized.length > 0 && !modelIds.includes(normalized);
  }, [customModel, modelIds]);

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

  const handleSubmit = async () => {
    setError(null);
    if (modelIds.length === 0) {
      setError("至少添加一个模型 ID");
      return;
    }
    setIsSaving(true);
    try {
      const response = await fetch("/api/admin/channels", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          provider: provider.trim(),
          endpoint: endpoint.trim(),
          api_key: apiKey.trim(),
          enabled: status === "启用",
          model_ids: modelIds,
          rpm_limit: Number.parseInt(rpmLimit, 10),
          max_inflight: Number.parseInt(maxInflight, 10),
          safety_factor: Number.parseFloat(safetyFactor),
          enabled_for_async: enabledForAsync === "true",
          dispatch_weight: Number.parseInt(dispatchWeight, 10)
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      window.location.reload();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "创建失败");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6 text-sm text-muted-foreground">
      <div className="grid gap-4">
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">渠道名称</label>
          <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：openai-main" />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">兼容类型</label>
          <ProviderSelect value={provider} onValueChange={setProvider} />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">请求地址</label>
          <Input value={endpoint} onChange={(event) => setEndpoint(event.target.value)} placeholder={getDefaultEndpointForProvider(provider)} />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">状态</label>
          <StatusSelect value={status} onValueChange={setStatus} />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">API Key</label>
          <div className="flex gap-2">
            <Input
              type={keyVisible ? "text" : "password"}
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder="填写上游渠道使用的 API Key"
              className="font-mono"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => setKeyVisible((current) => !current)}
              title={keyVisible ? "隐藏 Key" : "显示 Key"}
            >
              {keyVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </Button>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">每分钟额度</label>
            <Input value={rpmLimit} onChange={(event) => setRpmLimit(event.target.value)} inputMode="numeric" placeholder="1000" />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">最大 Inflight</label>
            <Input value={maxInflight} onChange={(event) => setMaxInflight(event.target.value)} inputMode="numeric" placeholder="32" />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">安全系数</label>
            <Input value={safetyFactor} onChange={(event) => setSafetyFactor(event.target.value)} inputMode="decimal" placeholder="0.9" />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">调度权重</label>
            <Input value={dispatchWeight} onChange={(event) => setDispatchWeight(event.target.value)} inputMode="numeric" placeholder="100" />
          </div>
          <div className="space-y-2 md:col-span-2">
            <label className="text-sm font-medium text-foreground">支持异步受理</label>
            <StatusSelect value={enabledForAsync === "true" ? "启用" : "禁用"} onValueChange={(value) => setEnabledForAsync(value === "启用" ? "true" : "false")} />
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-border bg-card/60 p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground">模型 ID</h3>
            <p className="mt-1 text-sm text-muted-foreground">创建渠道时一并录入该渠道承载的上游模型 ID。</p>
          </div>
          <span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
            {modelIds.length} 个模型
          </span>
        </div>

        <div className="mt-4 flex gap-2">
          <Input
            value={customModel}
            onChange={(event) => setCustomModel(event.target.value)}
            placeholder="输入模型 ID，例如：gpt-4.1"
            className="font-mono"
          />
          <Button type="button" variant="secondary" onClick={handleAddModel} disabled={!canAddModel} className="shrink-0 rounded-xl px-4">
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

      <div className="grid gap-4 rounded-2xl border border-border bg-card/60 p-4 md:grid-cols-2">
        <div>
          <div className="text-sm font-semibold text-foreground">认证与安全</div>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">渠道创建完成后，后端会直接把这些真实配置写入数据库。</p>
        </div>
        <div>
          <div className="text-sm font-semibold text-foreground">模型创建说明</div>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">提交后会为每个模型 ID 自动创建同名模型映射，并绑定到当前渠道。</p>
        </div>
      </div>

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" onClick={handleSubmit} className={isSaving ? "pointer-events-none" : ""}>
          {isSaving ? "保存中..." : "保存渠道"}
        </Button>
      </div>
    </div>
  );
}
