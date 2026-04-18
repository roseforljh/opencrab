import type { ReactNode } from "react";
import { redirect } from "next/navigation";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { ScrollToTopButton } from "@/components/layout/scroll-to-top-button";
import { ShellProvider } from "@/components/layout/shell-provider";
import { Topbar } from "@/components/layout/topbar";
import { getAdminAuthStatus } from "@/lib/admin-api-server";

export default async function ConsoleLayout({ children }: { children: ReactNode }) {
	const authStatus = await getAdminAuthStatus();
	if (!authStatus.initialized) {
		redirect("/init");
	}
	if (!authStatus.authenticated) {
		redirect("/login");
	}

  return (
    <ShellProvider>
      <main className="h-screen overflow-hidden bg-background text-foreground">
        <section className="flex h-full w-full">
          <AppSidebar />
          <div className="flex min-w-0 flex-1 flex-col bg-background">
            <Topbar />
            <div id="console-scroll-root" className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden [scrollbar-gutter:stable]">
              {children}
            </div>
          </div>
          <ScrollToTopButton targetId="console-scroll-root" />
        </section>
      </main>
    </ShellProvider>
  );
}
