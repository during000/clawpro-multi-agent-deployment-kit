# Admin · 云开发开通页（Cloud Dev Activation）

> 管控端「云开发管理」未开通态页面的**结构 spec + hero 配方骨架**。这是一个「功能开通 / 能力介绍」页，顶部用 DarkVeil 动态背景 hero（命中 `component-specs/dark-veil.md` §0 Auto-Trigger），下方用核心能力卡。本文聚焦**页面骨架与分区**；动态背景的参数与跨仓兜底见 `component-specs/dark-veil.md`，通用 hero 配方见 `references/page-recipes.md`「云开发开通页 hero」。

## 1. 适用场景

- **命中**：管控端某能力「未开通 / 首次引导」态的整页，需要一个有氛围感的开通引导（图标 + 标题 + 描述 + 主操作 + 权益预览 + 核心能力列表）。当前实例：`client/src/pages/admin/CloudDevActivation.tsx`。
- **不命中**：开通后的功能主体（列表 / 表单 / 监控）——那些回到 `references/page-recipes.md` 的列表 / 设置 / Dashboard 骨架，**不要**继续套动态背景 hero。

## 2. 页面结构（Anatomy）

```text
page-enter
  AdminPageHeader            ← 标题「云开发管理」+ 描述（页头，不在 hero 内）
  SurfaceCard (overflow-hidden)
    ├─ Hero 区  (relative overflow-hidden px-[60px] py-12)
    │    背景三层（见 §3，全部 pointer-events-none）
    │    内容层 (relative z-10, grid md:grid-cols-2 items-end gap-4)
    │      左列：图标徽章 + 主标题 + 描述 + 主操作按钮
    │      右列：权益卡 ×4（开通后可获得）
    ├─ 核心能力区 (px-[60px] pt-10 pb-10)
    │    SectionTitle「核心能力」
    │    grid md:grid-cols-2 gap-4 → SurfaceInner ×4（icon + 标题 + 描述）
    └─ 底部说明 (flex justify-center px-[60px] pb-10)
         Zap icon + 一句话补充说明
```

要点：
- 整页根 `page-enter`；页头用 `AdminPageHeader`（标题 + 描述），**不放进 hero**。
- 整块包在**单个** `SurfaceCard`（`overflow-hidden`，让 hero 背景不溢出圆角）；hero / 核心能力 / 底部说明是它内部的三段，不要各自再套 `SurfaceCard`（避免套娃）。
- 左右内边距统一 `px-[60px]`，让 hero、核心能力、底部说明三段左右对齐。

## 3. Hero 配方骨架（背景三层 + 内容层）

> 完整参数与算法见 `component-specs/dark-veil.md` §3/§4/§5。这里只给页面骨架位置。

```tsx
{/* Hero 区域 */}
<div className="relative overflow-hidden px-[60px] py-12">
  {/* 第 0 层：统一基底 #E0EBFE，整体均匀避免局部突兀 */}
  <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />
  {/* 第 1 层：DarkVeil 动态背景（配方见 dark-veil.md §4），顶部 mask 22% 淡出露基底 */}
  <DarkVeil
    speed={1.1} warpAmount={1.1} noiseIntensity={0.05} tintColor="#B2C3FF"
    className="pointer-events-none absolute inset-0 h-full w-full"
    style={{ transform: "translateY(72px)", maskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)", WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)" }}
  />
  {/* 第 2 层：柔化收束叠层，中部微提亮、底部收束回 #E0EBFE 与下方内容无缝 */}
  <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />

  {/* 内容层：永远 relative z-10 */}
  <div className="relative z-10 grid items-end gap-4 md:grid-cols-2">
    {/* 左列：图标徽章 + 主标题 + 描述 + 主操作 */}
    {/* 右列：权益卡 ×4 */}
  </div>
</div>
```

### 3.1 Hero 内容层（左列）

| 元素 | 写法 / 口径 |
|---|---|
| 图标徽章 | `h-14 w-14 rounded-[8px] border border-white/60 bg-white/30 backdrop-blur-md`，内置 16px 区放 `cloud-dev.svg`（`h-7 w-7`）。**玻璃拟态徽章，属 hero 装饰例外**（见 §6） |
| 主标题 | `TenantPageTitle`（hero 大标题语义），文案如「开通云开发能力」 |
| 描述 | `BodyText`，`max-w-[580px]`，`text-[var(--text-muted)]` |
| 主操作 | `Button variant="claw-primary" size="lg"`；loading 态显示 `animate-spin` 圈 + 「开通中…」，常态显示文案 + `<ArrowRight>`（lucide） |

### 3.2 Hero 内容层（右列：权益卡 ×4）

- 容器：`grid w-fit grid-cols-[repeat(2,minmax(0,240px))] gap-2.5`（2×2，卡宽自适应内容、上限 240px，底端与左列按钮对齐）。
- 单卡：`rounded-[9px] border border-white/60 bg-white/40 px-3.5 py-3 backdrop-blur-sm hover:border-white/80`，内含 16px 业务图标 + `text-xs leading-5 text-[var(--text-secondary)]` 文案。**玻璃拟态卡，属 hero 装饰例外**（见 §6）。

## 4. 核心能力区（SurfaceInner ×4）

- 容器：`px-[60px] pt-10 pb-10`，顶部 `SectionTitle`「核心能力」（`mb-6`）。
- 网格：`grid grid-cols-1 gap-4 md:grid-cols-2`。
- 单卡：`SurfaceInner`（**走组件，不手写卡**），`flex items-center p-4 transition-all duration-200 hover:-translate-y-0.5 hover:border-[var(--cp-border-control)]`；内含 36px 业务图标（`h-9 w-9`）+ `h4`（`text-sm font-semibold text-[var(--text-title)]`）+ `p`（`text-xs leading-relaxed text-[var(--text-muted)]`）。

## 5. 图标资产（全部走已登记业务 SVG，不用 lucide 占位）

> 通用动作图标（如 hero 按钮的 `ArrowRight`、底部 `Zap`）用 lucide；**业务/能力图标一律用已登记 registry SVG**（见 `SKILL.md` §17 / `references/assets-icons.md`）。

| 位置 | 资产路径 | resource-skill-map 槽位 | lucide 回退 |
|---|---|---|---|
| hero 图标徽章（左列） | `/assets/admin-sidebar/cloud-dev.svg` | `admin-sidebar`（业务能力图标） | 允许 |
| 核心能力①独立云开发环境 | `/assets/admin-skill-packages/advanced-dev-skill-package.svg` | `card-left-icon`（卡片左侧·块状多彩渐变） | **禁** |
| 核心能力②云数据库 | `/assets/admin-cloud-dev/cloud-database.svg` | `card-left-icon`（卡片左侧·块状多彩渐变） | **禁** |
| 核心能力③云函数 | `/assets/admin-cloud-dev/cloud-function.svg` | `card-left-icon`（卡片左侧·块状多彩渐变） | **禁** |
| 核心能力④静态网站托管 | `/assets/admin-cloud-dev/static-hosting.svg` | `card-left-icon`（卡片左侧·块状多彩渐变） | **禁** |
| 权益卡 ×4（hero 右列） | `/assets/admin-memory-management/version-compare/feature-token.svg` / `feature-tenant.svg` / `feature-backup.svg` / `feature-encrypt.svg` | `feature-card`（企业特性卡片图标） | **禁** |

> **槽位口径**：`card-left-icon` / `feature-card` 均为 `allowLucideFallback=否` 的不可回退槽位（见 `references/assets-icons.md §5.5`）——无合适候选时标 `needs-design-confirmation` 交设计补绘，**禁回退扁平 lucide、禁手搓 inline SVG**。
> **跨仓 / main 语境**：本表的 SVG 路径与 `resource-skill-map.json` 候选**仅在当前项目成立**；换皮或在无 `client/src/design-assets/` 的仓启用时，这两个槽位由宿主仓正式 registry 提供候选（参 `assets/icon-registry.example.json`），仍**禁 lucide**，无候选即标 `needs-design-confirmation`。

## 6. 圆角例外说明（hero 装饰玻璃元素）

> 管控端铁律是面板类元素 4px（`SKILL.md` §2.4）。本页 hero 内的**图标徽章 `rounded-[8px]`、权益卡 `rounded-[9px]`** 是**玻璃拟态装饰元素**，浮在 DarkVeil 背景上、不是标准数据面板，属**已知 hero 装饰例外**，不按 4px 收口。**例外仅限 hero 区装饰元素**；hero 以外（核心能力 `SurfaceInner`、底部说明、任何数据面板）一律 4px，不得借此例外扩散。详见 `SKILL.md` §2.4 hero 装饰例外条与 `references/conflict-log.md` C-018。

## 7. Portable Fallback

- 宿主仓能装 `ogl` → hero 走 DarkVeil **L0**（完整动态）。
- 不便引 ogl/WebGL → hero 背景降级 **L1**（纯 CSS 渐变光晕，`portable/css/dark-veil.css` + `portable/html-css/dark-veil.html`）。
- 禁脚本 / 静态导出 / `reduced-motion` → **L2**（纯色 `#E0EBFE` 或静态截图 `admin-cloud-dev-activation.png`）。
- 页面其余结构（`SurfaceCard` / `SurfaceInner` / Typography / Button）按各自 spec 的 portable 兜底；详见 `references/migration-map.md`（DarkVeil 归 L1）。

## 8. QA Checklist

- [ ] 页头用 `AdminPageHeader`（标题 + 描述），不塞进 hero。
- [ ] 全块单个 `SurfaceCard(overflow-hidden)`，内部三段不套娃。
- [ ] hero 背景三层齐全且 `pointer-events-none`，内容层 `relative z-10`（对照 `dark-veil.md`）。
- [ ] hero 参数对齐配方（speed 1.1 / warp 1.1 / noise 0.05 / tint #B2C3FF / mask 22% / translateY 72px）。
- [ ] 核心能力用 `SurfaceInner`（非手写卡），4px 圆角；hero 装饰玻璃元素的 8/9px 圆角是已登记例外。
- [ ] 业务图标全部走已登记 SVG，仅 `ArrowRight`/`Zap` 用 lucide。
- [ ] 宿主仓无 ogl 时 hero 已落 L1/L2，不空白。

## 9. References

- 实现：`client/src/pages/admin/CloudDevActivation.tsx`
- 动态背景组件 spec：`component-specs/dark-veil.md`
- hero 通用配方：`references/page-recipes.md`「云开发开通页 hero」
- 跨仓兜底：`references/migration-map.md`（DarkVeil → L1）
- 决策溯源：`references/conflict-log.md` C-018 / C-019
- 卡片 / Typography / Button：`component-specs/card-surface.md`、`references/components.md`
