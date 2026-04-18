import type { ReactNode } from "react";

import { EmptyState } from "@/components/shared/empty-state";

export type StaticTableColumn<T> = {
  header: string;
  cell: (row: T) => ReactNode;
  className?: string;
};

export function StaticTable<T>({
  columns,
  data,
  emptyTitle,
  emptyDescription,
  rowAction
}: {
  columns: StaticTableColumn<T>[];
  data: T[];
  emptyTitle: string;
  emptyDescription: string;
  rowAction?: (row: T, index: number) => ReactNode;
}) {
  if (data.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <div className="overflow-x-auto overflow-y-hidden rounded-xl border border-border bg-background shadow-sm">
      <table className="min-w-full divide-y divide-border/50 text-sm">
        <thead className="bg-secondary/30 text-left text-muted-foreground">
          <tr>
            {columns.map((column) => (
              <th key={column.header} className="px-4 py-3.5 font-medium">
                {column.header}
              </th>
            ))}
            {rowAction ? <th className="px-4 py-3.5 font-medium">操作</th> : null}
          </tr>
        </thead>
        <tbody className="divide-y divide-border/50 bg-background">
          {data.map((row, index) => (
            <tr key={index} className="transition-[background-color] duration-200 ease-[var(--ease-out-smooth)] hover:bg-secondary/30">
              {columns.map((column) => (
                <td key={column.header} className={`px-4 py-3.5 text-foreground ${column.className ?? ""}`.trim()}>
                  {column.cell(row)}
                </td>
              ))}
              {rowAction ? <td className="px-4 py-3.5 text-right">{rowAction(row, index)}</td> : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
