"use client";

import { useMemo, useState } from "react";
import { Copy, Eye, EyeOff, Plus, X } from "lucide-react";

import { ProviderSelect } from "@/app/(console)/channels/provider-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const presetModels = [
  "gpt-4.1",
  "gpt-4.1-mini",
  "o3-mini",
  "claude-3.7-sonnet",
  "gemini-2.5-pro"
];

export function NewChannelForm() {
  const [keyVisible, setKeyVisible] = useState(false);
  const [provider, setProvider] = useState("OpenAI");
  const [copiedModel, setCopiedModel] = useState<string | null>(null);
  const [customModel, setCustomModel] = useState("");
  const [models, setModels] = useState<string[]>(presetModels);

  const canAddModel = useMemo(() => {
    const normalized = customModel.trim();
    return normalized.length > 0 && !models.includes(normalized);
  }, [customModel, models]);

  const handleCopy = async (model: string) => {
    try {
      await navigator.clipboard.writeText(model);
      setCopiedModel(model);
      window.setTimeout(() => setCopiedModel((current) => (current === model ? null : current)), 1200);
    } catch {
      setCopiedModel(null);
    }
  };

  const handleAddModel = () => {
    const normalized = customModel.trim();
    if (!normalized || models.includes(normalized)) {
      return;
    }

    setModels((current) => [...current, normalized]);
    setCustomModel("");
  };

  const handleRemoveModel = (model: string) => {
    setModels((current) => current.filter((item) => item !== model));
    if (copiedModel === model) {
      setCopiedModel(null);
    }
  };

  return (
    <div className="space-y-6 text-sm text-muted-foreground">
      <div className="grid gap-4">
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">渠道名称</label>
          <Input defaultValue="openai-main" />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">兼容类型</label>
          <ProviderSelect value={provider} onValueChange={setProvider} />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">请求地址</label>
          <Input defaultValue="https://api.example.com/v1" />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">API Key</label>
          <div className="flex gap-2">
            <Input
              type={keyVisible ? "text" : "password"}
              defaultValue="sk-example-key"
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
      </div>

      <div className="rounded-2xl border border-border bg-card/60 p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground">模型列表</h3>
            <p className="mt-1 text-sm text-muted-foreground">一个渠道通常会承载多个模型。单击任意模型名称可直接复制。</p>
          </div>
          <span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
            {models.length} 个模型
          </span>
        </div>

        <div className="mt-4 flex gap-2">
          <Input
            value={customModel}
            onChange={(event) => setCustomModel(event.target.value)}
            placeholder="输入新的模型 ID，例如：deepseek-chat"
            className="font-mono"
          />
          <Button
            type="button"
            variant="secondary"
            onClick={handleAddModel}
            disabled={!canAddModel}
            className="shrink-0 rounded-xl px-4"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {models.map((model) => {
            const copied = copiedModel === model;

            return (
              <button
                key={model}
                type="button"
                onClick={() => void handleCopy(model)}
                className={`group inline-flex items-center gap-2 rounded-xl border px-3 py-2 font-mono text-xs transition-all duration-200 ease-[var(--ease-out-smooth)] ${
                  copied
                    ? "border-primary/40 bg-primary/10 text-primary shadow-[0_0_0_1px_rgba(255,255,255,0.04)]"
                    : "border-border bg-background text-foreground hover:-translate-y-0.5 hover:border-primary/30 hover:bg-muted"
                }`}
              >
                <Copy className="h-3.5 w-3.5" />
                <span>{model}</span>
                {copied ? <span className="text-[10px] text-primary">已复制</span> : null}
                <span
                  role="button"
                  tabIndex={0}
                  onClick={(event) => {
                    event.stopPropagation();
                    handleRemoveModel(model);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      event.stopPropagation();
                      handleRemoveModel(model);
                    }
                  }}
                  className="ml-1 inline-flex h-4 w-4 items-center justify-center rounded-full text-danger opacity-0 transition-opacity duration-200 ease-[var(--ease-out-smooth)] hover:bg-danger/10 group-hover:opacity-100"
                  title="删除模型"
                >
                  <X className="h-3 w-3" />
                </span>
              </button>
            );
          })}
        </div>
      </div>

      <div className="grid gap-4 rounded-2xl border border-border bg-card/60 p-4 md:grid-cols-2">
        <div>
          <div className="text-sm font-semibold text-foreground">认证与安全</div>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">后续可以在这里扩展组织 ID、请求头覆盖、请求体脱敏、超时策略等更细的渠道控制项。</p>
        </div>
        <div>
          <div className="text-sm font-semibold text-foreground">模型映射提示</div>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">模型 ID 会在路由配置页被进一步映射为公开别名，因此这里应该尽量填写上游真实模型名。</p>
        </div>
      </div>

      <div className="flex justify-end gap-3 pt-2">
        <Button variant="outline">测试连接</Button>
        <Button>保存渠道</Button>
      </div>
    </div>
  );
}
