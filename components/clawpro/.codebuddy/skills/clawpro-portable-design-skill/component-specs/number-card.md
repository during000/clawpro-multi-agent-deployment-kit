# NumberCard

## 0. Auto-Trigger / 强制识别规则（AI 必读）

> 这一段优先级高于本 spec 其它任何段落。**只要命中下面任一信号，必须无条件使用 `<NumberCard>`，禁止再用 `<SurfaceCard>` + 内联 SVG + `<StatNumber>` / `<div>` 自拼。**

**视觉信号（任一即命中）**：

1. 出现「**图标（≤ 20px）+ 短标题 + 大号数字（≥ 20px）**」三件套，并以卡片形式呈现；
2. 同一行/同一区域横排 ≥ 2 张相同结构的卡（典型布局 `grid grid-cols-{2|3|4} gap-5`）；
3. 数字带千分位 / 百分号 / 单位 / `tabular-nums` / DIN 字体；
4. 卡内只有数字（可附 `extra` 进度条/百分比、`footer` 趋势文案），**没有 chart**。

**语义信号（任一即命中）**：

- 文案是「KPI / 概览 / 监控 / Dashboard 顶部 / 配额消耗 / 余额 / 用量 / 累计 / 实例数 / Tokens / 请求数 / 在线数 / 活跃数 / 同比 / 环比 / 当月 / 今日」之类指标；
- 设计稿标注是「Tokens 分析 / 数据概览 / 统计卡 / KPI 卡 / NumberCard / StatCard」。

**例外（命中下列才允许 fallback 到其它组件）**：

- 卡内含 chart / sparkline → 走 `chart-stat.md` 的 `ChartCard`；
- 表单内的数字输入 → 走 `InputNumber`；
- 表格行内单元格 → 直接用 `StatNumber`，不必上卡片层级；
- 营销 / Landing 装饰大数字 → 不走本组件。

**典型反例（设计稿截图 → 必须自动改为 NumberCard）**：

> 「Tokens 分析」区域横排 3 张卡：每张都是「↓ 输入 Tokens / 1,234」「↑ 输出 Tokens / 5,678」「✦ 总 Tokens / 6,912」 ——
> ❌ 不允许用 `div + flex + tabular-nums + text-2xl` 拼；
> ✅ 必须 `grid grid-cols-3 gap-5` 套 3 张 `<NumberCard icon={...} label={...} value={...} />`。

---

## 1. Purpose

- 统一管控端 / 用户端「KPI 概览卡」视觉：渐变图标 + 14px/medium 标题 + DIN/tabular-nums 大数字。
- 替代各业务页面以前各自手写的 `SurfaceCard + 内联 SVG + StatNumber` 组合，避免数字字号、图标渐变、内边距、间距在每个 Dashboard 散开。
- 与 `chart-stat.md` 的关系：`chart-stat` 描述「图表 + 统计区域整体」的视觉口径；`number-card` 是其中**最常用的纯数字卡**的标准实现，业务侧只调用本组件。

## 2. Scope

- 适用端：Admin 优先（Tokens 监控 / OpenClaw 监控 / Dashboard 顶部）；Tenant / Shared 可复用相同 API，但容器层级走 `TenantCard`/`SurfaceCard` 自动分流。
- 必用场景：
  - Dashboard / 监控 / 额度 / 运营页顶部的 KPI 概览区
  - 单个数字 + 标题 + 图标的统计卡
  - 数字旁需要附加百分号 / 进度条 / 徽标（`extra` 槽）
  - 数字下方需要附加二级指标 / 趋势文案（`footer` 槽）
- 不适用场景：
  - 含 chart 的卡片（用 `ChartCard` / 自行组合 `chart-stat`）
  - 表单内的数字输入 / 编辑（用 `InputNumber`）
  - 列表行内的数字单元格（直接用 `StatNumber`，不必上卡片层级）
  - 营销 / Landing 装饰数字（不走本组件）

## 3. Visual Standard

| Item | Value | Notes |
|---|---|---|
| Container | `SurfaceCard` / `TenantCard` 自动分端 | Admin `radius=4px`；Tenant 业务卡 `radius=12px`；阴影由 SurfaceCard 控制，禁止覆盖 |
| Padding | `p-5`（20px） | 不允许业务侧通过 `className` 改 padding |
| Header Row | `mb-3 flex items-center gap-2` | 图标 + 标题水平排列，`mb-3 = 12px` |
| Icon Slot | `inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center` | 固定 18×18，`shrink-0` 避免被标题挤压 |
| Default Icon Style | `<GradientIcon>`：18×18，radialGradient `#202020 → #0080FF`，`gradientTransform="translate(2.81 9) scale(13.5 720)"` | `React.useId` 隔离 gradient id，避免多卡同 id 渲染塌陷 |
| Title | `text-sm font-medium text-[var(--text-title)]` | 14px / Medium / `--text-title` |
| Value | `<StatNumber>`（24px / semibold / `font-variant-numeric: tabular-nums` / DIN） | 由组件内部自动包裹，业务侧只传字符串/ReactNode |
| Extra Slot | 与 `value` 同行，`flex items-center gap-8`（32px） | 用于百分比、徽标、进度条 |
| Footer Slot | `mt-2`（8px） | 二级指标、趋势 / 解释文案 |
| Internal Icons | `RequestsIcon` / `InputTokensIcon` / `OutputTokensIcon` / `TotalTokensIcon` | 内置 4 枚为早期固化的常用项，**非图标上限** |

## 3.1 Icon Source（图标来源 · 强制）

> NumberCard 是渐变图标家族卡，图标必须保持「多彩渐变」同族，禁止退回扁平单色 lucide。

- **当前项目页面层**：图标必须从 `resource-skill-map.json` 的 `number-card` 槽位 `candidates` 中挑选（阶段 9 起候选已达 **22 枚渐变图标**）；该槽位 `allowLucideFallback=false`。
- 组件内置的 `RequestsIcon` / `InputTokensIcon` / `OutputTokensIcon` / `TotalTokensIcon` **4 枚**只是其中已固化的常用项，**不是上限**——不要因为「内置只有 4 枚」就退回 lucide。
- 新指标在 22 枚候选里无合适项时：标 `needs-design-confirmation` 交设计补绘渐变图标，**不回退扁平 lucide**（会破坏渐变家族）、**不在业务页面手搓 `<svg><defs><radialGradient>`**。设计补图落地后，用 `<GradientIcon>` 包其 path 接入。
- 跨仓 / 宿主仓语境的降级见 §7 Portable Fallback。

## 4. Anatomy

```text
NumberCard (root, data-component="number-card")
  └─ SurfaceCard (p-5)
       ├─ Header Row (mb-3 flex items-center gap-2)
       │    ├─ Icon  (18×18, number-card 槽位渐变候选 / GradientIcon / 内置 4 枚；勿用扁平 lucide)
       │    └─ Label (text-sm font-medium --text-title)
       ├─ Value Row
       │    ├─ <StatNumber> (24px / semibold / tabular-nums)
       │    └─ Extra (optional, gap-8 与数字)
       └─ Footer Row (optional, mt-2)
```

## 5. States

- **default**：图标 + 标题 + 数字三层信息完整。
- **with extra**：数字旁挂百分号、徽标、进度条；`extra` 与数字 `gap-8`，禁止业务侧手写 wrapper 改间距。
- **with footer**：数字下方一行 12px 辅助文字 / 二级指标。
- **loading**：组件不内置，调用方在外层用 `<Skeleton>` 占位，保留卡片尺寸不抖动。
- **empty**：禁止显示「— / 暂无数据」而无解释；空态由调用方提供，文案要说明无数据原因（与 `chart-stat.md` 一致）。
- **delta（正向 / 负向 / 中性）**：通过 `footer` 槽传带语义色的趋势文字，不依赖颜色单独表达。
- **no-permission**：调用方决定是否渲染，不在卡内显示「无权限」提示。

## 6. Demo Repo Usage

- 当前 demo 仓组件：`client/src/components/ui/number-card.tsx`
- 设计系统展示：`client/src/pages/DesignSystemComponents.tsx`（NumberCardPreview）
- 典型页面：
  - `client/src/pages/admin/TokensMonitor.tsx` 顶部 4 张概览卡
  - `client/src/pages/admin/OpenClawMonitor.tsx` 后续概览区（待迁移）
- 推荐组合：`grid grid-cols-4 gap-5` 横排 4 张

```tsx
import {
  NumberCard,
  RequestsIcon,
  InputTokensIcon,
  OutputTokensIcon,
  TotalTokensIcon,
} from "@/components/ui/number-card";

// 1) 开箱用法
<div className="grid grid-cols-4 gap-5">
  <NumberCard icon={<RequestsIcon />}     label="总请求数"    value="1,841" />
  <NumberCard icon={<InputTokensIcon />}  label="输入 Tokens" value="533,112" />
  <NumberCard icon={<OutputTokensIcon />} label="输出 Tokens" value="419,040" />
  <NumberCard icon={<TotalTokensIcon />}  label="总 Tokens"   value="952,152" />
</div>

// 2) extra 槽（百分比 + 进度条）
<NumberCard
  icon={<TotalTokensIcon />}
  label="今日全局配额消耗"
  value="68%"
  extra={<ProgressBar value={68} max={100} />}
/>

// 3) footer 槽（二级指标 / 趋势文案）
<NumberCard
  icon={<RequestsIcon />}
  label="总请求数"
  value="1,841"
  footer={<span className="text-xs text-[var(--text-secondary)]">较昨日 +8.2%</span>}
/>

// 4) 自定义渐变图标（任意 SVG path）
import { GradientIcon } from "@/components/ui/number-card";
<NumberCard
  icon={<GradientIcon><path d="..." /></GradientIcon>}
  label="自定义指标"
  value="—"
/>
```

## 7. Portable Fallback

### 7.1 If host repo already has KPI card

- 复用宿主仓的 KPI 卡组件，但视觉必须对齐：圆角（Admin 4px / Tenant 12px）、`p-5` 内边距、18×18 图标、14px Medium 标题、24px Semibold 数字、tabular-nums、`gap-8` 数字↔extra 间距、`mt-2` 数字↔footer 间距。
- 渐变图标如果宿主仓没有同款：**跨仓 / 可移植**语境可用品牌蓝单色填充 fallback（保留尺寸 / 标签 / aria-label），不要再造一套高饱和渐变；**当前项目页面层**则不走单色 fallback，而是标 `needs-design-confirmation` 交设计补绘渐变图标（当前项目 `number-card` 槽位 `allowLucideFallback=false`）。
- 数字字体如果宿主仓不能切到 DIN，至少加 `font-variant-numeric: tabular-nums` 保证对齐稳定。

### 7.2 Minimal React fallback

```tsx
export function PortableNumberCard({
  icon, label, value, extra, footer,
}: {
  icon?: React.ReactNode;
  label: React.ReactNode;
  value: React.ReactNode;
  extra?: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <section
      data-component="number-card"
      className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-5"
    >
      <div className="mb-3 flex items-center gap-2">
        {icon ? (
          <span className="inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center">
            {icon}
          </span>
        ) : null}
        <span className="text-sm font-medium text-[var(--cp-text-title)]">
          {label}
        </span>
      </div>
      {extra ? (
        <div className="flex items-center gap-8">
          <strong className="font-[var(--cp-font-sans)] text-2xl font-semibold tabular-nums text-[var(--cp-text-emphasis)]">
            {value}
          </strong>
          {extra}
        </div>
      ) : (
        <strong className="font-[var(--cp-font-sans)] text-2xl font-semibold tabular-nums text-[var(--cp-text-emphasis)]">
          {value}
        </strong>
      )}
      {footer ? <div className="mt-2">{footer}</div> : null}
    </section>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<section class="cp-number-card">
  <header>
    <span class="cp-number-card__icon"><!-- 18×18 SVG --></span>
    <span class="cp-number-card__label">总请求数</span>
  </header>
  <strong class="cp-number-card__value">1,841</strong>
  <p class="cp-number-card__footer">较昨日 +8.2%</p>
</section>
```

```css
.cp-number-card {
  border: 1px solid var(--cp-border);
  border-radius: 4px;
  background: var(--cp-surface);
  padding: 20px;
}
.cp-number-card header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.cp-number-card__icon {
  display: inline-flex;
  width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.cp-number-card__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--cp-text-title);
}
.cp-number-card__value {
  display: block;
  font-family: var(--cp-font-sans);
  font-size: 24px;
  line-height: 32px;
  font-weight: 600;
  color: var(--cp-text-emphasis);
  font-variant-numeric: tabular-nums;
}
.cp-number-card__footer {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--text-secondary);
}
```

## 8. Migration Rules

- 旧写法：在 Dashboard / 监控页内手写 `<SurfaceCard className="p-5"> <div className="flex items-center gap-2"><svg .../>…</div> <StatNumber>…</StatNumber> </SurfaceCard>`，每页约 12-48 行。
- 新口径：
  1. 一律替换为 `<NumberCard icon={...} label={...} value={...} />`；
  2. 图标从 `number-card` 槽位的 22 枚渐变候选挑选；若是「请求数 / 输入 Tokens / 输出 Tokens / 总 Tokens」之一，直接用内置 4 枚（它们是已固化常用项），不要再复制 SVG path；
  3. 候选里有对应渐变图标的，用其 `webPath` / `import` 经 `icon` 传入；候选确无合适项时标 `needs-design-confirmation` 交设计补绘，**不回退扁平 lucide**、不在业务页面手写 `<svg><defs><radialGradient>…`；设计补图后用 `<GradientIcon>` 包一层 path 接入；
  4. 数字旁的百分比 / 进度条走 `extra` 槽；趋势文案 / 二级指标走 `footer` 槽。
- 不允许新增：
  - 业务侧用 `<SurfaceCard>` + 内联 SVG + `<StatNumber>` 拼装出「伪 NumberCard」；
  - 通过 `className` 覆盖 NumberCard 的 padding / 圆角 / 字号 / 字重 / 颜色；
  - 把 `value` 文字直接写成 `text-2xl font-bold`，绕过 `StatNumber`；
  - 在多个 NumberCard 共享同一个 `<radialGradient id="...">`（必须使用 `GradientIcon` 让 `React.useId` 自动隔离）。

## 9. Do / Don't

Do:

- 用 `<NumberCard>` 替代所有「图标 + 标题 + 数字」三件套。
- 数字一律走 `value` 槽（自动 `StatNumber`，自动 tabular-nums）。
- 渐变图标从 `number-card` 槽位 22 枚候选挑选；命中内置 4 枚常用项可直接用 `<GradientIcon>` 或内置。
- `extra` / `footer` 槽分工固定：附加单位/进度走 extra，趋势/解释走 footer。

Don't:

- 不要在业务页面再手搓 `<SurfaceCard>` + 内联 SVG + `<StatNumber>`。
- 不要让 `value` 内嵌 `text-2xl font-bold` 等字号 / 字重 class。
- 不要通过 `className` 覆盖图标尺寸（必须 18×18）/ 标题色（必须 `--text-title`）/ 卡片 padding（必须 `p-5`）。
- 不要在多张 NumberCard 间复制同一份带固定 `id="grad-1"` 的 SVG（gradient 会塌陷，必须用 `GradientIcon`）。
- 不要把 chart 塞进 NumberCard；带 chart 的卡走 `chart-stat` 的 ChartCard。
- 不要用扁平单色 lucide 当统计卡图标（`number-card` 槽位 `allowLucideFallback=false`）；候选无合适项时标 `needs-design-confirmation` 交设计补绘，勿因「内置只有 4 枚」退回 lucide。

## 10. QA Checklist

- [ ] 容器走 `SurfaceCard`（Admin 4px / Tenant 12px），padding=20px
- [ ] 图标 18×18，标题 14px Medium `--text-title`
- [ ] 数字走 `StatNumber`，24px Semibold + tabular-nums
- [ ] 横排 4 卡时使用 `grid grid-cols-4 gap-5`
- [ ] 渐变图标多卡共存时无塌陷（`React.useId` 自动隔离 gradient id）
- [ ] 内置 4 枚图标（请求数 / 输入 / 输出 / 总）与 Tokens 监控页 1:1 对齐
- [ ] `extra` 槽与数字 gap=32px；`footer` 槽与数字 mt=8px
- [ ] 业务页面已无 `<SurfaceCard>` + 内联 SVG + `<StatNumber>` 手搓写法
- [ ] 宿主仓 fallback 可执行（不依赖 demo 仓 SurfaceCard / StatNumber）

## 11. References

- Demo code: `client/src/components/ui/number-card.tsx`
- Demo showcase: `client/src/pages/DesignSystemComponents.tsx`（NumberCardPreview）
- Demo page: `client/src/pages/admin/TokensMonitor.tsx`
- Related spec: `component-specs/chart-stat.md`、`component-specs/card-surface.md`
- Related token: `tokens/typography.md`（StatNumber / tabular-nums）、`tokens/colors.md`（`--text-title`）
- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`（owner: addietang）

## 代码对照（✅/❌）

### ❌ 错误：自由 padding / 自由 Card
```tsx
<div className="rounded-xl border p-3 shadow-md">
  <div className="text-gray-500">本月活跃</div>
  <div className="text-2xl font-bold">1,234</div>
</div>
```
**为什么错**：圆角/阴影/内边距与 SurfaceCard 不一致；多个 NumberCard 并排会产生视觉抖动。

### ✅ 正确：NumberCard 包装
```tsx
<NumberCard
  title="本月活跃"
  value={1234}
/>
{/* 内部走 SurfaceCard p-5 / 12px 圆角 */}
```

---

### ❌ 错误：数字硬编码字号字重
```tsx
<NumberCard title="实例数">
  <span className="text-3xl font-extrabold">{count}</span>
</NumberCard>
```
**为什么错**：与系统 24px Semibold + tabular-nums 不一致；位数变化会跳。

### ✅ 正确：value 走 StatNumber
```tsx
<NumberCard
  title="实例数"
  value={count}
/>
{/* value 内部默认渲染 <StatNumber>，无需手写 */}
```

---

### ❌ 错误：Icon 用大尺寸彩色
```tsx
<NumberCard title="本月活跃" value={1234}>
  <div className="absolute right-4 top-4">
    <ActivityIcon className="w-8 h-8 text-blue-500" />
  </div>
</NumberCard>
```
**为什么错**：图标过大喧宾夺主；蓝色与品牌蓝混淆，弱化数字。

### ✅ 正确：18×18 弱色 icon
```tsx
<NumberCard
  title="本月活跃"
  value={1234}
  icon={<ActivityIcon size={18} className="text-[var(--cp-text-weak)]" />}
/>
```

---

### ❌ 错误：title 用大字号
```tsx
<div className="text-base font-medium">本月活跃</div>
<StatNumber>{count}</StatNumber>
```
**为什么错**：title 与 value 字号差距不够，视觉层级混乱；title 应弱化。

### ✅ 正确：title 14px / weak
```tsx
<NumberCard
  title="本月活跃"  /* 内部 14px Medium / --cp-text-weak */
  value={count}
/>
```

---

### ❌ 错误：footer 写在 card 外
```tsx
<NumberCard title="实例数" value={42} />
<MetaText className="-mt-2 ml-5">较昨日 +5</MetaText>
```
**为什么错**：footer 与卡片视觉脱节；间距 hack 在不同 zoom 下错位。

### ✅ 正确：extra / footer 槽
```tsx
<NumberCard
  title="实例数"
  value={42}
  extra={<TrendBadge delta={5} />}     /* 右上角 */
  footer={<MetaText>较昨日 +5</MetaText>} /* 卡内底部 */
/>
```
