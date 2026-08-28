# Tree

## 1. Purpose

- 用于分组管理、文件树、目录导航等层级结构场景。
- 统一展开/收起、选中、hover、图标、缩进的视觉规范。

## 2. Scope

- 适用端：Admin 优先（Tenant 仅在明确需要树形导航时复用）
- 必用场景：用户分组管理、技能目录、文件树、网络管理树
- 不适用场景：扁平列表（用 Table）、仅两级的简单切换（用 Tabs）

## 3. Visual Standard

### 3.1 颜色规范

| 元素 | 色值 | 说明 |
|---|---|---|
| 文字（默认 & 选中） | `#09090b` | 前景色 |
| Hover / 选中背景 | `#f4f4f5` | 浅灰 |
| 箭头/图标 | `#71717a` | 中灰（统一色） |
| 计数/辅助文字 | `#a1a1aa` | 弱灰 |
| 禁用 | `#a1a1aa` + `opacity-60` | — |

### 3.2 尺寸参数

| Item | Value |
|---|---|
| 行高 | `32px` (h-8) |
| 圆角 | `4px` |
| 缩进 | `paddingLeft: 8 + depth × 16` 或 `ml-5` (20px) |
| 行间距 | `gap-1` (4px) 或 `mb-0.5` |
| 图标与文字间距 | `gap-1.5` (6px) 或 `gap-2` (8px) |
| 图标尺寸 | `16px × 16px` (w-4 h-4) |
| 箭头尺寸 | `14px × 14px` (w-3.5 h-3.5) 含在按钮中 |

### 3.3 交互状态

| 状态 | 样式 |
|---|---|
| 默认 | `text-[#09090b] bg-transparent` |
| Hover | `bg-[#f4f4f5]` |
| 选中 (Active) | `bg-[#f4f4f5] font-medium` |
| 展开箭头旋转 | `transition-transform rotate-90` |
| 禁用 | `text-[#a1a1aa] cursor-not-allowed opacity-60` |

## 4. Anatomy

```text
TreeContainer
  TreeItem (repeatable, nestable)
    ExpandArrow (ChevronRight → rotate-90 when open)
    Icon (Folder / File / custom)
    Label
    Count (optional, 弱色)
    Actions (optional, hover 显示)
  TreeItem
    Children (indented)
      TreeItem...
```

## 5. Portable Fallback

```tsx
function PortableTree({ items, activeId, onSelect }: any) {
  return (
    <div className="flex flex-col gap-0.5">
      {items.map((item: any) => (
        <TreeNode key={item.id} item={item} depth={0} activeId={activeId} onSelect={onSelect} />
      ))}
    </div>
  );
}

function TreeNode({ item, depth, activeId, onSelect }: any) {
  const [expanded, setExpanded] = React.useState(false);
  const isActive = activeId === item.id;
  const hasChildren = item.children?.length > 0;

  return (
    <>
      <div
        className={[
          "flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] transition-colors",
          isActive ? "bg-[#f4f4f5] font-medium text-[#09090b]" : "text-[#09090b] hover:bg-[#f4f4f5]",
        ].join(" ")}
        style={{ paddingLeft: 8 + depth * 16 }}
        onClick={() => { onSelect?.(item.id); if (hasChildren) setExpanded(!expanded); }}
      >
        {/* 箭头 */}
        {hasChildren ? (
          <span className={`w-4 h-4 flex items-center justify-center text-[#71717a] transition-transform ${expanded ? "rotate-90" : ""}`}>
            ›
          </span>
        ) : (
          <span className="w-4 h-4" />
        )}
        {/* 图标 */}
        <span className="w-4 h-4 text-[#71717a]">📁</span>
        {/* 标签 */}
        <span className="truncate">{item.label}</span>
        {/* 计数 */}
        {item.count != null && (
          <span className="text-[11px] tabular-nums text-[#a1a1aa] shrink-0">({item.count})</span>
        )}
      </div>
      {/* 子节点 */}
      {expanded && hasChildren && (
        <div>
          {item.children.map((child: any) => (
            <TreeNode key={child.id} item={child} depth={depth + 1} activeId={activeId} onSelect={onSelect} />
          ))}
        </div>
      )}
    </>
  );
}
```

## 6. Migration Rules

| 旧写法 | 新写法 |
|---|---|
| 图标用 `text-gray-400` | 统一 `#71717a` |
| 手写展开/收起动画 | 统一 `transition-transform rotate-90` |
| 自定义行高 | 固定 `h-8` (32px) |
| 不同页面不同缩进 | 统一 `8 + depth × 16` |

## 7. Do / Don't

**Do:**
- 所有图标统一 `#71717a`。
- 行高固定 `32px`。
- 缩进递增 `16px`。
- 选中态与 hover 态使用同一底色 `#f4f4f5`。

**Don't:**
- 不要用多种灰色给图标着色。
- 不要自定义行高。
- 不要在不同页面用不同缩进。
- 不要用 emoji 当文件夹图标（用 lucide icons）。

## 8. QA Checklist

- [ ] 图标统一 `#71717a`
- [ ] 行高 `32px`
- [ ] 圆角 `4px`
- [ ] 缩进 `8 + depth × 16`
- [ ] 选中态 `#f4f4f5` + font-medium
- [ ] 展开箭头有旋转动画
- [ ] fallback 可独立落地

## 9. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related specs: `component-specs/admin-sidebar.md`

## 代码对照（✅/❌）

### ❌ 错误：行高 40px
```tsx
<TreeItem className="h-10 text-sm">{node.name}</TreeItem>
```
**为什么错**：Tree 通常嵌入侧栏/弹窗，密集场景需要紧凑节奏；40px 与 DataTable 撞节奏。

### ✅ 正确：行高 32px
```tsx
<Tree rowHeight={32} />
{/* 32px 行高 / 14px 文字 / 8px 垂直 padding */}
```

---

### ❌ 错误：缩进 24px 固定
```tsx
<div style={{ paddingLeft: depth * 24 }}>
  <TreeItem ... />
</div>
```
**为什么错**：缩进过大，深层节点被推到右侧空白；首层也凭空缩 24px。

### ✅ 正确：8 + depth × 16
```tsx
<TreeItem
  depth={depth}
  style={{ paddingLeft: 8 + depth * 16 }}
/>
{/* 首层贴边 8px / 每层递进 16px / 16px 收手 */}
```

---

### ❌ 错误：图标多色
```tsx
<TreeItem>
  <FolderIcon className="text-yellow-500" />
  <FileIcon   className="text-blue-500" />
</TreeItem>
```
**为什么错**：彩色图标喧宾夺主；与 ClawPro 统一灰阶图标语言冲突。

### ✅ 正确：统一弱色图标
```tsx
<TreeItem
  icon={<FolderIcon size={16} className="text-[#71717a]" />}
/>
{/* 所有 Tree 图标统一 #71717a / 16px；
    选中态由行底色（#f4f4f5）传达，不靠图标变色 */}
```

---

### ❌ 错误：hover/selected 用品牌蓝
```tsx
<TreeItem
  className={cn(
    'hover:bg-blue-50',
    selected && 'bg-blue-100 text-blue-600'
  )}
/>
```
**为什么错**：Tree 经常多选/连续选，蓝色块过强会刺眼；与 Sidebar Active 蓝色语义冲突。

### ✅ 正确：中性灰底
```tsx
<Tree
  /* hover    : bg-[#f4f4f5] */
  /* selected : bg-[#f4f4f5] + text-[var(--cp-text-title)] font-medium */
/>
```

---

### ❌ 错误：上千节点不虚拟化
```tsx
{tree.map(node => <TreeItem key={node.id} {...node} />)}
{/* 渲染 2000+ DOM 节点 */}
```
**为什么错**：滚动卡顿；展开折叠掉帧；移动端崩溃。

### ✅ 正确：虚拟滚动
```tsx
<Tree
  data={tree}
  virtual
  height={400}
/>
{/* 内置 react-window；只渲染可视区 +/- buffer */}
```
