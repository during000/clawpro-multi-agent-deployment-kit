package model

import (
	"context"
	"encoding/json"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	MemoryTDAITaskTypeEnable  = "enable"
	MemoryTDAITaskTypeDisable = "disable"

	MemoryTDAIPluginStatusNotInstalled       = "NOT_INSTALLED"
	MemoryTDAIPluginStatusEnabling           = "ENABLING"
	MemoryTDAIPluginStatusEnabled            = "ENABLED"
	MemoryTDAIPluginStatusDisabling          = "DISABLING"
	MemoryTDAIPluginStatusDisabled           = "DISABLED"
	MemoryTDAIPluginStatusFailed             = "FAILED"
	MemoryTDAIPluginStatusUnsupportedVersion = "UNSUPPORTED_VERSION"
)

const (
	DefaultMemoryTDAIPluginName        = "@tencentdb-agent-memory/memory-tencentdb" // npm 全名，安装时使用
	DefaultMemoryTDAISupportedVersions = "[]"
	DefaultMemoryTDAIMinVersion        = "0.3.3" // ensure_memory_plugin 要求的最低插件版本，低于此版本自动升级
	// DefaultMemoryTDAIDistTag 指定从 npm 拉取的 dist-tag。
	// 生产环境用 "latest"；开发调试时改为 "beta" 拉 0.3.0-beta.x 等预发布版本。
	// 切换后需重新编译 hatchery；ensure_memory_plugin.sh 会用 `npm view <pkg>@<tag> version` 解析为具体版本号再安装。
	// 当 dist_tag != "latest" 时，min_version 短路判断会被忽略，永远跟随 dist-tag 指向的最新版本。
	DefaultMemoryTDAIDistTag = "latest"
)

// 记忆计划常量
const (
	MemoryPlanOff  = "OFF"
	MemoryPlanFree = "FREE"
	MemoryPlanPro  = "PRO"
)

// 切换状态常量
const (
	MemorySwitchStatusNone            = ""
	MemorySwitchStatusSwitchingToOff  = "SWITCHING_TO_OFF"
	MemorySwitchStatusSwitchingToFree = "SWITCHING_TO_FREE"
	MemorySwitchStatusSwitchingToPro  = "SWITCHING_TO_PRO"
)

// MemoryTDAIPlugin 记录实例维度的记忆插件当前状态。
type MemoryTDAIPlugin struct {
	gorm.Model
	// idx_memory_tda_iplugins_instance_id_deleted_at：让列表/概览接口的 LEFT JOIN
	// (ON instance_id = ? AND deleted_at IS NULL) 走索引，避免 nested loop 全表扫
	// plugin 表（生产实测 N×M → N×1）。deleted_at 列由下方覆写的 DeletedAt 字段补齐 priority:2。
	DeletedAt  gorm.DeletedAt `gorm:"index;index:idx_memory_tda_iplugins_instance_id_deleted_at,priority:2" json:"-"`
	Identifier string         `gorm:"uniqueIndex:idx_memory_tdai_instance_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	InstanceID string         `gorm:"uniqueIndex:idx_memory_tdai_instance_identifier;index:idx_memory_tda_iplugins_instance_id_deleted_at,priority:1;not null;default:''"`
	Status     string         `gorm:"index;not null;default:NOT_INSTALLED"`
	LastError  string         `gorm:"type:text"`
	RetryCount int            `gorm:"not null;default:0"`

	// 记忆计划（新增）
	// gorm size 与 sql/init.sql 中 varchar(N) 保持一致，便于代码阅读者感知 DB 字段长度约束。
	DesiredPlan    string     `gorm:"size:32;not null;default:OFF" json:"desired_plan"`
	CurrentPlan    string     `gorm:"size:32;index;not null;default:OFF" json:"current_plan"`
	SwitchStatus   string     `gorm:"size:64;index;not null;default:''" json:"switch_status"`
	LastTaskID     uint       `gorm:"default:0" json:"last_task_id"`
	LastSwitchedAt *time.Time `json:"last_switched_at,omitempty"`

	// Pro 绑定信息（新增），对应 sql 里 varchar(191)
	PoolID          string `gorm:"size:191;default:''" json:"pool_id,omitempty"`
	DatabaseName    string `gorm:"size:191;default:''" json:"database_name,omitempty"`
	Endpoint        string `gorm:"size:191;default:''" json:"endpoint,omitempty"`
	ApiKeySecretRef string `gorm:"size:191;default:''" json:"-"`
	VdbUsername     string `gorm:"size:191;default:''" json:"-"`
	EmbeddingModel  string `gorm:"size:191;default:''" json:"embedding_model,omitempty"` // 服务端分配的 embedding 模型

	// 一键升级功能字段
	MemoryPluginVersion string `gorm:"size:32;not null;default:''" json:"memory_plugin_version"` // 记忆插件当前版本
	OffloadEnabled      *bool  `gorm:"default:null" json:"offload_enabled"`                      // Offload 是否开启，NULL=未检测
}

// NormalizeMemoryTDAISupportedVersions 校验并规范化 memory_tdai_supported_versions。
// 输入为空时返回默认值 "[]"。
func NormalizeMemoryTDAISupportedVersions(raw string) (normalized string, versions []string, rerr error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultMemoryTDAISupportedVersions, []string{}, nil
	}

	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return "", nil, hcommon.I18nRichError(err, i18n.MsgMemoryTDAIVersionsMustBeJSONArray)
	}

	seen := make(map[string]struct{}, len(arr))
	cleaned := make([]string, 0, len(arr))
	for _, v := range arr {
		item := strings.TrimSpace(v)
		if item == "" {
			return "", nil, hcommon.I18nError(i18n.MsgMemoryTDAIVersionsCannotContainEmpty)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		cleaned = append(cleaned, item)
	}

	b, err := json.Marshal(cleaned)
	if err != nil {
		return "", nil, hcommon.I18nRichError(err, i18n.MsgMemoryTDAIVersionsNormalizeFailed)
	}
	return string(b), cleaned, nil
}

// EnsureMemoryTDAIPluginRow 为单个实例补齐 plugin 行（幂等）。
func EnsureMemoryTDAIPluginRow(ctx context.Context, instanceID string) {
	if instanceID == "" {
		return
	}
	var count int64
	DB(ctx).Model(&MemoryTDAIPlugin{}).Where("instance_id = ?", instanceID).Count(&count)
	if count > 0 {
		return
	}
	DB(ctx).Create(&MemoryTDAIPlugin{
		InstanceID:  instanceID,
		Status:      MemoryTDAIPluginStatusNotInstalled,
		CurrentPlan: MemoryPlanOff,
		DesiredPlan: MemoryPlanOff,
	})
}

// MigrateMemoryPlanFromStatus 一次性迁移：根据旧 status 字段推导 current_plan。
// ENABLED/ENABLING → FREE，其余保持 OFF。仅在 current_plan 仍为默认值 OFF 时才更新，幂等安全。
// 服务启动时调用一次即可。
func MigrateMemoryPlanFromStatus(ctx context.Context) {
	result := DB(ctx).Model(&MemoryTDAIPlugin{}).
		Where("status IN ? AND current_plan = ?",
			[]string{MemoryTDAIPluginStatusEnabled, MemoryTDAIPluginStatusEnabling},
			MemoryPlanOff,
		).
		Updates(map[string]any{
			"current_plan": MemoryPlanFree,
			"desired_plan": MemoryPlanFree,
		})
	if result.RowsAffected > 0 {
		slog.Info("[MemoryTDAI] 历史实例 current_plan 迁移完成",
			"migrated", result.RowsAffected,
		)
	}
}

// DeleteMemoryTDAIPluginRow 删除实例对应的 plugin 行（实例释放时调用）。
func DeleteMemoryTDAIPluginRow(ctx context.Context, instanceID string) {
	if instanceID == "" {
		return
	}
	DB(ctx).Where("instance_id = ?", instanceID).Delete(&MemoryTDAIPlugin{})
}

// GetMemoryTDAIPlugin 查询实例的 memory plugin 行，不存在返回 nil。
func GetMemoryTDAIPlugin(ctx context.Context, instanceID string) *MemoryTDAIPlugin {
	if instanceID == "" {
		return nil
	}
	var plugin MemoryTDAIPlugin
	if DB(ctx).Where("instance_id = ?", instanceID).First(&plugin).Error != nil {
		return nil
	}
	return &plugin
}
