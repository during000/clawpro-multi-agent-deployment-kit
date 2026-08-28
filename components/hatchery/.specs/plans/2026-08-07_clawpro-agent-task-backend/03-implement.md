# 03. Implement — 实现记录

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./...` 无错误
- [x] 新写接口已添加审计日志并使用 `WithAudit`
- [x] 数据库变更已同步 `sql/init.sql`、增量 migration 与 SQLite→MySQL 迁移
- [x] Handler 使用 `model.DB(r.Context())`
- [x] 未新增密钥、Token 或生产配置
