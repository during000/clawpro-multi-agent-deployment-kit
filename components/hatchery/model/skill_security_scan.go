package model

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// SkillSecurityScan 技能安全扫描记录
// 存储 CSIP SkillScan 检测结果，维度为 slug+version（每个版本独立扫描）
type SkillSecurityScan struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	Identifier     string         `gorm:"uniqueIndex:idx_hash_engine;index;default:''" json:"-"` // 多租户标识
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	SkillID        uint           `gorm:"index;not null" json:"skill_id"`
	SkillVersion   string         `gorm:"not null;default:''" json:"skill_version"`
	ContentHash    string         `gorm:"uniqueIndex:idx_hash_engine;not null" json:"content_hash"`   // SHA256 hash: sha256:...
	EngineVersion  int            `gorm:"uniqueIndex:idx_hash_engine;not null" json:"engine_version"` // 扫描引擎版本
	Status         string         `gorm:"index;not null;default:'SCANNING'" json:"status"`             // SCANNING, SUCCESS, FAILED
	RiskLevel      string         `gorm:"default:''" json:"risk_level"`                                // benign, suspicious, malicious
	PrimaryRuleID  string         `gorm:"default:''" json:"primary_rule_id"`                           // 主要风险规则 ID
	SecurityScore  int            `gorm:"default:100" json:"security_score"`                           // 0-100
	ScanResultData json.RawMessage `gorm:"type:json" json:"scan_result_data"`                           // 完整扫描结果 JSON
	ReportURL      string         `gorm:"default:''" json:"report_url"`                                // 签名报告地址
	ScannedAt      *time.Time     `json:"scanned_at"`                                                  // 扫描完成时间
	FailedAt       *time.Time     `json:"failed_at"`                                                   // 扫描失败时间
	FailureMessage string         `gorm:"type:text" json:"failure_message"`                             // 失败原因（TEXT 类型，MySQL 不带 DEFAULT）
}

// SkillScanViolation 技能扫描违规项
type SkillScanViolation struct {
	ID                  uint      `gorm:"primarykey" json:"id"`
	Identifier          string    `gorm:"index;default:''" json:"-"` // 多租户标识
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	SkillSecurityScanID uint      `gorm:"index;not null" json:"skill_security_scan_id"`
	RuleID              string    `gorm:"not null;default:''" json:"rule_id"`
	RuleName            string    `gorm:"not null;default:''" json:"rule_name"`
	ScanType            string    `gorm:"not null;default:''" json:"scan_type"` // AI, STATIC
	Description         string    `gorm:"type:text;not null" json:"description"`
	CapabilityTag       string    `gorm:"default:''" json:"capability_tag"`
	CapabilityTagName   string    `gorm:"default:''" json:"capability_tag_name"`
}

// GetLatestSkillSecurityScan 获取技能最新的安全扫描记录
func GetLatestSkillSecurityScan(ctx context.Context, skillID uint) (*SkillSecurityScan, error) {
	var scan SkillSecurityScan
	if err := DB(ctx).Where("skill_id = ?", skillID).
		Order("created_at DESC").
		First(&scan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &scan, nil
}

// GetSkillSecurityScanByHash 根据 ContentHash 和 EngineVersion 查询扫描记录
func GetSkillSecurityScanByHash(ctx context.Context, contentHash string, engineVersion int) (*SkillSecurityScan, error) {
	var scan SkillSecurityScan
	if err := DB(ctx).Where("content_hash = ? AND engine_version = ?", contentHash, engineVersion).
		First(&scan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &scan, nil
}

// GetPendingScanRecords 获取所有待扫描完成的记录（Status=SCANNING）
func GetPendingScanRecords(ctx context.Context) ([]SkillSecurityScan, error) {
	var scans []SkillSecurityScan
	if err := DB(ctx).Where("status = ?", "SCANNING").
		Order("created_at ASC").
		Find(&scans).Error; err != nil {
		return nil, err
	}
	return scans, nil
}

// GetScanViolations 获取扫描记录的所有违规项
func GetScanViolations(ctx context.Context, scanID uint) ([]SkillScanViolation, error) {
	var violations []SkillScanViolation
	if err := DB(ctx).Where("skill_security_scan_id = ?", scanID).
		Order("rule_id ASC, scan_type ASC").
		Find(&violations).Error; err != nil {
		return nil, err
	}
	return violations, nil
}

// GetSkillsSecurityStatus 批量查询技能安全状态（列表页用）
// 对每个 skillID 返回该技能最新的一条扫描记录
// 返回 map[skillID] → *SkillSecurityScan（nil 表示未检测）
func GetSkillsSecurityStatus(ctx context.Context, skillIDs []uint) (map[uint]*SkillSecurityScan, error) {
	if len(skillIDs) == 0 {
		return map[uint]*SkillSecurityScan{}, nil
	}

	// 子查询：每个 skill_id 取最新的 scan ID
	// 使用 Model(&SkillSecurityScan{}) 保证 GORM 回调自动注入 identifier 条件
	subQuery := DB(ctx).Model(&SkillSecurityScan{}).
		Select("MAX(id)").
		Where("skill_id IN ?", skillIDs).
		Group("skill_id")

	var scans []SkillSecurityScan
	if err := DB(ctx).Where("id IN (?)", subQuery).Find(&scans).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]*SkillSecurityScan, len(scans))
	for i := range scans {
		result[scans[i].SkillID] = &scans[i]
	}
	return result, nil
}
