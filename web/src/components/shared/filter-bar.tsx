import type { ReactNode } from "react";
import { Search, ChevronDown } from "lucide-react";

import { Input } from "@/components/ui/input";

export type FilterChip = {
  label: string;
  active?: boolean;
  onClick?: () => void;
};

export function FilterBar({
  placeholder,
  chips,
  trailingAction,
  searchValue,
  onSearchValueChange,
  showChipIcon = true,
  enableActiveStyle = true
}: {
  placeholder: string;
  chips: FilterChip[];
  trailingAction?: ReactNode;
  searchValue?: string;
  onSearchValueChange?: (value: string) => void;
  showChipIcon?: boolean;
  enableActiveStyle?: boolean;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border bg-muted/30 p-3 lg:flex-row lg:items-center lg:justify-between">
      <div className="flex flex-1 flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative w-full lg:max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={placeholder}
            value={searchValue}
            onChange={onSearchValueChange ? (event) => onSearchValueChange(event.target.value) : undefined}
            className="bg-background pl-9"
          />
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
          {chips.map((chip) => (
            <button
              key={chip.label}
              type="button"
              onClick={chip.onClick}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 ring-1 transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-smooth)] ${
                chip.active && enableActiveStyle
                  ? "bg-foreground text-background ring-foreground/20"
                  : "bg-background ring-border/60 hover:-translate-y-0.5 hover:bg-secondary/50 hover:text-foreground"
              }`}
            >
              {chip.label}
              {showChipIcon ? (
                <ChevronDown className={`h-3.5 w-3.5 ${chip.active && enableActiveStyle ? "text-background/80" : "text-muted-foreground"}`} />
              ) : null}
            </button>
          ))}
        </div>
      </div>
      {trailingAction ? <div className="shrink-0">{trailingAction}</div> : null}
    </div>
  );
}
