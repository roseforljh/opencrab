import { Badge } from "@/components/ui/badge";

export function StatusBadge({ status }: { status: string }) {
  const styleMap: Record<string, string> = {
    启用: "bg-success/10 text-success ring-success/20",
    成功: "bg-success/10 text-success ring-success/20",
    禁用: "bg-muted text-muted-foreground ring-border",
    待验证: "bg-warning/10 text-warning ring-warning/20",
    异常: "bg-danger/10 text-danger ring-danger/20"
  };

  return (
    <Badge className={styleMap[status] ?? "bg-primary/10 text-primary ring-primary/20"}>
      {status}
    </Badge>
  );
}
