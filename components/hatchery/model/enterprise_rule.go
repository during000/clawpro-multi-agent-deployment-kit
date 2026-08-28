package model

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// EnterpriseRule 类型 / 来源常量。
const (
	EnterpriseRuleTypePrompt = "prompt" // System Prompt / 全局说明类 md
	EnterpriseRuleTypeRule   = "rule"   // Rules 规则文件类 md
	EnterpriseRuleTypeHook   = "hook"   // 三期：Hook 资源（单触发时机 + 单命令，无文件）

	EnterpriseRuleSourceEnterprise = "enterprise" // 管理端上传
	EnterpriseRuleSourceLocal      = "local"      // 用户在本地手动安装（占位保留，与 skill 对齐）
)

// Hook 触发时机枚举（5 种）。
const (
	EnterpriseRuleHookEventSessionStart = "SessionStart"
	EnterpriseRuleHookEventUserPrompt   = "UserPromptSubmit"
	EnterpriseRuleHookEventPreToolUse   = "PreToolUse"
	EnterpriseRuleHookEventPostToolUse  = "PostToolUse"
	EnterpriseRuleHookEventStop         = "Stop"
)

// validHookEvents 是 Hook 触发时机的合法值集合。
var validHookEvents = map[string]bool{
	EnterpriseRuleHookEventSessionStart: true,
	EnterpriseRuleHookEventUserPrompt:   true,
	EnterpriseRuleHookEventPreToolUse:   true,
	EnterpriseRuleHookEventPostToolUse:   true,
	EnterpriseRuleHookEventStop:         true,
}

// IsValidHookEvent 校验触发时机是否合法。
func IsValidHookEvent(event string) bool {
	return validHookEvents[event]
}

// RuleDistributionRecord.Type / RuleDistributionTask.Type 常量。
// **完全对齐 skill：只有 distribute / uninstall 两种，不引入 update。**
// 「更新」= 上传新版本 → 重新 distribute → reporter 幂等覆写（同 local_agent.go:536 现有做法）。
const (
	RuleTaskTypeDistribute = "distribute"
	RuleTaskTypeUninstall  = "uninstall"
)

// RuleDistributionRecord.Status 复用 skill 一期的枚举语义。
const (
	RuleRecordStatusPending            = "pending"
	RuleRecordStatusSuccess            = "success"
	RuleRecordStatusFailed             = "failed"
	RuleRecordStatusUpgradeFailed      = "upgrade_failed"
	RuleRecordStatusUninstallFailedOld = "uninstall_failed_old"
	RuleRecordStatusCancelled          = "cancelled" // 已取消（report 已装但还有 pending）
)

// sync/ack 命令类型（由 RuleTypeCommandName 生成，用于 local-agent ack 区分 skill/rule 路径）
const (
	CommandTypeInstallPromptRule   = "install_prompt_rule"
	CommandTypeInstallRuleRule     = "install_rule_rule"
	CommandTypeUninstallPromptRule = "uninstall_prompt_rule"
	CommandTypeUninstallRuleRule   = "uninstall_rule_rule"
	// 三期：Hook 资源（复用 rule 下发/记录管道）
	CommandTypeInstallHookRule   = "install_hook_rule"
	CommandTypeUninstallHookRule = "uninstall_hook_rule"
)

// 三期：uninstall_teamai 本地任务命令类型（local_agent_tasks.type 对应 ack 路由）。
const (
	CommandTypeUninstallTeamai = "uninstall_teamai"
)

// EnterpriseRule 企业规范（管控端上传的单个 md 文件）。
//
// 与 skill 表结构对齐：字段命名 / 顺序 / 索引风格照抄 skills 表。
// 差异：
//   - 新增 Type（prompt / rule）
//   - COSKey 存单个 md 的 SMH object key（skill 用 zip，字段名 COSZipKey）
//   - ContentSHA256 显式记录（skill 依赖 zip 的 CI 自校，我们没有 zip）
//   - 无 category / 无 security_scan（一期不做）
//   - 保留 changelog / visibility_type（与 skill 对齐，RuleVisibilityGroup 中间表镶位）
//
// 迁移脚本：sql/YYMMDD-local-agent-rules.sql；初始化脚本：sql/init.sql
type EnterpriseRule struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	Identifier      string         `gorm:"uniqueIndex:idx_enterprise_rules_slug_ver_ident;index;default:''" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Slug            string         `gorm:"uniqueIndex:idx_enterprise_rules_slug_ver_ident;not null" json:"slug"`
	Name            string         `gorm:"not null" json:"name"`
	Description     string         `gorm:"type:text;not null;default:''" json:"description"`
	Type            string         `gorm:"type:varchar(16);not null;index" json:"type"`                        // prompt / rule
	Source          string         `gorm:"type:varchar(16);not null;default:'enterprise';index" json:"source"` // enterprise / local
	Version         string         `gorm:"uniqueIndex:idx_enterprise_rules_slug_ver_ident;not null;default:'1.0.0'" json:"version"`
	VersionMajor    int            `gorm:"not null;default:0" json:"-"`
	VersionMinor    int            `gorm:"not null;default:0" json:"-"`
	VersionPatch    int            `gorm:"not null;default:0" json:"-"`
	COSKey          string         `gorm:"not null;default:''" json:"cos_key"`
	FileSize        int64          `gorm:"not null;default:0" json:"file_size"`
	ContentSHA256   string         `gorm:"type:varchar(64);not null;default:''" json:"content_sha256"`
	VisibilityType  string         `gorm:"not null;default:'all'" json:"visibility_type"`            // all / group
	Changelog       string         `gorm:"type:varchar(10000);not null;default:''" json:"changelog"` // 版本更新说明
	DistributeCount int            `gorm:"not null;default:0" json:"distribute_count"`
	// 三期：Hook 资源专属字段（type=hook 时有效，prompt/rule 忽略）
	Event string `gorm:"type:varchar(32); not null;default:''" json:"event"` // 触发时机：SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop
	Cmd   string `gorm:"type:text" json:"cmd"`                            // 执行命令（管理员表单填写）
}

// TableName 表名固定。
func (EnterpriseRule) TableName() string { return "enterprise_rules" }

// ParseVersion 解析 Version 字符串到 major/minor/patch 三个 int 字段。
// 与 Skill.ParseVersion 对齐；解析失败返回 i18n error。
func (r *EnterpriseRule) ParseVersion() error {
	parts := strings.SplitN(r.Version, ".", 3)
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
	r.VersionMajor = major
	r.VersionMinor = minor
	r.VersionPatch = patch
	return nil
}

// LatestVersionRuleIDs 返回 GORM 子查询，选出每个 slug 最新版本的 rule ID。
// 与 LatestVersionSkillIDs 对齐，Model(&EnterpriseRule{}) 保证回调注入 identifier + deleted_at。
func LatestVersionRuleIDs(ctx context.Context) *gorm.DB {
	maxVersions := DB(ctx).Model(&EnterpriseRule{}).
		Select("slug, MAX(version_major * 1000000 + version_minor * 1000 + version_patch) AS max_ver").
		Group("slug")
	return DB(ctx).Model(&EnterpriseRule{}).
		Select("enterprise_rules.id").
		Joins("JOIN (?) mv ON mv.slug = enterprise_rules.slug AND mv.max_ver = enterprise_rules.version_major * 1000000 + enterprise_rules.version_minor * 1000 + enterprise_rules.version_patch", maxVersions)
}

// InheritRuleDistributeCount 从同 slug 的旧版本继承 distribute_count。
// 与 InheritSkillDistributeCount 对齐；无旧版本或旧版本值为 0 时不做任何操作。
func InheritRuleDistributeCount(tx *gorm.DB, slug string, newRuleID uint) error {
	var prev EnterpriseRule
	err := tx.Where("slug = ? AND id != ?", slug, newRuleID).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&prev).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return hcommon.I18nRichError(err, i18n.MsgSkillQueryOldVersionFailed)
	}
	if prev.DistributeCount <= 0 {
		return nil
	}
	if err := tx.Model(&EnterpriseRule{}).Where("id = ?", newRuleID).
		Update("distribute_count", prev.DistributeCount).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillUpdateDistributeCountFailed)
	}
	return nil
}

// RuleDistributionTask 规范下发/卸载任务（每次批量操作创建一条）。
//
// 与 SkillDistributionTask 差别：
//   - 新增 RuleType（prompt / rule）冗余从主表拷贝：sync / pending / ack 全链路不 JOIN 主表就能拿到类型
//   - 无 Source（企业规范只有 enterprise 一种来源）
//   - 无 SourceSkillsetSlug（无 skillset 概念）
//   - Type 只有 distribute / uninstall（对齐 skill）
type RuleDistributionTask struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	Identifier string         `gorm:"index;default:''" json:"-"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	RuleID     uint           `gorm:"index;not null" json:"rule_id"`
	Slug       string         `gorm:"type:varchar(191);not null;default:'';index:idx_rule_dist_tasks_slug" json:"slug"`
	RuleType   string         `gorm:"type:varchar(16);not null;default:'';index:idx_rule_dist_tasks_type" json:"rule_type"` // prompt / rule
	Version    string         `gorm:"not null;default:''" json:"version"`
	BatchID    string         `gorm:"type:varchar(64);not null;default:'';index:idx_rule_dist_tasks_batch" json:"batch_id"`
	OperatorID uint           `gorm:"not null;default:0" json:"operator_id"`
	Total      int            `gorm:"not null;default:0" json:"total"`
	Success    int            `gorm:"not null;default:0" json:"success"`
	Failed     int            `gorm:"not null;default:0" json:"failed"`
	Status     string         `gorm:"not null;default:'running'" json:"status"`  // running / completed
	Type       string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// TableName 表名固定。
func (RuleDistributionTask) TableName() string { return "rule_distribution_tasks" }

// RuleDistributionRecord 规范下发/卸载记录（每个实例一条）。
//
// 与 SkillDistributionRecord 差别：
//   - 无 rule_type 冗余（type 从 task 层通过 task_id JOIN 拿；避免两处写入不一致）
type RuleDistributionRecord struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Identifier  string         `gorm:"index;default:''" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID      uint           `gorm:"index;not null" json:"task_id"`
	RuleID      uint           `gorm:"index;not null" json:"rule_id"`
	InstanceID  uint           `gorm:"index;not null" json:"instance_id"`
	InstanceCID string         `gorm:"column:instance_c_id;not null;default:''" json:"instance_cid"`
	Version     string         `gorm:"not null;default:''" json:"version"`
	Status      string         `gorm:"not null;default:'pending'" json:"status"` // pending / success / failed / upgrade_failed / uninstall_failed_old
	Error       string         `gorm:"type:text" json:"error"`
	Type        string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// TableName 表名固定。
func (RuleDistributionRecord) TableName() string { return "rule_distribution_records" }

// ResolveRuleDistributeFailedStatus 判断规范下发失败时应写入的 status。
// 与 ResolveDistributeFailedStatus 语义对齐（rule 侧独立复制，避免耦合 skill 特有字段）。
func ResolveRuleDistributeFailedStatus(ctx context.Context, instanceID uint, ruleIDs []uint) string {
	if len(ruleIDs) == 0 {
		return RuleRecordStatusFailed
	}
	var count int64
	DB(ctx).Model(&RuleDistributionRecord{}).
		Where("instance_id = ? AND rule_id IN ? AND status = ? AND type = ?",
			instanceID, ruleIDs, RuleRecordStatusSuccess, RuleTaskTypeDistribute).
		Limit(1).Count(&count)
	if count > 0 {
		return RuleRecordStatusUpgradeFailed
	}
	return RuleRecordStatusFailed
}

// ResolveRuleUninstallFailedStatus 判断规范卸载失败时应写入的 status。
// 与 ResolveUninstallFailedStatus 语义对齐。
func ResolveRuleUninstallFailedStatus(ctx context.Context, instanceID uint, ruleIDs []uint, latestVersion string) string {
	if len(ruleIDs) == 0 {
		return RuleRecordStatusFailed
	}
	var lastSuccess RuleDistributionRecord
	err := DB(ctx).Where("instance_id = ? AND rule_id IN ? AND status = ? AND type = ?",
		instanceID, ruleIDs, RuleRecordStatusSuccess, RuleTaskTypeDistribute).
		Order("id DESC").Limit(1).First(&lastSuccess).Error
	if err == nil && lastSuccess.Version != "" && lastSuccess.Version != latestVersion {
		return RuleRecordStatusUninstallFailedOld
	}
	return RuleRecordStatusFailed
}

// RuleVersionScore 将 "x.y.z" 字符串转为整数分数。
// 用于版本比较（sync 时判断是否为更高版本）。解析失败返回 0。
// 与 VersionScore 完全一致，独立复制以保持 rule 侧代码内聚。
func RuleVersionScore(version string) int {
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

// RuleTypeCommandName 从 records.type × rule_type 组合拼出 sync 命令名。
// 4 组合：install_prompt_rule / install_rule_rule / uninstall_prompt_rule / uninstall_rule_rule。
// 未知组合返回空串，调用方应跳过该 record。
func RuleTypeCommandName(recordType, ruleType string) string {
	var action string
	switch recordType {
	case RuleTaskTypeDistribute:
		action = "install"
	case RuleTaskTypeUninstall:
		action = "uninstall"
	default:
		return ""
	}
	switch ruleType {
	case EnterpriseRuleTypePrompt, EnterpriseRuleTypeRule:
		return fmt.Sprintf("%s_%s_rule", action, ruleType)
	case EnterpriseRuleTypeHook:
		// hook 命令名：install_hook_rule / uninstall_hook_rule（与 prompt/rule 命名风格对齐）
		return fmt.Sprintf("%s_%s_rule", action, EnterpriseRuleTypeHook)
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// 规范实例查询（对齐 skill 的 BuildSkillInstanceQuery / InstallStatusCase）
//
// 设计要点：
//   - 主表为 instances（而非 rule_distribution_records），因此从未下发过的本地实例
//     也能出现（LEFT JOIN 无 record 行 → status 推导为 uninstalled）。
//   - rule 只面向本地 agent 实例（source='local'），因此在 WHERE 直接限定。
//   - LEFT JOIN 最新一条 rule_distribution_records（按 MAX(id)），在 SQL 层推导语义状态。
//   - LEFT JOIN local_instance_rules（reporter 上报的当前事实快照），用于改判
//     "records 说装着、但本地已手动卸载" 的实例为 uninstalled。
// -----------------------------------------------------------------------------

// BuildRuleInstanceQuery 构造规范实例查询（主表 instances + LEFT JOIN 最新 record + 事实快照）。
// latestVersion 用于 outdated 判定（已安装但版本低于最新版）。
// slug 用于关联 local_instance_rules 的 (instance_id, slug) 行。
func BuildRuleInstanceQuery(ctx context.Context, ruleIDs []uint, latestVersion, slug string) *gorm.DB {
	// 防御性校验：无效版本号回退为空字符串。
	if RuleVersionScore(latestVersion) == 0 {
		if strings.TrimSpace(latestVersion) != "" {
			slog.WarnContext(ctx, "[EnterpriseRule] invalid latest version; outdated check disabled",
				"latest_version", latestVersion, "slug", slug)
		}
		latestVersion = ""
	}
	selectClause := fmt.Sprintf(`instances.id AS instance_id,
		instances.instance_id AS cvm_instance_id,
		instances.name AS instance_name,
		instances.source AS source,
		COALESCE(u.username, '') AS username,
		%s AS status,
		COALESCE(lr.version, '') AS version,
		COALESCE(lr.error, '') AS error,
		lr.created_at AS created_at`, RuleInstallStatusCase())

	q := DB(ctx).Model(&Instance{}).Select(selectClause)
	// 用绑定参数构造单行派生表，CASE 通过列引用最新版本，
	// 避免在 SELECT/WHERE 表达式中拼接 SQL 字面值。
	q = q.Joins("CROSS JOIN (SELECT ? AS version) rule_latest", latestVersion)

	// users JOIN
	q = q.Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL")

	// rule_distribution_records JOIN — 取每个实例关于该 rule 的最新一条 record（MAX(id)）
	q = q.Joins(`LEFT JOIN rule_distribution_records lr
		ON lr.instance_id = instances.id
		AND lr.deleted_at IS NULL
		AND lr.id = (
			SELECT MAX(r2.id) FROM rule_distribution_records r2
			WHERE r2.rule_id IN ? AND r2.instance_id = instances.id AND r2.deleted_at IS NULL
		)`, ruleIDs)

	// local_instance_rules JOIN — 本地 agent 实例的事实校验（reporter 上报的当前快照）。
	// 注意：local_instance_rules 的唯一约束是 (scope, instance_id, workspace_path, slug)，
	// 同一 slug 在一个实例上可同时存在于 user / project 等多个 scope（每行独立）。
	// 若直接按 (instance_id, slug) JOIN 会把 instances 主表该行扇出成多行，导致
	// /admin/rules/instances 返回重复数据（与 skill/mcp 历史上的 JOIN 扇出 bug 同类）。
	// 因此用聚合子查询取每个 (instance_id, slug) 的 MAX(id) 一行，保证一个实例一个 slug
	// 至多匹配一行 lir；RuleInstallStatusCase 只用 lir.id IS NULL 判 uninstalled，语义不变。
	q = q.Joins(`LEFT JOIN local_instance_rules lir
		ON lir.id = (
			SELECT MAX(l2.id) FROM local_instance_rules l2
			WHERE l2.instance_id = instances.id AND l2.slug = ?
		)`, slug)

	// rule 只面向本地 agent 实例
	q = q.Where("instances.source = ?", InstanceSourceLocal)
	q = q.Where("instances.instance_id != ''")

	return q
}

// RuleInstallStatusCase 返回规范安装状态的 SQL CASE 表达式。
// 用于 SELECT 和 WHERE 子句，确保两处逻辑一致。
// 最新版本由 BuildRuleInstanceQuery 中的 rule_latest.version 提供，
// 其值通过 GORM 参数绑定，不在本表达式中拼接。
//
// 状态判断逻辑（基于最新一条 Record 的 type + status）：
//   - 无记录 → uninstalled
//   - 最新记录是卸载操作（type=uninstall）：
//   - success → uninstalled（卸载成功，恢复未安装状态）
//   - pending → uninstalling（正在卸载中）
//   - failed  → uninstall_failed（卸载失败，规范仍在）
//   - 最新记录是下发操作（type=distribute 或空）：
//   - success + 版本旧 → outdated
//   - success → installed
//   - pending → installing
//   - upgrade_failed → upgrade_failed（升级失败，旧版本仍在）
//   - failed  → failed（首次安装失败，实例上无规范）
//
// MAX(id) 自然处理"卸载后重新下发"的场景——最新的 Record 始终决定当前状态。
//
// 本地 agent 特殊分支：
//   - `local_instance_rules` 是 reporter 上报的当前事实快照。
//   - 对于 source='local' 的实例，如果 record 上说"规范应该在"（installed / outdated /
//     upgrade_failed / uninstall_failed / uninstall_failed_old），但 lir 中没有对应
//     slug 行（lir.id IS NULL），说明用户已经在本地把规范卸掉了，改判 uninstalled。
func RuleInstallStatusCase() string {
	return `CASE
		WHEN lr.status IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  AND instances.source = 'local' AND lir.id IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  THEN 'uninstall_failed'
		WHEN lr.status = 'success' AND rule_latest.version != '' AND lr.version != rule_latest.version THEN 'outdated'
		WHEN lr.status = 'success' AND instances.source = 'local' AND lir.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'success' THEN 'installed'
		WHEN lr.status = 'pending' THEN 'installing'
		WHEN lr.status = 'upgrade_failed' AND instances.source = 'local' AND lir.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
		WHEN lr.status = 'uninstall_failed_old' AND instances.source = 'local' AND lir.id IS NULL THEN 'uninstalled'
		WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
		WHEN lr.status = 'failed'  THEN 'failed'
		ELSE 'uninstalled'
	END`
}
