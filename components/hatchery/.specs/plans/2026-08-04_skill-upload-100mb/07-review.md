# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review（对照 `.specs/review.md`）
- [ ] 人工 Review

## 变更范围（业务）

| 文件 | 类型 |
|------|------|
| `controller/skill_upload.go` | 常量 100MB + `isSkillUploadTooLarge` + 新 i18n Key |
| `controller/skill_upload_test.go` | **新增（尚未 git add）** |
| `controller/admin_skills_test.go` | 注释同步 |
| `i18n/keys.go` / `i18n/en.go` | 文案拆分 |
| `docs/API.md` | 两处上限说明 |

## 维度结论

| 维度 | 结论 |
|------|------|
| 正确性 | 通过。`>` 边界（等于上限允许）合理；create/contribute 共用常量一致；上传与 Bundle 文案已拆分，避免误报 |
| 安全性 | 通过红线检查。放宽上限增大单次内存占用（已知、Clarify 接受）；无注入/权限/裸 SQL 变更 |
| 规范性 | 通过。无新写接口故审计不变；无 DB 变更；API 文档已更新；新 i18n Key 已有英译；无硬编码用户文案 |
| 可维护性 | 通过。`isSkillUploadTooLarge` 虽为一行包装，但为 Plan 约定的可测边界，可接受 |
| 性能 | 可接受。整包入内存既有模式，峰值约翻倍；本期不改流式 |

## 红线检查（`.specs/review.md`）

| # | 红线 | 结果 |
|---|------|------|
| 1 | 裸 SQL | 未触发 |
| 2 | 写接口无审计 | 无新写接口 |
| 3 | model 未同步 sql | 无 model 变更 |
| 4 | 破坏公共 API | 仅放宽上限，非破坏 |
| 5 | 硬编码密钥 | 未触发 |
| 6 | 裸 `model.DB` | 未改 DB 访问 |
| 7 | 硬编码中文用户文案 | 使用 i18n |
| 8 | 新 Key 无英译 | 已注册 |
| 9 | 新 API/参数无 IT | 无新接口/参数；IT 环境 BLOCKED 已记录 |

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/skill_upload.go` | 127 | 超限返回分支 UT 未覆盖（增量 80%） | 低 | 忽略（Plan 接受；边界函数已覆盖判定） |
| 2 | （部署侧） | - | 网关 body 限制可能仍 ≤50MB | 中 | 忽略（用户确认文档不加备注；部署侧自行确认） |
| 3 | `controller/skill_upload_test.go` | - | 新文件尚未 `git add`，Commit 时必须纳入 | 中 | **Commit 阶段处理** |

无高严重度问题；无必须在 Review 阶段改代码的项。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
- [x] **结论：PASS**，可进入 Commit
