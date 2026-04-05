"use client";

import Image from "next/image";
import { useTheme } from "@/components/theme-provider";
import { useI18n } from "@/components/i18n-provider";
import { useShell } from "@/components/layout/shell-provider";
import { PanelLeftClose, PanelLeftOpen, Moon, Sun, Languages } from "lucide-react";

export function Topbar() {
  const { theme, setTheme } = useTheme();
  const { language, setLanguage, t } = useI18n();
  const { collapsed, toggleCollapsed } = useShell();

  return (
    <header className="flex h-16 shrink-0 items-center justify-between border-b border-border bg-background/90 px-5 backdrop-blur md:px-6">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={toggleCollapsed}
          className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted-foreground transition-all duration-200 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:bg-accent hover:text-foreground"
          title={collapsed ? "展开侧边栏" : "收起侧边栏"}
        >
          {collapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
        </button>
        <div className="relative flex h-10 w-10 items-center justify-center overflow-hidden">
          <Image src="/logo.png" alt="OpenCrab Logo" width={28} height={28} className="object-contain pixelated" />
        </div>
        <div>
          <div className="text-sm font-semibold text-foreground">OpenCrab</div>
          <div className="text-xs text-muted-foreground">{t("topbar.gateway")}</div>
        </div>
      </div>
      <div className="flex items-center gap-3 text-sm">
        <div className="flex items-center gap-2">
          <span className="flex h-2.5 w-2.5 rounded-full bg-success shadow-[0_0_12px_rgba(34,197,94,0.45)]"></span>
          <span className="text-muted-foreground">{t("topbar.system_normal")}</span>
        </div>

        <span className="hidden rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground sm:inline-flex">
          {t("topbar.demo_data")}
        </span>

        <button
          onClick={() => setLanguage(language === "zh-CN" ? "en-US" : "zh-CN")}
          className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-background px-3 py-2 text-muted-foreground transition-[background-color,color,border-color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:bg-accent hover:text-foreground"
          title={language === "zh-CN" ? "Switch to English" : "切换到中文"}
        >
          <Languages className="h-4 w-4" />
          <span className="text-xs font-medium">{language === "zh-CN" ? t("lang.en") : t("lang.zh")}</span>
        </button>

        <button
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-background px-3 py-2 text-muted-foreground transition-[background-color,color,border-color,transform] duration-200 ease-[var(--ease-out-smooth)] hover:-translate-y-0.5 hover:bg-accent hover:text-foreground"
          title={theme === "dark" ? t("theme.light") : t("theme.dark")}
        >
          {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          <span className="hidden text-xs font-medium sm:inline">{theme === "dark" ? t("theme.light") : t("theme.dark")}</span>
        </button>
      </div>
    </header>
  );
}
