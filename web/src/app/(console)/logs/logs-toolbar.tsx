"use client";

import { Search, SlidersHorizontal } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { LOG_CATEGORY_OPTIONS, type LogCategory } from "@/app/(console)/logs/log-utils";

type LogsToolbarProps = {
  category: LogCategory;
  totalCount: number;
  visibleCount: number;
  query: string;
};

export function LogsToolbar({ category, totalCount, visibleCount, query }: LogsToolbarProps) {
  const showingFiltered = visibleCount !== totalCount || query.trim().length > 0 || category !== "all";

  return (
    <form className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px_auto] lg:items-center" method="get">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          name="q"
          defaultValue={query}
          placeholder="搜索请求 ID / 模型 / 渠道 / 执行器 / 错误"
          className="pl-9"
        />
      </div>

      <div className="flex items-center gap-2">
        <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
        <Select name="category" defaultValue={category}>
          <SelectTrigger aria-label="日志分类筛选">
            <SelectValue placeholder="选择分类" />
          </SelectTrigger>
          <SelectContent>
            {LOG_CATEGORY_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-wrap items-center justify-end gap-2">
        <Badge className={cn(
          "rounded-full border px-3 py-1 text-xs font-medium ring-0",
          showingFiltered ? "border-primary/20 bg-primary/10 text-primary" : "border-border/60 bg-muted/35 text-muted-foreground"
        )}>
          显示 {visibleCount} / {totalCount}
        </Badge>
        <button type="submit" className="inline-flex h-10 items-center justify-center rounded-xl border border-border bg-background px-4 text-sm font-medium text-foreground transition hover:border-primary/30 hover:bg-primary/5">
          应用筛选
        </button>
      </div>
    </form>
  );
}
