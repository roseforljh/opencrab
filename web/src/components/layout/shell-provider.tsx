"use client";

import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

type ShellContextValue = {
  collapsed: boolean;
  toggleCollapsed: () => void;
};

const ShellContext = createContext<ShellContextValue | null>(null);

export function ShellProvider({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    const saved = window.localStorage.getItem("opencrab-shell-collapsed");
    if (saved === "true") {
      setCollapsed(true);
    }
  }, []);

  useEffect(() => {
    document.body.dataset.scrollShell = "internal";

    return () => {
      delete document.body.dataset.scrollShell;
    };
  }, []);

  const value = useMemo(
    () => ({
      collapsed,
      toggleCollapsed: () => {
        setCollapsed((current) => {
          const next = !current;
          window.localStorage.setItem("opencrab-shell-collapsed", String(next));
          return next;
        });
      }
    }),
    [collapsed]
  );

  return <ShellContext.Provider value={value}>{children}</ShellContext.Provider>;
}

export function useShell() {
  const context = useContext(ShellContext);
  if (!context) {
    throw new Error("useShell must be used within ShellProvider");
  }

  return context;
}
