# 云开发开通页 · 动态 hero 未开通态 · admin-cloud-dev-activation

> **类别**：能力开通页 / 特性介绍页（Feature Activation · 动态 hero）
> **路由**：`/admin/cloud-dev`（云开发能力**未开通**时的初始态；点「立即开通」后切换为 `CloudDevManagement` 列表）
> **源码**：`client/src/pages/admin/CloudDevActivation.tsx`
> **截图**：`admin-cloud-dev-activation.png`（1440×900，dev 真实渲染）

> ⚠️ 此页是**能力开通引导态**，与 `admin-ops-observation.md`（CTA banner 型未开通态）互补：本页特征是 **hero 区带 DarkVeil 动态背景**。是否引入 DarkVeil 需命中 `component-specs/dark-veil.md` §0 Auto-Trigger 并经设计师拍板，**不可作为普通开通页默认模板套用**。

---

## 1. 视觉骨架

```
SurfaceCard（overflow-hidden）
├─ AdminPageHeader  Title「云开发管理」/ Subtitle「管理企业云开发环境的创建、分配与生命周期」
│
├─ ▼ Hero 区（relative overflow-hidden · px-[60px] py-12 · 动态背景三件套）
│   背景三层（全部 pointer-events-none）：
│     · 第 0 层 基底  absolute inset-0 bg-[#E0EBFE]
│     · 第 1 层 DarkVeil  absolute inset-0 h-full w-full（tint #B2C3FF / translateY 72px / 顶部 mask 22% 淡出）
│     · 第 2 层 收束叠层  bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]
│   内容层 relative z-10 · grid md:grid-cols-2 items-end gap-4：
│   ├─ 左列：
│   │   · 图标徽章（h-14 w-14 · rounded-[8px] · border-white/60 bg-white/30 backdrop-blur-md）内嵌 cloud-dev.svg 28px
│   │   · TenantPageTitle「开通云开发能力」
│   │   · BodyText（max-w-[580px] · text-muted）"为企业成员提供独立的云端开发环境…快速构建与部署应用"
│   │   · Button variant=claw-primary size=lg「立即开通 →」（loading 态：spinner +「开通中...」）
│   └─ 右列：开通后可获得 —— grid-cols-[repeat(2,minmax(0,240px))] gap-2.5，4 张玻璃拟态权益卡
│       卡：rounded-[9px] · border-white/60 bg-white/40 backdrop-blur-sm · icon 16px + 12px 文案
│       ① 管理员可创建、删除、管理云开发环境  ② 灵活分配环境给指定成员使用
│       ③ 统一监控环境运行状态与资源用量      ④ 企业级安全管控与审计日志
│
├─ ▼ 核心能力（px-[60px] pt-10 pb-10）
│   SectionTitle「核心能力」
│   └─ grid md:grid-cols-2 gap-4 · SurfaceInner ×4（图标 36px + h4 标题 + 副标题，hover 上浮 0.5）
│      · 独立云开发环境 · 云数据库 · 云函数 · 静态网站托管
│
└─ ▼ 底部说明（居中 · text-weak 12px）Zap icon +「开通即用，无需额外配置基础设施」
```

---

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/page-header.md` |
| 外层容器 | `SurfaceCard`（`overflow-hidden`）/ 核心能力卡 `SurfaceInner` | `@/components/ui/Surface` | `component-specs/card-surface.md` |
| **hero 动态背景** | `DarkVeil`（默认导出，依赖 `ogl`） | `@/components/ui/DarkVeil` | `component-specs/dark-veil.md` |
| 标题 | `TenantPageTitle` / `SectionTitle` | `@/components/ui/Typography` | `component-specs/typography.md` |
| 正文 / 弱说明 | `BodyText` / `MetaText` | `@/components/ui/Typography` | `component-specs/typography.md` |
| 主行动按钮 | `Button` variant=`claw-primary` size=lg（+ `ArrowRight`） | `@/components/ui/button` | `component-specs/button.md` |
| hero 图标徽章 | 已登记 SVG `/assets/admin-sidebar/cloud-dev.svg`（槽位 `admin-sidebar`，可回退） | — | `references/assets-icons.md` |
| **核心能力卡图标 ×4** | 已登记 SVG `/assets/admin-cloud-dev/*.svg` 等（槽位 **`card-left-icon` 卡片左侧·块状多彩渐变，禁 lucide**） | — | `references/assets-icons.md §5.5` |
| **hero 右列权益卡 ×4** | 已登记 SVG `/assets/admin-memory-management/version-compare/feature-*.svg`（槽位 **`feature-card` 企业特性卡片图标，禁 lucide**） | — | `references/assets-icons.md §5.5` |
| 装饰小图标 | lucide `Zap` / `ArrowRight` | `lucide-react` | — |

---

## 3. 关键 token / 规范要点

- **hero 背景三件套（自下而上，全部 `pointer-events-none`）**：基底 `#E0EBFE` → DarkVeil canvas（`tintColor="#B2C3FF"` / `speed=1.1` / `warpAmount=1.1` / `noiseIntensity=0.05` / `transform: translateY(72px)` / 顶部 `maskImage` 22% 淡出）→ 收束叠层 `via-white/10 to-[#E0EBFE]`。内容层 `relative z-10`。**调氛围只动 speed/warp/noise 三个数，不动三层结构**。完整口径见 `component-specs/dark-veil.md` §3/§4。
- **圆角例外（hero 限定）**：管控端铁律是 4px，但 hero 内**玻璃拟态装饰元素**例外——图标徽章 `rounded-[8px]`、权益卡 `rounded-[9px]`。仅限 hero 区的装饰玻璃元素，**功能区（核心能力 SurfaceInner、按钮、表单）仍遵守 4px**。理由见 `references/conflict-log.md` C-018 / `references/admin-cloud-dev-activation.md` §6。
- **玻璃拟态权益卡**：`border-white/60 bg-white/40 backdrop-blur-sm`，宽度 `minmax(0,240px)` 自适应不撑满，2 列。靠半透明白 + 模糊压在 DarkVeil 之上保证可读。
- **主按钮**：用 `claw-primary`（品牌蓝实心），不是深色 `default`——这是用户侧色彩的「主张性开通」手感；与 `admin-ops-observation` 的深色 CTA 区分（那是纯管控运维场景）。
- **图标策略**：业务/品牌图标（云开发、数据库、云函数、托管、权益）一律走**已登记 SVG**；仅 `Zap`/`ArrowRight` 等通用装饰用 lucide。其中**核心能力 4 卡 = `card-left-icon`（卡片左侧·块状多彩渐变）槽位**、**hero 右列权益卡 4 张 = `feature-card`（企业特性卡片图标）槽位**，二者均 `allowLucideFallback=否`，**禁回退 lucide、禁手搓 inline SVG**，无候选标 `needs-design-confirmation`。当前项目候选见 `resource-skill-map.json`；**跨仓/main 启用时该候选源不存在，须由宿主仓正式 registry 供图（参 `assets/icon-registry.example.json`），槽位禁 lucide 约束不变**。新增图标须先登记，见 `references/assets-icons.md §5.5`。
- **跨仓兜底**：宿主仓无 `ogl`/WebGL 时按 `dark-veil.md` §9 分档——L0 完整移植 / L1 纯 CSS 静态渐变（`portable/css/dark-veil.css`，`migration-map` 默认档）/ L2 纯色 `#E0EBFE` 或截图。至少做到 L1，`prefers-reduced-motion` 降 L2。

---

## 4. why-typical

- 是 ClawPro **「能力开通页 + 动态 hero」的唯一标杆样本**，定位区别于另外两类未开通/空态：
  - **空状态**（`admin-agent-template`）：已激活但暂无数据 → 居中 EmptyState。
  - **CTA 引导**（`admin-ops-observation`）：未开通服务、信息密度高 → 顶部深色 CTA banner + 章节化 FeatureGrid。
  - **动态 hero 开通**（本页）：要「科技感 / 流动光效」氛围、设计已拍板 → DarkVeil hero + 玻璃权益卡 + 核心能力卡。
- hero「品牌氛围背景 + 左文右权益 + 下方核心能力」结构可复用于其他**重点能力的开通/介绍页**，但 DarkVeil 本身受 Auto-Trigger 门控，不得无脑套到普通功能页。

---

## 5. ❌反例 / ✅正例

| 场景 | ❌ 反例 | ✅ 正例 |
|---|---|---|
| 何时上 DarkVeil | 任意开通页/列表页都铺动态背景图省事 | 仅命中 Auto-Trigger（开通页/能力 hero + 设计拍板）才用，单页最多 1 实例 |
| 背景层级 | 只放一层 DarkVeil canvas，文字直接压上去 | 基底 #E0EBFE + DarkVeil + 收束叠层三件套，内容 `z-10` |
| canvas 交互 | 把按钮/链接直接落在 canvas 上 | 背景三层全 `pointer-events-none`，交互元素只在内容层 |
| 权益卡圆角 | 为「统一」强行改成 4px 直角，玻璃感全失 | hero 装饰玻璃元素例外用 8/9px，功能区仍 4px |
| 主按钮 | 用 ghost/outline 弱按钮或深色 default | `claw-primary` 品牌蓝实心，强行动指引 |
| 无 ogl 环境 | 直接留白 / 报错 / 删掉背景 | 退 L1 纯 CSS 静态渐变兜底，绝不空白 |
| 图标 | 业务图标也用 lucide 凑合 | 业务/品牌走已登记 SVG，仅装饰用 lucide |
