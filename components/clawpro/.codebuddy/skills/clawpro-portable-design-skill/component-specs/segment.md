# Segment（分段切换器）

## 1. Purpose

- 统一 Admin 与 Tenant 的"局部切换 / 卡片内分区 / 工具栏筛选"控件，避免宿主仓把不同端的 Tab 视觉混用。
- 与 `tabs.md` 严格分流：Segment 不承担"页面顶部一级导航"，所有 page header 下方一级 Tab 都走 LineTabs。
- 这一类组件风险高，因为同样是切换器，**Admin（4px 方角）** 和 **Tenant（80px 胶囊）** 的造型差异是明确的端别标记，禁止互通。

## 2. Scope

- 适用端：Admin（方角 Segment）/ Tenant（胶囊 Segment）/ 双端轻量场景（TextSwitch 仅作为 Tenant 弱切换历史变体保留，不再扩展）
- 必用场景：
  - 卡片内部分区切换（概览 / 详情 / 配置）
  - 表格 / 列表上方的筛选条（"全部 / 分组"、"日 / 周 / 月"）
  - 弹窗内的局部模式切换
  - 工具栏内的视图模式切换（List / Grid）
- 不适用场景：
  - 页面标题下方的一级 Tab → 用 `tabs.md`（LineTabs）
  - 顶部主导航 → 用 `admin-sidebar.md`（Admin）/ `tenant-topnav.md`（Tenant）
  - 分页 → 用 `pagination.md`
  - 单个开关按钮 → 用 `selection-controls.md`
  - 多选筛选 → 用 `search-filter-bar.md`

## 3. Visual Standard

### 3.1 Admin Segment（4px 方角，两端共用同一参数）

| Token | Value | Notes |
|---|---|---|
| Container bg | `var(--bg-segment-track)` = `#DBDDE432` | 冷灰偏蓝 + 20% alpha |
| Container border | 同 bg（融为一体） | `border border-[var(--bg-segment-track)]` |
| Container radius | `6px` | 容器外圆角 |
| Container padding | `2px` ~ `3px` | 留出滑块呼吸空间 |
| Container height | `36px`（`h-9`） |  |
| Item radius | `4px` | 滑块内圆角，对齐 claw-* 按钮 |
| Item padding | `4px 16px`（`px-4 py-1`） |  |
| Item typography | `14px / Normal`（默认） / `14px / Semibold`（active） |  |
| Active bg | `#FFFFFF` |  |
| Active text | `#020617`（gray-950） |  |
| Active shadow | `0px 1px 2px rgba(0,0,0,0.05)` | `var(--shadow-segment)` |
| Inactive text | `#7B818F` |  |
| Hover text | `#4B5563` | 仅 inactive |
| Disabled text | `#D3D6DB` |  |

### 3.2 Tenant Segment（80px 胶囊）

| Token | Value | Notes |
|---|---|---|
| Container bg | `rgba(219, 221, 228, 0.32)` |  |
| Container radius | `80px` | 全圆角胶囊 |
| Container padding | `0` | 直接靠 item 自带 padding 撑 |
| Container height | `36px`（`h-9`） |  |
| Item radius | `40px` | 胶囊滑块 |
| Item padding | `4px 12px`（`px-3 py-1`） |  |
| Item typography | `14px / 22px line-height / 0.5% letter-spacing`，default 400 / active 500 |  |
| Active bg | `#FFFFFF` |  |
| Active text | `#020617`（gray-950） |  |
| Active outline | `outline outline-1 outline-[#CDD4DC]` | 是 outline 不是 border |
| Active shadow | `0px 1px 4px 0px rgba(0,0,0,0.05)` |  |
| Inactive text | `#334155`（slate-700） |  |
| Hover text | `#020617`（gray-950） | 仅 inactive |

### 3.3 关键差异点（必须记住）

- **Admin 6px 容器 / 4px 滑块 / 半透明灰底**；**Tenant 80px 容器 / 40px 滑块 / 半透明灰底**。
- Tenant active 用 **outline 1px** + 阴影；Admin active 只用阴影，**不画边框**。
- 不要把 Admin 用 `className="rounded-full"` 改成胶囊——破坏单一真理源；端别错了就换组件，不是改类名。

## 4. Anatomy

```text
Segment
  Container (track)
    Trigger × n
      label
      icon optional
    Active Indicator (white fill + shadow ± outline)
  Panel optional (受控模式时配合 SegmentContent)
```

## 5. States

| State | Admin 视觉 | Tenant 视觉 |
|---|---|---|
| default | inactive 灰字 #7B818F | inactive 灰字 #334155 |
| hover | inactive 文字加深到 #4B5563 | inactive 文字加深到 #020617 |
| active | 白底 + #020617 字 + Semibold + shadow-segment | 白底 + #020617 字 + Medium + outline #CDD4DC + shadow |
| focus-visible | `ring-[3px] ring-[#355EF1]/20` | 同左 |
| disabled | text #D3D6DB，pointer-events-none | text gray-300，pointer-events-none |
| overflow | 项数 > 5 改用左侧子导航 / Sidebar | 同左 |

## 6. Demo Repo Usage

代码仓单一来源：`client/src/components/ui/segment.tsx`，包含两套端别 + 一套 TextSwitch。

```tsx
import {
  Segment, SegmentList, SegmentItem, SegmentContent,
  SegmentGroup, SegmentOption,
  TenantSegment, TenantSegmentList, TenantSegmentItem, TenantSegmentContent,
  TenantSegmentGroup, TenantSegmentOption,
} from "@/components/ui/segment";

// Admin 受控（与 panel 联动）
<Segment defaultValue="basic">
  <SegmentList>
    <SegmentItem value="basic">基础配置</SegmentItem>
    <SegmentItem value="tools">工具管理</SegmentItem>
  </SegmentList>
  <SegmentContent value="basic">…</SegmentContent>
  <SegmentContent value="tools">…</SegmentContent>
</Segment>

// Admin 独立（自管 state）
<SegmentGroup>
  <SegmentOption active={mode === "all"} onClick={() => setMode("all")}>全部</SegmentOption>
  <SegmentOption active={mode === "group"} onClick={() => setMode("group")}>分组</SegmentOption>
</SegmentGroup>

// Tenant 受控
<TenantSegment defaultValue="all">
  <TenantSegmentList>
    <TenantSegmentItem value="all">全部</TenantSegmentItem>
    <TenantSegmentItem value="group">分组</TenantSegmentItem>
  </TenantSegmentList>
</TenantSegment>
```

- Admin 与 Tenant 接口完全一致，差别只在视觉与导出名前缀。
- TextSwitch 仅作为 Tenant 弱切换历史变体保留（"普通 / 多分组"），不要因为"想做轻一点"而新增新的弱切换皮肤。

## 7. Portable Fallback

### 7.1 If host repo already has Tabs

- 保留宿主仓 Tabs / Tabs.Trigger 逻辑层。
- 视觉层强制分流：写 `AdminSegment` / `TenantSegment` 两个 wrapper，不共用一套默认皮肤。
- 端别在路由 / context 层先确定再选 wrapper，**不要让组件内部自己判断 `pathname.startsWith('/admin')`**——这是端别污染。

### 7.2 Minimal React fallback

```tsx
export function PortableAdminSegment({ children }: { children: React.ReactNode }) {
  return (
    <div
      role="tablist"
      className="bg-[var(--cp-bg-subtle)] inline-flex h-9 w-fit items-center justify-center
                 rounded-[6px] border border-[var(--cp-bg-subtle)] p-[2px]"
    >
      {children}
    </div>
  );
}

export function PortableAdminSegmentOption({
  active, children, onClick,
}: { active: boolean; children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`inline-flex h-[calc(100%-1px)] items-center justify-center rounded-[4px]
                  border border-transparent px-4 py-1 text-sm whitespace-nowrap transition-all ${
        active
          ? "bg-white text-[var(--cp-text-emphasis)] font-semibold shadow-[0_1px_2px_rgba(0,0,0,0.05)]"
          : "text-[var(--text-secondary)] font-normal hover:text-[var(--cp-text-emphasis)]"
      }`}
    >
      {children}
    </button>
  );
}

export function PortableTenantSegment({ children }: { children: React.ReactNode }) {
  return (
    <div
      role="tablist"
      className="relative inline-flex h-9 w-fit items-center rounded-[80px] p-0"
      style={{ background: "rgba(219, 221, 228, 0.32)" }}
    >
      {children}
    </div>
  );
}

export function PortableTenantSegmentOption({
  active, children, onClick,
}: { active: boolean; children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`relative z-10 inline-flex h-full items-center justify-center rounded-[40px]
                  px-3 py-1 text-[14px] leading-[22px] tracking-[0.005em] whitespace-nowrap transition-all ${
        active
          ? "bg-white text-[var(--cp-text-emphasis)] font-medium outline outline-1 outline-[#CDD4DC] shadow-[0_1px_4px_rgba(0,0,0,0.05)]"
          : "text-slate-700 font-normal hover:text-[var(--cp-text-emphasis)]"
      }`}
    >
      {children}
    </button>
  );
}
```

### 7.3 HTML / CSS fallback（Tenant 胶囊）

```html
<div class="cp-segment cp-segment--tenant" role="tablist">
  <button class="cp-segment__item cp-segment__item--active" type="button">全部</button>
  <button class="cp-segment__item" type="button">分组</button>
</div>

<style>
  .cp-segment--tenant {
    display: inline-flex;
    align-items: center;
    height: 36px;
    border-radius: 80px;
    background: rgba(219, 221, 228, 0.32);
    padding: 0;
  }
  .cp-segment--tenant .cp-segment__item {
    height: 100%;
    padding: 4px 12px;
    border: 0;
    border-radius: 40px;
    background: transparent;
    font-size: 14px;
    line-height: 22px;
    letter-spacing: 0.005em;
    color: #334155;
    cursor: pointer;
    transition: color 150ms ease, background 150ms ease;
  }
  .cp-segment--tenant .cp-segment__item:hover { color: #020617; }
  .cp-segment--tenant .cp-segment__item--active {
    background: #fff;
    color: #020617;
    font-weight: 500;
    outline: 1px solid #CDD4DC;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  }
</style>
```

## 8. Migration Rules

- 旧写法：宿主仓任意 Tabs 组件直接套所有页面（无端别区分）。
- 新口径：
  1. 先判断"是不是 page header 下方"——是的话走 `tabs.md`（LineTabs），不是再继续。
  2. 再判断 Admin / Tenant——映射到方角 Segment 或胶囊 Segment。
  3. 不允许混用两套端别皮肤。
- 可以暂时兼容：宿主仓只有一套 Tabs 逻辑组件，但视觉必须分流（外层包 wrapper）。
- TextSwitch 不再作为新场景的迁移目标；不要为了"低密度局部切换"新增一套弱切换视觉，复用 Tenant 胶囊 + size 即可。
- 不允许新增：
  - Tenant 页面继续使用 Admin 方角 Segment，或 Admin 页面继续使用 Tenant 胶囊。
  - 在 Admin 组件上写 `className="rounded-full"` 临时改胶囊。
  - 把 Segment 当主导航或 page header 一级 Tab 使用。

## 9. Do / Don't

Do:

- 先判断 page header / 卡片内 → 选 LineTabs / Segment。
- 再判断 Admin / Tenant → 选方角 / 胶囊。
- Active 项视觉够强（白底 + 阴影 ± outline），不靠颜色微差。
- 项数 > 5 时换结构（左侧子导航 / 拆 Tab）。
- TextSwitch 仅在已存在的 Tenant 弱切换场景维护，不扩展。

Don't:

- 不要把方角 / 胶囊一套皮肤通吃双端。
- 不要在 Admin 组件上写 `className="rounded-full"` 改胶囊（破坏端别真理源）。
- 不要让 active 态只靠颜色微差异，看不清。
- 不要在小空间硬塞过多切换项。
- 不要在 page header 下方用 Segment——那是 LineTabs 的位置。
- 不要给 Segment 接管主导航。

## 10. QA Checklist

- [ ] 端别正确（Admin 方角 / Tenant 胶囊），没有跨端复用
- [ ] 没有 `className="rounded-full"` 把 Admin Segment 改胶囊
- [ ] active / hover / disabled / focus-visible 状态完整
- [ ] active 同时有 bg + 阴影（Tenant 还需 outline）
- [ ] 项数 ≤ 5；超出已转结构
- [ ] 没有用 Segment 替代 LineTabs（page header 下方）或主导航
- [ ] 没有为低密度场景新增弱切换皮肤
- [ ] portable fallback（React / HTML）能跑通

## 11. References

- Related rules: `references/admin.md`, `references/tenant.md`, `references/components.md`, `references/conflict-log.md`
- Related specs: `component-specs/tabs.md`（page header 下方一级 Tab）、`component-specs/admin-sidebar.md` / `component-specs/tenant-topnav.md`（主导航边界）、`component-specs/selection-controls.md`（单选 / 复选 / 开关边界）

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / §3 同口径。Segment 5 项高频误用 → ClawPro 正确写法。

### 12.1 端别分流：方角 / 胶囊不通吃

```tsx
// ❌ Admin 配置页用了 Tenant 胶囊
<div data-end="admin" className="inline-flex h-9 rounded-full bg-[var(--bg-segment-track)] px-1">
  <button className="rounded-full bg-white px-3">概览</button>
  <button className="px-3">详情</button>
</div>

// ❌ Tenant 业务页用了 Admin 方角 + 还硬改成胶囊
<SegmentGroup className="rounded-full">  {/* 破坏单一真理源 */}
  <SegmentOption active>全部</SegmentOption>
  <SegmentOption>分组</SegmentOption>
</SegmentGroup>

// ✅ Admin 走方角
<SegmentGroup>
  <SegmentOption active={tab === "overview"} onClick={() => setTab("overview")}>概览</SegmentOption>
  <SegmentOption active={tab === "detail"} onClick={() => setTab("detail")}>详情</SegmentOption>
</SegmentGroup>

// ✅ Tenant 走胶囊
<TenantSegmentGroup>
  <TenantSegmentOption active={filter === "all"} onClick={() => setFilter("all")}>全部</TenantSegmentOption>
  <TenantSegmentOption active={filter === "group"} onClick={() => setFilter("group")}>分组</TenantSegmentOption>
</TenantSegmentGroup>
```

### 12.2 Active 必须够辨识，不要靠颜色微差

```tsx
// ❌ Active 只把文字从灰变深灰，无背景 / 阴影变化
<SegmentOption className="text-[var(--text-secondary)]">概览</SegmentOption>
<SegmentOption className="text-[var(--cp-text-emphasis)]">详情</SegmentOption>  {/* "active" */}

// ✅ Admin：active 白底 + Semibold + shadow-segment（组件已内置，不要覆盖）
<SegmentOption active={tab === "overview"} onClick={() => setTab("overview")}>概览</SegmentOption>

// ✅ Tenant：active 白底 + Medium + outline + shadow（组件已内置）
<TenantSegmentOption active={filter === "all"} onClick={() => setFilter("all")}>全部</TenantSegmentOption>
```

### 12.3 项数过多换结构，不要硬挤

```tsx
// ❌ 8 个 Segment 项硬挤一排，触摸目标过窄、滑块跳动夸张
<SegmentGroup>
  <SegmentOption>概览</SegmentOption>
  <SegmentOption>配置</SegmentOption>
  <SegmentOption>权限</SegmentOption>
  <SegmentOption>路由</SegmentOption>
  <SegmentOption>限流</SegmentOption>
  <SegmentOption>缓存</SegmentOption>
  <SegmentOption>密钥</SegmentOption>
  <SegmentOption>审计</SegmentOption>
</SegmentGroup>

// ✅ 项数 > 5 时改用左侧子导航
<div className="flex gap-6">
  <aside className="w-40 space-y-1">
    {sections.map((s) => (
      <button key={s.value} data-active={tab === s.value}
        className="flex h-[34px] w-full items-center rounded-[4px] px-2 text-sm
                   data-[active=true]:bg-[var(--cp-brand-tint)]">
        {s.label}
      </button>
    ))}
  </aside>
  <main className="flex-1">…</main>
</div>
```

### 12.4 Segment 不替代 page header 一级 Tab，也不替代主导航

```tsx
// ❌ 把 page header 下方的"内置通道 / 自定义通道"做成 Tenant 胶囊
<TenantPageHeader title="通道配置" />
<TenantSegmentGroup>
  <TenantSegmentOption active>内置通道</TenantSegmentOption>
  <TenantSegmentOption>自定义通道</TenantSegmentOption>
</TenantSegmentGroup>

// ❌ 把"Agent / 模型 / 配额"做成 Admin Segment 当主导航
<SegmentGroup>
  <SegmentOption active>Agent</SegmentOption>
  <SegmentOption>模型</SegmentOption>
  <SegmentOption>配额</SegmentOption>
</SegmentGroup>

// ✅ page header 下方一级 Tab 走 LineTabs（参考 tabs.md §3）
<TenantPageHeader title="通道配置" description="…" />
<LineTabs
  tabs={[{ id: "in", label: "内置通道" }, { id: "custom", label: "自定义通道" }]}
  active={tab} onChange={setTab}
/>

// ✅ 主导航走 Sidebar（参考 admin-sidebar.md §3）
<AdminSidebar>
  <SidebarItem href="/admin/agents">Agent</SidebarItem>
  <SidebarItem href="/admin/models">模型</SidebarItem>
</AdminSidebar>
```

### 12.5 不要为低密度局部切换发明新弱切换皮肤

```tsx
// ❌ 局部 2~3 项弱切换硬塞一个手写 text-switch（双端不共用 / 无端别真理源）
<div className="flex items-center gap-4 text-sm">
  <button className="text-[var(--cp-text-brand)] underline-offset-4 underline">日</button>
  <button className="text-[var(--text-secondary)]">周</button>
  <button className="text-[var(--text-secondary)]">月</button>
</div>

// ✅ Tenant 复用胶囊（不为单一场景发明新视觉）
<TenantSegmentGroup>
  <TenantSegmentOption active={range === "day"} onClick={() => setRange("day")}>日</TenantSegmentOption>
  <TenantSegmentOption active={range === "week"} onClick={() => setRange("week")}>周</TenantSegmentOption>
  <TenantSegmentOption active={range === "month"} onClick={() => setRange("month")}>月</TenantSegmentOption>
</TenantSegmentGroup>

// ✅ Tenant 已存在的"普通 / 多分组"场景才能用 TextSwitch（不要扩展）
<TextSwitch>
  <TextSwitchOption active={mode === "single"} onClick={() => setMode("single")}>普通</TextSwitchOption>
  <TextSwitchOption active={mode === "multi"} onClick={() => setMode("multi")}>多分组</TextSwitchOption>
</TextSwitch>
```
