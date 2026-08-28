# Campaign 下线记录

目标产品仓库根目录下的 `.clawpro/campaign.yaml` 供 AI 识别仍在线的更新提醒、提醒对应产品经理，并在确认后定位代码创建下线 MR。方案与每个组件的 duration 均确认后，组件完成植入并通过必要校验时登记 Campaign；`launched_on` 记录当天日期，`duration` 从该日期开始计算。

方案阶段、未修改代码、必要校验未通过或仅识别为 `GuideAdminNotify` / `GuideUpdateBar` 交接项时不写入。路径按 `--campaign-file`、`--project-root/.clawpro/campaign.yaml`、当前 Git 仓库根目录下的 `.clawpro/campaign.yaml`、向上查找既有 `.clawpro/campaign.yaml` 的顺序解析。

## 字段

```yaml
# AI 合并保护：Vibe Coding 或处理代码冲突时，必须按 campaign_id 和 component_id 合并并保留不同产品开发产生的组件记录；不得用任一分支的完整文件覆盖另一分支。
schema_version: 3
campaigns:
  - campaign_id: "local-agent-access-r2"
    launched_on: "2026-07-17"
    current_user_id: "pm-123"
    components:
      - component_id: "local-agent-access-point-bubble"
        component_name: "GuidePointBubble"
        copy:
          - slot: "标题"
            text: "接入本地Agent"
          - slot: "说明"
            text: "复制安装 Prompt，把本地工具接入企业管理。"
        mount:
          page: "/tenant/my-openclaw"
          target: "复制安装 Prompt 按钮"
          placement: "按钮附近"
        code_paths:
          - "client/src/pages/tenant/MyOpenClaw.tsx"
        duration:
          mode: "fixed_days"
          days: 14
        state: "active"
      - component_id: "local-agent-access-new-tag"
        component_name: "GuideNewTag"
        copy:
          - slot: "标签"
            text: "新"
        mount:
          page: "/tenant/my-openclaw"
          target: "本地 Agent 导航项"
          placement: "名称右侧"
        code_paths:
          - "client/src/layout/TenantNavigation.tsx"
        duration:
          mode: "fixed_days"
          days: 14
        state: "active"
```

字段用途：

- `schema_version`：文件结构版本，只在根节点记录一次。
- `campaign_id`：同一上线批次的稳定唯一 ID；重复写入时拒绝追加。
- `launched_on`：方案已确认、组件已植入并通过必要校验后登记 Campaign 的日期，格式为 `YYYY-MM-DD`。脚本默认使用登记当天的本地日期，可用 `--launched-on` 显式覆盖。
- `current_user_id`：应接收下线提醒的当前产品经理用户 ID；未提供时使用当前系统用户 ID。写入后必须展示实际值，并提醒用户检查是否为正确的产品经理 ID。
- `components`：本批次实际上线的组件列表；一次更新可以包含多个组件。
- `component_id`：批次内组件的稳定唯一 ID，用于分别计算和追踪下线。
- `component_name`：实际植入的真实组件名。
- `copy`、`mount`：实际文案和展示位置，用于向产品经理说明要下线的提示。
- `code_paths`：仓库相对代码路径，供 AI 定位并创建下线 MR；不得写绝对路径。
- `duration.mode=fixed_days` 与 `duration.days`：从 `launched_on` 当天开始计算的在线天数。
- `duration.mode=permanent`：长期保留，不进入自动下线提醒。
- `state`：`active`、`removal_mr_opened` 或 `removed`；防止 AI 重复提醒或重复创建 MR。
- `removal_mr_url`：仅在已创建下线 MR 后添加。
- `removed_on`：仅在下线 MR 合入 `main` 后添加，格式为 `YYYY-MM-DD`。

Campaign 不记录 `recorded_at`、`launch_at` 或预先计算的 `remove_on`。AI 直接读取 `launched_on`，并以该日期加上组件的 `duration.days` 得到应下线日期。`launched_on` 只能在方案已确认、组件已植入并通过必要校验后登记；如果实际发布日期与登记当天不同，必须通过 `--launched-on` 写入真实日期，不得猜测。

## 默认时长

默认时长的唯一数据源是 `references/component_duration_defaults.json`，脚本直接读取该文件：

- `fixed_days`：可以自动使用配置中的天数，但写入时必须提醒产品经理检查。
- `permanent`：长期保留，不计算下线日期。
- `required`：没有可靠默认值，登记时必须明确提供 `duration_days`。

默认值只用于 Product Review Plan 的时长建议，不代表用户已经确认。调用 `record_campaign.py` 时，每个 `--component-json` 都必须包含 `"duration_confirmed": true`；缺少或为 false 时拒绝登记。固定天数应使用已确认的 `duration_days`，不得用登记阶段的默认回退替代产品确认。

默认值因组件而异，不得用一个统一天数覆盖全部组件。`GuideAdminNotify`、`GuideUpdateBar` 以及与 `GuideAdminNotify` 关联的 `ProductUpdatesDrawer` 条目不由本 skill 开发或登记。独立的管控端 `ProductUpdatesDrawer` 可以登记，默认长期保留。

## AI 下线流程

1. 只读取 `main` 上的 `.clawpro/campaign.yaml`，忽略尚未合入主线的记录。
2. 对每个 `state=active` 且 `duration.mode=fixed_days` 的组件，使用所在 Campaign 的 `launched_on` 计算到期日。
3. 临近或达到到期日时，根据 `current_user_id` 提醒产品经理，并展示组件名称、文案、位置和到期日。
4. 产品经理确认后，根据 `code_paths` 定位代码并创建下线 MR；不得仅凭到期自动修改代码或提交 MR。
5. 创建 MR 后写入 `state=removal_mr_opened` 和 `removal_mr_url`；MR 合入 `main` 后写入 `state=removed` 和 `removed_on`。

使用 `scripts/record_campaign.py` 写入。每个组件通过一个 `--component-json` 参数提供，并携带 `duration_confirmed=true`；同一批次可重复传入该参数。`--launched-on` 默认使用登记当天的本地日期。目标文件不存在时初始化 `schema_version: 3` 和空的 `campaigns` 列表；旧版空文件可自动升级，旧版文件包含记录时必须先为每条记录补充真实 `launched_on`，不得静默改写。

每次成功写入后必须输出：

```text
Campaign 已登记：{campaign_id}
上线日期 launched_on：{launched_on}
请检查 current_user_id 是否为负责本次组件下线的产品经理：{current_user_id}
```

该提醒不因 `current_user_id` 是自动读取还是明确传入而省略。
