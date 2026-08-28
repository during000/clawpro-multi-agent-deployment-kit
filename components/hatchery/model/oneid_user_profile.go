package model

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
)

// EscapeLike 转义 LIKE 模式中的特殊字符 % 和 _，防止用户输入意外匹配。
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// DeptIDLikePattern 生成按 department_id 搜索 departments_json 列的 LIKE 模式。
func DeptIDLikePattern(deptID string) string {
	return `%"department_id":"` + EscapeLike(deptID) + `"%`
}

// OneIDUserProfile 存储从 OneID 通讯录接口拉取的用户扩展信息。
// 通过 OneIDSub 与 users 表关联，不修改 users 表结构。
// 字段以 OneID batch_query_condition 接口返回的 AccountUserInfo 为准。
type OneIDUserProfile struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	Identifier     string `gorm:"uniqueIndex:idx_profile_sub_identifier;index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	OneIDSub       string `gorm:"uniqueIndex:idx_profile_sub_identifier;not null"`         // 对应 users.one_id_sub
	UnionID        string `gorm:"default:''"`
	Name           string `gorm:"default:''"`
	Email          string `gorm:"default:''"`
	Mobile         string `gorm:"default:''"`
	Position       string `gorm:"default:''"`
	EmployeeNumber string `gorm:"default:''"`
	Status         string `gorm:"default:''"`
	// 主部门信息（冗余存储，方便查询）
	MainDeptID       string `gorm:"default:''"`
	MainDeptName     string `gorm:"default:''"`
	MainDeptParentID string `gorm:"default:''"`
	// 全量部门列表（JSON 数组）
	DepartmentsJSON string    `gorm:"type:text;default:'[]'"`
	SyncedAt        time.Time `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OneIDDepartment 是部门信息的结构，对应 DepartmentsJSON 中的元素。
type OneIDDepartment struct {
	DepartmentID       string `json:"department_id"`
	DepartmentName     string `json:"department_name"`
	DepartmentParentID string `json:"department_parent_id"`
	IsMainDepartment   bool   `json:"is_main_department"`
}

// UpsertOneIDUserProfile 插入或更新 OneID 用户画像。
// 以 OneIDSub 作为唯一键，幂等操作。
func UpsertOneIDUserProfile(ctx context.Context, profile *OneIDUserProfile) error {
	profile.SyncedAt = time.Now()
	return DB(ctx).Where(OneIDUserProfile{OneIDSub: profile.OneIDSub}).
		Assign(*profile).
		FirstOrCreate(profile).Error
}

// GetOneIDUserProfile 根据 sub 查询用户画像。未找到时返回 nil, nil。
func GetOneIDUserProfile(ctx context.Context, sub string) (*OneIDUserProfile, error) {
	if sub == "" {
		return nil, nil
	}
	var p OneIDUserProfile
	if err := DB(ctx).Where("one_id_sub = ?", sub).First(&p).Error; err != nil {
		return nil, nil
	}
	return &p, nil
}

// GetOneIDUserProfiles 根据 sub 列表批量查询用户画像。
func GetOneIDUserProfiles(ctx context.Context, subs []string) []OneIDUserProfile {
	if len(subs) == 0 {
		return nil
	}
	var profiles []OneIDUserProfile
	DB(ctx).Where("one_id_sub IN ?", subs).Find(&profiles)
	return profiles
}

// BuildGlobalDeptMap 从所有用户 profile 的 DepartmentsJSON 中收集全局
// department_id → OneIDDepartment 映射。
// 用于构建任意部门的完整层级路径（通过 parent_id 向上查找）。
func BuildGlobalDeptMap(ctx context.Context) map[string]OneIDDepartment {
	var jsons []string
	DB(ctx).Model(&OneIDUserProfile{}).
		Where("departments_json != '' AND departments_json != '[]'").
		Pluck("departments_json", &jsons)

	m := make(map[string]OneIDDepartment)
	for _, raw := range jsons {
		var depts []OneIDDepartment
		if err := json.Unmarshal([]byte(raw), &depts); err != nil {
			slog.Warn("BuildGlobalDeptMap: failed to parse departments_json, skipping", "err", err)
			continue
		}
		for _, d := range depts {
			m[d.DepartmentID] = d
		}
	}
	return m
}
