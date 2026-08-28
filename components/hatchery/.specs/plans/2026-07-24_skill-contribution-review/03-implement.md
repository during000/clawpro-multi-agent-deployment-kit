# 03. Implement — 实现细节

> 基于 [02-plan.md](./02-plan.md) 的方案设计，记录关键实现细节和与 Plan 的差异。

---

## 一、已完成的改动

### 1. Model 层

| 文件 | 改动 |
|------|------|
| `model/skill.go` | Skill 结构体新增 `Status` + `UploaderID` 字段 + 3 个状态常量。**未修改 `LatestVersionSkillIDs`**（共享函数） |
| `model/review_request.go` | **新建**：ReviewRequest 模型 + 资源类型/操作类型/状态常量 + `HasPendingRequest` 互斥校验函数 |
| `model/db.go` | `allModels` 追加 `&ReviewRequest{}` |
| `model/migrate.go` | `MigrateFromSQLite` 添加 `ReviewRequest` 迁移逻辑（remap requester_id / reviewer_id / resource_id） |
| `model/catalog.go` | `ListSkillsByGroupWithDB` + `ListSkillsByProjectWithDB` 加 `WHERE skills.status = 'published'` |

### 2. SQL

| 文件 | 改动 |
|------|------|
| `sql/init.sql` | skills 表加 `status` + `uploader_id` 列；新建 `review_requests` 表 |
| `sql/0731-skill-contribution-review.sql` | **新建**：增量 migration（ALTER TABLE + CREATE TABLE） |

### 3. Controller

| 文件 | 改动 |
|------|------|
| `controller/contribution.go` | **新建**：管理员端 4 个 handler（列表/详情/通过 dispatch/拒绝 dispatch） |
| `controller/contribution_skill.go` | **新建**：员工端 4 个 handler + Skill 审核逻辑（approve/reject）+ 通知辅助 |
| `controller/openclaw_skillstore.go` | 4 处加 `status='published'`：HandleSkillStore / HandleSkillStoreDetail / HandleSkillStoreDistribute / HandleSkillStoreDownload |
| `controller/admin_skills.go` | HandleAdminSkills 加 `uploader_name` + `?status=` 筛选 + 批量 JOIN users 表 |
| `controller/admin_skill_distribution.go` | `prepareDistributeSkillItem` 加 `skill.Status != published` 校验 |
| `controller/audit.go` | `auditRules` 新增 4 条 |

### 4. 路由 + i18n

| 文件 | 改动 |
|------|------|
| `main.go` | 注册 8 个新路由 |
| `i18n/keys.go` | 新增 12 个消息 Key + 5 个通知标题 Key + 5 个通知消息 Key |
| `i18n/en.go` | 对应 22 个英文翻译 |

---

## 二、关键实现细节

### 2.1 员工提交技能（HandleContributeSkill）

- 复用管理员 `HandleCreateSkill` 的核心逻辑：`validateSkillZip`、`injectMetaIntoZip`、`parseVisibilityParams`、`getStorageClient`、COS 并发上传
- 关键差异：
  - `requireLogin` 替代 `requireAdmin`
  - `skill.Status = pending_review`，`skill.UploaderID = user.ID`
  - 事务内额外创建 `ReviewRequest`（action_type=publish）
- 不触发 `publishAssetVersionForChange`（技能尚未 published）
- 支持 `submit_scan=true` 参数触发安全扫描（与管理端逻辑一致，异步调用 `CreateSkillSecurityScan`）

### 2.2 审核通过（approveSkillContribution）

- publish：`Skill.Status` → `published`（事务内再次校验版本唯一性防并发）
- takedown：`Skill.Status` → `offline`
- 更新 `ReviewRequest`：status=approved, reviewer_id, reviewed_at

### 2.3 审核拒绝（rejectSkillContribution）

- publish：软删除 Skill 记录（技能不会上架）
- takedown：Skill 不变
- 更新 `ReviewRequest`：status=rejected, reviewer_id, reviewed_at, review_comment

### 2.4 status 过滤策略

- **不修改 `LatestVersionSkillIDs`**：该函数被 8+ 处调用（含管理端），改了会影响管理员
- 各用户端调用方自行加 `WHERE status = 'published'`
- 管理员端不加过滤，通过 `?status=` 参数按需筛选
- `model/catalog.go`（local agent 自动下发）加 `WHERE skills.status = 'published'`

### 2.5 不需要 status 过滤的接口

| 接口 | 原因 |
|------|------|
| `HandleSkillStoreInstances` | 查用户实例安装状态，offline 技能用户仍需看到以便卸载 |
| `HandleSkillStoreTasks` | 查下发任务历史，offline 技能仍有历史 |
| `HandleSkillStoreUninstall` | 卸载操作，offline 技能用户仍需能卸载 |
| `HandleSkillsList` / `HandleInstallSkills` | 查运行时/安装记录，不查 skills 表 |

---

## 三、与 Plan 差异

| # | Plan 描述 | 实际实现 | 原因 |
|---|----------|---------|------|
| 1 | `LatestVersionSkillIDs` 加 `status='published'` | **不改**，各调用方加过滤 | 共享函数，改了影响管理端 |
| 2 | `HandleSkillStoreDistribute` 间接过滤 | 直接查询，显式加 `AND status='published'` | 原判断有误 |
| 3 | `HandleSkillStoreDownload` 未提及 | 补上 `AND status='published'` | Plan 遗漏 |
| 4 | `model/catalog.go` 未提及 | 补上 `WHERE skills.status='published'` | Plan 遗漏 |
| 5 | 员工提交后触发安全扫描 | 支持 `submit_scan=true` 参数触发 | 产品要求，与管理端逻辑一致 |
| 6 | 通知管理员用 i18n.T | 使用 `i18n.T(ctx, ...)` | 遵循 i18n 规范 |

### 后续迭代变更

以下在 SOP 完成后根据产品反馈追加：

| # | 变更 | 说明 |
|---|------|------|
| 1 | 撤回功能 | `POST /openclaw/skills/contributions/withdraw`，publish 撤回软删除 Skill，takedown 撤回仅更新 ReviewRequest |
| 2 | 管理员下架/上架 | `POST /admin/skills/offline`（published→offline）、`POST /admin/skills/online`（offline→published），不经过审核流程 |
| 3 | `pending_review` 字段 | `GET /admin/skills` 返回值追加 `pending_review`（含 pending 申请单的 action_type 和 request_id） |
| 4 | `uploader_name` 字段 | `GET /admin/skills` 返回值追加技能上传者姓名 |
| 5 | 贡献列表聚合 | `GET /openclaw/skills/contributions` 按 slug 分组，返回 `skills[]`，每个含 `{slug, name, status, requests[]}` |
| 6 | 贡献列表增强 | 支持 `keyword` 搜索、`status`/`action_type` 过滤、`pending_total` 字段 |
| 7 | 软删除技能名查询 | 申请单关联 Skill 查询使用 `Unscoped()`，避免已删除技能的名称为空 |
| 8 | 管理员下发离线技能 | `prepareDistributeSkillItem` 仅拒绝 `pending_review`，允许 `offline` 技能的下发 |
| 9 | Rebase | 从 `Release/2026_07_24` rebase 到 `Release/2026_07_31`，SQL 文件重命名 `0724`→`0731` |

---

## 四、编译验证

```
go build ./...  → 通过（exit 0）
go vet ./...    → 通过（仅预存在的 skillhubclient/client_test.go 警告，与本次改动无关）
```

---

## 五、待后续步骤处理

- [ ] UT：编写 `controller/contribution_skill_test.go` 单元测试
- [ ] Docs：更新 `docs/API.md` 新增 8 个接口文档
- [ ] IT：集成测试覆盖
