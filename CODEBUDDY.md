# CodeBuddy 部署规则

## 目标

当用户要求部署本仓库时，把 GitHub Release 中的 ClawPro 多 Agent 协作套件部署到用户明确指定的服务器，并完成健康检查。

## 用户只需提供

1. SSH 目标，例如 `root@203.0.113.10`。
2. SSH 端口。
3. ClawPro 域名，不包含 `https://`。
4. SSH 私钥绝对路径；已配置 SSH Agent 时可不填。

不要让用户复制完整开发 Prompt，不要求用户提供 Hatchery 管理 Token、Cookie Secret 或 TeamAI Token。服务器密钥由安装脚本首次部署时生成。

## 执行流程

1. 读取本文件与 `README.md`。
2. 只读检查：
   - `gh auth status`
   - `ssh`、`scp`、`shasum` 是否存在
   - SSH 私钥是否存在且不是目录
3. 如果 GitHub 未登录、用户没有私有仓库访问权、SSH 参数缺失或域名尚未给出，停止并只询问缺失项。
4. 执行：

   ```bash
   bash scripts/deploy-remote.sh \
     --host '<SSH目标>' \
     --port '<SSH端口>' \
     --domain '<域名>' \
     --identity '<SSH私钥绝对路径>'
   ```

5. 脚本会自动：
   - 从 GitHub Release 下载部署包和校验文件。
   - 验证 SHA-256。
   - 对目标服务器执行系统、架构和依赖预检。
   - 上传并安装 ClawPro 前端、编排服务、Hatchery 和 Nginx。
   - 启动 systemd 服务并运行健康检查。
6. 部署完成后检查：
   - `https://<域名>/health`
   - `https://<域名>/project-collaboration`
7. 如果 HTTP 健康检查成功但 HTTPS 失败，说明 ClawPro 已安装、外部 DNS/TLS 网关尚未完成；不要虚构部署成功。

## 安全约束

- 不输出、提交或上传服务器生成的凭证。
- 不把服务器 `/opt/clawpro-multi-agent/config/hatchery.env` 下载到本地。
- 不关闭防火墙，不修改与本任务无关的 Nginx、网关或业务服务。
- 检测到 80 端口已被未知服务占用时停止，列出占用进程并让用户决定。
- 再次部署默认保留现有数据库和凭证，不删除 `/opt/clawpro-multi-agent/data`。
- 不将私有仓库改为公开仓库。

## 完成输出

只需告诉用户：

- 部署服务器和域名。
- 安装版本。
- 三个 systemd 服务状态。
- HTTP/HTTPS 健康检查结果。
- TeamAI 客户端下一步接入命令所在目录。
