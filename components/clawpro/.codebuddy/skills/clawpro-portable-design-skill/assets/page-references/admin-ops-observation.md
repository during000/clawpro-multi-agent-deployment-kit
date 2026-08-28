# 运维观测页 · 服务未开通态 · admin-ops-observation

> **类别**：服务开通引导页 / 特性介绍页（Onboarding · Feature Overview）
> **路由**：`/admin/ops-observation`（服务未开启时的初始态）
> **源码**：`client/src/pages/admin/OpsObservation.tsx`
> **截图**：`admin-Operation-page.png`（1440×900）

> ⚠️ 此截图是**未开通 CLS 日志服务**时的态。开通后页面会切换到大盘形态（与 `admin-tokens-monitor.md` 同结构），需另起一份样本。

---

## 1. 视觉骨架

```
ConsoleShell（左导航）
└─ ContentArea
   ├─ AlertBanner（warning：3 项基础配置未完成）
   ├─ AdminPageHeader
   │  ├─ Title「运维观测」
   │  ├─ Subtitle「全方位守护系统稳定运行，从被动救火到主动防御」
   │  └─ Right: DateRange [2026-06-10] — [2026-06-10]
   │
   ├─ ▼ CTA 卡（横长 banner · 主开通行动）
   │  SurfaceCard padding=24
   │  ├─ Left:
   │  │  · Title「运维观测需要开启 CLS 日志服务」（PanelTitle）
   │  │  · Body「开启后，为您赠送3个 ClawPro 专属 CLS 日志服务免费额度，
   │  │   预估可覆盖 500 台 Agent 机器3个月的日均使用量；服务到期后，
   │  │   CLS 将按量计费。计费详情↗」
   │  └─ Right: Button「开启 CLS 日志服务」（**深色实心 · primary**）
   │
   ├─ ▼ Section ①「开启CLS日志服务后您可以在此处获得以下观测数据：」
   │  └─ SurfaceCard 内 grid-cols-2 gap=16
   │     ├─ FeatureCell（icon + 标题 + 副标题）
   │     │   📈 业务运行健康度实时监控
   │     │   "聚焦消息处理总量，介入效率与卡死会话，保障系统稳定运行"
   │     └─ FeatureCell
   │         📊 应用日志与 OTEL 指标全景洞察
   │         "多维度分析日志级别与模块分布，精细化运营消息处理，队列状态与执行时延"
   │
   └─ ▼ Section ②「开启CLS日志服务后您还可以在Tokens监控和运维观测页面中获得以下观测数据：」
      └─ SurfaceCard 内 grid-cols-2 gap=16 row-gap=20
         ├─ 📈 高Token会话实时分析与管控
         ├─ 🔗 单会话全链路Tokens透视
         ├─ 📈 会话全局运行态势监控
         └─ 📈 会话详情与交互效率精细化分析
```

---

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 全局提醒 | `Alert` warning + `AlertOperationInfoIcon` | `@/components/ui/alert` | `component-specs/alert.md` |
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/admin-page-header.md` |
| 日期 | `DatePicker` ×2 | `@/components/ui/date-picker` | `component-specs/date-picker.md` |
| CTA 容器 | `SurfaceCard` | `@/components/ui/Surface` | `component-specs/surface.md` |
| CTA 按钮 | `Button` variant=default size=lg（**深色 / 强对比**） | `@/components/ui/button` | `component-specs/button.md` |
| 特性单元 | 自定义 FeatureCell（lucide icon 24 + Title + Body）—— 无 ui 组件，按页面规范手写 | — | 用法见此页源码 |
| 章节标题 | `BodyMedium` 14px / 文字 + 灰底章节背景 | `@/components/ui/Typography` | `component-specs/typography.md` |
| 计费详情链接 | `<a>` + `ArrowUpRight` icon | — | — |

---

## 3. 关键 token / 规范要点

- **整页背景层级**（自下而上）：
  - 页面底色 `tokens.color.bg.layer1`
  - 章节包裹卡 `bg.layer2` + `radius.lg` + `shadow.elevation1`
  - 章节内 FeatureGrid 直接铺在卡内，**不再额外加边框**
- **CTA banner 特征**：横向布局，**左大段文字 + 右一个深色按钮**，`Button` 用 default 变体，背景 `gray.900` —— 这是 ClawPro 管控端"主张性开通操作"的统一手感。
- **Feature 卡 grid**：固定 `grid-cols-2 gap-4`，**不要随屏宽自适应到 3/4 列**（运维场景 UI 较密，2 列阅读节奏最好）。
- **Feature icon**：lucide 24px，颜色 `brand.primary` 或 `gray.600`（与 icon 含义匹配）。**不要用大色块或带 SurfaceCard 圆形容器**——保持极简。
- **章节文案**：放在卡**外**（卡顶上方 12px），不是卡内 header；这样阅读时章节是分组标签，卡是内容容器。
- **"开启服务"动作**：截图里只有 1 个 CTA。如果业务需要多个服务并列引导，**不要做成 4 个并排卡**，应做成**单列垂直堆叠**的多个 CTA banner。

---

## 4. why-typical

- 是 ClawPro **服务未开通 / 引导开通态的标准模板**，与传统"空状态页（admin-agent-template）"互补：
  - **空状态**：用户已激活，但暂无数据 → 居中 EmptyState
  - **未开通**：用户尚未激活，需先付费/开通 → 顶部 CTA + 下方功能预览
- "开通后您将获得 X 项能力" 的 Feature Grid 是 SaaS 通用模式——配置中心、安全中心、日志服务、流水线等都能复用此页骨架。

---

## 5. ❌反例 / ✅正例

| 场景 | ❌ 反例 | ✅ 正例 |
|---|---|---|
| 未开通态怎么表达 | 整页只放一个 EmptyState「请先开通」 | **CTA banner + Feature 预览**，让用户提前看到价值再决策 |
| Feature grid 列数 | 响应式 1/2/3/4 列 | 固定 2 列（运维信息密度大） |
| Feature icon 装饰 | 给每个 icon 配独立圆形彩色卡片 | 24px 单色 lucide，简洁不抢戏 |
| CTA 按钮 | 用 ghost 或 outline 弱按钮 | 深色 primary 实心按钮，**强行动指引** |
| 章节标题位置 | 嵌在卡内当 header | 卡外做章节分组（卡顶上方） |
| 多个开通服务 | 网格 2×2 4 个 CTA 卡 | 垂直堆叠 N 个长 CTA banner，避免横向比较干扰 |
| 计费链接 | 单独放一个"了解计费"按钮 | 在文案中以 `计费详情↗` 内嵌链接出现 |
