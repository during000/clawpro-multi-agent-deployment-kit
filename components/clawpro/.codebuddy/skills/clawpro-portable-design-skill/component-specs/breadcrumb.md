# Breadcrumb

## 1. Purpose

- 提供页面层级导航，让用户知道"我在哪"。
- 统一面包屑的分隔符、文字色、交互。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：详情页、多级嵌套页、从列表进入子页面
- 不适用场景：仅有一级的页面（不显示面包屑）

## 3. Visual Standard

| Item | Value |
|---|---|
| 字号 | `14px` (text-sm) |
| 当前页文字 | `var(--cp-text-title)` / font-medium |
| 祖先页文字 | `var(--cp-text-muted)` (#737373) |
| 祖先页 hover | `var(--cp-text-title)` |
| 分隔符 | `/` 或 `>` / `var(--cp-text-weak)` |
| 间距 | 分隔符左右 `gap-1.5`（6px） |

## 4. Portable Fallback

```tsx
function PortableBreadcrumb({ items }: { items: { label: string; href?: string }[] }) {
  return (
    <nav className="flex items-center gap-1.5 text-sm">
      {items.map((item, i) => (
        <React.Fragment key={i}>
          {i > 0 && <span className="text-[var(--cp-text-weak)]">/</span>}
          {item.href ? (
            <a href={item.href} className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)]">
              {item.label}
            </a>
          ) : (
            <span className="font-medium text-[var(--cp-text-title)]">{item.label}</span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
}
```

## 5. Do / Don't

**Do:**
- 当前页不可点击，用深色 font-medium。
- 祖先页可点击，用灰色。

**Don't:**
- 不要给面包屑加背景色/边框。
- 不要在只有一级的页面显示面包屑。

## 6. QA Checklist

- [ ] 当前页不可点击
- [ ] 祖先页可点击且有 hover 态
- [ ] 分隔符使用弱色
- [ ] fallback 使用 `var(--cp-*)` CSS variable

## 7. References

- Related specs: `component-specs/page-header.md`, `component-specs/admin-sidebar.md`, `component-specs/tenant-topnav.md`

## 8. 代码对照（✅/❌）

> 与 SKILL.md §2 / `admin-sidebar.md` / `tenant-topnav.md` 同包同口径。Breadcrumb 5 项高频误用 → ClawPro 正确写法。

### 8.1 当前页不可点击，必须 font-medium 深色

```tsx
// ❌ 当前页和祖先页同色同 weight，无法分辨
<nav className="flex items-center gap-1.5 text-sm">
  <a href="/admin/agents" className="text-[var(--cp-text-muted)]">实例管理</a>
  <span>/</span>
  <a href="#" className="text-[var(--cp-text-muted)]">Agent 详情</a>
</nav>

// ❌ 当前页仍可点击，没有 aria-current
<nav>
  <a href="/admin/agents">实例管理</a>
  <span>/</span>
  <a href="/admin/agents/123">Agent 详情</a>  {/* 这是当前页 */}
</nav>

// ✅ 当前页：<span> + font-medium + 深色 + aria-current
<nav className="flex items-center gap-1.5 text-sm" aria-label="breadcrumb">
  <a href="/admin/agents" className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)]">实例管理</a>
  <span className="text-[var(--cp-text-weak)]">/</span>
  <span className="font-medium text-[var(--cp-text-title)]" aria-current="page">Agent 详情</span>
</nav>
```

### 8.2 不要给 Breadcrumb 加背景 / 边框 / 卡片包裹

```tsx
// ❌ Breadcrumb 套灰底卡片
<div className="bg-gray-50 border border-gray-200 rounded-[4px] px-3 py-2 mb-4">
  <Breadcrumb>...</Breadcrumb>
</div>

// ❌ 横向加背景色块
<nav className="flex items-center gap-1.5 text-sm bg-[var(--cp-bg-subtle)] px-4 py-2 rounded">
  ...
</nav>

// ✅ 仅文字 + 间距，不加任何容器装饰
<Breadcrumb className="mb-3" aria-label="breadcrumb">
  <BreadcrumbList>
    <BreadcrumbItem><BreadcrumbLink href="/admin/agents">实例管理</BreadcrumbLink></BreadcrumbItem>
    <BreadcrumbSeparator />
    <BreadcrumbItem><BreadcrumbPage>Agent 详情</BreadcrumbPage></BreadcrumbItem>
  </BreadcrumbList>
</Breadcrumb>
```

### 8.3 单级页面不要硬塞 Breadcrumb

```tsx
// ❌ 列表页（顶级）也加一级 Breadcrumb 占位
<Breadcrumb>
  <BreadcrumbList>
    <BreadcrumbItem><BreadcrumbPage>实例管理</BreadcrumbPage></BreadcrumbItem>
  </BreadcrumbList>
</Breadcrumb>
<AdminPageHeader title="实例管理" />

// ✅ 单级页面只用 PageHeader，不加 Breadcrumb
<AdminPageHeader title="实例管理" actions={<Button>创建</Button>} />
```

### 8.4 分隔符用弱灰，不要用纯黑 / 蓝色

```tsx
// ❌ 分隔符纯黑色，与文字层级冲突
<span className="text-black">/</span>

// ❌ 分隔符用品牌蓝
<span className="text-[var(--cp-brand-blue)]">/</span>

// ✅ var(--cp-text-weak) 弱灰
<span className="text-[var(--cp-text-weak)]">/</span>
```

### 8.5 祖先页一定要可点击，不要只是装饰文本

```tsx
// ❌ 祖先页用 <span> 包裹，看起来像导航实际上不可跳转
<nav className="flex items-center gap-1.5 text-sm">
  <span className="text-[var(--cp-text-muted)]">实例管理</span>
  <span>/</span>
  <span className="font-medium">Agent 详情</span>
</nav>

// ✅ 祖先页用 <a> / Link，hover 时颜色加深
<nav className="flex items-center gap-1.5 text-sm" aria-label="breadcrumb">
  <a href="/admin/agents" className="text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)]">
    实例管理
  </a>
  <span className="text-[var(--cp-text-weak)]">/</span>
  <span className="font-medium text-[var(--cp-text-title)]" aria-current="page">Agent 详情</span>
</nav>
```
