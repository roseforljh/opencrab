import type { ReactNode } from "react";
import { Search, ChevronDown } from "lucide-react";

import { Input } from "@/components/ui/input";

export type FilterChip = {
  label: string;
};

export function FilterBar({
  placeholder,
  chips,
  trailingAction
}: {
  placeholder: string;
  chips: FilterChip[];
  trailingAction?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border bg-muted/30 p-3 lg:flex-row lg:items-center lg:justify-between">
      <div className="flex flex-1 flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative w-full lg:max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input placeholder={placeholder} className="pl-9 bg-background" />
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
          {chips.map((chip) => (
            <button
              key={chip.label}
              className="flex items-center gap-1.5 rounded-lg bg-background px-3 py-1.5 ring-1 ring-border/60 transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:bg-secondary/50 hover:text-foreground"
            >
              {chip.label}
              <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
            </button>
          ))}
        </div>
      </div>
      {trailingAction ? <div className="shrink-0">{trailingAction}</div> : null}
    </div>
  );
}
