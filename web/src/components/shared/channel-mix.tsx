export function ChannelMix({
  items
}: {
  items: { label: string; value: number; color: string }[];
}) {
  return (
    <div className="space-y-4">
      {items.map((item) => (
        <div key={item.label} className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-foreground">{item.label}</span>
            <span className="text-muted-foreground">{item.value}%</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full transition-[width] duration-500 ease-out" style={{ width: `${item.value}%`, backgroundColor: item.color }} />
          </div>
        </div>
      ))}
    </div>
  );
}
