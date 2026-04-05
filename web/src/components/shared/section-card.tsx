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
    <section className={`rounded-xl border border-border bg-background p-5 shadow-sm ${className}`.trim()}>
      <div className="flex items-start justify-between gap-4 border-b border-border/50 pb-4">
        <div>
          <h3 className="text-base font-semibold text-foreground">{title}</h3>
          {description ? <p className="mt-1.5 text-sm text-muted-foreground">{description}</p> : null}
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}
