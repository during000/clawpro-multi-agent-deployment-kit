# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `model/agent_type_new_test.go` | 253 | 合并后 wantAny 缺少 `discord`，测试失败 | 高 | 已修 |
| 2 | `model/agent_type_extended_test.go` | 386 | hermes 通道数断言未更新（7→9），测试失败 | 高 | 已修 |
| 3 | `scripts/del_channel_hermes.sh` | 113-121 | lark 分支原始代码误用 `update_env` 而非 `delete_env`，且缺少 `ENV_FILE` 定义 | 高 | 已修 |

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致（保持 Release 分支的 4 空格缩进风格）
- [x] 安全基线检查通过（无裸 SQL、无硬编码密钥、无敏感信息泄露）
