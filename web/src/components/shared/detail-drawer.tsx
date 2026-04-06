"use client";

import type { ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";

import { Button } from "@/components/ui/button";

export function DetailDrawer({
  title,
  description,
  triggerLabel,
  trigger,
  children
}: {
  title: string;
  description: string;
  triggerLabel: string;
  trigger?: ReactNode;
  children: ReactNode;
}) {
  return (
    <Dialog.Root>
      <Dialog.Trigger asChild>
        {trigger ?? <Button variant="ghost" size="sm" className="text-primary hover:bg-primary/10 hover:text-primary">{triggerLabel}</Button>}
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="animate-overlay fixed inset-0 bg-background/80 backdrop-blur-sm" />
        <Dialog.Content className="animate-drawer fixed right-0 top-0 flex h-full w-full max-w-xl flex-col border-l border-border/60 bg-background shadow-2xl outline-none will-change-transform">
          <div className="flex items-start justify-between border-b border-border px-6 py-5">
            <div>
              <Dialog.Title className="text-lg font-semibold text-foreground">{title}</Dialog.Title>
              <Dialog.Description className="mt-2 text-sm leading-6 text-muted-foreground">{description}</Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button className="rounded-lg p-2 text-muted-foreground transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:rotate-90 hover:bg-muted hover:text-foreground" aria-label="关闭抽屉">
                <X className="h-4 w-4" />
              </button>
            </Dialog.Close>
          </div>
          <div className="flex-1 overflow-y-auto px-6 py-6">{children}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
