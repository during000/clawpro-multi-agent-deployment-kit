---
name: clawpro-global-components
description: >
  ClawPro 全局组件样式规范（owner: addietang）。
  此文件定义所有基础 UI 组件的视觉规范，包括字体、颜色、描边、圆角、阴影、交互状态。
  任何人修改页面时，组件样式必须严格遵循此规范，不允许覆盖或自由发挥。
  如有冲突，以此文件为准。
---

# ClawPro 全局组件样式规范

> **Owner**: addietang  
> **全局样式修改人**: addietang, miekoyychen  
> **优先级**: 最高——所有分支合并时，组件样式以此规范为准，不允许其他人修改组件源文件  
> **组件源码路径**: `client/src/components/ui/`  
> **CSS 变量定义**: `client/src/index.css`  
> **说明**: miekoyychen 负责全局样式的后续修改和调整，与 addietang 共同维护本规范

---

## 📚 配套规范（必读）

本文件是**全平台共享**的基础组件规范。在以下场景中必须叠加加载额外规范：

| 场景 | 必读规范 | 冲突优先级 |
|------|----------|------------|
| 用户端（Tenant）页面 / 组件 | 📄 [SKILL-TENANT.md](./SKILL-TENANT.md) | **Tenant > 本文件**（仅在用户端） |
| 管控端（Admin）页面 / 组件 | 📄 [SKILL.md](./SKILL.md) | 本文件 > SKILL.md |
| 设计语言 / 色彩 / 布局通则 | 📄 [SKILL.md](./SKILL.md) | — |

**用户端共享组件扩展约束**：用户端如需对 Button / Card / Tabs 等共享组件做差异化样式，**必须新增 `tenant-*` 变体**，禁止覆盖现有 `claw-*` 变体或默认样式，避免影响管控端。具体规则见 `SKILL-TENANT.md`。

---

## 0. Typography 字体组件（用户端基础文字入口）

**文件**: `client/src/components/ui/Typography.tsx`  
**研究页**: `client/public/research/tenant-typography-audit.html`  
**适用范围**: 所有用户端（Tenant）页面、用户端共享组件、后续新增业务组件。管控端可按需复用数字 / 代码类组件，但管控端导航等已有专属规范的组件以自身规范为准。

### 0.1 设计原则

1. **文字不再只按字号写 class，而按语义选组件**：页面标题用 `TenantPageTitle`，卡片标题用 `CardTitle`，正文用 `BodyText`，辅助信息用 `MetaText`。
2. **默认色必须由 Typography 组件提供**：业务侧默认不要再手写 `text-gray-*` / `text-[#...]` 表达基础文字色。
3. **字号、字重、行高、字体族、默认颜色绑定在同一个组件里**：避免同一个 `text-sm` 在正文、按钮、Tab、表格中语义漂移。
4. **组件内部文字也应优先复用 Typography token**：新写 Button、Card、Dialog、Popover、表格、空状态、状态标签时，先判断文字是否可映射到 Typography 层级。
5. **只允许通过 `tone` 切换语义色**：不要用 `className` 覆盖颜色；`className` 主要用于布局（如 `mt-1`、`truncate`、`text-center`）。

### 0.2 颜色 token

| `tone` | Tailwind token | 色值 | 用途 |
|--------|----------------|------|------|
| `primary` | `text-gray-900` | `#171717` | 标题、卡片标题、主内容 |
| `emphasis` | `text-gray-950` | `#0A0A0A` | 强调文字、按钮文字、关键字段 |
| `body` | `text-gray-900` | `#171717` | 正文主内容、表格内容 |
| `secondary` | `text-gray-700` | `#404040` | 同字号描述性正文、补充说明、次级信息 |
| `muted` | `text-gray-500` | `#737373` | 时间、描述、辅助说明、表头 |
| `weak` | `text-gray-400` | `#A3A3A3` | 占位、空状态、极弱提示 |
| `brand` | `text-[var(--brand-blue)]` | `#1447E6` | 链接、活跃态、步骤标识、英文 Badge |
| `danger` | `text-red-600` | `#DC2626` | 危险操作、错误提示 |
| `inherit` | `text-inherit` | 继承 | 放在已定义颜色的父级中 |

### 0.3 组件层级

| 组件 | 默认标签 | 字号 / 字重 / 行高 | 默认 tone | 使用场景 |
|------|----------|--------------------|------------|----------|
| `TenantHeroTitle` | `h1` | 26px / Medium / 35.56px | `primary` | 用户端 Hero 标题，如模型额度、技能广场 |
| `TenantPageTitle` | `h1` | 24px / Medium / 1.4 | `primary` | 页面标题、详情页主标题 |
| `TenantDocTitle` | `h1` | 20px / Semibold / 1.4 | `primary` | 帮助文档、文章标题 |
| `SectionTitle` | `h2` | 18px / Medium / 1.4 | `primary` | 页面内大模块标题 |
| `PanelTitle` | `h2` | 16px / Semibold / 1.4 | `primary` | Dialog / Sheet / 卡片区块 / 表格区块标题 |
| `CardTitle` | `h3` | 14px / Medium / 1.5 | `primary` | Agent 卡片名、技能卡标题、模型名、列表项标题 |
| `BodyText` | `p` | 14px / Regular / 1.5 | `body` | 普通正文、表格内容；描述行用 `tone="secondary"` |
| `BodyMedium` | `span` | 14px / Medium / 1.5 | `emphasis` | 按钮、Tab、Label、列表主字段 |
| `CompactText` | `span` | 13px / Regular / 1.5 | `secondary` | 紧凑列表、空间不足的轻量描述 |
| `MiniBodyText` | `span` | 12px / Regular / 1.5 | `body` | 紧凑表格正文、高密度列表主内容 |
| `MetaText` | `span` | 12px / Regular / 1.5 | `muted` | 时间、ID、Tooltip、辅助说明、空状态 |
| `MetaMedium` | `span` | 12px / Medium / 1.5 | `muted` | 表头、次级强调 |
| `SmallBodyText` | `span` | 12px / Medium / 12px / tracking 0.18px | `emphasis` | `StatusTag` 内文字、小型信息标签 |
| `TinyText` | `span` | 10px / Semibold / Open Sans | `brand` | `New` / `Beta` / 小角标 |
| `StatNumber` | `span` | 24px / Bold / DIN | `emphasis` | 统计数字、额度数字 |
| `InlineNumber` | `span` | 14px / DIN / tabular | `body` | 表格内 Token 数、请求数、百分比 |
| `CodeText` | `code` | 12px / Menlo | `secondary` | ID、Token、路径、命令、代码片段 |
| `StepText` | `span` | 14px / Medium / Menlo | `brand` | Step 1 / Step 2 / 步骤编号 |
| `UrlText` | `span` | 14px / Regular / PingFang SC / 1.5 / `break-all` / `#020617` | `inherit` | URL、回调地址、外链链接、版本号字符串等需要中性等宽呈现的引用文本 |

### 0.4 使用方式

```tsx
import {
  TenantPageTitle,
  PanelTitle,
  CardTitle,
  BodyText,
  BodyMedium,
  MetaText,
  SmallBodyText,
  StatNumber,
  CodeText,
  UrlText,
} from "@/components/ui/Typography";

<TenantPageTitle>Agent 详情</TenantPageTitle>
<BodyText>这里是当前 Agent 的模型、通道和技能配置说明。</BodyText>
<BodyText tone="secondary">这是同字号描述性文字，颜色浅一档。</BodyText>
<PanelTitle as="h3">模型使用汇总</PanelTitle>
<CardTitle>Alice 的技术助手</CardTitle>
<BodyMedium tone="brand">查看详情</BodyMedium>
<MetaText>更新于 2026-05-21 21:14</MetaText>
<SmallBodyText>用户</SmallBodyText>
<StatNumber>128,000</StatNumber>
<CodeText>ins-g71c6vud</CodeText>
<UrlText>https://api.example.com/v1/chat/completions</UrlText>
```

### 0.5 组件作者如何受影响

新建或修改全局组件时，按下面规则处理组件内部文字：

| 组件类型 | 内部文字推荐 |
|----------|--------------|
| Button / Tab / Segment item | `BodyMedium` 对应规格：14px / Medium / `emphasis`；组件内部可直接写等效 class，但不得偏离 Typography token |
| Dialog / Sheet 标题 | `PanelTitle` |
| Card 标题 | `CardTitle` |
| 表格表头 | 标准版用 `BodyMedium`（14px / Medium / `emphasis`）；紧凑版用 `MetaMedium` |
| 表格内容 | 标准版用 `BodyText` 或 `InlineNumber`（默认 `body`）；紧凑版用 `MiniBodyText`；同字号描述行用对应组件的 `tone="secondary"` |
| 空状态说明 | `MetaText tone="weak"` |
| Badge / New / Beta | `TinyText` 或 `MetaMedium`，英文 Badge 优先 `TinyText` |
| StatusTag / 小型信息标签 | `SmallBodyText` 对应规格：12px / Medium / `emphasis` / tracking 0.18px |
| 统计卡数字 | `StatNumber` |
| ID / Token / 路径 | `CodeText` |
| URL / 回调地址 / 外链 | `UrlText` |

> 注意：基础组件源码里不一定必须直接 import Typography（避免 Button 等低层组件依赖过深），但视觉参数必须与 Typography token 保持一致。业务页面与业务组件应优先直接使用 Typography 组件。

### 0.6 禁止事项

- 禁止在用户端新增页面中继续散落书写 `text-gray-900/700/500/400` 表达基础文字色；应改用 Typography 默认色或 `tone`。
- 禁止新增 inline `style={{ fontFamily: ... }}`；数字用 `StatNumber` / `InlineNumber`，代码用 `CodeText`，英文 Badge 用 `TinyText`。
- 禁止把 `text-[10px]` / `text-[11px]` 用于正文；只能用于 Badge、角标、通知小时间戳等极小信息。
- 禁止用 `className` 覆盖 Typography 组件颜色，除非是特殊业务状态色，并需能说明原因。
- 禁止直接全仓机械替换 `text-sm` / `text-xs`；必须按语义映射逐步迁移。

### 0.7 迁移策略

1. **新增页面 / 新增组件必须直接使用 Typography**。
2. **触达即同步**：修改用户端旧页面时，顺手把当前文件中明显的标题、正文、Meta、数字、代码文字替换为 Typography。
3. **优先迁移共享组件**：`components/topnav/**`、`components/agent/**`、用户端卡片/状态/空状态组件。
4. **再迁移核心页面**：`MyOpenClaw.tsx`、`OpenClawDetailGuide.tsx`、`SkillSquare.tsx`、`ModelQuota.tsx`。
5. **大型复杂页渐进处理**：`OpenClawDetail.tsx`、`ChatView.tsx` 按业务迭代逐步替换，避免一次性大重构。

### 0.8 Typography 迁移顺序

> 原则：不要从最大页面开始，不要全仓机械替换 `text-sm` / `text-xs`；先建立示范，再影响共享组件，最后通过“触达即同步”覆盖复杂页面。

| 阶段 | 优先级 | 范围 | 目标 |
|------|--------|------|------|
| 1. 示范 PR | 最高 | `components/agent/AgentCard.tsx` 或 `components/topnav/NotificationPanel.tsx` | 建立团队可复制的 import、`tone`、数字、ID、时间处理范式 |
| 2. 共享组件 | 高 | `client/src/components/topnav/**`、`client/src/components/agent/**`、用户端状态 / 空状态 / 卡片 / 表格区块组件 | 改一次，多处生效，快速统一用户端基础观感 |
| 3. 用户端核心页 | 中高 | `MyOpenClaw.tsx`、`OpenClawDetailGuide.tsx`、`SkillSquare.tsx`、`ModelQuota.tsx` | 覆盖用户高频主路径，让 Typography 的视觉收益尽快可见 |
| 4. 复杂大页 | 渐进 | `OpenClawDetail.tsx`、`ChatView.tsx`、`AgentChat.tsx` | 不做一次性大重构，改到哪个区块就同步哪个区块 |

复杂页面迁移时优先替换：页面标题、模块标题、卡片标题、Meta 信息、统计数字、ID / Token / 路径；聊天消息正文、Markdown 正文等内容型排版可按 §0.12 例外机制处理。

### 0.9 Vibe Coding / AI 辅助开发提示

同事使用 AI 生成用户端页面或组件时，必须在 prompt 中明确引用 Typography，而不是只说“注意字体规范”。推荐使用以下提示：

```text
这是 ClawPro 用户端页面 / 组件，请按 `SKILL-GLOBAL-COMPONENTS.md` 的 Typography 规范实现。

开工前必须读取：
1. `SKILL-GLOBAL-COMPONENTS.md`
2. `client/src/components/ui/Typography.tsx`
3. `client/public/research/typography-guideline.html`

要求：
- 优先使用 `@/components/ui/Typography`
- 不要自行发明字号、字重和基础文字色
- 不要新增 inline `fontFamily`
- 不要随意新增 `text-[xxpx]`
- 页面标题、卡片标题、正文、Meta、数字、代码文字必须按语义映射到 Typography 组件
- 如果现有 Typography 层级不满足，先说明原因，不要直接在页面里写新样式
```

更短版：

```text
这是用户端页面 / 组件，请遵循 `SKILL-GLOBAL-COMPONENTS.md` 的 Typography 规范，优先使用 `@/components/ui/Typography`，不要自行拼装基础文字样式。
```

### 0.10 Typography PR Review Checklist

用户端页面 / 组件提交时，Review 必须检查：

- 页面标题是否使用 `TenantHeroTitle` / `TenantPageTitle`。
- 大模块标题是否使用 `SectionTitle`。
- Dialog / Sheet / 表格区块 / 卡片区块标题是否使用或对齐 `PanelTitle`。
- 卡片对象名、Agent 名、技能名、模型名是否使用 `CardTitle`。
- 普通正文、说明、表格内容是否使用 `BodyText`。
- Label、Tab、按钮文字、列表主字段是否使用或对齐 `BodyMedium`。
- 时间、ID、Tooltip、描述、空状态是否使用 `MetaText` / `MetaMedium`。
- 统计大数字是否使用 `StatNumber`，表格内数字是否使用 `InlineNumber`。
- Token、路径、命令、实例 ID 是否使用 `CodeText`。
- 是否新增了散落的 `text-gray-*` / `text-[#...]` 表达基础文字色。
- 是否新增了 inline `style={{ fontFamily: ... }}`。
- 是否新增了无规范来源的 `text-[10px]` / `text-[11px]` / `text-[15px]` / 其他任意字号。

### 0.11 Typography 接入边界

| 类型 | 接入方式 | 说明 |
|------|----------|------|
| 用户端页面 | 直接 import Typography | 页面标题、正文、Meta、数字、代码必须优先使用 Typography |
| 用户端业务组件 | 直接 import Typography | Agent 卡、技能卡、模型卡、状态说明、空状态等应直接使用 |
| 用户端共享组件 | 优先直接 import Typography | `topnav`、`agent` 等共享组件改一次影响多处，优先接入 |
| 底层 UI 组件 | 不强制 import，但必须对齐 token | `Button`、`Input`、`Select`、`Dialog`、`Tabs`、`Segment` 等可写等效 class，避免依赖过深 |
| 特殊内容区 | 可局部豁免 | Markdown、聊天消息正文、代码编辑器、图表坐标轴等可保留专属排版，但标题 / Meta / 数字 / 代码仍应接入 Typography |

> 关键判断：业务层能直接使用 Typography 就直接使用；底层组件即使不 import，也必须能映射回 §0.3 的 Typography 层级。

### 0.12 Typography 例外机制

允许特殊场景例外，但必须满足以下条件：

1. **说明原因**：现有 Typography 层级为什么不适合该场景。
2. **不破坏色阶**：仍应使用 `#0A0A0A / #171717 / #404040 / #737373 / #A3A3A3 / #1447E6` 等既有 token。
3. **不新增无说明的字体族**：禁止新增 inline `fontFamily`；数字 / 代码 / 英文 Badge 优先使用 `font-din` / `font-mono` / `font-en`。
4. **不扩大豁免范围**：特殊内容区只豁免内容正文，不豁免页面标题、模块标题、Meta、数字、ID / Token / 路径。
5. **形成通用模式时反向沉淀**：如果某个例外被多个页面复用，应补充到 `Typography.tsx` 与本规范，而不是长期散落在页面里。

常见可豁免场景：Markdown 正文渲染、聊天消息正文、代码编辑器、图表坐标轴 / 图例、第三方富文本内容、极小空间内的特定角标。

---

## 1. 品牌色系

| 色值 | 变量/用途 |
|------|----------|
| `#355EF1` | 品牌蓝（`--color-blue-500`），hover/focus/选中态统一用色 |
| `#0A0A0A` | 强调文字（gray-950） |
| `#171717` | 主文字 / 正文（gray-900） |
| `#404040` | 次级文字（gray-700） |
| `#737373` | 辅助文字（gray-500） |
| `#A3A3A3` | placeholder 色 / HelperText 色（`--text-weak`） |
| `#d3d6db` | disabled 文字色 / Switch 关闭态轨道 / Checkbox 禁用勾选填充（**0605 起不再用作 Input/Select 描边**，描边统一走 `--border = #EAEEF4`） |
| `#E5E5E5` | 卡片描边色（gray-200） |
| `#F5F5F5` | 浅背景（gray-100） |
| `#f3f3f4` | disabled 背景 |
| `#d42a1e` | 错误/危险色 |

---

## 2. 圆角规范

| 组件 | 圆角 | 说明 |
|------|------|------|
| Button / Input / Select / DatePicker | `rounded-[4px]` | 4px，统一所有表单控件 |
| Dialog / Popover / DropdownMenu | `rounded-[8px]` | 8px，浮层类 |
| Card（SurfaceCard） | `rounded-xl`（12px） | 卡片容器 |
| Badge（状态徽章） | `rounded-full` | 圆形 |

**禁止**: 不允许使用 `rounded-lg`、`rounded-xl`、`rounded-2xl` 用在 Button/Input 上

---

## 3. 阴影规范

| 层级 | 值 | 用途 |
|------|-----|------|
| L1 卡片 | `0px 1px 4px rgba(0,0,0,0.05), 0px 0px 2px rgba(0,0,0,0.1)` | SurfaceCard |
| L2 内嵌 | `none` | 卡片内子卡 |
| L3 浮层 | `0px 4px 16px -2px rgba(0,0,0,0.08), 0px 2px 6px rgba(0,0,0,0.06)` | Dialog/Popover |
| L5 指示器 | `var(--shadow-segment)` | Tab 滑块 |

**禁止**: 不允许在组件上使用 inline `boxShadow`，统一用 CSS 变量

---

## 4. Button 组件

**文件**: `client/src/components/ui/button.tsx`

### 4.1 变体

| variant | 背景 | 边框 | 文字 | hover | disabled |
|---------|------|------|------|-------|----------|
| `claw-primary` / `default` | 纯黑 `#0A0A0A` | 无 | 白色 | `#404040`（与 dialog-confirm 对齐） | `#0A0A0A/40` + 文字50% |
| `dialog-confirm` | 纯黑 `#0A0A0A` | 无 | 白色 | `bg-[#404040]` | `bg-[#A3A3A3]` 白字 |
| `claw-outline` / `outline` | 白色 | `#EAEEF4` | `#020617` | `bg-[#f5f5f5]` | 文字`rgba(2,6,23,0.3)` |
| `destructive` | `#d42a1e` | 无 | 白色 | `#b91c1c` | 40%透明 |
| `ghost` | 无 | 无 | `#020617` | `bg-[#f5f5f5]` | 文字30%透明 |
| `plain` | 白色 | `#e4e4e4` | `#020617` | `border-[#020617]` | 文字`rgba(0,0,0,0.3)` |
| `link` | 无 | 无 | `#355EF1` | 加下划线 | `#d0dafa`（浅蓝灰） |
| `link-dark` | 无 | 无 | `#020617` | 文字`#525252` | 文字`rgba(2,6,23,0.3)` |

> **注**：`link` / `link-dark` 是**纯内联文字**形态，**强制清零 padding 与高度**（`!px-0 !py-0 !h-auto`），不受 `size` 影响——所以无论传 `size="default"` / `size="sm"`，按钮都是纯文字尺寸，方便在表格操作列、行内提示等场景与周围文字基线对齐。如果需要给 link 增加点击热区，请用外层包装容器自行加 `padding`。

### 4.2 尺寸

| size | 高度 | padding |
|------|------|---------|
| `claw-lg` / `lg` | 40px | px-6 |
| `claw` / `default` | 36px | px-6 |
| `claw-sm` / `sm` | 32px | px-4 |
| `icon` | 36×36 | — |
| `icon-sm` | 32×32 | — |

### 4.3 约束

- 同行所有控件高度必须一致（如 Input h-9 + Button h-9）
- disabled 态有 `cursor-not-allowed`，不用全局 `opacity-50`
- **刷新按钮标准写法**: `<Button variant="claw-outline" size="icon" className="w-9 h-9">`
- **表格操作列**：操作列必须使用 `<TableActionCell>` 组件包裹，内部按钮**必须**显式声明 `variant="link"`（品牌蓝文字按钮 `#355EF1`）。`TableActionCell` 内置 `flex items-center gap-6` 容器，**操作项间距固定 24px**，且与表头 `<TableHead>` 的 `px-4` 完全对齐，业务侧无需再手写外层 `<div>` wrapper。禁止省略 variant（会得到默认 claw-primary 实心按钮），禁止使用 outline、default、ghost、link-dark 或自定义样式。

```tsx
import { TableActionCell } from "@/components/ui/table";

// ✅ 正确：直接把 Button 平铺为 children，TableActionCell 自动应用 flex + gap-6 + 左对齐
<TableActionCell>
  <Button variant="link" onClick={onEdit}>编辑</Button>
  <Button variant="link" onClick={onView}>查看详情</Button>
  <Button variant="link" onClick={onDelete} disabled>删除</Button>
</TableActionCell>

// ✅ 删除按钮也统一蓝色 link，**不再用红色覆盖**（语义差异由二次确认 Dialog 承担）
<TableActionCell>
  <Button variant="link" onClick={onEdit}>编辑</Button>
  <Button variant="link" onClick={onDelete}>删除</Button>
</TableActionCell>

// ✅ 给内置 flex 容器追加 className（如固定高度让按钮组高度一致）
<TableActionCell actionsClassName="h-5">
  <Button variant="link">终端</Button>
  <Button variant="link">关机</Button>
</TableActionCell>

// ✅ 特殊布局（多行 / 自定义 wrapper）：设 rawChildren 关闭内置 flex 容器
<TableActionCell rawChildren>
  <div className="grid grid-cols-2 gap-2">...</div>
</TableActionCell>

// ❌ 错误：再嵌套 <div className="flex items-center gap-6"> wrapper（多余且会与内置容器叠加）
<TableActionCell>
  <div className="flex items-center gap-6">
    <Button variant="link">编辑</Button>
  </div>
</TableActionCell>

// ❌ 错误：省略 variant 会得到 claw-primary 实心纯黑按钮
<TableActionCell>
  <Button onClick={onEdit}>编辑</Button>
</TableActionCell>

// ❌ 错误：禁止再使用 link-dark
<TableActionCell>
  <Button variant="link-dark" onClick={onEdit}>编辑</Button>
</TableActionCell>
```

### link 四种状态（操作列按钮的标准色阶，对齐 button.tsx variant="link"）

| 状态 | 文字色 | 效果 |
|------|--------|------|
| Normal | `#355EF1` | 品牌蓝文字，无背景无边框 |
| Hover | `#355EF1` + 下划线 | 鼠标移入加下划线 |
| Active/Click | `#0a226f` | 点击反馈：深蓝 |
| Disabled | `#d0dafa` | 浅蓝灰，保留色相但大幅降弱（PR #443，2026-06） |

> **历史变更**：v2026.05 之前 TableActionCell 操作列约定 `variant="link-dark"`（黑色文字），已弃用。新规范一律改为 `variant="link"`（品牌蓝），与 Ant Design 等主流后台风格对齐。如发现存量页面里 TableActionCell 内还写着 `variant="link-dark"`，按"触达即同步"机制顺手换成 `variant="link"`。
>
> **2026-06 disabled 色调整**：之前 disabled 用 `rgba(20,71,230,0.4)`（40% 透明品牌蓝），与可用态对比不够明显，让人误以为还能点。改为 `#d0dafa`（浅蓝灰）后，保留蓝色色相暗示"这是个链接动作"，但视觉权重大幅降低，与"取消"等可用 link 形成清晰层级。
>
> **为什么 TableActionCell 不自动应用 link 样式？** Tailwind v4 + CVA 生成的 utility class specificity 相同（均为 0,1,0），父级选择器 `[&_[data-slot=button]]:text-...` 无法稳定覆盖 Button 自身 `variant` 携带的色值/背景/边框。强行用 `:where()` 降权又会破坏业务覆盖优先级。最稳的方式是要求业务侧显式声明 `variant="link"`。

#### link 调用规则（强制）

- **表格操作列内的所有按钮**（编辑、删除、保存、取消、详情、查看进度、重试...）**必须**使用 `<Button variant="link">`，由组件统一管控四态色值。
- **取消 + 保存** 同属一组操作，**必须**使用同一 `variant="link"`，色值一致；不允许"取消"用 `link-dark`、"保存"用 `link` 这种混搭。
- **禁止**在调用方 className 里硬编码 `text-[#1447E6]` / `text-[#355EF1]` / `text-[#A3A3A3]` / `text-[var(--text-weak)]` / `text-[var(--text-muted)]` 等覆盖 link variant 默认色 —— 后续要全局换色或调整 disabled 时无法统一。
- **禁止**在调用方写 `disabled:text-...`（如 `${disabled ? "text-[#A3A3A3] pointer-events-none" : "text-[#1447E6]"}`），传 `disabled` prop 即可，颜色由 button.tsx variant="link" 的 `disabled:text-[#d0dafa]` 自动接管。
- **禁止**在 `<Button variant="link">` 上写 `h-auto px-0`：variant 已带 `!h-auto !px-0 !py-0`（带 `!` 优先级最高），className 里再写一遍是冗余。需要改字号只写 `text-[12px]` 或 `text-[13px]`。
- 仅当**确实需要次要灰色 link**（如面包屑、辅助返回链接），可用 `variant="link-dark"` 表达，**不要**用 className 强行覆盖 `variant="link"` 的颜色。

#### 反例 → 正例

```tsx
// ❌ 反例：硬编码颜色 + h-auto px-0 冗余 + 三元覆盖 disabled
<Button variant="link" size="sm" className={`h-auto px-0 text-[12px] ${disableSave
  ? "text-[#A3A3A3] pointer-events-none"
  : "text-[#1447E6]"}`} disabled={disableSave} onClick={handleSave}>保存</Button>

// ✅ 正例：只声明字号，颜色 + disabled 全交给 variant
<Button variant="link" size="sm" className="text-[12px]" disabled={disableSave} onClick={handleSave}>保存</Button>
```

### 4.4 Plain 普通按钮（弹窗内筛选按钮）

**用途**：弹窗（Dialog）内的分类筛选切换按钮，交互风格与 §10.5 Tab 切换卡一致。

**四种状态（与 Tab 切换卡对齐）：**

| 状态 | 背景 | 边框 | 文字 |
|------|------|------|------|
| **Normal** | `#ffffff` | `#EAEEF4` | `#020617` |
| **Hover** | `#ffffff` | `#020617` | `#020617` |
| **Active（选中）** | `#020617` | `#020617` | 白色 |
| **Disabled** | `#ffffff` | `#EAEEF4` | `rgba(0,0,0,0.3)` |

**使用方式**：通过 `data-state="active"` 标记选中态。

```tsx
import { Button } from "@/components/ui/button";

// 弹窗内分类筛选按钮组
<div className="flex items-center gap-2 flex-wrap">
  {categories.map((cat) => (
    <Button
      key={cat.id}
      variant="plain"
      size="sm"
      data-state={activeCategory === cat.id ? "active" : undefined}
      onClick={() => setActiveCategory(cat.id)}
    >
      {cat.name}
    </Button>
  ))}
</div>
```

**适用场景**：
- 弹窗内的分类筛选（如技能分类、标签筛选）
- 需要多选/单选切换的按钮组
- 任何需要 active 态为黑底白字的切换场景

**与 Tab 切换卡的区别**：
- Tab 切换卡（§10.5）用原生 `<button>` 实现，适用于页面级分类筛选
- Plain 按钮用 `<Button variant="plain">` 实现，适用于弹窗内筛选，带有 Button 组件的标准圆角和尺寸

### 4.5 SmallIconStateButton（小图标按钮）

**文件**: `client/src/components/ui/button.tsx`（owner: miekoyychen）

用于列表行内的迷你操作按钮（如"添加"、"移除"），带图标 + 文字。

| 属性 | 说明 |
|------|------|
| 高度 | `h-6`（24px） |
| 圆角 | `rounded-[4px]` |
| padding | `px-2` |
| 字号 | `text-xs font-medium` |
| 图标 | `w-3 h-3`，与文字 `gap-1.5` |

| state | 边框 | 背景 | 文字 | hover |
|-------|------|------|------|-------|
| `default` | `#D4D4D4` | 白色 | `#0A0A0A` | `border-[#C9C9C9] bg-[#FAFAFA]` active:`bg-[#F5F5F5]` |
| `disabled` | `#D4D4D4` | 白色 | `#A3A3A3` | — |

**用法**:
```tsx
import { SmallIconStateButton } from "@/components/ui/button";

<SmallIconStateButton icon={Plus} label="添加" state="default" />
<SmallIconStateButton icon={Minus} label="移除" state="disabled" />
```

---

## 5. Input 组件

**文件**: `client/src/components/ui/input.tsx`

| 状态 | 边框 | 其他 |
|------|------|------|
| 默认 | `border-border` (#EAEEF4) | `rounded-[4px] h-9 px-3 text-[#020617]` |
| hover | `border-[#355EF1]` | — |
| focus | `border-[#355EF1]` | **无 ring、无 shadow** |
| 报错 | `border-[#d42a1e]` | — |
| disabled | `border-border` | `bg-[#f3f3f4] text-[var(--text-weak)]` |
| placeholder | — | `text-[var(--text-weak)]` (#94A3B8) |

---

## 6. 筛选/选择面板组件家族（Filter Panel Components）

> **预览页**: `/filter-panel-preview`（实现：`client/src/pages/admin/ComponentPreview.tsx`）
> **设计原则**：按数据结构同构性分类——同构的合并为变体，异构的保持独立。

### 6.0 组件总览

经过 v2 重构后，筛选/选择面板分为 3 类：

| 类别 | 组件 | 位置 | 数据结构 |
|------|------|------|---------|
| **可合并组（同构）** | `Select`（含 `SearchableSelect` / `FilterMultiSelect`） | `ui/select.tsx` | `Option[]` |
| | `TreeSelect`（含 `button` / `filter-icon` 两变体） | `ui/tree-select.tsx` | `TreeNode[]` |
| | `ScopeSelect`（含 `instant` / `confirm` 两模式） | `components/ScopeSelect.tsx` | `ScopeGroup[]` + 全部用户段 |
| **独立组件（异构）** | `GroupSelect` | `components/GroupSelect.tsx` | `UserGroup[]` + source 分桶 + 聚合 |
| | `TokenValueEditor` | `components/policy/TokenValueEditor.tsx` | mode + 数值（非列表） |
| | `DropdownMenu` / `MoreActionsDropdown` | `ui/dropdown-menu.tsx` 等 | 命令菜单 |
| **底层骨架** | `SelectPanel` | `ui/select-panel.tsx` | 三段式骨架 |
| | `FilterTrigger` | `ui/filter-trigger.tsx` | 触发器统一组件 |

> ⚠️ `_internal/` 目录下存放被新组件 wrapper 封装的旧实现（`TableHeaderFilter` / `TableHeaderTreeFilter` / `TreeSelectFilter` / `ScopeFilterDropdown` / `ScopeEditPopover`），**业务侧禁止直接 import**。

### 6.1 通用视觉规范（所有面板共享）

| 项 | 规范 |
|----|------|
| 面板外层 | `rounded-[4px]` + 无 border + `shadow-[var(--shadow-popover)]` |
| 内容区 padding | `p-2`（四边 8px） |
| 搜索框到选项间距 | 固定 8px |
| 选项行 | `h-8 px-3 rounded-[6px]` |
| 选项行间距 | `space-y-0.5`（2px） |
| **选中态** | `bg-[var(--bg-brand-selected)]` 蓝色淡背景；**文字保持 secondary 灰色，不变蓝、不加粗** |
| Hover 背景 | `hover:bg-[var(--bg-grey-hover)]` |
| **全选 Checkbox 三态** | 全选 ✅ checked、部分 ⎯ indeterminate（横线）、无 ☐ unchecked |
| Footer 容器 | `mx-2 border-t border-[#EAEEF4] py-2` |
| Footer 已选文字 | `<MetaText>` 默认 `tone="muted"` `#64748B` |
| 触发器 hover 描边 | `hover:border-blue-500` |
| 阴影 token | `--shadow-popover: 0 0 2px rgba(0,0,0,.1), 0 4px 16px rgba(0,0,0,.12)` |

### 6.2 Select（扁平 Option[] 列表）

**文件**：`ui/select.tsx`

#### Trigger
- 与 Input 完全一致：`h-9 rounded-[4px] border-border`，hover/open 变 `border-[#355EF1]`

#### Content（下拉面板）
- `bg-white rounded-[4px]` 无 border
- 阴影 = 通用面板阴影 token
- padding `p-2` + `space-y-0.5`

#### Item（基础）
- `h-8 rounded-[6px] px-3` hover `bg-[var(--bg-grey-hover)]`
- 选中态：`bg-[var(--bg-brand-selected)]`（背景）+ 文字保持 secondary 灰色（不变蓝不加粗）+ 蓝色 ✓ 勾号

#### 三种变体

| 变体 | API | 数据 | 适用场景 |
|------|-----|------|---------|
| **simple** | `<Select> + <SelectItem>` | `<SelectItem>` children | 表单基础静态单选 |
| **searchable** | `<SearchableSelect options={...} value onChange>` | `Option[]` 扁平 | VPC / 子网 / 主机等长列表单选 |
| **filter-multi** | `<FilterMultiSelect title options value onChange>` | `Option[]` 扁平 | 表头 filter-icon 触发 + 多选 + confirm 提交 |

### 6.3 TreeSelect（树形 TreeNode[] 列表）

**文件**：`ui/tree-select.tsx`

#### 数据结构
```ts
interface TreeNode {
  id: string;
  name: string;
  children?: TreeNode[];
  path?: string;  // 仅 button 变体用于底部面包屑
}
```

#### 两种变体（通过 `triggerVariant` 区分）

| triggerVariant | API | 适用场景 |
|----------------|-----|---------|
| **`"button"`（默认）** | `<TreeSelect nodes value onChange>` | toolbar 按钮触发 + confirm，底部带面包屑 |
| **`"filter-icon"`** | `<TreeSelect triggerVariant="filter-icon" title nodes value onChange>` | 表格列头漏斗图标触发 + confirm |

#### 子节点缩进
- 每层 16px：`paddingLeft: level * 16 + 12`
- **必须**：每个 `TreeNodeItem` 最外层 `<div>` 加 `space-y-0.5` 让子节点间也有 2px 间距（防止"父-第一个子节点"间距塌陷）

### 6.4 ScopeSelect（应用范围选择面板）

**文件**：`components/ScopeSelect.tsx`

#### 数据结构（两段式）
```ts
interface ScopeGroup { id: string; name: string; parentId?: string | null; }
type ScopeType = "all" | "groups";
// 面板内置 "全部用户" + "按组织" 两段
```

#### 两种模式（通过 `mode` 或 `scope` 推断）

| mode | API | 触发器 | 提交 | 适用场景 |
|------|-----|--------|------|---------|
| **`"instant"`（默认）** | `<ScopeSelect groups value onChange>` | 嵌入式（无） | 即时 | 列表筛选区即时多选 |
| **`"confirm"`**（传 `scope` 自动） | `<ScopeSelect scope selectedGroupIds groups onConfirm>` | badge-pencil 行内徽章 | confirm | 行内编辑应用范围 |

#### confirm 模式额外特性
- 顶部 SegmentGroup 切换 "全部用户" / "按组织"
- footer 三按钮：取消 / 保存
- 支持 `trigger`、`showBadges`、`scopeLabels`、`maxVisibleBadges` 自定义

### 6.5 GroupSelect（独立组件 — 不合并）

**文件**：`components/GroupSelect.tsx`

#### 为什么独立
数据结构 `UserGroup[]` 包含 `source` 多桶（部门 / 用户组 / 自定义）+ `parentId` 树 + `readonly` + `createdAt` 等业务字段，并自带"父子级联自动聚合/展开"算法，远超通用 Select/TreeSelect 的数据复杂度。

#### API
```tsx
<GroupSelect
  groups={UserGroup[]}
  selectedIds={string[]}
  onChange={(ids) => void}
  sourceFilter={["oneid-dept", "manual"]}  // 仅展示哪些 source
  enableAggregation={true}                  // 父子级联聚合（默认开）
  variant="default"                         // 或 "confirm"
  disabledIds={string[]}                    // 灰显项
  disabledTooltip="..."
/>
```

#### 视觉
- 触发器：Input 式 + 已选项 Badge 列表（带 X 按钮单独移除）
- hover 时右侧出现"清空"圆形按钮，无 hover 显示 ChevronDown
- 选中态文字 13px / `--text-secondary` / 不加粗（与 ScopeSelect 对齐）

### 6.6 TokenValueEditor（独立组件 — 不合并）

**文件**：`components/policy/TokenValueEditor.tsx`

#### 为什么独立
非列表选择，而是 mode + 数值输入，数据结构与所有 Select 家族都不兼容。

#### API
```tsx
<TokenValueEditor
  mode={"custom" | "unlimited"}
  valStr={string}
  onCommit={(mode, valStr) => void}
/>
```

#### 视觉
- 触发器仿 Select 单行下拉
- Popover 内顶部 SegmentGroup（无限制 / 自定义），选中"自定义"显示 Input
- 底部 confirm 提交（取消 / 确认）

### 6.7 DropdownMenu / MoreActionsDropdown（独立组件 — 不合并）

**文件**：`ui/dropdown-menu.tsx`、`ui/more-actions-dropdown.tsx`

#### 为什么独立
命令菜单（`MenuItem[]` 含 `icon` + `onClick` + `variant`），不是筛选/选择，与 Select 家族数据语义完全不同。

#### 四种变体

| 变体 | API | 触发器 |
|------|-----|--------|
| **more-icon** | `<MoreActionsDropdown items />` | 三点图标 |
| **more-text** | `<MoreActionsDropdown triggerType="text" items />` | 文字"更多" |
| **icon-trigger** | `<DropdownMenu><DropdownMenuTrigger>...自定义图标 button` | 自定义 icon |
| **button-trigger** | `<DropdownMenu><DropdownMenuTrigger><Button>...` | 按钮+箭头 |

#### Item 视觉
- `h-8 px-3 rounded-[6px] hover:bg-[var(--bg-grey-hover)]`
- `variant="destructive"` → 文字红色
- 支持 `icon` 前置 + `separatorBefore` 组织分隔

### 6.8 SelectPanel（底层骨架 — 不参与合并）

**文件**：`ui/select-panel.tsx`

#### 用途
搜索 + 列表 + footer 的可组合三段式壳子，所有面板组件可基于它构建（当前 SearchableSelect、TreeSelect 等都遵循其视觉节奏）。

#### API
```tsx
<SelectPanel
  commitMode={"instant" | "confirm"}
  showSearch={boolean}
  searchPlaceholder
  searchValue, onSearchChange
  showFooter={boolean}
  footerLeft={ReactNode}
  onConfirm, onCancel
  maxHeight={number}
>
  <SelectPanelItem selected onClick>...</SelectPanelItem>
</SelectPanel>
```

### 6.9 FilterTrigger（底层骨架 — 不参与合并）

**文件**：`ui/filter-trigger.tsx`

#### 三种变体

| variant | 适用 |
|---------|------|
| **`"button"`** | Input 式，带 placeholder（`<Select>` / `<SearchableSelect>` 触发器风格） |
| **`"icon"`** | 表头漏斗图标，未激活灰色，激活蓝色 |
| **`"badge-pencil"`** | 行内徽章 + 铅笔图标（`<ScopeSelect mode="confirm">` 默认触发器） |

### 6.10 业务接入指引

#### 禁止
- ❌ 直接 import `_internal/` 下任何组件
- ❌ 给筛选面板加自定义 `className` 覆盖选项行高度、圆角、选中色等基础样式
- ❌ 自行实现"全部用户 + 按组织"两段式面板（必用 `ScopeSelect`）
- ❌ 自行实现树形多选 + 父子聚合（必用 `GroupSelect`）

#### 必须
- ✅ 所有列表多选场景的"全选"行用 Checkbox **三态**（含 indeterminate）
- ✅ 选中态背景蓝、文字仍 secondary 灰色（不再设 `text-blue-500 font-medium`）
- ✅ 树形面板每层节点 `space-y-0.5` 间距
- ✅ Footer 已选数量用 `<MetaText>` 默认 muted 灰

#### 选型决策树
```
需求是？
├─ 单选短列表  → <Select>
├─ 单选长列表+搜索 → <SearchableSelect>
├─ 多选+表头漏斗 → <FilterMultiSelect>
├─ 树单选 → <TreeSelect>（toolbar 用 button、表头用 filter-icon）
├─ 范围（含"全部用户"段） → <ScopeSelect>（即时用默认、行内编辑传 scope）
├─ 组织多选+source 分桶 → <GroupSelect>
├─ Token 配额值编辑 → <TokenValueEditor>
└─ 操作菜单 → <DropdownMenu> / <MoreActionsDropdown>
```

#### Code Review 注意事项

> **背景**：v2 重构时，10 个旧组件通过 sed 批量改名映射到 6 个新组件入口。sed 仅替换组件名和 import 路径，**无法自动补充新增的必填 prop**。Code Review 时请重点检查以下"易漏点"：

##### 1. ⚠️ 表头列内的 `<TreeSelect>` 必须显式传 `triggerVariant="filter-icon"`

**原因**：旧的 `TableHeaderTreeFilter` 本身**只有** filter-icon 一种形态，无需指定；新的 `TreeSelect` 默认变体是 `"button"`（toolbar 按钮形态）。sed 替换名称后，原本表头列里的漏斗图标会"消失"，变成 toolbar 按钮，与表头视觉严重不符。

**触发场景**：以下任一条件命中即需检查
- `<TreeSelect>` 出现在 `<TableHead>` / `<th>` / 表头单元格内
- `<TreeSelect>` 传了 `title` prop（filter-icon 变体专属，button 变体不接受）
- `<TreeSelect>` 出现在表格列定义（columns config）的 header 渲染中

**正反例**：
```tsx
// ❌ 错误：title 传了但缺 triggerVariant，仍走 button 变体（toolbar 按钮）
<th>
  <TreeSelect title="部门" nodes={...} value={...} onChange={...} />
</th>

// ✅ 正确：表头列必须显式 filter-icon
<th>
  <TreeSelect triggerVariant="filter-icon" title="部门" nodes={...} value={...} onChange={...} />
</th>

// ✅ 正确：toolbar 区域无需指定，默认 button 变体
<div className="toolbar">
  <TreeSelect nodes={...} value={...} onChange={...} allLabel="全部部门" />
</div>
```

**自查命令**：
```bash
# 找出所有传了 title 但没传 triggerVariant 的 TreeSelect 调用
grep -rB 1 -A 5 "<TreeSelect" client/src/pages | grep -E "TreeSelect|title|triggerVariant"
```

##### 2. ⚠️ `<FilterMultiSelect>` / `<ScopeSelect>` 推荐用新 prop 名

虽然旧 prop 名（`selectedValues` / `selectedKeys` / `onConfirm`）通过兼容层仍可用，但 IDE 会显示 `@deprecated` 提示。新代码统一用：

| 组件 | 旧 prop（deprecated） | 新 prop（推荐） |
|------|----------------------|----------------|
| `<FilterMultiSelect>` | `selectedValues` `onConfirm` | `value` `onChange` |
| `<ScopeSelect mode="instant">` | `selectedKeys` | `value` |
| `<TreeSelect triggerVariant="filter-icon">` | `onConfirm` | `onChange` |

##### 3. ⚠️ 禁止 import `_internal/` 下的任何文件

`_internal/TableHeaderFilter` / `_internal/ScopeEditPopover` 等是被新组件 wrapper 内部封装的实现，**业务代码任何 `from "@/components/_internal/..."` 的 import 都视为违规**，必须替换为对应的新组件入口。

**自查命令**：
```bash
grep -rn "from\s*['\"]@/components/_internal/" client/src/pages
# 期望输出为空
```

##### 4. ⚠️ `<ScopeSelect>` 的 mode 推断规则

`<ScopeSelect>` 通过 `mode` 或 `scope` 字段自动推断模式：
- 传 `mode="instant"` 或 **不传 `scope`** → instant 模式（嵌入式即时多选）
- 传 `mode="confirm"` 或 **传 `scope`**（即使是 `"all"`）→ confirm 模式（badge-pencil 触发）

**易错**：如果业务场景需要 confirm 模式但忘了传 `scope`，会静默落到 instant 模式（嵌入式无触发器，视觉上完全错误）。

```tsx
// ❌ 静默走 instant 模式
<ScopeSelect groups={...} selectedGroupIds={...} onConfirm={...} />

// ✅ 显式 confirm
<ScopeSelect mode="confirm" groups={...} selectedGroupIds={...} onConfirm={...} />

// ✅ 传 scope 也会自动走 confirm
<ScopeSelect scope={scope} groups={...} selectedGroupIds={...} onConfirm={...} />
```

---

## 7. Dialog 组件

**文件**: `client/src/components/ui/dialog.tsx`

| 属性 | 值 |
|------|-----|
| 圆角 | `12px`（`rounded-[12px]`） |
| 遮罩 | `rgba(0,0,0,0.45)` |
| 阴影 | 三层阴影（见 L3） |
| 分割线 | **无**（Header/Footer 均无分割线） |
| 标题 | `16px font-semibold rgba(0,0,0,0.88)` |
| 关闭按钮 | `20px #7b818f` 右上角 `top-5 right-5` |
| Header | `pt-6 pb-3 -mx-6 px-6` |
| Footer | `pt-2 pb-6 -mx-6 px-6 mt-2` 右对齐（无分割线，与 `DialogContent` 的 `pb-6` 协同保持视觉间距） |

> **0608 修订**：Dialog 圆角统一为 `12px`（之前文档误标 8px，与代码 `rounded-[12px]` 不一致）。Footer padding 改为 `pt-2 pb-6 mt-2`（移除分割线 `border-t`），与 `DialogContent` 的 `pb-6` 协同。

### 7.1 内嵌组件强制规范（不可违反）

> 对话框 / 弹窗内出现的任何基础组件，**必须直接复用本设计 SKILL 中已定义的规范样式，禁止在弹窗内重新编造一套样式**。

| 组件 | 必须引用 | 关键约束 |
|------|----------|----------|
| Input | `client/src/components/ui/input.tsx`（见第 5 节） | 默认状态**禁止加底色**（无 `bg-gray-*` / `bg-[#FAFAFA]` 等），统一 `border-border` + 白底 |
| Select / 下拉 | `client/src/components/ui/select.tsx`（见第 6 节） | Trigger 与 Input 完全一致，默认状态**禁止加底色**；`Content` 面板沿用统一阴影 |
| Table / 表格 | 见第 11.6 节 Table 表格组件规范 | 表头、行高、分割线、空状态等必须沿用全局 Table 规范，禁止在弹窗内自定义新表格样式 |

**强制条款**：

1. 弹窗内的 Input、Select（下拉）、Table 三类组件**必须** `import` 自 `@/components/ui/*`，禁止以 `<input>` / `<select>` / `<table>` 原生标签 + 临时 class 的方式拼凑。
2. **严禁**为弹窗内的 Input、Select 重新调色或重写样式；尤其：
   - **默认状态禁止加任何底色**（如 `bg-gray-50`、`bg-[#F5F5F5]`、`bg-[#FAFAFA]` 等），必须保持白底 + `border-border`。
   - **禁用（disabled）状态禁止再添加 hover 样式**（不允许 `disabled:hover:*`、不允许在 disabled 下出现边框变蓝、底色加深等任何 hover 反馈）；disabled 视觉锁死为 `border-border bg-[#f3f3f4] text-[#b0b6c3] cursor-not-allowed`。
3. 弹窗内 Table 必须沿用全局 Table 表头 / 行 / 边框 / 空状态样式，禁止重新定义表头底色、行高、分割线颜色。
4. 若弹窗内确有特殊视觉需求，**必须在本 SKILL 文档中扩展规范**后再使用，禁止在业务代码内单点编造样式绕过规范。

### 7.2 Drawer / 右侧抽屉（管控端详情类）

**文件**: `client/src/components/ui/drawer.tsx`  
**适用场景**: 管控端详情查看 / 局部配置编辑，如 `OpenClawMonitor.tsx` 的「Agent 详情」抽屉。

> 右侧详情抽屉必须优先使用 shadcn `Drawer`（`direction="right"`），禁止继续手写 `fixed inset-0` + 自定义遮罩 + `shadow-lg` 结构。抽屉本体圆角固定为 `0`。

| 区域 | 规范 |
|------|------|
| Root | `<Drawer direction="right" open={open} onOpenChange={...}>` |
| Content | `w-[480px] sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0`；信息密度特别高时可扩到 `560px`，需说明原因 |
| Header | `flex flex-row items-center justify-between gap-4 p-4 bg-background text-left`；**不加底部分割线** |
| Title | `DrawerTitle asChild` + `PanelTitle`；不要手写 `text-lg font-semibold` |
| Header actions | 仅图标按钮用 `Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950"`；默认色对齐 Typography `primary`（`#171717`）；按钮间距 `gap-1` |
| Body | 优先使用 `<DrawerBody>`；等效样式为 `flex-1 overflow-y-auto bg-background [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`，内部 `p-4 space-y-6`；抽屉内容区必须隐藏滚动条但保留滚动能力 |
| 对象标题 | 使用 `PanelTitle`（较重要）或 `BodyMedium`（普通对象名）；下方 ID 使用 `CodeText`，链接使用 `MetaText tone="brand"` |
| 组织标题 | 使用 `MetaText`，如「已应用模型（0）」；右侧轻量操作也使用 `MetaText as="button" tone="brand"` |
| 空状态 | 使用 `MetaText tone="weak"` + `border border-dashed`，不要手写大字号灰字；添加入口默认放组织标题右侧，除非设计明确要求框内引导 |

#### 快速 Checklist

- 右侧详情抽屉：`Drawer direction="right"` + `DrawerContent rounded-none`。
- 抽屉宽度默认 `480px`；只有复杂高密度内容才扩到 `560px`。
- Header 不加底部分割线；右上角图标按钮用 `ghost`，不使用 `outline`。
- Body 优先使用 `<DrawerBody>`；如需手写容器，必须包含 `overflow-y-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`，隐藏滚动条但保留滚动能力。
- 标题、组织、正文、ID、链接必须映射到 Typography 组件，不直接手写文字 class。
- 添加入口默认放组织标题右侧，使用 link 蓝色文字按钮；空态框内保留弱提示文案。
- 编辑态确认按钮统一 `dialog-confirm`，不使用 primary / 渐变主按钮。

#### 详情抽屉内容模式

1. **避免装饰性大 icon**：详情抽屉首屏信息以文本为主，不放蓝色圆形机器人 / 资源 icon；除非 icon 是识别对象类型的必要信息。
2. **列表信息优先紧凑化**：仅展示名称的重复列表（如已安装技能）不要一项一张大卡片；使用 `Table density="compact"` 或紧凑信息块承载。
3. **组织添加入口**：添加模型 / 添加通道等轻量入口默认放在组织标题右侧，使用 `MetaText as="button" tone="brand"` + `Plus` 图标，颜色与「编辑凭证」等轻量操作统一；空态框内保留 `MetaText tone="weak"` 提示文案。若设计明确要求框内引导，可例外放入虚线空态框内。
4. **凭证 / Key-Value 信息块**：使用聚合卡片，不要把操作按钮放在右下角。
   - 外层：`border-t border-[#e5e5e5] bg-muted/30 p-3`
   - 内层：`rounded-[4px] border border-[#e5e5e5] bg-background overflow-hidden`
   - 顶部：左侧 `MetaMedium`（如「凭证信息」），右侧 `MetaText as="button" tone="brand"`（如「编辑凭证」）
   - 行布局：`grid grid-cols-[112px_minmax(0,1fr)] items-center gap-3 px-3 py-2`
   - 字段名：`MetaText`；字段值：`CodeText tone="emphasis"`
   - 密钥可见性 icon 紧跟值文本后方，使用 `text-gray-500 hover:text-gray-900`，不要贴到整行最右。
   - 编辑态确认按钮（如「保存」）使用 `Button variant="dialog-confirm"`，颜色走 confirm 语义；不要使用默认 primary / `claw-primary`。
5. **内联编辑表单**：如「已应用模型」新增 / 替换态、「已接入通道」新增态，视觉结构必须与凭证编辑保持一致。
   - 外层：`bg-muted/30 p-3`
   - 内层：`rounded-[4px] border border-[#e5e5e5] bg-background overflow-hidden`
   - 顶部标题：`MetaMedium`（如「模型配置」/「通道配置」）+ `border-b border-[#f0f0f0] px-3 py-2`
   - 字段行：`px-3 py-2 space-y-1.5`，字段之间用 `divide-y divide-[#f0f0f0]`
   - Select / Input：优先使用 `bg-background border-[#e5e5e5] h-8 text-xs`
   - 底部操作栏：`border-t border-[#f0f0f0] px-3 py-2`；取消用 `ghost`，保存用 `dialog-confirm`
   - 禁止使用蓝色激活边框（如 `border-[#355EF1]`）包裹整块编辑表单。

#### 推荐写法

```tsx
<Drawer direction="right" open={open} onOpenChange={setOpen}>
  <DrawerContent className="data-[vaul-drawer-direction=right]:w-[480px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
    <DrawerHeader className="flex flex-row items-center justify-between gap-4 p-4 bg-background text-left">
      <DrawerTitle asChild>
        <PanelTitle as="h2">Agent 详情</PanelTitle>
      </DrawerTitle>
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950" aria-label="刷新">
          <RefreshCw className="w-4 h-4" />
        </Button>
        <DrawerClose asChild>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950" aria-label="关闭">
            <X className="w-4 h-4" />
          </Button>
        </DrawerClose>
      </div>
    </DrawerHeader>

    <DrawerBody>
      <div className="p-4 space-y-6">
        <section className="min-w-0 space-y-1.5">
          <PanelTitle as="div" className="truncate leading-tight">对象名称</PanelTitle>
          <div className="flex items-center gap-2">
            <CodeText>ins-xxxx</CodeText>
            <MetaText as="button" tone="brand">去控制台管理</MetaText>
          </div>
        </section>

        <section>
          <MetaText as="div" className="mb-2">已安装技能（7）</MetaText>
          <div className="overflow-hidden rounded-[4px] border border-[#e5e5e5] bg-background">
            <Table density="compact">
              <TableHeader><TableRow><TableHead>技能名称</TableHead></TableRow></TableHeader>
              <TableBody>
                <TableRow><TableCell><MiniBodyText>feishu-doc</MiniBodyText></TableCell></TableRow>
              </TableBody>
            </Table>
          </div>
        </section>
      </div>
    </DrawerBody>
  </DrawerContent>
</Drawer>
```

**禁止事项**：
- 禁止右侧详情抽屉使用 `shadow-lg` 手写浮层；阴影与动画交给 `DrawerContent`。
- 禁止 Header 图标按钮使用 `outline` 边框态；详情抽屉头部操作一律 `ghost`。
- 禁止用单项大卡片堆叠纯文本列表；优先紧凑表格 / 紧凑信息块。
- 禁止在 Drawer 内散落 `text-xs text-[#737373]` / `text-sm text-[#0A0A0A]`，必须优先映射到 Typography 组件。
- 禁止将 Drawer 内编辑态确认按钮做成默认 primary / 渐变主按钮；确认动作使用 `dialog-confirm`。
- 禁止无故隐藏组织标题右侧的添加入口；添加模型 / 添加通道等入口默认保留在标题右侧，除非设计明确要求框内引导。

### 7.3 NormalDialog / AlertDialog 全局强制规范（19 条）

> **适用范围**：全平台（管控端 + 用户端）所有弹窗/对话框。如有冲突，以本节为准。

**弹窗三段式结构**：

| 区域 | 组件 | 固定高度 | 说明 |
|------|------|----------|------|
| **Header** | `<DialogHeader>` | 自适应（`py-4` 上下 16px） | 标题左对齐 + 关闭按钮右对齐，垂直居中 |
| **Content** | `<DialogBody>` 或直接子元素 | 自适应（可滚动） | 表单/列表/信息内容区；内容左边缘必须与 Header 标题左边缘对齐，滚动容器只允许 `scrollbar-gutter: stable`，禁止 `stable both-edges` |
| **Footer** | `<DialogFooter>` | 65px | 按钮右对齐，垂直居中 |

**强制规范（不可违反）**：

1. 项目中的弹窗**只能使用 NormalDialog（普通弹窗）或 AlertDialog（警示弹窗）**两种变体，禁止自行产出新样式
2. 弹窗内的下拉组件、Input 组件、按钮组件等**必须使用项目规范组件样式**（`@/components/ui/*`），禁止自行编造
3. NormalDialog 和 AlertDialog **均不加 Header/Footer 分割线**
4. 弹窗 Footer **必须使用 `<DialogFooter>` / `<AlertDialogFooter>` 组件**，禁止自行用 `<div>` 拼凑底部按钮区
5. **Header 上下边距 `py-4`（16px），高度自适应**，标题左对齐，关闭按钮右对齐垂直居中。Header 内部纵向堆叠（`flex-col justify-center`）：
    - 如有二级说明文字（`DialogDescription` / `HelperText`），**超过 5 个字必须在标题下方折行显示**，禁止与标题水平排列
    - 二级说明使用 `HelperText`（中性灰 `#A3A3A3`），与标题间距 `gap-0.5`（2px）
6. **Footer 高度固定 65px**，按钮在 Footer 内**垂直居中对齐**，弹窗滚动时 Footer 高度**禁止发生任何变化**
7. Footer 按钮**右对齐**，取消在左确认在右；取消按钮统一使用 `variant="outline"`
8. 弹窗内的标题 / 说明 / 字段 Label 等**所有可见文字必须使用 `@/components/ui/Typography` 组件**，禁止硬编码 Tailwind 类替代
9. 如果弹窗内有 info 图标，**必须使用相同的 SVG**（统一使用 `lucide-react` 的 `Info` / `AlertTriangle` 等图标组件）
10. 弹窗左上角大标题旁边**不使用图标**，标题保持纯文字
11. 弹窗内的 Input 和 Select 组件统一使用**灰色边框白色底**（`border-[#E5E5E5] bg-white`），禁止加入灰色底色（如 `bg-gray-50`、`bg-[#FAFAFA]`）
12. 弹窗内容区间距规范（**只允许使用 Tailwind spacing token**，禁止硬编码像素值）：
    - 不同模块之间：**`space-y-4`**（16px）
    - Label 与下方元素（Input/Select/Textarea 等）之间：**`space-y-2`**（8px）
    - **严禁**两个元素紧贴（即间距为 0），所有相邻元素必须保留可见间距

    **弹窗内不同元素间距规范**：

    | 元素关系 | Token | 实际值 |
    |----------|-------|--------|
    | Header 与 Content | 由 `DialogHeader` 组件控制 | 组件内置 |
    | Content 与 Footer | 由 `DialogFooter` 组件控制 | 组件内置 |
    | 不同模块（字段组）之间 | `space-y-4` | 16px |
    | Label 与下方元素（Input/Select） | `space-y-2` | 8px |
    | Alert 与下方内容 | `space-y-4` | 16px |
    | Section 标题与 Section 内容 | `space-y-3` | 12px |
    | Checkbox/Switch 与其说明文字 | `space-y-1` 或 `mt-1` | 4px |
    | 表格/列表与上方说明 | `space-y-2` 或 `space-y-3` | 8–12px |
    | 步骤条与步骤内容 | `mb-4` 或容器 `space-y-4` | 16px |

12. **弹窗和对话框右上角必须有关闭按钮**（X 图标，`top-5 right-5`，`#737373` → hover `#0A0A0A`，尺寸 `size-5`）：
    - NormalDialog（`@/components/ui/dialog`）已内置该关闭按钮，**禁止移除**
    - `AlertDialog`（`@/components/ui/alert-dialog`）**未自带**关闭按钮，必须**手动**在 `AlertDialogContent` 内添加
13. **弹窗主按钮必须显式声明 `variant`**，禁止使用 `<Button>` 默认变体或自行编造样式：

    | 场景 | variant | 颜色 |
    |---|---|---|
    | 管控端 NormalDialog 主按钮 | `dialog-confirm` | 纯黑底 `#0A0A0A` |
    | 管控端 AlertDialog 主按钮（`AlertDialogAction`） | 组件已内置 `destructive`，**无需声明** | 红底 `#d42a1e` |
    | NormalDialog 内的危险按钮（删除/关闭等） | `destructive` | 红底 `#d42a1e` |
    | 用户端普通弹窗主按钮 | `tenant-primary` | 黑→蓝渐变（圆角胶囊） |
    | 用户端危险弹窗主按钮 | `tenant-destructive` | 红底（圆角胶囊） |
    | 取消按钮（管控端 / 用户端） | `outline` / `tenant-outline` | 白底 |

14. **弹窗内表单字段 Label 必须统一使用 `<MetaMedium as="label" tone="secondary">`**（来自 `@/components/ui/Typography`），必填星号用 `<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>`
15. **弹窗内"非主信息"的辅助文字必须统一使用 `<HelperText>`**（来自 `@/components/ui/Typography`），覆盖 8 类场景：输入框提示、Label 副标题、字符计数器、Switch 说明、空状态、表头副说明、占位文案、搜索空态
16. **弹窗内步骤条（Stepper / Steps）必须左对齐**，使用 `@/components/ui/stepper` 组件，禁止手写
17. **弹窗内空状态禁止使用无意义占位图片**，仅用纯文字（`HelperText`）展示；有标题时用 `CardTitle` + `HelperText`
18. **弹窗内所有文字/颜色/间距必须使用 Design Token**，禁止原生样式或硬编码：
    - 文字：通过 Typography 组件输出，禁止 `<p>` / `<span>` + 硬编码 className
    - 颜色：使用 CSS 变量 token 或 Typography `tone` prop，禁止写死 hex 或 Tailwind 调色板
    - 间距：使用 Tailwind spacing token，禁止 inline style 写死像素值
19. **弹窗内表格必须使用压缩型表格组件**（`<Table variant="compact">` 或 `<Table variant="elevated-white">` + compact 行高），禁止使用默认宽松行高表格：
    - 表头使用 `text-xs` 字号，表体使用 `text-sm` 字号
    - 禁止在弹窗内使用页面级全宽表格样式

**尺寸规范**（仅允许 4 档）：

| 档位 | 宽度 | 适用场景 |
|------|------|----------|
| S 小 | `sm:max-w-[420px]` | 简单确认、单字段输入、警示弹窗 |
| M 中 | `sm:max-w-[560px]` | 表单弹窗（3-6个字段）、发布/编辑 |
| L 大 | `sm:max-w-[720px]` | 复杂表单、含表格/列表、多列内容 |
| XL 加大 | `sm:max-w-[920px]` | 含多列数据表格的批量操作弹窗、Tabs + 列表管理 |

---

## 8. Checkbox 组件

**文件**: `client/src/components/ui/checkbox.tsx`

### 8.1 视觉参数

| 属性 | 值 |
|------|-----|
| 尺寸 | `size-4` (16px × 16px) |
| 圆角 | `rounded-[4px]` (4px) |
| 边框宽度 | `1px` |
| 默认边框 | `border-[var(--border-control)]` = `#C8CFDA` |
| 默认背景 | `bg-white` |
| 勾大小 | `size-3.5` (14px，CheckIcon) |
| transition | `transition-colors` |
| outline | `outline-none`（focus 时由 ring 提供视觉反馈） |

### 8.2 状态规范

| 状态 | 边框 | 背景 | 勾色 | 备注 |
|------|------|------|------|------|
| 默认 | `border-[var(--border-control)]` (#C8CFDA) | `bg-white` | — | 使用 token，不要写 `border-gray-200` |
| hover | `#1447E6` | `bg-white` | — | 边框变品牌蓝 |
| checked | `#1447E6` | `#355EF1` | `text-white` | 蓝底白勾 |
| **indeterminate**（半选） | `#1447E6` | `#355EF1` | `text-white` | 视觉同 checked，由 Radix `data-state` 控制 indicator |
| focus-visible | `#1447E6` | — | — | 额外加 `ring-2 ring-[#355EF1]/20` |
| disabled（默认） | `border-[var(--border-control)]` | `bg-[#f3f3f4]` | — | `cursor-not-allowed` |
| disabled（checked） | `border-[var(--border-control)]` | `bg-[#d3d6db]` | `text-white` | 勾保留但整体灰化 |

### 8.3 用法

```tsx
import { Checkbox } from "@/components/ui/checkbox";

// 单独使用
<Checkbox checked={value} onCheckedChange={setValue} />

// 与 Label 配合（间距 gap-2）
<div className="flex items-center gap-2">
  <Checkbox id="terms" />
  <Label htmlFor="terms">我已阅读并同意</Label>
</div>

// 半选状态（如表格全选）
<Checkbox checked="indeterminate" onCheckedChange={...} />
```

### 8.4 表格内使用

表格固定列内的 Checkbox 列：
- 列宽固定 `44px`（保证勾选区与列名 padding 一致）
- 必须使用 `<TableHead fixedShadow={false}>` 关闭阴影（仅最外侧列保留阴影，详见 §15.1）
- Checkbox 与表格 row hover 联动：行 hover 时 Checkbox 不需要单独的 hover 反馈

### 8.5 禁止事项

- 禁止使用 `border-gray-200` / 任意 hex 描边色：默认描边必须用 `border-[var(--border-control)]` token（详见 §28 token 规范）
- 禁止使用 `opacity-50` 表达 disabled 态（视觉无差异且降低 a11y）；统一用 `bg-[#f3f3f4]`
- 禁止自定义勾大小或位置（`size-3.5` + 居中已对齐设计稿）
- 禁止使用 `border-2` 等加粗边框（默认 1px 已对齐设计稿与 Radio）
- 禁止单独修改选中色：选中色与 RadioGroup / 品牌按钮全局保持 `#1447E6`/`#355EF1` 双色组合

---

## 9. Switch (Toggle) 组件

**文件**: `client/src/components/ui/switch.tsx`

| 状态 | 样式 |
|------|------|
| unchecked | `bg-[#d3d6db]` 轨道 |
| checked | `bg-[#355EF1]` 轨道 |
| thumb | 白色圆形 4px 内缩 |
| 尺寸 | `h-5 w-9` |

---

## 10. DatePicker 组件

**文件**: `client/src/components/ui/date-picker.tsx`

- Popover + Calendar 组合
- Trigger 与 Input 一致：`h-9 rounded-[4px] border-border`
- hover/focus/open: `border-[#355EF1]`，**无外层 shadow**
- 右侧日历图标 `text-[#b0b6c3]`
- Calendar 选中日: `bg-[#355EF1] text-white`

---

## 11. Tab 切换卡（筛选标签按钮）

> Figma: node 1086:6426 (ClawPro 项目设计)
> 用于分类筛选场景（如技能库分类、技能列表分类等）

**使用原生 `<button>` 实现**（不使用 Button 组件，避免内置 hover 样式干扰）

### 四种状态

| 状态 | 背景 | 边框 | 文字 | 说明 |
|------|------|------|------|------|
| **Active（选中）** | `#020617` | `#020617` | 白色 | 黑底+黑边+白字 |
| **Hover（悬停）** | `#ffffff` | `#020617` | `#020617` | 白底+黑边+黑字 |
| **Normal（默认）** | `#ffffff` | `#EAEEF4` | `#020617` | 白底+灰边+黑字 |
| **Disabled（禁用）** | `#ffffff` | `#EAEEF4` | `rgba(0,0,0,0.3)` | 白底+灰边+淡字 |

### 视觉参数

| 属性 | 值 |
|------|-----|
| 高度 | `32px` (h-8) |
| 圆角 | `4px` (rounded-[4px]) |
| 内边距 | `px-4 py-[10px]` |
| 字号 | `14px` |
| 字重 | Regular (400) |
| 间距 | 标签之间 `gap-2`（8px） |

### 代码示例

```jsx
<div className="flex items-center gap-2 flex-wrap">
  <button
    onClick={() => setCategory(cat.id)}
    className={`h-8 px-4 rounded-[4px] text-sm leading-[22px] tracking-[0.07px] border transition-colors ${
      isActive
        ? 'bg-[#020617] border-[#020617] text-white'
        : 'bg-white border-[#EAEEF4] text-[#020617] hover:border-[#020617]'
    }`}
  >
    {cat.name}
  </button>
</div>
```

**注意**：设计稿中 Active 态的颜色是 `#165DFC`，但在代码实现中统一映射到 `claw-primary` variant（使用纯黑背景）。如需精确还原设计稿的纯蓝色 Active 态，可使用 className 覆盖。

---

## 11.5 LineTabs（线性标签页 / 下划线式）

> 使用场景：**仅限**页面标题下方的一级导航 Tab，用于切换同一页面内的不同内容区域。
> 不可用于卡片内部、弹窗内部或表格工具栏（那些场景用 §11 Tab 切换卡）。

### 视觉参数

| 属性 | 值 |
|------|-----|
| 容器 | `flex items-center gap-1 border-b border-[#dbe6ff]` |
| 单项 padding | `px-4 py-3` |
| 字号 | `text-[14px] font-medium` |
| 选中态 | `text-[var(--text-title)] border-b-2 border-[#0A0A0A] -mb-px` |
| 默认态 | `text-[var(--text-muted)]` |
| Hover | `hover:text-[var(--text-title)]` |

### 代码示例

```jsx
<div className="flex items-center gap-1 border-b border-[#dbe6ff]">
  {TABS.map((tab) => (
    <button
      key={tab.id}
      onClick={() => setActiveTab(tab.id)}
      className={`relative px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap ${
        activeTab === tab.id
          ? "text-[var(--text-title)] border-b-2 border-[#0A0A0A] -mb-px"
          : "text-[var(--text-muted)] hover:text-[var(--text-title)]"
      }`}
    >
      {tab.label}
    </button>
  ))}
</div>
```

### 使用场景约束

| 场景 | 使用 |
|------|------|
| 页面标题下方一级导航 | ✅ 使用本组件 |
| 弹窗/卡片内切换 | ❌ 用 §11 Tab 切换卡（黑底白字按钮式） |
| 表格工具栏筛选 | ❌ 用 §11 Tab 切换卡 |

---

## 11.6 Table 表格组件规范

**文件**: `client/src/components/ui/table.tsx`  
**展示台**: `client/src/pages/DesignSystemComponents.tsx` 的 Table 示例

### 设计原则

1. 表格统一使用 `@/components/ui/table` 的 `Table / TableHeader / TableBody / TableRow / TableHead / TableCell / TableActionCell`，禁止在业务页面用原生 `<table>` + 临时 class 拼装。
2. 表格支持两种信息密度：标准版与紧凑版。`density="compact"` 只调整文字规格、纵向高度与纵向 padding；**左右两端内容到表格边框的安全距离固定为 16px**，圆角、边框、分割线、hover、selected 状态必须与标准版完全一致。
3. 圆角由表格外壳容器统一控制，保持 `rounded-[4px]` / `--radius` 风格；表格组件内部不因密度变化新增圆角。
4. 边框与分割线统一使用 `border-gray-200` / `#E5E5E5`，紧凑版不得单独换色或减弱。
5. 表格正文颜色统一使用 `text-gray-900` / `#171717`，对齐 Typography 的 `body` 正文 token；关键字段可使用 `text-gray-950` / `#0A0A0A`。

### 密度规格

| 属性 | 标准版（默认） | 紧凑版 `density="compact"` |
|------|----------------|------------------------------|
| 表格字号 | `text-sm`（14px） | `text-xs`（12px） |
| 表头高度 | `h-10`（40px） | `h-9`（36px） |
| 表头文字 | `BodyMedium` 对应：14px / Medium / `text-gray-900` | `MetaMedium` 对应：12px / Medium / `text-gray-500` |
| 表头 padding | `px-4`（左右 16px） | `px-4`（左右 16px） |
| 正文单元格 | `h-[54px] px-4 py-3 text-sm` | `h-10 px-4 py-2 text-xs` |
| 正文颜色 | `text-gray-900` / `#171717` | 同标准版 |
| 纯文本行高 | 最小 54px，复杂内容允许自然撑高 | 最小 40px，复杂内容允许自然撑高 |
| 行分割线 | `border-gray-200` | 同标准版 |
| hover / selected | 全局 TableRow 状态 | 同标准版 |

### 使用方式

```tsx
import {
  Table,
  TableActionCell,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// 标准版（默认）
<Table>
  <TableHeader>
    <TableRow>
      <TableHead>组件</TableHead>
      <TableHead>分类</TableHead>
      <TableHead className="text-right">数量</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    <TableRow>
      <TableCell>Button</TableCell>
      <TableCell>操作组件</TableCell>
      <TableCell className="text-right tabular-nums">42</TableCell>
    </TableRow>
  </TableBody>
</Table>

// 紧凑版：只改变密度，不改变圆角 / 边框 / 分割线 / 状态色
<Table density="compact">
  {/* 同样的 TableHeader / TableBody 结构 */}
</Table>
```

### 外壳写法

```tsx
<div className="overflow-hidden rounded-[4px] border border-gray-200 bg-white">
  <Table density="compact">...</Table>
</div>
```

### 表格底部数量统计 + 分页器

当表格底部同时展示数量统计与分页器时，必须使用统一页脚布局：

```tsx
<div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
  <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
    共 {total} 条
  </span>
  <Pagination
    total={total}
    current={page}
    pageSize={PAGE_SIZE}
    className="justify-self-end justify-end flex-nowrap"
    onChange={setPage}
  />
</div>
```

规则：
- 页脚横向 padding 固定 `px-4`（16px），必须与 `TableHead` / `TableCell` 的左右 padding 对齐；禁止继续使用 `px-6`。
- 纵向 padding 固定 `py-3`，顶部使用 `border-t border-[#f0f0f0]`。
- 数量统计固定左对齐：`justify-self-start`，文字 `text-sm leading-[1.5] text-[#737373]`。
- 分页器固定右对齐：`justify-self-end justify-end flex-nowrap`，避免换行和居中漂移。
- 两种写法均可，二选一不要混用：
  - **① 外层 grid 写法**：外层 `grid grid-cols-[1fr_auto]` + 独立数量统计 `span`（`justify-self-start`）+ `Pagination`（`justify-self-end`，**不带** `showTotal`），如上示例。适合数量统计文案需要自定义样式/位置的场景。
  - **② Pagination 内置 showTotal 写法（v2026.06 起推荐简写）**：直接 `<Pagination showTotal={(t) => `共 ${t} 条记录`} ... />`。默认模式下只要传了 `showTotal`，组件内部会自动让总数靠左、页码/每页条数/跳转控件组靠右（`ml-auto`），同样实现「数量统计左、分页控件右」的两端对齐，无需再套外层 grid。

### 禁止事项

- 禁止为紧凑版单独设置新的圆角、边框色、分割线色、hover 色或 selected 色。
- 禁止把紧凑版首尾列横向 padding 缩到 `px-2`、`px-3` 等小于 16px 的值；标准版与紧凑版的左右贴边距离都必须保持 16px。
- 禁止在业务页面通过覆盖 `TableHead` / `TableCell` 的 padding 来临时制造第三种密度；如确有新密度需求，必须先扩展全局 Table 规范。
- 禁止将紧凑版正文改为 `text-gray-500` / `#737373`，紧凑正文仍是正文主内容，必须保持 `#0A0A0A`。

---

## 12. Alert 提示组件

**文件**: `client/src/components/ui/alert.tsx`、`client/src/components/ui/admin-notice-alert.tsx`  
**Token 定义**: `client/src/index.css`

### 基础规则

所有页面信息提示、操作信息提示、警告提示、产品动态通知必须使用 shadcn Alert 标准结构，不允许在业务页面手写 `bg-blue-50` / `bg-amber-50` / `border-*` / `rounded-*` 拼装。管控端顶部常驻公告条必须使用 `AdminNoticeAlert`，不要替换页面内普通 Alert。

统一视觉参数：圆角使用 `--alert-radius: var(--radius)`（当前为 `4px`，组件内为 `rounded-[var(--alert-radius)]`）、`px-4 py-2.5`（上下各 `10px`）、图标 `16px`、图标列固定 `16px`、图标与文字间距 `8px`。图标使用 `translate-y-px`，与 12px / 18px 行高文字首行视觉居中。标题与描述必须拆成 `AlertTitle` / `AlertDescription` 两个兄弟节点，默认 `gap-y-1`，标题与描述上下间距 `4px`。字体必须受 Typography 组件约束：`AlertDescription` 使用 `MetaText`（12px / regular / 1.5 / `tone="inherit"`），`AlertTitle` 使用 `MetaMedium`（12px / medium / 1.5 / `tone="inherit"`）。正文默认保持 inline flow，允许文案中的 `span` 在同一行展示。

| Token | 值 | 用途 |
|------|-----|------|
| `--alert-radius` | `var(--radius)`（当前 `4px`） | Alert 容器圆角 |

### Info 类型（标准信息提示）

用于页面常驻说明、表单辅助提示、非阻断的信息告知。

```tsx
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";

<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示文案</AlertDescription>
</Alert>
```

Info 标准图标必须使用 `AlertInfoIcon`，不要在业务侧直接引入 lucide `Info` 拼装。

| Token | 值 | 用途 |
|------|-----|------|
| `--alert-info-bg` | `#F0F3FC` | Info 背景 |
| `--alert-info-border` | `#BFCFFE` | Info 描边 |
| `--alert-info-foreground` | `#0A0A0A` | Info 正文 |
| `--alert-info-icon` | `#1447E6` | Info 图标 |

### 操作 Info 类型（标准操作说明）

用于操作前后的辅助说明、勾选确认说明、批量操作说明、表单内非警告的操作提示。必须使用 `Alert variant="operation-info"`，左侧图标必须使用 `AlertOperationInfoIcon`；该图标复用 `AlertInfoIcon`，与普通 `info` 类型图标形状完全一致，仅颜色由 `--alert-operation-info-icon` 控制。

```tsx
import { Alert, AlertDescription, AlertOperationInfoIcon, AlertTitle } from "@/components/ui/alert";

<Alert variant="operation-info">
  <AlertOperationInfoIcon />
  <AlertTitle>操作说明标题</AlertTitle>
  <AlertDescription>操作说明描述</AlertDescription>
</Alert>
```

| Token | 值 | 用途 |
|------|-----|------|
| `--alert-operation-info-bg` | `#FFFFFF` | 操作 Info 背景 |
| `--alert-operation-info-border` | `#E5E5E5` | 操作 Info 描边 |
| `--alert-operation-info-foreground` | `var(--alert-info-foreground)` | 操作 Info 正文 |
| `--alert-operation-info-icon` | `#737373` | 操作 Info 图标 |

操作 Info 与普通 `info` 的区别：`info` 用于蓝色页面说明或功能告知；`operation-info` 用于白底灰边的操作上下文说明。两者图标形状一致，颜色分别由 `--alert-info-icon` / `--alert-operation-info-icon` 控制。禁止用手写白底灰边容器替代。

### Warning 类型（标准警告提示）

用于配置缺失、配额不足、风险提示、需要处理但非阻断的警告信息。页面内警告提示使用 `Alert variant="warning"`；管控端顶部常驻公告条不要使用该变体，统一使用 `AdminNoticeAlert`。

```tsx
import { CircleAlert } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

<Alert variant="warning">
  <CircleAlert />
  <AlertTitle>警告标题</AlertTitle>
  <AlertDescription>警告描述</AlertDescription>
</Alert>
```

Warning 标准图标必须使用 `CircleAlert`，禁止使用 `AlertTriangle` 作为标准 warning 横幅图标。仅危险确认、错误态等非 Alert 横幅场景可按语义单独使用 `AlertTriangle`。

| Token | 值 | 用途 |
|------|-----|------|
| `--alert-warning-bg` | `#FFF7ED` | Warning 背景 |
| `--alert-warning-border` | `#FED7AA` | Warning 描边 |
| `--alert-warning-foreground` | `#0A0A0A` | Warning 正文 |
| `--alert-warning-icon` | `#FF6900` | Warning 图标 |

### 产品动态类型

用于管控端顶部产品发布、版本更新、功能上线等非阻断动态告知。必须使用 `Alert variant="product-news"`，左侧图标必须使用设计稿沉淀的 `AlertProductNewsIcon`。颜色 token 通过产品动态语义变量映射到 Info 色值，禁止在业务组件中硬编码蓝色或改用 lucide `Sparkles`。

```tsx
import { Alert, AlertDescription, AlertProductNewsIcon } from "@/components/ui/alert";

<Alert variant="product-news">
  <AlertProductNewsIcon />
  <AlertDescription>【产品动态】提示文案</AlertDescription>
</Alert>
```

| Token | 值 | 用途 |
|------|-----|------|
| `--alert-product-news-bg` | `var(--alert-info-bg)` | 产品动态背景 |
| `--alert-product-news-border` | `var(--alert-info-border)` | 产品动态描边 |
| `--alert-product-news-foreground` | `var(--alert-info-foreground)` | 产品动态正文 |
| `--alert-product-news-icon` | `var(--alert-info-icon)` | 产品动态图标 |

### 管控端彩色背景公告条（AdminNoticeAlert）

用于管控端顶部常驻通知条，设计稿场景包含「产品动态」「待配置」「资源告警」。必须使用 `AdminNoticeAlert`，只替换 `AdminNoticeBar` 这类顶部公告，不要迁移页面内普通 `Alert`。

```tsx
import { AdminNoticeAlert } from "@/components/ui/admin-notice-alert";

<AdminNoticeAlert type="product-news" controls={<span>4/5</span>}>
  <span>OpenClaw v2.4.0 已发布：记忆管理功能上线。</span>
</AdminNoticeAlert>

<AdminNoticeAlert type="pending-config" controls={<span>1/5</span>}>
  <span>有 3 项基础配置未完成，</span>
  <span className="font-medium text-[#020617] underline underline-offset-2">前往基础信息配置处理</span>
</AdminNoticeAlert>

<AdminNoticeAlert type="resource-alert" controls={<span>2/5</span>}>
  <span>私有网络（VPC）配额已耗尽，</span>
  <span className="text-[#020617] underline underline-offset-2">前往腾讯云控制台提交工单</span>
</AdminNoticeAlert>
```

| 类型 | 标签文案 | 图标 / 颜色 | 用途 |
|------|----------|-------------|------|
| `product-news` | 产品动态 | 星光图标 / 蓝色 | 产品发布、版本更新、功能上线 |
| `pending-config` | 待配置 | 感叹号图标 / 橙色 | 基础配置未完成 |
| `resource-alert` | 资源告警 | 感叹号图标 / 橙色 | VPC、云服务器机型等资源配额告警 |

视觉规则：外层高度 `40px`、圆角 `4px`、半透明白底 `rgba(255,255,255,0.76)`、白色描边、12px 正文；左侧标签高度 `22px`、圆角 `2px`、11px 文案；右侧操作区宽 `80.07px`，翻页控件宽 `44.07px`，关闭按钮位于 `left:64.07px; top:2px` 且 `16px` 常驻展示，颜色为 `#020617` / 50% 透明度，翻页控件仅在多条通知时展示。资源告警复用待配置的橙色标签样式和 icon，仅标签文案显示为「资源告警」。

### 普通 Alert 带右侧操作区写法

页面内普通 Alert 如需增加第三列操作区，可通过 `className` 扩展；颜色、字体、图标和基础间距必须由 Alert variant 与 token 控制。管控端顶部常驻公告条仍使用 `AdminNoticeAlert`。

```tsx
<Alert
  variant="warning"
  className="has-[>svg]:grid-cols-[16px_minmax(0,1fr)_auto] gap-y-0"
>
  <CircleAlert />
  <AlertDescription className="flex min-w-0 items-baseline flex-wrap gap-x-1 leading-[1.5]">
    警告文案
  </AlertDescription>
  <div className="col-start-3 shrink-0">操作区</div>
</Alert>
```

### 禁止事项

- 禁止业务页面继续手写 info / operation-info / warning / product-news 提示条样式。
- 禁止使用 warning/amber 样式承载普通信息提示；普通说明必须使用 `variant="info"` 或 `variant="operation-info"`。
- 禁止在业务组件中硬编码 Alert 色值，必须通过 `client/src/index.css` 的 `--alert-*` token。
- 禁止在 Alert 上使用 `rounded-lg` / `rounded-xl` / `shadow-*` / inline `boxShadow`。

---

## 13. 树结构组件（GroupTree / FileTree）

> 参考: shadcn/ui Collapsible FileTree（https://ui.shadcn.com/docs/components/base/collapsible#file-tree）
> 实现文件: `client/src/pages/admin/MemberManagement/GroupList.tsx`、`client/src/pages/admin/SkillLibrary/SkillDetail.tsx`

用于组织管理、文件树、目录导航等层级结构场景。**必须与 shadcn FileTree 完全一致。**

### 颜色规范（严格对齐 shadcn）

| 元素 | 颜色值 | CSS 变量语义 |
|------|--------|------|
| 文字（默认 & 选中） | `#09090b` | `text-foreground` |
| hover / 选中背景 | `#f4f4f5` | `bg-accent` |
| 箭头 / 图标（Chevron、Folder、File） | `#71717a` | `text-muted-foreground` |
| 计数 / 辅助文字 | `#a1a1aa` | `text-muted` |
| 禁用文字 | `#a1a1aa` + `opacity-60` | — |

### 图标规范

| 图标 | 用途 | 尺寸 | 颜色 |
|------|------|------|------|
| `ChevronRight` / `ChevronDown` | 展开/收起 | `w-4 h-4`（含在按钮中）或 `w-3.5 h-3.5` | `#71717a`（muted-foreground） |
| `FolderIcon` / `FolderOpen` | 文件夹 | `w-4 h-4` | `#71717a`（muted-foreground） |
| `FileIcon` / `FileText` | 文件 | `w-4 h-4` | `#71717a`（muted-foreground） |

> **重点**：所有图标统一使用 `text-[#71717a]`（muted-foreground），**不使用** gray-400 或其他灰色。

### 视觉参数

| 属性 | 值 |
|------|-----|
| 行高 | `h-8`（32px） |
| 圆角 | `rounded-[4px]` |
| 缩进 | shadcn 用 `ml-5`（20px），自定义实现用 `paddingLeft: 8 + depth * 16` |
| 行间距 | `gap-1`（4px）或 `mb-0.5` |
| 图标与文字间距 | `gap-1.5`（6px）或 `gap-2`（8px） |

### 交互状态

| 状态 | 样式 | 对应 shadcn class |
|------|------|------|
| 默认 | `text-[#09090b] bg-transparent` | `text-foreground` |
| Hover | `bg-[#f4f4f5] text-[#09090b]` | `hover:bg-accent hover:text-accent-foreground` |
| 选中（Active） | `bg-[#f4f4f5] text-[#09090b] font-medium` | `bg-accent text-accent-foreground` |
| 展开箭头旋转 | `transition-transform group-data-[state=open]:rotate-90` | shadcn 原生 |
| 禁用 | `text-[#a1a1aa] cursor-not-allowed opacity-60` | — |

### shadcn 标准实现（Collapsible 方式）

```tsx
import { ChevronRightIcon, FileIcon, FolderIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"

// 文件夹节点
<Collapsible>
  <CollapsibleTrigger asChild>
    <Button variant="ghost" size="sm" className="group w-full justify-start gap-2 text-[#09090b] hover:bg-[#f4f4f5] hover:text-[#09090b]">
      <ChevronRightIcon className="w-4 h-4 text-[#71717a] transition-transform group-data-[state=open]:rotate-90" />
      <FolderIcon className="w-4 h-4 text-[#71717a]" />
      {folderName}
    </Button>
  </CollapsibleTrigger>
  <CollapsibleContent className="ml-5">
    <div className="flex flex-col gap-1">
      {children}
    </div>
  </CollapsibleContent>
</Collapsible>

// 文件节点
<Button variant="ghost" size="sm" className="w-full justify-start gap-2 text-[#09090b] hover:bg-[#f4f4f5]">
  <FileIcon className="w-4 h-4 text-[#71717a]" />
  {fileName}
</Button>

// 文件节点（选中态）
<Button variant="ghost" size="sm" className="w-full justify-start gap-2 text-[#09090b] bg-[#f4f4f5] font-medium">
  <FileIcon className="w-4 h-4 text-[#71717a]" />
  {fileName}
</Button>
```

### 自定义实现（非 Collapsible 方式，如 GroupList）

```jsx
<div
  className={`group flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] mx-3 mb-0.5 transition-colors ${
    isActive
      ? "bg-[#f4f4f5] text-[#09090b] font-medium"
      : "text-[#09090b] hover:bg-[#f4f4f5]"
  }`}
  style={{ paddingLeft: 8 + depth * 16 }}
>
  {/* 展开箭头 */}
  {hasChildren ? (
    <button className="w-4 h-4 flex items-center justify-center text-[#71717a] hover:text-[#09090b] shrink-0">
      {isExpanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
    </button>
  ) : (
    <span className="w-4 h-4 shrink-0" />
  )}
  <FolderIcon className="w-3.5 h-3.5 text-[#71717a] shrink-0" />
  <span className="truncate">{name}</span>
  <span className="text-[11px] tabular-nums shrink-0 text-[#a1a1aa]">({count})</span>
</div>
```

### 配套元素

- **搜索框**: `h-8 rounded-[4px] border-[#E4E4E4] focus:border-[#020617]`
- **操作按钮**: `w-5 h-5 rounded text-[#d4d4d4] hover:text-[#09090b] hover:bg-[#f4f4f5]`
- **筛选按钮**: `w-8 h-8 rounded-[4px] border-[#E4E4E4]`，活跃态 `border-[#020617] bg-[#f5f5f5]`

---

## 14. Badge 徽标组件

**文件**: `client/src/components/ui/badge.tsx`

> 对齐 shadcn Radix UI 规范（https://ui.shadcn.com/docs/components/radix/badge）。
> 用于 New / Beta / 标签语义、轻量信息标识。**不要用于表达运行状态**（运行状态请使用 `StatusTag`，详见 §16）。

### 14.1 通用样式

| 属性 | 值 |
|------|-----|
| 圆角 | `rounded-full` |
| padding | `px-2.5 py-0.5` |
| 字号 | `text-xs` / Medium |
| 图标 | `[&>svg]:size-3` / `gap-1` |
| focus ring | `ring-[3px] ring-ring/50` |

### 14.2 Variant（默认 4 种，严格对齐 shadcn 截图）

| variant | 背景 | 文字 | 描边 | hover |
|---------|------|------|------|-------|
| `default` | `#0A0A0A` | 白色 | 无 | `bg/90` |
| `secondary` | `#F5F5F5` | `#0A0A0A` | 无 | `#EDEDED` |
| `destructive` | `red-100/60` | `red-600` | 无 | `red-100` |
| `outline` | 白色 | `#0A0A0A` | `#E5E5E5` | `#F5F5F5` |

```tsx
import { Badge } from "@/components/ui/badge";

<Badge>Badge</Badge>
<Badge variant="secondary">Secondary</Badge>
<Badge variant="destructive">Destructive</Badge>
<Badge variant="outline">Outline</Badge>
```

### 14.3 Custom Colors（仅四色，使用 `color` prop）

设置 `color` 后会覆盖 `variant` 视觉样式，仅保留尺寸/字号；对应 shadcn 官方 Custom Colors 示例。

| color | 背景 | 文字 | dark 模式 |
|-------|------|------|-----------|
| `blue` | `bg-[#E8ECFE]` | `text-[#1447E6]` | `bg-blue-950/40 text-blue-300` |
| `green` | `bg-green-50` | `text-green-700` | `bg-green-950/40 text-green-300` |
| `purple` | `bg-purple-50` | `text-purple-700` | `bg-purple-950/40 text-purple-300` |
| `red` | `bg-red-50` | `text-red-700` | `bg-red-950/40 text-red-300` |

```tsx
<Badge color="blue">Blue</Badge>
<Badge color="green">Green</Badge>
<Badge color="purple">Purple</Badge>
<Badge color="red">Red</Badge>
```

### 14.4 使用规则与禁止事项

- Custom Colors **仅允许 blue / green / purple / red** 四种；新增颜色需先在组件层补 token，禁止在业务侧自拼 `bg-xxx-50 text-xxx-700`。
- 表达运行/开关/任务状态（正常 / 禁用 / 失败 / 已弃用 / 当前版本 等）必须使用 `StatusTag`（详见 §16），禁止用 `Badge color="green"` 替代。
- 表达类型/范围/版本/分类等"信息标签"建议优先使用 `StatusTag mode="fill"`；当语义偏向 New / Beta / 通用强调时使用 `Badge`。
- 禁止覆盖组件圆角、字号、padding；如需自定义尺寸应在组件层扩展 `size` variant。
- 禁止在业务侧使用 `Badge variant="default"` 配合 `className="bg-xxx"` 改色；改色统一通过 `color` prop。

---

## 15. Table 表格组件规范

**文件**: `client/src/components/ui/table.tsx`

> 所有管控端/用户端的数据表格必须使用标准 Table 组件，禁止使用原生 `<table>` + 自定义 class。

**设计令牌：**

| Token | Value |
|-------|-------|
| container | `w-full caption-bottom text-[14px] text-[#09090b]` |
| header / bg | `bg-[#fafafa]` |
| header / border | `border-b border-[#f0f0f0]` |
| head cell / height | 标准版 `40px`（`h-10`）；紧凑版 `36px`（`h-9`） |
| head cell / padding | `px-4` |
| head cell / font | 统一 `text-xs font-medium text-[#737373]`（次级灰，不区分密度） |
| body row / border | `border-b border-[#f0f0f0]` |
| body row / hover | `hover:bg-[var(--bg-grey-hover-subtle)]` |
| body row / selected | **无背景高亮**（v2026.06 起移除选中行蓝底；选中状态请用 Checkbox 勾选态表达） |
| body cell / height | 标准版最小视觉高度 `54px`；紧凑版最小视觉高度 `40px`；复杂内容允许自然撑高 |
| body cell / padding | 标准版 `px-4 py-3`；紧凑版 `px-4 py-2` |
| body cell / font | 标准版 `text-sm`（14px）；紧凑版 `text-xs`（12px） |
| footer / layout | `grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]` |
| footer / total | `justify-self-start text-sm leading-[1.5] text-[#737373]` |
| footer / pagination | `justify-self-end justify-end flex-nowrap` |

**组件导出：**

```tsx
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
  TableFooter,
  TableCaption,
} from "@/components/ui/table";
```

**标准用法：**

```tsx
<div className="bg-white rounded-xl border border-[#e5e5e5] overflow-hidden">
  <Table>
    <TableHeader>
      <TableRow>
        <TableHead>名称</TableHead>
        <TableHead>状态</TableHead>
        <TableHead className="text-right">数量</TableHead>
        <TableHead>操作</TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      {data.map((item) => (
        <TableRow key={item.id}>
          <TableCell className="font-medium">{item.name}</TableCell>
          <TableCell><StatusTag mode="text" variant="green">运行中</StatusTag></TableCell>
          <TableCell className="text-right tabular-nums">{item.count}</TableCell>
          <TableActionCell>
            <Button onClick={...}>编辑</Button>
            <Button onClick={...}>删除</Button>
          </TableActionCell>
        </TableRow>
      ))}
    </TableBody>
  </Table>
  {/* 表格页脚：数量统计左对齐，分页器右对齐 */}
  <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
    <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
      共 {data.length} 条
    </span>
    <Pagination
      total={data.length}
      current={page}
      pageSize={PAGE_SIZE}
      className="justify-self-end justify-end flex-nowrap"
      onChange={setPage}
    />
  </div>
</div>
```

**操作列规则（强制）：**
- 操作列必须使用 `<TableActionCell>` 包裹 —— 内置 `flex items-center gap-6` 容器，**操作项间距固定 24px**，且与表头 `<TableHead>` 的 `px-4` 完全对齐
- 操作列必须使用**文字按钮**（如"编辑"、"删除"、"终端"、"关机"），禁止使用纯 icon 按钮
- **每个 Button 必须显式 `variant="link"`**（品牌蓝文字按钮）——不显式声明会得到默认 claw-primary 实心按钮（纯黑 + 白字）
- **删除按钮也统一蓝色 link**，不再用红色覆盖；危险操作的语义由文案 + AlertDialog 二次确认承担（参考 Ant Design 等现代后台规范）。禁止再加 `text-red-600` / `hover:text-red-700` / `disabled:text-red-300` 等红色样式
- **禁止业务侧再手写 `<div className="flex items-center gap-6">` wrapper**，直接把 Button 平铺为 TableActionCell 的 children 即可。如需在内置容器上追加 className（如固定高度 `h-5`），用 `actionsClassName` prop
- 特殊布局（多行 / 自定义 wrapper）：设 `rawChildren` 关闭内置 flex 容器
- 禁止在操作列使用 ghost、outline、default、link-dark 或自定义按钮样式
- **横向滚动固定**：当表格列过多需横向滚动时，操作列必须 `fixed="right"` 固定在最右侧。**统一使用 `<Table scrollX={...}>` + `fixed` 属性**，不允许手写 sticky + bg 的写法（详见下方「固定列（Fixed Columns）」章节）。

**单元格对齐规则（强制）：**
- **垂直对齐**：组件默认 `align-middle`（垂直居中），**禁止业务侧使用 `align-top` 覆盖**。多行内容（如名称 + slug 两行）可自然撑高行高，仍保持居中。
- **禁止使用原生 `<table>`**：所有数据表格必须使用标准 Table 组件，禁止裸写 `<table>` + 自定义 class。

**Variant 视觉变体（v2026.06 起收敛为 2 种）：**

| variant | 表头背景 | body 背景 | 外边框 | 适用场景 |
|---------|---------|----------|--------|----------|
| `"gray-header"`（默认） | `var(--bg-grey-normal)`（#FAFBFD） | `var(--bg-white)` | 继承外层容器 | 白色背景容器（SurfaceCard / 白底 Dialog 等）内的表格 |
| `"white"` | `var(--bg-white)` | `var(--bg-white)` | `border-[var(--bg-white)]` + `rounded-xl overflow-hidden` | 非白色背景（蓝色渐变 Hero / 灰色页面底）上的"浮起白卡"表格 |

> sticky 表头颜色随 variant 自动跟随（通过 `var(--table-head-bg)`），固定列单元格不会再出现「表头一块白一块灰」的色块割裂。

```tsx
// 白色背景容器内（默认灰色表头）—— 不传 variant 即可
<Table>...</Table>

// 非白色页面背景上（整体白色 + 浮起）
<Table variant="white">...</Table>
```

⚠️ **`variant="white"` 使用限制：**
- 禁止在 Dialog / AlertDialog / Sheet / Drawer 等弹窗/抽屉内使用（弹窗内本来就是白底，白上加白看不见）
- 禁止在白色背景（`bg-white`）容器上使用 —— 白底上白边框无法形成视觉层次，请用默认 `gray-header`

> **历史 variant 兼容**：`"default"` / `"elevated-white"` / `"collapsible"` 仍受内部 normalize 兼容（分别映射到 `gray-header` / `white` / `white`），但已 `@deprecated`，新代码禁止使用，下个版本将移除。

**表头样式规则（强制）：**
- 表头行**禁止 hover 变色**（组件已内置 `[thead_&]:hover:bg-transparent`）
- 固定列（`fixed="left"` / `fixed="right"`）表头背景色**必须与非固定列保持一致**。组件内所有表头单元格（普通列 + 固定列）统一使用同一背景表达式 `var(--table-head-bg, var(--bg-grey-normal))`：优先跟随 `<TableHeader>` 按 variant 注入的 `--table-head-bg`（gray-header → 灰 `#FAFBFD`、white → 白），缺失时统一 fallback 到灰。业务侧禁止手写 `bg-*` 覆盖。
  > **v2026.06 修复**：旧实现里普通列表头用了未定义的变量 `--bg-normal`（实际透明，靠 thead 兜底），而固定列 fallback 到白色。当 `--table-head-bg` 缺失（如直接写原生 `<thead>`、未用 `<TableHeader>` 包裹）时会出现「普通列灰 + 固定列白」的**灰白两色割裂**。现已统一两者背景表达式，彻底消除割裂。

**表格内状态/标签样式规则（强制）：**
- **状态列**（运行状态、下发状态）：必须使用 `StatusTag mode="text"`（纯文字变色，无底色无圆点），禁止在表格内使用 `mode="fill"`。⚠️ `mode="dot"` 已全局废弃（圆点视觉冗余），请勿在新代码中使用。
- **版本号**：使用纯文字（如 `v2.1.0`），禁止使用 `StatusTag mode="fill" variant="gray"` 包裹。
- **镜像来源 / 类型标签**（公共/自定义）：拆为独立列，纯文字显示，禁止使用彩色 `StatusTag` fill 标签。
- **辅助信息**（如"腾讯云维护更新"）：使用 `text-xs text-gray-400` 纯文字，禁止用 `StatusTag` 包裹。
- **总原则**：表格行内只允许「状态列」有颜色文字（通过 `StatusTag mode="text"`），其余列一律黑白灰纯文字层次，保持表格整洁。

### 15.1 固定列（Fixed Columns）

> 参考 Ant Design Table fixed columns（https://ant.design/components/table-cn#table-demo-fixed-header），但视觉与交互严格使用项目自身规范。
> 适用场景：列数较多、需要左右横向滚动，但首列（如"名称/Full Name"）或末列（操作列）需要常驻可见。

**核心 API：**

| API | 类型 | 说明 |
|-----|------|------|
| `<Table scrollX>` | `number \| string` | 表格最小内宽（数字 → px，字符串 → 直接作为 min-width，如 `"max-content"`）。**只要传了 scrollX，即开启横向滚动模式**。不需要固定列也可使用。 |
| `<TableHead fixed>` | `"left" \| "right"` | 表头单元格固定到左侧或右侧。 |
| `<TableHead fixedShadow>` | `boolean`，默认 `true` | 是否允许该边界列显示滚动阴影（柔和渐变投影）。组件会自动根据横向滚动状态控制实际显隐：无横滚不显示；在最左侧不显示 left 阴影；在最右侧不显示 right 阴影。多列同侧固定时（如复选框 + 名称列同时 `fixed="left"`），**只保留最外侧那一列为 `true`**，其余列设 `false`，否则中间会出现多余的阴影。<br>**v2026.06 起不再画 1px 硬分隔线**（表格列间本无竖线，硬线显突兀），固定列边界只保留柔和渐变阴影。 |
| `<TableCell fixed>` | `"left" \| "right"` | 内容单元格固定到左侧或右侧。 |
| `<TableCell fixedShadow>` | `boolean`，默认 `true` | 同 `TableHead`。 |
| `<TableActionCell fixed>` | `"left" \| "right"` | 操作单元格固定到左侧或右侧（操作列固定时使用 `fixed="right"`）。 |
| `<TableActionCell fixedShadow>` | `boolean`，默认 `true` | 同上。 |

**多列同侧固定的偏移规则（v2026.06 起自动化）**：业务侧**只需**在需要固定的列上声明 `fixed="left"` / `fixed="right"` 即可，组件会在挂载/resize 时按 DOM 顺序**自动累加同侧固定列宽度并写入 `left` / `right` 偏移**，支持任意非首列固定与多列同侧固定，**无需再手写 `style={{ left: 56 }}` 错开**。
> ⚠️ 旧规范（已废弃）：曾要求"第 2 个及后续同侧固定列必须用 `style={{ left: <累计宽度> }}` 手动错开"。现已由组件自动处理，存量代码里的手写 `left`/`right` 可保留（仍会被尊重）也可逐步移除。

**强制约束：**

1. 必须配合 `<Table scrollX={...}>` 使用，单独给单元格加 `fixed` 但不开启 scrollX 没有意义。
2. 同一列的 `<TableHead>` 与 `<TableCell>` 的 `fixed` 必须**完全一致**（要么都不固定，要么固定在同一侧），否则表头与内容会错位。
3. 操作列在固定模式下必须使用 `<TableActionCell fixed="right">`，禁止改用 `<TableCell>` + 手写按钮样式绕过 link-dark 规范。
4. 固定列的视觉效果（hover 跟随 / 阴影分隔 / 表头底色 / 单元格白底）由组件内部 token 自动处理，**禁止业务侧再手写 `sticky right-0 z-10 bg-white` 之类的 className**。
5. 表头底色统一 `var(--table-head-bg, var(--bg-grey-normal))`、单元格底色固定为 `#fff`、行 hover 时固定列底色自动跟随（`group-hover` → `var(--bg-grey-hover-subtle)`）。**v2026.06 起选中行不再有背景高亮**（见下方说明），固定列也不再有 selected 底色。

**视觉细节（已内置，无需手动处理）：**

- 固定列与滚动区之间**不画 1px 硬分隔线**（v2026.06 移除）：表格列与列之间本就无竖线，固定列若额外加一条硬线会显得突兀（像多出来的边框）。边界只靠下方的柔和渐变阴影区分，参考 Ant Design 固定列效果。
- 滚动阴影：`6px` 宽、`linear-gradient(rgba(0,0,0,0.06) → transparent)`，向滚动方向渐隐，是固定列边界**唯一**的视觉提示（不再叠加 1px 硬线）。**实现方式（v2026.06）**：用固定单元格自身的 `::before` 伪元素绘制（`top:0 / bottom:-1px` 满高且跨行相连），纵向连续不分段、整列首尾无空白；不再使用容器级游离 overlay（旧实现会错位留缝或叠加变重）。`::before` 取 `z-index:5`（低于固定单元格 `20/50`，不会盖住其它固定列）。仅在对应方向存在可滚动内容时出现（最左隐藏 left shadow，最右隐藏 right shadow，无横滚全部隐藏）
- 表头处于固定模式时仍保持灰底 `var(--table-head-bg, var(--bg-grey-normal))`；单元格白底，并通过 `group-hover` 自动跟随行 hover 变成 `var(--bg-grey-hover-subtle)`。**选中行无背景高亮**（v2026.06 起移除）。
- 横向滚动模式下 `<table>` 自动切换为 `border-separate border-spacing-0`，单元格自身补下分隔线，避免 `<tr>` border 在 separate 模式下失效
- **滚动条隐藏策略**：横向滚动模式下，容器默认应用全局 `.scrollbar-on-hover` 工具类——**滚动条默认隐藏**，仅当鼠标 hover 表格区域或正在滚动时才出现，离开后自动隐藏。无需业务侧手动处理。
- **z-index 分层**：表头固定列 `z-50`、body 固定列 `z-20`。**业务表头 / 单元格内部如需 `position:relative` + `z-*`（如带 Popover 筛选的列），`z-*` 必须 ≤ `z-40`**，否则会浮在固定列上方导致"滚动穿透"。Popover/Dialog 等浮层因为是 `fixed` / Portal 定位，不在表格 stacking context 内，不受此约束。

**标准用法：**

```tsx
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";

<div className="bg-white rounded-xl border border-[#e5e5e5] overflow-hidden">
  <Table scrollX={1500}>
    <TableHeader>
      <TableRow>
        {/* 左固定 */}
        <TableHead fixed="left" className="w-[120px]">Full Name</TableHead>
        {/* 中间滚动列 */}
        <TableHead className="w-[140px]">Column 1</TableHead>
        <TableHead className="w-[140px]">Column 2</TableHead>
        <TableHead className="w-[140px]">Column 3</TableHead>
        <TableHead className="w-[140px]">Column 4</TableHead>
        <TableHead className="w-[140px]">Column 5</TableHead>
        <TableHead className="w-[140px]">Column 6</TableHead>
        <TableHead className="w-[140px]">Column 7</TableHead>
        <TableHead className="w-[140px]">Column 8</TableHead>
        {/* 右固定（操作列） */}
        <TableHead fixed="right" className="w-[140px]">操作</TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      {data.map((row) => (
        <TableRow key={row.id}>
          <TableCell fixed="left" className="font-medium">{row.name}</TableCell>
          <TableCell>{row.col1}</TableCell>
          <TableCell>{row.col2}</TableCell>
          <TableCell>{row.col3}</TableCell>
          <TableCell>{row.col4}</TableCell>
          <TableCell>{row.col5}</TableCell>
          <TableCell>{row.col6}</TableCell>
          <TableCell>{row.col7}</TableCell>
          <TableCell>{row.col8}</TableCell>
          <TableActionCell fixed="right">
            <Button variant="link" onClick={...}>编辑</Button>
            <Button variant="link" onClick={...}>删除</Button>
          </TableActionCell>
        </TableRow>
      ))}
    </TableBody>
  </Table>
</div>
```

**多列同侧固定（如复选框列 + 名称列同时固定左侧）：**

```tsx
<Table scrollX={1500}>
  <TableHeader>
    <TableRow>
      {/* 第 1 列：复选框（非边界列，关闭阴影） */}
      <TableHead fixed="left" fixedShadow={false} style={{ width: 56, minWidth: 56 }}>
        <Checkbox ... />
      </TableHead>
      {/* 第 2 列：名称（边界列，保留阴影）。无需手写 left —— 组件自动按声明顺序累加偏移 */}
      <TableHead fixed="left" style={{ minWidth: 240 }}>名称 / ID</TableHead>
      {/* 中间滚动列 ... */}
      <TableHead fixed="right" style={{ width: 200 }}>操作</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    {data.map((row) => (
      <TableRow key={row.id}>
        <TableCell fixed="left" fixedShadow={false} style={{ width: 56, minWidth: 56 }}>
          <Checkbox ... />
        </TableCell>
        <TableCell fixed="left" style={{ minWidth: 240 }}>{row.name}</TableCell>
        {/* 中间滚动列 ... */}
        <TableActionCell fixed="right">...</TableActionCell>
      </TableRow>
    ))}
  </TableBody>
</Table>
```

**禁止事项：**

- ❌ 禁止业务侧手写 `<th className="sticky right-0 z-10 bg-[#fafafa]">` 来模拟固定列；必须使用 `fixed="left|right"` 属性。
- ❌ 禁止手写阴影分隔线（`<div style={{ background: "linear-gradient(...)" }} />` 等）；阴影由组件内部 `before` 伪元素提供，且与表格分隔线对齐。
- ❌ 禁止在 `fixed` 列上覆盖背景色（如 `bg-blue-50`）；固定列底色严格遵循 `<TableHeader>` 的 variant（自动跟随）/ 单元格白底 `var(--bg-white)` / 行 hover `var(--bg-grey-hover-subtle)`。**选中行已无背景高亮**（v2026.06 起移除，不要再期望 selected 蓝底）。
- ❌ **禁止在 `<TableCell fixed>` 上注入 inline `style={{ backgroundColor: ... }}`**（包括 `'white'` / `'#FFF'` / `var(--bg-white)` 等任何形式）。inline style 优先级高于 hover 类，会让 sticky 列在行 hover 时**不变色**（"白条不跟随行变色"）。固定列默认白底由组件内置类与 `[data-fixed]` CSS 规则提供。
  > 说明：`position: sticky` / `zIndex` 由组件内部以 inline style 注入（避免被 tailwind-merge 干掉），`left` / `right` 偏移由组件 measure 自动写入。业务侧可通过 `style` 传 width / minWidth / maxWidth 等**与背景、定位无关**的属性；一般无需再手传 `left` / `right`（自动累加），如确需覆盖偏移可手传，会被尊重。
- ❌ 禁止只给 `<TableHead fixed>` 不给同列的 `<TableCell fixed>`，或反过来。
- ❌ 多列同侧固定时，禁止全部保留 `fixedShadow={true}`（默认值），否则列之间会出现多余的分隔线/滚动阴影；只在最外侧的边界列保留 true，其余内部固定列必须设 `fixedShadow={false}`。
- ❌ 业务页面**禁止用原生 `<tr>` / `<td>` 手撸表格**（如 `<tr className="hover:bg-[#FAFAFA]">`），必须 `import` 自 `@/components/ui/table` 的 `TableRow` / `TableCell`，否则 hover 色 / 选中色 / sticky 等行为均与全局规范脱节。

**表头规则（强制，参照 /admin/audit-log 页面视觉）：**
- `TableHeader` 强制灰色背景（gray-header variant → `var(--bg-grey-normal)` = `#FAFBFD`），由 `var(--table-head-bg)` 统一控制，不允许覆盖
- `TableHead` 强制样式：标准版 `text-xs font-medium text-[#737373] h-10 px-4 py-0 text-left`；紧凑版 `text-xs font-medium text-[#737373] h-9 px-4 py-0 text-left`
- `TableCell` / `TableActionCell` 强制样式：标准版 `text-sm h-[54px] px-4 py-3`；紧凑版 `text-xs h-10 px-4 py-2`；`h-*` 作为 table-cell 的最小视觉高度，复杂内容允许自然撑高
- **表头与单元格横向 padding 必须一致**：标准版和紧凑版都使用 `px-4`（16px），确保左右内容到边框的距离一致；纵向 padding 由单元格控制，标准版 `py-3`、紧凑版 `py-2`；禁止紧凑版横向改成 `px-2` / `px-3`
- **每列标题和内容必须左对齐**，禁止使用 `text-center` 或 `text-right`（数字列除外）
- **标准表格分页器推荐使用 Pagination 默认尺寸**：页面级表格底部分页默认沿用 `size="default"`，如无空间压力不建议切到 `small`
- 禁止通过 className 覆盖表头的字体颜色、字号、字重、对齐方式
- className 仅用于布局属性：宽度 `w-[xx%]`、sticky 定位
- 禁止在 TableHead 上使用 `text-xs`、`text-gray-500`、`uppercase`、`tracking-wide` 等非标准样式
- 禁止使用原生 `<th>` 替代 `<TableHead>`、原生 `<td>` 替代 `<TableCell>`

**禁止事项：**
- 禁止使用原生 `<table>` + 自定义 class（如 `text-xs font-medium text-gray-500 uppercase tracking-wide`）
- 禁止自定义表头背景色（如 `bg-gray-50/50`），统一使用 TableHeader 的 `bg-[#fafafa]`
- 禁止自定义行 hover 效果（如 `hover:bg-gray-50/50`），使用 TableRow 内置 `hover:bg-[#fafafa]`
- 禁止在操作列使用非 link-dark 按钮或自定义 `<button>`
- 分页器必须放在 Table 外部、容器内部，用 `<div className="px-4 py-2 border-t border-[#f0f0f0]">` 包裹
- 不建议把页面级标准表格分页器切到 `size="small"`；`small` 更适合 Dialog 内空间受限场景

---

## 16. StatusTag 状态标签规范

**文件**: `client/src/components/ui/status-tag.tsx`

> 用于表格、卡片、列表中表示状态（运行中/已停止/待处理）或分类属性（角色、范围、版本）的轻量标签。组件内部文字必须复用 `SmallBodyText` 对应 token。

### 16.1 分类与 API

| 分类 | API | 适用场景 |
|------|-----|----------|
| 文本状态类 | `<StatusTag mode="text" variant="green">正常</StatusTag>` | **全场景状态默认**：14px Medium 纯彩色文字，无底色、无圆点。表格 / 详情 / 卡片 / 列表运行状态均使用此模式 |
| 填充信息类 | `<StatusTag mode="fill" variant="blue">全部用户</StatusTag>` | 范围、版本、类型、数量等辅助信息 |
| 角色类 | `<StatusTag preset="role-admin" />` / `<StatusTag preset="role-user" />` | 管控端「用户管理」表格角色列 |
| 自定义 icon 类 | `<StatusTag variant="role" icon={<SomeIcon />}>自定义</StatusTag>` | 低频自定义带 icon 标签；高频语义应沉淀为 `preset` |
| ⚠️ 已废弃 | `<StatusTag mode="dot" ...>` | **mode="dot" 已全局废弃**：组件内部 fallback 到 `mode="text"`，新代码请直接使用 `mode="text"` |

### 16.2 颜色 token

> 同一颜色在 `mode="text"`、`mode="fill"` 与 `mode="soft"` 中保持近似语义：text 使用主色字（无底色、无圆点），fill 使用同语义浅底色 + 主色文字，soft 使用浅底色 + 浅边框 + 深色字。后续新增颜色必须同时补齐 `text / bg / border` 三个 token；`dot` token 仍保留以维持向后兼容，但组件不再渲染圆点。

| variant | text | fill bg | soft border | 使用场景 |
|---------|------|---------|-------------|----------|
| `blue` | `#1447E6` | `#E8ECFE` | `#C7D7FE` | 进行中、全部用户、推荐/提示（对齐品牌蓝） |
| `green` | `#008236` | `#E9F8EB` | `#BFE8C8` | 正常、运行中、已完成、开启、生效 |
| `red` | `#DC2626` | `#FEF2F2` | `#FECACA` | 错误、失败、异常、风险 |
| `orange` | `#B45309` (amber-700) | `#FFFBEB` (amber-50) | `#FDE68A` (amber-200) | 警告、待处理、需关注（dot 用品牌警告色 `#F59E0B`） |
| `gray` | `#0A0A0A` | `#F5F5F5` | `#E5E5E5` | 默认、待处理、关闭、版本、范围 |
| `slate` / `zinc` / `stone` | Tailwind 700 / 500 | Tailwind 50 | Tailwind 200 | 中性色分类标签，低饱和组织 |
| `yellow` / `amber` / `lime` | Tailwind 700 / 500 | Tailwind 50 | Tailwind 200 | 暖色/高亮分类标签 |
| `emerald` / `teal` / `cyan` / `sky` | Tailwind 700 / 500 | Tailwind 50 | Tailwind 200 | 冷色/服务/通道分类标签 |
| `indigo` / `violet` / `purple` / `fuchsia` / `pink` / `rose` | Tailwind 700 / 500 | Tailwind 50 | Tailwind 200 | 多分类彩色标签（如镜像标签），仅用于 `mode="soft"` 或需要稳定色彩组织的场景 |

### 16.3 文本状态类 `mode="text"`（表格状态列默认）

| Token | Value |
|-------|-------|
| background | 无 |
| border | 无 |
| padding | 无（`px-0 py-0`） |
| dot | 不展示 |
| font | `14px` (`text-sm`) / Medium / `leading-[1.5]` |
| color | 使用当前 `variant` 的 text 主色 |

**全局状态语义统一使用 `mode="text"`**（表格/详情/卡片/列表）。表格内禁止使用 `mode="fill"` 表达运行状态；`mode="dot"` 已全局废弃，组件内部 fallback 到 `mode="text"`。

```tsx
<StatusTag mode="text" variant="green">正常</StatusTag>
<StatusTag mode="text" variant="red">禁用</StatusTag>
<StatusTag mode="text" variant="green">当前版本</StatusTag>
<StatusTag mode="text" variant="gray">已弃用</StatusTag>
```

### 16.4 状态点类 `mode="dot"` ⚠️ 已废弃

> **`mode="dot"` / `dot={true}` 已全局废弃**：表格状态列的圆点视觉冗余（列名已承担语义），全场景统一使用 `mode="text"`（彩色纯文本）。
>
> 组件内部已将 `mode="dot"` 静默 fallback 到 `mode="text"`，旧调用点视觉自动降级为无圆点；DEV 环境会有 `console.warn` 提示，请在新代码中直接使用 `mode="text"`。
>
> 详见 PR #418（status-tag 组件层）+ PR #422（绕过组件直接手写圆点的全局清理）。

```tsx
// ❌ 不要再写
<StatusTag mode="dot" variant="green">正常</StatusTag>

// ✅ 全场景统一写法
<StatusTag mode="text" variant="green">正常</StatusTag>
```

### 16.5 填充信息类 `mode="fill"`

| Token | Value |
|-------|-------|
| height | `20px` (`h-5`) |
| background | 使用当前 `variant` 的浅色 bg |
| border-radius | `full` (`rounded-full`) |
| padding | `px-2 py-[2px]` |
| dot | 不展示 |
| font | `SmallBodyText` |

```tsx
<StatusTag mode="fill" variant="blue">全部用户</StatusTag>
<StatusTag mode="fill" variant="gray">v1.2.0</StatusTag>
<StatusTag mode="fill" variant="green">已接入</StatusTag>
```

### 16.6 轻量彩色标签 `mode="soft"`

| Token | Value |
|-------|-------|
| height | `20px` (`h-5`) |
| background | 使用当前 `variant` 的浅色 bg |
| border | 使用当前 `variant` 的浅色 border |
| border-radius | `4px` (`rounded-[4px]`) |
| padding | `px-2 py-0` |
| icon | 可选；`size-3`，颜色跟随文字 |
| font | `SmallBodyText` |

适用于卡片顶部的分类 / 镜像 / 来源标签。需要稳定彩色组织时，从 `slate / zinc / stone / yellow / amber / lime / emerald / teal / cyan / sky / indigo / violet / purple / fuchsia / pink / rose` 中选色；禁止在业务代码中手写 `bg-*-50 text-*-700 border-*-200` 拼标签。

```tsx
<StatusTag mode="soft" variant="amber" icon={<Disc3 />}>
  OpenClaw on Ubuntu 24.04
</StatusTag>
<StatusTag mode="soft" variant="gray">最新版本</StatusTag>
```

### 16.7 角色类 StatusTag token（Figma 1300:6713 / 1300:6724）

| Token | 管理员 | 用户 |
|-------|--------|------|
| API | `preset="role-admin"` | `preset="role-user"` |
| width（设计稿） | `69px` | `57px` |
| height | `22px` | `22px` |
| background | `#FFFFFF` | `#FFFFFF` |
| border | `1px solid #E5E5E5` | `1px solid #E5E5E5` |
| border-radius | `20px` / `rounded-full` | `20px` / `rounded-full` |
| foreground | `#020617` | `#020617` |
| icon | `AdminRoleIcon` / `12px` | `UserRoleIcon` / `12px` |
| padding / gap | `px-2` / `gap-1` | `px-2` / `gap-1` |
| font | `SmallBodyText` | `SmallBodyText` |

```tsx
<StatusTag preset="role-admin" />
<StatusTag preset="role-user" />
```

### 16.7 兼容规则

旧写法仍兼容（组件内部 fallback），但新代码必须使用 `mode="text"`：

```tsx
// 旧：兼容，组件内部 fallback 到 mode="text"（无圆点）
<StatusTag variant="green" dot>正常</StatusTag>
<StatusTag mode="dot" variant="green">正常</StatusTag>

// ✅ 新代码统一写法（详情/卡片/列表 + 表格状态列）
<StatusTag mode="text" variant="green">正常</StatusTag>
```

### 16.8 使用规则

- **全场景状态语义统一使用 `mode="text"`**（14px Medium 纯彩色文字）：表格、详情页、卡片、Popover、列表运行状态均使用此模式。
- 信息/分类/版本/范围类标签使用 `mode="fill"`（带浅色底）；表格内禁用 `mode="fill"` 表达状态。
- ⚠️ `mode="dot"` 已全局废弃（组件内部 fallback 到 `mode="text"`），新代码不要再使用。
- 角色列必须使用 `preset="role-admin"` / `preset="role-user"`，禁止业务侧自行拼 icon、描边和文字。
- `icon` 插槽只用于低频自定义标签；同一语义复用 2 次以上，应沉淀为 `preset`。
- 传入 `icon` 时，业务侧只提供形状；icon 必须支持 `currentColor`，不要在业务侧写颜色和尺寸。

### 16.9 禁止事项

- 禁止使用自定义的 `bg-blue-50 text-blue-600 rounded-xl` 或 `bg-green-50 text-green-600` 等样式替代 StatusTag。
- 禁止在表格状态列使用 `mode="fill"`（有底色胶囊）；`mode="dot"` 已废弃同样禁止。
- 禁止使用自定义的红/绿色纯文字 `span`（如 `text-green-600` / `text-red-500` / `text-[#008236]`）表示开关或运行状态；统一改为 `<StatusTag mode="text">`。
- 禁止绕过 StatusTag 在业务代码中手写状态圆点（`<span class="w-1.5 h-1.5 rounded-full ...">` + 状态文字）；状态语义一律走 `<StatusTag mode="text">`。
- 禁止自定义标签圆角（如 `rounded-xl`），统一使用组件内置圆角。
- 禁止在用户管理角色列自行拼装 `span + icon + border`，统一使用 `StatusTag preset`。

---

## 17. DropdownMenu 下拉菜单规范

**文件**: `client/src/components/ui/dropdown-menu.tsx`
**变体使用场景**: 见 [§ 6.7 DropdownMenu / MoreActionsDropdown](#67-dropdownmenu--moreactionsdropdown独立组件--不合并)（4 种触发器变体）

| 属性 | 值 |
|------|-----|
| 圆角 | `rounded-[8px]` |
| 最小宽度 | `min-w-[8rem]` |
| padding (content) | `p-1` |
| padding (item) | `py-1.5 px-2 text-sm` |
| hover | `bg-[#f5f5f5]` |
| 文字 | `#020617` |
| 图标色 | `#7b818f` |
| disabled | `#d3d6db` |
| 分割线 | `bg-[#e5e5e5]` |
| 阴影 | `0_6px_16px rgba(0,0,0,0.08), 0_3px_6px rgba(0,0,0,0.12), 0_9px_28px rgba(0,0,0,0.05)` |

```tsx
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from "@/components/ui/dropdown-menu";
```

---

## 18. Tooltip 提示浮层规范

**文件**: `client/src/components/ui/tooltip.tsx`

| 属性 | 值 |
|------|-----|
| 背景 | `#020617` |
| 文字 | 白色 |
| 圆角 | `rounded-[4px]` |
| padding | `px-3 py-1.5` |
| 字号 | `text-xs leading-relaxed` |

```tsx
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

<Tooltip>
  <TooltipTrigger asChild><span>hover me</span></TooltipTrigger>
  <TooltipContent side="top" className="text-xs">提示文字</TooltipContent>
</Tooltip>
```

**禁止事项：**
- 禁止用 `p-0` 重置 padding 后自定义内部间距
- 禁止使用过大的 Tooltip（如需展示多行内容，应改用 Popover）

---

## 19. Popover 气泡卡片规范

**文件**: `client/src/components/ui/popover.tsx`

| 属性 | 值 |
|------|-----|
| 背景 | 白色 |
| 边框 | `border-[#e5e5e5]` |
| 圆角 | `rounded-[8px]` |
| 默认宽度 | `w-72` |
| padding | `p-4` |
| 阴影 | 与 DropdownMenu 一致（三层） |
| sideOffset | `4px` |

```tsx
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
```

---

## 20. Card 卡片规范

**文件**: `client/src/components/ui/card.tsx`

| 属性 | 值 |
|------|-----|
| 圆角 | `rounded-xl` |
| 边框 | `border-[#e5e5e5]` |
| 内间距 | header/content/footer 各 `px-6`，整体 `py-6 gap-6` |
| 阴影 | 无（如需阴影请使用 `SurfaceCard`） |

**子组件：** `CardHeader` / `CardTitle` / `CardDescription` / `CardContent` / `CardFooter` / `CardAction`

```tsx
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "@/components/ui/card";
```

---

## 21. RadioGroup 单选组规范

**文件**: `client/src/components/ui/radio-group.tsx`

> 选中色 / 边框色 / focus ring 与 §8 Checkbox 完全一致，**两者必须保持视觉同色**。默认描边色统一使用 `border-[var(--border-control)]` token（详见 §28）。

### 21.1 视觉参数

| 属性 | 值 |
|------|-----|
| 圆圈尺寸 | `size-4` (16px × 16px) |
| 圆角 | `rounded-full` |
| 圆圈背景 | `bg-white`（默认/未选中态强制白底，避免被父容器穿透） |
| 边框宽度 | `1px` |
| 默认边框 | `border-[var(--border-control)]` = `#C8CFDA` |
| 圆点尺寸 | `size-2` (8px × 8px，绝对居中) |
| 组容器 | `grid gap-3`（默认竖排，间距 12px） |
| Item 与 Label 间距 | `gap-2`（8px） |
| transition | `transition-colors` |
| outline | `outline-none`（focus 时由 ring 提供视觉反馈） |

### 21.2 状态规范

| 状态 | 边框 | 背景 | 圆点 | 备注 |
|------|------|------|------|------|
| 默认 | `border-[var(--border-control)]` (#C8CFDA) | `bg-white` | 隐藏 | 使用 token，不要写 `border-gray-200` |
| hover | `#1447E6` | `bg-white` | 隐藏 | 边框变品牌蓝 |
| checked | `#1447E6` | `bg-white` | `fill-[#355EF1]` | 白底蓝边 + 蓝点 |
| focus-visible | `#1447E6` | — | — | 额外加 `ring-[3px] ring-[#355EF1]/20` |
| aria-invalid | `border-destructive` | — | — | 表单校验失败 |
| disabled | `border-[var(--border-control)]` | `bg-[#f3f3f4]` | — | `cursor-not-allowed` |

### 21.3 用法

```tsx
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";

// 竖排（默认，由 RadioGroup 容器 grid gap-3 控制）
<RadioGroup value={value} onValueChange={setValue}>
  <div className="flex items-center gap-2">
    <RadioGroupItem value="a" id="a" />
    <Label htmlFor="a">选项 A</Label>
  </div>
  <div className="flex items-center gap-2">
    <RadioGroupItem value="b" id="b" />
    <Label htmlFor="b">选项 B</Label>
  </div>
</RadioGroup>

// 横排（覆盖容器 className）
<RadioGroup value={value} onValueChange={setValue} className="flex gap-6">
  <div className="flex items-center gap-2">
    <RadioGroupItem value="a" id="a" />
    <Label htmlFor="a">选项 A</Label>
  </div>
  <div className="flex items-center gap-2">
    <RadioGroupItem value="b" id="b" />
    <Label htmlFor="b">选项 B</Label>
  </div>
</RadioGroup>
```

### 21.4 卡片式 Radio（业务自拼）

部分场景（如付费方案选择）会在 Radio 外包裹一张卡片，让整张卡可点击：

```tsx
<RadioGroup value={value} onValueChange={setValue} className="grid grid-cols-3 gap-4">
  {plans.map((plan) => (
    <label
      key={plan.id}
      className={cn(
        "rounded-[8px] border p-4 cursor-pointer transition-colors",
        value === plan.id ? "border-[#1447E6] bg-[#F5F8FF]" : "border-[var(--border-control)] hover:border-[#1447E6]"
      )}
    >
      <div className="flex items-start gap-2">
        <RadioGroupItem value={plan.id} className="mt-1" />
        <div>
          <div className="font-medium">{plan.name}</div>
          <div className="text-sm text-gray-500">{plan.desc}</div>
        </div>
      </div>
    </label>
  ))}
</RadioGroup>
```

要点：
- 外层卡片选中态用 `border-[#1447E6]` + 浅蓝背景 `#F5F8FF`，与 Radio 边框色保持一致
- 卡片默认描边复用 `border-[var(--border-control)]` token，与内部 Radio 视觉一致
- Radio 本身不要隐藏，作为状态指示器一起出现
- 整张卡（`<label>`）可点击，无需手动绑定 onClick

### 21.5 禁止事项

- 禁止使用 `border-gray-200` / 任意 hex 描边色：默认描边必须用 `border-[var(--border-control)]` token
- 禁止使用 Tailwind 默认 `blue-500` (#3B82F6)；必须使用品牌蓝 `#1447E6` / `#355EF1`
- 禁止省略 `bg-white`：默认态圆圈底色必须是白色，否则在 `bg-[#f3f3f4]` 等灰色面板上会变灰
- 禁止单独修改选中色：与 §8 Checkbox 保持一致
- 禁止使用 `opacity-50` 表达 disabled 态；统一用 `bg-[#f3f3f4]`
- 卡片式 Radio 必须保留可见的 `<RadioGroupItem>`，不要"隐藏 Radio 仅靠卡片描边表达选中"

---

## 22. Avatar 头像规范

**文件**: `client/src/components/ui/avatar.tsx`

| 属性 | 值 |
|------|-----|
| 默认尺寸 | `size-8` (32px) |
| 圆角 | `rounded-full` |
| fallback 背景 | `bg-muted` |

```tsx
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";

<Avatar>
  <AvatarImage src={url} />
  <AvatarFallback>AB</AvatarFallback>
</Avatar>
```

---

## 23. Label 标签规范

**文件**: `client/src/components/ui/label.tsx`

| 属性 | 值 |
|------|-----|
| 字号 | `text-xs` |
| 字重 | `font-medium` |
| 颜色 | `#525252` |
| 行高 | `leading-none` |
| disabled | `opacity-50 cursor-not-allowed` |

```tsx
import { Label } from "@/components/ui/label";
<Label htmlFor="email">邮箱地址</Label>
```

---

## 24. Empty 空白页/空状态规范（空态 / 暂无数据 / 无数据占位 唯一标准）

> **本节是项目空状态的唯一权威来源**。关键词：空状态、空态、empty、暂无数据、无数据、占位、插画、Drawer 空态、下拉空态、Popover 空态。
> - 文档结构：本节通用规范 + §24.1 视觉令牌 + §24.2 容器场景速查 + §24.3-7 详解 + §24.8 已废弃。
> - **用户端差异**（按钮变体、卡片容器、padding）见 `SKILL-TENANT.md` §6 用户端空状态。
> - 弹窗内嵌区块空态见 `SKILL.md` §8.7 / §12 第 12 项。
> - 可视化总览：`/preview/empty-state`（Tabs 切换两端）。

**文件**: `client/src/components/ui/empty.tsx`
**插画资源**: `client/public/assets/admin-sidebar/empty-aiagent.png`（由设计统一维护）

### 24.1 视觉令牌

| 属性 | 管控端默认 | 用户端覆写（见 SKILL-TENANT §6）|
|------|-----------|------------------------------|
| 边框 | 默认 `border-dashed border-[#e5e5e5]`，业务统一加 `className="border-0"` | 同左 |
| 圆角 | `rounded-[4px]` | 外层换 `<TenantCard>`（12px 圆角） |
| padding | `py-12`（卡片内）/ `py-20`（独立区域）/ `py-10`（表格嵌入） | 默认 `py-16`（更松） |
| 插画区 | `w-[100px] h-20`（max-w/max-h 锁死，自动 `object-contain`） | 同左 |
| 标题（双行用） | 14px Medium `--text-title` (#0F172A) (`CardTitle`，组件内置) | 同左 |
| 描述 / 单行文字 | 12px `#94A3B8`（`--text-weak` slate-400，由 `MetaText tone="weak"` 组件内置） | 同左 |
| 主操作按钮 | `<Button>` 默认变体 | **`<Button variant="tenant-outline">`（线框胶囊，禁用 `tenant-primary` 实心按钮）** |
| 次操作按钮 | `<Button variant="outline">` | `<Button variant="tenant-outline">` |
| 按钮数量上限 | ≤ 2 | **≤ 2，多按钮全部线框、无主次差异**（见 SKILL-TENANT §5.6） |

### 24.2 容器场景速查（必查）

按空态所在的「容器类型」直接对照本表选答案，无需自行判断。

| 容器类型 | 空态写法 | 关键约束 |
|---------|---------|---------|
| 页面 / 大区域 | `Empty` + 兔子插画 + 标题（+ 描述 + 操作） | `border-0` + `py-12 ~ py-20` |
| 卡片（SurfaceCard / TenantCard） | 同上 | `Empty` 加 `border-0`，避免与卡片边框重叠 |
| **表格（所有场景）** | `<TableCell colSpan>` 内放 `HelperText × 2` 双行（默认）/ 单行（文案极短）。 | ❌ **一律不用兔子插画**；❌ 禁止硬编码 `text-[14px] font-medium` 等覆盖 HelperText 默认 12px `--text-weak`。 |
| Drawer 主内容区 | `Empty` + 兔子插画（同页面级） | 仅当 Drawer 第一层内容为列表/卡片区 |
| Drawer 嵌套子模块（Tab 下子列表） | `HelperText` 单行/双行，无图标 | 层级 ≥ 2 降级，避免插画过重 |
| Dialog / 弹窗内嵌区块 | `HelperText × 2` + `space-y-1` | **禁用**所有装饰图标 / 插画 |
| Dropdown / Select 下拉面板 | `HelperText` 单行 | 面板内 `px-3 py-6` 居中 |
| Combobox / 搜索下拉 | `HelperText` 单行 + 可选 brand 链接 | 如「+ 新建/邀请 XX」入口 |
| Popover / Tooltip 内容 | `HelperText` 单行 | 面板宽度自适应 |
| 侧栏 / 树筛选无结果 | `Empty` + 兔子插画 + `EmptyDescription` | `border-0` + 紧凑 padding |
| 字段值 / 行内「暂无」 | `MetaText tone="weak"` | 不另起行不加图 |

### 24.3 单行 vs 双行规则（强制）

| 文案长度 | 用法 | 视觉效果 |
|---------|------|---------|
| **双行**：标题 + 描述 | `<EmptyTitle>` + `<EmptyDescription>` | 标题 14px 黑（建立层级） + 描述 12px 灰 |
| **单行**：极短文案（如「暂无记录」） | **直接用 `<EmptyDescription>`** | 12px 灰，**不要**用大黑标题 |

> 📌 **判断口诀**：能写出有层级感的两行（标题概括 + 描述细化）→ 用双行；只能写一句话 → 单行用 `EmptyDescription`。
>
> ❌ 禁止：单行场景用 `<EmptyTitle>暂无记录</EmptyTitle>`，会渲染为 14px 黑标题，视觉过重，喧宾夺主。

### 24.4 标准用法

**统一形态**：所有页面级、卡片区、列表的空态都使用此结构。`EmptyMedia` 默认即渲染兔子插画，**业务无需传任何参数**。

> 📌 **唯一例外：表格空态不用兔子插画**，统一在 `<TableCell>` 内放纯文字（见 §24.5）。原因：表格本身已有表头/边框/列结构，再叠插画会形成双重视觉重心，且插画会把行高撑高、破坏列对齐。

```tsx
import {
  Empty, EmptyHeader, EmptyTitle, EmptyDescription, EmptyMedia, EmptyContent
} from "@/components/ui/empty";

// ① 双行：标题 + 描述
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyTitle>暂无数据</EmptyTitle>
    <EmptyDescription>当前没有可显示的内容</EmptyDescription>
  </EmptyHeader>
</Empty>

// ② 双行 + 操作引导
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyTitle>还没有创建任何 Agent</EmptyTitle>
    <EmptyDescription>创建你的第一个 Agent，开始自动化工作流</EmptyDescription>
  </EmptyHeader>
  <EmptyContent>
    <Button><Plus className="w-4 h-4" />新建 Agent</Button>
  </EmptyContent>
</Empty>

// ③ 单行（文案极短，禁用 EmptyTitle）
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyDescription>暂无记录</EmptyDescription>
  </EmptyHeader>
</Empty>

// ④ 侧栏 / 极简
<Empty className="border-0 px-4 py-10">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyDescription>暂无符合筛选条件的组织</EmptyDescription>
  </EmptyHeader>
</Empty>
```

### 24.5 表格空态（页面级 / 弹窗内 / 紧凑，统一规则）

**所有表格空态都不用兔子插画**，统一在 `<TableCell colSpan>` 内放 `HelperText` 纯文字。**默认双行**（与浮层 Dialog 内嵌空态保持一致），文案足够短时降级为单行。

```tsx
import { HelperText } from "@/components/ui/Typography";

// ✅ 双行（默认形态）：HelperText × 2 + space-y-1
<TableCell colSpan={N}>
  <div className="text-center py-12 space-y-1">
    <HelperText>暂无记录</HelperText>
    <HelperText>尝试调整筛选条件，或新建一条记录</HelperText>
  </div>
</TableCell>

// ✅ 单行（仅当真的没有第二行可写时使用）
<TableCell colSpan={N}>
  <div className="text-center py-10">
    <HelperText>没有符合条件的实例</HelperText>
  </div>
</TableCell>

// ❌ 禁止：表格内嵌 Empty + 兔子插画
<TableCell colSpan={N} className="p-0">
  <Empty><EmptyHeader><EmptyMedia />...</EmptyHeader></Empty>
</TableCell>

// ❌ 禁止：硬编码字号字色（破坏 token 系统，与浮层空态文字层级不一致）
<TableCell colSpan={N} className="text-center py-12">
  <div className="text-[14px] font-medium text-[var(--text-title)]">暂无记录</div>
  <div className="text-[12px] text-[var(--text-weak)]">...</div>
</TableCell>
```

**为什么表格用 `HelperText × 2` 双行而不是粗黑标题 + 灰描述？**
- 表格行业本身已是高密度信息容器，再叠粗黑标题（18px / 14px Medium）会把空态行的视觉权重抬到比普通数据行还高。
- 双行 `HelperText` 全部 12px `--text-weak`，与浮层 Dialog 内嵌、Drawer 嵌套子模块、Popover 等所有「无插画的纯文字空态」保持统一文字层级 — 减少认知负担。
- 单行 vs 双行只是文案长度选择，不再代表层级差异，进一步简化判断。

### 24.6 Drawer 抽屉空态

Drawer 内的空态需根据**层级**选不同写法：

```tsx
// ① 主内容区（Drawer 第一层内容为列表）— 用兔子插画
<Drawer>
  <DrawerContent>
    <Empty className="border-0 py-16">
      <EmptyHeader>
        <EmptyMedia />
        <EmptyTitle>暂无模板</EmptyTitle>
        <EmptyDescription>从右上角「新建模板」开始创建</EmptyDescription>
      </EmptyHeader>
    </Empty>
  </DrawerContent>
</Drawer>

// ② Drawer 内嵌套（Tab 下子列表 / 卡片区） — 降级为 HelperText
<DrawerContent>
  <Tabs>
    <TabsContent value="alarms">
      <div className="text-center py-12 space-y-1">
        <HelperText>暂无关联告警</HelperText>
        <HelperText>该资产当前未触发任何告警规则</HelperText>
      </div>
    </TabsContent>
  </Tabs>
</DrawerContent>
```

### 24.7 浮层空态（Dropdown / Popover / Dialog 等）

所有浮层（漂浮在主内容之上的容器）内的空态，**禁用插画**，统一用 `HelperText` 纯文字：

```tsx
// Dropdown / Select 面板内无可选项
<SelectContent>
  <div className="text-center py-6">
    <HelperText>暂无可选项</HelperText>
  </div>
</SelectContent>

// Combobox 搜索无匹配 + 新建入口
<CommandList>
  <CommandEmpty>
    <div className="text-center py-4 space-y-2">
      <HelperText>没有匹配的结果</HelperText>
      <button className="text-xs text-[var(--text-brand)] hover:underline">
        + 邀请「{keyword}」加入
      </button>
    </div>
  </CommandEmpty>
</CommandList>

// Popover 内空态（如通知中心）
<PopoverContent>
  <div className="text-center py-6">
    <HelperText>暂无未读通知</HelperText>
  </div>
</PopoverContent>

// Dialog 内嵌区块空态（详见 SKILL.md §8.7）
<DialogBody>
  <div className="text-center py-12 space-y-1">
    <HelperText>该角色还没有技能</HelperText>
    <HelperText>可从公共技能库或企业技能库添加</HelperText>
  </div>
</DialogBody>
```

### 24.8 强制约束（不可违反）

1. **统一插画**：所有**非表格**的页面级 / 卡片区空态都使用兔子插画，**不再使用 lucide 图标 + 灰色方块容器**的旧形态。
2. **表格空态禁用插画**：所有 `<TableCell colSpan>` 内的空态一律用纯文字（双行：标题 + 描述 / 单行：`text-[var(--text-weak)]`），详见 §24.5。
3. **业务零参数**：`<EmptyMedia />` 直接调用即可，禁止业务侧手写 `<img>` 或传 `src`、`children` 覆盖默认插画。
4. **禁止覆盖文字样式**：标题统一 14px Medium `CardTitle`（默认 `--text-title` #0F172A），描述统一 12px `--text-weak` (#94A3B8) `MetaText tone="weak"`。**禁止**用 `className="text-sm font-normal text-[#A3A3A3]"` 等方式改字号字色。
5. **单行场景禁用 `EmptyTitle`**：极短文案（如「暂无记录」「暂无可选项」）只用 `<EmptyDescription>`，避免 14px 黑标题视觉过重。
6. **浮层/弹窗禁用插画**：Dropdown / Popover / Dialog 内嵌、Drawer 嵌套子模块，必须用 `HelperText` 纯文字，禁用 `Empty` + 插画。
7. **插画 + 边框互斥**：使用插画时必须 `<Empty className="border-0 ...">` 去掉默认虚线框，避免双重视觉重心。
8. **新增其他场景插画**：如确需第二种语义插画（如错误态、权限态），先扩展 `EmptyMedia` 的 `variant`（如 `variant="error"`）并在本节登记，**禁止**通过 `children` 塞 `<img>` 绕过规范。
9. **用户端差异**：用户端（`pages/tenant/**`）的空态外层容器 / 按钮变体 / padding 与管控端不同，详见 `SKILL-TENANT.md` §6。

### 24.9 已废弃写法（任何一种命中即视为不规范）

| 写法 | 错误点 |
|------|--------|
| `<EmptyMedia variant="icon"><Inbox /></EmptyMedia>` | 旧 API，已下线，统一改 `<EmptyMedia />` |
| `<div className="text-center py-24"><Bot className="w-12 h-12 text-gray-200" /><p className="text-gray-400">...</p></div>` | 全手写，无组件复用 |
| `<EmptyMedia><img src="..." className="w-[100px] h-20" /></EmptyMedia>` | 手写 img，绕过组件内置插画 |
| `<EmptyTitle className="text-sm text-[#A3A3A3]">...</EmptyTitle>` | 覆盖字号字色，破坏层级 |
| `<EmptyTitle>暂无记录</EmptyTitle>`（仅一行文案） | 单行不用 `EmptyTitle`，应改 `EmptyDescription` |
| `<TableCell colSpan={N} className="p-0"><Empty><EmptyMedia />...</Empty></TableCell>` | 表格空态不用兔子插画，应改纯文字（见 §24.5） |
| `<td colSpan={N} className="text-gray-400">暂无</td>` | 表格空态硬编码灰，应改 `text-[var(--text-weak)]` |
| 弹窗 / Popover / Dropdown 内放 `<EmptyMedia />` 插画 | 浮层空态禁用插画，应改 `HelperText` |
| 用户端用 `<SurfaceCard>` 包空态 | 用户端必须用 `<TenantCard>`（12px 圆角），见 SKILL-TENANT §6 |
| 用户端用 `<Button variant="tenant-primary">` 实心按钮 | 用户端空态按钮统一 `tenant-outline` 线框胶囊（禁用实心，避免喧宾夺主），见 SKILL-TENANT §5.6 |
| 用户端空态 ≥ 3 个按钮 | 最多 2 个按钮，多按钮全部 `tenant-outline`、无主次 |

**Upload 上传区域**与 Empty 共享相同的 dashed 边框样式，额外约束：边框 `1px`，禁止使用默认 Upload 图标。

---

## 25. Segment 分段选择器规范

文件：`client/src/components/ui/segment.tsx`

> Segment 与 Tabs 的区别：Tabs 活跃态为品牌蓝 `#355EF1`；Segment 活跃态为深色 `#020617` + `font-semibold` + 白底浮起。Segment 适用于内容区域的子分类切换（如详情页各 Tab）。

**设计令牌（对齐 Figma）：**

| Token | Value |
|-------|-------|
| container / bg | `var(--bg-segment-track)` = `#DBDDE432`（冷灰偏蓝 +20% alpha，0605 二次修订；原 `#E4EAF7` 不透明，再之前 `#f3f3f4`） |
| container / border | `var(--bg-segment-track)` = `#DBDDE432`（与 bg 同色，保留 1px 边占位但视觉无可见描边，避免 hover/active 时整体偏移） |
| container / radius | `6px` |
| container / padding | `3px` |
| container / height | `36px` (h-9) |
| item / active bg | `#FFFFFF` |
| item / active text | `#020617` (font-semibold) |
| item / active shadow | `0px 1px 2px rgba(0,0,0,0.05)` |
| item / active radius | `4px` |
| item / inactive text | `#7b818f` (font-normal) |
| item / hover text | `#4b5563` |
| item / padding | `4px 16px` |
| item / disabled text | `#d3d6db` |

**使用方式：**
```jsx
import { Segment, SegmentList, SegmentItem, SegmentContent } from "@/components/ui/segment";

<Segment defaultValue="basic">
  <SegmentList>
    <SegmentItem value="basic">基础配置</SegmentItem>
    <SegmentItem value="tools">工具管理</SegmentItem>
    <SegmentItem value="memory">记忆管理</SegmentItem>
  </SegmentList>
  <SegmentContent value="basic">...</SegmentContent>
  <SegmentContent value="tools">...</SegmentContent>
</Segment>
```

**禁止事项：**
- 禁止用 className 覆盖 Segment 组件样式
- 页面内分类切换统一使用 Segment，不再自行写 button + border-l 的竖向导航

---

## 26. Pagination 分页器规范

文件：`client/src/components/ui/pagination.tsx`

> 全局统一高级分页组件，内置页码折叠、simple 模式、showSizeChanger 等能力。所有页面和弹窗中的分页必须使用此组件，禁止自行实现内联分页或使用 `@/components/Pagination` 等非标准实现。

**设计令牌（全部 token 化，零硬编码）：**

| Token | Tailwind / CSS Variable | Value | 说明 |
|-------|------------------------|-------|------|
| item / size (default) | `h-7 min-w-[28px]` | **28px** | 页面级表格底部分页 |
| item / size (small) | `h-6 min-w-[24px]` | **24px** | 弹窗/浮层内分页 |
| item / border-radius | `rounded-lg` | `8px` | 页码按钮统一圆角 |
| item / border (inactive) | `border-[var(--border)]` | `#EAEEF4` | 非激活态描边 |
| item / bg | `bg-white` | `#FFFFFF` | 所有状态均白底 |
| item / text (inactive) | `text-[var(--text-title)]` | — | 非激活态文字 |
| item / hover bg | `hover:bg-[var(--bg-grey-hover)]` | — | hover 弱灰 |
| **item / active border** | `border-[var(--text-brand)]` | `#355EF1` | **白底 + 蓝边 + 蓝字（禁止实心色块）** |
| **item / active text** | `text-[var(--text-brand)]` | `#355EF1` | 品牌蓝 |
| item / disabled | `opacity-40 cursor-not-allowed` | — | 首尾页按钮禁用 |
| total text | `text-[var(--text-muted)]` | `#A3A3A3` | 左侧总数文案 |
| font-size | `text-xs` | `12px` | 与表格口径一致 |

**使用方式：**

```tsx
import { Pagination } from "@/components/ui/pagination";

// ① 基础用法（页面级列表）
<Pagination total={100} current={page} pageSize={10} onChange={(p) => setPage(p)} />

// ② 表格底部标准布局（总数靠左 + 页码靠右）
<Pagination
  total={500}
  current={page}
  pageSize={pageSize}
  showTotal={(total) => `共 ${total} 条记录`}
  showSizeChanger
  pageSizeOptions={[10, 20, 50, 100]}
  className="w-full justify-between"
  onChange={(p, size) => { setPage(p); setPageSize(size); }}
/>

// ③ 弹窗/浮层（simple 模式：仅 < X/Y > 前后页 + 总数）
<Pagination
  total={total}
  current={page}
  pageSize={pageSize}
  mode="simple"
  showTotal={() => `共 ${total} 条，第 ${page} / ${totalPages} 页`}
  className="w-full justify-between"
  onChange={(p) => setPage(p)}
/>

// ④ 小尺寸（Dialog 内空间受限）
<Pagination
  total={total}
  current={page}
  pageSize={pageSize}
  size="small"
  showTotal={(t) => `共 ${t} 条`}
  className="w-full justify-between"
  onChange={(p) => setPage(p)}
/>
```

**Props 速查：**

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `total` | `number` | — | 数据总条数（必填） |
| `current` | `number` | — | 当前页码，从 1 开始（必填） |
| `pageSize` | `number` | — | 每页条数（必填） |
| `onChange` | `(page, pageSize) => void` | — | 页码/条数变化回调 |
| `showTotal` | `((total) => ReactNode) \| false` | `(t) => \`共 ${t} 条\`` | 左侧总数文案；传 `false` 不显示 |
| `showSizeChanger` | `boolean` | `false` | 显示每页条数选择器 |
| `pageSizeOptions` | `readonly number[]` | `[10,20,50,100]` | 条数选项 |
| `mode` | `"default" \| "simple"` | `"default"` | default = 完整页码；simple = 仅前后页 + `X/Y` 指示（弹窗/浮层用） |
| `size` | `"default" \| "small"` | `"default"` | default = 28px 按钮；small = 24px 按钮 |
| `hideOnSinglePage` | `boolean` | `false` | 单页时隐藏分页按钮（仅保留 showTotal） |
| `className` | `string` | — | 外层 `<nav>` 额外样式 |

**页码折叠策略：**
- 总页数 ≤ 7：全部展示
- 总页数 > 7：始终展示首尾页 + 当前页 ± 1，中间用 `···` 折叠

**尺寸使用规则（强制）：**

| 尺寸 | 按钮高度 | 使用场景 |
|------|---------|---------|
| `size="default"`（默认） | **28px** (h-7) | 页面级表格底部分页，所有常规列表 |
| `size="small"` | **24px** (h-6) | 弹窗（Dialog）/ Drawer 内的表格分页 |

**模式使用规则：**

| 模式 | 说明 | 使用场景 |
|------|------|---------|
| `mode="default"` | 完整页码按钮 + 省略号折叠 | 页面级列表、表格底部 |
| `mode="simple"` | 仅 `< X/Y >` 前后页指示 | 弹窗/浮层/空间紧凑区域 |

**禁止事项：**
- 禁止在页面中自行实现分页按钮逻辑（内联 prev/next button、Array.from 页码循环等）
- 禁止使用 `@/components/Pagination`（旧自定义组件），统一使用 `@/components/ui/pagination`
- 禁止自定义分页样式覆盖组件样式（如蓝色填充背景 `bg-[#355EF1] text-white`）
- 禁止 Active 状态做成实心色块——必须是白底 + 蓝边 + 蓝字
- 新页面/弹窗中出现列表分页，必须直接使用此组件
- 现有页面修改时如发现内联分页，应顺手替换为标准组件

---

## 27. Toast 通知组件

> 源码路径: `client/src/components/ui/sonner.tsx`  
> 基于: sonner 库  
> 全局 CSS 覆写: `client/src/index.css` 中 `[data-sonner-toast]` 规则

### 视觉规范

| 属性 | 值 |
|------|------|
| 背景色 | `#FFFFFF` |
| 文字色 | `#09090b` |
| 边框色 | `#EAEEF4` |
| 圆角 | `12px` (rounded-xl) |
| 内边距 | `12px 16px` |
| 字号 | `14px`，font-medium |
| 阴影 | shadow-lg |
| 定位 | 页面顶部居中 (top-center) |

### 布局结构

```
┌─────────────────────────────────────────┐
│  [icon]  消息文本内容          [×关闭]  │
└─────────────────────────────────────────┘
```

- **图标**：左侧，由 sonner 根据类型自动渲染（error=黑色感叹号，success=勾）
- **文本**：居中，14px font-medium
- **关闭按钮**：**右侧垂直居中**，20×20px，hover 时 bg-[#f4f4f5]

### 使用方式

```tsx
import { toast } from 'sonner';

toast.error("请输入用户 ID");
toast.success("操作成功");
toast("普通提示消息");
```

### 关键约束

- **关闭按钮必须在右侧**，禁止使用 sonner 默认的左上角定位
- 所有 toast 类型（error/success/info/warning）使用统一白色背景 + `#EAEEF4` 边框
- 禁止在业务代码中自行拼装弹出通知 UI，必须使用 `toast()` API
- Toast 层级固定 `z-index: 99999`，确保在 Dialog 之上

---

## 28. 全局描边颜色规则

### 28.1 描边 token 体系

| 用途 | Token / Tailwind class | 色值 | 说明 |
|------|------------------------|------|------|
| **通用描边** | `var(--border)` / `border-border` | `#EAEEF4` | 卡片、表格、分割线、面板，**含 Input / Select / DatePicker 等所有非可勾选控件**（全局统一 token） |
| **可勾选控件描边** | `border-[var(--border-control)]` | `#C8CFDA` | Checkbox、Radio 默认 / disabled 边框（详见 §8 / §21） |
| **激活描边** | hex `#1447E6` | `#1447E6` | hover / focus / checked 状态品牌蓝 |
| **focus ring 外环** | `var(--ring)` | `#1447E6` | shadcn focus-visible 自动应用 |

> **0605 修订**：原 Input / Select 硬编码 `#D3D6DB` 描边已**全量并入** `--border` (#EAEEF4)，与容器 / Tag / Card 等全局描边语言对齐，消除"输入控件描边深一档"的视觉撕裂。

### 28.2 选用决策树

```
需要描边时：
├─ 是 Checkbox / Radio 等可勾选控件？
│  └─ 用 `border-[var(--border-control)]`
├─ 是按钮、segment、tab 等行动控件？
│  └─ 各组件章节有专属规则（详见对应章节）
└─ 其余卡片、面板、分割线、表格边界、**Input / Select / DatePicker 输入控件**
   └─ 用 `border-border`（即 `var(--border)` = `#EAEEF4`）
```

### 28.3 禁止事项

- 禁止使用 `border-gray-100/200/300` 等 Tailwind 默认灰色描边（除非映射到本表 token）
- 禁止使用 `border-blue-300` / `border-green-300` 等 Tailwind 默认彩色描边
- 禁止在 Checkbox / Radio 上写 `border-gray-200` 或 hex 描边色：必须使用 `border-[var(--border-control)]` token，方便后续全局调整
- 卡片内的子描边、表单组织描边、辅助分割：统一使用 `border-border`，不要再创造新颜色

---

## 29. 强制执行规则

1. **组件源文件 (`client/src/components/ui/*.tsx`) 只有 addietang 可以修改**
2. 其他人使用组件时，不允许通过 className 覆盖组件定义的颜色/边框/圆角
3. 用户端新增页面 / 新增业务组件必须优先使用 `Typography.tsx` 中的文字组件，不再自行拼装基础文字色、字号、字重
4. 修改用户端旧页面时，遵循“触达即同步”：当前文件内明显的标题、正文、Meta、数字、代码文字应同步迁移到 Typography
5. 新增全局组件时，组件内部文字规格必须先映射到 Typography 层级；如确需新增文字层级，先更新本规范和 `Typography.tsx`
6. 如发现 rebase 后组件样式被改，以 addietang 和 miekoyychen 的版本为准强制恢复
7. 新增组件需经 addietang 和 miekoyychen 审核后才能合入基线
8. **对话框 / 弹窗 / 右侧抽屉内的 Input、下拉（Select）、Table 必须直接 import 自 `@/components/ui/*` 且与本 SKILL 第 5 / 6 / 11.1 / 7.2 节规范完全一致**：
   - 禁止在弹窗 / 抽屉中重新编造 Input / Select / Table / Drawer 样式
   - Input / Select **默认状态禁止加底色**（白底 + `border-border`；Drawer 详情内可按第 7.2 节使用 `bg-background` 语义）
   - Input / Select **禁用（disabled）状态禁止添加任何 hover 样式**（不允许 `disabled:hover:*`，不允许出现边框变蓝、底色加深等反馈）
   - 右侧详情抽屉必须优先使用 `@/components/ui/drawer` 的 `Drawer direction="right"`，禁止手写 fixed 浮层结构

---

## 30. 管控端左侧导航 AdminSidebar（owner: miekoyychen）

> **Owner**: miekoyychen  
> **源文件**: `client/src/components/ui/admin-sidebar.tsx`  
> **CSS 变量**: `client/src/index.css` 中 `--admin-sidebar-*` 部分  
> **修改权限**: 仅 miekoyychen 可修改 sidebar 相关源文件和 CSS 变量

### 30.1 CSS Token（定义在 `:root`）

> **0608 同步**：本表已与 `client/src/index.css` 实际 token 对齐。如代码与文档冲突，以代码为准（owner 决策权归 miekoyychen）。

| Token | 值 | 说明 |
|-------|-----|------|
| `--admin-sidebar-width` | `240px` | 展开宽度 |
| `--admin-sidebar-width-collapsed` | `64px` | 收起宽度 |
| `--admin-sidebar-header-height` | `104px` | 头部高度（容纳品牌区 + 工具行） |
| `--admin-sidebar-footer-height` | `72px` | 底部高度 |
| `--admin-sidebar-bg` | `#ffffff` | 背景色 |
| `--admin-sidebar-border` | `#EAEEF4` | 边框色（与全局浅边框一致） |
| `--admin-sidebar-foreground` | `#0a0a0a` | 主文字色 |
| `--admin-sidebar-muted` | `#737373` | 辅助文字色（组织标题、badge） |
| `--admin-sidebar-item-height` | `34px` | 菜单项高度 |
| `--admin-sidebar-item-radius` | `4px` | 菜单项圆角 |
| `--admin-sidebar-item-hover-bg` | `rgba(180, 191, 225, 0.14)` | 菜单项 hover 背景（蓝色淡彩） |
| `--admin-sidebar-item-active-bg` | `linear-gradient(90deg, #EBF4FF 0%, #DCE8FE 100%)` | 活跃项渐变背景 |
| `--admin-sidebar-action-bg` | `#ffffff` | 头部操作按钮背景 |
| `--admin-sidebar-action-border` | `var(--admin-sidebar-border)` | 头部操作按钮边框（继承） |
| `--admin-sidebar-action-hover-bg` | `var(--admin-sidebar-action-bg)` | 头部操作按钮 hover 背景（继承，无变化） |
| `--admin-sidebar-action-hover-border` | `var(--admin-sidebar-border)` | 头部操作按钮 hover 边框（继承） |
| `--admin-sidebar-logo-shadow` | `0px 1px 4px rgba(176, 182, 195, 0.3)` | Logo 阴影 |
| `--admin-sidebar-badge-bg` | `#f5f5f5` | badge 背景 |
| `--admin-sidebar-badge-brand-bg` | `color-mix(... 品牌蓝 8% ...)` | 品牌色 badge 背景（New/Beta） |
| `--admin-sidebar-badge-brand-border` | `color-mix(... 品牌蓝 16% ...)` | 品牌色 badge 边框 |
| `--admin-sidebar-avatar-bg` | `color-mix(...)` | Avatar 背景（取代旧的 green 渐变） |
| `--admin-sidebar-avatar-foreground` | `#020617` | Avatar 文字色 |

### 30.2 结构组件

| 组件 | 样式 |
|------|------|
| `AdminSidebar` | `fixed inset-y-0 left-0 z-40 flex flex-col border-r` + 宽度过渡 300ms |
| `AdminSidebarHeader` | `h-[var(--admin-sidebar-header-height)] px-4 border-b border-[--admin-sidebar-border]` |
| `AdminSidebarContent` | `flex-1 overflow-y-auto px-4 py-4` + 自定义滚动条（`.scrollbar-on-hover`） |
| `AdminSidebarFooter` | `h-[var(--admin-sidebar-footer-height)] px-6 border-t border-[--admin-sidebar-border]` |
| `AdminSidebarInset` | `flex-1 min-w-0 overflow-x-hidden` + `margin-left` 跟随侧边栏宽度 |

### 30.3 菜单项样式

| 状态 | 样式 |
|------|------|
| Normal | `h-[var(--admin-sidebar-item-height)] px-2 gap-2 rounded-[var(--admin-sidebar-item-radius)] text-sm text-[--admin-sidebar-foreground]` |
| Hover | `background: var(--admin-sidebar-item-hover-bg)` (蓝色淡彩) |
| Active | `background: var(--admin-sidebar-item-active-bg)` (蓝色渐变) + `font-medium` |
| Icon | `size-4 shrink-0` |
| 文字 | `tracking-[0.005em] leading-[22px]` |

### 30.4 组织标题

- `mb-1 text-xs font-normal tracking-[0.015em] text-[--admin-sidebar-muted]`，与菜单项间距 4px
- 带折叠/展开箭头（ChevronUp/Down `size-3`）
- hover: `text-gray-900`

### 30.5 Badge（New / 即将开放 / 数量）

- `h-[18px] rounded-full px-1.5 text-[11px] font-normal leading-none`
- 中性: 背景 `var(--admin-sidebar-badge-bg)` (#f5f5f5) / 文字 `var(--admin-sidebar-muted)` (#737373)
- 品牌（New/Beta）: 背景 `var(--admin-sidebar-badge-brand-bg)` / 边框 `var(--admin-sidebar-badge-brand-border)`

### 30.6 头部品牌区

- Logo: 36×28 SVG，外加 `box-shadow: var(--admin-sidebar-logo-shadow)`
- 品牌名: `text-sm font-medium text-[--admin-sidebar-foreground]` hover → `#355EF1`
- 副标题: `text-xs font-normal text-[--admin-sidebar-muted]` hover → `#355EF1`

### 30.7 头部操作按钮（前往用户端）

- `size-8 rounded-[4px] border`
- Normal: `bg-[--admin-sidebar-action-bg] border-[--admin-sidebar-action-border]`
- Hover: `bg-[--admin-sidebar-action-hover-bg] border-[--admin-sidebar-action-hover-border]`（hover 与 normal 同值，仅 focus ring 变化）
- Icon: `size-4`

### 30.8 底部用户区

- Avatar: `size-8 rounded-md` 背景 `var(--admin-sidebar-avatar-bg)` + 文字 `var(--admin-sidebar-avatar-foreground)`
- 用户名: `text-sm font-medium text-[--admin-sidebar-foreground]`
- 角色: `text-xs font-normal text-[--admin-sidebar-foreground]`
- 更多按钮: `size-8 rounded-[4px]` hover → `bg-[var(--bg-grey-hover)] text-gray-900`

### 30.9 过渡动画

- 侧边栏展开/收起: `transition-[width] duration-300`
- 内容区跟随: `transition-[margin-left] duration-300`
- 菜单项交互: `transition-all duration-150`

### 30.10 强制规则

1. **`admin-sidebar.tsx` 及 `index.css` 中 `--admin-sidebar-*` 变量仅 miekoyychen 可修改**
2. 其他人不得覆盖 sidebar 组件的样式、token 或结构
3. 如需新增侧边栏功能，需经 miekoyychen 审核

---

## 31. Transfer 穿梭框组件（owner: addietang）

> **Owner**: addietang  
> **源文件**: `client/src/components/ui/transfer.tsx`  
> **设计参考**: Ant Design Transfer（结构对齐 `index.tsx / Section.tsx / ListBody.tsx / search.tsx / Actions.tsx`）  
> **典型场景**: 弹窗内"从一组备选资产里挑出 N 个"的批量选择，例如 BashPolicy 批量添加策略时选择 AI Agent 资产、批量授权资产、批量绑定标签等。**禁止**业务侧再用 `<Table density="compact" />` + `<Checkbox />` 手搓"伪 Transfer"。

### 31.1 子组件结构

| 子组件 | 对齐 Ant | 职责 |
|---|---|---|
| `Transfer<T>` | `index.tsx` 顶层 | `useData` 派生 left/right 数据集；`useSelection` 派生左右内部勾选；`moveTo(direction)` 统一移动入口 |
| `TransferSection` | `Section.tsx` | 单侧面板：Header(checkbox + 标题 + 计数 + 「清空选择」) + Search + Body + Footer |
| `TransferSearch` | `search.tsx` | `SearchIcon` + `<Input>`，受控 `value/onChange` |
| `TransferTableBody` | `ListBody.tsx` + `ListItem.tsx` | 用 `<Table density="compact">` 渲染，自带分页、headerChecked / Indeterminate 计算 |
| `TransferActions` | `Actions.tsx` | 中间 `>` `<` 按钮，仅 `mode="batch"` 渲染 |

### 31.2 移动模式

- **`mode="instant"`（默认）**：左侧 row checkbox 勾上 → 立刻搬到右侧；右侧 header 显示「清空选择」link；右侧每行末尾显示 `X` 移除。**优先用这个**——交互最直接，弹窗内最省空间。
- **`mode="batch"`（Ant 经典）**：左右各维护内部勾选；中间渲染 `>` / `<` 按钮做批量穿梭。仅在"用户希望确认后才生效"时使用。
- **`oneWay={true}`**：右侧不可移回。`instant` 模式下隐藏行末 X，`batch` 模式下隐藏中间 `<` 按钮。

### 31.3 颜色 / 字号 token 规范

| 槽位 | Token | 来源章节 |
|---|---|---|
| 外壳描边 | `border-border`（= `var(--border)` = `#EAEEF4`） | §28 |
| 面板内分割线（header / search / footer） | `border-[#f0f0f0]` | §15 |
| 面板头部底色 | `bg-[var(--bg-grey-normal)]`（= `#FAFBFD`） | §0 |
| 标题文字 | `<BodyMedium>`（emphasis #0A0A0A） | §0 |
| 计数 / 已选 N 项 / 共 N 项 | `<MetaText>`（muted #737373） | §0 |
| 空态文案 | `<HelperText as="span">`（weak #94A3B8） | §0 |
| 「清空选择」link | `<Button variant="link" size="sm">` | §4 |
| Search 图标 | `text-[var(--text-weak)]` | §0 |
| Search 输入框 | 内部 `<Input>` = `border-border` | §28 / §5 |
| 行末 X 按钮 | `text-[var(--text-muted)]` + `hover:bg-[var(--bg-grey-hover-subtle)] hover:text-[var(--text-emphasis)]` | §0 |
| 表格行高 / 字号 / 行分割线 / hover | `<Table density="compact">` 全局态 | §15 |
| 分页 | `<Pagination simple size="small">` | §26 |
| 中间穿梭按钮（batch） | `<Button variant="outline" size="icon">` | §4 |

### 31.4 Props 速查

```ts
<Transfer<T>
  dataSource={T[]}                 // 全集（包含已选项）
  targetKeys={string[]}            // 受控：当前已选 keys
  onChange={(nextKeys, dir, moveKeys) => void}

  rowKey?={'key' | (item) => string}  // 默认 item.key
  titles?={[ReactNode, ReactNode]}    // 默认 ['全部', '已选']
  height?={number}                    // 单侧 body 高度，默认 330

  showSearch?={boolean}
  searchPlaceholder?={string | [string, string]}
  filterOption?={(input, item) => boolean}  // 默认全字段不区分大小写包含

  pagination?={ pageSize?: number } | boolean  // 默认 { pageSize: 10 }

  isItemDisabled?={(item) => boolean}
  renderDisabledTrigger?={(item, defaultCheckbox) => ReactNode}  // 在禁用行外侧挂 Tooltip 用

  columns?={TransferColumn<T>[]}      // 同时作用左右
  leftColumns?={TransferColumn<T>[]}  // 仅左侧
  rightColumns?={TransferColumn<T>[]} // 仅右侧

  mode?={'instant' | 'batch'}   // 默认 'instant'
  oneWay?={boolean}             // 右侧不可移回

  // 仅 batch 模式：受控内部勾选
  selectedKeys?={string[]}
  onSelectChange?={(sourceKeys, targetKeys) => void}
/>
```

### 31.5 标准用法（instant 模式 + 禁用项 + Tooltip）

```tsx
<Transfer<HostItem>
  dataSource={(aiAgentHostList ?? []).map((h) => ({ ...h, key: h.Quuid }))}
  rowKey="key"
  targetKeys={selectMachine}
  onChange={(nextKeys) => setSelectMachine(nextKeys)}
  showSearch
  searchPlaceholder={['搜索资产名称 / ID / IP', '搜索已选资产']}
  pagination={{ pageSize: 8 }}
  height={300}
  titles={['全部 AI Agent 资产', '已选 AI Agent 资产']}
  isItemDisabled={(h) => h.ProtectType !== 'Flagship'}
  renderDisabledTrigger={(_h, defaultCheckbox) => (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">{defaultCheckbox}</span>
      </TooltipTrigger>
      <TooltipContent>基础版资产请升级到旗舰版以使用该能力</TooltipContent>
    </Tooltip>
  )}
  filterOption={(input, h) => {
    const needle = input.toLowerCase();
    return [h.OpenClawName, h.MachineName, h.InstanceID, h.MachineIp]
      .filter((v): v is string => typeof v === 'string')
      .some((v) => v.toLowerCase().includes(needle));
  }}
  columns={[
    {
      key: 'name',
      header: 'Agent 名称 / ID',
      render: (h) => (
        <div className="min-w-0">
          <div className="truncate text-[var(--text-emphasis)]">
            {h.OpenClawName || h.MachineName || '-'}
          </div>
          <MetaText className="block truncate">{h.InstanceID || '-'}</MetaText>
        </div>
      ),
    },
    { key: 'version', header: '防护版本', width: 100, render: (h) => hostVersionMap[h.ProtectType] ?? '-' },
    { key: 'ip', header: '内网IP', width: 140, render: (h) => h.MachineIp || '-' },
  ]}
/>
```

### 31.6 强制规则

1. **组件源文件 `client/src/components/ui/transfer.tsx` 仅 addietang 可修改**；其他人通过 props 配置使用，不得通过 `className` 覆盖颜色 / 边框 / 圆角 / 字号。
2. **禁止业务侧用 `<Table>` + `<Checkbox>` 手搓"双列穿梭框"** —— 一律使用 `<Transfer>`。
3. 列定义中 **不得在 `render` 内自行拼装 `text-xs` / `text-[var(--text-*)]`**；正文用 Typography（`BodyText` / `BodyMedium`），辅助行用 `<MetaText>`。
4. 禁用项的提示信息**必须**通过 `renderDisabledTrigger` 包 Tooltip 给出原因，不允许把禁用项默默隐藏（违反 §15 表格"始终展示数据全集"原则）。
5. 弹窗内首选 `mode="instant"` + `pagination={{ pageSize: 8 }}` + `height={300}`；如果业务需要"先选再确认"，再切换到 `mode="batch"`。
6. 如确需新增 props（例如新的 footer 槽位），先更新本节规范并经 addietang 审核。

---

## 32. NumberCard 数字卡片组件（owner: addietang）

> **Owner**: addietang
> **源文件**: `client/src/components/ui/number-card.tsx`
> **设计参考**: Tokens 监控概览区四张卡片（总请求数 / 输入 Tokens / 输出 Tokens / 总 Tokens）
> **典型场景**: 管控端 / 用户端的 KPI 概览卡（Dashboard 顶部、Tokens 监控、配额消耗、运营统计）。**禁止**业务侧再用 `<SurfaceCard>` + 内联 SVG + `<StatNumber>` 手搓"伪 NumberCard"。

### 32.1 子组件结构

| 子组件 | 职责 |
|---|---|
| `NumberCard` | 主组件，等于 `SurfaceCard p-5` + `icon + label`（顶部）+ `StatNumber`（中部）+ 可选 `extra` / `footer` 扩展槽 |
| `GradientIcon` | 把任意 SVG `<path>` 包装成 18×18、黑→蓝（`#202020 → #0080FF`）radialGradient 图标；内部 `React.useId` 生成唯一 gradient id，避免多卡同 id 互相覆盖 |
| `RequestsIcon` / `InputTokensIcon` / `OutputTokensIcon` / `TotalTokensIcon` | 与 Tokens 监控页 1:1 对齐的 4 枚内置渐变图标 |

### 32.2 视觉令牌

| 槽位 | Token | 来源章节 |
|---|---|---|
| 容器 | `<SurfaceCard>` p-5（20px） | §3 / §20 |
| 容器圆角 / 描边 / 阴影 | 由 `SurfaceCard` 提供（4px 圆角 + 1.5px 浅描边 + L1 阴影） | §2 / §3 |
| 图标尺寸 | 18×18 | — |
| 图标渐变 | radialGradient `#202020 → #0080FF` | — |
| 图标 ↔ 标题间距 | `gap-2`（8px） | — |
| 标题 ↔ 数字间距 | `mb-3`（12px） | — |
| 标题文字 | `text-sm font-medium text-[var(--text-title)]`（14px / Medium / `--text-title`） | §0 |
| 主数字 | `<StatNumber>`（24px / Semibold / DIN / tabular-nums） | §0 |
| `extra` 槽（数字旁附加进度条 / 徽标） | 容器 `flex items-center gap-8`（32px） | — |

### 32.3 Props 速查

```ts
<NumberCard
  icon?={ReactNode}              // 18×18 图标，建议使用 GradientIcon 或内置 4 枚
  label={ReactNode}              // 标题 / 指标名称
  value={ReactNode}              // 主数字（自动套 StatNumber，已格式化好的字符串或 ReactNode）
  extra?={ReactNode}             // 数字旁附加内容（百分比标签 / 进度条 / 徽标）
  footer?={ReactNode}            // 数字下方的辅助文字 / 二级指标
  className?={string}            // 仅用于布局微调，禁止覆盖颜色 / 圆角 / 内边距
/>
```

### 32.4 三种标准用法

```tsx
// ① 开箱即用：使用内置 4 枚渐变图标（与 Tokens 监控页一致）
import {
  NumberCard, RequestsIcon, InputTokensIcon, OutputTokensIcon, TotalTokensIcon,
} from "@/components/ui/number-card";

<div className="grid grid-cols-4 gap-5">
  <NumberCard icon={<RequestsIcon />}     label="总请求数"    value="1,841" />
  <NumberCard icon={<InputTokensIcon />}  label="输入 Tokens" value="533,112" />
  <NumberCard icon={<OutputTokensIcon />} label="输出 Tokens" value="419,040" />
  <NumberCard icon={<TotalTokensIcon />}  label="总 Tokens"   value="952,152" />
</div>

// ② extra 槽：百分比 + 徽标 / 百分比 + 进度条
<NumberCard
  icon={<TotalTokensIcon />}
  label="今日全局配额消耗"
  value="68%"
  extra={<ProgressBar value={68} max={100} />}
/>

// ③ 自定义 SVG path 包成同款渐变图标
import { GradientIcon } from "@/components/ui/number-card";

<NumberCard
  icon={<GradientIcon><path d="..." /></GradientIcon>}
  label="自定义指标"
  value="—"
/>
```

### 32.5 强制规则

0. **自动识别 → 必须 NumberCard**：只要设计稿 / 需求 / 截图出现「图标（≤ 20px）+ 短标题 + 大号数字（≥ 20px）」三件套（无图表），或文案是 KPI / 概览 / 监控 / 配额 / Tokens / 请求数 / 实例数 / 用量 / 累计 等指标，**一律自动用 `<NumberCard>`**，禁止用 `SurfaceCard + 内联 SVG + StatNumber` / `<div>` 自拼。完整识别清单与例外见 `.codebuddy/skills/clawpro-portable-design-skill/component-specs/number-card.md §0`。
1. **组件源文件 `client/src/components/ui/number-card.tsx` 仅 addietang 可修改**；其他人通过 props 配置使用，不得通过 `className` 覆盖颜色 / 圆角 / 字号 / 内边距。
2. **禁止业务侧用 `<SurfaceCard>` + 内联 SVG + `<StatNumber>` 手搓 KPI 概览卡** —— 一律使用 `<NumberCard>`。
3. **图标尺寸固定 18×18**：使用 `GradientIcon` 或内置 4 枚 NumberCardIcon；如需任意 lucide 图标，请显式 `className="h-[18px] w-[18px]"`，并保留品牌色 `#1447E6`。
4. **数字必须走 `value` 槽**（组件内部自动套 `StatNumber`），禁止在 `value` 内手写 `text-2xl font-bold` 等字号 / 字重。
5. **标题文字色固定 `--text-title`**，不允许通过 `label` 内的 `className` 改色；如需 Tooltip / 帮助说明，可在 `label` 内嵌 `<UITooltip>`，但文字外层颜色仍由组件控制。
6. `extra` 槽与数字之间间距固定 `gap-8`（32px），禁止手写 wrapper 改间距；`footer` 槽与数字间距固定 `mt-2`。
7. 如确需新增视觉变体（例如带趋势箭头、迷你折线图）或新增内置图标，先更新本节规范并经 addietang 审核。

---

## 33. 引导体系组件（Onboarding Guide System）

> **源码路径**: `client/src/components/onboarding/`
> **统一入口**: `import { ... } from "@/components/onboarding"`（见 `index.ts`）
> **预览页**: `/preview/onboarding-guide`（由 `OnboardingDemoPanel` 渲染，右下角浮窗可逐个开启体验）
> **设计参考**: Figma「版本更新感知 / 新手引导体系」
> **定位**: 一套**全局浮层级**引导组件，叠加在真实管控端 / 用户端页面之上，用于「版本更新感知」「新功能引导」。所有组件均为受控组件（`open` + `onClose`），自带进出场动画，禁止业务侧手写同类浮层。

### 33.0 组件总览

| # | 组件 | 类型 id | 适用端 | 阻断性 | 对应场景 | 一句话 |
|---|------|---------|--------|--------|----------|--------|
| 1 | `GuideGlobalModal` | `global-modal` | both | **强阻断**（全屏蒙版） | 1.1/1.4/3.x/4.x | 影响面极大的更新：680×512 居中全局弹窗，单条 / 多条轮播 |
| 2 | `GuideModuleFloat` | `module-float` | both | 非阻断 | 1.2/2.6 | 右下角模块级浮窗，单 CTA / 多条翻页 |
| 3 | `GuideNavBubble` | `nav-bubble` | admin | 非阻断 | 1.5/1.3 | 依附导航项的新功能预览气泡（带 NEW 标 + 预览图 + 去看看） |
| 4 | `GuidePointBubble` | `point-bubble` | both | 非阻断 | 2.1~2.3 | 单 UI 点对点提示气泡：纯文本 / 文本+按钮 / 文本+图片 / 推送通知 |
| 5 | `GuideUpdateBar` | `update-bar` | admin | 非阻断（强提醒） | 2.5/2.6/5.x/6.1 | 导航下方 sticky 公告条（琥珀告警样式 + 查看详情） |
| 6 | `GuideChangelogDrawer` | `changelog-drawer` | admin | 半阻断（右抽屉） | 所有层级汇总 | 右侧滑出的版本更新记录抽屉 |
| 7 | `GuideHighlightBubble` | `highlight-bubble` | both | **强阻断**（Spotlight 蒙版） | 1.2/2.1~2.3 | 高亮镂空遮罩 + 步骤气泡，多区域串联导航 |

**通用约束（所有引导组件共享）**：
1. 受控渲染：`open=false` 时返回 `null`，禁止用 CSS `display:none` 隐藏。
2. 颜色 / 阴影 / 圆角全部走 `index.css` 中 `--guide-bubble-*`、`--module-float-*` 语义 token，禁止硬编码（除已对齐 Figma 的全局弹窗内部固定像素）。
3. 端差异只通过 `endpoint`（`"admin"` | `"tenant"`）切换按钮圆角等风格，禁止业务侧改写。
4. 层级（z-index）已内置且分层，业务侧不要覆盖：UpdateBar `49` → NavBubble/PointBubble `9980/9985` → ModuleFloat/HighlightBubble `9990` → ChangelogDrawer `9994/9995` → GlobalModal `9999` → 控制面板 `99999`。

### 33.1 CSS Token（统一定义于 `client/src/index.css`）

**气泡类（PointBubble / 通用浮层）`--guide-bubble-*`**

| Token | 值 | 用途 |
|---|---|---|
| `--guide-bubble-bg` | `#FFFFFF` | 浅色气泡底 |
| `--guide-bubble-border` | `#EAEEF4` | 气泡描边（等同 `--border`） |
| `--guide-bubble-arrow-stroke` | `#E5E5E5` | 三角箭头描边 |
| `--guide-bubble-radius` | `4px` | 气泡圆角 |
| `--guide-bubble-shadow` | `0 8px 48px -12px rgba(0,0,0,.1), 0 1px 3px rgba(0,0,0,.05)` | 浅色气泡阴影 |
| `--guide-bubble-title` | `#000000` | 标题纯黑 |
| `--guide-bubble-desc` | `rgba(0,0,0,.7)` | 描述/正文 |
| `--guide-bubble-close` | `rgba(0,0,0,.7)` | 关闭图标 |
| `--guide-bubble-btn-primary-bg` / `-text` | `#202020` / `rgba(255,255,255,.9)` | 主按钮 |
| `--guide-bubble-btn-secondary-bg` / `-border` / `-text` | `#FFFFFF` / `#E5E5E5` / `#000000` | 次按钮 |
| `--guide-bubble-push-gradient` | `linear-gradient(180deg,#2C59E9,#5980FF)` | 推送通知/深色气泡渐变 |
| `--guide-bubble-push-shadow` | `0 8px 48px -12px rgba(38,58,120,.28), 0 1px 3px rgba(38,58,120,.05)` | 深色气泡阴影 |
| `--guide-bubble-push-title` / `-desc` / `-arrow` | `#FFFFFF` / `rgba(255,255,255,.7)` / `#2C59E9` | 深色文字/箭头 |
| 步骤指引纯蓝底 | `#2C59E9` | dark + 多步骤模式背景（区别于 push 渐变） |

**浮窗类 `--module-float-*`**：宽 `360px`、padding `12px`、圆角 `4px`、底 `#FFFFFF`、描边 `var(--border)`、副标题 `rgba(0,0,0,.5)`、标题 `#000`、描述 `rgba(0,0,0,.7)`、链接 `var(--text-brand)` `#1447E6`、配图圆角 `4px`、翻页箭头 `38×28` 圆角 `24px`、单条 CTA 黑底白字圆角 `24px`。完整清单见 `index.css`。

### 33.2 GuideGlobalModal · 全局弹窗（强阻断）

**来源**: `GuideGlobalModal.tsx`（严格对齐 Figma node 4081-5304）。

- **结构**: `fixed inset-0` 全屏容器 → 半透明黑色蒙版（`bg-black/50 backdrop-blur-[1px]`，点击关闭）→ 居中 **680×512 / 圆角 8px** 卡片。
- **层次（从底到顶）**: 卡片背景图 `card-bg.png` → 媒体层（视频/图片/渐变占位，淡入淡出 0.35s）→ 底部白色渐变遮罩（保证文字可读）→ 大标题「全站视觉焕新升级 ✦」（24px Semibold 渐变填充 + 投影）→ 箭头/关闭/指示器/底部文案+按钮。
- **变体 `variant`**: `single`（单条，对应用户端样式）/ `carousel`（多条轮播，左右箭头 + 指示器圆点，对应管控端样式）。
- **按钮圆角随 `endpoint`**: admin = `4px`，tenant = `60px` 胶囊。主按钮渐变 `linear-gradient(90deg,#020617 70%,#1447E6 100%)`；传 `secondaryText` 显示双按钮。
- **指示器**: 激活 `18×4 #000`，非激活 `4×4 #CACFDD`，gap 4px。

```tsx
<GuideGlobalModal
  open={open} onClose={close}
  variant="single" | "carousel"
  slides={[{ titleLeft?, titleRight, desc, videoSrc? , imageSrc? }]}
  confirmText="立即体验" secondaryText?
  onConfirm? onSecondary?
  endpoint="admin" | "tenant"
/>
```

### 33.3 GuideModuleFloat · 非阻断浮窗

**来源**: `GuideModuleFloat.tsx`（Figma node 4088:7837）。固定 **右下角** `fixed bottom-6 right-6`，宽 360px。

- **结构**: 外层 column（padding 12 / gap 12）→ 内容区（副标题 + 标题 + 描述/行动链接 + 16:9 配图 672×376）→ 底部操作区。
- **变体 `variant`**: `single`（底部右对齐黑色胶囊 CTA「立即体验」，描述行内带 `跳转链接 →`）/ `multi`（底部左侧「n/N␣␣跳过引导」+ 右侧 `tenant-outline` 翻页箭头，末页右箭头变「我知道了」文字按钮）。
- 按钮统一使用全局 `Button`（`tenant-primary` / `tenant-outline`，`size="claw-sm"`）。

```tsx
<GuideModuleFloat
  open onClose subtitle="消息通知" title="Agent 新版本上线"
  items={[{ subtitle?, title, description, image?, actionText?, actionHref? }]}
  variant="single" | "multi" confirmText="立即体验" onConfirm?
/>
```

### 33.3.1 GuideAdminNotify · 管控端产品动态卡片

**来源**: `GuideAdminNotify.tsx`。管控端非阻断浮窗的具体实现，以「产品动态卡片」形式展示版本更新通知。

**卡片数量与合并规则（强制）**：
- **用户端卡片最多展示 1 张**，管控端卡片最多展示 1 张，整体最多 2 张（各端各一张）。
- **同端自动合并**：如果多人（或多次发布）上传了不同内容的同端卡片，必须自动合并为一张多条内容的卡片（卡片内逐条展示），禁止同端出现 2 张及以上独立卡片。
- 合并后的卡片按时间倒序排列条目，用户可逐条查看。
- 点击卡片「查看详情 / 立即体验」可跳转产品动态抽屉（`ProductUpdatesDrawer`），并高亮对应条目。

```tsx
<GuideAdminNotify
  open onClose
  variant="stacked"
  items={notifyItems}
  onAction={handleAction}
/>
```

### 33.4 GuideNavBubble · 导航预览气泡

**来源**: `GuideNavBubble.tsx`。依附 sidebar / 导航项旁，宽 **300px**，圆角 `rounded-xl`，带指向目标的白色三角箭头（`placement` = right/bottom/left）。

- **内容**: 可选预览图（顶部 140px）+ `NEW` 蓝标 + 标题（14px medium）+ 描述（12px 灰）+ 关闭按钮；传 `href` 时底部显示「去看看 →」（黑底按钮 `bg-gray-900`，28px 高）+「稍后再说」文字按钮。
- 实际接入用 Portal 定位到导航项；演示中通过外层 `fixed` + `style` 定位。

```tsx
<GuideNavBubble
  open onClose title description image?
  placement="right" | "bottom" | "left"
  href? actionText="去看看" style?
/>
```

### 33.5 GuidePointBubble · 单 UI 提示气泡

**来源**: `GuidePointBubble.tsx`（Figma node 4096:9477）。宽 **280px**，气泡 + 三角箭头 + 可选脉冲热点；箭头方向 `placement` = top/bottom/left/right。

- **颜色变体 `variant`**: `light`（白底黑字）/ `dark`（蓝渐变白字）。
- **内容变体 `contentVariant`**:
  - `text-only`（1.1 纯文本，可带 `listItems` 有序列表 ≤3 条）
  - `text-button`（1.2 文本+按钮，可带 `subtitle` 副标题、次要 `secondaryActionText`）
  - `text-image`（1.3 文本+图片，配图 146px，可带 `imageCaption`）
  - `push-notice`（1.4 重点推送，强制蓝渐变 + 图标 + 可选 `noticeImage` 大图）
- **步骤模式**: `showSteps` + `totalSteps>1` → 底部「currentStep/totalSteps」+ 上一步/下一步箭头，末步变文字按钮；`dark` + 多步骤 = 纯蓝底 `#2C59E9` 步骤指引样式。
- **热点 `hotspotShape`**: `circle`（呼吸脉冲圆点，默认）/ `rect`（蓝色圆角矩形标注，`hotspotSize` 控制）。
- 按钮圆角随 `endpoint`：admin = `4px`，tenant = 胶囊；浅色按钮用全局 `Button`（`claw-*` / `tenant-*`）。

```tsx
<GuidePointBubble
  open onClose title description subtitle?
  variant="light" | "dark"
  contentVariant="text-only" | "text-button" | "text-image" | "push-notice"
  image? noticeImage? imageCaption? listItems? tag?
  currentStep? totalSteps? showSteps
  actionText? onAction? secondaryActionText? onSecondaryAction?
  onNext? onPrev?
  placement="bottom" showHotspot hotspotShape="circle" | "rect" hotspotSize?
  endpoint="admin" | "tenant" style?
/>
```

### 33.6 GuideUpdateBar · 强提醒公告条

**来源**: `GuideUpdateBar.tsx`。通过 `createPortal` 插入到 `header.sticky` 之后（fallback 到 `main` 前），`position:sticky; top:64px; z-index:49`，**撑开页面、随滚动固定、不可手动关闭**（强提醒）。

- **样式**: 琥珀告警 `bg-[#FEF3C7]` + `border-[#F59E0B]/30`，`AlertTriangle` 图标 `#D97706`，可选版本标签 `bg-[#FDE68A] text-[#92400E]`，正文 `text-[#92400E]`，右侧「查看详情 ›」回调。
- 内容居中容器 `max-w-7xl mx-auto`。

```tsx
<GuideUpdateBar open message version? onDetail? detailText="查看详情" />
```

### 33.7 GuideChangelogDrawer · 更新记录抽屉

**来源**: `GuideChangelogDrawer.tsx`。右侧滑出抽屉，`z-9994` 遮罩（`bg-black/30 backdrop-blur-[2px]`）+ `z-9995` 面板（宽 `max-w-[420px]`，右滑入场）。

- **结构**: Header（「更新记录」标题 + 说明 + 关闭）→ 滚动区按 `version` 分组（版本标签 sticky）→ 每条 `entry`：层级标签 + 标题 + 描述 + 可选外链图标（hover 显现）。
- **层级标签配色 `tag`**: 结构=紫 / 元素=蓝 / 逻辑=琥珀 / 系统=红 / 跨端=绿。
- 常从 `GuideUpdateBar` 的「查看详情」触发。

```tsx
<GuideChangelogDrawer
  open onClose
  versions={[{ version, date, entries: [{ id, title, description, tag, date, href? }] }]}
/>
```

### 33.8 GuideHighlightBubble · 步骤指引气泡（带呼吸灯 / 矩形标注）

**来源**: `GuideHighlightBubble.tsx`。`fixed inset-0 z-9990 pointer-events-none` 覆盖层，按 `hotspotShape` 在目标元素上绘制 **呼吸灯**（`circle`，圆点贴边缘中点）或 **矩形标注**（`rect`，蓝色圆角描边框按 padding 外扩框住目标）+ 旁置 280px 步骤气泡（复用 `GuidePointBubble` dark 变体）。

- **气泡定位 `bubblePlacement`**: right/left/bottom/top（相对锚点框偏移 `GAP=8px`，呼吸灯额外预留 `HOT=16px`）。
- **多区域串联**: `regions` 数组 + 受控 `currentIndex` / `onIndexChange`；底部「currentIndex+1/N」+ 上一步/下一步，末步「我知道了」；单区域时仅「我知道了」。
- **附加列表 `showList`**: 可在气泡内附加有序列表（`region.listItems`，≤3 条）。
- 按钮圆角随 `endpoint`。

#### DOM 读取 / 定位规则（核心，必须遵守）

步骤指引气泡的热点与气泡位置**不由设计稿写死坐标，而是实时读取真实页面 DOM 测量得到**，确保任何滚动 / 动态布局 / 不同分辨率下都精确贴合目标元素：

1. **选择器优先**：`region.selector`（CSS 选择器）为推荐的定位方式。组件用 `document.querySelector(selector)` 命中目标元素后，调用 **`getBoundingClientRect()`**（视口坐标系）测量其真实 `top/left/width/height` 来绘制热点与贴靠气泡。
2. **兜底坐标**：仅当 `selector` 缺失或未命中元素（元素不存在 / `width=height=0`）时，回退到 `region` 上手写的 `top/left/width/height` 兜底坐标。
3. **双缺失不渲染**：`selector` 未命中且兜底坐标也缺失时**直接 `return null`**，禁止渲染出随机大小 / 位置的浮层。
4. **持续跟随**：测量在 `requestAnimationFrame` 循环内进行（带 rect 相等性比较去抖），实时跟随滚动与动态布局变化；气泡自身尺寸用 `ResizeObserver` 测量，用于自动匹配方向 + 视口边界 clamp，保证屏幕边缘元素的气泡不被截断。
5. **自动滚动定位**：切换步骤（`currentIndex` 变化）时，对命中的 `selector` 元素执行 `scrollIntoView({ behavior:"smooth", block:"center" })`，把目标滚动到可视区域中央再标注。
6. **矩形外扩 `padding`**：`hotspotShape="rect"` 时矩形框按 `region.padding`（默认 6px）向外扩展框住目标；气泡贴靠 `rect` 模式用外扩框、`circle` 模式用元素本身作为锚点。
7. **选择器稳定性**：业务侧应优先使用稳定且唯一的选择器（如 `data-guide="xxx"` 自定义属性），避免依赖易变的 class / nth-child，防止改版后命中漂移。

```tsx
<GuideHighlightBubble
  open onClose
  regions={[{
    id,
    selector?,        // 推荐：目标元素 CSS 选择器（实时 getBoundingClientRect 测量）
    top?, left?, width?, height?,  // 兜底坐标（selector 缺失/未命中时使用）
    padding?,         // rect 模式外扩内边距，默认 6
    title, description, bubblePlacement?, listItems?,
  }]}
  currentIndex onIndexChange
  hotspotShape="circle" | "rect"
  showList?
  endpoint="admin" | "tenant"
/>
```

### 33.9 场景 → 组件映射（版本更新感知场景清单）

> **详细场景规则**：见 `.codebuddy/skills/update-awareness-skill/references/scenario_rules.md`
> **组件映射规则**：见 `.codebuddy/skills/update-awareness-skill/references/surface_rules.md`
> **文案规范**：见 `.codebuddy/skills/update-awareness-skill/references/copy_guidelines.md`

**体验面板三级分类**：

| 分类 | 说明 | 对应组件 |
|------|------|----------|
| 最轻量提示 | 在 UI 附近直接展示提示气泡，打开页面后默认出现 | `point-bubble` |
| 最重量提示 | 仅在系统性重大变更或需用户明确知情同意时使用 | `global-modal` |
| 日常更新提示 | 日常功能更新，管控端展示产品动态卡片，用户端展示非阻断浮窗，通常点击后跳转对应页面衔接气泡 | `module-float` |

**完整场景 → 组件映射**：

| 场景 | 感知要点 | 管控端推荐组件 |
|------|----------|----------------|
| 1.1 新增子页面 | 用户不知道新页面存在 | 导航提示条 + New Tag |
| 1.2 页面重新排布 | 用户找不到原操作位置 | 导航提示条 + 重要操作变更气泡引导 |
| 1.3 功能位置变动 | 用户按旧路径找不到 | 导航提示条 + 页面入口气泡引导 |
| 1.4.1 页面整合（多合一） | 用户不知整合后含原功能 | 导航提示条 + 页面入口气泡 |
| 1.4.2 页面拆分（一拆多） | 用户需找到新入口 | 导航提示条 + 侧边栏说明详情 + 页面入口气泡引导 |
| 1.5.1 页面入口下线 | 用户以为功能丢失 | 预告期：导航提示条 + 禁用入口说明气泡；告知期：导航提示条 |
| 1.5.2 页面入口新增 | 用户可能长期不点击 | 非一级导航：导航提示条 + New Tag；一级导航：导航栏旁展示气泡 |
| 2.1 新增按钮/操作入口 | 用户可能视而不见 | 重要：导航提示条 + 功能入口附近气泡；次级：进入页面时展示气泡 |
| 2.2 新增表格列/字段 | 用户不理解新列含义 | 重要：导航提示条 + 功能入口附近气泡；次级：进入页面时展示气泡 |
| 2.3 新增筛选/排序/分组 | 不易被主动发现 | 重要：导航提示条 + 功能入口附近气泡；次级：进入页面时展示气泡 |
| 2.4 名称/文案变更 | 用户困惑是否同一功能 | 新名称右侧标注原名 或 元素附近解释气泡 |
| 2.5 新功能 New Tag | 轻量感知，需有下线时间 | New Tag（默认 14 天，最长 30 天） |
| 2.6 细节优化叠加 | 按影响程度分级 | 纯视觉：不展示；路径大幅调整：导航提示条 + 常驻引导；已知问题修复：导航提示条 + Alert |
| 3.1 底层逻辑变更 | 行为结果可能不同 | 导航提示条 + 对应页面内 Alert |
| 3.2 规则/策略变更 | 旧规则操作会被拒绝 | 重点：导航提示条 + 对应页面 Alert；普通：对应组件下方规则说明 |
| 3.3 计费/配额变更 | 影响成本和使用范围 | 重点：导航提示条 + 对应页面 Alert；普通：对应组件下方规则说明 |
| 4.1 账号体系变更 | 最高等级变更 | 导航提示条 + 页面 Alert + 必要时强提醒弹窗 |
| 4.2 权限体系变更 | 用户能力变化 | 导航提示条 + 页面 Alert + 权限影响说明 |
| 4.3 数据合规/隐私变更 | 需用户明确知情/同意 | 导航提示条 + 页面 Alert + 必要时确认弹窗 |
| 5.1 C端→管控端 | 管理员需了解 C 端变化 | 导航提示条 + 对应配置页 Alert |
| 5.2 管控端→C端 | 配置即时生效告知用户 | 管控端导航提示条 + C 端浮窗/气泡 |

**打扰度分级**（默认选择能解决认知问题的最低打扰度组合）：
- 高：强提醒弹窗、页面 Alert、常驻操作引导
- 中：导航提示条、侧边栏说明详情、气泡引导
- 低：New Tag、新名称右侧标注、组件下方规则说明

### 33.10 强制规则

1. **统一来源**：所有引导浮层必须使用 `@/components/onboarding` 导出的引导组件（GlobalModal / ModuleFloat / AdminNotify / NavBubble / PointBubble / UpdateBar / ChangelogDrawer / HighlightBubble / NewTag / ProductUpdatesDrawer 等），禁止业务页面手写 `fixed` + `absolute` 自拼引导弹窗/气泡/公告条。
   - **共享基础设施**：行为参数 / 埋点 / 持久化 / 气泡队列 / 文案校验 / i18n / New Tag 时长统一走 `onboardingShared.ts` + `onboardingHooks.ts`（`resolveBehavior` / `trackOnboarding` / `buildPersistenceKey` / `isDismissed` / `useBubbleQueue` / `useFocusTrap` / `useExposure` 等），禁止业务侧各自实现一套行为/埋点/持久化逻辑。详见 `docs/引导组件规范汇总.md §七`。
2. **受控渲染**：通过 `open` + `onClose` 控制，`open=false` 必须 `return null`，禁止常驻 DOM。
3. **颜色禁硬编码**：浅色/深色/渐变全部走 `--guide-bubble-*` / `--module-float-*` token；新增视觉变体需先更新本节并在 `index.css` 增补 token。
4. **端风格只用 `endpoint`**：按钮圆角等 admin/tenant 差异仅通过 `endpoint` 切换，禁止 `className` 覆盖圆角/底色/字号。
5. **层级不可改**：z-index 已内置分层（见 §33.0），业务侧不得自定义覆盖，避免与 Dialog / Toast 等抢层。
6. **强阻断需蒙版**：`GuideGlobalModal` 与 `GuideHighlightBubble` 必须保留半透明黑色蒙版（`bg-black/50` / `rgba(0,0,0,0.5)`），不得改为透明，确保「强阻断」语义可见。
6.1. **GuideGlobalModal 优先级最高**：当有多个一级弹窗（Dialog / AlertDialog / Sheet 等）同时出现时，`GuideGlobalModal` 必须优先展示。其 z-index 为 `9999`，高于所有业务弹窗（z-50），确保全局引导弹窗始终在最顶层，不会被业务弹窗遮挡。业务弹窗应在 GuideGlobalModal 关闭后再展示。
7. **预览体验**：调试任意组件统一在 `/preview/onboarding-guide` 页右下角「用户引导模拟」面板逐个开启，禁止在生产业务页常驻演示面板（`OnboardingDemoPanel` 仅供产研测试）。
8. **DOM 实时定位**：`GuideHighlightBubble` 的热点与气泡位置必须通过 `region.selector` 实时读取真实页面 DOM（`querySelector` + `getBoundingClientRect`）测量得出，禁止把热点 / 气泡的绝对坐标写死在业务代码里（手写 `top/left/width/height` 仅作为 selector 未命中时的兜底）；选择器应优先使用稳定唯一的 `data-guide` 自定义属性，避免依赖易变的 class / nth-child。详见 §33.8「DOM 读取 / 定位规则」。
9. **气泡数量与互不遮挡（强制）**：同一界面任意时刻**最多同时出现两个气泡**（`GuidePointBubble` / `GuideHighlightBubble` 步骤气泡 / `GuideNavBubble` 等任意气泡类组件，合计 ≤ 2 个）；**禁止同时出现 3 个及以上气泡**，超出时必须排队，前一个关闭后才能展示下一个。
   - **更新内容较多时使用分步骤引导**：当同一页面需要标注 3 个及以上变更点时，**禁止同时弹出多个独立气泡**，必须改用 `GuideHighlightBubble`（步骤指引气泡）将多个标注区域串联为分步骤引导流程（步骤 1/N → 2/N → …），用户逐步点击"下一步"依次查看，避免多气泡同屏堆叠。
   - **禁止相互重叠**：同屏的两个气泡的可视区域（含三角箭头、阴影范围）**不得相互重叠 / 交叠**，必须有清晰间距分隔。
   - **禁止组件遮挡**：气泡不得遮挡另一个引导组件（如气泡压住另一气泡的标题 / 关闭按钮 / 内容，见反例图），也不得被对方裁切。两个气泡贴靠不同锚点时，定位算法（方向自动翻转 + 视口 clamp）必须在检测到与另一气泡 rect 相交时主动避让（改方向或换锚点），无法避让时则改为排队展示。
   - 反例：左侧 `point-bubble`（"标题文本介绍"）与右侧 `highlight-bubble` 步骤气泡（"设置平台名称与品牌"）边缘交叠、互相遮挡——属违规，需通过避让或排队消除。
10. **New Tag 存续时间（强制）**：`GuideNewTag` 的展示时间**建议不超过两周（14 天）**。超过 14 天仍挂着 New 标识会失去"新功能"语义，反而干扰用户判断。必须在接入时明确设定下线条件（到期自动移除、用户首次点击后移除、或按曝光次数移除），最长不得超过 30 天。

---

## 34. DarkVeil 动态背景组件（owner: miekoyychen）

> **Owner**: miekoyychen
> **源文件**: `client/src/components/ui/DarkVeil.tsx`（默认导出 `DarkVeil`，依赖 `ogl`）
> **设计参考**: 管控端「开通云开发能力」hero（`client/src/pages/admin/CloudDevActivation.tsx`）
> **适用范围**: **管控端开通页 / 能力介绍 hero 的装饰性动态背景**（WebGL CPPN 流动纹理）。**纯背景组件**，只负责观感，不承载任何信息 / 交互 / 可点击元素。**不是通用背景**——普通列表 / 表单 / 详情 / Dashboard / 设置页禁止滥用。完整定位、Auto-Trigger 与跨仓兜底分级见 `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dark-veil.md`。

### 34.1 何时用（命中条件，详见 dark-veil.md §0 Auto-Trigger）

| 条件（需同时满足 A + B） | 说明 |
|---|---|
| **A. 场景** = 管控端功能开通页 / 能力介绍 hero / 首次引导空态的**顶部 hero 区** | 不是整页背景，是 `SurfaceCard` 内顶部 hero 区块的局部背景 |
| **B. 设计意图**明确要「动态流动 / 光效 / 科技感」氛围，且设计师已拍板 | 仅靠 AI 审美判断不足以引入；模糊时记 `conflict-log.md` 标 `needs-design-confirmation` |

**不命中即不要用**：列表 / 表单 / 表格 / 详情 / 设置等功能页主体区、整页 `body` / `AdminSidebarInset` 全屏背景、把 DarkVeil 当内容容器直接铺文字 / 按钮。

### 34.2 hero 三层结构

> hero 区是 `SurfaceCard`（`overflow-hidden`）内的 `relative overflow-hidden` 容器，自下而上严格三层 + 内容层；三层背景全部 `pointer-events-none`。

| 层 | 角色 | 关键写法 |
|---|---|---|
| 第 0 层（最底） | 统一基底色 | `pointer-events-none absolute inset-0 bg-[#E0EBFE]` |
| 第 1 层 | DarkVeil canvas | `pointer-events-none absolute inset-0 h-full w-full`，顶部 `maskImage` 淡出露基底 |
| 第 2 层 | 柔化收束叠层 | `pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]` |
| 内容层 | 文字 / 按钮 / 卡片 | `relative z-10`，永远压在背景三层之上 |

### 34.3 hero 参数配方（抄 `CloudDevActivation.tsx`）

```tsx
import DarkVeil from "@/components/ui/DarkVeil";

<div className="relative overflow-hidden px-[60px] py-12">
  <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />
  <DarkVeil
    speed={1.1}
    warpAmount={1.1}
    noiseIntensity={0.05}
    tintColor="#B2C3FF"
    className="pointer-events-none absolute inset-0 h-full w-full"
    style={{
      transform: "translateY(72px)",
      maskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
      WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
    }}
  />
  <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />
  <div className="relative z-10">{/* 图标 + 标题 + 描述 + 按钮 + 权益卡 */}</div>
</div>
```

| 槽位 | 值 | 说明 |
|---|---|---|
| 基底色 | `#E0EBFE` | 第 0 层 + 第 2 层底部收束同色，无缝 |
| 纹理着色 `tintColor` | `#B2C3FF` | 输出「白底 + 该色单色流动纹理」；hero 必须传 |
| `speed` / `warpAmount` / `noiseIntensity` | `1.1` / `1.1` / `0.05` | 调氛围强弱只动这三个，不动三件套结构 |
| 顶部淡出蒙版 | `linear-gradient(to bottom, transparent 0%, #000 22%)` | 顶部 22% 渐隐露基底 |
| 纵向偏移 | `transform: translateY(72px)` | 飘带下移到 hero 中下部，canvas 仍铺满不露白 |

### 34.4 跨仓兜底分级（详见 dark-veil.md §9，禁止改写 L0/L1/L2 含义）

| 档 | 名称 | 依赖 | 何时用 |
|---|---|---|---|
| **L0** | 完整移植（首选） | `ogl` + WebGL | 宿主仓能装 ogl 且支持 WebGL，复制整文件 1:1 还原 |
| **L1** | 静态 CSS 兜底（migration-map 默认档） | 纯 CSS | 不便引 ogl / WebGL，但要保留蓝紫流动观感；见 `portable/css/dark-veil.css` |
| **L2** | 纯色 / 截图兜底（最低） | 零脚本 | 禁脚本 / 静态导出 / 低端 / `prefers-reduced-motion` |

> 跨仓接入**最少做到 L1**；唯一新依赖是 `ogl`。`references/migration-map.md` 把「DarkVeil 动态背景」标为 **L1** 默认兜底档。

### 34.5 强制规则

1. **组件源文件 `client/src/components/ui/DarkVeil.tsx` 仅 miekoyychen 可修改**；其他人通过 props 配置使用，不得改 shader / 清理逻辑（`loseContext`）去「精简」。
2. **只在命中 34.1（开通页 / 能力 hero + 设计师拍板）时使用**，禁止扩散到列表 / 表单 / 详情 / 设置 / Dashboard / 整页背景，禁止用于 Tenant / Landing（Tenant 走白底 + 极淡蓝雾）。
3. **必须铺「基底 `#E0EBFE` + DarkVeil + 收束叠层 `via-white/10 to-[#E0EBFE]`」三件套**，内容层 `relative z-10`；背景三层全部 `pointer-events-none`，不得在 canvas 上落可点击 / 可聚焦元素。
4. **单页最多 1 个 DarkVeil 实例**（WebGL 上下文昂贵）；`tintColor` 贴合蓝紫基底，DPR 已封顶 2、`speed` 不超过 1.5。
5. **宿主仓无 ogl / WebGL 时按 L0/L1/L2 分档兜底，至少做到 L1 静态 CSS**，不是直接空白；尊重 `prefers-reduced-motion`，必要时降到 L2。
6. 如确需调整 hero 配方常量（基底色 / tint / 蒙版）或新增使用场景，先更新本节与 `component-specs/dark-veil.md` 并经 miekoyychen 审核。

