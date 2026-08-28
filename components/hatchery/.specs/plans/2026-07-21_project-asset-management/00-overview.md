# [2026-07-21] 项目资产管理

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/local-agent2-final` |
| 摘要 | 项目 CRUD、项目资产绑定与本地 Workspace 项目按需同步 |
| 状态 | 已完成（后端交付） |
| 创建日期 | 2026-07-21 |
| 负责人 | Codex |

## Progress

- [x] 01. Clarify → [01-clarify.md](./01-clarify.md)
- [x] 02. Plan → [02-plan.md](./02-plan.md)
- [x] 03. Implement  → [03-implement.md](./03-implement.md)
- [x] 04. UT         → [04-ut.md](./04-ut.md)
- [x] 05. Docs       → [05-docs.md](./05-docs.md)
- [x] 06. IT         → [06-it.md](./06-it.md)
- [x] 07. Review     → [07-review.md](./07-review.md)
- [x] 08. Commit     → [08-commit.md](./08-commit.md)

## 当前步骤

- **步骤**：✅ 已完成
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-21 14:18

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-15 00:00:00 | 2026-07-15 00:00:00 | 0 | OpenSpec 已确认 |
| 02. Plan | 2026-07-15 00:00:00 | 2026-07-15 00:00:00 | 0 | 设计已收敛 |
| 03. Implement | 2026-07-15 00:00:00 | 2026-07-21 14:18:16 | 历史过程未精确计时 | 项目资产、Local Agent、版本记录与同步模式已交付 |
| 04. UT | 2026-07-21 14:18:16 | 2026-07-21 14:18:16 | 本轮补录 | `go test ./controller -count=1` 通过 |
| 05. Docs | 2026-07-21 14:18:16 | 2026-07-21 14:18:16 | 本轮补录 | API、Local Agent 与前端资产契约已同步 |
| 06. IT | 2026-07-21 14:18:16 | 2026-07-21 14:18:16 | 本轮补录 | 远程集成报告与本地 schema 校验已留证 |
| 07. Review | 2026-07-21 14:18:16 | 2026-07-21 14:18:16 | 本轮补录 | 路由冲突、schema 与静态检查已复核 |
| 08. Commit | 2026-07-21 14:18:16 | 2026-07-21 14:18:16 | 本轮补录 | 历史功能提交已推送，本次补齐 SOP 交付记录 |

## 关键决策备忘

- 本期仅四张表：`projects`、`project_members`、`project_config_bindings`、`local_agent_scope_bindings`；不创建 `instance_projects`。
- 云端实例项目绑定和下发延后；未来以独立关系表通过 `project_id` 复用项目配置绑定。
- 项目删除保留 Workspace 原始 `project_id`，已删除项目不返回资产。
- 项目资产只有技能和规范；写入绑定不自动下发，仅 TeamAI sync 按需计算。
- OpenSpec 任务 6.2–6.5 保留为后续前端契约复核清单：不在本次后端 SOP 完结时伪标完成。

## 子任务：资产版本与同步模式（sync_mode + 版本记录）

> 方案见 [02-plan.md](./02-plan.md)；实现计划与测试设计见 [03-implement.md](./03-implement.md) 对应小节。

### 背景

TAPD #1020422209135817508【clawpro】新增项目与资产管理页。资产 Tab 需要：
1. **同步模式（sync_mode）**：管理员为分组/项目指定资产同步策略（initial_only 仅初始下发 / continuous 持续同步）。
2. **版本记录**：记录资产的每一次变更（手动保存 + 工具库自动变更），供前端时间线展示。

### 目标

- `/admin/assets/save` 新增 `sync_mode` 必填参数，保存时生成版本历史并按模式决定是否下发。
- 新增 `/admin/assets/versions` 分页接口，返回版本记录（结构化 segments，前端拼文案）。
- 工具库资产变更（发版/删除/范围调整）自动生成版本历史（部分场景触发存量实例下发）。

### 范围

- 本模块提供 seam 函数，调用方在现有 handler 内调用。
- 覆盖：model（`AssetVersionRecord` + 加列）、内部函数（`RecordAssetSave` / `PublishAssetVersion` / `InstallAssetToTargets`）、versions 查询 handler、migration SQL、单测。
- 不覆盖：现有的 save/create/delete/update handler 主体（本模块只插入调用 + 胶水代码）、下发任务执行逻辑。

### 非目标 / 明确不做

- 不做 `/admin/assets/version-detail` 接口（changes 随列表返回）。
- 不承载下发状态（sync_status/batch 等由现有 task 表负责）。
- 后端不拼 summary 文案，只返回结构化字段。
- group 不查子孙组（现有返回即最终结果）。

### 关键决策（与用户确认）

1. save 仅新增 `sync_mode` 一个请求参数。
2. 版本记录两个封装函数（手动 `RecordAssetSave` / 自动 `PublishAssetVersion`），5 类触发全部落库、事务内同步。
3. version 用 SQL 表达式原子自增，首建即 v1。
4. 下发判定：continuous + 有 added/updated 才下发；initial_only 或仅 removed 只记录。
5. 响应：分页 + `segments[].items` 结构化，`operator` 返回 type+id+name（system 时 id=0/name=""），不返回 summary/changes。
6. `sync_mode` 段统一用 `items[].name` 承载模式值（continuous/initial_only），不使用独立 `value` 字段（对齐 iwiki §5.1.1 示例 A）。
