# 源码开发与重新部署

## 一、直接开发

克隆仓库后，源码已经位于 `components/`：

- `components/clawpro`：ClawPro 前端。
- `components/hatchery`：任务、HTTPS 与 WSS 后端。
- `components/teamai-cli`：TeamAI 本地任务执行插件。
- `components/orchestrator`：工作流编排服务。

不需要运行初始化脚本，也不需要下载 Release 源码附件。组件的原始仓库、分支、基线和未提交变更范围记录在 [SOURCE_STATE.md](SOURCE_STATE.md)。

## 二、按组件验证

```bash
# ClawPro 前端
cd components/clawpro
npm ci
npm run build

# TeamAI
cd ../teamai-cli
corepack pnpm install --frozen-lockfile
corepack pnpm typecheck
corepack pnpm test
corepack pnpm build

# Hatchery
cd ../hatchery
go test ./controller ./model

# 编排服务
cd ../orchestrator
python3 -m unittest discover -p 'test_*.py'
```

具体组件命令以各组件现有 `package.json`、`go.mod` 和测试文件为准。

## 三、一键构建部署包

在仓库根目录执行：

```bash
bash scripts/package-development.sh
```

脚本会构建前端、检查 TeamAI、执行相关测试、编译 Linux amd64 Hatchery、打包 TeamAI npm 包，并生成完整部署包及 SHA-256 文件。

本地输出位于 `.local-releases/`，不会提交到 GitHub。仅在明确需要快速排查打包问题时使用 `--skip-tests`，不要把它作为正式交付结果。

## 四、部署修改后的版本

使用构建脚本输出的 `ARCHIVE` 与 `CHECKSUM`：

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

## 五、开发依赖

- Node.js 22、npm、Corepack/pnpm。
- Go，版本满足 `components/hatchery/go.mod`。
- Python 3.9 或更高版本。
- GitHub CLI（部署 Release 或访问私有仓库时需要）。
- 可 SSH 登录的 Linux x86_64 测试服务器（部署时需要）。

如果修改需要回到原工蜂或 TeamAI 上游仓库提交，请依据 `SOURCE_STATE.md` 的来源与基线，将对应组件的改动迁移到上游 feature 分支并按其协作规范发起 MR。
