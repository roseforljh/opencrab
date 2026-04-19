"use client";

import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState, useTransition } from "react";
import { Search, SlidersHorizontal } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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

export function ClearLogsButton({
  category,
  totalCount,
  visibleCount,
  query
}: {
  category: LogCategory;
  totalCount: number;
  visibleCount: number;
  query: string;
}) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [isConfirming, setIsConfirming] = useState(false);
  const [draftQuery, setDraftQuery] = useState(query);
  const [draftCategory, setDraftCategory] = useState<LogCategory>(category);
  const initializedRef = useRef(false);
  const showingFiltered = visibleCount !== totalCount || query.trim().length > 0 || category !== "all";

  const targetUrl = useMemo(() => {
    const params = new URLSearchParams();
    const nextQuery = draftQuery.trim();
    if (nextQuery) {
      params.set("q", nextQuery);
    }
    if (draftCategory !== "all") {
      params.set("category", draftCategory);
    }
    return params.toString() ? `/logs?${params.toString()}` : "/logs";
  }, [draftCategory, draftQuery]);

  useEffect(() => {
    setDraftQuery(query);
  }, [query]);

  useEffect(() => {
    setDraftCategory(category);
  }, [category]);

  useEffect(() => {
    if (!initializedRef.current) {
      initializedRef.current = true;
      return;
    }

    const currentParams = new URLSearchParams();
    const currentQuery = query.trim();
    if (currentQuery) {
      currentParams.set("q", currentQuery);
    }
    if (category !== "all") {
      currentParams.set("category", category);
    }
    const currentUrl = currentParams.toString() ? `/logs?${currentParams.toString()}` : "/logs";

    if (targetUrl === currentUrl) {
      return;
    }

    const timer = window.setTimeout(() => {
      router.replace(targetUrl, { scroll: false });
    }, 250);

    return () => window.clearTimeout(timer);
  }, [category, query, router, targetUrl]);

  async function handleConfirm() {
    setError(null);

    const response = await fetch("/api/admin/logs", {
      method: "DELETE"
    });

    if (!response.ok) {
      const message = await response.text();
      setError(message || "清空日志失败");
      return;
    }

    setIsConfirming(false);
    startTransition(() => {
      router.refresh();
    });
  }

  return (
    <div className="space-y-4">
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_240px_auto] lg:items-center">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={draftQuery}
            onChange={(event) => setDraftQuery(event.target.value)}
            placeholder="搜索请求 ID / 模型 / 渠道 / 执行器 / 错误"
            className="pl-9"
          />
        </div>

        <div className="flex items-center gap-2">
          <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
          <Select value={draftCategory} onValueChange={(value) => setDraftCategory(value as LogCategory)}>
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
          {showingFiltered ? (
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                setDraftQuery("");
                setDraftCategory("all");
                router.replace("/logs", { scroll: false });
              }}
            >
              重置
            </Button>
          ) : null}
          <Button type="button" variant="outline" className="gap-2 text-danger hover:text-danger" disabled={isPending} onClick={() => setIsConfirming(true)}>
            一键清空日志
          </Button>
        </div>
      </div>

      {isConfirming ? (
        <div className="rounded-xl border border-danger/20 bg-danger/5 p-4 text-sm text-foreground">
          <div className="font-medium text-danger">确认清空全部请求日志？</div>
          <p className="mt-1 text-muted-foreground">会立即删除当前全部请求日志记录，此操作不可撤销。</p>
          <div className="mt-3 flex gap-2">
            <Button type="button" variant="outline" size="sm" onClick={handleConfirm} disabled={isPending}>
              {isPending ? "刷新中..." : "确认清空"}
            </Button>
            <Button type="button" variant="ghost" size="sm" onClick={() => setIsConfirming(false)} disabled={isPending}>
              取消
            </Button>
          </div>
        </div>
      ) : null}
      {error ? <p className="text-xs text-danger">{error}</p> : null}
    </div>
  );
}
