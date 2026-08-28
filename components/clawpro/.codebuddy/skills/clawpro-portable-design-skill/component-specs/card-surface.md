# Card / Surface

> **Showcase mapping**: `surface-card` · `surface-inner` · `surface-config` · `surface-overlay` · `tenant-card` · `card`（`client/src/pages/DesignSystemComponents.tsx`）

## 1. Purpose

- 用于定义 ClawPro 页面里的卡片层级，而不是让页面继续用 `div + bg-[var(--cp-surface)] + shadow` 自由拼。
- 核心问题不是“有没有卡片”，而是 Admin 与 Tenant 的卡片语义不同。

## 2. Scope

- 适用端：Shared / Admin / Tenant
- 必用场景：页面主卡、统计卡、配置卡、Tenant 业务列表卡、空页面承载容器
- 不适用场景：纯文本段落、极轻量行内分组、表格内普通单元格

## 3. Visual Standard

| Surface | Radius | Border | Shadow | Scope |
|---|---|---|---|---|
| SurfaceCard | 4px | `--cp-border` | 默认无阴影；hover / selected 才有极轻阴影 | Admin / Shared |
| SurfaceInner | 4px | `--cp-border` | 无阴影 | 卡内二级分组 |
| SurfaceOverlay | 4px | `--cp-border` | `--cp-shadow-overlay` | 浮层 |
| SurfaceConfig | 4px | `--cp-border` | 默认无阴影；确需强调时显式加 `--shadow-config` | Admin 配置强调块 |
| TenantCard | 12px | normal `--border`，static `--cp-border`，hover 透明描边 | normal 使用 `--shadow-tenant-card`；hover 增强；static 无阴影 | Tenant 业务卡 |

### 3.1 子元素字号 / 色 token（改容器必贯彻，P3）

> ⚠️ 改卡片容器（圆角 / 边框 / 阴影 / padding）时**必须同时核对卡内子元素走语义 token**，不得只换外层、内部文字仍停在宿主仓旧字号 / 旧色：
>
> | 子元素 | 字号 / 字重 | 颜色 token |
> |---|---|---|
> | 卡片标题 | 走 Typography 标题语义 | `var(--cp-text-title)` |
> | 卡内正文 | `text-sm`（14px） | `var(--cp-text-body)` |
> | 辅助 / 说明 / 时间 | `text-xs`（12px） | `var(--cp-text-muted)` / `var(--cp-text-weak)` |
> | 卡内表格 | 见 `table.md`（全表 12px） | 同 `table.md` |
>
> 一律走 token，不在卡内散写 `text-[#xxx]` / `text-gray-*` / 自定字号。

## 4. Anatomy

```text
SurfaceCard / TenantCard
  Header optional
  Body
  Footer optional

SurfaceInner
  Nested content only
```

## 5. States

- Admin SurfaceCard default: 白底、4px、蓝灰描边、默认无阴影。
- Admin SurfaceCard hover: 浅蓝灰描边、极轻阴影、微抬。
- Admin SurfaceCard selected: 品牌蓝描边、极轻阴影。
- Admin SurfaceConfig default: 4px、蓝灰描边、默认无阴影；确需强调时显式加 `--shadow-config`。
- TenantCard normal: 12px、浅描边、`--shadow-tenant-card` 单层阴影。
- TenantCard hover: 描边消失，阴影增强，轻微上浮。
- TenantCard static: 无阴影，纯展示。

## 6. Demo Repo Usage

- 当前 demo 仓组件：`client/src/components/ui/Surface.tsx`
- 典型 Admin 页面：`client/src/pages/admin/BasicInfo.tsx`
- 典型 Tenant 语义：`TenantCard` 供用户端业务卡使用

```tsx
import { SurfaceCard, SurfaceInner, TenantCard } from "@/components/ui/Surface";

<SurfaceCard className="p-6" hover>
  <h3>管理端列表卡</h3>
</SurfaceCard>

<SurfaceInner className="p-4">
  <p>卡片内二级分组</p>
</SurfaceInner>

<TenantCard interactive>
  <h3>客户端业务卡</h3>
  <p>12px 圆角，三态</p>
</TenantCard>
```

## 7. Portable Fallback

### 7.1 If host repo already has Card components

- Admin 场景可继续用宿主仓现有 Card，但把圆角压到 4px，描边统一到 `--cp-border`，默认不加阴影，不要沿用宿主仓大圆角和重阴影。
- Tenant 场景如果宿主仓只有通用 Card，必须额外加一个 12px Tenant 变体或等效 class。

### 7.2 Minimal React fallback

```tsx
export function PortableSurfaceCard({ children }: { children: React.ReactNode }) {
  return (
    <section className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-6">
      {children}
    </section>
  );
}

export function PortableTenantCard({ children }: { children: React.ReactNode }) {
  return (
    // allow-radius: Tenant business card requires 12px in portable fallback.
    <section data-note="allow-radius" className="rounded-[12px] border border-[var(--border)] bg-[var(--cp-surface)] p-5 shadow-[var(--shadow-tenant-card)]">
      {children}
    </section>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

见 `portable/html-css/card.html`。

## 8. Migration Rules

- 旧写法：`div.bg-white.rounded-xl.shadow-md` 之类的手写卡片。
- 新口径：先判断端，再映射到 Admin Surface 或 TenantCard。
- 可以暂时兼容：宿主仓原有 Card 组件，但必须做视觉覆写。
- 不允许新增：Admin 页面继续加 12px 业务卡；Tenant 页面继续机械用 4px 主业务卡。

## 9. Do / Don't

Do:

- 先判断这是 Admin 还是 Tenant 卡片。
- 卡片内二级分组用 `SurfaceInner` 逻辑，不再套多层大卡。
- 配置引导块和普通列表卡分语义。

Don't:

- 不要把所有卡片都做成一种圆角。
- 不要继续用 Tailwind 重阴影冒充层级。
- 不要在页面里新增大量手写 boxShadow。

## 10. QA Checklist

- [ ] 已明确区分 Admin 4px 与 Tenant 12px
- [ ] 没有新增手写重阴影卡片
- [ ] 配置卡、普通卡、内嵌卡语义清楚
- [ ] fallback 在宿主仓也能成立

## 11. References

- Demo code: `client/src/components/ui/Surface.tsx`
- Demo page: `client/src/pages/admin/BasicInfo.tsx`
- Related rules: `references/foundation.md`
- Related rules: `references/tenant.md`

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2.2 同口径。Admin 4px Surface 与 Tenant 12px Card 的端别分流。

### 12.1 不要手写 div + bg + shadow 拼卡片

```tsx
// ❌ 自由拼卡片，跨页风格漂移
<div className="bg-white rounded-xl shadow-md p-6">
  <h3>策略列表</h3>
</div>

// ✅ Admin / Shared：用 SurfaceCard
<SurfaceCard className="p-6">
  <h3>策略列表</h3>
</SurfaceCard>

// ✅ Tenant 业务对象：用 TenantCard
<TenantCard interactive>
  <h3>我的 Agent</h3>
</TenantCard>
```

### 12.2 端别圆角分流（Admin 4px / Tenant 12px）

```tsx
// ❌ Admin 配置卡用了 Tenant 12px 圆角
<div className="rounded-[12px] border border-[var(--cp-border)] p-6">
  <h3>基础信息</h3>
</div>

// ❌ Tenant 业务卡用了 Admin 4px（视觉太硬）
<SurfaceCard className="p-5">
  <h3>我的 Agent</h3>
</SurfaceCard>

// ✅ Admin：4px
<SurfaceCard className="p-6">
  <h3>基础信息</h3>
</SurfaceCard>

// ✅ Tenant：12px
<TenantCard interactive>
  <h3>我的 Agent</h3>
</TenantCard>
```

### 12.3 卡片二级分组用 SurfaceInner，不嵌套大卡

```tsx
// ❌ 卡内套卡，制造 5 层视觉景深
<SurfaceCard className="p-6">
  <SurfaceCard className="p-4 mb-4">
    <h4>子模块 A</h4>
  </SurfaceCard>
  <SurfaceCard className="p-4">
    <h4>子模块 B</h4>
  </SurfaceCard>
</SurfaceCard>

// ✅ 二级分组用 SurfaceInner（视觉更轻、不重复主卡阴影）
<SurfaceCard className="p-6 space-y-4">
  <SurfaceInner className="p-4">
    <h4>子模块 A</h4>
  </SurfaceInner>
  <SurfaceInner className="p-4">
    <h4>子模块 B</h4>
  </SurfaceInner>
</SurfaceCard>
```

### 12.4 不要重阴影冒充层级

```tsx
// ❌ Admin 列表卡硬塞重阴影
<SurfaceCard className="p-6 shadow-2xl">…</SurfaceCard>

// ❌ Tenant 卡片再叠一层 Tailwind 阴影
<TenantCard className="shadow-lg" interactive>…</TenantCard>

// ✅ Admin 默认无阴影，需要时用 SurfaceCard hover prop
<SurfaceCard className="p-6" hover>…</SurfaceCard>

// ✅ Tenant 阴影由组件自带 token 控制（normal / hover / static），不外加
<TenantCard interactive>…</TenantCard>
```

### 12.5 TenantCard 三态语义（normal / hover / static）

```tsx
// ❌ 把展示型 Tenant 卡片错配成 interactive
<TenantCard interactive>
  {/* 这是只读详情，没有 hover 语义 */}
  <h3>当前配额</h3>
  <p>每月 1000 次</p>
</TenantCard>

// ✅ 仅展示用 static（无阴影 / 不响应 hover）
<TenantCard static>
  <h3>当前配额</h3>
  <p>每月 1000 次</p>
</TenantCard>

// ✅ 业务卡可点击 → interactive（normal 阴影 + hover 增强）
<TenantCard interactive onClick={enterDetail}>
  <h3>我的 Agent</h3>
</TenantCard>
```
