# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | controller/local_agent.go | 683 | `HandleLocalAgentSync` 拍 `last_report_at` 用 `model.DB(r.Context())`（带租户隔离），非裸 SQL，多租户 OK | 低 | 已确认无问题 |
| 2 | controller/local_agent.go | 1087 | ack failed 分支 `Where("id = ? AND current_operation = ?")` 乐观锁依赖内存 `inst` 对象，但 report 不碰 current_operation，竞态安全 | 低 | 已确认无问题 |
| 3 | i18n/keys.go | - | 新增 4 个 hook 校验 key（MsgRuleTypeRequired 等）均在 en.go 有英文翻译 | 低 | 已确认无问题 |
| 4 | controller/admin_instances.go | 4496 | `HandleAdminLocalAgentRemove` 事务包裹（调用方传 tx），与用户端对称 | 低 | 已确认无问题 |

## 红线自查（对照 MEMORY 项目红线）

| 红线 | 结果 |
|------|------|
| 1. 禁止裸 SQL（必须用 GORM） | ✅ 无 `db.Exec/Raw/Table` 命中 |
| 2. 写接口注册 auditRules + WithAudit | ✅ `/admin/local-agent/remove` 经管控写接口路径（见 02-plan F6） |
| 3. 改 GORM model 同步 sql/init.sql + migration | ✅ `LocalAgentOpUninstall` 为纯常量，无 schema 变更；三期 migration 已合并 |
| 4. handler 用 `model.DB(r.Context())` | ✅ 删除/查询均用带租户隔离句柄 |
| 5. 腾讯云 SDK Client 用工厂 | N/A 本次无云 API 调用 |
| 6. 面向用户文案用 i18n.Key | ✅ 错误响应均走 `i18n.MsgXxx` |
| 7. 新增 i18n.Key 在 en.go 加英文 | ✅ 4 个新 key 均有翻译 |
| 8. 异步 goroutine 用 `DetachContext` | N/A 本次无新增异步 goroutine |
| 9. 改 API 更新 docs/API.md | ✅ remove 双端/sync cmds/ack 枚举已更新 |
| 10/13. 集成测试覆盖 | ⚠️ 本地无 Docker/K8s 环境，受阻（见 06-it.md） |

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致（gofmt 已格式化，go vet 通过）
- [x] 安全基线检查通过（多租户隔离、i18n、事务原子性均合规）

## 待人工 Review 项

- `feature/local_agent3_1` → `Release/2026_07_28` 的 MR 需人工 review（重点：卸载中间态语义、四表清理、report 心跳竞态修复）
- 真环境集成测试（06 IT）需有 Docker/K8s/AK-SK 资源后运行
