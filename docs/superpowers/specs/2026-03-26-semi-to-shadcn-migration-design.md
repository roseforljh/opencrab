# Semi UI 到 shadcn/ui 全量迁移设计

- 日期：2026-03-26
- 项目：opencrab
- 范围：`web/` 前端
- 目标：彻底移除 `@douyinfe/semi-ui`、`@douyinfe/semi-icons`、`@douyinfe/vite-plugin-semi` 及相关全局样式/入口依赖，统一到现有 shadcn 风格组件体系

## 1. 背景与目标

当前仓库前端同时存在两套 UI 体系：

1. 新的 shadcn 风格基础层：`web/src/components/ui/*`、Tailwind、`@base-ui/react`、`lucide-react`、`sonner`
2. 旧的 Semi UI 残留：入口层、核心表格区、设置页、顶栏、个人设置、setup 流程、helpers 与 hooks

用户要求将所有 Semi 彻底替换为 shadcn/ui，并明确偏好：

- 迁移范围：彻底替换，不保留 Semi 依赖
- 执行策略：单分支分阶段落地
- 验收优先级：功能优先，迁移过程中主流程始终可运行
- 目标倾向：一次性完成完整迁移，不保留 Semi 残骸

## 2. 现状摘要

### 2.1 Semi 使用现状

当前仓库中 Semi 仍广泛存在于：

- 入口层：`web/src/index.jsx`
- 构建层：`web/vite.config.js`
- 核心业务表格：`web/src/components/table/**/*`
- 通用 UI：`web/src/components/common/ui/**/*`
- 顶栏与布局：`web/src/components/layout/**/*`
- 设置页：`web/src/pages/Setting/**/*`
- 个人设置：`web/src/components/settings/personal/**/*`
- setup 流程：`web/src/components/setup/**/*`
- helpers/hooks：`web/src/helpers/*`、`web/src/hooks/*`

典型残留包括：

- `@douyinfe/semi-ui/dist/css/semi.css`
- `LocaleProvider`
- `Toast` / `Notification`
- `Modal` / `Modal.confirm`
- `Tabs` / `TabPane`
- `Typography`
- `Button` / `Dropdown` / `Tag` / `Avatar`
- `Form` / `Row` / `Col`
- `Pagination` / `Empty` / `Spin` / `Banner`
- `@douyinfe/semi-icons`

### 2.2 已有 shadcn 基础

项目已经存在一套较成熟的 shadcn 风格基础设施，可直接复用：

- `web/src/components/ui/*`
- `web/src/lib/utils.js` 的 `cn()`
- Tailwind + `index.css` 设计 token
- `@base-ui/react`
- `class-variance-authority`
- `tailwind-merge`
- `lucide-react`
- `sonner`
- `@tanstack/react-table`

## 2.3 现有 shadcn 组件完整性审计

当前已确认 `web/src/components/ui/` 中存在并可复用的组件包括：

- `button.jsx`
- `input.jsx`
- `textarea.jsx`
- `select.jsx`
- `switch.jsx`
- `checkbox.jsx`
- `tabs.jsx`
- `dropdown-menu.jsx`
- `dialog.jsx`
- `alert-dialog.jsx`
- `badge.jsx`
- `avatar.jsx`
- `card.jsx`
- `tooltip.jsx`
- `popover.jsx`
- `table.jsx`
- `data-table/index.jsx`
- `pagination.jsx`
- `skeleton.jsx`
- `progress.jsx`
- `calendar.jsx`
- `navigation-menu.jsx`
- `scroll-area.jsx`
- `separator.jsx`
- `slider.jsx`
- `label.jsx`
- `sonner.jsx`

结论：当前仓库已具备本次迁移所需的大部分基础组件，不需要从零建设 shadcn 组件库。

执行要求：

- Phase 0 需逐项确认上述组件在当前分支上的可用性与 API 适配程度
- 若发现个别组件能力不足，应在进入业务迁移前先补足基础能力
- 不允许在 Phase 3 以后才回头补基础组件，否则会导致并行阶段返工

## 2.4 `components/table/` 与 `components/common/ui/` 的职责关系

本次迁移中，这两个目录承担不同职责：

- `web/src/components/common/ui/`：放置跨页面复用的基础壳与通用能力，例如 `CardTable.jsx`、`CardPro.jsx`
- `web/src/components/table/`：放置 channel / models / tokens 等业务域的页面级表格、tabs、column defs、业务弹窗与操作逻辑

迁移原则：

- `CardTable.jsx` 作为项目级表格壳在 Phase 2 先稳定下来
- `components/table/` 下各业务文件在 Phase 3 基于稳定后的 `CardTable` 和 `components/ui/*` 完成业务层迁移
- 不新增第二套表格壳，不允许在 `components/table/` 内复制一套新的通用表格实现


## 3. 设计原则

1. **彻底替换**：迁移完成后不保留任何 Semi 运行时依赖或入口绑定
2. **功能优先**：短期允许局部视觉细节与当前实现不完全一致，但核心交互、页面可用性与数据操作必须稳定
3. **先基础后业务**：先拔掉全局 Semi 根，再替高复用组件与核心业务区
4. **复用现有基础**：优先使用 `web/src/components/ui/*` 与现有 shadcn 风格能力
5. **不做兼容层包袱**：不构建长期存在的“伪 Semi 兼容层”
6. **不顺手重构业务逻辑**：本次只做 UI 与交互层迁移，不借机重写状态管理、接口层或页面逻辑
7. **保持可验证**：每个阶段结束后都能进行搜索、构建与页面验证

## 4. 迁移边界

### 4.1 本次覆盖

- `web/src/index.jsx`
- `web/vite.config.js`
- `web/src/components/table/**/*`
- `web/src/components/common/ui/**/*`
- `web/src/components/common/modals/**/*`
- `web/src/components/common/markdown/**/*`
- `web/src/components/layout/**/*`
- `web/src/pages/Setting/**/*`
- `web/src/components/settings/**/*`
- `web/src/components/setup/**/*`
- `web/src/components/auth/**/*`
- `web/src/helpers/**/*`
- `web/src/hooks/**/*` 中使用 Semi 的文件
- 所有 `@douyinfe/semi-icons` 使用点

### 4.2 明确不做

- 不重做整体视觉设计系统
- 不引入新的大型表单框架作为迁移前提
- 不全面重写页面状态管理
- 不为保留旧 API 而引入长期兼容层
- 不扩展超出当前需求的新功能

## 5. 统一替换策略

### 5.1 基础组件映射

| Semi 能力 | 替代方案 |
|---|---|
| Button | `components/ui/button` |
| Input | `components/ui/input` |
| TextArea | `components/ui/textarea` |
| Select | `components/ui/select` |
| Switch | `components/ui/switch` |
| Checkbox | `components/ui/checkbox` |
| Tabs / TabPane | `components/ui/tabs` |
| Dropdown / Menu | `components/ui/dropdown-menu` |
| Modal | `components/ui/dialog` |
| Modal.confirm | `components/ui/alert-dialog` 或受控确认弹窗 |
| Tag | `components/ui/badge` |
| Avatar | `components/ui/avatar` |
| Card | `components/ui/card` |
| Tooltip | `components/ui/tooltip` |
| Popover | `components/ui/popover` |
| Badge | `components/ui/badge` |
| Table | `components/ui/data-table` + `components/ui/table` |
| Pagination | `components/ui/pagination` |
| Skeleton | `components/ui/skeleton` |
| Empty | 项目内统一空状态组合 |
| Spin | loading 组件 / skeleton |
| Typography | 原生标签 + Tailwind class |
| Banner | alert/card notice block |
| Notification / Toast | `sonner` |
| Calendar | `components/ui/calendar` |
| Progress | `components/ui/progress` |

### 5.2 图标替换策略

- 全量移除 `@douyinfe/semi-icons`
- 通用操作图标优先迁移到 `lucide-react`
- 品牌类图标继续保留现有自定义 logo 组件
- 语义不完全等价时，以行为和可识别性优先，不追求像素级一致

典型映射：

- `IconEdit` → `Pencil`
- `IconDelete` → `Trash2`
- `IconClose` → `X`
- `IconMenu` → `Menu`
- `IconSearch` → `Search`
- `IconCopy` → `Copy`
- `IconAlertTriangle` → `TriangleAlert`
- `IconMail` → `Mail`
- `IconLock` → `Lock`
- `IconUser` → `User`
- `IconExit` → `LogOut`
- `IconChevronUp/Down` → `ChevronUp/Down`
- `IconPlus` → `Plus`
- `IconEyeOpened/Closed` → `Eye` / `EyeOff`
- `IconKey` → `Key`

### 5.3 表单策略

设置页大量采用 Semi `Form + Row + Col + Spin` 模式，本次不引入全站新表单框架。

迁移方式：

- 保留现有提交逻辑、状态逻辑与接口交互
- 仅替换渲染层：`Form` → 原生表单结构 / 分组容器
- `Row/Col` → Tailwind `grid` / `flex`
- `Spin` → skeleton / loading state

理由：

- 迁移目标是切 UI，不是重写页面逻辑
- 这样更符合“功能优先、快速收敛”的要求

### 5.4 设置页表单统一规范

为避免 Phase 5 中不同文件各自发明迁移写法，设置页统一遵循以下规范：

- 页面区块：使用 `card` 或现有设置分区容器承载，不再使用 Semi `Form` 外壳
- 字段分组：使用统一的 field-group 结构，推荐“标题/说明 + 控件区”两段式布局
- 布局方式：
  - 单列表单使用垂直 `space-y-*`
  - 双列或多列区域使用 Tailwind `grid`，不再使用 `Row/Col`
- 输入组件：统一使用现有 `input`、`textarea`、`select`、`switch`、`checkbox`、`button`
- 提交区：统一放在区块底部，主操作按钮在右侧或末尾，避免每页自定义按钮摆放风格
- 加载态：
  - 首屏加载使用 skeleton 或局部 loading block
  - 保存中状态直接体现在按钮 disabled/loading 文案，不再依赖 Semi `Spin` 包裹整个表单
- 错误提示：
  - 全局提交失败、保存失败统一使用 `sonner`
  - 字段级错误沿用现有业务逻辑输出，不额外引入新表单校验体系
- 文本样式：标题、字段标签、说明文案一律使用原生标签 + Tailwind class，不再使用 `Typography`

该规范的目标是保证设置页迁移后结构一致、维护成本可控，同时不强迫页面重写状态逻辑。

## 6. 阶段划分

### Phase 0：替换基线建立

目标：在真正改大面积文件前先统一规则，避免不同模块各自发明替代方式，并完成关键技术预研。

输出：

- 组件映射表
- 图标映射表
- 通知/弹窗/分页/空状态/加载态统一方案
- 仍依赖 `--semi-color-*` 的位置清单
- `web/src/components/ui/` 现有组件可用性审计
- Tabs 使用模式审计（区分简单 UI 替换与涉及状态迁移的场景）
- `CardTable` 重建方案草图（桌面 data-table、移动卡片模式、对外 props 边界）

说明：

- 如果发现基础组件缺口，必须在 Phase 0 或 Phase 1 补齐，不能拖到业务迁移阶段再处理
- Phase 0 不只是文档整理，而是明确后续并行实施的技术前提

### Phase 1：全局根切换

目标文件：

- `web/src/index.jsx`
- `web/vite.config.js`
- `web/src/helpers/utils.jsx`
- `web/src/helpers/render.jsx`
- `web/src/helpers/dashboard.jsx`

工作内容：

- 删除 `semi.css`
- 删除 `LocaleProvider` 与相关 locale import
- 去掉 Vite Semi 插件
- 将 helpers 中的 `Toast`、`Pagination`、`Modal`、`Tag`、`Typography`、`Avatar`、`Progress`、`Empty` 等间接依赖迁移到新体系

验收：

- 全局入口不再依赖 Semi
- 构建链不再依赖 Semi
- helper 不再向业务页隐式扩散 Semi 能力
- 最小验证：前端可构建，首页可加载，`index.jsx` / `vite.config.js` / helpers 中不再存在 Semi import

### Phase 2：基础通用组件层迁移

目标文件：

- `web/src/components/common/ui/CardTable.jsx`
- `CardPro.jsx`
- `JSONEditor.jsx`
- `Loading.jsx`
- `RenderUtils.jsx`
- `SelectableButtonGroup.jsx`
- `ChannelKeyDisplay.jsx`
- `MarkdownRenderer.jsx`
- `RiskAcknowledgementModal.jsx`

关键策略：

#### 6.2.1 CardTable 作为迁移枢纽

`CardTable.jsx` 是本次迁移的核心文件：

- 桌面端统一接入 `data-table`
- 移动端保留卡片式列表模式
- 内部去掉 Semi Table/Card/Skeleton/Collapsible 依赖
- 对外 props 尽量保持稳定，减少上层页面联动改造成本

#### 6.2.2 CardPro 与通用容器

- 改为纯 shadcn 容器
- 不再依赖 Semi `Card`/`Divider`/`Typography`
- 保持当前页面壳职责，不额外抽象新层

#### 6.2.3 JSONEditor / MarkdownRenderer / SelectableButtonGroup

- 用现有 input/button/tooltip/popover/dropdown 等能力重组
- 保留现有交互行为，不做功能性重设计

### Phase 3：核心业务区迁移

目标目录：

- `web/src/components/table/channels/**/*`
- `web/src/components/table/models/**/*`
- `web/src/components/table/tokens/**/*`

#### channels

重点：

- `ChannelsTabs.jsx`
- `ChannelsTable.jsx`
- `ChannelsColumnDefs.jsx`
- 各类 modals（尤其 `EditChannelModal.jsx`）

策略：

- Tabs 改受控 `tabs`
- Tag 改 `badge`
- Empty 改统一空状态
- 图标改 lucide
- Modal 改 `dialog/alert-dialog`
- 表格操作区统一到 `button/dropdown-menu/popover`

#### models

重点：

- `ModelsTabs.jsx`
- `ModelsTable.jsx`
- `ModelsColumnDefs.jsx`
- 相关管理弹窗

策略：

- 去掉 `TabPane` 风格写法
- Dropdown + confirm 流程改 shadcn 组合
- Banner 改 alert/card notice block

#### tokens

重点：

- `TokensColumnDefs.jsx`
- `TokensTable.jsx`
- `index.jsx`
- 相关 token modals

策略：

- `SplitButtonGroup` 改为主按钮 + `dropdown-menu`
- Notification/Toast 全换成 `sonner`
- 对外行为保持一致，不追求旧组件 API 对齐

验收：

- channel / models / tokens 三个主页面不再 import Semi
- 主要表格、筛选、弹窗、分页、批量操作可正常工作
- 最小验证：三大主页面均可进入，tab 切换、表格渲染、常见弹窗打开/关闭、常见操作按钮响应正常

### Phase 4：布局层与头部区域迁移

目标目录：

- `web/src/components/layout/headerbar/**/*`
- `web/src/components/layout/NoticeModal.jsx`
- `web/src/components/layout/components/SkeletonWrapper.jsx`

重点文件：

- `UserArea.jsx`
- `ThemeToggle.jsx`
- `NotificationButton.jsx`
- `MobileMenuButton.jsx`
- `NewYearButton.jsx`
- `HeaderLogo.jsx`

策略：

- Dropdown 改 `dropdown-menu`
- Avatar 改 `avatar`
- Badge 改 `badge`
- Typography 改原生文本 + class
- Button 全部统一 `button`

### Phase 5：设置页批量迁移

目标目录：

- `web/src/pages/Setting/Operation/**/*`
- `web/src/pages/Setting/Model/**/*`
- `web/src/pages/Setting/RateLimit/**/*`
- `web/src/pages/Setting/Performance/**/*`
- `web/src/components/settings/**/*`

策略：

- 采用“保状态逻辑、换渲染层”的轻量迁移
- `Form` → 原生表单分组
- `Row/Col` → `grid` / `flex`
- `Input/Select/Button/Switch/Checkbox` → 现有 ui 组件
- `Spin` → loading / skeleton
- `Tag` → `badge`
- `Banner` → alert/card notice
- 严格遵循第 5.4 节“设置页表单统一规范”，避免不同文件形成不同迁移风格

重点高复杂页面：

- `SettingsChannelAffinity.jsx`
- `SettingsLog.jsx`
- `SettingsHeaderNavModules.jsx`
- `SettingsSidebarModulesAdmin.jsx`
- `SettingGlobalModel.jsx`
- `SettingModelDeployment.jsx`
- `SettingsPerformance.jsx`

验收：

- 设置页目标目录中不再有 Semi import
- 常见输入、切换、选择、保存动作可正常工作
- 最小验证：至少抽检一个 Operation 页面、一个 Model 页面、一个 RateLimit/Performance 页面，确认表单渲染、值回填、保存按钮和提示反馈正常

### Phase 6：个人设置 / setup / 零散页

目标目录：

- `web/src/components/settings/personal/**/*`
- `web/src/components/setup/**/*`
- `web/src/pages/Forbidden/index.jsx`
- `web/src/pages/NotFound/index.jsx`
- `web/src/components/auth/LoginForm.jsx`
- `web/src/components/common/logo/**/*`
- `web/src/components/common/DocumentRenderer/index.jsx`

说明：

- 这部分分散，但不再阻塞主流程
- 主要是弹窗、空状态、表单容器、图标与加载态替换

验收：

- 目标目录中不再残留 Semi import
- login / setup / personal / 空状态页均可正常进入和基本交互

### Phase 7：依赖移除与收尾

工作内容：

- 删除 `@douyinfe/semi-ui`
- 删除 `@douyinfe/semi-icons`
- 删除 `@douyinfe/vite-plugin-semi`
- 如无使用，删除 `@visactor/vchart-semi-theme`
- 清理所有 unused imports / dead code
- 搜索并清零所有 Semi 相关引用
- 评估并逐步清理 `--semi-color-*` token 映射

最终搜索目标：

- 0 处 `@douyinfe/semi-ui`
- 0 处 `@douyinfe/semi-icons`
- 0 处 `vite-plugin-semi`
- 0 处 `semi.css`

验收：

- 前端构建通过
- 关键页面抽检通过
- package / vite 配置 / 入口层 / 业务层均已不含 Semi 残留

## 7. 最难点与对应策略

### 7.1 `web/src/components/common/ui/CardTable.jsx`

难点：

- 同时承担桌面表格与移动端卡片渲染
- 内部高度耦合 Semi Table/Card/Skeleton/Collapsible

策略：

- 直接作为“项目级表格壳”重建
- 对外 API 尽量稳定
- 内部统一使用 shadcn/data-table 体系

### 7.2 `ChannelsTabs.jsx` / `ModelsTabs.jsx`

难点：

- Semi Tabs、Tag、Dropdown、操作按钮耦合度高

策略：

- 放弃兼容旧 `TabPane` 写法
- 直接改为 shadcn `tabs` 的受控实现

### 7.3 `helpers/utils.jsx` / `helpers/render.jsx`

难点：

- 属于隐藏扩散源，很多页面通过 helper 间接使用 Semi

策略：

- 在迁移前期优先处理
- 否则后续每个业务域都会持续被 Semi 反向污染

### 7.4 表单页

难点：

- 数量多、重复模式多、字段复杂度不一

策略：

- 不引入全站新表单方案
- 保留行为逻辑，仅替换布局与输入组件

## 8. Agent Team 拆分建议

本任务适合并行推进，但存在前置关系，应按“基础先行 + 业务域并行 + 最终收口”模式组织。

### 第一波：基础先行

#### Agent A：入口与构建清理
负责：

- `web/src/index.jsx`
- `web/vite.config.js`
- package 依赖移除准备

#### Agent B：helpers 与全局能力
负责：

- `web/src/helpers/utils.jsx`
- `web/src/helpers/render.jsx`
- `web/src/helpers/dashboard.jsx`

#### Agent C：通用组件层
负责：

- `web/src/components/common/ui/**/*`
- `web/src/components/common/modals/**/*`
- `web/src/components/common/markdown/**/*`

### 第二波：核心业务域并行

#### Agent D：channels
#### Agent E：models
#### Agent F：tokens

前提：`CardTable` 与通用替换方案已稳定。

### 第三波：扫尾并行

#### Agent G：layout/headerbar
#### Agent H：setting pages
#### Agent I：personal/setup/零散页

### 主控职责

- 统一处理跨模块冲突
- 组织最终搜索清零
- 执行 build/lint 验证
- 收尾删除依赖与死代码

## 9. 验证策略

### 9.1 静态验证

- 搜索目标目录内是否还有 Semi import
- 搜索 `semi.css`、`LocaleProvider`、`@douyinfe/semi-icons`
- 检查是否引入新的重复 UI 实现

### 9.2 构建验证

- 前端构建通过
- 前端 lint 通过（如果当前脚本可用）

### 9.3 页面验证

重点覆盖：

- 登录页
- channel 页面
- models 页面
- token 页面
- setting 页面
- personal 页面
- setup 页面
- 顶栏用户菜单 / 主题切换 / 通知入口
- 主要弹窗、删除确认、分页、表格、筛选

## 10. 完成标准

满足以下条件才视为迁移完成：

1. 前端代码中不再引用 `@douyinfe/semi-ui`
2. 前端代码中不再引用 `@douyinfe/semi-icons`
3. `web/src/index.jsx` 不再引入 `semi.css` 与 `LocaleProvider`
4. `web/vite.config.js` 不再依赖 Semi 插件
5. channels / models / tokens / setting / personal / setup 主流程页面可进入并完成核心操作
6. 构建通过，主要交互正常
7. 无明显 Semi 视觉残留或运行时依赖残留

## 11. 结论

这是一次 UI 基础设施级迁移，而不是简单的 import 替换。该仓库已经具备较成熟的 shadcn 风格基础，因此适合一次性完成彻底替换。最优路径是：先收口全局与基础层，再并行推进核心业务域，最后批量扫掉设置页与零散残留，并在最终阶段删除所有依赖与构建残留。
