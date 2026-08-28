# Clarify

## 背景

企业需要以「项目」管理本地 Agent Workspace 的资产配置；既有分组能力保留为用户级组织范围，项目不继承分组树。前端还需要在项目、资产管理、工具库和实例列表中展示并维护这层关系。

## 目标

- 提供项目 CRUD、成员维护和项目应用范围。
- 统一分组/项目的 skill、rule 资产绑定查询与保存。
- Workspace 以 `project_id` 绑定项目；TeamAI 依据项目资产持续下发。
- 删除项目后保留本地 Workspace 历史 ID，用失效状态兼容展示，不再下发资产。

## 已确认决策

- 关系表只引入 `projects`、`project_members`、`project_config_bindings`、`local_agent_scope_bindings`；不引入 `instance_projects`。
- 资产绑定类型仅支持企业 skill 与 rule；可见范围绑定和资产绑定在同一 binding 表中按 `config_type` 区分。
- `scope=user` 对应分组资产，`scope=workspace` 对应项目直接资产。
- 本期不做云端 CVM 项目绑定；版本记录与同步模式方案纳入 [02-plan.md](./02-plan.md)。

## 验收口径

- 管理端可分页查询项目、成员、本地 Agent 实例及其项目摘要。
- skill/rule 支持组织与项目应用范围的读写、列表筛选及响应展示。
- 有效项目持续按绑定资产补齐；失效项目不报错、不下发，并在实例详情中标识失效。
