package controller

// oneid_sync.go — OneID 通讯录同步
//
// 功能：
//  1. 获取租户级 access_token（client_credentials 模式，内存缓存）
//  2. 调 batch_query_condition 接口拉取用户部门信息
//  3. 登录时触发单用户同步（SyncOneIDUserProfile）
//  4. 定时全量同步（StartOneIDProfileSyncer）
//
// 依赖：SiteConfig.OneIDAPIBaseURL / OneIDClientID / OneIDPrivateKeyJWK / OneIDTokenURL

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// ── 通过 Gateway 代理全量同步（遍历部门树）────────────────────────────────────

// syncAffectedUser 记录同步过程中被禁用/删除的用户信息。
type syncAffectedUser struct {
	Username        string  `json:"username"`
	InstanceCount   int64   `json:"instance_count"`    // 名下 OpenClaw 数量
	Action          string  `json:"action"`            // "disabled" / "hard_deleted"
	VpcId           *string `json:"vpc_id"`            // 用户关联的 VPC ID，nil 表示无 VPC
	VpcHasResources bool    `json:"vpc_has_resources"` // VPC 下是否有资源占用
}

// syncResult 是 syncViaGateway 的返回值，用于手动同步接口返回详情。
type syncResult struct {
	AffectedUsers []syncAffectedUser `json:"affected_users,omitempty"` // 被禁用/删除的用户列表

	// 🆕 v6.12 P1 OneID 部门落地相关
	DeptLanded            bool                          `json:"-"`                                 // 本次是否走了部门落地逻辑
	DeptGroupCount        int64                         `json:"dept_group_count"`                  // 同步完成后的 oneid_dept 组数量
	AffectedDeptGroups    []usergroup.AffectedDeptGroup `json:"affected_dept_groups,omitempty"`    // 本次新打 to_be_deleted 的分组明细（group_id + full_path）
	ChangedParentGroupIDs []uint                        `json:"change_parent_group_ids,omitempty"` // 🆕 v6.13：本次发生了父节点切换的 oneid_dept 组 ID 列表
	MoveGroupUserIDs      []usergroup.MovedGroupUser    `json:"move_group_user_ids,omitempty"`     // 🆕 v6.13：本次被移出 oneid_dept 分组的 (user_id, from_group_id) 列表
	LandingFailures       []usergroup.LandingFailure    `json:"landing_failures,omitempty"`        // 🐛 v6 hotfix: 本次 landing 未落地的部门明细（曾被 silent continue 吞掉）
}

// buildAffectedUser 构建被影响用户的详情，包括实例数和 VPC 信息。
// 注意：此函数仅用于同步结果展示，查询失败时使用默认值（instance_count=0, vpc_has_resources=false）继续，
// 不影响同步主流程。日志级别为 Warn，便于事后排查数据不准确的情况。
func buildAffectedUser(ctx context.Context, user *model.User, action string) syncAffectedUser {
	var instCount int64
	if err := model.DB(ctx).Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&instCount).Error; err != nil {
		slog.Warn("buildAffectedUser: failed to count instances, defaulting to 0",
			"user_id", user.ID, "err", err)
	}

	au := syncAffectedUser{
		Username:      user.Username,
		InstanceCount: instCount,
		Action:        action,
	}

	if user.VpcId != "" {
		au.VpcId = &user.VpcId
		if hasRes, err := vpcHasResources(ctx, user.VpcId); err != nil {
			slog.Warn("buildAffectedUser: failed to check VPC resources, defaulting to false",
				"user_id", user.ID, "vpc_id", user.VpcId, "err", err)
		} else {
			au.VpcHasResources = hasRes
		}
	}

	return au
}

// cleanupOutOfScopeResult 是 cleanupOutOfScopeUsers 的返回值。
type cleanupOutOfScopeResult struct {
	disabled      int
	hardDeleted   int
	affectedUsers []syncAffectedUser
}

// cleanupOutOfScopeUsers 删除/禁用已不在权限范围内的本地用户。
//
// 查出所有有 one_id_sub 的本地用户（含已软删除的），
// 如果其 sub 不在 syncedSubs 集合中：
//   - 无资源 → 硬删除（彻底清理）
//   - 有资源且未软删除 → 软删除（禁用）
//   - 有资源且已软删除 → 跳过（已经是禁用状态）
//
// syncIncomplete=true 时跳过此步骤，避免因网络抖动导致误删仍在职用户。
// 安全阈值：如果要处理的用户超过本地 OneID 用户总数的 50%，
// 视为数据源异常，仅软删除、不硬删除。
func cleanupOutOfScopeUsers(ctx context.Context, syncedSubs map[string]bool, syncIncomplete bool) cleanupOutOfScopeResult {
	result := cleanupOutOfScopeResult{}

	if len(syncedSubs) == 0 || syncIncomplete {
		return result
	}

	var localUsers []model.User
	if err := model.DB(ctx).Unscoped().Where("one_id_sub IS NOT NULL AND one_id_sub != ''").Find(&localUsers).Error; err != nil {
		slog.Error("cleanupOutOfScopeUsers: failed to query local users", "err", err)
		return result
	}

	for _, lu := range localUsers {
		if lu.OneIDSub == nil || syncedSubs[*lu.OneIDSub] {
			continue
		}
		user := lu // 局部副本，避免传递迭代变量指针

		// 无资源 → 硬删除（物理清理），有资源 → 软删除（禁用）
		if tryHardDeleteUser(ctx, &user) {
			result.hardDeleted++
			slog.Info("cleanupOutOfScopeUsers: user hard-deleted (not in scope, no resources)",
				"sub", *user.OneIDSub, "username", user.Username)
			result.affectedUsers = append(result.affectedUsers, syncAffectedUser{
				Username:      user.Username,
				InstanceCount: 0,
				Action:        "hard_deleted",
			})
		} else if !user.DeletedAt.Valid {
			// 活跃用户有资源 → 软删除
			if err := model.DB(ctx).Delete(&user).Error; err != nil {
				slog.Warn("cleanupOutOfScopeUsers: disable user failed",
					"sub", *user.OneIDSub, "username", user.Username, "err", err)
			} else {
				result.disabled++
				slog.Info("cleanupOutOfScopeUsers: user soft-disabled (not in scope, has resources)",
					"sub", *user.OneIDSub, "username", user.Username)
				result.affectedUsers = append(result.affectedUsers, buildAffectedUser(ctx, &user, "disabled"))
			}
		} else {
			// 已软删除且有资源 → 维持现状，报告
			result.affectedUsers = append(result.affectedUsers, buildAffectedUser(ctx, &user, "disabled"))
		}
	}

	return result
}

// syncViaGateway 通过 Gateway 的 /api/oneid-contacts 代理端点遍历部门树，
// 拉取企业下全量成员（包括未登录过的），并自动创建本地用户和同步画像。
// 使用分布式锁确保多实例部署时同一时刻只有一个节点执行同步。
// 返回同步结果详情（被禁用/删除的用户列表），定时同步忽略返回值。
func SyncViaGateway(ctx context.Context) *syncResult {
	sr := &syncResult{}
	// taskCtx 是传入 ctx 的别名，供内部所有 DB 操作复用
	taskCtx := ctx

	lock, err := model.TryLock(ctx, "oneid_sync")
	if err != nil {
		slog.Debug("[OneIDSync] 拿不到分布式锁，跳过本次同步", "error", err)
		return sr
	}
	defer lock.Release()

	enterpriseID := hcommon.TenantIDFromCtx(ctx)
	if enterpriseID == "" {
		slog.Warn("OneID gateway sync: no enterprise ID available (missing ONEID_ACCOUNT_ID)")
		return sr
	}

	slog.Info("OneID gateway sync: starting full sync", "enterprise_id", enterpriseID)

	// ── 0. 查询应用通讯录权限范围 ──────────────────────────────────────────
	// 先通过 data_scope 接口获取当前应用有权限查看的部门 ID 和用户 ID，
	// 只同步有权限的部门/用户，而非遍历整个企业组织架构。
	scope, err := gwGetAppDataScope(ctx, enterpriseID)
	if err != nil {
		slog.Error("OneID gateway sync: get app data scope failed, aborting sync", "err", err)
		return sr
	}
	slog.Info("OneID gateway sync: app data scope",
		"scoped_departments", len(scope.DepartmentIDs),
		"scoped_users", len(scope.UserIDs))

	// ── 1. 收集有权限的部门及其子部门 ──────────────────────────────────────
	// 权限范围返回的 department_ids 是授权的顶层部门，
	// 需要递归获取其所有子部门才能完整同步。
	var allDepts []gwDeptInfo
	scopedDeptSet := make(map[string]bool) // 用于去重

	for _, deptID := range scope.DepartmentIDs {
		if deptID == "" {
			continue // 跳过空的部门 ID
		}
		// 先获取部门自身信息
		deptBody, deptErr := gwCallContacts(ctx, gwContactsRequest{
			Action:       "get_department",
			AccountID:    enterpriseID,
			DepartmentID: deptID,
		})
		if deptErr != nil {
			slog.Warn("OneID gateway sync: get scoped department failed", "dept_id", deptID, "err", deptErr)
			// 即使获取详情失败，也把它加入以便拉取用户
			if !scopedDeptSet[deptID] {
				scopedDeptSet[deptID] = true
				allDepts = append(allDepts, gwDeptInfo{DepartmentID: deptID})
			}
		} else {
			// OneID v3 返回: { "data": { "department": { "department_id": "...", ... } } }
			var deptResult struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					Department struct {
						DepartmentID       string `json:"department_id"`
						DepartmentName     string `json:"department_name"`
						DepartmentParentID string `json:"department_parent_id"`
						HasChild           bool   `json:"has_child"`
					} `json:"department"`
				} `json:"data"`
			}
			if parseErr := json.Unmarshal(deptBody, &deptResult); parseErr == nil && deptResult.Code == 0 {
				dept := deptResult.Data.Department
				resolvedID := dept.DepartmentID
				if resolvedID == "" {
					resolvedID = deptID // fallback: 使用原始 deptID
				}
				slog.Info("OneID gateway sync: get_department result",
					"dept_id", deptID,
					"resolved_id", resolvedID,
					"dept_name", dept.DepartmentName,
					"has_child", dept.HasChild)
				if !scopedDeptSet[resolvedID] {
					scopedDeptSet[resolvedID] = true
					allDepts = append(allDepts, gwDeptInfo{
						DepartmentID:       resolvedID,
						DepartmentName:     dept.DepartmentName,
						DepartmentParentID: dept.DepartmentParentID,
						HasChild:           dept.HasChild,
					})
				}
			}
		}

		// 递归获取该部门下所有子部门（BFS）
		childDepts := gwListSubDepartments(ctx, enterpriseID, deptID)
		for _, cd := range childDepts {
			if !scopedDeptSet[cd.DepartmentID] {
				scopedDeptSet[cd.DepartmentID] = true
				allDepts = append(allDepts, cd)
			}
		}
	}

	slog.Info("OneID gateway sync: departments collected (scoped + children)", "count", len(allDepts))
	for i, d := range allDepts {
		slog.Info("OneID gateway sync: department",
			"index", i,
			"dept_id", d.DepartmentID,
			"dept_name", d.DepartmentName,
			"parent_id", d.DepartmentParentID,
			"has_child", d.HasChild)
	}

	// 保存部门信息
	for _, d := range allDepts {
		if err := model.UpsertDepartment(taskCtx, &model.OneIDDepartmentRecord{
			DepartmentID:       d.DepartmentID,
			DepartmentName:     d.DepartmentName,
			DepartmentParentID: d.DepartmentParentID,
			HasChild:           d.HasChild,
		}); err != nil {
			slog.Warn("OneID gateway sync: upsert department failed", "dept_id", d.DepartmentID, "err", err)
		}
	}

	// ── 1.5 清理已删除/移出授权范围的旧部门记录 ────────────────────────────────
	// 将不在本次同步结果中的本地部门记录删除，保持本地数据与 OneID 一致。
	if len(allDepts) > 0 {
		activeDeptIDs := make([]string, 0, len(allDepts))
		for deptID := range scopedDeptSet {
			activeDeptIDs = append(activeDeptIDs, deptID)
		}
		deleted, delErr := model.DeleteDepartmentsNotIn(taskCtx, activeDeptIDs)
		if delErr != nil {
			slog.Warn("OneID gateway sync: cleanup stale departments failed", "err", delErr)
		} else if deleted > 0 {
			slog.Info("OneID gateway sync: stale departments cleaned up", "deleted_count", deleted)
		}
	}

	// ── 2. 遍历有权限的部门，分页拉取用户 ────────────────────────────────────
	totalUsers := 0
	totalCreated := 0
	totalSyncDisabled := 0              // 因 OneID 停用而被软删除（禁用）的用户数
	syncedSubs := make(map[string]bool) // 记录本次同步到的所有 union_id
	syncIncomplete := false             // 任意分页失败时置 true，禁止步骤 3 删除

	for _, dept := range allDepts {
		if dept.DepartmentID == "" {
			continue // 跳过空的部门 ID，避免 Gateway 400 错误
		}
		users, ok := gwListDepartmentUsers(ctx, enterpriseID, dept.DepartmentID)
		if !ok {
			syncIncomplete = true
			slog.Warn("OneID gateway sync: department user list incomplete, disable step will be skipped",
				"dept_id", dept.DepartmentID, "dept_name", dept.DepartmentName)
		}
		deptNewCount := 0
		deptDupCount := 0
		for _, u := range users {
			if syncedSubs[u.UnionID] {
				deptDupCount++
				continue // 跨部门去重
			}
			deptNewCount++
			totalUsers++
			syncedSubs[u.UnionID] = true

			// 保存/更新用户画像
			deptJSON, _ := json.Marshal(u.Departments)
			profile := &model.OneIDUserProfile{
				OneIDSub:        u.UnionID,
				UnionID:         u.UnionID,
				Name:            u.Name,
				Email:           u.Email,
				Mobile:          u.Mobile,
				Position:        u.Position,
				EmployeeNumber:  u.EmployeeNumber,
				Status:          u.Status,
				DepartmentsJSON: string(deptJSON),
			}
			for _, d := range u.Departments {
				if d.IsMainDepartment {
					profile.MainDeptID = d.DepartmentID
					profile.MainDeptName = d.DepartmentName
					profile.MainDeptParentID = d.DepartmentParentID
					break
				}
			}
			if err := model.UpsertOneIDUserProfile(taskCtx, profile); err != nil {
				slog.Warn("OneID gateway sync: upsert profile failed", "sub", u.UnionID, "err", err)
			}

			// 自动创建/更新/禁用本地用户
			result := ensureLocalUser(taskCtx, u, enterpriseID)
			if result.Created {
				totalCreated++
			}
			if result.Disabled {
				totalSyncDisabled++
			}
		}
		slog.Info("OneID gateway sync: department users processed",
			"dept_id", dept.DepartmentID,
			"dept_name", dept.DepartmentName,
			"fetched", len(users),
			"new", deptNewCount,
			"duplicate", deptDupCount,
			"running_total", totalUsers)
	}

	// ── 2.5 同步按人授权的用户（不在授权部门下，直接按 union_id 授权） ─────────
	// data_scope 返回的 user_ids 中可能有不属于任何授权部门的用户，需要单独同步。
	var scopedOnlyUserIDs []string
	for _, uid := range scope.UserIDs {
		if !syncedSubs[uid] {
			scopedOnlyUserIDs = append(scopedOnlyUserIDs, uid)
		}
	}
	if len(scopedOnlyUserIDs) > 0 {
		slog.Info("OneID gateway sync: syncing individually scoped users",
			"count", len(scopedOnlyUserIDs),
			"user_ids", scopedOnlyUserIDs)
		// 按 100 个一批调 batch_query_users
		for i := 0; i < len(scopedOnlyUserIDs); i += 100 {
			end := i + 100
			if end > len(scopedOnlyUserIDs) {
				end = len(scopedOnlyUserIDs)
			}
			batch := scopedOnlyUserIDs[i:end]

			batchBody, batchErr := gwCallContacts(ctx, gwContactsRequest{
				Action:    "batch_query_users",
				AccountID: enterpriseID,
				UnionIDs:  batch,
			})
			if batchErr != nil {
				slog.Warn("OneID gateway sync: batch query scoped users failed", "err", batchErr)
				continue
			}
			var batchResult struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					Users []gwUserInfo `json:"users"`
				} `json:"data"`
			}
			if parseErr := json.Unmarshal(batchBody, &batchResult); parseErr != nil || batchResult.Code != 0 {
				slog.Warn("OneID gateway sync: parse batch scoped users failed",
					"code", batchResult.Code, "msg", batchResult.Msg, "err", parseErr)
				continue
			}

			var scopedUserNames []string
			for _, u := range batchResult.Data.Users {
				scopedUserNames = append(scopedUserNames, u.Name+"("+u.UnionID+")")
			}
			slog.Info("OneID gateway sync: batch scoped users result",
				"requested", len(batch),
				"returned", len(batchResult.Data.Users),
				"users", scopedUserNames)

			for _, u := range batchResult.Data.Users {
				if syncedSubs[u.UnionID] {
					continue
				}
				totalUsers++
				syncedSubs[u.UnionID] = true

				deptJSON, _ := json.Marshal(u.Departments)
				profile := &model.OneIDUserProfile{
					OneIDSub:        u.UnionID,
					UnionID:         u.UnionID,
					Name:            u.Name,
					Email:           u.Email,
					Mobile:          u.Mobile,
					Position:        u.Position,
					EmployeeNumber:  u.EmployeeNumber,
					Status:          u.Status,
					DepartmentsJSON: string(deptJSON),
				}
				for _, d := range u.Departments {
					if d.IsMainDepartment {
						profile.MainDeptID = d.DepartmentID
						profile.MainDeptName = d.DepartmentName
						profile.MainDeptParentID = d.DepartmentParentID
						break
					}
				}
				if err := model.UpsertOneIDUserProfile(taskCtx, profile); err != nil {
					slog.Warn("OneID gateway sync: upsert scoped user profile failed", "sub", u.UnionID, "err", err)
				}
				result := ensureLocalUser(taskCtx, u, enterpriseID)
				if result.Created {
					totalCreated++
				}
				if result.Disabled {
					totalSyncDisabled++
				}
			}
		}
	}

	// ── 3. 删除/禁用已不在权限范围内的本地用户 ───────────────────────────────
	cleanupResult := cleanupOutOfScopeUsers(taskCtx, syncedSubs, syncIncomplete)
	totalDisabled := cleanupResult.disabled
	totalHardDeleted := cleanupResult.hardDeleted
	sr.AffectedUsers = append(sr.AffectedUsers, cleanupResult.affectedUsers...)

	// ── 4. 同步管理员角色（通过 Gateway 的 check_role action，role_id 由 Gateway 配置提供）
	if GatewayURL != "" {
		gwSyncAdminRoles(taskCtx, enterpriseID)
	}

	slog.Info("OneID gateway sync completed",
		"scoped_departments", len(scope.DepartmentIDs),
		"scoped_users", len(scope.UserIDs),
		"total_departments_synced", len(allDepts),
		"total_users_synced", totalUsers,
		"new_users_created", totalCreated,
		"users_disabled_by_status", totalSyncDisabled,
		"users_disabled_out_of_scope", totalDisabled,
		"users_hard_deleted_out_of_scope", totalHardDeleted,
		"sync_incomplete", syncIncomplete)

	return sr
}

// gwSyncAdminRoles 通过 Gateway 的 check_role action 批量检查所有 OneID 用户的管理员角色，
// 并更新本地 users 表的 role 字段。
func gwSyncAdminRoles(ctx context.Context, enterpriseID string) {
	// 查出所有有 one_id_sub 的用户
	var users []model.User
	if err := model.DB(ctx).Where("one_id_sub IS NOT NULL AND one_id_sub != ''").Find(&users).Error; err != nil {
		slog.Warn("gwSyncAdminRoles: query users failed", "err", err)
		return
	}
	if len(users) == 0 {
		return
	}

	// 按 100 个一批调 check_role
	adminCount := 0
	for i := 0; i < len(users); i += 100 {
		end := i + 100
		if end > len(users) {
			end = len(users)
		}
		batch := users[i:end]

		var unionIDs []string
		userMap := map[string]*model.User{} // union_id -> User
		for idx := range batch {
			sub := ""
			if batch[idx].OneIDSub != nil {
				sub = *batch[idx].OneIDSub
			}
			if sub != "" {
				unionIDs = append(unionIDs, sub)
				userMap[sub] = &batch[idx]
			}
		}
		if len(unionIDs) == 0 {
			continue
		}

		body, err := gwCallContacts(ctx, gwContactsRequest{
			Action:    "check_role",
			AccountID: enterpriseID,
			UnionIDs:  unionIDs,
			// RoleID 留空，由 Gateway 使用自身配置的 admin_role_id
		})
		if err != nil {
			slog.Warn("gwSyncAdminRoles: check_role failed", "err", err)
			continue
		}

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Exists map[string]bool `json:"exists"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 {
			slog.Warn("gwSyncAdminRoles: parse check_role response failed",
				"code", result.Code, "msg", result.Msg, "err", err)
			continue
		}

		// 更新角色：遍历本批所有用户，而非只遍历 API 返回的 exists map，
		// 确保 API 未返回的用户也被降级为 user。
		for _, sub := range unionIDs {
			u := userMap[sub]
			if u == nil {
				continue
			}
			isAdmin := result.Data.Exists[sub]
			newRole := "user"
			if isAdmin {
				newRole = "admin"
				adminCount++
			}
			if u.Role != newRole {
				if err := model.DB(ctx).Model(u).Update("role", newRole).Error; err != nil {
					slog.Warn("gwSyncAdminRoles: update role failed", "sub", sub, "err", err)
				} else {
					slog.Info("gwSyncAdminRoles: role updated", "sub", sub, "username", u.Username, "old_role", u.Role, "new_role", newRole)
				}
			}
		}
	}
	slog.Info("gwSyncAdminRoles: completed", "total_users", len(users), "admins", adminCount)
}

// isOneIDUserDisabled 判断 OneID 用户状态是否应在 ClawPro 中禁用。
//
// OneID 文档枚举：Active(启用)、Suspended(管理员停用)、LockedOut(平台封禁)
// 实际 API 观察到的值还包括：Disabled（已停用/已删除）
//
// 映射规则（已停用/删除 → 禁用，其他 → 正常）：
//   - Suspended（已停用）  → ClawPro 禁用
//   - Disabled（已停用）   → ClawPro 禁用
//   - LockedOut（平台封禁）→ ClawPro 禁用
//   - Active（活跃）       → ClawPro 正常
//   - 其他值（不活跃/未激活/空字符串等） → ClawPro 正常
func isOneIDUserDisabled(status string) bool {
	switch status {
	case "Suspended", "Disabled", "LockedOut":
		return true
	default:
		return false
	}
}

// ensureLocalUserResult 是 ensureLocalUser 的返回值。
type ensureLocalUserResult struct {
	Created  bool   // 新创建了本地用户
	Disabled bool   // 因 OneID 停用而被软删除（禁用）的用户
	UserID   uint   // 被处理的用户 ID（禁用时有值）
	Username string // 被处理的用户名（禁用时有值）
}

// ensureLocalUser 检查本地 users 表是否已有该 OneID 用户，没有则自动创建。
// 同时根据 OneID 的 Status 字段决定是否禁用/恢复本地用户：
//   - OneID Suspended/Disabled/LockedOut（已停用/封禁） → 本地用户软删除（禁用），不硬删除（用户还在 OneID，可能恢复）
//   - OneID Active / 其他状态（不活跃/未激活等） → 本地用户保持正常
func ensureLocalUser(ctx context.Context, u gwUserInfo, enterpriseID string) ensureLocalUserResult {
	shouldDisable := isOneIDUserDisabled(u.Status)

	// 查看是否已存在（包括软删除的）
	var existing model.User
	if model.DB(ctx).Unscoped().Where("one_id_sub = ?", u.UnionID).First(&existing).Error == nil {
		// 已存在，更新用户名等信息
		updates := map[string]interface{}{}

		// ── 根据 OneID 状态决定禁用/恢复 ──
		if shouldDisable {
			// OneID 中已停用(Suspended)/封禁(LockedOut) → 本地软删除（禁用）
			// 不硬删除：用户还在 OneID 侧，只是暂时不可用，可能恢复
			newlyDisabled := !existing.DeletedAt.Valid
			if newlyDisabled {
				updates["deleted_at"] = time.Now()
				slog.Info("OneID gateway sync: user disabled (OneID status Suspended/Disabled/LockedOut)",
					"sub", u.UnionID, "username", existing.Username, "oneid_status", u.Status)
			} else {
				slog.Info("OneID gateway sync: user already disabled, still in disabled state",
					"sub", u.UnionID, "username", existing.Username, "oneid_status", u.Status)
			}
			if len(updates) > 0 {
				model.DB(ctx).Unscoped().Model(&existing).Updates(updates)
			}
			// Disabled=true: 无论是本次新禁用还是之前已禁用，都返回 true，
			// 确保手动同步接口能展示所有处于禁用状态的用户。
			return ensureLocalUserResult{Disabled: true, UserID: existing.ID, Username: existing.Username}
		}

		// OneID 非禁用状态（Active/不活跃/未激活等） → 同步用户名、恢复等
		// 用 OneID 的姓名（name）同步到本地用户名。
		// 但如果当前用户名已经是目标名的变体（创建时因冲突追加了后缀，如 "张三_8779"），
		// 就不再尝试改回原名，避免用户名在每次全量同步时来回变动。
		if u.Name != "" && u.Name != u.UnionID && existing.Username != u.Name {
			if isUsernameVariant(existing.Username, u.Name) {
				// 当前用户名已经是目标名的变体，不再修改
			} else {
				var nameConflict int64
				model.DB(ctx).Unscoped().Model(&model.User{}).Where("username = ? AND id != ?", u.Name, existing.ID).Count(&nameConflict)
				if nameConflict == 0 {
					updates["username"] = u.Name
				} else {
					slog.Warn("ensureLocalUser: skip username sync due to conflict",
						"sub", u.UnionID, "current", existing.Username, "desired", u.Name)
				}
			}
		}
		// 同步 OneID 登录名
		if u.Username != "" {
			updates["oneid_login_name"] = u.Username
		}
		// 如果用户之前被禁用（软删除），现在 OneID 不是禁用状态，恢复之
		if existing.DeletedAt.Valid {
			updates["deleted_at"] = nil
			slog.Info("OneID gateway sync: user re-enabled (was soft-deleted, now not Suspended/LockedOut in OneID)",
				"sub", u.UnionID, "username", existing.Username, "oneid_status", u.Status)
		}
		if len(updates) > 0 {
			model.DB(ctx).Unscoped().Model(&existing).Updates(updates)
		}
		return ensureLocalUserResult{}
	}

	// 本地不存在 — 如果 OneID 中是禁用状态（Suspended/LockedOut），不创建
	if shouldDisable {
		slog.Info("OneID gateway sync: skip creating user (OneID status Suspended/LockedOut)",
			"sub", u.UnionID, "name", u.Name, "oneid_status", u.Status)
		return ensureLocalUserResult{}
	}

	// 创建新用户
	sub := u.UnionID
	// 用户名使用 OneID 的 name（姓名）；如果含非 ASCII（如中文），用 union_id 代替
	username := uniqueUsername(ctx, u.Name, sub, 0)
	// OneID 登录名存到 oneid_login_name 字段
	var loginName *string
	if u.Username != "" {
		loginName = &u.Username
	}
	cfg := model.GetSiteConfig(ctx)

	newUser := model.User{
		Username:        username,
		Email:           u.Email,
		Role:            "user",
		InstanceQuota:   cfg.DefaultInstanceQuota,
		TokenQuotaDay:   cfg.DefaultTokenQuotaDay,
		TokenQuotaRules: cfg.DefaultTokenQuotaRules, // 原样复制，不转换
		OneIDSub:        &sub,
		OneIDLoginName:  loginName,
	}
	// Unscoped: 如果同 sub 的用户曾被硬删除后又在 OneID 侧恢复，
	// 可能残留一条软删除记录（被其他流程创建的）。用 Unscoped 确保能命中它，
	// 避免因唯一约束冲突导致创建失败。
	result := model.DB(ctx).Unscoped().Where("one_id_sub = ?", sub).FirstOrCreate(&newUser)
	if err := result.Error; err != nil {
		slog.Warn("OneID gateway sync: create user failed", "sub", u.UnionID, "name", u.Name, "err", err)
		return ensureLocalUserResult{}
	}
	slog.Info("OneID gateway sync: user created", "sub", u.UnionID, "username", username)

	// P0 Fix: Try to assign user to main department group and apply group quotas
	if len(u.Departments) > 0 {
		for _, d := range u.Departments {
			if d.IsMainDepartment && d.DepartmentID != "" {
				// Find group corresponding to this OneID department
				if deptGroup, err := model.GroupBySourceRef(ctx, model.GroupSourceOneIDDept, d.DepartmentID); err == nil && deptGroup != nil {
					// Assign user to group
					if err := model.UpdateUserGroupMemberships(model.DB(ctx), newUser.ID, []uint{deptGroup.ID}); err != nil {
						slog.Warn("OneID gateway sync: failed to assign user to department group", "sub", u.UnionID, "user_id", newUser.ID, "group_id", deptGroup.ID, "err", err)
					} else {
						slog.Info("OneID gateway sync: assigned user to department group", "sub", u.UnionID, "user_id", newUser.ID, "group_id", deptGroup.ID, "group_name", deptGroup.Name)
					}
				}
				break
			}
		}
	}

	return ensureLocalUserResult{Created: true}
}

// ── Gateway 通讯录 API 调用 ──────────────────────────────────────────────────

type gwContactsRequest struct {
	Action       string   `json:"action"`
	AccountID    string   `json:"account_id"`
	DepartmentID string   `json:"department_id,omitempty"`
	PageSize     int      `json:"page_size,omitempty"`
	PageToken    string   `json:"page_token,omitempty"`
	UnionIDs     []string `json:"union_ids,omitempty"`
	Query        string   `json:"query,omitempty"`
	RoleID       string   `json:"role_id,omitempty"`
	DSVersion    string   `json:"ds_version,omitempty"` // app_data_scope: 通讯录版本
}

type gwDeptInfo struct {
	DepartmentID       string `json:"department_id"`
	DepartmentName     string `json:"department_name"`
	DepartmentParentID string `json:"department_parent_id"`
	HasChild           bool   `json:"has_child"`
	DirectUserCount    int    `json:"direct_user_count"`
	RecurveUserCount   int    `json:"recurve_user_count"`
}

type gwUserInfo struct {
	UnionID        string                  `json:"union_id"`
	Name           string                  `json:"name"`
	Username       string                  `json:"username"` // OneID 登录名
	Email          string                  `json:"email"`
	Mobile         string                  `json:"mobile"`
	Position       string                  `json:"position"`
	EmployeeNumber string                  `json:"employee_number"`
	Status         string                  `json:"status"`
	Departments    []model.OneIDDepartment `json:"departments"`
}

// gwCallContacts 调用 Gateway 的 /api/oneid-contacts 代理端点。
// 携带 X-Internal-Token 认证头（HMAC-SHA256 签名）。
func gwCallContacts(ctx context.Context, req gwContactsRequest) (json.RawMessage, error) {
	apiURL := strings.TrimRight(GatewayURL, "/") + "/api/oneid-contacts"
	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDGwBuildRequestFailed)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 添加内部认证头（Hatchery 持有 per-tenant 派生密钥，需带上 tenant_id 让 Gateway 派生相同密钥验签）
	if hcommon.InternalSecretFromCtx(ctx) != "" {
		httpReq.Header.Set("X-Internal-Token", signInternalRequest(hcommon.InternalSecretFromCtx(ctx)))
		if hcommon.TenantIDFromCtx(ctx) != "" {
			httpReq.Header.Set("X-Internal-Tenant", hcommon.TenantIDFromCtx(ctx))
		}
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDGatewayRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, hcommon.I18nError(i18n.MsgOneIDGwReturnedError, resp.StatusCode, string(body))
	}
	return body, nil
}

// gwAppDataScopeResult 应用通讯录权限查询结果。
type gwAppDataScopeResult struct {
	DepartmentIDs []string // 有权限的部门 ID
	UserIDs       []string // 有权限的用户 union_id
}

// gwGetAppDataScope 通过 Gateway 分页获取应用通讯录权限范围。
// 返回应用有权限查看的 department_ids 和 user_ids（union_id）。
func gwGetAppDataScope(ctx context.Context, accountID string) (*gwAppDataScopeResult, error) {
	result := &gwAppDataScopeResult{}
	dsVersion := ""
	pageToken := ""

	for {
		req := gwContactsRequest{
			Action:    "app_data_scope",
			AccountID: accountID,
			PageSize:  2000,
			PageToken: pageToken,
			DSVersion: dsVersion,
		}
		body, err := gwCallContacts(ctx, req)
		if err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgOneIDGwDataScopeFailed)
		}

		var resp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				DepartmentIDs []string `json:"department_ids"`
				DSVersion     string   `json:"ds_version"`
				HasMore       bool     `json:"has_more"`
				PageToken     string   `json:"page_token"`
				UserIDs       []string `json:"user_ids"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgOneIDGwParseDataScopeFailed)
		}
		if resp.Code != 0 {
			return nil, hcommon.I18nError(i18n.MsgOneIDGwDataScopeCodeError, resp.Code, resp.Msg)
		}

		result.DepartmentIDs = append(result.DepartmentIDs, resp.Data.DepartmentIDs...)
		result.UserIDs = append(result.UserIDs, resp.Data.UserIDs...)

		// 记住 ds_version，后续分页必须带上
		if resp.Data.DSVersion != "" {
			dsVersion = resp.Data.DSVersion
		}

		if !resp.Data.HasMore || resp.Data.PageToken == "" {
			break
		}
		pageToken = resp.Data.PageToken
	}

	// 过滤空的 department_id（OneID 某些情况下会返回空字符串）
	var filtered []string
	for _, id := range result.DepartmentIDs {
		if id != "" {
			filtered = append(filtered, id)
		}
	}
	if len(filtered) != len(result.DepartmentIDs) {
		slog.Warn("gwGetAppDataScope: filtered empty department_ids",
			"before", len(result.DepartmentIDs), "after", len(filtered))
	}
	result.DepartmentIDs = filtered

	slog.Info("gwGetAppDataScope: done",
		"department_count", len(result.DepartmentIDs),
		"department_ids", result.DepartmentIDs,
		"user_count", len(result.UserIDs),
		"user_ids", result.UserIDs)
	return result, nil
}

// signInternalRequest 生成 X-Internal-Token 值。
// 格式: timestamp.hex_hmac (HMAC-SHA256(timestamp, secret))
func signInternalRequest(secret string) string {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmacSHA256([]byte(secret), []byte(ts))
	return ts + "." + hex.EncodeToString(mac)
}

// hmacSHA256 计算 HMAC-SHA256。
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// gwListSubDepartments 从指定部门开始，BFS 递归获取其所有子部门。
// 不包含起始部门本身（调用方已单独处理）。
func gwListSubDepartments(ctx context.Context, accountID, parentDeptID string) []gwDeptInfo {
	var allDepts []gwDeptInfo
	visited := make(map[string]bool) // 防止循环引用导致无限 BFS
	queue := []string{parentDeptID}
	visited[parentDeptID] = true

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		pageToken := ""
		for {
			body, err := gwCallContacts(ctx, gwContactsRequest{
				Action:       "list_direct_children",
				AccountID:    accountID,
				DepartmentID: currentID,
				PageSize:     100,
				PageToken:    pageToken,
			})
			if err != nil {
				slog.Warn("gwListSubDepartments: list direct children failed", "parent_id", currentID, "err", err)
				break
			}

			// 记录 raw response 用于诊断子部门丢失问题
			slog.Debug("gwListSubDepartments: raw response", "parent_id", currentID, "body", string(body))

			var result struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					HasMore     bool   `json:"has_more"`
					PageToken   string `json:"page_token"`
					Departments []struct {
						DepartmentID   string `json:"department_id"`
						DepartmentName string `json:"department_name"`
						HasChild       bool   `json:"has_child"`
					} `json:"departments"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 {
				slog.Warn("gwListSubDepartments: parse direct children response failed",
					"parent_id", currentID, "code", result.Code, "msg", result.Msg, "err", err,
					"raw_body", string(body))
				break
			}

			slog.Info("gwListSubDepartments: list_direct_children result",
				"parent_id", currentID,
				"children_count", len(result.Data.Departments),
				"has_more", result.Data.HasMore)

			for _, dept := range result.Data.Departments {
				if dept.DepartmentID == "" {
					slog.Warn("gwListSubDepartments: skipping child with empty department_id",
						"parent_id", currentID, "name", dept.DepartmentName)
					continue
				}
				deptInfo := gwDeptInfo{
					DepartmentID:       dept.DepartmentID,
					DepartmentName:     dept.DepartmentName,
					DepartmentParentID: currentID,
					HasChild:           dept.HasChild,
				}
				allDepts = append(allDepts, deptInfo)
				// 无论 has_child 是否为 true 都入队查询，因为 API 返回的 has_child 可能不准确
				if !visited[dept.DepartmentID] {
					visited[dept.DepartmentID] = true
					queue = append(queue, dept.DepartmentID)
				}
			}

			if !result.Data.HasMore || result.Data.PageToken == "" {
				break
			}
			pageToken = result.Data.PageToken
		}
	}

	var childNames []string
	for _, d := range allDepts {
		childNames = append(childNames, d.DepartmentName+"("+d.DepartmentID+")")
	}
	slog.Info("gwListSubDepartments: done",
		"parent_dept_id", parentDeptID,
		"child_count", len(allDepts),
		"children", childNames)

	return allDepts
}

// gwListAllDepartments 递归获取企业下所有部门。
// 先调 list_roots 获取根部门，再用 list_direct_children（游标分页）BFS 遍历子部门。
func gwListAllDepartments(ctx context.Context, accountID string) []gwDeptInfo {
	var allDepts []gwDeptInfo

	// 1. 获取根部门
	body, err := gwCallContacts(ctx, gwContactsRequest{
		Action:    "list_roots",
		AccountID: accountID,
	})
	if err != nil {
		slog.Warn("OneID gateway sync: list roots failed", "err", err)
		return nil
	}

	var rootResult struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Roots []struct {
				DepartmentID   string `json:"department_id"`
				DepartmentName string `json:"department_name"`
				HasChild       bool   `json:"has_child"`
			} `json:"roots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &rootResult); err != nil || rootResult.Code != 0 {
		slog.Warn("OneID gateway sync: parse roots response failed",
			"code", rootResult.Code, "msg", rootResult.Msg, "err", err)
		return nil
	}

	// 将根部门加入列表，并入队 BFS
	visited := make(map[string]bool) // 防止循环引用
	queue := []string{}
	for _, root := range rootResult.Data.Roots {
		if root.DepartmentID == "" {
			slog.Warn("OneID gateway sync: skipping root with empty department_id", "name", root.DepartmentName)
			continue
		}
		allDepts = append(allDepts, gwDeptInfo{
			DepartmentID:   root.DepartmentID,
			DepartmentName: root.DepartmentName,
			HasChild:       root.HasChild,
		})
		visited[root.DepartmentID] = true
		if root.HasChild {
			queue = append(queue, root.DepartmentID)
		}
	}

	// 2. BFS 遍历子部门
	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]

		pageToken := ""
		for {
			body, err := gwCallContacts(ctx, gwContactsRequest{
				Action:       "list_direct_children",
				AccountID:    accountID,
				DepartmentID: parentID,
				PageSize:     100,
				PageToken:    pageToken,
			})
			if err != nil {
				slog.Warn("OneID gateway sync: list direct children failed", "parent_id", parentID, "err", err)
				break
			}

			var result struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					HasMore     bool   `json:"has_more"`
					PageToken   string `json:"page_token"`
					Departments []struct {
						DepartmentID   string `json:"department_id"`
						DepartmentName string `json:"department_name"`
						HasChild       bool   `json:"has_child"`
					} `json:"departments"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 {
				slog.Warn("OneID gateway sync: parse direct children response failed",
					"parent_id", parentID, "code", result.Code, "msg", result.Msg, "err", err)
				break
			}

			for _, dept := range result.Data.Departments {
				if dept.DepartmentID == "" {
					slog.Warn("OneID gateway sync: skipping child with empty department_id",
						"parent_id", parentID, "name", dept.DepartmentName)
					continue
				}
				deptInfo := gwDeptInfo{
					DepartmentID:       dept.DepartmentID,
					DepartmentName:     dept.DepartmentName,
					DepartmentParentID: parentID,
					HasChild:           dept.HasChild,
				}
				allDepts = append(allDepts, deptInfo)
				// 无论 has_child 是否为 true 都入队查询
				if !visited[dept.DepartmentID] {
					visited[dept.DepartmentID] = true
					queue = append(queue, dept.DepartmentID)
				}
			}

			if !result.Data.HasMore || result.Data.PageToken == "" {
				break
			}
			pageToken = result.Data.PageToken
		}
	}

	var deptNames []string
	for _, d := range allDepts {
		deptNames = append(deptNames, d.DepartmentName+"("+d.DepartmentID+")")
	}
	slog.Info("gwListAllDepartments: done",
		"total_count", len(allDepts),
		"departments", deptNames)

	return allDepts
}

// gwListDepartmentUsers 分页拉取指定部门下的直属用户。
// 使用游标分页（page_token），对应 OneID API: GET /openapi/v3/contacts/departments/:id/direct_users
// 注意：direct_users 返回的是简要信息（union_id, name, status 等），没有 email/mobile/departments。
// 需要后续用 batch_query_users 补全详情。
// 返回值 ok=false 表示至少有一页请求失败，结果不完整，调用方应跳过软删除等破坏性操作。
func gwListDepartmentUsers(ctx context.Context, accountID, deptID string) ([]gwUserInfo, bool) {
	var allUsers []gwUserInfo
	pageToken := ""
	ok := true

	for {
		body, err := gwCallContacts(ctx, gwContactsRequest{
			Action:       "list_direct_users",
			AccountID:    accountID,
			DepartmentID: deptID,
			PageSize:     2000,
			PageToken:    pageToken,
		})
		if err != nil {
			slog.Warn("OneID gateway sync: list direct users failed", "dept_id", deptID, "err", err)
			ok = false
			break
		}

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				HasMore   bool   `json:"has_more"`
				PageToken string `json:"page_token"`
				Users     []struct {
					UnionID string `json:"union_id"`
					Name    string `json:"name"`
					Status  string `json:"status"`
					AliasID string `json:"alias_id"`
				} `json:"users"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 {
			slog.Warn("OneID gateway sync: parse direct users response failed",
				"dept_id", deptID, "code", result.Code, "msg", result.Msg, "err", err)
			break
		}

		// 收集 union_ids，用 batch_query_users 拉取完整用户信息
		var unionIDs []string
		var directUserNames []string
		for _, u := range result.Data.Users {
			unionIDs = append(unionIDs, u.UnionID)
			directUserNames = append(directUserNames, u.Name+"("+u.UnionID+")")
		}
		slog.Info("gwListDepartmentUsers: direct_users fetched",
			"dept_id", deptID,
			"count", len(result.Data.Users),
			"has_more", result.Data.HasMore,
			"users", directUserNames)

		// 按 100 个一批调 batch_query_users
		for i := 0; i < len(unionIDs); i += 100 {
			end := i + 100
			if end > len(unionIDs) {
				end = len(unionIDs)
			}
			batch := unionIDs[i:end]

			batchBody, err := gwCallContacts(ctx, gwContactsRequest{
				Action:    "batch_query_users",
				AccountID: accountID,
				UnionIDs:  batch,
			})
			if err != nil {
				slog.Warn("OneID gateway sync: batch query users failed", "dept_id", deptID, "err", err)
				continue
			}

			var batchResult struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					Users []gwUserInfo `json:"users"`
				} `json:"data"`
			}
			if err := json.Unmarshal(batchBody, &batchResult); err != nil || batchResult.Code != 0 {
				slog.Warn("OneID gateway sync: parse batch query response failed",
					"dept_id", deptID, "code", batchResult.Code, "msg", batchResult.Msg, "err", err)
				continue
			}

			var batchUserNames []string
			for _, u := range batchResult.Data.Users {
				batchUserNames = append(batchUserNames, u.Name+"("+u.UnionID+")")
			}
			slog.Info("gwListDepartmentUsers: batch_query_users result",
				"dept_id", deptID,
				"requested", len(batch),
				"returned", len(batchResult.Data.Users),
				"users", batchUserNames)

			allUsers = append(allUsers, batchResult.Data.Users...)
		}

		if !result.Data.HasMore || result.Data.PageToken == "" {
			break
		}
		pageToken = result.Data.PageToken

		// 安全上限，防止无限循环
		if len(allUsers) > 100000 {
			slog.Warn("OneID gateway sync: too many users, stopping", "dept_id", deptID)
			break
		}
	}

	return allUsers, ok
}

// ── 手动同步端点 ──────────────────────────────────────────────────────────────

// syncStatus 跟踪同步状态，防止并发执行。
var syncStatus struct {
	mu        sync.Mutex
	running   bool
	lastSync  time.Time
	lastCount int
	lastErr   string
}

// HandleSyncOneIDUsers 手动触发全量同步 OneID 用户。
// POST /admin/oneid-sync-users
// 返回同步结果摘要。
//
// 🆕 v6.12 P1：Body 支持 `{sync_dept?: bool}`：
//   - true  → 强制本次同步进入部门落地流程（即使本地尚无 oneid_dept 组）
//   - false / 未传 → 只要本地已有 oneid_dept 组，就继续维护；否则跳过部门落地
func HandleSyncOneIDUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	// 解析可选 body
	var req struct {
		SyncDept bool `json:"sync_dept"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	shouldLandDepartment := req.SyncDept || userGroupsHasOneIDDept(r.Context())

	// 防止并发同步
	syncStatus.mu.Lock()
	if syncStatus.running {
		syncStatus.mu.Unlock()
		jsonOK(w, map[string]interface{}{
			"ok":      false,
			"message": i18n.T(r.Context(), i18n.MsgOneIDSyncInProgress),
		})
		return
	}
	syncStatus.running = true
	syncStatus.mu.Unlock()

	// 同步执行，完成后再返回结果
	defer func() {
		syncStatus.mu.Lock()
		syncStatus.running = false
		syncStatus.mu.Unlock()
	}()

	sr := syncViaGatewayWithOpts(r.Context(), shouldLandDepartment)

	syncStatus.mu.Lock()
	syncStatus.lastSync = time.Now()
	syncStatus.mu.Unlock()

	slog.Info("OneID manual sync completed", "sync_dept", shouldLandDepartment)

	// 统计同步结果
	var profileCount int64
	model.DB(r.Context()).Model(&model.OneIDUserProfile{}).Count(&profileCount)
	var deptCount int64
	model.DB(r.Context()).Model(&model.OneIDDepartmentRecord{}).Count(&deptCount)
	var userCount int64
	model.DB(r.Context()).Model(&model.User{}).Where("one_id_sub IS NOT NULL AND one_id_sub != ''").Count(&userCount)

	// 🆕 v6.12 P1：dept_group_count（本次未落地时固定为 0）
	var deptGroupCount int64
	if sr.DeptLanded {
		model.DB(r.Context()).Model(&model.UserGroup{}).
			Where("source = ? AND to_be_deleted = 0", model.GroupSourceOneIDDept).
			Count(&deptGroupCount)
	}

	resp := map[string]interface{}{
		"ok":                      true,
		"message":                 i18n.T(r.Context(), i18n.MsgOneIDSyncCompleted),
		"profile_count":           profileCount,
		"dept_count":              deptCount,
		"user_count":              userCount,
		"affected_users":          sr.AffectedUsers,
		"dept_group_count":        deptGroupCount,
		"affected_dept_groups":    nilIfEmptyDeptGroups(sr.AffectedDeptGroups),
		"change_parent_group_ids": nilIfEmptyUints(sr.ChangedParentGroupIDs),
		"move_group_user_ids":     nilIfEmptyMovedUsers(sr.MoveGroupUserIDs),
	}
	// 🐛 v6 hotfix：暴露 landing 过程中未落地的部门明细，避免类似"同名子部门被
	// 残留 idx_ug_identifier_name 唯一索引静默吞掉"的问题再次掩藏。
	if len(sr.LandingFailures) > 0 {
		resp["landing_failures"] = sr.LandingFailures
	}
	jsonOK(w, resp)
}

// nilIfEmpty 用于 JSON 响应：空切片转为 nil（序列化为 null），保持与现网语义一致。
func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// nilIfEmptyDeptGroups 同 nilIfEmpty 但针对 AffectedDeptGroup 切片。
func nilIfEmptyDeptGroups(s []usergroup.AffectedDeptGroup) []usergroup.AffectedDeptGroup {
	if len(s) == 0 {
		return nil
	}
	return s
}

// nilIfEmptyUints 同 nilIfEmpty 但针对 []uint。
func nilIfEmptyUints(s []uint) []uint {
	if len(s) == 0 {
		return nil
	}
	return s
}

// nilIfEmptyMovedUsers 同 nilIfEmpty 但针对 MovedGroupUser 切片。
func nilIfEmptyMovedUsers(s []usergroup.MovedGroupUser) []usergroup.MovedGroupUser {
	if len(s) == 0 {
		return nil
	}
	return s
}

// userGroupsHasOneIDDept 返回本地 user_groups 是否已有 oneid_dept 组（未软删）。
// 用于 OneID 定时同步决定是否走部门落地流程。
func userGroupsHasOneIDDept(ctx context.Context) bool {
	var cnt int64
	err := model.DB(ctx).Model(&model.UserGroup{}).
		Where("source = ?", model.GroupSourceOneIDDept).
		Count(&cnt).Error
	if err != nil {
		return false
	}
	return cnt > 0
}

// syncViaGatewayWithOpts 是 syncViaGateway 的可配置包装：
//   - shouldLandDepartment=true：主同步流程跑完后，执行 Step 1.8 部门组落地 + Step 2.6 成员落地
//   - shouldLandDepartment=false：保持旧行为（仅同步用户），不维护 oneid_dept 组/成员
//
// 注意：部门落地失败不会中断主同步，仅记 WARN 并把 DeptLanded=false 回写到结果里，
// 这样前端可以通过 dept_group_count 推断是否真正落地成功。
func syncViaGatewayWithOpts(ctx context.Context, shouldLandDepartment bool) *syncResult {
	sr := SyncViaGateway(ctx)
	sr.DeptLanded = shouldLandDepartment

	if !shouldLandDepartment {
		return sr
	}

	// Step 1.8：部门组落地
	landRes, err := usergroup.LandOneIDDepartmentsToGroups(ctx)
	if err != nil {
		slog.Warn("[OneIDSync] LandOneIDDepartmentsToGroups 失败，跳过成员落地", "err", err)
		sr.DeptLanded = false
		return sr
	}
	if landRes != nil {
		sr.AffectedDeptGroups = landRes.NewlyMarkedToBeDeleted
		sr.ChangedParentGroupIDs = landRes.ChangedParentGroupIDs
		sr.LandingFailures = landRes.LandingFailures
	}

	// Step 2.6：成员落地
	memRes, err := usergroup.SyncOneIDMemberships(ctx)
	if err != nil {
		slog.Warn("[OneIDSync] SyncOneIDMemberships 失败", "err", err)
		// 部门已落地但成员失败 —— 仍标记已落地，下次再试补救成员
	}
	if memRes != nil {
		sr.MoveGroupUserIDs = memRes.MovedUsers
	}

	// Step 2.7：对本次换了父级的分组，异步写 parent_change_pending 记录 + 发通知
	// （与 admin_user_groups.go 换父路径一致，复用 markStaleForSubtree）
	// 不打 stale_group 标记：场景 D 的 group_id 仍正确，配置由 Resolver 自动解析。
	// 通知按 (user_id, group_id) 聚合，每个用户在每个分组下的 agent 各发一条。
	if len(sr.ChangedParentGroupIDs) > 0 {
		go func() {
			detachedCtx := hcommon.DetachContext(ctx)
			for _, gid := range sr.ChangedParentGroupIDs {
				markStaleForSubtree(detachedCtx, gid)
			}
		}()
	}

	return sr
}

// HandleSyncOneIDUsersStatus 查询同步状态。
// GET /admin/oneid-sync-users/status
func HandleSyncOneIDUsersStatus(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	syncStatus.mu.Lock()
	defer syncStatus.mu.Unlock()

	lastSync := ""
	if !syncStatus.lastSync.IsZero() {
		lastSync = syncStatus.lastSync.Format(time.RFC3339)
	}

	// 统计已同步的 OneID 用户画像数
	var profileCount int64
	model.DB(r.Context()).Model(&model.OneIDUserProfile{}).Count(&profileCount)

	var deptCount int64
	model.DB(r.Context()).Model(&model.OneIDDepartmentRecord{}).Count(&deptCount)

	var userCount int64
	model.DB(r.Context()).Model(&model.User{}).Where("one_id_sub IS NOT NULL AND one_id_sub != ''").Count(&userCount)

	// 🆕 v6.12 P1：本地 oneid_dept 组数量
	var oneidDeptGroupCount int64
	model.DB(r.Context()).Model(&model.UserGroup{}).
		Where("source = ? AND to_be_deleted = 0", model.GroupSourceOneIDDept).
		Count(&oneidDeptGroupCount)

	jsonOK(w, map[string]interface{}{
		"running":                syncStatus.running,
		"last_sync":              lastSync,
		"profile_count":          profileCount,
		"dept_count":             deptCount,
		"oneid_user_count":       userCount,
		"oneid_dept_group_count": oneidDeptGroupCount,
	})
}

// ── 企业信息同步 ──────────────────────────────────────────────────────────────

type oneIDAccountResponse struct {
	Code int `json:"code"`
	Data struct {
		Account struct {
			AccountID string `json:"account_id"`
			Name      string `json:"name"`
			LogoURL   string `json:"logo_url"`
			Domain    string `json:"domain"`
		} `json:"account"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// HandleSyncEnterprise 通过 Gateway 代理从 OneID 拉取企业信息（名称和 logo），更新到 SiteConfig 独立字段。
// POST /admin/oneid-sync-enterprise
//
// Pod 无法直连 OneID OpenAPI，所以通过 Gateway 的 GET /api/enterprise?tenant=xxx 接口代理获取。
func HandleSyncEnterprise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	if GatewayURL == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDSyncGatewayNotConfigured))
		return
	}

	// 获取企业 ID（启动时通过 ONEID_ACCOUNT_ID 环境变量注入）
	enterpriseID := hcommon.TenantIDFromCtx(r.Context())
	if enterpriseID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDTenantNotConfigured))
		return
	}

	// 通过 Gateway 代理查询企业信息（携带内部认证头）
	apiURL := fmt.Sprintf("%s/api/enterprise?tenant=%s", strings.TrimRight(GatewayURL, "/"), enterpriseID)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncCreateRequestFailed, err))
		return
	}
	if hcommon.InternalSecretFromCtx(r.Context()) != "" {
		req.Header.Set("X-Internal-Token", signInternalRequest(hcommon.InternalSecretFromCtx(r.Context())))
		if hcommon.TenantIDFromCtx(r.Context()) != "" {
			req.Header.Set("X-Internal-Tenant", hcommon.TenantIDFromCtx(r.Context()))
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("SyncEnterprise: Gateway request failed", "url", apiURL, "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncRequestGatewayFailed, err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("SyncEnterprise: Gateway API error", "status", resp.StatusCode, "body", string(body))
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncGatewayReturnedError, resp.StatusCode, string(body)))
		return
	}

	var result oneIDAccountResponse
	if err := json.Unmarshal(body, &result); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncParseResponseFailed, err))
		return
	}
	if result.Code != 0 {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncOneIDReturnedError, result.Code, result.Msg))
		return
	}

	// 构建需要更新的字段：覆写 name（网站名称）+ OneID domain
	updates := map[string]interface{}{}
	if result.Data.Account.Name != "" {
		updates["name"] = result.Data.Account.Name
	}
	if result.Data.Account.Domain != "" {
		updates["one_id_domain"] = result.Data.Account.Domain
	}

	// 下载企业 Logo 并覆写 logo / logo_mime（供 /api/logo 接口直接返回）
	if result.Data.Account.LogoURL != "" {
		logoResp, err := (&http.Client{Timeout: 15 * time.Second}).Get(result.Data.Account.LogoURL)
		if err != nil {
			slog.Warn("SyncEnterprise: download logo failed, skip logo update", "err", err)
		} else {
			defer logoResp.Body.Close()
			logoData, err := io.ReadAll(logoResp.Body)
			if err != nil {
				slog.Warn("SyncEnterprise: read logo body failed, skip logo update", "err", err)
			} else {
				mime := logoResp.Header.Get("Content-Type")
				if mime == "" {
					mime = "image/png"
				}
				// 只保留 mime 主类型，去掉参数（如 ; charset=...）
				if idx := strings.Index(mime, ";"); idx > 0 {
					mime = strings.TrimSpace(mime[:idx])
				}
				updates["logo"] = logoData
				updates["logo_mime"] = mime
			}
		}
	}

	if err := model.UpdateSiteConfig(r.Context(), updates); err != nil {
		slog.Error("SyncEnterprise: update site config failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDSyncUpdateConfigFailed, err))
		return
	}

	slog.Info("SyncEnterprise: success", "name", result.Data.Account.Name, "logo", result.Data.Account.LogoURL)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// fetchOneIDDomain 通过 Gateway 拉取 OneID 企业信息中的 domain 字段。
// 获取成功后写入 site_configs.one_id_domain。
func fetchOneIDDomain(ctx context.Context) string {
	if GatewayURL == "" {
		return ""
	}
	enterpriseID := hcommon.TenantIDFromCtx(ctx)
	if enterpriseID == "" {
		return ""
	}

	apiURL := fmt.Sprintf("%s/api/enterprise?tenant=%s", strings.TrimRight(GatewayURL, "/"), enterpriseID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		slog.Warn("fetchOneIDDomain: build request failed", "err", err)
		return ""
	}
	if secret := hcommon.InternalSecretFromCtx(ctx); secret != "" {
		req.Header.Set("X-Internal-Token", signInternalRequest(secret))
		req.Header.Set("X-Internal-Tenant", enterpriseID)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		slog.Warn("fetchOneIDDomain: gateway request failed", "err", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Warn("fetchOneIDDomain: gateway returned error", "status", resp.StatusCode)
		return ""
	}

	var result oneIDAccountResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Code != 0 {
		slog.Warn("fetchOneIDDomain: parse response failed", "err", err, "code", result.Code)
		return ""
	}

	domain := result.Data.Account.Domain
	if domain == "" {
		return ""
	}

	// 写入 DB
	if err := model.UpdateSiteConfig(ctx, map[string]interface{}{"one_id_domain": domain}); err != nil {
		slog.Warn("fetchOneIDDomain: update one_id_domain failed", "err", err)
	} else {
		slog.Info("fetchOneIDDomain: one_id_domain synced from OneID", "domain", domain)
	}
	return domain
}
