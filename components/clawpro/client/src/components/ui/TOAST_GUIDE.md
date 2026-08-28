/**
 * Toast Component Implementation Guide
 * ─────────────────────────────────────────────────────────────────
 * 完整的 Toast 组件实现文档与使用示例。
 */

/* ═══════════════════════════════════════════════════════════════════
   ① 整体架构
   ═══════════════════════════════════════════════════════════════════

   项目中包含两套 Toast 实现：

   1. **sonner.tsx** - 基于 sonner 库（推荐）
      ├─ Toaster 组件：全局容器
      ├─ toast API：简便调用
      └─ withClose(id)：自定义关闭按钮

   2. **toast.tsx** - 独立实现（无依赖）
      ├─ PortableToast 组件：单个 toast
      ├─ ToastContainer 组件：容器管理
      └─ useToast Hook：状态管理

   ═══════════════════════════════════════════════════════════════════
*/

/* ═══════════════════════════════════════════════════════════════════
   ② 使用 sonner（推荐）
   ═══════════════════════════════════════════════════════════════════
*/

// 文件：src/main.tsx (或 App.tsx)
import { Toaster } from "@/components/ui/sonner";

export function App() {
  return (
    <>
      <Toaster /> {/* 一次性挂载，全局可用 */}
      {/* ... 其他组件 */}
    </>
  );
}

// 文件：src/pages/Users.tsx
import { toast, withClose } from "sonner";

export function UserForm() {
  async function handleSubmit(data) {
    try {
      await api.createUser(data);
      // ✅ 简单成功提示（自动消失）
      toast.success("用户创建成功");
    } catch (error) {
      // ✅ 带关闭按钮的错误提示（需要用户主动关闭）
      const id = Date.now();
      toast.error(
        () => (
          <>
            {`创建失败：${error.message}`}
            {withClose(id)}
          </>
        ),
        { id }
      );
    }
  }

  return <form onSubmit={handleSubmit}>{/* ... */}</form>;
}

/* ═══════════════════════════════════════════════════════════════════
   ③ 使用独立 Toast（无 sonner）
   ═══════════════════════════════════════════════════════════════════
*/

// 方式 A：单个 Toast 组件
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
          duration={4000}
          onClose={() => setShowToast(false)}
        />
      )}

      <button onClick={() => setShowToast(true)}>Save</button>
    </>
  );
}

// 方式 B：使用 useToast Hook（推荐）
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

/* ═══════════════════════════════════════════════════════════════════
   ④ 所有类型演示
   ═══════════════════════════════════════════════════════════════════
*/

// ✅ 成功
toast.success("操作成功");
showToast("success", "操作成功");

// ❌ 错误
toast.error("操作失败");
showToast("error", "操作失败");

// ⓘ 信息
toast.info("数据已更新");
showToast("info", "数据已更新");

// ⚠️  警告
toast.warning("配额即将用尽");
showToast("warning", "配额即将用尽");

/* ═══════════════════════════════════════════════════════════════════
   ⑤ 高级用法
   ═══════════════════════════════════════════════════════════════════
*/

// A. 自定义持续时间
toast.success("操作成功", { duration: 2000 }); // 2 秒
showToast("success", "操作成功", 2000);

// B. 长文本处理
const id = Date.now();
toast.error(
  () => (
    <>
      {`操作失败：${error.message}`}
      {withClose(id)}
    </>
  ),
  { id, duration: 0 } // duration: 0 = 永不自动消失，需要用户手动关闭
);

// C. 异步操作反馈
async function handleUpload() {
  const uploadPromise = api.upload(file);
  toast.promise(uploadPromise, {
    loading: "上传中...",
    success: "上传成功",
    error: "上传失败",
  });
}

// D. 多个 Toast 堆叠
showToast("info", "操作1");
showToast("success", "操作2");
showToast("warning", "操作3"); // 会自动堆叠显示

/* ═══════════════════════════════════════════════════════════════════
   ⑥ 设计规范再确认（强制规则）
   ═══════════════════════════════════════════════════════════════════
*/

/* ✅ DO - 正确做法 */

// 1. 所有类型统一白底 + 蓝灰边框（不按类型换底色）
toast.success("成功"); // → 白底 + 绿勾
toast.error("失败"); // → 白底 + 黑感叹号
toast.warning("警告"); // → 白底 + 橙感叹号
toast.info("信息"); // → 白底 + 蓝 i

// 2. 关闭按钮在右上角外侧（使用 withClose）
const id = Date.now();
toast.error(() => <>{`失败了`}{withClose(id)}</>, { id });

// 3. 文字左对齐
// ✓ 内置实现，自动左对齐

// 4. 顶部居中定位
// ✓ position: top-center 已配置

// 5. z-index >= 99999（在 Dialog 之上）
// ✓ 已配置为 99999

/* ❌ DON'T - 禁止做法 */

// 1. ❌ 按类型换底色
toast.error("错误", { className: "bg-red-600 text-white" });

// 2. ❌ 使用 sonner 内置 closeButton
<Toaster closeButton position="top-center" />;

// 3. ❌ 关闭按钮在其他位置
<button className="absolute -left-2 -top-2">×</button>;

// 4. ❌ 文字居中对齐
toast.success("成功", { className: "text-center" });

// 5. ❌ 自行拼装通知 UI
const [show, setShow] = useState(false);
return show ? <div>通知</div> : null;

/* ═══════════════════════════════════════════════════════════════════
   ⑦ 集成到项目
   ═══════════════════════════════════════════════════════════════════
*/

// Step 1: 在 App 根组件引入 Toaster
// src/App.tsx
import { Toaster } from "@/components/ui/sonner";

export function App() {
  return (
    <>
      <Toaster />
      {/* ... */}
    </>
  );
}

// Step 2: 在需要的地方导入并使用
// src/pages/Users.tsx
import { toast, withClose } from "@/components/ui/sonner";

// Step 3: 在样式中隐藏 sonner 内置关闭按钮残留（如果使用 sonner）
// src/index.css
[data-sonner-toast] > [data-close-button] {
  display: none;
}

/* ═══════════════════════════════════════════════════════════════════
   ⑧ TypeScript 类型定义
   ═══════════════════════════════════════════════════════════════════
*/

interface PortableToastProps {
  type: "success" | "error" | "info" | "warning";
  message: string;
  duration?: number; // 默认 4000ms
  onClose: () => void;
}

interface ToastItem {
  id: string | number;
  type: "success" | "error" | "info" | "warning";
  message: string;
  duration?: number;
}

interface ToastContainerProps {
  toasts: ToastItem[];
  onRemove: (id: string | number) => void;
}

/* ═══════════════════════════════════════════════════════════════════
   ⑨ 常见问题
   ═══════════════════════════════════════════════════════════════════

Q1: Toast 显示不出来？
A: 确保在 App 根组件挂载了 <Toaster />

Q2: 关闭按钮点击无效？
A: 检查 Toast 容器是否添加了 relative + overflow-visible

Q3: 多个 Toast 堆叠不对？
A: 确保使用 ToastContainer 或 Toaster，而不是多次调用组件

Q4: 样式不符合设计规范？
A: 检查是否改动了内置 classNames（px-4 py-3 应该是 12px 16px）

Q5: Toast 自动消失不工作？
A: 确保没有设置 duration: 0，或者检查 onClose 回调是否正确

   ═══════════════════════════════════════════════════════════════════
*/
