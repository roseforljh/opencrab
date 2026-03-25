import * as React from 'react';
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';

export function DataTable({
  columns,
  data,
  loading = false,
  emptyMessage = 'No results.',
}) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div className='rounded-xl border border-white/10 bg-white/6 overflow-hidden'>
      <Table>
        <TableHeader className='bg-white/5 hover:bg-white/5 border-b border-white/10'>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow
              key={headerGroup.id}
              className='border-white/10 hover:bg-transparent'
            >
              {headerGroup.headers.map((header) => {
                return (
                  <TableHead
                    key={header.id}
                    className='text-white/60 font-medium'
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext(),
                        )}
                  </TableHead>
                );
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {loading ? (
            Array.from({ length: 5 }).map((_, index) => (
              <TableRow
                key={`skeleton-${index}`}
                className='border-white/10 hover:bg-white/5'
              >
                {columns.map((col, i) => (
                  <TableCell key={`skeleton-col-${i}`}>
                    <Skeleton className='h-5 w-[80%] bg-white/10' />
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                data-state={row.getIsSelected() && 'selected'}
                className='border-white/10 hover:bg-white/5 transition-colors'
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id} className='text-white/80'>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell
                colSpan={columns.length}
                className='h-24 text-center text-white/50'
              >
                {emptyMessage}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
