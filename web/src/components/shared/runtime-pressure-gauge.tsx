type RuntimePressureGaugeProps = {
  pressureScore: number;
  headline: string;
  summary: string;
  environment: string;
  readiness: string;
  health: string;
  enabledChannels: number;
  requestsPerMinute: number;
  tokensPerMinute: number;
  averageLatency: string;
  errorRate: string;
  cacheHitRate: string;
};

function describePressure(score: number) {
  if (score >= 75) {
    return {
      label: "紧张",
      accent: "var(--chart-3)",
      glow: "rgba(244,63,94,0.28)",
      track: "rgba(244,63,94,0.18)"
    };
  }

  if (score >= 45) {
    return {
      label: "关注",
      accent: "var(--chart-5)",
      glow: "rgba(245,158,11,0.28)",
      track: "rgba(245,158,11,0.16)"
    };
  }

  return {
    label: "稳定",
    accent: "var(--chart-4)",
    glow: "rgba(34,197,94,0.24)",
    track: "rgba(34,197,94,0.16)"
  };
}

export function RuntimePressureGauge({
  pressureScore,
  headline,
  summary,
  environment,
  readiness,
  health,
  enabledChannels,
  requestsPerMinute,
  tokensPerMinute,
  averageLatency,
  errorRate,
  cacheHitRate
}: RuntimePressureGaugeProps) {
  const tone = describePressure(pressureScore);
  const gaugeBackground = `conic-gradient(${tone.accent} 0 ${pressureScore}%, color-mix(in srgb, ${tone.accent} 20%, transparent) ${pressureScore}% 100%)`;
  const metricGroups = [
    {
      title: "服务状态",
      items: [
        { label: "后端健康", value: health },
        { label: "服务就绪", value: readiness },
        { label: "运行环境", value: environment }
      ]
    },
    {
      title: "实时负载",
      items: [
        { label: "启用渠道", value: `${enabledChannels} 个` },
        { label: "近 60 秒 RPM", value: `${requestsPerMinute}` },
        { label: "近 60 秒 TPM", value: `${tokensPerMinute}` }
      ]
    },
    {
      title: "风险信号",
      items: [
        { label: "平均耗时", value: averageLatency },
        { label: "错误率", value: errorRate },
        { label: "缓存命中率", value: cacheHitRate }
      ]
    }
  ];

  return (
    <div className="relative overflow-hidden rounded-2xl border border-border/70 bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.08),transparent_38%),linear-gradient(180deg,rgba(255,255,255,0.04),transparent)] p-5">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,rgba(255,255,255,0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.05)_1px,transparent_1px)] bg-[size:24px_24px] opacity-30" />
      <div className="relative flex h-full flex-col gap-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Runtime Gauge</p>
            <h4 className="mt-2 text-2xl font-semibold tracking-tight text-foreground">{headline}</h4>
            <div className="mt-3 inline-flex max-w-full items-center gap-3 rounded-2xl border border-border/60 bg-background/50 px-3 py-2 text-sm text-muted-foreground backdrop-blur-sm">
              <span className="h-7 w-1 rounded-full" style={{ backgroundColor: tone.accent }} />
              <p className="min-w-0 leading-6">{summary}</p>
            </div>
          </div>
          <div className="rounded-full border border-border/60 bg-background/80 px-3 py-1 text-xs font-medium text-foreground shadow-[0_0_0_1px_rgba(255,255,255,0.03)]">
            {tone.label}
          </div>
        </div>

        <div className="grid gap-6 xl:grid-cols-[minmax(220px,240px)_minmax(0,1fr)] xl:items-start">
          <div className="flex items-center justify-center xl:justify-center">
            <div className="relative flex h-56 w-56 items-center justify-center rounded-full border border-border/50 bg-background/70 shadow-[0_24px_80px_rgba(0,0,0,0.24)]">
              <div className="absolute inset-4 rounded-full" style={{ background: gaugeBackground, boxShadow: `0 0 48px ${tone.glow}` }} />
              <div className="absolute inset-[26px] rounded-full border border-background/80 bg-black/80" />
              <div className="relative flex flex-col items-center justify-center text-center">
                <span className="text-[11px] uppercase tracking-[0.24em] text-muted-foreground">Pressure</span>
                <span className="mt-3 text-5xl font-semibold tracking-tight text-foreground">{pressureScore}</span>
                <span className="mt-2 text-xs text-muted-foreground">0 越轻松 · 100 越紧张</span>
              </div>
              <div className="absolute inset-x-10 bottom-8 h-px" style={{ background: `linear-gradient(90deg, transparent, ${tone.track}, transparent)` }} />
            </div>
          </div>

          <div className="grid min-w-0 gap-4 xl:grid-cols-2">
            {metricGroups.map((group) => (
              <section key={group.title} className="min-w-0 rounded-2xl border border-border/60 bg-background/70 p-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] backdrop-blur-sm last:xl:col-span-2">
                <div className="flex items-center justify-between gap-3 border-b border-border/50 pb-3">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.22em] text-muted-foreground">{group.title}</p>
                  </div>
                  <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: tone.accent, boxShadow: `0 0 16px ${tone.glow}` }} />
                </div>
                <div className="mt-3 space-y-2.5">
                  {group.items.map((item) => (
                    <div key={item.label} className="flex min-w-0 items-center justify-between gap-4 rounded-xl border border-border/50 bg-black/25 px-3 py-3">
                      <div className="min-w-0 text-sm text-muted-foreground">{item.label}</div>
                      <div className="shrink-0 text-right text-[clamp(1rem,1.4vw,1.5rem)] font-semibold leading-none tracking-tight text-foreground">{item.value}</div>
                    </div>
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
