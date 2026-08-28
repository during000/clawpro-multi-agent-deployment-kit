# 02. Plan — ClawPro 本地 Agent 任务后端

## 实施范围

1. 扩展 `local_agent_tasks`，记录项目、Workspace、Agent 类型、Prompt、会话和执行结果。
2. 新增 `POST /agent-tasks/create` 与 `GET /agent-tasks`。
3. `/local-agent/sync` 返回 `execute_agent_task`。
4. `/local-agent/commands/ack` 支持 `running`、增量结果、终态结果与幂等。
5. 同步 MySQL 初始化 SQL、增量迁移、SQLite→MySQL 迁移、API 文档和审计规则。

## 测试用例

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 创建任务 | 自有本地 Agent、已绑定项目 Workspace、有效 Prompt | 返回 200，任务为 pending 并正确落库 | P0 |
| 2 | 阻止任意路径 | Workspace 未由目标 Agent 上报绑定 | 返回 400，不创建任务 | P0 |
| 3 | 阻止越权实例 | 选择其他用户的本地 Agent | 返回 404 | P0 |
| 4 | Sync 领取任务 | 本地 Agent 存在 pending execute_agent_task | commands/cmds 均返回执行字段 | P0 |
| 5 | 实时结果快照 | running ACK 连续回传完整 result 快照 | 状态为 running，以最新快照为准且重试不重复 | P0 |
| 6 | 成功终态 | success ACK 带 session_id 与最终结果 | 状态、时间、会话和结果正确落库 | P0 |
| 7 | 终态幂等 | success 后再次 ACK | 不覆盖既有终态和结果 | P1 |
| 8 | 查询隔离 | 用户查询自己的任务并按项目/状态筛选 | 只返回本人任务 | P0 |
| 9 | 非法状态/超大结果 | ACK 非法 status 或超过限制的 result | 返回 400 | P1 |
