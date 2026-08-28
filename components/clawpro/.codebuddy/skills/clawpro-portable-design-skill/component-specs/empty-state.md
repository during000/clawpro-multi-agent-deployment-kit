# Empty State

## 1. Purpose

- 统一"暂无数据"的分层写法，防止每个页面各搞一套。
- 覆盖所有场景：页面级、卡片内、表格内、弹窗/抽屉内、下拉面板内、行内字段。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：列表无数据、初次引导、筛选无结果、弹窗子区块无内容
- 不适用场景：行内字段缺值（直接用弱文字 `—`）、tooltip 极小容器

## 3. Visual Standard

### 3.1 视觉令牌

| Item | Admin 默认 | Tenant 覆写 |
|---|---|---|
| 边框 | `border-dashed border-[var(--cp-border)]`，业务统一加 `border-0` 去掉默认虚线 | 同左 |
| 圆角 | `4px` | 外层换 Tenant 卡片容器（12px 圆角） |
| Padding | 卡片内 `py-12` / 独立区域 `py-20` / 表格嵌入 `py-10` | 默认 `py-16`（更松） |
| 插画区 | `100px × 80px`，`object-contain` | 同左 |
| 标题（双行时） | 14px Medium `var(--cp-text-title)` | 同左 |
| 描述 / 单行文字 | 12px `var(--cp-text-weak)` | 同左 |
| 主操作按钮 | 宿主仓默认按钮变体 | Tenant：线框胶囊按钮，禁用实心 |
| 按钮数量上限 | ≤ 2 | ≤ 2，全部线框、无主次差异 |

## 4. Anatomy

```text
Empty
  EmptyHeader
    EmptyMedia (optional — 仅页面级/卡片级使用)
    EmptyTitle (optional — 双行时使用)
    EmptyDescription (单行 or 双行的描述行)
  EmptyContent (optional — 操作按钮区)
```

## 5. 容器场景速查（强制）

按空态所在的「容器类型」直接对照选写法，不需要自行判断。

| 容器类型 | 空态写法 | 关键约束 |
|---|---|---|
| **页面 / 大区域** | 插画 + 标题 + 描述（+ 操作按钮） | `border-0` + `py-12 ~ py-20` |
| **卡片容器** | 同页面级 | 加 `border-0`，避免与卡片本身边框重叠 |
| **表格（所有场景）** | `<td colspan>` 内放纯文字，默认双行 | **不用插画**；不要硬编码粗黑标题覆盖文字样式 |
| **Drawer 主内容区** | 插画 + 标题 + 描述 | 仅 Drawer 第一层内容为列表/卡片时 |
| **Drawer 嵌套子模块** | 纯文字单行/双行 | 层级 ≥ 2 降级，避免插画过重 |
| **Dialog / 弹窗内嵌区块** | 纯文字双行 + `space-y-1` | **禁用**所有装饰图标 / 插画 |
| **Dropdown / Select 下拉** | 纯文字单行 | 面板内 `px-3 py-6` 居中 |
| **SearchableSelect / 搜索下拉**（旧名 Combobox） | 纯文字单行 + 可选品牌色链接 | 如「+ 新建 XX」入口 |
| **Popover 内容** | 纯文字单行 | 面板宽度自适应 |
| **侧栏 / 树筛选无结果** | 插画 + 描述 | `border-0` + 紧凑 padding |
| **字段值 / 行内缺值** | 弱色文字 `—` | 不另起行不加图 |

## 6. 单行 vs 双行规则（强制）

| 文案长度 | 用法 | 视觉效果 |
|---|---|---|
| **双行**：能写出标题概括 + 描述细化 | 标题 + 描述两行 | 标题 14px 黑 + 描述 12px 灰 |
| **单行**：只有一句极短文案 | 只用描述行 | 12px 灰，**不要**用粗黑标题 |

> 判断口诀：能写出有层级感的两行 → 用双行；只能写一句话 → 单行只用描述。  
> 禁止：单行场景用粗黑标题，会让视觉过重、喧宾夺主。

## 7. States

- **initial-empty**: 第一次进入，无任何数据。
- **filtered-empty**: 有筛选条件但无匹配结果。
- **permission-empty**: 无权限查看或未配置前置条件。
- **embedded-empty**: 弹窗/抽屉/浮层内嵌空态。

## 8. Demo Repo Usage

如果宿主仓有 demo 仓的 `Empty` 组件，可直接使用：

```tsx
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from "@/components/ui/empty";

// 双行 + 操作引导
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyTitle>还没有创建任何 Agent</EmptyTitle>
    <EmptyDescription>创建你的第一个 Agent，开始自动化工作流</EmptyDescription>
  </EmptyHeader>
  <EmptyContent>
    <Button>新建 Agent</Button>
  </EmptyContent>
</Empty>

// 单行（文案极短）
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyDescription>暂无记录</EmptyDescription>
  </EmptyHeader>
</Empty>
```

## 9. Portable Fallback

### 9.1 React fallback（页面级 / 卡片级）

```tsx
function PortableEmpty({ title, description, action }: {
  title?: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-12 text-center">
      {/* 插画区：100×80，宿主仓自行替换为对应空态插画 */}
      <img
        src="/assets/empty-placeholder.png"
        alt=""
        className="h-20 w-[100px] object-contain"
      />
      <div className="flex max-w-sm flex-col items-center gap-1">
        {title && (
          <p className="text-sm font-medium text-[var(--cp-text-title)]">{title}</p>
        )}
        <p className="text-xs text-[var(--cp-text-weak)]">{description}</p>
      </div>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
```

### 9.2 React fallback（表格空态）

```tsx
// 表格空态：不用插画，放在 <td colSpan> 内
function PortableTableEmpty({ line1, line2 }: { line1: string; line2?: string }) {
  return (
    <div className="text-center py-12 space-y-1">
      <p className="text-xs text-[var(--cp-text-weak)]">{line1}</p>
      {line2 && <p className="text-xs text-[var(--cp-text-weak)]">{line2}</p>}
    </div>
  );
}
```

### 9.3 React fallback（浮层 / 弹窗 / Dropdown / Popover）

```tsx
// 浮层空态：不用插画，纯文字
function PortableOverlayEmpty({ text }: { text: string }) {
  return (
    <div className="text-center py-6">
      <p className="text-xs text-[var(--cp-text-weak)]">{text}</p>
    </div>
  );
}
```

### 9.4 HTML/CSS fallback

见 `portable/html-css/empty-state.html`。

### 9.5 If host repo already has Empty component

- 可继续使用宿主仓已有 Empty 组件。
- 但必须按 §5 容器场景速查表分层：页面级用插画，表格级不用插画，浮层用纯文字。
- 不要一种模板通吃所有场景。

## 10. Migration Rules

### 旧写法 → 新写法

| 旧页面常见写法 | 应迁移到 |
|---|---|
| `<div className="text-center py-24"><Bot className="w-12 h-12 text-gray-200" /><p className="text-gray-400">暂无数据</p></div>` | 按容器场景速查选正确层级 |
| 表格内放整个 Empty + 插画组件 | 表格空态改纯文字双行 |
| Dialog/Popover/Drawer 子区块放大图标 | 降级为纯文字 |
| 用 emoji 当空状态图标 | 用统一插画或纯文字 |
| 单行文案用粗黑标题 | 改为描述行（12px 灰） |

### 不允许新增

- Dialog/Drawer 内嵌区块继续使用大插画。
- 表格空态在 `<td>` 外另包一层大卡片。
- 用 lucide 图标 + 灰色方块容器充当空态。
- 单行场景用粗黑标题。
- Tenant 端用实心主按钮（应用线框按钮）。

## 11. Do / Don't

**Do:**

- 先判断容器类型，再选对应空态层级。
- 页面级空态写清楚对象名称和下一步操作引导。
- 表格空态放回表格结构内（`<td colSpan>`）。
- 双行时标题概括 + 描述细化。
- 单行时只用描述行（12px 灰）。
- 插画使用统一资产，业务侧零参数调用。
- 使用插画时去掉默认虚线边框（`border-0`），避免双重视觉重心。

**Don't:**

- 不要只写"暂无数据"三个字。
- 不要在表格、弹窗、下拉等小容器里塞插画。
- 不要用 emoji / lucide 图标充当空态插画。
- 不要在宿主仓里用 `<img>` 手写路径覆盖统一插画。
- 不要覆盖空态文字的字号字色（交给 token 统一管控）。
- 不要给空态按钮加超过 2 个。
- Tenant 端不要用实心主按钮，统一线框。

## 12. QA Checklist

- [ ] 空态层级与容器类型匹配（参照 §5 速查表）
- [ ] 页面级有具体标题和描述，不是只写"暂无数据"
- [ ] 表格空态没有使用插画，没有另包大卡片
- [ ] Dialog/Popover/Dropdown 空态是纯文字，没有装饰图标
- [ ] 单行场景没有使用粗黑标题
- [ ] 使用插画时已去掉默认虚线边框
- [ ] 按钮数量 ≤ 2
- [ ] Tenant 端空态按钮使用线框变体
- [ ] fallback 使用 `var(--cp-*)` CSS variable，不散写 hex
- [ ] 跨仓 fallback 可独立落地

## 13. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related tokens: `tokens/colors.md` (`--cp-text-title`, `--cp-text-weak`, `--cp-border`)
- Related recipes: `references/page-recipes.md`
- Tenant overrides: `references/tenant.md §6`

## 14. 代码对照（✅/❌）

> 与 SKILL.md §2.5 同口径。容器场景 → 写法的 5 组高频对照。

### 14.1 表格空态：不用插画

```tsx
// ❌ 表格里塞页面级 Empty（含插画），与表格视觉冲突
<tbody>
  {rows.length === 0 && (
    <tr><td colSpan={5}>
      <Empty>
        <EmptyHeader><EmptyMedia /><EmptyTitle>暂无策略</EmptyTitle></EmptyHeader>
      </Empty>
    </td></tr>
  )}
</tbody>

// ✅ 表体内 colSpan 纯文字双行（参考 table.md §9）
<tbody>
  {rows.length === 0 && (
    <tr><td colSpan={5}>
      <div className="text-center py-12 space-y-1">
        <p className="text-xs text-[var(--cp-text-weak)]">暂无策略</p>
        <p className="text-xs text-[var(--cp-text-weak)]">尝试调整筛选条件，或新建一条策略</p>
      </div>
    </td></tr>
  )}
</tbody>
```

### 14.2 双行 vs 单行：单行不要粗黑标题

```tsx
// ❌ 单行场景套上粗黑标题，视觉过重
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyTitle>暂无记录</EmptyTitle>
  </EmptyHeader>
</Empty>

// ✅ 单行只用描述行（12px 灰）
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyDescription>暂无记录</EmptyDescription>
  </EmptyHeader>
</Empty>

// ✅ 真有标题 + 细化描述 → 双行
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyTitle>还没有创建任何 Agent</EmptyTitle>
    <EmptyDescription>创建你的第一个 Agent，开始自动化工作流</EmptyDescription>
  </EmptyHeader>
</Empty>
```

### 14.3 Dialog / Popover / Dropdown：纯文字，禁装饰图标

```tsx
// ❌ 弹窗子区块塞大插画 + lucide 灰色图标
<DialogContent>
  <div className="flex flex-col items-center py-8">
    <FolderOpen className="w-12 h-12 text-gray-200" />
    <p className="text-gray-400">暂无文件</p>
  </div>
</DialogContent>

// ✅ 纯文字双行 + space-y-1
<DialogContent>
  <div className="text-center py-6 space-y-1">
    <p className="text-xs text-[var(--cp-text-weak)]">暂无文件</p>
    <p className="text-xs text-[var(--cp-text-weak)]">从左侧选择目录后查看文件列表</p>
  </div>
</DialogContent>
```

### 14.4 使用插画时去掉默认虚线边框

```tsx
// ❌ 同时出现外层卡片边框 + Empty 默认虚线，双重视觉重心
<SurfaceCard>
  <Empty className="py-12">
    <EmptyHeader>
      <EmptyMedia />
      <EmptyDescription>还没有数据</EmptyDescription>
    </EmptyHeader>
  </Empty>
</SurfaceCard>

// ✅ 显式 border-0
<SurfaceCard>
  <Empty className="border-0 py-12">
    <EmptyHeader>
      <EmptyMedia />
      <EmptyDescription>还没有数据</EmptyDescription>
    </EmptyHeader>
  </Empty>
</SurfaceCard>
```

### 14.5 emoji / lucide 图标不能充当空态插画

```tsx
// ❌ emoji 当插画
<div className="text-center py-12">
  <span className="text-5xl">📭</span>
  <p>暂无数据</p>
</div>

// ❌ lucide 图标 + 灰色方块容器冒充插画
<div className="text-center py-12">
  <div className="w-12 h-12 mx-auto bg-gray-100 rounded-md flex items-center justify-center">
    <Inbox className="w-6 h-6 text-gray-300" />
  </div>
  <p>暂无数据</p>
</div>

// ✅ 用统一 Empty 组件家族（业务侧零参数调用插画）
<Empty className="border-0 py-12">
  <EmptyHeader>
    <EmptyMedia />
    <EmptyDescription>暂无数据</EmptyDescription>
  </EmptyHeader>
</Empty>
```
