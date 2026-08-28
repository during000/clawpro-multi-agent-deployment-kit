# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 在龙虾医生章节补充诊断会话自动结束策略：NoFiles 激活时间、存量回退、文件 mtime、STS 刷新语义及定时清理频率 |
| `docs/INDEX.md` | 无需修改 | 未新增独立功能文档 |
| `docs/openapi.json` | 无需修改 | 没有路由、参数或响应结构变化 |
| `sql/init.sql` | Implement 已更新 | 全量 schema 已包含 `doctor_sessions.activated_at` |
| `sql/0730-add-doctor-session-activated-at.sql` | Implement 已新增 | 记录存量 MySQL 的发布迁移；必须先迁移再部署应用 |

## 文档结论

- API 请求参数、响应字段、状态枚举均未改变。
- 用户可感知的 12 小时自动结束语义已写入现有龙虾医生 API 章节。
- 文档明确 STS 刷新不是用户活动，避免后续维护再次将技术字段更新时间作为业务超时依据。
- 数据库字段与迁移说明由 `sql/init.sql` 和增量 migration 维护，不另建重复的 schema 文档。
- `.specs/docs` 是 `docs/` 的软链接，因此无需双份同步。

## 检查项

- [x] `docs/API.md` 已更新（无契约变化，仅补充生命周期行为）
- [x] `.specs/docs/` 相关文档已同步（软链接自动一致）
- [x] 参数表格式符合 CLAUDE.md 要求（本次未新增参数表）
