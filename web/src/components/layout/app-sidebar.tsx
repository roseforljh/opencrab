"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { navigationItems } from "@/lib/navigation";
export function AppSidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden w-60 shrink-0 border-r border-slate-900/80 bg-slate-950 px-4 py-5 text-slate-100 lg:flex lg:flex-col">
      <div className="mb-8">
        <p className="text-xs uppercase tracking-[0.24em] text-slate-400">OpenCrab</p>
        <h1 className="mt-3 text-xl font-semibold">控制台</h1>
        <p className="mt-2 text-sm leading-6 text-slate-400">个人部署的大模型聚合 API 管理台</p>
      </div>

      <nav className="space-y-2 text-sm text-slate-300">
        {navigationItems.map((item) => {
          const active = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`block rounded-lg px-3 py-2 transition-colors ${active ? "bg-slate-900 text-white" : "hover:bg-slate-900/70 hover:text-white"}`}
            >
              <div className="font-medium">{item.label}</div>
              <div className="mt-1 text-xs leading-5 text-slate-400">{item.description}</div>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
