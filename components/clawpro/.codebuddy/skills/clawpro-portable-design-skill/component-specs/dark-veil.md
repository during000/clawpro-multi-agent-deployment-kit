# DarkVeil

> Admin 端**装饰性动态背景**（WebGL CPPN 流动纹理）的完整视觉、行为与跨仓兜底规范。本组件**只负责背景观感**，不承载任何信息 / 交互 / 可点击元素。命中场景见 §0 Auto-Trigger；它的宿主页面骨架（hero 区 + 核心能力卡）见 `references/admin-cloud-dev-activation.md`，hero 配方见 `references/page-recipes.md`「云开发开通页 hero」。

## 0. Auto-Trigger（先判要不要用，再谈怎么用）

> **DarkVeil 是「锦上添花」而非「页面默认」。** 普通列表 / 表单 / 详情 / Dashboard / 设置页**不要**无脑加动态背景——白底 + `--page-bg` 浅蓝雾已是管控端默认。只有命中下表才考虑 DarkVeil。

| 命中条件（需同时满足 A + B） | 说明 |
|---|---|
| **A. 场景** = 管控端**功能开通页 / 能力介绍 hero / 首次引导空态**的**顶部 hero 区**（如「开通云开发能力」） | 不是整页背景，是 `SurfaceCard` 内顶部 hero 区块的局部背景 |
| **B. 设计意图**明确要「动态流动 / 光效 / 科技感」氛围，且设计师已拍板 | 仅靠 AI 审美判断不足以引入动态背景；模糊时记 `conflict-log.md` 标 `needs-design-confirmation` |

**不命中即不要用**：
- ❌ 列表 / 表单 / 表格 / 详情 / 设置等功能页主体区。
- ❌ 整页 `body` / `AdminSidebarInset` 全屏背景（性能与可读性双输）。
- ❌ 把 DarkVeil 当内容容器（里面塞文字 / 按钮直接铺在 canvas 上而不加可读性叠层）。

---

## 1. Purpose

- 统一管控端 hero 区「动态流动背景」的实现方式、参数配方与可读性收口，避免每个开通页各写一套 WebGL / 渐变。
- 锁定 DarkVeil 的**纯装饰定位**：背景层永远 `pointer-events-none`、永远在内容层之下（`z-10` 内容压在其上）、永远配「基底色 + 顶部蒙版 + 底部收束叠层」三件套保证文字可读。
- 给宿主仓三档可执行兜底（L0/L1/L2），即使没有 ogl / WebGL 也能拿到视觉一致的静态版本，而不是直接白板。

### 1.1 真相源优先级（必读！生成前强制遵循）

> **本 spec 的文本描述是「设计意图 + 约束 + 参数配方」层，不是逐字节 DOM 生成指令。** 做 1:1 还原时，**必须以实际组件代码 + 宿主页面用法为准**，spec 文本仅用于验证「有没有遗漏规则」。

当 spec 文本与实际资产冲突时，可信度排序：

| 优先级 | 来源 | 用途 |
|---|---|---|
| **P0** | `client/src/components/ui/DarkVeil.tsx` 组件源码 | Props 默认值、shader uniform、canvas 铺满逻辑、resize / 清理时机、`tintColor` → 「白底 + 单色纹理」算法 |
| **P1** | `client/src/pages/admin/CloudDevActivation.tsx` hero 实际用法 | 真实参数配方（speed/warp/noise/tint）、基底色、蒙版 / 收束叠层、`translateY` 偏移、z 层级 |
| **P2** | 本 spec §3 视觉标准 / §4 配方 | 设计意图、约束边界、可读性三件套、Do/Don't |
| **✗ 禁止** | 凭 spec 文本猜 shader 内部 / 猜 ogl API / 猜参数 | — |

**硬规则**：
- 还原 DarkVeil hero 前，**先读 `DarkVeil.tsx` 确认 Props 与默认值，再读 `CloudDevActivation.tsx` 抄 hero 的参数配方与三件套叠层**，最后才参考本 spec 补约束。
- 组件已写死的 shader / uniform / 清理逻辑 → **直接复用整文件**，不要凭 spec 文本「精简」或重写 shader。
- 参数配方（speed/warp/noise/tint/translateY/mask）→ 以 `CloudDevActivation.tsx` 为准，本 spec 数值若与代码不符**以代码为准并回写 spec**。

---

## 2. Scope

- 适用端：**仅 Admin**（管控端 hero 区）。Tenant 端装饰背景走 `references/tenant.md`（白底 + 极淡蓝雾，已移除点阵装饰，**不引入 DarkVeil**）。
- 必用场景：命中 §0 Auto-Trigger 的管控端开通页 / 能力 hero。
- 不适用场景：列表 / 表单 / 详情 / Dashboard / 设置主体区、整页背景、Tenant / Landing 页。
- 禁止：
  - 在 DarkVeil 上直接铺正文 / 按钮而不加可读性叠层（基底 + 蒙版 + 收束）。
  - 给 DarkVeil 绑定点击 / hover 交互（它永远 `pointer-events-none`）。
  - 在一个页面叠多个 DarkVeil 实例（WebGL 上下文昂贵，单页最多 1 个）。

---

## 3. Visual Standard

### 3.1 组件定位与层级（hero 区三层结构）

> hero 区是 `SurfaceCard`（`overflow-hidden`）内的一个 `relative overflow-hidden` 容器，自下而上**严格三层 + 内容层**：

| 层 | 角色 | 关键写法 |
|---|---|---|
| 第 0 层（最底） | **统一基底色** | `pointer-events-none absolute inset-0 bg-[#E0EBFE]`，整体均匀，避免局部突兀、避免露白 |
| 第 1 层 | **DarkVeil canvas** | `pointer-events-none absolute inset-0 h-full w-full`，参数见 §4；顶部用 `maskImage` 淡出 → 露出的是基底色而非白 |
| 第 2 层 | **柔化收束叠层** | `pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]`，中部微提亮、底部收束回基底色，保证下方与卡片内容无缝衔接 + 文字可读 |
| 内容层 | 文字 / 按钮 / 卡片 | `relative z-10`，永远压在背景三层之上 |

### 3.2 颜色配方（管控云开发 hero 当前口径）

| Item | Value | Notes |
|---|---|---|
| 基底色 | `#E0EBFE` | 蓝紫浅底，第 0 层 + 第 2 层底部收束同色，无缝 |
| 纹理着色 `tintColor` | `#B2C3FF` | 传入后 DarkVeil 输出「白底 + 该色单色流动纹理」（见 §5 算法） |
| 中部提亮 | `white/10`（10% 白） | 第 2 层 `via-white/10`，**弱提亮**，过强会突兀 |
| 顶部淡出蒙版 | `linear-gradient(to bottom, transparent 0%, #000 22%)` | 顶部 22% 渐隐 → 露出基底色，飘带从中下部浮现 |

> **hex 说明**：`#E0EBFE` / `#B2C3FF` 是该 hero 的**装饰配方常量**（同 component-spec 视觉标准惯例，参考 `admin-sidebar.md` 引用 `#EAEEF4` 等）。换肤 / 跨仓时优先映射到宿主仓的浅蓝雾 token；无对应 token 时保留这两个常量并在 PR 标注。

### 3.3 纹理几何（避免拉伸 / 露白）

| Item | Value | Notes |
|---|---|---|
| canvas 尺寸 | `width:100%; height:100%`（铺满父容器） | 由组件内联 style 固定，**不要外部拉伸** |
| 纹理纵向偏移 | hero 用 `transform: translateY(72px)` | 让飘带整体下移到 hero 中下部；canvas 仍铺满，不会露白边 |
| 纹理图案内偏移 | Props `offsetY`（着色器内偏移） | 与外层 `translateY` 区别：`offsetY` 在 shader 内移图案、canvas 不动；hero 当前用外层 `translateY`，`offsetY` 保持默认 0 |
| DPR | `Math.min(devicePixelRatio, 2)` | 组件内已封顶 2，**不要调高**（性能） |

---

## 4. Recipe（hero 参数配方 —— 抄 `CloudDevActivation.tsx`）

> 这是「开通云开发能力」hero 的**已拍板配方**。复用同类开通页 hero 时直接抄；要改氛围强弱，只动 speed / warpAmount / noiseIntensity，**不要乱改三件套结构**。

```tsx
{/* hero 容器：SurfaceCard(overflow-hidden) 内 */}
<div className="relative overflow-hidden px-[60px] py-12">
  {/* 第 0 层：统一基底 */}
  <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />

  {/* 第 1 层：DarkVeil 动态背景 */}
  <DarkVeil
    speed={1.1}
    warpAmount={1.1}
    noiseIntensity={0.05}
    tintColor="#B2C3FF"
    className="pointer-events-none absolute inset-0 h-full w-full"
    style={{
      opacity: 1,
      transform: "translateY(72px)",
      maskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
      WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
    }}
  />

  {/* 第 2 层：柔化收束叠层 */}
  <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />

  {/* 内容层：永远 relative z-10 */}
  <div className="relative z-10 ...">{/* 图标 + 标题 + 描述 + 按钮 + 权益卡 */}</div>
</div>
```

| Prop | hero 值 | 含义 / 调节建议 |
|---|---|---|
| `speed` | `1.1` | 流动速度。氛围更安静→调低（0.5 默认）；更活跃→略升，不建议 >1.5 |
| `warpAmount` | `1.1` | 扭曲强度。决定飘带弯曲感 |
| `noiseIntensity` | `0.05` | 颗粒噪点。极弱即可，过高显脏 |
| `tintColor` | `"#B2C3FF"` | 着色 → 白底单色纹理；不传则输出原始彩色 CPPN（**hero 必须传**以贴合蓝紫基底） |
| `scanlineIntensity` / `scanlineFrequency` | 默认 0 | hero **不用**扫描线 |
| `hueShift` / `resolutionScale` / `offsetY` | 默认 | hero 不动 |

---

## 5. Props API & 关键算法

```ts
export interface DarkVeilProps {
  hueShift?: number;          // 默认 0，色相旋转（YIQ 空间）
  noiseIntensity?: number;    // 默认 0
  scanlineIntensity?: number; // 默认 0
  speed?: number;             // 默认 0.5
  scanlineFrequency?: number; // 默认 0
  warpAmount?: number;        // 默认 0
  resolutionScale?: number;   // 默认 1
  offsetY?: number;           // 默认 0，纹理图案纵向偏移（shader 内，canvas 不动、不露白）
  tintColor?: string;         // hex，如 "#B2C3FF" → 输出「白底 + 该色单色流动纹理」
  className?: string;
  style?: CSSProperties;
}
```

- **着色算法（`tintColor` 传入时）**：以纹理亮度为强度，在「白底」与目标色之间插值（`mix(vec3(1.0), uTint, intensity)`），并做 `intensity = pow(clamp(intensity*2.4,0,1), 0.75)` 增益 + gamma，让飘带更聚拢、与白底对比更强 → 得到「白底 + 单色流动纹理」。
- **canvas 铺满**：组件内联 `style={{ width:"100%", height:"100%", display:"block" }}`，配合父容器 `relative`，自动铺满；外部只透传 `className` / `style`（做 mask / transform / opacity）。
- **资源释放**：`useEffect` cleanup 内 `cancelAnimationFrame` + `geometry.remove()` + `program.remove()` + `WEBGL_lose_context().loseContext()`，避免页面切换 / 重渲染时 WebGL 上下文泄漏。**复制组件时不要删这段清理。**
- **props 变化重建**：所有 props 在依赖数组里，变化会重建 Program；仅 `uTime` 逐帧更新。

---

## 6. Demo Repo Usage

- 组件实现：`client/src/components/ui/DarkVeil.tsx`（默认导出 `DarkVeil`）。
- 依赖：`ogl`（WebGL 渲染库）。宿主仓需 `npm i ogl`，详见 §9 L0。
- hero 用法：`client/src/pages/admin/CloudDevActivation.tsx`（搜 `<DarkVeil`）。
- 设计系统展示：`client/src/pages/DesignSystemComponents.tsx`（基础视觉 / Global 分类，搜 "DarkVeil"）。

最小用法（任意需要动态背景的容器）：

```tsx
import DarkVeil from "@/components/ui/DarkVeil";

<div className="relative overflow-hidden">
  <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />
  <DarkVeil tintColor="#B2C3FF" speed={1.1} warpAmount={1.1}
    className="pointer-events-none absolute inset-0 h-full w-full" />
  <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />
  <div className="relative z-10">{/* 内容 */}</div>
</div>
```

---

## 7. States & Behavior

- DarkVeil **无交互态**（纯背景，`pointer-events-none`）。
- 唯一「态」是渲染环境差异，决定走哪档兜底：

| 环境 | 行为 | 兜底档 |
|---|---|---|
| 宿主仓可装 `ogl` + 浏览器支持 WebGL | 复制组件，1:1 动态背景 | **L0** |
| 不便引 ogl / WebGL，但要保留蓝紫流动观感 | 纯 CSS 渐变光晕（可选 CSS 动画） | **L1** |
| 极简 / 禁脚本 / 静态导出 / 低端设备 | 纯色基底或一张静态截图 | **L2** |

- **降级触发**：`prefers-reduced-motion: reduce` 时，L0/L1 应停掉动画或直接降到 L2 静态，尊重用户偏好。

---

## 8. Accessibility

- DarkVeil 是纯装饰：canvas **不放可聚焦元素**，`pointer-events-none`，对屏幕阅读器透明（无需 `alt`，因为它不是 `<img>`；如用 L2 截图 PNG 兜底，给 `alt="" aria-hidden="true"`）。
- 文字可读性由「基底 + 蒙版 + 收束叠层」三件套保证；内容文字仍走 Typography 语义色，**不要因为背景花哨而调低正文对比度**。
- 尊重 `prefers-reduced-motion`（见 §7）。

---

## 9. Portable Fallback（三档：L0 完整 / L1 静态 CSS / L2 纯色·截图）

> **跨仓换皮 / 宿主仓接入时按此分档。** 三档是「同一视觉的不同保真度」，不是三个独立组件。`references/migration-map.md` 把「DarkVeil 动态背景」标为 **L1**（默认兜底档）——即宿主仓多数情况下最少应做到 L1 静态渐变，能装 ogl 则上 L0，环境极简才退 L2。

### 9.0 档位总览

| 档 | 名称 | 依赖 | 保真度 | 何时用 |
|---|---|---|---|---|
| **L0** | 完整移植（首选） | `ogl` + WebGL | 1:1 动态 | 宿主仓能装 ogl 且支持 WebGL |
| **L1** | 静态 CSS 兜底（migration-map 默认档） | 纯 CSS | 静态/轻动画，神似 | 不便引 ogl / WebGL，但要保留蓝紫流动观感 |
| **L2** | 纯色 / 截图兜底（最低） | 零脚本 | 纯色或单图 | 禁脚本 / 静态导出 / 低端 / `reduced-motion` |

### 9.1 L0 完整移植（首选）

1. `npm i ogl`（唯一新依赖）。
2. 直接复制 `client/src/components/ui/DarkVeil.tsx` 整文件（含 shader、resize、清理逻辑），**不要改 shader**。
3. 按 §4 配方在 hero 容器里铺三层 + 内容层。
- 跨仓注入要点：组件零业务依赖（仅 `react` + `ogl`），不依赖 `@/lib/utils` / shadcn / token 体系；`tintColor` / 基底色按宿主仓蓝雾 token 映射即可。

### 9.2 L1 静态 CSS 兜底（无 ogl / WebGL）

> 用纯 CSS 在基底色上叠几层径向渐变光晕，模拟 DarkVeil 的「白底 + 蓝紫飘带」；可选一段极慢的 CSS 动画做轻微流动。**零 WebGL、零依赖。**

```html
<div class="cp-darkveil-hero">
  <div class="cp-darkveil-hero__bg" aria-hidden="true"></div>
  <div class="cp-darkveil-hero__content"><!-- 图标 + 标题 + 描述 + 按钮 + 权益卡 --></div>
</div>
```

```css
:root {
  --cp-darkveil-base: #E0EBFE;   /* 基底色，映射宿主仓浅蓝雾 token */
  --cp-darkveil-tint: #B2C3FF;   /* 飘带色 */
}

.cp-darkveil-hero { position: relative; overflow: hidden; padding: 48px 60px; }

/* 第 0+1+2 层合一：基底 + 径向光晕飘带 + 顶部淡出 + 底部收束 */
.cp-darkveil-hero__bg {
  position: absolute; inset: 0; pointer-events: none;
  background:
    radial-gradient(120% 80% at 30% 120%, var(--cp-darkveil-tint) 0%, transparent 55%),
    radial-gradient(90% 70% at 75% 130%, color-mix(in srgb, var(--cp-darkveil-tint) 70%, #fff) 0%, transparent 50%),
    var(--cp-darkveil-base);
  /* 顶部淡出露基底（对齐 L0 的 mask 22%） */
  -webkit-mask-image: linear-gradient(to bottom, transparent 0%, #000 22%);
          mask-image: linear-gradient(to bottom, transparent 0%, #000 22%);
}
/* 底部收束叠层（对齐 L0 第 2 层）——单独一层避免被上面的 mask 吃掉 */
.cp-darkveil-hero::after {
  content: ""; position: absolute; inset: 0; pointer-events: none;
  background: linear-gradient(to bottom, transparent, rgba(255,255,255,.1), var(--cp-darkveil-base));
}
.cp-darkveil-hero__content { position: relative; z-index: 10; }

/* 可选：极慢流动；尊重 reduced-motion */
@media (prefers-reduced-motion: no-preference) {
  .cp-darkveil-hero__bg { animation: cp-darkveil-drift 18s ease-in-out infinite alternate; }
}
@keyframes cp-darkveil-drift {
  from { background-position: 0 0, 0 0, 0 0; }
  to   { background-position: 6% 4%, -5% 3%, 0 0; }
}
```

可选 React 包装（无 ogl 时的等价组件，保持同一 className 语义）：

```tsx
export function DarkVeilStaticFallback({ children }: { children?: React.ReactNode }) {
  return (
    <div className="cp-darkveil-hero">
      <div className="cp-darkveil-hero__bg" aria-hidden="true" />
      <div className="cp-darkveil-hero__content">{children}</div>
    </div>
  );
}
```

### 9.3 L2 纯色 / 截图兜底（最低）

- **纯色**：直接给 hero 容器 `background: var(--cp-darkveil-base, #E0EBFE)`，零脚本零动画，保证不空白。
- **截图**：用已导出的 hero 静态图 `admin-cloud-dev-activation.png`（见 `assets/page-references/`），`<img src=... alt="" aria-hidden="true">` 铺底 + 内容层 `z-10`。
- 适用：禁脚本环境、静态导出、低端设备、`prefers-reduced-motion`。**L2 不追求流动感，只保证版式与配色不崩。**

---

## 10. Migration Rules

- 旧写法：每个开通页自己写一套 canvas / 渐变背景，参数 / 基底色 / 蒙版各不相同。
- 新口径：命中 §0 → 统一走 DarkVeil（L0）或其 L1/L2 兜底；hero 三层结构 + 内容 `z-10` 固定不变。
- `references/migration-map.md`：「DarkVeil 动态背景」归 **L1** 默认兜底档（宿主仓最少做到静态 CSS；能装 ogl 升 L0）。
- 不把 DarkVeil 扩散到列表 / 表单 / 整页背景（见 §0 / §2 禁止项）。
- 单页最多 1 个 WebGL 实例；多 hero 复用时优先 L1 静态。

---

## 11. Do / Don't

Do:
- 命中 §0 才用；hero 区局部背景，不是整页。
- 永远铺「基底 + DarkVeil + 收束叠层」三件套，内容层 `relative z-10`。
- `tintColor` 贴合基底色（蓝紫系），背景层 `pointer-events-none`。
- 宿主仓按 L0/L1/L2 分档兜底，至少做到 L1。
- 尊重 `prefers-reduced-motion`，必要时降到 L2。

Don't:
- 不要在列表 / 表单 / 详情 / 设置 / Dashboard / 整页背景滥用动态背景。
- 不要把文字 / 按钮直接铺在 canvas 上而不加可读性叠层。
- 不要改 `DarkVeil.tsx` 的 shader / 清理逻辑去「精简」。
- 不要在一页叠多个 DarkVeil（WebGL 上下文昂贵）。
- 不要把 DarkVeil 用到 Tenant / Landing（Tenant 走白底 + 极淡蓝雾）。
- 不要调高 DPR（组件已封顶 2）或把 speed 设到 >1.5 显得躁动。

---

## 12. QA Checklist

- [ ] 命中 §0 Auto-Trigger（开通页 / 能力 hero + 设计师拍板），不是给普通功能页硬加。
- [ ] hero 三层结构齐全：基底 `#E0EBFE` + DarkVeil + 收束叠层 `via-white/10 to-[#E0EBFE]`，内容 `relative z-10`。
- [ ] 背景层全部 `pointer-events-none`，无可点击 / 可聚焦元素落在 canvas 上。
- [ ] 参数对齐配方（speed 1.1 / warp 1.1 / noise 0.05 / tint #B2C3FF），顶部 mask 22% 淡出露基底。
- [ ] canvas 铺满父容器、不拉伸；纵向偏移用外层 `translateY` 而非破坏比例。
- [ ] 单页仅 1 个 DarkVeil 实例；组件清理逻辑（loseContext）未被删。
- [ ] 宿主仓无 ogl/WebGL 时已落 L1 静态 CSS（或 L2），不是直接空白。
- [ ] `prefers-reduced-motion` 下动画停止或降级 L2。
- [ ] 正文对比度未因背景被削弱，文字走 Typography 语义色。

---

## 13. References

- Demo code: `client/src/components/ui/DarkVeil.tsx`（含 ogl shader / 清理逻辑）
- Demo usage: `client/src/pages/admin/CloudDevActivation.tsx`（hero 配方）
- Demo page: `client/src/pages/DesignSystemComponents.tsx`（基础视觉 / Global 分类）
- 宿主页面骨架：`references/admin-cloud-dev-activation.md`
- hero 配方：`references/page-recipes.md`「云开发开通页 hero」
- 跨仓兜底分档：`references/migration-map.md`（DarkVeil → L1）
- 决策溯源：`references/conflict-log.md` C-018 / C-019
- 静态兜底样例：`portable/css/dark-veil.css`、`portable/html-css/dark-veil.html`
