# Tenant Reference

> 用户端页面规范。适用：`client/src/pages/tenant/**`、`client/src/components/topnav/**`、任何由 `TenantLayout` 包裹的页面。

## 1. 加载顺序

1. `foundation.md`
2. `components.md`
3. 本文件
4. 设计冲突与已确认裁决：`conflict-log.md`

## 2. 与全局 / 管控端差异

| 维度 | Tenant | Admin / 全局 |
|---|---|---|
| 按钮 | `tenant-*` 变体，全圆角 | `claw-*` 变体，4px |
| Tab | 按页面密度分流：Tenant 胶囊 Tabs / 既有基础控件；不定义 Text Switch | 4px Segment |
| 业务卡片 | 业务对象卡使用 12px `TenantCard` 三态系统 | 4px `SurfaceCard`，默认无阴影 |
| 数据密集容器 | 表格、弹窗、Popover、内嵌区块按组件 spec | 按 Admin spec |
| 顶部导航 | 当前实现为半透明 / backdrop blur，周一不作为阻塞项 | 管控端无 TopNav |
| 页面背景 | 白底 + 极淡蓝雾 | 管控端 `AdminSidebarInset` 背景 |
| 点阵装饰 | 已废弃，不新增点阵、贯穿线、装饰线 | 不适用 |

上述规则已按 `conflict-log.md` 收口；客户端专项项如后续由 owner 更新，以 `conflict-log.md` 新增条目回写。

## 3. TopNav

- 高度 64px，`fixed top-0 left-0 right-0 z-50`。
- 三栏 Grid：左 Logo、中间 Tabs、右操作区。
- 最小宽度 1200px，小屏横向滚动而非折行。
- 右侧操作顺序：使用指南、通知、管控端入口（管理员可见）、用户菜单。
- 图标跟随文字色，勿使用彩色 emoji。

## 4. 用户端布局骨架

默认业务页保留统一宽度节奏：

```tsx
<TenantLayout>
  <div className="min-w-[1200px] overflow-x-clip">
    <div className="max-w-[1920px] mx-auto flex items-stretch page-enter">
      <div aria-hidden className="shrink-0 w-20 self-stretch" />
      <main className="flex-1 min-w-0 px-[42px] py-8">...</main>
      <div aria-hidden className="shrink-0 w-20 self-stretch" />
    </div>
  </div>
</TenantLayout>
```

例外：登录、注册、重置密码等窄表单页不套 80px 占位带。

## 5. 按钮

| 场景 | 用法 |
|---|---|
| 页面主 CTA | `Button variant="tenant-primary" size="claw-lg"` |
| 卡片内次级操作 | `Button variant="tenant-outline" size="claw"` |
| 弹窗确认 | `tenant-primary` / `claw-sm` |
| 弹窗取消 | `tenant-outline` / `claw-sm` |
| 危险操作 | `tenant-destructive` |

禁止：`className="rounded-full"` 覆盖 `claw-*`。

## 6. Tab / Segment

- Tenant Text Switch 删除：当前没有引用该样式，周一交付包不再定义弱切换 Text Switch。
- Tenant 胶囊 Tabs / Segment 用于顶部导航、页面主分类或较强切换。
- 低密度局部切换复用已有基础控件或胶囊体系，不新增独立 Text Switch 视觉。
- 不把 Admin 4px 矩形 Segment 直接套进 Tenant 业务页；确需使用时按具体页面记录原因。
- 推荐 token：Inactive 使用 `--text-secondary`，active 文字使用 `--text-emphasis`，描边使用 `--border` 或组件专属 token。

## 7. 卡片

Tenant 业务对象卡按 12px `TenantCard` 三态系统执行：

1. 业务对象卡、Agent 卡、技能卡等展示型业务卡使用 `TenantCard`。
2. 表格容器、弹窗、Popover、数据密集容器按对应组件 spec 分流，不统一套 12px。
3. `TenantCard normal` 默认有 `--shadow-tenant-card`，hover 增强投影，static 无阴影。
4. 如果页面已接入 `SurfaceCard` 且与最新设计稿一致，不强行回退；但同一页面不要混用两套业务卡圆角系统。

## 8. Typography

用户端推荐语义组件：`TenantHeroTitle`、`TenantPageTitle`、`SectionTitle`、`PanelTitle`、`CardTitle`、`BodyText`、`MetaText`、`StatNumber`、`CodeText`。

当前确认口径为“推荐优先、渐进迁移”，不把所有 Tenant 文案强制改造为 Typography；新增或触达区域不要再散落写 `text-gray-900/700/500/400` 或基础文字色 hex。

## 9. 状态、空状态与表单控件

- 空状态沿用 Shared / Admin 的 `Empty` 层级，只在文案语气上更引导；不为 Tenant 单独新建一套空态体系。
- 空状态必须解释“为什么为空”和“下一步做什么”，不使用插画堆叠替代信息结构。
- 表单控件采用双轨：搜索 / 筛选控件可胶囊；普通表单、弹窗表单默认 4px；DatePicker 跟随所在场景。
- 危险操作同样使用二次确认。

## 10. 禁止事项

- 不使用管控端 `SurfaceConfig` 强调卡作为用户端默认业务卡。
- 不把管控端侧边栏 / 管控端矩形 Segment 套到用户端。
- 不通过 className 强改全局 Button、Card、Tabs 的核心样式。
- 不新增未登记图标；用 `assets-icons.md` 的流程登记。
