"use client";

import type { ReactNode } from "react";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";

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
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50 text-left text-slate-500">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id} className="px-4 py-3 font-medium">
                  {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
              {rowAction ? <th className="px-4 py-3 font-medium">操作</th> : null}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-slate-200 bg-white">
          {table.getRowModel().rows.map((row, index) => (
            <tr key={row.id} className="transition-colors hover:bg-slate-50/50">
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id} className="px-4 py-3 text-slate-600">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
              {rowAction ? <td className="px-4 py-3 text-right">{rowAction(row.original, index)}</td> : null}
            </tr>
          ))}
        </tbody>
      </table>
      <div className="flex items-center justify-between border-t border-slate-200 bg-slate-50 px-4 py-3 sm:px-6">
        <div className="flex flex-1 justify-between sm:hidden">
          <button className="relative inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">
            上一页
          </button>
          <button className="relative ml-3 inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">
            下一页
          </button>
        </div>
        <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
          <div>
            <p className="text-sm text-slate-700">
              显示第 <span className="font-medium">1</span> 到 <span className="font-medium">{data.length}</span> 条，共 <span className="font-medium">{data.length}</span> 条
            </p>
          </div>
          <div>
            <nav className="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
              <button className="relative inline-flex items-center rounded-l-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0">
                <span className="sr-only">上一页</span>
                <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fillRule="evenodd" d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z" clipRule="evenodd" />
                </svg>
              </button>
              <button aria-current="page" className="relative z-10 inline-flex items-center bg-blue-600 px-4 py-2 text-sm font-semibold text-white focus:z-20 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600">1</button>
              <button className="relative inline-flex items-center rounded-r-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0">
                <span className="sr-only">下一页</span>
                <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fillRule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clipRule="evenodd" />
                </svg>
              </button>
            </nav>
          </div>
        </div>
      </div>
    </div>
  );
}
