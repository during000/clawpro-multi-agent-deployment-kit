# ClawPro 多 Agent 协作源码

这是 ClawPro 多 Agent 协作 Demo 的完整可开发源码仓库。克隆 `main` 后即可查看、修改、测试和重新构建，不需要再下载或解压源码快照。

## 源码目录

```text
components/
├── clawpro/       ClawPro 项目协作前端
├── hatchery/      任务持久化、HTTPS 领取与 WSS 唤醒后端
├── teamai-cli/    本地任务监听、CodeBuddy ACP 与 iMate 调用
└── orchestrator/  工作流编排、节点流转、交接与产物管理
```

仓库同时包含构建、打包与服务器部署脚本。各组件当前来源、分支和基线见 [SOURCE_STATE.md](SOURCE_STATE.md)。

## 克隆后继续开发

```bash
gh auth login
gh repo clone during000/clawpro-multi-agent-deployment-kit
cd clawpro-multi-agent-deployment-kit
```

在 CodeBuddy 中直接打开这个目录并描述需求即可。CodeBuddy 会读取 [CODEBUDDY.md](CODEBUDDY.md)，定位对应组件、执行验证并按需生成部署包。无需复制完整开发 Prompt。

开发与构建命令见 [DEVELOPMENT.md](DEVELOPMENT.md)。

## 构建完整部署包

准备 Node.js 22、pnpm、Go 和 Python 3 后执行：

```bash
bash scripts/package-development.sh
```

生成的部署包与 SHA-256 文件位于 `.local-releases/`，不会提交到 Git。

## 部署已有 Release

```bash
bash scripts/deploy-remote.sh \
  --host root@服务器地址 \
  --port 22 \
  --domain clawpro.example.com \
  --identity /absolute/path/to/id_ed25519
```

也可以通过 `--archive` 和 `--checksum` 部署本地重新构建的版本。

## 部署结果

服务器将安装 ClawPro 前端、工作流编排服务、Hatchery 和 Nginx。用户电脑仍需运行 TeamAI 与目标 Agent Runtime；CodeBuddy 节点通过 ACP 调用，iMate 节点通过 iMate CLI/Daemon 调用。

## 边界

- 仓库包含可开发源码、构建配置、锁文件、测试和部署脚本。
- 不提交依赖目录、构建输出、运行日志、数据库、环境变量或密钥。
- 这是私有测试/演示仓库，不是正式生产发行版。
- 原组件 Git 历史不嵌套进本仓库；来源基线记录在 `SOURCE_STATE.md`。
- 安全说明见 [SECURITY.md](SECURITY.md)。

当前部署 Release：[v2026.08.28-poc.1](https://github.com/during000/clawpro-multi-agent-deployment-kit/releases/tag/v2026.08.28-poc.1)
