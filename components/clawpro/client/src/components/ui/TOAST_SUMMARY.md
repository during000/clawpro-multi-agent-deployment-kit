# Toast 组件实现总结

## 📋 项目文件清单

已为 ClawPro 设计系统创建完整的 Toast 组件实现。以下是所有相关文件：

### 核心实现文件

| 文件 | 位置 | 描述 |
|------|------|------|
| `sonner.tsx` | `src/components/ui/` | ✅ **推荐** - 基于 sonner 库的全局 Toast 管理器 |
| `toast.tsx` | `src/components/ui/` | ✅ **新增** - 独立的 Toast 组件（无依赖） |
| `toast.css` | `src/components/ui/` | ✅ **新增** - Toast 完整样式定义 |
| `TOAST_TOKENS.css` | `src/components/ui/` | ✅ **新增** - 设计令牌与 CSS 变量 |
| `TOAST_GUIDE.md` | `src/components/ui/` | ✅ **新增** - 完整使用指南与示例 |

---

## 🎨 设计规范亮点

### 视觉标准
```
背景        #FFFFFF (白)
文字色      #09090b (深灰/黑)
边框        #EAEEF4 (蓝灰)
圆角        12px (rounded-xl)
内边距      12px 16px (py-3 px-4)
字号/字重   14px / 500 (text-sm / font-medium)
文字对齐    左对齐
阴影        shadow-lg
定位        页面顶部居中 (top-center)
Z-Index     99999 (确保在 Dialog 之上)
自动消失    4000ms (4秒)
```

### 类型定义
| 类型 | 图标 | 颜色 | 说明 |
|------|------|------|------|
| `success` | ✓ 勾 | #10b981 (emerald) | 操作成功 |
| `error` | ! 感叹号 | #09090b (黑) | 操作失败 |
| `warning` | ⚠ 三角 | #f59e0b (amber) | 警告提示 |
| `info` | i 圆圈 | #3b82f6 (blue) | 普通提示 |

**关键规则**：所有类型统一白色背景 + 蓝灰边框，不按类型换底色。

### 关闭按钮
- **位置**：右上角外侧（`-right-2 -top-2`）
- **样式**：20×20 圆形，白底，#EAEEF4 边框
- **悬停**：背景变 #f4f4f5，文字变深灰
- **实现**：使用 `withClose(id)` 函数（自定义，不用 sonner 内置）

---

## 🚀 快速开始

### 方式 1：使用 sonner（推荐）

**1. 在 App 根组件挂载**
```tsx
import { Toaster } from "@/components/ui/sonner";

export function App() {
  return (
    <>
      <Toaster /> {/* 一次性挂载 */}
      {/* ... 其他组件 */}
    </>
  );
}
```

**2. 在业务代码中使用**
```tsx
import { toast, withClose } from "@/components/ui/sonner";

// 简单提示（自动消失）
toast.success("操作成功");
toast.error("操作失败");

// 需要用户关闭的长提示
const id = Date.now();
toast.error(() => <>{`失败：${error}`}{withClose(id)}</>, { id });
```

### 方式 2：使用独立 Toast（无 sonner）

**1. 使用 Hook（推荐）**
```tsx
import { useToast, ToastContainer } from "@/components/ui/toast";

export function MyPage() {
  const { toasts, showToast, removeToast } = useToast();

  async function handleSave() {
    try {
      await save();
      showToast("success", "保存成功");
    } catch (error) {
      showToast("error", `保存失败：${error.message}`);
    }
  }

  return (
    <>
      <ToastContainer toasts={toasts} onRemove={removeToast} />
      <button onClick={handleSave}>Save</button>
    </>
  );
}
```

**2. 使用单个组件**
```tsx
import { PortableToast } from "@/components/ui/toast";
import { useState } from "react";

export function MyPage() {
  const [showToast, setShowToast] = useState(false);

  return (
    <>
      {showToast && (
        <PortableToast
          type="success"
          message="保存成功"
          onClose={() => setShowToast(false)}
        />
      )}
      <button onClick={() => setShowToast(true)}>Save</button>
    </>
  );
}
```

---

## ⚙️ 项目集成

### 1. 复制文件到项目
```bash
# 已创建的文件
client/src/components/ui/toast.tsx        # 新增独立组件
client/src/components/ui/toast.css        # 新增样式
client/src/components/ui/TOAST_TOKENS.css # 新增令牌
client/src/components/ui/TOAST_GUIDE.md   # 新增文档

# 已存在，可选更新
client/src/components/ui/sonner.tsx       # 现有实现（保持不变）
```

### 2. 在全局样式中引入（可选）
```css
/* src/index.css */
@import "@/components/ui/toast.css";

/* 隐藏 sonner 内置关闭按钮残留 */
[data-sonner-toast] > [data-close-button] {
  display: none;
}
```

### 3. 类型声明（TypeScript）
所有类型已在 `toast.tsx` 中定义：
```typescript
type ToastType = "success" | "error" | "info" | "warning";
```

---

## 📖 使用示例

### 成功提示
```tsx
toast.success("数据保存成功");
showToast("success", "数据保存成功");
```

### 错误提示
```tsx
toast.error("保存失败，请重试");
const id = Date.now();
toast.error(() => <>{`保存失败：${err.msg}`}{withClose(id)}</>, { id });
showToast("error", "保存失败，请重试");
```

### 警告提示
```tsx
toast.warning("配额即将用尽");
showToast("warning", "配额即将用尽");
```

### 信息提示
```tsx
toast.info("数据已更新");
showToast("info", "数据已更新");
```

### 自定义持续时间
```tsx
toast.success("操作成功", { duration: 2000 }); // 2 秒
showToast("success", "操作成功", 2000);

toast.error(() => <>{msg}{withClose(id)}</>, { id, duration: 0 }); // 永不消失
```

### 异步操作反馈
```tsx
async function handleUpload() {
  const uploadPromise = api.upload(file);
  toast.promise(uploadPromise, {
    loading: "上传中...",
    success: "上传成功",
    error: "上传失败",
  });
}
```

---

## ✅ 强制规则（必须遵守）

### DO（正确做法）
- ✅ 所有类型统一白底 + 蓝灰边框
- ✅ 使用 `toast` API 或 `showToast` Hook
- ✅ 关闭按钮通过 `withClose(id)` 实现
- ✅ 关闭按钮在右上角外侧
- ✅ 文字左对齐
- ✅ 容器加 `relative` + `overflow-visible`

### DON'T（禁止做法）
- ❌ 按类型换底色（红底/绿底）
- ❌ 使用 sonner 内置 `closeButton` 属性
- ❌ 关闭按钮在其他位置（左侧/内部）
- ❌ 文字居中对齐
- ❌ 业务侧自行拼装通知 UI（useState + setTimeout）
- ❌ Z-index < 99999

---

## 🔍 QA 检查清单

在部署前检查以下内容：

- [ ] 白色背景 + #EAEEF4 边框
- [ ] 关闭按钮在右上角外侧（-right-2 -top-2）
- [ ] 文字左对齐
- [ ] 圆角 12px
- [ ] 内边距 12px 16px
- [ ] 字号 14px，字重 500
- [ ] 阴影 shadow-lg
- [ ] 定位顶部居中
- [ ] Z-index 99999
- [ ] 自动消失 4000ms
- [ ] 所有类型图标颜色正确
- [ ] sonner closeButton 已隐藏（若使用 sonner）
- [ ] Toast 容器有 relative + overflow-visible
- [ ] 移动端响应式正常

---

## 📚 文档结构

### `toast.tsx`
- `PortableToast` - 单个 Toast 组件
- `ToastContainer` - Toast 容器（管理多个）
- `useToast` - 状态管理 Hook

### `sonner.tsx`
- `Toaster` - 全局 Toast 管理器
- `toast` - Toast API（success/error/info/warning）
- `withClose(id)` - 自定义关闭按钮

### `toast.css`
- 基础 Toast 容器样式
- 类型变体样式
- 内部元素样式（图标、文字、关闭按钮）
- 动画效果
- 响应式适配
- 可访问性样式

### `TOAST_TOKENS.css`
- CSS 变量定义
- Tailwind 配置参考
- SCSS 变量示例
- 设计系统对齐说明
- 响应式断点
- 可访问性（WCAG 2.1 AA）
- QA 检查清单

### `TOAST_GUIDE.md`
- 整体架构说明
- 使用 sonner 的方式
- 使用独立 Toast 的方式
- 所有类型演示
- 高级用法
- 设计规范再确认
- 常见问题解答

---

## 🎯 后续步骤

1. **集成到项目**
   - 复制 `toast.tsx`、`toast.css`、`TOAST_TOKENS.css` 到项目
   - 在 `App.tsx` 挂载 `<Toaster />`

2. **全局引入**
   - 在 `src/index.css` 引入 `toast.css`
   - 添加隐藏 sonner 内置关闭按钮的 CSS

3. **业务代码迁移**
   - 找出现有的自定义 Toast/Alert 实现
   - 替换为统一的 `toast` API
   - 删除重复的通知 UI 代码

4. **质量检查**
   - 在各端（Desktop/Tablet/Mobile）测试
   - 验证样式符合设计规范
   - 无障碍测试（屏幕阅读器）

5. **文档维护**
   - 保持 `TOAST_GUIDE.md` 最新
   - 在设计系统更新时同步修改

---

## 📞 支持

如有问题，参考：
1. `TOAST_GUIDE.md` - 使用指南与常见问题
2. `TOAST_TOKENS.css` - 设计令牌与验证清单
3. 原始设计规范：`component-specs/toast.md`

---

**版本**：1.0  
**创建时间**：2026-06-11  
**最后更新**：2026-06-11  
**状态**：✅ 生产就绪
