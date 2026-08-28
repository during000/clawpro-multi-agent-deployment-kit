# Alert

## 1. Purpose

- 统一页面信息提示、操作说明提示、警告提示、产品动态通知、成功反馈、错误反馈的样式与结构。
- 替代业务页面自行拼装的 `bg-blue-50` / `bg-amber-50` / `border-*` / `rounded-*` 容器。
- 管控端顶部常驻公告条使用 `AdminNoticeAlert`（独立组件），不复用页面内 Alert。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：页面常驻说明、表单辅助提示、配置缺失警告、产品动态告知、操作上下文说明、操作成功反馈、操作失败反馈
- 不适用场景：需要自动消失的操作反馈（用 Toast）、需要用户确认的阻断提示（用 Dialog）

## 3. Source Files

| 文件 | 用途 |
|------|------|
| `client/src/components/ui/alert.tsx` | Alert 核心组件 |
| `client/src/components/ui/admin-notice-alert.tsx` | 管控端顶部公告条 |
| `client/src/pages/DesignSystemComponents.tsx` | 组件展示页示例 |
| `client/src/index.css` | Token 定义 |

## 4. Visual Standard

### 4.1 基础参数

| Item | Value |
|------|-------|
| 圆角 | `var(--alert-radius)` = `var(--radius)` = `4px` |
| 内边距 | `px-4 py-2.5`（左右 16px，上下 10px） |
| 图标尺寸 | `16px` |
| 图标列宽 | `16px` |
| 图标与文字间距 | `8px` |
| 图标垂直对齐 | `translate-y-px`（与 12px/18px 行高文字首行居中） |
| 标题与描述间距 | `gap-y-1`（4px） |
| AlertTitle 字体 | `MetaMedium`（12px / medium / 1.5） |
| AlertDescription 字体 | `MetaText`（12px / regular / 1.5 / `tone="inherit"`） |

### 4.2 Variants Token 表

#### Info（标准信息提示）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-info-bg` | `#F0F3FC` | 背景 |
| `--alert-info-border` | `#BFCFFE` | 描边 |
| `--alert-info-foreground` | `#0A0A0A` | 正文 |
| `--alert-info-icon` | `#1447E6` | 图标 |

#### Operation-Info（操作说明）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-operation-info-bg` | `#FFFFFF` | 背景 |
| `--alert-operation-info-border` | `#EAEEF4` | 描边 |
| `--alert-operation-info-foreground` | `var(--alert-info-foreground)` | 正文 |
| `--alert-operation-info-icon` | `var(--text-muted)` | 图标 |

#### Warning（标准警告）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-warning-bg` | `#FFF7ED` | 背景 |
| `--alert-warning-border` | `#FED7AA` | 描边 |
| `--alert-warning-foreground` | `#0A0A0A` | 正文 |
| `--alert-warning-icon` | `#FF6900` | 图标 |

#### Product-News（产品动态）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-product-news-bg` | `var(--alert-info-bg)` | 背景 |
| `--alert-product-news-border` | `var(--alert-info-border)` | 描边 |
| `--alert-product-news-foreground` | `var(--alert-info-foreground)` | 正文 |
| `--alert-product-news-icon` | `var(--alert-info-icon)` | 图标 |

#### Success（成功提示）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-success-bg` | `#ECFDF5` | 背景（淡翠绿） |
| `--alert-success-border` | `#A7F3D0` | 描边 |
| `--alert-success-foreground` | `#0A0A0A` | 正文 |
| `--alert-success-icon` | `#059669` | 图标 |

#### Error（错误提示）

| Token | Value | 用途 |
|-------|-------|------|
| `--alert-error-bg` | `#FEF2F2` | 背景（淡红） |
| `--alert-error-border` | `#FECACA` | 描边 |
| `--alert-error-foreground` | `#0A0A0A` | 正文 |
| `--alert-error-icon` | `#DC2626` | 图标 |

## 5. Usage Examples

### Info
```tsx
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";

<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示文案</AlertDescription>
</Alert>
```

### Operation-Info
```tsx
import { Alert, AlertDescription, AlertOperationInfoIcon, AlertTitle } from "@/components/ui/alert";

<Alert variant="operation-info">
  <AlertOperationInfoIcon />
  <AlertTitle>操作说明标题</AlertTitle>
  <AlertDescription>操作说明描述</AlertDescription>
</Alert>
```

### Warning
```tsx
import { CircleAlert } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

<Alert variant="warning">
  <CircleAlert />
  <AlertTitle>警告标题</AlertTitle>
  <AlertDescription>警告描述</AlertDescription>
</Alert>
```

### Product-News
```tsx
import { Alert, AlertDescription, AlertProductNewsIcon } from "@/components/ui/alert";

<Alert variant="product-news">
  <AlertProductNewsIcon />
  <AlertDescription>【产品动态】提示文案</AlertDescription>
</Alert>
```

### 带右侧操作区
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

### Success（成功提示）
```tsx
import { Alert, AlertDescription, AlertSuccessIcon } from "@/components/ui/alert";

<Alert variant="success">
  <AlertSuccessIcon />
  <AlertDescription>配置保存成功</AlertDescription>
</Alert>

<Alert variant="success">
  <AlertSuccessIcon />
  <AlertTitle>操作成功</AlertTitle>
  <AlertDescription>已成功创建 3 个 Agent</AlertDescription>
</Alert>
```

### Error（错误提示）
```tsx
import { Alert, AlertDescription, AlertErrorIcon } from "@/components/ui/alert";

<Alert variant="error">
  <AlertErrorIcon />
  <AlertDescription>保存失败，请稍后重试</AlertDescription>
</Alert>

<Alert variant="error">
  <AlertErrorIcon />
  <AlertTitle>请求失败</AlertTitle>
  <AlertDescription>网络异常，请检查网络连接后重试</AlertDescription>
</Alert>
```

## 6. AdminNoticeAlert（管控端顶部公告条）

用于管控端顶部常驻通知条，支持「产品动态」「待配置」「资源告警」三种类型。

```tsx
import { AdminNoticeAlert } from "@/components/ui/admin-notice-alert";

<AdminNoticeAlert type="product-news" controls={<span className="text-xs tabular-nums text-[var(--text-secondary)]">4/5</span>}>
  <span>OpenClaw v2.4.0 已发布：记忆管理功能上线。</span>
</AdminNoticeAlert>

<AdminNoticeAlert type="pending-config" controls={<span className="text-xs tabular-nums text-[var(--text-secondary)]">4/5</span>}>
  <span>有 3 项基础配置未完成（导入企业用户、配置至少一个通道、配置安全组），未完成配置将影响用户端的正常使用，</span>
  <span className="font-medium text-[var(--text-emphasis)] underline underline-offset-2">前往基础信息配置处理</span>
</AdminNoticeAlert>

<AdminNoticeAlert type="resource-alert" controls={<span className="text-xs tabular-nums text-[var(--text-secondary)]">4/5</span>}>
  <span>私有网络（VPC）配额已耗尽，将影响用户端云设备的正常创建与使用，</span>
  <span className="text-[var(--text-emphasis)] underline underline-offset-2">前往腾讯云控制台提交工单</span>
</AdminNoticeAlert>
```

### AdminNoticeAlert 视觉参数

| Item | Value |
|------|-------|
| 外层高度 | `40px` |
| 圆角 | `4px` |
| 布局 | `flex items-center gap-2.5 px-3` |
| 背景 | `rgba(255,255,255,0.75)` |
| 边框 | `1px solid #FFFFFF` |
| 正文字号 | `12px / 18px` |
| 左侧标签 | `22px` 高 / `gap-[5px]` / `px-[6px]` / `2px` 圆角 / `11px` 文案 |
| 控制区 | 通过 `controls` 自定义传入；展示页示例使用 `4/5` |

| Type | 标签文案 | 标签背景 | 描边 / 文字 / 图标 |
|------|----------|----------|--------------------|
| `product-news` | 产品动态 | `linear-gradient(139deg, #EFF3FF 18%, #F3F6FF 51%, #ECF1FF 100%)` | `#C6D4FF` / `#2547B1` / `sparkle` |
| `pending-config` | 待配置 | `linear-gradient(139deg, #FFF2E6 18%, #FFF9F4 51%, #FFF2E6 100%)` | `#F8DDC4` / `#D76610` / `alert`（`#EE7A23`） |
| `resource-alert` | 资源告警 | `linear-gradient(139deg, #FFF2E6 18%, #FFF9F4 51%, #FFF2E6 100%)` | `#F8DDC4` / `#D76610` / `alert`（`#EE7A23`） |

设计系统展示页会把多条 `AdminNoticeAlert` 放进一个额外的蓝色渐变容器：

```tsx
<div className="rounded-[4px] bg-[linear-gradient(180deg,#F7FAFF_0%,#EEF4FB_100%)] px-5 py-4">
  <div className="space-y-3">...</div>
</div>
```

这层渐变包装不是 `AdminNoticeAlert` 组件本体，而是展示页布局。

## 7. Variant 选择决策

| 场景 | 使用 |
|------|------|
| 页面说明 / 功能告知 | `variant="info"` |
| 白底灰边操作上下文说明 | `variant="operation-info"` |
| 配置缺失 / 风险提示 | `variant="warning"` |
| 产品发布 / 版本更新 | `variant="product-news"` |
| 操作成功 / 状态正常 | `variant="success"` |
| 操作失败 / 错误提示 | `variant="error"` |
| 管控端顶部常驻公告 | `AdminNoticeAlert` |
| 需要自动消失的操作反馈 | ❌ 不要用 Alert，用 Toast |

## 8. Prohibitions

- 禁止业务页面手写 info / operation-info / warning / product-news / success / error 提示条样式
- 禁止使用 warning/amber 样式承载普通信息提示（用 `info` 或 `operation-info`）
- 禁止硬编码 Alert 色值，必须通过 `--alert-*` token
- 禁止在 Alert 上使用 `rounded-lg` / `rounded-xl` / `shadow-*` / inline `boxShadow`
- Info 图标必须使用 `AlertInfoIcon`，不要直接引入 lucide `Info`
- Warning 图标必须使用 `CircleAlert`，禁止使用 `AlertTriangle` 作为标准横幅图标
- Success 图标必须使用 `AlertSuccessIcon`（CheckCircle2），失败/错误图标必须使用 `AlertErrorIcon`（XCircle）
- 管控端顶部公告条必须使用 `AdminNoticeAlert`，不要用普通 Alert 替代
- `AdminNoticeAlert` 本身不内置关闭按钮；如业务需要额外控制区，必须通过 `controls` 插槽组合

## 9. Accessibility

- Alert 容器需带 `role="alert"` 或 `role="status"`
- 图标需设置 `aria-hidden="true"`
- 确保色彩对比度 ≥ 4.5:1

## 10. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Alert 5 项高频误用 → ClawPro 正确写法。

### 10.1 不要业务页面手写 bg-blue-50 / bg-amber-50

```tsx
// ❌ 自己拼一个浅蓝提示条
<div className="rounded-lg bg-blue-50 border border-blue-200 px-4 py-3 flex items-start gap-2">
  <Info className="h-4 w-4 text-blue-500 mt-0.5" />
  <span className="text-sm text-blue-900">数据每 5 分钟更新一次</span>
</div>

// ❌ 自己拼一个琥珀色警告
<div className="rounded bg-amber-50 border-amber-200 border p-3">
  <AlertTriangle className="text-amber-600" />
  请先完成基础配置
</div>

// ✅ 用统一 Alert + variant
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>数据每 5 分钟更新一次</AlertDescription>
</Alert>

<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>请先完成基础配置</AlertDescription>
</Alert>
```

### 10.2 Variant 选择：信息别用 warning 当感叹号

```tsx
// ❌ 普通信息提示用了 warning，过度强调
<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>数据每 5 分钟更新一次（这只是信息）</AlertDescription>
</Alert>

// ❌ 操作上下文说明用了 info（蓝底太抢戏）
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>下方修改将影响所有 Agent</AlertDescription>
</Alert>

// ✅ 普通信息：info（浅蓝底）
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>数据每 5 分钟更新一次</AlertDescription>
</Alert>

// ✅ 操作上下文：operation-info（白底灰边，更克制）
<Alert variant="operation-info">
  <AlertOperationInfoIcon />
  <AlertTitle>修改提示</AlertTitle>
  <AlertDescription>下方修改将影响所有 Agent，请确认后再保存</AlertDescription>
</Alert>

// ✅ 真正的风险 / 配置缺失：warning（橙色）
<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>有 3 项基础配置未完成</AlertDescription>
</Alert>
```

### 10.3 图标用规范导出，不要直接 import lucide Info

```tsx
// ❌ 直接用 lucide Info / AlertTriangle，颜色 / 尺寸自己控
import { Info, AlertTriangle } from "lucide-react";

<Alert variant="info">
  <Info className="h-5 w-5 text-blue-500" />
  <AlertDescription>提示</AlertDescription>
</Alert>

<Alert variant="warning">
  <AlertTriangle className="h-4 w-4" />  {/* AlertTriangle 不是规范图标 */}
  <AlertDescription>警告</AlertDescription>
</Alert>

// ✅ Info 类用 AlertInfoIcon（自带 16px + 品牌蓝）
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示</AlertDescription>
</Alert>

// ✅ Warning 用 CircleAlert（lucide）作为标准
import { CircleAlert } from "lucide-react";
<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>警告</AlertDescription>
</Alert>
```

### 10.4 不要在 Alert 上加 rounded-lg / shadow / inline boxShadow

```tsx
// ❌ 自己加大圆角 + 阴影，与全局视觉漂移
<Alert variant="info" className="rounded-xl shadow-md">
  <AlertInfoIcon />
  <AlertDescription>提示</AlertDescription>
</Alert>

// ❌ 用 style 注入 boxShadow
<Alert variant="warning" style={{ boxShadow: "0 4px 12px rgba(0,0,0,0.08)" }}>
  <CircleAlert />
  <AlertDescription>警告</AlertDescription>
</Alert>

// ✅ 严守 4px 圆角无阴影
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示</AlertDescription>
</Alert>
```

### 10.5 管控端顶部公告必须用 AdminNoticeAlert

```tsx
// ❌ 顶部公告用普通 Alert（高度 / 圆角 / 渐变都不对）
<Alert variant="warning" className="mx-6 mt-4">
  <CircleAlert />
  <AlertDescription>有 3 项基础配置未完成</AlertDescription>
</Alert>

// ❌ 直接拼一个 40px 横条，背景渐变手写
<div className="h-10 px-3 flex items-center bg-gradient-to-r from-orange-50 to-orange-100">
  <span className="text-xs">资源告警</span>
</div>

// ✅ 用 AdminNoticeAlert（自动渐变标签 / 40px / 控制区插槽）
<AdminNoticeAlert
  type="pending-config"
  controls={<span className="text-xs tabular-nums text-[var(--text-secondary)]">4/5</span>}
>
  <span>有 3 项基础配置未完成（导入企业用户、配置至少一个通道、配置安全组），</span>
  <span className="font-medium text-[var(--text-emphasis)] underline underline-offset-2">前往基础信息配置处理</span>
</AdminNoticeAlert>
```
