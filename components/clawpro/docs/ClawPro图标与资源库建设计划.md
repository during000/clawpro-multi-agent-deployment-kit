# ClawPro 图标与资源库建设计划

> 适用项目：`/Users/miekoyychen/openclaw-enterprise`  
> 文档定位：说明 ClawPro 资源库的单页功能、数据产物、重复治理、canonical 入口、组件资源映射与风险约束、skill 连接、安全落地和实施阶段。  
> 前置阅读：执行本计划前，必须先阅读 `docs/ClawPro图标与资源库背景目标.md`，确认项目背景、资源范围、分类体系和本阶段目标边界。  
> 配套沉淀（阶段 9 起）：`docs/ClawPro资源库-阶段9决策溯源(ADR).md`（决策背景与硬证据，接手前必读）、`docs/ClawPro资源库-新资源接入SOP.md`（后续新增资源的可执行流程）。

---

# 一、计划目标

本计划基于 `docs/ClawPro图标与资源库背景目标.md`，落地以下能力：

```text
项目资源审查治理
+ 资源库单页
+ 重复资源实际治理
+ 静态资源数据产物
+ canonical 资源统一入口
+ 使用自有 SVG 的组件资源映射与风险约束
+ skill-map 连接
+ 安全校验与验收
```

本计划不改变背景目标文档中的边界：

- 不做私有 npm 包。
- 不做全量资源迁移。
- 不做强注册体系。
- 不做在线上传 / 删除 / SVG 编辑。
- 不做全量组件重构。

---

# 二、项目资源审查治理计划

项目资源审查治理是资源库落成的前置工作。资源库单页不是先凭空做页面，而是先完成项目资源审查、分类、重复判断、风险分级和治理记录，再把结果沉淀为页面数据、canonical 入口、组件资源使用映射和 skill-map。所有组件均视为已被开发仓库使用的共享组件，本阶段不修改组件源码。

## 2.1 审查目标

从当前阶段到资源库落成，资源审查治理需要完成以下目标：

1. 盘清项目内所有纳入范围的图标和图片资源。
2. 明确每个资源的来源目录、资源类型、使用位置和使用次数。
3. 对自有 SVG 补充使用场景和视觉类型。
4. 识别重复资源、疑似重复资源、未使用资源和待确认资源。
5. 对可安全处理的重复资源做实际治理。
6. 选出高频 canonical 资源，并建立 `canonicalAssets` 统一入口。
7. 梳理使用自有 SVG / 图片资源的组件，建立资源使用映射与风险约束；继续使用 `lucide-react` 的组件不纳入改造。
8. 输出 `resource-skill-map.json`，让 skill 在当前项目页面生成时可以稳定选择资源。
9. 形成治理报告，记录已处理、暂不处理和待设计确认的资源。

## 2.2 审查对象

审查对象包括：

| 对象 | 审查重点 |
|---|---|
| `client/public/assets/**` | 是否有重复 SVG / PNG、是否属于业务高频资源 |
| `client/public/assets/avatars/**` | Agent 头像是否重复、命名是否清晰、是否需要进入 canonical 入口 |
| `client/public/assets/admin-channel-icons/**` | 渠道图标是否固定色、是否被错误当作普通 icon 使用 |
| `client/public/icon/**` | 旧图标是否与其他目录重复、是否仍被引用 |
| `client/src/assets/**` | import 型资源是否与 public 资源重复 |
| `icon/**` | 根目录旧素材是否仍有价值、是否应归档或清理 |
| `assets/**` | Figma / CodeBuddy 导出资源是否为临时资源、是否污染当前资源体系 |
| TSX 内 inline SVG | 是否应保留组件私有、替换为已有资源、或抽为资源库资源 |
| `lucide-react` import | 当前使用哪些 lucide 图标，是否可进入 skill-map 候选 |

## 2.3 审查维度

每个资源至少按以下维度审查：

| 维度 | 说明 |
|---|---|
| 资源类型 | SVG、PNG、Lucide、Agent 头像、渠道图标、品牌 Logo、业务图片 |
| 来源路径 | 原始文件路径或 lucide import 名称 |
| 使用位置 | 被哪些页面、组件、样式或配置引用 |
| 使用次数 | 未使用、单处使用、多处使用 |
| 使用场景 | navigation、action、status、metric、agent、model-ai 等 |
| 视觉类型 | line、solid、gradient、brand-fixed、monochrome-currentColor 等 |
| 端别 | Admin、Tenant、Landing、Global |
| 重复状态 | 非重复、疑似重复、确认重复 |
| canonical 状态 | 是否为推荐保留资源，是否已接入统一入口 |
| 治理状态 | normal、duplicate、resolved、needs-review、avoid、deprecated |
| 安全状态 | SVG 是否存在脚本、事件属性、外链、`foreignObject` 等风险 |

## 2.4 审查组织

资源审查结果需要分为以下处理组：

| 组织 | 定义 | 后续动作 |
|---|---|---|
| 可直接保留 | 分类清晰、无重复、仍在使用 | 进入资源库展示，可按需进入 skill-map |
| 高频 canonical | 使用次数高或页面层 / 资源库中常用 | 接入仅供当前项目页面层使用的 `canonicalAssets`，优先进入 skill-map |
| 可自动清理 | 完全重复、未被引用、明显临时产物 | 删除并记录到治理报告 |
| 可半自动治理 | 重复明确但存在引用 | 小范围替换引用到 canonical，再删除或标记旧资源 |
| 待设计确认 | 视觉近似、语义不明、颜色不同或品牌相关 | 页面标记 `needs-review`，不自动处理 |
| 暂不纳入 | Landing 大图、页面截图、临时设计素材等 | 不进入资源库展示或仅进入排除报告 |
| 风险资源 | SVG 存在脚本、事件属性、外链等风险 | 阻断入库，必须先修复或移除 |

## 2.5 治理动作

治理动作必须可 review、可回滚、可记录。

| 动作 | 适用范围 | 要求 |
|---|---|---|
| 删除未使用重复文件 | A 类重复资源 | 只删除确认未引用资源，记录原因 |
| 替换重复引用 | B 类重复资源 | 小范围替换到 canonical，避免大 diff |
| 标记待确认 | C 类资源 | 不删除、不替换，进入页面待确认视图 |
| 建立 canonical 入口 | 高频或已确认 canonical 资源 | 写入 `client/src/design-assets/canonical-assets.ts`，仅供当前项目页面层使用 |
| 建立组件资源映射 | 使用自有 SVG / 图片资源的组件 | 记录组件槽位、当前资源、风险等级和推荐资源；不修改组件源码 |
| 更新 skill-map | skill 可用资源 | 只纳入稳定、非 deprecated、非 needs-review 资源 |
| 输出治理报告 | 所有治理动作 | 记录处理前后、原因、影响范围和暂不处理项 |

## 2.6 审查治理产物

资源审查治理阶段需要输出：

```text
client/src/design-assets/generated/resource-inventory.generated.json
client/src/design-assets/generated/resource-usage.generated.json
client/src/design-assets/generated/resource-duplicates.generated.json
client/src/design-assets/generated/resource-governance.generated.json
client/src/design-assets/canonical-assets.ts
client/src/design-assets/resource-skill-map.json
docs/resource-governance-report.md
```

其中：

- `resource-inventory.generated.json` 是资源库单页的基础数据。
- `resource-usage.generated.json` 记录使用位置和使用次数。
- `resource-duplicates.generated.json` 记录重复组。
- `resource-governance.generated.json` 记录治理状态。
- `canonical-assets.ts` 承接当前项目页面层高频 canonical 资源的一改多处生效能力，不供共享组件源码直接依赖。
- `resource-skill-map.json` 承接组件槽位和 skill 资源选择。
- `docs/resource-governance-report.md` 记录本次治理的审查结论和实际处理。

## 2.7 审查治理完成标准

资源审查治理完成后，应满足：

1. 纳入范围内的资源均有清单记录。
2. 资源使用位置和使用次数可在页面详情查看。
3. 自有 SVG 已有初始使用场景和视觉类型。
4. 重复资源已组织，并区分 A / B / C 类治理策略。
5. A 类重复资源已清理。
6. B 类明确重复资源已小范围治理，或在报告中说明暂不治理原因。
7. C 类资源已进入待确认清单。
8. 高频 canonical 资源已接入 `canonicalAssets`，或说明暂不接入原因。
9. 使用自有 SVG / 图片资源的组件已建立资源使用映射和风险约束。
10. 继续使用 `lucide-react` 的组件保持现状，不做资源库改造。
11. skill-map 不包含 `avoid`、`deprecated`、`needs-review` 资源。
12. SVG 安全检查通过。
13. 资源库单页可以展示审查治理结果。

---

# 三、资源库单页功能设计

## 3.1 页面定位

新增单页：

```text
/design-system/assets
```

页面名称：

```text
ClawPro 资源库
```

与现有全局组件展示台并列：

```text
/design-system/components  全局组件展示台
/design-system/assets      图标与资源库
```

页面定位：

> 面向产品设计团队、产品前端和 vibe coding 流程的资源浏览、分类、重复治理和使用参考页面。

页面不承担在线上传、删除、编辑 SVG path、发布资源包等职责。

## 3.2 信息架构

页面采用 Admin / Design System 场景布局，建议结构：

```text
顶部 PageHeader
  - 标题：ClawPro 资源库
  - 描述：汇总当前项目中的图标与高频图片资源，支持分类浏览、重复治理和使用参考
  - 操作：重新扫描说明 / 查看治理报告 / 复制 skill-map 路径

统计总览
  - 总资源数
  - 自有 SVG 数量
  - 重复资源组数量
  - 未分类 / 待确认资源数量

主体区域
  - 左侧：分类导航
  - 中间：资源卡片网格 / 列表
  - 右侧：资源详情面板
```

## 3.3 顶部总览区

顶部总览区只保留 3-4 个核心指标，用于快速回答“资源规模、重点资源、治理风险”。更细的数据放到分类导航、筛选项和详情面板中，不在顶部堆叠。

建议指标：

| 指标 | 说明 |
|---|---|
| 总资源数 | 所有纳入资源库展示的资源数量 |
| 自有 SVG | ClawPro 自有 SVG 图标数量，是本次分类和治理重点 |
| 重复资源组 | 疑似或确认重复的资源组数量，体现治理工作量 |
| 未分类 / 待确认 | 仍需设计团队确认分类、语义或 canonical 的资源数量 |

可选替换项：如果后续更关注治理进度，可以用“已治理重复组”替代“自有 SVG”或“未分类 / 待确认”，但顶部总数不超过 4 个。

## 3.4 左侧分类导航

左侧分类是页面的主要浏览入口。分类定义来自 `docs/ClawPro图标与资源库背景目标.md`。

建议结构：

```text
全部资源
Lucide 图标
自有 SVG 图标
  导航入口
  操作行为
  搜索筛选
  状态反馈
  数据指标
  Agent 业务
  模型 / AI
  安全权限
  文件资源
  计费配额
  品牌产品
  轻量提示
Agent 头像
渠道图标
品牌 Logo
业务图片
重复治理
未分类 / 待确认
```

交互要求：

1. 点击分类后，中间列表只展示该分类资源。
2. 自有 SVG 支持展开 / 收起二级分类。
3. 分类项展示数量，例如 `操作行为 24`。
4. `重复治理` 展示重复组数量，而不是资源数量。
5. `未分类 / 待确认` 应高亮提醒，但不阻断页面使用。

## 3.5 筛选与搜索区

筛选区位于主体上方，服务设计团队快速定位资源。

建议筛选项：

| 筛选项 | 说明 |
|---|---|
| 搜索 | 支持名称、文件名、路径、标签 |
| 资源类型 | SVG、PNG、Lucide、Avatar、Logo、Channel Icon |
| 使用场景 | navigation、action、status、metric 等 |
| 视觉类型 | line、solid、gradient、brand-fixed 等 |
| 端别 | Admin、Tenant、Landing、Global |
| 来源目录 | `client/public/assets`、`client/src/assets`、`assets` 等 |
| 使用次数 | 未使用、单处使用、多处使用 |
| 治理状态 | 正常、重复、已治理、待确认、不建议使用、已废弃 |
| canonical 状态 | canonical、非 canonical、已接入统一入口、未接入统一入口 |

交互要求：

1. 搜索与筛选条件可以组合。
2. 筛选条件以标签形式展示，支持一键清空。
3. 默认不展示“推荐复用”视图，但 `preferred` 资源在卡片上显示标签。
4. 搜索不到结果时使用统一 Empty 组件，不手写空态。

## 3.6 资源展示区

资源展示区默认使用卡片网格，必要时提供列表模式。

### 卡片网格

每张资源卡片展示：

- 资源预览；
- 资源名称；
- 文件类型；
- 来源路径简写；
- 使用次数；
- 使用场景；
- 视觉类型；
- 状态标签；
- 是否属于重复组；
- 是否已治理；
- 是否 canonical；
- 是否已接入统一入口。

卡片状态标签示例：

```text
优先使用
重复资源
已治理
待确认
不建议使用
品牌固定色
可改色
canonical
已接入统一入口
```

### 列表模式

列表模式用于资源较多、需要横向比较时使用。

建议列：

| 列 | 内容 |
|---|---|
| 预览 | 资源缩略图 |
| 名称 | displayName / 文件名 |
| 类型 | SVG / PNG / Lucide 等 |
| 使用场景 | 主分类 |
| 视觉类型 | line / gradient / brand-fixed 等 |
| 来源路径 | 文件路径或 lucide import |
| 使用次数 | 引用数量 |
| 状态 | normal / duplicate / resolved 等 |
| canonical 状态 | canonical / 非 canonical |
| 统一入口状态 | 已接入 / 未接入 |

## 3.7 资源详情面板

点击资源后打开右侧详情面板。详情面板是单页的核心管理能力。

### 基础信息

展示：

- 名称；
- ID；
- 文件名；
- 来源路径；
- 资源类型；
- 文件尺寸；
- 文件大小；
- 来源目录；
- 使用次数；
- 最近扫描时间。

### 视觉预览

展示：

- 原始预览；
- 浅色背景预览；
- 深色背景预览；
- 16px / 20px / 24px / 32px / 48px 尺寸预览；
- 是否支持 `currentColor`；
- 是否固定色；
- 是否适合改色。

### 分类信息

展示：

- 一级分类；
- 使用场景；
- 辅助使用场景；
- 视觉类型；
- 端别；
- 标签；
- 资源状态；
- 是否可进入 skill-map。

### canonical 信息

展示：

- 是否为 canonical 资源；
- canonical ID；
- 是否已接入 `canonicalAssets`；
- `canonicalAssets` key；
- 当前仍有哪些使用位置没有接入统一入口；
- 修改该资源预计会影响哪些引用位置。

### 使用位置

展示当前项目中的引用位置：

```text
client/src/pages/admin/xxx.tsx
client/src/components/yyy.tsx
```

要求：

1. 支持复制文件路径。
2. 支持按文件组织展示。
3. 未使用资源要明确显示“当前未发现使用位置”。

### 复制用法

根据资源类型提供不同复制方式。

Lucide：

```tsx
import { Search } from "lucide-react";

<Search className="size-4" />
```

public 资源：

```tsx
<img src="/assets/xxx.svg" alt="" />
```

src asset：

```tsx
import iconUrl from "@/assets/xxx.svg";
```

canonical 入口：

```tsx
import { canonicalAssets } from "@/design-assets/canonical-assets";

<img src={canonicalAssets.icons.search} alt="" />
```

业务组件：

```tsx
<AgentAvatar avatarId="developer" />
<ChannelIcon channel="wecom" />
<BrandLogo name="openclaw" />
```

说明：复制用法只作为参考，不代表强制迁移。

### 组件槽位

展示这个资源适合哪些组件槽位。下表为**阶段 8 组件资源审计产出的可用槽位白名单**（事实源 `client/src/design-assets/manual-overrides/component-resource-map.json`，资源库详情面板据此展示风险约束）：

| 槽位 | 组件槽位路径 | 推荐资源类型 | lucide fallback | 风险 | 红线 |
|---|---|---|---|---|---|
| AdminSidebar 菜单图标 | `AdminSidebar.itemIcon` | custom-svg | 可 | 低 | 否 |
| 卡片左侧图标 | `Card.leftIcon` | custom-svg | 不可 | 中 | 否 |
| NumberCard 卡片图标 | `NumberCard.icon` | custom-svg | 不可 | 低 | 否 |
| 文件类型图标 | `FileSpace.fileTypeIcon` | custom-svg / lucide | 可 | 中 | 否 |
| 运行状态图标 | `AgentCard.statusIcon` | custom-svg | 不可 | 中 | 否 |
| 企业特性卡片图标 | `ComparisonTable.featureIcon` | custom-svg | 不可 | 高 | 否 |
| Agent 头像 | `AgentCard.avatar` | agent-avatar | 不可 | 高 | 是 |
| 渠道图标 | `ChannelConfig.channelIcon` | channel-icon | 不可 | 低 | 是 |
| 品牌 Logo | `BrandLogo.logo` | brand-logo | 不可 | 中 | 是 |

> 仅使用 `lucide-react` 的通用槽位（如按钮前置图标、搜索框图标、表格操作图标）不纳入资源库槽位白名单，保持 lucide。红线槽位（头像 / 渠道 / 品牌）禁当普通 UI 图标改色、禁进普通 icon 槽位、跨仓由宿主注入。`run-status` / `file-type` 当前由组件内 inline SVG 实现，对应静态文件多为设计源（详见阶段 8 基线小节的据实发现）。

> **修订补注（2026-06-22，据阶段 9 真实产出）**：`number-card` 槽位 `lucide fallback` 由「可」翻为「不可」（`recommendedResourceType` 去 lucide → `custom-svg`）。原标「可」是因 NumberCard 内置仅 4 枚 inline 渐变图标、怕 AI 选不到合适项才允许 lucide 兜底；阶段 9 派生候选已扩至 **22 枚渐变图标**（OpenClawMonitor 抽出 `monitor-*` 并复用 `instance-total` 后槽位充足），扁平 lucide 反而破坏渐变家族，故据实撤回兜底、对齐 `card-left-icon`：无合适候选时标 `needs-design-confirmation` 交设计补绘，不回退 lucide、不手搓 inline svg。属计划 §5 待定项允许的合法修订（「是否允许 lucide 回退」以审计真实产出为准）；事实源 `component-resource-map.json` 已翻转、下游 `resource-skill-map.json` 经 `build:skill-map` 重生成、`check:skill-map` 防漂移校验通过。

### 重复治理信息

如果资源属于重复组，展示：

- 重复组编号；
- 重复资源列表；
- 相似原因；
- canonical 建议；
- 当前治理状态；
- 已执行治理动作；
- 后续处理建议。

## 3.8 重复治理视图

`重复治理` 是一级入口，必须做实际治理，不只是告警。

展示粒度为“重复组”，不是单个资源。

每个重复组展示：

| 信息 | 说明 |
|---|---|
| 重复组编号 | 例如 `duplicate-group-012` |
| 并排预览 | 方便设计团队判断是否真的重复 |
| 资源路径 | 每个候选资源的路径 |
| 使用次数 | 判断哪个资源更适合作为 canonical |
| 相似原因 | 文件 hash 相同、SVG 归一化 hash 相同、文件名相似等 |
| canonical 建议 | 建议保留的资源 |
| 治理状态 | 待确认、已替换引用、已删除重复等 |
| 操作记录 | 本次任务中实际做过什么 |

治理状态：

```text
pending-review       待确认
canonical-confirmed  已确认 canonical
references-updated   已替换引用
duplicates-removed   已删除重复
keep-multiple        保留多个
no-action            无需处理
```

## 3.9 未分类 / 待确认视图

用于承接扫描无法自动判断的资源。

展示内容：

- 资源预览；
- 来源路径；
- 文件类型；
- 文件大小；
- 使用位置；
- 猜测分类；
- 需要确认的问题。

该视图的目标是让设计团队可以集中处理分类问题，而不是散落在代码 review 中讨论。

---

# 四、重复资源治理方案

本次任务仍需要做实际治理，但治理范围应可控，避免变成全项目资源迁移。

## 4.1 重复识别方式

| 方式 | 说明 |
|---|---|
| 文件 hash | 找完全相同文件 |
| SVG 归一化 hash | 去掉格式差异后识别相同 SVG |
| 文件名相似度 | 找命名近似资源 |
| 尺寸相同 + 路径相似 | 找疑似重复图片 |
| 视觉并排预览 | 供设计师人工确认 |
| 使用位置统计 | 辅助判断 canonical |

## 4.2 治理分级

### A 类：可自动治理

满足以下条件可以直接治理：

- 文件 hash 完全一致；
- 资源未被业务引用；
- 明显是临时导出或重复副本；
- 删除后不影响运行。

动作：

1. 删除重复未使用文件。
2. 在治理报告中记录删除原因。
3. 页面中不再展示已删除资源，只在治理报告保留记录。

### B 类：可半自动治理

满足以下条件：

- 文件内容完全一致或 SVG 归一化 hash 一致；
- 多处路径存在；
- 有业务引用；
- 可以明确选出 canonical。

动作：

1. 保留 canonical 资源。
2. 小范围替换引用到 canonical。
3. 删除或标记重复副本。
4. 每组单独 review，避免大面积回归。

### C 类：需要人工确认

满足以下条件：

- SVG 形状近似但不完全相同；
- 名称类似但视觉不同；
- 颜色不同；
- 品牌 / 渠道资源；
- 设计语义不明确。

动作：

1. 页面标记为 `needs-review`。
2. 不自动删除。
3. 不自动替换。
4. 由设计团队确认后再进入后续治理。

## 4.3 治理交付物

本次资源库任务必须交付：

1. 资源扫描结果。
2. 重复资源组织。
3. A 类重复资源清理结果。
4. B 类明确重复资源的小范围引用替换结果。
5. C 类待确认清单。
6. 治理报告。
7. `/design-system/assets` 页面展示治理状态。

建议治理报告路径：

```text
docs/resource-governance-report.md
```

---

# 五、数据产物设计

页面和 skill 都不应该直接扫描源码运行时获取数据，而应消费静态数据产物。

## 5.1 页面数据

建议生成：

```text
client/src/design-assets/generated/resource-inventory.generated.json
client/src/design-assets/generated/resource-usage.generated.json
client/src/design-assets/generated/resource-duplicates.generated.json
client/src/design-assets/generated/resource-governance.generated.json
```

用途：

| 文件 | 用途 |
|---|---|
| `resource-inventory.generated.json` | 页面展示完整资源清单 |
| `resource-usage.generated.json` | 展示资源使用位置和使用次数 |
| `resource-duplicates.generated.json` | 展示重复资源组织 |
| `resource-governance.generated.json` | 展示治理状态和已执行动作 |

## 5.2 资源清单字段建议

资源清单不是强 Registry，但需要结构化字段支撑页面和 skill-map。

```ts
interface ResourceInventoryItem {
  id: string;
  displayName: string;
  type: "svg" | "png" | "lucide" | "agent-avatar" | "channel-icon" | "brand-logo" | "business-image";
  source: "public" | "src" | "lucide-react" | "inline-svg" | "root-assets";
  sourcePath?: string;
  importName?: string;
  primaryCategory: string;
  usageScenarios: string[];
  visualType?: string;
  scenes: Array<"admin" | "tenant" | "landing" | "global">;
  tags: string[];
  usageCount: number;
  status: "normal" | "preferred" | "duplicate" | "resolved" | "needs-review" | "avoid" | "deprecated";
  duplicateGroupId?: string;
  canonicalId?: string;
  canRecolor?: boolean;
  canUseInSkillMap?: boolean;
}
```

## 5.3 canonical 资源入口

如果希望某个高频图标修改后影响所有使用处，需要为已确认的 canonical 资源建立统一入口。

本阶段不要求所有资源都接入统一入口，只对以下资源优先处理：

1. 高频使用资源；
2. 已完成重复治理并确认 canonical 的资源；
3. 组件槽位高频使用资源；
4. Agent 头像、渠道图标、品牌 Logo 等业务专属资源；
5. 后续 skill 会稳定生成的资源候选。

建议新增轻量入口：

```text
client/src/design-assets/canonical-assets.ts
```

实际产出（阶段 6 已据治理产出回填，与 `client/src/design-assets/canonical-assets.ts` 对齐）：

阶段 6 实际纳入 **14 个 key**，组织维度按治理后存活的业务专属类目收敛为 `brands / channels / avatars`（仅收已确认 `normal`、运行时可服务的 public 资源；多处使用但仍 `needs-review` 的 `/icon/*.svg` 不纳入）：

```ts
export const canonicalAssets = {
  brands: {
    clawproLogo: "/assets/admin-sidebar/clawpro-logo.svg",   // dup-075 红线，不归并
  },
  channels: {
    wechat: "/assets/admin-channel-icons/channel-wechat.svg",
    qq: "/assets/admin-channel-icons/channel-qq.svg",
    wecom: "/assets/admin-channel-icons/channel-wecom.svg",       // dup-074 红线
    wecomApp: "/assets/admin-channel-icons/channel-wecom-app.svg", // 与 wecom 字节一致，仍分列不归并
    dingtalk: "/assets/admin-channel-icons/channel-dingtalk.svg",
    feishu: "/assets/admin-channel-icons/channel-feishu.svg",
  },
  avatars: {
    default: "/assets/avatars/avatar-default.png",
    designer: "/assets/avatars/avatar-designer.png",
    analyst: "/assets/avatars/avatar-analyst.png",
    creator: "/assets/avatars/avatar-creator.png",
    developer: "/assets/avatars/avatar-developer.png",
    pm: "/assets/avatars/avatar-pm.png",
    operator: "/assets/avatars/avatar-operator.png",
  },
} as const;
```

> 入口文件由 `client/src/design-assets/scripts/build-canonical-assets.mjs` 据真实审计产出生成（含校验：未命中 inventory / 已被阶段 5 删除 / 类目或 status 不符 / 磁盘缺失即中止）。当前 `channels.*` 6 个 key 已由页面 `ChannelConfig.tsx` 接入，`brands` / `avatars` 8 个 key 仅建入口、暂未接入（不做全量迁移）；接入与一改多处生效详情见 `docs/resource-governance-report.md` 第十节。

这样可以实现两层“一改多处生效”：

1. 多处引用同一个物理资源文件时，修改该文件会影响所有引用处。
2. 多处引用 `canonicalAssets` 入口时，修改入口映射会影响所有接入处。

## 5.4 一改多处生效的边界

资源库建设完成后，不承诺所有历史引用自动同步。只有以下场景可以一改多处生效：

| 场景 | 是否自动同步 | 说明 |
|---|---:|---|
| 多处引用同一个物理文件 | 是 | 修改该文件会影响所有引用处 |
| 多处引用 `canonicalAssets` 入口 | 是 | 修改入口映射会影响所有接入处 |
| 页面层通过现有组件 props 传入同一 canonical 资源 | 是 | 不改组件源码，只要多个调用处传入同一 canonical 资源，修改该资源文件会统一生效 |
| 重复文件尚未治理 | 否 | 修改其中一个文件不会影响其他副本 |
| inline SVG | 否 | 写在组件里的 SVG 不受资源文件影响 |
| 直接使用 `lucide-react` | 否 | lucide 来自第三方包，不受本地资源文件影响 |
| 复制出来的 React SVG 组件 | 否 | 除非接入统一入口，否则不会自动同步 |

页面详情中需要展示：

- 是否为 canonical 资源；
- canonical ID；
- 是否已接入 `canonicalAssets`；
- 当前仍有哪些使用位置没有接入统一入口；
- 修改该资源预计会影响哪些引用位置。

## 5.5 skill-map 数据

skill 不读取页面，而是读取精简稳定的资源映射。

文件（阶段 9 已落地）：

```text
client/src/design-assets/resource-skill-map.json
```

由 `client/src/design-assets/scripts/build-resource-skill-map.mjs` 从 inventory 审计字段确定性派生（`npm run build:skill-map`），请勿手改；校验由 `npm run check:skill-map` 落地。

真实结构（阶段 9 产出）：

```json
{
  "stage": 9,
  "version": "<ISO 生成时间>",
  "registrySample": ".codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json",
  "sources": { "inventory": "...", "componentResourceMap": "...", "governance": "..." },
  "usageScopeLegend": { "current-project-only": "...", "host-injected": "...", "portable-safe": "..." },
  "summary": { "slots": 9, "candidates": 155, "redlineCandidates": 14, "bySlot": { } },
  "slots": {
    "<slot>": {
      "label": "...", "componentSlotPath": "...", "owningComponents": [], "recommendedResourceType": "...",
      "allowLucideFallback": false, "riskLevel": "...", "redline": false, "hostInjection": "...",
      "constraint": "...", "finding": null, "candidateCount": 0,
      "candidates": [
        { "id": "...", "displayName": "...", "slot": "...", "via": "component-slot|canonical-redline",
          "type": "svg|png", "source": "public|src", "primaryCategory": "...", "visualType": "...",
          "scenes": [], "landing": "file|inline", "sourcePath": "...", "webPath": "...", "importPath": "...",
          "canonicalKey": "...", "reference": { "kind": "...", "value": "..." },
          "redline": false, "canRecolor": false, "allowLucideFallback": false,
          "recommendedResourceType": "...", "usageScope": "current-project-only|host-injected" }
      ]
    }
  }
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `registrySample` | skill 自带的「可移植身份样例」`icon-registry.example.json`，仅供跨仓移交时参照建立宿主仓正式 registry；**不**作为候选准入闸门 |
| `slots` | 9 槽位白名单，取自阶段 8 `component-resource-map.json`；每槽位挂槽位约束 + `candidates` |
| `candidates` | 由 inventory 审计字段确定性派生（`status=normal` 且 未排除/未归档 且 有 `componentSlot`，或带红线 `canonicalKey`；不引用 `removedIds`），共 **155**（槽位 141 + 红线 14） |
| `usageScope` | `current-project-only`（当前项目页面专属）/ `host-injected`（红线，跨仓宿主注入）/ `portable-safe`（如 lucide，不在本映射内） |

> **口径修正（阶段 9，据真实审计产出）**：原计划设想「skill-map 候选必须能在 `icon-registry.example.json` 中找到且 `status` 为 `approved`」。但阶段 8 审计表明：本项目带确认槽位的资源有 141 项、红线 14 项，与那份 28 条 registry 样例几乎无交叉关联（inventory 848 项中仅 1 项 `registry.registered`）。结论：**那份 registry 是 skill 的可移植身份样例，inventory 才是本项目资源真相**。故 registry 不再作为候选准入闸门，候选准入以 inventory 审计字段为准。槽位字段集（视觉类型、是否允许 lucide 回退等）以阶段 8 组件资源审计的真实产出为准。

---

# 六、与 skill 的连接方式

## 6.1 连接原则

`clawpro-portable-design-skill` 需要能合理使用资源库资源，但必须区分“当前项目页面”和“共享组件 / 跨仓代码”。

连接方式是：

```text
资源扫描 / 人工分类
  -> resource-inventory.generated.json
  -> canonical-assets.ts（仅当前项目页面层使用）
  -> resource-skill-map.json（记录自有资源候选、组件资源映射和使用边界）
  -> clawpro-portable-design-skill references/assets-icons.md
  -> 页面生成时按上下文选择 lucide / canonical / 宿主注入资源
```

## 6.2 skill 使用上下文

skill 生成代码前必须先判断上下文：

| 上下文 | 资源使用规则 |
|---|---|
| 当前项目页面 | 可以使用 `canonicalAssets` 和资源库中的 canonical 资源 |
| 当前项目页面级非组件代码 | 可以在调用层使用 `canonicalAssets`；不得改动任何组件源码 |
| 组件源码 | 不得 import `canonicalAssets`，不得写死当前项目 `/assets/...` 路径 |
| 开发仓库 / 跨仓页面 | 不得引用当前项目资源路径；优先使用 `lucide-react`、宿主仓已有资源或 props 注入 |
| 已使用 lucide 的组件 | 继续使用 `lucide-react`，不因资源库建设而改造 |
| 使用自有 SVG / 图片的组件 | 只建立资源使用映射和风险约束，不改组件源码 |

## 6.3 skill 使用规则

生成页面或组件时，skill 应按以下流程选择资源：

1. 判断上下文：当前项目页面、当前项目页面级非组件代码、组件源码、开发仓库 / 跨仓页面。
2. 判断端别：Admin / Tenant / Landing / Global。
3. 判断资源需求：lucide 通用图标、自有 SVG、Agent 头像、渠道图标、品牌 Logo、业务图片。
4. 选图规则（阶段 9 已落地，详见 skill `references/assets-icons.md` §1 优先级 + §5.5 槽位选图）：通用 UI（导航 / 操作 / 状态 / 表单辅助）默认 `lucide-react`；业务 / 自有 SVG 命中组件槽位时从 `resource-skill-map.json` 对应槽位 `candidates` 按视觉风格 / 尺寸 / 场景挑选；`allowLucideFallback=true` 的槽位无合适候选可回退 lucide，`=false`（多彩渐变 / 动画 / 特性卡 / **统计卡 number-card**）无候选时标 `needs-design-confirmation`，不私自画图、不回退扁平 lucide（破坏渐变家族）。
5. 如果是当前项目页面：查询 `resource-skill-map.json` 对应槽位候选——红线资源（头像 / 渠道 / 品牌）经 `canonicalAssets.<group>.<key>` 引用，非红线候选经其 `webPath` 或 ESM `importPath` 引用。
6. 如果是 lucide 已覆盖的通用组件或通用操作：继续使用 `lucide-react`，不强行替换为资源库自有 SVG。
7. 如果是共享组件源码：不得引用 `canonicalAssets` 或当前项目资源路径，只能使用组件既有 API、props / slot、`lucide-react` 或宿主资源注入。
8. 如果是开发仓库 / 跨仓页面：不得引用当前项目 `/assets/...` 或 `canonicalAssets`，需要使用宿主仓资源、lucide fallback 或标记 `needs-design-confirmation`。
9. 如果没有合适资源：标记 `needs-design-confirmation`，不私自画 inline SVG。
10. 生成代码后自检是否出现 emoji、未登记 inline SVG、错误品牌资源使用、跨仓引用当前项目路径等问题。

## 6.4 skill 禁止规则

skill 生成页面或组件时禁止：

- 使用 emoji 图标；
- 新增未登记 inline SVG；
- 在共享组件源码中 import `canonicalAssets`；
- 在共享组件源码中写死当前项目 `/assets/...` 路径；
- 在开发仓库 / 跨仓页面中引用当前项目资源路径；
- 普通业务页面随意引用 Landing 图片；
- Agent 头像脱离资源库或宿主注入路径随便找图；
- 渠道图标脱离资源库或宿主注入路径随便找图；
- 品牌 Logo 当普通图标随意改色；
- 将 `brand-fixed`、`asset-fixed-color` 资源放进普通按钮图标槽位；
- 使用 `avoid`、`deprecated`、`needs-review` 资源作为默认候选。

## 6.5 如何保证 skill 用对

需要三层保证。

### 第一层：规则保证

更新 skill 文档：

```text
.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md
```

写明：

- 生成当前项目页面时，资源选择可以查 `resource-skill-map.json` 和 `canonicalAssets`。
- 生成共享组件源码或跨仓代码时，禁止引用当前项目 `canonicalAssets`。
- 已使用 lucide 的组件继续使用 `lucide-react`。
- 只有涉及自有 SVG / 图片资源的组件才参考资源库映射。
- Agent / 渠道 / 品牌资源必须匹配专属槽位或由宿主仓注入。
- 没有合适资源时不私自画图。

### 第二层：数据保证

`resource-skill-map.json` 必须通过校验（由 `npm run check:skill-map` 落地）：

- candidate 的 `id` 必须存在于资源清单 inventory；
- candidate 对应 inventory 资源 `status` 必须为 `normal`（`deprecated` / `avoid` / `needs-review` / 已归档不得入），且不在 `governance.removedIds`；
- candidate 的 `sourcePath` 必须存在（`landing=file` 磁盘校验；`landing=inline` 不校验静态文件）；
- candidate 的 slot 必须在 `component-resource-map.slots` 白名单内；
- candidate 必须标记 `usageScope`：`current-project-only` / `portable-safe` / `host-injected`；
- 当前项目路径资源只能用于 `current-project-only`；红线资源为 `host-injected`；
- `brand-fixed` / `avatar-like` 红线资源不能进入普通 icon slot，只能落对应红线槽位；
- `agent-avatar` 只能进入 Agent 头像 slot；
- `channel-icon` 只能进入渠道 slot；
- candidate 的 `allowLucideFallback` / `recommendedResourceType` 必须与阶段 8 槽位规格一致（防漂移）。

> **口径修正（阶段 9）**：原条目「candidate 必须能在 `icon-registry.example.json` 中找到且 `status` 为 `approved`」已移除——registry 降格为 skill 可移植身份样例、不作候选准入闸门，候选准入以 inventory 审计字段为准（见 §5.5 口径修正）。

### 第三层：检查脚本保证

阶段 9 已落地的检查脚本：

```text
client/src/design-assets/scripts/build-resource-skill-map.mjs   # npm run build:skill-map（确定性重建）
client/src/design-assets/scripts/check-resource-skill-map.mjs   # npm run check:skill-map（数据保证校验）
client/src/design-assets/scripts/check-component-resource-usage.mjs  # npm run check:component-resources（组件源码层防新增违规）
```

检查内容：

- `resource-skill-map.json` 中引用的资源是否存在（id 在 inventory、file 候选磁盘存在）；
- skill-map 是否引用了已废弃 / 已治理移除资源（status≠normal 或在 removedIds 即报错）；
- 当前项目页面是否新增未登记资源路径；
- 组件源码是否错误引用 `canonicalAssets` 或当前项目 `/assets/...`；
- 业务代码是否新增 inline SVG；
- 业务代码是否新增 emoji 图标；
- 是否把品牌 / 渠道资源用于错误槽位；
- 是否直接复用了未治理重复资源。

---

# 七、组件资源使用映射与风险约束

组件相关工作本阶段只做“资源使用映射与风险约束”，不做组件源码改造。由于所有组件都已被开发仓库使用，所有组件均按共享组件处理，不应因为当前项目资源库建设而修改 API、默认资源或内部资源引用。

## 7.1 处理原则

1. **只处理使用自有 SVG / 图片资源的组件**：继续使用 `lucide-react` 的组件保持现状，不纳入资源库改造。
2. **不修改任何组件源码**：所有组件均视为开发仓库已使用的共享组件，不改 API、不改默认图标、不新增当前项目资源依赖。
3. **资源库只约束“资源该如何被选择”**：不强迫组件实现依赖资源库。
4. **`canonicalAssets` 只用于当前项目页面层**：任何组件源码都不得 import。
5. **组件资源映射是生成建议，不是组件实现依赖**。
6. **如需改组件资源实现，必须单独立项并评估兼容性**。

## 7.2 需要梳理的组件范围

只梳理满足以下任一条件的组件：

1. 当前内部使用了自有 SVG、PNG、品牌图、渠道图、Agent 头像等项目资源。
2. 当前通过 props 接收业务图片资源，但资源语义容易混用。
3. 与 Agent、渠道、品牌、指标图标等强业务资源相关。
4. 存在当前项目资源路径依赖风险。

不纳入本阶段组件映射的范围：

1. 仅使用 `lucide-react` 的组件。
2. 不接收、不渲染业务图片资源的纯 UI 组件。
3. 与资源无关的布局、表单、表格、弹窗等基础组件。

## 7.3 映射内容

每个纳入范围的组件只记录映射，不改源码：

| 字段 | 说明 |
|---|---|
| 组件名 | 例如 `AgentCard`、`ChannelConfig`、`BrandLogo` |
| 是否共享组件 | 固定为是，所有组件均按共享组件处理 |
| 当前资源来源 | 当前是否引用自有 SVG / PNG / `/assets/...` |
| 资源槽位 | `AgentCard.avatar`、`ChannelConfig.channelIcon` 等 |
| 推荐资源类型 | `agent-avatar`、`channel-icon`、`brand-logo`、`custom-svg` |
| 是否允许 lucide fallback | 通用图标可允许，品牌 / 渠道 / 头像通常不允许 |
| 当前风险 | 跨仓缺资源、路径写死、品牌误用、重复资源等 |
| 推荐用法 | 当前项目页面用 `canonicalAssets`；跨仓由宿主注入 |
| 是否需要单独改造 | 如需改组件 API 或默认资源，必须另起计划 |

下表为**阶段 8 组件资源审计产出的纳入映射组件清单**（事实源 `client/src/design-assets/manual-overrides/component-resource-map.json`，已从举例收敛为真实清单）。所有组件统一按「开发仓库已使用 / 共享组件」处理（是否共享=是），本阶段只记录映射与风险、不改任何源码：

| 组件名 | 当前资源来源 | 资源槽位 | 推荐资源类型 | lucide fallback | 当前风险 | 是否需单独改造 |
|---|---|---|---|---|---|---|
| `AgentAvatar` | 硬编码 7 个 `/assets/avatars/avatar-*.png` | `agent-avatar` | agent-avatar | 不可 | 高·组件层写死当前项目路径、跨仓缺资源 | 是 |
| `StatusBadge` | 8 态全部组件内 inline 动画 SVG（未引用 `agent-card/status-*.svg`） | `run-status` | custom-svg | 不可 | 中·与资源库静态文件「标槽位但未引用」不一致 | 否 |
| `NumberCard` | 不硬编码 `/assets`；内置 4 inline 渐变图标（早期固化常用项·非上限）+ `icon` 槽位由页面传入 | `number-card` | custom-svg | 不可 | 低·组件无路径依赖 | 否 |
| `AdminSidebar` | 内嵌品牌 Logo inline SVG；菜单图标由页面经 img/svg 传入；背景图 `url(/admin_content_bg.png)` | `admin-sidebar` / `brand-logo` | custom-svg / brand-logo | 菜单图标可、Logo 不可 | 中·内嵌 Logo inline、背景图路径写死 | 否 |
| `ComparisonTable` | 硬编码 6 个 `/assets/admin-memory-management/version-compare/*.svg`；另用 lucide `Check` | `feature-card` | custom-svg | 不可 | 高·组件层写死当前项目路径 | 是 |
| `ChannelConfig` | 页面层经 `canonicalAssets.channels.*` 引用（阶段 6 已接入的正确范例） | `channel-icon` | channel-icon | 不可 | 低·已走 canonical 入口 | 否 |
| `FileSpace` | 文件类型/工具栏图标全部 inline SVG；仅关闭态 import 4 个功能介绍图标 | `file-type` | custom-svg / lucide | 可 | 中·与资源库 21 个静态文件「标槽位但仅 4 个被 import」不一致 | 否 |
| `Empty` | 硬编码 `/assets/admin-sidebar/empty-aiagent*.png`（资源已标 `excludeFromLibrary`） | `empty-media` | business-image | 不可 | 中·组件层写死当前项目路径 | 否 |

> 推荐用法统一为：当前项目页面层用 `canonicalAssets`，组件 / 跨仓由宿主通过 props / 槽位注入；组件源码不得 import `canonicalAssets` 或写死当前项目 `/assets/...`。如确需改组件 API 或默认资源，必须脱离本资源库任务单独立项。

## 7.4 不同组件的资源策略

| 组件类型 | 本阶段策略 |
|---|---|
| 已使用 `lucide-react` 的通用组件 | 保持现状，不纳入资源库改造 |
| 使用自有 SVG 的组件 | 只记录资源映射和风险，不改源码；当前项目页面层可在调用处使用 `canonicalAssets` |
| 所有组件 | 均按开发仓库已使用的共享组件处理，只记录资源映射和风险，不改源码 |
| Agent 相关组件 | 记录头像资源槽位，当前项目页面可用 canonical，跨仓由宿主注入 |
| 渠道相关组件 | 记录渠道图标槽位，保持固定色，跨仓由宿主注入 |
| 品牌 Logo 相关组件 | 记录品牌资源槽位，不当普通 icon 使用，跨仓由宿主注入 |
| 指标 / 能力卡组件 | 若使用自有 SVG，记录适合的 `metric` / `gradient` 资源；若用 lucide，保持 lucide |

## 7.5 本阶段交付物

1. 梳理使用自有 SVG / 图片资源的组件清单。
2. 将所有组件统一标记为“开发仓库已使用 / 共享组件”。
3. 为这些组件建立“推荐资源映射”。
4. 在资源库页面展示“适合组件槽位”。
5. 在 skill 里说明生成这些组件相关代码时如何选资源。
6. 明确 `canonicalAssets` 只在当前项目页面层使用。
7. 如需改组件，必须单独立项并评估兼容性。

## 7.6 本阶段明确不做

1. 不修改任何组件源码。
2. 不重构组件 icon API。
3. 不强制旧组件立即改为统一资源入口。
4. 不把使用 lucide 的组件替换成自有 SVG。
5. 不让共享组件 import `canonicalAssets`。
6. 不让共享组件新增当前项目 `/assets/...` 依赖。
7. 不把品牌、渠道、头像资源抽象成通用 Icon 组件。

---

# 八、安全落地原则

## 8.1 页面安全

1. 不在页面中执行扫描命令。
2. 不通过页面上传文件。
3. 不通过页面删除资源。
4. 不提供在线 SVG path 编辑器。
5. 不使用用户输入路径读取文件。
6. 不使用 `dangerouslySetInnerHTML` 渲染未清洗 SVG。
7. 外链搜索不自动请求用户输入 URL。

## 8.2 SVG 安全

单页不做在线 SVG 安全扫描，但资源入库脚本和 CI 需要做安全检查。

需要禁止：

- `<script>`；
- `<foreignObject>`；
- `on*` 事件属性；
- 外部 `href` / `xlink:href`；
- 远程图片引用；
- 可执行或不可信嵌入内容。

## 8.3 治理安全

1. A 类重复资源才允许直接删除。
2. B 类重复资源必须小范围替换引用，并保证 diff 清晰。
3. C 类资源只标记待确认，不自动删除或替换。
4. 品牌、渠道、Logo 类资源不根据 hash 自动归并。
5. 删除或替换资源必须在治理报告中记录。

## 8.4 合入 main 前检查

合入前至少确认：

- `/design-system/assets` 可以打开；
- 页面只消费静态 JSON 数据；
- 资源扫描脚本可重复执行；
- 重复治理报告存在；
- A / B 类治理 diff 清晰；
- `resource-skill-map.json` 校验通过；
- 无新增危险 SVG；
- 无新增未登记资源路径；
- 所有组件源码均未新增 `canonicalAssets` 或当前项目 `/assets/...` 依赖；
- 不影响现有业务页面和开发仓库已使用的共享组件。

---

# 九、从当前阶段到资源库落成的完整实施计划

## 进度看板（跨对话推进用）

本计划阶段较多，可能分多个对话执行。本看板是阶段之间的**唯一进度事实源**：

- **每个新对话开始**：先读本看板，确认当前要做的阶段、以及它依赖的上一阶段产物是否齐备，再动手。
- **每个阶段结束**：把对应行「状态」改为 `已完成`，必要时在「备注」补当前产物的实际文件路径。
- **执行顺序**：阶段 0 → 11 原则上串行，后一阶段依赖前一阶段产物；其中 6 → 8 → 9 为文档回填的关键链路，不可乱序。
- 状态取值：`未开始` / `进行中` / `已完成`。

**新对话开场白模板**（每开一个对话，把 `X` 换成本次要做的阶段号，直接发给执行者）：

> 请先读 `docs/ClawPro图标与资源库建设计划.md` 与 `docs/ClawPro图标与资源库背景目标.md`，我们本次做**阶段 X**。
> 1. 先看建设计划第九章「进度看板」，确认阶段 X 的状态、以及它依赖的上一阶段产物文件是否已齐备；缺产物先停下告诉我。
> 2. 严守边界：不做 npm 包、不做全量迁移、不做强注册体系、不做在线上传/删除/SVG 编辑、不做全量组件重构。
> 3. 严守 skill 纪律：非「阶段 9」不动 skill；即便阶段 9 改 skill，也必须以审计真实产出为准、改前先比对原文、不丢原有规则。
> 4. 完成后：把看板里阶段 X 的「状态」改为 `已完成`，补上实际产物路径；若该阶段带回填动作（6/8/9），一并回填对应文档占位。

| 阶段 | 名称 | 状态 | 关键产物 | 备注 |
|---|---|---|---|---|
| 0 | 读背景目标 & 锁定范围 | 已完成 | 已确认的实施边界 | 仅确认边界，不产数据；已确认边界见 §9 阶段 0 末「实施边界确认结果」 |
| 1 | 资源审查准备 | 已完成 | 扫描规则 / 分类枚举 / 重复规则 / SVG 安全规则 | 口径见 §9 阶段 1 末「资源审查准备口径」；目录已据实核验 |
| 2 | 全量扫描 & 使用统计 | 已完成 | `resource-inventory.generated.json` / `resource-usage.generated.json` | 数据基线；扫描脚本 `client/src/design-assets/scripts/scan-resources.mjs`；产出见 §9 阶段 2 末「扫描数据基线」 |
| 3 | 资源分类 & 待确认清单 | 已完成 | inventory 分类字段 / 待确认清单 | 分类脚本 `client/src/design-assets/scripts/classify-resources.mjs`；产出 `resource-needs-review.generated.json`；详见 §9 阶段 3 末「资源分类基线」 |
| 4 | 重复审查 & 治理分级 | 已完成 | `resource-duplicates.generated.json` / A·B·C 分级 / canonical 建议 | 识别脚本 `client/src/design-assets/scripts/detect-duplicates.mjs`；回填 inventory `classification.duplicate*`；详见 §9 阶段 4 末「重复审查与分级基线」 |
| 5 | 重复资源实际治理 | 已完成 | `resource-governance-report.md` / `resource-governance.generated.json` / 治理 diff | 治理脚本 `client/src/design-assets/scripts/apply-governance.mjs`（独立现场复核 + 可重复执行，默认 dry-run，`--apply` 才删）；A 73 组共删 **127** 个冗余副本（保留 canonical，回收 ~181KB）、B 0 组、C 11 组只标待确认；registry 事实源 / 品牌·渠道·头像红线 / 被引用资源全部保留；删除经 git 可回滚；详见 §9 阶段 5 末「重复治理实际处理基线」 |
| 6 | canonical 资源入口 | 已完成 | `canonical-assets.ts` | 入口 `client/src/design-assets/canonical-assets.ts`（脚本 `scripts/build-canonical-assets.mjs` 生成，可重复+校验）；纳入 **14** key（brands 1 / channels 6 / avatars 7，仅已确认 normal 业务专属资源，needs-review 的 /icon SVG 不纳入）；红线 wecom/wecom-app 分列不归并；channels 6 key 已由 `ChannelConfig.tsx` 页面层接入，其余 8 key 仅建入口（不做全量迁移）；已治理重复副本无引用故无迁移；回填 inventory `canonicalSummary`+逐项 canonical 字段、governance `canonical` 段、报告第十节、**已回填 §5.3**；详见 §9 阶段 6 末「canonical 入口建设基线」 |
| 7 | 资源库单页落地 | 已完成 | `/design-system/assets` 页面 / 资源详情面板 / 重复治理视图 / 未分类·待确认视图 | 页面 `client/src/pages/DesignSystemAssets.tsx`，路由 `/design-system/assets`（`client/src/App.tsx` 注册）；**只消费**阶段 2~6 静态 JSON（inventory/usage/governance/needs-review）+ 构建期 `import.meta.glob` 预览，不扫描/上传/删除/编辑文件、不写回 JSON；含顶部 4 统计、左侧分类导航、搜索+场景/状态/入口组合筛选、网格·列表切换、资源详情面板（多尺寸+深浅底预览、基础/分类/canonical 信息、使用位置、复制引用、重复治理信息）、重复治理视图（按 state 组织展示 84 组 / 已删 127）、canonical 入口视图（14 key 接入状态）、未分类·待确认视图（746 项分类分页）；「适合组件槽位」按纪律留阶段 8 占位、不臆测；`tsc --noEmit` 通过、无新增 lint |
| 8 | 组件资源映射 & 风险约束 | 已完成 | 组件清单 / 槽位信息 / 风险约束 | 产物：手工策展事实源 `client/src/design-assets/manual-overrides/component-resource-map.json`（9 槽位白名单 + 风险约束 + 8 组件清单 + 据实发现）；资源库详情面板「适合组件槽位」已接入真实白名单与风险约束（`DesignSystemAssets.tsx`，replace 占位）；项目侧检查脚本 `client/src/design-assets/scripts/check-component-resource-usage.mjs`（基线模式，wire 至 `npm run check:component-resources`）；**已回填 §3.7 槽位白名单、§7.3 组件清单**；据实记录 run-status / file-type 静态文件「标槽位但组件实为 inline」的不一致；全程不改组件源码、不动 skill（阶段 9）；详见 §9 阶段 8 末「组件资源映射与风险约束基线」 |
| 9 | skill 连接 & 规则落地 | 已完成 | `resource-skill-map.json` / `assets-icons.md` | 生成器 `client/src/design-assets/scripts/build-resource-skill-map.mjs`（`npm run build:skill-map`）从 inventory 审计字段**确定性派生** `client/src/design-assets/resource-skill-map.json`：9 槽位 **155** 候选（槽位 141：card-left-icon 64 / number-card 22 / admin-sidebar 22 / file-type 21 / run-status 8 / feature-card 4；红线 14：avatars 7 / channels 6 / brands 1，经 canonicalAssets·host-injected）；候选准入只读 inventory（status=normal + 有 componentSlot 或红线 canonicalKey + 未排除/未被治理移除），**不回写** generated inventory（保持纯扫描基线）；校验脚本 `check-resource-skill-map.mjs`（`npm run check:skill-map`，fail-fast：id/status/removedIds/slot 白名单/磁盘/usageScope/红线落槽/防漂移）已通过；更新 skill `references/assets-icons.md`（保留 lucide 优先原立场，补 §5.5 槽位选图，**registry 降格可移植样例、不作 approved 闸门**）；**已回填 §5.5 真实结构 + 口径修正、§6.3 选图规则、§6.5 第二/三层、背景目标 §六**；详见 §9 阶段 9 末「skill 连接与规则落地基线」 |
| 10 | 安全校验 & 回归检查 | 未开始 | 检查脚本结果 / 治理报告最终版 / 构建·类型·lint 结果 | |
| 11 | 落成验收 & 交付 | 未开始 | 全部交付物汇总（见本章末交付物清单） | |

## 阶段 0：读取背景目标文档与锁定范围

目标：确保执行者理解项目定位和边界，避免实施中重新扩大范围。

工作内容：

1. 阅读 `docs/ClawPro图标与资源库背景目标.md`。
2. 确认资源范围、分类体系、资源状态和本阶段不做项。
3. 确认本次目标是“审查治理 + 资源库单页 + canonical 入口 + 组件 / skill 对齐”。
4. 明确不做 npm 包、全量资源迁移、强注册体系、在线上传 / 删除 / SVG 编辑。

产物：

```text
已确认的实施边界
```

### 实施边界确认结果（阶段 0 产物）

> 本节为阶段 0 的唯一产物，对应看板「已确认的实施边界」。仅锁定范围，不产任何扫描 / 数据 / 代码。

**1. 本次要做的事（范围内）**

- 审查治理：盘点纳入范围的图标 / 图片资源，识别重复，做 A / B / C 分级与实际治理。
- 资源库单页：新增 `/design-system/assets`，只消费静态 JSON，不在线扫描 / 上传 / 删除 / 编辑。
- canonical 入口：对高频、已确认 canonical 资源建立 `canonical-assets.ts`，仅供当前项目页面层使用。
- 组件 / skill 对齐：建立组件资源使用映射与风险约束、`resource-skill-map.json`，并对齐 `clawpro-portable-design-skill`。

**2. 明确不做（硬边界，实施中不得扩大）**

- 不做私有 npm 包。
- 不做全量资源迁移。
- 不做强注册体系（资源库是清单 + 治理视图，不强制先注册再使用）。
- 不做在线上传 / 删除 / SVG 编辑器。
- 不做全量组件重构（不改任何组件源码、不改组件 icon API）。
- 不做复杂审批流、不做「推荐复用」独立视图、不做 Landing 大图 / 截图管理。
- 不承诺所有历史引用自动同步（仅同物理文件 / `canonicalAssets` / 专属组件接入处具备一改多处生效能力）。

**3. 关键事实源与不漂移约束（阶段 9 据真实审计修正）**

- 本项目资产真相与候选准入事实源：`client/src/design-assets/generated/resource-inventory.generated.json`（阶段 1 扫描、逐阶段审计的 848 项台账）。
- `resource-skill-map.json` 候选由 inventory 审计字段确定性派生（`status=normal` + 有 `componentSlot` 或红线 `canonicalKey` + 未排除 / 未被治理移除），不引用已废弃 / 已治理移除资源；`generated/*.json` 为扫描产物，skill-map 不反向覆盖之。
- `icon-registry.example.json` 降格为 skill 自带的**可移植身份样例**（2026-06-06 随 skill 诞生、早于本资源库计划、仅初始登记 `icon/` 根目录 28 项），**不作候选准入闸门**；跨仓移交时供宿主仓建立正式 registry 参照。
- 资产新增 / 废弃 / 迁移：当前项目先更新分类 / 治理输入并重跑 `npm run build:skill-map`，跨仓再同步宿主仓 registry。

> **口径修正（阶段 9）**：原写「registry 是资产身份唯一事实源、skill-map 只引用其 `approved` 资产、先改 registry」。阶段 8 审计表明本项目 848 项中仅 1 项 `registry.registered`，带确认槽位 141 + 红线 14 与那 28 条样例几乎无交叉，故据实改为「以 inventory 为本项目真相、registry 为可移植样例」。红线（品牌 / 渠道 / 头像）硬约束不变。

**4. skill 改动纪律（贯穿全程）**

- 非「阶段 9」不动 skill。
- 即便阶段 9 改 skill：以阶段 8 组件审计的真实产出为准、改前先比对原文、不丢原有合理规则（原始立场为「默认 lucide + 业务/品牌用已登记 registry SVG」）。
- 品牌 Logo、渠道图标、Agent 头像为硬约束：保留品牌固定色，禁止当普通 UI 图标改色。

**5. 选图规则 / 槽位字段的待定项（不在前置阶段预设）**

- 选图判断顺序、槽位字段（`allowedTypes`、视觉类型、尺寸区间、是否允许 lucide 回退、candidates 等）以阶段 8 组件资源审计真实产出为准。
- 文档中标注「示例 / 待阶段 8 审计」的占位（§3.7、§5.3、§5.5、§6.3 及背景目标 §六）在对应阶段（6/8/9）据实回填，不提前臆测。

## 阶段 1：项目资源审查准备

目标：确定扫描目录、审查规则、分类枚举和治理口径。

工作内容：

1. 明确扫描目录：`client/public/assets/**`、`client/public/icon/**`、`client/src/assets/**`、`icon/**`、`assets/**` 等。
2. 明确代码扫描范围：页面、组件、布局、配置文件、CSS 中的资源引用。
3. 明确资源类型枚举：SVG、PNG、Lucide、Agent 头像、渠道图标、品牌 Logo、业务图片、inline SVG。
4. 明确使用场景和视觉类型枚举，保持与背景目标文档一致。
5. 明确重复识别规则：文件 hash、SVG 归一化 hash、文件名相似度、尺寸和路径语义。
6. 明确安全检查规则：禁止脚本、事件属性、外链、`foreignObject` 等危险 SVG 内容。

产物：

```text
资源扫描规则
资源分类枚举
重复识别规则
SVG 安全规则
```

### 资源审查准备口径（阶段 1 产物）

> 本节为阶段 1 的唯一产物，定义后续阶段 2 扫描脚本与数据生成的统一口径。仅定规则，不产数据、不写脚本、不动 skill。
> 下列目录均已对当前仓库据实核验；计划 §2.2「审查对象」为概述，本节为可执行口径，二者冲突时以本节为准。

**1. 资源扫描规则**

1.1 文件型资源扫描目录（include，纳入资源库展示）：

| 目录 | 实测内容 | 说明 |
|---|---|---|
| `client/public/assets/**` | 87 SVG / 13 PNG | 管控端业务资源；含 `avatars/`、`admin-channel-icons/`、`admin-sidebar/`、`admin-*/` 等子目录 |
| `client/public/icon/**` | 22 SVG / 4 PNG | 旧业务图标，命名/尺寸不完全统一 |
| `client/src/assets/**` | 48 SVG | Vite import 型资源；含 `agent-card/`、`icons/`、`topnav/` |
| `icon/**`（根目录） | 27 SVG | registry 已登记的业务 SVG 事实源（path 形如 `icon/xxx.svg`） |
| `assets/**`（根目录） | 471 SVG / 6 PNG | Figma / CodeBuddy 导出资源；**仅扫描、不默认纳入**，主要用于识别重复副本 / 污染资源 |

1.2 排除目录（exclude，不进入资源库展示，仅可进入排除报告）：

| 目录 / 文件 | 原因 |
|---|---|
| `client/public/landing-assets/**`（354 SVG / 66 PNG / 13 MP4） | Landing 大图、视频，页面专用、复用低 |
| `client/public/figma-replica/**`、`client/public/__manus__/`、`client/public/design-audit/`、`client/public/page-references/`、`client/public/research/`、`client/public/images/` | 设计审查 / 参考 / 临时素材，非运行时资源库资源 |
| `client/public/fonts/` | 字体，非图标 / 图片资源 |
| 根目录散落 PNG/WEBP（`model_config_*.png/.webp`、`tokens_monitor_*.png/.webp`、`dashboard_background.*`、`*_bg.*` 等） | 页面截图 / 背景大图，复用低 |
| `**/node_modules/**`、`*.zip`、`.git/**`、`.codebuddy/**` | 非项目资源 |

1.3 代码引用扫描范围（用于使用位置 / 使用次数统计）：

| 范围 | 匹配内容 |
|---|---|
| `client/src/**/*.{tsx,ts}` | `lucide-react` import、`@/assets/...` Vite import、`<img src="/assets/...">`、字符串路径引用 |
| `client/src/**/*.{tsx,ts}` | inline `<svg>...</svg>`（作为治理对象单独标记 source=`inline-svg`） |
| `client/src/**/*.css`、内联 style | `url(...)` 资源引用 |
| 配置 / 数据文件（如菜单、渠道、Agent 配置 `*.ts`/`*.json`） | 以路径字符串方式引用的资源 |

1.4 扫描原则：只读、可重复执行、产物落到 `client/src/design-assets/generated/`（该目录当前不存在，阶段 2 创建）；不在运行时扫描、不在页面内扫描。

**2. 资源分类枚举**（与背景目标 §5 严格一致，不新增、不漂移）

- `type`（资源类型）：`svg` / `png` / `lucide` / `agent-avatar` / `channel-icon` / `brand-logo` / `business-image` / `inline-svg`
- `source`（来源）：`public` / `src` / `lucide-react` / `inline-svg` / `root-assets`
- 一级分类：全部资源 / Lucide 图标 / 自有 SVG 图标 / Agent 头像 / 渠道图标 / 品牌 Logo / 业务图片 / 重复治理 / 未分类·待确认
- `usageScenarios`（使用场景，自有 SVG 主分类唯一、辅助可多个）：`navigation` / `action` / `search-filter` / `status` / `metric` / `agent` / `model-ai` / `channel` / `security` / `file-resource` / `billing-quota` / `brand-product` / `empty-hint`
- `visualType`（视觉类型）：`line` / `solid` / `duotone` / `gradient` / `brand-fixed` / `avatar-like` / `badge-emblem` / `illustrative-icon` / `monochrome-currentColor` / `asset-fixed-color`
- `scenes`（端别）：`admin` / `tenant` / `landing` / `global`
- `status`（资源状态）：`normal` / `preferred` / `duplicate` / `resolved` / `needs-review` / `avoid` / `deprecated`
- 硬约束：`channel-icon` / `brand-logo` / `agent-avatar` 与 `brand-fixed` / `asset-fixed-color` / `avatar-like` 资源禁止进入普通 UI 图标槽位、禁止改色（不因扫描自动归并）。

> 选图判断顺序、槽位字段（`allowedTypes` / 尺寸区间 / 是否允许 lucide 回退 / candidates 等）不在本阶段定义，待阶段 8 审计产出。

**3. 重复识别规则**

| 方式 | 口径 | 输出 |
|---|---|---|
| 文件 hash（精确） | 文件字节 SHA-256 完全一致 | 标记「确认重复」候选，A 类 |
| SVG 归一化 hash | 去除注释 / 空白 / `width`·`height` / `id` 等格式差异后再 hash，识别内容相同格式不同的 SVG | 标记「确认重复」候选，B 类 |
| 文件名相似度 | 文件名归一化后相似（去前后缀、编号、大小写） | 标记「疑似重复」，C 类 |
| 尺寸相同 + 路径语义相似 | 同尺寸且目录/命名语义近似的 PNG/SVG | 标记「疑似重复」，C 类 |
| 视觉并排预览 | 仅供人工确认，不自动判定 | C 类辅助 |
| 使用位置统计 | 使用次数高 / 路径稳定者优先作为 canonical 建议 | 辅助选 canonical |

重复识别安全红线：品牌 Logo、渠道图标、Agent 头像**禁止仅凭 hash 自动归并**，一律进 C 类人工确认。

**4. SVG 安全规则**（入库前必检，命中即阻断入库，记入治理报告）

禁止出现：

- `<script>` 标签；
- `<foreignObject>` 标签；
- `on*` 事件属性（`onload` / `onclick` 等）；
- 外部 `href` / `xlink:href`（远程链接）；
- 远程图片引用（`<image href="http...">`）；
- 其他可执行 / 不可信嵌入内容。

页面侧约束：页面只消费静态 JSON，不执行扫描命令、不上传 / 删除 / 编辑文件、不用 `dangerouslySetInnerHTML` 渲染未清洗 SVG。

## 阶段 2：全量资源扫描与使用位置统计

目标：生成资源库的基础数据，回答“项目里有哪些资源、在哪里用”。

工作内容：

1. 扫描文件型 SVG / PNG 资源。
2. 扫描 `lucide-react` import。
3. 扫描 TSX 内 inline SVG。
4. 扫描 `<img src>`、CSS `url(...)`、Vite import 等资源引用。
5. 统计每个资源的使用位置和使用次数。
6. 标记未使用资源、单处使用资源、多处使用资源。
7. 输出初始资源清单。

产物：

```text
client/src/design-assets/generated/resource-inventory.generated.json
client/src/design-assets/generated/resource-usage.generated.json
```

### 扫描数据基线（阶段 2 产物）

> 本节为阶段 2 的产物说明。阶段 2 **只产事实**（项目里有哪些资源、在哪里被引用、被引用多少次），**不做任何分类 / 重复 / canonical 判定**——清单中分类字段一律 `classification.pending=true` 留空，交阶段 3+ 据实回填。脚本只读、可重复执行，不修改任何资源文件、不动 skill。

**1. 扫描脚本（可重复执行，只读）**

```text
client/src/design-assets/scripts/scan-resources.mjs
```

- 零依赖（仅 `node:fs/path/crypto`），在仓库根目录运行：`node client/src/design-assets/scripts/scan-resources.mjs`。
- 扫描口径完全遵循阶段 1「资源审查准备口径」：include 目录 = `client/public/assets`、`client/public/icon`、`client/src/assets`、根 `icon`；scan-only = 根 `assets`（Figma 导出，仅扫描不默认纳入）；代码引用扫描根 = `client/src`（`.ts/.tsx/.js/.jsx/.css`，跳过 `node_modules/.git/dist/build/.codebuddy/generated`）。
- 解析口径据实校准：`@/` → `client/src`、Vite root=`client` 故 `/assets`·`/icon` → `client/public`、`@assets` → `attached_assets`。

**2. 两份产物的结构**

| 文件 | 内容 |
|---|---|
| `resource-inventory.generated.json` | `summary` 统计 + `items[]` 资源清单；每条含 `id/displayName/type/source/scanScope/sourcePath/webPath/importPath/bytes/svgMeta/usageCount/usageRefs/dynamicDirReferenced` 与留空的 `classification` |
| `resource-usage.generated.json` | `usage`（resourceId → 引用位置 `{file,line,kind}`）+ `dirReferences`（目录级动态引用证据）+ `unresolvedRefs`（未解析引用事实） |

字段约定：`usageCount` = 去重引用文件数，`usageRefs` = 引用出现次数，`kind` ∈ `import-lucide` / `vite-import` / `string-ref` / `css-url` / `inline-define`。

**3. 扫描结果（本次基线，与阶段 1 计数对账一致）**

- 资源条目总计 **1002**：`svg 655` / `png 25` / `lucide 173`（已使用的 lucide 图标种类）/ `inline-svg 149`。
- 按来源：`public 126`（assets 87+13、icon 22+4）/ `src 50` / `root-assets 504`（根 icon 27 + 根 assets 477）/ `lucide-react 173` / `inline-svg 149`。
- 按 scope：`include 525`（203 文件 + 173 lucide + 149 inline）/ `scan-only 477`（根 `assets/` Figma 导出）。
- 引用：`usageRefs 1509` 条，命中清单的资源 395 个；`unresolvedRefs 206` 条；目录级动态引用目录 4 个。
- 文件资源 680 中：多处引用 16，目录级动态引用 47；无静态字面量引用 607（其中无任何引用线索 568）。

**4. 据实校正 / 需后续阶段注意的事实**

- **校正阶段 1 口径**：`client/src/assets/**` 实为 **48 SVG + 2 PNG = 50 个文件**（阶段 1 口径仅写「48 SVG」，漏计 2 PNG）；本阶段以扫描事实为准。
- **`usageCount=0` ≠ 可删（红线）**：项目存在「基路径常量 + 模板字符串拼接」引用（如 `const BASE = "/assets/admin-sidebar"` 后 `` `${BASE}/x.svg` ``）和 `Record<string,string>` 映射表。带变量的拼接无法被静态完整路径匹配命中，会落入 `unresolvedRefs`，对应目标文件 `usageCount` 显示为 0。为此脚本额外采集「资源所在目录被字面量引用」事实并对文件打 `dynamicDirReferenced` 标记。**阶段 4/5 判断重复与治理时，禁止仅凭 `usageCount=0` 删除**，必须结合 `dynamicDirReferenced`、`unresolvedRefs` 与人工复核。
- **607「无静态引用」的构成**：主要为根 `assets/` 的 477 个 scan-only Figma 导出（多为真正未被运行时引用，是后续识别重复副本/污染的重点），其余为按目录动态拼接或确属冗余的业务图标，留待阶段 3/4 据实组织。
- **`inline-svg 149`、`lucide 173`** 为治理 / 候选对象事实记录，分类与是否进 skill-map 不在本阶段判定。

## 阶段 3：资源分类与待确认清单

目标：让资源具备设计团队可理解的分类和筛选基础。

工作内容：

1. 按一级分类归类：Lucide、自有 SVG、Agent 头像、渠道图标、品牌 Logo、业务图片等。
2. 对自有 SVG 补充使用场景：navigation、action、status、metric、agent、model-ai 等。
3. 对自有 SVG 补充视觉类型：line、solid、gradient、brand-fixed、monochrome-currentColor 等。
4. 标记端别：Admin、Tenant、Landing、Global。
5. 对无法自动判断的资源标记 `needs-review`。
6. 形成未分类 / 待确认清单。

产物：

```text
resource-inventory.generated.json 中的分类字段
未分类 / 待确认资源清单
```

### 资源分类基线（阶段 3 产物）

> 本节为阶段 3 的产物说明。阶段 3 在阶段 2 事实基线上**回填分类字段**（一级分类 / 使用场景 / 视觉类型 / 端别 / 状态），并产出待确认清单。核心纪律：**只在强信号下确定分类（目录语义 / registry 登记 / SVG 内容信号 / usage 引用端别），弱信号一律标 `needs-review` 入待确认清单，绝不臆测**；**不做任何重复 / canonical 判定**（`duplicateGroupId` / `canonicalId` 留阶段 4 / 6）。脚本只读输入、可重复执行、不动 skill。

**1. 分类脚本（可重复执行，只读输入）**

```text
client/src/design-assets/scripts/classify-resources.mjs
```

- 输入：阶段 2 的 `resource-inventory.generated.json` + `resource-usage.generated.json` + 资产事实源 `icon-registry.example.json`。
- 流水线顺序固定为「先 `scan-resources.mjs` 再 `classify-resources.mjs`」；分类为纯规则、幂等，重跑可复现。
- 读取 SVG 文件内容仅用于检测视觉信号（gradient / currentColor / 固定色），不修改任何资源文件。

**2. 分类规则（透明、可审计；precedence 自上而下先命中先定）**

| # | 命中条件 | 一级分类 | 关键判定 |
|---|---|---|---|
| 1 | `type=lucide` | `lucide` | `monochrome-currentColor`、可改色、场景不细分 |
| 2 | `type=inline-svg` | `inline-svg` | 治理对象 → `needs-review`（抽取/替换/保留待定） |
| 3 | `scanScope=scan-only`（根 `assets/CodeBuddyAssets`） | `scan-only-export` | 未默认纳入 → `needs-review` |
| 4 | `avatars/` 或 `agent-card/avatar-sprite` | `agent-avatar` | `avatar-like`、场景 `agent`、禁改色 |
| 5 | `admin-channel-icons/channel-*` | `channel-icon` | `brand-fixed`、场景 `channel`、禁改色 |
| 6 | `clawpro-logo` / 文件名含 `logo` | `brand-logo` | `brand-fixed`、场景 `brand-product`、禁改色 |
| 7 | 渠道目录下 `empty-*` | `own-svg` | 空态归类待定 → `needs-review` |
| 8 | `empty-*.png` 空态插画 | `business-image` | registry approved 者 normal，否则 `needs-review`（大插画默认不纳入） |
| 9 | 其它 PNG（栅格业务图标） | `business-image` | `asset-fixed-color` → `needs-review`（是否纳入/是否被 SVG 取代） |
| 10 | SVG（registry 登记或 include 目录自有） | `own-svg` | 见下 |

第 10 类（自有 SVG）的细分：

- **视觉类型**仅凭内容确定性信号：含 `linearGradient/radialGradient` 或 `data-figma-skip-parse` → `gradient`（不改色）；含 `currentColor`（无渐变）→ `monochrome-currentColor`（可改色）；其余固定色无法区分 line/solid → **留空待确认**。
- **使用场景**：registry 登记者以 registry `category` 为准映射（`metric/security/status/navigation/action/empty-state/upload-placeholder` 干净映射，其余如 `memory/business/feature/integration/misc/infrastructure` **不强行映射、场景留空但状态仍 normal**）；非 registry 者按目录强信号（`admin-sidebar`·`topnav`→`navigation`、`admin-security`→`security`、`agent-card`→`agent`），其余业务模块目录语义未映射 → `needs-review`。
- **端别**：优先用 usage 引用文件路径反推（`/pages/admin|tenant|landing|design-system/`），无引用再退回目录信号（`admin-*` / registry 业务图标 → `admin`），仍无则留空。

> 硬约束落实：`channel-icon` / `brand-logo` / `agent-avatar` 与 `gradient` 资源 `canRecolor=false`，不会被当普通 UI 图标改色；`needs-review` / `inline-svg` / `scan-only` 资源 `canUseInSkillMap=false`（skill-map 候选排除）。`canUseInSkillMap` 的最终取值是阶段 9 决策，本阶段仅排除明确不可者，其余留 `null` 未决。

**3. 分类结果（本次基线，合计 1002，与阶段 2 一致）**

- 一级分类：`scan-only-export 477` / `own-svg 176` / `lucide 173` / `inline-svg 149` / `business-image 10` / `agent-avatar 9` / `channel-icon 6` / `brand-logo 2`。
- 状态：`normal 256` / `needs-review 746`。
- 视觉类型：`monochrome-currentColor 178` / `gradient 128` / `asset-fixed-color 10` / `avatar-like 9` / `brand-fixed 8` / `illustrative-icon 1`（其余固定色 SVG 视觉类型留空待确认）。
- 使用场景：已解析 242，待补但非 review 14（全部为 registry 已 approved、但其 `category` 无法干净映射到本项目场景枚举的图标：business 3 / memory 4 / feature 3 / integration 1 / misc 1 / infrastructure 2）。
- 端别来源：`usage 反推 154` / `lucide 默认 global 173` / `目录信号 92` / `无 583`（无 583 主要为 scan-only 导出）。

**4. 待确认清单产物**

```text
client/src/design-assets/generated/resource-needs-review.generated.json
```

- 共 **746** 条：`scan-only-export 477`（根 `assets/` Figma 导出，待确认是否纳入 / 是否重复副本）/ `inline-svg 149`（治理对象）/ `own-svg 111`（业务模块目录语义未映射，需设计确认主场景）/ `business-image 9`（栅格业务图标 / 空态插画，待确认是否纳入）。
- 每条含 `guessPrimaryCategory` 与 `reasons`，集中交设计团队确认；本阶段**不删除、不替换**，且全部 `canUseInSkillMap=false`，不进入 skill-map 候选。

**5. 据实记录 / 交后续阶段的事实**

- 发现 `icon/AI Agent资产.svg` 与 `client/public/icon/AI Agent资产 2.svg` 字节完全一致（疑似重复副本），但**重复判定属阶段 4**，本阶段仅记录、不归并。
- registry 已 approved 但场景枚举无干净映射的 14 个图标，保持 `normal` 且场景留空（`usageScenarioResolved=false`），不强行套场景；如需补全场景需设计确认，但不影响其身份与 skill-map 资格。
- inventory 顶层新增 `classificationSummary`，`stage` 升至 3，分类字段写入每条 `items[].classification`。

## 阶段 4：重复资源审查与治理分级

目标：识别重复资源，并确定每组资源的治理方式。

工作内容：

1. 按文件 hash 找完全重复资源。
2. 按 SVG 归一化 hash 找内容相同但格式不同的 SVG。
3. 按文件名、尺寸、路径语义找疑似重复资源。
4. 结合使用次数和路径稳定性给出 canonical 建议。
5. 将重复组分为 A / B / C 三类：可自动治理、可半自动治理、需人工确认。
6. 对品牌、渠道、Logo 资源禁止仅凭 hash 自动归并。

产物：

```text
client/src/design-assets/generated/resource-duplicates.generated.json
重复资源 A / B / C 分级清单
canonical 建议清单
```

### 重复审查与分级基线（阶段 4 产物）

> 本节为阶段 4 的产物说明。阶段 4 在阶段 3 分类基线上**识别重复、给 A/B/C 分级、给 canonical 建议**。核心纪律：**只识别 / 只分级 / 只建议，不删除、不替换、不改任何资源文件**（实际治理是阶段 5，canonical 入口是阶段 6）。脚本只读输入（读字节算 hash、读 SVG 文本做归一化），幂等可重复。

**1. 识别脚本（可重复执行，只读输入）**

```text
client/src/design-assets/scripts/detect-duplicates.mjs
```

- 输入：阶段 2/3 的 `resource-inventory.generated.json` + `resource-usage.generated.json`（阶段 2/3 产物未存 hash，本脚本对 680 个文件资源现场只读计算）。
- 流水线顺序固定：`scan-resources` → `classify-resources` → `detect-duplicates`。本脚本会回填 inventory 的 `classification.duplicateGroupId / duplicateRole / duplicateGrade`；**重跑 `classify-resources` 会清空这些字段，需再跑本脚本**（脚本内已先清空旧标记保证幂等）。

**2. 重复识别方式（三种，可审计）**

| 方式 | 口径 | 本仓库实测 |
|---|---|---|
| `exact-file-hash` | 文件字节 SHA-256 完全一致（涵盖 SVG / PNG） | 83 组 |
| `svg-normalized-hash` | SVG 去注释 / `width`·`height`·`id`·`class` 等格式差异后内容一致 | **0 组**（本仓库所有 SVG 内容重复均字节一致，无「内容同格式异」；逻辑保留以备后续） |
| `filename-similar` | 仅 include 范围、跨 ≥2 目录、非纯数字命名的归一化同名（疑似副本） | 1 组（`public/icon/已关机.png` vs `icon/已关机.svg` 同名跨格式） |

> SVG 用归一化 hash 组织（自然涵盖字节相同 + 格式差异），PNG 用文件 hash 组织；scan-only 的纯数字命名（Figma 导出 `1.svg`/`20.svg`）被排除出文件名相似度组织以避免噪声。

**3. 分级与 canonical 建议规则（透明）**

- **canonical 建议优先级**：`被引用`（usage>0 / 目录级动态引用 / unresolved 命中）＞ `可服务`（有 webPath/importPath）＞ `include 范围` ＞ `registry 已登记` ＞ 来源(src>public>root)；冗余后缀名（" 2" / copy / 副本）轻微降权，不会被选为 canonical。即 canonical 取**运行时可用且确在使用者**，而非不可服务的 root `icon/` 事实源。
- **逐成员角色 / 动作**：canonical → `canonical-suggested`(keep)；registry 已登记的非 canonical 成员 → `keep-registry-source`(keep，**永不删**，事实源)；被引用的非 canonical 成员 → `duplicate`(`replace-refs-then-remove`，B 信号)；无任何引用线索的冗余副本 → `duplicate`(`remove-redundant`，A 信号)。
- **分级**：
  - **A**：内容完全一致 + 冗余副本无任何引用线索 → 阶段 5 可自动清理候选（保留 canonical）。
  - **B**：内容完全一致 + 存在业务引用 → 阶段 5 小范围替换引用到 canonical 后再移除。
  - **C**：红线资源 / 文件名疑似 / 仅余 registry 事实源副本（keep-multiple）→ 人工确认，不自动处理。
- **硬约束（红线）**：成员命中 `primaryCategory ∈ {brand-logo, channel-icon, agent-avatar}` 或 `visualType ∈ {brand-fixed, avatar-like}` → 整组强制 C、不归并、不给删除/替换动作。
- **安全红线**：`usageCount=0` 仅在**同时**无目录级动态引用（`dynamicDirReferenced=false`）且不在 `unresolvedRefs` 的 resolved 命中时，才作为删除候选；registry 事实源即便不是 canonical 也永不进入删除候选。

**4. 分级结果（本次基线）**

- 共 **84** 组、组内成员 **225**（223 内容重复 + 2 文件名疑似）。
- 按分级：**A 73 / B 0 / C 11**。`B=0` 经核实属实——唯一含「>1 被引用成员」的组是渠道红线组 `dup-074`（`channel-wecom.svg` 与 `channel-wecom-app.svg` 字节一致、均被引用），被红线正确判为 C 而非 B。
- 按方法：`exact-file-hash 83` / `filename-similar 1`（`svg-normalized-hash 0`）。
- A 类可清理候选 **127** 个（绝大多数为根 `assets/` 的 Figma 导出重复副本与 `public/icon/X 2.svg` 拷贝）；registry 事实源保留成员 **11** 个；canonical 建议组 **81**。
- 红线组 **2**：`dup-074`（渠道 wecom/wecom-app）、`dup-075`（`admin-sidebar/clawpro-logo.svg` 与某 Figma 导出 `assets/CodeBuddyAssets/747_183/19.svg` 字节一致）——均 C，不自动归并。

**5. 据实记录 / 交后续阶段的事实**

- 阶段 3 记录的 `icon/AI Agent资产.svg` 与 `public/icon/AI Agent资产 2.svg` 字节一致一事，本阶段已纳入对应重复组并给出 canonical 建议与角色。
- **8 个 C「keep-multiple」组**为 `public/icon/X.svg`（运行时 canonical，可服务）+ `icon/X.svg`（registry 事实源保留）的**有意双存在**，阶段 5 不应据此删除事实源；是否保留双份交设计/工程确认。
- A 类 127 个删除候选**本阶段不删除**，仅作为阶段 5 的清理候选；阶段 5 删除前须再次核验引用（结合 `dynamicDirReferenced` / `unresolvedRefs` 与人工复核），并记入治理报告、保证 diff 可回滚。
- inventory 顶层新增 `duplicateSummary`，`stage` 升至 4；canonical 入口（`canonicalId` / `canonical-assets.ts`）仍留阶段 6，本阶段只产 canonical *建议*。

## 阶段 5：重复资源实际治理

目标：完成本次任务范围内的实际治理，而不是只在页面上告警。

工作内容：

1. 清理 A 类重复资源：完全重复、未引用、明显临时副本。
2. 对 B 类重复资源做小范围引用替换：替换到 canonical 文件或统一入口。
3. 对 C 类资源标记待确认：不删除、不替换。
4. 记录每个重复组的处理前后、处理原因、影响范围和回滚方式。
5. 输出治理报告和治理状态数据。

产物：

```text
docs/resource-governance-report.md
client/src/design-assets/generated/resource-governance.generated.json
已处理的 A / B 类治理 diff
C 类待确认清单
```

### 重复治理实际处理基线（阶段 5 产物）

> 本节为阶段 5 的产物说明。阶段 5 在阶段 4 分级基线上**做实际治理**：A 类清理冗余副本、B 类引用替换（本仓库 B=0）、C 类只标待确认。核心纪律：**删除是破坏性动作，不信任上游结论、逐候选独立现场复核**，任一红线不过即跳过转待确认；registry 事实源 / 品牌·渠道·头像红线 / 被引用资源永不删；绝不删最后一份（删前确认同组 canonical 在场且字节完全一致）。脚本默认 dry-run，`--apply` 才真正删除；删除全部由 git 跟踪可回滚。**本阶段不建 canonical 入口（阶段 6）、不动组件源码、不动 skill（阶段 9）。**

**1. 治理脚本（可重复执行，默认 dry-run 不删）**

```text
client/src/design-assets/scripts/apply-governance.mjs
```

- 输入：阶段 4 `resource-duplicates.generated.json` + 阶段 2/3 `resource-inventory.generated.json` / `resource-usage.generated.json`（只读）。
- 流水线顺序固定：`scan-resources → classify-resources → detect-duplicates → apply-governance`。
- 运行：`node client/src/design-assets/scripts/apply-governance.mjs`（演练）/ `… --apply`（实际清理）。

**2. 删除前独立现场复核（逐候选执行，任一不过即跳过转 needs-review）**

- 组分级=A 且组非红线；成员 `role=duplicate` 且 `action=remove-redundant`。
- 成员非红线类目（`brand-logo`/`channel-icon`/`agent-avatar`）、非红线视觉（`brand-fixed`/`avatar-like`）。
- `usageCount=0` **且** `dynamicDirReferenced=false` **且** usage 表无该 id **且** 不在 `unresolvedRefs` 命中 **且** 父目录不在 `dirReferences`。
- 候选文件在磁盘、同组 canonical 文件在磁盘、且候选与 canonical **现场字节 hash 完全一致**（内容已被保留的副本承载）。

> `usageCount=0` 单独不构成删除依据；删除依据是「无任何引用线索 + 同组 canonical 在场且字节完全一致」。

**3. 治理结果（本次基线，与阶段 4 分级一致）**

- 84 组：A 73 / B 0 / C 11。A 类 **127** 个 remove-redundant 候选**全部通过独立复核并已删除**（skipped=0），回收 **185244** 字节。
- 被删项绝大多数为根 `assets/CodeBuddyAssets/` Figma 导出重复件，及少量 `client/public/icon/X 2.svg` 拷贝、`client/public/assets/admin-security/runtime-control.svg`（与 `icon-enterprise-security.svg` 字节一致）、`empty-agent-template.png`（与 `empty-resource-management.png` 字节一致）；内容均已由同组保留的 canonical 承载，运行时与页面引用不受影响。
- **B 类引用替换：0 项**。阶段 4 已核实唯一含「>1 被引用成员」的组是渠道红线组 `dup-074`，被红线正确判为 C 而非 B。
- **C 类 11 组只标待确认、不删不换**：红线 2 组（`dup-074` 渠道 wecom/wecom-app、`dup-075` `clawpro-logo.svg` 与 Figma 导出 `747_183/19.svg`）、keep-multiple 8 组（`public/icon/X.svg` 运行时 canonical + `icon/X.svg` registry 事实源的有意双存在）、文件名疑似 1 组（`dup-083` 已关机 png/svg 跨格式同名）。

**4. 产物**

```text
docs/resource-governance-report.md                                     # 治理报告（由脚本据治理事实渲染）
client/src/design-assets/generated/resource-governance.generated.json  # 治理状态数据（含 removedIds / removed / skipped / cNeedsReview / 逐组 state）
client/src/design-assets/scripts/apply-governance.mjs                  # 治理脚本
已删除的 127 个 A 类冗余副本（git diff 可审计、可回滚）
```

**5. 据实记录 / 交后续阶段的事实**

- 治理状态数据 `resource-governance.generated.json` 为**治理事实记录**；阶段 2/3/4 的 inventory/duplicates 仍保留为**治理前基线**，资源库单页（阶段 7）用其中的 `removedIds` 隐藏已删资源、并用逐组 `state`（`duplicates-removed` / `keep-multiple` / `pending-review`）展示治理进度。
- **C 类 8 个 keep-multiple 组**（`public/icon/` 运行时 canonical + `icon/` registry 事实源，二者字节一致但用途不同）**结论已定为方案 A：长期保留双份现状，不物理合并、不动 `/icon/` 运行时路径**；隐患是双份可能漂移（单边改动致两边不一致），故**留阶段 6 顺手补一致性校验**（比对同名文件字节是否一致、有无单边新增/缺失，漂移即报警），方案 B 物理合并因迁移面大、超出"不做全量迁移/强注册"边界暂不采纳。阶段 5 一律不动。
- 治理后若重跑 `scan-resources → classify-resources → detect-duplicates`，将反映治理后现状（重复组收敛），属正常；本阶段产物已固化治理前后对照与回滚信息。
- canonical 入口（`canonicalId` / `canonical-assets.ts`）仍留阶段 6；本阶段只做实际清理与状态记录，未建立统一入口。
- **C 类红线确认闭环（后续迭代补充，2026-06-14）**：原 C 类红线重复组长期挂「待人工确认」无收尾出口，为此新增人工确认事实源 `client/src/design-assets/manual-overrides/duplicate-review.json`（按成员 id 排序集合稳定匹配，不受组号重排影响）；`detect-duplicates.mjs` 读取它打标 `reviewStatus`（`confirmed` / `pending`），`apply-governance.mjs` 据此产出新状态 `reviewed-confirmed` 并移出待确认清单，资源库单页 `DesignSystemAssets.tsx` 新增绿色「已人工确认」状态展示。据此处理两组红线：①「企业微信 / 企业微信应用」渠道字节一致组（基线 `dup-074`）经人工确认为「两渠道有意暂复用同一品牌图标」，两文件均保留、不归并，转 `reviewed-confirmed`，待设计给「企业微信应用」独立图标后再单独替换 `channel-wecom-app.svg`；②品牌 Logo 红线组（基线 `dup-075`）中无引用的 Figma 导出副本 `assets/CodeBuddyAssets/747_183/19.svg`，与全仓无任何代码引用的 `client/src/assets/topnav/clawpro-logo.svg`（横版带文字 logo，经核查 TopNav 实际用 `/landing-assets/60.svg`、原「由 TopNav 使用」注释为错误并已修正）均已删除（git 可回滚），相应字节一致 / 文件名相似组随之消失。当前红线 C 类待确认已清零（`reviewedConfirmedGroups` 计数生效）；keep-multiple 8 组维持「方案 A 保留双份」现状不变。注意：完整流水线为 6 阶段（…→ `apply-governance` → `build-canonical-assets`），删改资源后须跑完整链路（含阶段 6 回填 `governance.canonical`），否则页面 `canonical 入口` 视图会因缺段报错。

## 阶段 6：canonical 资源入口建设

目标：让高频、已确认 canonical 的资源具备“一改多处生效”的基础能力。

工作内容：

1. 从治理结果中筛选高频 canonical 资源。
2. 新增或维护 `client/src/design-assets/canonical-assets.ts`。
3. 对已治理重复资源涉及的页面层 / 非组件源码引用，优先替换到 canonical 文件或 `canonicalAssets` 入口；组件源码内引用只记录风险，不在本阶段替换。
4. 在资源详情中展示“是否已接入统一入口”。
5. 在治理报告中记录哪些资源已具备一改多处生效能力，哪些仍是散落引用。
6. 依据本阶段 canonical 接入产出，回填《ClawPro图标与资源库建设计划》§5.3 `canonicalAssets` 示例中清空的占位（具体 key 与资源路径、组织维度），使文档示例与真实 `canonical-assets.ts` 对齐。

产物：

```text
client/src/design-assets/canonical-assets.ts
resource-inventory.generated.json 中的 canonical 字段
resource-governance.generated.json 中的 canonical 接入状态
docs/resource-governance-report.md 中的 canonical 接入记录
```

### canonical 入口建设基线（阶段 6 产物）

> 本节为阶段 6 的产物说明。阶段 6 在阶段 5 治理后现状上**建立 canonical 统一入口并回填接入状态**。核心纪律：**只收已确认 canonical**（status=normal）、**红线不归并**（品牌/渠道/头像各自一 key）、**不动组件源码、不动 skill（阶段 9）、不做全量迁移**；脚本校验失败即停、幂等可重复。

**1. 入口脚本（可重复执行，校验失败即停、幂等）**

```text
client/src/design-assets/scripts/build-canonical-assets.mjs
```

- 流水线顺序固定：`scan-resources → classify-resources → detect-duplicates → apply-governance → build-canonical-assets`。
- 输入：阶段 2~5 的 `resource-inventory / resource-usage / resource-duplicates / resource-governance` JSON（只读）。
- 校验：SPEC 中每个条目必须在 inventory 命中、未被阶段 5 删除、磁盘存在、`primaryCategory` 与组织期望一致、`status=normal`，任一不符立即报错退出、不写任何产物。
- 接入检测：扫描 `client/src` 中对 `canonicalAssets.<group>.<key>` 的真实引用，据实回填「是否已接入入口」。
- 幂等：重跑先清空 inventory 全量 canonical 字段再写入，报告段按 `<!-- CANONICAL:START/END -->` 标记替换。

**2. 入口纳入口径（仅已确认 normal 业务专属资源）**

- 纳入 **14** key：`brands 1`（ClawPro Logo）/ `channels 6`（微信·QQ·企微·企微应用·钉钉·飞书）/ `avatars 7`（default·designer·analyst·creator·developer·pm·operator）。
- 组织维度 `brands / channels / avatars` 来自治理后存活业务类目的真实结论，非预设。
- **暂不纳入的高频资源**：多处使用的 `/icon/*.svg`（域名/所在地域/关联腾讯云账号/api文档/arrow-left-stroke/功能上新/上传图片默认/体验优化）与 `empty-aiagent.png` 当前 `status=needs-review`（多属 C 类 keep-multiple 组 dup-076~084），按纪律不作为「已确认 canonical」，待其经设计确认转 normal 后再纳入。
- **红线不归并**：`channels.wecom` 与 `channels.wecomApp` 字节一致（dup-074 红线）仍分列两 key；`brands.clawproLogo` 命中 dup-075 红线（与某 Figma 导出字节一致），不影响其作为入口。
- **品牌 Logo 第二物理副本**：`@/assets/topnav/clawpro-logo.svg`（src，TopNav 组件使用）与入口的 sidebar Logo 为不同物理文件，本阶段不合并、不改组件源码。

**3. 接入与一改多处生效**

- **已接入**（6 key）：`channels.*` 已由页面层 `client/src/pages/admin/ChannelConfig.tsx` 改用 `canonicalAssets.channels.*`（行为等价，路径不变）；修改 `canonical-assets.ts` 中其路径即统一影响该页。
- **仅建入口、暂未接入**（8 key）：`brands.clawproLogo` 与 `avatars.*`，现有散落引用保持原样、可增量接入，**不做全量迁移**（边界）。
- **已治理重复资源的引用迁移**：无需替换——阶段 5 实际移除的 127 个 A 类冗余副本均无任何引用线索，故无页面层 / 非组件源码引用需迁移。
- **组件源码**：`AgentAvatar.tsx`（`ROLE_AVATAR` 角色→头像映射）、`TopNav` 等组件内引用按共享组件处理，本阶段只记录、不改源码、不引入口；如确需组件迁移须按阶段 8 单独立项评估开发仓库兼容性。

**4. 产物**

```text
client/src/design-assets/canonical-assets.ts                            # 入口本体（脚本生成，含用途/边界注释）
client/src/design-assets/scripts/build-canonical-assets.mjs             # 入口构建脚本（可重复执行）
client/src/design-assets/generated/resource-inventory.generated.json   # 回填 canonicalStage=6 / canonicalSummary / 逐项 isCanonical·canonicalId·canonicalKey·connectedToCanonicalEntry
client/src/design-assets/generated/resource-governance.generated.json  # 回填 canonical 接入状态（entries / adoption / oneChangeMultiEffect）
docs/resource-governance-report.md                                     # 第十节「canonical 接入记录（阶段 6）」
```

**5. 据实记录 / 交后续阶段的事实**

- 入口值统一为 public webPath 字符串（与现有 `ChannelConfig.CHANNEL_ICON_SRC`、`AgentAvatar.ROLE_AVATAR` 的 web 路径用法一致），`tsc --noEmit` 通过、无新增 lint 错误。
- `canonicalId` / `canonicalKey` 采用 `group.key`（如 `channels.wecom`），供阶段 7 资源库单页详情展示「是否 canonical / canonicalAssets key / 是否已接入统一入口」。
- 入口只引用已确认资源，未引用 registry，不与 `icon-registry.example.json` 漂移；skill 连接（skill-map / assets-icons.md）仍留阶段 9，本阶段不动 skill。

## 阶段 7：资源库单页落地

目标：实现 `/design-system/assets` 页面，让设计团队可以浏览、筛选、查看详情和治理结果。

页面能力：

1. 顶部 3-4 个核心统计。
2. 左侧分类导航。
3. 搜索与组合筛选。
4. 资源网格 / 列表展示。
5. 资源详情面板。
6. 使用位置查看。
7. 复制引用方式。
8. 重复治理组展示。
9. 未分类 / 待确认视图。
10. canonical 状态和统一入口接入状态展示。
11. 适合组件槽位展示。

产物：

```text
/design-system/assets
资源详情面板
重复治理视图
未分类 / 待确认视图
```

## 阶段 8：组件资源使用映射与风险约束

目标：只针对使用自有 SVG / 图片资源的组件建立映射和风险约束；继续使用 `lucide-react` 的组件保持现状。

工作内容：

1. 梳理当前使用自有 SVG、PNG、Agent 头像、渠道图标、品牌 Logo 的组件。
2. 将所有组件统一标记为“开发仓库已使用 / 共享组件”。
3. 为这些组件建立“推荐资源映射”，记录资源槽位、当前资源、推荐资源类型和风险等级。
4. 在资源库详情中展示“适合组件槽位”。
5. 在检查脚本中识别组件源码错误引用 `canonicalAssets`、当前项目 `/assets/...`、inline SVG、emoji 图标和错误品牌资源用法。
6. 对仅使用 `lucide-react` 的组件不做资源库改造。
7. 如果确实需要改组件源码，必须脱离本资源库任务单独立项，并评估开发仓库兼容性。
8. 依据本阶段组件审计产出，回填两份文档中标注为“示例 / 待阶段 8 审计”的占位：《ClawPro图标与资源库建设计划》§3.7 组件槽位白名单、§7.3 纳入映射的组件清单，使其从举例收敛为真实清单。

产物：

```text
使用自有 SVG / 图片资源的组件清单
所有组件均按开发仓库已使用的共享组件标记
资源库详情面板中的组件槽位信息
组件资源使用风险约束
```

### 组件资源映射与风险约束基线（阶段 8 产物）

> 本节为阶段 8 的产物说明。阶段 8 **只针对使用自有 SVG / 图片资源的组件建立映射与风险约束**；仅用 `lucide-react` 的组件保持现状。核心纪律：**不改任何组件源码、不改组件 icon API、不做全量组件重构、不引注册体系、不动 skill（阶段 9）**；所有组件统一按「开发仓库已使用 / 共享组件」处理；映射与白名单**全部以真实组件源码审计为准，不臆测**。

**1. 事实源（人工策展，非生成）**

```text
client/src/design-assets/manual-overrides/component-resource-map.json
```

- `slots`：9 个可用组件槽位白名单（6 个自有 SVG 槽位 admin-sidebar / card-left-icon / number-card / file-type / run-status / feature-card + 3 个业务资源槽位 agent-avatar / channel-icon / brand-logo），每槽位含 `recommendedResourceType / allowLucideFallback / riskLevel / redline / currentResource / hostInjection / constraint`。
- `components`：8 个纳入映射的组件清单（对应 §7.3），逐项标 `isShared=是 / currentResourceSource / resourceSlot / recommendedResourceType / allowLucideFallback / currentRisk / recommendedUsage / needsSeparateRefactor`。
- `findings`：审计中据实发现的两处不一致（见下文第 3 点）。
- `boundary` / `checkRules`：硬边界提醒与检查脚本落地说明。

**2. 资源库详情面板接入（只读展示，不臆测）**

- `client/src/pages/DesignSystemAssets.tsx` 详情面板「适合组件槽位」已由占位替换为**真实白名单展示**：按资源 `classification.componentSlot`（自有 SVG）或 `primaryCategory`（头像 / 渠道 / 品牌）解析到对应槽位，展示风险等级、推荐资源类型、是否允许 lucide fallback、是否红线、槽位约束与跨仓注入约束；未匹配专属槽位的资源给出据实说明。头注释已同步更新指向本阶段事实源。

**3. 据实记录的关键发现（不改组件、不删文件）**

- `run-status`：分类将 8 个 `src/assets/agent-card/status-*.svg` 标为 run-status 槽位，但 `StatusBadge` 实际以组件内 **inline 动画 SVG** 渲染状态，未通过这些静态路径引用 → 静态文件按设计源 / 低引用记录。
- `file-type`：分类将 21 个 `src/assets/icons/file-space/*` 标为 file-type 槽位，但 `FileSpace` 实际以 **inline SVG** 渲染文件类型与工具栏图标，仅 ESM import 4 个关闭态功能介绍图标 → 据实记录，如抽资源 / 改 lucide 须单独立项。
- 硬编码当前项目 `/assets` 路径的组件（存量风险、本阶段不修）：`AgentAvatar`（7 头像 PNG）、`Empty`（empty-aiagent PNG，资源已 `excludeFromLibrary`）、`ComparisonTable`（version-compare SVG）。
- 正确范例：`ChannelConfig` 已在页面层经 `canonicalAssets.channels.*` 消费渠道图标（阶段 6 接入），组件不写死路径。

**4. 项目侧检查脚本（基线模式，只防新增；手动可选、未接 CI）**

```text
client/src/design-assets/scripts/check-component-resource-usage.mjs   # wire: npm run check:component-resources
```

- 扫描 `client/src/components/**`，按规则独立基线（更新时间 2026-06-14）：`component-imports-canonical` 基线 **0**（零容忍：组件禁 import canonicalAssets）、`component-hardcoded-assets` 基线 **18**、`inline-svg` 基线 **40**、`emoji-icon` 基线 **18**（后三者锁存量、防新增，目标趋于 0）。
- 仿 `scripts/check-card-shadow.mjs` 基线机制：当前计数 ≤ 基线即 `exit 0`；超基线（新增违规）`exit 1`；`STRICT=1` 任何违规即报错；`COUNT=1` 仅打印计数供校准；行级豁免 `// allow-asset: <理由>`。
- `brand-resource-wrong-slot`（红线资源误当普通图标改色 / 误进普通槽位）静态检测易误报，作为**手工 CR 约束**记录于事实源，不自动扫描。
- **触发方式（重要边界）：当前为「手动可选」，不会自动触发**。该脚本仅注册为 `package.json` 的 `check:component-resources`，**未接入任何 CI（无 `.github/workflows`）、git hook（仅默认 `.sample`）、husky / lint-staged，也未并进 `build` / 默认 `check`**。只有人工执行 `npm run check:component-resources` 时才运行；不执行则零副作用、不卡任何人，对日常开发 / vibe coding **无任何影响**。`component-resource-map.json` 中「不阻塞 CI」一语仅描述脚本的设计意图（将来若接 CI 也只拦新增、不卡存量），**不代表已接入 CI**。是否升级为 CI / pre-commit 门禁属后续决策，需团队评估「治理收益 vs 开发速度」后再定，本阶段不擅自接入。

**5. 边界与交后续阶段的事实**

- 全程零组件源码改动、零 skill 改动（skill 连接 / `resource-skill-map.json` / `assets-icons.md` 仍属阶段 9）；`tsc --noEmit` 通过、无新增 lint（仅存量 `execCommand` 弃用 hint）。
- 阶段 9 据本阶段 `component-resource-map.json` 的 `slots` / `components` 生成 `resource-skill-map.json` 与 `assets-icons.md` 选图规则时，须沿用此处「页面层用 canonical、组件 / 跨仓由宿主注入、组件禁引 canonical / `/assets`」的口径，不漂移。

## 阶段 9：skill 连接与规则落地

目标：让 `clawpro-portable-design-skill` 能按使用上下文选择资源，避免把当前项目资源路径带到共享组件或开发仓库。

工作内容：

1. 基于资源清单和组件资源映射生成或维护 `resource-skill-map.json`。
2. 更新 `references/assets-icons.md`。
3. 增加使用上下文判断：当前项目页面、当前项目页面级非组件代码、组件源码、开发仓库 / 跨仓页面。
4. 增加 skill 禁止规则：共享组件和跨仓代码不得引用 `canonicalAssets` 或当前项目 `/assets/...`。
5. 增加 skill-map 校验，校验 `usageScope`、资源状态、路径存在性和槽位合法性。
6. 让 skill 在当前项目页面优先选择 canonical 资源；在跨仓 / 共享组件场景优先使用 lucide、宿主注入资源或标记 `needs-design-confirmation`。
7. 对已经使用 lucide 的组件，skill 继续生成 lucide 用法，不强制替换为自有 SVG。
8. 依据本阶段 `resource-skill-map.json` 与 `assets-icons.md` 的落地产出，回填两份文档中依赖审计的选图占位：《ClawPro图标与资源库建设计划》§5.5 槽位字段表与示例 JSON、§6.3 选图规则，以及《ClawPro图标与资源库背景目标》§六 图标选择决策模型；使文档据实补全，与 skill-map / skill 不漂移。
9. 评估是否为 `needs-design-confirmation` 占位增加可追踪校验：在「仅搜索捞取 / CI warning / CI fail」三选一中确定方案，需结合阶段 8 组件审计的真实产出与使用上下文判断（current-project / 组件源码 / 跨仓）一起设计，避免孤立加 grep 规则；如确定加校验，落点在 `check-design-usage.mjs`，按 skill 改动纪律以阶段 8 产出为准、改前比对原文、不丢原有规则。

产物：

```text
client/src/design-assets/resource-skill-map.json
.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md
skill-map 校验规则
skill 使用上下文规则
```

### skill 连接与规则落地基线（阶段 9 产物）

- **生成器**：`client/src/design-assets/scripts/build-resource-skill-map.mjs`（`npm run build:skill-map`）。只读 inventory（join 表）+ `component-resource-map.json`（阶段 8 槽位约束）+ governance（`removedIds`），**确定性派生** `client/src/design-assets/resource-skill-map.json`，不读页面、不臆造候选、可重跑幂等。
- **产物结构**：9 槽位白名单（取自阶段 8）+ 每槽位约束 + `candidates`；附 `registrySample`（可移植样例，非闸门）、`usageScopeLegend`、`summary`。
- **候选规模**：**155**（槽位 141：card-left-icon 64 / number-card 22 / admin-sidebar 22 / file-type 21 / run-status 8 / feature-card 4；红线 14：avatars 7 / channels 6 / brands 1）。
- **候选准入口径**：`status=normal` 且未排除 / 未归档，且（有 `componentSlot` 或带红线 `canonicalKey`），且不在 `removedIds`。红线 slot = `primaryCategory`（`channel-icon` / `brand-logo` / `agent-avatar`），`usageScope=host-injected`、经 `canonicalAssets` 引用；非红线 `usageScope=current-project-only`、经 `webPath` / `importPath` 引用。
- **`canUseInSkillMap` 落法**（用户拍板 b）：只在生成器内据规则计算并产出到 skill-map，**不回写** `generated/resource-inventory.generated.json`（保持其纯扫描 / 分类 / 治理基线产物）。
- **registry 降格**（用户拍板 A）：`icon-registry.example.json` 不再作 `approved` 准入闸门，降格为 skill 可移植身份样例；本项目候选真相为 inventory（审计表明带确认槽位资源 141 + 红线 14，与 28 条样例几乎无交叉，848 项仅 1 项 registered）。
- **校验脚本**：`client/src/design-assets/scripts/check-resource-skill-map.mjs`（`npm run check:skill-map`，fail-fast）。校验 id 在 inventory、`status=normal`、不在 `removedIds`、slot 白名单、`landing=file` 磁盘存在、`usageScope` 合法、红线落对槽位、`allowLucideFallback` / `recommendedResourceType` 不漂移、summary 计数一致。已通过（155 候选 / 9 槽位）。
- **skill 文档**：更新 `references/assets-icons.md`——保留 lucide 优先原立场与原有命名 / SVG / 禁止 / 状态等规则，新增顶部语境分流（跨仓 registry 样例 vs 当前项目 skill-map）、§1 优先级措辞、§5.5 槽位选图表（9 槽位 + 红线 + usageScope + lucide 回退）、§9 校验脚本三件套。
- **文档回填**：本计划 §5.5（真实结构 + 口径修正）、§6.3（选图规则）、§6.5 第二 / 三层（删 `approved` 闸门、脚本已落地）；背景目标 §六（决策模型据实化）、§七 7.1（registry 降格、inventory 为真相）、§八第 11 条。
- **阶段 9 第 9 项（`needs-design-confirmation` 可追踪校验）**：本次**未**新增独立 grep 规则，沿用 `check-design-usage.mjs` 现状——避免孤立加规则；skill-map 校验已覆盖候选准入与红线落槽，`needs-design-confirmation` 作为无合适候选时的人工标记由 CR 把关，符合「不臆测、以真实产出为准」纪律。
- **纪律遵守**：全程不改组件源码、不改组件 API、不做迁移；改 skill / 文档前比对原文、以阶段 8 真实审计为准、不丢原有合理规则（lucide 优先、红线硬约束、不漂移、三层机制均保留）。
- **沉淀记录（溯源 + 接入 SOP）**：阶段 9 的决策背景与日常接入流程已沉淀为两份配套文档，供后续接手者 / AI 溯源与扩展：
  - `docs/ClawPro资源库-阶段9决策溯源(ADR).md`——阶段 9 解决什么、产物链路、决策 A（registry 降格）硬证据（git 时间线 / 848-28-1 交叠 / registry 自述）、三层兜底、守门 10 项、关键不变量。**接手前必读，防误改 155 候选与红线。**
  - `docs/ClawPro资源库-新资源接入SOP.md`——4+1 个接入场景的可执行命令（普通图标 / 红线 / 废弃替换 / 删冗余 / 新增槽位类型）+ 验收清单 + 红线纪律。**核心结论：日常加图标只需喂进流水线、在 `classification.json` 指派槽位、跑 `build:skill-map` + `check:skill-map` 两条命令，无需改 skill；仅「新增全新槽位类型」才动 skill。**

## 阶段 10：安全校验与回归检查

目标：确保资源库可以安全落地，不影响现有业务页面和开发仓库已使用的共享组件。

工作内容：

1. 校验 SVG 安全规则。
2. 校验 `resource-skill-map.json` 中的资源都存在且未废弃。
3. 校验 `canonicalAssets` 中的路径都存在。
4. 校验业务代码未新增危险 SVG、emoji 图标、未登记资源路径。
5. 校验所有组件源码均未新增 `canonicalAssets` 或当前项目 `/assets/...` 依赖。
6. 校验 A / B 类治理 diff 清晰、可 review、可回滚。
7. 校验页面只消费静态 JSON 数据，不执行扫描、上传、删除、编辑文件动作。

产物：

```text
资源库检查脚本结果
资源治理报告最终版
构建 / 类型 / lint 检查结果
```

## 阶段 11：资源库落成验收与交付

目标：资源库达到“可浏览、可治理追踪、可服务组件、可服务 skill”的完成状态。

验收内容：

1. 页面可打开。
2. 分类浏览符合设计团队理解方式。
3. 自有 SVG 支持使用场景和视觉类型细分。
4. 重复资源组可以查看。
5. 本次治理有实际结果和报告。
6. 高频 canonical 资源已有统一入口或说明暂不接入原因。
7. 使用自有 SVG / 图片资源的组件已建立资源映射和风险约束。
8. 继续使用 `lucide-react` 的组件保持现状。
9. skill-map 能被校验。
10. 页面能展示 canonical、使用位置、组件槽位、治理状态。
11. 不引入 npm 包、不做全量迁移、不影响业务页面和开发仓库已使用的共享组件。

交付物：

```text
/design-system/assets
client/src/design-assets/generated/*.json
client/src/design-assets/canonical-assets.ts
client/src/design-assets/resource-skill-map.json
docs/resource-governance-report.md
.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md
```

---

# 十、验收标准

## 10.1 页面验收

- 可以打开 `/design-system/assets`。
- 可以看到资源总览统计。
- 可以按一级分类浏览资源。
- 自有 SVG 可以按使用场景细分。
- 自有 SVG 可以按视觉类型筛选。
- 可以搜索资源名称、文件名、路径、标签。
- 可以查看资源详情。
- 可以查看不同尺寸和不同背景下的预览。
- 可以查看资源使用位置。
- 可以复制使用参考代码。
- 可以查看重复资源组。
- 可以查看资源是否为 canonical，以及是否已接入统一入口。
- 可以查看未分类 / 待确认资源。
- 不存在“推荐复用”独立视图。

## 10.2 治理验收

- 已生成重复资源组织。
- A 类重复资源已完成清理。
- B 类明确重复资源已完成小范围治理或给出无法治理原因。
- C 类资源已进入待确认清单。
- 已生成 `docs/resource-governance-report.md`。
- 页面可以展示治理状态。
- 高频 canonical 资源已接入 `canonicalAssets`，或在治理报告中说明暂不接入原因。
- 对已接入统一入口的资源，修改 canonical 文件或入口映射可以影响所有接入处。
- 品牌、渠道、Logo 资源未被自动错误归并。

## 10.3 组件资源映射与风险约束验收

- 已梳理使用自有 SVG / 图片资源的组件清单。
- 所有组件均已标记为开发仓库已使用的共享组件。
- 已为这些组件建立推荐资源映射，包含资源槽位、当前资源、推荐资源类型和风险等级。
- 资源详情面板可以展示适合使用的组件槽位。
- 继续使用 `lucide-react` 的组件保持现状，不因资源库建设改造。
- 所有组件源码均未新增 `canonicalAssets` 依赖。
- 所有组件源码均未新增当前项目 `/assets/...` 路径依赖。
- 品牌固定色、渠道图标、Agent 头像不会进入普通 icon 槽位。
- 如需改组件源码，已在治理报告中标记为“不属于本资源库任务，需单独立项评估开发仓库兼容性”。

## 10.4 skill 验收

当团队要求生成新页面 / 组件时：

- skill 能读取或引用 `resource-skill-map.json` 的规则。
- skill 先判断使用上下文：当前项目页面、当前项目页面级非组件代码、组件源码、开发仓库 / 跨仓页面。
- 当前项目页面中，skill 可以优先生成 canonical 资源或 `canonicalAssets` 入口的引用。
- 组件源码和开发仓库 / 跨仓页面中，skill 不引用当前项目 `canonicalAssets` 或 `/assets/...` 路径。
- 已使用 lucide 的组件继续生成 `lucide-react` 用法，不强制替换为自有 SVG。
- 不出现 emoji 图标。
- 不新增未登记 inline SVG。
- 不随意引用 Landing 图片。
- Agent 头像、渠道图标、品牌 Logo 来自资源库候选或宿主仓注入，不被当成普通图标改色。
- 自有 SVG 只在符合使用场景、视觉类型和使用上下文的槽位使用。
- 没有合适资源时标记 `needs-design-confirmation`，不私自画图。

## 10.5 安全验收

- 页面不执行扫描命令。
- 页面不上传、删除、编辑资源文件。
- 页面不使用未清洗 SVG HTML。
- SVG 安全检查通过。
- `resource-skill-map.json` 校验通过。
- 无新增危险 SVG。
- 无新增未登记资源路径。
- 所有组件源码均未新增 `canonicalAssets` 或当前项目 `/assets/...` 依赖。
- 不影响现有业务页面和开发仓库已使用的共享组件。
