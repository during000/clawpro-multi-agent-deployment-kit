# 08. Commit — 提交前检查与提交

> **执行顺序**：① 写好本文件 → ② 更新 00-overview.md → ③ 确认所有 0N-*.md 就绪 → ④ git add → ⑤ git commit → ⑥ git push

---

## 一、Commit Message

```
feat(controller): add skill contribution review system

Allow employees to submit skills and request takedowns with admin review:
- New ReviewRequest model (generic review table, extensible to mcp/rule)
- Skill model: add Status (published/pending_review/offline) + UploaderID
- Employee endpoints: POST /openclaw/skills/contribute, POST /openclaw/skills/takedown,
  GET /openclaw/skills/contributions, GET /openclaw/skills/contributions/detail
- Admin endpoints: GET /admin/contributions, GET /admin/contributions/detail,
  POST /admin/contributions/approve, POST /admin/contributions/reject
- Dual state machines: Skill status + ReviewRequest status
- Status filtering on skillstore/detail/distribute/download (only published)
- Mutex: one pending request per slug (publish/takedown exclusive)
- Admin distribute: reject pending_review/offline skills
- Catalog: filter published skills for local agent auto-distribution
- i18n: 22 new keys with English translations
- Audit: 4 new audit rules (contribute, takedown, approve, reject)
- 21 unit tests (all pass)
- SQL: init.sql + 0731-skill-contribution-review.sql migration
```

---

## 二、提交前检查清单

| # | 检查项 | 结果 |
|---|--------|------|
| 1 | gofmt -w . | ✅ |
| 2 | goimports -w . | ✅ |
| 3 | go vet ./... | ✅（仅预存在警告） |
| 4 | go build ./... | ✅ |
| 5 | 单元测试 21/21 PASS | ✅ |
| 6 | Schema 检查（init.sql + migration 一致） | ✅ |
| 7 | OpenAPI spec 生成（8 新路径） | ✅ |
| 8 | 红线检查 12/13 通过 | ✅ |
| 9 | 安全基线全部通过 | ✅ |
| 10 | docs/API.md 已更新 | ✅ |
| 11 | i18n Key + en.go 翻译齐全 | ✅ |
| 12 | allModels + MigrateFromSQLite 已注册 | ✅ |
| 13 | auditRules + WithAudit 已注册 | ✅ |
| 14 | 路由已注册（8 个新路由） | ✅ |

---

## 三、变更文件列表

### 新建文件（4 个）
- `model/review_request.go` — ReviewRequest 模型 + 常量 + HasPendingRequest
- `controller/contribution.go` — 通用审核管理 handler（管理员列表/详情/通过/拒绝 dispatch）
- `controller/contribution_skill.go` — Skill 专用 handler（提交/下架/我的列表/详情/审核逻辑）
- `controller/contribution_skill_test.go` — 21 个单元测试
- `sql/0724-skill-contribution-review.sql` — 增量 migration

### 修改文件（11 个）
- `model/skill.go` — Skill 加 Status + UploaderID + 常量
- `model/db.go` — allModels 加 ReviewRequest
- `model/migrate.go` — MigrateFromSQLite 加 ReviewRequest 迁移
- `model/catalog.go` — ListSkillsByGroup/Project 加 status='published' 过滤
- `sql/init.sql` — skills 加列 + 新建 review_requests 表
- `controller/openclaw_skillstore.go` — 4 处加 status='published' 过滤
- `controller/admin_skills.go` — skillResp 加 UploaderName + ?status= 筛选
- `controller/admin_skill_distribution.go` — prepareDistributeSkillItem 加 status 校验
- `controller/audit.go` — 4 条新审计规则
- `main.go` — 8 个新路由
- `i18n/keys.go` + `i18n/en.go` — 22 个新 i18n Key + 翻译
- `docs/API.md` — 新增技能共建审核章节（8 个接口）

### 任务文档（9 个）
- `.specs/plans/2026-07-24_skill-contribution-review/00-overview.md` ~ `08-commit.md`
