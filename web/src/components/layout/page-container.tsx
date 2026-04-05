import type { ReactNode } from "react";
export function PageContainer({ children }: { children: ReactNode }) {
  return <div className="flex flex-1 flex-col gap-5 px-5 py-5 md:px-6 md:py-6">{children}</div>;
}
