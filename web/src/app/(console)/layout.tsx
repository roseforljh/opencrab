import type { ReactNode } from "react";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { ShellProvider } from "@/components/layout/shell-provider";
import { Topbar } from "@/components/layout/topbar";
export default function ConsoleLayout({ children }: { children: ReactNode }) {
  return (
    <ShellProvider>
      <main className="h-screen overflow-hidden bg-background text-foreground">
        <section className="flex h-full w-full">
          <AppSidebar />
          <div className="flex min-w-0 flex-1 flex-col bg-background">
            <Topbar />
            <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">{children}</div>
          </div>
        </section>
      </main>
    </ShellProvider>
  );
}
