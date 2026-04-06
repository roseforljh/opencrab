"use client";

import * as React from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

export const Select = SelectPrimitive.Root;
export const SelectValue = SelectPrimitive.Value;

export function SelectTrigger({ className, children, ...props }: React.ComponentProps<typeof SelectPrimitive.Trigger>) {
  return (
    <SelectPrimitive.Trigger
        className={cn(
          "flex h-10 w-full cursor-pointer items-center justify-between rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground outline-none transition-[border-color,box-shadow,background-color,transform] duration-200 ease-[var(--ease-out-smooth)] placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/15 data-[state=open]:border-primary/30 data-[state=open]:bg-card",
          className
        )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon asChild>
        <ChevronDown className="h-4 w-4 text-muted-foreground" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  );
}

export function SelectContent({ className, children, position = "popper", ...props }: React.ComponentProps<typeof SelectPrimitive.Content>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        position={position}
        side="bottom"
        align="start"
        className={cn(
          "z-50 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-2xl border border-border bg-background text-foreground shadow-lg animate-select will-change-[opacity,transform]",
          className
        )}
        sideOffset={6}
        {...props}
      >
        <SelectPrimitive.Viewport className="min-w-[var(--radix-select-trigger-width)] p-1">{children}</SelectPrimitive.Viewport>
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  );
}

export function SelectItem({ className, children, ...props }: React.ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
        className={cn(
          "relative flex w-full cursor-pointer select-none items-center gap-3 rounded-xl px-3 py-2.5 text-sm outline-none transition-[background-color,color,transform] duration-150 ease-[var(--ease-out-smooth)] data-[highlighted]:bg-muted data-[highlighted]:text-foreground data-[state=checked]:bg-primary/10 data-[state=checked]:text-primary",
          className
        )}
      {...props}
    >
      <span className="absolute right-3 flex h-3.5 w-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <Check className="h-4 w-4" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  );
}
