# Foundation

> 全局 token / 字体 / 色彩 / 圆角 / 阴影唯一源。分端文件只能声明差异，不能重复改写基础事实。

## 1. 设计语言

- 设计语言：流动蓝图（Fluid Blueprint）
- 产品类型：ClawPro 是产品型 UI，设计服务工作流效率与可读性，不追求装饰优先。
- 技术栈：React 19、TypeScript、Vite、Tailwind CSS v4、shadcn/ui、wouter、lucide-react、recharts、sonner、framer-motion。

## 2. 品牌色与语义色

| Token | 值 | 用途 |
|---|---:|---|
| Brand Blue | `#1447E6` | 主色、活跃态、链接、主按钮、Switch 开启 |
| Brand Black | `#020617` | CTA 渐变起点、强调文字 |
| Brand Blue Tint | `#EFF6FF` | 活跃菜单项底色、弱强调背景 |
| Success | `#16A34A` | 运行中 / 成功 |
| Danger | `#DC2626` | 停用 / 删除 / 错误 |
| Warning | `#F59E0B` | 待处理 / 警告 |
| Border | `#EAEEF4` | 蓝灰描边；卡片、表格、分割线、面板、Input / Select / DatePicker 默认描边 |
| Border Control | `#C8CFDA` | Checkbox / Radio 等可勾选控件默认描边 |

主 CTA 渐变：

```css
linear-gradient(90deg, #020617 70%, #1447E6 100%)
```

品牌主色统一使用 `Brand Blue`、`Brand Black` 和当前 CTA 渐变，不再引入其他主品牌色方案。

## 3. 背景

| 区域 | 值 |
|---|---|
| 用户端通用背景 | 白底 + 极淡蓝雾（详见 `tenant.md` §2 背景） |
| 管控端通用背景 | 左侧导航 `ADMIN_NAV_GROUPS` 到达页统一使用 `AdminSidebarInset` 的浅蓝系渐变底图：`#FFFFFF` 兜底 + `url(/admin_content_bg.png)`，`cover` / `center top` / `no-repeat` / `fixed` |
| 卡片 / 面板 | `#FFFFFF` |
| 表格表头 / 斑马纹 | `bg-gray-50/50` |
| Segment 容器 | `#F5F5F5` |

## 4. 文字层级

当前文字色以运行时 `client/src/index.css` 与 `client/src/components/ui/Typography.tsx` 的 `--text-*` 蓝灰 / slate 语义 token 为准。

| Token | 色值 | 用途 |
|---|---:|---|
| `--text-emphasis` | `#020617` | 强强调、关键数字、按钮文字、强标题 |
| `--text-title` | `#0F172A` | 页面标题、模块标题、卡片标题 |
| `--text-body` | `#1E293B` | 普通正文、表格主内容 |
| `--text-secondary` | `#334155` | 描述、补充说明、表格次要字段 |
| `--text-muted` | `#64748B` | 时间、备注、辅助信息、表头 |
| `--text-weak` | `#94A3B8` | 占位符、空状态、禁用提示、HelperText |
| `--text-brand` | `#1447E6` | 链接、选中态、品牌强调 |
| `--text-danger` | `#DC2626` | 删除、错误、危险操作 |

新增页面优先使用 Typography 语义层或映射到上述 token，不在页面中散落定义基础文字色。

## 5. 字体

```css
--font-sans: 'PingFang SC', -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif;
--font-mono: 'Menlo', 'Consolas', 'Courier New', monospace;
--font-din: 'DIN Alternate', 'DIN', 'Helvetica Neue', sans-serif;
--font-en: 'Open Sans', 'Helvetica Neue', sans-serif;
```

- 中文 UI 默认 PingFang SC。
- 数字使用 `tabular-nums`，统计数字优先 `font-din` 或 `StatNumber`。
- 代码、Token、路径、实例 ID 使用 Menlo 或 `CodeText`。
- 不新增 inline `fontFamily`，除非组件规范已声明例外。

## 6. 圆角

全局 v2 基线：`2px / 3px / 4px / full`。下表为**设计语义 token**；**写代码（className/style）必须用运行时 CSS 变量**，二者数值错位，完整对照与红线见 `tokens/radius-shadow.md §1.1`。

| 设计语义 Token | 值 | 运行时 CSS 变量 | 用途 |
|---|---:|---|---|
| `radius-xs` | 2px | `var(--radius-sm)` | 小 Badge、小状态块 |
| `radius-sm` | 3px | `var(--radius-md)` | Segment active 滑块 |
| `radius-md` | 4px | `var(--radius-lg)` / `var(--radius-xl)` | 管控端按钮、输入框、卡片、Dialog、Popover |
| `radius-full` | 9999px | `rounded-full` | 头像、状态点、胶囊标签、Switch |

> 🔒 2px 必须写 `var(--radius-sm)`（运行时无 `--radius-xs`）；4px 写 `var(--radius-lg)`，**勿**改成 `var(--radius-md)`（运行时=3px）。
> 注：StatusTag 真实组件 `mode="soft"` 消费 `rounded-[4px]`、`fill`/`preset` 消费 `rounded-full`，仅 `text` 模式无圆角（以 `client/src/components/ui/status-tag.tsx` 为准）。

注意：Tenant 业务卡片圆角当前按 `tenant.md` 的差异规则处理；如与全局 4px 基线冲突，先查 `conflict-log.md`。

## 7. 阴影与 Surface 层级

| 层级 | 组件 / Token | 用途 |
|---|---|---|
| L1 | `SurfaceCard` | Admin / Shared 页面主卡、列表卡、统计卡；默认无阴影，仅保留蓝灰描边 |
| L1 interactive | `SurfaceCard hover` / selected | Admin / Shared 可交互或选中态；使用极轻阴影和描边变化 |
| L2 | `SurfaceInner` / `--shadow-inner` | 卡内表格、内嵌面板；无阴影 |
| L3 | `SurfaceOverlay` / `--shadow-overlay` | Dialog、Sheet、Popover、Dropdown |
| L4 | `SurfaceConfig` / `--shadow-config` | 管控端高亮配置卡 / 引导卡；默认无阴影，确需强调时显式加 |
| L5 | `--shadow-segment` | Segment / Tab active 滑块 |
| L6 | `TenantCard` / `--shadow-tenant-card` | Tenant 业务卡；normal 默认有轻投影，hover 增强，static 无阴影 |

禁止在业务页面新增：

- 自定义重阴影替代 Surface 层级
- 未带 `allow-shadow` 说明的 inline `boxShadow`
- 手写 `bg-white rounded... border...` 冒充卡片层级

## 8. 动效

- 页面根：保留 `page-enter`。
- 常规 hover：`transition-all duration-150` 或 `transition-colors`。
- 弹窗：使用 shadcn/Radix 默认动效或 `animate-in fade-in-0 zoom-in-95 duration-200`。
- 所有复杂动画必须考虑 `prefers-reduced-motion`。

## 10. 设计检查口径

- 先判断端：Admin / Tenant / Landing。
- 再判断层级：页面结构 / 业务组件 / 基础组件 / 资产。
- 最后判断冲突：有冲突时记录，不用个人偏好覆盖既有规范。
