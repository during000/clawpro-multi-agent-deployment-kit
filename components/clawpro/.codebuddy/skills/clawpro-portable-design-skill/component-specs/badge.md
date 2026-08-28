# Badge

> **用户/Agent 在描述"标签"时，唯一对应组件就是 Badge**——版本号 / 范围 / 类别 / 分类 / new / Beta / Pro / 企业版 / 公共版 / AI 等所有"非状态"的轻量胶囊统一走它。
> 与 `status-tag.md` 严格分工：**有"运行 / 成功 / 异常 / 警告 / 进行中 / 已完成"语义就用 StatusTag；只要用户说"标签 / tag / 角标 / 分类 / 版本"，第一选择永远是 Badge**。

## 0. 语义路由（先看这一段）

| 用户/PRD 中的措辞 | 命中组件 |
|---|---|
| "加个**标签**" / "打个**tag**" / "**角标**" / "**分类**" | **Badge**（本文件） |
| "**版本号**" / "v2.1.0" / "Beta / Alpha / Pro / 企业版 / 公共版 / 默认" | **Badge** |
| "**new** 标 / 新功能标" / "**Beta** 标" / "**AI** 标" | **Badge** |
| "**状态**" / "运行中 / 已停止 / 异常 / 待审核 / 在线 / 已完成 / 失败" | StatusTag（→ `status-tag.md`） |
| "可关闭 / 可点选 / 用户自建标签" | Tag（→ `tag-label.md`） |
| "数字红点 / 通知气泡 / 99+" | NumberIndicator（宿主，不走 Badge） |

> 命中规则：用户语义里出现"状态/运行/进度/在线"等动词性词 → StatusTag；出现"标签/版本/分类/角标"等名词性分类词 → Badge。

## 1. Purpose

- 统一版本 / 范围 / 类别 / new 角标 / 一切"标签"的视觉标准（圆角胶囊、12px、不带交互）。
- 与 StatusTag 解耦：Badge 不参与状态色板，不承担运行状态语义。
- 锁住 4 种 variant + 4 种 custom color，禁止业务侧自拼"小胶囊"。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景（凡是用户语义里出现"标签 / tag / 角标 / 分类 / 版本"都走 Badge）：
  - 版本号（v2.1.0 / Beta / Alpha）
  - 范围 / 类别（企业版 / 公共版 / 默认 / 公共 / 私有）
  - new / coming-soon 角标
  - 卡片右上角中性分类
  - 行内特性标（AI / Pro / 实验 / Beta）
- 不适用场景：
  - 运行 / 成功 / 异常 / 警告 / 进行中 / 已完成 / 在线 → `status-tag.md`（用户说"状态"全部走 StatusTag）
  - 用户自建可关闭 Tag → `tag-label.md`
  - 主操作 / 链接（Badge 是非交互元素）
  - 通知数字角标（小红点 / 数字气泡）→ 走宿主 NumberBadge / Indicator，不复用 Badge

## 3. Visual Standard

### 3.1 几何固定值

| 项 | 值 |
|---|---|
| 圆角 | `rounded-full`（9999px） |
| Padding | `px-2.5 py-0.5`（10px / 2px） |
| 字号 | 12px / Regular |
| 高度 | 由 padding + line-height 决定，约 20px |
| 边框 | 1px（仅 outline 显示，其它 variant 走 transparent border 占位） |
| Icon 尺寸 | 12×12（`[&>svg]:size-3`），与文字 `gap-1` |

### 3.2 Variant（4 个）

| Variant | bg | text | border | 用途 |
|---|---|---|---|---|
| `default` | `#0A0A0A`（黑） | `#FFFFFF` | transparent | 强调分类 / 高对比角标（少用） |
| `secondary` | `#F5F5F5` | `#0A0A0A` | transparent | 中性默认（最常用） |
| `outline` | `#FFFFFF` | `#0A0A0A` | `#E5E5E5`（gray-200） | 卡片外的轻分类、版本号 |
| `destructive` | `red-100/60` | `text-red-600` | transparent | 风险类分类（如 deprecated） |

### 3.3 Custom Color（4 色，仅用于 shadcn 官方 Custom Colors 示例语义）

| Color | bg | text |
|---|---|---|
| `blue` | `#E8ECFE` | `#1447E6` |
| `green` | `bg-green-50` | `text-green-700` |
| `purple` | `bg-purple-50` | `text-purple-700` |
| `red` | `bg-red-50` | `text-red-700` |

> 设置 `color` 后会**覆盖** `variant` 的视觉，仅保留尺寸 / 字号。
> 这 4 色只用于轻分类（如 "AI" / "Pro" / "实验"），**禁止用来表达状态**——状态语义请用 `StatusTag`，更明确且与 5 主语义色板一致。

## 4. Anatomy

```text
Badge (span | a, asChild 支持任意元素)
├── data-slot="badge"
├── data-color="blue|green|purple|red"  (optional)
│
├── [optional] <Icon /> 12×12，currentColor
└── <text>{children}</text>
```

- `whitespace-nowrap`：不换行。
- `w-fit shrink-0`：不被 flex 父容器拉伸。
- `gap-1`：icon 与文字间距。
- `transition-[color,box-shadow]`：focus 态切换平滑（仅 asChild 包成 link 时才会触发）。

## 5. States

| State | 视觉 |
|---|---|
| default | 静态展示，不响应 hover |
| asChild={a&} | 包成链接时，`hover:bg-{variant}/90`（轻微变深） |
| focus-visible | `ring-[3px] ring-ring/50`，仅在交互场景出现 |
| aria-invalid | `border-destructive`（极少用，留给表单校验失败的标签） |

> Badge 默认非交互；只有用 `asChild` 包成 `<a>` 时才进入交互态。

## 6. Interaction

无（默认）。
如果业务需要"标签可筛选 / 可关闭"，应该用 `tag-label.md` 里的 Tag，而不是给 Badge 加点击。

## 7. Accessibility

- Badge 是装饰性分类，朗读时随 children 文本朗读，**不要**给它加 `role="status"` 或 `aria-live`。
- 用 `asChild` 包成 `<a>` 时必须保留可见焦点环（`focus-visible:ring-[3px] ring-ring/50`）。
- Color 不是唯一信息载体：如果分类语义重要，必须配合文字（"v2.1.0" / "企业版"），不可只用色块。
- 装饰类 icon 加 `aria-hidden="true"`。

## 8. Demo Repo Usage

- 当前组件：`client/src/components/ui/badge.tsx`
- 典型页面：
  - `client/src/pages/admin/AuditLog.tsx`（操作类型分类）
  - `client/src/pages/admin/Models.tsx`（模型版本 / 范围）
  - `client/src/pages/admin/PermissionTemplates.tsx`（权限模板范围）
  - `client/src/pages/DesignSystemComponents.tsx`（demo / showcase）

```tsx
import { Badge } from "@/components/ui/badge";

// Variants
<Badge>默认</Badge>                          {/* 黑底白字，少用 */}
<Badge variant="secondary">企业版</Badge>     {/* 中性默认 */}
<Badge variant="outline">v2.1.0</Badge>       {/* 版本号最常用 */}
<Badge variant="destructive">已废弃</Badge>

// Custom Colors（轻分类，不用于状态）
<Badge color="blue">AI</Badge>
<Badge color="green">Pro</Badge>
<Badge color="purple">Beta</Badge>

// asChild 用法（包成链接）
<Badge asChild variant="outline">
  <a href="/changelog">v2.1.0</a>
</Badge>

// 带 icon
<Badge variant="secondary">
  <Sparkles />
  New
</Badge>
```

## 9. Portable Fallback

### 9.1 If host repo already has Badge / Tag

- 复用宿主仓已有 Badge 组件即可，但必须在 token 层确认它**不承担状态语义**。
- 如果宿主只有 Tag 没有 Badge，可让 Tag 接受 `variant="badge"` 走相同视觉（rounded-full + px-2.5 + py-0.5 + 12px）。

### 9.2 Minimal React fallback

```tsx
// portable/react/badge.tsx (示意)
type Variant = "default" | "secondary" | "outline" | "destructive";
type Color = "blue" | "green" | "purple" | "red";

const VARIANTS: Record<Variant, string> = {
  default:     "border-transparent bg-[var(--cp-text-title)] text-white",
  secondary:   "border-transparent bg-[var(--muted)] text-[var(--cp-text-title)]",
  outline:     "border-[var(--cp-border)] bg-[var(--cp-surface)] text-[var(--cp-text-title)]",
  destructive: "border-transparent bg-red-50 text-red-600",
};

const COLORS: Record<Color, string> = {
  blue:   "border-transparent bg-[#E8ECFE] text-[#1447E6]",
  green:  "border-transparent bg-green-50 text-green-700",
  purple: "border-transparent bg-purple-50 text-purple-700",
  red:    "border-transparent bg-red-50 text-red-700",
};

export function PortableBadge({
  variant = "secondary",
  color,
  asChild,
  children,
  ...rest
}: {
  variant?: Variant;
  color?: Color;
  asChild?: boolean;
  children: React.ReactNode;
}) {
  const Comp: any = asChild ? "a" : "span";
  const cls = [
    "inline-flex items-center justify-center rounded-full border px-2.5 py-0.5 text-xs font-normal w-fit whitespace-nowrap shrink-0 gap-1",
    color ? COLORS[color] : VARIANTS[variant],
  ].join(" ");
  return <Comp className={cls} {...rest}>{children}</Comp>;
}
```

### 9.3 Minimal HTML/CSS fallback

```html
<span class="cp-badge cp-badge-secondary">企业版</span>
<span class="cp-badge cp-badge-outline">v2.1.0</span>
<span class="cp-badge cp-badge-color-blue">AI</span>
```

```css
.cp-badge {
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: 9999px; border: 1px solid transparent;
  padding: 2px 10px; font-size: 12px; font-weight: 400;
  width: fit-content; white-space: nowrap; gap: 4px;
}
.cp-badge-default     { background: var(--cp-text-title); color: #fff; }
.cp-badge-secondary   { background: var(--muted); color: var(--cp-text-title); }
.cp-badge-outline     { background: var(--cp-surface); color: var(--cp-text-title); border-color: var(--cp-border); }
.cp-badge-destructive { background: rgba(254, 226, 226, 0.6); color: #DC2626; }

.cp-badge-color-blue   { background: #E8ECFE; color: #1447E6; }
.cp-badge-color-green  { background: #F0FDF4; color: #15803D; }
.cp-badge-color-purple { background: #FAF5FF; color: #7E22CE; }
.cp-badge-color-red    { background: #FEF2F2; color: #B91C1C; }
```

## 10. Migration Rules

- 旧写法：手写 `span.bg-gray-100.rounded-full` / 用 Badge 表达"运行中" / 用 Badge 表达数字角标。
- 新口径：分类 / 版本 / 范围 / new 角标 → Badge；状态语义 → StatusTag；数字角标 → 宿主 NumberIndicator。
- 兼容期：宿主仓已有 Tag 组件可保留，但要锁住 Badge 与 StatusTag 的分工。
- 禁止：用 Badge 表达运行状态；自由扩 Custom Color；给 Badge 加 hover 反馈伪装可点。

## 11. Do / Don't

Do:

- 默认从 `secondary` / `outline` 选起，能选中性就不选彩色。
- 版本号优先 `outline`。
- 彩色分类用 `color="blue|green|purple|red"`，不要再混入 StatusTag 的 5 主语义色。
- new / Beta / Alpha 角标尽量短（≤ 5 字符）。

Don't:

- 不要用 Badge 表达运行状态。
- 不要自由新增 Custom Color。
- 不要把 Badge 做成 `rounded-[4px]`（那是 Tag）。
- 不要用按钮 / 链接样式假装 Badge。
- 不要把 Badge 用作通知数字角标（容器形态不同）。

## 12. QA Checklist

- [ ] 没有出现"运行 / 成功 / 异常 / 警告"语义放进 Badge
- [ ] 版本号 / 范围 / new 角标走 Badge，没有自拼小胶囊
- [ ] Custom Color 仅在轻分类（不传递状态）下使用
- [ ] 没有给 Badge 加 hover / focus 反馈伪装可点（除非 asChild）
- [ ] 数字角标没有错用 Badge
- [ ] 宿主仓 fallback 可执行

## 13. References

- Demo code: `client/src/components/ui/badge.tsx`
- shadcn Radix UI Badge: https://ui.shadcn.com/docs/components/radix/badge
- 相关 spec: `component-specs/status-tag.md`、`component-specs/tag-label.md`

## 14. 代码对照（✅/❌）

> 与 SKILL.md §2 / status-tag.md §14 同口径。Badge 5 项高频误用 → ClawPro 正确写法。

### 14.1 状态语义不进 Badge

```tsx
// ❌ 用 Badge 表达运行状态，丢失 StatusTag 的语义系统
<Badge variant="outline">运行中</Badge>
<Badge color="green">成功</Badge>
<Badge color="red">异常</Badge>

// ✅ 状态语义全部走 StatusTag（mode="text" 在表格内最常用）
<StatusTag mode="text" variant="green">运行中</StatusTag>
<StatusTag mode="text" variant="red">异常</StatusTag>

// ✅ 分类 / 版本 / 范围才走 Badge
<Badge variant="outline">v2.1.0</Badge>
<Badge variant="secondary">企业版</Badge>
```

### 14.2 不要自由扩 Custom Color

```tsx
// ❌ 在业务文件里自拼 fuchsia / amber / sky 一类自由色
<Badge className="bg-fuchsia-50 text-fuchsia-700">实验</Badge>
<span className="px-2.5 py-0.5 rounded-full bg-amber-100 text-amber-700 text-xs">Beta</span>

// ✅ 只用 4 个 Custom Color（蓝 / 绿 / 紫 / 红）
<Badge color="purple">实验</Badge>
<Badge color="purple">Beta</Badge>

// ✅ 没有合适色就退回中性
<Badge variant="secondary">Beta</Badge>
```

### 14.3 不要把 Badge 做成方角

```tsx
// ❌ 自定义 Tag 形状，破坏 Badge 与 Tag 的视觉分工
<Badge className="rounded-[4px]">企业版</Badge>
<span className="inline-flex h-[22px] rounded-[4px] border px-2 text-xs">企业版</span>

// ✅ Badge 永远 rounded-full
<Badge variant="outline">企业版</Badge>

// ✅ 需要方角胶囊（用户自建标签 / 卡片左上角彩色标签）→ Tag
<Tag color="blue">用户自建</Tag>
<Tag color="purple">企业技能</Tag>
```

### 14.4 不要给 Badge 加 hover / focus 反馈伪装可点

```tsx
// ❌ 给 Badge 加按钮样式，让用户误以为可点击
<Badge
  variant="outline"
  className="cursor-pointer hover:border-[var(--cp-brand-blue)]"
  onClick={handleFilter}
>
  企业版
</Badge>

// ✅ Badge 是非交互；需要点击 / 筛选 → 用 Tag（带可关闭 / 可选状态）
<Tag selected={selected} onClick={handleFilter}>企业版</Tag>

// ✅ 真的要包链接，用 asChild
<Badge asChild variant="outline">
  <a href="/changelog/v2.1.0">v2.1.0</a>
</Badge>
```

### 14.5 数字角标不要复用 Badge

```tsx
// ❌ 用 Badge 当作数字红点 / 通知气泡
<Badge variant="destructive">99+</Badge>
<Badge color="red" className="absolute -top-1 -right-1">3</Badge>

// ✅ 数字角标用宿主 NumberIndicator / 自定义 dot，不挪用 Badge 容器形态
<span className="absolute -top-1 -right-1 inline-flex min-w-[16px] h-4 items-center justify-center rounded-full bg-[var(--text-danger)] px-1 text-[10px] font-medium text-white">
  99+
</span>
```
