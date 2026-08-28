package model

import (
	"context"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Plugin 企业插件
type Plugin struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	Identifier      string         `gorm:"uniqueIndex:idx_plugin_slug_version_identifier;index;default:''" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Slug            string         `gorm:"uniqueIndex:idx_plugin_slug_version_identifier;not null" json:"slug"`
	Name            string         `gorm:"not null" json:"name"`
	Description     string         `gorm:"type:text;not null" json:"description"`
	Version         string         `gorm:"uniqueIndex:idx_plugin_slug_version_identifier;not null;default:'1.0.0'" json:"version"`
	VersionMajor    int            `gorm:"not null;default:0" json:"-"`
	VersionMinor    int            `gorm:"not null;default:0" json:"-"`
	VersionPatch    int            `gorm:"not null;default:0" json:"-"`
	PluginID        string         `gorm:"not null;default:''" json:"plugin_id"`             // openclaw.plugin.json 中的 id
	PluginFormat    string         `gorm:"not null;default:'openclaw'" json:"plugin_format"` // "openclaw" | "bundle"
	Kind            string         `gorm:"not null;default:''" json:"kind"`                  // "memory" | "context-engine" | ""
	COSZipKey       string         `gorm:"not null;default:''" json:"cos_zip_key"`
	COSDirKey       string         `gorm:"not null;default:''" json:"cos_dir_key"`
	FileList        string         `gorm:"type:text" json:"file_list"`
	FileSize        int64          `gorm:"not null;default:0" json:"file_size"`
	NpmPackage      string         `gorm:"not null;default:''" json:"npm_package"`                   // npm 包名（可选）
	ConfigSchema    string         `gorm:"type:text" json:"config_schema"`                           // configSchema JSON
	Providers       string         `gorm:"type:text" json:"providers"`                               // providers JSON 数组
	Channels        string         `gorm:"type:text" json:"channels"`                                // channels JSON 数组
	Changelog       string         `gorm:"type:varchar(10000);not null;default:''" json:"changelog"` // 版本更新说明
	DistributeCount int            `gorm:"not null;default:0" json:"distribute_count"`               // 累计下发成功数
	VisibilityType  string         `gorm:"not null;default:'all'" json:"visibility_type"`            // 应用范围：all=全部用户可见, group=按分组可见
}

// ParseVersion 解析版本号为 major/minor/patch
func (p *Plugin) ParseVersion() error {
	parts := strings.SplitN(p.Version, ".", 3)
	if len(parts) != 3 {
		return hcommon.I18nError(i18n.MsgPluginVersionFormatInvalid, p.Version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return hcommon.I18nError(i18n.MsgPluginVersionMajorInvalid, p.Version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return hcommon.I18nError(i18n.MsgPluginVersionMinorInvalid, p.Version)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return hcommon.I18nError(i18n.MsgPluginVersionPatchInvalid, p.Version)
	}
	// 校验版本号范围，防止 LatestVersionPluginIDs 中的版本计算溢出
	if major > 999 || minor > 999 || patch > 999 {
		return hcommon.I18nError(i18n.MsgPluginVersionExceeds999, p.Version)
	}
	if major < 0 || minor < 0 || patch < 0 {
		return hcommon.I18nError(i18n.MsgPluginVersionNegative, p.Version)
	}
	p.VersionMajor = major
	p.VersionMinor = minor
	p.VersionPatch = patch
	return nil
}

// LatestVersionPluginIDs 返回 GORM 子查询，选出每个 slug 最新版本的 plugin ID。
func LatestVersionPluginIDs(ctx context.Context) *gorm.DB {
	maxVersions := DB(ctx).Model(&Plugin{}).
		Select("slug, MAX(version_major * 1000000 + version_minor * 1000 + version_patch) AS max_ver").
		Group("slug")
	return DB(ctx).Model(&Plugin{}).
		Select("plugins.id").
		Joins("JOIN (?) mv ON mv.slug = plugins.slug AND mv.max_ver = plugins.version_major * 1000000 + plugins.version_minor * 1000 + plugins.version_patch", maxVersions)
}

// PluginInstallStatusCase 返回完整的 CASE 表达式，区分下发/卸载/待更新等状态。
// latestVersion 必须是合法的 x.y.z 格式，否则返回 error。
func PluginInstallStatusCase(latestVersion string) (string, error) {
	safeVersion, err := sanitizeVersion(latestVersion)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgPluginVersionFormatInvalid, latestVersion)
	}
	return fmt.Sprintf(`CASE
		WHEN lr.status IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  THEN 'uninstall_failed'
		WHEN lr.status = 'success' AND lr.version != '%s' THEN 'outdated'
		WHEN lr.status = 'success' THEN 'installed'
		WHEN lr.status = 'pending' THEN 'installing'
		WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
		WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
		WHEN lr.status = 'failed'  THEN 'failed'
		ELSE 'uninstalled'
	END`, safeVersion), nil
}

// sanitizeVersion 解析版本号并用解析后的数字重新拼接，彻底消除 SQL 注入风险。
func sanitizeVersion(version string) (string, error) {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) != 3 {
		return "", hcommon.I18nError(i18n.MsgPluginVersionFormatInvalid, version)
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return "", hcommon.I18nError(i18n.MsgPluginVersionParseFailed, version)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// // BuildPluginInstanceQuery 构造实例安装状态查询。
// 使用 Model(&Instance{}) 保证 GORM 回调自动注入 identifier 和 deleted_at 条件。
// 只查询支持插件的实例类型（通过 AgentType.SupportsPlugin 配置判断）。
// 注意：不在 SQL 层过滤 last_cvm_state='RUNNING'，由调用方通过 CVM API 实时判断运行状态后在内存中过滤。
func BuildPluginInstanceQuery(ctx context.Context, pluginIDs []uint, latestVersion string) (*gorm.DB, error) {
	caseClause, err := PluginInstallStatusCase(latestVersion)
	if err != nil {
		return nil, err
	}
	safeVersion, err := sanitizeVersion(latestVersion)
	if err != nil {
		return nil, err
	}
	selectClause := "instances.id AS instance_id, " +
		"instances.instance_id AS cvm_instance_id, " +
		"instances.name AS instance_name, " +
		"COALESCE(instances.agent_type, '') AS instance_type, " +
		"instances.user_id AS user_id, " +
		"instances.source AS source, " +
		"instances.last_cvm_state AS last_cvm_state, " +
		"instances.last_stable_state AS last_stable_state, " +
		"instances.current_operation AS current_operation, " +
		"instances.current_operation_state AS current_operation_state, " +
		"instances.agent_ready AS agent_ready, " +
		"instances.cls_agent_status AS cls_agent_status, " +
		"instances.cls_agent_status_at AS cls_agent_status_at, " +
		"COALESCE(u.username, '') AS username, " +
		caseClause + " AS install_status, " +
		"COALESCE(lr.version, '') AS version, " +
		"'" + safeVersion + "' AS latest_version"

	q := DB(ctx).Model(&Instance{}).Select(selectClause)

	// users JOIN — user_id 来自已隔离的 instances，无需重复过滤 identifier
	q = q.Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL")

	// plugin_distribution_records JOIN — instance_id 和 plugin_db_id 均来自已隔离数据，
	// 关联到的 records 必然属于当前租户，无需额外 identifier 过滤
	q = q.Joins(`LEFT JOIN plugin_distribution_records lr
		ON lr.instance_id = instances.id
		AND lr.deleted_at IS NULL
		AND lr.id = (
			SELECT MAX(r2.id) FROM plugin_distribution_records r2
			WHERE r2.plugin_db_id IN ? AND r2.instance_id = instances.id AND r2.deleted_at IS NULL
		)`, pluginIDs)

	// deleted_at 和 identifier 由 GORM 回调自动注入，无需手动添加
	// 只查询支持插件的实例类型（通过 AgentType.SupportsPlugin 配置判断）
	// 不过滤 last_cvm_state，由调用方通过实时 CVM API 判断
	supportedTypes := GetPluginSupportedAgentTypes(ctx)
	q = q.Where("instances.instance_id != '' AND instances.agent_type IN ?", supportedTypes)

	// 排序：installed/installing 排最后，其余 created_at DESC
	q = q.Order(`CASE
		WHEN (` + caseClause + `) IN ('installed', 'installing') THEN 1
		ELSE 0
	END ASC, instances.created_at DESC`)

	return q, nil
}

// PluginDistributionTask 插件下发任务（每次批量下发创建一个 Task）
type PluginDistributionTask struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	Identifier string         `gorm:"index;default:''" json:"-"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	PluginDBID uint           `gorm:"index;not null" json:"plugin_db_id"` // 关联 Plugin 表的 ID
	Version    string         `gorm:"not null;default:''" json:"version"`
	OperatorID uint           `gorm:"not null;default:0" json:"operator_id"`
	Total      int            `gorm:"not null;default:0" json:"total"`
	Success    int            `gorm:"not null;default:0" json:"success"`
	Failed     int            `gorm:"not null;default:0" json:"failed"`
	Status     string         `gorm:"not null;default:'running'" json:"status"`  // running / completed
	Type       string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// PluginDistributionRecord 插件下发记录（每个实例一条）
type PluginDistributionRecord struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Identifier  string         `gorm:"index;default:''" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID      uint           `gorm:"index;not null" json:"task_id"`
	PluginDBID  uint           `gorm:"index;not null" json:"plugin_db_id"`
	InstanceID  uint           `gorm:"index;not null" json:"instance_id"`
	InstanceCID string         `gorm:"not null;default:''" json:"instance_cid"`
	Version     string         `gorm:"not null;default:''" json:"version"`
	Status      string         `gorm:"not null;default:'pending'" json:"status"` // pending / success / failed
	Error       string         `gorm:"type:text" json:"error"`
	Type        string         `gorm:"not null;default:'distribute'" json:"type"` // distribute / uninstall
}

// ResolvePluginUninstallFailedStatus 判断插件卸载失败时应写入的 status。
// 逻辑对齐 skill 侧 ResolveUninstallFailedStatus：
// - 已安装版本 != 最新版本 → "uninstall_failed_old"
// - 已安装版本 == 最新版本（或无成功记录）→ "failed"
func ResolvePluginUninstallFailedStatus(
	ctx context.Context, instanceID uint, pluginIDs []uint, latestVersion string,
) string {
	var lastSuccess PluginDistributionRecord
	err := DB(ctx).Where(
		"instance_id = ? AND plugin_db_id IN ? AND status = ? AND type = ?",
		instanceID, pluginIDs, "success", "distribute",
	).Order("id DESC").Limit(1).First(&lastSuccess).Error
	if err == nil && lastSuccess.Version != "" && lastSuccess.Version != latestVersion {
		return "uninstall_failed_old"
	}
	return "failed"
}
