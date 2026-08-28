# Tabs（LineTabs · 下划线一级 Tab）

## 1. Purpose

- 统一**页面标题下方的一级内容分区切换**，是 ClawPro 唯一允许出现在 page header 下的 Tab 视觉。
- 与 `segment.md` 严格分流：LineTabs 只承担"页面级一级导航 Tab"，所有"卡片内 / 弹窗内 / 表格工具栏内"的切换全部走 Segment。
- 这一类组件的最大风险点是被"借皮"：开发者经常把下划线 Tab 复用到卡片或弹窗里当切换器，破坏视觉层级。

## 2. Scope

- 适用端：Admin / Tenant / Shared（双端共用同一套 LineTabs 视觉）
- 必用场景：页面标题（`AdminPageHeader` / Tenant 页头）下方的一级内容分区切换，例如「内置通道 / 自定义通道」「初始技能包 / 角色设定 / 技能安装来源」。
- 不适用场景：
  - 卡片内部 / 弹窗内部切换 → 用 `segment.md`
  - 表格上方筛选条 → 用 `segment.md`（"全部 / 分组"胶囊）
  - 顶部主导航 → 用 `admin-sidebar.md`（Admin）/ `tenant-topnav.md`（Tenant）
  - 分页 → 用 `pagination.md`
  - 单个开关按钮 → 用 `selection-controls.md`

## 3. Visual Standard

| Token | Value | Notes |
|---|---|---|
| Container | `flex items-center gap-1 border-b border-[#dbe6ff]` | 浅蓝灰底边，不是黑色 |
| Item padding | `px-4 py-3` | 单项点击区 |
| Item typography | 14px / Medium（`BodyMedium`） | 选中 / 默认同字号同字重 |
| Active indicator | `border-b-2 border-[#0A0A0A] -mb-px` | 黑色 2px 下划线，靠 `-mb-px` 盖住容器 1px 底边 |
| Active text | `text-[var(--text-title)]` | `tone="primary"` |
| Default text | `text-[var(--text-muted)]` | `tone="muted"` |
| Hover | `hover:text-[var(--text-title)] transition-colors` | 仅文字加深，不出下划线 |
| Disabled | `text-[var(--text-muted)] cursor-not-allowed` | 保留位置，不参与点击 |
| `comingSoon` 右侧 Badge | `Badge variant="outline" px-1.5 py-0.5` | 文案"即将开放" |

> 关键差异点：**LineTabs 没有"轨道背景"**。如果出现灰色或胶囊背景，那是 Segment，不是 LineTabs。

## 4. Anatomy

```text
LineTabs
  Track (border-b only, no fill)
    Trigger × n
      label (BodyMedium)
      ComingSoon Badge optional
    Active Indicator (border-b-2 black, on active trigger)
  Description optional (CompactText, muted, mt-3 mb-6)
```

## 5. States

| State | 视觉 |
|---|---|
| default | muted 文字，无下划线 |
| hover | 文字加深为 title，无下划线 |
| active | title 文字 + 2px 黑色下划线 |
| disabled | disabled 文字色，鼠标 not-allowed |
| with-coming-soon | 文字 + 右侧 outline Badge |

无 selected-hover / selected-pressed 单独态——active 项不再响应 hover 视觉。

## 6. Demo Repo Usage

代码仓单一来源：`client/src/components/ui/line-tabs.tsx`，导出 `LineTabs`。

```tsx
import LineTabs from "@/components/ui/line-tabs";

<LineTabs
  tabs={[
    { id: "preset", label: "初始技能包" },
    { id: "roles",  label: "角色设定" },
    { id: "source", label: "技能安装来源", comingSoon: true },
  ]}
  active={tab}
  onChange={setTab}
  description="当前 Tab 的描述文案"
/>
```

- 不要新增 LineTabs 的 size 变体（s/m/l 都不开放）。
- 不要给 LineTabs 加图标前缀；如果业务真的需要图标，先评估是否应该改为 Segment。
- "即将开放" Badge 走 `badge.md` 的 outline variant，不要内嵌自定义样式。

## 7. Portable Fallback

### 7.1 If host repo already has Tabs

- 保留宿主仓 Tabs 逻辑层（state / onChange）。
- **必须重写视觉层**：宿主仓 Radix Tabs 默认皮肤通常是胶囊或方角 Segment，不能直接当 LineTabs 用。包一个 wrapper 把 `data-state="active"` 映射成 `border-b-2 border-[#0A0A0A] -mb-px`。

### 7.2 Minimal React fallback

```tsx
export function PortableLineTabs<T extends string>({
  tabs,
  active,
  onChange,
}: {
  tabs: { id: T; label: string }[];
  active: T;
  onChange: (id: T) => void;
}) {
  return (
    <div className="flex items-center gap-1 border-b border-[#dbe6ff]">
      {tabs.map((tab) => {
        const isActive = active === tab.id;
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            className={`px-4 py-3 text-sm font-medium transition-colors whitespace-nowrap ${
              isActive
                ? "text-[var(--cp-text-emphasis)] border-b-2 border-[#0A0A0A] -mb-px"
                : "text-[var(--cp-text-secondary)] hover:text-[var(--cp-text-emphasis)]"
            }`}
          >
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
```

### 7.3 HTML / CSS fallback

```html
<div class="cp-line-tabs">
  <button class="cp-line-tab cp-line-tab--active" type="button">内置通道</button>
  <button class="cp-line-tab" type="button">自定义通道</button>
</div>

<style>
  .cp-line-tabs {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    border-bottom: 1px solid #dbe6ff;
  }
  .cp-line-tab {
    padding: 12px 16px;
    font-size: 14px;
    font-weight: 500;
    color: var(--cp-text-secondary);
    background: transparent;
    border: 0;
    cursor: pointer;
    transition: color 150ms ease;
  }
  .cp-line-tab:hover { color: var(--cp-text-emphasis); }
  .cp-line-tab--active {
    color: var(--cp-text-emphasis);
    border-bottom: 2px solid #0A0A0A;
    margin-bottom: -1px;
  }
</style>
```

## 8. Migration Rules

- 旧写法：宿主仓 Radix Tabs 默认皮肤直接套页面顶部 → 视觉常表现为胶囊或方角 Segment，与 ClawPro page header 视觉脱节。
- 新口径：页面标题下方一律改用 LineTabs（下划线 + `border-b-[#dbe6ff]` + `border-b-2 border-[#0A0A0A]`），配合 page header 留白 `mb-1` / 描述 `mt-3 mb-6`。
- 可以暂时兼容：宿主仓只有一个 Tabs 逻辑组件 → 在最外层包一层 wrapper 做视觉重写，state / onChange 不变。
- 不允许新增：
  - 在卡片 / 弹窗内部使用 LineTabs（必须改成 Segment）。
  - 给 LineTabs 自定义 active 颜色（active 下划线必须是 `#0A0A0A`，不是 `--cp-brand` 蓝色）。
  - LineTabs 套两层（外层一级 Tab + 内层一级 Tab）→ 内层必须改为 Segment。

## 9. Do / Don't

Do:

- 只用在 page header 下方做"页面级一级 Tab"。
- Active 同时给 title 文字色 + 2px 黑色下划线 + `-mb-px` 盖容器底边。
- 项数 > 5 时考虑改为左侧子导航或拆页面。
- "即将开放"统一用 outline Badge，不发明新视觉。

Don't:

- 不要在卡片 / 弹窗 / 表格工具栏里用 LineTabs，那些是 Segment 的领地。
- 不要把 active 下划线换成品牌蓝（蓝色是按钮主操作色，不是 Tab 选中态）。
- 不要给 active 项加背景填充（一旦填充就不是 LineTabs 了，去用 Segment）。
- 不要把 active / inactive 字重做差异化（都是 14/Medium，靠颜色 + 下划线区分）。
- 不要在 LineTabs 里塞 icon-only 项；icon-only 一律走 Segment 或 toolbar。

## 10. QA Checklist

- [ ] 容器 `border-b border-[#dbe6ff]` 已设置（不是黑色 / 灰色）
- [ ] active 项 `border-b-2 border-[#0A0A0A] -mb-px` 完整
- [ ] active / default / hover / disabled 文字色全部命中 token，没有写死的灰阶
- [ ] 项数 ≤ 5；超出已转结构
- [ ] 没有用 LineTabs 替代主导航或卡片内切换
- [ ] 没有自定义 active 颜色 / 字重 / 背景填充
- [ ] portable fallback（React / HTML）能跑通

## 11. References

- Related rules: `references/admin.md`, `references/tenant.md`, `references/components.md`
- Related specs: `component-specs/segment.md`（卡片内 / 工具栏切换）、`component-specs/badge.md`（comingSoon Badge）、`component-specs/admin-sidebar.md` / `component-specs/tenant-topnav.md`（主导航边界）

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / §3 同口径。LineTabs 5 项高频误用 → ClawPro 正确写法。

### 12.1 不要在卡片 / 弹窗 / 工具栏里用 LineTabs

```tsx
// ❌ 把卡片内的"概览 / 详情"做成 LineTabs
<Card>
  <div className="flex items-center gap-1 border-b border-[#dbe6ff]">
    <button className="px-4 py-3 border-b-2 border-[#0A0A0A] -mb-px">概览</button>
    <button className="px-4 py-3 text-[var(--text-muted)]">详情</button>
  </div>
  <CardContent>…</CardContent>
</Card>

// ✅ 卡片内切换走 Segment（参考 segment.md §3）
<Card>
  <SegmentGroup>
    <SegmentOption active={tab === "overview"} onClick={() => setTab("overview")}>概览</SegmentOption>
    <SegmentOption active={tab === "detail"} onClick={() => setTab("detail")}>详情</SegmentOption>
  </SegmentGroup>
  <CardContent>…</CardContent>
</Card>
```

### 12.2 Active 必须是 2px 黑色下划线 + 盖底边，不要换成品牌蓝

```tsx
// ❌ Active 用品牌蓝下划线（蓝色是按钮主操作色，不是 Tab 选中态）
<button className="px-4 py-3 border-b-2 border-[var(--brand-blue)] text-[var(--cp-text-brand)]">
  内置通道
</button>

// ❌ Active 只换文字色，没下划线 → 视觉太弱，看不出选中
<button className="px-4 py-3 text-[var(--cp-text-emphasis)] font-semibold">内置通道</button>

// ✅ 标准 active：黑色下划线 + -mb-px 压住容器底边 + title 文字色
<button className="px-4 py-3 text-sm font-medium text-[var(--cp-text-emphasis)]
                   border-b-2 border-[#0A0A0A] -mb-px">
  内置通道
</button>
```

### 12.3 不要给 LineTabs 项加背景填充（变成 Segment 就不是 LineTabs）

```tsx
// ❌ 给 active 项加白底胶囊 → 视觉变成 Segment，但容器还有 border-b
<div className="flex items-center gap-1 border-b border-[#dbe6ff]">
  <button className="px-4 py-3 rounded-full bg-white shadow-sm">内置通道</button>
  <button className="px-4 py-3 text-[var(--cp-text-secondary)]">自定义通道</button>
</div>

// ✅ 选其一：要下划线就完整走 LineTabs（不填背景）
<LineTabs tabs={tabs} active={tab} onChange={setTab} />

// ✅ 或完整走 Segment（去掉 border-b 容器）
<TenantSegmentGroup>
  <TenantSegmentOption active={tab === "in"} onClick={() => setTab("in")}>内置通道</TenantSegmentOption>
  <TenantSegmentOption active={tab === "custom"} onClick={() => setTab("custom")}>自定义通道</TenantSegmentOption>
</TenantSegmentGroup>
```

### 12.4 LineTabs 不替代主导航，也不嵌套自身

```tsx
// ❌ 把"Agent / 模型 / 配额"做成顶部 LineTabs 当主导航
<LineTabs
  tabs={[{ id: "agents", label: "Agent" }, { id: "models", label: "模型" }, { id: "quota", label: "配额" }]}
  active={mainNav}
  onChange={setMainNav}
/>

// ❌ LineTabs 套两层：外层一级 Tab，内层又一级 Tab
<LineTabs tabs={primary} active={p} onChange={setP} />
<LineTabs tabs={secondary} active={s} onChange={setS} />  {/* 视觉无法区分层级 */}

// ✅ 主导航走 Sidebar
<AdminSidebar>
  <SidebarItem href="/admin/agents">Agent</SidebarItem>
</AdminSidebar>

// ✅ 二级切换换 Segment（与外层 LineTabs 自然分层）
<LineTabs tabs={primary} active={p} onChange={setP} />
<SegmentGroup>
  <SegmentOption active={s === "a"} onClick={() => setS("a")}>子 A</SegmentOption>
  <SegmentOption active={s === "b"} onClick={() => setS("b")}>子 B</SegmentOption>
</SegmentGroup>
```

### 12.5 "即将开放"角标走 outline Badge，不要内嵌自定义胶囊

```tsx
// ❌ 在 LineTab 里手写一个浅色胶囊
<button className="px-4 py-3 inline-flex items-center gap-1.5">
  技能安装来源
  <span className="px-1.5 py-0.5 rounded-full bg-[#F3F4F6] text-[11px] text-[#6B7280]">
    即将开放
  </span>
</button>

// ✅ 复用 outline Badge（参考 badge.md §3）
<button className="px-4 py-3 inline-flex items-center gap-1.5">
  <BodyMedium tone="muted">技能安装来源</BodyMedium>
  <Badge variant="outline" className="px-1.5 py-0.5">即将开放</Badge>
</button>

// ✅ 或直接走 LineTabs comingSoon
<LineTabs
  tabs={[{ id: "source", label: "技能安装来源", comingSoon: true }]}
  active={tab} onChange={setTab}
/>
```
