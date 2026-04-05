# Semi UI 到 shadcn/ui 迁移设计规范

## 1. 背景与目标

### 1.1 迁移背景
OpenCrab 前端目前同时使用两套 UI 组件库：
- **@douyinfe/semi-ui**（下称 Semi UI）— 现有主要组件库
- **@base-ui/react** + **shadcn/ui** + **lucide-react**（下称 shadcn）— 新引入的现代化组件库

当前 `components/ui/` 目录已完成向 shadcn 的迁移，但大量业务组件仍重度依赖 Semi UI。本文档定义全量迁移的设计规范与实施路径。

### 1.2 迁移目标
- 移除所有 `@douyinfe/semi-ui` 和 `@douyinfe/semi-icons` 依赖
- 统一使用 shadcn/ui + @base-ui/react + lucide-react + @lobehub/icons 作为组件库
- 保持功能行为完全一致，用户无感知
- 逐步推进，避免单次大规模变更引入风险

### 1.3 迁移原则
1. **功能等价优先**：迁移后组件行为须与原 Semi 组件一致
2. **分批推进**：按优先级分阶段迁移，每阶段可独立测试
3. **不破坏现有功能**：仅做替换，不在迁移过程中重构业务逻辑
4. **样式变量映射**：建立 Semi CSS 变量到 Tailwind/CSS 变量的映射表

---

## 2. 技术选型与替代方案

### 2.1 组件替代映射

| Semi UI 组件 | 替代方案 | 备注 |
|---|---|---|
| `Typography` (Text/Title) | shadcn `Text` + Tailwind CSS | Typography 直接替换 |
| `Button` | shadcn `button` | 行为一致 |
| `Modal` | shadcn `dialog` | 需要调整 API |
| `Tabs` / `TabPane` | shadcn `tabs` | 需要封装以支持 card type + collapsible |
| `Tag` | shadcn `badge` | 视觉对齐 |
| `Card` | shadcn `card` | 行为一致 |
| `Skeleton` | shadcn `skeleton` | 行为一致 |
| `Pagination` | shadcn `pagination` 或自实现 | `createCardProPagination` 需要改造 |
| `Form` / `Input` / `Select` | shadcn `form` + `input` + `select` |  |
| `Checkbox` | shadcn `checkbox` |  |
| `Switch` | shadcn `switch` |  |
| `Slider` | shadcn `slider` |  |
| `Tooltip` | shadcn `tooltip` |  |
| `Popover` | shadcn `popover` |  |
| `Dropdown` / `Menu` | shadcn `dropdown-menu` |  |
| `Avatar` | shadcn `avatar` |  |
| `Badge` | shadcn `badge` |  |
| `Spinner` / `Spin` | shadcn `spinner` 或 Lucide `Loader2` |  |
| `Empty` | shadcn `empty` 或自实现 |  |
| `Banner` | shadcn `alert` 或自实现 |  |
| `Collapsible` | shadcn `collapsible` | CardTable 移动端使用 |
| `Divider` | shadcn `separator` |  |
| `Calendar` | shadcn `calendar` | CheckinCalendar 需要定制 |
| `Image` | HTML `img` |  |
| `Notification` | shadcn `sonner` (已引入) |  |
| `Toast` | shadcn `sonner` (已引入) | `helpers/utils.jsx` 需要改造 |
| `LocaleProvider` | i18next 原生方案 | `index.jsx` 全局改造 |
| `Space` | Tailwind `flex gap-x` | 通用模式 |

### 2.2 图标替代映射

| Semi Icons | 替代方案 |
|---|---|
| `IconKey` | `lucide-react` `Key` |
| `IconMail` | `lucide-react` `Mail` |
| `IconLock` | `lucide-react` `Lock` |
| `IconDelete` | `lucide-react` `Trash2` |
| `IconUser` | `lucide-react` `User` |
| `IconCopy` | `lucide-react` `Copy` |
| `IconClose` | `lucide-react` `X` |
| `IconMenu` | `lucide-react` `Menu` |
| `IconExit` | `lucide-react` `LogOut` |
| `IconUserSetting` | `lucide-react` `Settings` |
| `IconEdit` | `lucide-react` `Pencil` |
| `IconChevronDown` | `lucide-react` `ChevronDown` |
| `IconChevronUp` | `lucide-react` `ChevronUp` |
| `IconTreeTriangleDown` | `lucide-react` `ChevronDown` |
| `IconMore` | `lucide-react` `MoreHorizontal` |
| `IconAlertTriangle` | `lucide-react` `AlertTriangle` |
| `IconPlus` | `lucide-react` `Plus` |
| `IconEyeOpened` / `IconEyeClosed` | `lucide-react` `Eye` / `EyeOff` |
| `IconCopy` | `lucide-react` `Copy` |

> 注：`@lobehub/icons` 已部分引入，可作为补充图标库。

### 2.3 样式变量映射

Semi 的 CSS 变量在多个文件中直接引用内联样式，迁移时需要建立映射：

| Semi CSS 变量 | 映射目标 |
|---|---|
| `--semi-color-text-0` | `text-foreground` |
| `--semi-color-text-1` | `text-foreground` |
| `--semi-color-text-2` | `text-muted-foreground` |
| `--semi-color-primary` | `bg-primary` |
| `--semi-color-warning` | `text-yellow-500` |
| `--semi-color-danger` | `text-destructive` |
| `--semi-color-border` | `border-border` |
| `--semi-color-fill-0` | `bg-muted/50` |
| `--semi-color-warning-light-hover` | `border-yellow-200` |
| `--semi-color-danger-light-hover` | `border-red-200` |
| `--semi-color-danger-light-default` | `bg-red-50` |

---

## 3. 遗留模块详细分析

### 3.1 优先级 P0 — 入口层（阻塞全局）

#### `web/src/index.jsx`
- 引入 `@douyinfe/semi-ui/dist/css/semi.css`（全局样式）
- `LocaleProvider` 顶层包装用于 i18n
- **影响**：所有页面
- **改造方案**：
  1. 移除 `semi.css` 引入，依赖 Tailwind CSS
  2. 将 `LocaleProvider` 替换为 i18next 原生方案（`i18n.use()` 即可，无需 Provider）
  3. 确认 zh_CN / en_GB locale 数据是否还有额外作用

### 3.2 优先级 P1 — 核心业务

#### `components/common/ui/CardTable.jsx`
- 桌面端使用 Semi `Table`，移动端使用 Semi `Card` + `Skeleton` + `Collapsible`
- 分叉逻辑复杂，是最难迁移的文件之一
- **改造方案**：
  1. 桌面端：将 Semi `Table` 替换为 ag-grid 或自实现 table
  2. 移动端：将 Semi `Card` + `Skeleton` 替换为 shadcn `card` + `skeleton`
  3. Collapsible 使用 shadcn `collapsible`
  4. 或者：使用 `@tanstack/react-table` 统一实现，桌面端用原生 table，移动端用卡片列表

#### `components/table/channels/ChannelsTabs.jsx`
- Semi `Tabs` + `TabPane` + `Tag`，带 `collapsible` 和 `tabBarExtraContent`
- TabPane 内嵌渠道图标和 Tag
- **改造方案**：封装 shadcn `tabs` 以支持 card type + collapsible 行为

#### `components/table/models/ModelsTabs.jsx`
- Semi `Tabs` + `TabPane` + `Tag` + `Dropdown`（内含 IconEdit/IconDelete）+ `Modal.confirm`
- Tab 内的 Dropdown Menu 包含操作按钮，与 Modal 联动
- **改造方案**：
  1. shadcn `tabs` 替代 Tabs
  2. shadcn `dropdown-menu` 替代 Dropdown Menu
  3. shadcn `alert-dialog` 替代 Modal.confirm

#### `components/table/channels/ChannelsColumnDefs.jsx` / `TokensColumnDefs.jsx`
- 混用 Semi Icons 和 shadcn 组件
- `SplitButtonGroup`（Semi）需要评估是否有必要保留
- **改造方案**：统一使用 shadcn button 或 button group 替代

#### `helpers/render.jsx`
- `renderGroup` 等函数返回 Semi `Modal` / `Tag` / `Typography` / `Avatar`
- 被多个页面调用
- **改造方案**：
  1. 将返回的 Semi JSX 改为返回 shadcn 组件
  2. 确保调用方正确导入

#### `helpers/utils.jsx`
- `createCardProPagination` 函数中直接使用 Semi `Pagination`
- **改造方案**：
  1. 提取 Pagination 组件为独立文件
  2. 使用 shadcn `pagination` 组件
  3. 或使用 `@base-ui/react` 的分页组件

### 3.3 优先级 P2 — 通用组件

| 文件 | 关键组件 | 改造说明 |
|---|---|---|
| `CardPro.jsx` | Card + Divider + Typography + Button + Icons | shadcn 替代 |
| `MarkdownRenderer.jsx` | Button + Tooltip + Toast + IconCopy | shadcn 替代 Toast/Tooltip |
| `JSONEditor.jsx` | Button + Icons + 状态管理 | 图标替换 |
| `SelectableButtonGroup.jsx` | ButtonGroup + Icons | shadcn button group |
| `Loading.jsx` | Spin | 替换为 Lucide `Loader2` |
| `RenderUtils.jsx` | Space + Tag + Typography + Popover | shadcn 替代 |
| `ChannelKeyDisplay.jsx` | Card + Button + Typography + Tag | shadcn 替代 |
| `RiskAcknowledgementModal.jsx` | Modal + Checkbox + Input（防复制粘贴）| 最复杂：需要 shadcn dialog + checkbox + input，保留防复制逻辑 |

### 3.4 优先级 P3 — 表单类设置页

#### `pages/Setting/Operation/` — 7 个页面
- `SettingsChannelAffinity`、`SettingsCheckin`、`SettingsCreditLimit`、`SettingsHeaderNavModules`、`SettingsLog`、`SettingsSensitiveWords`、`SettingsSidebarModulesAdmin`
- 共同模式：`Form` + `Row/Col` + `Spin` + `Typography`
- **改造方案**：
  1. 使用 shadcn `form`（基于 react-hook-form + zod）
  2. `Row/Col` 替换为 Tailwind grid/flex
  3. Spin 替换为 Loader2
  4. Typography 替换为 Tailwind

#### `pages/Setting/Model/` — 5 个页面
- `SettingClaudeModel`、`SettingGeminiModel`、`SettingGrokModel`、`SettingGlobalModel`、`SettingModelDeployment`
- 部分文件直接 import Semi `Text`：`import Text from '@douyinfe/semi-ui/lib/es/typography/text'`
- **改造方案**：替换为 Tailwind `span`/`p`/`h1-h6`

#### `pages/Setting/RateLimit/` 和 `pages/Setting/Performance/`
- 类似表单模式，统一改造

### 3.5 优先级 P4 — 相对独立

| 文件 | 关键组件 | 改造说明 |
|---|---|---|
| `SetupWizard.jsx` | Card + Title/Text（直接 import）| 样式变量映射，Title/Text 替换 |
| `DatabaseStep.jsx` | Banner | shadcn alert 替代 |
| `AdminStep.jsx` | Banner + IconKey | 组件 + 图标替换 |
| `components/settings/personal/cards/CheckinCalendar.jsx` | Calendar | shadcn calendar 定制 |
| `components/settings/personal/cards/AccountManagement.jsx` | Icons | 纯图标替换 |
| `components/settings/personal/modals/*.jsx` | Modal + Input + Button | shadcn dialog + input |
| `components/layout/headerbar/UserArea.jsx` | Avatar + Button + Dropdown + Typography + Icons | 批量替换 |
| `components/layout/headerbar/*.jsx` | Button + Dropdown + Badge + Icons | 批量替换 |
| `helpers/dashboard.jsx` | Progress + Divider + Empty | 替换 |
| `pages/Forbidden`、`pages/NotFound` | Empty | 简单替换 |

---

## 4. 迁移实施计划

### 阶段一：入口与基础设施（P0）
**目标**：解除全局阻塞依赖
1. 移除 `index.jsx` 中的 `semi.css` 引入
2. 替换 `LocaleProvider` 为 i18next 原生方案
3. 验证所有页面正常渲染

### 阶段二：核心业务组件（P1）
**目标**：迁移用户最常用的管理界面
1. `helpers/render.jsx` — 工具函数改造
2. `helpers/utils.jsx` — 分页组件改造
3. `components/table/channels/ChannelsTabs.jsx`
4. `components/table/models/ModelsTabs.jsx`
5. `components/common/ui/CardTable.jsx`（最难，单独测试）
6. `components/table/channels/ChannelsColumnDefs.jsx`
7. `components/table/tokens/TokensColumnDefs.jsx`

### 阶段三：通用组件库（P2）
**目标**：扫清剩余通用组件
1. `components/common/ui/` 剩余文件
2. `components/common/modals/RiskAcknowledgementModal.jsx`
3. `components/layout/headerbar/` 全部 6 个文件

### 阶段四：设置页面批量改造（P3）
**目标**：完成所有 Settings 表单页
1. `pages/Setting/Operation/` 全部 7 个页面
2. `pages/Setting/Model/` 全部 5 个页面
3. `pages/Setting/RateLimit/`
4. `pages/Setting/Performance/`

### 阶段五：收尾（P4）
**目标**：完成所有遗留文件
1. `components/setup/`
2. `components/settings/personal/`
3. `pages/Forbidden`、`pages/NotFound`
4. `helpers/dashboard.jsx`

---

## 5. 风险与缓解措施

| 风险 | 影响 | 缓解措施 |
|---|---|---|
| Semi CSS 变量散落各处 | 样式不一致 | 建立变量映射表，逐文件替换 |
| `LocaleProvider` 涉及全局 i18n | 迁移失败影响所有页面 | 先在测试页面验证 i18next 原生方案可行性 |
| `CardTable` 双模逻辑复杂 | 迁移后移动端体验退化 | 使用 @tanstack/react-table 统一实现，或保留两套独立实现 |
| `SplitButtonGroup` 无直接替代 | 功能缺失 | 使用 shadcn `button-group` 或自定义实现 |
| `createCardProPagination` 在工具函数中嵌入组件 | 违反关注点分离 | 重构为独立组件文件 |
| 图标替换后尺寸/对齐不一致 | 视觉差异 | 统一使用 lucide-react 的 default size，通过 className 调整 |

---

## 6. 验收标准

1. `package.json` 中移除 `@douyinfe/semi-ui` 和 `@douyinfe/semi-icons`
2. 全项目 `grep "@douyinfe/semi" **/*.{js,jsx}"` 无结果
3. `index.jsx` 中无 Semi CSS 引入
4. 所有页面功能测试通过
5. 移动端和桌面端响应式布局正常
6. i18n 中英文切换正常
