# 02. Plan — 方案设计

## 改动文件（实际）

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `model/agent_proxy_route.go` | 修改 | 新增 `AgentProxyRouteKindLine = "line"` 常量 |
| `controller/agent_proxy.go` | 修改 | 注册 LINE 路由规格（端口 8646, 路径 /line/webhook）；kind→channel 映射；GET+POST 方法支持；`checkAgentProxyKindAllowed` 新增 channel 校验 |
| `controller/openclaw_channel.go` | 修改 | `applyManualChannelConfig`/`setChannel` 中 LINE 代理路由创建；`deleteChannel` 中 LINE 代理路由禁用+安全组刷新 |
| `controller/admin_security_group.go` | 修改 | 安全组条件评估新增 `agent_proxy_line_enable` case |
| `model/agent_type.go` | 修改 | `agentTypeChannelWhitelist[AgentTypeHermes]` 新增 `"line": true` |
| `model/ai_channel.go` | 修改 | LINE 渠道定义（凭证字段、海外范围、预定义渠道列表） |
| `scripts/set_channel_hermes.sh` | 修改 | LINE case 分支：写入 4 个环境变量 + 写入 `config.yaml` 的 `gateway.platforms.line.enabled=true` + 重启 gateway |
| `scripts/del_channel_hermes.sh` | 修改 | LINE case 分支：删除 4 个环境变量 + 从 `config.yaml` 删除 `gateway.platforms.line` 块 + 重启 gateway |
| `config/clawpro_required_sg_rules.json` | 修改 | 新增 `allow_agent_proxy_line` 规则组（TCP 8646） |
| `docs/API.md` | 修改 | 所有相关接口文档同步更新 LINE 支持说明 |
| `controller/agent_proxy_test.go` | 修改 | 新增 LINE 相关单元测试 |
| `model/agent_type_extended_test.go` | 修改 | 测试用例适配 LINE 白名单变更 |
| `model/agent_type_new_test.go` | 修改 | 测试用例适配 |
| `model/ai_channel_test.go` | 修改 | 测试用例适配 |

## 调用链

### 设置 LINE 通道

```
POST /openclaw/channel (channel=line, key=channel_token, value=xxx, key=channel_secret, value=yyy)
  → HandleSetChannel → handleSetChannel → setChannel
    → validateManualChannelConfig
      → channelInCurrentSiteScope(ctx, "line")      // 海外站点校验
      → model.AgentTypeChannelAllowed(ctx, hermes, "line")  // Hermes 白名单校验
    → ensureAgentProxyRoute(r, instance, AgentProxyRouteKindLine)
      → agentProxyRouteSpecForKind → {Port: 8646, Path: "/line/webhook"}
      → model.DB.CreateOrUpdate → AgentProxyRoute 记录
    → channelScriptRunner(ctx, ...) → set_channel_hermes.sh
      → 写入 LINE_CHANNEL_ACCESS_TOKEN / LINE_CHANNEL_SECRET / LINE_PORT / LINE_ALLOW_ALL_USERS
      → 写入 ~/.hermes/config.yaml 的 gateway.platforms.line.enabled=true
      → restart_gateway
    → jsonOK(w, resp{proxy_route_id, proxy_endpoint})
```

### 删除 LINE 通道

```
POST /openclaw/channel/del (channel=line)
  → HandleDelChannel → handleDelChannel → deleteChannel
    → ResolveScript(ctx, "del_channel", hermes) → del_channel_hermes.sh
    → delChannelScriptRunner(ctx, ...)
      → 删除 LINE_CHANNEL_ACCESS_TOKEN / LINE_CHANNEL_SECRET / LINE_PORT / LINE_ALLOW_ALL_USERS
      → 从 ~/.hermes/config.yaml 删除 gateway.platforms.line 块
      → restart_gateway
    → model.DB.Update("enabled", false) → 禁用 AgentProxyRoute
    → RefreshAllRuleSetsForRequiredRules(ctx) → 刷新安全组规则
```

### 代理请求转发

```
GET/POST /api/proxy/{routeID}/line/webhook
  → HandleAgentProxy
    → 解析 routeID → 查询 AgentProxyRoute (kind=line, enabled=true)
    → 方法校验：GET+POST 允许
    → 反向代理到实例 {TargetIP}:8646/line/webhook
```

## 测试用例设计（自然语言描述）

### agent_proxy.go

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| 1 | unknown kind 直接返回 nil | `kind="unknown_kind"` | `err == nil` |
| 2 | domestic 环境 LINE 被 site scope 拒绝 | `kind="line"`, domestic context | `err != nil` (MsgChannelNotExist) |
| 3 | Agent type 不支持 LINE | `kind="line"`, AgentTypeDeepSeekTUI, overseas | `err != nil` (MsgAgentTypeNotSupportChannel) |

### openclaw_channel.go

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| 4 | msteams 通道设置成功（海外环境） | `channel=msteams`, overseas, Hermes | 200, resp 含 `proxy_route_id`, `proxy_endpoint`, `teams_endpoint` |
| 5 | LINE 通道设置成功（海外环境） | `channel=line`, overseas, Hermes | 200, resp 含 `proxy_route_id`, `proxy_endpoint` |
| 6 | LINE 通道设置时代理路由创建失败 | `channel=line`, overseas, Hermes, resolveInstanceAccessIP 失败 | 500 |
| 7 | LINE 通道删除成功 | `channel=line`, overseas, Hermes, 已有启用路由 | 200, 路由 `enabled=false` |
| 8 | LINE 通道删除（路由已存在） | 同上 | 200, 安全组规则刷新失败仅 WARN 不阻塞 |
| 9 | `applyManualChannelConfig` LINE 分支 | `preset.Channel="line"`, overseas, Hermes | 返回 `ProxyRouteID != ""`, `ProxyEndpoint != ""` |

## 风险评估

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | LINE Webhook 需要 GET 请求，不同于 Teams | 中 | `HandleAgentProxy` 中单独判断 LINE 允许 GET+POST |
| 2 | 安全组规则暴露 8646 端口 | 低 | 通过 `agent_proxy_line_enable` 条件动态控制 |
| 3 | Hermes 脚本新增 LINE 分支可能影响现有通道 | 低 | case 分支隔离，不影响现有通道逻辑 |
| 4 | LINE 仅海外可用，国内误配需拦截 | 低 | `ChannelScopeOverseas` + `channelInCurrentSiteScope` 双重校验 |
