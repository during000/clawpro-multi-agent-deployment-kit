# 复杂列表页 · AI Agent 安全 (admin/security-management)

> 类别：**复杂列表页（NumberCard + LineTabs + Toolbar + 子列表 + 行内 Alert）**
> 路由：`/admin/security-management`
> 源码：`client/src/pages/admin/Security/index.tsx`（1175 行）+ 子目录 `Security/AIAgent/*`
> 截图：`./admin-security-management.png`（1440×900）

## 1. 视觉骨架（自上而下）

```
┌──────────────────────────────────────────────────────────────────┐
│ AdminLayout · 顶部 Banner Alert                                   │
│ AdminPageHeader · 标题"AI Agent 安全"                             │
│   └ 副描述（多行）+ "功能使用说明 ⌄"（折叠展开）                    │
│ NumberCard Row（3 列等宽）                                        │
│   ├─ AI Agent 资产    [icon] 0 ←（选中态：蓝色描边 + 蓝底）       │
│   ├─ 存在风险/告警资产 [icon] 0 + InfoTooltip                     │
│   └─ 威胁告警        [icon] 0                                    │
│ LineTabs：AI Agent 资产 / 管控配置 / 审计日志 / 恶意 Skills / 威胁告警 │
│ SubSection                                                       │
│   ├─ Title "AI Agent 列表"  + 右侧工具栏：刷新 ⟳ · 批量开启防护 · 同步资产 │
│   └─ Content area                                                │
│       └─ EmptyState（暂无 AI Agent 资产 + 引导文案）              │
│ Bottom Bar Right                                                 │
│   ├─ 左：MockData Pill（黄点 + Mock 数据 (仅设计走查用) · 已关闭）  │
│   └─ 右：计费状态模拟  ⌃                                          │
└──────────────────────────────────────────────────────────────────┘
```

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/page-header.md` |
| 描述折叠 | `Tooltip` + 自实现"功能使用说明 ⌄" | `@/components/ui/tooltip` + `lucide ChevronDown/Up` | `component-specs/tooltip.md` |
| 数字卡片 | `NumberCard`（设计台 chart-stat 类） | `@/components/ui/*` 或 业务封装 | `component-specs/chart-stat.md` / `component-specs/number-card.md` |
| 信息提示 | `Info` icon + `Tooltip` / `InfoPopover` | `@/components/ui/info-popover`、`@/components/ui/tooltip` | `component-specs/tooltip.md` |
| 章节切换 | `LineTabs` | `@/components/ui/line-tabs` | `component-specs/tabs.md` |
| 子区域标题 | `<h3>` + 右侧 IconButton 组 | — | `component-specs/page-header.md` §SubSection |
| 工具栏图标按钮 | `Button variant="icon"` + lucide | `@/components/ui/button` | `component-specs/button.md` |
| 主操作 | `Button` 主蓝（同步资产）+ 次操作（批量开启防护） | 同上 | 同上 |
| 空态 | `Empty` 组件 | `@/components/ui/empty` | `component-specs/empty-state.md` |
| 表格（数据态） | `Table` 系列 | `@/components/ui/table` | `component-specs/table.md` |
| 弹层 | `Dialog` (BindErrModal/TrialModal/LockPage 等) | `@/components/ui/dialog` | `component-specs/dialog-drawer.md` |
| 反馈 | `Alert` / `toast`(sonner) | `@/components/ui/alert` / `sonner` | `component-specs/alert.md` / `toast.md` |
| 加载 | `Spinner` | `@/components/ui/spinner` | `component-specs/loading-progress.md` |

## 3. 关键 token / 规范要点

- **NumberCard 选中态**：左侧色条 + 整卡蓝色描边 + `bg-blue-50` 浅蓝底（见图第一张卡）
- 3 列 NumberCard 等宽 `grid-cols-3 gap-4`，单卡内布局：`[icon] 标题` 一行 + 大数字一行（24-30px 字重 600）
- LineTabs 紧贴 NumberCard 下方，**不要**再包一层 SurfaceCard
- 子区域 Header 是简单 `<h3 + 右侧 actions>`，不复用 PageHeader
- 工具栏右侧顺序：**辅助操作（icon 按钮）→ 次要按钮 → 主按钮**（参考图：刷新 ⟳ · 批量开启防护 · 同步资产）
- 主操作右侧不再有按钮（"同步资产"是收尾动作）
- EmptyState 文案模式：**第一行说现状（暂无 X）+ 第二行说怎么获取（点击同步...从 OpenClaw 主机识别）**
- Mock Pill 走 `--cp-bg-warning-soft` + 黄点指示，**仅在 design 走查环境出现**，生产/Tenant 隐藏

## 4. 为何典型 (why-typical)

- 唯一一个"统计 + 多 Tab + 子列表 + 工具栏 + 空态"全套组合的页面
- 演示了 NumberCard 的"选中状态"——其他页面的卡片大多只是展示态
- 演示了**多层级标题**（PageHeader 大标题 + 子区域 h3）的留白节奏
- 工具栏的"icon 按钮 → 次要 → 主要"三段式，是后续 dashboard / 资产页的复用范式
- 同时含 EmptyState（截图态）和 Table（mock 数据开启时），覆盖**有/无数据**两态

## 5. 易错点 / 反例

| ❌ 反例 | ✅ 正例 |
|---|---|
| NumberCard 选中态用 `border-2 border-blue-500` 直接画粗线 | 用规范的 `bg-blue-50` + `border --cp-brand` 浅描边 + 左色条 |
| LineTabs 与 NumberCard 之间垫 `<SurfaceCard>` 又加阴影 | 直接相邻，靠 `mt-6` 等留白节奏 |
| 子区域标题用 `AdminPageHeader` 复用 | 用 `<h3 className="font-medium">` + 右侧 actions，避免视觉双标题 |
| 工具栏顺序：`同步资产 · 批量开启 · ⟳` | 顺序应为 `⟳（辅助） · 批量开启（次） · 同步资产（主）` |
| EmptyState 仅一行"暂无数据" | 第一行业务文案 + 第二行操作引导 |
| 数字卡片把数字放上、标题放下 | 标题在上（含 icon）+ 数字在下，视觉层级"先理解后看值" |
| Mock 数据切换写在 PageHeader 右侧 | 固定底部 bar 右侧，不污染主内容区域 |
