"use client";

import type { ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { AlertCircle, X } from "lucide-react";

import { Button } from "@/components/ui/button";

export function NoticeDialog({
  open,
  onOpenChange,
  title,
  description,
  actionLabel = "知道了",
  tone = "danger"
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  actionLabel?: string;
  tone?: "danger" | "default";
}) {
  const toneClass = tone === "danger"
    ? "border-danger/20 bg-danger/8 text-danger"
    : "border-border bg-muted/60 text-foreground";

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="animate-overlay fixed inset-0 bg-background/80 backdrop-blur-sm" />
        <Dialog.Content className="animate-modal fixed left-1/2 top-1/2 w-[calc(100%-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border/60 bg-background p-6 shadow-2xl outline-none">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-3">
              <div className={`mt-0.5 inline-flex h-10 w-10 items-center justify-center rounded-xl border ${toneClass}`}>
                <AlertCircle className="h-4 w-4" />
              </div>
              <div>
                <Dialog.Title className="text-lg font-semibold text-foreground">{title}</Dialog.Title>
                <Dialog.Description asChild>
                  <div className="mt-2 text-sm leading-6 text-muted-foreground">{description}</div>
                </Dialog.Description>
              </div>
            </div>
            <Dialog.Close asChild>
              <button className="rounded-lg p-2 text-muted-foreground transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:rotate-90 hover:bg-muted hover:text-foreground" aria-label="关闭对话框">
                <X className="h-4 w-4" />
              </button>
            </Dialog.Close>
          </div>
          <div className="mt-6 flex justify-end">
            <Dialog.Close asChild>
              <Button variant={tone === "danger" ? "danger" : "default"}>{actionLabel}</Button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
