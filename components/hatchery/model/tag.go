package model

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Tag is one managed Tencent Cloud tag key/value pair.
type Tag struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"uniqueIndex:idx_tags_key_value;index:idx_tags_visibility,priority:1;size:191;not null;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	TagKey         string    `gorm:"uniqueIndex:idx_tags_key_value;size:128;not null" json:"tag_key"`
	TagValue       string    `gorm:"uniqueIndex:idx_tags_key_value;size:256;not null" json:"tag_value"`
	VisibilityType string    `gorm:"size:16;not null;default:'all';index:idx_tags_visibility,priority:2" json:"visibility_type"`
}

// TagVisibilityGroup binds a group-scoped tag to a user group.
type TagVisibilityGroup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_tvg_unique;index;size:191;not null;default:''" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	TagID      uint      `gorm:"uniqueIndex:idx_tvg_unique;not null;index" json:"tag_id"`
	GroupID    uint      `gorm:"uniqueIndex:idx_tvg_unique;not null;index" json:"group_id"`
}

// TagItem is the legacy default_tags JSON item shape.
type TagItem struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ParseTagItems parses legacy SiteConfig.DefaultTags JSON.
func ParseTagItems(raw string) []TagItem {
	if raw == "" {
		return []TagItem{}
	}
	var tags []TagItem
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []TagItem{}
	}
	return normalizeTagItems(tags)
}

// MarshalTagItems serializes tags using the legacy default_tags JSON shape.
func MarshalTagItems(tags []TagItem) string {
	if len(tags) == 0 {
		return "[]"
	}
	data, err := json.Marshal(normalizeTagItems(tags))
	if err != nil {
		return "[]"
	}
	return string(data)
}

func normalizeTagItems(tags []TagItem) []TagItem {
	result := make([]TagItem, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		t.Key = strings.TrimSpace(t.Key)
		if t.Key == "" {
			continue
		}
		k := t.Key + "\x00" + t.Value
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		result = append(result, t)
	}
	return result
}

// GetGlobalTagItemsForConfig returns global tags for /admin/config.
// If no global rows exist yet, it falls back to legacy default_tags.
func GetGlobalTagItemsForConfig(ctx context.Context, legacyRaw string) ([]TagItem, error) {
	var rows []Tag
	err := DB(ctx).Where("visibility_type = ?", VisibilityAll).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return ParseTagItems(legacyRaw), nil
	}
	return tagRowsToItems(rows), nil
}

// ResolveTagsForGroup returns tags that should be applied to a newly created instance.
// New table data wins once any tag row exists; legacy default_tags is only a pre-migration fallback.
func ResolveTagsForGroup(ctx context.Context, groupID uint, legacyRaw string) ([]TagItem, error) {
	var count int64
	if err := DB(ctx).Model(&Tag{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return ParseTagItems(legacyRaw), nil
	}

	var rows []Tag
	if err := DB(ctx).Where("visibility_type = ?", VisibilityAll).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if groupID > 0 {
		groupIDs, err := ClosureAncestors(ctx, groupID, true)
		if err != nil {
			return nil, err
		}
		if len(groupIDs) == 0 {
			groupIDs = []uint{groupID}
		}
		var scoped []Tag
		if err := DB(ctx).Model(&Tag{}).
			Joins("JOIN tag_visibility_groups ON tag_visibility_groups.tag_id = tags.id").
			Where("tags.visibility_type = ? AND tag_visibility_groups.group_id IN ?", VisibilityGroup, groupIDs).
			Order("tags.id ASC").
			Find(&scoped).Error; err != nil {
			return nil, err
		}
		rows = append(rows, scoped...)
	}
	return tagRowsToItems(rows), nil
}

func tagRowsToItems(rows []Tag) []TagItem {
	items := make([]TagItem, 0, len(rows))
	seenKey := make(map[string]int, len(rows))
	for _, row := range rows {
		item := TagItem{Key: row.TagKey, Value: row.TagValue}
		if idx, ok := seenKey[item.Key]; ok {
			items[idx] = item
			continue
		}
		seenKey[item.Key] = len(items)
		items = append(items, item)
	}
	return items
}

// EnsureLegacyDefaultTagsMigrated migrates old SiteConfig.DefaultTags into global tag rows, then clears the old field.
func EnsureLegacyDefaultTagsMigrated(ctx context.Context) error {
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		return ensureLegacyDefaultTagsMigratedTx(tx)
	})
}

func ensureLegacyDefaultTagsMigratedTx(tx *gorm.DB) error {
	var config SiteConfig
	if err := tx.First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	oldTags := ParseTagItems(config.DefaultTags)
	for _, item := range oldTags {
		row := Tag{
			TagKey:         item.Key,
			TagValue:       item.Value,
			VisibilityType: VisibilityAll,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	if config.DefaultTags != "" && config.DefaultTags != "[]" {
		if err := tx.Model(&config).Update("default_tags", "[]").Error; err != nil {
			return err
		}
	}
	return nil
}

func ReplaceGlobalTags(ctx context.Context, items []TagItem) error {
	items = normalizeTagItems(items)
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureLegacyDefaultTagsMigratedTx(tx); err != nil {
			return err
		}
		var oldIDs []uint
		if err := tx.Model(&Tag{}).Where("visibility_type = ?", VisibilityAll).Pluck("id", &oldIDs).Error; err != nil {
			return err
		}
		if len(oldIDs) > 0 {
			if err := tx.Where("tag_id IN ?", oldIDs).Delete(&TagVisibilityGroup{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("visibility_type = ?", VisibilityAll).Delete(&Tag{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			if err := tx.Create(&Tag{TagKey: item.Key, TagValue: item.Value, VisibilityType: VisibilityAll}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ReplaceTags(ctx context.Context, items []TagWithScope) error {
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureLegacyDefaultTagsMigratedTx(tx); err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&TagVisibilityGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&Tag{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			key := strings.TrimSpace(item.Key)
			visibilityType := item.VisibilityType
			if visibilityType == "" {
				visibilityType = VisibilityAll
			}
			groupIDs := dedupeGroupIDs(item.GroupIDs)
			if err := validateTagInput(key, visibilityType, groupIDs); err != nil {
				return err
			}
			row := Tag{TagKey: key, TagValue: item.Value, VisibilityType: visibilityType}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if err := replaceTagVisibilityGroupsTx(tx, row.ID, visibilityType, groupIDs); err != nil {
				return err
			}
		}
		return nil
	})
}

type TagWithScope struct {
	Key            string
	Value          string
	VisibilityType string
	GroupIDs       []uint
}

func ListTags(ctx context.Context) ([]Tag, map[uint][]uint, error) {
	var rows []Tag
	if err := DB(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.VisibilityType == VisibilityGroup {
			ids = append(ids, row.ID)
		}
	}
	groups := make(map[uint][]uint, len(ids))
	if len(ids) == 0 {
		return rows, groups, nil
	}
	var bindings []TagVisibilityGroup
	if err := DB(ctx).Where("tag_id IN ?", ids).Order("tag_id ASC, group_id ASC").Find(&bindings).Error; err != nil {
		return nil, nil, err
	}
	for _, b := range bindings {
		groups[b.TagID] = append(groups[b.TagID], b.GroupID)
	}
	return rows, groups, nil
}

func CreateTag(ctx context.Context, key, value, visibilityType string, groupIDs []uint) (Tag, error) {
	key = strings.TrimSpace(key)
	if err := validateTagInput(key, visibilityType, groupIDs); err != nil {
		return Tag{}, err
	}
	var row Tag
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureLegacyDefaultTagsMigratedTx(tx); err != nil {
			return err
		}
		row = Tag{TagKey: key, TagValue: value, VisibilityType: visibilityType}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return replaceTagVisibilityGroupsTx(tx, row.ID, visibilityType, groupIDs)
	})
	return row, err
}

func UpdateTag(ctx context.Context, id uint, key, value, visibilityType string, groupIDs []uint) (Tag, error) {
	key = strings.TrimSpace(key)
	if err := validateTagInput(key, visibilityType, groupIDs); err != nil {
		return Tag{}, err
	}
	var row Tag
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureLegacyDefaultTagsMigratedTx(tx); err != nil {
			return err
		}
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		row.TagKey = key
		row.TagValue = value
		row.VisibilityType = visibilityType
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		return replaceTagVisibilityGroupsTx(tx, row.ID, visibilityType, groupIDs)
	})
	return row, err
}

func DeleteTag(ctx context.Context, id uint) error {
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureLegacyDefaultTagsMigratedTx(tx); err != nil {
			return err
		}
		var row Tag
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := tx.Where("tag_id = ?", id).Delete(&TagVisibilityGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
}

// IsGroupUsedByTagVisibility checks whether a user group is referenced by tag scope.
func IsGroupUsedByTagVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&TagVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func replaceTagVisibilityGroupsTx(tx *gorm.DB, tagID uint, visibilityType string, groupIDs []uint) error {
	if err := tx.Where("tag_id = ?", tagID).Delete(&TagVisibilityGroup{}).Error; err != nil {
		return err
	}
	if visibilityType != VisibilityGroup {
		return nil
	}
	for _, gid := range dedupeGroupIDs(groupIDs) {
		if err := tx.Create(&TagVisibilityGroup{TagID: tagID, GroupID: gid}).Error; err != nil {
			return err
		}
	}
	return nil
}

func validateTagInput(key, visibilityType string, groupIDs []uint) error {
	if key == "" {
		return hcommon.I18nError(i18n.MsgTagKeyEmpty)
	}
	if visibilityType != VisibilityAll && visibilityType != VisibilityGroup {
		return hcommon.I18nError(i18n.MsgInvalidVisibilityForModel)
	}
	if visibilityType == VisibilityGroup && len(groupIDs) == 0 {
		return hcommon.I18nError(i18n.MsgTagGroupRequired)
	}
	return nil
}

func dedupeGroupIDs(ids []uint) []uint {
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if slices.Contains(result, id) {
			continue
		}
		result = append(result, id)
	}
	return result
}
