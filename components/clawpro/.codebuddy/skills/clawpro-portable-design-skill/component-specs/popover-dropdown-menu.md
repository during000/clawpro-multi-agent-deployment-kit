# Popover / Dropdown Menu

> **Showcase mapping**: `popover` · `info-popover` · `dropdown-menu` · `hover-card` · `context-menu` · `more-actions-dropdown`（`client/src/pages/DesignSystemComponents.tsx`）。Tooltip 单独走 `tooltip.md`。

## 1. Purpose

- 统一 Popover、DropdownMenu、HoverCard、Tooltip、更多操作菜单和轻量浮层的视觉、状态和迁移规则。
- 避免宿主仓在表格操作、筛选面板、字段说明、三点菜单里继续散写不同圆角、阴影、padding 和危险操作样式。

## 2. Scope

- 适用端：Admin / Tenant / Shared。
- 必用场景：更多操作菜单、用户菜单、表格列筛选、字段说明、状态说明、轻量配置面板、组合型 SearchableSelect / 筛选面板。
- 不适用场景：复杂表单、跨页对象选择、批量危险操作确认；这些应使用 Dialog / Drawer / AlertDialog。
- Tooltip 只承载短说明；长说明、可点击内容、字段说明优先用 HoverCard / Popover。

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| Surface | 白底浮层 | 不使用彩色大面板 |
| Radius | 4px | demo 仓部分 8px 属历史兼容，不作为新增目标 |
| Border | `--border` / `--cp-border` | 蓝灰描边 |
| Shadow | `--cp-shadow-overlay` / overlay 层级 | 不新增临时重阴影 |
| z-index | overlay 层，通常 `z-50` | Toast 另有更高层级 |
| Portal | 必须支持 | 避免被滚动容器裁剪 |
| Trigger Height | 通常 36px / `h-9` | 与同排控件对齐 |
| Menu Item | `text-sm`、紧凑 padding | 更多操作建议带 16px icon |
| Divider | 1px 弱分割线 | 用于分隔危险操作或分组 |
| Tooltip | 黑底白字、4px、12px | 只放短说明 |
| Empty | 单行弱提示 | 不使用页面级 Empty 插画 |

## 4. Anatomy

```text
Trigger
Overlay
  Header optional
  Search optional
  Item / Option * n
  Separator optional
  Footer optional
```

## 5. States

- closed: 默认关闭。
- open: trigger 可显示品牌蓝边框或 icon 旋转。
- hover: trigger 和 item 有轻微背景变化。
- focus-visible: 键盘可达。
- selected / checked: 有 Check、品牌蓝弱底或品牌色文本。
- disabled: 不可点击，文字和图标降级。
- destructive: 危险菜单项用 danger 语义，最终动作仍需二次确认。
- loading: 菜单项可展示 spinner 或 disabled，避免重复触发。
- empty: 无选项时显示单行弱提示。
- overflow: 内容过多时面板内部滚动。
- confirmable: 多选 / 树形 Popover 使用临时值 + footer 确认 / 取消。

## 6. Demo Repo Usage

- Popover：`client/src/components/ui/popover.tsx`
- DropdownMenu：`client/src/components/ui/dropdown-menu.tsx`
- HoverCard：`client/src/components/ui/hover-card.tsx`
- Tooltip：`client/src/components/ui/tooltip.tsx`
- InfoPopover：`client/src/components/ui/info-popover.tsx`
- MoreActionsDropdown：`client/src/components/ui/more-actions-dropdown.tsx`
- 典型组合：`client/src/components/ui/select.tsx` 内的 `SearchableSelect`（旧名 Combobox / OpenClawCombobox 已废弃，未在仓库存在）、`client/src/components/ScopeEditPopover.tsx`

## 7. Portable Fallback

### 7.1 If host repo already has overlay components

- 保留宿主仓已有 Popover / Dropdown / Tooltip 逻辑。
- 覆盖视觉：4px、蓝灰描边、overlay shadow、白底、紧凑 item、Portal。
- 长内容不要塞 Tooltip；改用 Popover / HoverCard。
- 危险菜单项只表达入口，真正执行前进入确认弹窗。

### 7.2 Minimal React fallback

```tsx
import * as React from "react";

export function PortableDropdownMenu() {
  const [open, setOpen] = React.useState(false);

  return (
    <div className="relative inline-flex">
      <button type="button" onClick={() => setOpen((v) => !v)} className="h-9 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-title)]">更多</button>
      {open && (
        <div className="absolute right-0 top-11 z-50 min-w-[160px] rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-1 shadow-[var(--cp-shadow-overlay)]">
          <button className="flex h-8 w-full items-center rounded-[4px] px-2 text-left text-sm text-[var(--cp-text-title)] hover:bg-[var(--cp-bg-subtle)]">编辑</button>
          <button className="flex h-8 w-full items-center rounded-[4px] px-2 text-left text-sm text-[var(--cp-text-title)] hover:bg-[var(--cp-bg-subtle)]">复制</button>
          <div className="my-1 h-px bg-[var(--cp-border)]" />
          <button className="flex h-8 w-full items-center rounded-[4px] px-2 text-left text-sm text-[var(--cp-text-danger)] hover:bg-[var(--cp-bg-subtle)]">删除</button>
        </div>
      )}
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-menu">
  <button class="cp-menu-trigger">更多</button>
  <div class="cp-menu-panel">
    <button>编辑</button>
    <button>复制</button>
    <hr />
    <button class="danger">删除</button>
  </div>
</div>
```

```css
.cp-menu { position: relative; display: inline-flex; }
.cp-menu-trigger { height: 36px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 0 12px; font-size: 14px; color: var(--cp-text-title); }
.cp-menu-panel { position: absolute; right: 0; top: 44px; z-index: 50; min-width: 160px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 4px; box-shadow: var(--cp-shadow-overlay); }
.cp-menu-panel button { display: flex; width: 100%; height: 32px; align-items: center; border: 0; border-radius: 4px; background: transparent; padding: 0 8px; text-align: left; font-size: 14px; color: var(--cp-text-title); }
.cp-menu-panel button:hover { background: var(--cp-bg-subtle); }
.cp-menu-panel .danger { color: var(--cp-text-danger); }
.cp-menu-panel hr { height: 1px; border: 0; background: var(--cp-border); }
```

## 8. Migration Rules

- 旧写法：页面内绝对定位浮层，手写 `shadow-lg`、`rounded-lg`、不同 padding。
- 新口径：统一 overlay 层级，默认 4px、蓝灰描边、轻阴影、Portal。
- Tooltip 只用于短句；复杂说明迁移到 HoverCard / Popover。
- Dropdown 里的危险操作必须转入确认流程，不直接执行破坏性动作。
- Popover 内列表过长必须内部滚动，不能撑破页面。

## 9. Do / Don't

Do:

- 让浮层逃逸滚动容器裁剪。
- 用单行弱提示处理浮层内空态。
- 给键盘 focus 可见状态。

Don't:

- 不要用 Tooltip 承载长说明或可点击内容。
- 不要新增重阴影、大圆角浮层。
- 不要在菜单项点击后直接执行危险操作。

## 10. QA Checklist

- [ ] 浮层使用 4px、蓝灰描边、overlay shadow
- [ ] 浮层通过 Portal 或等效方式避免裁剪
- [ ] hover / focus / disabled / selected 状态完整
- [ ] Tooltip 只承载短说明
- [ ] 长说明或可点击内容使用 Popover / HoverCard
- [ ] 危险操作有二次确认
- [ ] 空态为单行弱提示
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/popover.tsx`
- Demo code: `client/src/components/ui/dropdown-menu.tsx`
- Demo code: `client/src/components/ui/hover-card.tsx`
- Demo code: `client/src/components/ui/tooltip.tsx`
- Related spec: `component-specs/dialog-drawer.md`
- Related spec: `component-specs/combobox.md`（alias 文档；实际组件为 `select.tsx` 的 `SearchableSelect`）（alias 文档；实际组件为 `select.tsx` 的 `SearchableSelect`）

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Popover / Dropdown 5 项高频误用 → ClawPro 正确写法。

### 12.1 浮层逃逸 overflow，必须用 Portal

```tsx
// ❌ 直接 absolute 摆在表格行内，被表格 overflow 裁掉
<tr className="overflow-hidden">
  <td>
    <div className="relative">
      <button>更多</button>
      {open && (
        <div className="absolute right-0 top-full bg-white shadow rounded">
          <button>编辑</button>
        </div>
      )}
    </div>
  </td>
</tr>

// ✅ DropdownMenu 内置 Portal，自动逃逸滚动容器
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="ghost" size="icon"><MoreHorizontal className="h-4 w-4" /></Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent align="end">  {/* 内部 Portal 到 body */}
    <DropdownMenuItem>编辑</DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

### 12.2 危险菜单项不直接执行，必走二次确认

```tsx
// ❌ 点删除直接调 deletePolicy，没有任何确认
<DropdownMenuItem
  className="text-[var(--cp-text-danger)]"
  onSelect={() => deletePolicy(policy.id)}
>
  删除
</DropdownMenuItem>

// ✅ 菜单项只触发确认弹窗，真正动作在 AlertDialog 里
const [confirmOpen, setConfirmOpen] = useState(false);

<DropdownMenu>
  <DropdownMenuContent align="end">
    <DropdownMenuItem
      className="text-[var(--cp-text-danger)] focus:text-[var(--cp-text-danger)]"
      onSelect={() => setConfirmOpen(true)}
    >
      删除
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>

<AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>删除「{policy.name}」？</AlertDialogTitle>
      <AlertDialogDescription>操作不可恢复。</AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>取消</AlertDialogCancel>
      <AlertDialogAction onClick={() => deletePolicy(policy.id)}>确认删除</AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

### 12.3 圆角 / 阴影：4px + overlay shadow，不要 rounded-lg / shadow-2xl

```tsx
// ❌ 大圆角 + 重阴影
<DropdownMenuContent className="rounded-lg shadow-2xl border-gray-200">
  <DropdownMenuItem>编辑</DropdownMenuItem>
</DropdownMenuContent>

// ❌ 默认 popover.tsx 的 8px / w-72 / p-4 也应该被覆盖
<PopoverContent>...</PopoverContent>

// ✅ DropdownMenu 默认即 4px + overlay，无需自定义
<DropdownMenuContent align="end">
  <DropdownMenuItem>编辑</DropdownMenuItem>
</DropdownMenuContent>

// ✅ Popover 在使用方显式覆盖（参考 combobox.md §3 / 实际看 SearchableSelect 实现）
<PopoverContent
  className="rounded-[4px] border-[var(--cp-border)] p-0 shadow-[var(--cp-shadow-overlay)]"
>...</PopoverContent>
```

### 12.4 长内容用 Popover，不要塞 Tooltip

```tsx
// ❌ Tooltip 塞 5 行字段说明
<Tooltip>
  <TooltipTrigger asChild>
    <Info className="h-4 w-4" />
  </TooltipTrigger>
  <TooltipContent className="max-w-md">
    本字段控制 Agent 调用模型时的温度参数。0.0 时输出最确定，
    2.0 时最具创造性。建议代码生成场景用 0.2，文案场景用 0.7。
  </TooltipContent>
</Tooltip>

// ✅ 字段说明 / 长文档 → InfoPopover
<InfoPopover title="温度参数">
  <p className="text-sm leading-relaxed">
    本字段控制 Agent 调用模型时的温度参数。0.0 时输出最确定，
    2.0 时最具创造性。建议代码生成场景用 0.2，文案场景用 0.7。
  </p>
  <a href="/docs/llm-temperature" className="text-sm text-[var(--cp-text-brand)] underline">
    查看完整说明
  </a>
</InfoPopover>
```

### 12.5 列表过长：内部滚动，不撑破视口

```tsx
// ❌ Dropdown 列出 200 个 Agent，下拉撑出屏幕
<DropdownMenuContent>
  {agents.map(a => <DropdownMenuItem key={a.id}>{a.name}</DropdownMenuItem>)}
</DropdownMenuContent>

// ✅ Content 限高 + overflow，长列表场景应改用 SearchableSelect
<DropdownMenuContent className="max-h-[320px] overflow-y-auto">
  {agents.slice(0, 50).map(a => <DropdownMenuItem key={a.id}>{a.name}</DropdownMenuItem>)}
</DropdownMenuContent>

// ✅ 真正的"长列表 + 搜索"用 SearchableSelect（旧名 Combobox；spec 见 combobox.md，实现见 select.tsx）
<SearchableSelect options={agents.map(a => ({ value: a.id, label: a.name }))} value={agentId} onChange={setAgentId} placeholder="选择 Agent" />
```
