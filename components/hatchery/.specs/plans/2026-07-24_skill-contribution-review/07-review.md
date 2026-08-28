# 07. Review — 代码审查

> 基于 [CODEBUDDY.md 红线](../../CODEBUDDY.md) 和安全基线进行逐条审查。

---

## 一、红线检查（13 条）

| # | 红线 | 结果 | 说明 |
|---|------|------|------|
| 1 | 禁止裸 SQL（db.Exec/db.Raw/db.Table/db.Row/db.Rows/db.ScanRows/db.Connection） | ✅ 通过 | 新增代码全部使用 GORM ORM 接口（Create/Where/First/Find/Updates/Delete/Count） |
| 2 | 写接口必须有审计日志（auditRules + WithAudit） | ✅ 通过 | 4 个写接口全部注册：`/openclaw/skills/contribute`、`/openclaw/skills/takedown`、`/admin/contributions/approve`、`/admin/contributions/reject` |
| 3 | 修改 GORM model 必须同步 init.sql + 增量 migration + MigrateFromSQLite | ✅ 通过 | Skill 加 status+uploader_id（init.sql + 0724-skill-contribution-review.sql）；ReviewRequest 新表（init.sql + migration + MigrateFromSQLite + allModels） |
| 4 | 禁止破坏公共 API 兼容性 | ✅ 通过 | 仅新增接口和字段（status/uploader_id 向后兼容），不删除/修改现有字段语义 |
| 5 | handler 中使用 model.DB(r.Context()) 而非 model.DB() | ✅ 通过 | 所有 handler 内 DB 操作使用 `model.DB(r.Context())`；model 层函数使用传入的 ctx |
| 6 | 禁止直接 New 腾讯云 SDK Client | ✅ 通过 | 新增代码不涉及腾讯云 SDK 调用 |
| 7 | 禁止硬编码配置信息 | ✅ 通过 | 无 IP/端口/密钥硬编码 |
| 8 | 异步 goroutine 用 DetachContext | ✅ 通过 | COS 清理 goroutine 不使用 context（纯 COS 操作），不涉及 DB；通知使用同步 `model.CreateNotificationWithCategory(r.Context(), ...)` |
| 9 | 禁止在 master/main 分支开发 | ✅ 通过 | 当前分支 `feature/skill-contribution-review` |
| 10 | 修改 API 必须更新 docs/API.md | ✅ 通过 | 新增「技能共建审核」章节，8 个接口完整文档 |
| 11 | 面向用户文案使用 i18n.T() | ✅ 通过 | 所有 writeError 使用 `hcommon.I18nError(i18n.MsgXxx)`；通知使用 `i18n.T(ctx, i18n.NotifXxx)`；无硬编码中文 |
| 12 | 新增 i18n.Key 必须在 en.go 中添加英文翻译 | ✅ 通过 | 22 个新 Key 全部有英文翻译 |
| 13 | 新增 API 接口集成测试覆盖 | ⏳ 待 IT | 7 个集成测试场景已规划（06-it.md），待 CI 环境执行 |

---

## 二、安全基线检查

| # | 安全规则 | 结果 | 说明 |
|---|---------|------|------|
| 1 | SQLi：值绑定 | ✅ | 所有查询使用 GORM 参数绑定（`Where("slug = ?", slug)`） |
| 2 | RCE：避免命令执行 | ✅ | 新增代码不执行 shell 命令 |
| 3 | AuthZ：权限和归属检查 | ✅ | 员工端 `requireLogin`；管理员端 `requireAdmin`；下架校验 `UploaderID == user.ID` |
| 4 | XSS：转义 | ✅ | JSON 响应（forceJSONMiddleware 自动设置 Accept: application/json） |
| 5 | SSRF | N/A | 新增代码不请求外部域名 |
| 6 | Deserialization | N/A | 新增代码不反序列化外部数据 |
| 7 | Secrets：env-only | ✅ | 无密钥硬编码 |

---

## 三、代码质量检查

| # | 检查项 | 结果 |
|---|--------|------|
| 1 | gofmt 格式化 | ✅ |
| 2 | goimports 导入排序 | ✅ |
| 3 | go vet 静态检查 | ✅（仅预存在 skillhubclient 警告） |
| 4 | go build 编译 | ✅ |
| 5 | 单元测试 21/21 | ✅ PASS |
| 6 | 无未使用的导入/变量 | ✅ |
| 7 | 错误处理完整（所有 tx 操作有 Rollback） | ✅ |
| 8 | 事务一致性（DB 操作 + COS 上传在同一事务内，失败回滚） | ✅ |

---

## 四、多租户合规

| # | 检查项 | 结果 |
|---|--------|------|
| 1 | ReviewRequest 模型有 Identifier 字段 | ✅ |
| 2 | Handler 内使用 model.DB(r.Context()) | ✅ |
| 3 | GORM 回调自动注入 identifier | ✅（Model(&ReviewRequest{}) 保证回调生效） |
| 4 | MigrateFromSQLite 中 ReviewRequest 迁移有 remap | ✅（requester_id / reviewer_id / resource_id） |

---

## 五、审查结论

**通过**。13 条红线中 12 条通过，1 条（集成测试覆盖）待 CI 环境执行。安全基线 7 条全部通过或 N/A。代码质量 8 项全部通过。多租户合规 4 项全部通过。

**无阻塞问题**，可进入 Commit 步骤。
