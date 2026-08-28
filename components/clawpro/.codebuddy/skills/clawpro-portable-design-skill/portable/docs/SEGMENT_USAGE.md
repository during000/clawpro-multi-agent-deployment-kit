# Portable Segment 组件 - 使用指南

## 概述

Portable Segment 是 ClawPro 设计系统的可移植分段切换器组件，提供 **Admin（方角）** 和 **Tenant（胶囊）** 两个端别实现。

- **Admin**：4px 方角项 + 6px 圆角容器 + 半透明灰底 + 白底活跃项 + 无 outline
- **Tenant**：40px 胶囊项 + 80px 圆角容器 + 半透明灰底 + 白底活跃项 + 1px outline

---

## 核心原则

⚠️ **重要**：Admin 和 Tenant **绝不共用一套皮肤**。端别必须在**路由 / context 层**先确定，不要让组件内部自己判断。

```typescript
// ❌ 错误：端别污染
if (location.pathname.startsWith('/admin')) {
  return <PortableAdminSegment>...</PortableAdminSegment>
}
// ❌ 这样让组件内部自己判断端别，破坏单一真理源

// ✅ 正确：提前确定端别
const endpoint = useEndpoint(); // 从 context / router 获取端别
if (endpoint === 'admin') {
  return <PortableAdminSegment>...</PortableAdminSegment>
} else {
  return <PortableTenantSegment>...</PortableTenantSegment>
}
```

---

## 导入

```typescript
import {
  // Admin 受控组件
  PortableAdminSegment,
  PortableAdminSegmentItem,
  // Admin 非受控组件
  PortableAdminSegmentGroup,
  PortableAdminSegmentOption,
  // Tenant 受控组件
  PortableTenantSegment,
  PortableTenantSegmentItem,
  // Tenant 非受控组件
  PortableTenantSegmentGroup,
  PortableTenantSegmentOption,
} from "clawpro-portable/react";
```

同时在项目入口导入 CSS：

```typescript
// main.tsx 或 index.tsx
import "clawpro-portable/css/portable.css";
```

---

## 用法示例

### 1️⃣ Admin Segment（受控）

```tsx
import { useState } from "react";
import {
  PortableAdminSegment,
  PortableAdminSegmentItem,
} from "clawpro-portable/react";

export function AdminTabExample() {
  const [tab, setTab] = useState("overview");

  return (
    <div className="space-y-4">
      <PortableAdminSegment value={tab} onValueChange={setTab}>
        <PortableAdminSegmentItem value="overview">
          概览
        </PortableAdminSegmentItem>
        <PortableAdminSegmentItem value="detail">
          详情
        </PortableAdminSegmentItem>
        <PortableAdminSegmentItem value="setting">
          配置
        </PortableAdminSegmentItem>
      </PortableAdminSegment>

      {tab === "overview" && <div>概览内容</div>}
      {tab === "detail" && <div>详情内容</div>}
      {tab === "setting" && <div>配置内容</div>}
    </div>
  );
}
```

### 2️⃣ Admin SegmentGroup（非受控）

适合自管状态的场景（如筛选条、模式切换）：

```tsx
import { useState } from "react";
import {
  PortableAdminSegmentGroup,
  PortableAdminSegmentOption,
} from "clawpro-portable/react";

export function AdminFilterExample() {
  const [mode, setMode] = useState("all");

  return (
    <PortableAdminSegmentGroup>
      <PortableAdminSegmentOption
        active={mode === "all"}
        onClick={() => setMode("all")}
      >
        全部
      </PortableAdminSegmentOption>
      <PortableAdminSegmentOption
        active={mode === "group"}
        onClick={() => setMode("group")}
      >
        分组
      </PortableAdminSegmentOption>
      <PortableAdminSegmentOption
        active={mode === "custom"}
        onClick={() => setMode("custom")}
      >
        自定义
      </PortableAdminSegmentOption>
    </PortableAdminSegmentGroup>
  );
}
```

### 3️⃣ Tenant Segment（受控）

```tsx
import { useState } from "react";
import {
  PortableTenantSegment,
  PortableTenantSegmentItem,
} from "clawpro-portable/react";

export function TenantTabExample() {
  const [filter, setFilter] = useState("all");

  return (
    <div className="space-y-4">
      <PortableTenantSegment value={filter} onValueChange={setFilter}>
        <PortableTenantSegmentItem value="all">
          全部
        </PortableTenantSegmentItem>
        <PortableTenantSegmentItem value="active">
          已激活
        </PortableTenantSegmentItem>
        <PortableTenantSegmentItem value="inactive">
          已禁用
        </PortableTenantSegmentItem>
      </PortableTenantSegment>

      <div className="p-4 bg-gray-50 rounded">
        {filter === "all" && "显示所有项"}
        {filter === "active" && "显示已激活项"}
        {filter === "inactive" && "显示已禁用项"}
      </div>
    </div>
  );
}
```

### 4️⃣ Tenant SegmentGroup（非受控）

```tsx
import { useState } from "react";
import {
  PortableTenantSegmentGroup,
  PortableTenantSegmentOption,
} from "clawpro-portable/react";

export function TenantDateRangeExample() {
  const [range, setRange] = useState("day");

  return (
    <PortableTenantSegmentGroup>
      <PortableTenantSegmentOption
        active={range === "day"}
        onClick={() => setRange("day")}
      >
        日
      </PortableTenantSegmentOption>
      <PortableTenantSegmentOption
        active={range === "week"}
        onClick={() => setRange("week")}
      >
        周
      </PortableTenantSegmentOption>
      <PortableTenantSegmentOption
        active={range === "month"}
        onClick={() => setRange("month")}
      >
        月
      </PortableTenantSegmentOption>
    </PortableTenantSegmentGroup>
  );
}
```

---

## Props

### PortableAdminSegment / PortableTenantSegment

受控组件容器，与 `...Item` 搭配使用。

| 属性 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `value` | `string` | ✅ | 当前活跃项的 value |
| `onValueChange` | `(value: string) => void` | ✅ | 项变化回调 |
| `children` | `React.ReactNode` | ✅ | 子元素（通常是 `...Item`） |
| `className` | `string` | ❌ | 额外 className（Tailwind） |

### PortableAdminSegmentItem / PortableTenantSegmentItem

受控组件项。

| 属性 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `value` | `string` | ✅ | 唯一标识符 |
| `children` | `React.ReactNode` | ✅ | 显示文本 |
| `disabled` | `boolean` | ❌ | 是否禁用（默认 false） |
| `className` | `string` | ❌ | 额外 className |
| `onClick` | `() => void` | ❌ | 点击额外回调 |

### PortableAdminSegmentGroup / PortableTenantSegmentGroup

非受控组件容器。

| 属性 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `children` | `React.ReactNode` | ✅ | 子元素（通常是 `...Option`） |
| `className` | `string` | ❌ | 额外 className |

### PortableAdminSegmentOption / PortableTenantSegmentOption

非受控组件项。

| 属性 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `active` | `boolean` | ✅ | 是否为活跃态 |
| `onClick` | `() => void` | ✅ | 点击回调 |
| `children` | `React.ReactNode` | ✅ | 显示文本 |
| `disabled` | `boolean` | ❌ | 是否禁用（默认 false） |
| `className` | `string` | ❌ | 额外 className |

---

## 视觉规范

### Admin Segment

**容器**
- 圆角：`6px`
- 背景：`rgba(219, 221, 228, 0.32)`（半透明灰）
- 高度：`36px`
- 内边距：`2px`

**项（Item）**
- 圆角：`4px`
- 内边距：`4px 16px`（`py-1 px-4`）
- 字号：`14px / Normal`
- 字高：`20px`

**状态**

| 状态 | 背景 | 文字 | 字重 | 阴影 |
|-----|------|------|------|------|
| Default | 透明 | `#7B818F` | 400 | 无 |
| Hover | 透明 | `#4B5563` | 400 | 无 |
| Active | `#FFFFFF` | `#020617` | 600 | `0px 1px 2px rgba(0,0,0,0.05)` |
| Disabled | 透明 | `#D3D6DB` | 400 | 无 |
| Focus-visible | - | - | - | `ring-[3px] ring-[#355EF1]/20` |

### Tenant Segment

**容器**
- 圆角：`80px`（完全胶囊）
- 背景：`rgba(219, 221, 228, 0.32)`（半透明灰）
- 高度：`36px`
- 内边距：`0`（项自带 padding）

**项（Item）**
- 圆角：`40px`（胶囊）
- 内边距：`4px 12px`（`py-1 px-3`）
- 字号：`14px`
- 字高：`22px`
- 字距：`0.005em`

**状态**

| 状态 | 背景 | 文字 | 字重 | Outline | 阴影 |
|-----|------|------|------|---------|------|
| Default | 透明 | `#334155` | 400 | 无 | 无 |
| Hover | 透明 | `#020617` | 400 | 无 | 无 |
| Active | `#FFFFFF` | `#020617` | 500 | `1px #CDD4DC` | `0px 1px 4px rgba(0,0,0,0.05)` |
| Disabled | 透明 | `#CBD5E1` | 400 | 无 | 无 |
| Focus-visible | - | - | - | - | `ring-[3px] ring-[#355EF1]/20` |

---

## 禁用项

```tsx
// 使用 disabled prop
<PortableAdminSegmentItem value="locked" disabled>
  已锁定
</PortableAdminSegmentItem>

// 或非受控的
<PortableAdminSegmentOption active={false} disabled onClick={() => {}}>
  已锁定
</PortableAdminSegmentOption>
```

---

## CSS 原生用法（HTML / Vanilla JS）

如果不用 React，直接用 HTML + CSS：

```html
<!-- Admin 方角 -->
<div class="cp-segment cp-segment--admin" role="tablist">
  <button
    class="cp-segment__item cp-segment__item--active"
    type="button"
    role="tab"
    aria-selected="true"
  >
    概览
  </button>
  <button class="cp-segment__item" type="button" role="tab">
    详情
  </button>
</div>

<!-- Tenant 胶囊 -->
<div class="cp-segment cp-segment--tenant" role="tablist">
  <button
    class="cp-segment__item cp-segment__item--active"
    type="button"
    role="tab"
    aria-selected="true"
  >
    全部
  </button>
  <button class="cp-segment__item" type="button" role="tab">
    分组
  </button>
</div>
```

只需在 CSS 中导入：

```css
@import "clawpro-portable/css/segment.css";
```

---

## 常见场景

### ✅ 适用场景

1. **卡片内部分区切换**（概览 / 详情 / 配置）
2. **表格/列表上方的筛选条**（"全部 / 分组"、"日 / 周 / 月"）
3. **弹窗内的局部模式切换**
4. **工具栏内的视图模式切换**（List / Grid）

### ❌ 不适用场景

1. **页面标题下方的一级 Tab** → 使用 `PortableLineTabs`（参考 `tabs.md`）
2. **顶部主导航** → 使用 `PortableAdminSidebar`（Admin）或自定义顶部导航（Tenant）
3. **分页** → 使用 `PortablePagination`
4. **单个开关** → 使用 `PortableSwitch`
5. **多选筛选** → 使用 `PortableSearchFilterBar`

---

## 最佳实践

| ✅ 推荐 | ❌ 禁止 |
|--------|--------|
| 项数 ≤ 5 | 项数 > 5（改用侧边栏或分页） |
| Admin 方角 / Tenant 胶囊 | 混用两套皮肤 |
| 提前在路由层确定端别 | 在组件内部自己判断端别 |
| Active 同时有背景 + 阴影（±outline） | 仅靠文字颜色微差表示活跃 |
| 关键操作需要禁用时用 `disabled` | 用 CSS `opacity-50` 伪装禁用 |
| 非受控场景用 `SegmentGroup` + `SegmentOption` | 自管状态但硬用 `Segment` + `SegmentItem` |

---

## 文件结构

```
portable/
├── css/
│   ├── segment.css           ← 样式文件
│   ├── tokens.css            ← 全局设计 token
│   ├── portable.css          ← 总入口（已导入 segment.css）
│   └── ...
└── react/
    ├── segment.tsx           ← React 组件
    ├── index.ts              ← 导出入口（已导出）
    └── ...
```

---

## 参考规范

详见 `component-specs/segment.md`：
- §3 视觉规范（Admin / Tenant）
- §5 状态机
- §7 Portable Fallback
- §9 Do / Don't
- §12 代码对照（正确用法）

---

## Troubleshooting

**Q: 怎样导入 CSS？**

A: 在项目入口（如 `main.tsx`）添加：

```typescript
import "clawpro-portable/css/portable.css";
```

**Q: Admin 和 Tenant 怎样切换？**

A: 不是用 className 切换，而是在路由/context 层先确定端别，然后选用不同的组件：

```typescript
// ❌ 错误
className={isAdmin ? "admin-segment" : "tenant-segment"}

// ✅ 正确
{isAdmin ? <PortableAdminSegment /> : <PortableTenantSegment />}
```

**Q: 如何禁用某个项？**

A: 使用 `disabled` prop：

```typescript
<PortableAdminSegmentItem value="x" disabled>
  禁用项
</PortableAdminSegmentItem>
```

**Q: 能自定义样式吗？**

A: 可以传 `className`，但建议**不要修改核心样式**（如圆角、高度、颜色），因为这破坏了端别的单一真理源。如有风格差异需求，请在 `component-specs/segment.md` 中反馈。

---

## 更新日志

- **v1.0.0**（2026-06-11）初始版本
  - Admin Segment（方角）
  - Tenant Segment（胶囊）
  - 受控组件（Item）+ 非受控组件（Option）
  - 完整 CSS + React 实现
