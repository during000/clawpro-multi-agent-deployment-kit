package model

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// getID extracts the ID field (uint) from any GORM model via reflection.
func getID(ptr any) uint {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// 优先直接取顶层字段，覆盖独立声明 ID 和 gorm.Model 内嵌两种情况
	f := v.FieldByName("ID")
	if !f.IsValid() {
		panic(fmt.Sprintf("model %T has no ID field", ptr))
	}
	return uint(f.Uint())
}

// setID sets the ID field (uint) on any GORM model via reflection.
func setID(ptr any, id uint) {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName("ID")
	if !f.IsValid() {
		panic(fmt.Sprintf("model %T has no ID field", ptr))
	}
	f.SetUint(uint64(id))
}

// tableOf returns the GORM table name for model type T.
// Uses GORM's internal schema cache, so repeated calls are fast.
func tableOf[T any](ctx context.Context) string {
	var zero T
	stmt := &gorm.Statement{DB: DB(ctx)}
	if err := stmt.Parse(&zero); err != nil {
		panic(fmt.Sprintf("failed to parse model %T: %v", zero, err))
	}
	return stmt.Schema.Table
}

// hasSoftDelete checks if model type T has a DeletedAt field (supports both
// direct declaration and embedded via gorm.Model).
func hasSoftDelete[T any]() bool {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if _, ok := t.FieldByName("DeletedAt"); ok {
		return true
	}
	return false
}

// idMapping tracks old_id → new_id per table for FK remapping.
type idMapping map[string]map[uint]uint

func (m idMapping) set(table string, oldID, newID uint) {
	if _, ok := m[table]; !ok {
		m[table] = make(map[uint]uint)
	}
	m[table][oldID] = newID
}

// missingIDs tracks (table, oldID) pairs already warned about, to avoid spamming logs.
var missingIDs = make(map[string]map[uint]bool)

func (m idMapping) get(table string, oldID uint) uint {
	if oldID == 0 {
		return 0
	}
	if tbl, ok := m[table]; ok {
		if newID, ok := tbl[oldID]; ok {
			return newID
		}
	}
	if missingIDs[table] == nil {
		missingIDs[table] = make(map[uint]bool)
	}
	if !missingIDs[table][oldID] {
		missingIDs[table][oldID] = true
		slog.Warn("[migrate] ID not found in mapping", "table", table, "old_id", oldID)
	}
	return 0
}

// MigrateFromSQLite reads all data from the source SQLite database and writes
// it into the current MySQL database under the current identifier (tenant).
// The entire migration runs in a single MySQL transaction for atomicity.
func MigrateFromSQLite(ctx context.Context, sqlitePath string, identifier string) {
	if dbDriver != "mysql" {
		slog.Error("[migrate] --db-migrate requires MySQL backend")
		os.Exit(1)
	}
	if identifier == "" {
		slog.Error("[migrate] --db-migrate requires --identifier to be set")
		os.Exit(1)
	}

	// Check if tenant data already exists
	var existing SiteConfig
	if DB(ctx).First(&existing).Error == nil {
		slog.Info("[migrate] SiteConfig data already exists, skipping migration",
			"identifier", identifier)
		return
	}

	slog.Info("[migrate] Starting SQLite -> MySQL migration",
		"source", sqlitePath, "identifier", identifier)

	// Open source SQLite (read-only)
	srcDB, err := openSQLiteReadOnly(sqlitePath)
	if err != nil {
		slog.Error("[migrate] Failed to open source SQLite", "path", sqlitePath, "error", err)
		os.Exit(1)
	}
	defer closeSQLite(srcDB)

	// Begin MySQL transaction
	tx := DB(ctx).Begin()
	if tx.Error != nil {
		slog.Error("[migrate] Failed to begin transaction", "error", tx.Error)
		os.Exit(1)
	}

	ids := make(idMapping)
	migrated := make(map[string]bool) // tracks which tables have been migrated
	startTime := time.Now()

	// Level 0: root entities (no FK dependencies)
	migrateTable[User](ctx, srcDB, tx, ids, migrated, nil)
	migrated[tableOf[PasswordlessLoginToken](ctx)] = true
	migrateTable[AIModel](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[AIChannel](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[AIImage](ctx, srcDB, tx, ids, migrated, nil)
	migrated[tableOf[ImageHistory](ctx)] = true
	migrateTable[SkillCategory](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[Skill](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[SkillBundle](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[PublicSkill](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[PublicSkillSet](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[SMHSpace](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[OpenClawRole](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[SessionBlacklist](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[OneIDDepartmentRecord](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[Plugin](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[PluginCategory](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[PluginBundle](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[PublicPlugin](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[ResourcePolicy](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[UserGroup](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[OneIDUserProfile](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[McpServer](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[RuleSet](ctx, srcDB, tx, ids, migrated, nil)
	// VpcConfig：本身无 FK 依赖（VisibilityType 控制可见，不指向其他主键）。
	migrateTable[VpcConfig](ctx, srcDB, tx, ids, migrated, nil)
	// TenantDomain：全局表（无 Identifier 隔离），SQLite→MySQL 迁移路径不适用，
	// 但需要标记为已覆盖以通过 checkMigrationCoverage 校验。
	migrated[tableOf[TenantDomain](ctx)] = true
	// FeatureAllowlist：同上，全局表。
	migrated[tableOf[FeatureAllowlist](ctx)] = true
	// EnterpriseRule：企业规范主表（phase2）。无 FK 依赖（slug/version 为字符串快照），
	// 作为根实体在 Level 0 迁移，供下游 rule 分发任务/可见性关联引用其新 ID。
	migrateTable[EnterpriseRule](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[Project](ctx, srcDB, tx, ids, migrated, nil)

	// Level 1: depends on Level 0
	migrateTable[Instance](ctx, srcDB, tx, ids, migrated, func(r *Instance) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.AIModelID = ids.get(tableOf[AIModel](ctx), r.AIModelID)
		r.RoleID = ids.get(tableOf[OpenClawRole](ctx), r.RoleID)
		// stale-instances v1.0：移交相关 FK
		r.HandoverTargetUserID = ids.get(tableOf[User](ctx), r.HandoverTargetUserID)
		r.HandoverRejectedByUserID = ids.get(tableOf[User](ctx), r.HandoverRejectedByUserID)
		// HandoverInitiatedAt 是时间戳，不需要 remap
	})
	migrateTable[BundleSkill](ctx, srcDB, tx, ids, migrated, func(r *BundleSkill) {
		r.SkillBundleID = ids.get(tableOf[SkillBundle](ctx), r.SkillBundleID)
	})
	migrateTable[OpenClawRoleSkill](ctx, srcDB, tx, ids, migrated, func(r *OpenClawRoleSkill) {
		r.OpenClawRoleID = ids.get(tableOf[OpenClawRole](ctx), r.OpenClawRoleID)
	})
	migrateTable[OpenClawRolePlugin](ctx, srcDB, tx, ids, migrated, func(r *OpenClawRolePlugin) {
		r.OpenClawRoleID = ids.get(tableOf[OpenClawRole](ctx), r.OpenClawRoleID)
	})
	migrateTable[SkillCategoryMapping](ctx, srcDB, tx, ids, migrated, func(r *SkillCategoryMapping) {
		r.SkillID = ids.get(tableOf[Skill](ctx), r.SkillID)
		r.CategoryID = ids.get(tableOf[SkillCategory](ctx), r.CategoryID)
	})
	migrateTable[PluginCategoryMapping](ctx, srcDB, tx, ids, migrated, func(r *PluginCategoryMapping) {
		r.PluginID = ids.get(tableOf[Plugin](ctx), r.PluginID)
		r.CategoryID = ids.get(tableOf[PluginCategory](ctx), r.CategoryID)
	})
	migrateTable[BundlePlugin](ctx, srcDB, tx, ids, migrated, func(r *BundlePlugin) {
		r.PluginBundleID = ids.get(tableOf[PluginBundle](ctx), r.PluginBundleID)
	})
	migrateTable[UserGroupMember](ctx, srcDB, tx, ids, migrated, func(r *UserGroupMember) {
		r.UserGroupID = ids.get(tableOf[UserGroup](ctx), r.UserGroupID)
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
	})
	migrateTable[GroupConfigBinding](ctx, srcDB, tx, ids, migrated, func(r *GroupConfigBinding) {
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
		if r.ConfigType != ConfigTypeResourcePolicy {
			return
		}
		oldPolicyID, err := strconv.ParseUint(r.ConfigKey, 10, 64)
		if err != nil {
			slog.Warn(
				"[migrate] invalid resource policy binding key",
				"config_key", r.ConfigKey,
				"error", err,
			)
			return
		}
		r.ConfigKey = strconv.FormatUint(uint64(ids.get(tableOf[ResourcePolicy](ctx), uint(oldPolicyID))), 10)
	})
	migrateTable[ProjectMember](ctx, srcDB, tx, ids, migrated, func(r *ProjectMember) {
		r.ProjectID = ids.get(tableOf[Project](ctx), r.ProjectID)
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.CreatedBy = ids.get(tableOf[User](ctx), r.CreatedBy)
	})
	migrateTable[ProjectConfigBinding](ctx, srcDB, tx, ids, migrated, func(r *ProjectConfigBinding) {
		r.ProjectID = ids.get(tableOf[Project](ctx), r.ProjectID)
	})
	migrateTable[McpVersion](ctx, srcDB, tx, ids, migrated, func(r *McpVersion) {
		r.MCPID = ids.get(tableOf[McpServer](ctx), r.MCPID)
	})
	migrateTable[McpHostedKey](ctx, srcDB, tx, ids, migrated, func(r *McpHostedKey) {
		r.MCPID = ids.get(tableOf[McpServer](ctx), r.MCPID)
	})
	migrateTable[ManagedSGPool](ctx, srcDB, tx, ids, migrated, func(r *ManagedSGPool) {
		r.RuleSetID = ids.get(tableOf[RuleSet](ctx), r.RuleSetID)
	})
	// GroupClosure 没有自增 ID（复合主键 identifier+ancestor+descendant），不能用泛型
	// migrateTable（依赖 ID 字段做新旧映射）。这里用专用函数：read → remap → 批量插入。
	migrateGroupClosureTable(ctx, srcDB, tx, ids, migrated, identifier)

	// Level 2: depends on Level 0+1
	// 这三个大表的 ID 没有下游引用，使用批量插入提升性能
	migrateBulkTable[LLMUsageLog](ctx, srcDB, tx, ids, migrated, func(r *LLMUsageLog) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.AIModelID = ids.get(tableOf[AIModel](ctx), r.AIModelID)
	})
	migrateDailyUsageSummaryTable(ctx, srcDB, tx, ids, migrated, identifier)
	migrateBulkTable[AuditLog](ctx, srcDB, tx, ids, migrated, func(r *AuditLog) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
	})
	migrateTable[SkillDistributionTask](ctx, srcDB, tx, ids, migrated, func(r *SkillDistributionTask) {
		r.SkillID = ids.get(tableOf[Skill](ctx), r.SkillID)
		r.OperatorID = ids.get(tableOf[User](ctx), r.OperatorID)
	})
	migrateTable[SMHPersonalSpace](ctx, srcDB, tx, ids, migrated, func(r *SMHPersonalSpace) {
		r.UserId = ids.get(tableOf[User](ctx), r.UserId)
		r.InstanceId = ids.get(tableOf[Instance](ctx), r.InstanceId)
	})
	migrateTable[Notification](ctx, srcDB, tx, ids, migrated, func(r *Notification) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// stale-instances v1.0：实例标记表（依赖 instances）
	migrateTable[InstanceFlag](ctx, srcDB, tx, ids, migrated, func(r *InstanceFlag) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// stale-instances v1.0：处理记录表（依赖 instances + users + user_groups + notifications）
	migrateTable[InstanceChangeGroupRecord](ctx, srcDB, tx, ids, migrated, func(r *InstanceChangeGroupRecord) {
		r.InstancePK = ids.get(tableOf[Instance](ctx), r.InstancePK)
		r.UserIDBefore = ids.get(tableOf[User](ctx), r.UserIDBefore)
		r.UserIDAfter = ids.get(tableOf[User](ctx), r.UserIDAfter)
		r.GroupIDBefore = ids.get(tableOf[UserGroup](ctx), r.GroupIDBefore)
		r.GroupIDAfter = ids.get(tableOf[UserGroup](ctx), r.GroupIDAfter)
		// actor_type=oneid_sync 时 actor_id=0；ids.get(_,0) 返回 0 兼容
		r.ActorID = ids.get(tableOf[User](ctx), r.ActorID)
		r.NotificationID = ids.get(tableOf[Notification](ctx), r.NotificationID)
		// instance_id（CVM ins-xxx 字符串）不参与 remap
	})
	migrateTable[MemoryTDAIPlugin](ctx, srcDB, tx, ids, migrated, nil) // InstanceID is string, no remap
	migrateTable[TdaiJob](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[InstanceAdjustment](ctx, srcDB, tx, ids, migrated, func(r *InstanceAdjustment) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateBulkTable[SGDrainState](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[SkillInstallation](ctx, srcDB, tx, ids, migrated, func(r *SkillInstallation) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateTable[PluginDistributionTask](ctx, srcDB, tx, ids, migrated, func(r *PluginDistributionTask) {
		r.PluginDBID = ids.get(tableOf[Plugin](ctx), r.PluginDBID)
		r.OperatorID = ids.get(tableOf[User](ctx), r.OperatorID)
	})
	migrateTable[PluginInstallation](ctx, srcDB, tx, ids, migrated, func(r *PluginInstallation) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// 本地 agent 实例心跳/主机信息（一对一附属于 Instance）。
	migrateTable[LocalInstanceInfo](ctx, srcDB, tx, ids, migrated, func(r *LocalInstanceInfo) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// 本地 agent 已装 skill 快照（多对一附属于 Instance）。
	// Slug/Version 是字符串快照，不指向 Skill 主键，无需 remap。
	migrateTable[LocalInstanceSkill](ctx, srcDB, tx, ids, migrated, func(r *LocalInstanceSkill) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// 通用本地 Agent 任务：关联实例、发起用户与可选项目。
	migrateTable[LocalAgentTask](ctx, srcDB, tx, ids, migrated, func(r *LocalAgentTask) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.OperatorID = ids.get(tableOf[User](ctx), r.OperatorID)
		if r.ProjectID > 0 {
			r.ProjectID = ids.get(tableOf[Project](ctx), r.ProjectID)
		}
	})
	migrateTable[LocalAgentScopeBinding](ctx, srcDB, tx, ids, migrated, func(r *LocalAgentScopeBinding) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
		// project_id 不设外键：项目物理删除后仍需保留原始值，供本地 Workspace
		// 兼容展示和后续人工重新绑定，因此迁移时不可因目标项目不存在而归零。
	})
	// 资产版本记录（AssetVersionRecord）：运行时生成的版本历史，SQLite→MySQL 迁移无需搬历史数据。
	// 标记为已覆盖以通过 checkMigrationCoverage 校验。
	migrated[tableOf[AssetVersionRecord](ctx)] = true
	// 本地 agent CLS 凭据（按租户隔离，无外键 remap）。
	migrateTable[LocalAgentCLSCredential](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[McpDistributionTask](ctx, srcDB, tx, ids, migrated, func(r *McpDistributionTask) {
		r.MCPID = ids.get(tableOf[McpServer](ctx), r.MCPID)
	})
	migrateTable[Tag](ctx, srcDB, tx, ids, migrated, nil)
	migrateTable[TagVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *TagVisibilityGroup) {
		r.TagID = ids.get(tableOf[Tag](ctx), r.TagID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	migrateTable[ModelVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *ModelVisibilityGroup) {
		r.AIModelID = ids.get(tableOf[AIModel](ctx), r.AIModelID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	migrateTable[SkillVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *SkillVisibilityGroup) {
		r.SkillID = ids.get(tableOf[Skill](ctx), r.SkillID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	// RuleDistributionTask：规范下发/卸载任务（phase2）。
	// 依赖 EnterpriseRule（RuleID）+ User（OperatorID）。
	migrateTable[RuleDistributionTask](ctx, srcDB, tx, ids, migrated, func(r *RuleDistributionTask) {
		r.RuleID = ids.get(tableOf[EnterpriseRule](ctx), r.RuleID)
		r.OperatorID = ids.get(tableOf[User](ctx), r.OperatorID)
	})
	// RuleVisibilityGroup：规范-分组可见性关联（phase2）。
	// 依赖 EnterpriseRule（RuleID）+ UserGroup（GroupID）。
	migrateTable[RuleVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *RuleVisibilityGroup) {
		r.RuleID = ids.get(tableOf[EnterpriseRule](ctx), r.RuleID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	migrateTable[SkillBundleVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *SkillBundleVisibilityGroup) {
		r.SkillBundleID = ids.get(tableOf[SkillBundle](ctx), r.SkillBundleID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	migrateTable[RoleVisibilityGroup](ctx, srcDB, tx, ids, migrated, func(r *RoleVisibilityGroup) {
		r.OpenClawRoleID = ids.get(tableOf[OpenClawRole](ctx), r.OpenClawRoleID)
		r.GroupID = ids.get(tableOf[UserGroup](ctx), r.GroupID)
	})
	migrateTable[InstanceModel](ctx, srcDB, tx, ids, migrated, func(r *InstanceModel) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.AIModelID = ids.get(tableOf[AIModel](ctx), r.AIModelID)
	})
	migrateTable[AgentMigration](ctx, srcDB, tx, ids, migrated, func(r *AgentMigration) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateTable[SkillSecurityScan](ctx, srcDB, tx, ids, migrated, func(r *SkillSecurityScan) {
		r.SkillID = ids.get(tableOf[Skill](ctx), r.SkillID)
	})
	// AgentCommand 模板表，仅依赖 User（CreatedByUserID）。
	// 后续 AgentCommandInvocation 通过 CommandID 引用其 ID，故必须用 migrateTable
	// 跟踪新旧 ID 映射，不能用 migrateBulkTable。
	migrateTable[AgentCommand](ctx, srcDB, tx, ids, migrated, func(r *AgentCommand) {
		r.CreatedByUserID = ids.get(tableOf[User](ctx), r.CreatedByUserID)
	})

	// Level 3: depends on Level 2
	migrateTable[SkillScanViolation](ctx, srcDB, tx, ids, migrated, func(r *SkillScanViolation) {
		r.SkillSecurityScanID = ids.get(tableOf[SkillSecurityScan](ctx), r.SkillSecurityScanID)
	})
	migrateTable[SkillDistributionRecord](ctx, srcDB, tx, ids, migrated, func(r *SkillDistributionRecord) {
		r.TaskID = ids.get(tableOf[SkillDistributionTask](ctx), r.TaskID)
		r.SkillID = ids.get(tableOf[Skill](ctx), r.SkillID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// RuleDistributionRecord：规范下发/卸载记录（phase2）。
	// 依赖 RuleDistributionTask（TaskID）+ EnterpriseRule（RuleID）+ Instance（InstanceID）。
	migrateTable[RuleDistributionRecord](ctx, srcDB, tx, ids, migrated, func(r *RuleDistributionRecord) {
		r.TaskID = ids.get(tableOf[RuleDistributionTask](ctx), r.TaskID)
		r.RuleID = ids.get(tableOf[EnterpriseRule](ctx), r.RuleID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	// LocalInstanceRule：本地 agent 实例已装规范快照（phase2）。
	// 对称 LocalInstanceSkill（硬删、无 DeletedAt），仅依赖 Instance（InstanceID）。
	// Slug/Version 为字符串快照，不指向 EnterpriseRule 主键，无需 remap。
	migrateTable[LocalInstanceRule](ctx, srcDB, tx, ids, migrated, func(r *LocalInstanceRule) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateTable[RoleDistributionRecord](ctx, srcDB, tx, ids, migrated, func(r *RoleDistributionRecord) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.RoleID = ids.get(tableOf[OpenClawRole](ctx), r.RoleID)
		r.OperatorID = ids.get(tableOf[User](ctx), r.OperatorID)
	})
	migrateTable[PluginDistributionRecord](ctx, srcDB, tx, ids, migrated, func(r *PluginDistributionRecord) {
		r.TaskID = ids.get(tableOf[PluginDistributionTask](ctx), r.TaskID)
		r.PluginDBID = ids.get(tableOf[Plugin](ctx), r.PluginDBID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateTable[McpDistributionRecord](ctx, srcDB, tx, ids, migrated, func(r *McpDistributionRecord) {
		r.TaskID = ids.get(tableOf[McpDistributionTask](ctx), r.TaskID)
		r.MCPID = ids.get(tableOf[McpServer](ctx), r.MCPID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})
	migrateTable[McpInstallation](ctx, srcDB, tx, ids, migrated, func(r *McpInstallation) {
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
		r.MCPID = ids.get(tableOf[McpServer](ctx), r.MCPID)
		r.LastTaskID = ids.get(tableOf[McpDistributionTask](ctx), r.LastTaskID)
	})

	// AgentCommandDispatch：依赖 AgentCommand + User。后续 AgentCommandInvocation
	// 与 AgentCommandTask 通过 DispatchID 引用其 ID，故必须用 migrateTable 跟踪映射。
	migrateTable[AgentCommandDispatch](ctx, srcDB, tx, ids, migrated, func(r *AgentCommandDispatch) {
		r.CommandID = ids.get(tableOf[AgentCommand](ctx), r.CommandID)
		r.TriggeredByUserID = ids.get(tableOf[User](ctx), r.TriggeredByUserID)
		r.TestTargetInstanceID = ids.get(tableOf[Instance](ctx), r.TestTargetInstanceID)
	})

	// AgentCommandInvocation：依赖 AgentCommandDispatch。后续 AgentCommandTask 通过
	// InvocationID 引用其 ID，必须用 migrateTable 跟踪映射。
	migrateTable[AgentCommandInvocation](ctx, srcDB, tx, ids, migrated, func(r *AgentCommandInvocation) {
		r.DispatchID = ids.get(tableOf[AgentCommandDispatch](ctx), r.DispatchID)
	})
	// AgentCommandTask：依赖 AgentCommandInvocation + AgentCommandDispatch + Instance。
	// 其 ID 没有任何下游表引用，使用 migrateBulkTable 批量插入提升大表性能。
	migrateBulkTable[AgentCommandTask](ctx, srcDB, tx, ids, migrated, func(r *AgentCommandTask) {
		r.DispatchID = ids.get(tableOf[AgentCommandDispatch](ctx), r.DispatchID)
		r.InvocationID = ids.get(tableOf[AgentCommandInvocation](ctx), r.InvocationID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})

	// AgentCommandSchedule：定时任务配置，依赖 AgentCommand + User。
	// InstanceIDsJSON 引用 Instance，但以 JSON 存储，跨租户迁移时一并重映射 ID 避免悬空引用。
	migrateTable[AgentCommandSchedule](ctx, srcDB, tx, ids, migrated, func(r *AgentCommandSchedule) {
		r.CommandID = ids.get(tableOf[AgentCommand](ctx), r.CommandID)
		r.CreatedByUserID = ids.get(tableOf[User](ctx), r.CreatedByUserID)
		if oldIDs := r.InstanceIDsList(); len(oldIDs) > 0 {
			newIDs := make([]uint, 0, len(oldIDs))
			for _, oid := range oldIDs {
				newIDs = append(newIDs, ids.get(tableOf[Instance](ctx), oid))
			}
			_ = r.SetInstanceIDs(newIDs)
		}
	})

	// AgentCommandScheduleRecord：依赖 AgentCommandSchedule。无下游引用，批量迁移即可。
	migrateBulkTable[AgentCommandScheduleRecord](ctx, srcDB, tx, ids, migrated, func(r *AgentCommandScheduleRecord) {
		r.ScheduleID = ids.get(tableOf[AgentCommandSchedule](ctx), r.ScheduleID)
	})

	// Doctor 诊断会话表（依赖 User + Instance）
	migrateTable[DoctorSession](ctx, srcDB, tx, ids, migrated, func(r *DoctorSession) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.TargetInstanceID = ids.get(tableOf[Instance](ctx), r.TargetInstanceID)
		if r.DoctorInstanceID != nil && *r.DoctorInstanceID != 0 {
			newID := ids.get(tableOf[Instance](ctx), *r.DoctorInstanceID)
			r.DoctorInstanceID = &newID
		}
	})

	migrateTable[DoctorAuthorization](ctx, srcDB, tx, ids, migrated, func(r *DoctorAuthorization) {
		r.UserID = ids.get(tableOf[User](ctx), r.UserID)
		r.InstanceID = ids.get(tableOf[Instance](ctx), r.InstanceID)
	})

	// 技能共建审核：ReviewRequest 需 remap requester_id / reviewer_id / resource_id
	// resource_id 指向 Skill.ID，已通过 migrateTable[Skill] 完成 remap。
	migrateTable[ReviewRequest](ctx, srcDB, tx, ids, migrated, func(r *ReviewRequest) {
		r.RequesterID = ids.get(tableOf[User](ctx), r.RequesterID)
		r.ReviewerID = ids.get(tableOf[User](ctx), r.ReviewerID)
		// resource_id 指向 Skill.ID，需 remap
		if r.ResourceType == ResourceTypeSkill {
			r.ResourceID = ids.get(tableOf[Skill](ctx), r.ResourceID)
		}
	})

	// Final: SiteConfig 最后写入，作为迁移完成的标记。
	migrateTable[SiteConfig](ctx, srcDB, tx, ids, migrated, func(r *SiteConfig) {
		if r.DefaultModelID != 0 {
			r.DefaultModelID = ids.get(tableOf[AIModel](ctx), r.DefaultModelID)
		}
	})

	// Validate: ensure all models in allModels are covered by migration
	if missing := checkMigrationCoverage(ctx, migrated); len(missing) > 0 {
		tx.Rollback()
		slog.Error("[migrate] Migration aborted: some models in allModels are not covered by migration logic",
			"missing_tables", missing)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		slog.Error("[migrate] Failed to commit transaction", "error", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)
	slog.Info("[migrate] Migration completed successfully",
		"identifier", identifier,
		"duration", elapsed.Round(time.Millisecond))
}

// migrateTable reads all records of type T from SQLite, remaps FKs, and inserts into MySQL within tx.
// Table name and soft-delete behavior are auto-detected from the GORM model.
// remap: optional function to remap FK fields using idMapping.
func migrateTable[T any](ctx context.Context, srcDB *gorm.DB, tx *gorm.DB, ids idMapping, migrated map[string]bool, remap func(*T)) {
	tableName := tableOf[T](ctx)
	migrated[tableName] = true
	var records []T
	query := srcDB
	if hasSoftDelete[T]() {
		query = query.Unscoped()
	}
	if err := query.Find(&records).Error; err != nil {
		tx.Rollback()
		slog.Error("[migrate] Failed to read from SQLite", "table", tableName, "error", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		slog.Info("[migrate] No records to migrate", "table", tableName)
		return
	}

	for i := range records {
		p := &records[i]
		oldID := getID(p)
		setID(p, 0) // let MySQL assign new ID

		if remap != nil {
			remap(p)
		}

		if err := tx.Create(p).Error; err != nil {
			tx.Rollback()
			slog.Error("[migrate] Failed to insert into MySQL",
				"table", tableName, "old_id", oldID, "error", err)
			os.Exit(1)
		}

		ids.set(tableName, oldID, getID(p))
	}

	slog.Info("[migrate] Migrated table", "table", tableName, "count", len(records))
}

// migrateBulkTable reads large tables in batches, remaps FKs, and bulk-inserts into MySQL.
// Unlike migrateTable, it does NOT track ID mappings — use only for tables whose IDs
// are not referenced as FK by any other table.
func migrateBulkTable[T any](ctx context.Context, srcDB *gorm.DB, tx *gorm.DB, ids idMapping, migrated map[string]bool, remap func(*T)) {
	tableName := tableOf[T](ctx)
	migrated[tableName] = true
	const batchSize = 500
	var total int

	query := srcDB.Session(&gorm.Session{})
	if hasSoftDelete[T]() {
		query = query.Unscoped()
	}

	var batch []T
	result := query.FindInBatches(&batch, batchSize, func(batchTx *gorm.DB, batchNum int) error {
		for i := range batch {
			p := &batch[i]
			setID(p, 0)

			if remap != nil {
				remap(p)
			}
		}

		if err := tx.CreateInBatches(&batch, batchSize).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgMigrateBulkInsert, batchNum)
		}

		total += len(batch)
		if total%5000 == 0 || batchNum == 1 {
			slog.Info("[migrate] Batch progress", "table", tableName, "processed", total)
		}
		return nil
	})

	if result.Error != nil {
		tx.Rollback()
		slog.Error("[migrate] Failed to migrate bulk table",
			"table", tableName, "error", result.Error)
		os.Exit(1)
	}

	slog.Info("[migrate] Migrated table", "table", tableName, "count", total)
}

type dailyUsageSummaryKey struct {
	Identifier string
	Date       time.Time
	UserID     uint
	InstanceID uint
	AIModelID  uint
}

func dateOnlyUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// normalizeMigratedDailyUsageSummaries collapses date values to UTC date-only
// labels and merges rows that would collide on MySQL's composite unique key.
//
// This is intentionally a migration-time compatibility fix for historical bad
// rows in SQLite. We only preserve the UTC calendar day already encoded in the
// stored value and truncate the time portion to 00:00:00, without reinterpreting
// the value through the current business timezone.
func normalizeMigratedDailyUsageSummaries(records []DailyUsageSummary, identifier string) []DailyUsageSummary {
	merged := make([]DailyUsageSummary, 0, len(records))
	indexByKey := make(map[dailyUsageSummaryKey]int, len(records))

	for _, record := range records {
		record.ID = 0
		record.Date = dateOnlyUTC(record.Date)
		if identifier != "" {
			record.Identifier = identifier
		}

		key := dailyUsageSummaryKey{
			Identifier: record.Identifier,
			Date:       record.Date,
			UserID:     record.UserID,
			InstanceID: record.InstanceID,
			AIModelID:  record.AIModelID,
		}
		if idx, ok := indexByKey[key]; ok {
			merged[idx].PromptTokens += record.PromptTokens
			merged[idx].CompletionTokens += record.CompletionTokens
			merged[idx].TotalTokens += record.TotalTokens
			merged[idx].RequestCount += record.RequestCount
			continue
		}

		indexByKey[key] = len(merged)
		merged = append(merged, record)
	}

	return merged
}

// migrateGroupClosureTable 把 group_closure 表从 SQLite 复制到 MySQL。
// 该表无自增 ID（复合主键 identifier+ancestor_id+descendant_id），所以不能复用
// 泛型 migrateTable（依赖 setID(0) 让 MySQL 自增）。流程：
//  1. 读 SQLite 全表
//  2. 把 ancestor_id / descendant_id 按 user_groups 的 idMapping 折射成新 ID
//  3. 批量 CreateInBatches 写入 MySQL
//
// 折射时若旧 ID 不在 mapping（理论上不应发生）会落到 0；这种行直接跳过避免脏数据。
func migrateGroupClosureTable(ctx context.Context, srcDB *gorm.DB, tx *gorm.DB, ids idMapping, migrated map[string]bool, identifier string) {
	tableName := tableOf[GroupClosure](ctx)
	migrated[tableName] = true

	var records []GroupClosure
	if err := srcDB.Find(&records).Error; err != nil {
		tx.Rollback()
		slog.Error("[migrate] Failed to read from SQLite", "table", tableName, "error", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		slog.Info("[migrate] No records to migrate", "table", tableName)
		return
	}

	ugTable := tableOf[UserGroup](ctx)
	out := make([]GroupClosure, 0, len(records))
	skipped := 0
	for _, r := range records {
		newAnc := ids.get(ugTable, r.AncestorID)
		newDesc := ids.get(ugTable, r.DescendantID)
		if newAnc == 0 || newDesc == 0 {
			// 任一端 user_group 没在新库里 → 闭包行作废，跳过
			skipped++
			continue
		}
		r.AncestorID = newAnc
		r.DescendantID = newDesc
		if identifier != "" {
			r.Identifier = identifier
		}
		out = append(out, r)
	}

	if len(out) > 0 {
		const batchSize = 500
		if err := tx.CreateInBatches(out, batchSize).Error; err != nil {
			tx.Rollback()
			slog.Error("[migrate] Failed to insert into MySQL",
				"table", tableName, "error", err)
			os.Exit(1)
		}
	}

	attrs := []any{"table", tableName, "count", len(out)}
	if skipped > 0 {
		attrs = append(attrs, "skipped", skipped, "source_count", len(records))
	}
	slog.Info("[migrate] Migrated table", attrs...)
}

func migrateDailyUsageSummaryTable(ctx context.Context, srcDB *gorm.DB, tx *gorm.DB, ids idMapping, migrated map[string]bool, identifier string) {
	tableName := tableOf[DailyUsageSummary](ctx)
	migrated[tableName] = true

	var records []DailyUsageSummary
	if err := srcDB.Find(&records).Error; err != nil {
		tx.Rollback()
		slog.Error("[migrate] Failed to read from SQLite", "table", tableName, "error", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		slog.Info("[migrate] No records to migrate", "table", tableName)
		return
	}

	for i := range records {
		records[i].UserID = ids.get(tableOf[User](ctx), records[i].UserID)
		records[i].InstanceID = ids.get(tableOf[Instance](ctx), records[i].InstanceID)
		records[i].AIModelID = ids.get(tableOf[AIModel](ctx), records[i].AIModelID)
	}

	normalized := normalizeMigratedDailyUsageSummaries(records, identifier)

	const batchSize = 500
	for start := 0; start < len(normalized); start += batchSize {
		end := start + batchSize
		if end > len(normalized) {
			end = len(normalized)
		}
		if err := tx.CreateInBatches(normalized[start:end], batchSize).Error; err != nil {
			tx.Rollback()
			slog.Error("[migrate] Failed to insert into MySQL",
				"table", tableName,
				"batch_start", start,
				"batch_end", end,
				"error", err)
			os.Exit(1)
		}
	}

	attrs := []any{"table", tableName, "count", len(normalized)}
	if len(normalized) != len(records) {
		attrs = append(attrs, "source_count", len(records), "merged_count", len(records)-len(normalized))
	}
	slog.Info("[migrate] Migrated table", attrs...)
}

// checkMigrationCoverage compares the migrated table set against allModels.
// Returns a list of table names present in allModels but missing from migration.
func checkMigrationCoverage(ctx context.Context, migrated map[string]bool) []string {
	var missing []string
	for _, m := range allModels {
		stmt := &gorm.Statement{DB: DB(ctx)}
		if err := stmt.Parse(m); err != nil {
			slog.Warn("[migrate] Failed to parse model for table name", "model", fmt.Sprintf("%T", m), "error", err)
			continue
		}
		tableName := stmt.Schema.Table
		if !migrated[tableName] {
			missing = append(missing, tableName)
		}
	}
	return missing
}

func openSQLiteReadOnly(path string) (*gorm.DB, error) {
	dsn := path + "?mode=ro&_pragma=foreign_keys(OFF)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMigrateOpenSQLite, path)
	}
	return db, nil
}

func closeSQLite(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}
