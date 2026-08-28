package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// SkillDistributionTask / SkillDistributionRecord 的 Type 字段常量
const (
	TaskTypeDistribute = "distribute" // 下发操作
	TaskTypeUninstall  = "uninstall"  // 卸载操作
)

// Skill source constants.
const (
	SkillSourceEnterprise = "enterprise"
	SkillSourcePublic     = "public"
)

// Record status 常量
const (
	RecordStatusPending            = "pending"
	RecordStatusSuccess            = "success"
	RecordStatusFailed             = "failed"               // 首次安装失败（实例上无技能）
	RecordStatusUpgradeFailed      = "upgrade_failed"       // 升级失败（旧版本仍在实例上）
	RecordStatusUninstallFailedOld = "uninstall_failed_old" // 卸载旧版本失败（upgrade_failed 后尝试卸载旧版本失败）
	RecordStatusCancelled          = "cancelled"            // 已取消（切换分组清理 / report 已装但还有 pending）
)

// sync/ack 命令类型（与 local-agent sync 返回的 command.type 一致）
const (
	CommandTypeInstallSkill   = "install_skill"
	CommandTypeUninstallSkill = "uninstall_skill"
)

// SkillDistributionTask.Status 常量（RuleDistributionTask 复用同一组值）
const (
	TaskStatusRunning   = "running"   // 进行中
	TaskStatusCompleted = "completed" // 已完成
)

// ResolveDistributeFailedStatus 判断下发失败时应写入的 status。
// 如果该实例+技能有过历史成功记录，说明是升级失败（旧版本仍在），返回 "upgrade_failed"；
// 否则是首次安装失败，返回 "failed"。
func ResolveDistributeFailedStatus(ctx context.Context, instanceID uint, skillIDs []uint) string {
	var count int64
	DB(ctx).Model(&SkillDistributionRecord{}).
		Where("instance_id = ? AND skill_id IN ? AND status = ? AND type = ?",
			instanceID, skillIDs, RecordStatusSuccess, TaskTypeDistribute).
		Limit(1).Count(&count)
	if count > 0 {
		return RecordStatusUpgradeFailed
	}
	return RecordStatusFailed
}

// ResolvePublicSkillDistributeFailedStatus 判断公共技能下发失败时应写入的 status。
// 仅当该实例+公共技能的最新历史记录表明旧版本仍在实例上时，返回 "upgrade_failed"；
// 如果最新状态是未安装（例如成功卸载后重新安装失败），返回 "failed"。
// currentRecordID 为当前失败中的 pending record；解析历史状态时必须排除它。
func ResolvePublicSkillDistributeFailedStatus(ctx context.Context, instanceID uint, slug string, currentRecordID uint) string {
	var latest SkillDistributionRecord
	query := DB(ctx).Model(&SkillDistributionRecord{}).
		Joins("JOIN skill_distribution_tasks t ON t.id = skill_distribution_records.task_id AND t.deleted_at IS NULL").
		Where("skill_distribution_records.instance_id = ? AND t.source = ? AND t.slug = ?",
			instanceID, SkillSourcePublic, slug)
	if currentRecordID > 0 {
		query = query.Where("skill_distribution_records.id <> ?", currentRecordID)
	}
	err := query.Order("skill_distribution_records.id DESC").Limit(1).First(&latest).Error
	if err == nil && recordMeansSkillStillInstalled(latest.Type, latest.Status) {
		return RecordStatusUpgradeFailed
	}
	return RecordStatusFailed
}

func recordMeansSkillStillInstalled(recordType, status string) bool {
	switch recordType {
	case TaskTypeUninstall:
		return status == RecordStatusFailed || status == RecordStatusUninstallFailedOld
	default:
		return status == RecordStatusSuccess || status == RecordStatusUpgradeFailed
	}
}

// ResolveUninstallFailedStatus 判断卸载失败时应写入的 status。
// 逻辑：查找该实例最近一条下发成功的 Record，比较其版本与技能最新版本。
// - 如果已安装版本 != 最新版本 → 说明是卸载旧版本失败，返回 "uninstall_failed_old"
// - 如果已安装版本 == 最新版本（或无成功记录）→ 正常卸载失败，返回 "failed"
//
// latestVersion 为当前技能库中该 slug 的最新版本号。
func ResolveUninstallFailedStatus(ctx context.Context, instanceID uint, skillIDs []uint, latestVersion string) string {
	// 查询该实例最近一条下发成功的记录，获取已安装版本
	var lastSuccessRecord SkillDistributionRecord
	err := DB(ctx).Where("instance_id = ? AND skill_id IN ? AND status = ? AND type = ?",
		instanceID, skillIDs, RecordStatusSuccess, TaskTypeDistribute).
		Order("id DESC").Limit(1).First(&lastSuccessRecord).Error
	if err == nil && lastSuccessRecord.Version != "" && lastSuccessRecord.Version != latestVersion {
		return RecordStatusUninstallFailedOld
	}
	return RecordStatusFailed
}

// ResolvePublicSkillUninstallFailedStatus 判断公共技能卸载失败时应写入的 status。
func ResolvePublicSkillUninstallFailedStatus(ctx context.Context, instanceID uint, slug, latestVersion string) string {
	if latestVersion == "" {
		return RecordStatusFailed
	}
	var lastSuccessRecord SkillDistributionRecord
	err := DB(ctx).Model(&SkillDistributionRecord{}).
		Joins("JOIN skill_distribution_tasks t ON t.id = skill_distribution_records.task_id AND t.deleted_at IS NULL").
		Where("skill_distribution_records.instance_id = ? AND t.source = ? AND t.slug = ? AND skill_distribution_records.status = ? AND skill_distribution_records.type = ?",
			instanceID, SkillSourcePublic, slug, RecordStatusSuccess, TaskTypeDistribute).
		Order("skill_distribution_records.id DESC").Limit(1).First(&lastSuccessRecord).Error
	if err == nil && lastSuccessRecord.Version != "" && lastSuccessRecord.Version != latestVersion {
		return RecordStatusUninstallFailedOld
	}
	return RecordStatusFailed
}

// Skill 技能
type Skill struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	Identifier      string         `gorm:"uniqueIndex:idx_slug_version_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Slug            string         `gorm:"uniqueIndex:idx_slug_version_identifier;not null" json:"slug"`
	Name            string         `gorm:"not null" json:"name"`
	Description     string         `gorm:"type:text;not null" json:"description"`
	Version         string         `gorm:"uniqueIndex:idx_slug_version_identifier;not null;default:'1.0.0'" json:"version"`
	VersionMajor    int            `gorm:"not null;default:0" json:"-"` // semver major，用于 DB 层排序
	VersionMinor    int            `gorm:"not null;default:0" json:"-"` // semver minor
	VersionPatch    int            `gorm:"not null;default:0" json:"-"` // semver patch
	COSZipKey       string         `gorm:"not null;default:''" json:"cos_zip_key"`
	COSDirKey       string         `gorm:"not null;default:''" json:"cos_dir_key"`
	FileList        string         `gorm:"type:text" json:"file_list"` // JSON []string
	FileSize        int64          `gorm:"not null;default:0" json:"file_size"`
	VisibilityType  string         `gorm:"not null;default:'all'" json:"visibility_type"`            // 可见范围：all=全部用户, group=按分组
	Changelog       string         `gorm:"type:varchar(10000);not null;default:''" json:"changelog"` // 版本更新说明
	DistributeCount int            `gorm:"not null;default:0" json:"distribute_count"`               // 累计下发成功数
	Status          string         `gorm:"not null;default:'published'" json:"status"`               // published | pending_review | offline
	UploaderID      uint           `gorm:"not null;default:0" json:"uploader_id"`                    // 0=admin, >0=员工 user_id
}

// Skill status 常量
const (
	SkillStatusPublished     = "published"
	SkillStatusPendingReview = "pending_review"
	SkillStatusOffline       = "offline"
)

// LatestVersionSkillIDs 返回 GORM 子查询，选出每个 slug 最新版本的 skill ID。
// 使用 Model(&Skill{}) 保证 GORM 回调自动注入 identifier 和 deleted_at 条件。
// 算法：对每个 slug，将版本号编码为单个整数 (major*1000000 + minor*1000 + patch)，
// 取 MAX 得到最高版本，再通过 JOIN 反查对应的 skill ID。
func LatestVersionSkillIDs(ctx context.Context) *gorm.DB {
	// 子查询：每个 slug 的最高版本分数（回调自动注入 identifier + deleted_at）
	maxVersions := DB(ctx).Model(&Skill{}).
		Select("slug, MAX(version_major * 1000000 + version_minor * 1000 + version_patch) AS max_ver").
		Group("slug")
	// 外层：通过 slug + 版本分数匹配，取对应的 ID（回调自动注入 identifier + deleted_at）
	return DB(ctx).Model(&Skill{}).
		Select("skills.id").
		Joins("JOIN (?) mv ON mv.slug = skills.slug AND mv.max_ver = skills.version_major * 1000000 + skills.version_minor * 1000 + skills.version_patch", maxVersions)
}

// BuildSkillInstanceQuery 构造实例安装状态查询。
// 使用 Model(&Instance{}) 保证 GORM 回调自动注入 identifier 和 deleted_at 条件。
// JOIN 的关联表通过外键引用已隔离的主表数据（instance_id、skill_id），无需额外 identifier 过滤。
// latestVersion 为该技能的最新版本号，用于判断已安装实例是否需要更新（outdated）。
// slug 用于本地 agent 实例的事实校验：
//   - LEFT JOIN local_instance_skills lis（当前 slug 对应行）；
//   - 如果本地实例（source='local'）的最新 record 是 distribute+success 但 reporter
//     上报里没有该 slug（lis.id IS NULL），说明用户在本地把技能卸掉了，改判 uninstalled；
//   - CVM 实例（source != 'local'）不受此 JOIN 影响，行为保持不变。
func BuildSkillInstanceQuery(ctx context.Context, skillIDs []uint, latestVersion, slug string) *gorm.DB {
	// 防御性校验：确保 latestVersion 是合法的 x.y.z 格式，防止 SQL 注入。
	// latestVersion 来自数据库查询结果，正常情况下已经通过 ParseVersion 校验过，
	// 但作为深度防御，若格式不合法则回退为空字符串（所有已安装实例都显示为 installed）。
	if VersionScore(latestVersion) == 0 {
		latestVersion = ""
	}
	// 注意：latestVersion 通过 fmt.Sprintf 嵌入 SQL 字面值而非 ? 绑定参数。
	// 原因：GORM 的 Select() 绑定参数与后续 Where() 子查询参数共享同一个 Vars 列表，
	// 当存在子查询（如 group_id 过滤）时会导致参数顺序错位。
	// slug 同理内嵌为字面值，只在这里做安全校验（isValidSlug 已在 controller 层校验）
	selectClause := fmt.Sprintf(`instances.id AS instance_id,
		instances.instance_id AS cvm_instance_id,
		instances.name AS instance_name,
		COALESCE(instances.agent_type, '') AS instance_type,
		instances.user_id AS user_id,
		instances.source AS source,
		instances.last_cvm_state AS last_cvm_state,
		instances.last_stable_state AS last_stable_state,
		instances.current_operation AS current_operation,
		instances.current_operation_state AS current_operation_state,
		instances.agent_ready AS agent_ready,
		instances.cls_agent_status AS cls_agent_status,
		instances.cls_agent_status_at AS cls_agent_status_at,
		COALESCE(u.username, '') AS username,
		%s AS install_status,
		COALESCE(lr.version, '') AS version,
		'%s' AS latest_version`, InstallStatusCase(latestVersion), latestVersion)

	q := DB(ctx).Model(&Instance{}).Select(selectClause)

	// users JOIN — user_id 来自已隔离的 instances，无需重复过滤 identifier
	q = q.Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL")

	// skill_distribution_records JOIN — instance_id 和 skill_id 均来自已隔离数据，
	// 关联到的 records 必然属于当前租户，无需额外 identifier 过滤
	q = q.Joins(`LEFT JOIN skill_distribution_records lr
		ON lr.instance_id = instances.id
		AND lr.deleted_at IS NULL
		AND lr.id = (
			SELECT MAX(r2.id) FROM skill_distribution_records r2
			WHERE r2.skill_id IN ? AND r2.instance_id = instances.id AND r2.deleted_at IS NULL
		)`, skillIDs)

	// local_instance_skills JOIN — 用于本地 agent 实例的事实校验（reporter 上报的当前快照）。
	// 注意：local_instance_skills 的唯一约束是 (scope, instance_id, workspace_path, slug)，
	// 同一 slug 在一个实例上可同时存在于 user / project 等多个 scope（每行独立）。
	// 若直接按 (instance_id, slug) JOIN 会把 instances 主表该行扇出成多行，导致
	// /admin/skills/instances 返回重复数据（与 enterprise_rule 历史上的 JOIN 扇出 bug 同类）。
	// 因此用聚合子查询取每个 (instance_id, slug) 的 MAX(id) 一行，保证一个实例一个 slug
	// 至多匹配一行 lis；InstallStatusCase 只用 lis.id IS NULL 判 uninstalled，语义不变。
	q = q.Joins(`LEFT JOIN local_instance_skills lis
		ON lis.id = (
			SELECT MAX(s2.id) FROM local_instance_skills s2
			WHERE s2.instance_id = instances.id AND s2.slug = ?
		)`, slug)

	// deleted_at 和 identifier 由 GORM 回调自动注入，无需手动添加
	q = q.Where("instances.instance_id != ''")

	return q
}

// InstallStatusCase 返回安装状态的 SQL CASE 表达式。
// 用于 SELECT 和 WHERE 子句，确保两处逻辑一致。
// latestVersion 已通过 VersionScore 校验，直接嵌入 SQL 字面值。
//
// 状态判断逻辑（基于最新一条 Record 的 type + status）：
//   - 无记录 → uninstalled
//   - 最新记录是卸载操作（type=uninstall）：
//   - success → uninstalled（卸载成功，恢复未安装状态）
//   - pending → uninstalling（正在卸载中）
//   - failed  → uninstall_failed（卸载失败，技能仍在）
//   - 最新记录是下发操作（type=distribute 或空）：
//   - success + 版本旧 → outdated
//   - success → installed
//   - pending → installing
//   - upgrade_failed → upgrade_failed（升级失败，旧版本仍在）
//   - failed  → failed（首次安装失败，实例上无技能）
//
// MAX(id) 自然处理"卸载后重新下发"的场景——最新的 Record 始终决定当前状态。
//
// 本地 agent 特殊分支：
//   - `local_instance_skills` 是 reporter 上报的当前事实快照。
//   - 对于 source='local' 的实例，如果 record 上说"技能应该在"（installed / outdated /
//     upgrade_failed / uninstall_failed / uninstall_failed_old），但 lis 中没有对应
//     slug 行（lis.id IS NULL），说明用户已经在本地把技能卸掉了，改判 uninstalled。
//   - CVM 实例（source != 'local'）永远不会有 lis 行，走原分支不受影响。
func InstallStatusCase(latestVersion string) string {
	return fmt.Sprintf(`CASE
		WHEN lr.status IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  THEN 'uninstall_failed'
		WHEN lr.status = 'success' AND lr.version != '%s' THEN 'outdated'
		WHEN lr.status = 'success' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'success' THEN 'installed'
		WHEN lr.status = 'pending' THEN 'installing'
		WHEN lr.status = 'upgrade_failed' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
		WHEN lr.status = 'uninstall_failed_old' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
		WHEN lr.status = 'failed'  THEN 'failed'
		ELSE 'uninstalled'
	END`, latestVersion)
}

// FilterSkillInstallStatuses 按安装状态筛选实例。
// CASE 与 InstallStatusCase 保持一致；latestVersion 和 statuses 均通过 GORM 参数绑定。
func FilterSkillInstallStatuses(latestVersion string, statuses []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(statuses) == 0 {
			return db
		}
		return db.Where(`(CASE
			WHEN lr.status IS NULL THEN 'uninstalled'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed' THEN 'uninstall_failed'
			WHEN lr.status = 'success' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
			WHEN lr.status = 'success' AND lr.version != ? THEN 'outdated'
			WHEN lr.status = 'success' THEN 'installed'
			WHEN lr.status = 'pending' THEN 'installing'
			WHEN lr.status = 'upgrade_failed' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
			WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
			WHEN lr.status = 'uninstall_failed_old' AND instances.source = 'local' AND lis.id IS NULL THEN 'uninstalled'
			WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
			WHEN lr.status = 'failed' THEN 'failed'
			ELSE 'uninstalled'
		END) IN ?`, latestVersion, statuses)
	}
}

func (s *Skill) ParseVersion() error {
	parts := strings.SplitN(s.Version, ".", 3)
	if len(parts) != 3 {
		return hcommon.I18nError(i18n.MsgMcpVersionFormatInvalid)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return hcommon.I18nError(i18n.MsgSkillVersionMajorInvalid)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return hcommon.I18nError(i18n.MsgSkillVersionMinorInvalid)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return hcommon.I18nError(i18n.MsgSkillVersionPatchInvalid)
	}
	s.VersionMajor = major
	s.VersionMinor = minor
	s.VersionPatch = patch
	return nil
}

// NormalizeSkillVersion parses a Skill version and rebuilds it exclusively from integer components.
// Request-sourced versions must use the returned value before SQL interpolation.
func NormalizeSkillVersion(version string) (string, error) {
	if strings.ContainsAny(version, "+-") {
		return "", hcommon.I18nError(i18n.MsgMcpVersionFormatInvalid)
	}
	skill := Skill{Version: version}
	if err := skill.ParseVersion(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", skill.VersionMajor, skill.VersionMinor, skill.VersionPatch), nil
}

// VersionScore 将 "x.y.z" 字符串转为整数分数 (major*1000000 + minor*1000 + patch)。
// 用于版本比较（如 sync 时判断是否为更高版本）。解析失败返回 0。
func VersionScore(version string) int {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) != 3 {
		return 0
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0
	}
	return major*1000000 + minor*1000 + patch
}

// InheritSkillDistributeCount 从同 slug 的旧版本继承 distribute_count。
// 如果无旧版本或旧版本 distribute_count 为 0，不做任何操作。
func InheritSkillDistributeCount(tx *gorm.DB, slug string, newSkillID uint) error {
	var prevSkill Skill
	err := tx.Where("slug = ? AND id != ?", slug, newSkillID).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&prevSkill).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // 无旧版本（首次上传），不需要继承
		}
		return hcommon.I18nRichError(err, i18n.MsgSkillQueryOldVersionFailed)
	}
	if prevSkill.DistributeCount <= 0 {
		return nil
	}
	if err := tx.Model(&Skill{}).Where("id = ?", newSkillID).
		Update("distribute_count", prevSkill.DistributeCount).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillUpdateDistributeCountFailed)
	}
	return nil
}

// SkillDistributionTask 下发/卸载任务（每次批量操作创建一个 Task）
type SkillDistributionTask struct {
	ID                 uint           `gorm:"primarykey" json:"id"`
	Identifier         string         `gorm:"index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
	SkillID            uint           `gorm:"index;not null" json:"skill_id"`
	Version            string         `gorm:"not null;default:''" json:"version"`
	Source             string         `gorm:"type:varchar(20);not null;default:'enterprise';index:idx_skill_distribution_tasks_source_slug,priority:1;index:idx_skill_distribution_tasks_source_skillset,priority:1" json:"source"` // enterprise / public
	Slug               string         `gorm:"type:varchar(191);not null;default:'';index:idx_skill_distribution_tasks_source_slug,priority:2" json:"slug"`
	SourceSkillsetSlug string         `gorm:"type:varchar(191);not null;default:'';index:idx_skill_distribution_tasks_source_skillset,priority:2" json:"source_skillset_slug"`
	BatchID            string         `gorm:"type:varchar(64);not null;default:'';index:idx_skill_distribution_tasks_batch_id" json:"batch_id"`
	OperatorID         uint           `gorm:"not null;default:0" json:"operator_id"`
	Total              int            `gorm:"not null;default:0" json:"total"`
	Success            int            `gorm:"not null;default:0" json:"success"`
	Failed             int            `gorm:"not null;default:0" json:"failed"`
	Status             string         `gorm:"not null;default:'running'" json:"status"`  // running / completed
	Type               string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// SkillDistributionRecord 下发/卸载记录（每个实例一条）
type SkillDistributionRecord struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Identifier  string         `gorm:"index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID      uint           `gorm:"index;not null" json:"task_id"`
	SkillID     uint           `gorm:"index;not null" json:"skill_id"`
	InstanceID  uint           `gorm:"index;not null" json:"instance_id"`
	InstanceCID string         `gorm:"not null;default:''" json:"instance_cid"`  // CVM InstanceId
	Version     string         `gorm:"not null;default:''" json:"version"`       // 下发时的技能版本号（冗余）
	Status      string         `gorm:"not null;default:'pending'" json:"status"` // pending / success / failed
	Error       string         `gorm:"type:text" json:"error"`
	Type        string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// DistributedSkillState is the effective state of one Admin-distributed skill
// on an instance. Only successful distribute and uninstall records change the
// state.
type DistributedSkillState struct {
	Slug      string
	Source    string
	Version   string
	SkillID   uint
	RecordID  uint
	Installed bool
}

type distributedSkillStateRow struct {
	RecordID uint   `gorm:"column:record_id"`
	SkillID  uint   `gorm:"column:skill_id"`
	Slug     string `gorm:"column:slug"`
	Source   string `gorm:"column:source"`
	Version  string `gorm:"column:version"`
	Type     string `gorm:"column:type"`
}

// ListDistributedSkillStates returns the effective state of the requested
// slugs on one instance. The result includes successfully uninstalled slugs
// with Installed set to false so callers can implement idempotent operations.
func ListDistributedSkillStates(ctx context.Context, instanceID uint, slugs []string) (map[string]DistributedSkillState, error) {
	slugSet := make(map[string]struct{}, len(slugs))
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug != "" {
			slugSet[slug] = struct{}{}
		}
	}

	states := make(map[string]DistributedSkillState, len(slugSet))
	if len(slugSet) == 0 {
		return states, nil
	}

	uniqueSlugs := make([]string, 0, len(slugSet))
	for slug := range slugSet {
		uniqueSlugs = append(uniqueSlugs, slug)
	}

	var rows []distributedSkillStateRow
	err := DB(ctx).Model(&SkillDistributionRecord{}).
		Select(`skill_distribution_records.id AS record_id,
			skill_distribution_records.skill_id,
			COALESCE(NULLIF(t.slug, ''), s.slug) AS slug,
			t.source,
			skill_distribution_records.version,
			skill_distribution_records.type`).
		Joins("JOIN skill_distribution_tasks t ON t.id = skill_distribution_records.task_id AND t.deleted_at IS NULL").
		Joins("LEFT JOIN skills s ON s.id = skill_distribution_records.skill_id").
		Where("skill_distribution_records.instance_id = ?", instanceID).
		Where("skill_distribution_records.status = ?", RecordStatusSuccess).
		Where("skill_distribution_records.type IN ?", []string{TaskTypeDistribute, TaskTypeUninstall}).
		Where("(t.slug IN ? OR s.slug IN ?)", uniqueSlugs, uniqueSlugs).
		Order("skill_distribution_records.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list distributed skill states: %w", err)
	}

	for _, row := range rows {
		if _, ok := slugSet[row.Slug]; !ok {
			continue
		}
		states[row.Slug] = DistributedSkillState{
			Slug:      row.Slug,
			Source:    row.Source,
			Version:   row.Version,
			SkillID:   row.SkillID,
			RecordID:  row.RecordID,
			Installed: row.Type == TaskTypeDistribute,
		}
	}
	return states, nil
}
