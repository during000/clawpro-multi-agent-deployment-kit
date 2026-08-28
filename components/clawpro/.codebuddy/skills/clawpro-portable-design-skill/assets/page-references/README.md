# Page References · 典型页面参考样本

> 用途：作为 LLM 在使用 `clawpro-portable-design-skill` 生成新页面时的"标杆样本"，让生成结果与现网页面在结构、组件选型、token 使用上保持一致。

## 1. 何时引用

在生成页面前，先按"页面分类"在本目录找最接近的样本，**结构骨架优先复用**，再对照 `references/page-recipes.md` 与 `component-specs/` 调整细节。

## 2. 样本清单（8 类典型页面）

| 类别 | 样本 | 路由 | 关键组合 | 详细 spec |
|---|---|---|---|---|
| 配置页 | 通道配置 | `/admin/channel-config` | PageHeader + LineTabs + Table（带行内 Switch） | `admin-channel-config.md` |
| 列表页（基础） | 用户管理 | `/admin/members` | PageHeader + Segment + Search + Toolbar + Table + Pagination | `admin-members.md` |
| 列表页（富功能） | Agent 列表 | `/admin/openclaw-monitor` | AlertBanner + 4 NumberCard（可选中筛选）+ Toolbar + 列头筛选 Table + 行操作 + Drawer | `admin-openclaw-monitor.md` |
| 空页面 | 资源管理（即将开放） | `/admin/agent-template` | 整页 EmptyState（占位插画 + 标题 + 描述） | `admin-agent-template.md` |
| 复杂列表页 | AI Agent 安全 | `/admin/security-management` | PageHeader + 3 NumberCard + LineTabs + 子表区（Toolbar + Empty/Table） | `admin-security-management.md` |
| 数据看板 | Tokens 监控 | `/admin/tokens-monitor` | AlertBanner + 5 NumberCard（含进度条变体）+ LineChart + LineTabs + 明细 Table | `admin-tokens-monitor.md` |
| 服务开通引导 | 运维观测（未开通态） | `/admin/ops-observation` | AlertBanner + PageHeader + CTA banner + 章节化 FeatureGrid（2 列） | `admin-ops-observation.md` |
| 能力开通页（动态 hero） | 开通云开发能力 | `/admin/cloud-dev` | DarkVeil 动态背景 hero（基底 + DarkVeil + 收束三件套，内容 `z-10`）+ 核心能力卡（FeatureGrid） | `admin-cloud-dev-activation.md` |

## 3. 文件命名约定

每个样本由两个文件组成（同 slug）：

- `admin-<slug>.md` —— 结构 spec（路由 / 视觉骨架 / 组件清单 / token 要点 / 易错点）
- `admin-<slug>.png` —— 视觉参考截图（1440×900 视口，dev 环境真实渲染）

> **补充（`admin-cloud-dev-activation`）**：该样本含 DarkVeil 动态背景，已按惯例提供同 slug 的 `admin-cloud-dev-activation.md`（样本卡）+ `admin-cloud-dev-activation.png`（dev 真截，逻辑 1440×900）。样本卡内部再外链两份更深的真相源：`references/admin-cloud-dev-activation.md`（页面骨架配方）与 `component-specs/dark-veil.md`（动态背景 spec / L0·L1·L2 兜底）。

## 4. 与既有资源的关系

| 资源 | 角色 |
|---|---|
| `references/page-recipes.md` | 抽象页面骨架（List / Form / Detail / Dashboard / Settings / Empty） |
| `component-specs/*.md` | 单组件规范（含 ✅/❌ 代码对照） |
| **`assets/page-references/*`（本目录）** | **真实页面快照 + 组合 recipe，作为生成参照** |
| `portable/html-css/*.html` | 跨仓 fallback 的纯 HTML/CSS demo |
| `portable/react/*.tsx` | 跨仓可复制的独立 React 组件 |

## 5. 更新规则

新增样本需满足：

1. 是当前 master/feature 分支上线版本的真实截图（非草稿/原型）
2. 视觉骨架包含 ≥3 种已规范化组件（spec 已存在）
3. 同时补 .md（结构 spec）+ .png（截图），并在本 README 表格登记

如果页面已大改但 `assets/page-references/` 还是旧版，请在 PR 描述里同步更新或删除过期样本。

## 6. 选样指引（生成时如何挑参考）

| 你的目标页面是… | 优先看 |
|---|---|
| 一个简单 CRUD 列表 | `admin-members.md` |
| 一个运维型富功能列表（带状态卡 / 列头筛选 / 行操作 / 抽屉详情） | `admin-openclaw-monitor.md` |
| 一个数据看板（大盘卡 + 时序图 + 多维度切换明细） | `admin-tokens-monitor.md` |
| 一个含 NumberCard + LineTabs 的复杂多视图列表 | `admin-security-management.md` |
| 一个有 Tabs 切分组的配置页（行内 Switch） | `admin-channel-config.md` |
| 一个尚未开放的占位页 | `admin-agent-template.md` |
| 一个"未开通服务、引导用户先开通"的页面 | `admin-ops-observation.md` |
| 一个**能力开通页 / hero 要动态流动光效**（已命中 DarkVeil Auto-Trigger、设计已拍板） | `admin-cloud-dev-activation.md`（样本卡 → 外链页面骨架 + `component-specs/dark-veil.md` L0 完整动态 hero 配方） |
| 同上但**宿主仓无 ogl/WebGL 或需静态导出**（动态背景退静态兜底） | `admin-cloud-dev-activation.md` →  `component-specs/dark-veil.md` §9（L1 静态 CSS / L2 纯色·截图兜底款） |
