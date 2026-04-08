import type { ReactNode } from "react";
export function PageContainer({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`flex flex-1 flex-col gap-5 px-5 py-5 md:px-6 md:py-6 ${className}`.trim()}>{children}</div>;
}
