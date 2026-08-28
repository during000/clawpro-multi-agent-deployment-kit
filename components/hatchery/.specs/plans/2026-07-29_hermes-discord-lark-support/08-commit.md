# 08. Commit — 提交记录

---

## 提交前检查

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt` + `go vet` 通过
- [x] 单元测试通过

## Commit Message

```
feat(scripts): hermes 支持 discord 和 lark 通道

- 新增 discord 通道：写入 DISCORD_BOT_TOKEN/DISCORD_ALLOWED_USERS 到 ~/.hermes/.env
- 新增 lark 通道：复用 feishu 配置，FEISHU_DOMAIN 设为 lark
- 更新 agent_type.go 白名单，加入 discord 和 lark
- 更新 del_channel_hermes.sh，支持删除 discord 和 lark 通道配置
- 更新 set_channel.sh discord 分支，支持 user_id 参数
- 更新 ai_channel.go ChannelParams，discord 新增 user_id 字段
- 更新 docs/API.md，补充 discord user_id 参数说明
```

## 提交信息

- **Commit hash**: 2ac76948
- **分支**: feat/hermes-discord-support
- **Base**: Release/2026_07_28 (31f4f314)
- **改动文件**: 8 files changed, 70 insertions(+), 8 deletions(-)
- **已 push**: 是（force push，rebase 改写历史）
