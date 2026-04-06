"use client";

import type { ReactNode } from "react";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/shared/empty-state";

export function DataTable<TData>({
  columns,
  data,
  emptyTitle,
  emptyDescription,
  rowAction
}: {
  columns: ColumnDef<TData>[];
  data: TData[];
  emptyTitle: string;
  emptyDescription: string;
  rowAction?: (row: TData, index: number) => ReactNode;
}) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  if (data.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-background shadow-sm">
      <table className="min-w-full divide-y divide-border/50 text-sm">
        <thead className="bg-secondary/30 text-left text-muted-foreground">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id} className="px-4 py-3.5 font-medium">
                  {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
              {rowAction ? <th className="px-4 py-3.5 font-medium">操作</th> : null}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-border/50 bg-background">
          {table.getRowModel().rows.map((row, index) => (
            <tr key={row.id} className="transition-[background-color] duration-200 ease-[var(--ease-out-smooth)] hover:bg-secondary/30">
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id} className="px-4 py-3.5 text-foreground">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
              {rowAction ? <td className="px-4 py-3.5 text-right">{rowAction(row.original, index)}</td> : null}
            </tr>
          ))}
        </tbody>
      </table>
      <div className="flex items-center justify-between border-t border-border/50 bg-secondary/10 px-4 py-3 sm:px-6">
        <div className="flex flex-1 justify-between sm:hidden">
          <Button variant="outline" size="sm">上一页</Button>
          <Button variant="outline" size="sm">下一页</Button>
        </div>
        <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
          <div>
            <p className="text-sm text-muted-foreground">
              显示第 <span className="font-medium text-foreground">1</span> 到 <span className="font-medium text-foreground">{data.length}</span> 条，共 <span className="font-medium text-foreground">{data.length}</span> 条
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm">上一页</Button>
            <Button variant="secondary" size="sm">1</Button>
            <Button variant="outline" size="sm">下一页</Button>
          </div>
        </div>
      </div>
    </div>
  );
}
