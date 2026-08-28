# 产品审核方案与技术实现方案

更新提示方案分为两层：

1. `Product Review Plan`：默认展示给产品经理并作为人工确认对象。
2. `Implementation Plan`：保留给 AI 和开发者执行，不默认展示。

## Product Review Plan

默认输出紧凑的中文内容，不要直接倾倒完整 JSON。结构如下：

```json
{
  "plan_id": "update-awareness-{功能短名}-{版本或日期}",
  "plan_revision": 1,
  "status": "awaiting_confirmation | approved | handoff_required",
  "update": "用户会感知到的更新内容",
  "target_users": ["管理员"],
  "items": [
    {
      "target_users": ["管理员"],
      "component": {
        "type_zh": "功能入口附近气泡",
        "name": "GuidePointBubble"
      },
      "problem": "帮助用户发现新的筛选入口",
      "mount": {
        "page": "插件列表页 /admin/plugins",
        "target": "状态筛选器",
        "placement": "目标元素下方"
      },
      "trigger": "首次进入页面并看到筛选器时",
      "content": {
        "variant": "纯文本气泡",
        "slots": [
          {"name": "标题", "text": "按分发状态筛选"},
          {"name": "说明", "text": "现在可以快速查看不同分发状态的插件。"}
        ]
      },
      "display_strategy": "每位用户展示一次，可关闭",
      "duration": {
        "mode": "fixed_days",
        "days": 14,
        "label": "14 天",
        "source": "组件默认建议",
        "confirmation_required": true,
        "confirmation_status": "awaiting_user_confirmation"
      },
      "visual_asset": {
        "required": true,
        "status": "待设计",
        "request": "请联系设计团队进行图片设计",
        "spec": "16:9，推荐 672×376",
        "launch_gate": "素材交付并确认前不可上线"
      }
    }
  ],
  "confirmation_prompt": "确认无误后回复：确认执行方案 {plan_id} v{plan_revision}"
}
```

### 审核字段要求

- `update`：说明用户能感知到的变化，不写代码实现。
- `target_users`：只允许 `管理员` 和 `普通用户`。运营、审核、配置等后台角色统一归入管理员，不单独增加角色枚举。
- 用户端更新若同时影响管理员管理、治理、支持、解释或故障排查，顶层 `target_users` 同时写 `普通用户` 和 `管理员`，并在每个组件项中分别标明实际展示对象。这项判断独立于场景层级。
- `component.type_zh`：明确写中文组件类型。
- `component.name`：明确写真实组件名，但不展示 `type_id`。
- `problem`：说明该组件避免用户出现什么认知或操作问题，不写“用于提示更新”这类循环描述。
- `selection_basis`：仅重型组件展示。`GuideModuleFloat` 必须说明影响等级、跨页面触达必要性及为何 New Tag/局部气泡不足；`GuideGlobalModal` 必须说明强阻断必要性。
- `mount.page`：明确组件挂载在哪个页面或路由。
- `mount.target`：明确挂到哪个导航、按钮、筛选器、字段或页面区域。
- `mount.placement`：明确在目标元素的上、下、左、右、行内或全局固定区域；不展示 selector。
- `trigger`：使用产品可理解的触发条件，不写监听器或状态变量。
- `content`：只展示该组件真实支持的内容槽位。无按钮的组件不添加按钮字段；只有组件变体支持且方案确实需要操作时才生成操作文案。
- `display_strategy`：合并说明展示次数和关闭方式。
- `duration`：每个可执行组件单独展示存在时长。`fixed_days` 显示天数，`permanent` 显示长期保留，`required` 在用户提供天数前显示“待确认”。默认值只作为建议，用户必须随当前 `plan_id + plan_revision` 一并确认。
- `visual_asset`：仅在组件或所选变体实际需要图片/视频时展示。必须写素材状态、规格和上线前置条件；使用 `GuideModuleFloat`、`GuideGlobalModal` 时不得省略。
- `GuideModuleFloat` 的 `visual_asset.request` 必须明确提醒用户联系设计团队进行图片设计；素材交付并审核通过前，该项不可进入执行。
- `GuideAdminNotify` 必须增加 `workflow_gate.status=handoff_required`，并写明 `next_action=进行卡片文案生成与审核`、`development_decision=不由 update-awareness 决策`。内容槽位只作为待生成/待审核 brief，不得输出为可直接开发终稿。
- `GuideUpdateBar` 必须增加 `workflow_gate.status=handoff_required`，并写明 `next_action=进行公告文案生成与审核`、`development_decision=不由 update-awareness 决策`。内容槽位只作为待生成/待审核 brief，不得输出为可直接开发终稿。

### 组件内容槽位

严格按 `references/component_contract.md` 的“组件内容槽位”生成内容：

- `GuideUpdateBar` 只有公告正文、可选版本和可选详情入口，并且只形成待生成/待审核 brief，不生成可开发终稿。
- `GuideNewTag` 使用源码支持的变体：`new` 显示 `New`，`coming-soon` 显示 `即将开放`，仅 `custom` 使用自定义短标签；不生成标题、正文或按钮。
- `GuidePointBubble` 默认使用纯文本气泡，只生成标题和说明；只有明确需要用户操作时才切到文本按钮变体。
- `GuideHighlightBubble` 生成步骤标题和步骤说明，步骤导航由组件控制。
- `GuideChangelogDrawer` 生成版本、条目标题、描述、层级标签和日期；`ProductUpdatesDrawer` 生成更新类型、所属端、标题、描述、日期及可选体验链接；两者都不生成“知道了”。
- 其他组件也必须服从真实 props 和变体，不得为了统一结构填入不存在的内容。

产品经理确认只绑定以上产品字段。任一字段实质变化，或组件内容变体、槽位、duration、视觉素材要求发生变化时，将 `plan_revision` 加一，状态重置为 `awaiting_confirmation`。存在 `duration.mode=required` 且天数缺失的组件时，不得接受整份方案确认，必须先补充天数并输出新的修订版。

### 多场景要求

- 一次更新可以产生多个 `items`，不要求只保留一个主场景。
- 不同目标用户、认知问题、页面/目标元素或 Trigger 分别生成提示项。
- 同一用户、同一问题、同一位置且同时触发时可以合并。
- 场景优先级只决定顺序、排队和冲突避让，不删除其他必要提示。

## Implementation Plan

内部技术方案继续保留原有完整字段：

- 端类型、语气、全部命中场景、每个通知单元关联的场景编号、证据与置信度
- 组件 `component_name`、`type_id`、来源、导入路径和设计状态
- `surface_endpoint`（展示端）与 `content_endpoint`（内容所属端）；跨端管控卡片不得把两者混为一谈
- route、selector、placement、instance scope 和目标元素
- 与真实导出一致的 props、详情承接组件和条目 id；纯文本 PointBubble 显式写 `contentVariant=text-only`
- 视觉素材的 `required/status/type/spec/source_or_placeholder/launch_gate`；生产方案不得把源码占位画面记为可用素材
- 每个组件独立的已确认 duration、展示周期、持久化 key、关闭状态和 New Tag 下线策略；组件组合不得共用一套行为导致互相污染
- 曝光、点击、关闭埋点及参数
- 注入策略、目标文件、设计规范引用、假设和技术占位符

实现方案必须满足 `references/component_contract.md` 和 `references/onboarding_component_spec.md`。默认不向产品经理展示；用户明确要求完整 JSON 或技术方案时再展开。

`Implementation Plan` 中的 `GuideAdminNotify` 与 `GuideUpdateBar` 只能记录需求识别和交接信息，必须设置 `executable=false`，且不得生成注入文件、props、持久化 key 或埋点的执行指令。混合方案中的其他组件仍可独立执行。

- 纯交接方案：顶层 `status=handoff_required`；`behavior`、`analytics`、`design_component` 只允许包含 `status=not_applicable` 和原因；`injection.strategy=不注入，仅交接`、`target_file=null`。
- 混合方案：顶层生命周期、持久化、埋点、设计接入和注入字段只从 `executable=true` 的其他组件生成，不得引用 `GuideAdminNotify` 或 `GuideUpdateBar`。
- 只要存在 `GuideAdminNotify` 或 `GuideUpdateBar`，增加 `execution_guard.enforcement=hard_block`。执行前重新过滤组件；没有剩余可执行组件时必须拒绝执行。
- 拒绝 `design_component=GuideAdminNotify` 或 `design_component=GuideUpdateBar`，并确保混合方案的 `design_component` 与 `injection.target_file` 只对应其他可执行组件。

## 确认闸门

- 初次输出和每次修订后，有可执行组件时状态为 `awaiting_confirmation`；纯交接方案为 `handoff_required`。
- 第一次调用中的预先授权不算确认。
- 用户在后续消息中明确确认当前 `plan_id + plan_revision` 后才能修改产品代码。
- 执行前核对批准的 plan id 和 revision 与当前方案一致。
- 技术实现若改变目标用户、组件、解决问题、展示位置、Trigger、文案、展示策略、duration 或视觉素材要求，必须返回新的 `Product Review Plan` 重新确认。
- 方案确认不授权开发 `GuideAdminNotify` 或 `GuideUpdateBar`；二者始终转交文案生成、审核和后续开发决策流程。

## Campaign 登记

Campaign 不属于产品审核字段，只在组件实际开发并校验通过后写入目标产品仓库根目录下的 `.clawpro/campaign.yaml`。路径优先使用 `--campaign-file` 或 `--project-root`，不得依赖 skill 所在位置。

同一上线批次记录一个 `campaign_id`，其 `components` 可包含一个或多个实际开发组件。每个组件分别记录稳定 `component_id`、真实组件名、实际文案、挂载位置、仓库相对代码路径、存在时长和状态；批次记录 `launched_on` 和 `current_user_id`。

使用 `scripts/record_campaign.py` 写入并遵循 `references/campaign_schema.md`。组件默认时长只从 `references/component_duration_defaults.json` 读取并作为审核建议。方案确认、duration 已确认、组件植入并通过校验后登记 Campaign；登记输入必须包含 `duration_confirmed=true`。`launched_on` 默认使用登记当天的本地日期，AI 以该日期按 `duration` 计算到期日。不得在方案阶段预写，也不得登记 `GuideAdminNotify`、`GuideUpdateBar` 及其各自关联详情条目。

## 不展示场景

若判断无需提醒，仍输出精简审核方案：

- `component` 写 `不展示`
- `problem` 说明不打扰用户的原因
- `placement`、`trigger` 和文案写 `不适用`
- `display_strategy` 写 `不展示，不注入代码`
