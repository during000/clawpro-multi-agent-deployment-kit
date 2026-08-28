package model

import (
	"context"
	"time"

	"hatchery/common"

	"gorm.io/gorm"
)

// ImageHistory records official image update history globally.
// It intentionally has no Identifier field: official image history is shared by all tenants.
type ImageHistory struct {
	gorm.Model
	ImageId      string    `gorm:"type:varchar(191);not null;index" json:"image_id"`
	AgentType    string    `gorm:"type:varchar(32);not null;default:'';index" json:"agent_type"`
	AgentVersion string    `gorm:"type:varchar(64);not null;default:'';index" json:"agent_version"`
	PublishedAt  time.Time `gorm:"not null;index" json:"published_at"`
}

func (ImageHistory) TableName() string { return "image_history" }

// LatestOfficialImageHistories returns the latest non-deleted history item for each official image.
func LatestOfficialImageHistories(ctx context.Context) (map[string]ImageHistory, error) {
	imageIDs := make([]string, 0, len(common.CandidateImages))
	for _, candidate := range common.CandidateImages {
		imageIDs = append(imageIDs, candidate.ImageId)
	}
	return LatestImageHistoriesByImageID(ctx, imageIDs)
}

// LatestImageHistoriesByImageID returns the latest non-deleted history item for each image_id.
func LatestImageHistoriesByImageID(ctx context.Context, imageIDs []string) (map[string]ImageHistory, error) {
	result := make(map[string]ImageHistory, len(imageIDs))
	if len(imageIDs) == 0 {
		return result, nil
	}
	var histories []ImageHistory
	if err := DBGlobal(ctx).Where("image_id IN ?", imageIDs).
		Order("published_at desc, id desc").Find(&histories).Error; err != nil {
		return nil, err
	}
	for _, history := range histories {
		if _, exists := result[history.ImageId]; !exists {
			result[history.ImageId] = history
		}
	}
	return result, nil
}
