export default function ConsoleLoading() {
  return (
    <div className="flex min-h-[calc(100vh-64px)] items-center justify-center px-6 py-10">
      <div className="flex flex-col items-center gap-5">
        <div className="relative flex h-14 w-14 items-center justify-center">
          <span className="absolute inset-0 rounded-full border border-white/10" />
          <span className="absolute inset-0 rounded-full border-2 border-transparent border-t-white border-r-white/70 animate-smooth-spin" />
          <span className="absolute inset-[10px] rounded-full border border-white/10" />
        </div>
        <div className="space-y-1 text-center">
          <p className="text-sm font-medium text-foreground">页面切换中</p>
          <p className="text-xs text-muted-foreground">正在准备下一个视图...</p>
        </div>
      </div>
    </div>
  );
}
