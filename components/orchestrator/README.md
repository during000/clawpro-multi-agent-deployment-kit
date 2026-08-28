# ClawPro Edge Runtime PoC

这是一个本地联调页面，不连接 ClawPro 生产环境、不修改任何业务仓库。默认链路会连接本机 Hatchery 测试服务，由 TeamAI 真实拉取 `execute_agent_task`，再通过 ACP stdio 调用本机 CodeBuddy，并把状态与结果回传 Hatchery。

它把本地执行链路拆成两层：

- **Hatchery**：创建任务、校验项目/实例/工作区归属，并记录任务状态与结果。
- **TeamAI**：常驻进程先通过 `/local-agent/wake` 接收轻量 WSS 唤醒，再通过 `/local-agent/sync` 拉取完整任务，执行后通过 `/local-agent/commands/ack` 回传 running/success/failed。
- **CodeBuddy ACP**：在 Hatchery 已绑定的独立工作区内真实执行任务。

它演示：

- 本机 Runtime 发现。
- TeamAI 设备通道在线/离线及 Hatchery 队列保留。
- WSS 只发送 `task_id`，完整 Prompt 和执行参数始终通过带认证的 HTTPS sync 领取。
- WSS 断开后自动重连，并在连接建立时全量 sync 补拉离线任务。
- TeamAI 真实收取 `execute_agent_task` 并交给 CodeBuddy ACP。
- TeamAI 按节点能力为 CodeBuddy 加载受控 MCP：iWiki 节点只开放 `metadata` / `getSpacePageTree` / `getDocument`，用户 PAT 仅从 TeamAI 进程环境继承，不写入任务、工作区或 MCP 配置。
- `CodeBuddy → iMate` 双节点协作：上游完成后由 ClawPro 自动生成 Context Package，再创建下游 iMate OpenClaw 任务。
- 使用 ClawPro Handoff v2 交接：`NodeInput v2` 映射 `$input/$vars/$artifacts`，`NodeResult v2` 分离数据与文件，产物携带版本、SHA-256 和血缘。
- 每个节点建立独立 `node-workspaces/<node>/`，输入落到 `.upstream_artifacts/`；完成后强制生成并校验 `handoff/.handoff.json`、`.handoff.md` 和正式产物。
- 结构化 IssueFix 工作流在分析完成和测试完成后暂停：用户可先查看节点产物，再确认进入修复或 MR 阶段；确认由后端门禁校验，不是前端提示。
- 任务 queued → claimed → running → completed。
- 真实 ACP JSON-RPC 2.0 stdio 握手。
- `session/new`、模型配置、`session/prompt`。
- `session/update` 文本和工具事件归一化。
- Handoff 交接结果生成。
- 页面按事件序号轮询和恢复。
- 执行中取消。

## 启动

```bash
HATCHERY_API=http://127.0.0.1:8091 \
HATCHERY_ADMIN_TOKEN=<本地测试管理员Token> \
TAI_PAT_TOKEN=<用户在 tai.it.woa.com 创建的 PAT> \
python3 server.py --port 4188
```

打开：<http://127.0.0.1:4188/>

## 说明

- 默认选项 `Hatchery → TeamAI → CodeBuddy` 是真实云—本链路；`本地直连 CodeBuddy ACP` 和 `Mock ACP Agent` 仅用于对比回归。
- “常驻 TeamAI”是正式架构 PoC；“兼容模式”保留原来的 `hook-dispatch` 链路，便于在同一页面对比。
- 每个云—本任务的独立目录位于 `real-agent-workspaces/hatchery-teamai/<task_id>`。
- TeamAI 默认只向 CodeBuddy 开放 `Read,Write,Edit,Glob,Grep`，不开放 Bash。
- 未配置 `TAI_PAT_TOKEN` 时 CodeBuddy 仍可执行普通节点；只有声明 `iwiki.read` 的节点会显示“待授权”并禁止选择，不再整体禁用 CodeBuddy。
- `Mock ACP Agent` 保留为无模型调用的快速协议回归模式。
- 页面同时检测本机 Codex，但尚未开放 Codex 真实执行选项。
- “CodeBuddy → iMate 协作”已使用 ClawPro Handoff v2（借鉴 iMate 的结构化输入与 Handoff 清单）。当前远端 iMate 节点仍以内联方式接收选定 UTF-8 文件；二进制、大文件和跨设备下载在正式版本接入 ClawPro Artifact Store。
- 页面使用本地临时测试用户、项目和 Token，不得传入生产管理员 Token。
- TeamAI 设备通道断开时，新任务保留在 Hatchery 队列；重新连接后触发 TeamAI 拉取并执行。
