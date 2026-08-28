# 01. Clarify — 需求澄清

## 背景

Hatchery 平台目前已支持飞书、企业微信、钉钉、QQ Bot、Slack、Microsoft Teams 等多个 IM 通道。业务方提出需要接入 LINE Messaging API，为 Hermes Agent 类型实例提供 LINE 即时通讯能力。

LINE 是日本及东南亚地区使用最广泛的即时通讯应用之一，接入后可扩展 Hatchery 在海外市场的 IM 通道覆盖范围。

## 目标

1. 为 Hermes Agent 新增 LINE 通道的配置能力（设置/删除）
2. 复用现有的 AgentProxyRoute 反向代理基础设施，为 LINE Webhook 创建专属代理路由
3. 自动管理安全组规则（放通 LINE Webhook 端口 8646）
4. 更新 API 文档和通道白名单

## 范围

### 包含

- **数据模型**：新增 `AgentProxyRouteKindLine` 常量，复用现有 `AgentProxyRoute` 表
- **反向代理路由**：LINE 代理监听 8646 端口，路径 `/line/webhook`
- **HTTP 方法**：LINE 支持 GET+POST（Webhook 验证需要 GET 请求），不同于 Teams 仅 POST
- **通道白名单**：Hermes Agent 白名单新增 `line`
- **渠道定义**：LINE 需要 `channel_token` + `channel_secret` 两个凭证，标记为海外范围
- **安全组**：新增 `allow_agent_proxy_line` 规则组，通过 `agent_proxy_line_enable` 条件控制
- **设置脚本**：Hermes 写入 4 个环境变量（`LINE_CHANNEL_ACCESS_TOKEN`、`LINE_CHANNEL_SECRET`、`LINE_PORT`、`LINE_ALLOW_ALL_USERS`）
- **删除脚本**：删除上述 4 个环境变量并重启 gateway
- **API 文档**：所有相关接口同步更新

### 不包含

- OpenClaw/ACE Agent 的 LINE 支持（仅 Hermes）
- 国内站点可见（LINE 仅海外）
- LINE 前端 UI 展示（复用现有通道卡片框架）

## 待确认问题

| # | 问题 | 决策 |
|---|------|------|
| 1 | LINE 通道是否仅 Hermes 使用？ | 是，仅 `AgentTypeHermes` 白名单包含 `line` |
| 2 | LINE 代理端口使用哪个？ | 8646，参考 LINE Messaging API 标准端口 |
| 3 | LINE Webhook 是否需要 GET 请求？ | 是，LINE Webhook 验证流程需要响应 GET 请求 |
| 4 | 安全组是否需动态控制？ | 是，通过 `agent_proxy_line_enable` 条件，仅配置 LINE 的实例放通 |

## 变更文件清单（预估）

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `model/agent_proxy_route.go` | 修改 | 新增 `AgentProxyRouteKindLine` 常量 |
| `controller/agent_proxy.go` | 修改 | 注册 LINE 路由规格、kind→channel 映射、GET+POST 方法支持 |
| `controller/openclaw_channel.go` | 修改 | setChannel/deleteChannel 中 LINE 处理分支 |
| `controller/admin_security_group.go` | 修改 | 安全组条件评估新增 `agent_proxy_line_enable` |
| `model/agent_type.go` | 修改 | Hermes 白名单新增 `line` |
| `model/ai_channel.go` | 修改 | LINE 渠道定义、凭证字段、海外范围标记 |
| `scripts/set_channel_hermes.sh` | 修改 | LINE 环境变量写入逻辑 |
| `scripts/del_channel_hermes.sh` | 修改 | LINE 环境变量删除逻辑 |
| `config/clawpro_required_sg_rules.json` | 修改 | 新增 `allow_agent_proxy_line` 安全组规则 |
| `docs/API.md` | 修改 | 所有相关接口文档同步更新 |
| `controller/agent_proxy_test.go` | 修改 | 新增 LINE 相关单元测试 |
| `model/agent_type_extended_test.go` | 修改 | 测试用例适配 |
| `model/agent_type_new_test.go` | 修改 | 测试用例适配 |
| `model/ai_channel_test.go` | 修改 | 测试用例适配 |
