package controller

import (
	"slices"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

type distributionSelection struct {
	InstanceIDs []uint   `json:"instance_ids"`
	SelectAll   bool     `json:"select_all"`
	Statuses    []string `json:"statuses"`
	GroupIDs    []uint   `json:"group_ids"`
	Search      string   `json:"search"`
}

func (s distributionSelection) validate() error {
	if s.SelectAll {
		if len(s.InstanceIDs) > 0 {
			return hcommon.I18nError(i18n.MsgDistributionTargetModeConflict)
		}
		return nil
	}
	if len(s.Statuses) > 0 || len(s.GroupIDs) > 0 || s.Search != "" {
		return hcommon.I18nError(i18n.MsgDistributionSelectAllRequired)
	}
	if len(s.InstanceIDs) == 0 {
		return hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty)
	}
	return nil
}

func applyDistributionSearch(query *gorm.DB, search string) *gorm.DB {
	if search == "" {
		return query
	}
	if runes := []rune(search); len(runes) > 50 {
		search = string(runes[:50])
	}
	like := "%" + escapeSQLLike(search) + "%"
	return query.Where(
		"instances.name LIKE ? OR instances.instance_id LIKE ? OR u.username LIKE ?",
		like, like, like,
	)
}

func normalizeDistributionStatuses(statuses, allowed, transitional []string) ([]string, error) {
	if len(statuses) == 0 {
		return slices.Clone(allowed), nil
	}

	seen := make(map[string]struct{}, len(statuses))
	normalized := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if slices.Contains(transitional, status) {
			return nil, hcommon.I18nError(i18n.MsgDistributionTransitionalStatus, status)
		}
		if !slices.Contains(allowed, status) {
			return nil, hcommon.I18nError(i18n.MsgDistributionStatusInvalid, status)
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		normalized = append(normalized, status)
	}
	return normalized, nil
}
