const chartColors = {
  requests: "var(--chart-1)",
  success: "var(--chart-4)",
  errors: "var(--chart-3)"
};

export function DashboardChart({
  data
}: {
  data: { label: string; requests: number; success: number; errors: number }[];
}) {
  const width = 760;
  const height = 260;
  const padding = 20;
  const max = Math.max(...data.map((item) => item.requests));

  const buildPoints = (key: "requests" | "success" | "errors") =>
    data.map((item, index) => {
      const x = padding + (index / (data.length - 1)) * (width - padding * 2);
      const y = height - padding - (item[key] / max) * (height - padding * 2);
      return { x, y };
    });

  const buildPath = (key: "requests" | "success" | "errors") => {
    const points = buildPoints(key);

    return points.reduce((acc, point, index, array) => {
      if (index === 0) {
        return `M ${point.x} ${point.y}`;
      }

      const previous = array[index - 1];
      const controlX = (previous.x + point.x) / 2;

      return `${acc} C ${controlX} ${previous.y}, ${controlX} ${point.y}, ${point.x} ${point.y}`;
    }, "");
  };

  const requestPoints = buildPoints("requests");

  return (
    <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-b from-card to-background p-4">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,rgba(255,255,255,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.06)_1px,transparent_1px)] bg-[size:28px_28px] opacity-20 dark:opacity-100" />
      <div className="relative flex items-center justify-between pb-4">
        <div className="flex gap-4 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-2"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: chartColors.requests }} />总请求</span>
          <span className="inline-flex items-center gap-2"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: chartColors.success }} />成功</span>
          <span className="inline-flex items-center gap-2"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: chartColors.errors }} />异常</span>
        </div>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="relative h-[260px] w-full overflow-visible">
        <path fill="none" stroke={chartColors.requests} strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" d={buildPath("requests")} />
        <path fill="none" stroke={chartColors.success} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" d={buildPath("success")} />
        <path fill="none" stroke={chartColors.errors} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" d={buildPath("errors")} />
        {requestPoints.map((point, index) => {
          const item = data[index];
          return <circle key={item.label} cx={point.x} cy={point.y} r="4" fill={chartColors.requests} />;
        })}
      </svg>
      <div className="relative mt-3 grid grid-cols-6 gap-2 text-xs text-muted-foreground">
        {data.map((item) => (
          <span key={item.label}>{item.label}</span>
        ))}
      </div>
    </div>
  );
}
