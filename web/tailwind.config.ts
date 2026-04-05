import type { Config } from "tailwindcss";

// 这个配置文件用于把 DESIGN.md 中已经确定的设计方向映射到前端样式体系。
//
// 当前先定义最基础的扫描范围与主题扩展入口，
// 后续会继续把颜色、圆角、阴影、字号等 token 全部收敛到这里。
const config: Config = {
  content: [
    "./src/app/**/*.{ts,tsx}",
    "./src/components/**/*.{ts,tsx}",
    "./src/lib/**/*.{ts,tsx}"
  ],
  theme: {
    extend: {
      colors: {
        border: "#E2E8F0",
        background: "#FFFFFF",
        foreground: "#0F172A",
        muted: "#F8FAFC",
        accent: "#2563EB",
        success: "#16A34A",
        warning: "#D97706",
        danger: "#DC2626"
      },
      borderRadius: {
        sm: "6px",
        md: "8px",
        lg: "12px",
        xl: "16px"
      },
      boxShadow: {
        soft: "0 8px 24px rgba(15, 23, 42, 0.06)"
      }
    }
  },
  plugins: []
};

export default config;
