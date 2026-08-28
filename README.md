# ClawPro 多 Agent 协作部署入口

这个仓库用于让 CodeBuddy 部署和继续开发当前 ClawPro 多 Agent 协作 Demo。

服务器程序以 GitHub Release 附件保存，不会把 70 MB 构建产物写进 Git 历史。部署脚本会自动下载最新 Release、校验 SHA-256、上传服务器并完成安装。

## 交给 CodeBuddy 部署

先登录 GitHub CLI 并 clone 私有仓库：

```bash
gh auth login
gh repo clone during000/clawpro-multi-agent-deployment-kit
cd clawpro-multi-agent-deployment-kit
```

然后在 CodeBuddy 中打开该目录，只需告诉它：

```text
请按照 CODEBUDDY.md 部署 ClawPro。
服务器：root@服务器地址
SSH 端口：22
域名：clawpro.example.com
SSH 私钥：/absolute/path/to/id_ed25519
```

CodeBuddy 会读取仓库内的 [CODEBUDDY.md](CODEBUDDY.md)，自动完成环境检查、Release 下载、完整性校验、服务器上传、安装和健康检查。

## 继续开发源码

在仓库中告诉 CodeBuddy：

```text
请按照 CODEBUDDY.md 初始化源码开发环境，我要继续开发 ClawPro 多 Agent 协作功能。
```

CodeBuddy 会下载经过校验的源码快照，得到 ClawPro、Hatchery、TeamAI 和编排服务四个源码目录。修改完成后，它可以重新构建完整部署包并部署到指定服务器。详细流程见 [DEVELOPMENT.md](DEVELOPMENT.md)。

## 直接运行

也可以不经过 CodeBuddy，直接执行：

```bash
bash scripts/deploy-remote.sh \
  --host root@服务器地址 \
  --port 22 \
  --domain clawpro.example.com \
  --identity /absolute/path/to/id_ed25519
```

## 部署结果

服务器将安装：

- ClawPro 项目协作前端。
- 工作流编排服务。
- Hatchery 任务持久化、HTTPS 领取任务与 WSS 唤醒服务。
- Nginx 反向代理。

用户电脑仍需安装 TeamAI、CodeBuddy；使用 iMate 节点时还需安装并登录 iMate CLI。TeamAI 客户端安装包位于下载后的 Release 中。

## 前置条件

- clone 私有仓库的 GitHub 用户已被添加为协作者。
- 本机已安装并登录 GitHub CLI：`gh auth status` 可通过。
- 本机可以通过 SSH 登录目标服务器。
- 目标服务器为 Linux x86_64，具有 root 权限。
- 域名已经解析到服务器或其 HTTPS 网关。
- 正式远程 TeamAI 链路必须使用 HTTPS/WSS；仓库不会自动修改企业网关或 DNS。

## Release

当前版本：[v2026.08.28-poc.1](https://github.com/during000/clawpro-multi-agent-deployment-kit/releases/tag/v2026.08.28-poc.1)

这是测试/演示套件，不是正式生产发行版。安全边界见 [SECURITY.md](SECURITY.md)。
