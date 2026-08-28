# Radius & Shadow

## 1. 圆角基线

> ⚠️ **两套命名并存，务必区分**：下表是**设计语义 token**（讨论设计时用）；写代码（className / inline style）时必须用**运行时 CSS 变量**，两者**数值错位**，对照见 §1.1。

| 设计语义 Token | Value | Usage |
|---|---|---|
| `radius-xs` | `2px` | 小 badge、小状态块 |
| `radius-sm` | `3px` | Segment active 滑块 |
| `radius-md` | `4px` | Admin 按钮、输入框、普通卡片、Dialog |
| `radius-card-tenant` | `12px` | Tenant 业务卡片 |
| `radius-full` | `9999px` | 胶囊按钮、Tenant 切换器、状态点 |

### 1.1 设计语义 Token ↔ 运行时 CSS 变量对照（必读）

运行时 `index.css` / `design-tokens.json`（**生成物，禁手改**）的 CSS 变量名与上面的语义名**不是一一对应**，写代码时**只认 CSS 变量**：

| 设计语义 | 数值 | 运行时 CSS 变量（写代码用这个） |
|---|---|---|
| `radius-xs` | **2px** | **`var(--radius-sm)`** |
| `radius-sm` | **3px** | `var(--radius-md)` |
| `radius-md` | **4px** | `var(--radius-lg)` / `var(--radius-xl)` |
| `radius-card-tenant` | **12px** | `var(--radius-card)` |
| `radius-full` | 9999px | `rounded-full` |

> 🔒 **2px 数值红线（小 Badge / 状态徽章等）**：要 2px **必须写 `var(--radius-sm)`**。运行时**没有 `--radius-xs` 变量**，按语义名写 `var(--radius-xs)` 会渲染失败。
> 🔒 **4px 不要改名**：代码里 `var(--radius-lg)`=4px 是**正确**写法，**禁止**因「语义叫 radius-md」而改写成 `var(--radius-md)`（运行时该变量=3px，会渲染错）。
> StatusTag **确实消费圆角**（以真实组件 `client/src/components/ui/status-tag.tsx` 为准）：`mode="soft"` 用写死的 `rounded-[4px]`，`mode="fill"`/`preset` 用 `rounded-full`，仅 `mode="text"` 无圆角。2px 的真实消费者是小 Badge / 状态徽章。

## 2. 阴影层级

| Token / Component | Usage |
|---|---|
| `SurfaceCard` | Admin / Shared 页面主卡、列表卡、统计卡；默认无阴影，仅保留蓝灰描边 |
| `SurfaceCard hover` / selected | Admin / Shared 可交互或选中态；极轻阴影 + 描边变化 |
| `--shadow-inner` | 内嵌分组、卡内面板；无阴影 |
| `--shadow-overlay` | Dialog、Drawer、Popover、Dropdown |
| `--shadow-config` | Admin 强调配置块；默认不加，确需强调时显式使用 |
| `--shadow-segment` | Segment / Tab active |
| `--shadow-tenant-card` | Tenant 业务卡 normal / active；hover 态可增强，static 无阴影 |

## 3. 使用原则

- Admin / Shared 以 4px + 蓝灰描边为主基线，普通卡默认无投影。
- Admin 只有 hover / selected / overlay / 明确强调配置块才出现轻阴影。
- Tenant 业务卡以 12px + `--shadow-tenant-card` 为独立分流，不应与 Admin 业务卡混用。
- 宿主仓换皮时优先映射语义层级，不要延续大量手写阴影 class。

