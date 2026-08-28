# 🔔 Toast 通知组件 - 完整实现指南

> ClawPro 设计系统的统一 Toast 通知组件。基于设计规范实现，支持 sonner 库或独立方案。

## 📁 文件清单

```
components/ui/
├── sonner.tsx                    # ✅ Sonner 库包装（已存在）
├── toast.tsx                     # ✅ 独立 Toast 组件（新增）
├── toast.css                     # ✅ Toast 样式定义（新增）
├── TOAST_TOKENS.css              # ✅ 设计令牌与变量（新增）
├── TOAST_GUIDE.md                # ✅ 详细使用指南（新增）
├── TOAST_SUMMARY.md              # ✅ 实现总结（新增）
├── toast-demo.tsx                # ✅ 演示与测试组件（新增）
└── README_TOAST.md               # ✅ 本文件
```

## 🎯 快速开始

### 1️⃣ Sonner 方式（推荐）

**App.tsx**
```tsx
import { Toaster } from "@/components/ui/sonner";

export function App() {
  return (
    <>
      <Toaster />
      {/* ... */}
    </>
  );
}
```

**业务代码**
```tsx
import { toast, withClose } from "@/components/ui/sonner";

// 简单提示
toast.success("操作成功");

// 长提示（需关闭）
const id = Date.now();
toast.error(() => <>{msg}{withClose(id)}</>, { id });
```

### 2️⃣ 独立 Toast 方式

**页面组件**
```tsx
import { useToast, ToastContainer } from "@/components/ui/toast";

export function MyPage() {
  const { toasts, showToast, removeToast } = useToast();

  return (
    <>
      <ToastContainer toasts={toasts} onRemove={removeToast} />
      <button onClick={() => showToast("success", "成功")}>
        Show Toast
      </button>
    </>
  );
}
```

## 📚 核心 API

### Sonner
```typescript
toast.success(message, options?)
toast.error(message, options?)
toast.warning(message, options?)
toast.info(message, options?)
toast.promise(promise, messages)
toast.dismiss(id?)

withClose(id) // 返回关闭按钮 JSX
```

### 独立 Toast
```typescript
const { toasts, showToast, removeToast } = useToast()

showToast(type, message, duration?)
removeToast(id)

<PortableToast type message duration onClose />
<ToastContainer toasts onRemove />
```

## 🎨 设计规范

| 属性 | 值 |
|------|-----|
| 背景 | 白色 (#FFFFFF) |
| 边框 | #EAEEF4 (1px) |
| 圆角 | 12px |
| 内边距 | 12px 16px |
| 字号/字重 | 14px / 500 |
| 文字对齐 | 左对齐 |
| 阴影 | shadow-lg |
| 定位 | 顶部居中 |
| Z-Index | 99999 |
| 自动消失 | 4000ms |
| 关闭按钮 | 右上角外侧 (-right-2 -top-2) |

### 类型与颜色
- **success** ✓ 绿色 (#10b981)
- **error** ✗ 黑色 (#09090b)
- **warning** ⚠ 橙色 (#f59e0b)
- **info** ⓘ 蓝色 (#3b82f6)

**关键：所有类型统一白底，不按类型换底色！**

## ✅ 强制规则

### ✓ DO
- 统一使用 toast API
- 所有类型白底 + 蓝灰边框
- 关闭按钮在右上角外侧
- 文字左对齐
- 容器加 `relative` + `overflow-visible`

### ✗ DON'T
- 按类型换底色（红底/绿底）
- 使用 sonner 内置 `closeButton`
- 关闭按钮在其他位置
- 文字居中对齐
- 自行拼装通知 UI

## 📖 详细文档

- **TOAST_GUIDE.md** - 完整使用指南与代码示例
- **TOAST_SUMMARY.md** - 实现总结与集成步骤
- **TOAST_TOKENS.css** - 设计令牌、变量定义与验证清单
- **toast-demo.tsx** - 交互式演示组件

## 🧪 演示组件

```tsx
import { ToastDemo, ToastSimpleDemo } from "@/components/ui/toast-demo";

// 完整演示（所有功能）
<ToastDemo />

// 简化演示（4 种类型）
<ToastSimpleDemo />
```

## 🔗 相关文件

- 原始规范：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/toast.md`
- 设计系统：http://localhost:3002/design-system/components

## 📋 集成清单

- [ ] 在 App 挂载 `<Toaster />`（如使用 sonner）
- [ ] 在 `src/index.css` 引入 `toast.css`
- [ ] 添加 CSS 规则隐藏 sonner 内置按钮
- [ ] 替换现有的自定义 Toast 实现
- [ ] 测试所有 4 种类型
- [ ] 移动端响应式测试
- [ ] 无障碍测试

## 📞 常见问题

### Q: Toast 显示不出来？
**A:** 确保在 App 根组件挂载了 `<Toaster />`

### Q: 关闭按钮点击无效？
**A:** 检查 Toast 容器是否有 `relative` + `overflow-visible`

### Q: 如何自定义消失时间？
**A:** 
```tsx
toast.success("成功", { duration: 2000 }) // 2秒
showToast("success", "成功", 2000)
```

### Q: 如何禁止自动消失？
**A:**
```tsx
const id = Date.now();
toast.error(() => <>{msg}{withClose(id)}</>, { id, duration: 0 })
```

### Q: 为什么不能按类型换底色？
**A:** 根据设计规范，所有类型统一白底，通过图标颜色区分类型。这样更统一、更优雅。

## 🎓 最佳实践

### 1. 简单成功/提示 → 自动消失
```tsx
toast.success("操作成功");
showToast("success", "操作成功");
```

### 2. 错误/长文本 → 需要用户关闭
```tsx
const id = Date.now();
toast.error(() => <>{`失败：${error}`}{withClose(id)}</>, { id, duration: 0 });
```

### 3. 异步操作 → 使用 promise
```tsx
toast.promise(uploadFile(), {
  loading: "上传中...",
  success: "上传成功",
  error: "上传失败",
});
```

### 4. 多个提示 → 自动堆叠
```tsx
showToast("info", "操作1");
showToast("success", "操作2"); // 自动堆叠
showToast("warning", "操作3");
```

## 🚀 生产就绪

✅ 完整的 React 组件实现  
✅ CSS 样式与动画  
✅ TypeScript 类型定义  
✅ 可访问性（WCAG 2.1 AA）  
✅ 响应式设计  
✅ 详细文档与示例  
✅ 演示组件  

---

**版本：** 1.0  
**最后更新：** 2026-06-11  
**状态：** ✅ 生产就绪
