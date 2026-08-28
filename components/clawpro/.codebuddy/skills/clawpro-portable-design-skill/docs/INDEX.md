# Index

## 1. 根目录文件

| File | Role |
|---|---|
| `README.md` | 面向人类的总入口 |
| `HANDOFF.md` | 面向交付动作的说明 |
| `DEVELOPER-USAGE.md` | 面向产品前端的换皮使用说明 |
| `PRODUCT-USAGE.md` | **产品同事零门槛入口**：做列表页 / 适配 Tenant / FAQ 等 |
| `QUICK-REFERENCE.md` | **产品快速查询表**：颜色 / 尺寸 / 文字 / 组件速查 + 问题决策树 |
| `COMMON-ERRORS.md` | **常见错误排查手册**：20 个最容易犯的错误 + 诊断 + 修复 |
| `DESIGN-AUDIT-PLAYBOOK.md` | 结合 impeccable 的统一设计审查与换皮验收流程 |
| `DELIVERY-CHECKLIST.md` | 周一交付前核对清单 |
| `SKILL.md` | 面向 AI 的总入口（包含技术栈、图表、动画、图标、反馈、Checklist 规范）|
| `STATUS.md` | 新对话续接状态板 |
| `STRUCTURE.md` | 目录结构解释 |
| `INDEX.md` | 文件索引 |
| `MANIFEST.json` | 交付包文件清单 |

## 2. `references/`

| File | Role |
|---|---|
| `foundation.md` | 全局设计基线 |
| `admin.md` | 管理端规则 |
| `tenant.md` | 客户端规则 |
| `landing.md` | 落地页方向规则 |
| `components.md` | 组件决策表 |
| `page-recipes.md` | 页面模板 |
| `migration-map.md` | 宿主仓迁移映射 |
| `assets-icons.md` | 资产与图标规则 |
| `conflict-log.md` | 冲突与待裁决项 |

## 3. `component-specs/`

| File | Role |
|---|---|
| `table.md` | 表格底座规范与 fallback |
| `data-table.md` | 数据驱动表格壳子（`Table` 之上）规范，新建列表页强制使用 |
| `card-surface.md` | 卡片层级与 Admin/Tenant 分流 |
| `button.md` | 按钮变体与 fallback |
| `empty-state.md` | 空状态层级与 fallback |
| `page-header.md` | 管理端页头 |
| `dialog-drawer.md` | 弹窗与抽屉 |
| `tabs.md` | LineTabs 下划线一级 Tab（page header 下方）规范与 fallback |
| `segment.md` | Admin 方角 / Tenant 胶囊 Segment（卡片内 / 工具栏切换）规范与 fallback |
| `form-controls.md` | 表单控件和筛选区 |
| `status-tag.md` | StatusTag 状态标签（运行 / 角色 / 信息态）规范与 fallback |
| `stepper.md` | Stepper 分步向导步骤条（24px 圆圈 + ChevronRight 分隔，状态由 current 推导）规范与 fallback |
| `badge.md` | Badge 轻量分类胶囊（版本 / 范围 / 类别 / new 角标）规范与 fallback |
| `pagination.md` | 分页器规范与 fallback |
| `search-filter-bar.md` | 列表搜索 / 筛选 / 刷新工具条 |
| `date-picker.md` | 日期选择器规范与 fallback |
| `datetime-display.md` | 卡片/列表中创建时间·最近活跃等 datetime 展示（自适应降级 + Tooltip）规范 |
| `combobox.md` | **Alias 文档**：原 Combobox 已并入 Select 的 `searchable` 模式（`SearchableSelect`），仅保留迁移指引 |
| `batch-actions-bar.md` | 表格多选 / 跨页选择 / 批量操作条规范与 fallback |
| `popover-dropdown-menu.md` | Popover / Dropdown / Tooltip 浮层规范与 fallback |
| `admin-sidebar.md` | Admin 端左侧导航壳（Provider / Header / Group / Menu / Footer / Inset）完整规范与 fallback |
| `tenant-topnav.md` | Tenant 端顶部导航（Logo / CenterTabs / 右侧操作 / Avatar）规范与 fallback |
| `loading-progress.md` | 加载、骨架屏、进度与操作反馈规范与 fallback |
| `chart-stat.md` | 图表、统计数字与 Dashboard 数据展示规范与 fallback |
| `number-card.md` | KPI 数字卡（图标 + 标题 + 大数字）规范与 fallback，替代手搓 `SurfaceCard + 内联 SVG + StatNumber` |
| `upload-file-browser.md` | 上传、文件浏览与文件列表规范与 fallback |
| `file-browser.md` | 多版本资产包只读浏览器（版本/文件树/内容三栏）规范与 fallback |
| `transfer.md` | 双面板穿梭框规范与 fallback（替代旧 `CvmSelectComponent`） |
| `tree-select.md` | TreeSelect 树形单选下拉（button / filter-icon 两变体）规范与 fallback；多选走 `transfer.md` |
| `input-select.md` | Input 输入框 + Select 下拉选择器规范与 fallback |
| `toast.md` | Toast 消息提示规范与 fallback |
| `selection-controls.md` | Switch / Checkbox / Radio 选择控件规范与 fallback |
| `tag-label.md` | Tag / Label 用户标签规范与 fallback |
| `breadcrumb.md` | 面包屑导航规范与 fallback |
| `tooltip.md` | Tooltip 工具提示规范与 fallback |
| `avatar.md` | 头像组件规范与 fallback |
| `tree.md` | 树结构 / 目录导航规范与 fallback |

## 4. `portable/`

### 4.1 `portable/html-css/`

- `table.html`
- `input-select-table.html`
- `data-table.html`
- `dialog-drawer.html`
- `empty-state.html`
- `card.html`
- `alert.html`
- `batch-actions-bar.html`
- `breadcrumb.html`
- `button.html`
- `chart-stat.html`
- `search-filter-bar.html`
- `date-picker.html`
- `admin-sidebar.html`
- `admin-control-page.html`
- `admin-list-page.html`
- `file-browser.html`

### 4.2 `portable/react/`

- `table.tsx`
- `empty-state.tsx`
- `card.tsx`
- `search-filter-bar.tsx`
- `date-picker.tsx`
- `button.tsx`
- `input-select.tsx`
- `dialog-drawer.tsx`
- `pagination.tsx`
- `number-card.tsx`
- `status-tag.tsx`
- `selection-controls.tsx`
- `tabs.tsx`
- `badges.tsx`
- `breadcrumb.tsx`（items 驱动，当前页不可点击 + 祖先页可点击 hover 加深）
- `tooltip.tsx`（hover/focus 触发，深黑底白字，纯 CSS 绝对定位四向）
- `avatar.tsx`（4 档尺寸 + 首字母 fallback + AvatarGroup 溢出 +N）
- `page-header.tsx`（title/description/actions/titleAccessory 三段插槽 + mb-6/mb-8）
- `loading-progress.tsx`（Spinner / Skeleton / Progress / TableSkeleton）
- `form-controls.tsx`（Label / FieldGroup / FieldRow / HelperText / FieldError 基元）
- `tree.tsx`（递归 Tree / 目录导航，depth 缩进 + a11y role=tree/treeitem + 键盘 Enter/Space）
- `popover-menu.tsx`（DropdownMenu + Popover，点击外部 / Esc 关闭，danger 项 + 分割线 + 空态）
- `transfer.tsx`（双面板穿梭框，instant/batch/oneWay 三模式 + 搜索 + simple 分页 + 4 种空态）
- `chart-stat.tsx`（ChartCard 外壳 + Legend + Tooltip + Delta + 加载/空/错误/无权限态，宿主仓自带图表库）
- `file-browser.tsx`（多版本三栏只读浏览器，版本侧栏 + 文件树 + Preview/Source 内容，复用 .fb__* 类名）
- `admin-sidebar.tsx`（Admin 左侧导航 21 个子组件 1:1 全量兜底，无 shadcn/cva/radix 依赖）
- `alert.tsx`（Alert 6 variant + 全部图标 + AdminNoticeAlert，内联 SVG，无 cva/lucide/Typography 依赖）

### 4.3 `portable/css/`

> 约定：每个 `portable/react/*.tsx` 都有一份同名 `.cp-*` 语义类 css（纯 CSS，零 Tailwind/shadcn 依赖），由 `portable.css` 按序聚合引入。

- `portable.css`（**总入口**：按 tokens → globals → 各组件 css 顺序 `@import`）
- `tokens.css`（`--cp-*` / `--radius` / `--alert-*` 全量设计 token）
- `globals.css`（基础 reset / page-enter 动画等）
- `button.css`（配套 `button.tsx`：6 variant × 3 size，4px 圆角）
- `badge.css`（配套 `badges.tsx`）
- `input.css`（配套 `input-select.tsx`：Input / Select 5 状态）
- `admin-sidebar.css`（配套 `admin-sidebar.tsx`：`--cp-admin-sidebar-*` token、active 渐变、SubGroup 引导线、scrollbar-on-hover 等）
- `alert.css`（配套 `alert.tsx`：6 variant 的 `--alert-*` token 映射、grid 图标列布局、AdminNoticeAlert 渐变标签）
- `segment.css`（配套 `segment.tsx`：Admin 方角 / Tenant 胶囊）
- `table.css`（配套 `table.tsx`：表头底色 / 12px 单元格）
- `card.css`（配套 `card.tsx`：SurfaceCard / Inner / Config + TenantCard 12px）
- `status-tag.css`（配套 `status-tag.tsx`：5 主语义色纯文本）
- `pagination.css`（配套 `pagination.tsx`：28/24 按钮 + 8px allow-radius 例外）
- `empty-state.css`（配套 `empty-state.tsx`：页面级虚线空态 + 表格内空态）
- `number-card.css`（配套 `number-card.tsx`：KPI 卡 + StatNumber DIN）
- `tabs.css`（配套 `tabs.tsx`：LineTabs 黑色 2px 下划线）
- `date-picker.css`（配套 `date-picker.tsx`：触发器 5 状态 + 日历网格）
- `search-filter-bar.css`（配套 `search-filter-bar.tsx`：搜索 + 筛选 + 操作三槽）
- `dialog-drawer.css`（配套 `dialog-drawer.tsx`：Dialog / AlertDialog / Drawer + Footer）
- `selection-controls.css`（配套 `selection-controls.tsx`：Switch / Checkbox / Radio，描边走 `--cp-border-control`）
- `breadcrumb.css`（配套 `breadcrumb.tsx`：14px 文字 + 6px 间距，无背景边框）
- `tooltip.css`（配套 `tooltip.tsx`：#020617 深黑底白字 + 240px 最大宽度 + 四向定位）
- `avatar.css`（配套 `avatar.tsx`：圆形 4 档尺寸 + fallback + AvatarGroup 叠放）
- `page-header.css`（配套 `page-header.tsx`：24px 标题 + 三段布局 + mb-6/mb-8）
- `loading-progress.css`（配套 `loading-progress.tsx`：Spinner/Skeleton/Progress/TableSkeleton + reduced-motion 降级）
- `form-controls.css`（配套 `form-controls.tsx`：Label/FieldGroup/FieldRow/HelperText/FieldError 基元）
- `tree.css`（配套 `tree.tsx`：32px 行 + 4px 圆角 + depth 缩进 + chevron rotate-90 + 弱色计数）
- `popover-menu.css`（配套 `popover-menu.tsx`：白底 4px overlay 阴影面板 + danger 项 + 分割线 + 空态）
- `transfer.css`（配套 `transfer.tsx`：等宽双面板 + 36px 灰头 + 40px 行 + instant X / batch move-btn + simple 分页）
- `chart-stat.css`（配套 `chart-stat.tsx`：`--cp-chart-*` 调色板 + ChartCard + Legend + Tooltip + 状态层）
- `file-browser.css`（配套 `file-browser.tsx`：三栏 14/22/flex-1 + 8px allow-radius 例外 + 版本/树/内容 `.fb__*`）
- `toast/`（配套 `react/toast/`）

## 5. `tokens/`

| File | Role |
|---|---|
| `design-tokens.json` | 最小 token JSON |
| `colors.md` | 颜色口径 |
| `typography.md` | 字体与语义文字层 |
| `radius-shadow.md` | 圆角与阴影 |
| `spacing.md` | 间距与布局节奏 |

## 6. `qa/`

| File | Role |
|---|---|
| `admin-checklist.md` | 管理端验收 |
| `tenant-checklist.md` | 客户端验收 |
| `landing-checklist.md` | 落地页验收 |
| `component-review-checklist.md` | 组件 spec 自查 |

## 7. `assets/` 与 `scripts/`

| File | Role |
|---|---|
| `assets/icon-registry.example.json` | 资产清单示例 |
| `assets/page-references/README.md` | 典型页面参考样本入口（7 类） |
| `assets/page-references/admin-channel-config.{md,png}` | 配置页样本（Tabs + 行内开关 Table） |
| `assets/page-references/admin-members.{md,png}` | 列表页样本（基础版：Segment + Search + Table） |
| `assets/page-references/admin-openclaw-monitor.{md,png}` | 列表页样本（富功能版：状态卡 + 列头筛选 + 行操作 + Drawer） |
| `assets/page-references/admin-agent-template.{md,png}` | 整页 EmptyState 样本 |
| `assets/page-references/admin-security-management.{md,png}` | 复杂列表样本（NumberCard + LineTabs + 子表 + Empty） |
| `assets/page-references/admin-tokens-monitor.{md,png}` | 数据看板样本（NumberCard + LineChart + LineTabs + Table） |
| `assets/page-references/admin-ops-observation.{md,png}` | 服务开通引导样本（CTA banner + FeatureGrid） |
| `scripts/check-design-usage.mjs` | 设计使用检查脚本 |
| `scripts/check-spec-symbols.mjs` | 扫 spec 内 import 是否在 client/src/ 真实导出，揪 ghost identifier（如旧 Combobox / OpenClawCombobox） |
| `scripts/sync-tokens.mjs` | 从 client/src/index.css 同步 token 定义到 tokens/design-tokens.json，并检查 spec markdown 里的 token 引用是否都有效 |
| `scripts/package-portable-skill.sh` | 打包 zip |
| `scripts/verify-portable-skill.mjs` | 校验交付包是否完整 |


