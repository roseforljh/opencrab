# OpenCrab 前端计划

## 1. 前端目标

前端的目标是提供一个清晰、克制、适合开发者使用的管理台。

这个管理台不是营销官网，也不是花哨展示页，而是一个真正用于：

1. 配置渠道。
2. 配置模型映射。
3. 查看日志。
4. 管理 API Key。
5. 查看系统状态。

## 2. 前端技术栈

1. 框架：Next.js
2. 样式：Tailwind CSS
3. 组件：shadcn/ui
4. 底层交互：Radix UI
5. 表格：TanStack Table
6. 表单：React Hook Form + Zod

## 3. 视觉方向

前端统一采用 Vercel / Geist 风格。

这个方向强调：

1. 纯净。
2. 克制。
3. 高信息密度。
4. 适合日志、表格、配置表单。

## 4. 字体方案

### 主字体

- Geist Sans

### 等宽字体

- Geist Mono

### 中文回退字体

- PingFang SC
- Microsoft YaHei
- Noto Sans SC

## 5. 页面结构

首版页面固定为：

1. Dashboard
2. Channels
3. Models & Routing
4. API Keys
5. Logs
6. Settings

## 6. 布局结构

统一采用：

1. 左侧导航。
2. 顶部状态栏。
3. 主内容区。
4. 抽屉用于编辑。
5. 对话框用于确认。

## 7. 组件策略

1. 表格页面统一使用 TanStack Table。
2. 表单页面统一使用 React Hook Form + Zod。
3. 抽屉、弹窗、Tabs、Popover 统一使用 Radix 风格交互。
4. 所有视觉组件都尽量复用 shadcn/ui 基座。

## 8. 页面职责

### Dashboard

展示系统概况、请求趋势、错误率、最近调用情况。

### Channels

管理上游渠道，支持新增、编辑、禁用、测试连通性。

### Models & Routing

管理模型别名、模型映射、路由优先级。

### API Keys

管理访问密钥，支持创建、查看状态、禁用。

### Logs

展示请求日志摘要，支持筛选、查看详情。

### Settings

管理全局设置和基础系统参数。

## 9. 前端验证重点

1. 管理台能正常启动。
2. 页面布局统一。
3. 中文界面可读性好。
4. 页面都能接真实 API。
5. 表格、表单、抽屉交互一致。
