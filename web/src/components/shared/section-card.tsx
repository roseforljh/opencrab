import type { ReactNode } from "react";
export function SectionCard({
  title,
  description,
  action,
  children,
  className = ""
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm ${className}`.trim()}>
      <div className="flex items-center justify-between gap-4 border-b border-border/40 bg-muted/20 px-5 py-3.5">
        <div>
          <h3 className="text-sm font-medium text-foreground">{title}</h3>
          {description ? <p className="mt-0.5 text-xs text-muted-foreground">{description}</p> : null}
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </div>
      <div className="flex-1 p-5">{children}</div>
    </section>
  );
}
