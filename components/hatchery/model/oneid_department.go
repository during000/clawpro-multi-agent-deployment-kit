package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// OneIDDepartmentRecord 存储从 OneID 部门接口拉取的部门信息。
// 独立于用户画像中的 DepartmentsJSON，用于补全整棵部门树（特别是根部门和中间节点）。
type OneIDDepartmentRecord struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement"`
	Identifier         string    `gorm:"uniqueIndex:idx_dept_identifier;index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	DepartmentID       string    `gorm:"uniqueIndex:idx_dept_identifier;not null"`         // OneID 部门 ID
	DepartmentName     string    `gorm:"default:''"`
	DepartmentParentID string    `gorm:"index;default:''"` // 父部门 ID，根部门为空
	HasChild           bool      `gorm:"default:false"`
	DirectUserCount    int       `gorm:"default:0"`
	RecurveUserCount   int       `gorm:"default:0"`
	SyncedAt           time.Time `gorm:"not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (OneIDDepartmentRecord) TableName() string {
	return "oneid_departments"
}

// UpsertDepartment 插入或更新一条部门记录。
func UpsertDepartment(ctx context.Context, dept *OneIDDepartmentRecord) error {
	dept.SyncedAt = time.Now()
	return DB(ctx).Where(OneIDDepartmentRecord{DepartmentID: dept.DepartmentID}).
		Assign(*dept).
		FirstOrCreate(dept).Error
}

// GetDepartment 按 department_id 查询部门记录。未找到返回 nil, nil。
func GetDepartment(ctx context.Context, deptID string) (*OneIDDepartmentRecord, error) {
	if deptID == "" {
		return nil, nil
	}
	var d OneIDDepartmentRecord
	if err := DB(ctx).Where("department_id = ?", deptID).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// DeleteDepartmentsNotIn 删除不在 activeDeptIDs 集合中的部门记录。
// 用于同步完成后清理已从 OneID 中删除或移出授权范围的部门。
func DeleteDepartmentsNotIn(ctx context.Context, activeDeptIDs []string) (int64, error) {
	if len(activeDeptIDs) == 0 {
		// 如果没有任何活跃部门，说明同步可能失败了，不做删除以防误删
		return 0, nil
	}
	result := DB(ctx).Where("department_id NOT IN ?", activeDeptIDs).Delete(&OneIDDepartmentRecord{})
	return result.RowsAffected, result.Error
}

// BuildFullDeptMap 从 oneid_departments 表构建全局 department_id → 部门信息 映射。
// 这是构建部门树/路径的唯一数据源，不再从用户画像中补充（用户画像存的是用户归属部门，
// 可能包含不在数据授权范围内的上级部门，会导致部门树不准确）。
func BuildFullDeptMap(ctx context.Context) map[string]OneIDDepartment {
	var records []OneIDDepartmentRecord
	DB(ctx).Find(&records)

	m := make(map[string]OneIDDepartment, len(records))
	for _, r := range records {
		m[r.DepartmentID] = OneIDDepartment{
			DepartmentID:       r.DepartmentID,
			DepartmentName:     r.DepartmentName,
			DepartmentParentID: r.DepartmentParentID,
		}
	}

	return m
}
