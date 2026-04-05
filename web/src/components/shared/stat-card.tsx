import { Sparkline } from "@/components/shared/sparkline";

export function StatCard({
  title,
  description,
  value,
  trend,
  accent
}: {
  title: string;
  description: string;
  value: string;
  trend?: number[];
  accent?: string;
}) {
  return (
    <section className="group relative overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm transition-all duration-300 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:border-foreground/20 hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_8px_20px_rgba(0,0,0,0.1)] dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_8px_20px_rgba(0,0,0,0.3)]">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-foreground/20 to-transparent" />
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-xs font-medium text-muted-foreground">{title}</h3>
          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: accent ?? "var(--chart-1)" }} />
        </div>
        <div className="mt-1 flex items-baseline gap-2">
          <p className="text-2xl font-semibold tracking-tight text-foreground">{value}</p>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
        {trend ? <div className="mt-2 h-8"><Sparkline values={trend} colorVar={accent ?? "var(--chart-1)"} /></div> : null}
      </div>
    </section>
  );
}
