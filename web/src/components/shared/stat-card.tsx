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
    <section className="group relative overflow-hidden rounded-2xl border border-border bg-gradient-to-b from-card to-background p-5 shadow-sm transition-all duration-300 hover:-translate-y-0.5 hover:border-foreground/20 hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_18px_40px_rgba(0,0,0,0.22)] dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.06),0_18px_40px_rgba(0,0,0,0.55)]">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-foreground/30 to-transparent" />
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-sm font-medium text-muted-foreground">{title}</h3>
          <span className="h-2 w-2 rounded-full" style={{ backgroundColor: accent ?? "var(--chart-1)" }} />
        </div>
        <p className="mt-1 text-3xl font-semibold tracking-tight text-foreground">{value}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
        {trend ? <Sparkline values={trend} colorVar={accent ?? "var(--chart-1)"} /> : null}
      </div>
    </section>
  );
}
