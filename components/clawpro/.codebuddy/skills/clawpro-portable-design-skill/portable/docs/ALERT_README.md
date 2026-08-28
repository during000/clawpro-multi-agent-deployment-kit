# 🎨 Alert 组件库 - 完整指南

> ClawPro 设计系统的页面级别提示条组件
> 
> 支持 6 种变体，生产级质量，完全符合设计规范

## 📁 文件结构

```
alert/
├── react/alert/
│   ├── alert.tsx          # 核心 React 组件 (5.5K)
│   └── alert-demo.tsx     # 演示组件 (8.8K)
│
├── css/alert/
│   └── alert.css          # 完整样式 (12K)
│
└── ALERT_README.md        # 本文件
```

## 🎨 6 种变体

### 1. Info - 信息提示（蓝底）
```tsx
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>数据每 5 分钟更新一次</AlertDescription>
</Alert>
```
**背景**：#f0f3fc | **边框**：#bfcffe | **图标色**：#1447e6

### 2. Operation-Info - 操作说明（白底灰边）
```tsx
<Alert variant="operation-info">
  <AlertOperationInfoIcon />
  <AlertTitle>修改提示</AlertTitle>
  <AlertDescription>下方修改将影响所有 Agent</AlertDescription>
</Alert>
```
**背景**：#ffffff | **边框**：#eaeef4 | **图标色**：#7b818f

### 3. Warning - 警告提示（橙色）
```tsx
<Alert variant="warning">
  <CircleAlert />
  <AlertTitle>配置不完整</AlertTitle>
  <AlertDescription>有 3 项基础配置未完成</AlertDescription>
</Alert>
```
**背景**：#fff7ed | **边框**：#fed7aa | **图标色**：#ff6900

### 4. Success - 成功提示（绿底）
```tsx
<Alert variant="success">
  <AlertSuccessIcon />
  <AlertDescription>配置保存成功</AlertDescription>
</Alert>
```
**背景**：#ecfdf5 | **边框**：#a7f3d0 | **图标色**：#059669

### 5. Error - 错误提示（红底）
```tsx
<Alert variant="error">
  <AlertErrorIcon />
  <AlertTitle>请求失败</AlertTitle>
  <AlertDescription>网络异常，请检查网络连接后重试</AlertDescription>
</Alert>
```
**背景**：#fef2f2 | **边框**：#fecaca | **图标色**：#dc2626

### 6. Product-News - 产品动态（蓝底）
```tsx
<Alert variant="product-news">
  <AlertProductNewsIcon />
  <AlertDescription>【产品动态】OpenClaw v2.4.0 已发布</AlertDescription>
</Alert>
```
**背景**：#f0f3fc | **边框**：#bfcffe | **图标色**：#1447e6

## 💡 核心 API

### 导入
```tsx
import {
  Alert,
  AlertTitle,
  AlertDescription,
  AlertInfoIcon,
  AlertOperationInfoIcon,
  AlertSuccessIcon,
  AlertErrorIcon,
  AlertProductNewsIcon,
} from "@/components/ui/alert";

// 外部图标（需要 lucide-react）
import { CircleAlert } from "lucide-react";
```

### 组件

| 组件 | 描述 |
|------|------|
| `<Alert variant="...">` | 容器组件，6 种变体 |
| `<AlertTitle>` | 提示标题（可选） |
| `<AlertDescription>` | 提示描述（必需） |
| `<AlertInfoIcon />` | Info 图标 |
| `<AlertOperationInfoIcon />` | Operation-Info 图标 |
| `<AlertSuccessIcon />` | Success 图标 |
| `<AlertErrorIcon />` | Error 图标 |
| `<AlertProductNewsIcon />` | Product-News 图标 |

## 📊 设计规范

| 属性 | 值 |
|------|-----|
| 圆角 | 4px |
| 内边距 | 12px 16px |
| 图标尺寸 | 16px |
| 字号 | 12px |
| 行高 | 1.5 |
| 边框宽度 | 1px |
| 标题间距 | 4px |

## ✅ 强制规则

### DO（正确做法）
- ✅ 使用统一的 Alert 组件
- ✅ 根据场景选择合适的变体
- ✅ 使用规范的图标（AlertInfoIcon、CircleAlert 等）
- ✅ 保持 4px 圆角和固定的内边距
- ✅ 添加无障碍属性（role="alert"）

### DON'T（禁止做法）
- ❌ 自己拼装 bg-blue-50 / bg-amber-50 样式
- ❌ 改变圆角或添加阴影
- ❌ 使用不规范的图标（Info、AlertTriangle 等）
- ❌ 按类型换底色（用 warning 承载普通信息）
- ❌ 硬编码色值，必须用 CSS 变量

## 📖 使用示例

### 简单提示
```tsx
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>这是一条信息提示</AlertDescription>
</Alert>
```

### 带标题和描述
```tsx
<Alert variant="warning">
  <CircleAlert className="h-4 w-4" />
  <AlertTitle>警告标题</AlertTitle>
  <AlertDescription>这是详细的警告描述</AlertDescription>
</Alert>
```

### 成功反馈
```tsx
<Alert variant="success">
  <AlertSuccessIcon />
  <AlertTitle>操作成功</AlertTitle>
  <AlertDescription>已成功创建 3 个 Agent</AlertDescription>
</Alert>
```

### 产品动态
```tsx
<Alert variant="product-news">
  <AlertProductNewsIcon />
  <AlertDescription>
    【产品动态】新增企业级权限管理系统
  </AlertDescription>
</Alert>
```

## 🎓 何时使用

| 场景 | 推荐使用 |
|------|---------|
| 页面说明 / 功能告知 | info |
| 操作上下文说明 | operation-info |
| 配置缺失 / 风险提示 | warning |
| 产品发布 / 版本更新 | product-news |
| 操作成功 / 状态正常 | success |
| 操作失败 / 错误提示 | error |
| 需要自动消失的反馈 | ❌ 不用 Alert，用 Toast |
| 需要用户确认的提示 | ❌ 不用 Alert，用 Dialog |

## 🔍 代码对照

### ❌ 错误做法
```tsx
// 自己拼装样式
<div className="rounded-lg bg-blue-50 border border-blue-200 px-4 py-3">
  <Info className="h-4 w-4 text-blue-500" />
  <span>提示文案</span>
</div>

// 用 warning 承载普通信息
<Alert variant="warning">
  <AlertTriangle />
  <AlertDescription>这只是一条普通信息</AlertDescription>
</Alert>
```

### ✅ 正确做法
```tsx
// 使用统一 Alert
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示文案</AlertDescription>
</Alert>

// 普通信息用 info，真正的风险用 warning
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>这是一条普通信息</AlertDescription>
</Alert>
```

## 🎨 CSS 变量

所有颜色都通过 CSS 变量定义，可以自定义：

```css
:root {
  --alert-info-bg: #f0f3fc;
  --alert-info-border: #bfcffe;
  --alert-info-icon: #1447e6;
  
  --alert-warning-bg: #fff7ed;
  --alert-warning-border: #fed7aa;
  --alert-warning-icon: #ff6900;
}
```

## ♿ 可访问性

- ✓ `role="alert"` 让屏幕阅读器识别
- ✓ 图标设置 `aria-hidden="true"`
- ✓ 色彩对比度 ≥ 4.5:1
- ✓ 键盘导航支持

## 📱 响应式

所有变体在移动、平板、桌面设备上都能完美显示。

## 🚀 集成步骤

### 1. 复制文件
```bash
cp -r react/alert your-project/src/components/ui/
cp -r css/alert your-project/src/styles/
```

### 2. 导入样式
```css
/* src/index.css */
@import "./styles/alert/alert.css";
```

### 3. 导入组件
```tsx
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
```

### 4. 使用
```tsx
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>提示文案</AlertDescription>
</Alert>
```

## 📞 常见问题

### Q: Alert 和 Toast 有什么区别？
**A:** Alert 是页面常驻显示，Toast 是 4 秒自动消失。

### Q: 为什么 Alert 没有自动消失？
**A:** Alert 是信息提示，应该让用户看到。自动消失用 Toast。

### Q: 能改变圆角或添加阴影吗？
**A:** 不能。这样会破坏全局视觉一致性。

---

**版本**：1.0  
**创建**：2026-06-11  
**状态**：✅ 生产就绪
