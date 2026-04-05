import type { ReactNode } from "react";

export function ErrorState({
  title,
  description,
  action
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex min-h-48 flex-col items-center justify-center rounded-xl border border-danger/20 bg-danger/5 px-6 py-10 text-center">
      <h4 className="text-base font-semibold text-danger">{title}</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-danger/80">{description}</p>
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}
