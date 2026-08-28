# 页面布局间距规范 · DESIGN_SYSTEM_PAGE_LAYOUT

> 适用于 ClawPro 管控端 / 租户端的内容页（Header + 介绍卡片网格 / 表格 / Tab 等组合）。
> **参考实现**：`client/src/pages/admin/SecurityManagement.tsx`
> 现有页面如与本规范不一致，逐步对齐到本规范，不要新增不一致的页面。

---

## 0. 设计 Token 速查

| 用途 | Token | px |
|---|---|---|
| Header（含副标题/Badge）↔ 下方主体 | `mb-8` | 32 |
| Header（仅标题）↔ 下方主体 / 内容区段间隔 | `mb-6` | 24 |
| 二级标题 ↔ 下方表格 | `mb-4` | 16 |
| Tab ↔ Tab 描述 | `mt-3` | 12 |
| Tab 描述 ↔ 下方内容 | `mb-6` | 24 |
| Header 标题 ↔ 副标题 | `mt-1` | 4 |
| 卡片网格 gap | `gap-4` | 16 |
| 卡片内 padding | `p-5` | 20 |
| 段落式表单字段间距 | `space-y-3` | 12 |

> **一句话原则**：**32 / 24 / 16 / 12 / 4** 五档。
> **核心判断**：Header 用 32 还是 24，看它**有没有副标题/Badge**——视觉块"厚"则 32，"薄"则 24。其余跨内容区一律 24，二级内聚 16。

---

## 1. 页面外层

```tsx
<div className="page-enter">
  {/* Header */}
  {/* (可选) Tab + Tab 描述 */}
  {/* 内容主体（卡片网格 / 表格 / 表单） */}
</div>
```

- 必须使用 `page-enter` 类承载入场动画与 padding。
- 不要在外层再包一层额外 `div` 加 padding/margin。

---

## 2. Header（标题区）

Header 按**视觉块厚薄**分两形态，决定底部间距：

### 2.1 B 型 Header — 仅标题（视觉块薄）→ `mb-6`

```tsx
<div className="mb-6">
  <h1 className="text-2xl font-bold text-[#0A0A0A]">页面标题</h1>
</div>
```

**何时用**：标题就是一行文字，下面直接接 Tab / 表格 / 表单（如 网络管理、技能配置、通道配置）。
**为什么是 24**：标题视觉块只有一行高，再留 32px 会让标题孤零零飘在上面，反而显得空。

### 2.2 A 型 Header — 标题 + 副标题（或 + Badge / + Bell）→ `mb-8`

```tsx
<div className="mb-8">
  <div className="flex items-center gap-2.5">
    <h1 className="text-2xl font-bold text-[#0A0A0A]">页面标题</h1>
    <Badge variant="outline">即将开放</Badge>
  </div>
  <p className="text-sm text-[#737373] mt-1">
    一句话副标题，描述本页核心价值。
  </p>
</div>
```

**何时用**：Header 至少有「标题 + 副标题」，或带 Badge / Bell 等附加元素，视觉块明显比一行字厚。
**为什么是 32**：副标题让 Header 变成一个完整段落，需要更大的"段落级"留白把它和下方主体明确隔开。

### 强约束

- Header 容器只允许 `mb-6` 或 `mb-8`，**严禁 `mb-4` / `mb-10` / `mt-16` 等其他值**。
- 选 24 还是 32，**只看 Header 有没有副标题/Badge/Bell**——薄的 24，厚的 32。
- 标题字号 `text-2xl font-bold`，字色 `#0A0A0A`。
- 标题↔Badge/Bell 间距 `gap-2.5`（10px），Badge 用 `variant="outline"`。
- 副标题 `text-sm text-[#737373] mt-1`（**不要**用 `text-[13px]`、`text-xs`、`leading-relaxed`）。


---

## 3. Tab + Tab 描述（可选）

```tsx
{/* Tab 切换器 */}
<div className="mb-1">
  <SegmentGroup>...</SegmentGroup>
</div>

{/* Tab 描述 */}
<div className="mt-3 mb-6">
  <p className="text-sm text-[#737373] leading-relaxed">{currentTab.description}</p>
</div>
```

**强约束**：
- Tab 容器 `mb-1`（4px），让描述视觉上贴在 Tab 下方而不是悬空。
- Tab 描述外容器 `mt-3 mb-6`，文字统一 `text-sm leading-relaxed text-[#737373]`。
- **不要**用 `text-[13px]`，统一到 `text-sm`。

---

## 4. 介绍卡片网格（即将开放页 / 能力介绍页）

```tsx
<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
  {features.map((feature, index) => (
    <div
      key={index}
      className="bg-white rounded-xl border border-[#e5e5e5] p-5"
    >
      <div className="flex items-start gap-4">
        <img src={feature.iconSrc} alt="" className="shrink-0" />
        <div className="flex flex-col pt-0.5">
          <h3 className="text-sm font-semibold text-[#0A0A0A]">
            {feature.title}
          </h3>
          <p className="mt-1 text-xs text-[#737373] leading-relaxed">
            {feature.description}
          </p>
        </div>
      </div>
    </div>
  ))}
</div>
```

**强约束**：
- 网格 gap 统一 `gap-4`（16px），不写 `gap-3` / `gap-5` / `gap-6`。
- 卡片：`bg-white rounded-xl border border-[#e5e5e5] p-5`。圆角 **不要** 写成 `rounded-[4px]` 或 `rounded-[8px]`。
- 图标固定 36×36 渐变 SVG，使用 `<img>` 不要用 `<div>` 套 lucide 图标（保持视觉统一）。
- 图标↔文字 `gap-4`（16px），右侧文字列加 `pt-0.5` 微调，使图标与标题视觉对齐。
- 标题 `text-sm font-semibold text-[#0A0A0A]`；描述 `mt-1 text-xs text-[#737373] leading-relaxed`。

---

## 5. 列表/表格区（参考 ChannelConfig 内置通道 Tab）

```tsx
<SurfaceCard className="overflow-hidden">
  <Table>...</Table>
</SurfaceCard>
```

- 表格容器统一用 `SurfaceCard`，不要手写 `bg-white rounded-* border-*`。
- 表格上方如有"操作行"（链接 + 主按钮），与表格之间 `space-y-3`（12px）。

---

## 6. 复合卡片（标题 + 分割线 + 内容，参考 SkillConfig SkillSourceTab）

```tsx
<div className="bg-white rounded-[4px] border border-[#e5e5e5] overflow-hidden">
  {/* 卡片标题 */}
  <div className="flex items-center gap-3.5 px-6 py-5">
    <Icon36 />
    <h2 className="text-sm font-medium text-[#020617] leading-[22px]">卡片标题</h2>
  </div>
  {/* 分割线 */}
  <div className="mx-6 h-px bg-[#eaeaea]" />
  {/* 内容区 */}
  <div className="px-6 py-5 flex flex-col gap-2">
    ...
  </div>
</div>
```

**强约束**：
- 复合卡片用 `rounded-[4px]`、`px-6 py-5`、`gap-3.5` 图标↔标题。
- 分割线 `mx-6 h-px bg-[#eaeaea]`，左右与 padding 对齐。
- 与第 4 节"介绍卡片网格"区分：介绍卡片用 `rounded-xl + p-5`；复合卡片用 `rounded-[4px] + px-6 py-5`。

---

## 7. 颜色与字号收敛清单

| 角色 | 颜色 | 字号 |
|---|---|---|
| 页面 H1 | `#0A0A0A` | `text-2xl font-bold` |
| 介绍卡片 H3 | `#0A0A0A` | `text-sm font-semibold` |
| 复合卡片 H2 | `#020617` | `text-sm font-medium` |
| 副标题 / Tab 描述 / 卡片描述 | `#737373` | 副标题 `text-sm`、卡片描述 `text-xs leading-relaxed` |
| 描边 | `#e5e5e5` | — |
| 分割线 | `#eaeaea` | — |

> 同一页面内，两个本质相同的角色不要混用 `#0A0A0A` 和 `#020617`；以本表为准。

---

## 8. 反例（请勿在新页面出现）

```tsx
// ❌ 副标题字号不统一
<p className="text-[13px] text-[#737373] leading-relaxed">

// ❌ 卡片圆角混用
<div className="rounded-[8px] border ...">

// ❌ Tab 描述使用 mt-* 但不使用 mb-6，导致与下方内容贴太近
<div className="flex items-center gap-3 mt-3">

// ❌ B 型 Header（仅标题）用 mb-8，标题孤零零显得空
<div className="mb-8">
  <h1 className="text-2xl font-bold text-[#0A0A0A]">网络管理</h1>
</div>

// ❌ A 型 Header（带副标题）用 mb-6，下方主体贴得太紧
<div className="mb-6">
  <h1 className="text-2xl font-bold text-[#0A0A0A]">Agent 类型</h1>
  <p className="text-sm text-[#737373] mt-1">通过启用镜像决定…</p>
</div>

// ❌ 用 !mt-16 这种强制的"魔法值"撑大间距，应统一回 token
<div className="!mt-16">…</div>
```

// ❌ Header 顶部多余 padding
<div className="page-enter pt-4">

// ❌ 介绍卡片用纯色块图标 + lucide 图标，与 36×36 渐变 SVG 同时出现
<div className="w-10 h-10 rounded-[8px]" style={{ background: card.color }}>
  <Icon className="w-5 h-5 text-white" />
</div>

// ❌ Tab 描述使用 mt-* 但不使用 mb-6，导致与下方内容贴太近
<div className="flex items-center gap-3 mt-3">
```

---

## 9. 落地检查清单（提交前自检）

- [ ] Header 容器是否按形态正确取值？仅标题用 `mb-6`，含副标题/Badge/Bell 用 `mb-8`。
- [ ] 副标题是否使用 `text-sm text-[#737373] mt-1`？
- [ ] Tab 区是否遵循 `mb-1` + `mt-3 mb-6` 组合？
- [ ] 卡片网格 gap 是否 `gap-4`？卡片是否 `rounded-xl + p-5`？
- [ ] 图标↔文字是否 `gap-4`？标题↔描述是否 `mt-1`？
- [ ] 标题字色是否按第 7 节表选择，未在同一页面混用？
- [ ] 是否仍有 `text-[13px]`、`rounded-[8px]`、`gap-3.5` 出现在介绍卡片中？

> 当对 SecurityManagement.tsx 做任何样式调整时，**先更新本规范**，再同步全部相关页面，避免规范被偷偷改掉。
