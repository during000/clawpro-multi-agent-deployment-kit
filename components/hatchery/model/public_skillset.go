package model

import "time"

// PublicSkillSet 收藏的公共技能包（Skillset）
// 本地仅保存 slug 用于标识，其他信息解包时从 SkillHub API 实时获取
type PublicSkillSet struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_public_skillsets_identifier_slug;index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Slug       string    `gorm:"not null;default:'';uniqueIndex:idx_public_skillsets_identifier_slug" json:"slug"`
}

// TableName 指定表名，与 SQL 已有表名保持一致（GORM 默认会生成 public_skill_sets）
func (PublicSkillSet) TableName() string { return "public_skillsets" }
