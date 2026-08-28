# Tenant TopNav

> Tenant 端顶部导航壳（Logo / CenterTabs / 右侧操作 / Avatar）的视觉与行为规范。Admin 左侧导航见 `admin-sidebar.md`，跨端面包屑见 `breadcrumb.md`，本 spec 只聚焦 Tenant TopNav。

## 1. Purpose

- 统一 Tenant 端顶部导航的高度、布局、背景与右侧入口节奏。
- 让宿主仓在不同 Tenant 业务页（Agent 中心 / Skill / Plugin / 个人空间…）共享同一壳子，不再各自手写 header。
- 与 Admin Sidebar 严格分流，避免端别样式互相污染。

## 2. Scope

- 适用端：**仅 Tenant**。
- 必用场景：所有 Tenant 业务页（Agent 列表 / Skill 详情 / 资产中心 / 设置…）。
- 不适用场景：Admin 页面（用 `admin-sidebar.md`）、Landing 营销页（用营销导航）、详情页内的局部 Tabs / Segment（用 `tabs.md` / `segment.md`）。
- 禁止：
  - Tenant 页面外层套 `AdminSidebar`。
  - Admin 页面外层套 `TenantTopNav`。
  - 在每个 Tenant 页面重复实现顶导骨架（应统一走 `TenantLayout`）。

## 3. Visual Standard

| Item | Value | Notes |
|---|---|---|
| Height | 64px | 顶部导航基线，**不要 80px / 56px** |
| Layout | 左 Logo / 中 Tabs / 右操作 | 三栏结构，flex justify-between |
| Background | 白底半透明 (`bg-white/85`) + `backdrop-blur` 可选 | **不要**复用 AdminSidebar 纯白底 |
| Border | `#E2E8F0` 或 `--cp-border` token | 底部 1px 分割 |
| Min Width | 1200px 策略 | 与 Tenant 主体布局一致 |
| Icon Button | 32×32 (`size-8 rounded-[4px]`) | 右侧消息 / 帮助 / 设置入口统一规格 |
| Avatar | 31px / 圆形 | 用户菜单入口；展开走 `popover-dropdown-menu.md` |
| Z-index | `z-40` | 固定定位 |

## 4. Anatomy

```text
TenantLayout
  TopNav
    Brand (Logo + 产品名)
    CenterTabs (一级业务入口)
    RightCluster
      NavIconButton × N (消息 / 帮助 / 设置 …)
      UserMenu (Avatar → DropdownMenu)
  PageContent
```

## 5. States

- default / hover / active：CenterTabs 各项的常规交互。
- notification：右侧图标按钮可叠加红点或数量徽标。
- user-menu-open：用户菜单浮层按 `popover-dropdown-menu.md` 收口。
- mobile/窄屏：低于 min-width 策略时容器横向滚动，TopNav 自身高度不变。

## 6. Demo Repo Usage

- Tenant 布局：`client/src/components/TenantLayout.tsx`
- Tenant TopNav：`client/src/components/topnav/TopNav.tsx`
- CenterTabs：`client/src/components/topnav/CenterTabs.tsx`
- NavIconButton：`client/src/components/topnav/NavIconButton.tsx`
- UserMenu：`client/src/components/topnav/UserMenu.tsx`

## 7. Portable Fallback

### 7.1 If host repo already has Tenant shell

- 保留宿主仓路由 / 权限 / 用户态逻辑，只对齐：64px 高度、三栏结构、白底半透明、底部 1px 分割。
- 右侧图标按钮统一 32×32，Avatar 31px 圆形，避免每个入口尺寸不一致。

### 7.2 Minimal React fallback

```tsx
export function PortableTenantTopNav({ tabs, right }: { tabs: React.ReactNode; right: React.ReactNode }) {
  return (
    <header className="sticky top-0 z-40 flex h-16 items-center justify-between border-b border-[#E2E8F0] bg-white/85 px-6 backdrop-blur">
      <div className="flex items-center gap-3">
        <img src="/logo.svg" alt="ClawPro" className="h-7" />
        <span className="text-sm font-semibold text-[var(--cp-text-title)]">ClawPro</span>
      </div>
      <nav className="flex items-center">{tabs}</nav>
      <div className="flex items-center gap-2">{right}</div>
    </header>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<header class="cp-tenant-topnav">
  <div class="cp-tenant-brand"><img src="/logo.svg" alt="ClawPro" /><span>ClawPro</span></div>
  <nav class="cp-tenant-tabs">...</nav>
  <div class="cp-tenant-right">
    <button class="cp-tenant-iconbtn"><i class="icon-bell"></i></button>
    <button class="cp-tenant-avatar"><img src="/avatar.png" /></button>
  </div>
</header>
```

```css
.cp-tenant-topnav { position: sticky; top: 0; z-index: 40; height: 64px; padding: 0 24px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #E2E8F0; background: rgba(255,255,255,.85); backdrop-filter: blur(8px); }
.cp-tenant-iconbtn { width: 32px; height: 32px; border-radius: 4px; }
.cp-tenant-avatar { width: 31px; height: 31px; border-radius: 50%; overflow: hidden; }
```

## 8. Migration Rules

- 旧写法：每个页面自己写 header，自己定高度，自己放 Logo / 用户菜单。
- 新口径：统一走 `TenantLayout` + `TopNav`，业务页只关心内容区。
- 不允许把 Admin Sidebar 的纯白底 / 240px 宽度结构复用到 Tenant。

## 9. Do / Don't

Do:

- 维持 64px 高度 + 三栏结构。
- 右侧图标按钮统一 32×32，Avatar 统一 31px 圆形。
- 用户菜单浮层走 `popover-dropdown-menu.md` 标准。

Don't:

- 不要 80px / 56px 之类的非标高度。
- 不要把 TopNav 做成多行堆叠（Logo 一行 / 导航一行）。
- 不要把 Tenant 胶囊导航样式套到 Admin。

## 10. QA Checklist

- [ ] Height = 64px，三栏结构正确（左 Logo / 中 Tabs / 右操作）
- [ ] 背景 `bg-white/85` + `backdrop-blur`（或对齐 token）
- [ ] 底部 1px `--cp-border` 分割
- [ ] 右侧图标按钮均为 32×32，Avatar 31px 圆形
- [ ] 用户菜单走 `popover-dropdown-menu.md`，不自定义浮层
- [ ] 与 AdminSidebar / Landing 导航零样式串扰

## 11. References

- Demo code: `client/src/components/TenantLayout.tsx`
- Demo code: `client/src/components/topnav/TopNav.tsx`
- Demo code: `client/src/components/topnav/CenterTabs.tsx`
- Demo code: `client/src/components/topnav/NavIconButton.tsx`
- Demo code: `client/src/components/topnav/UserMenu.tsx`
- Related spec: `component-specs/admin-sidebar.md`（Admin 端导航对照）
- Related spec: `component-specs/breadcrumb.md`（详情页路径）
- Related spec: `component-specs/popover-dropdown-menu.md`（用户菜单浮层）
- Related reference: `references/tenant.md`

## 12. 代码对照（✅/❌）

> Tenant TopNav 高频误用 → ClawPro 正确写法。

### 12.1 必须 64px + 三栏，不要 80px / 单栏堆叠

```tsx
// ❌ Tenant 顶部导航做成 80px / 单栏堆叠
<header className="h-20 px-8 flex flex-col">
  <Logo />
  <nav className="mt-2">...</nav>
</header>

// ✅ 64px + 左 Logo / 中 Tabs / 右操作
<header className="h-16 px-6 flex items-center justify-between border-b border-[#E2E8F0] bg-white/85 backdrop-blur">
  <Logo />
  <CenterTabs />
  <div className="flex items-center gap-2">
    <NavIconButton><Bell className="h-4 w-4" /></NavIconButton>
    <UserMenu />
  </div>
</header>
```

### 12.2 不要把 AdminSidebar 套进 Tenant 页面

```tsx
// ❌ Tenant 业务页套了 AdminSidebar，导航宽度 / 风格不对
<div data-end="tenant">
  <AdminSidebar />
  <main className="ml-[240px]">
    <h1>我的 Agent</h1>
  </main>
</div>

// ✅ 端别分流到对应 Layout
<TenantLayout>
  <h1>我的 Agent</h1>
  ...
</TenantLayout>
```

### 12.3 右侧入口尺寸不要散开，按 32 / 31 收口

```tsx
// ❌ 右侧图标按钮各自尺寸（28 / 36 / 40 混杂）
<div className="flex gap-2">
  <button className="size-7"><Bell /></button>
  <button className="size-9"><HelpCircle /></button>
  <img src="/avatar.png" className="size-10 rounded-full" />
</div>

// ✅ 图标按钮统一 32×32，Avatar 31px 圆形
<div className="flex items-center gap-2">
  <NavIconButton><Bell className="h-4 w-4" /></NavIconButton>
  <NavIconButton><HelpCircle className="h-4 w-4" /></NavIconButton>
  <UserMenu /> {/* 内部 Avatar 31px 圆形 */}
</div>
```
