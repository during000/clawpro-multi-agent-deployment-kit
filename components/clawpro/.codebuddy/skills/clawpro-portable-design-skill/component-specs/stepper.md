# Stepper（步骤条）

## 1. Purpose

- 统一多步流程（向导 / 引导 / 分步表单）的步骤指示：当前在第几步、哪些已完成、哪些待办。
- 收敛各页面手写「圆圈 + 序号 + 连接线」的视觉差异（圆圈尺寸、连接符、完成态图标、文字态）。

## 2. Scope

- 适用端：Admin 优先（数据接入向导、配置向导、分步创建流程）。Tenant 仅在确有分步流程时复用。
- 必用场景：≥3 步、步骤线性推进、需要让用户感知整体进度与当前位置（如「选择数据源 → 添加凭证 → 字段映射 → 完成」）。
- 不适用场景：单步表单（无需步骤条）；非线性 / 可任意跳转的分区（用 `tabs.md`）；仅两态切换（用 `segment.md`）。

## 3. Visual Standard

> 真实实现：`client/src/components/ui/stepper.tsx`（`Stepper` 组件）。下表为该组件实际渲染口径。

### 3.1 整体

| Item | Value | Notes |
|---|---|---|
| 容器 | `flex items-center gap-2 flex-wrap`，`role="list"` | 横向排列，过窄时换行 |
| 步骤间分隔 | `ChevronRight` `w-4 h-4`（16px），灰色 | 步骤之间插入，最后一步后不插 |
| 步骤单元 | `flex items-center gap-2`，激活步 `aria-current="step"` | 圆圈 + 标题成组 |

### 3.2 步骤圆圈（序号）

| 状态 | 背景 / 文字 | 内容 |
|---|---|---|
| completed（序号 < current） | 实心品牌蓝圆圈 + 白字 | `Check` 图标 `w-3.5 h-3.5` |
| active（序号 === current） | 实心品牌蓝圆圈 + 白字 | 序号数字 |
| pending（序号 > current） | 浅灰底 + 弱灰字 | 序号数字 |

- 圆圈尺寸固定 `w-6 h-6`（24px）、`rounded-full`、`text-xs font-medium tabular-nums`（序号等宽防抖）。

### 3.3 步骤标题

| 状态 | 字号 / 字重 / 色 |
|---|---|
| active | `text-sm` / `font-medium` / 主标题色（最深） |
| completed | `text-sm` / normal / 次级灰 |
| pending | `text-sm` / normal / 弱灰 |

### 3.4 ⚠️ 实现现状 / 对齐缺口（如实记录，勿据此改组件）

> `stepper.tsx` 当前**硬编码**了下列色值，与本 skill 的 token 口径存在偏差，**列为已知对齐项**（与 M4-4「实现侧去硬编码」同批处理），本 spec 不改组件代码：
>
> | 元素 | 组件现状 | 本 skill 目标 token |
> |---|---|---|
> | 完成 / 激活圆圈底 | `bg-blue-500`（#3B82F6） | 品牌蓝 `var(--cp-brand-blue)`（#1447E6） |
> | pending 圆圈底 / 字 | `bg-gray-100` / `text-gray-400` | `var(--cp-bg-subtle)` / `var(--cp-text-weak)` |
> | active 标题 | `text-gray-950` | `var(--cp-text-title)` |
> | completed 标题 | `text-gray-500` | `var(--cp-text-muted)` |
> | pending 标题 / 分隔箭头 | `text-gray-400` | `var(--cp-text-weak)` |
>
> 新建页面**直接引用 `Stepper` 组件即可**（视觉已可用）；当组件迁移到 token 时，按上表对齐，不要在调用处用 className 私自覆盖颜色。

## 4. Anatomy

```text
Stepper (role=list)
  Step (repeatable)
    Circle (24px)
      ├ completed → 品牌蓝底 + Check
      ├ active    → 品牌蓝底 + 序号
      └ pending   → 浅灰底 + 序号
    Label (text-sm，状态决定字重/色)
  ChevronRight (16px，步骤之间)
  Step ...
```

## 5. API

```ts
export interface StepperItem {
  /** 步骤标题 */
  label: string;
}

export interface StepperProps {
  /** 当前激活步骤（1-based） */
  current: number;
  /** 步骤列表 */
  steps: StepperItem[];
  /** 自定义容器类名 */
  className?: string;
}
```

```tsx
import { Stepper } from "@/components/ui/stepper";

<Stepper
  current={2}
  steps={[
    { label: "选择数据源方式" },
    { label: "添加应用凭证" },
    { label: "设置字段映射" },
    { label: "完成" },
  ]}
/>
```

- 状态由 `current` 与各步索引自动推导（`序号 < current` → completed，`=== current` → active，`> current` → pending），调用方**不需手动传状态**。

## 6. Portable Fallback

> 宿主仓无该组件时的最小兜底（已按本 spec 的 token 目标口径书写，圆圈用品牌蓝）：

```tsx
function PortableStepper({ current, steps }: { current: number; steps: { label: string }[] }) {
  return (
    <div className="flex items-center gap-2 flex-wrap" role="list">
      {steps.map((step, idx) => {
        const stepNum = idx + 1;
        const status = stepNum < current ? "completed" : stepNum === current ? "active" : "pending";
        return (
          <React.Fragment key={idx}>
            <div className="flex items-center gap-2" role="listitem" aria-current={status === "active" ? "step" : undefined}>
              <span
                className={[
                  "w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium tabular-nums shrink-0",
                  status === "pending"
                    ? "bg-[var(--cp-bg-subtle)] text-[var(--cp-text-weak)]"
                    : "bg-[var(--cp-brand-blue)] text-white",
                ].join(" ")}
              >
                {status === "completed" ? "✓" : stepNum}
              </span>
              <span
                className={[
                  "text-sm",
                  status === "active" && "font-medium text-[var(--cp-text-title)]",
                  status === "completed" && "text-[var(--cp-text-muted)]",
                  status === "pending" && "text-[var(--cp-text-weak)]",
                ].filter(Boolean).join(" ")}
              >
                {step.label}
              </span>
            </div>
            {idx < steps.length - 1 && (
              <span className="w-4 h-4 text-[var(--cp-text-weak)] shrink-0" aria-hidden>›</span>
            )}
          </React.Fragment>
        );
      })}
    </div>
  );
}
```

## 7. Migration Rules

| 旧写法 | 新写法 |
|---|---|
| 手写「圆圈 + 横线」连接 | 用 `Stepper`；步骤之间统一 `ChevronRight`，不画实线 |
| 圆圈尺寸各页不一 | 固定 `24px`（`w-6 h-6`） |
| 完成态只换文字色、不换图标 | 完成态圆圈内显示 `Check` 图标，区别于 active 的序号 |
| 序号字体非等宽导致跳动 | `tabular-nums` |

## 8. Do / Don't

**Do:**
- 直接引用 `Stepper` 组件，状态交给 `current` 自动推导。
- 步骤标题简短（动宾短语），避免折行。
- 完成态用 `Check` 图标、active 用序号，二者视觉可区分。

**Don't:**
- 不在调用处用 className 覆盖圆圈 / 文字颜色（颜色对齐缺口由组件侧统一，见 §3.4）。
- 不用实线 / 进度条替代 `ChevronRight` 分隔（与设计稿不一致）。
- 不把可任意跳转的导航做成 Stepper（那是 Tabs 的职责）。

## 9. QA Checklist

- [ ] 步骤数 ≥ 3 且线性推进才用 Stepper
- [ ] 圆圈 `24px` / `rounded-full` / 序号 `tabular-nums`
- [ ] completed = 品牌蓝底 + Check；active = 品牌蓝底 + 序号；pending = 浅灰底 + 序号
- [ ] 标题 `text-sm`，active 加粗最深、completed 次级、pending 最弱
- [ ] 步骤之间用 `ChevronRight`，末步后不加
- [ ] 调用处未私自覆盖颜色
- [ ] fallback 可独立落地

## 10. References

- 真实实现：`client/src/components/ui/stepper.tsx`
- Related specs：`tabs.md`（非线性分区）、`segment.md`（两态切换）、`form-controls.md`（分步表单内的控件）
- 色 token：`tokens/colors.md`
