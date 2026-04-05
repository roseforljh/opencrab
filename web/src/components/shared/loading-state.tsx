export function LoadingState({ label = "加载中" }: { label?: string }) {
  return (
    <div className="flex min-h-48 items-center justify-center rounded-xl border border-dashed border-border/60 bg-secondary/20 px-6 py-10 text-sm text-muted-foreground">
      <div className="flex items-center gap-3">
        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        <span>{label}...</span>
      </div>
    </div>
  );
}
