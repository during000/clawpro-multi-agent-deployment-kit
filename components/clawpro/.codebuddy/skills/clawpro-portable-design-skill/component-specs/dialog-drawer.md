# Dialog / Drawer

> **Showcase mapping**: `dialog` · `alert-dialog` · `drawer` · `sheet`（`client/src/pages/DesignSystemComponents.tsx`）

## 1. Purpose

- 统一确认弹窗、表单弹窗、侧边抽屉的层级、尺寸和信息结构。
- 防止宿主仓继续把任意浮层做成一套临时样式。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：危险确认、轻量信息确认、短表单、中长表单、详情侧边面板
- 不适用场景：页面主内容、极轻量 hover 提示、长说明 tooltip

## 3. Visual Standard

| Item | Dialog | Drawer | Notes |
|---|---|---|---|
| Container | 白底浮层 | 白底侧滑面板 | 都属于 overlay 层 |
| Radius | 跟随分端规则 | 跟随分端规则 | Admin 优先 4px，Tenant 跟随对应差异 |
| Shadow | overlay shadow | overlay shadow | 不自发明新阴影 |
| Header | 标题 + 可选描述 | 标题 + 可选描述 | 信息层级清楚 |
| Footer | 主次操作清楚 | 可固定在底部 | 危险确认必须显式 |

### 3.1 子元素字号 / 色 token（改容器必贯彻，P3）

> ⚠️ 改弹窗 / 抽屉容器（圆角 / 阴影 / padding）时**必须同时核对内部子元素走语义 token**，不得只换外层：
>
> | 子元素 | 字号 / 字重 | 颜色 token |
> |---|---|---|
> | Title | `text-base`（16px）/ Semibold | `var(--cp-text-title)` |
> | Description | `text-sm`（14px） | `var(--cp-text-muted)` |
> | Body 正文 | `text-sm`（14px） | `var(--cp-text-body)` |
> | 空区降级提示 | `text-xs`（12px） | `var(--cp-text-weak)` |
> | 危险确认按钮 | — | `bg-[var(--cp-text-danger)]`（见 §10 / SKILL.md §5「危险操作→destructive」） |
>
> 一律走 token，不散写 `text-[#xxx]` / `text-gray-*` / 自定字号。

## 4. Anatomy

```text
Dialog / Drawer
  Header
    Title
    Description optional
  Body
    Sections / Fields / Summary
  Footer optional
    Primary action
    Secondary action
```

## 5. States

- default: 打开后内容清楚、聚焦明确。
- loading: 提交按钮内 loading，不锁整页。
- validation-error: 字段错误贴近字段，不漂浮在远处。
- empty-section: 内嵌区块空态降级为轻量文字，不用大插画。
- destructive-confirm: 用更强确认语义，避免误触。

## 6. Demo Repo Usage

- 当前 demo 仓主要依赖共享 Dialog / Drawer 组件。
- 对话框类典型页面可参考：`client/src/pages/admin/DocManagement.tsx`
- 长内容抽屉和表单场景可沿用宿主仓已有 Drawer，只需按本 spec 对齐视觉和结构。

```tsx
<Dialog open={open} onOpenChange={setOpen}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>添加文档</DialogTitle>
    </DialogHeader>
    <div className="space-y-4">...</div>
    <DialogFooter>
      <Button variant="claw-outline">取消</Button>
      <Button variant="dialog-confirm">确认</Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

## 7. Portable Fallback

### 7.1 If host repo already has Modal / Drawer

- 可直接复用宿主仓已有 Modal / Drawer。
- 但必须统一 header、body、footer 结构，不要每个弹窗自由排版。
- 需要危险确认时，优先使用宿主仓最强确认弹窗能力，而不是普通 modal 假装危险确认。

### 7.2 Minimal React fallback

```tsx
export function PortableDialogShell({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20 p-4">
      <section className="w-full max-w-[560px] rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-6">
        <header className="mb-4">
          <h2 className="text-base font-semibold text-[var(--cp-text-title)]">{title}</h2>
        </header>
        <div className="space-y-4">{children}</div>
      </section>
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-overlay">
  <section class="cp-dialog">
    <header class="cp-dialog-header">
      <h2>添加文档</h2>
    </header>
    <div class="cp-dialog-body">...</div>
    <footer class="cp-dialog-footer">...</footer>
  </section>
</div>
```

```css
.cp-overlay { position: fixed; inset: 0; display: flex; align-items: center; justify-content: center; padding: 16px; background: var(--cp-overlay-bg); }
.cp-dialog { width: 100%; max-width: 560px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 24px; }
.cp-dialog-footer { display: flex; justify-content: flex-end; gap: 12px; margin-top: 24px; }
```

## 8. Migration Rules

- 旧写法：直接在页面里拼 `fixed + absolute + shadow` 浮层。
- 新口径：优先复用宿主仓 Modal / Drawer 容器，只统一结构和视觉 token。
- 可以暂时兼容：宿主仓不同弹窗组件并存，但 header/body/footer 要统一。
- 不允许新增：大段说明塞 Tooltip、内嵌区块空态仍用大插画、危险操作无二次确认。

## 9. Do / Don't

Do:

- 用 Header / Body / Footer 固定结构。
- 长表单控制最大高度并允许滚动。
- 危险操作单独强调确认。

Don't:

- 不要把长说明放进 Tooltip。
- 不要让弹窗 footer 的主次操作顺序每个页面都不一样。
- 不要在弹窗里继续新增自由阴影和自由圆角。

## 10. QA Checklist

- [ ] overlay 层级清楚
- [ ] header、body、footer 结构完整
- [ ] 长内容可滚动
- [ ] 危险确认有单独确认路径
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo page: `client/src/pages/admin/DocManagement.tsx`
- Related rules: `references/components.md`
- Related recipe: `references/page-recipes.md`


## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / §6 同口径。Dialog / Drawer 5 项高频误用 → ClawPro 正确写法。

### 12.1 不要手写 fixed + absolute + shadow 浮层

```tsx
// ❌ 自己拼一个 modal
{open && (
  <div className="fixed inset-0 z-50 bg-black/30 flex items-center justify-center">
    <div className="bg-white rounded-lg shadow-2xl p-6 w-[480px]">
      <h2 className="text-lg font-bold">添加文档</h2>
      ...
      <div className="flex justify-end gap-2 mt-4">
        <button>取消</button>
        <button className="bg-blue-500 text-white px-4 py-2">确认</button>
      </div>
    </div>
  </div>
)}

// ✅ 使用统一 Dialog 组件家族
<Dialog open={open} onOpenChange={setOpen}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>添加文档</DialogTitle>
    </DialogHeader>
    <div className="space-y-4">…</div>
    <DialogFooter>
      <Button variant="claw-outline">取消</Button>
      <Button variant="dialog-confirm">确认</Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

### 12.2 Header / Body / Footer 必须分明，不要自由排版

```tsx
// ❌ 标题、表单、按钮全塞在一个 div，按钮夹在表单中间
<DialogContent>
  <h2>添加文档</h2>
  <Input />
  <Button>确认</Button>
  <Textarea />
  <Button variant="ghost">取消</Button>
</DialogContent>

// ✅ Header / Body / Footer 三段式
<DialogContent>
  <DialogHeader>
    <DialogTitle>添加文档</DialogTitle>
    <DialogDescription>支持 PDF / Word / Markdown</DialogDescription>
  </DialogHeader>

  <div className="space-y-4">
    <Field><Label>名称</Label><Input /></Field>
    <Field><Label>说明</Label><Textarea /></Field>
  </div>

  <DialogFooter>
    <Button variant="claw-outline">取消</Button>
    <Button variant="dialog-confirm">确认</Button>
  </DialogFooter>
</DialogContent>
```

### 12.3 危险确认：用 destructive，不靠红字伪装

```tsx
// ❌ 用普通 Dialog + 红色文字按钮当危险确认
<Dialog>
  <DialogContent>
    <DialogTitle>删除策略？</DialogTitle>
    <DialogFooter>
      <Button variant="ghost">取消</Button>
      <Button className="text-red-600">确认删除</Button>
    </DialogFooter>
  </DialogContent>
</Dialog>

// ✅ 走专用 destructive variant + 强语义文案
<AlertDialog>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>删除「内容审核策略」？</AlertDialogTitle>
      <AlertDialogDescription>
        删除后该策略下的 12 个 Agent 将立即失去过滤能力，操作不可恢复。
      </AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>取消</AlertDialogCancel>
      <AlertDialogAction className="bg-[var(--cp-text-danger)] text-white hover:bg-[var(--cp-text-danger)]/90">
        确认删除
      </AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

### 12.4 长表单：max-h + 内部滚动，不撑破视口

```tsx
// ❌ 长表单一直撑到底部，超出视口要滚整页
<DialogContent>
  <DialogHeader><DialogTitle>批量导入</DialogTitle></DialogHeader>
  <div>
    {Array.from({ length: 30 }).map((_, i) => <Field key={i}>…</Field>)}
  </div>
  <DialogFooter>…</DialogFooter>
</DialogContent>

// ✅ Body 限高 + overflow-auto，header / footer 固定
<DialogContent className="max-h-[80vh] flex flex-col">
  <DialogHeader>
    <DialogTitle>批量导入</DialogTitle>
  </DialogHeader>
  <div className="flex-1 overflow-y-auto space-y-4 -mx-6 px-6">
    {Array.from({ length: 30 }).map((_, i) => <Field key={i}>…</Field>)}
  </div>
  <DialogFooter>
    <Button variant="claw-outline">取消</Button>
    <Button variant="dialog-confirm">导入</Button>
  </DialogFooter>
</DialogContent>
```

### 12.5 内嵌空态用纯文字，不上大插画

```tsx
// ❌ Dialog 内子区块没有数据时塞页面级 Empty 大插画
<DialogContent>
  <DialogTitle>选择文档</DialogTitle>
  <Empty className="py-16">
    <EmptyHeader>
      <EmptyMedia />
      <EmptyTitle>暂无可选文档</EmptyTitle>
    </EmptyHeader>
  </Empty>
</DialogContent>

// ✅ Dialog 内空态用纯文字双行
<DialogContent>
  <DialogTitle>选择文档</DialogTitle>
  <div className="text-center py-8 space-y-1">
    <p className="text-xs text-[var(--cp-text-weak)]">暂无可选文档</p>
    <p className="text-xs text-[var(--cp-text-weak)]">先到「文档管理」上传 Markdown 文件</p>
  </div>
</DialogContent>
```
