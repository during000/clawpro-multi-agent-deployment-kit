package controller

import (
	"context"
	"encoding/json"

	"hatchery/model"
)

// userDeptInfo 是单个用户的 OneID 部门聚合信息。
//
// 由 enrichUserDepartments 计算填充，被 /admin/users 与 /admin/instances 等
// 列表接口共享使用，作为 JSON 响应中部门字段（department / departments /
// department_path）的中间载体。
type userDeptInfo struct {
	Department     string         // 主部门名（OneID 画像 main_dept_name），无画像时为空串
	Departments    []deptWithPath // 完整部门列表，每项含本部门的 department_path
	DepartmentPath string         // 主部门完整路径，如 "OpenClaw企业版体验/新组/市场组/市场二组"
}

// enrichUserDepartments 根据用户切片批量构造每个用户的 OneID 部门聚合信息。
//
// 返回 user_id → userDeptInfo 的映射；只为本批次中确实有 OneID 画像的用户生成 entry，
// 其余用户在返回 map 中没有对应键（调用方按 ok 判断后回退到默认值）。
//
// 性能短路：当传入用户中没有任何 OneIDSub（典型场景：未开通 OneID 的部署、
// 或当前页只有纯密码登录的本地用户），不会触发 oneid_user_profiles 与
// oneid_departments 的查询，直接返回空 map，零额外 DB 调用。
//
// 调用方语义：
//   - 单次调用对应"列表接口的当前一页用户"，在分页接口里不需要为全量用户预热。
//   - 反序列化失败、profile 缺失、部门 ID 在全局映射中找不到等情况均按 best-effort
//     降级，不返回 error，对应字段保持零值（空串 / 空 path）。
func enrichUserDepartments(ctx context.Context, users []model.User) map[uint]userDeptInfo {
	info := make(map[uint]userDeptInfo, len(users))
	if len(users) == 0 {
		return info
	}

	// 收集本批次有 OneIDSub 的用户，建立 sub → user_id 映射
	subs := make([]string, 0, len(users))
	userBySub := make(map[string]uint, len(users))
	for _, u := range users {
		if u.OneIDSub != nil && *u.OneIDSub != "" {
			subs = append(subs, *u.OneIDSub)
			userBySub[*u.OneIDSub] = u.ID
		}
	}
	// 短路：本批次零 OneID 用户 → 跳过 oneid_user_profiles 与 oneid_departments 的查询
	if len(subs) == 0 {
		return info
	}

	// 批量查 OneID 画像（一次 IN 查询）
	profiles := model.GetOneIDUserProfiles(ctx, subs)
	if len(profiles) == 0 {
		return info
	}

	// 仅在确实拿到画像后才扫全局部门表，构建 department_id → OneIDDepartment 映射
	globalDeptMap := model.BuildFullDeptMap(ctx)

	for i := range profiles {
		p := &profiles[i]
		uid, ok := userBySub[p.OneIDSub]
		if !ok {
			continue
		}
		d := userDeptInfo{Department: p.MainDeptName}
		if p.DepartmentsJSON != "" && p.DepartmentsJSON != "[]" {
			var depts []model.OneIDDepartment
			if err := json.Unmarshal([]byte(p.DepartmentsJSON), &depts); err == nil {
				deptsWithPath := make([]deptWithPath, len(depts))
				for j, dept := range depts {
					deptsWithPath[j] = deptWithPath{
						OneIDDepartment: dept,
						DepartmentPath:  buildDepartmentPath(globalDeptMap, dept.DepartmentID),
					}
				}
				d.Departments = deptsWithPath
				d.DepartmentPath = buildDepartmentPath(globalDeptMap, p.MainDeptID)
			}
		}
		info[uid] = d
	}
	return info
}
