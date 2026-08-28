# 空页面 · 资源管理（即将开放）(admin/agent-template)

> 类别：**整页 EmptyState（占位/即将开放页）**
> 路由：`/admin/agent-template`
> 源码：`client/src/pages/admin/ResourceManagement.tsx`（35 行，极简）
> 截图：`./admin-agent-template.png`（1440×900）

## 1. 视觉骨架

```
┌──────────────────────────────────────────────────────────────────┐
│ AdminLayout（保留 Sidebar + 顶部 Banner Alert）                   │
│   ┌──────────────────────────────────────────────────────────┐   │
│   │  flex 居中容器（min-h:calc(100vh-200px)）                │   │
│   │   ┌──────────────────────────────┐                       │   │
│   │   │  Empty illustration (120px)  │                       │   │
│   │   │  TenantPageTitle "资源管理"   │                       │   │
│   │   │  BodyText muted（2 行描述）   │                       │   │
│   │   └──────────────────────────────┘                       │   │
│   └──────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

## 2. 组件清单与 spec 对照

| 区域 | 组件 / 资产 | 来源 | spec |
|---|---|---|---|
| 占位插画 | `<img>` 引用 `empty-no-data.png` | `/assets/admin-resource-management/empty-no-data.png`（公共资源） | `component-specs/empty-state.md` §3 / `assets/icon-registry.example.json` |
| 标题 | `TenantPageTitle` | `@/components/ui/Typography` | Typography 通用规范 |
| 描述 | `BodyText tone="muted"` | `@/components/ui/Typography` | 同上 |

> ⚠️ 这里**没有**用 shadcn `Empty` 组件，而是自行用 `flex + 图 + Typography` 拼装。原因是文件源码注释里说明：占位页要走"语义 token + 极简骨架"，避免引入 Empty 卡片边框/SurfaceCard。

## 3. 实现参考（35 行完整源码 ≈ 可直接复用）

```tsx
import { TenantPageTitle, BodyText } from "@/components/ui/Typography";

export default function ResourceManagement() {
  return (
    <div className="page-enter flex h-full min-h-[calc(100vh-200px)] items-center justify-center">
      <div className="flex max-w-[590px] flex-col items-center text-center">
        <img
          src="/assets/admin-resource-management/empty-no-data.png"
          alt=""
          aria-hidden="true"
          className="mb-4 h-[120px] w-auto shrink-0 object-contain"
        />
        <div className="w-full">
          <TenantPageTitle className="mb-[10px]">资源管理</TenantPageTitle>
          <BodyText tone="muted">
            在此统一管理企业内可复用的 Agent 模板，包括预设的系统提示词、工具配置与模型参数。
            <br />
            管理员可发布模板供用户一键创建标准化 Agent，降低配置门槛，保障使用规范。
          </BodyText>
        </div>
      </div>
    </div>
  );
}
```

## 4. 关键 token / 规范要点

- 容器 `min-h:calc(100vh-200px)` 让插画**视觉居中**（减去 Header + 底部计费条高度）
- 插画固定高度 `120px`（不是 144/160）；宽度 auto；`object-contain` 保持比例
- 标题用 `TenantPageTitle`（`tone="primary"` 默认）→ 绑定 `--text-title`
- 描述用 `BodyText tone="muted"` → 绑定 `--text-muted`（**比 default 更浅**的辅助语义色）
- 字号、颜色**全部走 Typography**，禁止硬编码 `text-xl text-gray-500` 等

## 5. 为何典型 (why-typical)

- "即将开放 / 暂无入口"类页面的最小集，直接复制 35 行源码即可上线
- 不引入 Empty 组件，避免在"整页占位"语义下出现卡片描边/阴影感
- 演示了**资产路径规范**：业务专属插画放在 `/public/assets/<feature-name>/<purpose>.png`，并在 `assets/icon-registry.example.json` 登记

## 6. 易错点 / 反例

| ❌ 反例 | ✅ 正例 |
|---|---|
| 直接用 `<Empty>...</Empty>` 包成卡片 | 用 flex 居中 + Typography（保持空旷无边） |
| 标题用 `<h1 className="text-2xl">资源管理</h1>` | `<TenantPageTitle>资源管理</TenantPageTitle>` |
| 描述用 `<p className="text-gray-500">...</p>` | `<BodyText tone="muted">...</BodyText>` |
| 把插画放在 `client/src/assets/` 引入 | 放在 `client/public/assets/<feature>/...` 通过 URL 引用 |
| 插画给 `width:200px height:200px` 拉伸 | 固定 height + auto width + `object-contain` |
| 占位文案写 "暂无数据" 一句 | 写**业务说明**（功能价值 + 适用场景），让用户理解未来能力 |
