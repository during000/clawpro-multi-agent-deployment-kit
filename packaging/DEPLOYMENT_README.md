# ClawPro 多 Agent 协作部署包

这个包用于把当前已经跑通的 ClawPro 多 Agent 协作 Demo 部署到另一台 Linux 服务器，并让不同用户电脑上的 TeamAI、CodeBuddy 和 iMate 接入执行任务。

## 包含内容

- `server/frontend/`：ClawPro 项目协作前端构建产物。
- `server/orchestrator/`：工作流编排、节点流转、人工确认、输入输出与产物回传服务。
- `server/bin/hatchery-linux-amd64`：任务持久化、用户 API Token、HTTPS 领取任务和 WSS 唤醒服务。
- `client/teamai/teamai-cli-0.19.0.tgz`：包含 ClawPro 任务监听能力的 TeamAI 安装包。
- `server/install-server.sh`：服务器一键安装脚本。
- `client/teamai/`：用户电脑上的 TeamAI 安装、配置与监听脚本。

任务链路如下：

```text
ClawPro 创建并保存任务
→ WSS 通知用户电脑上的 TeamAI
→ TeamAI 通过 HTTPS 领取完整节点任务和配置资产
→ TeamAI 调用本地 CodeBuddy，或调用 iMate CLI 管理的 OpenClaw
→ TeamAI 通过 HTTPS 回传状态、结果和文件
→ ClawPro 展示结果并流转下一节点
```

## 适用范围

这是已经验证过真实本地 Agent 链路的测试/演示部署包，不是正式生产发行版。当前服务器包只支持 `Linux x86_64`；包内 TeamAI 接入脚本适用于安装 Node.js 22 的 macOS 和 Linux。

部署前需要：

- 一台 Linux x86_64 服务器，具有 root 权限。
- 已安装 Docker、systemd、Python 3 和 OpenSSL。
- 一个指向服务器的域名。
- 域名前有 HTTPS/WSS 网关或反向代理。包内 Nginx 监听 HTTP 80，TLS 建议由现有网关终止。
- 用户电脑能通过 HTTPS/WSS 访问该域名。

## 一、部署服务器

把压缩包上传到服务器并解压，然后执行：

```bash
cd clawpro-multi-agent-deployment-kit-20260828
sudo bash server/install-server.sh --domain clawpro.example.com
sudo /opt/clawpro-multi-agent/bin/healthcheck
```

安装脚本会：

1. 安装文件到 `/opt/clawpro-multi-agent`。
2. 首次运行时生成新的管理员密码、后台 Token 和 Cookie 密钥。
3. 注册并启动 Hatchery、编排服务和 Nginx 三个 systemd 服务。
4. 不覆盖已经存在的凭证文件和数据库。

管理员初始凭证保存在服务器的：

```text
/opt/clawpro-multi-agent/config/hatchery.env
```

该文件权限为 `0600`，不要发到群聊、工单或代码仓库。登录 `https://你的域名` 后，在用户设置中创建自己的 API Token，TeamAI 必须使用用户 Token，不能使用后台管理员 Token。

常用运维命令：

```bash
systemctl status clawpro-hatchery clawpro-orchestrator clawpro-nginx
journalctl -u clawpro-orchestrator -f
journalctl -u clawpro-hatchery -f
```

## 二、用户电脑接入 TeamAI

用户电脑需要 Node.js 22、CodeBuddy CLI；使用 iMate 节点时还需要已安装并登录的 iMate CLI。

安装 TeamAI：

```bash
bash client/teamai/install-teamai.sh
```

把在 ClawPro 页面创建的用户 API Token 保存到本机文件并限制权限：

```bash
mkdir -p "$HOME/.teamai"
chmod 700 "$HOME/.teamai"
printf '%s' '在这里粘贴用户 API Token' > "$HOME/.teamai/clawpro-token"
chmod 600 "$HOME/.teamai/clawpro-token"
```

在需要执行任务的源码工作区完成接入：

```bash
bash client/teamai/configure-teamai.sh \
  --endpoint https://clawpro.example.com \
  --token-file "$HOME/.teamai/clawpro-token" \
  --workspace /absolute/path/to/workspace \
  --project-id <ClawPro项目ID>
```

启动常驻监听：

```bash
bash client/teamai/run-listener.sh --workspace /absolute/path/to/workspace
```

监听启动后，ClawPro 创建的任务才会下发到这台电脑。别人打开网页不会自动使用部署者的电脑；每个执行者都需要用自己的 Token、工作区和 TeamAI 监听进程完成接入。

## 三、验收

服务器健康检查：

```bash
sudo /opt/clawpro-multi-agent/bin/healthcheck
```

客户端检查：

```bash
teamai --version
codebuddy --version
```

在 ClawPro 中创建一个极简任务，选择已经在线的 CodeBuddy Runtime。预期可看到：

1. TeamAI 收到 WSS 通知并领取任务。
2. 节点状态从待执行变为执行中、待确认或已完成。
3. 节点页面能查看输入、输出、配置资产和文件产物。
4. 需要人工门禁的节点确认后才流转下一节点。

## 四、可选能力

- iMate：需在用户电脑安装并登录 iMate CLI，并保证相应 iMate 项目和 Agent 已授权。
- iWiki/MCP：需在用户电脑或受控执行环境配置对应 MCP、用户授权和网络访问；部署包不附带任何访问凭证。
- CloudAgent：在服务器的 `orchestrator.env` 中配置服务端网关与专用凭证后启用；默认关闭。

## 五、升级与卸载

再次运行 `install-server.sh` 会更新程序文件，并保留现有数据库和凭证。升级前仍建议备份：

```bash
cp /opt/clawpro-multi-agent/data/hatchery.db /root/hatchery.db.backup
```

停止服务：

```bash
systemctl disable --now clawpro-nginx clawpro-orchestrator clawpro-hatchery
```

安装目录包含任务数据库和运行产物，不应直接删除；需要卸载时请先备份后再人工处理。

## 安全说明

包内不包含当前测试环境的域名、IP、Cookie、个人 Token、CloudAgent 密钥或 iMate 凭证。进一步约束见 [SECURITY.md](SECURITY.md)。
