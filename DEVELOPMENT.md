# 源码开发与重新部署

## 一、初始化源码工作区

在已经 clone 的部署仓库中执行：

```bash
bash scripts/setup-development.sh
```

脚本会从当前私有 GitHub Release 下载源码快照、验证 SHA-256，并解压到：

```text
workspace/clawpro-multi-agent-source-20260828/
```

源码包括：

- `repos/clawpro`：ClawPro 前端。
- `repos/hatchery`：任务、HTTPS 和 WSS 后端。
- `repos/teamai-cli`：TeamAI 本地任务执行插件。
- `repos/orchestrator`：工作流编排服务。

打开源码根目录后，CodeBuddy 会读取源码快照内的 `CODEBUDDY.md`，根据需求只修改对应组件。

## 二、验证并生成新的部署包

完成代码修改后执行：

```bash
bash scripts/package-development.sh \
  --source-root "$PWD/workspace/clawpro-multi-agent-source-20260828"
```

默认会：

1. 安装缺失的前端和 TeamAI 项目依赖。
2. 构建 ClawPro 前端。
3. 执行 TeamAI 类型检查和任务执行器测试。
4. 执行 Python 编排服务测试。
5. 执行 Hatchery 相关 Go 测试。
6. 交叉编译 Linux amd64 Hatchery。
7. 打包新的 TeamAI npm 安装包。
8. 生成新的完整部署包和 SHA-256 文件。

本地开发部署包生成在 `.local-releases/`，不会提交到 GitHub。

## 三、部署修改后的版本

使用上一步输出的 `ARCHIVE` 与 `CHECKSUM`：

```bash
bash scripts/deploy-remote.sh \
  --host root@服务器地址 \
  --port 22 \
  --domain clawpro.example.com \
  --identity /absolute/path/to/id_ed25519 \
  --archive /absolute/path/to/dev-package.tar.gz \
  --checksum /absolute/path/to/dev-package.tar.gz.sha256
```

部署脚本保留服务器已有数据库和凭证，只更新程序文件并重启服务。

## 四、开发依赖

- Node.js 22、npm、Corepack/pnpm。
- Go，版本需满足 Hatchery `go.mod`。
- Python 3.9 或更高版本。
- GitHub CLI，且具有当前私有仓库访问权。
- 可 SSH 登录的 Linux x86_64 测试服务器。

源码快照不包含原仓库 Git 历史。需要向工蜂提交时，应由拥有工蜂权限的开发者把修改迁移回对应 feature 分支，再按 ClawPro 协作规范提交 MR。
