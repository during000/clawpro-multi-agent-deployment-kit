# StatusTag

> **用户/Agent 在描述"状态"时，唯一对应组件就是 StatusTag**——运行中 / 已停止 / 异常 / 待审核 / 即将到期 / 已完成 / 在线 / 失败 等所有"运行/信息/进度"语义统一走它。
> 与 `badge.md` 严格分工：**有"状态"语义就用 StatusTag；普通"标签 / 版本 / 范围 / new" 走 Badge**。

## 1. Purpose

- 统一表格状态列、行内状态描述、信息状态短文本三类高频场景的视觉与语义。
- 锁住状态色板（5 主语义），禁止业务侧自拼 `text-emerald-500 font-medium` / `bg-green-100 text-green-700`。
- **推荐默认形态**：彩色纯文本（无底色 / 无边框 / 无圆角）——密度最低、可读性最强、与表格行高最契合，表格状态列首选。
- **以组件实现为准**：StatusTag 实为多形态组件（`text` / `fill` / `soft` / `role`），其中 `fill`·`role` 消费 `rounded-full`、`soft` 消费 `rounded-[4px]`、`text` 无圆角（完整形态见 §3.4）。规范**不否认**这些形态，只就"何时用哪种"给出建议。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景（凡是用户语义里出现"状态"二字都走 StatusTag）：
  - 表格运行状态列（运行中 / 已停止 / 异常 / 待审核）
  - 行内状态文字（"已接入" / "未启用" / "即将到期"）
  - 任务 / 流程 / 实例的进度态（"进行中" / "已完成" / "失败"）
- 不适用场景：
  - 普通分类 / 版本 / 范围 / new / Beta / 企业版 / 公共 → `badge.md`（用户说"标签"的全部走 Badge）
  - 角色身份（管理员 / 用户）→ 业务侧改用 `<RoleTag>` 或宿主 RolePreset，不再由 StatusTag 承担
  - 用户自建可关闭 Tag → `tag-label.md`
  - 主操作按钮 / 链接（StatusTag 是非交互元素，**没有 hover / focus / active**）

## 3. Visual Standard

### 3.1 推荐形态：mode="text"（表格状态列首选）

> StatusTag 是多形态组件，下表是表格场景**推荐**的 text 形态；组件还提供 `fill` / `soft` / `role` 形态且**消费圆角**，完整对照见 §3.4。**以组件实现为准，不再声称"唯一形态"。**

| 项 | 值 |
|---|---|
| 形态 | 彩色纯文本，无底色 / 无边框 / 无圆点 |
| 字号 | 组件 `text` 本体为 **`text-sm` / 14px / Medium**；**表格状态列由应用层加 `className="text-xs"` 压到 12px**，与 12px 数据列对齐（组件不改，见 §3.5） |
| 行高 | 1.5（约 18px 可视高度，跟随父行高） |
| Padding | 0 |
| 圆角 | —（text 形态无圆角；fill / soft / role 见 §3.4） |
| Icon（可选） | 12×12，currentColor，与文字 `gap-1`（4px） |
| 换行 | `whitespace-nowrap` |

> **组件支持的其它形态（以组件为准，均消费圆角）**：
> - `mode="fill"`（组件默认 mode）：彩色实底胶囊，`rounded-full`。
> - `mode="soft"`：浅底 + 边框胶囊，`rounded-[4px]`。
> - `mode="role"` / `preset="role-*"`：角色胶囊，`rounded-full`。
> - `mode="dot"`：已废弃，组件内部 fallback 到 `text`（DEV 告警）。
>
> 表格状态列**推荐**用 text 形态保持低密度；信息态彩色胶囊 / 卡片彩色分类可用 `fill`/`soft`，或按"标签"语义改用 Badge。详见 §3.4。

### 3.2 主语义色板（5 色，不可扩展）

| Variant | 语义 | text 色值 | 典型词 |
|---|---|---|---|
| `blue` | 信息 / 进行中 / 处理中 | `#1447E6` | 进行中、处理中、全部用户、已接入 |
| `green` | 成功 / 已完成 / 在线 / 健康 | `#008236` | 运行中、已完成、在线、健康、已发布 |
| `red` | 失败 / 错误 / 危险 / 离线 | `#DC2626` | 异常、失败、离线、已禁用、严重 |
| `orange` | 警告 / 即将到期 / 需关注（amber 系） | `#B45309` | 即将到期、需关注、待审核、低配额 |
| `gray` | 中性 / 默认 / 未启用 / 未开始 | `#0A0A0A` | 未启用、未开始、草稿、默认 |

> `orange` 走 amber 系（`#B45309`），**禁止使用 `text-orange-*`**。
> 5 色之外一律不允许业务侧扩展；找不到合适色 → 退回 `gray`。

### 3.3 形态 / 语义取舍建议（组件均支持，按语义选择）

> 以下不是"已删除"——组件均可渲染，这里给的是**语义取舍建议**（状态 vs 标签）：

| 形态 / 旧 API | 行为 | 建议 |
|---|---|---|
| `mode="fill"` | 彩色实底胶囊 | 信息态胶囊可直接用；纯"标签"语义 → `Badge color="*"` |
| `mode="soft"` | 浅底+边框胶囊 | 卡片彩色分类可直接用；纯"标签"语义 → `Badge color="*"` |
| `mode="role"` / `preset="role-*"` | 角色身份 | 可直接用，或业务侧 `<RoleTag>` / 宿主 RolePreset |
| `mode="dot"` / `dot` 属性 | 状态圆点（已废弃） | 组件 fallback 到 `text`，新代码用 `mode="text"` |
| 扩展色（slate/zinc/violet/...） | soft 卡片分类色 | 状态语义退回 5 主色；分类场景可用扩展色或宿主 Tag |

### 3.4 组件真实形态与圆角（以组件实现为准）

> 源：`client/src/components/ui/status-tag.tsx`。**StatusTag 消费圆角**，圆角值随 mode 不同：

| mode | 形态 | 高度 | 圆角（实测 className） | 背景 / 边框 |
|---|---|---|---|---|
| `text` | 彩色纯文字 | 跟随行高 | **无** | 无 |
| `fill`（**组件默认 mode**） | 彩色实底胶囊 | `h-5`（20px） | **`rounded-full`** | `color.bg` |
| `soft` | 浅底 + 边框胶囊 | `h-5` | **`rounded-[4px]`** | `color.bg` + `color.border` |
| `role` / `preset="role-*"` | 角色胶囊 | `h-[22px]` | **`rounded-full`** | 白底 + 灰边 |

- ⚠️ **默认形态注意**：未显式传 `mode` 且未传 `dot` 时，组件默认 mode = **`fill`**（彩色胶囊），并非 text。表格若要纯文字状态列，需显式 `mode="text"`。
- 本表与 `radius-shadow.md §1.1` / `foundation.md §6`「StatusTag 消费圆角（fill·role=full、soft=4px、text 无）」口径一致。

### 3.5 字号：组件 14px，表格应用层压到 12px（不改组件）

> ⚠️ 关键区分，**别和"形态 className 覆盖"混为一谈**：
> - **组件本体**：`text` 形态字号是 `text-sm`（14px），**保持不动，不改组件**。
> - **表格 / 密集场景应用层**：在使用处加 `className="text-xs"`（12px），让状态文字与同行 12px 数据列对齐——这是**被允许、且为表格标准做法**的字号覆盖（实例：`client/src/pages/admin/MemberManagement/NodeContentPanel.tsx`）。
> - **仍然禁止**的是用 className 覆盖**形态**（`bg-*` / `border` / `rounded-*`）——那属于伪造组件形态（见 §11 / §14.5）。
>
> 一句话：**字号 `text-xs` 覆盖 = 允许（表格密度对齐）；形态 `bg/border/rounded` 覆盖 = 禁止。**

## 4. Anatomy

```text
StatusTag (span, role=presentation)
├── data-slot="status-tag"
├── data-variant="blue|green|red|orange|gray"
│
├── [optional] Icon slot   ← 12×12，currentColor，shrink-0
└── Label                  ← currentColor（组件 14px；表格应用 text-xs=12px）
```

- `inline-flex items-center gap-1`：图标与文字 4px 间距。
- `whitespace-nowrap`：状态文本不换行（截断由父级 `max-w + truncate` 负责）。

## 5. States

| State | 视觉 |
|---|---|
| default | 彩色文本 / Medium（组件 14px；表格用 `text-xs`=12px） |
| with-icon | 12×12 currentColor 图标 + `gap-1` |
| 长文本截断 | 由父级 `max-w + truncate` 处理，组件自身不裁 |
| 低优先级 | 用 `gray` variant，不用透明度 |

> StatusTag 不响应 hover / focus / active，也没有 disabled。它是非交互的语义标签。

## 6. Interaction

无。如需点击 / 跳转，套外层 `<button>` / `<a>`，不要给 StatusTag 加 onClick / 链接样式。

## 7. Accessibility

- 颜色不是唯一信息载体：必须配合文字（"运行中" / "异常"），不允许只用色块。
- 装饰类 icon 加 `aria-hidden="true"`。
- 不加 `role="status"`：避免读屏当作动态区域反复朗读。
- `data-variant` 暴露给自动化测试，避免颜色断言。

## 8. Demo Repo Usage

- 当前组件：`client/src/components/ui/status-tag.tsx`
- 典型页面：
  - `client/src/pages/admin/AuditLog.tsx`（运行状态列）
  - `client/src/pages/admin/DocManagement.tsx`（文档发布状态）
  - `client/src/pages/DesignSystemComponents.tsx`（demo / showcase）

```tsx
import { StatusTag } from "@/components/ui/status-tag";

// 表格运行状态：显式 mode="text" + className="text-xs"（12px，对齐数据列）
<StatusTag mode="text" variant="green" className="text-xs">运行中</StatusTag>
<StatusTag mode="text" variant="red" className="text-xs">异常</StatusTag>
<StatusTag mode="text" variant="orange" className="text-xs">即将到期</StatusTag>
<StatusTag mode="text" variant="gray" className="text-xs">未启用</StatusTag>
<StatusTag mode="text" variant="blue" className="text-xs">进行中</StatusTag>

// 信息态彩色胶囊（组件支持，消费圆角）
<StatusTag mode="fill" variant="blue">全部用户</StatusTag>   {/* rounded-full */}
<StatusTag mode="soft" variant="green">已接入</StatusTag>    {/* rounded-[4px] */}

// 带图标（表格内同样加 text-xs）
<StatusTag mode="text" variant="green" icon={<Check />} className="text-xs">已接入</StatusTag>
<StatusTag mode="text" variant="red" icon={<AlertCircle />} className="text-xs">失败</StatusTag>
```

> ⚠️ ① 未显式传 `mode` 时组件默认渲染 `fill`（彩色胶囊，`rounded-full`）；表格要纯文字状态列须显式 `mode="text"`（见 §3.4）。
> ② 组件 `text` 形态本体是 14px；表格里要 12px（对齐数据列）须加 `className="text-xs"`——这是允许的字号覆盖，组件不改（见 §3.5）。

## 9. Portable Fallback

### 9.1 If host repo already has Status component

- 必须先区分"状态语义"（StatusTag）与"普通分类标签"（Badge），**不允许一个组件混演**。
- 状态色统一登记到 token 层（5 色 hex 锁死），不放任 demo 仓自由扩色。
- 任何 `mode="fill" / "soft"` 的旧用法在 portable 化时必须被改写为 Badge 或保留为 StatusTag text。

### 9.2 Minimal React fallback

```tsx
// portable/react/status-tag.tsx
const STATUS_COLORS = {
  blue:   "#1447E6",
  green:  "#008236",
  red:    "#DC2626",
  orange: "#B45309",
  gray:   "#0A0A0A",
} as const;

type Variant = keyof typeof STATUS_COLORS;

export function PortableStatusTag({
  variant = "gray",
  icon,
  children,
}: {
  variant?: Variant;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <span
      data-slot="status-tag"
      data-variant={variant}
      className="inline-flex items-center gap-1 whitespace-nowrap text-xs font-medium leading-[1.5]"
      style={{ color: STATUS_COLORS[variant] }}
    >
      {icon && (
        <span aria-hidden="true" className="inline-flex size-3 shrink-0 items-center justify-center">
          {icon}
        </span>
      )}
      <span>{children}</span>
    </span>
  );
}
```

### 9.3 Minimal HTML/CSS fallback

```html
<span class="cp-status-tag cp-variant-green">运行中</span>
<span class="cp-status-tag cp-variant-red">异常</span>
<span class="cp-status-tag cp-variant-orange">即将到期</span>
<span class="cp-status-tag cp-variant-gray">未启用</span>
<span class="cp-status-tag cp-variant-blue">进行中</span>
```

```css
.cp-status-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.5;
}
.cp-variant-blue   { color: #1447E6; }
.cp-variant-green  { color: #008236; }
.cp-variant-red    { color: #DC2626; }
.cp-variant-orange { color: #B45309; }
.cp-variant-gray   { color: #0A0A0A; }
```

## 10. Migration Rules

| 旧写法 | 迁移目标 |
|---|---|
| `<StatusTag mode="fill" variant="*">...</StatusTag>` | → 改 `<Badge color="blue|green|purple|red">` 或保留为 `<StatusTag variant="*">`（如果语义是状态） |
| `<StatusTag mode="soft" variant="*">...</StatusTag>` | → `<Badge color="*">` 或宿主 Tag |
| `<StatusTag mode="dot">` / `<StatusTag dot>` | → `<StatusTag variant="*">`（直接去掉 dot） |
| `<StatusTag preset="role-admin" />` | → 业务侧 `<RoleTag role="admin" />` 或宿主 RolePreset |
| 表格状态列 14px 字号 | → 加 `className="text-xs"` 压到 **12px**（与数据列对齐；组件 `text` 本体 14px 不改，见 §3.5） |
| 自拼 `text-emerald-500` / `bg-green-100 text-green-700` 状态文字 | → `<StatusTag variant="green">` |
| 5 主色之外的 amber/sky/fuchsia 状态色 | → 退回到 5 主色，找不到对应就用 `gray` |

## 11. Do / Don't

Do:

- 表格状态列用 `<StatusTag mode="text" variant="*" className="text-xs">`，整列只有 12px 彩色文字、没有色块密集排列。
- 用户对话里出现"状态" / "运行" / "进度" / "在线" / "失败" / "已完成" → 一律 StatusTag。
- 找不到合适的色就退回 `gray`，不要新增色。

Don't:

- 不要表达 v2.1.0 / 企业版 / 公共 / new / Beta（→ Badge）。
- 不要传已废弃的 `mode="dot"` / `dot`（会 fallback 到 text）；`mode="text|fill|soft"` / `preset` 是组件有效 API（见 §3.4）。
- 表格状态列推荐 `mode="text"`，并加 `className="text-xs"` 压到 12px 对齐数据列——**字号覆盖是允许的表格做法**（组件 `text` 本体 14px，不改组件，见 §3.5）。
- 但**不要**用 `className`（`bg-*` / `rounded-*` / `border`）伪造或覆盖 StatusTag 的**形态**——圆角/底色由组件 `mode` 决定（text 无 / fill·role=full / soft=4px，见 §3.4）；纯"标签"语义走 Badge。
- 不要包成 button / a 让用户误以为可点。
- 不要用透明度模拟 disabled。

## 12. QA Checklist

- [ ] 表格状态列加了 `className="text-xs"`（12px，对齐数据列）；组件 `text` 本体 14px 属正常，**不要求改组件**（见 §3.5）
- [ ] 没有残留已废弃的 `mode="dot"` / `dot`（fill/soft/role 是有效形态，见 §3.4）
- [ ] 圆角/底色来自组件 mode（text 无圆角 / fill·role=full / soft=4px），未手写 className 覆盖
- [ ] 5 主色没有被业务侧扩到 6 / 7 / 16 色
- [ ] 角色身份不再走 StatusTag（已迁到 RoleTag 或 RolePreset）
- [ ] 没有 hover / focus / 链接样式
- [ ] 宿主仓 fallback 可执行（5 色 hex 已锁定）

## 13. References

- Demo code: `client/src/components/ui/status-tag.tsx`
- Demo page: `client/src/pages/DesignSystemComponents.tsx`
- 相关 spec: `component-specs/badge.md`、`component-specs/tag-label.md`、`component-specs/data-table.md`

## 14. 代码对照（✅/❌）

> 与 SKILL.md §2 / table.md §17.2 同口径。
>
> 注：以下 ✅/❌ 针对的是**表格场景推荐 text 形态**的强约束；组件实际支持 `fill`/`soft`/`role` 多形态且**消费圆角**（见 §3.4），按语义需要可使用，勿据此误判组件无圆角。

### 14.1 表格内运行状态：mode="text" + text-xs（12px 彩色文字）

```tsx
// ❌ 每行都用浅底色胶囊，整列颜色块密集
<TableCell>
  <span className="px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-xs">运行中</span>
</TableCell>

// ❌ 自拼 14px 彩色文字
<TableCell>
  <span className="text-sm font-medium text-[#008236]">运行中</span>
</TableCell>

// ⚠️ 只写 variant：组件会默认渲染 fill 胶囊、且字号 14px，并非表格想要的 12px 纯文字
<TableCell>
  <StatusTag variant="green">运行中</StatusTag>
</TableCell>

// ✅ 表格状态列：mode="text" + className="text-xs"（12px，对齐数据列；组件不改，见 §3.5）
<TableCell>
  <StatusTag mode="text" variant="green" className="text-xs">运行中</StatusTag>
</TableCell>
```

### 14.2 状态色不要业务侧自拼

```tsx
// ❌ 自拼 emerald / sky / fuchsia 表达自定义状态
<span className="text-emerald-500 font-medium">健康</span>
<span className="px-2 rounded-full bg-fuchsia-100 text-fuchsia-700">实验中</span>

// ❌ 使用 #00C853 / #FF5722 等非品牌色
<span className="text-[#00C853]">在线</span>

// ✅ 走 5 主色 variant
<StatusTag variant="green">健康</StatusTag>
<StatusTag variant="orange">实验中</StatusTag>
<StatusTag variant="green">在线</StatusTag>
```

### 14.3 状态 vs 标签：StatusTag 不承担版本 / 范围 / 分类

```tsx
// ❌ 用 StatusTag 表达版本号 / 范围 / 类别
<StatusTag variant="gray">v2.1.0</StatusTag>
<StatusTag variant="blue">企业版</StatusTag>
<StatusTag variant="green">公共</StatusTag>

// ✅ 状态 → StatusTag
<StatusTag variant="green">运行中</StatusTag>
<StatusTag variant="red">异常</StatusTag>

// ✅ 分类 / 标签 / 版本 / 范围 → Badge（用户说"标签"全部走 Badge）
<Badge variant="outline">v2.1.0</Badge>
<Badge variant="secondary">企业版</Badge>
<Badge color="blue">公共</Badge>
```

### 14.4 mode / preset 是有效 API；仅 `mode="dot"` 已废弃

```tsx
// ❌ 已废弃：mode="dot" / dot（组件会 fallback 到 text，DEV 告警）
<StatusTag mode="dot" variant="green">运行中</StatusTag>
<StatusTag dot variant="red">异常</StatusTag>

// ✅ 表格状态列：推荐 text 形态（纯彩色文字）
<StatusTag mode="text" variant="green">运行中</StatusTag>
<StatusTag mode="text" variant="red">异常</StatusTag>

// ✅ 信息态彩色胶囊 / 卡片分类：组件 fill/soft 形态（消费圆角），或按"标签"语义改 Badge
<StatusTag mode="fill" variant="blue">全部用户</StatusTag>
<StatusTag mode="soft" variant="violet">企业技能</StatusTag>
<Badge color="blue">全部用户</Badge>

// ✅ 角色身份：组件 preset，或业务侧 RoleTag / 宿主 RolePreset
<StatusTag preset="role-admin" />
<RoleTag role="admin" />
```

### 14.5 用组件 mode 控制形态，不要手写 className 覆盖

```tsx
// ❌ 手写 className 拼 bg/border/rounded 覆盖组件形态
<StatusTag variant="green" className="bg-green-50 px-2 rounded-full">运行中</StatusTag>
<StatusTag variant="red" className="border border-red-200">失败</StatusTag>

// ✅ 想要彩色胶囊就用组件形态（圆角随 mode：fill=full / soft=4px）
<StatusTag mode="soft" variant="green">已发布</StatusTag>
<Badge color="green">已发布</Badge>

// ✅ 真正想要"圆点 + 文字"的语义 → 套外层圆点
<span className="inline-flex items-center gap-1.5">
  <span className="size-1.5 rounded-full bg-[#008236]" />
  <StatusTag mode="text" variant="green">运行中</StatusTag>
</span>
```
