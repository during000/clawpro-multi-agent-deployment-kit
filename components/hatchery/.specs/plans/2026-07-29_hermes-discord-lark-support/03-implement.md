# 03. Implement — 实现记录

---

## 关键实现细节

### Discord 通道（set_channel_hermes.sh）

- 新增变量 `DISCORD_BOT_TOKEN="{{bot_token}}"` 和 `DISCORD_USER_ID="{{user_id}}"`
- 白名单 case 加入 `discord`
- 新增 case 分支：校验 bot_token 和 user_id 非空 → 写入 `DISCORD_BOT_TOKEN` 和 `DISCORD_ALLOWED_USERS`（值为 user_id，因为 Hermes Discord 不支持通配符匹配）→ restart gateway

### Lark 通道（set_channel_hermes.sh）

- 复用 feishu 的 APP_ID / APP_SECRET 变量
- 白名单 case 加入 `lark`
- 新增 case 分支：校验 app_id 和 app_secret 非空 → 写入 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` / `FEISHU_DOMAIN=lark` / `FEISHU_ALLOWED_USERS=*` / `FEISHU_ALLOW_ALL_USERS=true` → restart gateway

### del_channel_hermes.sh

- Discord 删除：清理 `DISCORD_BOT_TOKEN` 和 `DISCORD_ALLOWED_USERS`
- Lark 删除：清理所有 `FEISHU_*` 变量（APP_ID / APP_SECRET / DOMAIN / ALLOWED_USERS / ALLOW_ALL_USERS）
- **修复了 lark 分支原始代码的 bug**：原代码在删除脚本中误用 `update_env`（写入函数）而非 `delete_env`（删除函数），且缺少 `ENV_FILE` 变量定义

### set_channel.sh（openclaw 路径）

- discord 分支新增 `--arg user_id "{{user_id}}"` 参数
- `allowFrom` 从 `["*"]` 改为 `[$user_id]`（指定用户 ID 而非通配符）

### model 层

- `agent_type.go`：`AgentTypeHermes` 白名单 map 新增 `"discord": true` 和 `"lark": true`
- `ai_channel.go`：`ChannelParams["discord"]` 新增 `{Key: "user_id", Label: "Discord User Id"}`

### 分支合并策略

- 两个 feature 分支分别 rebase 到 `Release/2026_07_28`
- discord 分支 rebase 时 set_channel_hermes.sh 冲突（原分支做了大量缩进/镜像源变更），解决策略：以 release 版本为基础，只保留 discord 功能改动
- 合并后 squash 为单个 commit

## 与 Plan 差异

无。

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./...` 无错误
- [x] 无数据库变更（不涉及 sql/init.sql / migration）
- [x] 无硬编码密钥/配置
- [x] 用户可见文案：脚本中的中文提示为运维日志，非 i18n 范畴
