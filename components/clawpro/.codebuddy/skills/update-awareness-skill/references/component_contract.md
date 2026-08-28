# 引导组件接入契约

本文件是 `onboarding_component_spec.md` 的实现侧摘要。前者说明组件体系全貌，本文件只回答三个问题：

- 这次更新应该落到哪个真实组件
- `Implementation Plan` 里要写哪些稳定字段
- 页面植入时必须遵守哪些实现约束
- 如何在 onboarding 组件、项目既有组件、设计系统组件和占位适配器之间选择

## 读取顺序

1. 先读 `scenario_rules.md` 判定场景。
2. 再读 `surface_rules.md` 选择组件组合。
3. 按端类型读取设计规范：管控端读 `clawpro-portable-design-skill/SKILL.md` 与 `references/components.md`；用户端读 `references/tenant.md` 与 `references/components.md`。
4. 最后读本文件，把抽象组件名映射到真实组件和 `type_id`。

## 真实组件清单

| 中文组件类型 | React 组件名 | `type_id` | 适用端 | 典型用途 |
| --- | --- | --- | --- | --- |
| 强提醒弹窗 | `GuideGlobalModal` | `global-modal` | both | 重大更新、规则确认、账号/合规告知 |
| 用户端全局产品动态浮窗 | `GuideModuleFloat` | `module-float` | tenant | 高影响更新的跨页面触达，固定含 16:9 配图区 |
| 管控端产品动态卡片 | `GuideAdminNotify` | `module-float` | admin | 管控端产品动态卡片，多条自动合并 |
| 导航预览气泡 | `GuideNavBubble` | `nav-bubble` | admin | 一级导航、侧边栏、入口搬迁说明 |
| 单点提示气泡 | `GuidePointBubble` | `point-bubble` | both | 按钮、筛选、字段、局部说明 |
| 导航提示条 | `GuideUpdateBar` | `update-bar` | admin | 管控端导航下方强提醒 Banner/公告条；本 skill 只识别并交接，不开发 |
| 更新记录抽屉 | `GuideChangelogDrawer` | `changelog-drawer` | admin | 汇总版本记录、承接“查看详情” |
| 高亮步骤引导 | `GuideHighlightBubble` | `highlight-bubble` | both | 多步骤、新路径迁移、聚焦定位 |
| 新功能标签 | `GuideNewTag` | `new_tag` | both | `New`、`即将开放` 或自定义短标签 |
| 产品动态抽屉 | `ProductUpdatesDrawer` | `product-updates-drawer` | admin | 管控端产品动态详情承接 |
| 不展示 | 无 | `none` | both | 纯视觉优化或无需提示 |

说明：

- `GuideAdminNotify` 是管控端产品动态卡片，但实现归属于 `module-float` 浮窗能力，`type_id` 固定写 `module-float`。
- `GuideUpdateBar` 由本 skill 识别为管控端 Banner/公告条需求，但公告文案终稿、开发、植入和关联详情条目均交由独立流程处理。
- `GuideAdminNotify` 实际固定在管控端左侧导航底部、用户账号上方；不要按用户端浮窗写成右下角。
- `GuideNewTag` 使用 `new_tag`；默认 `variant="new"` 显示 `New`，`coming-soon` 显示 `即将开放`，仅 `custom` 使用自定义 children。
- `ProductUpdatesDrawer` 是产品动态详情承接组件，不是 `GuideFlow.steps[].componentType` 的步骤组件，但仍从 `@/components/onboarding` 统一导出。
- `GuideAdminNotify`、`GuideNewTag`、`ProductUpdatesDrawer` 也不属于 `GuideFlow.steps[].componentType`；按独立表面组件接入，不要塞入 `GuideFlow.steps`。
- `GuideOnboardingModal` 虽从统一入口导出，但用于首次登录/首次访问的新手欢迎流程，不属于版本更新感知组件；本 skill 不选择它替代 `GuideGlobalModal`。
- `OnboardingSimToggle`、`OnboardingDemoPanel` 仅用于预览或调试，禁止植入生产更新提示。

## 组件来源决策

先判断能力归属，再决定 import 和实现方式：

| 能力归属 | 首选来源 | 可占位 | 禁止 |
| --- | --- | --- | --- |
| 更新浮层、气泡、New Tag、更新条、全局弹窗、更新抽屉 | `@/components/onboarding` | 否，除非仓库确实没有该组件 | 业务页手写 `fixed` / `absolute` 拼同类浮层 |
| 页面 Alert、字段规则说明、原名标注 | 项目既有组件或设计系统组件 | 是 | 伪装成 `GuidePointBubble` 或新增 onboarding 变种 |
| 普通按钮、卡片、表格、空态、表单、Popover、Tabs | 项目既有组件或设计系统组件 | 仅缺组件时可最小占位 | `div + className` 重写已有组件视觉 |
| 管控端产品动态详情 | `ProductUpdatesDrawer` 或项目已有详情抽屉 | 是 | 新建一套独立详情弹层 |

设计规范要求能调用组件就调用组件，不要用临时 markup 拼视觉。组件缺失时，计划中必须写明缺少的是组件、props、状态持久化、埋点还是视觉 token。

## 抽象场景与真实组件的映射

| 抽象叫法 | 默认真实组件 | 备注 |
| --- | --- | --- |
| 导航提示条 | `GuideUpdateBar` | 仅管控端使用；本 skill 只识别并交接。用户端先选入口附近气泡/New Tag，只有高影响且需要跨页面触达时才升级为 `GuideModuleFloat` |
| 新功能标签 | `GuideNewTag` | 默认显示 `New`；存续建议 14 天，最长 30 天 |
| 页面入口气泡引导 | `GuideNavBubble` | 目标是导航、侧边栏、Tab 入口时优先 |
| 重要操作变更气泡引导 | `GuideHighlightBubble` | 页面重排或多步骤路径迁移时优先 |
| 功能入口附近气泡 | `GuidePointBubble` | 单元素说明默认选它 |
| 禁用入口说明气泡 | `GuidePointBubble` 或 `GuideNavBubble` | 看目标是否是导航入口 |
| 页面 Alert | 项目已有 Alert；无标准 onboarding 组件时用占位适配器 | 不要伪装成 `GuidePointBubble` |
| 侧边栏说明详情 | `GuideChangelogDrawer` 或项目已有侧边栏说明容器 | 优先复用真实抽屉容器 |
| 常驻操作引导 | `GuideHighlightBubble` | 通过多步骤和持久化控制常驻周期 |
| 强提醒弹窗 | `GuideGlobalModal` | 只用于高打扰度场景 |
| 日常更新提示 | `GuideModuleFloat` / `GuideAdminNotify` | 用户端仅高影响跨页面更新使用 `GuideModuleFloat`；管控端优先 `GuideAdminNotify` |

端侧映射必须落到真实组件：

- 管控端 `日常更新提示` → 识别为 `GuideAdminNotify` 需求，`type_id=module-float`，预期位置为左侧导航底部、用户账号上方，CTA 通常承接 `ProductUpdatesDrawer`。本 skill 只发起文案生成与审核交接，不决策或执行组件开发。
- 用户端 `日常更新提示` 只有在影响等级为高/重大、需要跨页面触达且轻量组件不足时才映射到 `GuideModuleFloat`，`type_id=module-float`。
- 用户端抽象命中 `导航提示条` 或页面入口气泡时，先改选 `GuidePointBubble`、`GuideNewTag` 或不展示；`GuideNavBubble` 仅用于管控端，不得仅因缺少用户端条幅就自动升级为更重的 `GuideModuleFloat`。
- 用户端命中侧边栏说明详情时，复用项目已有详情抽屉；项目缺少时仅允许最小占位适配器，不得使用仅管控端的 `GuideChangelogDrawer`。
- `ProductUpdatesDrawer` 仅展示在管控端。独立详情抽屉可以开发和登记；与 `GuideAdminNotify` 关联的条目必须随卡片交接，不由本 skill 开发或登记。

## 组件内容槽位

先确定真实组件和内容变体，再生成文案。禁止把统一的“标题 + 正文 + 去看看 + 知道了”复制给所有组件。

| 真实组件 | 支持的产品内容槽位 | 默认要求 |
| --- | --- | --- |
| `GuideGlobalModal` | 标题、说明、确认操作；次操作可选 | 只在确有第二条路径时生成次操作 |
| `GuideModuleFloat` | 副标题可选、标题、说明、单条主操作；多条模式使用翻页/跳过 | 不生成额外“知道了”，多条末页文案由组件控制 |
| `GuideAdminNotify` | 卡片标题、卡片描述、卡片按钮 | 只形成 `title / desc / btnText` 长度 brief 并标记待生成/待审核，不生成次按钮或开发终稿 |
| `GuideNavBubble` | NEW 标、标题、说明、预览图可选；存在跳转时显示主操作和稍后关闭 | 只有提供 `href` 时生成主操作 |
| `GuidePointBubble` | 标题、说明、副标题可选、列表可选；按钮仅 `text-button` 等需要操作的变体可用 | 默认 `text-only`，不生成按钮 |
| `GuideUpdateBar` | 公告正文、版本可选、详情入口可选 | 只形成待生成/待审核 brief；不输出可开发终稿 |
| `GuideChangelogDrawer` | 版本、条目标题、条目描述、标签、日期、外链可选 | 不生成统一 CTA 或“知道了” |
| `GuideHighlightBubble` | 每个步骤的标题、说明、列表可选 | 步骤导航由组件控制，不生成普通主次按钮 |
| `GuideNewTag` | 变体和短标签 | 默认 `new → New`；`coming-soon → 即将开放`；仅 custom 自定义文案，不生成标题、正文或按钮 |
| `ProductUpdatesDrawer` | 更新类型、所属端、标题、描述、日期、近期状态、可选体验链接 | 不生成统一 CTA 或“知道了” |

页面 Alert、字段规则说明和原名标注服从项目真实组件 props；未确认 props 时，只列出已知文本槽位并标记待确认，不虚构按钮。

`GuideAdminNotify` 的槽位定义只用于形成文案 brief。由本 skill 输出时，标题、描述和按钮必须标记为待生成/待审核，不得认定为开发终稿。

## 视觉素材要求

产品审核方案在以下情况增加 `visual_asset`，并明确“请联系设计团队进行图片设计；素材交付前不可上线”：

| 组件或变体 | 是否需要设计素材 | 建议规格 |
| --- | --- | --- |
| `GuideModuleFloat` | 必须；媒体区始终渲染 | 16:9，推荐 672×376 |
| `GuideGlobalModal` | 必须；无素材会显示 1080×608 占位画面 | 图片 1080×608，或产品确认的视频 |
| `GuideNavBubble` | 仅方案传 `image` 时 | 约 300×140 展示比例，按设计稿提供高清图 |
| `GuidePointBubble contentVariant="text-image"` | 必须 | 约 16:9，组件展示高 146px |
| `GuidePointBubble push-notice` | 仅方案传 `noticeImage` 时 | 组件展示高 128px，按设计稿提供高清图 |

组件源码里的渐变块、空白卡片或尺寸标记只用于占位/预览，不得作为正式上线素材。视觉素材属于产品确认字段；新增、取消或改变素材类型需提升 `plan_revision` 重新确认。

## GuideModuleFloat 使用门槛

只有影响等级为高/重大，并且满足至少一项时才选择：

- 上线普通用户必须知道的全新核心能力，且入口不容易自然发现
- 更新影响大量普通用户，错过告知会导致明显误解、失败或支持成本
- 更新本身跨多个页面生效，无法靠单一目标位置解释
- 同一版本包含多项高价值更新，需要合并为全局产品动态

以下情况不得使用：新增普通按钮、筛选、字段、下载/帮助入口、文案调整、纯视觉优化、普通问题修复。分别使用 PointBubble、NavBubble、New Tag、页面内说明或不展示。Product Review Plan 必须写 `selection_basis`，说明为何需要跨页面重提示，不能只写“用于告知更新”。

一旦命中，还必须主动提醒用户联系设计团队进行图片设计；图片交付并审核通过是执行前置条件，不得用源码占位图替代。

## 跨端产品动态判断规则

对每一项用户端变化执行独立的管理员管理/知情判断，不限于入口、New Tag 或跨端层。用户端变化影响配置、下发、更新、管理、权限、审核、治理、安全、数据访问、状态解释、统计、故障排查、用户支持或咨询口径时，在管控端生成产品动态卡片。用户使用云端 Agent 对话、模型调用、工具调用或外部连接等重要能力时，即使没有直接配置入口，也属于管理员需要知情：

这里的“生成产品动态卡片”仅指识别出 `GuideAdminNotify` 告知需求并生成文案 brief。必须提醒用户进行文案生成与审核；组件是否开发、如何开发和最终文案均不由本 skill 决策。

- 抽象类型：`日常更新提示`
- 真实组件：`GuideAdminNotify`
- `type_id`：`module-float`
- 适用端：管控端承载，内容描述用户端变化
- 标题建议：`用户端 | {功能名}`
- 详情承接：优先 `ProductUpdatesDrawer`，用 `relatedIds` 或 `detail_entry_id` 关联对应更新条目
- 挂载点：管控端左侧导航底部、用户账号上方；详情从卡片进入 `ProductUpdatesDrawer`
- 文案长度：标题建议 `≤16` 个中文字符，描述建议 `≤30` 个中文字符，按钮文案建议 `≤6` 个中文字符
- 文案职责：标题只写端前缀和能力名，描述只写管理影响，不要在标题里塞入“影响范围、说明口径、查看详情”等补充句
- 内容槽位对应：`卡片标题` → `AdminNotifyItem.title`，`卡片描述` → `AdminNotifyItem.desc`，`卡片按钮` → `AdminNotifyItem.btnText`

不生成管控端卡片的典型情况：

- 用户端只是新增下载 skill 的地方
- 帮助文档、教程、静态展示页、资源浏览页
- 不改变管理员配置、治理、支持、解释或风险认知的纯入口

管理员需要管理或知情时，这条卡片不能被用户端 `GuideNewTag` 替代。`GuideNewTag` 负责让普通用户看到新入口，`GuideAdminNotify` 负责让管理员知道用户侧新增能力及其管理、支持或治理影响。

如果是聚合卡片：

- 标题由组件自动聚合为 `管控端有 N 项新增` / `用户端有 N 项新增`
- skill 只需要保证每条原始卡片标题可独立成立，不要手写新的聚合标题模板

## Implementation Plan 必填字段

每个组件项都必须包含：

- `type`: 中文组件类型，保留业务语义
- `component_name`: 真实 React 组件名，例如 `GuidePointBubble`
- `type_id`: 稳定类型标识，例如 `point-bubble`
- `purpose`: 该组件解决的认知问题
- `trigger`: 展示触发条件
- `mount.route`: 页面路由
- `mount.selector`: 目标元素选择器；无法提供时写明原因
- `mount.target_label`: 目标元素中文名
- `mount.placement`: 气泡或挂载方向
- `mount.instance_scope`: `首个实例 | 指定实例 | 全局唯一`
- `design_status`: `已接入 | 占位 | 待确认`
- `component_source`: `@/components/onboarding | 项目既有组件 | 设计系统组件 | 本地占位适配器 | 无`
- `import_path`: 真实导入路径；未知时写 `待确认`
- `placeholder_allowed`: 当前场景是否允许占位
- `behavior`: 当前组件自己的展示次数、关闭方式、下线时间和持久化 key；不得直接复用组合中其他组件的行为

如组件需要承接详情，还应补：

- `detail_component_name`: 例如 `GuideChangelogDrawer` 或 `ProductUpdatesDrawer`
- `detail_entry_id`: 用于抽屉高亮的条目 id
- `surface_endpoint`: 组件实际展示在哪一端；管控端为 `管控端`，用户端为 `用户端`
- `content_endpoint`: 组件内容描述哪一端的更新；用户端变化同步到管控端时，分别写 `surface_endpoint=管控端`、`content_endpoint=用户端`
- `props`: 只填写统一入口真实导出组件支持的 props；内容变体必须显式落到 props

## 实现约束

### 0. 文案和生命周期

- 普通 onboarding 文案遵循共享软校验：标题 `≤14` 字、正文 `≤60` 字、CTA `≤6` 字。
- `GuideAdminNotify` 额外遵循卡片限制：标题 `≤16` 字、描述 `≤30` 字、按钮 `≤6` 字。
- `New Tag` 建议展示 14 天，最长 30 天；计划中必须写 `new_tag_expires_at` 或 `new_tag_remove_on_first_click=true`。
- 持久化 key 优先使用 `buildPersistenceKey(component, updateId)` 的格式：`onboarding.{component}.{updateId}.dismissed`。
- 埋点优先使用 `trackOnboarding` 和统一事件：`onboarding_impression`、`onboarding_click`、`onboarding_dismiss`。

### 1. 统一导入入口

onboarding 组件必须从统一入口导入：

```tsx
import { GuidePointBubble, GuideNewTag } from "@/components/onboarding";
```

不要直接跨文件深链导入具体实现。

页面基础组件必须按项目设计规范调用已有导出，例如 Button、Alert、SurfaceCard、TenantCard、Table、Empty、Dialog、Popover 等。不要为了更新提醒局部覆盖它们的核心颜色、圆角、阴影或层级。

### 2. 渲染控制

- `GuideGlobalModal`、`GuideModuleFloat`、`GuideAdminNotify`、`GuideNavBubble`、`GuidePointBubble`、`GuideChangelogDrawer`、`ProductUpdatesDrawer`、`GuideHighlightBubble` 使用 `open + onClose`；`open=false` 时不渲染。
- `GuideUpdateBar` 只有 `open`，没有 `onClose`，属于不可手动关闭的强提醒条；该实现信息仅供交接负责人参考，本 skill 不执行接入。
- `GuideNewTag` 没有 `open/onClose`；由业务条件、`isNewTagExpired()` 和首次点击状态决定是否渲染。
- 禁止用 CSS 隐藏来保留浮层 DOM；`GuideNewTag` 也应在条件不满足时直接不渲染。

### 3. endpoint 规则

- 组件支持 `endpoint` prop 时，管控端写 `endpoint="admin"`，用户端写 `endpoint="tenant"`；没有该 prop 的组件不得虚构。
- `GuideAdminNotify` 本体固定展示在管控端；`AdminNotifyItem.endpoint` 表示卡片内容所属端，而不是展示端。用户端更新同步给管理员时写 `item.endpoint="tenant"`。
- `ProductUpdateItem.endpoint` 同样表示更新内容所属端。
- 不允许通过业务侧样式分支复制一套新视觉

### 3.1 统一入口的导出边界

- 所有正式导入仍固定为 `@/components/onboarding`，禁止深链到组件文件。
- `GuideModuleFloat` 可直接接收 `sources`，但统一入口当前未导出 `ModuleFloatSource` 和 `mergeModuleFloats`；业务侧可以内联 `sources` 数据，不得为拿到类型或 helper 改用深链导入。
- `GuideOnboardingModal`、`OnboardingSimToggle`、`OnboardingDemoPanel` 虽然从统一入口导出，但本 skill 不使用。

### 4. GuideHighlightBubble 定位规则

- 优先提供稳定唯一的 `selector`
- 优先使用 `data-guide="xxx"` 一类属性
- `selector` 未命中且无兜底坐标时，本步骤不得渲染
- 多步骤切换时应允许自动滚动到目标区域

### 5. 重复组件的挂载范围

- 当目标元素来自列表卡片、表格行、瀑布流、宫格等重复渲染结构时，默认只挂到首个实例。
- 选择器应表达“首个实例”语义，例如稳定父容器下的首项 `data-guide` 标识，而不是让所有同类节点同时命中。
- 只有当某个非首项实例具备唯一业务含义时，才允许用 `指定实例`；计划中必须解释原因。
- 除非需求明确要求批量高亮，否则不要把同一条 `GuidePointBubble` / `GuideHighlightBubble` 绑定到多个重复实例。

### 6. GuideAdminNotify 合并规则

以下内容只作为后续负责人的实现参考。本 skill 不执行这些规则对应的代码或配置修改。

- 传入原始 `items`，每项填写正确的内容端 `endpoint` 和 `relatedIds`；使用 `variant="stacked"` 触发 `buildProductUpdateNotices()` 自动聚合
- 不要在 skill 中提前手写聚合卡标题或把多条原始更新丢掉
- 管控端最多 1 张，用户端最多 1 张，整体最多 2 张
- 同端聚合描述由各原始标题去掉端前缀后用顿号拼接；执行前校验拼接后仍 `≤30` 字，必要时缩短各能力名
- 点击 CTA 时优先打开 `ProductUpdatesDrawer`

### 6.1 GuideModuleFloat 合并规则

- 在用户端全局布局最多渲染一个 `GuideModuleFloat` 实例，跨页面共享同一未读、曝光和关闭状态。
- 登录后任一允许的用户端页面都可触发，不把展示 Trigger 绑定到功能目标页；CTA 可跳转到目标页。
- 排除登录页、首次 onboarding、全屏/沉浸任务和 GlobalModal 打开期间；路由切换不得重新计为一次曝光。
- 多个独立更新来源同时命中时传 `sources`：一个来源保留 `single`，两个及以上来源自动转为 `multi`，每个来源占一页并按 `order` 排序。
- 每个 `source.items` 在合并场景中只放一条主要内容。源码在单 source 时返回 `variant=single`，若该 source 放多条 items，界面只会展示第一条而没有翻页入口。
- 单个更新本身需要多页时，不使用单 source 合并路径，直接传 `variant="multi" + items`。
- `sources` 合并只解决同类浮窗的视觉冲突，不得把不同认知问题、目标用户、挂载点或 Trigger 合并成同一通知单元。

### 7. 占位适配器的边界

只有这些能力可以暂时占位：

- 页面 Alert
- 侧边栏说明详情
- 字段下方规则说明
- 新名称右侧原名标注

占位时必须在计划中写明：

- 最终要替换成什么真实组件或设计系统组件
- 当前缺什么：props、埋点、状态持久化或视觉
- 为什么不能直接复用 `@/components/onboarding`

占位适配器必须保持最小：只提供挂载和数据结构衔接，不复制完整视觉系统；后续替换目标必须指向真实组件或设计系统组件。

## 组件选择的硬规则

1. 需要用户明确知情或确认时，才允许使用 `GuideGlobalModal`。
2. 单元素提示默认先看 `GuidePointBubble`，不是 `GuideHighlightBubble`。
3. 导航入口级提示默认先看 `GuideNavBubble`；若管控端确需全局 Banner，则识别为 `GuideUpdateBar` 交接项。
4. 管控端“产品动态”类汇总识别为 `GuideAdminNotify + ProductUpdatesDrawer` 需求后，转交文案生成、审核和后续开发决策；本 skill 不执行。
5. 多步骤路径迁移优先 `GuideHighlightBubble`，并提供稳定选择器。
6. `none` 必须解释为什么不展示，并把 `injection.strategy` 设为 `不注入`。
7. Alert、字段说明、原名标注不是 onboarding 组件；必须走项目既有组件、设计系统组件或明确占位。
8. 管控端和用户端组件调用差异以设计规范为准，不用 `endpoint` 之外的业务侧样式分支改造 onboarding 外观。
9. `New Tag` 只负责轻量标新，不负责解释规则、权限、计费、逻辑变化。
10. 页面级规则、权限、计费、逻辑变化优先 `页面 Alert`；不要降级成 `GuidePointBubble`。
11. 同一认知问题默认只保留一个主组件和一个定位组件；不要生成三个组件重复说同一件事。
12. 用户端日常提醒默认先看 `GuideModuleFloat`，管控端日常提醒默认先看 `GuideAdminNotify`；两者不能互相替代。
13. 用户端只是新增下载、帮助、浏览、展示入口时，不得映射为 `GuideAdminNotify`。
14. 已导出的 onboarding 组件不得标记为本地占位；只有页面 Alert、字段说明、原名标注、侧边栏说明详情等非 onboarding 能力可在缺组件时占位。
15. `GuidePointBubble` 选纯文本时必须显式传 `contentVariant="text-only"`，因为源码默认值是 `text-button`。
16. `GuideUpdateBar` 依赖页面存在 `header.sticky` 或 `main` 作为 Portal 插入点；该检查由交接后的开发流程负责，本 skill 不补容器、不植入组件，也不得声称已挂载成功。
17. `GuideGlobalModal` 当前始终可通过关闭按钮、蒙版和 Esc 关闭，只能表达强提醒/知情，不足以证明法律或合规“明确同意”；必须同意的场景应改用业务确认组件或先扩展组件并重新审核。
18. `GuideGlobalModal` 为 `z-9999`，而 `GuideModuleFloat` 源码为 `z-10000`；编排层必须保证两者不同时打开，GlobalModal 展示期间暂停浮窗队列。
19. 每个通知单元使用独立持久化 key；同一更新中的多个 `point-bubble`、双端提示或详情组件不得共用一个 key。
20. `GuideAdminNotify` 只由本 skill 识别需求并转交文案生成与审核；不得将方案确认解释为开发授权，也不得修改该组件或其详情条目。
21. `GuideUpdateBar` 只由本 skill 识别需求并转交公告文案生成、审核与后续开发决策；不得将方案确认解释为开发授权，也不得修改该组件或其关联详情条目。
22. 每个可执行组件必须在 Product Review Plan 中单独展示并确认 `duration`；默认值只是建议，`required` 缺少天数时不得进入执行，Campaign 登记必须携带 `duration_confirmed=true`。

## 组合收敛规则

当多个规则同时命中时，按下面方式收敛：

- `GuideUpdateBar` 解决“全局知道变化了”
- `GuideNavBubble` / `GuidePointBubble` 解决“具体位置在哪”
- `页面 Alert` 解决“为什么规则或结果变了”
- `GuideModuleFloat` / `GuideAdminNotify` 解决“日常汇总告知”

如果一个组件已经完整解决当前认知问题，就不要再追加同类组件。例如：

- 已有 `页面 Alert` 解释规则变化时，不再追加 `GuidePointBubble` 重复同一段规则
- 已有 `GuideAdminNotify` 做跨端产品动态时，不再额外放一条同义的 `GuideModuleFloat`
- 已有 `New Tag` + `GuidePointBubble` 指向同一新增入口时，不再补一条内容重复的 `GuideModuleFloat`
