# [2026-08-07] ClawPro 本地 Agent 任务后端

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/clawpro-agent-task-backend` |
| 摘要 | 复用 local-agent report/sync/ack 通道，实现 ClawPro 向本地 TeamAI/Edge Runtime 下发 Agent 执行任务并回传结果 |
| 状态 | 本地开发完成 |
| 创建日期 | 2026-08-07 |

## Progress

- [x] 01. Clarify（后端先完成任务契约，不提交、不推送、不创建 MR）
- [x] 02. Plan（见 [02-plan.md](./02-plan.md)）
- [x] 03. Implement（任务 API、sync/ack 协议、数据库迁移完成）
- [x] 04. UT（全量 `go test ./...` 通过，新增函数覆盖率 80.6%–100%）
- [x] 05. Docs（API 与数据库迁移已同步）
- [x] 06. Review（gofmt、go vet、改动文件 staticcheck 通过）

## 关键决策

1. 复用现有 `/local-agent/sync` 与 `/local-agent/commands/ack`，不新增长连接和第二套设备鉴权。
2. 新命令类型为 `execute_agent_task`；TeamAI 负责调用本地 Agent CLI，Hatchery 不直接执行用户机器上的命令。
3. 创建任务时必须校验目标实例归属、项目成员关系和已上报的 Workspace 绑定，禁止下发任意本地路径。
4. TeamAI 用 `running` ACK 领取任务，可持续覆盖完整结果快照，最终以 `success` / `failed` 结束；ClawPro 通过任务查询接口轮询展示。
