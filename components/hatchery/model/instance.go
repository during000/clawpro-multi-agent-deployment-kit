package model

import (
	"context"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// CLS Agent 安装状态常量。
const (
	CLSAgentNotInstalled = 0 // 未安装
	CLSAgentInstalled    = 1 // 已安装
	CLSAgentInstalling   = 2 // 安装中
	CLSAgentUninstalling = 3 // 卸载中
	CLSAgentSkipped      = 4 // 已跳过（机器已释放等不可操作状态）
)

type Instance struct {
	gorm.Model
	// idx_instances_identifier_agent_type_deleted_at：覆盖 overview / list / count 公共过滤
	// (identifier, agent_type, deleted_at IS NULL)，避免扫描全库活跃行。
	// deleted_at 列由下方覆写的 DeletedAt 字段补齐 priority:3。
	DeletedAt          gorm.DeletedAt `gorm:"index;index:idx_instances_identifier_agent_type_deleted_at,priority:3" json:"-"`
	Identifier         string         `gorm:"index;index:idx_instances_identifier_agent_type_deleted_at,priority:1;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	Name               string         `gorm:"not null;default:''"`
	InstanceId         string         `gorm:"index"`
	InstanceChargeType string         `gorm:"type:varchar(32);not null;default:'PREPAID'" json:"instance_charge_type"`
	UserID             uint           `gorm:"not null;default:0;index"`
	VpcId              string         `gorm:"default:''"`
	SubnetId           string         `gorm:"default:''"`

	// 实例当前绑定的安全组 ID（base 或 shard）。对齐 Worker 按此字段识别未对齐实例。
	SecurityGroupId string `gorm:"column:security_group_id;type:varchar(64);default:''"`

	// 代理层字段
	ProxyToken        *string `gorm:"uniqueIndex"` // 代理访问 Token，"sk-" 前缀，注入实例时使用；NULL 表示尚未分配（占位记录阶段）
	AIModelID         uint    `gorm:"index"`       // FK 到 AIModel，代理层据此查询模型配置
	MaxTokens         int     `gorm:"default:0"`   // 单次请求 Token 上限，0=不限
	CustomModelConfig string  `gorm:"type:text"`   // 自定义模型配置 JSON
	UserData          string  `gorm:"type:text"`   // 用户创建实例时提交的原始 base64 UserData

	// CLS Agent状态
	CLSAgentStatus   int        `gorm:"default:0"`     // CLS Agent 安装状态: 0=未安装, 1=已安装, 2=安装中, 3=卸载中, 4=已跳过(机器已释放)
	CLSAgentStatusAt *time.Time `gorm:"default:null"`  // CLS Agent 状态最后变更时间，用于超时恢复
	CLSPluginVersion string     `gorm:"default:'1.0'"` // CLS 插件版本，1.0=旧版（无 trace），2.0=新版（含 trace）

	// ======== 生命周期管理字段（本次新增） ========
	CurrentOperation          string     `gorm:"type:varchar(32);default:'';index"`                                                                                           // 当前操作：create/reboot/reinstall/delete/空
	CurrentOperationState     string     `gorm:"type:varchar(32);default:''"`                                                                                                 // 操作状态：processing/success/failed
	CurrentOperationUpdatedAt *time.Time `gorm:"default:null;index"`                                                                                                          // 操作状态最后更新时间（乐观锁/超时检测）
	LastStableState           string     `gorm:"type:varchar(32);default:''"`                                                                                                 // 上次稳定态：running/stopped/空（用于状态恢复）
	LastCVMState              string     `gorm:"column:last_cvm_state;type:varchar(32);default:'';index"`                                                                     // 上次 CVM 状态缓存（减少 DescribeInstances 调用）
	AgentReady                int        `gorm:"default:0;index"`                                                                                                             // Agent 就绪状态：0=未就绪, 1=已就绪
	RuntimeUser               string     `gorm:"type:varchar(64);default:''"`                                                                                                 // openclaw 实际运行用户，由首次 Agent 就绪时检测填充
	RuntimeHome               string     `gorm:"type:varchar(255);default:''"`                                                                                                // openclaw 实际安装 HOME 目录（如 /home/<user> 或 /root）
	RoleID                    uint       `gorm:"default:0;index"`                                                                                                             // 关联角色 ID，0 = 通用助手
	SoulSetAt                 *time.Time `gorm:"default:null"`                                                                                                                // Soul 最后下发到实例的时间，NULL 表示未下发
	DistributedRoleVersion    string     `gorm:"type:varchar(16);not null;default:''" json:"distributed_role_version"`                                                        // 最近一次成功推送到此实例的角色版本号，X.Y 格式；空串=未下发过
	RoleSyncStatus            string     `gorm:"type:varchar(16);not null;default:'';index" json:"role_sync_status"`                                                          // 角色同步状态：""(未初始化)/pending/updating/updated/failed
	GroupID                   uint       `gorm:"default:0;index" json:"group_id"`                                                                                             // 分组 ID；CVM 为创建时指定，本地 Agent 同步当前用户级主组织，0 = 未指定
	AgentType                 string     `gorm:"type:varchar(32);default:'openclaw';index;index:idx_instances_identifier_agent_type_deleted_at,priority:2" json:"agent_type"` // 智能体类型
	AgentVersion              string     `gorm:"type:varchar(64);default:''" json:"agent_version"`                                                                            // 智能体版本号

	// ======== 版本信息字段 ========
	PluginVersionsJSON string     `gorm:"type:text;default:''"` // 已安装插件版本 JSON，格式: {"slug": "version", ...}
	VersionFetchedAt   *time.Time `gorm:"default:null"`         // 版本信息最后拉取时间，用于防重复触发

	// ======== 升级断点续传字段（多副本部署下用 DB 持久化，替代单进程内的 sync.Map 缓存） ========
	PendingArchivePath string     `gorm:"column:pending_archive_path;type:varchar(255);default:''"` // CVM 上未上传完成的备份压缩包路径
	PendingArchiveSize int64      `gorm:"column:pending_archive_size;default:0"`                    // 备份压缩包大小（字节）
	PendingSMHFileKey  string     `gorm:"column:pending_smh_file_key;type:varchar(255);default:''"` // SMH 文件 key，用于复用同一个 ConfirmKey 续传
	PendingUploadAt    *time.Time `gorm:"column:pending_upload_at;default:null"`                    // 写入续传记录的时间，便于运维判断陈旧程度

	// ======== 龙虾医生字段 ========
	IsDoctorNode bool `gorm:"not null;default:false;index"` // 是否为龙虾医生临时诊断节点

	// ======== 状态缓存字段（后台 cvm-status-reconcile 任务维护） ========
	// 迁移脚本: sql/0618-instance-status-cache.sql | 初始化脚本: sql/init.sql (instances 表)
	LastKnownStatus string     `gorm:"column:last_known_status;type:varchar(32);default:'';index"` // 最终语义状态(running/stopped/load_failed/destroyed...)
	CVMTagsJSON     string     `gorm:"column:cvm_tags_json;type:text;default:''"`                  // CVM 标签缓存 JSON，供标签过滤
	ImgId           string     `gorm:"column:img_id;type:varchar(64);default:''"`                  // CVM 镜像 ID 缓存，供 IsOfficialImage 判断
	StatusSyncedAt  *time.Time `gorm:"column:status_synced_at;default:null"`                       // 缓存最后同步时间，用于竞态保护

	// ======== CVM 资源缓存（后台 reconcile 与资源调整 worker 维护） ========
	CVMInstanceType            string `gorm:"column:cvm_instance_type;type:varchar(64);not null;default:'';index" json:"cvm_instance_type"`
	CVMCPU                     int64  `gorm:"column:cvm_cpu;not null;default:0" json:"cpu"`
	CVMMemoryGB                int64  `gorm:"column:cvm_memory_gb;not null;default:0" json:"memory_gb"`
	SystemDiskType             string `gorm:"column:system_disk_type;type:varchar(32);not null;default:''" json:"system_disk_type"`
	SystemDiskSize             int64  `gorm:"column:system_disk_size;not null;default:0;index" json:"system_disk_size"`
	CVMPublicIP                string `gorm:"column:cvm_public_ip;type:varchar(64);not null;default:''" json:"public_ip"`
	CVMInternetChargeType      string `gorm:"column:cvm_internet_charge_type;type:varchar(64);not null;default:''" json:"internet_charge_type"`
	CVMInternetMaxBandwidthOut int64  `gorm:"column:cvm_internet_max_bandwidth_out;not null;default:0" json:"internet_max_bandwidth_out"`

	// ======== 本地 agent（clawpro 一期）字段 ========
	// 迁移脚本: sql/0624-local-agent.sql | 初始化脚本: sql/init.sql (instances 表)
	Source string `gorm:"type:varchar(16);not null;default:'cvm';index" json:"source"` // 实例来源：cvm（云上）/ local（本地 agent）

	// ======== 本地 agent 二期字段 ========
	// 迁移脚本: sql/0706-local-agent-resources.sql | 初始化脚本: sql/init.sql (instances 表)
	// 存本地 agent 的分组绑定 + workspace 列表（JSON，结构见 LocalAgentResources）。
	// source != 'local' 时为空。
	LocalAgentResources *LocalAgentResources `gorm:"type:text" json:"local_agent_resources,omitempty"`

	// ======== 存量实例分组归属处理（stale-instances v1.0） ========
	// 同组移交目标用户 ID；0 表示无 pending 移交。接收/拒绝/取消后清零。用于用户端 inbox「列我作为接收方的所有 pending 移交」热路径。
	HandoverTargetUserID uint `gorm:"column:handover_target_user_id;not null;default:0;index"`
	// 最近一次拒绝移交的用户 ID；0 表示无。原用户重新发起或选其它选项后清零。pending 与 rejected 是两个独立事实，故用两个独立字段。
	HandoverRejectedByUserID uint `gorm:"column:handover_rejected_by_user_id;not null;default:0"`
	// 移交发起时间（用于超时清理）
	HandoverInitiatedAt *time.Time `gorm:"column:handover_initiated_at;default:null"`
}

// 实例 source 取值常量。
const (
	InstanceSourceCVM   = "cvm"
	InstanceSourceLocal = "local"
)

// GetInstanceCountsByType 一次查询获取各类型当前实例数量。
func GetInstanceCountsByType(ctx context.Context) (map[string]int64, error) {
	type stat struct {
		AgentType string
		Count     int64
	}
	var stats []stat

	err := DB(ctx).Model(&Instance{}).
		Select("agent_type, COUNT(*) as count").
		Group("agent_type").
		Scan(&stats).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgInstanceQueryStatsFailed)
	}

	result := make(map[string]int64, len(stats))
	for _, s := range stats {
		result[NormalizeAgentType(s.AgentType)] += s.Count
	}
	return result, nil
}

// CleanupStaleCreatingInstances 清理当前租户中超过指定时间仍处于创建中的占位记录（instance_id 为空）。
// 这些记录通常是由于服务崩溃或异常退出导致未被正常清理的残留数据。
func CleanupStaleCreatingInstances(ctx context.Context, timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := DB(ctx).Where("instance_id = '' AND created_at < ?", cutoff).Delete(&Instance{})
	if result.Error != nil {
		slog.Error("[Cleanup] 清理残留占位记录失败", "error", result.Error)
		return 0, hcommon.I18nRichError(result.Error, i18n.MsgInstanceCleanupPlaceholder)
	}
	if result.RowsAffected > 0 {
		slog.Info("[Cleanup] 已清理残留占位记录", "count", result.RowsAffected)
	}
	return result.RowsAffected, nil
}

// CleanupDestroyedInstances 清理已销毁超过指定时间的实例。
// 通过关联 notifications 表中 external_destroy 类型通知的创建时间判断销毁时间。
// instanceIDs 为需要检查的已销毁实例 ID 列表，返回被清理的实例 ID 列表。
func CleanupDestroyedInstances(ctx context.Context, instanceIDs []uint, timeout time.Duration) []uint {
	if len(instanceIDs) == 0 {
		return nil
	}

	cutoff := time.Now().Add(-timeout)
	var cleanedIDs []uint

	// 查询这些实例对应的外部销毁通知
	var notifications []Notification
	DB(ctx).Where("instance_id IN ? AND type = ?", instanceIDs, NotifyTypeExternalDestroy).Find(&notifications)

	// 构建 instanceID → 通知创建时间 的映射
	notifyTimeMap := make(map[uint]time.Time)
	for _, n := range notifications {
		notifyTimeMap[n.InstanceID] = n.CreatedAt
	}

	for _, id := range instanceIDs {
		shouldClean := false
		if notifyTime, ok := notifyTimeMap[id]; ok {
			// 有外部销毁通知，以通知创建时间为准
			if notifyTime.Before(cutoff) {
				shouldClean = true
			}
		} else {
			// 没有外部销毁通知（可能是通知创建失败），回退到 updated_at 判断
			var inst Instance
			if DB(ctx).Select("updated_at").First(&inst, id).Error == nil {
				if inst.UpdatedAt.Before(cutoff) {
					shouldClean = true
				}
			}
		}

		if shouldClean {
			var inst Instance
			if err := DB(ctx).Select("instance_id").First(&inst, id).Error; err == nil {
				if err := DisableAgentProxyRoutesForInstance(ctx, inst.InstanceId); err != nil {
					slog.Error("[Cleanup] 禁用实例代理路由失败", "id", id, "instance_id", inst.InstanceId, "error", err)
				}
			}
			if err := DB(ctx).Delete(&Instance{}, id).Error; err != nil {
				slog.Error("[Cleanup] 清理已销毁实例失败", "id", id, "error", err)
			} else {
				cleanedIDs = append(cleanedIDs, id)
				slog.Info("[Cleanup] 已清理销毁超过1天的实例", "id", id)
			}
		}
	}

	if len(cleanedIDs) > 0 {
		slog.Info("[Cleanup] 已销毁实例清理完成", "cleaned_count", len(cleanedIDs), "cleaned_ids", cleanedIDs)
	}
	return cleanedIDs
}

// InstanceTable is the table name for Instance model.
const InstanceTable = "instances"

// FilterInstancesByUserGroups restricts an instance query to users in any of
// the given groups. Group ID zero selects users without any group membership.
func FilterInstancesByUserGroups(ctx context.Context, db *gorm.DB, groupIDs []uint) *gorm.DB {
	if len(groupIDs) == 0 {
		return db
	}

	seen := make(map[uint]struct{}, len(groupIDs))
	normalIDs := make([]uint, 0, len(groupIDs))
	includeUngrouped := false
	for _, groupID := range groupIDs {
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		if groupID == 0 {
			includeUngrouped = true
			continue
		}
		normalIDs = append(normalIDs, groupID)
	}

	ungrouped := DB(ctx).Model(&UserGroupMember{}).Select("DISTINCT user_id")
	switch {
	case includeUngrouped && len(normalIDs) > 0:
		grouped := DB(ctx).Model(&UserGroupMember{}).
			Select("DISTINCT user_id").
			Where("user_group_id IN ?", normalIDs)
		return db.Where("instances.user_id NOT IN (?) OR instances.user_id IN (?)", ungrouped, grouped)
	case includeUngrouped:
		return db.Where("instances.user_id NOT IN (?)", ungrouped)
	default:
		grouped := DB(ctx).Model(&UserGroupMember{}).
			Select("DISTINCT user_id").
			Where("user_group_id IN ?", normalIDs)
		return db.Where("instances.user_id IN (?)", grouped)
	}
}

// UpdateInstanceCachedStatus 操作即时写 last_known_status + status_synced_at。
// 供不经过 setOperation 路径的场景使用（如 doctor 修复、外部回调）。
// 常规操作（create/reboot/reinstall/upgrade/migrate/delete）通过 setOperation/
// setOperationNonExclusive 统一写入，无需单独调用此函数。
// 确保 List 纯读模式下状态立即可见且不被后台轮覆盖。
func UpdateInstanceCachedStatus(ctx context.Context, instanceID uint, operation string) {
	transitStatus, ok := OperationTransitStatus[operation]
	if !ok {
		return
	}
	now := time.Now()
	if err := DB(ctx).Model(&Instance{}).Where("id = ?", instanceID).
		Updates(map[string]any{
			"last_known_status": transitStatus,
			"status_synced_at":  now,
		}).Error; err != nil {
		slog.Warn("[CachedStatus] 即时写失败", "id", instanceID, "operation", operation, "error", err)
	}
}

// InstanceStatusCacheItem 后台任务批量写回的单条缓存数据。
type InstanceStatusCacheItem struct {
	ID       uint
	Status   string
	TagsJSON *string // 非 nil 时写入 cvm_tags_json（仅存量补齐时设置）
	ImageId  *string // 非 nil 时写入 img_id（仅存量补齐时设置）

	CVMInstanceType            *string
	CVMCPU                     *int64
	CVMMemoryGB                *int64
	SystemDiskType             *string
	SystemDiskSize             *int64
	CVMPublicIP                *string
	CVMInternetChargeType      *string
	CVMInternetMaxBandwidthOut *int64
}

// BatchUpdateInstanceStatusCache 批量写回 last_known_status / status_synced_at。
// tags 和 image_id 主要由各 handler 在操作时直写，后台 reconcile 仅对空值补齐。
// TagsJSON / ImageId 为指针：nil 表示本次不更新该字段，非 nil 表示需要补齐写入。
// 带竞态保护条件：仅覆盖 status_synced_at 比本轮更旧（或为 NULL）的行，
// 不覆盖操作即时写的更新鲜数据。
// roundStartedAt 为本轮开始时间，用作 status_synced_at 的写入值和条件判断。
func BatchUpdateInstanceStatusCache(ctx context.Context, items []InstanceStatusCacheItem, roundStartedAt time.Time) {
	if len(items) == 0 {
		return
	}

	const batchSize = 500
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		for _, it := range batch {
			statusUpdates := map[string]any{
				"last_known_status": it.Status,
				"status_synced_at":  roundStartedAt,
			}
			if it.TagsJSON != nil {
				statusUpdates["cvm_tags_json"] = *it.TagsJSON
			}
			if it.ImageId != nil {
				statusUpdates["img_id"] = *it.ImageId
			}
			if it.CVMInstanceType != nil {
				statusUpdates["cvm_instance_type"] = *it.CVMInstanceType
			}
			if it.CVMCPU != nil {
				statusUpdates["cvm_cpu"] = *it.CVMCPU
			}
			if it.CVMMemoryGB != nil {
				statusUpdates["cvm_memory_gb"] = *it.CVMMemoryGB
			}
			if it.SystemDiskType != nil {
				statusUpdates["system_disk_type"] = *it.SystemDiskType
			}
			if it.SystemDiskSize != nil {
				statusUpdates["system_disk_size"] = *it.SystemDiskSize
			}
			if it.CVMPublicIP != nil {
				statusUpdates["cvm_public_ip"] = *it.CVMPublicIP
			}
			if it.CVMInternetChargeType != nil {
				statusUpdates["cvm_internet_charge_type"] = *it.CVMInternetChargeType
			}
			if it.CVMInternetMaxBandwidthOut != nil {
				statusUpdates["cvm_internet_max_bandwidth_out"] = *it.CVMInternetMaxBandwidthOut
			}
			statusResult := DB(ctx).Model(&Instance{}).
				Where("id = ? AND (status_synced_at IS NULL OR status_synced_at < ?)", it.ID, roundStartedAt).
				Updates(statusUpdates)
			if statusResult.Error != nil {
				slog.Error("[StatusCache] 状态与资源写回失败", "id", it.ID, "error", statusResult.Error)
			}
		}
	}
}
