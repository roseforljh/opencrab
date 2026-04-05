"use client";

import type { ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";

import { Button } from "@/components/ui/button";

export function DetailDrawer({
  title,
  description,
  triggerLabel,
  children
}: {
  title: string;
  description: string;
  triggerLabel: string;
  children: ReactNode;
}) {
  return (
    <Dialog.Root>
      <Dialog.Trigger asChild>
        <Button variant="ghost" size="sm" className="text-blue-600 hover:bg-blue-50 hover:text-blue-700">{triggerLabel}</Button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-slate-950/20 backdrop-blur-[1px]" />
        <Dialog.Content className="fixed right-0 top-0 flex h-full w-full max-w-xl flex-col border-l border-slate-200 bg-white shadow-2xl outline-none">
          <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
            <div>
              <Dialog.Title className="text-lg font-semibold text-slate-950">{title}</Dialog.Title>
              <Dialog.Description className="mt-2 text-sm leading-6 text-slate-500">{description}</Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button className="rounded-lg p-2 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900" aria-label="关闭抽屉">
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
