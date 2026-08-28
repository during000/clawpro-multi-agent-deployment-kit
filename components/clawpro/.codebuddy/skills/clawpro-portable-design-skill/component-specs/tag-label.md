# Tag / Label

## 1. Purpose

- 区分「用户自定义标签 Tag」和「状态标签 StatusBadge」的使用场景。
- 统一 Tag 的视觉表现（颜色、圆角、尺寸）。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：用户自建标签、分类标签、筛选 chip、资源标记
- 不适用场景：运行状态（用 `status-tag.md`）、New/Beta 角标（用 `badge.md`）

## 3. Visual Standard

### 3.1 基础 Tag

| Item | Value |
|---|---|
| 高度 | `22px` |
| 圆角 | `4px` |
| Padding | `px-2 py-0.5` |
| 字号 | `12px` / Regular |
| 背景 | `#F5F5F5` |
| 文字 | `var(--cp-text-body)` (#1E293B) |
| 边框 | 无（默认）；可选 `border border-[var(--cp-border)]` |

### 3.2 可关闭 Tag

- 右侧 `X` 按钮，`12px`，`var(--cp-text-muted)` 色
- Hover 时 X 变深

### 3.3 彩色 Tag（分类用）

从以下颜色中选取，保持低饱和：

| 色系 | 背景 | 文字 | 边框 |
|---|---|---|---|
| Blue | `#E8ECFE` | `#1447E6` | `#C7D7FE` |
| Green | `#E9F8EB` | `#008236` | `#BFE8C8` |
| Orange | `#FFFBEB` | `#B45309` | `#FDE68A` |
| Gray | `#F5F5F5` | `#0A0A0A` | `#E5E5E5` |

> 新增颜色需先在 token 层登记，禁止业务侧自拼 `bg-*-50 text-*-700`。

## 4. Portable Fallback

```tsx
function PortableTag({ children, color = 'gray', closable, onClose }: any) {
  const colors: Record<string, string> = {
    gray: 'bg-[#F5F5F5] text-[#0A0A0A]',
    blue: 'bg-[#E8ECFE] text-[#1447E6]',
    green: 'bg-[#E9F8EB] text-[#008236]',
    orange: 'bg-[#FFFBEB] text-[#B45309]',
  };
  return (
    <span className={`inline-flex items-center gap-1 h-[22px] px-2 rounded-[4px] text-xs ${colors[color]}`}>
      {children}
      {closable && (
        <button className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-body)]" onClick={onClose}>×</button>
      )}
    </span>
  );
}
```

## 5. Do / Don't

**Do:**
- Tag 用于用户标签/分类；StatusBadge 用于运行状态。
- 彩色 Tag 从预定义色板选。

**Don't:**
- 不要用 Tag 表达运行状态。
- 不要业务侧自拼颜色组合。
- 不要把 Tag 做成 `rounded-full`（那是 Badge）。

## 6. QA Checklist

- [ ] Tag 与 StatusBadge 场景未混用
- [ ] 颜色从预定义色板选取
- [ ] 圆角 `4px`（不是 full）
- [ ] 可关闭 Tag 的 X 按钮在右侧

## 7. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related specs: `component-specs/status-tag.md`、`component-specs/badge.md`

## 8. 代码对照（✅/❌）

> 与 SKILL.md §2 / status-tag.md §14 / badge.md §14 同口径。Tag / Label 5 项高频误用 → ClawPro 正确写法。

### 8.1 Tag 不要做成 rounded-full（那是 Badge 角色）

```tsx
// ❌ 自定义标签做成胶囊，与状态 Badge 混淆
<span className="inline-flex h-[22px] items-center rounded-full bg-[#F5F5F5] px-2 text-xs">
  前端
</span>

// ✅ Tag = rounded-[4px]（4px 矩形胶囊）
<Tag color="gray">前端</Tag>

// ✅ Badge 才是 rounded-full（参考 badge.md §3）
<Badge color="blue">全部用户</Badge>

// ✅ 状态语义则走 StatusTag（12px 彩色文本，无底色无圆角，参考 status-tag.md §3）
<StatusTag variant="green">运行中</StatusTag>
```

### 8.2 颜色必须从预定义 4 色板选

```tsx
// ❌ 业务侧自拼 bg-rose-50 text-rose-700
<span className="inline-flex h-[22px] rounded-[4px] bg-rose-50 text-rose-700 px-2 text-xs">
  紧急
</span>

// ❌ 引入 violet / amber 扩展色（不在预定义 4 色板）
<Tag className="bg-violet-100 text-violet-700">实验</Tag>

// ✅ 从 gray / blue / green / orange 4 色中选
<Tag color="orange">紧急</Tag>
<Tag color="blue">前端</Tag>

// ✅ 真需要新色：先在 token 层登记 + 走独立 spec PR（参考 §3.3 注释）
```

### 8.3 Tag 不替代运行状态（用 StatusTag）

```tsx
// ❌ 用 Tag 表达 "运行中" / "异常" / "已停止" 等运行状态
<Tag color="green">运行中</Tag>
<Tag color="orange">异常</Tag>

// ✅ 运行状态走 StatusTag mode="text"（14px Medium 彩色文字）
<StatusTag mode="text" variant="green">运行中</StatusTag>
<StatusTag mode="text" variant="orange">异常</StatusTag>

// ✅ Tag 用于"自定义标签 / 分类 / 资源标记"
<Tag color="blue">前端</Tag>
<Tag color="gray">v2.1.0</Tag>
```

### 8.4 可关闭 Tag 的 X 必须在右侧 + hover 变深

```tsx
// ❌ X 在左侧
<span className="inline-flex h-[22px] items-center gap-1 rounded-[4px] bg-[#F5F5F5] px-2 text-xs">
  <button>×</button>
  前端
</span>

// ❌ X 不响应 hover，永远灰色
<Tag closable>
  前端
  <X className="ml-1 h-3 w-3 text-gray-400" />
</Tag>

// ✅ X 在右侧 + var(--cp-text-muted) → hover 变深
<Tag color="gray" closable onClose={() => removeTag("前端")}>前端</Tag>
{/* 内部：
   <span className="inline-flex h-[22px] items-center gap-1 rounded-[4px] bg-[#F5F5F5] px-2 text-xs">
     前端
     <button onClick={onClose} className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-body)]">×</button>
   </span>
*/}
```

### 8.5 不要把 New / Beta 角标做成 Tag

```tsx
// ❌ New / Beta 角标用 Tag（4px 矩形）
<h3 className="flex items-center gap-2">
  记忆管理
  <Tag color="blue">New</Tag>
</h3>

// ✅ New / Beta 用 Badge（rounded-full 胶囊）
<h3 className="flex items-center gap-2">
  记忆管理
  <Badge variant="secondary" className="rounded-full px-2 py-0.5 text-[10px] uppercase">
    New
  </Badge>
</h3>

// ✅ 用户自建 / 资源标记 / 分类 → Tag
<div className="flex items-center gap-1">
  <Tag color="blue">前端</Tag>
  <Tag color="gray">v2.1.0</Tag>
  <Tag color="green">已上线</Tag>
</div>
```
