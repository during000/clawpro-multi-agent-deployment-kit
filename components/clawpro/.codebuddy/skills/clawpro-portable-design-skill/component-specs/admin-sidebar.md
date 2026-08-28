# Admin Sidebar

> Admin 端左侧导航壳（Provider / Sidebar / Header / Content / Group / Menu / Footer / Inset / Trigger）的完整视觉与行为规范。Tenant TopNav 见 `tenant-topnav.md`，跨端面包屑见 `breadcrumb.md`，本 spec 只聚焦 Admin。

## 1. Purpose

- 统一 Admin 端侧栏视觉、宽度、节奏、active / hover 状态。
- 让宿主仓在没有 demo 仓完整组件的前提下，也能 1:1 还原 ClawPro Admin 导航。
- 收口"展开/收起"两套形态在 Header / Group / Footer / SubGroup 上的差异，避免每个页面自行拼装。
- 锁定 Provider 行为：localStorage 偏好持久化 + 视口 ≤1200px 自动收起。

### 1.1 真相源优先级（必读！生成前强制遵循）

> **本 spec 的文本描述是"设计意图 + 约束"层，不是像素级 DOM 生成指令。**
> 做 1:1 还原时，**必须以实际组件代码为准**，spec 文本仅用于验证"有没有遗漏规则"。

当 spec 文本与实际资产（组件源码 / demo DOM / asset 文件）冲突时，可信度排序：

| 优先级 | 来源 | 用途 | 示例 |
|---|---|---|---|
| **P0** | `admin-sidebar.tsx` 组件源码 | DOM 结构、className、SVG 形状、组件嵌套关系、数据映射 | `SidebarCollapseIcon` 的 SVG path、`AdminSidebarHeader` 的内部 div 层数、`adminNav.ts` 的 icon→item 映射 |
| **P1** | demo 仓实际渲染 DOM（DevTools copy） | 像素级验证：实际 CSS 值（tracking、margin、宽高）、flex 方向、兄弟/父子关系 | "前往用户端"在 header 顶部行内还是行外、副标题 `tracking` 实际值 |
| **P2** | `admin-sidebar-style.css` | Token 值、data-slot 选择器作用域、hover/active 状态的具体 background | `--cp-admin-sidebar-item-hover` 的 RGBA 值、SubGroup 引导线的 `::before/::after` |
| **P3** | 本 spec 文本 | 设计意图、约束边界、Do/Don't、行为规则（折叠持久化、视口阈值） | "HeaderAction hover 不动 bg/border"、"子项不显示 icon"、"折叠按钮必须挂 Tooltip" |
| **✗ 禁止** | 凭 spec 文本猜测 DOM 结构或 SVG 形状 | — | 猜 `SidebarCollapseIcon` 是简单 chevron、猜 Header 是单层 div |

**硬规则**：
- 生成任何 Admin Sidebar 页面 / 组件时，**先读 `admin-sidebar.tsx` 找到对应组件的 render 代码，再读 demo DOM 验证**，最后才参考 spec 文本补约束。
- 组件里已经写死的 className、SVG、嵌套层级 → **直接照抄**，不要凭 spec 文本"优化"或"简化"。
- spec 文本里没有但组件代码里有的细节 → **以组件代码为准，反过来更新 spec**（不要删组件代码里的东西去凑 spec）。
- 图标文件（`.svg`）→ 使用 `adminNav.ts` 里的 `icon` 字段映射，**禁止凭菜单项名称猜测图标**。

### 1.2 高频翻车项 & 防呆规则（Header 特化）

> Header 是 Admin Sidebar 中**出错率最高的区域**。以下均为真实踩坑后固化，生成时必须逐条自检。

#### 翻车清单 & 正确做法

| 翻车项 | ❌ 错误做法 | ✅ TSX 源码真相 |
|---|---|---|
| 折叠按钮图标 | 看到 `SidebarCollapseIcon` 名字，自己画 chevron | `admin-sidebar.tsx:195-201` 写死了三横线 + 箭头 SVG（20×20 viewBox, 16px 渲染），**逐字节照抄** |
| Header 布局 | 凭 spec 文字猜"前往用户端"在顶部行内还是行外 | TSX `AdminSidebarHeader` 是 `flex-col`，children = `Brand` + `HeaderAction`，二者是**兄弟**，demo 页给 Brand 加了顶部行 wrapper |
| Logo 尺寸 | 展开态写 `width="36" height="28"` 裸 SVG | demo：`size-11`(44px) 容器 + 内部 logo `w-8`(32px) 缩放；收起态展开按钮：logo 原生 36×28，**不做缩放** |
| 图标映射 | 凭菜单项名称猜测图标文件 | `adminNav.ts` 的 `icon` 字段是唯一映射源，如"网盘管理"→`file-management.svg`，**禁止联想** |
| 图标资产 | 手画 `<svg>` 内联图形 | 使用 `<img src="assets/admin-sidebar/{icon}.svg">` 引用真实文件，保持 `alt="" aria-hidden="true"` |
| 折叠态双按钮 | 加了 `.cp-expand-btn` 但忘了关 `.cp-header-top` | 展开/收起是**互斥状态**，每个 visibility 都必须对照 TSX 的 `collapsed ? A : B` 写成互斥 CSS |
| 折叠态用户端 icon | 把 32×32 按钮塞文字+箭号，靠 `overflow:hidden` 裁切 | 折叠态应显示独立 16px → 箭头 SVG（`cp-tenant-icon-collapsed`），**不能靠溢出裁切凑** |
| 折叠态 Header padding | 凭感觉写 `12px 8px 0` | TSX: `py-3 px-2`=`12px 8px`（**含底 padding**） |
| 折叠态 Content padding | 手写 `12px` | TSX `AdminSidebarContent` 永远 `px-4`=16px，**不因折叠改变** |
| 副标题 tracking | 跟 spec 写 `0.18px` | demo 实际渲染 `0.06px`，以 DOM 实测为准 |

#### 硬规则（Header 专项）

1. **Header 生成前，先逐行读 `admin-sidebar.tsx` 中 `AdminSidebarHeader` / `AdminSidebarBrand` / `AdminSidebarHeaderAction` / `SidebarCollapseIcon` / `AdminSidebarLogo` 五个组件的完整 render 代码。**
2. 任何涉及"展开/收起"双态的模块（不仅是 Header，Footer、Menu、SubGroup 同理），**必须列出 visibility 矩阵**后再写 CSS：

```text
CSS selector / element        | expanded | collapsed
───────────────────────────────────────────────────
.cp-header-top                | ✓ flex   | ✗ none
.cp-expand-btn                | ✗ none   | ✓ flex
.cp-tenant-text               | ✓ inline | ✗ none
.cp-tenant-icon-collapsed     | ✗ none   | ✓ block
.cp-arrow-wrap                | ✓ (hover)| ✗ none
[data-slot="admin-sidebar-footer"] | ✓ | ✗ none
[data-slot="admin-sidebar-group-label"] | ✓ | ✗ none
```

**矩阵中每个 `✓` 和 `✗` 必须能追溯到 TSX 源码中的 `collapsed ? ... : ...` 或 className 分支。** 不允许凭"感觉应该显示/隐藏"填入。

3. 图标永远走 `<img src="path/to/asset.svg">` + `adminNav.ts` 映射，**禁止内联手画。**

## 2. Scope

- 适用端：**仅 Admin**。
- 必用场景：所有 Admin 页面骨架（基础信息 / Agent 配置 / 安全策略 / 实例管理 / 审计…）。
- 不适用场景：Tenant 业务页（用 TopNav）、Landing 营销页（用营销导航）、设计系统组件展示页内的 *局部* 演示（在容器内通过 `[--admin-sidebar-*]` token override 即可）。
- 禁止：
  - Admin 页面外层套 `TenantTopNav`。
  - Tenant / Landing 页面外层套 `AdminSidebar`。
  - 在页面内重复实现侧栏（应统一走 `AdminLayout`）。

### 2.1 前置集成清单（必读！）

**在使用 AdminLayout 之前，必须确保以下 3 项准备就绪：**

| 检查项 | 用途 | 失效后果 |
|---|---|---|
| **导入 CSS 变量文件** | `import "@/index.css"` | 所有样式不生效，导航背景变灰，间距错乱 |
| **AdminLayout 包含 `admin-theme` 类** | 应用 token 主题 | 页面背景不正确，导航样式无法识别 token |
| **CSS 变量完整集合已定义** | 所有 `--admin-sidebar-*` 和 `--page-bg` 存在 | 特定样式缺失或使用浏览器默认值 |

**快速检查**（浏览器 DevTools）：
```javascript
// 打开 Console，粘贴下面代码
const adminSidebarBg = getComputedStyle(document.documentElement).getPropertyValue('--admin-sidebar-bg');
console.log('CSS 变量已加载:', adminSidebarBg !== ''); // 应该输出 true
```

## 3. Visual Standard

### 3.1 整体框架

| Item | Token / Value | Notes |
|---|---|---|
| 展开宽度 | `--admin-sidebar-width: 240px` | 已确认口径 |
| 收起宽度 | `--admin-sidebar-width-collapsed: 64px` | 图标-only 状态 |
| Header 高度 (展开) | `--admin-sidebar-header-height: 104px` | 含 Brand + 前往用户端按钮 |
| Footer 高度 | `--admin-sidebar-footer-height: 72px` | 顶部带 1px 分割线 |
| Background | `--admin-sidebar-bg: #ffffff` | 纯白底，**不加阴影** |
| Border | `--admin-sidebar-border: #EAEEF4` | 仅右侧 1px |
| 前景色 | `--admin-sidebar-foreground: #0a0a0a` | 主文字 / 选中文字 |
| 辅助色 | `--admin-sidebar-muted: #737373` | Group label / 收起态图标 |
| Z-index | `z-40` | 固定定位 |
| 折叠动画 | `transition-[width] duration-300` | width 同步 inset margin-left |

### 3.2 菜单项 (Menu Item)

| Item | Token / Value | Notes |
|---|---|---|
| 高度 | `--admin-sidebar-item-height: 34px` | **不要 44/48** |
| 圆角 | `--admin-sidebar-item-radius: 4px` | **不要胶囊** |
| 字号 | `text-sm` (14px) `leading-[22px]` `tracking-[0.005em]` | |
| 内边距 | `px-2` | 8px 左右 |
| 图标 | 16px (`size-4`)，`gap-2` 与文本 | 图标在前 |
| Hover bg | `--admin-sidebar-item-hover-bg: rgba(180, 191, 225, 0.14)` | 弱化 |
| Active bg | `--admin-sidebar-item-active-bg: linear-gradient(90deg, #e9f3ff 0%, #e3eaff 100%)` | **弱蓝渐变** |
| Active 图标 | `filter: brightness(0) saturate(100%)` | 强制纯黑覆盖图标自带渐变 |
| Active 文字 | `--admin-sidebar-foreground` | 不加粗、不变色条 |
| Focus ring | `ring-2 ring-[var(--brand-blue)]` | 仅键盘聚焦时 |

### 3.3 Group / SubGroup

| Item | Token / Value | Notes |
|---|---|---|
| Group label 字号 | `text-xs` `leading-[1.5]` `tracking-[0.015em]` | `--admin-sidebar-muted` |
| Group label 高度 | `AdminSidebarGroupLabel`: `h-5` `mb-2`；`AdminSidebarGroupTrigger`: `h-5` `mb-1` | 纯标签下间距 8px，可折叠触发器下间距 4px，**勿混用** |
| Group 间距 | `mb-5 last:mb-0` | 分组间留白 |
| Group Trigger Chevron | 12×12，open 时正向，closed 时 `rotate-180` | `--text-muted` |
| 子项缩进 | `padding-left: 32px` | 通过 `.admin-sidebar-subitem-button` 实现 |
| 子项图标 | `none` | **展开态二级子项不显示 icon**，仅保留文本 + badge + 引导线 |
| 子项左侧引导线 | `1px` `--admin-sidebar-border` (`::before`) | 16px 处垂直贯穿 |
| 子项 hover/active 标记 | `1px×18px` `--admin-sidebar-foreground` (`::after`) | 仅 hover/active 时 opacity:1 |

### 3.4 Header

| 形态 | 内容 | 关键样式 |
|---|---|---|
| 展开 | `Brand`（Logo + "管控端" + "ClawPro Admin"）+ 折叠按钮 | `flex h-[72px] items-start justify-between px-4 pt-4`，Logo 套 44×44 白底 + `--admin-sidebar-logo-shadow`；主标题 `text-base`(16px) `font-medium` `leading-[22px]` `tracking-[0.08px]`（**不是 14px**），副标题 `text-xs`(12px) `leading-5` `tracking-[0.18px]` `--admin-sidebar-muted` |
| 展开 | "前往用户端" 行动按钮 | `mx-4 h-8 w-[208px]`，**hover 时 bg / border 不变**，反馈仅靠右侧 → arrow 从 0px 滑入到 14px 宽 + opacity 0→1 |
| 展开 | 折叠按钮（`SidebarCollapseIcon`） | `mt-3 size-5`，`hover:text-[var(--text-brand)]`，**必须挂 `Tooltip`**（"收起导航"，`side="right" sideOffset={8}`，`delayDuration={0}`） |
| 收起 | 展开按钮（Logo ↔ 三横线 hover 切换）+ 跳转用户端图标按钮 | **展开按钮**：`h-10 w-9`(40×36)，默认显示 **Logo 原生尺寸 36×28**（与 SVG `width="36" height="28"` 一致，**不再压成 size-8**），hover/focus 时 Logo 淡出、三横线图标（**20×20**）淡入并转 `--text-brand`；**跳转用户端图标按钮**：仍 32×32 `size-8 rounded-[4px]` 内含 16px 图标；**两个按钮各自挂 Tooltip**（"展开导航" / "前往用户端"） |

### 3.5 Footer

| 形态 | 内容 | 关键样式 |
|---|---|---|
| 展开 | `AdminSidebarUser`（圆形 32px 头像 + 名字 + 角色）+ 更多操作 (`MoreHorizontal`) | `h-[72px] px-4`，顶部 1px 分割线 (`::before`) |
| 收起 | **整个 Footer 隐藏**（不展示头像 / 用户名 / 角色 / 更多操作） | `display:none`；用户信息与退出登录等操作改由展开态访问，**不再以单头像 + HoverCard 呈现** |
| 头像底色 | `--admin-sidebar-avatar-bg: color-mix(srgb, var(--brand-blue) 32%, #fff)` | 弱蓝 |
| 头像文字 | `--admin-sidebar-avatar-foreground: #020617` `font-mono` `text-[14.22px]` | |

### 3.6 Badge

| Variant | Background | Foreground |
|---|---|---|
| `new` | `--admin-sidebar-badge-brand-bg`（10% brand-blue mix） | `--brand-blue` |
| `coming-soon` | `--admin-sidebar-badge-bg: #f5f5f5` | `--admin-sidebar-muted` |
| `custom` (字符串) | 默认同 `coming-soon`；菜单项 active 时切换为 brand 配色 | 同上 |
| 形态 | `h-[18px]` `rounded-full` `px-1.5` `text-[11px]` `ml-auto` | 仅展开态显示 |

### 3.7 Action Button (Header / Footer)

| Item | Token | Notes |
|---|---|---|
| 尺寸 | `size-8` | 32×32 |
| 圆角 | `rounded-[4px]` | |
| 描边 | `--admin-sidebar-action-border` (= `--admin-sidebar-border`) | |
| 背景 | `--admin-sidebar-action-bg: #fff` | |
| **Hover 行为** | **bg / border 完全不变**；仅 `transition-[color,box-shadow]` 锁色 | 反馈来源：展开态 → arrow 滑入动效（group-hover 控制 `width 0→14px` + `opacity 0→1`）；收起态 Tooltip（"前往用户端" / "展开导航"） |
| 图标 | `size-4` | 16px |

> **历史 token**：`--admin-sidebar-action-hover-bg` / `--admin-sidebar-action-hover-border` 早期版本曾用于 hover 反馈，现已**全线断开引用**——组件 className（`AdminSidebarHeaderAction` 源码移除 `hover:bg-* hover:border-*`）、装配层（`AdminLayout.tsx` 收起态 `<a>`）、`client/src/index.css` `@layer components` 原 `[data-slot="admin-sidebar-header-action"]:hover` 兜底覆写规则均已删除。token 本身保留以兼容外部消费方的 CSS override，但 ClawPro 内部新代码**严禁**再去引用它们（裁决见 `references/conflict-log.md` C-015 / C-017）。

## 4. Anatomy

```text
AdminSidebarProvider                ← 控制 collapsed 状态 + localStorage
  AdminSidebar (aside, fixed left)  ← 240/64 + 白底 + 右边线
    AdminSidebarHeader               ← 展开 104px / 收起 padded
      AdminSidebarBrand              ← Logo + 标题 + 副标题
        AdminSidebarLogo
      AdminSidebarHeaderAction       ← "前往用户端" / 图标按钮
    AdminSidebarContent (nav)        ← 滚动区，scrollbar-on-hover
      AdminSidebarGroup              ← 可折叠
        AdminSidebarGroupTrigger     ← 12px chevron
        AdminSidebarGroupContent
          AdminSidebarMenu (ul)
            AdminSidebarMenuItem (li)
              AdminSidebarMenuButton (a/button)
                <icon> <label> <AdminSidebarBadge?>
            // 子分组（可嵌套）
            SubGroupBlock
              MenuButton (toggle)
              MenuItem * N (展开时缩进 32px)
    AdminSidebarFooter               ← 72px + 顶部 1px 分割线
      AdminSidebarUser               ← 32px 头像 + name/role
      AdminSidebarFooterAction       ← MoreHorizontal → DropdownMenu
  AdminSidebarInset (main)           ← marginLeft 与 sidebar 宽度联动
    [page content]
```

## 5. States

| 状态 | 触发 | 行为 |
|---|---|---|
| `expanded` | 默认 / `localStorage.admin_sidebar_collapsed === "false"` / 视口 >1200px 首次进入 | width 240，Header/Footer/Group 完整文字 |
| `collapsed` | 用户点击 trigger / 视口 ≤1200px / `localStorage` 偏好 | width 64，仅图标，hover 显示 Tooltip / HoverCard |
| `hover` (item) | 鼠标悬停菜单项 | 弱化背景 `rgba(180,191,225,0.14)` |
| `hover` (header action) | 鼠标悬停"前往用户端" / 图标按钮 | **bg / border 不变**；展开态靠右侧 → arrow 滑入（width 0→14px + opacity 0→1）；收起态靠 Tooltip 文本 |
| `active` (item) | `data-active="true"` / `aria-current="page"` | 弱蓝渐变 + 图标转纯黑 |
| `focus-visible` | 键盘 Tab | `ring-2 ring-[var(--brand-blue)]` |
| `group-open` / `group-closed` | 用户点击 GroupTrigger | chevron 旋转，content 显示/隐藏 |
| `subgroup-open` / `subgroup-closed`（展开态） | 用户点击 SubGroupBlock | 子项展开/收起 |
| `subgroup-collapsed-active`（收起态） | 任一子项 active | HoverCard trigger 上出现 active 弱蓝渐变 |
| `notification` | Badge `variant="new"` 或自定义 | 弱品牌色背景；收起态隐藏 |
| `unavailable` | 当前未实现 | Badge `variant="coming-soon"` |
| `auto-collapse` | window 跨过 1200px 阈值 | mql change → 自动同步 collapsed 状态并写回 localStorage |

## 6. Interaction & Behavior

- **持久化**：`AdminSidebarProvider` 使用 `localStorage["admin_sidebar_collapsed"]` 缓存用户偏好。组件挂载时优先读 localStorage，其次按视口推断。
- **视口阈值**：`window.matchMedia("(max-width: 1200px)")` change 事件 → 自动切换 collapsed，并写回偏好。**不要在初始化时强制覆盖 localStorage**。
- **Trigger**：点击 `AdminSidebarTrigger` / Header 折叠按钮 → `toggleSidebar()`，立即生效。
- **Inset 联动**：`AdminSidebarInset` 通过 `marginLeft: var(--admin-sidebar-width|width-collapsed)` 同步偏移，`transition: margin-left 300ms`。
- **滚动条**：`AdminSidebarContent` 仅在滚动/wheel 时显示滚动条 700ms（`data-scrolling` 切换）；CSS 钩子 `scrollbar-on-hover`。
- **收起态 Tooltip / HoverCard**：
  - 普通菜单项：右侧 `Tooltip`，`sideOffset: 8`。
  - SubGroup：右侧 `HoverCard`，`min-w-[140px]`，含 SubGroup 标签 + 完整子菜单。
  - Footer：**收起态整个隐藏**（不再以单头像 + HoverCard 形式呈现）；用户信息 / 模式切换 / 退出登录仅在展开态可见。
  - **收起态** Header 折叠按钮（"展开导航"）+ 跳转用户端图标按钮（"前往用户端"）：均挂 `Tooltip`，因为收起态没有文字。
- **展开态 Header 折叠按钮：必须挂 `Tooltip`「收起导航」**（`side="right" sideOffset={8}`、`delayDuration={0}`）。设计标准要求 hover 时给出明确文本反馈，仅靠 `aria-label` + 颜色变化不足以让用户知道点击后会"收起"还是"展开"。早期版本曾顾虑黑色浮层从 sidebar 弹出去会越界遮挡告警条 / 面包屑，但实测 `side="right"` 浮层会朝右弹到 sidebar 外侧空白 / `Inset` 上方，不影响内容区可读性。
- **Active 路由匹配**：`location === href || location.startsWith(href + "/")`；额外通过 `ADMIN_ROUTE_ALIASES` 支持别名（demo 仓样例：`/admin/agent-types` ↔ `/admin/image-management`）。
- **Badge 显示**：仅展开态展示，收起态隐藏（避免 64px 内拥挤）。
- **图标 Active 处理**：选中项 `<img>` 通过 `filter: brightness(0) saturate(100%)` 强制黑色，覆盖 SVG 自带渐变。

## 7. Accessibility

- `aside` 元素，外部包 `aria-label="管理后台导航"` 在 `AdminSidebarContent`（nav）。
- 当前页 MenuButton：`data-active="true"` + `aria-current="page"`。
- 折叠按钮：`aria-label="展开导航" / "收起导航"`。
- SubGroup 触发按钮：`aria-expanded={open}`，`aria-label` 含"展开/收起 + 分组名"。
- 所有交互元素 `focus-visible:ring-2 ring-[var(--brand-blue)]`。
- 收起态下文本仅在 Tooltip 中可见，确保 Tooltip 内容 = MenuItem 文本。

## 8. Demo Repo Usage

- 组件实现：`client/src/components/ui/admin-sidebar.tsx`
- 装配示例：`client/src/components/AdminLayout.tsx`
- 导航数据源：`client/src/config/adminNav.ts`（`ADMIN_NAV_GROUPS`）
- 全局样式（active / hover / subgroup 引导线）：`client/src/index.css` 第 290–500 行
- 设计系统展示：`client/src/pages/DesignSystemComponents.tsx`（搜 "AdminSidebar"）

最小用法：

```tsx
import {
  AdminSidebar, AdminSidebarProvider, AdminSidebarHeader, AdminSidebarBrand,
  AdminSidebarLogo, AdminSidebarContent, AdminSidebarGroup, AdminSidebarGroupTrigger,
  AdminSidebarGroupContent, AdminSidebarMenu, AdminSidebarMenuItem, AdminSidebarMenuButton,
  AdminSidebarBadge, AdminSidebarFooter, AdminSidebarUser, AdminSidebarInset,
  AdminSidebarTrigger,
} from "@/components/ui/admin-sidebar";

<AdminSidebarProvider>
  <div className="flex h-screen overflow-hidden admin-theme">
    <AdminSidebar>
      <AdminSidebarHeader>
        <AdminSidebarBrand asChild>
          <a href="/"><AdminSidebarLogo /><span>管控端</span></a>
        </AdminSidebarBrand>
      </AdminSidebarHeader>

      <AdminSidebarContent aria-label="管理后台导航">
        <AdminSidebarGroup defaultOpen>
          <AdminSidebarGroupTrigger>基础信息</AdminSidebarGroupTrigger>
          <AdminSidebarGroupContent>
            <AdminSidebarMenu>
              <AdminSidebarMenuItem>
                <AdminSidebarMenuButton asChild isActive>
                  <a href="/admin/basic-info">
                    <img src="/icons/basic.svg" alt="" />
                    <span>基础信息配置</span>
                  </a>
                </AdminSidebarMenuButton>
              </AdminSidebarMenuItem>
              <AdminSidebarMenuItem>
                <AdminSidebarMenuButton asChild>
                  <a href="/admin/skill-config">
                    <img src="/icons/skill.svg" alt="" />
                    <span>技能配置</span>
                    <AdminSidebarBadge variant="new" />
                  </a>
                </AdminSidebarMenuButton>
              </AdminSidebarMenuItem>
            </AdminSidebarMenu>
          </AdminSidebarGroupContent>
        </AdminSidebarGroup>
      </AdminSidebarContent>

      <AdminSidebarFooter>
        <AdminSidebarUser name="jingsujiang" role="管理员" />
        <AdminSidebarTrigger />
      </AdminSidebarFooter>
    </AdminSidebar>

    <AdminSidebarInset>{children}</AdminSidebarInset>
  </div>
</AdminSidebarProvider>
```

## 9. Portable Fallback

> **首选：完整可移植实现（1:1 还原）。** 宿主仓没有 `admin-sidebar.tsx` 时，**直接复制** `portable/react/admin-sidebar.tsx` + `portable/css/admin-sidebar.css`，即可拿到与 demo 仓完全一致的 21 个导出：`AdminSidebarProvider` / `AdminSidebar` / `AdminSidebarHeader` / `AdminSidebarLogo` / `AdminSidebarBrand` / `AdminSidebarHeaderAction` / `AdminSidebarContent`（scrollbar-on-hover）/ `AdminSidebarGroup`（可折叠）/ `AdminSidebarGroupTrigger` / `AdminSidebarGroupLabel` / `AdminSidebarGroupContent` / `AdminSidebarMenu` / `AdminSidebarMenuItem` / `AdminSidebarMenuButton`（三态）/ `AdminSidebarBadge`（new / coming-soon / custom）/ `AdminSidebarFooter` / `AdminSidebarUser` / `AdminSidebarFooterAction` / `AdminSidebarInset` / `AdminSidebarTrigger` / `SidebarCollapseIcon`（另导出 `useAdminSidebar` hook）。SubGroup 二级缩进 + 1px 引导线不是独立组件，而是给子项 `AdminSidebarMenuButton` 加 `className="cp-admin-sidebar-subitem-button"`（样式在配套 css 内）。该文件不依赖 shadcn / cva / @radix-ui/react-slot / @/lib/utils，仅需 React + Tailwind。
>
> 下面的 §9.2 / §9.3 是**极简降级版**（无 Logo / 无「前往用户端」/ 无分组折叠 / 无 SubGroup 引导线 / 无激活态图标变黑），仅在无法引入完整文件时临时兜底，**不要**当成默认方案——它与线上视觉差距较大。

### 9.1 If host repo already has a sidebar shell

只对齐这些口径即可，不要替换宿主仓的路由 / 权限 / 图标体系：

- 容器宽度 240 / 64，背景纯白，右侧 1px `#EAEEF4`，**不要阴影**。
- Header 104px（展开）；Footer 72px，顶部 1px 分割线。
- 菜单项 34px / 圆角 4px / 字号 14px。
- Active：弱蓝渐变 `linear-gradient(90deg, #e9f3ff 0%, #e3eaff 100%)`，**禁止实心蓝**、**禁止左侧色条**。
- Hover：`rgba(180, 191, 225, 0.14)`。
- 收起态：仅图标 + Tooltip / HoverCard。
- Group label：12px / `#737373` / 20px 高度（`h-5`）/ 下间距 8px（`mb-2`），**不抢菜单项**。
- Badge：圆胶囊 18px / 11px 字号 / brand 弱底（new）或灰底（coming-soon / 自定义）。
- 展开态二级子项（SubGroup children）**不显示 icon**，只保留 32px 缩进、左侧 1px 引导线、文本与 badge。
- 视口 ≤1200px 自动收起，并把用户偏好写到 `localStorage`。

### 9.2 Minimal React fallback

```tsx
import { useEffect, useState, createContext, useContext } from "react";

const SidebarCtx = createContext<{ collapsed: boolean; toggle: () => void } | null>(null);

export function PortableAdminShell({ children, nav }: {
  children: React.ReactNode;
  nav: { label: string; items: { href: string; label: string; icon?: string; active?: boolean; badge?: "new" | "coming-soon" | string }[] }[];
}) {
  const [collapsed, setCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    const v = window.localStorage.getItem("admin_sidebar_collapsed");
    if (v === "true") return true;
    if (v === "false") return false;
    return window.matchMedia("(max-width: 1200px)").matches;
  });

  useEffect(() => {
    const mql = window.matchMedia("(max-width: 1200px)");
    const onChange = (e: MediaQueryListEvent) => {
      setCollapsed(e.matches);
      window.localStorage.setItem("admin_sidebar_collapsed", String(e.matches));
    };
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, []);

  const toggle = () => {
    setCollapsed(c => {
      const next = !c;
      window.localStorage.setItem("admin_sidebar_collapsed", String(next));
      return next;
    });
  };

  const w = collapsed ? "var(--cp-admin-sidebar-w-collapsed,64px)" : "var(--cp-admin-sidebar-w,240px)";

  return (
    <SidebarCtx.Provider value={{ collapsed, toggle }}>
      <div className="flex h-screen overflow-hidden bg-white">
        <aside style={{ width: w }} className="fixed inset-y-0 left-0 z-40 flex flex-col overflow-hidden border-r border-[var(--cp-admin-sidebar-border,#EAEEF4)] bg-[var(--cp-admin-sidebar-bg,#fff)] transition-[width] duration-300">
          <header className={collapsed ? "flex flex-col items-center px-2 py-3" : "h-[104px] px-4 pt-4 flex items-start justify-between"}>
            <a href="/" className="flex items-center gap-2.5 rounded-[4px]">
              <span className="size-11 grid place-items-center rounded-[8px] bg-white shadow-[0_1px_4px_rgba(176,182,195,0.3)]">
                {/* place ClawPro logo here */}
              </span>
              {!collapsed && (
                <span className="flex flex-col">
                  <span className="text-base font-medium leading-[22px] tracking-[0.08px] text-[#0a0a0a]">管控端</span>
                  <span className="text-xs leading-5 tracking-[0.18px] text-[#737373]">ClawPro Admin</span>
                </span>
              )}
            </a>
            {!collapsed && (
              <button onClick={toggle} aria-label="收起导航" className="mt-3 size-5 grid place-items-center rounded-[4px] text-[#0a0a0a] hover:text-[var(--brand-blue,#1447E6)]">≡</button>
            )}
          </header>

          <nav aria-label="管理后台导航" className="min-h-0 flex-1 overflow-y-auto px-4 mt-4 mb-3">
            {nav.map(group => (
              <div key={group.label} className="mb-5 last:mb-0">
                {!collapsed && (
                  <div className="mb-2 px-2 h-5 text-xs leading-[1.5] tracking-[0.015em] text-[#737373]">{group.label}</div>
                )}
                <ul className="flex w-full flex-col gap-0.5">
                  {group.items.map(item => (
                    <li key={item.href}>
                      <a
                        href={item.href}
                        data-active={item.active || undefined}
                        aria-current={item.active ? "page" : undefined}
                        className="flex h-[34px] w-full items-center gap-2 rounded-[4px] px-2 text-sm leading-[22px] text-[#0a0a0a] hover:bg-[rgba(180,191,225,0.14)] data-[active=true]:bg-[linear-gradient(90deg,#e9f3ff_0%,#e3eaff_100%)]"
                      >
                        {item.icon && <img src={item.icon} alt="" className="size-4 shrink-0" />}
                        {!collapsed && <span className="min-w-0 flex-1 truncate">{item.label}</span>}
                        {!collapsed && item.badge && (
                          <span className="ml-auto inline-flex h-[18px] items-center rounded-full px-1.5 text-[11px]"
                            style={
                              item.badge === "new"
                                ? { background: "color-mix(in srgb, var(--brand-blue,#1447E6) 10%, #fff)", color: "var(--brand-blue,#1447E6)" }
                                : { background: "#f5f5f5", color: "#737373" }
                            }
                          >
                            {item.badge === "new" ? "New" : item.badge === "coming-soon" ? "即将开放" : item.badge}
                          </span>
                        )}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>

          <div className="relative flex h-[72px] shrink-0 items-center gap-2 px-4 before:absolute before:left-4 before:right-4 before:top-0 before:h-px before:bg-[#EAEEF4] before:content-['']">
            <div className="flex size-8 items-center justify-center rounded-full bg-[color-mix(in_srgb,var(--brand-blue,#1447E6)_32%,#fff)] font-mono text-[14.22px] text-[#020617]">J</div>
            {!collapsed && (
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-[#0a0a0a]">jingsujiang</p>
                <p className="truncate text-xs text-[#0a0a0a]">管理员</p>
              </div>
            )}
          </div>
        </aside>

        <main style={{ marginLeft: w }} className="flex-1 min-w-0 h-screen overflow-x-hidden transition-[margin-left] duration-300">
          {children}
        </main>
      </div>
    </SidebarCtx.Provider>
  );
}
```

### 9.3 Minimal HTML/CSS fallback

```html
<aside class="cp-admin-sidebar" aria-label="管理后台导航">
  <header class="cp-admin-sidebar__header">
    <a class="cp-admin-sidebar__brand" href="/">
      <span class="cp-admin-sidebar__logo"><!-- svg --></span>
      <span class="cp-admin-sidebar__brand-text">
        <strong>管控端</strong>
        <small>ClawPro Admin</small>
      </span>
    </a>
  </header>

  <nav class="cp-admin-sidebar__content">
    <div class="cp-admin-sidebar__group">
      <div class="cp-admin-sidebar__group-label">基础信息</div>
      <ul class="cp-admin-sidebar__menu">
        <li><a class="cp-admin-sidebar__item" data-active="true" aria-current="page" href="/admin/basic-info"><img src="" alt=""><span>基础信息配置</span></a></li>
        <li><a class="cp-admin-sidebar__item" href="/admin/skill-config"><img src="" alt=""><span>技能配置</span><span class="cp-admin-sidebar__badge cp-admin-sidebar__badge--new">New</span></a></li>
      </ul>
    </div>
  </nav>

  <footer class="cp-admin-sidebar__footer">
    <div class="cp-admin-sidebar__avatar">J</div>
    <div class="cp-admin-sidebar__user">
      <p>jingsujiang</p>
      <p>管理员</p>
    </div>
  </footer>
</aside>
<main class="cp-admin-main">…</main>
```

```css
:root {
  --cp-admin-sidebar-w: 240px;
  --cp-admin-sidebar-w-collapsed: 64px;
  --cp-admin-sidebar-bg: #ffffff;
  --cp-admin-sidebar-border: #EAEEF4;
  --cp-admin-sidebar-fg: #0a0a0a;
  --cp-admin-sidebar-muted: #737373;
  --cp-admin-sidebar-item-h: 34px;
  --cp-admin-sidebar-item-r: 4px;
  --cp-admin-sidebar-item-hover: rgba(180, 191, 225, 0.14);
  --cp-admin-sidebar-item-active: linear-gradient(90deg, #e9f3ff 0%, #e3eaff 100%);
  --cp-brand-blue: #1447E6;
}

.cp-admin-sidebar { position: fixed; inset: 0 auto 0 0; width: var(--cp-admin-sidebar-w); display: flex; flex-direction: column; overflow: hidden;
  background: var(--cp-admin-sidebar-bg); border-right: 1px solid var(--cp-admin-sidebar-border); transition: width .3s; z-index: 40; }
.cp-admin-sidebar[data-collapsed="true"] { width: var(--cp-admin-sidebar-w-collapsed); }

.cp-admin-sidebar__header { height: 104px; padding: 16px; display: flex; align-items: flex-start; justify-content: space-between; }
.cp-admin-sidebar__brand { display: flex; align-items: center; gap: 10px; border-radius: var(--cp-admin-sidebar-item-r); }
.cp-admin-sidebar__logo { width: 44px; height: 44px; display: grid; place-items: center; background: #fff; border-radius: 8px; box-shadow: 0 1px 4px rgba(176,182,195,.3); }
.cp-admin-sidebar__brand-text strong { display: block; font-size: 16px; line-height: 22px; font-weight: 500; letter-spacing: 0.08px; color: var(--cp-admin-sidebar-fg); }
.cp-admin-sidebar__brand-text small { display: block; font-size: 12px; line-height: 20px; letter-spacing: 0.18px; color: var(--cp-admin-sidebar-muted); }

.cp-admin-sidebar__content { flex: 1 1 auto; min-height: 0; overflow-y: auto; padding: 0 16px; margin-top: 16px; margin-bottom: 12px; }
.cp-admin-sidebar__group { margin-bottom: 20px; }
.cp-admin-sidebar__group-label { padding: 0 8px; height: 20px; margin-bottom: 8px; font-size: 12px; color: var(--cp-admin-sidebar-muted); letter-spacing: .015em; }
.cp-admin-sidebar__menu { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }

.cp-admin-sidebar__item { display: flex; align-items: center; gap: 8px; height: var(--cp-admin-sidebar-item-h); padding: 0 8px;
  border-radius: var(--cp-admin-sidebar-item-r); font-size: 14px; line-height: 22px; color: var(--cp-admin-sidebar-fg); text-decoration: none; transition: all .15s; }
.cp-admin-sidebar__item img { width: 16px; height: 16px; flex: 0 0 auto; }
.cp-admin-sidebar__item:hover { background: var(--cp-admin-sidebar-item-hover); }
.cp-admin-sidebar__item[data-active="true"] { background: var(--cp-admin-sidebar-item-active); }
.cp-admin-sidebar__item[data-active="true"] img { filter: brightness(0) saturate(100%); }

.cp-admin-sidebar__badge { display: inline-flex; align-items: center; height: 18px; padding: 0 6px; border-radius: 999px; font-size: 11px; margin-left: auto; background: #f5f5f5; color: var(--cp-admin-sidebar-muted); }
.cp-admin-sidebar__badge--new { background: color-mix(in srgb, var(--cp-brand-blue) 10%, #fff); color: var(--cp-brand-blue); }

.cp-admin-sidebar__footer { position: relative; height: 72px; flex: 0 0 auto; display: flex; align-items: center; gap: 8px; padding: 0 16px; }
.cp-admin-sidebar__footer::before { content: ''; position: absolute; left: 16px; right: 16px; top: 0; height: 1px; background: var(--cp-admin-sidebar-border); }
.cp-admin-sidebar__avatar { width: 32px; height: 32px; border-radius: 999px; display: grid; place-items: center; font: 14.22px/1 ui-monospace, SFMono-Regular, Menlo, monospace; color: #020617; background: color-mix(in srgb, var(--cp-brand-blue) 32%, #fff); }
.cp-admin-sidebar__user p { margin: 0; }
.cp-admin-sidebar__user p:first-child { font-size: 14px; line-height: 20px; color: var(--cp-admin-sidebar-fg); font-weight: 500; }
.cp-admin-sidebar__user p:last-child { font-size: 12px; line-height: 20px; color: var(--cp-admin-sidebar-fg); }

.cp-admin-main { margin-left: var(--cp-admin-sidebar-w); height: 100vh; overflow-x: hidden; transition: margin-left .3s; }
.cp-admin-sidebar[data-collapsed="true"] ~ .cp-admin-main { margin-left: var(--cp-admin-sidebar-w-collapsed); }
```

## 10. Migration Rules

- 旧写法：每个页面自己写一遍 `<aside>` + 自己定 width / active 颜色 / 字号。
- 新口径：进入 Admin 端 → 统一通过 `AdminLayout` (内部 `AdminSidebarProvider + AdminSidebar`) 渲染壳，业务页面只关心 `<Page>` 内容。
- 导航数据：必须从单一 source 来（demo 仓为 `client/src/config/adminNav.ts`），禁止页面内 inline 写菜单。
- 折叠偏好：迁移到本组件后保留 `localStorage["admin_sidebar_collapsed"]` 兼容；旧 cookie / sessionStorage 偏好可一次性迁移。
- 视觉断点：宿主仓如有自己的 `breakpoint.lg` ≠ 1200px，需要明确选其一并文档化，不要两套并存。
- Active：旧 demo 用品牌实心蓝 / 左色条的，全部替换为弱蓝渐变 token。
- Sidebar 内部不挂业务弹窗 / Drawer，Drawer 走 `AdminSidebarInset` 内部触发。

## 11. Do / Don't

Do:

- 用 240 / 64 双形态宽度 + 4px 圆角 + 34px 行高。
- 用弱蓝渐变 active + 弱化 hover。
- 用 Group label / SubGroup 区分层级，子项缩进 32px 并保留左侧引导线。
- 用 Tooltip / HoverCard 在收起态承担文本。
- 持久化用户折叠偏好。

Don't:

- 不要给 sidebar 加阴影（白底 + 1px 边足够）。
- 不要把 active 做成实心蓝块、左色条、或粗体强调。
- 不要在 64px 收起态显示 Badge / 文字。
- 不要在每个页面自行包 `<aside>` 实现导航（违反 single shell）。
- 不要把 Tenant TopNav 视觉（64px / 三栏 / blur）套到 Admin Sidebar。
- 不要在视口阈值切换时丢弃用户偏好。
- **不要在展开态给 Header 折叠按钮"省"掉 `Tooltip`**——展开态折叠按钮必须挂 `Tooltip「收起导航」`、收起态必须挂 `Tooltip「展开导航」`，两态文案对称。仅靠 `aria-label` 不够，hover 时屏幕用户拿不到文本反馈。
- **不要给 HeaderAction (`前往用户端` / 图标按钮) 加 hover bg / hover border**。三层都禁止：组件 className（`hover:bg-* hover:border-*`）、装配层（手写 `<a>` 叠 `hover:bg-*`）、`client/src/index.css` raw CSS 兜底（`[data-slot="admin-sidebar-header-action"]:hover { background: ... }`）。展开态已有 → arrow 滑入动效、收起态已有 Tooltip，足以提供反馈；额外的 hover 颜色变化会破坏 Sidebar 白底 + 1px 边的轻量调性，且与设计标准实测不一致。

## 12. QA Checklist

- [ ] 展开 240 / 收起 64，Header 104 / Footer 72，单位精确。
- [ ] 视口 ≤1200px 自动收起，且首次进入读 `localStorage` 偏好优先。
- [ ] 用户主动 toggle 后 `localStorage.admin_sidebar_collapsed` 写入正确。
- [ ] Menu item 高度 34、圆角 4、字号 14、图标 16，无变形。
- [ ] Active 弱蓝渐变 + 图标转纯黑；Hover 弱化背景；Focus ring 蓝。
- [ ] Group label 12px `#737373`，SubGroup 子项缩进 32px + 左引导线。
- [ ] Badge：new 用 brand 弱底；coming-soon / 自定义 用灰底；收起态隐藏。
- [ ] 收起态 Tooltip / HoverCard 内容 = 展开态文本，且 `aria-label` 正确。
- [ ] `aria-current="page"` 与 `data-active` 同步。
- [ ] `AdminSidebarInset` margin-left 与 sidebar width 联动，无抖动。
- [ ] 滚动条仅在滚动 / wheel 时短暂显现 (`scrollbar-on-hover`)。
- [ ] 不与 TenantTopNav / 营销导航视觉混用。
- [ ] 所有图标具备 `alt=""` + `aria-hidden`，键盘可达。

## 13. References

- Demo code: `client/src/components/ui/admin-sidebar.tsx`
- Demo code: `client/src/components/AdminLayout.tsx`
- Demo data: `client/src/config/adminNav.ts`
- Global CSS: `client/src/index.css`（搜 `--admin-sidebar-` / `admin-sidebar-subitem-button`）
- Demo page: `client/src/pages/DesignSystemComponents.tsx`
- 同包参考：`tenant-topnav.md`（Tenant 顶导口径）、`breadcrumb.md`（详情页路径口径）
- 同包参考：`references/admin.md`、`references/foundation.md`、`tokens/colors.md`

## 14. 代码对照（✅/❌）

> 与 `tenant-topnav.md` §12 / `breadcrumb.md` §8 同包但端别 / 范畴不同，本节仅聚焦 AdminSidebar 自身高频误用。

### 14.1 不要在每个页面自己包 `<aside>`

```tsx
// ❌ 业务页面里自包侧栏，导致 sidebar 行为 / 视觉与全局不一致
function BasicInfoPage() {
  return (
    <div className="flex">
      <aside className="w-[240px]">{/* 自己拼一套 */}</aside>
      <main>...</main>
    </div>
  );
}

// ✅ 全部走 AdminLayout，业务页面只关心内容
function BasicInfoPage() {
  return (
    <>
      <AdminPageHeader title="基础信息" />
      ...
    </>
  );
}
// AdminLayout 内部统一挂 AdminSidebarProvider + AdminSidebar + AdminSidebarInset
```

### 14.2 Active 必须是弱蓝渐变，不是实心蓝 / 色条 / 粗体

```tsx
// ❌ 实心品牌蓝 + 白字
<AdminSidebarMenuButton className="bg-[#1447E6] text-white">实例管理</AdminSidebarMenuButton>

// ❌ 左侧 4px 强色条
<AdminSidebarMenuButton className="relative pl-3">
  <span className="absolute left-0 top-0 bottom-0 w-1 bg-[#1447E6]" />实例管理
</AdminSidebarMenuButton>

// ❌ 用 font-bold 强调
<AdminSidebarMenuButton isActive className="font-bold">实例管理</AdminSidebarMenuButton>

// ✅ 走 isActive，由 token 接管视觉
<AdminSidebarMenuButton asChild isActive>
  <Link href="/admin/agents">实例管理</Link>
</AdminSidebarMenuButton>
```

### 14.3 收起态不要塞文字 / Badge

```tsx
// ❌ 收起 64px 还在塞 Badge / 文字，挤爆
{collapsed && (
  <AdminSidebarMenuButton>
    <img src={icon} />
    <span>技能配置</span>
    <AdminSidebarBadge variant="new" />
  </AdminSidebarMenuButton>
)}

// ✅ 收起态只放图标，文字 / Badge 都走 Tooltip
{collapsed ? (
  <Tooltip>
    <TooltipTrigger asChild>
      <Link href={item.href} className="flex h-[34px] w-full items-center justify-center rounded-[4px]">
        <img src={icon} className="size-4" />
      </Link>
    </TooltipTrigger>
    <TooltipContent side="right">技能配置</TooltipContent>
  </Tooltip>
) : (
  <AdminSidebarMenuButton asChild>
    <Link href={item.href}>
      <img src={icon} /><span>技能配置</span><AdminSidebarBadge variant="new" />
    </Link>
  </AdminSidebarMenuButton>
)}
```

### 14.4 不要漏掉 Inset margin-left 联动

```tsx
// ❌ <main> 写死 ml-[240px]，收起后 sidebar 64px 但 main 还偏 240，留出大空白
<aside className={collapsed ? "w-[64px]" : "w-[240px]"}>...</aside>
<main className="ml-[240px]">{children}</main>

// ✅ 用 AdminSidebarInset，自动联动 var(--admin-sidebar-width|width-collapsed)
<AdminSidebar>...</AdminSidebar>
<AdminSidebarInset>{children}</AdminSidebarInset>
```

### 14.5 不要在初始化时强制覆盖 localStorage 偏好

```tsx
// ❌ 每次挂载都按视口写一遍 localStorage，用户主动设置的偏好被冲掉
useEffect(() => {
  const collapsed = window.matchMedia("(max-width: 1200px)").matches;
  setCollapsed(collapsed);
  window.localStorage.setItem("admin_sidebar_collapsed", String(collapsed));
}, []);

// ✅ 初始 state 优先读 localStorage；只有 mql change 事件才同步并写回
const [collapsed, setCollapsed] = useState(() => {
  const v = window.localStorage.getItem("admin_sidebar_collapsed");
  if (v === "true") return true;
  if (v === "false") return false;
  return window.matchMedia("(max-width: 1200px)").matches;
});
useEffect(() => {
  const mql = window.matchMedia("(max-width: 1200px)");
  const onChange = (e: MediaQueryListEvent) => {
    setCollapsed(e.matches);
    window.localStorage.setItem("admin_sidebar_collapsed", String(e.matches));
  };
  mql.addEventListener("change", onChange);
  return () => mql.removeEventListener("change", onChange);
}, []);
```

### 14.6 展开 / 收起两态 Header 折叠按钮都要挂 Tooltip

```tsx
// ❌ 展开态只放裸 button + aria-label，hover 时屏幕用户拿不到 "收起导航" 文本提示
<button
  onClick={toggleSidebar}
  className="mt-3 flex size-5 shrink-0 items-center justify-center rounded-[4px]
             text-[var(--admin-sidebar-foreground)] transition-colors
             hover:text-[var(--text-brand)]
             focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
  aria-label="收起导航"
>
  <SidebarCollapseIcon className="size-4" />
</button>

// ✅ 展开态：包 Tooltip「收起导航」，side="right" sideOffset={8} delayDuration={0}
//    Tooltip 浮层朝右弹到 sidebar 外侧空白 / Inset 留白，不会越界遮挡告警条 / 面包屑
<Tooltip delayDuration={0}>
  <TooltipTrigger asChild>
    <button
      onClick={toggleSidebar}
      aria-label="收起导航"
      className="mt-3 flex size-5 shrink-0 items-center justify-center rounded-[4px]
                 text-[var(--admin-sidebar-foreground)] transition-colors
                 hover:text-[var(--text-brand)]
                 focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
    >
      <SidebarCollapseIcon className="size-4" />
    </button>
  </TooltipTrigger>
  <TooltipContent side="right" sideOffset={8}>收起导航</TooltipContent>
</Tooltip>

// ✅ 收起态（64px）：同样挂 Tooltip「展开导航」，文案与展开态对称
<Tooltip delayDuration={0}>
  <TooltipTrigger asChild>
    <button onClick={toggleSidebar} aria-label="展开导航" className="size-7 ...">
      {sidebarHovered ? <SidebarCollapseIcon /> : <AdminSidebarLogo />}
    </button>
  </TooltipTrigger>
  <TooltipContent side="right" sideOffset={8}>展开导航</TooltipContent>
</Tooltip>
```

### 14.7 HeaderAction hover 不动 bg / border（仅靠 → arrow 或 Tooltip 反馈）

```tsx
// ❌ 给 HeaderAction 加 hover bg / hover border 反馈：
//    破坏 Sidebar 白底 + 1px 边的轻量调性，与新设计标准冲突
<AdminSidebarHeaderAction className="hover:!bg-[#F5F5F5]">
  <GoTenantIcon className="size-4" />
</AdminSidebarHeaderAction>

// ❌ 收起态手写 <a>，再叠 hover:bg-[#f5f5f5]
<a
  href="/my-openclaw"
  className="flex size-8 rounded-[4px] border border-[var(--admin-sidebar-action-border)]
             bg-[var(--admin-sidebar-action-bg)] hover:bg-[#f5f5f5] transition-colors"
>
  <GoTenantIcon className="size-4" />
</a>

// ❌ 组件源里写死 hover:bg-* / hover:border-* token
//    （早期 admin-sidebar.tsx AdminSidebarHeaderAction className 写过，已移除）
className="... hover:bg-[var(--admin-sidebar-action-hover-bg)]
              hover:border-[var(--admin-sidebar-action-hover-border)]"

// ❌ index.css `@layer components` 用 raw CSS 兜底覆写 hover bg / border
//    （早期 client/src/index.css 行 435-438 / 502-505 写过，C-017 已删除）
[data-slot="admin-sidebar-header-action"]:hover {
  background: var(--admin-sidebar-action-hover-bg) !important;
  border-color: var(--admin-sidebar-action-hover-border) !important;
}

// ✅ 组件 default 不挂 hover bg / border；transition 仅锁 color & box-shadow
//    展开态：→ arrow 通过 group-hover 控制 width 0→14px + opacity 0→1 滑入
<AdminSidebarHeaderAction asChild className="group mx-4 !h-8 !w-[208px] justify-center">
  <Link href="/my-openclaw">
    <span className="inline-flex items-center justify-center">
      <MiniBodyText as="span" tone="emphasis">前往用户端</MiniBodyText>
      <span className="ml-0 inline-flex w-0 overflow-hidden
                       transition-[width,margin] duration-300
                       group-hover:ml-1 group-hover:w-3.5">
        <img
          src={arrow}
          className="size-3.5 translate-x-[-6px] opacity-0
                     transition-[transform,opacity] duration-300
                     group-hover:translate-x-0 group-hover:opacity-100"
        />
      </span>
    </span>
  </Link>
</AdminSidebarHeaderAction>

// ✅ 收起态：仅图标 + Tooltip 提供文本反馈，不加 hover 颜色
<Tooltip delayDuration={0}>
  <TooltipTrigger asChild>
    <a
      href="/my-openclaw"
      className="flex size-8 items-center justify-center rounded-[4px]
                 border border-[var(--admin-sidebar-action-border)]
                 bg-[var(--admin-sidebar-action-bg)]
                 text-[var(--admin-sidebar-foreground)]"
    >
      <GoTenantIcon className="size-4" />
    </a>
  </TooltipTrigger>
  <TooltipContent side="right" sideOffset={8}>前往用户端</TooltipContent>
</Tooltip>
```
