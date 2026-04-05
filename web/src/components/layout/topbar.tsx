export function Topbar() {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-5 md:px-6">
      <div className="text-sm text-slate-500">OpenCrab Personal Gateway</div>
      <div className="flex items-center gap-3 text-sm text-slate-500">
        <span className="rounded-full bg-emerald-50 px-3 py-1 text-emerald-700">系统正常</span>
        <span>浅色模式</span>
        <span className="rounded-full bg-slate-100 px-3 py-1 text-slate-600">本地演示数据</span>
      </div>
    </header>
  );
}
