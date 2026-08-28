# Update Awareness Skill 更新日志

本文件用于记录 `update-awareness-skill` 的结构、规则和脚本变更，避免后续更新后无法快速判断：

- 这次改了什么
- 为什么要改
- 哪些文件受影响
- 是否影响 `Notification Plan` 输出或接入方式

## 2026-07-22

### 调整

- 将 `GuideUpdateBar` 调整为与 `GuideAdminNotify` 相同的交接型组件：仅识别管控端 Banner / 公告条需求并形成待审核 brief，不由本 skill 决策、开发、植入或登记。
- 为 `GuideUpdateBar` 增加硬执行闸门：设置 `executable=false`，不生成可开发终稿、props、持久化、埋点、设计接入或注入目标；关联的 `GuideChangelogDrawer` 条目一并排除。
- Campaign 登记脚本禁止登记 `GuideUpdateBar`；纯交接方案使用 `handoff_required`，混合方案只允许执行其他组件。
- 以 `references/component_contract.md` 的适用端为准收敛端侧匹配：`GuideNavBubble`、`GuideChangelogDrawer`、`ProductUpdatesDrawer` 仅在管控端展示。
- 用户端导航提示条或页面入口气泡不再映射为 `GuideNavBubble`，改用目标附近的 `GuidePointBubble`；用户端侧边栏详情改用项目已有抽屉或最小 `SidebarDetailAdapter` 占位。
- 用户端直接请求管控端产品动态卡片或 `ProductUpdatesDrawer` 时，计划生成器现在明确拒绝并提示更正端类型。
- 区分 `ProductUpdatesDrawer` 的两种开发边界：独立的管控端详情抽屉允许开发、登记并默认长期保留；与 `GuideAdminNotify` 关联的条目继续标记为 `handoff_required`，不由本 skill 开发或登记。
- 将每个可执行组件的存在时长 `duration` 升级为 Product Review Plan 的独立必审字段；默认值只作为建议，必须随 `plan_id + plan_revision` 一并由用户确认。
- 计划生成脚本新增可重复的 `--component-duration 组件类型或组件名=天数|permanent`；没有可靠默认值的组件在补充具体天数前阻止方案确认。
- Campaign 登记新增 `duration_confirmed=true` 硬校验，缺少时拒绝写入，避免登记阶段使用默认值绕过产品确认。
- 同步更新 `SKILL.md`、组件契约、表面规则、计划结构、Campaign 规范、使用说明、默认时长配置和 Agent 默认提示词。

### 校验

- Skill 结构校验和 Python 脚本语法检查通过。
- 已验证纯 `GuideUpdateBar` 方案只交接且无注入目标，混合方案只执行其他组件，Campaign 脚本拒绝登记 `GuideUpdateBar`。
- 已验证用户端入口映射为 `GuidePointBubble`、用户端详情使用项目抽屉占位、用户端管控抽屉请求被拒绝。
- 已验证独立 `ProductUpdatesDrawer` 可以登记 Campaign，与 `GuideAdminNotify` 关联时不可执行且不生成注入目标。
- 已验证默认 duration 建议、required 时长缺失拦截、用户指定天数和 Campaign 时长确认闸门。

### 影响范围

- 影响组件匹配、端侧校验、Product Review Plan / Implementation Plan、执行闸门和 Campaign 登记。
- 不修改产品业务代码；只调整 `update-awareness-skill` 的规则、参考文件和脚本。

## 2026-07-20

### 调整

- Campaign 升级为 `schema_version: 3`，在批次级新增必填 `launched_on` 字段。
- `launched_on` 默认记录方案已确认、组件已植入并通过校验后登记 Campaign 的当天日期，可通过 `--launched-on` 指定实际发布日期。
- 到期日改为直接按 `launched_on + duration.days` 计算，不再从 Git 历史推导上线日期。
- 旧版空 Campaign 可由脚本自动升级；包含记录的旧版文件必须先为每条 Campaign 补充真实 `launched_on`，禁止静默使用迁移当天日期。

### 影响范围

- 影响 Campaign schema、登记脚本、下线日期计算方式和使用说明。

## 2026-07-17

### 调整

- Campaign 默认路径从目标产品仓库根目录的 `campaign.yaml` 迁移为 `.clawpro/campaign.yaml`；脚本在默认目录不存在时自动创建 `.clawpro/`。
- 在 Campaign 文件顶部加入 AI 合并保护备注，要求冲突处理时按 `campaign_id` 和 `component_id` 合并，禁止覆盖其他产品开发产生的组件记录；新建文件默认携带该备注。
- Campaign 升级为 `schema_version: 2`：同一上线批次使用一个 `campaign_id`，并通过 `components` 同时记录一个或多个实际组件。
- 每个组件独立记录 `component_id`、实际文案、挂载位置、仓库相对代码路径、`duration` 和下线状态，支持 AI 提醒产品经理并在确认后创建下线 MR。
- 上线日期不再写入 Campaign；AI 以 Campaign 记录首次进入 `main` 的日期作为上线日，再按各组件的 `duration.days` 计算到期日。
- 移除 `duration_source`、`review_status`、`needs_confirmation` 和 `recorded_at`；组件默认时长统一从 `references/component_duration_defaults.json` 读取。
- 每次 Campaign 登记成功后强制展示实际写入的 `current_user_id`，并提醒用户检查它是否为负责组件下线的产品经理。
- 使用说明中的调用示例统一改为 `$update-awareness`，并补充目标仓库的 Git 主线前提。
- 明确未提供 `current_user_id` 时脚本回退的是本机操作系统用户名，不保证等同于产品经理平台账号。
- 修正更新记录链接、补齐组件示例，并将 Agent 接入说明调整为按工作阶段加载引用文件。
- 按当前 Skill 规范将 `SKILL.md` frontmatter 精简为 `name` 和 `description`。

### 影响范围

- 影响 `campaign.yaml` 结构、登记脚本参数、Campaign 规范和使用说明。
- 当前空 Campaign 已升级为第 2 版并迁移到 `.clawpro/campaign.yaml`；包含旧记录的文件必须显式迁移。

## 2026-07-16

### 调整

- 简化 Campaign 负责人信息：移除 `current_owner` 的角色、姓名和来源结构，改为单字段 `current_user_id`；未明确提供时使用当前系统用户 ID。
- 将 Campaign 的 `recorded_at` 从精确到秒的时间改为 `YYYY-MM-DD` 登记日期。

### 影响范围

- 影响 `campaign.yaml` 字段结构、登记脚本参数和相关使用说明。
- 不影响产品代码和现有空 Campaign 文件。

## 2026-07-15

### 调整

- 将用户端 `GuideModuleFloat` 改为全局布局组件：跨页面可展示、全局唯一、路由切换不重复曝光，并增加登录页、首次 onboarding、沉浸任务和强提醒冲突等限制。
- 提高 `GuideModuleFloat` 使用门槛：仅高/重大且必须跨页面触达的更新允许使用；用户端缺少 UpdateBar 时不再自动降级为重浮窗，优先使用 NavBubble、PointBubble、New Tag 或不展示。
- Product Review Plan 新增条件化 `visual_asset`：`GuideModuleFloat`、`GuideGlobalModal` 必须提示联系设计出图，带图 NavBubble/PointBubble 也需提示；素材未交付前禁止使用源码占位画面上线。
- 从头对照 `onboarding/` 的真实导出、props、默认值和渲染行为：修正 `GuideModuleFloat` 为用户端组件，区分浮层/抽屉、仅 `open` 的 `GuideUpdateBar` 与无 `open/onClose` 的 `GuideNewTag`。
- 在 Implementation Plan 中新增并落实 `surface_endpoint` / `content_endpoint`、真实 props、通知单元 id、独立持久化 key 和同类组件渲染编排；跨端 `GuideAdminNotify` 使用 `item.endpoint=tenant` 表示用户端内容。
- 修正 `GuidePointBubble` 源码默认是 `text-button` 的风险，纯文本方案现在显式生成 `contentVariant=text-only`。
- 修正 `GuideHighlightBubble` 生命周期为一次展示；多个同类提示不再共用持久化 key。
- 补充 `GuideAdminNotify variant=stacked` 原始 items 聚合、`GuideModuleFloat sources` 合并、聚合文案长度和单 source 多 items 丢页风险。
- 增加 `GuideUpdateBar` Portal 挂载前提、GlobalModal 与 z-10000 ModuleFloat 互斥、GlobalModal 不能充当不可绕过合规同意控件等实现边界。
- 明确 `GuideAdminNotify`、`GuideNewTag`、`ProductUpdatesDrawer` 不进入 `GuideFlow.steps`，并排除生产环境使用 `GuideOnboardingModal`、`OnboardingSimToggle`、`OnboardingDemoPanel`。
- 计划生成脚本会拒绝空组件和未知组件名，避免静默降级成错误的 `GuidePointBubble`；同时移除会让几乎所有用户端“更新”误判为管理员影响的泛化关键词。
- 所有 onboarding 正式导入仍保持为 `@/components/onboarding`；参考文件夹未修改，也不作为导入路径。
- 将跨端告知改为独立判断维度：所有用户端变化都判断管理员是否需要管理、治理、支持、解释或排查，不再只依赖跨端层、入口或 New Tag；补充云端 Agent 对话等重要能力示例。
- 调整多场景处理：保留所有独立认知问题，按目标用户、问题、页面和 Trigger 拆分通知单元；优先级只决定顺序和避让，不再删除低优先级但必要的提示。
- 计划生成脚本支持重复传入附加场景，并将全部场景写入内部实现方案。
- 根据真实组件定义校正 `GuideNewTag` 的 `new / coming-soon / custom` 变体、`GuideAdminNotify` 的左侧导航底部挂载位置、`ProductUpdatesDrawer` 条目字段，并明确排除仅用于首次访问的 `GuideOnboardingModal`。
- 保持 onboarding 组件正式导入路径为 `@/components/onboarding`；新增源码目录仅用于规则核对。
- 将产品审核方案的目标用户枚举收敛为 `管理员` 和 `普通用户`；运营等后台角色归入管理员。用户端更新有明确管理影响时，顶层目标用户同时包含两类，并按组件分别标注。
- 产品审核方案中的组件改为明确展示中文类型和真实组件名，挂载信息拆分为页面、目标元素和相对方位。
- 新增组件内容槽位契约；先选真实组件和变体，再按支持的槽位生成文案，禁止给所有组件统一添加“去看看 / 知道了”。
- 计划生成脚本改为逐组件生成内容：公告条、New Tag、纯文本气泡、步骤气泡、产品动态卡片和抽屉分别输出不同内容结构。
- 将原先冗长的 `Notification Plan` 拆为面向产品经理的 `Product Review Plan` 和 AI 内部使用的 `Implementation Plan`。
- 默认只展示更新内容、目标用户、组件、解决问题、展示位置、Trigger、文案和展示策略；技术字段仅在用户要求时展开。
- 更新计划生成脚本：默认输出精简审核方案，使用 `--include-implementation` 时附带完整技术方案。
- 增加强制人工确认闸门，把工作流拆为“方案阶段”和“执行阶段”。
- 第一次调用只输出可审核方案，不再自动修改产品代码；预先授权不能替代方案展示后的确认。
- 方案发生实质变化时递增 `plan_revision` 并重新确认；执行中发现偏差时停止并返回修订方案。
- `Notification Plan` 新增 `plan_revision` 和 `approval` 字段，脚本默认生成 `awaiting_confirmation` 状态。
- 更新宿主默认提示词和使用说明，使调用入口明确体现“先确认、后修改”。

### 影响范围

- 影响 skill 执行流程、计划结构、初稿生成脚本和宿主接入提示。
- 未经后置人工确认，任何调用都不得修改产品代码。

## 2026-06-30

### 新增

- 新增本文件，作为 skill 内部统一更新日志。

### 调整

- 对齐 onboarding 组件定义，补充并收敛了 `GuideAdminNotify`、`GuideModuleFloat`、`GuideNewTag`、`ProductUpdatesDrawer` 等组件边界。
- 更新 `scripts/create_notification_plan.py` 的默认组件映射：
  - 管控端 `日常更新提示` 默认映射到 `GuideAdminNotify`
  - 用户端 `日常更新提示` 默认映射到 `GuideModuleFloat`
  - 用户端抽象命中 `导航提示条` 但缺少等价条幅时，默认降级为 `GuideModuleFloat`
- 更新 `Notification Plan` 默认行为：
  - 持久化 key 改为 `onboarding.{type_id}.{update_id}.dismissed`
  - 默认埋点改为 `onboarding_impression` / `onboarding_click` / `onboarding_dismiss`
  - `New Tag` 补充 14 天默认下线时间和首次点击移除字段
- 对齐文案限制：
  - 普通 onboarding 文案：标题 `<=14`、正文 `<=60`、CTA `<=6`
  - `GuideAdminNotify`：标题 `<=16`、正文 `<=30`、按钮 `<=6`

### 文档

- 更新 `SKILL.md`、`使用说明.md`、`references/component_contract.md`、`references/copy_guidelines.md`、`references/notification_plan_schema.md`，使其与当前组件契约和脚本输出一致。
- 将 `使用说明.md` 中对 `agents/openai.yaml` 的描述从 `Codex 展示名称` 调整为宿主接入配置，避免把 skill 锁死在单一 agent。

### 影响范围

- 影响 skill 文档和 `Notification Plan` 初稿生成逻辑。
- 不影响 `references/onboarding_component_spec.md`。
- 不修改 skill 目录外的任何内容。

## 维护规则

- 每次修改 skill 的规则、脚本、输出结构或接入说明时，都应追加一条日志。
- 如果变更会影响 `Notification Plan` 字段、默认组件映射、文案限制或接入约束，必须在日志中显式写出。
- 如果只是错别字或纯表述调整，可合并到最近一条日志的“文档”部分，但不要省略记录。
