package model

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// InstanceModel 表示实例与 AI 模型的绑定关系（多模型 Fallback）。
// 一个实例可绑定多个模型，其中一个是 primary（主模型），其余为 fallback（备选）。
//
// 联合唯一索引 idx_instance_model = (instance_id, ai_model_id, custom_model_id)：
//   - 预配置模型：ai_model_id>0、custom_model_id=”，用 (instance_id, ai_model_id) 防重
//   - 自定义模型：ai_model_id=0、custom_model_id='<model_id>'，用 (instance_id, custom_model_id) 防重
type InstanceModel struct {
	gorm.Model
	Identifier string `gorm:"index;default:''"` // 多租户标识

	InstanceID        uint   `gorm:"not null;uniqueIndex:idx_instance_model;index:idx_instance_id"`        // 关联的实例 ID
	AIModelID         uint   `gorm:"not null;default:0;uniqueIndex:idx_instance_model"`                    // 预配置模型 ID（>0 内置，=0 自定义）
	CustomModelID     string `gorm:"type:varchar(128);not null;default:'';uniqueIndex:idx_instance_model"` // 自定义模型的 model_id（预配置模型为空字符串）
	Role              string `gorm:"type:varchar(16);not null;default:'fallback'"`                         // "primary" 或 "fallback"
	SortOrder         int    `gorm:"not null;default:0;index"`                                             // 排序值（越大越晚添加；fallback 链按 sort_order ASC）
	CustomModelConfig string `gorm:"type:text;default:''"`                                                 // 自定义模型配置 JSON（仅 ai_model_id=0 时有效）
}

const (
	ModelRolePrimary  = "primary"
	ModelRoleFallback = "fallback"
)

// migrateCustomConfig 从 CustomModelConfig JSON 中解析出 model_id，用于填充 CustomModelID。
type migrateCustomConfig struct {
	ModelID string `json:"model_id"`
}

const migrateBatchSize = 100

// MigrateInstanceModels 将存量单模型数据（Instance.AIModelID / Instance.CustomModelConfig）
// 迁移到 instance_models 表。幂等设计：已迁移过的记录不会重复插入。
//
// 服务启动时调用（SQLite 与 MySQL 均执行），确保老版本升级后的存量数据
// 能被纳入多模型管理体系。
func MigrateInstanceModels(ctx context.Context) {
	// 分布式锁：多实例同时启动时只有一个节点执行迁移，避免并发重复写入。
	// SQLite 模式下锁为空操作，不影响单机部署。
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	lock, err := AcquireLock(ctx, "migrate:instance_models", 0)
	if err != nil {
		// 获取锁失败说明其他节点正在迁移，跳过本次执行
		slog.Info("[Migrate] 其他节点正在执行迁移，跳过本次", "error", err)
		return
	}
	defer lock.Release()

	migrated := 0
	offset := 0

	for {
		// 分页查询，避免一次性加载全部实例导致内存溢出
		var instances []Instance
		DB(ctx).Where("ai_model_id > 0 OR (custom_model_config IS NOT NULL AND custom_model_config != '')").
			Offset(offset).Limit(migrateBatchSize).Find(&instances)

		if len(instances) == 0 {
			break
		}

		for _, inst := range instances {
			// 幂等：检查该实例是否已有任何 primary 记录
			var count int64
			DB(ctx).Model(&InstanceModel{}).
				Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).
				Count(&count)
			if count > 0 {
				continue
			}

			im := InstanceModel{
				InstanceID: inst.ID,
				AIModelID:  inst.AIModelID,
				Role:       ModelRolePrimary,
				SortOrder:  1,
			}

			// 自定义模型：解析 JSON 获取 model_id 填入 CustomModelID，用于联合唯一索引防重
			if inst.AIModelID == 0 && inst.CustomModelConfig != "" {
				var cfg migrateCustomConfig
				if err := json.Unmarshal([]byte(inst.CustomModelConfig), &cfg); err != nil {
					slog.Warn("[Migrate] 解析自定义模型配置失败，跳过该实例",
						"instance_id", inst.ID, "error", err)
					continue
				}
				im.CustomModelID = cfg.ModelID
				im.CustomModelConfig = inst.CustomModelConfig
			}

			if err := DB(ctx).Create(&im).Error; err != nil {
				slog.Error("[Migrate] 迁移实例模型失败",
					"instance_id", inst.ID,
					"ai_model_id", inst.AIModelID,
					"custom_model_id", im.CustomModelID,
					"error", err)
				continue
			}
			migrated++
			slog.Info("[Migrate] 已迁移实例模型",
				"instance_id", inst.ID,
				"ai_model_id", inst.AIModelID,
				"custom_model_id", im.CustomModelID,
				"im_id", im.ID)
		}

		if len(instances) < migrateBatchSize {
			break
		}
		offset += migrateBatchSize
	}

	if migrated > 0 {
		slog.Info("[Migrate] 存量实例模型迁移完成", "migrated_count", migrated)
	}
}

// ======== ResolvedModel / ResolveModelForRequest ========

// ResolvedModel 是一次 LLM 代理请求所解析出的目标模型上下文。
// 屏蔽"内置 vs 自定义"差异，供 proxy 统一发起上游调用与记账。
//
// 字段说明：
//   - AIModelID / CustomModelID 标识这一次请求命中的绑定（仅内部使用，非 0 对应内置）
//   - ModelID 是真正透传给上游的 model 名
//   - Provider / ModelType 决定日志和 provider 分派
//   - URL / APIKey 是上游请求参数
//   - QuotaDay: 内置模型沿用 ai_models.quota_day；自定义模型固定 -1（不限；单请求仍受 user/global 配额约束）
//   - IsCustom: true 表示来自自定义模型配置
//   - UsageBucketKey: 统计聚合键；内置模型 = AIModelID，自定义模型 = 0（与老数据一致，避免迁移期间聚合键漂移）
type ResolvedModel struct {
	AIModelID     uint
	CustomModelID string
	// ModelID 保存原始模型 ID（保真，支持 "/"）。
	// 直接用于下发到真实 LLM 的 body.model 字段；如需作为 openclaw/hermes/ACE 的
	// providerKey / ref / TAT 参数，请通过 model.SlugifyModelID(ModelID) 做 slug 化。
	ModelID           string
	ModelName         string
	Provider          string
	ModelType         string
	URL               string
	APIKey            string
	QuotaDay          int
	IsCustom          bool
	UsageBucketKey    uint
	CustomHTTPHeaders map[string]string // 需要透传的自定义请求头
}

// resolvedCustomConfig 用于解析 CustomModelConfig JSON。
// 与 controller.customModelConfig 结构保持字段兼容（JSON 等价）。
type resolvedCustomConfig struct {
	Provider          string            `json:"provider"`
	ModelID           string            `json:"model_id"`
	ModelName         string            `json:"model_name"`
	APIKey            string            `json:"api_key"`
	URL               string            `json:"url"`
	ModelType         string            `json:"model_type"`
	InputTypes        []string          `json:"input_types"`
	ContextLen        int               `json:"context_len"`
	CustomHTTPHeaders map[string]string `json:"custom_http_headers,omitempty"`
}

// ErrNoResolvableModel 表示当前实例无法解析出任何可用模型。
var ErrNoResolvableModel = errors.New("no resolvable model for instance")

// ResolveModelForRequest 按请求里的模型名（大小写不敏感）为实例挑选一个可用的目标模型。
//
// 解析优先级：
//  1. 在 instance_models 表里查找与 reqModelName 大小写不敏感匹配的绑定
//     - 内置绑定 (ai_model_id>0)：比较 ai_models.model_id
//     - 自定义绑定 (ai_model_id=0)：比较 custom_model_id（同时也是 CustomModelConfig.model_id）
//  2. 如果 reqModelName 为空，或绑定表里没有匹配项，则回退到 instance 的 primary 模型：
//     - instance.AIModelID > 0：用对应内置模型（需 enabled=true）
//     - 否则：解析 instance.CustomModelConfig
//  3. 仍无法命中则返回 ErrNoResolvableModel
//
// 对存量实例兼容：即使 instance_models 尚未迁移，也能从 instance 主字段解析出 primary。
func ResolveModelForRequest(ctx context.Context, instance *Instance, reqModelName string) (*ResolvedModel, error) {
	if instance == nil {
		return nil, ErrNoResolvableModel
	}

	req := strings.ToLower(strings.TrimSpace(reqModelName))

	// Step 1: 在 instance_models 中按请求模型名匹配
	if req != "" {
		if rm, ok := resolveFromBindings(ctx, instance.ID, req); ok {
			return rm, nil
		}
	}

	// Step 2: 回退到 instance 主字段上的 primary 配置
	// 先查 instance_models 表的 primary 绑定（OpenClaw gateway 的配置方式）
	if rm, ok := resolvePrimaryFromBindings(ctx, instance.ID); ok {
		return rm, nil
	}

	// 再回退到 instance 主字段（老版本兼容）
	if instance.AIModelID > 0 {
		var m AIModel
		if err := DB(ctx).Where("id = ? AND enabled = ?", instance.AIModelID, true).First(&m).Error; err == nil {
			return builtinResolved(&m), nil
		}
	}
	if instance.CustomModelConfig != "" {
		var cfg resolvedCustomConfig
		if err := json.Unmarshal([]byte(instance.CustomModelConfig), &cfg); err == nil && cfg.ModelID != "" {
			return customResolved(&cfg), nil
		}
	}

	return nil, ErrNoResolvableModel
}

// resolveFromBindings 在 instance_models 表里按 reqLower（已小写）查找绑定。
// 同时覆盖内置与自定义两种场景；任何一种命中即返回。
func resolveFromBindings(ctx context.Context, instanceID uint, reqLower string) (*ResolvedModel, bool) {
	// 内置模型：join ai_models 按 LOWER(model_id) 比较
	// 注意 ai_models 必须 enabled=true，避免把已下架的模型路由出去
	var builtin AIModel
	err := DB(ctx).Model(&AIModel{}).
		Joins("JOIN instance_models ON instance_models.ai_model_id = ai_models.id").
		Where("instance_models.instance_id = ? AND instance_models.ai_model_id > 0 AND ai_models.enabled = ? AND LOWER(ai_models.model_id) = ?",
			instanceID, true, reqLower).
		Select("ai_models.*").
		First(&builtin).Error
	if err == nil {
		return builtinResolved(&builtin), true
	}

	// slug 化兼容匹配：openclaw 的 ref 会将 model_id 中的 "/" 替换为 "-"，
	// 导致请求的 model name（如 "deepseek-v3.1-deepseek-v3.1"）与 DB 中含 "/" 的
	// 原始值（如 "DeepSeek-V3.1/DeepSeek-V3.1"）无法直接匹配。
	// 对 DB 值也做相同的 slug 化后比较，确保含 "/" 的模型也能被正确路由。
	err = DB(ctx).Model(&AIModel{}).
		Joins("JOIN instance_models ON instance_models.ai_model_id = ai_models.id").
		Where("instance_models.instance_id = ? AND instance_models.ai_model_id > 0 AND ai_models.enabled = ? AND LOWER(REPLACE(REPLACE(ai_models.model_id, '/', '-'), ':', '-')) = ?",
			instanceID, true, reqLower).
		Select("ai_models.*").
		First(&builtin).Error
	if err == nil {
		return builtinResolved(&builtin), true
	}

	// 自定义模型：比较 instance_models.custom_model_id
	var im InstanceModel
	err = DB(ctx).Where("instance_id = ? AND ai_model_id = 0 AND LOWER(custom_model_id) = ?",
		instanceID, reqLower).
		First(&im).Error
	if err == nil && im.CustomModelConfig != "" {
		var cfg resolvedCustomConfig
		if jerr := json.Unmarshal([]byte(im.CustomModelConfig), &cfg); jerr == nil && cfg.ModelID != "" {
			return customResolved(&cfg), true
		}
	}

	// 自定义模型 slug 化兼容匹配
	err = DB(ctx).Where("instance_id = ? AND ai_model_id = 0 AND LOWER(REPLACE(REPLACE(custom_model_id, '/', '-'), ':', '-')) = ?",
		instanceID, reqLower).
		First(&im).Error
	if err == nil && im.CustomModelConfig != "" {
		var cfg resolvedCustomConfig
		if jerr := json.Unmarshal([]byte(im.CustomModelConfig), &cfg); jerr == nil && cfg.ModelID != "" {
			return customResolved(&cfg), true
		}
	}

	return nil, false
}

// resolvePrimaryFromBindings 在 instance_models 表中查找 role='primary' 的绑定。
// 用于 OpenClaw gateway 配置场景：模型绑定在 instance_models 中，但 instance 主字段为空。
func resolvePrimaryFromBindings(ctx context.Context, instanceID uint) (*ResolvedModel, bool) {
	var im InstanceModel
	if err := DB(ctx).Where("instance_id = ? AND role = ?", instanceID, ModelRolePrimary).
		First(&im).Error; err != nil {
		return nil, false
	}

	if im.AIModelID > 0 {
		var m AIModel
		if err := DB(ctx).Where("id = ? AND enabled = ?", im.AIModelID, true).First(&m).Error; err == nil {
			return builtinResolved(&m), true
		}
	}

	if im.CustomModelConfig != "" {
		var cfg resolvedCustomConfig
		if err := json.Unmarshal([]byte(im.CustomModelConfig), &cfg); err == nil && cfg.ModelID != "" {
			return customResolved(&cfg), true
		}
	}

	return nil, false
}

func builtinResolved(m *AIModel) *ResolvedModel {
	return &ResolvedModel{
		AIModelID:         m.ID,
		ModelID:           m.ModelID,
		ModelName:         m.ModelName,
		Provider:          m.Provider,
		ModelType:         m.ModelType,
		URL:               m.URL,
		APIKey:            m.APIKey,
		QuotaDay:          m.QuotaDay,
		IsCustom:          false,
		UsageBucketKey:    m.ID,
		CustomHTTPHeaders: m.GetCustomHTTPHeaders(),
	}
}

func customResolved(cfg *resolvedCustomConfig) *ResolvedModel {
	return &ResolvedModel{
		AIModelID:         0,
		CustomModelID:     cfg.ModelID,
		ModelID:           cfg.ModelID,
		ModelName:         cfg.ModelName,
		Provider:          cfg.Provider,
		ModelType:         cfg.ModelType,
		URL:               cfg.URL,
		APIKey:            cfg.APIKey,
		QuotaDay:          -1, // 自定义模型不做模型级配额限制（仍受 user/global 限制）
		IsCustom:          true,
		UsageBucketKey:    0, // 保持与现有 daily_usage_summary 聚合键一致
		CustomHTTPHeaders: cfg.CustomHTTPHeaders,
	}
}

// CleanupInstanceModelsByAIModelID 物理删除所有绑定到指定 ai_model_id 的 instance_models 记录，
// 并自动处理 primary 提升和 instances.ai_model_id 同步。
//
// 删除前，若某实例的被删模型是 primary 且还有 fallback，则自动提升 sort_order 最小的 fallback
// 为 primary，并同步更新 instances.ai_model_id 为新 primary 的 AIModelID。
// 若某实例删除后无模型，则重置 instances.ai_model_id 为 0。
//
// 返回受影响的实例 ID 列表（去重）。
func CleanupInstanceModelsByAIModelID(tx *gorm.DB, aiModelID uint) ([]uint, error) {
	var instanceIDs []uint
	if err := tx.Model(&InstanceModel{}).
		Where("ai_model_id = ?", aiModelID).
		Pluck("DISTINCT instance_id", &instanceIDs).Error; err != nil {
		return nil, err
	}

	for _, instID := range instanceIDs {
		// 检查该实例是否还有其他 primary（说明被删的不是 primary，或还有其他 primary）
		var otherPrimaryCount int64
		if err := tx.Model(&InstanceModel{}).
			Where("instance_id = ? AND role = ? AND ai_model_id != ?", instID, ModelRolePrimary, aiModelID).
			Count(&otherPrimaryCount).Error; err != nil {
			return nil, err
		}

		var newAIModelID uint = 0

		if otherPrimaryCount == 0 {
			// 被删的是唯一 primary，尝试提升 fallback
			var nextPrimary InstanceModel
			if err := tx.Where("instance_id = ? AND ai_model_id != ?", instID, aiModelID).
				Order("sort_order ASC").
				First(&nextPrimary).Error; err == nil {
				// 有 fallback 可提升为 primary
				if err := tx.Model(&nextPrimary).Update("role", ModelRolePrimary).Error; err != nil {
					return nil, err
				}
				newAIModelID = nextPrimary.AIModelID
			}
			// 无剩余模型时 newAIModelID 保持 0
		} else {
			// 还有其他 primary，ai_model_id 保持现状
			continue
		}

		// 同步更新 instances.ai_model_id
		if err := tx.Model(&Instance{}).Where("id = ?", instID).Update("ai_model_id", newAIModelID).Error; err != nil {
			return nil, err
		}
	}

	// 物理删除绑定记录
	if err := tx.Unscoped().Where("ai_model_id = ?", aiModelID).Delete(&InstanceModel{}).Error; err != nil {
		return nil, err
	}
	return instanceIDs, nil
}

// HardDeleteInstanceModels 物理删除指定实例的所有模型绑定（含软删除残留）。
// 适用于重装、迁移等"推倒重来"场景。
// tx 应来自 model.DB(ctx).Transaction() 内部，已包含租户上下文。
func HardDeleteInstanceModels(tx *gorm.DB, instanceID uint) error {
	return tx.Unscoped().Where("instance_id = ?", instanceID).Delete(&InstanceModel{}).Error
}

// HardDeleteInstanceModelByKey 物理删除指定实例的单条绑定记录（精确匹配 ai_model_id + role）。
// 适用于 injectDefaultModel 失败回滚场景。
// tx 应来自 model.DB(ctx).Transaction() 内部，已包含租户上下文。
func HardDeleteInstanceModelByKey(tx *gorm.DB, instanceID, aiModelID uint, role string) error {
	return tx.Unscoped().Where("instance_id = ? AND ai_model_id = ? AND role = ?",
		instanceID, aiModelID, role).Delete(&InstanceModel{}).Error
}

// CleanSoftDeletedInstanceModel 清理指定 (instance_id, ai_model_id) 的软删除残留。
// 防御性方法：仅删除 deleted_at IS NOT NULL 的行，不影响有效数据。
// tx 应来自 model.DB(ctx).Transaction() 内部，已包含租户上下文。
func CleanSoftDeletedInstanceModel(tx *gorm.DB, instanceID, aiModelID uint) {
	if err := tx.Unscoped().
		Where("instance_id = ? AND ai_model_id = ? AND deleted_at IS NOT NULL", instanceID, aiModelID).
		Delete(&InstanceModel{}).Error; err != nil {
		slog.Warn("[InstanceModel] 清理软删除残留失败（非致命）",
			"instance_id", instanceID, "ai_model_id", aiModelID, "error", err)
	}
}

// CleanInstanceModelSoftDeleteRemnants 清理 instance_models 表中所有软删除残留。
// 幂等操作，启动期由 task scheduler 调用。
func CleanInstanceModelSoftDeleteRemnants(db *gorm.DB) {
	result := db.Unscoped().
		Where("deleted_at IS NOT NULL").
		Delete(&InstanceModel{})
	if result.Error != nil {
		slog.Warn("[DB] 清理 instance_models 软删除残留失败", "error", result.Error)
	} else if result.RowsAffected > 0 {
		slog.Info("[DB] 清理 instance_models 软删除残留完成", "rows", result.RowsAffected)
	}
}

// ListInstanceModels 返回指定实例所有绑定模型的轻量视图，用于 /v1/models 等接口。
// 包含 primary 和 fallback，按 (role=primary 优先, sort_order ASC) 排序。
func ListInstanceModels(ctx context.Context, instanceID uint) []ResolvedModel {
	var bindings []InstanceModel
	DB(ctx).Where("instance_id = ?", instanceID).
		Order("CASE WHEN role = 'primary' THEN 0 ELSE 1 END, sort_order ASC, id ASC").
		Find(&bindings)

	// 收集内置模型 id 批量查询，减少 N+1
	builtinIDs := make([]uint, 0, len(bindings))
	for _, b := range bindings {
		if b.AIModelID > 0 {
			builtinIDs = append(builtinIDs, b.AIModelID)
		}
	}
	builtinMap := make(map[uint]*AIModel, len(builtinIDs))
	if len(builtinIDs) > 0 {
		var ms []AIModel
		DB(ctx).Where("id IN ? AND enabled = ?", builtinIDs, true).Find(&ms)
		for i := range ms {
			builtinMap[ms[i].ID] = &ms[i]
		}
	}

	result := make([]ResolvedModel, 0, len(bindings))
	for _, b := range bindings {
		if b.AIModelID > 0 {
			m, ok := builtinMap[b.AIModelID]
			if !ok {
				continue // 已禁用/被删除的模型跳过
			}
			result = append(result, *builtinResolved(m))
			continue
		}
		// 自定义模型
		if b.CustomModelConfig == "" {
			continue
		}
		var cfg resolvedCustomConfig
		if err := json.Unmarshal([]byte(b.CustomModelConfig), &cfg); err != nil || cfg.ModelID == "" {
			continue
		}
		result = append(result, *customResolved(&cfg))
	}
	return result
}
