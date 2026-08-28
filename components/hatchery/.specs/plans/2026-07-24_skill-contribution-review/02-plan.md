# 02. Plan — 方案设计

> 基于 [01-clarify.md](./01-clarify.md) 确认的需求和方案，细化改动文件、调用链、测试用例和风险。

---

## 一、改动文件清单

| # | 文件 | 操作 | 说明 |
|---|------|------|------|
| 1 | `model/skill.go` | 改 | Skill 结构体加 `Status` + `UploaderID` 字段 + 状态常量。**不改 `LatestVersionSkillIDs`**（共享函数，改了会影响管理员端） |
| 2 | `model/review_request.go` | **新建** | `ReviewRequest` 模型 + 资源类型/操作类型/状态常量 |
| 3 | `model/db.go` | 改 | `allModels` 追加 `&ReviewRequest{}` |
| 4 | `model/migrate.go` | 改 | `MigrateFromSQLite` 添加 `review_requests` 表迁移 + `skills` 表新字段迁移 |
| 5 | `sql/init.sql` | 改 | `skills` 表加 `status` + `uploader_id` 列；新建 `review_requests` 表 |
| 6 | `sql/0724-skill-contribution-review.sql` | **新建** | 增量 migration |
| 7 | `controller/contribution.go` | **新建** | 通用：管理员申请列表、详情、审核通过 dispatch、审核拒绝 dispatch |
| 8 | `controller/contribution_skill.go` | **新建** | Skill 专用：员工提交技能、申请下架、审核通过逻辑、审核拒绝逻辑、员工申请列表、申请详情 |
| 9 | `controller/openclaw_skillstore.go` | 改 | `HandleSkillStore` + `HandleSkillStoreDetail` + `HandleSkillStoreDistribute` + `HandleSkillStoreDownload` 加 `status='published'` 过滤 |
| 10 | `controller/openclaw_skill.go` | **不改** | `HandleSkillsList` 查 agent 运行时已安装技能，不查 skills 表，无需 status 过滤 |
| 11 | `controller/admin_skills.go` | 改 | `HandleAdminSkills` 响应加 `uploader_name`；查询支持 `?status=` 筛选 |
| 12 | `controller/audit.go` | 改 | `auditRules` 新增 4 条审计规则 |
| 13 | `main.go` | 改 | 注册 8 个新路由 |
| 14 | `i18n/keys.go` | 改 | 新增 ~20 个 i18n Key |
| 15 | `i18n/en.go` | 改 | 对应英文翻译 |
| 16 | `docs/API.md` | 改 | 新增 7 个接口文档 |
| 17 | `controller/contribution_skill_test.go` | **新建** | 单元测试 |

---

## 二、数据模型定义

### 2.1 Skill 模型改动（`model/skill.go`）

```go
// Skill 结构体新增字段（在 DistributeCount 后面）
type Skill struct {
    // ... 既有字段 ...
    DistributeCount int    `gorm:"not null;default:0" json:"distribute_count"`
    Status          string `gorm:"not null;default:'published'" json:"status"`       // published | pending_review | offline
    UploaderID      uint   `gorm:"not null;default:0" json:"uploader_id"`           // 0=admin, >0=员工 user_id
}
```

`LatestVersionSkillIDs` **不修改**——该函数被 8+ 处调用（含管理员端 `admin_assets.go`、`admin_project_config.go`、`admin_skills.go` 和 `model/catalog.go`），加 status 过滤会导致管理员看不到 pending_review/offline 技能。改为在各用户端调用方加过滤。

新增常量：

```go
const (
    SkillStatusPublished     = "published"
    SkillStatusPendingReview = "pending_review"
    SkillStatusOffline       = "offline"
)
```

### 2.2 ReviewRequest 模型（`model/review_request.go`）

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

// 资源类型常量
const (
    ResourceTypeSkill = "skill"
    ResourceTypeMcp   = "mcp"   // 未来扩展
    ResourceTypeRule  = "rule"  // 未来扩展
)

// 操作类型常量
const (
    ActionTypePublish  = "publish"
    ActionTypeTakedown = "takedown"
)

// 审批状态常量
const (
    ReviewStatusPending  = "pending"
    ReviewStatusApproved = "approved"
    ReviewStatusRejected = "rejected"
)

type ReviewRequest struct {
    ID            uint           `gorm:"primarykey" json:"id"`
    Identifier    string         `gorm:"index;default:''" json:"-"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
    RequesterID   uint           `gorm:"not null;default:0" json:"requester_id"`
    ResourceType  string         `gorm:"type:varchar(32);not null;default:'skill'" json:"resource_type"`
    ResourceID    uint           `gorm:"not null;default:0" json:"resource_id"`
    ActionType    string         `gorm:"type:varchar(16);not null;default:'publish'" json:"action_type"`
    Slug          string         `gorm:"type:varchar(191);not null;default:''" json:"slug"`
    Status        string         `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
    Reason        string         `gorm:"type:text" json:"reason"`
    ReviewerID    uint           `gorm:"not null;default:0" json:"reviewer_id"`
    ReviewedAt    *time.Time     `gorm:"default:null" json:"reviewed_at"`
    ReviewComment string         `gorm:"type:text" json:"review_comment"`
}

// HasPendingRequest 检查指定 slug 是否有进行中的申请（互斥校验）
func HasPendingRequest(ctx context.Context, resourceType, slug string) bool {
    var count int64
    DB(ctx).Model(&ReviewRequest{}).
        Where("resource_type = ? AND slug = ? AND status = ?", resourceType, slug, ReviewStatusPending).
        Count(&count)
    return count > 0
}
```

### 2.3 SQL DDL

#### `sql/init.sql` — skills 表新增列

```sql
-- 在 distribute_count 后追加
`status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'published',
`uploader_id` bigint unsigned NOT NULL DEFAULT '0',
```

#### `sql/init.sql` — 新建 review_requests 表

```sql
CREATE TABLE IF NOT EXISTS `review_requests` (
  `id`             bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`     varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`     datetime(3) DEFAULT NULL,
  `updated_at`     datetime(3) DEFAULT NULL,
  `deleted_at`     datetime(3) DEFAULT NULL,
  `requester_id`   bigint unsigned NOT NULL DEFAULT '0',
  `resource_type`  varchar(32) NOT NULL DEFAULT 'skill',
  `resource_id`    bigint unsigned NOT NULL DEFAULT '0',
  `action_type`    varchar(16) NOT NULL DEFAULT 'publish',
  `slug`           varchar(191) NOT NULL DEFAULT '',
  `status`         varchar(16) NOT NULL DEFAULT 'pending',
  `reason`         text,
  `reviewer_id`    bigint unsigned NOT NULL DEFAULT '0',
  `reviewed_at`    datetime(3) DEFAULT NULL,
  `review_comment` text,
  PRIMARY KEY (`id`),
  KEY `idx_rr_requester` (`identifier`,`requester_id`),
  KEY `idx_rr_resource` (`identifier`,`resource_type`,`resource_id`),
  KEY `idx_rr_status` (`identifier`,`status`),
  KEY `idx_rr_slug_mutex` (`identifier`,`resource_type`,`slug`,`status`),
  KEY `idx_review_requests_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### `sql/0724-skill-contribution-review.sql` — 增量 migration

```sql
-- 0724-skill-contribution-review.sql
-- 技能共建审核：Skill 加 status + uploader_id；新建 review_requests 通用审批表

ALTER TABLE `skills`
  ADD COLUMN `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'published' AFTER `distribute_count`,
  ADD COLUMN `uploader_id` bigint unsigned NOT NULL DEFAULT '0' AFTER `status`;

CREATE TABLE IF NOT EXISTS `review_requests` (
  -- 同上 init.sql 中的定义
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## 三、API 接口设计

### 3.1 员工端接口

#### `POST /openclaw/skills/contribute` — 提交技能/新版本

- **守卫**：`requireLogin` + `requireSMHEnabled`
- **Content-Type**：`multipart/form-data`
- **参数**：与 `/admin/skills/create` 完全一致（slug/name/version/description/changelog/file/category_ids/visibility_type/group_ids/project_ids）
- **额外校验**：
  - `HasPendingRequest(ctx, "skill", slug)` → 互斥拒绝
  - `(slug, version)` 唯一性校验（含 pending_review 状态的 Skill）
  - 版本递增校验（新版本 > 同 slug 最高版本）
- **流程**：
  1. 互斥校验
  2. 复用 `validateSkillZip` 校验 zip
  3. 复用 `injectMetaIntoZip` 注入 ownerId
  4. 上传 COS（复用现有 SMH 上传逻辑）
  5. 事务内：创建 Skill（status=pending_review, uploader_id=user.ID）+ 处理分类关联 + 可见范围
  6. 事务内：创建 ReviewRequest（action_type=publish, status=pending, resource_id=Skill.ID, slug=Skill.Slug）
  7. 通知管理员
- **响应**：`{ ok: true, skill_id: N, request_id: N }`

#### `POST /openclaw/skills/takedown` — 申请下架

- **守卫**：`requireLogin`（纯 DB 操作，不需要 `requireSMHEnabled`）
- **Content-Type**：`application/json`
- **参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 要下架的技能 slug |
| reason | string | 是 | 下架理由 |

- **校验**：
  - Skill 存在且 `status='published'`
  - `skill.UploaderID == user.ID`（只能下架自己上传的）
  - `HasPendingRequest(ctx, "skill", slug)` → 互斥拒绝
- **流程**：创建 ReviewRequest（action_type=takedown, status=pending, resource_id=Skill.ID, slug=Skill.Slug, reason=reason）
- **响应**：`{ ok: true, request_id: N }`

#### `GET /openclaw/skills/contributions` — 我的申请列表

- **守卫**：`requireLogin`
- **查询参数**：`status`（可选筛选）、`action_type`（可选筛选）、分页

- **响应**：

```json
{
  "requests": [
    {
      "id": 1,
      "resource_type": "skill",
      "action_type": "publish",
      "slug": "my-skill",
      "status": "pending",
      "reason": "",
      "reviewer_id": 0,
      "reviewed_at": null,
      "review_comment": "",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### `GET /openclaw/skills/contributions/detail` — 申请详情

- **守卫**：`requireLogin`
- **查询参数**：`id`（申请 ID）
- **校验**：申请人本人或管理员可查看
- **响应**：申请详情 + 关联 Skill 信息

### 3.2 管理员端接口

#### `GET /admin/contributions` — 所有申请列表

- **守卫**：`requireAdmin`
- **查询参数**：`resource_type`、`action_type`、`status`、分页
- **响应**：申请列表（含申请人 username）

#### `GET /admin/contributions/detail` — 申请详情

- **守卫**：`requireAdmin`
- **查询参数**：`id`

#### `POST /admin/contributions/approve` — 审核通过

- **守卫**：`requireAdmin` + `WithAudit`
- **Content-Type**：`application/json`
- **参数**：`{ "id": N }`
- **流程**（switch resource_type dispatch）：
  - **skill + publish**：`Skill.Status = published`，`ReviewRequest.Status = approved`
  - **skill + takedown**：`Skill.Status = offline`，`ReviewRequest.Status = approved`
  - 通知申请人
- **响应**：`{ ok: true }`

#### `POST /admin/contributions/reject` — 审核拒绝

- **守卫**：`requireAdmin` + `WithAudit`
- **Content-Type**：`application/json`
- **参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |
| review_comment | string | 是 | 拒绝理由 |

- **流程**（switch resource_type dispatch）：
  - **skill + publish**：软删除 Skill，`ReviewRequest.Status = rejected`
  - **skill + takedown**：`ReviewRequest.Status = rejected`（Skill 不变）
  - 通知申请人
- **响应**：`{ ok: true }`

### 3.3 审计规则（`controller/audit.go`）

```go
"/openclaw/skills/contribute":       {"skill_contribute", "review_request"},
"/openclaw/skills/takedown":         {"skill_takedown_request", "review_request"},
"/admin/contributions/approve":      {"review_approve", "review_request"},
"/admin/contributions/reject":       {"review_reject", "review_request"},
```

### 3.4 路由注册（`main.go`）

```go
// 员工端 — 技能共建
http.HandleFunc("/openclaw/skills/contribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleContributeSkill)))
http.HandleFunc("/openclaw/skills/takedown", controller.WithAudit(controller.WithOpenAPI(controller.HandleTakedownSkill)))
http.HandleFunc("/openclaw/skills/contributions", controller.WithOpenAPI(controller.HandleMyContributions))
http.HandleFunc("/openclaw/skills/contributions/detail", controller.WithOpenAPI(controller.HandleMyContributionDetail))

// 管理员端 — 审核管理
http.HandleFunc("/admin/contributions", controller.WithOpenAPI(controller.HandleAdminContributions))
http.HandleFunc("/admin/contributions/detail", controller.WithOpenAPI(controller.HandleAdminContributionDetail))
http.HandleFunc("/admin/contributions/approve", controller.WithAudit(controller.WithOpenAPI(controller.HandleApproveContribution)))
http.HandleFunc("/admin/contributions/reject", controller.WithAudit(controller.WithOpenAPI(controller.HandleRejectContribution)))
```

---

## 四、调用链

### 4.1 发布流程调用链

```
HandleContributeSkill (contribution_skill.go)
  ├─ requireLogin → 获取 *model.User
  ├─ requireSMHEnabled
  ├─ ParseMultipartForm (50MB)
  ├─ 校验 slug/name/version 非空 + isValidSlug
  ├─ skill.ParseVersion()
  ├─ model.HasPendingRequest(ctx, "skill", slug) → 互斥校验
  ├─ (slug, version) 唯一性校验（含 pending_review Skill）
  ├─ 版本递增校验（> 同 slug 最高版本）
  ├─ validateSkillZip(zipData, slug) → 复用 admin_skills.go 现有函数
  ├─ injectMetaIntoZip(zipData, slug, {ownerId, slug, version}) → 复用现有函数
  ├─ SMH 上传 zip → COS 正式路径 {slug}/{slug}-{version}.zip
  ├─ tx.Begin()
  │   ├─ tx.Create(&skill)  // status=pending_review, uploader_id=user.ID
  │   ├─ 处理 category_ids → tx.Create(&SkillCategoryMapping{...})
  │   ├─ parseVisibilityParams → model.SetSkillVisibility(tx, ...)
  │   ├─ model.ReplaceResourceProjectBindings(tx, ...)
  │   ├─ tx.Create(&ReviewRequest{resource_type="skill", action_type="publish", ...})
  │   └─ tx.Commit()
  ├─ 通知管理员（CreateNotificationWithCategory）
  └─ jsonOK
```

### 4.2 审核通过调用链（publish）

```
HandleApproveContribution (contribution.go)
  ├─ requireAdmin
  ├─ 查询 ReviewRequest by ID
  ├─ 校验 request.Status == "pending"
  ├─ switch request.ResourceType:
  │   └─ case "skill":
  │       └─ approveSkillContribution(request) (contribution_skill.go)
  │           ├─ switch request.ActionType:
  │           │   └─ case "publish":
  │           │       ├─ 查询 Skill by request.ResourceID
  │           │       ├─ 再次校验 (slug, version) 唯一性（防并发）
  │           │       ├─ tx: Skill.Status = "published"
  │           │       └─ tx: ReviewRequest.Status = "approved", ReviewerID, ReviewedAt
  │           │   └─ case "takedown":
  │           │       ├─ 查询 Skill by request.ResourceID
  │           │       ├─ tx: Skill.Status = "offline"
  │           │       └─ tx: ReviewRequest.Status = "approved", ReviewerID, ReviewedAt
  │           └─ tx.Commit()
  ├─ 通知申请人
  └─ jsonOK
```

### 4.3 审核拒绝调用链（publish）

```
HandleRejectContribution (contribution.go)
  ├─ requireAdmin
  ├─ 查询 ReviewRequest by ID
  ├─ 校验 request.Status == "pending"
  ├─ switch request.ResourceType:
  │   └─ case "skill":
  │       └─ rejectSkillContribution(request) (contribution_skill.go)
  │           ├─ switch request.ActionType:
  │           │   └─ case "publish":
  │           │       ├─ 查询 Skill by request.ResourceID
  │           │       ├─ tx: 软删除 Skill
  │           │       └─ tx: ReviewRequest.Status = "rejected", ReviewerID, ReviewedAt, ReviewComment
  │           │   └─ case "takedown":
  │           │       └─ tx: ReviewRequest.Status = "rejected", ReviewerID, ReviewedAt, ReviewComment
  │           └─ tx.Commit()
  ├─ 通知申请人
  └─ jsonOK
```

---

## 五、现有查询改动

### 5.1 需要加 `status='published'` 过滤的查询（用户端）

| # | 接口 | 文件:行 | 查询方式 | 改动 |
|---|------|---------|---------|------|
| 1 | `HandleSkillStore` | `openclaw_skillstore.go:34` | `LatestVersionSkillIDs` 子查询 | 调用方加 `.Where("status = ?", model.SkillStatusPublished)` |
| 2 | `HandleSkillStoreDetail` | `openclaw_skillstore.go:254,279` | 直接 `WHERE slug=?` | 加 `AND status = 'published'`（最新版本 + 全版本列表两处） |
| 3 | `HandleSkillStoreDistribute` | `openclaw_skillstore.go:780` | 直接 `WHERE slug=?` | 加 `AND status = 'published'`（**原 Plan 误写"间接过滤"，实际不经 LatestVersionSkillIDs**） |
| 4 | `HandleSkillStoreDownload` | `openclaw_skillstore.go:985` | 直接 `WHERE slug=?` | 加 `AND status = 'published'`（**原 Plan 遗漏**） |

### 5.2 不需要改动的用户端查询

| 接口 | 原因 |
|------|------|
| `GET /openclaw/skills`（`HandleSkillsList`） | 查 agent 运行时已安装技能（TAT/本地 agent 上报），不查 skills 表 |
| `GET /openclaw/install-skills`（`HandleInstallSkills`） | 查 `skill_installations` 表（实例安装记录），不查 skills 表 status |
| `HandleSkillStoreInstances`（`openclaw_skillstore.go:421`） | 查用户实例安装状态，offline 技能用户仍需看到安装状态以便卸载 |
| `HandleSkillStoreTasks`（`openclaw_skillstore.go:578`） | 查下发任务历史，offline 技能仍有历史记录 |
| `HandleSkillStoreUninstall`（`openclaw_skillstore.go:1061`） | 卸载操作，offline 技能用户仍需能卸载 |

### 5.3 需要加 `status='published'` 校验的管理员端查询

| # | 接口 | 文件:行 | 说明 |
|---|------|---------|------|
| 1 | `HandleDistributeSkill` | `admin_skills.go:2295` | 下发前校验 `status='published'`，不允许下发 pending_review/offline |
| 2 | `admin_skill_distribution.go` | `admin_skill_distribution.go:204` | 同上，批量下发路径 |

### 5.4 需要加 `status='published'` 过滤的 model 层查询

| # | 函数 | 文件 | 说明 |
|---|------|------|------|
| 1 | `ListSkillsByGroupWithDB` | `model/catalog.go:33` | local agent 自动下发用，加 `WHERE skills.status = 'published'` |
| 2 | `ListSkillsByProjectWithDB` | `model/catalog.go:50` | 同上 |

### 5.5 管理员技能列表 `HandleAdminSkills`（`admin_skills.go:418`）

**不**加 status 过滤（管理员看到所有状态），但：
- `skillResp` 加 `UploaderName` 字段
- 批量 JOIN users 表填充 `uploader_name`
- 支持 `?status=` 查询参数筛选

### 5.6 不修改 `LatestVersionSkillIDs`

`LatestVersionSkillIDs`（`model/skill.go:156`）被以下调用方共享：

| 调用方 | 文件 | 用户端/管理端 |
|--------|------|-------------|
| `HandleSkillStore` | `openclaw_skillstore.go:34` | 用户端 |
| `HandleAdminSkills` | `admin_skills.go:418` | 管理端 |
| `admin_assets.go` | `admin_assets.go:87,160,408` | 管理端 |
| `admin_project_config.go` | `admin_project_config.go:92` | 管理端 |
| `ListSkillsByGroupWithDB` | `model/catalog.go:33` | 内部（自动下发） |
| `ListSkillsByProjectWithDB` | `model/catalog.go:50` | 内部（自动下发） |

如果修改此函数加 `status='published'`，管理端也看不到 pending_review/offline 技能。**改为在各调用方按需加过滤。**

---

## 六、i18n Key 清单

```go
// 技能共建审核
MsgSkillContributeSuccess          = Key{string: "技能提交成功，等待管理员审核"}
MsgSkillContributePendingExists    = Key{string: "该技能已有进行中的申请，请等待审核完成"}
MsgSkillTakedownSuccess            = Key{string: "下架申请已提交，等待管理员审核"}
MsgSkillTakedownNotOwner           = Key{string: "只能下架自己上传的技能"}
MsgSkillTakedownReasonRequired     = Key{string: "下架理由不能为空"}
MsgReviewRequestNotFound           = Key{string: "审核申请不存在"}
MsgReviewRequestNotPending         = Key{string: "该申请已审核，无法重复操作"}
MsgReviewRejectCommentRequired     = Key{string: "拒绝理由不能为空"}
MsgReviewApproveSuccess            = Key{string: "审核通过"}
MsgReviewRejectSuccess             = Key{string: "已拒绝"}
MsgReviewSkillNotExist             = Key{string: "关联的技能不存在"}
MsgReviewNotOwner                  = Key{string: "无权查看此申请"}

// 通知标题
NotifTitleSkillReviewApproved      = Key{string: "技能审核通过"}
NotifTitleSkillReviewRejected      = Key{string: "技能审核未通过"}
NotifTitleSkillTakedownApproved    = Key{string: "技能下架通过"}
NotifTitleSkillTakedownRejected    = Key{string: "技能下架未通过"}
NotifTitleNewReviewRequest         = Key{string: "新的技能审核申请"}

// 通知消息
NotifMsgSkillReviewApproved        = Key{string: "您提交的技能「%s」已通过审核，已上架到企业技能库"}
NotifMsgSkillReviewRejected        = Key{string: "您提交的技能「%s」未通过审核，原因：%s"}
NotifMsgSkillTakedownApproved      = Key{string: "您申请下架的技能「%s」已通过审核，已下架"}
NotifMsgSkillTakedownRejected      = Key{string: "您申请下架的技能「%s」未通过审核，原因：%s"}
NotifMsgNewReviewRequest           = Key{string: "员工 %s 提交了技能「%s」的%s申请，请前往审核"}
```

---

## 七、测试用例设计（自然语言描述）

### 7.1 发布流程

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| P1 | 员工提交新技能成功 | 合法 zip + 元数据，slug 不存在 | Skill 创建（pending_review），ReviewRequest 创建（pending），返回 200 |
| P2 | 员工提交新版本成功 | 同 slug 更高版本号 | 新 Skill 记录创建（pending_review），旧版本保持 published |
| P3 | 互斥校验——slug 已有 pending 申请 | slug 存在 pending ReviewRequest | 返回 400，提示已有进行中的申请 |
| P4 | 版本号重复 | (slug, version) 已存在 Skill | 返回 400，提示版本已存在 |
| P5 | 版本号未递增 | 新版本 <= 同 slug 最高版本 | 返回 400，提示版本号必须大于现有版本 |
| P6 | 未登录提交 | 无 session | 返回 401 |
| P7 | zip 格式不合法 | 缺少 SKILL.md | 返回 400 |
| P8 | SMH 未开启 | requireSMHEnabled 失败 | 返回 403 |

### 7.2 下架流程

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| T1 | 员工申请下架自己的技能成功 | slug 存在，uploader_id 匹配 | ReviewRequest 创建（pending），返回 200 |
| T2 | 下架他人技能 | slug 存在但 uploader_id 不匹配 | 返回 403 |
| T3 | 下架管理员上传的技能 | uploader_id=0 | 返回 403 |
| T4 | 下架不存在的技能 | slug 不存在 | 返回 404 |
| T5 | 缺少 reason | reason 为空 | 返回 400 |
| T6 | 互斥校验 | slug 已有 pending 申请 | 返回 400 |
| T7 | 下架 pending_review 状态技能 | skill.status=pending_review | 返回 400 |

### 7.3 审核流程

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| A1 | 管理员通过 publish 申请 | request.status=pending | Skill.status→published，request.status→approved，通知发送 |
| A2 | 管理员拒绝 publish 申请 | request.status=pending + review_comment | Skill 软删除，request.status→rejected，通知发送 |
| A3 | 管理员通过 takedown 申请 | request.status=pending | Skill.status→offline，request.status→approved，通知发送 |
| A4 | 管理员拒绝 takedown 申请 | request.status=pending + review_comment | request.status→rejected，Skill 不变，通知发送 |
| A5 | 重复审核 | request.status=approved | 返回 400，提示已审核 |
| A6 | 拒绝缺少 comment | review_comment 为空 | 返回 400 |
| A7 | 非管理员审核 | role=user | 返回 403 |
| A8 | 申请不存在 | id 不存在 | 返回 404 |
| A9 | 并发——通过时版本冲突 | Skill (slug, version) 被管理员抢先上传 | 审核失败，返回 409 |

### 7.4 查询接口

| # | 场景 | 输入 | 预期输出 |
|---|------|------|---------|
| Q1 | 员工查看自己的申请列表 | requester_id 匹配 | 只返回自己的申请 |
| Q2 | 员工查看他人申请详情 | 非本人且非管理员 | 返回 403 |
| Q3 | 管理员查看所有申请 | 无筛选 | 返回所有申请（含申请人 username） |
| Q4 | 管理员按 status 筛选 | status=pending | 只返回 pending 申请 |
| Q5 | 管理员技能列表含 status | 无筛选 | 返回的技能包含 status 和 uploader_name 字段 |
| Q6 | 技能广场不显示 pending_review | — | skillstore 只返回 published 技能 |
| Q7 | 技能广场不显示 offline | — | skillstore 只返回 published 技能 |
| Q8 | 技能详情不显示 pending_review | slug 对应 pending_review 技能 | `HandleSkillStoreDetail` 返回 404 |
| Q9 | 技能下载不允许 pending_review | slug 对应 pending_review 技能 | `HandleSkillStoreDownload` 返回 404 |
| Q10 | 技能下发不允许 offline | slug 对应 offline 技能 | `HandleSkillStoreDistribute` 返回 400 |
| Q11 | 管理员下发不允许 pending_review | slug 对应 pending_review 技能 | `HandleDistributeSkill` 返回 400 |
| Q12 | 用户可查看 offline 技能的安装状态 | slug 对应 offline 技能 | `HandleSkillStoreInstances` 正常返回 |
| Q13 | 用户可卸载 offline 技能 | slug 对应 offline 技能 | `HandleSkillStoreUninstall` 正常执行 |
| Q14 | local agent 自动下发不选 offline 技能 | catalog 查询 | `ListSkillsByGroupWithDB` 只返回 published |

### 7.5 状态机一致性

| # | 场景 | 预期 |
|---|------|------|
| S1 | 拒绝后重新提交同 slug | 旧 request=rejected 不阻塞，新 request=pending 可创建 |
| S2 | 通过 publish 后申请 takedown | Skill=published → 可创建 takedown request |
| S3 | takedown 通过后重新提交同 slug | Skill=offline，新 publish 需用更高版本号 |
| S4 | 管理员直接删除 pending_review 的 Skill | 软删除成功（管理员不受限） |

---

## 八、风险评估

| # | 风险 | 严重度 | 缓解措施 |
|---|------|--------|---------|
| R1 | `LatestVersionSkillIDs` 不修改，pending_review 技能可能被 `model/catalog.go` 的自动下发逻辑选中 | 中 | `ListSkillsByGroupWithDB` / `ListSkillsByProjectWithDB` 显式加 `WHERE skills.status = 'published'` |
| R2 | 员工提交后 COS 文件已上传，但审核拒绝后 Skill 软删除，COS 文件成为孤儿 | 低 | 拒绝时在事务外异步清理 COS 文件（与现有 HandleDeleteSkill 的 COS 清理逻辑一致） |
| R3 | 并发提交——两个员工同时提交同 slug | 中 | `HasPendingRequest` 检查 + (slug, version) 唯一索引兜底；事务内再次校验 |
| R4 | 管理员通过 publish 时 (slug, version) 已被管理员抢先上传 | 低 | 通过时事务内再次校验唯一性，冲突则审核失败并提示 |
| R5 | 技能下发时 Skill 状态为 pending_review 或 offline | 中 | `HandleDistributeSkill` / `HandleSkillStoreDistribute` 增加 `status='published'` 校验；`admin_skill_distribution.go` 同理 |
| R6 | 用户端查询遗漏 status 过滤 | 中 | Plan 已逐个列出需改动的查询点（5.1 节 4 处 + 5.3 节 2 处 + 5.4 节 2 处）；UT 覆盖验证 |
| R7 | 通知机制——Notification.InstanceID 必填但技能审核不涉及实例 | 低 | InstanceID 传 0，Type 用新常量 |
| R8 | 多租户——ReviewRequest 需正确注入 identifier | 低 | 使用 `model.DB(r.Context())`，GORM 回调自动注入 |
| R9 | offline 技能仍存在于技能包（skill bundle）引用中，可能导致技能包下发异常 | 低 | 下发逻辑已加 `status='published'` 校验；技能包下发时对 offline 技能跳过或报错提示 |

---

## 九、与 Plan 差异追踪

> 实现阶段如果与本 Plan 有差异，在 `03-implement.md` 中记录。
