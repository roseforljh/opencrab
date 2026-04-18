"use client";

import { useMemo, useState } from "react";

import { StatusSelect } from "@/app/(console)/api-keys/status-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export type NewApiKeyDraft = {
  name: string;
  enabled: boolean;
  secondaryPassword: string;
  channelNames: string[];
  modelAliases: string[];
};

function MultiChoice({
  title,
  options,
  selectedValues,
  onToggle,
}: {
  title: string;
  options: string[];
  selectedValues: string[];
  onToggle: (value: string) => void;
}) {
  return (
    <div className="space-y-2">
      <label className="text-sm font-medium text-foreground">{title}</label>
      <div className="rounded-2xl border border-border bg-card/50 p-3">
        <button
          type="button"
          onClick={() => {
            selectedValues.forEach(() => undefined);
          }}
          className={`mb-2 inline-flex rounded-full border px-3 py-1 text-xs transition-[background-color,border-color,color] duration-200 ${
            selectedValues.length === 0
              ? "border-foreground bg-foreground text-background"
              : "border-border bg-background text-muted-foreground hover:border-foreground/30 hover:text-foreground"
          }`}
        >
          全部
        </button>
        <div className="flex flex-wrap gap-2">
          {options.map((item) => {
            const checked = selectedValues.includes(item);
            return (
              <button
                key={item}
                type="button"
                onClick={() => onToggle(item)}
                className={`rounded-full border px-3 py-1 text-xs transition-[background-color,border-color,color] duration-200 ${
                  checked
                    ? "border-foreground bg-foreground text-background"
                    : "border-border bg-background text-muted-foreground hover:border-foreground/30 hover:text-foreground"
                }`}
              >
                {item}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export function NewApiKeyForm({
  onCreate,
  onCancel,
  requiresSecondaryPassword,
  channelOptions,
  modelOptions,
}: {
  onCreate: (draft: NewApiKeyDraft) => Promise<void>;
  onCancel: () => void;
  requiresSecondaryPassword: boolean;
  channelOptions: string[];
  modelOptions: string[];
}) {
  const [name, setName] = useState("new-api-key");
  const [status, setStatus] = useState("启用");
  const [secondaryPassword, setSecondaryPassword] = useState("");
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedModels, setSelectedModels] = useState<string[]>([]);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sortedChannels = useMemo(() => [...channelOptions].sort((left, right) => left.localeCompare(right)), [channelOptions]);
  const sortedModels = useMemo(() => [...modelOptions].sort((left, right) => left.localeCompare(right)), [modelOptions]);

  const toggleValue = (values: string[], value: string) =>
    values.includes(value) ? values.filter((item) => item !== value) : [...values, value];

  const handleCreate = async () => {
    setError(null);
    setIsCreating(true);
    try {
      await onCreate({
        name: name.trim() || "new-api-key",
        enabled: status === "启用",
        secondaryPassword,
        channelNames: selectedChannels,
        modelAliases: selectedModels,
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

      <div className="grid gap-4 md:grid-cols-2">
        <MultiChoice
          title="限制渠道"
          options={sortedChannels}
          selectedValues={selectedChannels}
          onToggle={(value) => setSelectedChannels((current) => toggleValue(current, value))}
        />
        <MultiChoice
          title="限制模型"
          options={sortedModels}
          selectedValues={selectedModels}
          onToggle={(value) => setSelectedModels((current) => toggleValue(current, value))}
        />
      </div>

      <div className="rounded-2xl border border-border bg-card/60 p-4 text-sm leading-6 text-muted-foreground">
        两项都可不选。不选表示全部可用，也支持只选渠道或只选模型。
      </div>

      {requiresSecondaryPassword ? (
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">二级密码</label>
          <Input
            type="password"
            value={secondaryPassword}
            onChange={(event) => setSecondaryPassword(event.target.value)}
            placeholder="创建前请输入二级密码"
          />
        </div>
      ) : null}

      {error ? <div className="rounded-xl border border-danger/20 bg-danger/5 px-3 py-2 text-xs text-danger">{error}</div> : null}

      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button type="button" onClick={handleCreate} className={isCreating ? "pointer-events-none" : ""}>
          {isCreating ? "生成中..." : "生成密钥"}
        </Button>
      </div>
    </div>
  );
}
