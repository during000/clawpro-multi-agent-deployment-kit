---
name: update-awareness
description: 当管控端或用户端 Demo / 产品功能开发完成，需要识别版本更新内容、匹配提醒组件、生成产品经理可审核的更新提示方案或植入更新提示时使用。默认只展示精简 Product Review Plan，技术字段保留在内部 Implementation Plan；必须先等待人工确认，再修改产品代码。GuideModuleFloat 必须先提醒联系设计团队出图；GuideAdminNotify 与 GuideUpdateBar 只做需求识别及文案审核交接，不由本 skill 决策或开发。
---

# 版本更新感知 Skill

当开发者声明“Demo 功能已开发完成”“版本更新已完成”“需要自动加更新提醒”时，使用本 skill。

本 skill 面向版本更新感知，不做泛化营销弹窗，也不允许脱离既有引导组件体系另写一套浮层。

## 强制人工确认闸门

把一次完整调用拆成两个独立阶段，禁止在同一轮中从方案直接推进到代码修改：

1. `方案阶段`：只分析变更、选择组件、生成文案与精简的 `Product Review Plan`，然后停止并请求用户确认。不得修改业务代码、配置、样式、组件或测试文件。
2. `执行阶段`：仅当用户已经看过当前方案，并在后续消息中明确表示“确认执行”“按此方案修改”或同等含义后，才允许修改代码。

即使用户第一次调用时说“直接做”“自动植入”“不用问我”或同时要求生成方案并开发，也不得跳过方案后的确认。第一次调用中的预先授权不算确认；确认必须发生在完整方案展示之后。

以下情况不视为确认：用户只补充信息、询问原因、要求解释、表达倾向，或只确认其中一个字段。用户要求调整文案、组件、挂载点、受众、触发条件、生命周期或埋点时，更新方案修订号，再次停止等待确认。

确认只覆盖当前 `plan_id + plan_revision` 中展示给产品经理的产品决策。执行时发现证据与方案不一致，或需要改变目标用户、组件、解决问题、展示位置、Trigger、文案、展示策略或视觉素材要求时，停止修改、输出修订方案并重新确认。纯技术实现细节可在已确认范围内处理。

- 管控端：文案可以更专业，允许使用管理员熟悉的业务术语、配置术语、规则术语。
- 用户端：文案必须更易懂，优先描述用户能做什么、会看到什么，避免后台术语和实现细节。

## 特殊组件前置闸门

以下规则优先于通用的“确认后执行”流程：

- 认为需要 `GuideModuleFloat` 时，必须在方案中主动提醒用户联系设计团队进行图片设计，写明图片规格、交付状态和“设计素材交付并确认前不可上线”。即使产品方案已确认，只要设计图片尚未交付并审核通过，就停止执行 `GuideModuleFloat` 植入，不得使用源码占位图替代。
- 认为需要 `GuideAdminNotify` 时，只识别“存在管控端产品动态告知需求”，提醒用户先进行卡片文案生成与审核。不得把自动生成的文案视为可开发终稿，也不得直接开发、植入或修改 `GuideAdminNotify`、`ProductUpdatesDrawer` 条目及相关配置。
- 认为需要 `GuideUpdateBar` 时，只识别“存在管控端 Banner/公告条告知需求”，提醒用户先进行公告文案生成与审核。不得把自动生成的文案视为可开发终稿，也不得直接开发、植入或修改 `GuideUpdateBar`、其关联的 `GuideChangelogDrawer` 条目及相关配置。
- `GuideAdminNotify` 是否开发、如何开发以及最终文案，不由本 skill 决策。Product Review Plan 必须把该项标记为 `handoff_required`，说明“需先完成文案生成与审核，再由对应负责人决定后续开发”；用户对整份方案的确认不构成对该组件的开发授权。
- 混合方案中可以在后续确认后执行其他组件，但必须把 `GuideAdminNotify` 及其详情承接排除在修改范围外。
- `GuideUpdateBar` 是否开发、如何开发以及最终文案，同样不由本 skill 决策；方案确认不构成开发授权。
- 在任何执行动作前建立可执行组件集合并硬校验：排除 `GuideAdminNotify`、`GuideUpdateBar` 及其各自关联的详情条目；如果集合为空，立即拒绝执行并只返回交接提醒。不得因为用户再次要求“直接开发”而解除该拦截。
- 拒绝把 `GuideAdminNotify` 或 `GuideUpdateBar` 指定为 `design_component`、注入目标或技术实现对象；混合方案中的目标文件和设计组件必须属于剩余可执行组件。

## 目标

从“功能开发完成”自动推进到“页面中出现规范的更新提示”：

1. 识别本次产品变更。
2. 判断这是管控端还是用户端。
3. 判断变更属于结构层、元素层、逻辑层、系统层或跨端层。
4. 匹配对应的更新提醒组件组合和真实 React 组件。
5. 按端侧语气生成中文提醒文案；`GuideAdminNotify` 与 `GuideUpdateBar` 仅生成待文案处理的 brief，不生成可开发终稿。
6. 输出带修订号和确认状态的 `Product Review Plan` 供产品经理审核。
7. 人工明确确认后，按既有引导组件契约和设计规范植入可执行组件；`GuideAdminNotify` 与 `GuideUpdateBar` 始终转交其他流程。
8. 方案确认且组件实际开发并校验通过后，把同一上线批次的上线日期、全部组件、最终文案、挂载位置、代码路径、存在时长和当前用户 ID 写入目标产品仓库根目录下的 `.clawpro/campaign.yaml`。

## 可接收输入

尽量使用已有上下文，不要因为信息不完整就阻塞。

- 产品或 Demo 名称
- 端类型：`管控端` 或 `用户端`
- 目标页面、路由、组件路径
- 开发者完成说明、PR 摘要、提交差异、变更文件
- 影响用户：`管理员` 或 `普通用户`；运营、审核、配置等后台角色统一归入管理员
- `@/components/onboarding` 的真实引导组件规范
- 管控端 / 用户端设计规范、组件标准或 Portable fallback 规范
- 仓库里已有公告、气泡、New Tag、Alert、引导组件的使用方式
- 版本号、发布日期、灰度条件、功能开关
- 当前用户 ID；建议显式提供能接收下线提醒的产品经理 ID。未提供时脚本回退到本机操作系统用户名，写入后必须检查它是否对应正确的产品经理账号
- 组件存在时长；未提供时按组件默认值提出审核建议，必须由用户在 Product Review Plan 中确认后才能执行和登记

信息不足时，可从文件名、路由名、页面文案、组件名称和代码差异推断，并在计划里标注“低置信度假设”。

## 工作流程

### 1. 识别产品变更

优先读取目标页面和相关变更文件。关注这些信号：

- 新增页面、子页面、导航项、Tab、侧边栏入口
- 页面布局、模块顺序、分组方式、入口层级变化
- 新增按钮、下拉项、表格列、展示字段、筛选项、排序项
- 名称、文案、状态枚举、规则说明变化
- 底层逻辑、业务规则、计费配额、权限、账号、合规变化
- C 端与管控端之间的联动变化

如果存在 git 差异，优先分析差异；否则基于当前页面结构和代码语义判断。

### 2. 判断端类型

在生成文案前，必须先判断目标页面属于：

- `管控端`：面向管理员、运营、配置人员、审核人员、超级管理员等
- `用户端`：面向普通终端用户、业务使用者、消费者或非后台操作者

判断信号：

- 路由、页面名、导航名是否包含管理、配置、后台、控制台、Admin、Console 等语义
- 当前页面主要任务是“配置规则、分配权限、查看后台状态”，还是“直接使用产品能力”
- 用户描述中是否明确提到管理员、租户管理员、普通用户、C 端用户

如果无法确定，在内部 `Implementation Plan` 标记为待确认，并在产品审核方案中明确目标用户尚待确认；不要直接复用管控端文案。

### 3. 输出变更摘要

先形成简短中文摘要：

```json
{
  "端类型": "管控端",
  "变更摘要": "新增了企业插件库的分发状态筛选能力",
  "变更层级": "元素层",
  "变更场景": "2.3 新增筛选/排序/分组选项",
  "用户价值": "管理员可以更快定位不同分发状态的插件",
  "证据": ["文件路径、路由、组件名或用户说明"],
  "置信度": "高 | 中 | 低"
}
```

### 4. 读取设计规范并确认组件边界

在匹配提醒组件前，先按端类型读取设计规范，明确哪些组件可直接调用、哪些必须走项目既有组件或占位适配器：

- 管控端 / Admin：若工作区存在 `clawpro-portable-design-skill/`，读取 `clawpro-portable-design-skill/SKILL.md` 与 `clawpro-portable-design-skill/references/components.md`。
- 用户端 / Tenant：若工作区存在 `clawpro-portable-design-skill/`，读取 `clawpro-portable-design-skill/references/tenant.md` 与 `clawpro-portable-design-skill/references/components.md`。
- 只处理 onboarding 组件细节时，仍以 `references/component_contract.md` 和 `references/onboarding_component_spec.md` 为准。

设计规范优先级：

1. 真实业务组件或设计系统组件能表达语义时，直接调用组件，不在页面里手写视觉样式。
2. `@/components/onboarding` 只承载全局浮层、气泡、New Tag、更新条、更新抽屉等引导能力。
3. 页面 Alert、字段规则说明、原名标注、普通按钮、卡片、表格、空态等必须优先使用项目既有组件或设计系统组件。
4. 只有当项目缺少对应组件、props 或视觉未接入时，才允许最小占位适配器，并在计划中标明缺口和替换目标。

### 5. 匹配更新场景

读取 `references/scenario_rules.md`，识别所有会产生独立用户认知问题的场景，不要强行压缩成一个主场景。

把命中的场景按“目标用户 + 认知问题 + 展示页面/目标元素 + Trigger”拆成通知单元：

- 任一维度不同，且都需要主动提示时，分别生成提示项。
- 多个场景面向同一用户、解决同一问题、挂在同一位置且同时触发时，可以合并为一个提示，文案需覆盖全部变化。
- 优先级只用于决定展示顺序、冲突避让和主次关系，不用于删除仍需提示的场景。
- 每个通知单元仍遵循“一个主组件 + 最多一个定位组件”，但一次更新可以包含多个通知单元。

展示冲突时按用户影响排序：

1. 系统层：账号、安全、权限、合规
2. 逻辑层：底层逻辑、规则、计费、配额
3. 结构层：页面、入口、导航、布局
4. 跨端层：C 端与管控端联动
5. 元素层：按钮、字段、筛选、文案、视觉细节

### 6. 匹配提醒组件

读取顺序必须是：

1. `references/surface_rules.md`
2. `references/component_contract.md`
3. `references/onboarding_component_spec.md`

默认使用组合式提醒，而不是只选一个组件。

选型时先写抽象组件类型，再映射到真实组件：

- `导航提示条` → 管控端识别为 `GuideUpdateBar` 需求并交接公告文案审核与开发决策；用户端优先使用目标附近的 `GuidePointBubble` / `GuideNewTag`，只有高影响且必须跨页面触达时才使用 `GuideModuleFloat`
- `New Tag` → `GuideNewTag`
- `页面入口气泡引导` → `GuideNavBubble`
- `功能入口附近气泡` → `GuidePointBubble`
- `重要操作变更气泡引导` / `常驻操作引导` → `GuideHighlightBubble`
- `强提醒弹窗` → `GuideGlobalModal`
- `日常更新提示` → 管控端用 `GuideAdminNotify`；用户端只有高影响、需要跨页面触达的更新才用 `GuideModuleFloat`
- `侧边栏说明详情` → `GuideChangelogDrawer` 或项目已有抽屉
- `页面 Alert` / `字段规则说明` / `原名标注` → 优先复用项目组件或设计系统组件，不可伪装成新的 onboarding 组件

原则：用户会找不到入口、误解规则或按旧路径失败时，必须提示；纯视觉优化、轻微流畅性优化通常不展示。

跨端同步判断是独立维度，不由主场景是否属于“跨端层”决定。识别每一项用户端变化后，都额外判断管理员是否需要知情：

- 需要通知管控端的情况：用户端变化会影响管理员的配置、下发、更新、管理、权限、审核、治理、安全、数据访问、状态解释、统计、故障排查、用户支持或咨询口径。
- 用户端新增重要能力，即使没有直接的管控端配置入口，只要管理员需要知道用户现在能做什么、可能产生什么数据或风险、如何支持和解释，也需要通知管理员。例如用户端支持使用云端 Agent 对话、模型调用、工具调用或外部连接。
- 是否通知管理员与结构层、元素层、逻辑层、系统层或跨端层分类相互独立；任一层级都可能产生管理员知情项。
- 不需要通知管控端的情况：只是新增一个用户可见入口或内容页，且不改变管理员需要管理或解释的对象。例如只新增“下载 skill 的地方”、帮助文档、展示页、静态资源列表、纯浏览或纯下载入口。
- 命中管理员管理或知情条件时，除用户端自身提示外，附加一条 `日常更新提示` → `GuideAdminNotify` 的需求识别项，作为 `5.1 C 端变化需告知管控端` 的文案生成与审核交接；不得将其作为本 skill 可直接开发的管控端产品动态卡片。
- 如果影响不确定，不要直接生成管控端卡片；在产品审核方案中标注“待确认管理员是否需要管理或知情”，并在内部技术占位符记录判断缺口。

组件输出必须同时包含：

- 中文组件类型，例如 `导航提示条`
- 真实组件名，例如 `GuideUpdateBar`
- 稳定类型标识，例如 `update-bar`
- 挂载目标，例如路由、选择器、目标元素名称
- 实例范围，例如 `首个实例 | 指定实例 | 全局唯一`
- 设计接入状态，例如 `已接入 | 占位 | 待确认`
- 组件来源，例如 `@/components/onboarding | 项目既有组件 | 设计系统组件 | 本地占位适配器`

面向产品经理的方案至少展示中文组件类型、真实组件名，以及分开的挂载页面、目标元素和相对方位；不得只写一个含糊的路径字符串。

若仓库存在 `@/components/onboarding`，优先直接复用统一导出入口；禁止业务页面手写 `fixed` + `absolute` 重新拼装同类浮层。只有 `页面 Alert`、`字段规则说明`、`原名标注` 等非 onboarding 能力允许占位。

端侧组件边界必须与 onboarding 导出一致：

- `GuideAdminNotify` 是管控端产品动态卡片，`type_id` 仍写 `module-float`，点击后优先承接 `ProductUpdatesDrawer`；本 skill 仅识别需求并发起文案生成与审核交接，不决策或执行该组件开发。
- `GuideModuleFloat` 是用户端全局产品动态浮窗，挂在登录后的用户端全局布局，不绑定单一业务页面。它固定包含 16:9 配图区，打扰度和视觉重量都较高：仅用于高影响、普通用户必须跨页面知道、轻量入口标记或局部气泡容易漏看的更新。低/中影响更新优先 `GuideNewTag`、`GuidePointBubble` 或不展示。使用时必须在 Product Review Plan 主动提醒用户“请联系设计团队进行图片设计”，图片未交付前不得用源码占位图完成生产植入。同屏多个浮窗来源用一个实例的 `sources` 合并，不要渲染多个实例。
- `GuideUpdateBar` 是管控端导航下方强提醒 Banner/公告条；本 skill 只识别需求并发起公告文案生成、审核与后续开发决策交接，不执行植入。用户端没有等价条幅时不要自动升级为 `GuideModuleFloat`，先选择更轻的目标附近气泡、New Tag 或不展示。
- `GuideNewTag` 使用 `type_id=new_tag`，必须写明 14 天建议下线、最长 30 天或首次点击移除。
- `ProductUpdatesDrawer` 是仅管控端使用的产品动态详情承接组件，从 `@/components/onboarding` 导出，但不是单步引导气泡。独立详情抽屉可以作为可执行组件开发；与 `GuideAdminNotify` 关联的条目必须随该卡片交接，不进入本 skill 的执行或 Campaign 登记范围。
- `GuideAdminNotify`、`GuideNewTag`、`ProductUpdatesDrawer` 不进入 `GuideFlow.steps`；`GuideOnboardingModal`、`OnboardingSimToggle`、`OnboardingDemoPanel` 不用于生产版本更新提示。
- `GuideAdminNotify` 始终展示在管控端，但 item 的 `endpoint` 表示内容所属端。用户端更新同步给管理员时，技术方案写 `surface_endpoint=管控端`、`content_endpoint=用户端`、`item.endpoint=tenant`。
- 组件适用端必须以 `references/component_contract.md` 的真实组件清单为准：`GuideNavBubble`、`GuideChangelogDrawer`、`ProductUpdatesDrawer` 仅在管控端展示。用户端入口提示改用 `GuidePointBubble` / `GuideNewTag`；用户端详情说明复用项目已有抽屉或最小占位适配器。

若目标是列表卡片、表格行、宫格项等“同页重复出现的同类组件”，默认只对首个实例展示一次指引，不要给每个重复实例都挂同一条 onboarding。只有当不同实例承载不同认知任务时，才允许指定某个非首项实例，并在计划中写明原因。

### 7. 生成产品经理审核方案

读取 `references/notification_plan_schema.md`。先在内部形成完整 `Implementation Plan`，再默认只向用户输出精简的 `Product Review Plan`。

`Product Review Plan` 只包含产品经理需要判断的内容：

- 更新内容
- 目标用户
- 选择的组件，包括中文类型和真实组件名
- 组件解决的问题
- 挂载信息：页面或路由、目标元素、相对方位；不展示 selector
- Trigger
- 组件内容：内容变体，以及该组件真实支持的文案槽位
- 展示策略：次数和关闭方式
- 存在时长 `duration`：固定天数或长期保留；默认值只作为建议，必须显式展示并由用户确认
- 视觉素材：当组件或内容变体存在配图区时，明确是否需要联系设计团队出图、图片/视频规格和“素材交付前不可上线”
- `plan_id`、`plan_revision` 和确认状态

若组件为 `GuideAdminNotify` 或 `GuideUpdateBar`，该项不得提供可直接执行的终稿文案或开发确认入口；改为展示 `status=handoff_required`、待生成/审核的文案槽位、负责方待确认，以及“本 skill 不决策开发”的下一步提醒。

目标用户只使用 `管理员` 和 `普通用户` 两个值。用户端更新存在明确的管理员管理或知情影响时，方案同时包含普通用户项和管理员项；每个组件必须标明自己的实际展示对象。

默认不要展示 `type_id`、selector、instance scope、import path、props、持久化 key、埋点参数、注入策略、设计规范路径、证据数组或技术占位符。仅当用户明确要求“查看技术方案/完整 JSON”，或执行阶段需要交付开发信息时，才输出 `Implementation Plan`。

`Implementation Plan` 必须继续保留可执行组件的完整技术字段，作为 AI 执行依据。`GuideAdminNotify` 与 `GuideUpdateBar` 只保留需求识别和交接字段，不生成执行参数。技术细节可以在确认后补全，只要不改变已确认的目标用户、组件、解决问题、展示位置、Trigger、文案、展示策略、存在时长或视觉素材要求；若改变其中任何一项，必须递增修订号并重新确认。

纯交接方案（只含 `GuideAdminNotify`、`GuideUpdateBar` 或二者）必须使用顶层 `status=handoff_required`，并把 `behavior`、`analytics`、`design_component` 标记为 `not_applicable`；不得出现持久化 key、埋点事件、设计导入或目标文件。混合方案的这些技术字段只能从其他可执行组件生成。

生成组件内容前，必须读取 `references/component_contract.md` 的“组件内容槽位”和 `references/onboarding_component_spec.md` 对应组件详解。先确定真实组件及变体，再生成它支持的文案：

- 不得把统一的标题、正文、主按钮、次按钮复制给所有组件。
- `GuideUpdateBar` 只形成公告正文、可选版本和可选详情入口的待生成/待审核 brief，不生成可开发终稿。
- `GuideNewTag` 只生成短标签。
- `GuidePointBubble` 默认使用纯文本变体，只生成标题和说明；只有确实需要操作时才选择带按钮变体。
- 纯文本 `GuidePointBubble` 必须显式传 `contentVariant="text-only"`，不能依赖源码默认值（源码默认是 `text-button`）。
- `GuideHighlightBubble` 生成步骤标题和说明，步骤导航交给组件。
- 抽屉生成条目内容，不生成“知道了”。
- 所有可选按钮都必须同时满足“组件支持该槽位”和“用户确实需要该动作”，否则省略。
- `GuideModuleFloat` 与 `GuideGlobalModal` 固定包含媒体展示区，方案必须增加视觉素材要求。`GuideNavBubble.image`、`GuidePointBubble` 的 `text-image` 或带 `noticeImage` 的 `push-notice` 只在实际选择带图内容时增加该要求。
- 不得把组件内置占位画面当成正式素材。产品确认后若素材仍未提供，暂停执行并请求设计稿或已批准的图片/视频资源，不得先用占位图上线。

可使用 `scripts/create_notification_plan.py` 生成初始方案；脚本默认输出 `Product Review Plan`，只有传入 `--include-implementation` 才附带完整技术方案。

文案生成必须满足组件长度约束：

- 普通 onboarding 文案：标题 `≤14` 字、正文 `≤60` 字、CTA `≤6` 字。
- `GuideAdminNotify`：仅把标题、正文、按钮及长度要求作为文案生成与审核 brief；标题带 `管控端 |` 或 `用户端 |` 前缀且 `≤16` 字，正文对应 `desc`，建议 `≤30` 字，按钮对应 `btnText`，建议 `≤6` 字。不得由本 skill 产出并认定可开发终稿。
- 不要把影响范围、解释口径、查看详情等补充说明塞进 `GuideAdminNotify.title`；标题只保留端前缀和能力名。

### 8. 停止并等待人工确认

输出完整的产品审核方案后，以可直接回复的确认语句结束，例如：

`请审核以上方案及每个组件的存在时长。确认无误后回复“确认执行方案 {plan_id} v{plan_revision}”；如需调整，请直接指出文案、组件、挂载位置或 duration。`

每个可执行组件必须在 Product Review Plan 中单独展示 `duration`。`fixed_days` 展示建议天数，`permanent` 展示长期保留，`required` 在用户明确给出天数前保持“待确认”，不得接受整份方案确认。用户修改或补充 duration 时递增 `plan_revision` 并重新等待确认。交接组件的 duration 标记为“由交接流程确认”，不纳入本 skill 的执行授权。

此时必须结束当前轮次，不得继续编辑产品代码。把 `Product Review Plan.status` 和内部确认状态保持为 `awaiting_confirmation`。

收到后续消息时：

- 明确确认当前 `plan_id + plan_revision`：把状态视为 `approved`，进入执行阶段。
- 提出任何实质调整：更新方案，将 `plan_revision` 加一，状态重置为 `awaiting_confirmation`，再次请求确认。
- 表述含糊或无法确定确认的是哪一版方案：继续停留在方案阶段，请用户明确确认。

若方案包含 `GuideAdminNotify` 或 `GuideUpdateBar`，上述确认只适用于其他可执行组件；这些交接项始终保持 `handoff_required`，直到脱离本 skill 的文案生成、审核与开发决策流程完成。

不要声称系统已记录真实姓名或审批时间，除非用户或宿主明确提供这些信息。

### 9. 按已确认方案植入页面

先读取 `execution_guard` 并重新计算可执行组件集合。若待执行对象包含 `GuideAdminNotify`、`GuideUpdateBar` 或其各自关联的详情条目，拒绝该部分；若没有剩余可执行组件，停止本轮且不得修改任何产品文件。

执行前再次核对当前方案已由用户后续消息明确批准，并且批准对象与当前 `plan_id + plan_revision` 一致。只修改确认范围内的文件与行为。

编辑页面前先读取 `references/component_contract.md` 和 `references/onboarding_component_spec.md`。

如果仓库已有 `@/components/onboarding`：

- 从 `@/components/onboarding` 统一入口导入。
- 任何 onboarding 正式导入都固定为 `@/components/onboarding`，不得因参考源码位置、类型或 helper 未导出而深链。`GuideModuleFloat.sources` 可内联传值，不导入统一入口未导出的 `ModuleFloatSource` / `mergeModuleFloats`。
- 使用规范中已有 props 和变体，不自行复制视觉样式。
- 遵循现有状态存储、关闭逻辑、埋点、z-index 和 `endpoint` 规范。
- 持久化 key 优先走 `buildPersistenceKey(component, updateId)` 的 `onboarding.{component}.{updateId}.dismissed` 格式。
- 埋点优先走 `trackOnboarding`，事件名使用 `onboarding_impression` / `onboarding_click` / `onboarding_dismiss`。
- 每个通知单元使用独立持久化 key；同一更新中的双端提示或多个同类气泡不得共用 key。
- `GuideModuleFloat` 只在用户端全局布局渲染一个实例：登录后任一允许页面均可展示；路由切换不重复曝光；登录页、首次 onboarding、全屏/沉浸任务、GlobalModal 打开期间不展示；关闭或完成状态跨页面持久化。
- `GuideGlobalModal` 打开时暂停 `GuideModuleFloat` 和其他业务弹窗，避免源码层级冲突；不得把可关闭的 GlobalModal 当作不可绕过的法律同意控件。
- 无论用户如何确认，均不得通过本 skill 导入、渲染、修改或配置 `GuideAdminNotify`、`GuideUpdateBar` 及其各自关联的详情条目；只交付文案生成与审核提醒。

如果目标能力属于页面基础组件或业务组件：

- 管控端按钮、卡片、表格、空态、弹窗等遵循设计规范里的 Admin 组件调用规则。
- 用户端业务卡、按钮、顶部导航、Tab 等遵循 Tenant 组件调用规则。
- 不用 `div + className` 临时拼出已有组件能表达的视觉。
- 不用 `className` 覆盖 Button、Card、Tabs、onboarding 组件的核心颜色、圆角、阴影或层级。

如果目标能力不在 onboarding 组件体系内：

- 只添加最小占位配置或轻量适配层。
- 明确标注组件名、import、props、持久化、埋点为占位。
- 不临时发明完整设计系统。

### 10. 校验

根据项目能力执行最小必要校验：

- 类型检查、lint 或单测
- 本地页面渲染
- 必要时用浏览器截图确认提示出现位置、遮挡和关闭行为

确认提醒只在目标条件下展示，且符合对应组件的关闭、过期或下线策略。

### 11. 登记 Campaign

读取 `references/campaign_schema.md`。只有可执行组件已经实际植入并通过必要校验后，才使用 `scripts/record_campaign.py` 将其写入目标产品仓库根目录下的 `.clawpro/campaign.yaml`，不得根据 skill 文件夹位置推算目标路径。

路径解析优先级：

1. 用户或调用方明确提供的 `--campaign-file`。
2. 已确认目标产品仓库根目录对应的 `--project-root/.clawpro/campaign.yaml`。
3. 当前工作目录所属 Git 仓库根目录下的 `.clawpro/campaign.yaml`。
4. 从当前工作目录向上找到的首个既有 `.clawpro/campaign.yaml`。

仍无法确定时停止并请求提供 `--project-root` 或 `--campaign-file`，不得猜测。目标路径已明确但 `.clawpro/` 或文件不存在时，脚本创建目录并初始化标准空结构。

- 同一上线批次写成一条 `campaign_id` 记录，实际开发的一个或多个组件分别放入 `components`；每个组件必须有唯一 `component_id`。
- `launched_on` 记录方案已确认、组件已植入并通过必要校验后登记 Campaign 的日期，格式为 `YYYY-MM-DD`；默认使用登记当天的本地日期，实际发布日期不同时通过 `--launched-on` 明确指定。
- 组件名称、文案、挂载位置和仓库相对 `code_paths` 必须来自已确认且实际落地的结果。
- 每个组件分别记录已在 Product Review Plan 中确认的 `duration`。默认值只用于形成审核建议，不得绕过确认直接登记；`fixed_days` 登记确认后的天数，`permanent` 登记长期保留，`required` 必须先取得明确天数。调用登记脚本时必须传入 `duration_confirmed=true`。
- `current_user_id` 只记录当前用户 ID；建议通过参数明确指定能接收下线提醒的产品经理 ID。未提供时脚本回退到本机操作系统用户名，该值不保证等同于产品经理平台账号，写入后必须检查。
- 每次 Campaign 写入成功后，必须在执行结果中输出 `Campaign 已登记：{campaign_id}` 和实际写入的 `launched_on`，紧接着展示实际写入的 `current_user_id`，并明确提醒用户检查它是否为负责本次组件下线的产品经理；即使该 ID 由用户明确提供，也不得省略检查提醒。
- AI 直接以 Campaign 的 `launched_on` 作为上线日期，再按各组件的 `duration.days` 计算到期日；不再从 Git 历史推导上线日期，也不记录 `recorded_at`、`launch_at` 或预计算的 `remove_on`。
- 方案阶段、未修改代码、校验失败或只完成文案交接时，不写入目标产品仓库的 `.clawpro/campaign.yaml`。
- `GuideAdminNotify`、`GuideUpdateBar` 及其各自关联的详情条目永远不得登记为本 skill 的开发结果。
- 写入前检查 `campaign_id` 和批次内 `component_id`；重复时拒绝追加。写入失败时不得声称登记完成。
- AI 只扫描 `main` 上 `state=active` 的固定天数组件，按 `launched_on + duration.days` 计算到期日；到期时先提醒 `current_user_id`，获得确认后才能根据 `code_paths` 创建下线 MR。MR 创建和合入后分别更新状态，不得重复操作。

## 输出格式

方案阶段使用中文，并只包含：

- `plan_id`、`plan_revision`；有可执行组件时使用 `status=awaiting_confirmation`，纯交接方案使用 `status=handoff_required`
- 可审核的更新内容、目标用户、中文组件类型、真实组件名、挂载页面、目标元素、相对方位、Trigger、组件适配文案、展示策略、每个可执行组件的 duration 和必要的视觉素材要求
- 明确说明“尚未修改产品代码”
- 一条绑定当前方案版本的确认语句

优先使用简短小标题或紧凑表格，不要默认输出大段 JSON。存在多个组件时，每个组件单独列出其问题、位置、Trigger 和文案。

执行阶段的回复包含已执行的 `plan_id + plan_revision`、已修改文件、校验结果、实际写入的 `.clawpro/campaign.yaml` 路径、`campaign_id`、`launched_on`、组件 ID 和仍保留的技术占位符。Campaign 成功写入时必须原样展示脚本返回的“Campaign 已登记”、上线日期和 `current_user_id` 检查提醒。不要在未确认时使用“已完成植入”“已修改”或“已登记”等表述。

保持简洁，避免解释与任务无关的实现细节。

## 参考文件

- `references/scenario_rules.md`：版本更新场景分类
- `references/surface_rules.md`：场景与提醒组件映射
- `references/copy_guidelines.md`：端侧区分后的中文文案规范
- `references/notification_plan_schema.md`：标准计划结构
- `references/component_contract.md`：真实引导组件映射与接入契约
- `references/onboarding_component_spec.md`：完整引导组件规范汇总
- `references/campaign_schema.md`：实际开发组件的 Campaign 登记结构
- `references/component_duration_defaults.json`：组件默认存在时长的唯一数据源
