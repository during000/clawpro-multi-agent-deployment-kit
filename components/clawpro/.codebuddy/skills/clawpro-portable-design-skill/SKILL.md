---
name: clawpro-portable-design-skill
user-invocable: false
description: >
  设计到代码的可执行交付包（Admin / 后台管理 / 非客户端场景）。包含设计规范、组件标准、跨仓 fallback 实现、迁移映射。
  当任务涉及"做个列表页""适配换皮""页面设计审查""组件规范评审""跨仓设计落地""Portable Fallback 实现"时必须加载。
  特别地，当涉及新建或修改管理端页面（列表 / 表单 / 详情 / 看板 / 设置 / 空态 / 错误）、
  生成或审查组件（Surface / Card / Button / Table / Form / Input / Select / SearchableSelect / DatePicker / Empty /
  Dialog / Sheet / Popover / Sidebar / Pagination / BatchActions / SearchFilterBar / Upload / Tabs / Segment /
  Badge / Status / Chart / Stat）、走查设计 Token 与色彩体系、实现宿主仓 fallback 时，必须加载本 skill。
  **如果是客户端（Tenant / 用户端）场景请改加载 `references/tenant.md`**（UI 规范有差异）。
  关键词：列表页、表单页、详情页、管理后台、设计规范、token 化、可移植、fallback、换皮、
  组件标准、视觉对齐、跨仓、Admin 端、管控端、brand blue #1447E6、4px 圆角。

# 可移植设计规范（管控端 / 非客户端场景）

> **技术栈**：React（纯）· Tailwind CSS · 原生 HTML/CSS  
> **核心特点**：零库依赖、可跨仓复用、设计 ↔ 代码 100% 同步  
> **开发环境**：Vite + TypeScript（可选）  
> **库选择**：
>   - 路由：wouter 或 React Router
>   - 图表：recharts
>   - Toast：sonner
>   - 动画：framer-motion（可选）
>   - 图标：lucide-react
>   - 组件：纯 React Hooks + Tailwind（不依赖 shadcn/ui）

> **AI 操作手册**：先读完本文件，按 §3 Workflow 执行，遇到细节问题再钻进 `references/*` 与 `component-specs/*`。
> 不要把本 skill 当资料库一篇篇读完才开始写。

---

## 0. Scope / 场景路由（必读，先判场景再写代码）

管理端 UI 规范按"场景"分流，**先判场景，再读规则**。AI 在生成/审查任何页面之前，必须先回答：这是客户端（Tenant）还是非客户端（管理端 / Landing）？

| 场景 | 触发条件（任一命中） | 必须加载的 Skill |
|---|---|---|
| **客户端（Tenant）** | 路径 `client/src/pages/tenant/**`、`client/src/components/topnav/**`；被 `<TenantLayout>` 包裹；用户端共享业务组件（Agent / 技能 / 模型额度等卡片） | **`references/tenant.md`**（用户端差异化规则：12px 卡片圆角 / 顶部导航透明模糊 / 按钮 variant=tenant-* / 搜索胶囊全圆角等） |
| **非客户端（管控端 Admin）** | 路径 `client/src/pages/admin/**`；被 `<AdminLayout>` / `<AdminSidebar>` 包裹；Admin 后台、配额、审计、租户、模型供应、通道、技能管理等管控页 | **本文件 `SKILL.md`**（默认场景） |
| **Landing / 营销 / 文档站** | `client/src/pages/landing/**`、独立营销页 | 本文件 + `references/landing.md` |
| **跨场景共享基础组件** | `client/src/components/ui/**`（Button / Input / Table / Empty 等基础组件源码） | 本文件 + `component-specs/*.md` |

### 0.1 非客户端（管控端）场景默认值（与客户端的关键差异）

> 下表是本 skill 默认锁定的视觉值，**未经设计师拍板不得在管控端引用 Tenant 规则**。

| 维度 | 管控端（本 skill 默认） | 客户端（去 `references/tenant.md`）|
|---|---|---|
| 按钮 / Input / Select / Dialog / 卡片 / SurfaceInner / Tabs 圆角 | **统一 4px**（`--radius-lg`） | 普通 4px / 卡片 12px / 搜索胶囊 full |
| 主操作按钮 variant | `claw-primary` / `claw-outline` | `tenant-primary` / `tenant-outline` |
| 卡片 | `SurfaceCard`（细描边 + 弱阴影） / `SurfaceConfig`（0.5px 描边高亮） | `TenantCard`（12px 圆角 + 业务对象语义） |
| 顶部导航 | `AdminSidebar`（左侧渐变选中态） | 顶部 `TopNav`（透明 + 背景模糊） |
| 装饰背景 | 右侧渐变背景图（详见 `references/admin.md`） | 已移除点阵装饰 |
| 头像 / 状态点 / Switch 圆角 | full | full（一致） |

**铁律**：

1. 在管控端**任何**地方写 `rounded-xl` / `rounded-2xl` / `rounded-[8px]` / `rounded-[12px]` / `rounded-full`（除头像 / 状态点 / Switch / 标签胶囊外）= 直接违规，回到 `--radius-lg`（4px）。
2. 看到 Figma 标注是 12px 圆角的卡片但当前页是管控端 → **不要照抄**，回设计师确认或落到 4px；反之看到 4px 但页面是客户端，也不要照搬。
3. 不要在管控端用 `tenant-primary` / `TenantCard`，反之亦然。

---

## 0.2 🔧 集成前置清单（在生成任何 Admin 页面前必读！）

**问题**：AI 生成的 Admin 页面可能看起来"样式错乱"（左侧导航背景是灰色，按钮圆角不对，页面没有浅蓝背景）。原因是**缺少 CSS 变量和主题配置**。

**解决**：生成页面时确保以下 3 项：

| 检查项 | 代码示例 | 验证方法 |
|---|---|---|
| **1. 导入 CSS 变量** | 在生成的页面顶部<br>`import "@/index.css"` | 浏览器 DevTools Console 执行：<br>`getComputedStyle(document.documentElement).getPropertyValue('--admin-sidebar-bg')` 有输出 ✓ |
| **2. AdminLayout 包含 admin-theme 类** | 应自动有<br>`<div className="... admin-theme">` | 检查 AdminLayout.tsx 第 326 行 ✓ |
| **3. CSS 变量完整集合** | 项目 `client/src/index.css` 中必须定义所有 `--admin-sidebar-*` 和 `--page-bg` | 运行：`grep "admin-sidebar-bg" client/src/index.css` 有输出 ✓ |

**如果以上任一项缺失**，组件会**降级到浏览器默认样式**，导致：
- ❌ 左侧导航背景变灰（而非 `#ffffff`）
- ❌ 导航项间距 / 圆角 / 激活态样式不生效
- ❌ 页面背景没有浅蓝色

**快速修复**：
```typescript
// pages/my-admin-page.tsx
import "@/index.css"  // ← 必须在顶部
import { AdminLayout } from "@/components/AdminLayout"

export default function AdminMonitor() {
  return (
    <AdminLayout>
      {/* 页面内容 */}
    </AdminLayout>
  )
}
```

详见 `component-specs/admin-sidebar.md` §2.1 前置集成清单。

---

## 1. Principles 顶层原则

1. **可移植优先**：本 skill 不假设宿主仓一定有当前 demo 仓的组件、token、目录结构。每个产物都要回答"宿主仓没有怎么办"。
2. **组件不 markup**：能用业务组件就不用 `div + className` 拼视觉。Alert / Empty / Surface / Toast / Separator / Skeleton / Badge 都有专属语义组件。
3. **Token 不硬编码**：颜色、圆角、阴影、间距走 `--cp-*` token 或语义类，不写 `#1447E6` / `border-gray-200` / `rounded-xl` 之类的字面值。
4. **场景分流（先判场景再选组件）**：客户端（Tenant）与非客户端（管控端 Admin / Landing）在圆角、阴影、按钮 variant、卡片层级上有强差异。本文件默认是**管控端**场景；客户端必须切到 `references/tenant.md`，禁止在管控端引用 Tenant 规则（详见 §0 Scope）。
5. **冲突回到设计师**：spec 与产品需求冲突，或 token / variant / 圆角拍板模糊时，记录到 `references/conflict-log.md` 并标 `needs-design-confirmation`，**不由 AI 私自裁决**。

### 1.5 冲突处理的铁律

**核心原则**：
> 当 Spec 与代码现状冲突、或 Token / Variant / 圆角 / 图标槽位等设计拍板模糊时，
> **必须停下来回到设计师**，而不由 AI 或工程师私自裁决。

**冲突的三个来源**：
1. **规范与现实冲突** — Spec 说 A，但宿主仓已实现 B
   - 例：Spec 要求 4px 圆角，但宿主仓按钮都是 8px
   - 处理：标 `needs-design-confirmation`，交设计师裁决"遵 spec 还是保持现状"
   - 不允许：AI 私自选择用 8px（"为了兼容"）

2. **业务需求与规范冲突** — 产品要求超出当前组件范围
   - 例：产品要求 StatusTag 增加"待处理"状态，但 Spec 只有 4 个状态
   - 处理：记录到 `references/conflict-log.md`，交设计师评估"新增状态或用 Badge"
   - 不允许：AI 直接给 StatusTag 加"待处理"（规范逃逸）

3. **设计选择模糊** — 两个选项都"看起来没问题"，但有隐藏成本
   - 例：图标显示为"圆形渐变"还是"方形扁平"，都符合规范
   - 处理：列出两个方案的成本（维护/扩展/学习成本），交设计师定夺
   - 不允许：AI 凭"个人偏好"选一个

**冲突记录流程**：
1. AI / 工程师遇到冲突，先在 `references/conflict-log.md` 记录：
   ```markdown
   ## 冲突 #N — [简述冲突是什么]
   - 位置：文件名 + 代码位置
   - 冲突方 A：[Spec / 现状 / 需求] 说什么
   - 冲突方 B：[另一方] 说什么
   - 成本分析：A 方成本 vs B 方成本
   - 需要设计师决策：【选项】/ 【时间框】
   ```

2. 设计师 review 并在冲突记录后加：
   ```markdown
   ## ✅ 设计师裁决（日期 | 设计师名字）
   - 选择：A / B / 其他
   - 原因：为什么这样选
   - 后续：后续怎样避免这类冲突
   ```

3. 根据裁决执行改动，并标注 `Ref: conflict-log.md #N`

**不允许的做法**：
- ❌ "为了快速交付"就私自裁决
- ❌ "这样写对用户更好"就违反规范
- ❌ "现状已经这样了"就放弃规范对齐
- ❌ 对模糊的图标槽位"选一个看起来靠谱的 lucide 图标"而不标 needs-design-confirmation

**允许的做法**：
- ✅ 冲突时立刻标记，让设计师在下一个 review 周期统一处理
- ✅ 对图标槽位无候选时，标 needs-design-confirmation，用临时占位符，等设计确认
- ✅ 遇到新需求超过规范范围时，先补规范再落地实现

---

## 2. Critical Rules（带 ✅/❌ 对照）

### 2.1 颜色与 token

```tsx
// ❌ 硬编码品牌色 / 灰阶
<div className="border-gray-200 bg-white text-[#1447E6]" />

// ✅ 走 token
<div className="border-[var(--cp-border)] bg-[var(--cp-surface)] text-[var(--cp-brand-blue)]" />
```

详见 `references/foundation.md` §2 / §4 与 `tokens/*`。

### 2.2 卡片与层级

```tsx
// ❌ 手写卡片
<div className="bg-white rounded-xl shadow-md p-6">...</div>

// ✅ 用 Surface 层级（Admin）
<SurfaceCard>...</SurfaceCard>

// ✅ Tenant 业务对象
<TenantCard>...</TenantCard>
```

详见 `references/components.md` §2 与 `component-specs/card-surface.md`。

### 2.3 按钮 variant

```tsx
// ❌ 业务次级按钮借用 shadcn outline 伪装
<Button variant="outline">取消</Button>

// ✅ 用分端语义 variant
<Button variant="claw-primary">创建</Button>      {/* Admin 主操作 */}
<Button variant="tenant-primary">创建</Button>    {/* Tenant 主操作 */}
<Button variant="claw-outline">取消</Button>      {/* Admin 次级 */}
<Button variant="destructive">删除</Button>
```

`outline` 仅留给 SearchableSelect / Popover trigger 等控件外壳并打 `allow-shadcn-outline` 注释。

### 2.4 圆角（管控端默认 4px，客户端请切 references/tenant.md）

> **管控端铁律**：本 skill 适用的非客户端场景下，**几乎所有面板类元素圆角统一 4px**，不存在 8px / 12px / 16px / `rounded-xl` 这些值。看到 Figma 标 12px 但当前页是管控端 → 不要照抄，落到 4px 或回设计师确认。

| 端 | 按钮 / Input / Select / Dialog / Drawer | 卡片 / SurfaceInner / Tabs 容器 | 头像 / 状态点 / Switch / 标签胶囊 |
|---|---|---|---|
| **管控端 Admin（本 skill）** | **4px** (`--radius-lg`) | **4px** | full |
| **客户端 Tenant**（→ `references/tenant.md`） | 4px（普通）/ full（搜索胶囊） | 12px（业务卡）/ 4px（控件类） | full |

```tsx
// ❌ 在管控端硬写非 4px 圆角
<div className="rounded-xl bg-[var(--cp-surface)]" />
<button className="rounded-[12px]">...</button>

// ✅ 管控端走 token，永远 4px
<div className="rounded-[var(--radius-lg)]" />
<Button variant="claw-primary">创建</Button>      {/* 内置 4px */}
<SurfaceCard>...</SurfaceCard>                   {/* 内置 4px */}
```

> **唯一例外 · hero 装饰玻璃元素**：命中 `component-specs/dark-veil.md` §0 的**开通页 / 能力 hero** 内，浮在动态背景上的**玻璃拟态装饰元素**（图标徽章 / 权益小卡，如 `rounded-[8px]` / `rounded-[9px]`）允许非 4px 圆角，因其是装饰而非标准数据面板。**例外严格限于 hero 区装饰元素**；hero 以外的任何数据面板（`SurfaceCard` / `SurfaceInner` / 表格 / 弹窗 / 表单）一律 4px，不得借此例外扩散。裁决见 `references/conflict-log.md` C-018、用法见 `references/admin-cloud-dev-activation.md` §6。

### 2.5 文字层级

```tsx
// ❌ 散落定义文字色
<h2 className="text-[#0F172A] text-lg font-semibold">实例列表</h2>
<p className="text-gray-500">最近 7 天</p>

// ✅ 用 Typography 语义
<SectionTitle>实例列表</SectionTitle>
<MetaText>最近 7 天</MetaText>
```

### 2.6 空状态

```tsx
// ❌ 拼接空态
<div className="text-center py-12">
  <span className="text-3xl">📭</span>
  <p>暂无数据</p>
</div>

// ✅ 用 Empty 组件家族（页面级）
<Empty>
  <EmptyMedia>{/* 已登记插画 100×100 */}</EmptyMedia>
  <EmptyHeader>
    <EmptyTitle>还没有创建任何 Agent</EmptyTitle>
    <EmptyDescription>从公共技能库或企业技能库添加，或<a>查看文档</a></EmptyDescription>
  </EmptyHeader>
  <EmptyContent><Button variant="tenant-primary">创建 Agent</Button></EmptyContent>
</Empty>
```

### 2.7 间距与成组

```tsx
// ❌ 给每个控件单独 margin
<Input className="mr-3" /><Select className="mr-3" /><Button />

// ✅ 用 flex + gap 成组
<div className="flex items-center gap-3"><Input/><Select/><Button/></div>
```

### 2.8 图标（核心规则：先判槽位，再选图）

> **图标选择的铁律**：先判是否命中 9 个不可回退槽位 → 
> 命中时**禁 lucide**，只能用 `resource-skill-map.json` 的候选 SVG → 
> 无合适候选时标 `needs-design-confirmation` 提交设计，不能回退 lucide。

**9 个不可回退槽位**（详见 `references/assets-icons.md §5.5` 和 ADR）：
- `number-card` — 18×18 渐变图标（22 枚候选）
- `status-tag` / `badge` — 运行/成功/异常等状态（候选自定义）
- `card-left-icon` — 卡片左侧装饰（候选自定义）
- `run-status-indicator` — 在线/离线状态点（2 枚）
- `feature-card` — 功能卡上方标题图标（候选自定义）
- 其他 4 个…（见 `docs/ClawPro资源库-阶段9决策溯源(ADR).md`）

```tsx
// ❌ emoji / inline SVG
<span>📤</span>

// ✅ 第一步：先判是否命中不可回退槽位
import { isForbiddenLucideSlot } from "@/utils/icon-registry";

if (isForbiddenLucideSlot(iconSlot)) {
  // ✅ 命中 → 必须用 resource-skill-map SVG
  import { someIcon } from "@/assets/icon-registry.json";
  <someIcon className="size-5 text-[var(--cp-text-weak)]" />
} else {
  // ✅ 未命中 → 可用 lucide-react
  import { Upload } from "lucide-react";
  <Upload className="size-4 text-[var(--cp-text-weak)]" />
}

// ❌ 若不可回退槽位无合适候选，禁止这样做：
<Upload /> // 违规！这是 lucide，会被脚本拦截

// ✅ 正确做法：标记待设计确认
<Icon 
  /* TODO: needs-design-confirmation - number-card 槽位需要"上升趋势"图标 */ 
  as={TrendingUpIcon}  // 临时占位
/>
```

### 2.9 KPI 概览 / 数字统计卡（自动识别 → 必须 NumberCard）

> **触发条件（满足任一即视为命中）**：
> 1. 设计稿/需求出现「**图标 + 标题 + 大号数字**」三件套（无图表），无论横排几张；
> 2. 文案是 KPI / 概览 / 监控 / 配额 / 实例数 / Tokens / 请求数 / 用量 / 余额 / 累计 等指标；
> 3. 数字字号明显大于正文（≥ 20px），常带千分位 / 百分号 / 单位。
>
> **强制行为**：一律用 `<NumberCard>`，禁止用 `SurfaceCard + 内联 SVG + StatNumber` / 任意 `<div>` 自拼。
> 详细规范见 `component-specs/number-card.md`。

```tsx
// ❌ 手搓 KPI 卡
<div className="rounded-md border p-5">
  <div className="flex items-center gap-2">
    <DownloadIcon className="size-[18px]" />
    <span className="text-sm font-medium">输入 Tokens</span>
  </div>
  <strong className="text-2xl font-semibold tabular-nums">1,234</strong>
</div>

// ✅ NumberCard（自动套 SurfaceCard p-5 + 18×18 渐变图标 + StatNumber）
import {
  NumberCard, InputTokensIcon, OutputTokensIcon, TotalTokensIcon,
} from "@/components/ui/number-card";

<div className="grid grid-cols-3 gap-5">
  <NumberCard icon={<InputTokensIcon />}  label="输入 Tokens" value="1,234" />
  <NumberCard icon={<OutputTokensIcon />} label="输出 Tokens" value="5,678" />
  <NumberCard icon={<TotalTokensIcon />}  label="总 Tokens"   value="6,912" />
</div>
```

### 2.10 导航结构（Admin / Tenant 框架级 → 禁止擅自修改）

> **适用范围**：`<AdminLayout>` / `<AdminSidebar>`（左侧导航）与 `<TenantLayout>` / `<TopNav>`（顶部导航）是框架级基础设施，不是页面组件。

**❌ 禁止事项**：

1. **不要私自添加/删除/隐藏菜单项** → 通过设计师 + 产品确认后记录到 `references/migration-map.md`
2. **不要改导航项的顺序** → 违反用户的心智模型（历史操作路径）
3. **不要改导航项的图标或标签** → 除非设计师明确批复
4. **不要嵌套超过 2 层菜单** → Admin 保持扁平化，Tenant TopNav 禁止下拉
5. **不要在导航中混合权限判断** → 权限逻辑用 `<ProtectedRoute>` 或 Ability Provider 统一管理，不散落在导航组件里
6. **不要擅自改导航宽度/高度** → Admin Sidebar: w-64 fixed（不改）；TopNav: h-16 fixed（不改）

**✅ 正确做法**：

当需要修改导航时：
1. 在 `references/conflict-log.md` 记录需求（Why + What + Who approved）
2. 等设计 + 产品 review 通过
3. **同时更新** 三处地点：
   - `client/src/layouts/AdminSidebar.tsx` 或 `TopNav.tsx`（实现）
   - `references/admin.md` 或 `references/tenant.md`（规范）
   - `component-specs/admin-sidebar.md` 或 `component-specs/tenant-topnav.md`（spec）
4. 运行 `npm run check-spec-symbols` 和 `npm run verify-portable` 验证

**例外情况**（不需要 design review）：
- 权限导致的菜单项隐藏（用 `ability.can()` 判断，不改导航结构）
- 国际化（i18n）的标签翻译（不改导航结构）
- 图标库更新导致的图标替换（保持视觉一致性即可）

详见 `component-specs/admin-sidebar.md` 与 `component-specs/tenant-topnav.md`。

---

## 3. Workflow — 生成新页面 SOP（9 步）

> 这是 AI 收到产品指令后**应当严格执行**的顺序。每一步都对应一个交付物。

| # | 步骤 | 动作 | 交付物 / 检查 |
|---|---|---|---|
| 1 | **探测宿主仓** | 跑 §4 Project Context Detection 7 项 | 一句话宿主仓画像（Vite/Next/CRA、Tailwind v4？、`--cp-*` 是否存在、是否有 `Typography.tsx`、alias） |
| 2 | **判场景** | 客户端 Tenant / 非客户端（管控端 Admin / Landing），见 §0 Scope | 客户端立刻切换到 `references/tenant.md` 不再读本文；非客户端继续走本 skill，圆角默认 4px |
| 3 | **匹配 Page Recipe** | 查 `references/page-recipes.md` 的 6 类，并先判断是”完整页面”还是”组件级局部预览” | 选定骨架（列表 / 表单 / 详情 / Dashboard / 设置 / 空态），若是 Admin 完整页面则默认带 `AdminSidebar` / `AdminLayout` |
| 4 | **选组件**（决策表）| 查 §5 Component Selection + §6 Forms 决策表 | 列出本页用到的 8-15 个组件 + import 路径 |
| 4.5 | **🔴 选图（核心规则检查）** | 逐组件检查是否含高风险图标槽位；若是则查 `references/assets-icons.md §5.5` 与 `resource-skill-map.json`，确认 SVG 候选；若无则标 `needs-design-confirmation` | 列出用到的 icon 槽位 + 对应 SVG 源（或标 needs-design-confirmation 的位置）；验证无违规 lucide 被用在禁用槽位 |
| 5 | **拼骨架** | 按 recipe 的伪代码顺序拼；图标位置按第 4.5 步的结果直接用 SVG | 跑 §7 Key Patterns 现成片段，能复用就不要重写 |
| 6 | **校 token** | 高风险组件查 `component-specs/*.md` 的 §3 视觉标准 | 每个写死的颜色 / 圆角 / 阴影回去比对，列出”硬编码 → token”的偏差表 |
| 7 | **跑 Self-Audit**（§8）| 8 项（新增图标检查）30 秒自检 | 0 项遗留再交付 |
| 8 | **标 Fallback** | 宿主仓没有 `SurfaceCard` / `TenantCard` / Typography / `--cp-*` 时，参 `references/migration-map.md` 给出降级写法 | 在 PR 描述里列”宿主仓接入 N 个 fallback” |
| 9 | **产 PR 摘要** | 数字列表罗列改动点 + Token 偏差表（如果 §6 发现） + 端别决策 + Icon 槽位表（第 4.5 步产出） | 走 PR 模板（参考 PR #491 风格） |

---

## 4. Project Context Detection（7 项探测）

> 在执行 Workflow 第 1 步之前，AI 必须先探测以下 7 项，结果决定后续是直接用 demo 组件还是走 portable fallback。

| # | 探测项 | 命令 / 路径 | 影响 |
|---|---|---|---|
| 1 | UI 组件目录 | `client/src/components/ui/*` 是否存在 | 没有 → 走 `portable/react/*`（含 `admin-sidebar.tsx` + `css/admin-sidebar.css` 的 1:1 全量兜底，直接复制即可还原 21 个子组件，无 shadcn/cva/radix 依赖） |
| 2 | CSS Token | `client/src/index.css` 有无 `--cp-*` 变量 | 没有 → 走 `portable/html-css/*` 把 token 内联 |
| 3 | Typography 语义层 | `client/src/components/ui/Typography.tsx` | 没有 → 退回手写 + `--text-*` 或 portable |
| 4 | 构建 | `vite.config.*` / `next.config.*` / Tailwind v4 | 影响 import 写法、CSS 处理方式 |
| 5 | 路由 | `wouter` / `react-router` / Next App Router | 影响页面级 `page-enter` 与跳转写法 |
| 6 | 仓库形态 | 是否 monorepo（`pnpm-workspace.yaml` / `apps/*`） | 影响 alias、Skill 复用范围 |
| 7 | tsconfig path alias | `paths: { "@/*": ... }` | 直接决定 import 写法 |
| 8 | WebGL / ogl 可用性（**仅命中 hero 动态背景场景才探测**） | 浏览器支持 WebGL + 宿主仓可装 `ogl`；并读 `prefers-reduced-motion` | 决定装饰性动态背景（DarkVeil）走哪档：可装 ogl + 支持 WebGL → **L0** 完整动态；不便引 ogl/WebGL → **L1** 纯 CSS 渐变；禁脚本 / 静态导出 / 低端 / `reduced-motion` → 保底 **L2** 纯色或静态截图。**L2 是任何环境都能落地的下限**，不允许直接空白。详见 `component-specs/dark-veil.md` §9 |

**最低产出**：写代码前给一句"宿主仓画像"，例如：  
> "Vite + Tailwind v4 + `--cp-*` 已就绪 + `Typography.tsx` 已存在 + alias `@/*` → 可直接用 demo 组件，不需要 portable fallback"。

---

## 5. Component Selection（决策表入口）

> 完整版见 `references/components.md`。这里是 AI 入口必读。

| 需求 | 用什么 | 禁止 |
|---|---|---|
| Admin 主卡 / 列表卡 / 统计卡 | `SurfaceCard` | 手写 `div.bg-white + shadow + border` |
| Tenant 业务对象卡（Agent / 技能） | `TenantCard` | 用 `SurfaceCard` 机械替代 |
| 卡内表格 / 分组容器 | `SurfaceInner` | 嵌套多个 `SurfaceCard` |
| Dialog / Sheet / Popover 浮层 | `SurfaceOverlay` 或宿主浮层 | 手写重阴影浮层 |
| 高亮配置 / Pro 推荐卡 | `SurfaceConfig` | 普通卡片硬加大阴影 |
| Admin 主操作按钮 | `Button variant="claw-primary"` | inline 渐变 |
| Tenant 主操作按钮 | `Button variant="tenant-primary"` | 硬编码颜色 |
| 危险操作 | `variant="destructive"` / `tenant-destructive` | 手写红色 |
| 仅图标方按钮 | `size="claw-square"` 或 tenant 对应 size | 手写 size |
| 提示横幅 | `Alert` | `div + bg-yellow-50` |
| Toast | `sonner` | 手写浮层 |
| 空状态 | `Empty` 系列（见 §2.6）| 拼接 `text-center py-12` |
| 加载占位 | `Skeleton` | 手写灰块 |
| 分隔线 | `Separator` | `<hr />` 加自定义样式 |
| 状态标签 | `Badge` / `StatusTag` 或 `badge-running` 等预设 | 自定义状态色 |

---

## 6. Forms 决策表（三份 spec 归口）

> 做表单页时按下表选 spec，不要三份并列读。

| 我要做的 | 看哪份 spec | 重点 |
|---|---|---|
| 表单字段成组（Label + Control + Helper） | `component-specs/form-controls.md` | Field / FieldGroup 结构、`gap-3`、Label 必填红点 |
| 单个 Input / Select 的视觉与状态 | `component-specs/input-select.md` | 36px 高 / 4px 圆角 / 默认 `--cp-border` / focus 品牌蓝 / 报错 / 禁用 5 状态 |
| Switch / Checkbox / Radio | `component-specs/selection-controls.md` | 默认描边走 `--cp-border-control` (#C8CFDA)，**不是** `--cp-border` |
| 带搜索的对象选择 | `component-specs/combobox.md`（alias） + `select.tsx` 内 `SearchableSelect` | 用 SearchableSelect（旧名 Combobox，已并入 Select 的 searchable 模式），不是基础 Select |
| 日期选择 | `component-specs/date-picker.md` | 5 状态 + range pair + min/max |
| 文件上传 | `component-specs/upload-file-browser.md` | dashed 边框与 Empty 共享，但禁用默认 Upload 图标 |
| 资产包多版本文件浏览（Skill / Plugin / MCP 详情）| `component-specs/file-browser.md` | 三栏只读浏览器（版本/树/内容），不要再手写 |
| 分步向导 / 步骤指示 | `component-specs/stepper.md` | 24px 圆圈 + `ChevronRight` 分隔；状态由 `current` 自动推导 |
| 树形单选下拉（部门 / 分组 / 目录筛选） | `component-specs/tree-select.md` | button / filter-icon 两变体；**多选**走 `transfer.md` |

**Forms 通用速查**：

```tsx
// ✅ 字段成组（推荐）
<FieldGroup>
  <Field>
    <Label required>实例名称</Label>
    <Input placeholder="请输入..." />
    <HelperText>仅支持字母、数字、连字符</HelperText>
  </Field>
  <Field>
    <Label>规格</Label>
    <SearchableSelect options={...} />
  </Field>
</FieldGroup>
```

---

## 7. Key Patterns（✅/❌ 代码速查）

### 7.1 列表页骨架（最高频）

前提：下面这段是页面主内容骨架，不等于可以省略 Admin 页面外层壳。对管控端完整页面，默认应放在 `<AdminLayout>` / `<AdminSidebarInset>` 内；只有用户明确说“只做局部列表 demo”时，才允许单独截这一段。

```tsx
<div className="page-enter">
  <PageHeader title="实例列表" actions={<Button variant="claw-primary">创建实例</Button>} />
  <SurfaceCard>
    <div className="flex items-center gap-3 px-6 py-4">
      <Input placeholder="搜索..." className="w-64" />
      <Select ... />
      <Button variant="claw-outline">刷新</Button>
    </div>
    <SurfaceInner>
      <Table>...</Table>
    </SurfaceInner>
    <Pagination ... />
  </SurfaceCard>
</div>
```

### 7.2 Token 引用

```tsx
// ❌
<div className="border-gray-200 bg-[#1447E6]" />
// ✅
<div className="border-[var(--cp-border)] bg-[var(--cp-brand-blue)]" />
```

### 7.3 场景分流（管控端 vs 客户端）

```tsx
// ❌ 一个组件硬写死 / 直接照搬 Tenant 规则到管控端
<button className="rounded-full bg-[#1447E6]" />
<div className="rounded-xl">...</div>          {/* 管控端不允许 12px */}

// ✅ 管控端默认值（本 skill 适用范围）
<Button variant="claw-primary">创建</Button>
<SurfaceCard>...</SurfaceCard>                  {/* 4px 圆角 */}

// ✅ 客户端请切去 references/tenant.md，不要在管控端写下面这种
// <Button variant="tenant-primary">创建</Button>
// <TenantCard>...</TenantCard>                 {/* 12px 圆角，仅 Tenant */}
```

### 7.4 Surface 嵌套（不要套娃）

```tsx
// ❌ 卡套卡
<SurfaceCard><SurfaceCard>...</SurfaceCard></SurfaceCard>

// ✅ 内层用 SurfaceInner
<SurfaceCard><SurfaceInner>...</SurfaceInner></SurfaceCard>
```

### 7.5 表格表头色 / 单元格色

```tsx
// ✅ portable 兜底（宿主仓没有 Table 组件时）
<thead className="bg-gray-50/50">
  <tr><th className="px-6 py-3 text-xs font-medium text-[var(--cp-text-muted)]">列名</th></tr>
</thead>
<tbody>
  <tr><td className="px-6 py-4 text-xs text-[var(--cp-text-body)]">值</td></tr>
</tbody>
```

### 7.6 Hover Card / Popover 选用

```tsx
// ❌ 长说明塞 Tooltip
<Tooltip content="一大段说明...">
// ✅
<HoverCard>...</HoverCard>          // 不可点击
<Popover>...</Popover>              // 内容可点击
```

### 7.7 数字 / 统计

```tsx
// ❌
<span className="text-2xl">{1234}</span>
// ✅
<StatNumber>{1234}</StatNumber>     {/* tabular-nums + DIN */}
```

### 7.8 危险确认

```tsx
// ❌ 普通 Dialog 当危险确认
<Dialog>...</Dialog>
// ✅
<AlertDialog>
  <AlertDialogAction asChild>
    <Button variant="destructive">确认删除</Button>
  </AlertDialogAction>
</AlertDialog>
```

---

## 8. Self-Audit Checklist（30 秒自检）

> AI 写完代码立刻自检的 9 项，区别于 `qa/*-checklist.md`（验收侧）。

- [ ] **[新增] 图标槽位用对、无违规 lucide**
    - [ ] 检查是否有 9 个不可回退槽位（number-card / status-tag / card-left-icon / run-status / feature-card / 其他）
    - [ ] 若有，确认用的是 SVG（from resource-skill-map.json），不是 lucide
    - [ ] 若某槽位无候选，标记为 `needs-design-confirmation`，而非降级 lucide
- [ ] 没有硬编码颜色（`#xxxxxx` / `rgb(...)` / `text-gray-*` / `bg-yellow-50`）
- [ ] 没有硬编码圆角（`rounded-xl` / `rounded-2xl` / `rounded-[8px]` / `rounded-[12px]`）；管控端面板类元素必须 4px (`--radius-lg`)
- [ ] 间距用 `flex + gap` 或 `grid + gap`，不是逐个 margin
- [ ] Empty 用 `Empty` 系列，不是 `text-center py-12` 拼接
- [ ] Surface 没有套娃（`SurfaceCard > SurfaceInner` 而不是 `SurfaceCard > SurfaceCard`）
- [ ] **场景识别正确**：管控端没有混进 `tenant-*` variant / `TenantCard` / 12px 卡片圆角 / 顶部透明导航；客户端没有混进 `claw-*` variant / `SurfaceConfig` / Admin 渐变侧边栏
- [ ] 按钮 variant 没有借用 shadcn `outline` 伪装次级按钮
- [ ] 文字色走 Typography 语义或 `--text-*`，没散落 `text-[#xxx]`
- [ ] **图标槽位用对**：业务 / 侧栏图标已查 §9 + `assets-icons.md §5.5` 的 resource-skill-map 槽位候选，命中槽位未回退 lucide、未手搓 inline SVG；无候选已标 `needs-design-confirmation`（不私自占位）
- [ ] **改容器必贯彻子元素**（P3）：凡改了容器（卡片 / 表格 / 弹窗 / 表单等）的圆角·边框·间距，已回头核对内部子元素的字号·行高·颜色 token 是否同步到位，不留"外层改了、里面没改"

任何一项命中"是"，回到 §2 / §7 修正后再交付。

---

## 9. 高风险组件 Spec（按需读，不必全读）

> 任务命中即读对应 spec 的 §3 视觉标准与 §5 状态部分。

| 组件 | spec 文件 | 高风险点 |
|---|---|---|
| Surface / Card | `card-surface.md` | 层级嵌套、阴影 token |
| Button | `button.md` | variant / 渐变 / outline 借用 |
| Table | `table.md` | 12px 字号 / 表头底色 |
| Empty | `empty-state.md` | 容器场景速查（页面 / 表格 / Dialog / Popover） |
| PageHeader | `page-header.md` | 主操作位置 / 文案 |
| SearchFilterBar | `search-filter-bar.md` | gap-3 / 搜索图标 |
| DatePicker | `date-picker.md` | 5 状态 + range + min/max |
| SearchableSelect（旧名 Combobox） | `combobox.md`（alias 文档） | 已并入 Select 的 searchable 模式；与基础 Select 的边界 |
| BatchActions | `batch-actions-bar.md` | 选中数量 + 取消入口 |
| Popover / DropdownMenu | `popover-dropdown-menu.md` | Portal 逃逸 |
| Admin Sidebar | `admin-sidebar.md` | 240/64 / 弱蓝渐变 active；portable 全量兜底已补齐（`portable/react/admin-sidebar.tsx` + `portable/css/admin-sidebar.css`） |
| Tenant TopNav | `tenant-topnav.md` | 64px / 三栏 / 32 图标 |
| Breadcrumb | `breadcrumb.md` | 当前页不可点击 |
| Loading / Progress | `loading-progress.md` | 骨架行 / 局部 spinner |
| Chart / Stat | `chart-stat.md` | 主色 #1447E6 / tabular-nums |
| Upload | `upload-file-browser.md` | dashed 边框 / 禁用默认图标 |
| FileBrowser（多版本只读） | `file-browser.md` | 三栏：版本 14% / 文件树 22% / 内容 flex-1 |
| Form Controls | `form-controls.md` | 字段成组 |
| Input / Select | `input-select.md` | 36px / 4px / 5 状态 |
| Selection Controls | `selection-controls.md` | `--cp-border-control` ≠ `--cp-border` |
| Dialog / Drawer | `dialog-drawer.md` | 长表单 `max-h-[90vh]` |
| LineTabs（page header 下方一级 Tab） | `tabs.md` | active 项 `border-b-2 border-[#0A0A0A] -mb-px` |
| Segment（卡片内 / 工具栏切换） | `segment.md` | Admin 方角 / Tenant 胶囊分流；active 滑块走 `--shadow-segment` |
| Stepper（分步向导步骤条） | `stepper.md` | 24px 圆圈 / `ChevronRight` 分隔 / `current` 自动推状态 |
| TreeSelect（树形单选下拉） | `tree-select.md` | button / filter-icon 两变体；多选走 `transfer.md` |
| DarkVeil（装饰动态背景） | `dark-veil.md` | 仅 hero / 纯装饰 / `pointer-events-none` / 三层叠层保可读；先判 §0 Auto-Trigger 再用；ogl 依赖 + L0·L1·L2 跨仓兜底 |

---

## 10. References Map（规则 ↔ 文档双向）

| 任务 | 必读 | 按需 |
|---|---|---|
| 任意设计规范应用 | `references/foundation.md`、`references/components.md` | — |
| Admin 页 | `references/admin.md` | `tenant.md` 对照差异 |
| Tenant 页 | `references/tenant.md` | `admin.md` 对照差异 |
| Landing 页 | `references/landing.md` | — |
| 页面骨架决策 | `references/page-recipes.md` | `assets/page-references/*`（真实页面样本，配置 / 列表 / 空 / 复杂列表 4 类） |
| 能力开通页 / 动态背景 hero | `component-specs/dark-veil.md`、`references/admin-cloud-dev-activation.md` | `references/page-recipes.md`「能力开通页 Hero」、`references/migration-map.md`（DarkVeil → L1） |
| 跨仓 fallback / 替换宿主组件 | `references/migration-map.md` | `portable/html-css/*`、`portable/react/*` |
| 图标 / 插画 | `references/assets-icons.md`（§5.5 槽位选图）；当前项目事实源 `client/src/design-assets/resource-skill-map.json`，跨仓样例 `assets/icon-registry.example.json` | — |
| 高风险组件 | `component-specs/*.md` | — |
| 冲突 | `references/conflict-log.md` | — |
| 验收 / 走查 | `qa/*-checklist.md` | — |

---

## 11. 输出要求

### 11.1 做页面
必须说明：① 端别 ② 选用的 page recipe ③ 用到的高风险组件清单 ④ 宿主仓 fallback 策略（命中 §4 探测后再决定）。

### 11.2 做组件
必须输出：适用范围 / 视觉标准 / 状态 / demo 仓用法 / portable fallback / 迁移规则 / QA checklist。

### 11.3 做审查
按 P0 / P1 / P2 输出，关注：① 是否偏离 component-spec ② 是否硬编码 token ③ 是否缺 fallback ④ 是否引入旧写法 ⑤ Admin/Tenant 分流是否正确。

---

## 12. 冲突处理（完整版）

### 12.1 冲突定义

冲突是指以下情况：
- Spec 与实现不一致
- 需求超出规范范围
- Token / Variant / 图标槽位 / 圆角 / 阴影 等设计拍板模糊
- 宿主仓现状与 Spec 差异

### 12.2 冲突处理流程

**第 1 步 — 识别冲突**
- AI / 工程师在开发时遇到冲突时，不要私自裁决
- 立刻停下来，准备冲突记录

**第 2 步 — 记录冲突**
- 位置：`references/conflict-log.md`
- 格式：见 §1.5 的"冲突记录流程"
- 时间：同步记录，不要积累

**第 3 步 — 交设计师裁决**
- 设计师在下一个 review 周期统一处理
- 设计师补充"裁决原因"和"避免办法"
- 转化为后续改动或规范补充

**第 4 步 — 执行改动**
- 按设计师裁决执行
- 在 commit message 里标 `Ref: conflict-log.md #N`
- 同步更新规范，防止再犯

### 12.3 不同类型冲突的处理方式

| 冲突类型 | 示例 | 处理方式 | 时间框 |
|--------|------|--------|--------|
| **Spec vs 现状** | Spec 要 4px，现状 8px | 标记冲突，交设计师选 Spec 还是现状 | 下一个 design review |
| **需求超范围** | 新增未定义的状态 | 先补规范，再落地实现 | 前置 1-2 天 |
| **图标无候选** | 业务图标找不到合适 SVG | 标 needs-design-confirmation，用临时占位 | 后续由设计提供 |
| **Token 模糊** | "这个应该用什么 token？" | 列出 2-3 个可选方案，交设计选 | 下一个 design review |
| **宿主仓兼容** | 宿主仓组件与我们的差异 | 列出差异，交设计拍板"兼容还是改宿主" | 需要产品+设计对齐 |

### 12.4 铁律

**一句话**：冲突没有设计师裁决就不动手。

具体禁止事项：
- ❌ 为快速交付私自裁决冲突
- ❌ 对模糊的图标槽位直接选 lucide（应标 needs-design-confirmation）
- ❌ 看到 Spec 与现状不一致就"为了兼容"忽视 Spec
- ❌ 新需求超范围就自行扩展规范（应先补规范文档）
- ❌ 累积冲突到下个月再处理（应同步记录）

### 12.5 图标槽位特殊规则

基于 §2.8 的图标决策流程，图标冲突单独处理：

**无合适候选时的标准做法**：
1. 检查 resource-skill-map.json 是否真的无候选（可能新增了）
2. 查 icon-registry 是否有可用的相似图标
3. 如果真的无合适的：
   - 标 `needs-design-confirmation: [槽位名] 需要 [描述] 的图标`
   - 用临时占位符（可用其他槽位已有的类似图标）
   - 记录到"图标待设计确认清单"
   - 等设计提供

**不允许的做法**：
- ❌ 直接降级用 lucide（违反 §2.8 的铁律）
- ❌ 手搓 inline SVG（应先确认是否需要）
- ❌ 自定义 SVG 颜色（应用 token）

### 12.6 冲突日志审查

- 设计师每周 review 一次 conflict-log.md
- 积压超过 3 个冲突未处理，立刻升级（可能需要产品参与）
- 定期复盘冲突根因，改进规范或流程

### 12.7 冲突的信息出口

冲突的所有记录 + 裁决 = 知识库：
- 新成员通过阅读旧冲突理解设计决策的背景
- 发现规律性冲突（如"Token 定义模糊"）时，补规范
- 定期从 conflict-log.md 生成"设计决策史"文档

---

## 13. 禁止事项

- ❌ 把本 skill 当作"demo 仓源码镜像"。
- ❌ 只写组件名，不写 fallback。
- ❌ 把"可安装组件库"作为默认前提。
- ❌ 让产品前端从多个规范文件里自行仲裁。
- ❌ 在 §3 Workflow 第 1 步未跑探测的情况下直接生成代码。
- ❌ 未跑 §8 Self-Audit 就提交。

---

## 15. 图表规范

Admin 端数据可视化统一使用 **recharts**（LineChart 为主）。

### 7.1 基础配置

```jsx
import { LineChart, Line, BarChart, Bar, CartesianGrid, XAxis, YAxis, Tooltip, Legend, ResponsiveContainer } from 'recharts';

// 标准线图
<ResponsiveContainer width="100%" height={300}>
  <LineChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
    <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
    <XAxis tick={{ fontSize: 12, fill: "var(--cp-text-muted)" }} />
    <YAxis tick={{ fontSize: 12, fill: "var(--cp-text-muted)" }} />
    <Tooltip 
      contentStyle={{
        backgroundColor: "rgba(255, 255, 255, 0.95)",
        border: "1px solid var(--cp-border)",
        borderRadius: "4px"
      }}
    />
    <Legend />
    <Line type="monotone" dataKey="value" stroke="var(--cp-brand-blue)" strokeWidth={2} />
  </LineChart>
</ResponsiveContainer>
```

### 7.2 颜色系统（对标 Icon 渐变）

| 数据线 | 颜色 | 用途 |
|---|---|---|
| 主要指标 | `#1447E6`（品牌蓝） | 输入 Tokens / 总数统计 |
| 次要指标 | `#8B5CF6`（紫） | 输出 Tokens |
| 成功态 | `#16A34A`（绿） | 成功率 / 完成度 |
| 警告态 | `#EA8C00`（橙） | 警告 / 使用率 > 60% |
| 错误态 | `#DC2626`（红） | 错误 / 超额 |
| 中立态 | `#64748B`（灰） | 已停用 / 无效数据 |

### 7.3 图表类型选择

| 场景 | 组件 | 说明 |
|---|---|---|
| 时间序列趋势 | `LineChart` | 最常用，显示变化趋势 |
| 数值对比 | `BarChart` | 横向/纵向对比数据 |
| 占比分布 | `PieChart` | 显示部分与整体的关系 |
| 多维度对比 | `ComposedChart` | 混合图表（线+柱） |

### 7.4 数据标签与交互

```jsx
// 带数据标签的柱状图
<Bar dataKey="value" label={{ position: 'top', fill: 'var(--cp-text-title)', fontSize: 12 }} />

// 自定义 Tooltip
<Tooltip 
  formatter={(value) => `${value.toLocaleString()} tokens`}
  labelFormatter={(label) => `时间: ${label}`}
/>

// 图例位置（默认底部）
<Legend verticalAlign="top" height={36} />
```

### 7.5 响应式与容器

```jsx
// 始终用 ResponsiveContainer 确保适应容器宽度
<ResponsiveContainer width="100%" height={300}>
  <LineChart data={data}>...</LineChart>
</ResponsiveContainer>

// 容器最小宽度 400px（避免图表压缩）
<div className="min-w-[400px] h-[300px]">
  <ResponsiveContainer width="100%" height="100%">
    ...
  </ResponsiveContainer>
</div>
```

---

## 16. 动画系统

### 8.1 页面进入动画

所有页面根元素必须包含 `page-enter` class 实现进入动画：

```css
.page-enter {
  animation: pageEnter 0.25s ease-out;
}

@keyframes pageEnter {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

```jsx
// 页面组件顶层包裹
<div className="page-enter">
  {/* 页面内容 */}
</div>
```

### 8.2 交互过渡

| 场景 | 推荐值 | Tailwind 类 | 说明 |
|---|---|---|---|
| 通用 hover | 150ms | `transition-all duration-150` | 按钮、链接、菜单项 |
| 颜色变化 | 150ms | `transition-colors duration-150` | 文字、背景单独变化 |
| 卡片 hover | 200ms | `transition-all duration-200 hover:shadow-lg` | 卡片提升、阴影加强 |
| Popover/Drawer | 200ms | `transition-all duration-200` | 浮层平滑出入 |
| 加载状态 | 无限 | `animate-spin` | Loading icon 旋转 |

### 8.3 Dialog / Drawer 动画

使用 framer-motion 或 CSS 动画：

```jsx
// 使用 framer-motion（推荐）
import { motion } from 'framer-motion';

<motion.div
  initial={{ opacity: 0, scale: 0.95 }}
  animate={{ opacity: 1, scale: 1 }}
  exit={{ opacity: 0, scale: 0.95 }}
  transition={{ duration: 0.2 }}
  className="fixed inset-0 bg-black/50 flex items-center justify-center"
>
  {/* Dialog 内容 */}
</motion.div>

// 纯 CSS 方案
<div className="animate-in fade-in-0 zoom-in-95 duration-200">
  {/* Dialog 内容 */}
</div>
```

### 8.4 列表动画（framer-motion）

```jsx
<motion.div layout>
  {items.map((item) => (
    <motion.div
      key={item.id}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.15 }}
    >
      {item.content}
    </motion.div>
  ))}
</motion.div>
```

---

## 17. 图标规范

**默认/通用 UI**（导航、按钮、表格行操作等未命中业务槽位的图标）：用 **lucide-react**。
**业务 / 品牌图标**（命中 `references/assets-icons.md` §5.5 已登记槽位）：用已登记 registry SVG——**当前项目以 `client/src/design-assets/resource-skill-map.json` 的槽位候选为准（事实源，由 `npm run build:skill-map` 派生，勿手改）**；跨仓 / 可移植语境参照 `assets/icon-registry.example.json` 身份样例、由宿主仓正式 registry 提供。无合适候选时标 `needs-design-confirmation` 交设计补绘、由产品联系设计师，**不回退 lucide、不手搓 inline SVG**。
**始终禁止**：emoji、FontAwesome 及其他图标库。

### 9.1 尺寸规范

| 用途 | 尺寸 | Tailwind 类 | 说明 |
|---|---|---|---|
| 导航项 icon | 16px | `w-4 h-4` | Sidebar 或 Navbar 中 |
| 按钮内 icon | 16px | `w-4 h-4` | 按钮文字旁 |
| 统计卡片 icon | 20px | `w-5 h-5` | 渐变容器内 |
| 表格行操作 | 14px | `w-3.5 h-3.5` | 删除/编辑/详情等 |
| 空状态 icon | 48px | `w-12 h-12` | 大占位符 |
| 状态徽章 dot | 6px | `w-1.5 h-1.5` | 运行中/已停止 |

### 9.2 Admin 通用动作图标（lucide）

> ⚠️ **本表只列通用动作图标**（下载 / 删除 / 编辑 / 搜索 / 刷新等无业务语义的操作图标），这类可直接用 lucide。
> **侧栏菜单 / 业务模块图标（基础设置、成员管理、模型配置、通道配置、技能配置、镜像管理、安全组、监控看板、Token 监控、审计日志、帮助文档等）一律不在此表**——必须先查 `references/assets-icons.md §5.5` 的 `admin-sidebar` 槽位候选（事实源 `client/src/design-assets/resource-skill-map.json`）；命中槽位用已登记 registry SVG，无合适候选才标 `needs-design-confirmation` 交设计补绘，**严禁照搬 lucide 占位**（见 §9 图标决策与 ADR「9 槽位禁回退 lucide」）。

| 动作 | lucide-react 组件 |
|---|---|
| 下载 | `Download` |
| 删除 | `Trash2` |
| 编辑 | `Edit2` |
| 搜索 | `Search` |
| 刷新 | `RefreshCw` |

### 9.3 图标色系

```jsx
// 默认色（灰）
<Settings className="w-4 h-4 text-[var(--cp-text-muted)]" />

// 活跃态（品牌蓝）
<Settings className="w-4 h-4 text-[var(--cp-brand-blue)]" />

// 成功态（绿）
<CheckCircle className="w-4 h-4 text-[var(--cp-text-success)]" />

// 错误态（红）
<AlertCircle className="w-4 h-4 text-[var(--cp-text-danger)]" />

// 警告态（橙）—— cp 集合无 warning 文本变量，用运行时 --text-warning
<AlertTriangle className="w-4 h-4 text-[var(--text-warning)]" />
```

### 9.4 导入使用

```jsx
import { Settings, Users, Brain, Download, Trash2 } from 'lucide-react';

export function MyComponent() {
  return (
    <button className="flex items-center gap-2">
      <Download className="w-4 h-4" />
      下载
    </button>
  );
}
```

---

## 18. 操作反馈规范

### 10.1 Toast 通知（使用 sonner）

```jsx
import { toast } from 'sonner';

// 成功反馈
toast.success('操作成功', {
  description: '数据已保存',
  duration: 3000
});

// 错误反馈
toast.error('操作失败', {
  description: '请检查网络连接后重试',
  duration: 3000
});

// 加载反馈
toast.loading('正在处理...', {
  id: 'loading-1'
});

// 更新加载状态为成功
toast.success('处理完成', { id: 'loading-1' });

// 自定义内容
toast.custom((t) => (
  <div className="bg-white rounded-lg p-4 shadow-lg">
    自定义通知内容
  </div>
));
```

### 10.2 Toast 位置与样式

```jsx
// 在应用入口配置 Toaster
<Toaster
  position="top-right"
  richColors
  theme="light"
/>

// 可选位置：top-left / top-right / top-center / bottom-left / bottom-right / bottom-center
```

### 10.3 确认对话框（删除/销毁操作）

```jsx
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";

<AlertDialog open={open} onOpenChange={setOpen}>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>确认删除？</AlertDialogTitle>
      <AlertDialogDescription>
        此操作无法撤销，请确认后继续
      </AlertDialogDescription>
    </AlertDialogHeader>
    <div className="flex justify-end gap-2">
      <AlertDialogCancel>取消</AlertDialogCancel>
      <AlertDialogAction 
        className="bg-[var(--cp-text-danger)] text-white hover:bg-[var(--cp-text-danger)]/90"
        onClick={handleDelete}
      >
        确认删除
      </AlertDialogAction>
    </div>
  </AlertDialogContent>
</AlertDialog>
```

### 10.4 加载状态

```jsx
// 按钮加载态
<button disabled className="flex items-center gap-2">
  <Loader2 className="w-4 h-4 animate-spin" />
  正在保存...
</button>

// 表格加载态
<div className="flex justify-center py-8">
  <Loader2 className="w-6 h-6 animate-spin text-[var(--cp-brand-blue)]" />
</div>

// 骨架屏：用 Skeleton 组件（§5 决策表「加载占位 → Skeleton」），勿手写灰块
import { Skeleton } from "@/components/ui/skeleton";
<div className="space-y-2">
  <Skeleton className="h-4 w-full" />
  <Skeleton className="h-4 w-2/3" />
</div>
```

### 10.5 信息 / 警告横幅

```jsx
// 信息横幅：用 Alert variant="info"（§5 决策表「提示横幅 → Alert」，勿手写 bg-blue-50；见 component-specs/alert.md）
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示信息内容</AlertDescription>
</Alert>

// 警告横幅：用 Alert variant="warning" + CircleAlert（勿手写 bg-amber-50 / 勿用 AlertTriangle 作标准图标）
import { CircleAlert } from "lucide-react";
<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>警告信息内容</AlertDescription>
</Alert>

// 错误横幅：用 Alert variant="error"（勿手写 bg-red-50）
import { Alert, AlertDescription, AlertErrorIcon } from "@/components/ui/alert";
<Alert variant="error">
  <AlertErrorIcon />
  <AlertDescription>错误信息内容</AlertDescription>
</Alert>
```

---

## 19. 新页面 Checklist

创建任何新页面或组件前，逐项确认：

### 布局与结构
- [ ] 选择正确的 Layout（`<AdminLayout>` 或 `<TenantLayout>`）
- [ ] 根元素包含 `page-enter` class（页面进入动画）
- [ ] 页面 max-width 符合规范（Admin: max-w-3xl/5xl；Tenant: max-w-7xl）
- [ ] 内容区 padding 正确（Admin: p-8；Tenant: px-6 py-8）

### 组件使用
- [ ] 卡片使用 `rounded-lg`（4px）+ 阴影走 token（`tokens/radius-shadow.md` 定义的阴影 / 由 `SurfaceCard` 提供），不手写 inline `boxShadow`（见 `foundation.md §7`）
- [ ] 按钮使用正确的 variant（claw-primary / claw-outline / destructive）
- [ ] 表格使用原生 `<table>` + 规范的 thead/tbody 类
- [ ] 表单字段 Label 到 Input 间距 `space-y-2`
- [ ] 状态徽章使用 `badge-running` / `badge-stopped` / `badge-pending`
- [ ] 列表空状态有友好提示和操作按钮
- [ ] 危险操作（删除）使用 `<AlertDialog>` + 红色确认按钮

### 色彩与间距
- [ ] 颜色走 token（`--cp-brand-blue` 等），不硬编码 hex
- [ ] 圆角全部 4px（不要 8/12/16px），除头像/Switch/胶囊
- [ ] 间距使用规范值（p-8 / p-6 / p-5 / gap-3 / gap-4 等）
- [ ] 文字层级符合规范（标题 24px bold / 正文 14px regular / 数据 12px）

### 🔴 图表与图标（高风险，必检）
- [ ] 数据可视化使用 recharts（LineChart / BarChart）
- [ ] 图表配色遵循色系规范（蓝/紫/绿/橙/红）
- [ ] **⚠️ 图标槽位合规**（核心规则，见 §2.8）：
    - [ ] 命中 **9 个不可回退槽位**（number-card / status-tag / card-left-icon / run-status / feature-card / 其他 4 个）？
    - [ ] ✅ 是 → 必须用 `resource-skill-map.json` SVG，**禁 lucide**
    - [ ] ✅ 否 → 可用 lucide-react，或已登记 registry SVG
    - [ ] ⚠️ 拿不准 → 标 `needs-design-confirmation`，交设计确认
    - [ ] ❌ 绝不可 → 用 emoji 或手搓 inline SVG（除非是 portable zero-dependency 场景，需加注释）
- [ ] **图标尺寸合规**（导航 w-4 / 统计 w-5 / 空状态 w-12 / number-card 18×18 渐变）

### 动画与交互
- [ ] 页面有进入动画（`page-enter`）
- [ ] Hover 过渡时间正确（默认 150-200ms）
- [ ] Dialog/Drawer 有打开/关闭动画
- [ ] 加载按钮有旋转动画 icon
- [ ] 列表项删除/添加有动画过渡

### 反馈与验证
- [ ] 操作成功/失败使用 toast 通知（sonner）
- [ ] 删除/销毁操作使用 AlertDialog 确认
- [ ] 表单验证错误有清晰提示
- [ ] 长操作有加载状态反馈
- [ ] 空状态有说明文字和操作按钮

### 响应式与兼容性
- [ ] 在不同屏幕宽度测试布局
- [ ] 图表、表格有最小宽度限制（避免压缩）
- [ ] 长文本有截断或换行规范
- [ ] 移动端有合理降级方案

### 代码质量
- [ ] 没有硬编码 hex 色值
- [ ] 没有使用 shadcn Card / Table（用原生 div / table）
- [ ] 没有使用 emoji 作为图标
- [ ] 没有自己发明新的 CSS 类和样式
- [ ] 组件可重用，不是一次性代码

### 测试与验收
- [ ] 对照 `qa/admin-checklist.md` 逐项验收（如是 Admin 端）
- [ ] 对照 `qa/tenant-checklist.md` 逐项验收（如是 Tenant 端）
- [ ] 设计稿与实现一致（或有设计师确认的偏差）
- [ ] 没有明显的视觉瑕疵或动画卡顿

---

## 20. 工具

- 设计使用检查：  
  `node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-design-usage.mjs`
- 严格模式：  
  `node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-design-usage.mjs --strict`
