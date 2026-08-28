# Toast

## 1. Purpose

- 统一操作反馈（成功/失败/提示）的顶部弹出通知样式。
- 替代各页面自行拼装的弹出 UI。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：表单提交成功/失败、操作确认反馈、异步任务完成通知
- 不适用场景：需要用户主动操作才消失的提示（用 Alert）、长文本说明（用 Dialog）

## 3. Visual Standard

| Item | Value |
|---|---|
| 背景 | 白色 `#FFFFFF` |
| 文字色 | `#09090b` |
| 边框 | `var(--cp-border)` (#EAEEF4) |
| 圆角 | `12px` (rounded-xl) |
| 内边距 | `12px 16px` |
| 字号 | `14px` / font-medium |
| 文字对齐 | **左对齐** |
| 阴影 | shadow-lg |
| 定位 | 页面顶部居中 (top-center) |
| z-index | `99999`（确保在 Dialog 之上） |
| 自动消失 | `4000ms`（4 秒） |
| 关闭按钮 | **右上角外侧**，20×20 圆形白底 + 蓝灰边框 + 轻阴影，hover 时 bg-[#f4f4f5] |

### 关闭按钮实现

不使用 sonner 内置 `closeButton`（存在幽灵 toast 兼容问题）。改用自定义 `withClose(id)` 函数：

```tsx
import { toast } from "sonner";
import { withClose } from "@/components/ui/sonner";
import { X } from "lucide-react";

// withClose 渲染一个绝对定位的圆形关闭按钮（-right-2 -top-2，浮在 toast 右上角外侧）
function withClose(id: string | number) {
  return (
    <button
      className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center rounded-full bg-white border border-[#EAEEF4] shadow-sm text-[#7b818f] hover:bg-[#f4f4f5] hover:text-[#09090b]"
      onClick={() => toast.dismiss(id)}
    >
      <X className="h-3.5 w-3.5" />
    </button>
  );
}
```

Toast 容器必须加 `overflow-visible` + `relative` 让关闭按钮能超出边界显示。

### 布局结构

```
┌─────────────────────────────────────────┐
│  [icon]  消息文本（左对齐）               │  × (浮在右上角外侧)
└─────────────────────────────────────────┘
```

### 类型

| 类型 | 图标 | 说明 |
|---|---|---|
| `error` | 黑色感叹号 | 操作失败、校验不通过 |
| `success` | 绿色勾 | 操作成功 |
| `info` | 蓝色 i | 普通提示 |
| `warning` | 橙色感叹号 | 警告提示 |

> 所有类型统一**白色背景 + 蓝灰边框**，不按类型换底色。

## 4. Portable Fallback

### 4.1 React fallback

```tsx
function PortableToast({ type, message, onClose }: {
  type: 'success' | 'error' | 'info' | 'warning';
  message: string;
  onClose: () => void;
}) {
  return (
    <div
      className="fixed top-6 left-1/2 -translate-x-1/2 z-[99999] relative overflow-visible flex items-start gap-3 rounded-xl border border-[var(--cp-border)] bg-white px-4 py-3 shadow-lg max-w-[420px] text-left"
      role="alert"
    >
      <span className="shrink-0 mt-0.5">{type === 'success' ? '✓' : type === 'error' ? '!' : type === 'warning' ? '⚠' : 'i'}</span>
      <span className="text-sm font-medium text-[#09090b] flex-1">{message}</span>
      <button
        className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center rounded-full bg-white border border-[var(--cp-border)] shadow-sm text-[#7b818f] hover:bg-[#f4f4f5]"
        onClick={onClose}
      >
        ×
      </button>
    </div>
  );
}
```

### 4.2 使用方式（如果宿主仓有 sonner）

```tsx
import { toast } from 'sonner';
import { withClose } from "@/components/ui/sonner";

// 带关闭按钮的 toast
const id = Date.now();
toast.error(() => <>{`操作失败：请联系管理员`}{withClose(id)}</>, { id });

// 不需要关闭按钮的简单 toast
toast.success("操作成功");
```

## 5. 强制规则

- **不使用 sonner 内置 `closeButton` 属性**（存在幽灵 toast 兼容问题）。
- 关闭功能通过自定义 `withClose(id)` 实现，渲染为右上角外侧浮按钮。
- **关闭按钮在右上角外侧**（`-right-2 -top-2`），圆形白底带边框。
- **文字左对齐**，不居中。
- 所有类型统一白色背景 + 蓝灰边框，不按类型换底色。
- 禁止在业务代码中自行拼装弹出通知 UI。
- Toast 层级固定 `z-index: 99999`，确保在 Dialog 之上。
- Toast 容器必须 `overflow-visible` + `relative`。
- index.css 中用 `[data-sonner-toast] > [data-close-button] { display: none }` 隐藏 sonner 内置 close button 残留。

## 6. Do / Don't

**Do:**
- 统一使用 toast API 触发。
- 关闭按钮在右侧。
- 所有类型白底统一风格。

**Don't:**
- 不要自行拼装通知弹层 UI。
- 不要按类型换底色（如红底错误、绿底成功）。
- 不要把关闭按钮放在左上角。

## 7. QA Checklist

- [ ] 白色背景 + 蓝灰边框
- [ ] 关闭按钮在右侧
- [ ] 顶部居中定位
- [ ] z-index 在 Dialog 之上
- [ ] 未自行拼装通知 UI
- [ ] fallback 使用 `var(--cp-*)` CSS variable

## 8. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related tokens: `--cp-border`

## 9. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Toast 5 项高频误用 → ClawPro 正确写法。

### 9.1 不要按类型换底色

```tsx
// ❌ 错误用红底白字、成功用绿底白字，视觉过激 + 与全局白底口径冲突
toast.error("操作失败", {
  className: "bg-red-600 text-white border-red-700",
});
toast.success("保存成功", {
  className: "bg-emerald-600 text-white",
});

// ✅ 全部白底 + 蓝灰边框，类型靠左侧图标区分
toast.error("操作失败");        // 内置：白底 + 黑感叹号
toast.success("保存成功");      // 内置：白底 + 绿勾
toast.info("数据已更新");       // 内置：白底 + 蓝 i
toast.warning("配额即将用尽"); // 内置：白底 + 橙感叹号
```

### 9.2 不要用 sonner 内置 closeButton

```tsx
// ❌ 直接传 closeButton，会触发幽灵 toast bug（旧 toast 残留）
<Toaster closeButton position="top-center" />

// ✅ Toaster 不传 closeButton，使用 withClose(id) 自定义渲染
<Toaster position="top-center" />

const id = Date.now();
toast.error(() => <>{`操作失败：请联系管理员`}{withClose(id)}</>, { id });

// ✅ 不需要关闭按钮的简单 toast 直接调
toast.success("保存成功");
```

### 9.3 关闭按钮在右上角外侧（不在内部 / 不在左侧）

```tsx
// ❌ 关闭按钮在 toast 内部右下角
<div className="rounded-xl bg-white p-3 flex justify-between">
  <span>消息</span>
  <button className="text-xs">关闭</button>
</div>

// ❌ 关闭按钮在左上角
<div className="rounded-xl bg-white p-3 relative">
  <button className="absolute -left-2 -top-2">×</button>
  <span>消息</span>
</div>

// ✅ withClose 内部：absolute -right-2 -top-2 圆形白底浮按钮
function withClose(id: string | number) {
  return (
    <button
      className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center rounded-full bg-white border border-[#EAEEF4] shadow-sm text-[#7b818f] hover:bg-[#f4f4f5]"
      onClick={() => toast.dismiss(id)}
    >
      <X className="h-3.5 w-3.5" />
    </button>
  );
}
// 配合 Toaster 容器：toastOptions.classNames.toast = "overflow-visible relative"
```

### 9.4 文字左对齐，不居中

```tsx
// ❌ 居中对齐，长文案反复换行视觉跳动
toast.success("保存成功", {
  className: "text-center",
});

// ✅ 左对齐（icon 在左，消息从左展开）
toast.success("保存成功");
// 内置：text-left + flex items-start gap-3
```

### 9.5 不要业务侧自己拼通知 UI

```tsx
// ❌ 自己 useState 维护 visible + 自己 setTimeout 关闭
const [show, setShow] = useState(false);
function handleSave() {
  await save();
  setShow(true);
  setTimeout(() => setShow(false), 3000);
}
return (
  <>
    {show && <div className="fixed top-4 left-1/2 ...">保存成功</div>}
    <Button onClick={handleSave}>保存</Button>
  </>
);

// ✅ 一行调用 toast，全局生命周期统一
async function handleSave() {
  try {
    await save();
    toast.success("保存成功");
  } catch (e) {
    const id = Date.now();
    toast.error(() => <>{`保存失败：${e.message}`}{withClose(id)}</>, { id });
  }
}
```
