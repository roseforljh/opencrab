import type { ReactNode } from "react";

export default function ConsoleTemplate({ children }: { children: ReactNode }) {
  return <div className="animate-page-transition will-change-transform">{children}</div>;
}
