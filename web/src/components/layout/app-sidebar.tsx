"use client";

import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { Gauge, Network, Route, KeyRound, ScrollText, Settings2 } from "lucide-react";

import { useI18n } from "@/components/i18n-provider";
import { useShell } from "@/components/layout/shell-provider";
import { navigationItems } from "@/lib/navigation";

const navIconMap = {
  dashboard: Gauge,
  channels: Network,
  models: Route,
  apikeys: KeyRound,
  logs: ScrollText,
  settings: Settings2
} as const;

export function AppSidebar() {
  const pathname = usePathname();
  const { t } = useI18n();
  const { collapsed } = useShell();
  const [pendingHref, setPendingHref] = useState<string | null>(null);

  useEffect(() => {
    setPendingHref(null);
  }, [pathname]);

  return (
    <aside
      className={`hidden h-screen shrink-0 border-r border-border bg-background transition-[width] duration-300 ease-[var(--ease-emphasized)] lg:flex lg:flex-col ${
        collapsed ? "w-20" : "w-64"
      }`}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-6">
        <div className={`mb-8 ${collapsed ? "px-0" : "px-2"}`}>
          <div className={`flex items-center ${collapsed ? "justify-center" : "gap-3"}`}>
            <div className="relative flex h-11 w-11 items-center justify-center overflow-hidden">
              <Image src="/logo.png" alt="OpenCrab Logo" width={30} height={30} className="object-contain pixelated" />
            </div>
            <div
              className={`overflow-hidden whitespace-nowrap transition-[max-width,opacity,transform] duration-300 ease-[var(--ease-emphasized)] ${
                collapsed ? "max-w-0 opacity-0 -translate-x-2" : "max-w-[220px] opacity-100 translate-x-0"
              }`}
            >
              <div>
                <h1 className="text-lg font-semibold leading-none tracking-tight">{t("app.name")}</h1>
                <p className="mt-1 text-xs text-muted-foreground">{t("app.description")}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="mb-5 h-px bg-gradient-to-r from-transparent via-foreground/15 to-transparent" />
        <nav className="space-y-1">
          {navigationItems.map((item) => {
            const active = pathname === item.href;
            const switching = pendingHref === item.href;
            const Icon = navIconMap[item.id as keyof typeof navIconMap];

            return (
              <Link
                key={item.href}
                href={item.href}
                prefetch={true}
                onClick={() => {
                  if (!active) {
                    setPendingHref(item.href);
                  }
                }}
                aria-current={active ? "page" : undefined}
                title={collapsed ? t(`nav.${item.id}`) : undefined}
                className={`group relative flex w-full items-center overflow-hidden rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-200 ease-[var(--ease-out-smooth)] ${
                  active
                    ? "border border-white/12 bg-[linear-gradient(180deg,rgba(255,255,255,0.14),rgba(255,255,255,0.06))] text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.18),inset_0_-1px_0_rgba(255,255,255,0.04),0_10px_30px_rgba(0,0,0,0.22)] backdrop-blur-xl before:absolute before:inset-[1px] before:rounded-[11px] before:border before:border-white/6 before:content-['']"
                    : "border border-transparent text-muted-foreground hover:border-white/6 hover:bg-white/4 hover:text-foreground"
                } ${switching ? "animate-pulse opacity-80" : ""}`}
              >
                <div className={`flex h-5 w-5 shrink-0 items-center justify-center ${collapsed ? "mx-auto" : "mr-3"}`}>
                  <Icon className={`h-4 w-4 ${active ? "drop-shadow-[0_0_8px_rgba(255,255,255,0.18)]" : ""}`} strokeWidth={2} />
                </div>
                <div
                  className={`relative z-[1] flex min-w-0 flex-1 items-center justify-between overflow-hidden whitespace-nowrap transition-[max-width,opacity,transform] duration-300 ease-[var(--ease-emphasized)] ${
                    collapsed ? "max-w-0 opacity-0 translate-x-1" : "max-w-[220px] opacity-100 translate-x-0"
                  }`}
                >
                  <span className="truncate">{t(`nav.${item.id}`)}</span>
                  {switching ? <span className="ml-3 text-[10px] uppercase tracking-widest text-muted-foreground">...</span> : null}
                </div>
              </Link>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}
