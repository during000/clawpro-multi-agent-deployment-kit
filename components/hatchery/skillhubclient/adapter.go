package skillhubclient

import (
	"time"
)

// HatcherySkill 技能列表响应中的单条技能记录（保持与 Hatchery 老格式一致）。
// 用于格式适配层，将 SkillHub API 响应转换为前端期望的旧格式。
type HatcherySkill struct {
	ID               uint                      `json:"id"`
	Slug             string                    `json:"slug"`
	Name             string                    `json:"name"`
	Description       string                    `json:"description"`
	Version          string                    `json:"version"`
	VersionMajor     int                       `json:"version_major"`
	VersionMinor     int                       `json:"version_minor"`
	VersionPatch     int                       `json:"version_patch"`
	FileSize         int64                     `json:"file_size"`
	DistributeCount  int                       `json:"distribute_count"`
	Changelog        string                    `json:"changelog"`
	VisibilityType   string                    `json:"visibility_type"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	Categories       []map[string]interface{}  `json:"categories"`
	VisibilityGroups []map[string]interface{}  `json:"visibility_groups"`
	LastTask         *map[string]interface{}  `json:"last_task"`
	SecurityScan     *map[string]interface{}  `json:"security_scan"`
}

// ConvertSkillHubListToHatchery 将 SkillHub 技能列表响应转换为 Hatchery 老格式。
// Phase 2 仅转换技能列表基本字段，categories/visibility_groups 填空数组，
// last_task/security_scan 保持 nil，确保前端不报错。
func ConvertSkillHubListToHatchery(resp *SkillListResponse) []HatcherySkill {
	if resp == nil {
		return []HatcherySkill{}
	}
	result := make([]HatcherySkill, 0, len(resp.Items))
	for _, item := range resp.Items {
		createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, item.UpdatedAt)
		// 时间解析失败时兜底——不影响列表展示
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		result = append(result, HatcherySkill{
			ID:               uint(item.ID),
			Slug:             item.Slug,
			Name:             item.DisplayName,
			Description:      item.Summary,
			Version:          item.Version,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
			VisibilityType:   "all",
			Categories:       []map[string]interface{}{},
			VisibilityGroups: []map[string]interface{}{},
			// LastTask 和 SecurityScan 保持 nil，前端兼容
		})
	}
	return result
}
