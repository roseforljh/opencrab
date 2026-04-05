import { Badge } from "@/components/ui/badge";

export function StatusBadge({ status }: { status: string }) {
  const styleMap: Record<string, string> = {
    启用: "bg-emerald-50 text-emerald-700 ring-emerald-100",
    成功: "bg-emerald-50 text-emerald-700 ring-emerald-100",
    禁用: "bg-slate-100 text-slate-600 ring-slate-200",
    待验证: "bg-amber-50 text-amber-700 ring-amber-100",
    异常: "bg-rose-50 text-rose-700 ring-rose-100"
  };

  return (
    <Badge className={styleMap[status] ?? "bg-blue-50 text-blue-700 ring-blue-100"}>
      {status}
    </Badge>
  );
}
