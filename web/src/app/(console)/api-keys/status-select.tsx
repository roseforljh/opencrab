"use client";

import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";

const statusOptions = [
  { value: "启用", dotClassName: "bg-success ring-2 ring-success/20" },
  { value: "禁用", dotClassName: "bg-danger ring-2 ring-danger/20" },
] as const;

export function StatusSelect({ value, onValueChange }: { value: string; onValueChange: (value: string) => void }) {
  const active = statusOptions.find((item) => item.value === value) ?? statusOptions[0];

  return (
    <Select value={active.value} onValueChange={onValueChange}>
      <SelectTrigger>
        <div className="flex items-center gap-3">
          <span className={`inline-flex h-3 w-3 shrink-0 rounded-full ${active.dotClassName}`} />
          <span>{active.value}</span>
        </div>
      </SelectTrigger>
      <SelectContent>
        {statusOptions.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            <div className="flex items-center gap-3 pr-6">
              <span className={`inline-flex h-3 w-3 shrink-0 rounded-full ${option.dotClassName}`} />
              <span>{option.value}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
