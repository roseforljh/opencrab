import type { Metadata } from "next";
import type { ReactNode } from "react";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import "./globals.css";
import { ThemeProvider } from "@/components/theme-provider";
import { I18nProvider } from "@/components/i18n-provider";

// metadata 用于定义整个管理台的页面基础信息。
//
// 当前先放最基础的标题与描述，后续会再根据页面和品牌完善 SEO 与图标配置。
export const metadata: Metadata = {
  title: "OpenCrab 控制台",
  description: "面向个人部署的大模型聚合 API 管理台"
};

// RootLayout 是前端所有页面共享的根布局。
//
// 这里当前只负责包裹 html 和 body，
// 等我们开始正式实现左侧导航和顶部状态栏时，会在这里继续补统一框架。
export default function RootLayout({
  children
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <html lang="zh-CN" className="dark" suppressHydrationWarning>
      <body className={`${GeistSans.variable} ${GeistMono.variable}`}>
        <ThemeProvider defaultTheme="dark" attribute="class">
          <I18nProvider defaultLanguage="zh-CN">
            {children}
          </I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
