export type NavigationItem = {
  id: string;
  href: string;
  label: string;
  description: string;
};

export const navigationItems: NavigationItem[] = [
  { id: "dashboard", href: "/", label: "Dashboard", description: "系统概览与状态总览" },
  { id: "channels", href: "/channels", label: "Channels", description: "管理上游渠道与连通状态" },
  { id: "models", href: "/models", label: "Models & Routing", description: "管理模型映射与路由策略" },
  { id: "apikeys", href: "/api-keys", label: "API Keys", description: "管理访问密钥与启用状态" },
  { id: "logs", href: "/logs", label: "Logs", description: "查看请求日志与异常明细" },
  { id: "settings", href: "/settings", label: "Settings", description: "管理全局设置与危险操作" }
];
