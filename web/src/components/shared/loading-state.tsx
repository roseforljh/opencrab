export function LoadingState({ label = "加载中" }: { label?: string }) {
  return (
    <div className="flex min-h-48 items-center justify-center rounded-xl border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-sm text-slate-500">
      {label}...
    </div>
  );
}
