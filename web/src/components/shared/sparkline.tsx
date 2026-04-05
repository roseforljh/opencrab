export function Sparkline({ values, colorVar = "var(--chart-1)" }: { values: number[]; colorVar?: string }) {
  const width = 120;
  const height = 36;
  const max = Math.max(...values);
  const min = Math.min(...values);
  const range = max - min || 1;

  const points = values
    .map((value, index) => {
      const x = (index / (values.length - 1)) * width;
      const y = height - ((value - min) / range) * (height - 6) - 3;
      return { x, y };
    })
    ;

  const path = points.reduce((acc, point, index, array) => {
    if (index === 0) {
      return `M ${point.x} ${point.y}`;
    }

    const previous = array[index - 1];
    const controlX = (previous.x + point.x) / 2;

    return `${acc} C ${controlX} ${previous.y}, ${controlX} ${point.y}, ${point.x} ${point.y}`;
  }, "");

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-9 w-full">
      <path
        fill="none"
        stroke={colorVar}
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        d={path}
      />
    </svg>
  );
}
