package model

import (
	"context"
	"encoding/json"
	"errors"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"strings"
	"time"

	"gorm.io/gorm"
)

const MaxCustomAgentTypeNameLen = 20

// CustomAgentType 管理员创建的自定义智能体类型。
// Name 同时作为标识和展示名；CompatibleWith 为空表示不兼容任何内置类型。
type CustomAgentType struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Identifier     string         `gorm:"uniqueIndex:idx_custom_agent_type_identifier_name;index;default:''" json:"-"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"uniqueIndex:idx_custom_agent_type_identifier_name;size:20;not null" json:"name"`
	CompatibleWith string         `gorm:"size:32;not null;default:''" json:"compatible_with"`
}

func ValidateCustomAgentTypeName(name string) error {
	if name == "" {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNameRequired)
	}
	if strings.TrimSpace(name) != name {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNameHasTrimSpace)
	}
	if len([]rune(name)) > MaxCustomAgentTypeNameLen {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNameTooLong, MaxCustomAgentTypeNameLen)
	}
	if IsBuiltinAgentType(name) {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNameConflictBuiltin, name)
	}
	return nil
}

func ValidateCompatibleAgentType(compatibleWith string) error {
	if compatibleWith == "" {
		return nil
	}
	if !IsBuiltinAgentType(compatibleWith) {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeCompatibleInvalid)
	}
	return nil
}

func GetCustomAgentTypeByName(ctx context.Context, name string) (*CustomAgentType, error) {
	if name == "" || IsBuiltinAgentType(name) {
		return nil, nil
	}
	var t CustomAgentType
	err := DB(ctx).Where("name = ?", name).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgCustomAgentTypeQueryDBFailed)
	}
	return &t, nil
}

func ListCustomAgentTypes(ctx context.Context) ([]CustomAgentType, error) {
	var types []CustomAgentType
	err := DB(ctx).Order("id asc").Find(&types).Error
	return types, err
}

func IsCustomAgentType(ctx context.Context, name string) bool {
	if name == "" || IsBuiltinAgentType(name) {
		return false
	}
	t, err := GetCustomAgentTypeByName(ctx, name)
	return err == nil && t != nil
}

func CreateCustomAgentType(ctx context.Context, name, compatibleWith string) (*CustomAgentType, error) {
	if err := ValidateCustomAgentTypeName(name); err != nil {
		return nil, err
	}
	if err := ValidateCompatibleAgentType(compatibleWith); err != nil {
		return nil, err
	}
	existing, err := GetCustomAgentTypeByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, hcommon.I18nError(i18n.MsgCustomAgentTypeAlreadyExists, name)
	}
	t := &CustomAgentType{Name: name, CompatibleWith: compatibleWith}
	if err := DB(ctx).Create(t).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCustomAgentTypeCreateDBFailed)
	}
	return t, nil
}

func DeleteCustomAgentType(ctx context.Context, name string) error {
	if strings.TrimSpace(name) != name {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNameHasTrimSpace)
	}
	if IsBuiltinAgentType(name) {
		return hcommon.I18nError(i18n.MsgBuiltinAgentTypeCannotDel)
	}
	t, err := GetCustomAgentTypeByName(ctx, name)
	if err != nil {
		return err
	}
	if t == nil {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeNotFound, name)
	}
	if GetDefaultAgentType(ctx) == name {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeIsDefaultDelete)
	}
	var enabledImageCount int64
	if IsAgentTypeEnabled(ctx, name) {
		if err := DB(ctx).Model(&AIImage{}).Where("agent_type = ? AND enabled = ?", name, true).Count(&enabledImageCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCustomAgentTypeDeleteDBFailed)
		}
		if enabledImageCount > 0 {
			return hcommon.I18nError(i18n.MsgCustomAgentTypeDisableBeforeDelete)
		}
	}
	var count int64
	if err := DB(ctx).Model(&Instance{}).Where("agent_type = ?", name).Count(&count).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCustomAgentTypeDeleteDBFailed)
	}
	if count > 0 {
		return hcommon.I18nError(i18n.MsgCustomAgentTypeHasInstances)
	}
	if txErr := DB(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		if err := tx.Where("agent_type = ?", name).Delete(&AIImage{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(t).Error; err != nil {
			return err
		}

		var config SiteConfig
		if err := tx.First(&config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		disabled := config.GetDisabledAgentTypes()
		filtered := make([]string, 0, len(disabled))
		changed := false
		for _, disabledType := range disabled {
			if disabledType == name {
				changed = true
				continue
			}
			filtered = append(filtered, disabledType)
		}
		if !changed {
			return nil
		}
		data, err := json.Marshal(filtered)
		if err != nil {
			return err
		}
		return tx.Model(&config).Update("disabled_agent_types", string(data)).Error
	}); txErr != nil {
		return hcommon.I18nRichError(txErr, i18n.MsgCustomAgentTypeDeleteDBFailed)
	}
	return nil
}
