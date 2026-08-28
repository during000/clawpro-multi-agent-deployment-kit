# Typography

## 1. Purpose

- 统一全平台文字语义组件，按语义选组件而非按字号手写 class。
- 默认色由 Typography 组件提供，业务侧不再手写 `text-gray-*` / `text-[#...]`。
- 字号、字重、行高、字体族、默认颜色绑定在同一个组件里，避免语义漂移。

## 2. Scope

- 适用端：Tenant 主端；Admin / Shared 可按需复用数字/代码类组件
- 文件：`client/src/components/ui/Typography.tsx`
- 研究页：`client/public/research/tenant-typography-audit.html`

## 3. Design Principles

1. 文字按语义选组件：页面标题 → `TenantPageTitle`，卡片标题 → `CardTitle`，正文 → `BodyText`，辅助 → `MetaText`
2. 默认色由 Typography 组件提供，业务不手写颜色
3. 字号/字重/行高/字体/颜色绑定在组件内
4. 组件内部文字也优先复用 Typography token
5. 只通过 `tone` 切换语义色，`className` 仅用于布局（`mt-1`、`truncate`、`text-center`）

## 4. Color Tokens (tone)

> 颜色统一**引用**运行时 `--text-*` slate / 蓝灰语义 token（与 `Typography.tsx`、`tokens/colors.md §3`、`foundation.md §4` 完全一致），本表只引用、不复写 hex；调亮度改 `index.css` 的 `--text-*` 单点即可。

| `tone` | Token (class) | Value | Usage |
|--------|----------------|-------|-------|
| `primary` | `text-[var(--text-title)]` | `#0F172A` | 标题、卡片标题、主内容 |
| `emphasis` | `text-[var(--text-emphasis)]` | `#020617` | 强调文字、按钮文字、关键字段 |
| `body` | `text-[var(--text-body)]` | `#1E293B` | 正文主内容 |
| `secondary` | `text-[var(--text-secondary)]` | `#334155` | 描述性正文、补充说明、次级信息、表格次要字段 |
| `muted` | `text-[var(--text-muted)]` | `#64748B` | 时间、描述、辅助说明、表头 |
| `weak` / `helper` | `text-[var(--text-weak)]` | `#94A3B8` | 占位、空状态、极弱提示、HelperText |
| `brand` | `text-[var(--text-brand)]` | `#1447E6` | 链接、活跃态、步骤标识 |
| `danger` | `text-[var(--text-danger)]` | `#DC2626` | 危险操作、错误提示 |
| `inherit` | `text-inherit` | 继承 | 放在已定义颜色的父级中 |

## 5. Component Hierarchy

| Component | Default Tag | Size / Weight / Leading | Default Tone | Usage |
|-----------|-------------|-------------------------|--------------|-------|
| `TenantHeroTitle` | `h1` | 26px / Medium / 35.56px | `primary` | Hero 标题 |
| `TenantPageTitle` | `h1` | 24px / Medium / 1.4 | `primary` | 页面标题、详情页主标题 |
| `TenantDocTitle` | `h1` | 20px / Semibold / 1.4 | `primary` | 帮助文档、文章标题 |
| `SectionTitle` | `h2` | 18px / Medium / 1.4 | `primary` | 大模块标题 |
| `PanelTitle` | `h2` | 16px / Semibold / 1.4 | `primary` | Dialog / Sheet / 卡片区块标题 |
| `CardTitle` | `h3` | 14px / Medium / 1.5 | `primary` | 卡片名、列表项标题 |
| `BodyText` | `p` | 14px / Regular / 1.5 | `body` | 普通正文（非表格） |
| `BodyMedium` | `span` | 14px / Medium / 1.5 | `emphasis` | 按钮、Tab、Label、列表主字段 |
| `CompactText` | `span` | 13px / Regular / 1.5 | `secondary` | 紧凑列表、轻量描述 |
| `MiniBodyText` | `span` | 12px / Regular / 1.5 | `body` | **表格正文（全表 12px）**、高密度列表 |
| `MetaText` | `span` | 12px / Regular / 1.5 | `muted` | 时间、ID、Tooltip、辅助说明 |
| `MetaMedium` | `span` | 12px / Medium / 1.5 | `muted` | 表头、次级强调 |
| `SmallBodyText` | `span` | 12px / Medium / 12px / tracking 0.18px | `emphasis` | StatusTag 内文字 |
| `TinyText` | `span` | 10px / Semibold / Open Sans | `brand` | New / Beta / 小角标 |
| `StatNumber` | `span` | 24px / Bold / DIN | `emphasis` | 统计数字、额度数字 |
| `InlineNumber` | `span` | 14px / DIN / tabular | `body` | 行内数字、百分比（非表格） |
| `CodeText` | `code` | 12px / Menlo | `secondary` | ID、Token、路径、命令 |
| `StepText` | `span` | 14px / Medium / Menlo | `brand` | 步骤编号 |
| `UrlText` | `span` | 14px / Regular / PingFang SC / 1.5 / break-all / #020617 | `inherit` | URL、回调地址、版本号 |

## 6. Font Stacks

| Token | Value | Usage |
|-------|-------|-------|
| `--font-sans` | `'PingFang SC', -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif` | 中文 UI 默认 |
| `--font-mono` | `'Menlo', 'Consolas', 'Courier New', monospace` | 代码、路径、Token、ID |
| `--font-din` | `'DIN Alternate', 'DIN', 'Helvetica Neue', sans-serif` | 数字优先 |
| `--font-en` | `'Open Sans', 'Helvetica Neue', sans-serif` | 英文标签 |

## 7. Usage Examples

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

// 页面标题
<TenantPageTitle>技能广场</TenantPageTitle>

// 卡片标题
<CardTitle>GPT-4o 模型</CardTitle>

// 正文 + 描述
<BodyText>Agent 已创建成功</BodyText>
<BodyText tone="secondary">创建于 2024-03-15，状态正常</BodyText>

// 辅助信息
<MetaText>更新时间：2024-03-15 14:30</MetaText>

// 统计数字
<StatNumber>1,234,567</StatNumber>

// 代码/路径
<CodeText>sk-xxxx...xxxx</CodeText>

// URL
<UrlText>https://api.example.com/v1/chat/completions</UrlText>
```

## 8. Variant Selection Decision

| 场景 | 使用组件 |
|------|----------|
| 页面主标题 | `TenantPageTitle` / `TenantHeroTitle` |
| 模块标题 | `SectionTitle` |
| 弹窗/卡片标题 | `PanelTitle` |
| 列表项标题 | `CardTitle` |
| 正文内容 | `BodyText` |
| 按钮/Tab/Label | `BodyMedium` |
| 辅助描述 | `MetaText` / `CompactText` |
| 表头 | `MetaMedium` |
| 表格正文 / 表格内数字 | `MiniBodyText`（全表统一 12px，见 `table.md §3.3`）；**数字/数值列须在单元格追加 `tabular-nums` 做等宽对齐**（真实表格通行做法）。注：`DIN` 字体仅用于非表格的统计大数/行内计数，表格单元格不用 DIN |
| 状态标签内 | `SmallBodyText` |
| 统计大数字 | `StatNumber` |
| 行内数字（非表格） | `InlineNumber` |
| 代码/ID/路径 | `CodeText` |
| 步骤编号 | `StepText` |
| URL/回调 | `UrlText` |

## 9. Prohibitions

- 禁止在业务页面手写 `text-gray-*` / `text-[#xxx]` 表达基础文字色
- 禁止用 `className` 覆盖颜色，颜色只通过 `tone` prop 切换
- 禁止跨层级使用（如正文场景用 `TenantPageTitle`）
- 禁止在业务侧新增 `fontFamily`
- 禁止 `text-sm` / `text-xs` 等原子类替代语义组件

## 代码对照（✅/❌）

### ❌ 错误：手写 text-gray-* 颜色
```tsx
<p className="text-sm text-gray-500">已绑定 12 个实例</p>
<span className="text-xs text-gray-400">5 分钟前更新</span>
```
**为什么错**：`text-gray-500` / `text-gray-400` 与 ClawPro token 不对齐，深浅模式切换会失效。

### ✅ 正确：语义文字组件 + tone
```tsx
<BodyText tone="weak">已绑定 12 个实例</BodyText>
<MetaText>5 分钟前更新</MetaText>
```

---

### ❌ 错误：页面标题手写 h1
```tsx
<h1 className="text-2xl font-bold mb-4">实例列表</h1>
```
**为什么错**：字号/字重/间距不受设计系统约束；端别（Admin 18px / Tenant 20px）也无法分流。

### ✅ 正确：语义化标题
```tsx
{/* Tenant 端 */}
<TenantPageTitle>实例列表</TenantPageTitle>
{/* Admin 端 */}
<AdminPageTitle>实例列表</AdminPageTitle>
```

---

### ❌ 错误：数字直接当文本
```tsx
<div className="text-3xl font-bold">{count}</div>
<div className="text-sm text-gray-500">本月新增</div>
```
**为什么错**：数字未启用 `tabular-nums`，跳变时位数错位；且字重/字号与系统节奏不一致。

### ✅ 正确：StatNumber
```tsx
<StatNumber>{count}</StatNumber>
<MetaText>本月新增</MetaText>
{/* StatNumber 内部已 24px Semibold + tabular-nums */}
```

---

### ❌ 错误：MetaText 加自定义字号
```tsx
<MetaText className="text-[10px]">2026-06-10 14:30</MetaText>
```
**为什么错**：MetaText 已固定 12px / `--cp-text-weak`，覆盖字号会破坏系统一致性。

### ✅ 正确：直接使用
```tsx
<MetaText>2026-06-10 14:30</MetaText>
{/* 12px / --cp-text-weak 已内置 */}
```

---

### ❌ 错误：用 className 改文字色
```tsx
<BodyText className="text-blue-600">查看详情</BodyText>
<CardTitle className="text-red-500">告警中</CardTitle>
```
**为什么错**：颜色应通过 `tone` 走 token，而非硬编码 hex/Tailwind 颜色。

### ✅ 正确：tone prop
```tsx
<BodyText tone="brand">查看详情</BodyText>
<CardTitle tone="danger">告警中</CardTitle>
{/* tone: default | weak | brand | success | warning | danger */}
```
