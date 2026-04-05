import type { ReactNode } from "react";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { Topbar } from "@/components/layout/topbar";
export default function ConsoleLayout({ children }: { children: ReactNode }) {
  return (
    <main className="min-h-screen bg-slate-50 text-slate-900">
      <section className="mx-auto flex min-h-screen w-full max-w-[1440px]">
        <AppSidebar />
        <div className="flex min-h-screen flex-1 flex-col bg-white">
          <Topbar />
          {children}
        </div>
      </section>
    </main>
  );
}
