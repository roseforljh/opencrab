import { Sparkline } from "@/components/shared/sparkline";

export function StatCard({
  title,
  description,
  value,
  trend,
  accent,
  valueClassName
}: {
  title: string;
  description?: string;
  value: string;
  trend?: number[];
  accent?: string;
  valueClassName?: string;
}) {
  return (
    <section className="group relative overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm transition-all duration-300 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:border-foreground/20 hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_8px_20px_rgba(0,0,0,0.1)] dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_8px_20px_rgba(0,0,0,0.3)]">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-foreground/20 to-transparent" />
      <div className="flex h-full flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <h3 className="text-xs font-medium text-muted-foreground">{title}</h3>
            {description ? <p className="text-xs leading-5 text-muted-foreground/80">{description}</p> : null}
          </div>
          <span className="mt-1 h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: accent ?? "var(--chart-1)" }} />
        </div>

        <div className="flex min-h-[56px] items-end">
          <p className={`text-3xl font-semibold tracking-tight text-foreground ${valueClassName ?? ""}`.trim()}>{value}</p>
        </div>

        <div className="mt-auto">{trend ? <div className="h-8"><Sparkline values={trend} colorVar={accent ?? "var(--chart-1)"} /></div> : <div className="h-8" />}</div>
      </div>
    </section>
  );
}
