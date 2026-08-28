package model

import (
	"context"
	"fmt"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// MCP 安装状态枚举（对齐 SkillInstallation / PluginInstallation）
const (
	McpInstallNone      = 0 // 未安装
	McpInstalling       = 1 // 安装中
	McpInstallSuccess   = 2 // 安装成功
	McpInstallFailed    = 3 // 安装失败
	McpInstallCancelled = 4 // 已取消
)

// McpServer MCP 主表
type McpServer struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Identifier      string    `gorm:"uniqueIndex:uk_identifier_service_id;index;default:''" json:"-"` // 多租户标识
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ServiceID       string    `gorm:"uniqueIndex:uk_identifier_service_id;not null;size:128" json:"service_id"`
	Name            string    `gorm:"not null;size:128" json:"name"`
	Description     string    `gorm:"size:1024" json:"description"`
	TransportType   string    `gorm:"not null;size:32" json:"transport_type"`                // 最新版本的冗余，便于列表页筛选
	LatestVersionID uint      `gorm:"default:0" json:"latest_version_id"`                    // 指向 mcp_versions.id（不设 FK 约束）
	CreatedBy       string    `gorm:"not null;size:128;default:''" json:"created_by"`        // session user 或 admin token 对应的 admin username
	VisibilityType  string    `gorm:"size:16;not null;default:'all'" json:"visibility_type"` // 应用范围：'all'=全部用户, 'group'=按组可见

	// 凭据托管（headers 中含占位符时自动标记）
	KeyHosted   bool   `gorm:"not null;default:false" json:"key_hosted"`
	IPWhitelist string `gorm:"size:2048;not null;default:''" json:"ip_whitelist"` // 逗号分隔的 IP/CIDR 白名单，空=不限制
}

// McpHostedKey 模板级托管字段定义（管理员定义哪些占位符 key 需要托管 + 可选默认值）
type McpHostedKey struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	Identifier   string     `gorm:"uniqueIndex:uk_mcp_key;index;default:''" json:"-"`
	MCPID        uint       `gorm:"uniqueIndex:uk_mcp_key;not null;index;column:mcp_id" json:"mcp_id"`
	Key          string     `gorm:"uniqueIndex:uk_mcp_key;not null;size:128;column:key" json:"key"` // 占位符 key，如 "your-token"
	Placeholder  string     `gorm:"size:256;not null;default:''" json:"placeholder"`                // 原始占位符，如 "<your-token>"
	DefaultValue string     `gorm:"size:1024;not null;default:''" json:"-"`                         // 管理员默认值（不返回前端）
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (McpHostedKey) TableName() string {
	return "mcp_hosted_keys"
}

// McpVersion MCP 版本表
type McpVersion struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Identifier    string    `gorm:"uniqueIndex:uk_mcp_version;index;default:''" json:"-"` // 多租户标识
	CreatedAt     time.Time `json:"created_at"`
	MCPID         uint      `gorm:"uniqueIndex:uk_mcp_version;not null;index;column:mcp_id" json:"mcp_id"`
	Version       string    `gorm:"uniqueIndex:uk_mcp_version;not null;size:32" json:"version"` // 1.0.0 / 1.1.0 / ...
	TransportType string    `gorm:"not null;size:32" json:"transport_type"`
	ConfigJSON    string    `gorm:"type:text;not null" json:"config_json"` // 内层服务配置 JSON
	UsageDocMD    string    `gorm:"type:text" json:"usage_doc_md"`         // 使用说明 Markdown
	ToolDocMD     string    `gorm:"type:text" json:"tool_doc_md"`          // 工具说明 Markdown
	CreatedBy     string    `gorm:"not null;size:128;default:''" json:"created_by"`
}

// McpDistributionTask MCP 批次任务表（对齐 SkillDistributionTask / PluginDistributionTask）
type McpDistributionTask struct {
	ID                   uint           `gorm:"primarykey" json:"id"`
	Identifier           string         `gorm:"index;default:''" json:"-"` // 多租户标识
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
	MCPID                uint           `gorm:"index;not null;column:mcp_id" json:"mcp_id"`
	McpSnapshotServiceID string         `gorm:"not null;size:128" json:"mcp_snapshot_service_id"` // 冗余快照
	McpSnapshotName      string         `gorm:"not null;size:128" json:"mcp_snapshot_name"`       // 冗余快照
	VersionID            uint           `gorm:"default:0" json:"version_id"`
	VersionSnapshot      string         `gorm:"not null;size:32" json:"version_snapshot"` // 冗余快照
	OperatorID           uint           `gorm:"not null;default:0" json:"operator_id"`
	Total                int            `gorm:"not null;default:0" json:"total"`
	Success              int            `gorm:"not null;default:0" json:"success"`
	Failed               int            `gorm:"not null;default:0" json:"failed"`
	Status               string         `gorm:"not null;default:'running';size:16" json:"status"` // running / completed
}

// McpDistributionRecord MCP 逐实例下发记录（对齐 SkillDistributionRecord / PluginDistributionRecord）
type McpDistributionRecord struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	Identifier      string         `gorm:"uniqueIndex:uk_task_instance;index;default:''" json:"-"` // 多租户标识
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID          uint           `gorm:"uniqueIndex:uk_task_instance;index;not null" json:"task_id"`
	MCPID           uint           `gorm:"index;column:mcp_id" json:"mcp_id"`
	InstanceID      uint           `gorm:"uniqueIndex:uk_task_instance;index;not null" json:"instance_id"`
	InstanceCID     string         `gorm:"not null;size:64;default:''" json:"instance_cid"` // CVM InstanceId（冗余快照）
	VersionSnapshot string         `gorm:"not null;size:32" json:"version_snapshot"`
	Status          string         `gorm:"not null;default:'pending';size:16" json:"status"` // pending / success / failed
	Error           string         `gorm:"type:text" json:"error"`
}

// McpInstallation 实例当前 MCP 状态表（对齐 SkillInstallation / PluginInstallation）
type McpInstallation struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Identifier    string    `gorm:"uniqueIndex:uk_instance_service;index;default:''" json:"-"` // 多租户标识
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	InstanceID    uint      `gorm:"not null;uniqueIndex:uk_instance_service;index" json:"instance_id"`
	MCPID         uint      `gorm:"index;column:mcp_id" json:"mcp_id"`
	ServiceID     string    `gorm:"not null;uniqueIndex:uk_instance_service;size:128" json:"service_id"`
	Name          string    `gorm:"not null;size:128;default:''" json:"name"`   // 冗余，MCP 删除后仍可展示
	Version       string    `gorm:"not null;default:'';size:32" json:"version"` // 当前已装版本；失败时为空
	InstallStatus int       `gorm:"not null;default:0" json:"install_status"`   // 0=None 1=Installing 2=Success 3=Failed 4=Cancelled
	LastTaskID    uint      `gorm:"default:0" json:"last_task_id"`              // 最近一次 task
	ErrorMessage  string    `gorm:"not null;size:2048;default:''" json:"error_message"`

	// ── 用户端 MCP 管理扩展字段 ──
	ConfigJSON       string     `gorm:"type:text;not null" json:"config_json"`                 // 实际配置 JSON（管理端=模板原文，用户端=替换密钥后）
	Source           string     `gorm:"size:16;not null;default:'admin'" json:"source"`        // 来源：admin=管理端下发，user=用户自选
	ConnectionStatus string     `gorm:"size:16;not null;default:''" json:"connection_status"`  // connected/failed/unsupported/unconfigured/""
	ToolsJSON        string     `gorm:"type:text;not null" json:"tools_json"`                  // 工具列表 JSON，如 ["tool1","tool2"]
	ConnectionError  string     `gorm:"size:1024;not null;default:''" json:"connection_error"` // 连接失败时的错误信息
	ProbedAt         *time.Time `json:"probed_at"`                                             // 最近一次探测时间

	// ── 凭据托管扩展字段 ──
	HostedValues string `gorm:"type:text" json:"-"` // 实例级托管字段值 JSON: {"Authorization":"Bearer xxx"}
}

// BuildMcpInstanceQuery 构造 MCP 可下发实例查询。
// scope: "pending_or_failed" | "all" | "success_on_older_version"
// serviceID: MCP 的 service_id
// latestVersion: MCP 最新版本号（用于 success_on_older_version 判断）
func BuildMcpInstanceQuery(ctx context.Context, serviceID, scope, latestVersion string) *gorm.DB {
	q := DB(ctx).Model(&Instance{}).
		Select(`instances.id AS instance_id,
			instances.instance_id AS cvm_instance_id,
			instances.name AS instance_name,
			COALESCE(instances.agent_type, '') AS instance_type,
			instances.last_cvm_state AS last_cvm_state,
			instances.last_stable_state AS last_stable_state,
			COALESCE(mi.version, '') AS current_mcp_version,
			COALESCE(mi.install_status, 0) AS install_status,
			COALESCE(mi.last_task_id, 0) AS last_task_id`).
		Joins(`LEFT JOIN mcp_installations mi
			ON mi.instance_id = instances.id
			AND mi.service_id = ?`, serviceID).
		Where("instances.instance_id != ''").
		Where("instances.agent_type IN ?", GetMCPSupportedAgentTypes(ctx)).
		Where("instances.last_stable_state = 'RUNNING'")

	switch scope {
	case "all":
		// 不加额外过滤
	case "success_on_older_version":
		q = q.Where("mi.install_status = ? AND mi.version != ?", McpInstallSuccess, latestVersion)
	default: // "pending_or_failed"
		// 未安装、失败、或 Installing 超过 30 分钟的孤儿
		q = q.Where(`(mi.id IS NULL
			OR mi.install_status IN (?, ?)
			OR (mi.install_status = ? AND mi.updated_at < ?))`,
			McpInstallNone, McpInstallFailed,
			McpInstalling, time.Now().Add(-30*time.Minute))
	}

	return q
}

// McpInstallStatusCase 返回 MCP 安装状态的 SQL CASE 表达式。
// 将 mcp_installations 的 int install_status 映射为语义字符串，
// 对齐技能的 InstallStatusCase 输出（uninstalled/outdated/installed/installing/failed）。
// latestVersion 已通过 mcpVersionRegex 校验，直接嵌入 SQL 字面值。
func McpInstallStatusCase(latestVersion string) string {
	return fmt.Sprintf(`CASE
		WHEN mi.id IS NULL OR mi.install_status = %d THEN 'uninstalled'
		WHEN mi.install_status = %d AND mi.version != '%s' THEN 'outdated'
		WHEN mi.install_status = %d THEN 'installed'
		WHEN mi.install_status = %d THEN 'installing'
		WHEN mi.install_status IN (%d, %d) THEN 'failed'
		ELSE 'uninstalled'
	END`, McpInstallNone,
		McpInstallSuccess, latestVersion,
		McpInstallSuccess,
		McpInstalling,
		McpInstallFailed, McpInstallCancelled)
}

// BuildMcpInstanceQueryV2 构造 MCP 实例查询（用于实例安装情况页面）。
// 对齐 BuildSkillInstanceQuery：LEFT JOIN users + mcp_installations，
// SELECT 包含 CVM 状态相关字段（供 ResolveInstanceStatus 使用）。
// 不硬编码 agent_type 和 last_stable_state，交由 CVM API + 内存过滤判断实际运行状态。
func BuildMcpInstanceQueryV2(ctx context.Context, serviceID, latestVersion string) *gorm.DB {
	// 防御性校验：确保 latestVersion 是合法的 x.y.z 格式，防止 SQL 注入。
	if latestVersion != "" {
		var major, minor, patch int
		if _, err := fmt.Sscanf(latestVersion, "%d.%d.%d", &major, &minor, &patch); err != nil {
			latestVersion = ""
		}
	}

	// 注意：latestVersion 通过 fmt.Sprintf 嵌入 SQL 字面值而非 ? 绑定参数。
	// 原因：GORM 的 Select() 绑定参数与后续 Where() 子查询参数共享同一个 Vars 列表，
	// 当存在子查询（如 group_id 过滤）时会导致参数顺序错位。
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
		COALESCE(instances.agent_version, '') AS agent_version,
		COALESCE(u.username, '') AS username,
		%s AS install_status,
		COALESCE(mi.version, '') AS version,
		'%s' AS latest_version`, McpInstallStatusCase(latestVersion), latestVersion)

	q := DB(ctx).Model(&Instance{}).Select(selectClause)

	// users JOIN — user_id 来自已隔离的 instances，无需重复过滤 identifier
	q = q.Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL")

	// mcp_installations JOIN — instance_id 来自已隔离数据，
	// 关联到的 installations 通过 service_id 精确匹配，属于当前租户
	q = q.Joins(`LEFT JOIN mcp_installations mi
		ON mi.instance_id = instances.id
		AND mi.service_id = ?`, serviceID)

	// deleted_at 和 identifier 由 GORM 回调自动注入，无需手动添加
	q = q.Where("instances.instance_id != ''")

	return q
}

// NextMcpVersion 在事务内计算下一个版本号（格式 MAJOR.MINOR.PATCH，与技能包一致）。
// 使用 Go 层面查出所有版本号后在内存中计算 max，兼容 SQLite 和 MySQL。
// 自增规则：在最大版本号基础上 PATCH + 1。
// 例如：无版本 → 1.0.0；已有 1.0.0 → 1.0.1；已有 1.2.3 → 1.2.4。
func NextMcpVersion(tx *gorm.DB, mcpID uint) (string, error) {
	var versions []string
	if err := tx.Model(&McpVersion{}).
		Where("mcp_id = ?", mcpID).
		Pluck("version", &versions).Error; err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgMcpQueryVersionListFailed)
	}

	if len(versions) == 0 {
		return "1.0.0", nil
	}

	// 找到最大版本号（按 major, minor, patch 三段比较）
	var maxMajor, maxMinor, maxPatch int
	found := false
	for _, v := range versions {
		var major, minor, patch int
		if _, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch); err == nil {
			if !found || major > maxMajor ||
				(major == maxMajor && minor > maxMinor) ||
				(major == maxMajor && minor == maxMinor && patch > maxPatch) {
				maxMajor, maxMinor, maxPatch = major, minor, patch
				found = true
			}
		}
	}

	if !found {
		return "1.0.0", nil
	}

	return fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, maxPatch+1), nil
}

// MaxMcpVersion 返回指定 MCP 的当前最大版本号（格式 MAJOR.MINOR.PATCH）。
// 不存在版本时返回空字符串。
func MaxMcpVersion(tx *gorm.DB, mcpID uint) (string, error) {
	var versions []string
	if err := tx.Model(&McpVersion{}).
		Where("mcp_id = ?", mcpID).
		Pluck("version", &versions).Error; err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgMcpQueryVersionListFailed)
	}
	if len(versions) == 0 {
		return "", nil
	}

	var maxMajor, maxMinor, maxPatch int
	found := false
	for _, v := range versions {
		var major, minor, patch int
		if _, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch); err == nil {
			if !found || CompareSemver(v, fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, maxPatch)) > 0 {
				maxMajor, maxMinor, maxPatch = major, minor, patch
				found = true
			}
		}
	}
	if !found {
		return "", nil
	}
	return fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, maxPatch), nil
}

// CompareSemver 比较两个 MAJOR.MINOR.PATCH 格式的版本号。
// 返回 -1（a < b）、0（a == b）、1（a > b）。
// 无法解析的版本号视为 0.0.0。
func CompareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}
