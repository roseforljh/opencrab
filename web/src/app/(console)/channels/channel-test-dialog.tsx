"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import * as Dialog from "@radix-ui/react-dialog";
import { CircleCheckBig, LoaderCircle, PlugZap, TriangleAlert, X } from "lucide-react";

import { Button } from "@/components/ui/button";

type ChannelRow = {
  id: number;
  name: string;
  modelIds: string[];
};

type ModelState = "idle" | "testing" | "success" | "error";

export function ChannelTestDialog({ row }: { row: ChannelRow }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [states, setStates] = useState<Record<string, ModelState>>({});
  const [messages, setMessages] = useState<Record<string, string>>({});

  const models = row.modelIds.length > 0 ? row.modelIds : ["默认测试"];

  const stateMap = useMemo(() => {
    const next: Record<string, ModelState> = {};
    models.forEach((model) => {
      next[model] = states[model] ?? "idle";
    });
    return next;
  }, [models, states]);

  const runSingleTest = async (model: string) => {
    setStates((current) => ({ ...current, [model]: "testing" }));
	  setMessages((current) => ({ ...current, [model]: "" }));

	  try {
		  const response = await fetch(`/api/admin/channels/${row.id}/test`, {
			  method: "POST",
			  headers: { "Content-Type": "application/json" },
			  body: JSON.stringify({ model: model === "默认测试" ? "" : model })
		  });
		  if (!response.ok) {
			  throw new Error(await response.text());
		  }
		  const result = (await response.json()) as { message: string; model: string };
		  setStates((current) => ({ ...current, [model]: "success" }));
		  setMessages((current) => ({ ...current, [model]: result.message || `模型 ${result.model} 测试成功` }));
	  } catch (error) {
		  const message = error instanceof Error ? error.message : "测试失败";
		  setStates((current) => ({ ...current, [model]: "error" }));
		  setMessages((current) => ({ ...current, [model]: message }));
	  } finally {
		  router.refresh();
	  }
  };

  const runAllTests = async () => {
	  await Promise.all(models.map((model) => runSingleTest(model)));
  };

  const renderState = (state: ModelState) => {
    if (state === "testing") {
      return (
        <span className="inline-flex items-center gap-2 text-[11px] text-muted-foreground">
          <LoaderCircle className="h-3.5 w-3.5 animate-smooth-spin" />
          测试中
        </span>
      );
    }

    if (state === "success") {
      return (
        <span className="inline-flex items-center gap-2 text-[11px] text-success">
          <CircleCheckBig className="h-3.5 w-3.5" />
          连接成功
        </span>
      );
    }

    if (state === "error") {
      return (
        <span className="inline-flex items-center gap-2 text-[11px] text-danger">
          <TriangleAlert className="h-3.5 w-3.5" />
          连接失败
        </span>
      );
    }

    return <span className="text-[11px] text-muted-foreground">待测试</span>;
  };

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button variant="outline" size="icon" title="测试当前渠道模型连通性">
          <PlugZap className="h-4 w-4" />
        </Button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="animate-overlay fixed inset-0 bg-background/80 backdrop-blur-sm" />
        <Dialog.Content className="animate-modal fixed left-1/2 top-1/2 w-full max-w-2xl -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-background shadow-2xl outline-none">
          <div className="flex items-start justify-between border-b border-border px-6 py-5">
            <div>
              <Dialog.Title className="text-lg font-semibold text-foreground">测试渠道连通性</Dialog.Title>
              <Dialog.Description className="mt-2 text-sm leading-6 text-muted-foreground">
                当前渠道：{row.name}。单击某个模型可单独测试，也可以一键测试当前渠道下所有模型。
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button className="rounded-lg p-2 text-muted-foreground transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:rotate-90 hover:bg-muted hover:text-foreground" aria-label="关闭测试对话框">
                <X className="h-4 w-4" />
              </button>
            </Dialog.Close>
          </div>

          <div className="space-y-4 px-6 py-6">
            <div className="grid gap-3">
              {models.map((model) => (
                <button
                  key={model}
                  type="button"
                  onClick={() => void runSingleTest(model)}
                  className={`flex items-center justify-between rounded-2xl border px-4 py-4 text-left transition-all duration-200 ease-[var(--ease-out-smooth)] ${
                    stateMap[model] === "success"
                      ? "border-success/25 bg-success/5"
                      : stateMap[model] === "error"
                        ? "border-danger/25 bg-danger/5"
                        : stateMap[model] === "testing"
                          ? "border-primary/25 bg-primary/5"
                          : "border-border bg-card/60 hover:-translate-y-0.5 hover:border-primary/20 hover:bg-muted/40"
                  }`}
                >
                  <div>
                    <div className="font-mono text-sm font-medium text-foreground">{model}</div>
                    <div className="mt-1">{renderState(stateMap[model])}</div>
                    {messages[model] ? <div className="mt-1 text-[11px] text-muted-foreground">{messages[model]}</div> : null}
                  </div>
                  <span className="text-xs text-muted-foreground">单击测试</span>
                </button>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-3 border-t border-border px-6 py-5">
            <Dialog.Close asChild>
              <Button variant="outline" type="button">关闭</Button>
            </Dialog.Close>
            <Button type="button" onClick={() => void runAllTests()}>一键测试</Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
