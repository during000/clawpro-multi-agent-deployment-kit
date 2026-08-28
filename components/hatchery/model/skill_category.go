package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// SkillCategory 技能分类
type SkillCategory struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Identifier  string         `gorm:"uniqueIndex:idx_category_name_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"uniqueIndex:idx_category_name_identifier;not null" json:"name"`
	Description string         `gorm:"type:text;not null" json:"description"`
}

var predefinedCategories = []struct {
	Name          string
	Description   string
	NameEn        string
	DescriptionEn string
}{
	{"通用办公", "文档总结、邮件润色、PPT 大纲、翻译助手", "General Office", "Document Summarization, Email Proofreading, PPT Outline, Translation Assistant"},
	{"研发工具", "代码 Review、接口调试、技术文档解析、架构建议", "Development Tools", "Code Review, API Debugging, Technical Document Parsing, Architecture Suggestions"},
	{"系统运维", "资源巡检、环境部署、日志分析、告警诊断", "System Operations", "Resource Inspection, Environment Deployment, Log Analysis, Alarm Diagnosis"},
	{"质量测试", "用例生成、自动化脚本编写、Bug 辅助定位", "Quality Assurance", "Test Case Generation, Automation Script Writing, Bug Assistance"},
	{"需求设计", "需求评审、PRD 辅助写作、UI/UX 设计灵感", "Requirement Design", "Requirement Review, PRD Assistance, UI/UX Design Inspiration"},
	{"信息检索", "企业知识库查询、竞品实时监控、技术趋势搜索", "Information Retrieval", "Enterprise Knowledge Base Query, Competitor Real-Time Monitoring, Technical Trend Search"},
	{"项目管理", "进度汇总、周报自动生成、风险预警、任务拆解", "Project Management", "Progress Summary, Weekly Report Generation, Risk Warning, Task Decomposition"},
	{"数据分析", "SQL 自动编写、报表解释、数据清洗逻辑", "Data Analysis", "SQL Auto-Generation, Report Interpretation, Data Cleaning Logic"},
	{"安全合规", "权限审计、代码漏洞扫描、合规性自查", "Security Compliance", "Permission Audit, Code Vulnerability Scanning, Compliance Self-Check"},
	{"其他", "其他分类", "Other", "Other Category"},
}

// SeedCategories inserts the predefined skill categories into the database.
// Categories that already exist (matched by name, including soft-deleted) are skipped.
// tx 为调用方传入的事务句柄。
func SeedCategories(ctx context.Context, tx *gorm.DB) error {
	// 批量查出已存在的名称（含软删除）
	names := make([]string, len(predefinedCategories)*2)
	for i, cat := range predefinedCategories {
		names[i*2] = cat.Name
		names[i*2+1] = cat.NameEn
	}
	var existing []SkillCategory
	if err := tx.Unscoped().Where("name IN ?", names).Find(&existing).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillCatSeedQuery)
	}
	existSet := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		existSet[e.Name] = struct{}{}
	}

	defaultLang := hcommon.DefaultLangFromCtx(ctx)

	// 收集需要插入的记录，一次批量写入
	var toCreate []SkillCategory
	for _, cat := range predefinedCategories {
		_, enExists := existSet[cat.NameEn]
		_, zhExists := existSet[cat.Name]
		if !zhExists && !enExists { // 中文与英文名称均不存在才插入
			var skillCategory SkillCategory
			if defaultLang == "zh" {
				skillCategory = SkillCategory{
					Name:        cat.Name,
					Description: cat.Description,
				}
			} else {
				skillCategory = SkillCategory{
					Name:        cat.NameEn,
					Description: cat.DescriptionEn,
				}
			}

			toCreate = append(toCreate, skillCategory)
		}
	}
	if len(toCreate) > 0 {
		if err := tx.Create(&toCreate).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCatSeedInsert)
		}
	}
	return nil
}

// SkillCategoryMapping 技能-分类多对多关联表
type SkillCategoryMapping struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Identifier string `gorm:"uniqueIndex:idx_skill_category_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	SkillID    uint   `gorm:"uniqueIndex:idx_skill_category_identifier;not null" json:"skill_id"`
	CategoryID uint   `gorm:"uniqueIndex:idx_skill_category_identifier;not null" json:"category_id"`
}
