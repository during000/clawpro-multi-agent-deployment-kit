# Page Header

## 1. Purpose

- 统一管理端页面顶部的标题、说明和主操作区。
- 防止每个页面单独拼一套“标题 + 按钮 + 描述”布局。

## 2. Scope

- 适用端：Admin 优先
- 必用场景：管理端列表页、配置页、详情页顶部
- 不适用场景：Tenant Hero、营销落地页首屏

## 3. Visual Standard

| Item | Value | Notes |
|---|---|---|
| Title | 大标题，深色 | 左侧主信息 |
| Description | 次级说明，位于标题下方 | 可选 |
| Actions | 右侧操作组 | 主操作在最显著位置 |
| Layout | 左信息右操作，支持换行 | gap 清楚 |
| Bottom Spacing | 常规 `mb-6` 或 `mb-8` | 与下方筛选区 / 数据区拉开 |

## 4. Anatomy

```text
PageHeader
  Content
    Title
    Description optional
    TitleAccessory optional
  Actions optional
```

## 5. States

- default: 标题、说明、操作区都存在。
- minimal: 只有标题。
- with-accessory: 标题右侧带 badge、状态、辅助说明。
- wrap: 小空间下信息区和操作区允许换行，不要挤压变形。

## 6. Demo Repo Usage

- 当前 demo 仓组件：`client/src/components/ui/admin-page-header.tsx`
- 典型页面：`client/src/pages/admin/DocManagement.tsx`、`client/src/pages/admin/BasicInfo.tsx`

```tsx
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { Button } from "@/components/ui/button";

<AdminPageHeader
  title="帮助文档"
  description="此处配置的文档将展示在企业用户看到的帮助文档中。"
  actions={<Button variant="claw-primary" size="claw">添加文档</Button>}
/>
```

## 7. Portable Fallback

### 7.1 If host repo already has PageHeader / Toolbar

- 可复用宿主仓页头容器。
- 但必须保留左侧主信息、右侧操作区、标题下说明这三个层次。

### 7.2 Minimal React fallback

```tsx
export function PortableAdminPageHeader() {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0 flex-1">
        <h1 className="text-2xl font-medium text-[var(--cp-text-title)]">帮助文档</h1>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">此处配置的文档将展示在企业用户看到的帮助文档中。</p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <button className="inline-flex h-9 items-center rounded-[4px] bg-[var(--cp-brand-black)] px-6 text-sm text-white">添加文档</button>
      </div>
    </div>
  );
}
```

## 8. Migration Rules

- 旧写法：页面顶部手写标题、按钮、说明，各页面 spacing 不一致。
- 新口径：统一页头结构，再让宿主仓组件或 class 去实现。
- 可以暂时兼容：宿主仓既有 toolbar 组件。
- 不允许新增：说明文字和操作区乱序、主操作跑到内容区内部、页头 spacing 到处不一致。

## 9. Do / Don't

Do:

- 标题、说明、操作区层次明确。
- 主操作保持在右侧。
- 用统一 gap 和底部间距。

Don't:

- 不要把页头写成一段自由排版的普通内容。
- 不要让说明文字和按钮混在同一行拥挤排列。
- 不要每个页面自己定义一套 spacing。

## 10. QA Checklist

- [ ] 左信息右操作结构清楚
- [ ] 标题和说明层级正确
- [ ] 主操作在右侧且显著
- [ ] 宿主仓可用 fallback 实现

## 11. References

- Demo code: `client/src/components/ui/admin-page-header.tsx`
- Demo page: `client/src/pages/admin/DocManagement.tsx`
- Demo page: `client/src/pages/admin/BasicInfo.tsx`


## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。PageHeader 5 项高频误用 → ClawPro 正确写法。

### 12.1 不要每页手写"标题 + 按钮"自由排版

```tsx
// ❌ 自由布局，跨页 spacing / 层级漂移
<div>
  <div className="flex justify-between mb-3">
    <h2 className="text-lg font-bold">帮助文档</h2>
    <button className="bg-blue-500 text-white px-4 py-2 rounded">添加文档</button>
  </div>
  <p className="text-gray-500 mb-2">此处配置...</p>
</div>

// ✅ 用 AdminPageHeader 三段插槽
<AdminPageHeader
  title="帮助文档"
  description="此处配置的文档将展示在企业用户看到的帮助文档中。"
  actions={<Button variant="claw-primary" size="claw">添加文档</Button>}
/>
```

### 12.2 主操作必须右侧，不能跑到内容区里

```tsx
// ❌ 主操作放在标题下方居中，与右侧次操作冲突
<div className="text-center mb-6">
  <h1 className="text-2xl font-medium">实例管理</h1>
  <Button variant="claw-primary" size="claw" className="mt-3">创建实例</Button>
</div>
<div className="flex justify-end mb-3">
  <Button variant="claw-outline">导出</Button>
</div>

// ✅ 所有页头操作都收在右侧 actions 槽
<AdminPageHeader
  title="实例管理"
  actions={
    <div className="flex items-center gap-2">
      <Button variant="claw-outline" size="claw">导出</Button>
      <Button variant="claw-primary" size="claw">创建实例</Button>
    </div>
  }
/>
```

### 12.3 Description 与 Actions 不要挤一行

```tsx
// ❌ 描述与按钮硬塞同一行，小屏拥挤
<div className="flex items-center justify-between mb-6">
  <div className="flex items-center gap-4">
    <h1 className="text-2xl font-medium">帮助文档</h1>
    <p className="text-sm text-[var(--text-secondary)]">此处配置的文档将展示在企业用户看到的帮助文档中</p>
  </div>
  <Button variant="claw-primary" size="claw">添加文档</Button>
</div>

// ✅ 标题下方一行 description，右侧操作独立
<AdminPageHeader
  title="帮助文档"
  description="此处配置的文档将展示在企业用户看到的帮助文档中。"
  actions={<Button variant="claw-primary" size="claw">添加文档</Button>}
/>
```

### 12.4 底部 spacing 用 mb-6 / mb-8，不每页定义

```tsx
// ❌ 每个页面定义自己的间距
<AdminPageHeader title="审计日志" className="mb-3" />  {/* A 页面 */}
<AdminPageHeader title="实例管理" className="mb-12" /> {/* B 页面 */}
<AdminPageHeader title="模型配额" className="mt-2 mb-5" /> {/* C 页面 */}

// ✅ 统一 mb-6（紧凑）/ mb-8（宽松），项目内只有这两档
<AdminPageHeader title="审计日志" />              {/* 默认 mb-6 */}
<AdminPageHeader title="实例管理" spacing="loose" /> {/* mb-8 */}
```

### 12.5 标题右侧 accessory 用 TitleAccessory，不外贴 Badge

```tsx
// ❌ Badge 直接拼在 title 后，垂直对齐 / 间距全靠手写
<div className="flex items-center gap-3 mb-2">
  <h1 className="text-2xl font-medium">Agent 详情</h1>
  <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">运行中</span>
  <span className="text-xs text-gray-500">v2.1.0</span>
</div>

// ✅ 用 titleAccessory 槽（自动处理对齐 / 间距）
<AdminPageHeader
  title="Agent 详情"
  titleAccessory={
    <span className="flex items-center gap-2">
      <span className="text-sm font-medium text-[var(--text-success)]">运行中</span>
      <span className="text-xs text-[var(--cp-text-weak)]">v2.1.0</span>
    </span>
  }
/>
```
