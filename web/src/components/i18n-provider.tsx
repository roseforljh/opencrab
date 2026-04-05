"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { dictionaries, type Language } from "@/lib/i18n-shared";

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
  const router = useRouter();

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
        document.cookie = `opencrab-ui-lang=${lang}; path=/; max-age=31536000; samesite=lax`;
        setLanguage(lang);
        router.refresh();
      },
      t,
    }),
    [language, router, storageKey, t]
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
