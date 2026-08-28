# Loading / Progress

## 1. Purpose

- 统一页面加载、局部刷新、按钮执行中、进度条和操作反馈的视觉与状态。
- 避免页面在加载时整页锁死、spinner 尺寸不一、骨架屏和最终布局不一致。

## 2. Scope

- 适用端：Admin / Tenant / Shared。
- 必用场景：表格加载、卡片列表加载、按钮提交中、上传 / 批处理进度、长任务执行中、局部刷新。
- 不适用场景：全局错误页、完整空状态；这些按 `empty-state.md` 或页面 recipe 处理。

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| Spinner | 16px / 20px，品牌蓝或弱灰 | 按钮内使用小尺寸 |
| Skeleton | 与最终布局同尺寸 | 不要用一块大灰条替代复杂布局 |
| Progress | 4px-6px 高，品牌蓝进度 | 背景弱灰 / 弱蓝 |
| Button Loading | 按钮内部 spinner + disabled | 不锁死整页 |
| Table Loading | 骨架行或局部 loading | 行数接近最终页面 |
| Card Loading | 骨架卡片数量与最终 grid 一致 | Tenant 卡片保持圆角 |
| Toast / Alert | 操作结果反馈 | 不替代页面错误态 |
| Motion | 轻量、支持 reduced motion | 不做夸张循环动画 |

## 4. Anatomy

```text
LoadingState
  Skeleton / Spinner / Progress
  Message optional
  Retry optional for error
```

## 5. States

- initial-loading: 首次加载，可使用骨架屏。
- refreshing: 局部刷新，保留已有数据，按钮或局部区域 loading。
- submitting: 表单 / 按钮执行中，按钮 disabled + spinner。
- progress: 可量化任务展示进度条。
- indeterminate: 不可量化任务展示 spinner 或循环条。
- success: 操作成功给出 toast / inline 状态。
- error: 显示原因和重试入口。
- empty-after-load: 加载完成无数据时进入 Empty，不继续显示 loading。
- reduced-motion: 复杂动效降级。

## 6. Demo Repo Usage

- Spinner：`client/src/components/ui/spinner.tsx`
- Skeleton：`client/src/components/ui/skeleton.tsx`
- Progress：`client/src/components/ui/progress.tsx`
- Toast：`client/src/components/ui/sonner.tsx`
- Alert：`client/src/components/ui/alert.tsx`
- 页面 recipe：`references/page-recipes.md`

## 7. Portable Fallback

### 7.1 If host repo already has loading components

- 复用宿主仓 Spinner / Skeleton / Progress 逻辑。
- 统一尺寸、颜色、圆角、位置和与最终布局的对应关系。
- 表格和卡片列表优先 skeleton，不要只在整页中央放一个 spinner。

### 7.2 Minimal React fallback

```tsx
export function PortableTableSkeleton() {
  return (
    <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
      {Array.from({ length: 5 }).map((_, index) => (
        <div key={index} className="flex h-[54px] items-center gap-4 border-b border-[var(--cp-border)] px-4 last:border-b-0">
          <span className="h-4 w-32 rounded-[4px] bg-[var(--cp-bg-subtle)]" />
          <span className="h-4 w-20 rounded-[4px] bg-[var(--cp-bg-subtle)]" />
          <span className="h-4 flex-1 rounded-[4px] bg-[var(--cp-bg-subtle)]" />
        </div>
      ))}
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-progress"><span style="width: 48%"></span></div>
```

```css
.cp-progress { height: 4px; overflow: hidden; border-radius: 9999px; background: var(--cp-bg-subtle); }
.cp-progress span { display: block; height: 100%; border-radius: inherit; background: var(--cp-brand-blue); }
```

## 8. Migration Rules

- 旧写法：整页 spinner、按钮无 disabled、加载完成后仍显示“加载中”。
- 新口径：按区域加载，尽量保留页面结构；按钮操作只锁按钮，不锁整页。
- 表格 loading 用骨架行或局部提示；卡片列表 loading 用骨架卡片。
- 失败态必须有原因和重试入口。
- 进度条只在有实际进度时使用；没有进度时使用 indeterminate 状态。

## 9. Do / Don't

Do:

- 让 skeleton 尺寸贴近最终内容。
- 按钮执行中显示 spinner 并 disabled。
- 加载失败提供重试。

Don't:

- 不要用全屏遮罩阻断普通局部刷新。
- 不要让 spinner 和 skeleton 同时表达同一层加载。
- 不要用无限动画替代明确错误态。

## 10. QA Checklist

- [ ] 首屏 / 局部 / 按钮 loading 分层清楚
- [ ] Skeleton 与最终布局一致
- [ ] 按钮执行中 disabled，避免重复提交
- [ ] 表格 / 卡片列表有对应加载态
- [ ] 失败态有原因和重试入口
- [ ] 操作成功 / 失败反馈明确
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/skeleton.tsx`
- Demo code: `client/src/components/ui/spinner.tsx`
- Demo code: `client/src/components/ui/progress.tsx`
- Related recipe: `references/page-recipes.md`

## 代码对照（✅/❌）

### ❌ 错误：整页 loading 锁住所有按钮
```tsx
{loading && (
  <div className="fixed inset-0 bg-white/60 z-50 flex items-center justify-center">
    <Spinner size="lg" />
  </div>
)}
```
**为什么错**：用户无法继续操作其他卡片/导航；后台请求不应锁页。

### ✅ 正确：局部 spinner
```tsx
<SurfaceCard>
  {loading ? (
    <div className="flex items-center justify-center py-12">
      <Spinner size="md" />
    </div>
  ) : (
    <DataTable {...} />
  )}
</SurfaceCard>
```

---

### ❌ 错误：Skeleton 与最终布局不匹配
```tsx
<Skeleton className="h-20 w-full" />
{/* 实际渲染却是表格、行高 40px */}
```
**为什么错**：骨架到内容会"跳一下"，用户感知比无骨架还差。

### ✅ 正确：骨架对齐最终布局
```tsx
<DataTable
  loading={loading}
  rowHeight={40}
  columns={columns}
  data={data}
/>
{/* DataTable 内置 skeleton 行高/列宽 = 真实数据 */}
```

---

### ❌ 错误：按钮 loading 改文案
```tsx
<Button onClick={save} disabled={saving}>
  {saving ? '保存中...' : '保存'}
</Button>
```
**为什么错**：文案抖动改变按钮宽度；spinner 信号被淹没。

### ✅ 正确：保留文案 + spinner
```tsx
<Button onClick={save} loading={saving}>
  保存
</Button>
{/* loading 自动渲染左侧 spinner，文案不变，宽度稳定 */}
```

---

### ❌ 错误：进度条用渐变彩色
```tsx
<div className="h-2 rounded-full bg-gradient-to-r from-pink-400 via-purple-500 to-blue-500" />
```
**为什么错**：彩色渐变破坏 ClawPro 克制风格；进度信号被装饰淹没。

### ✅ 正确：单色品牌蓝
```tsx
<Progress value={percent} />
{/* 内部 bg-[var(--cp-brand-blue)] / 轨道 bg-[var(--cp-bg-subtle)] */}
```

---

### ❌ 错误：用 emoji 代替进度提示
```tsx
{loading && <span>⏳ 加载中...</span>}
{success && <span>✅ 完成</span>}
```
**为什么错**：emoji 在不同系统渲染不一致；与 Lucide 图标体系冲突；a11y 难以朗读。

### ✅ 正确：spinner + 文字 / icon + 文字
```tsx
{loading && (
  <div className="flex items-center gap-2 text-[var(--cp-text-weak)] text-sm">
    <Spinner size="sm" />
    加载中...
  </div>
)}
```
