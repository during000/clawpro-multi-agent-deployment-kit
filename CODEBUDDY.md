# CodeBuddy 开发与部署规则

## 仓库定位

本仓库已经直接包含完整源码，不需要下载源码包或创建额外工作区：

- 页面与交互：`components/clawpro`
- 任务持久化、鉴权、HTTPS 与 WSS：`components/hatchery`
- 本地 Agent 监听与调用：`components/teamai-cli`
- 工作流节点、交接与产物：`components/orchestrator`

先读取 `README.md`、`DEVELOPMENT.md` 和 `SOURCE_STATE.md`，再根据需求只修改对应组件。不要把 Release 中的旧源码包覆盖到 `components/`。

## 开发流程

1. 根据需求定位组件和文件。
2. 检查当前工作区修改，保护用户已有改动。
3. 只修改需求范围内的源码。
4. 运行对应组件的最小验证；跨组件改动运行完整打包验证。
5. 需要部署时执行 `scripts/package-development.sh` 生成本地部署包，再使用 `scripts/deploy-remote.sh --archive ... --checksum ...` 部署。
6. 不提交 `.local-releases/`、依赖目录、构建产物、日志、数据库或凭证。

用户只需描述开发需求，无需复制完整 Prompt，也无需执行源码初始化命令。

## 组件验证

- ClawPro：在 `components/clawpro` 安装依赖并执行前端检查与构建。
- Hatchery：在 `components/hatchery` 执行相关 Go 测试。
- TeamAI：在 `components/teamai-cli` 执行类型检查、相关测试与构建。
- Orchestrator：在 `components/orchestrator` 执行 Python 单元测试。

## 服务器部署

用户要求部署时，需要：SSH 目标、端口、域名和可用的 SSH 私钥。执行：

```bash
bash scripts/deploy-remote.sh \
  --host '<SSH目标>' \
  --port '<SSH端口>' \
  --domain '<域名>' \
  --identity '<SSH私钥绝对路径>'
```

默认部署 GitHub Release。部署当前源码构建结果时必须同时传入 `--archive` 与 `--checksum`。

部署完成后检查：

- `https://<域名>/health`
- `https://<域名>/project-collaboration`
- 三个 systemd 服务状态

如果 HTTP 健康检查成功但 HTTPS 失败，应如实说明 DNS/TLS 网关尚未完成。

## 安全约束

- 不输出、提交或上传服务器生成的凭证。
- 不把服务器 `/opt/clawpro-multi-agent/config/hatchery.env` 下载到本地。
- 不关闭防火墙，不修改无关 Nginx、网关或业务服务。
- 80 端口被未知服务占用时停止并报告。
- 再次部署保留现有数据库与凭证，不删除 `/opt/clawpro-multi-agent/data`。
- 不将私有仓库改为公开仓库。
