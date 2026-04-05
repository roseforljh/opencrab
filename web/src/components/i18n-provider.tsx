"use client";

import * as React from "react";

type Language = "zh-CN" | "en-US";

type I18nProviderProps = {
  children: React.ReactNode;
  defaultLanguage?: Language;
  storageKey?: string;
};

type I18nProviderState = {
  language: Language;
  setLanguage: (language: Language) => void;
  t: (key: string) => string;
};

const dictionaries: Record<Language, Record<string, string>> = {
  "zh-CN": {
    "app.name": "OpenCrab",
    "app.description": "个人部署的大模型聚合 API 管理台",
    "nav.dashboard": "仪表盘",
    "nav.channels": "渠道管理",
    "nav.models": "模型路由",
    "nav.apikeys": "API Keys",
    "nav.logs": "请求日志",
    "nav.settings": "系统设置",
    "topbar.system_normal": "系统正常",
    "topbar.demo_data": "本地演示数据",
    "topbar.gateway": "个人网关",
    "theme.light": "浅色模式",
    "theme.dark": "深色模式",
    "theme.system": "跟随系统",
    "lang.zh": "中文",
    "lang.en": "English",
    "dashboard.title": "系统概览",
    "dashboard.description": "优先展示最重要的健康信息、关键指标和最近活动，让首页承担真正的系统总览职责。",
    "channels.title": "渠道管理",
    "channels.description": "统一管理上游渠道、兼容类型、地址、状态和模型覆盖范围。",
    "models.title": "模型映射与路由",
    "models.description": "配置对外暴露的模型别名，并设置它们如何路由到具体的上游渠道模型。",
    "apikeys.title": "访问密钥",
    "apikeys.description": "管理访问密钥、状态、用量和最近使用记录。",
    "logs.title": "请求日志",
    "logs.description": "集中查看请求日志、状态码、耗时和 JSON 明细。",
    "settings.title": "系统设置",
    "settings.description": "管理 OpenCrab 的全局配置、运行策略和高风险操作。",
    "common.create": "新建",
    "common.edit": "编辑",
    "common.save": "保存",
    "common.details": "查看详情",
    "common.search": "搜索",
  },
  "en-US": {
    "app.name": "OpenCrab",
    "app.description": "Personal LLM API Gateway",
    "nav.dashboard": "Dashboard",
    "nav.channels": "Channels",
    "nav.models": "Models",
    "nav.apikeys": "API Keys",
    "nav.logs": "Logs",
    "nav.settings": "Settings",
    "topbar.system_normal": "System Normal",
    "topbar.demo_data": "Demo Data",
    "topbar.gateway": "Personal Gateway",
    "theme.light": "Light",
    "theme.dark": "Dark",
    "theme.system": "System",
    "lang.zh": "中文",
    "lang.en": "English",
    "dashboard.title": "Overview",
    "dashboard.description": "Surface the most important health signals, key metrics, and recent activity on the home screen.",
    "channels.title": "Channels",
    "channels.description": "Manage upstream providers, compatibility type, endpoint, and availability.",
    "models.title": "Models & Routing",
    "models.description": "Configure public aliases and define how they route to upstream models.",
    "apikeys.title": "API Keys",
    "apikeys.description": "Manage access keys, status, usage, and recent activity.",
    "logs.title": "Request Logs",
    "logs.description": "Inspect request history, status codes, latency, and JSON details.",
    "settings.title": "Settings",
    "settings.description": "Manage global configuration, runtime policy, and dangerous actions.",
    "common.create": "Create",
    "common.edit": "Edit",
    "common.save": "Save",
    "common.details": "Details",
    "common.search": "Search",
  },
};

const initialState: I18nProviderState = {
  language: "zh-CN",
  setLanguage: () => null,
  t: (key: string) => key,
};

const I18nProviderContext = React.createContext<I18nProviderState>(initialState);

export function I18nProvider({
  children,
  defaultLanguage = "zh-CN",
  storageKey = "opencrab-ui-lang",
  ...props
}: I18nProviderProps) {
  const [language, setLanguage] = React.useState<Language>(defaultLanguage);

  React.useEffect(() => {
    const savedLang = localStorage.getItem(storageKey) as Language;
    if (savedLang && (savedLang === "zh-CN" || savedLang === "en-US")) {
      setLanguage(savedLang);
    }
  }, [storageKey]);

  React.useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  const t = React.useCallback(
    (key: string) => {
      return dictionaries[language][key] || key;
    },
    [language]
  );

  const value = React.useMemo(
    () => ({
      language,
      setLanguage: (lang: Language) => {
        localStorage.setItem(storageKey, lang);
        setLanguage(lang);
      },
      t,
    }),
    [language, storageKey, t]
  );

  return (
    <I18nProviderContext.Provider {...props} value={value}>
      {children}
    </I18nProviderContext.Provider>
  );
}

export const useI18n = () => {
  const context = React.useContext(I18nProviderContext);

  if (context === undefined)
    throw new Error("useI18n must be used within a I18nProvider");

  return context;
};
