# Docs

- 已更新 `docs/API.md`：项目 CRUD、成员、项目资产、本地 Workspace 反查、`GET /projects/mine`、`project_ids` 应用范围与 sync 命令 `project_id`。
- 已更新 `docs/local-agent-phase2-api.md`：Workspace 的 `project_id` 语义及 `scope=workspace` 命令字段。
- 已运行 `make openapi`，成功解析 390 个路径、399 个 operation，并生成当前 `docs/openapi.json`。

## 2026-07-21 最终核对

- `docs/project-asset-api.md` 已覆盖项目、分组资产详情/候选/保存、实例列表与资产版本记录契约。
- 本地 Agent 的 TeamAI 协议已统一收敛到 `docs/API.md`：用户级使用 `group_id`，Workspace 只使用 `project_id`，调用路径带 `/api` 前缀。
- 本轮重新执行 `make openapi` 成功；本期新增/调整的公开接口已具备文档来源。
