# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/doctor.go`、`task/doctor_cleanup_test.go` | 非业务行 | 全文件 gofmt 顺带删除既有空行、调整无关测试字段对齐，产生与本修复无关的 diff 噪音 | 低 | 已修：还原无关格式变更 |

未发现高或中严重度问题。

## 审查结论

### 正确性

- `activated_at` 只在会话完成就绪检查、真正切换为 `active` 时写入，并与状态在同一次数据库更新中落库。
- NoFiles 分支优先使用 `activated_at`，存量 NULL/零值回退 `created_at`，不再依赖可被其他业务写入污染的 `updated_at`。
- 有 session 文件时仍只按远端 mtime 判断空闲时间，没有引入全局绝对超时。
- `RefreshDoctorSTS` 使用 `UpdateColumn` 更新单一技术字段，不触发 GORM 自动更新时间；数据库错误会被记录且不会输出误导性的成功日志。
- 激活状态落库失败返回 `false`，保留 `creating` 以供下一轮重试，不会形成“内存成功、数据库未激活”的假成功。
- `time.Time` 比较使用绝对时间，数据库毫秒精度与 12 小时阈值匹配。

### 数据库与兼容性

- GORM model、`sql/init.sql` 和 0730 增量 migration 已同步。
- MySQL 模式不运行 AutoMigrate，发布顺序必须保持 migration 先于应用；文档和任务决策均已记录。
- SQLite 通过现有 `allModels` 自动迁移；`MigrateFromSQLite` 已使用通用 `DoctorSession` 结构迁移，无需额外字段映射。
- 增量字段可空且不回填，避免伪造历史激活时间；MySQL 8.0.46 实测存量行保持 NULL。
- 未新增索引，符合现有按 `status` 查询后逐条判断的访问模式。

### 安全与规范

- 未使用裸 SQL、未拼接用户输入、未新增 SDK Client、未引入硬编码密钥。
- 所有业务数据库调用继续使用传入的 `context.Context`，多租户 identifier 回调仍适用。
- 未新增或修改 HTTP 写接口，不涉及审计规则变更；超时结束继续使用现有审计逻辑。
- API 路由、参数与响应结构未变化，无兼容性破坏。
- 没有新增用户可见错误文案或 i18n key。

### 测试充分性

- 5 个 P0、1 个 P1 和 2 个数据库失败分支均在 race 下通过。
- Doctor 相关 `controller`、`task` 回归集合在审查修正后再次通过 race。
- 本次 20 个生产代码可执行增量行全部覆盖。
- MySQL 增量迁移、全量初始化 schema 和 OpenAPI 生成均已验证。

### 已知非本次阻断项

- 全仓 `go vet ./...` 存在未修改文件
  `skillhubclient/client_test.go:278` 的既有告警。
- 全仓 race 存在未修改的
  `TestCheckAllAgents_SecureFirstBootTriggered_CustomImage` 后台 goroutine 与全局测试 DB cleanup 竞争。
- K8s/CVM 真实 12 小时链路因集群不可达且缺少镜像、AK/SK 未执行。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
- [x] `git diff --check` 通过
- [x] `go vet ./model ./controller ./task` 通过
- [x] Doctor 相关 race 回归通过

**结论：PASS，可进入 Commit 阶段。**
