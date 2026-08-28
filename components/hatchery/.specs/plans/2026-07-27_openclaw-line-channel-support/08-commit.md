# 08. Commit — 提交

## Commit History

### Commit 1: 主功能
```
e47c6849 feat(model,controller,scripts): 新增 LINE IM 通道支持（仅 Hermes Agent）

- 新增 AgentProxyRouteKindLine 常量，复用 AgentProxyRoute 表
- LINE 代理路由：端口 8646，路径 /line/webhook，支持 GET+POST
- Hermes Agent 通道白名单新增 line
- LINE 渠道定义：ChannelScopeOverseas，凭证 channel_token + channel_secret
- 新增 allow_agent_proxy_line 安全组规则组（TCP 8646）
- set_channel_hermes.sh / del_channel_hermes.sh 新增 LINE 分支
- checkAgentProxyKindAllowed 新增 channel 校验逻辑
- API 文档同步更新
```

### Commit 2: 单元测试补充
```
b887f78d test(controller): 补充 LINE 通道相关单元测试

- agent_proxy_test.go: 新增 3 个 checkAgentProxyKindAllowed 测试
- openclaw_channel_extended_test.go: 新增 6 个 LINE/msteams 通道测试
- 新增 delChannelScriptRunner 变量支持 deleteChannel 测试 mock
```

### 前置依赖 Commit
```
87427ffb fix: 通过上下文传递I18n Printer给后台任务，保证后台任务消息国际化
```

## 提交前检查

| # | 检查项 | 状态 |
|---|--------|------|
| 1 | `gofmt -w .` + `goimports -w .` | ✅ |
| 2 | `go vet ./...` | ✅ |
| 3 | `go test ./... -v -race` | ✅ |
| 4 | 无裸 SQL | ✅ |
| 5 | 无硬编码密钥 | ✅ |
| 6 | i18n 文案已注册 | ✅ |
| 7 | API.md 已更新 | ✅ |
| 8 | 无破坏性 API 变更 | ✅ |
