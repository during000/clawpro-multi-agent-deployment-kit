# 07. Review — 代码审查（TAPD 160256132）

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

审查依据：`.specs/review.md`、`CLAUDE.md`、`.codebuddy/rules/code.md`、`docs/testing.md`。

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/admin_skills.go` | ZIP 文件名规范化 | 在转码前替换 `0x5c` 会破坏部分 GB18030 多字节字符 | 中 | 已修 |
| 2 | `controller/admin_skills.go` | ZIP Header | 合法 UTF-8 但缺失 flag 的 ZIP 不能按 `NonUTF8` 强制二次转码 | 中 | 已修 |
| 3 | `controller/admin_skills.go` | `HandleCreateSkill` | 首次返回的 `files` 立即被最终 ZIP 文件列表覆盖 | 低 | 已修 |
| 4 | `skillhubclient/client_test.go` | `TestReadBody` | 未检查 `http.Get` 等错误，阻塞全仓 vet | 低 | 已修 |
| 5 | IT 环境 | N/A | 无法访问 K8s 且缺少指定 AK/SK | 中 | 已记录，需环境补测 |
| 6 | `controller/admin_skills.go` | 文件名编码策略 | ZIP 不记录具体本地编码，按 GB18030 自动转码可能把 Big5、Shift-JIS 等输入静默改成错误名称 | 高 | 已修：所有非 UTF-8 文件名统一拒绝并明确提示 |

## SOP 维度结论

- 正确性：拒绝多种旧编码、保留合法 UTF-8、修复 flag 的路径均有测试；锚点和文件列表行为保持。
- 安全性：不新增 SQL/SDK 调用；编码校验在所有路径处理前执行，Zip Slip、特殊字符、大小限制继续执行。
- 规范性：审计路由既有；i18n、API 文档同步；无 DB/API Schema 变更。
- 可维护性：直接依据 UTF-8 有效性判断，不维护不完整的编码猜测列表。
- 性能：每个 ZIP 条目增加一次线性字节校验，无额外网络/DB 调用。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
- [x] `git diff --check` 通过
- [x] `go vet ./...` 通过
