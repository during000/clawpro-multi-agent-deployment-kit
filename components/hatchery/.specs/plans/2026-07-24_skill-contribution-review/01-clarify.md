# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

当前系统只有管理员（`role=admin`）能上传技能到企业技能库，上传入口为 `POST /admin/skills/create`（`HandleCreateSkill`），由 `requireAdmin` 守卫。普通员工（`role=user`）只能通过技能广场（`/openclaw/skillstore`）浏览和安装技能，无法贡献自己的技能。

现有 `Skill` 模型（`model/skill.go`）**没有 `UploaderID` 字段**——上传者信息仅注入到 zip 内的 `_meta.json` 中（`ownerId`），不持久化到数据库。这意味着当前无法从 DB 层面追溯"谁上传了哪个技能"。

技能删除（下架）同样仅限管理员，入口为 `POST /admin/skills/delete`（`HandleDeleteSkill`），执行 GORM 软删除。

为支持企业技能库共建，需要开放员工提交技能的通道，引入管理员审核机制。同时考虑未来 MCP、Rule 等资源也可能需要审核，采用通用审批表设计。

---

## 目标

1. 员工可提交技能（含 zip 包），管理员审核通过后自动上架到企业技能库
2. 员工可申请下架自己上传的技能，管理员审核通过后自动下架
3. 同一 Skill（按 slug 维度）同时只能存在一个进行中的申请
4. 审核流程可追溯（谁提交、谁审核、何时审核、审核意见）
5. 采用通用审批表设计，未来可低成本扩展支持 MCP、Rule 等资源的审核

- [ ] 补全 `docs/` 下本次需求相关模块的 TODO 占位文档

---

## 范围

| 包含 | 不包含 |
|------|--------|
| 员工提交技能（上传 zip + 元数据），进入待审核状态 | 员工直接编辑已上架技能（修改仍走管理员） |
| 管理员审核发布申请（通过/拒绝 + 审核意见） | 管理员上传技能仍走现有 `/admin/skills/create`（不需审核，保持不变） |
| 员工申请下架自己上传的技能，进入待审核状态 | 员工下架他人上传的技能 |
| 管理员审核下架申请（通过/拒绝 + 审核意见） | 管理员删除技能仍走现有 `/admin/skills/delete`（不需审核，保持不变） |
| 同一 slug 只允许一个进行中的申请（发布/下架互斥） | 技能版本回滚 |
| 审核通过后自动上架/下架 | 批量审核（一次审核多个申请） |
| 员工查看自己的申请列表和状态 | 管理员查看所有待审核申请列表（含筛选） |
| 审核状态变更通知（站内通知） | 邮件/即时消息通知 |
| `Skill` 模型新增 `Status` + `UploaderID` 字段 | 修改现有管理员上传/删除技能的流程 |
| `review_requests` 通用审批表，预留 MCP/Rule 扩展 | 本期实现 MCP/Rule 审批 |

---

## 待确认问题（已确认）

| # | 问题 | 结论 |
|---|------|------|
| 1 | zip 包 COS 上传路径？ | **正式路径**，与管理员上传一致（`{slug}/{slug}-{version}.zip`） |
| 2 | "同一 Skill"互斥维度？ | **按 slug 维度**。同一 slug 若有 pending 申请，不允许再提交新申请 |
| 3 | 员工提交参数？ | **与管理员 `/admin/skills/create` 一致**（slug/name/version/description/changelog/file/category_ids/visibility 等） |
| 4 | 拒绝后可重新提交？ | **是**。被拒绝的申请变为 rejected 终态，不再阻塞新申请 |
| 5 | 下架范围？ | **整个 slug**（所有版本），与现有管理员全版本删除对齐 |
| 6 | 管理员审核时可修改元数据？ | **暂不支持**。一期审核只做通过/拒绝，不修改元数据 |
| 7 | 下架后用户侧可见性？ | **用户不可见**。Skill 状态变为 `offline`，技能广场不展示 |
| 8 | 员工提交新版本也走审核？ | **是**。发新版本 = 同 slug 更高版本号的 publish 申请，走审核流程 |
| 9 | 审批表是否通用？ | **是**。采用通用 `review_requests` 表，`resource_type` 区分资源类型，预留扩展 |

---

## 约束与依赖

- **数据库**：需新增 `review_requests` 通用审批表；`Skill` 模型新增 `Status`（`published`/`pending_review`/`offline`）和 `UploaderID` 字段，需同步更新 `sql/init.sql`、增量 migration SQL、`allModels`、`MigrateFromSQLite`。
- **COS/SMH**：员工提交的 zip 立即上传到 COS 正式路径（与管理员一致）；Skill 记录在提交时即创建（status=`pending_review`），审核通过后翻转为 `published`。
- **权限**：员工端接口用 `requireLogin` 守卫；管理员审核接口用 `requireAdmin` 守卫。
- **审计**：所有写接口（提交、审核）必须注册 `auditRules` + `WithAudit()` 包装。
- **i18n**：所有用户可见文案必须使用 `i18n.T()`，新增 `i18n.Key` 需同步英文翻译。
- **多租户**：新表需包含 `Identifier` 字段，遵循 GORM 回调自动注入规范；handler 内用 `model.DB(r.Context())`。
- **通知**：审核状态变更通过现有 `Notification` 模型发送站内通知。
- **API 兼容**：不修改现有 `/admin/skills/*` 和 `/openclaw/skillstore/*` 接口的语义，仅新增接口和字段。
- **扩展性**：通用审批表设计，`resource_type` 字段区分资源类型，未来增加 MCP/Rule 审批只需加常量 + 资源特定 handler。

---

## 整体方案

### 一、双状态机设计

#### 状态机一：Skill 状态（资源状态）

新增 `Skill.Status` 字段，三个状态：

```
                          ┌── admin 直接上传 ──────────────────→ published
                          │
      (不存在) ───────────┤
                          │   员工提交 publish 申请 ──→ pending_review
                          │                                 │
                          │                    ┌────────────┴────────────┐
                          │                    │                         │
                          │              admin 审核通过             admin 审核拒绝
                          │                    │                         │
                          │                    ▼                         ▼
                          │               published            (软删除 Skill 记录)
                          │                    │
                          │            员工申请 takedown
                          │            (Skill 保持 published)
                          │                    │
                          │              admin 审核通过
                          │                    │
                          │                    ▼
                          │                 offline
                          │                    │
                          │            admin 直接删除 /admin/skills/delete
                          │                    │
                          │                    ▼
                          └───────────── (软删除)
```

| 当前状态 | 事件 | 目标状态 | 说明 |
|---------|------|---------|------|
| (不存在) | admin 直接上传 | `published` | 现有流程不变，`UploaderID=0` |
| (不存在) | 员工提交 publish | `pending_review` | 创建 Skill + 创建 Request |
| `pending_review` | admin 通过 publish | `published` | 翻转状态，通知员工 |
| `pending_review` | admin 拒绝 publish | (软删除) | 清理 Skill 记录，通知员工 |
| `published` | 员工申请 takedown | `published`（不变） | 创建 Request，Skill 不动 |
| `published` | admin 通过 takedown | `offline` | 翻转状态，通知员工 |
| `published` | admin 拒绝 takedown | `published`（不变） | 通知员工 |
| 任意 | admin 直接删除 | (软删除) | 现有 `/admin/skills/delete` 不变 |

#### 状态机二：审批单状态（ContributionRequest.Status）

```
  pending ──── admin 通过 ───→ approved
     │
     └──── admin 拒绝 ────────→ rejected
```

| 当前状态 | 事件 | 目标状态 |
|---------|------|---------|
| (不存在) | 员工创建申请 | `pending` |
| `pending` | admin 通过 | `approved`（终态） |
| `pending` | admin 拒绝 | `rejected`（终态） |

#### 两个状态机的联动

| 申请类型 | Request 状态 | Skill 状态 | Skill 操作 |
|---------|-------------|-----------|-----------|
| publish | `pending` | `pending_review` | 已在提交时创建 |
| publish | `approved` | `published` | 更新 Skill status |
| publish | `rejected` | (软删除) | 软删除 Skill 记录 |
| takedown | `pending` | `published` | 无操作 |
| takedown | `approved` | `offline` | 更新 Skill status |
| takedown | `rejected` | `published`（不变） | 无操作 |

---

### 二、数据模型

#### 1. 新增通用审批表 `review_requests`

```sql
CREATE TABLE IF NOT EXISTS `review_requests` (
  `id`              bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`      varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`      datetime(3) DEFAULT NULL,
  `updated_at`      datetime(3) DEFAULT NULL,
  `deleted_at`      datetime(3) DEFAULT NULL,
  `requester_id`    bigint unsigned NOT NULL DEFAULT '0'  COMMENT '申请人 user_id',
  `resource_type`   varchar(32) NOT NULL DEFAULT 'skill'  COMMENT '资源类型：skill/mcp/rule/...',
  `resource_id`     bigint unsigned NOT NULL DEFAULT '0'   COMMENT '关联资源 ID（Skill.ID 等）',
  `action_type`     varchar(16) NOT NULL DEFAULT 'publish' COMMENT '操作类型：publish/takedown',
  `slug`            varchar(191) NOT NULL DEFAULT ''        COMMENT '资源 slug，冗余存储便于互斥查询',
  `status`          varchar(16) NOT NULL DEFAULT 'pending'  COMMENT '审批状态：pending/approved/rejected',
  `reason`          text                                     COMMENT '申请理由（takedown 必填）',
  `reviewer_id`     bigint unsigned NOT NULL DEFAULT '0'    COMMENT '审核人 user_id',
  `reviewed_at`     datetime(3) DEFAULT NULL                COMMENT '审核时间',
  `review_comment`  text                                     COMMENT '审核意见',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_review_requests_identifier` (`identifier`),
  KEY `idx_rr_requester` (`identifier`,`requester_id`),
  KEY `idx_rr_resource` (`identifier`,`resource_type`,`resource_id`),
  KEY `idx_rr_status` (`identifier`,`status`),
  KEY `idx_rr_slug_mutex` (`identifier`,`resource_type`,`slug`,`status`),
  KEY `idx_review_requests_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**互斥查询**：`WHERE resource_type='skill' AND slug=? AND status='pending'`，存在则拒绝新申请。

#### 2. 修改 `Skill` 模型

新增两个字段：

```go
Status     string `gorm:"not null;default:'published'" json:"status"`       // published（默认）| pending_review | offline
UploaderID uint   `gorm:"not null;default:0" json:"uploader_id"`           // 0=管理员上传，>0=员工 user_id
```

- 存量数据自动为 `published`（default 值），`uploader_id=0`（管理员）
- 管理员通过 `/admin/skills/create` 上传 → `Status=published`, `UploaderID=0`
- 员工贡献审核通过后 → `Status=published`, `UploaderID=requester_id`
- 员工提交时 → `Status=pending_review`, `UploaderID=user_id`

#### 3. 通用审批模型（Go）

```go
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
    ContributionStatusPending  = "pending"
    ContributionStatusApproved = "approved"
    ContributionStatusRejected = "rejected"
)

type ContributionRequest struct {
    ID              uint           `gorm:"primarykey" json:"id"`
    Identifier      string         `gorm:"index;default:''" json:"-"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
    RequesterID     uint           `gorm:"not null;default:0" json:"requester_id"`
    ResourceType    string         `gorm:"type:varchar(32);not null;default:'skill'" json:"resource_type"`
    ResourceID      uint           `gorm:"not null;default:0" json:"resource_id"`
    ActionType      string         `gorm:"type:varchar(16);not null;default:'publish'" json:"action_type"`
    Slug            string         `gorm:"type:varchar(191);not null;default:''" json:"slug"`
    Status          string         `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
    Reason          string         `gorm:"type:text" json:"reason"`
    ReviewerID      uint           `gorm:"not null;default:0" json:"reviewer_id"`
    ReviewedAt      *time.Time     `gorm:"default:null" json:"reviewed_at"`
    ReviewComment   string         `gorm:"type:text" json:"review_comment"`
}
```

---

### 三、API 接口

#### 员工端（`requireLogin`）

| 接口 | 方法 | 说明 |
|------|------|------|
| `POST /openclaw/skills/contribute` | POST | 提交技能/新版本（multipart：zip + 元数据，参数与 `/admin/skills/create` 一致） |
| `POST /openclaw/skills/takedown` | POST | 申请下架（slug + reason） |
| `GET /openclaw/skills/contributions` | GET | 查看我的申请列表（支持 status 筛选） |
| `GET /openclaw/skills/contributions/detail` | GET | 查看申请详情 |

#### 管理员端（`requireAdmin`）

| 接口 | 方法 | 说明 |
|------|------|------|
| `GET /admin/contributions` | GET | 查看所有申请列表（支持 resource_type/action_type/status 筛选） |
| `GET /admin/contributions/detail` | GET | 查看申请详情 |
| `POST /admin/contributions/approve` | POST | 审核通过（id） |
| `POST /admin/contributions/reject` | POST | 审核拒绝（id + review_comment） |

所有写接口均需 `WithAudit()` + `auditRules` 注册。

---

### 四、业务流程

#### 发布流程

```
员工提交 POST /openclaw/skills/contribute
  │
  ├─ 校验：slug 无 pending 申请（互斥：resource_type='skill' AND slug=? AND status='pending'）
  ├─ 校验：(slug, version) 不与现有 Skill 或 pending_review Skill 冲突
  ├─ 校验：zip 格式（复用 validateSkillZip）
  ├─ 注入 _meta.json（复用 injectMetaIntoZip）
  ├─ 上传 COS → {slug}/{slug}-{version}.zip（与管理员一致，复用现有 SMH 上传逻辑）
  ├─ 创建 Skill 记录（status=pending_review, uploader_id=user_id）
  ├─ 创建 ContributionRequest（resource_type='skill', action_type='publish', status='pending', resource_id=新Skill.ID）
  └─ 通知管理员有待审核
        │
管理员审核 POST /admin/contributions/approve
  │
  ├─ 校验：Request.status == 'pending'
  ├─ 更新 Skill status → published
  ├─ 更新 Request: status=approved, reviewer_id, reviewed_at
  └─ 通知员工：审核通过，已上架

管理员拒绝 POST /admin/contributions/reject
  │
  ├─ 校验：Request.status == 'pending'
  ├─ 软删除 Skill 记录
  ├─ 更新 Request: status=rejected, reviewer_id, reviewed_at, review_comment
  └─ 通知员工：审核未通过
```

#### 下架流程

```
员工申请 POST /openclaw/skills/takedown (slug + reason)
  │
  ├─ 校验：Skill 存在且 skill.UploaderID == user.ID
  ├─ 校验：slug 无 pending 申请（互斥）
  ├─ 创建 ContributionRequest（resource_type='skill', action_type='takedown', status='pending', resource_id=Skill.ID, slug=Skill.Slug）
  └─ 通知管理员有待审核
        │
管理员审核 POST /admin/contributions/approve
  │
  ├─ 校验：Request.status == 'pending'
  ├─ 更新 Skill status → offline
  ├─ 更新 Request: status=approved, reviewer_id, reviewed_at
  └─ 通知员工：审核通过，已下架

管理员拒绝 POST /admin/contributions/reject
  │
  ├─ 校验：Request.status == 'pending'
  ├─ 更新 Request: status=rejected, reviewer_id, reviewed_at, review_comment
  └─ 通知员工：下架申请未通过
```

---

### 五、对现有查询的影响

新增 `status` 字段默认 `published`，**存量数据自动为 `published`**。需要改动的查询：

| 查询位置 | 改动 |
|---------|------|
| 技能广场 `/openclaw/skillstore` | 加 `WHERE status = 'published'` 过滤 |
| `LatestVersionSkillIDs`（`model/skill.go`） | 加 `status = 'published'` 条件 |
| 用户技能列表 `/openclaw/skills` | 加 `status = 'published'` 过滤 |
| 管理员技能列表 `/admin/skills` | **不过滤**，显示所有状态（附带 status 字段供前端展示） |
| 管理员删除/下发等操作 | 不受影响（管理员可操作所有状态的 Skill） |
| 管理员更新 `/admin/skills/update` | 不受影响（管理员直接更新，不走审核） |
新增 `status` 字段默认 `published`，**存量数据自动为 `published`**。需要改动的查询：

| 查询位置 | 改动 |
|---------|------|
| 技能广场 `/openclaw/skillstore` | 加 `WHERE status = 'published'` 过滤 |
| `LatestVersionSkillIDs`（`model/skill.go`） | 加 `status = 'published'` 条件 |
| 用户技能列表 `/openclaw/skills` | 加 `status = 'published'` 过滤 |
| 管理员技能列表 `/admin/skills` | **不过滤**，显示所有状态（附带 status 字段供前端展示） |
| 管理员删除/分发等操作 | 不受影响（管理员可操作所有状态的 Skill） |

---

### 六、不改动的部分

| 现有功能 | 说明 |
|---------|------|
| `/admin/skills/create` | 管理员上传，不走审核，`Status=published`, `UploaderID=0` |
| `/admin/skills/update` | 管理员更新元数据，不走审核 |
| `/admin/skills/delete` | 管理员删除，不走审核（软删除） |
| `/openclaw/skillstore/*` | 技能广场查询，仅加 status 过滤 |
| `/openclaw/install-skills` 等 | 技能安装/卸载，无需改动 |

---

### 七、扩展性设计

通用审批表 `review_requests` 通过 `resource_type` 字段区分资源类型。未来增加 MCP 审批只需：

| 改动项 | 工作量 |
|--------|--------|
| `review_requests` 表 | **不改**（已有 `resource_type` 字段） |
| `model/mcp.go` | +1 行（加 `Status` 字段） |
| `model/contribution_request.go` | +1 行（加 `ResourceTypeMcp = "mcp"` 常量） |
| `controller/contribution_mcp.go` | 新建（MCP 专用的 submit/takedown/approve/reject 逻辑） |
| `main.go` | +4 行（注册 4 个新路由） |
| migration SQL | MCP 表加 status 列（通用审批表无需改动） |

代码结构：

```
model/
├── contribution_request.go     # 通用 ContributionRequest 模型 + 常量

controller/
├── contribution.go              # 通用：列表查询、详情、审核分发（switch resource_type）
├── contribution_skill.go        # Skill 专用：提交、下架申请、审核通过/拒绝的具体逻辑
├── contribution_mcp.go          # 未来：MCP 专用逻辑
├── contribution_rule.go         # 未来：Rule 专用逻辑
```

审核 handler dispatch 模式：

```go
func HandleContributionApprove(w, r) {
    req := loadRequest(id)
    switch req.ResourceType {
    case model.ResourceTypeSkill:
        approveSkillContribution(req)   // Skill: pending_review → published
    case model.ResourceTypeMcp:
        approveMcpContribution(req)     // 未来
    }
}
```

---

### 八、改动文件清单预估

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/skill.go` | 改 | Skill 加 `Status` + `UploaderID` 字段 |
| `model/contribution_request.go` | 新建 | `ContributionRequest` 模型 + 常量 |
| `model/db.go` | 改 | `allModels` 加入 `ContributionRequest` |
| `model/migrate.go` | 改 | 添加新表迁移逻辑 |
| `sql/init.sql` | 改 | skills 加 `status` + `uploader_id` 列 + 新建 `review_requests` 表 |
| `sql/0724-skill-contribution-review.sql` | 新建 | 增量 migration |
| `controller/contribution.go` | 新建 | 通用：列表、详情、审核分发 |
| `controller/contribution_skill.go` | 新建 | Skill 专用：提交、下架、审核逻辑 |
| `controller/openclaw_skillstore.go` | 改 | 技能广场查询加 `status='published'` 过滤 |
| `controller/openclaw_skill.go` | 改 | 用户技能列表加 status 过滤 |
| `controller/admin_skills.go` | 改 | 管理员技能列表返回 status 字段 |
| `model/skill.go` | 改 | `LatestVersionSkillIDs` 加 status 过滤 |
| `controller/audit.go` | 改 | 注册新接口的 audit rules |
| `main.go` | 改 | 注册新路由 |
| `i18n/keys.go` + `i18n/en.go` | 改 | 新增 i18n key + 英文翻译 |
| `docs/API.md` | 改 | 新增接口文档 |

---

## Discovery（已调研的代码现状）

### 现有技能上传流程（管理员）
- **入口**：`POST /admin/skills/create` → `HandleCreateSkill`（`controller/admin_skills.go:676`）
- **守卫**：`requireAdmin` + `requireSMHEnabled`
- **流程**：解析 multipart form → 校验 slug/version → 版本递增校验 → (slug,version) 唯一性校验 → 读取 zip → `validateSkillZip` 校验 → `injectMetaIntoZip` 注入 ownerId → 上传 COS → 创建 DB 记录（事务）→ 处理分类关联 + 可见范围
- **COS 路径**：`{slug}/{slug}-{version}.zip`（zip）和 `{slug}/{slug}-{version}/`（解压目录）

### 现有技能删除流程（管理员）
- **入口**：`POST /admin/skills/delete` → `HandleDeleteSkill`（`controller/admin_skills.go:1125`）
- **守卫**：`requireAdmin` + `requireSMHEnabled`
- **流程**：按 slug（+可选 version）查找 → 检查 running tasks → 事务内级联删除（bundle 引用、COS 文件记录等）→ GORM 软删除

### Skill 模型现状
- 无 `UploaderID` / `Status` 字段
- `ownerId` 仅注入到 zip 的 `_meta.json` 中（`admin_skills.go:782-784`），不持久化到 DB
- 有 `VisibilityType`（all/group）和 `SkillVisibilityGroup` 关联表控制可见范围

### 用户模型
- `User.Role` 字段：`"admin"` 或 `"user"`
- `requireAdmin` 检查 `role == "admin"`
- `requireLogin` 返回 `*model.User`，包含 `ID`、`Username` 等

### 现有通知机制
- `Notification` 模型已存在，在 allModels 中注册
- 有 `NotifTitle*` / `NotifMsg*` 的 i18n key 模式

### allModels 和 migration
- 新模型需加入 `model/db.go` 的 `allModels` 切片（约 297 行）
- `MigrateFromSQLite` 需添加新表的迁移逻辑
- `checkMigrationCoverage` 会校验所有 `allModels` 表均被覆盖

---

## Challenge（关键设计挑战及决策）

### 挑战 1：审核中技能的存储策略
- **决策**：立即创建 Skill 记录（`status=pending_review`）+ COS 上传正式路径
- **原因**：用户确认 COS 正式路径（Q1）；Skill 有显式状态机，提交时即创建记录；审核通过后只需翻转状态，无需文件迁移

### 挑战 2：版本唯一性冲突
- **问题**：员工提交 (slug, version) 待审核期间，管理员直接上传同 slug 同版本
- **方案**：Skill 记录在提交时即创建，唯一索引 `idx_slug_version_identifier` 自动防止冲突；管理员上传会收到版本已存在的错误

### 挑战 3：下架权限校验
- **决策**：`Skill.UploaderID` 字段，员工申请下架时校验 `skill.UploaderID == user.ID`；管理员上传的技能 `UploaderID=0`，员工无法申请下架

### 挑战 4：同一 slug 的申请互斥
- **决策**：`review_requests` 表查询 `WHERE resource_type='skill' AND slug=? AND status='pending'`，存在则拒绝

### 挑战 5：扩展性
- **决策**：通用 `review_requests` 表 + `resource_type` 字段，审核 handler 用 switch dispatch；未来增加新资源类型只需加常量 + 资源特定 handler
