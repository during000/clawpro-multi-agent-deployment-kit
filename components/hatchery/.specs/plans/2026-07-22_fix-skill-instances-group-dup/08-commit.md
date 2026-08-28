# 08. Commit — 提交

---

## Commit Message

```
fix(controller): replace JOIN with subquery in group_id filters to prevent duplicate instances

When filtering instances by multiple group_ids (e.g., group_id=112,119,81),
the JOIN on user_group_members produced duplicate rows for users belonging
to multiple groups.

Fixed 3 occurrences by replacing JOIN with WHERE ... IN (DISTINCT subquery):
- admin_skills.go:        GET  /admin/skills/instances?group_id=...
- admin_mcp_instances.go: GET  /admin/mcp/instances?group_id=...
- admin_skill_distribution.go: POST /admin/skills/instances (public skillset)

Pattern aligned with existing correct implementations in admin_plugins.go
and admin_rules.go.

Added 3 tests:
- TestHandleAdminSkillInstances_GroupIDFilter_MultiGroupNoDuplicate
- TestHandleAdminMcpInstances_GroupIDFilter_MultiGroupNoDuplicate
- TestHandleAdminSkillInstances_PublicSkillset_GroupIDFilter_MultiGroupNoDuplicate
```

## Pre-commit Checklist

- [x] 编译通过 (`go build ./...`)
- [x] 全部测试通过 (skills instances + MCP + public skillset)
- [x] 修改模式与 `admin_plugins.go`、`admin_rules.go` 线上实现一致
- [x] 不需要 API 文档更新（无接口变更）
- [x] 不需要 schema 迁移（无数据库变更）
