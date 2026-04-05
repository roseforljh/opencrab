# Design System — OpenCrab

## Product Context
- **What this is:** OpenCrab 是一个面向个人部署的大模型聚合 API 网关与管理台，用来统一接入多个上游模型渠道，并通过一个稳定、清晰的控制台完成配置、查看和维护。
- **Who it's for:** 主要面向个人开发者、独立开发者、技术型自部署用户。
- **Space/industry:** AI 开发者工具、自托管后台控制台、模型网关管理平台。
- **Project type:** Web app / dashboard / admin console。

## Aesthetic Direction
- **Direction:** 开发者控制台极简风格，主参考为 Vercel / Geist。
- **Decoration level:** minimal
- **Mood:** 整体气质要冷静、清晰、专业，不追求花哨视觉刺激，而是强调信息组织、操作效率和长期可维护性。用户打开管理台后，应该第一眼看到系统状态和关键信息，而不是装饰性元素。
- **Reference sites:** https://vercel.com, https://github.com/VoltAgent/awesome-design-md

## Typography
- **Display/Hero:** Geist Sans — 用于页面标题、大标题、关键数值标题。它和整体开发者工具气质最匹配，现代、干净、没有多余装饰。
- **Body:** Geist Sans — 用于正文、表单、标签、按钮文案。统一字体能降低视觉噪音，让界面更稳定。
- **UI/Labels:** Geist Sans — 控制台场景下，标题、标签、按钮统一更利于系统一致性。
- **Data/Tables:** Geist Sans（开启 tabular-nums）— 表格和统计数字尽量统一使用等宽数字表现，保证日志、耗时、计数更整齐。
- **Code:** Geist Mono — 用于 API Key、模型 ID、错误码、JSON 片段、日志明细。
- **Loading:** 使用 Next.js 字体方案优先加载 Geist Sans 与 Geist Mono，并提供中文回退字体：`PingFang SC`、`Microsoft YaHei`、`Noto Sans SC`。
- **Scale:**
  - 页面主标题：32px / 40px
  - 页面二级标题：24px / 32px
  - 区块标题：18px / 28px
  - 正文：14px / 22px
  - 辅助说明：12px / 18px
  - 超大指标数字：36px / 40px

## Color
- **Approach:** restrained
- **Primary:** `#111827` — 主文字与关键深色区域的核心色，用于强调稳定、专业、技术感。
- **Secondary:** `#2563EB` — 主交互强调色，用于按钮、链接、激活状态、焦点边框。蓝色比紫色更稳定，也更适合基础设施类产品。
- **Neutrals:**
  - `#FFFFFF` 页面主背景
  - `#F8FAFC` 次级背景
  - `#F1F5F9` 卡片弱背景
  - `#E2E8F0` 分割线与边框
  - `#94A3B8` 辅助文字
  - `#475569` 次级正文
  - `#0F172A` 深色标题与高强调文本
- **Semantic:**
  - success `#16A34A`
  - warning `#D97706`
  - error `#DC2626`
  - info `#0284C7`
- **Dark mode:** 深色模式不是简单反色，而是重新定义背景层级。背景使用 `#020617`、`#0F172A`、`#111827` 三层，状态色降低 10% 到 15% 饱和度，避免高亮色在深色背景下刺眼。

## Spacing
- **Base unit:** 4px
- **Density:** compact
- **Scale:** 2xs(2) xs(4) sm(8) md(12) lg(16) xl(24) 2xl(32) 3xl(48)

## Layout
- **Approach:** grid-disciplined
- **Grid:**
  - 1440px 以上：12 列内容网格
  - 1024px 到 1439px：12 列内容网格
  - 768px 到 1023px：8 列内容网格
  - 767px 以下：4 列内容网格
- **Max content width:** 1440px
- **Border radius:**
  - sm: 6px
  - md: 8px
  - lg: 12px
  - xl: 16px
  - full: 9999px

## Page Shell

### Overall Layout
整个管理台统一采用“左侧主导航 + 顶部状态栏 + 主内容区 + 右侧抽屉”的控制台结构。

1. **左侧主导航**
   - 固定宽度 240px
   - 放核心一级导航
   - 适合承载 Dashboard、Channels、Models、API Keys、Logs、Settings
   - 深色背景，突出全局框架感

2. **顶部状态栏**
   - 固定高度 56px
   - 左侧放页面标题与面包屑
   - 右侧放主题切换、系统状态、用户菜单
   - 背景尽量保持浅色或透明叠加，避免抢主内容注意力

3. **主内容区**
   - 使用统一页面容器
   - 上部是页面标题区
   - 中部是页面主要模块
   - 每个页面最多保留 2 到 3 层信息层级，不把页面做成信息堆场

4. **右侧抽屉**
   - 用于新增、编辑、查看详情
   - 配置型操作尽量在抽屉完成，避免频繁整页跳转

### Information Hierarchy
每个页面统一采用这三层：

1. **页面层**：说明这是哪个模块，当前能做什么。
2. **区块层**：把数据、表格、表单拆成独立区块。
3. **操作层**：把新增、保存、删除、筛选等操作放在区块标题附近。

## Core Page Layouts

### Dashboard
布局采用“顶部指标卡 + 中部趋势图 + 底部最近活动表格”。

1. 第一行显示 4 个核心指标卡：总请求量、成功率、平均耗时、活跃渠道数。
2. 第二行显示趋势图和错误率图。
3. 第三行显示最近请求日志与最近异常。

### Channels
布局采用“页面标题区 + 筛选条 + 表格区 + 编辑抽屉”。

1. 主体是渠道列表。
2. 上方提供新增按钮和搜索筛选。
3. 点击某行后在抽屉中编辑，不单独跳新页面。

### Models & Routing
布局采用“双栏结构”。

1. 左侧是模型别名与模型列表。
2. 右侧是路由规则、优先级和默认回退策略。
3. 这个页面是首版最复杂的配置页，必须把“列表”和“配置详情”分开，不能全部堆在一张大表单里。

### API Keys
布局采用“概览卡片 + 表格区 + 详情抽屉”。

1. 顶部显示当前 Key 数量、启用数量、禁用数量。
2. 下方表格展示 Key 列表。
3. 抽屉展示具体详情与状态切换。

### Logs
布局采用“紧凑型筛选栏 + 高密度表格 + JSON 明细抽屉”。

1. 顶部筛选栏必须常驻。
2. 表格支持时间、渠道、模型、状态筛选。
3. 点击记录后在右侧展开详情，显示请求摘要和响应摘要。
4. 日志页信息密度最高，应尽量减少无效留白。

### Settings
布局采用“分组配置页”。

1. 系统配置按类别分区。
2. 普通设置放在上方。
3. 危险操作区固定放最底部，并用明显边界区分。

## Components

### Primary Component Stack
1. **shadcn/ui** 作为主组件基座。
2. **Radix UI** 作为交互与无障碍基础。
3. **TanStack Table** 负责高密度表格。
4. **React Hook Form + Zod** 负责表单与校验。

### Component Usage Rules
1. 表格一律优先使用统一封装的数据表格组件。
2. 表单一律优先使用统一表单字段组件。
3. 抽屉用于编辑，弹窗用于确认。
4. 状态标签统一颜色规则，不允许同一种状态在不同页面不同颜色。

## Motion
- **Approach:** minimal-functional
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out)
- **Duration:** micro(80ms) short(150ms) medium(220ms) long(320ms)

动效规则：

1. 只保留对理解有帮助的动效。
2. 抽屉、弹窗、下拉、toast 允许轻微过渡。
3. 禁止炫技型大动画。
4. 日志和高密度表格页面尽量减少动效干扰。

## Safe Choices
- 使用 Vercel / Geist 风格作为主方向，因为开发者控制台对这套视觉语言最熟悉，学习成本最低。
- 使用左侧导航 + 顶部状态栏的标准控制台结构，因为后台管理场景天然需要稳定信息框架。
- 使用蓝色作为主交互色，因为它最适合基础设施和开发者工具场景。

## Risks
- 把整体密度做得比普通 SaaS 后台更紧凑，收益是日志页和配置页更专业，代价是必须严格控制排版和间距。
- 在深色左导航和浅色内容区之间做强对比，收益是结构清晰，代价是颜色层级必须控制好，否则容易显得割裂。
- 尽量减少装饰和插画，让页面几乎完全靠排版、边界和层级说话，收益是长期耐看，代价是组件细节必须打磨足够干净。

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-06 | 确定前端采用 Vercel / Geist 风格 | 最适合开发者控制台和日志、表格、配置表单场景 |
| 2026-04-06 | 确定组件基座为 shadcn/ui + Radix UI | 可控性高，适合长期演进 |
| 2026-04-06 | 确定整体布局为左侧导航 + 顶部状态栏 + 主内容区 + 右侧抽屉 | 最适合后台控制台信息结构 |
| 2026-04-06 | 确定主字体为 Geist Sans，代码字体为 Geist Mono | 与整体气质一致，适合开发者工具 |
