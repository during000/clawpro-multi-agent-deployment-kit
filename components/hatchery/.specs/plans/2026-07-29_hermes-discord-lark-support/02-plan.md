# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/agent_type.go` | 修改 | Hermes 白名单新增 `discord` 和 `lark` |
| `model/ai_channel.go` | 修改 | discord ChannelParams 新增 `user_id` 字段 |
| `model/agent_type_new_test.go` | 修改 | hermes-supported-channels 测试用例 wantAny 加入 `lark` 和 `discord` |
| `model/agent_type_extended_test.go` | 修改 | hermes 通道数断言从 7 改为 9 |
| `scripts/set_channel_hermes.sh` | 修改 | 新增 discord / lark 变量定义、白名单、case 分支 |
| `scripts/del_channel_hermes.sh` | 修改 | 新增 discord / lark 通道删除逻辑 |
| `scripts/set_channel.sh` | 修改 | discord 分支新增 user_id 参数，allowFrom 改为指定用户 |
| `docs/API.md` | 修改 | discord 参数说明新增 user_id |

## 调用链 / 数据流

```
前端配置通道 → controller → 渲染 set_channel_hermes.sh 模板 → 执行脚本 → 写入 ~/.hermes/.env → restart gateway
```

- Discord：写入 `DISCORD_BOT_TOKEN` + `DISCORD_ALLOWED_USERS`（user_id）
- Lark：写入 `FEISHU_APP_ID` + `FEISHU_APP_SECRET` + `FEISHU_DOMAIN=lark` + `FEISHU_ALLOWED_USERS=*` + `FEISHU_ALLOW_ALL_USERS=true`

## 数据库变更

无。通道白名单为 Go 代码硬编码的 map，无数据库表变更。

## 测试用例设计（自然语言描述）

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | Hermes 支持的通道列表包含 discord | `SupportedChannelsByAgentType(ctx, "hermes")` | 结果包含 `discord` | P0 |
| 2 | Hermes 支持的通道列表包含 lark | `SupportedChannelsByAgentType(ctx, "hermes")` | 结果包含 `lark` | P0 |
| 3 | Hermes 通道总数为 9 | `SupportedChannelsByAgentType(ctx, "hermes")` | len == 9 | P0 |

### 集成测试（IT）

跳过：本次改动为通道配置脚本和白名单 map 变更，无新增 API 接口，无需集成测试覆盖。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | Discord 不支持通配符，用户 ID 填错导致 bot 不响应 | 中 | 中 | 脚本校验 user_id 非空，前端应做输入校验 |
| 2 | Lark 删除通道时残留 feishu 变量影响 feishu 通道 | 低 | 中 | del_channel_hermes.sh 清理所有 FEISHU_* 变量 |
| 3 | 两个 feature 分支合并冲突 | 高 | 低 | 已手动解决，保留两者功能 |
