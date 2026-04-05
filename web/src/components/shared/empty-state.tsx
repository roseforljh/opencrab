import type { ReactNode } from "react";
export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) {
  return (
    <div className="flex min-h-48 flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-center">
      <h4 className="text-base font-semibold text-slate-900">{title}</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-slate-500">{description}</p>
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}
