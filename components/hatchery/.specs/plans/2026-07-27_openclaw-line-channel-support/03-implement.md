# 03. Implement — 实现

## 对应 Commit

- `87427ffb` — fix: 通过上下文传递 I18n Printer 给后台任务（前置依赖修复）
- `e47c6849` — feat(model,controller,scripts): 新增 LINE IM 通道支持（仅 Hermes Agent）

## 关键实现细节

### 1. 数据模型层（model）

**`model/agent_proxy_route.go`**：新增 `AgentProxyRouteKindLine = "line"` 常量，复用现有 `AgentProxyRoute` 表，无需 DDL 变更。

**`model/agent_type.go`**：`agentTypeChannelWhitelist[AgentTypeHermes]` 新增 `"line": true`。仅 Hermes 支持 LINE，OpenClaw/ACE 不支持。

**`model/ai_channel.go`**：
- 凭证字段：`channel_token`（Channel Access Token）、`channel_secret`（Channel Secret）
- 渠道范围：`ChannelScopeOverseas`（仅海外站点可见）
- 预定义渠道列表新增 `{ChannelID: "line", Name: "LINE", NameEn: "LINE"}`

### 2. 控制器层（controller）

**`controller/agent_proxy.go`**：
- `agentProxyRouteSpecs` 新增 LINE 路由规格：端口 8646，路径 `/line/webhook`
- `agentProxyKindToChannel` 新增 `"line" → "line"` 映射
- `HandleAgentProxy` 中 LINE 路由允许 GET+POST（不同于 Teams 仅 POST），因为 LINE Webhook 验证需要响应 GET 请求
- `checkAgentProxyKindAllowed` 新增 channel 校验逻辑（通过 `agentProxyKindToChannel` 映射到 channel ID，再校验 site scope 和 agent type 白名单）

**`controller/openclaw_channel.go`**：
- `applyManualChannelConfig`：LINE 通道预设配置时自动创建代理路由
- `setChannel`：LINE 通道设置时调用 `ensureAgentProxyRoute`，响应中返回 `proxy_route_id`、`proxy_endpoint`
- `deleteChannel`：LINE 通道删除时禁用代理路由（`enabled=false`），刷新安全组规则
- 新增 `delChannelScriptRunner` 变量（用于测试 mock）

**`controller/admin_security_group.go`**：安全组条件评估新增 `agent_proxy_line_enable` case。

### 3. 脚本层（scripts）

**`scripts/set_channel_hermes.sh`**：
- 白名单新增 `line`
- LINE case 分支写入 4 个环境变量：`LINE_CHANNEL_ACCESS_TOKEN`、`LINE_CHANNEL_SECRET`、`LINE_PORT`(8646)、`LINE_ALLOW_ALL_USERS`(true)
- 新增 `enable_line_platform_in_config_yaml()` 函数，幂等地在 `~/.hermes/config.yaml` 写入 `gateway.platforms.line.enabled: true`
  - 三层 fallback：`yq` → Python+PyYAML → Python 纯文本（re）处理
  - 兼容空文件、已有 `gateway:` 但无 `platforms:`、已有 `platforms:` 含其他通道、`line:` 已存在等多种场景
- 配置完成后重启 gateway

**`scripts/del_channel_hermes.sh`**：
- 白名单新增 `line`
- 新增 `find_hermes_python()` 函数（与 set 脚本一致）
- 新增 `disable_line_platform_in_config_yaml()` 函数，从 `~/.hermes/config.yaml` 删除 `gateway.platforms.line` 块，保留其他平台；幂等（不存在则跳过）
  - 同样三层 fallback：`yq` → PyYAML → 纯文本
- LINE case 分支删除 4 个环境变量 + 删除 config.yaml 中 line 块 + 重启 gateway

### 4. 配置层（config）

**`config/clawpro_required_sg_rules.json`**：新增 `allow_agent_proxy_line` 安全组规则组，放通 TCP 8646 端口入站流量。

### 5. 文档层（docs）

**`docs/API.md`**：所有相关接口文档同步更新 LINE 支持说明，包括 `POST /openclaw/proxy/prepare`、`POST /openclaw/set-channel`、`POST /admin/instances/proxy/prepare`、`POST /admin/instances/set-channel`。

## 与 Plan 的差异

- Plan 中预估的 `AgentTypeClawproACE` 白名单变更，实际实现为 `AgentTypeHermes` 白名单（Hermes 是 Clawpro ACE 的运行时类型，实际代码中使用的是 `AgentTypeHermes` 常量）
- 新增了 `delChannelScriptRunner` 变量以支持 `deleteChannel` 中 `RunScript` 的测试 mock（Plan 阶段未预见）
