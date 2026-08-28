package model

import (
	"context"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AssetTypeSkill = "skill"
	AssetTypeRule  = "rule"
)

// AssetBindingConfigType 将资产类型转换为绑定表中的配置类型。
// 仅资产绑定使用 asset_*；skill/rule 则表示资源的可见范围绑定。
func AssetBindingConfigType(assetType string) (string, bool) {
	switch assetType {
	case AssetTypeSkill:
		return AssetBindingTypeSkill, true
	case AssetTypeRule:
		return AssetBindingTypeRule, true
	default:
		return "", false
	}
}

// AssetBindingTypeSkill / AssetBindingTypeRule 表示目标直接选择的资产。
// 项目与分组均使用同一组类型，只是分别存入各自的绑定表。
const (
	AssetBindingTypeSkill = "asset_skill"
	AssetBindingTypeRule  = "asset_rule"
)

// ProjectConfigTypeSkill / ProjectConfigTypeRule 表示资源的项目应用范围：
// 资源可在该项目的资产页被选择，但不会仅因这条绑定被 TeamAI 下发。
const (
	ProjectConfigTypeSkill = "skill"
	ProjectConfigTypeRule  = "rule"
)

const (
	// LocalAgentScopeUser 表示用户级资源；其下发资产只能来自当前用户绑定的分组。
	LocalAgentScopeUser = "user"
	// LocalAgentScopeWorkspace 表示 Workspace 级资源；其下发资产只能来自该 Workspace 绑定的项目。
	LocalAgentScopeWorkspace = "workspace"
)

// ProjectVisibilityConfigTypes 是项目工具应用范围；这些绑定存在时不能删除项目。
// 后续接入新的项目级工具类型时，只需在此集合追加即可。
var ProjectVisibilityConfigTypes = []string{ProjectConfigTypeSkill, ProjectConfigTypeRule}

// ProjectAssetConfigTypes 是项目当前直接资产；删除项目时应清理，但不作为阻塞项。
var ProjectAssetConfigTypes = []string{AssetBindingTypeSkill, AssetBindingTypeRule}

// ErrInvalidProjectName 供 controller 层通过 errors.Is 精确识别并返回国际化错误。
var ErrInvalidProjectName = hcommon.I18nError(i18n.MsgProjectInvalidName)

// Project 是 Hatchery 自主管理的扁平项目，不与 OneID 或组织树关联。
type Project struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier  string    `gorm:"size:191;not null;default:'';uniqueIndex:uk_project_name" json:"-"`
	Name        string    `gorm:"size:191;not null;uniqueIndex:uk_project_name" json:"name"`
	Description string    `gorm:"type:text;not null" json:"description"`
	SyncMode    string    `gorm:"size:32;not null;default:'continuous'" json:"sync_mode"` // initial_only | continuous；项目固定 continuous
	CreatedBy   uint      `gorm:"not null;default:0" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectMember struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier string    `gorm:"size:191;not null;default:'';uniqueIndex:uk_project_member;index:idx_project_member_user,priority:1" json:"-"`
	ProjectID  uint      `gorm:"not null;uniqueIndex:uk_project_member;index:idx_project_member_user,priority:3" json:"project_id"`
	UserID     uint      `gorm:"not null;uniqueIndex:uk_project_member;index:idx_project_member_user,priority:2" json:"user_id"`
	CreatedBy  uint      `gorm:"not null;default:0" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type ProjectConfigBinding struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier string    `gorm:"size:191;not null;default:'';uniqueIndex:uk_project_config;index:idx_project_config_resource,priority:1" json:"-"`
	ProjectID  uint      `gorm:"not null;uniqueIndex:uk_project_config;index:idx_project_config_project,priority:1" json:"project_id"`
	ConfigType string    `gorm:"size:32;not null;uniqueIndex:uk_project_config;index:idx_project_config_project,priority:2;index:idx_project_config_resource,priority:2" json:"config_type"`
	ConfigKey  string    `gorm:"size:191;not null;uniqueIndex:uk_project_config;index:idx_project_config_resource,priority:3" json:"config_key"`
	ValueJSON  string    `gorm:"type:varchar(4096);not null;default:'{}'" json:"value_json"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type LocalAgentScopeBinding struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier string     `gorm:"size:191;not null;default:'';uniqueIndex:uk_local_agent_scope;index:idx_local_agent_project,priority:1" json:"-"`
	InstanceID uint       `gorm:"not null;uniqueIndex:uk_local_agent_scope" json:"instance_id"`
	Scope      string     `gorm:"size:16;not null;uniqueIndex:uk_local_agent_scope" json:"scope"`
	ScopeKey   string     `gorm:"size:512;not null;default:'';uniqueIndex:uk_local_agent_scope" json:"scope_key"`
	ScopeName  string     `gorm:"size:191;not null;default:''" json:"scope_name"`
	IDEType    string     `gorm:"column:ide_type;size:32;not null;default:''" json:"ide_type"`
	GroupID    uint       `gorm:"not null;default:0;index" json:"group_id"`
	ProjectID  uint       `gorm:"not null;default:0;index:idx_local_agent_project,priority:2" json:"project_id"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// LocalAgentTargetInstance 表示一个目标下的本地 Agent 及其匹配的 scope 绑定。
// group 目标只匹配 scope=user，project 目标只匹配 scope=workspace。
type LocalAgentTargetInstance struct {
	Instance      Instance
	ScopeBindings []LocalAgentScopeBinding
}

// ListLocalAgentInstancesByScope 查询指定 scope 绑定到目标的本地 Agent。
// 该函数只读取 local_agent_scope_bindings，不依赖 instances.local_agent_resources JSON。
func ListLocalAgentInstancesByScope(ctx context.Context, scope string, targetID uint) ([]LocalAgentTargetInstance, error) {
	return ListLocalAgentInstancesByScopeWithDB(DB(ctx), scope, targetID)
}

// ListLocalAgentInstancesByScopeWithDB 是事务安全版本，供下发与管理端复用。
func ListLocalAgentInstancesByScopeWithDB(tx *gorm.DB, scope string, targetID uint) ([]LocalAgentTargetInstance, error) {
	return ListLocalAgentInstancesByScopeTargetsWithDB(tx, scope, []uint{targetID})
}

// ListLocalAgentInstancesByScopeTargets 批量查询多个目标下的本地 Agent。
// 分组树场景用它一次聚合当前分组及全部后代，避免循环查询数据库。
func ListLocalAgentInstancesByScopeTargets(ctx context.Context, scope string, targetIDs []uint) ([]LocalAgentTargetInstance, error) {
	return ListLocalAgentInstancesByScopeTargetsWithDB(DB(ctx), scope, targetIDs)
}

// ListLocalAgentInstancesByScopeTargetsWithDB 是批量查询的事务安全版本。
func ListLocalAgentInstancesByScopeTargetsWithDB(tx *gorm.DB, scope string, targetIDs []uint) ([]LocalAgentTargetInstance, error) {
	targetIDs = uniqueProjectIDs(targetIDs)
	if len(targetIDs) == 0 {
		return []LocalAgentTargetInstance{}, nil
	}
	query := tx.Model(&LocalAgentScopeBinding{})
	switch scope {
	case LocalAgentScopeUser:
		query = query.Where("scope = ? AND group_id IN ?", scope, targetIDs)
	case LocalAgentScopeWorkspace:
		query = query.Where("scope = ? AND project_id IN ?", scope, targetIDs)
	default:
		return []LocalAgentTargetInstance{}, nil
	}
	var bindings []LocalAgentScopeBinding
	if err := query.Order("instance_id ASC, scope_key ASC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	return collectLocalAgentTargetInstances(tx, bindings)
}

func collectLocalAgentTargetInstances(tx *gorm.DB, bindings []LocalAgentScopeBinding) ([]LocalAgentTargetInstance, error) {
	if len(bindings) == 0 {
		return []LocalAgentTargetInstance{}, nil
	}
	ids := make([]uint, 0, len(bindings))
	byInstance := make(map[uint][]LocalAgentScopeBinding)
	for _, binding := range bindings {
		if _, exists := byInstance[binding.InstanceID]; !exists {
			ids = append(ids, binding.InstanceID)
		}
		byInstance[binding.InstanceID] = append(byInstance[binding.InstanceID], binding)
	}
	var instances []Instance
	if err := tx.Where("id IN ? AND source = ?", ids, InstanceSourceLocal).Find(&instances).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint]Instance, len(instances))
	for _, instance := range instances {
		byID[instance.ID] = instance
	}
	items := make([]LocalAgentTargetInstance, 0, len(ids))
	for _, id := range ids {
		instance, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, LocalAgentTargetInstance{Instance: instance, ScopeBindings: byInstance[id]})
	}
	return items, nil
}

func NormalizeProjectName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidProjectName
	}
	if len([]rune(name)) > 191 {
		return "", ErrInvalidProjectName
	}
	return name, nil
}

func ReplaceProjectMembers(tx *gorm.DB, projectID uint, userIDs []uint, operatorID uint) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&ProjectMember{}).Error; err != nil {
		return err
	}
	seen := make(map[uint]struct{}, len(userIDs))
	records := make([]ProjectMember, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		records = append(records, ProjectMember{ProjectID: projectID, UserID: userID, CreatedBy: operatorID})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
}

func ReplaceProjectConfigBindings(tx *gorm.DB, projectID uint, configType string, keys []string) error {
	if err := tx.Where("project_id = ? AND config_type = ?", projectID, configType).Delete(&ProjectConfigBinding{}).Error; err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(keys))
	records := make([]ProjectConfigBinding, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		records = append(records, ProjectConfigBinding{ProjectID: projectID, ConfigType: configType, ConfigKey: key})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
}

// ReplaceResourceProjectBindings 全量替换某个工具 slug 的项目应用范围。
func ReplaceResourceProjectBindings(tx *gorm.DB, configType, configKey string, projectIDs []uint) error {
	if err := tx.Where("config_type = ? AND config_key = ?", configType, configKey).
		Delete(&ProjectConfigBinding{}).Error; err != nil {
		return err
	}
	seen := make(map[uint]struct{}, len(projectIDs))
	records := make([]ProjectConfigBinding, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		if projectID == 0 {
			continue
		}
		if _, ok := seen[projectID]; ok {
			continue
		}
		seen[projectID] = struct{}{}
		records = append(records, ProjectConfigBinding{
			ProjectID: projectID, ConfigType: configType, ConfigKey: configKey,
		})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
}

// CleanupProjectBindings 删除逻辑资源的应用范围与项目资产绑定。
func CleanupProjectBindings(tx *gorm.DB, configType, configKey string) error {
	configTypes := []string{configType}
	switch configType {
	case ProjectConfigTypeSkill:
		configTypes = append(configTypes, AssetBindingTypeSkill)
	case ProjectConfigTypeRule:
		configTypes = append(configTypes, AssetBindingTypeRule)
	}
	return tx.Where("config_type IN ? AND config_key = ?", configTypes, configKey).
		Delete(&ProjectConfigBinding{}).Error
}

func ValidateProjectIDs(tx *gorm.DB, projectIDs []uint) error {
	projectIDs = uniqueProjectIDs(projectIDs)
	if len(projectIDs) == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&Project{}).Where("id IN ?", projectIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(projectIDs)) {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func uniqueProjectIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func ListUserProjects(ctx context.Context, userID uint) ([]Project, error) {
	var projects []Project
	err := DB(ctx).Model(&Project{}).Joins("JOIN project_members ON project_members.project_id = projects.id").Where("project_members.user_id = ?", userID).Order("project_members.created_at ASC, project_members.id ASC").Find(&projects).Error
	return projects, err
}
