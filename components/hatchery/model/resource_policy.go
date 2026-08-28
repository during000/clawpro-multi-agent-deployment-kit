package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"hatchery/i18n"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultResourcePolicyName = "企业默认资源策略"

var (
	ErrResourcePolicyNotFound      = errors.New("resource policy not found")
	ErrResourcePolicyNameConflict  = errors.New("resource policy name conflict")
	ErrResourcePolicyGroupOccupied = errors.New("resource policy group occupied")
	ErrResourcePolicyGroupNotFound = errors.New("resource policy group not found")
	ErrDefaultResourcePolicy       = errors.New("default resource policy is protected")
)

// ResourcePolicy is a tenant-scoped resource creation policy. ConfigJSON owns
// the policy content; application scope uses GroupConfigBinding with
// ConfigTypeResourcePolicy and the policy ID as ConfigKey.
type ResourcePolicy struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier string    `gorm:"size:191;not null;default:'';uniqueIndex:idx_rp_ident_name,priority:1;index:idx_rp_ident_default,priority:1" json:"-"`
	Name       string    `gorm:"size:128;not null;uniqueIndex:idx_rp_ident_name,priority:2" json:"name"`
	IsDefault  bool      `gorm:"not null;default:false;index:idx_rp_ident_default,priority:2" json:"is_default"`
	ConfigJSON string    `gorm:"type:text;not null" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DisplayName localizes only the enterprise default policy. Ordinary policy
// names are tenant-authored data and must be returned verbatim.
func (p ResourcePolicy) DisplayName(ctx context.Context) string {
	if p.IsDefault {
		return i18n.T(ctx, i18n.MsgDefaultResourcePolicyName)
	}
	return p.Name
}

// ResolvedResourcePolicy records the winning policy and the group that binds
// it directly. SourceGroupID=0 denotes the tenant default policy.
type ResolvedResourcePolicy struct {
	Policy        ResourcePolicy
	SourceGroupID uint
	Depth         int
}

func defaultResourcePolicyConfigFromTemplate(template string) string {
	raw := extractResourceConfigFromTemplate(template)
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "{}" {
		raw = extractResourceConfigFromTemplate(DefaultCVMTemplate)
	}
	return raw
}

func initialDefaultResourcePolicyConfig(ctx context.Context) string {
	return defaultResourcePolicyConfigFromTemplate(GetSiteConfig(ctx).CVMTemplate)
}

// GetOrCreateDefaultResourcePolicy lazily materializes the tenant default.
// The tenant/name unique index makes concurrent first access idempotent.
func GetOrCreateDefaultResourcePolicy(ctx context.Context) (*ResourcePolicy, error) {
	initialConfig := initialDefaultResourcePolicyConfig(ctx)
	var result ResourcePolicy
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("name = ?", DefaultResourcePolicyName).First(&result).Error
		if err == nil {
			if !result.IsDefault {
				return ErrResourcePolicyNameConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		candidate := ResourcePolicy{
			Name:       DefaultResourcePolicyName,
			IsDefault:  true,
			ConfigJSON: initialConfig,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "identifier"}, {Name: "name"}},
			DoNothing: true,
		}).Create(&candidate).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", DefaultResourcePolicyName).First(&result).Error; err != nil {
			return err
		}
		if !result.IsDefault {
			return ErrResourcePolicyNameConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func GetResourcePolicy(ctx context.Context, id uint) (*ResourcePolicy, error) {
	var policy ResourcePolicy
	if err := DB(ctx).First(&policy, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResourcePolicyNotFound
		}
		return nil, err
	}
	return &policy, nil
}

func ListResourcePolicies(ctx context.Context, page, pageSize int) ([]ResourcePolicy, int64, error) {
	if _, err := GetOrCreateDefaultResourcePolicy(ctx); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var total int64
	db := DB(ctx).Model(&ResourcePolicy{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var policies []ResourcePolicy
	if err := db.Order("is_default DESC, id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&policies).Error; err != nil {
		return nil, 0, err
	}
	return policies, total, nil
}

func resourcePolicyKey(policyID uint) string {
	return strconv.FormatUint(uint64(policyID), 10)
}

func parseResourcePolicyKey(key string) (uint, error) {
	id, err := strconv.ParseUint(key, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid resource policy binding key %q", key)
	}
	return uint(id), nil
}

func GetResourcePolicyGroups(ctx context.Context, policyIDs []uint) (map[uint][]UserGroup, error) {
	result := make(map[uint][]UserGroup, len(policyIDs))
	if len(policyIDs) == 0 {
		return result, nil
	}
	keys := make([]string, 0, len(policyIDs))
	for _, policyID := range policyIDs {
		if policyID != 0 {
			keys = append(keys, resourcePolicyKey(policyID))
		}
	}
	if len(keys) == 0 {
		return result, nil
	}
	var bindings []GroupConfigBinding
	if err := DB(ctx).Where("config_type = ? AND config_key IN ?", ConfigTypeResourcePolicy, keys).
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	groupIDs := make([]uint, 0, len(bindings))
	for _, binding := range bindings {
		groupIDs = append(groupIDs, binding.GroupID)
	}
	if len(groupIDs) == 0 {
		return result, nil
	}
	var groups []UserGroup
	if err := DB(ctx).Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, err
	}
	groupsByID := make(map[uint]UserGroup, len(groups))
	for _, group := range groups {
		groupsByID[group.ID] = group
	}
	for _, binding := range bindings {
		policyID, err := parseResourcePolicyKey(binding.ConfigKey)
		if err != nil {
			return nil, err
		}
		if group, ok := groupsByID[binding.GroupID]; ok {
			result[policyID] = append(result[policyID], group)
		}
	}
	for policyID := range result {
		sort.Slice(result[policyID], func(i, j int) bool {
			left, right := result[policyID][i], result[policyID][j]
			if left.FullPath == right.FullPath {
				return left.ID < right.ID
			}
			return left.FullPath < right.FullPath
		})
	}
	return result, nil
}

func normalizePolicyGroupIDs(groupIDs []uint) []uint {
	seen := make(map[uint]struct{}, len(groupIDs))
	result := make([]uint, 0, len(groupIDs))
	for _, id := range groupIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func lockAndValidatePolicyGroups(tx *gorm.DB, policyID uint, groupIDs []uint) error {
	groupIDs = normalizePolicyGroupIDs(groupIDs)
	if len(groupIDs) == 0 {
		return ErrResourcePolicyGroupNotFound
	}
	var groups []UserGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", groupIDs).Order("id ASC").Find(&groups).Error; err != nil {
		return err
	}
	if len(groups) != len(groupIDs) {
		return ErrResourcePolicyGroupNotFound
	}

	var occupied []GroupConfigBinding
	if err := tx.Where("config_type = ? AND group_id IN ?", ConfigTypeResourcePolicy, groupIDs).
		Find(&occupied).Error; err != nil {
		return err
	}
	policyKey := resourcePolicyKey(policyID)
	for _, binding := range occupied {
		if binding.ConfigKey != policyKey {
			return ErrResourcePolicyGroupOccupied
		}
	}
	return nil
}

func replaceResourcePolicyGroups(tx *gorm.DB, policyID uint, groupIDs []uint) error {
	groupIDs = normalizePolicyGroupIDs(groupIDs)
	if err := lockAndValidatePolicyGroups(tx, policyID, groupIDs); err != nil {
		return err
	}
	policyKey := resourcePolicyKey(policyID)
	if err := tx.Where("config_type = ? AND config_key = ?", ConfigTypeResourcePolicy, policyKey).
		Delete(&GroupConfigBinding{}).Error; err != nil {
		return err
	}
	bindings := make([]GroupConfigBinding, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		bindings = append(bindings, GroupConfigBinding{
			ConfigType: ConfigTypeResourcePolicy,
			ConfigKey:  policyKey,
			GroupID:    groupID,
			ValueJSON:  "{}",
		})
	}
	return tx.Create(&bindings).Error
}

func CreateResourcePolicy(ctx context.Context, name, configJSON string, groupIDs []uint) (*ResourcePolicy, error) {
	name = strings.TrimSpace(name)
	if name == DefaultResourcePolicyName {
		return nil, ErrDefaultResourcePolicy
	}
	var policy ResourcePolicy
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		policy = ResourcePolicy{Name: name, ConfigJSON: configJSON}
		if err := tx.Create(&policy).Error; err != nil {
			if IsDuplicateError(err) {
				return ErrResourcePolicyNameConflict
			}
			return err
		}
		return replaceResourcePolicyGroups(tx, policy.ID, groupIDs)
	})
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func UpdateResourcePolicy(ctx context.Context, id uint, name, configJSON string, groupIDs []uint) (*ResourcePolicy, error) {
	name = strings.TrimSpace(name)
	var policy ResourcePolicy
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&policy, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourcePolicyNotFound
			}
			return err
		}
		if policy.IsDefault {
			if (name != "" && name != policy.Name) || len(normalizePolicyGroupIDs(groupIDs)) != 0 {
				return ErrDefaultResourcePolicy
			}
			if err := tx.Model(&policy).Update("config_json", configJSON).Error; err != nil {
				return err
			}
			policy.ConfigJSON = configJSON
			return nil
		}
		if name == DefaultResourcePolicyName {
			return ErrDefaultResourcePolicy
		}
		if err := replaceResourcePolicyGroups(tx, policy.ID, groupIDs); err != nil {
			return err
		}
		if err := tx.Model(&policy).Updates(map[string]interface{}{
			"name":        name,
			"config_json": configJSON,
		}).Error; err != nil {
			if IsDuplicateError(err) {
				return ErrResourcePolicyNameConflict
			}
			return err
		}
		policy.Name = name
		policy.ConfigJSON = configJSON
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func DeleteResourcePolicy(ctx context.Context, id uint) error {
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		var policy ResourcePolicy
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&policy, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourcePolicyNotFound
			}
			return err
		}
		if policy.IsDefault {
			return ErrDefaultResourcePolicy
		}
		if err := tx.Where("config_type = ? AND config_key = ?", ConfigTypeResourcePolicy, resourcePolicyKey(id)).
			Delete(&GroupConfigBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&policy).Error
	})
}

func GetDirectResourcePoliciesByGroup(ctx context.Context, groupIDs []uint) (map[uint]ResourcePolicy, error) {
	result := make(map[uint]ResourcePolicy)
	groupIDs = normalizePolicyGroupIDs(groupIDs)
	if len(groupIDs) == 0 {
		return result, nil
	}
	bindings, err := GetBindingsByGroups(ctx, groupIDs, ConfigTypeResourcePolicy)
	if err != nil {
		return nil, err
	}
	policyIDs := make([]uint, 0, len(bindings))
	policyIDByGroup := make(map[uint]uint, len(bindings))
	for _, binding := range bindings {
		policyID, err := parseResourcePolicyKey(binding.ConfigKey)
		if err != nil {
			return nil, err
		}
		if existing, ok := policyIDByGroup[binding.GroupID]; ok && existing != policyID {
			return nil, ErrResourcePolicyGroupOccupied
		}
		policyIDByGroup[binding.GroupID] = policyID
		policyIDs = append(policyIDs, policyID)
	}
	if len(policyIDs) == 0 {
		return result, nil
	}
	var policies []ResourcePolicy
	if err := DB(ctx).Where("id IN ?", policyIDs).Find(&policies).Error; err != nil {
		return nil, err
	}
	policiesByID := make(map[uint]ResourcePolicy, len(policies))
	for _, policy := range policies {
		policiesByID[policy.ID] = policy
	}
	for groupID, policyID := range policyIDByGroup {
		policy, ok := policiesByID[policyID]
		if !ok {
			return nil, ErrResourcePolicyNotFound
		}
		result[groupID] = policy
	}
	return result, nil
}

// ResolveEffectiveResourcePolicy returns the nearest directly bound policy, or
// lazily materializes the tenant default when no group or ancestor binds one.
func ResolveEffectiveResourcePolicy(ctx context.Context, groupID uint) (*ResolvedResourcePolicy, error) {
	if groupID == 0 {
		policy, err := GetOrCreateDefaultResourcePolicy(ctx)
		if err != nil {
			return nil, err
		}
		return &ResolvedResourcePolicy{Policy: *policy, Depth: -1}, nil
	}

	var closures []GroupClosure
	if err := DB(ctx).Where("descendant_id = ?", groupID).Order("depth ASC").Find(&closures).Error; err != nil {
		return nil, err
	}
	if len(closures) == 0 {
		return nil, ErrResourcePolicyGroupNotFound
	}
	ancestorIDs := make([]uint, 0, len(closures))
	for _, closure := range closures {
		ancestorIDs = append(ancestorIDs, closure.AncestorID)
	}
	direct, err := GetDirectResourcePoliciesByGroup(ctx, ancestorIDs)
	if err != nil {
		return nil, err
	}
	for _, closure := range closures {
		policy, ok := direct[closure.AncestorID]
		if !ok {
			continue
		}
		return &ResolvedResourcePolicy{
			Policy:        policy,
			SourceGroupID: closure.AncestorID,
			Depth:         closure.Depth,
		}, nil
	}
	policy, err := GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		return nil, err
	}
	return &ResolvedResourcePolicy{Policy: *policy, Depth: -1}, nil
}
