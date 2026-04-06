"use client";

import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";

const statusOptions = [
  { value: "启用", color: "bg-success/10 text-success ring-success/20" },
  { value: "禁用", color: "bg-danger/10 text-danger ring-danger/20" }
] as const;

export function StatusSelect({ value, onValueChange }: { value: string; onValueChange: (value: string) => void }) {
  const active = statusOptions.find((item) => item.value === value) ?? statusOptions[0];

  return (
    <Select value={active.value} onValueChange={onValueChange}>
      <SelectTrigger>
        <div className="flex items-center gap-3">
          <span className={`inline-flex h-2.5 w-2.5 rounded-full ${active.color}`} />
          <span>{active.value}</span>
        </div>
      </SelectTrigger>
      <SelectContent>
        {statusOptions.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            <div className="flex items-center gap-3 pr-6">
              <span className={`inline-flex h-2.5 w-2.5 rounded-full ${option.color}`} />
              <span>{option.value}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
