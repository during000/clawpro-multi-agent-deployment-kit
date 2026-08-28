# 源码状态

源码汇入时间：2026-08-28（Asia/Shanghai）

本文件记录 `components/` 中完整可开发源码的上游基线。源码已直接提交到本仓库 `main`，不是必须另行下载的附件。

| 组件 | 上游 | 分支 | 基线提交 | 当前源码包含内容 |
| --- | --- | --- | --- | --- |
| ClawPro | `cvm-openclaw/openclaw-enterprise` | `feature/tenant-project-collaboration` | `c68231da86218d8b4d387d62ff814e5d88eec70d` | 包含项目协作前端当前未提交修改和新增节点资产 |
| Hatchery | `cvm-openclaw/hatchery` | `feature/clawpro-agent-task-backend` | `695e30d7941adb66726e5d62cb3eaa04939cc6d8` | 包含 Agent 任务、WSS 唤醒与回传链路当前未提交修改 |
| TeamAI CLI | `https://github.com/Tencent/teamai-cli.git` | `codex/clawpro-agent-task-executor` | `ae1642a667263ab1209c77ac03aeedfc4a57a34f` | 包含 ClawPro 任务监听、CodeBuddy ACP、iWiki MCP 代理当前未提交修改 |
| 编排服务 | 本地 POC | 无独立 Git 分支 | 2026-08-28 源码状态 | 包含工作流、Handoff、人工门禁、产物和多 Runtime 编排 |

## 已排除内容

- 所有 `.git` 目录与历史。
- `node_modules`、`dist`、Go `build` 和已生成 npm 包。
- `.env*`、数据库、日志、运行工作区和 TeamAI 本地状态。
- 当前服务器域名、IP、个人路径和本轮出现过的访问凭证。
