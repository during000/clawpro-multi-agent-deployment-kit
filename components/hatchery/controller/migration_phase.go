package controller

// migration_phase.go — Phase 1/2/3 和 Reconcile 的具体实现。
//
// 所有 Phase 函数接收 migrateCtx（已注入 job.Config 的 snapshot），
// 复用现有 OneID 函数，零改动原有逻辑。

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"
)

// PhaseResult 单个 phase 的执行结果。
type PhaseResult struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Batches int `json:"batches,omitempty"` // Phase 3 专用
}

// ReconcileResult reconcile 的完整结果。
type ReconcileResult struct {
	OK         bool        `json:"ok"`
	Phase1     PhaseResult `json:"phase1"`
	Phase2     PhaseResult `json:"phase2"`
	Phase3     PhaseResult `json:"phase3"`
	MirrorDiff struct {
		RoleChanged int `json:"role_changed"`
		DeptChanged int `json:"dept_changed"`
	} `json:"mirror_diff"`
	Disabled PhaseResult `json:"disabled"`
}

// ── Phase 1: 分组 → OneID 部门 ───────────────────────────────────────────────

func runMigratePhase1(ctx context.Context, job *JobState) PhaseResult {
	// 只迁 manual 分组（独立模式下不应有 oneid_dept，但加条件更安全）
	var groups []model.UserGroup
	if err := model.DB(ctx).
		Where("(source_ref = '' OR source_ref IS NULL) AND source = ?", model.GroupSourceManual).
		Find(&groups).Error; err != nil {
		slog.Error("[migrate/phase1] 查询 user_groups 失败", "err", err)
		return PhaseResult{}
	}

	result := PhaseResult{Total: len(groups)}

	// 拓扑排序：父→子（ParentID=0 的排最前）
	sorted := topoSortGroups(groups)

	for _, g := range sorted {
		g := g // capture
		if _, err := oneIDEnsureGroupHasDept(ctx, &g); err != nil {
			slog.Error("[migrate/phase1] 同步分组失败",
				"group_id", g.ID, "name", g.Name, "err", err)
			job.addFailure(MigrateFailureRecord{
				Phase: 1, TargetID: g.ID, TargetName: g.Name,
				Error: migrateFailureMessage(ctx, err), At: time.Now(),
			})
			result.Failed++
		} else {
			result.Success++
		}
	}
	return result
}

// topoSortGroups 按父→子顺序排列分组（BFS）。
func topoSortGroups(groups []model.UserGroup) []model.UserGroup {
	byID := make(map[uint]*model.UserGroup, len(groups))
	for i := range groups {
		g := groups[i]
		byID[g.ID] = &g
	}
	// 将 ParentID=0 的根节点放最前，依次添加其子节点
	var sorted []model.UserGroup
	queued := make(map[uint]bool)
	var enqueue func(parentID uint)
	enqueue = func(parentID uint) {
		for _, g := range groups {
			if g.ParentID == parentID && !queued[g.ID] {
				queued[g.ID] = true
				sorted = append(sorted, g)
				enqueue(g.ID)
			}
		}
	}
	enqueue(0)
	// 把未排进去的（孤儿节点，理论上不应有）也加进来
	for _, g := range groups {
		if !queued[g.ID] {
			sorted = append(sorted, g)
		}
	}
	return sorted
}

// ── Phase 2: 用户 → OneID 用户 ───────────────────────────────────────────────

const phase2Concurrency = 10

func runMigratePhase2(ctx context.Context, job *JobState) PhaseResult {
	superAdminID := initialAdminID(ctx)
	var users []model.User
	// 未绑定用户（需创建）+ 所有 admin（需确保 admin 角色已同步）。
	// 后者覆盖“创建成功但 admin 角色绑定失败”的用户——此前这类用户因 one_id_sub
	// 已写入而被 WHERE one_id_sub IS NULL 永久漏掉，admin 角色无法重试补绑。
	// 超管已由 bindAdminUser 绑定 AdminUnionID，且其 OneID 账号已有超管权限，
	// 不应走创建/加角色流程，故在此排除。
	q := model.DB(ctx).
		Where("one_id_sub IS NULL OR one_id_sub = '' OR role = 'admin'")
	if superAdminID != 0 {
		q = q.Where("id != ?", superAdminID)
	}
	if err := q.Find(&users).Error; err != nil {
		slog.Error("[migrate/phase2] 查询 users 失败", "err", err)
		return PhaseResult{}
	}

	// 过滤已同步用户：已绑定 one_id_sub 且 mirror 显示 role 已同步 → 跳过，
	// 避免对已完成迁移的用户重复幂等调用。mirror 由 migrateOneUser 成功后更新，
	// 失败的用户不会更新 mirror，下次仍会被处理。重启后 mirror 为空，首次全量幂等处理一次。
	var pending []model.User
	for i := range users {
		u := users[i]
		if u.OneIDSub != nil && *u.OneIDSub != "" {
			job.mu.Lock()
			m, has := job.UserMirror[u.ID]
			job.mu.Unlock()
			if has && m.Role == u.Role {
				continue
			}
		}
		pending = append(pending, u)
	}
	users = pending

	result := PhaseResult{Total: len(users)}
	var mu sync.Mutex
	sem := make(chan struct{}, phase2Concurrency)
	var wg sync.WaitGroup

	for i := range users {
		u := users[i]
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := migrateOneUser(ctx, job, &u); err != nil {
				job.addFailure(MigrateFailureRecord{
					Phase: 2, TargetID: u.ID, TargetName: u.Username,
					Error: migrateFailureMessage(ctx, err), At: time.Now(),
				})
				mu.Lock()
				result.Failed++
				mu.Unlock()
			} else {
				mu.Lock()
				result.Success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return result
}

// migrateOneUser 在 OneID 创建用户并回填 one_id_sub；若为 admin 则绑定 admin 角色。
// 幂等可重入：已绑定 one_id_sub 的用户复用 union_id、跳过创建，但仍重试 admin 角色绑定，
// 覆盖“创建成功但 admin 角色绑定失败”的重试场景——此前这类用户因 one_id_sub 已写入
// 而被 phase2 的 WHERE one_id_sub IS NULL 永久漏掉。admin 角色绑定失败即返回 error，
// 不更新 mirror，下次 reconcile 仍会被处理。
func migrateOneUser(ctx context.Context, job *JobState, u *model.User) error {
	unionID := ""

	// 已绑定：复用 union_id，跳过 OneID 创建（避免重复建用户）。
	if u.OneIDSub != nil && *u.OneIDSub != "" {
		unionID = *u.OneIDSub
		slog.Info("[migrate/phase2] 用户已绑定 one_id_sub，跳过创建复用 union_id",
			"user", u.Username, "union_id", unionID)
	} else {
		// 未绑定：创建 OneID 用户
		deptIDs, err := resolveDeptIDsForUser(ctx, u.ID)
		if err != nil {
			slog.Warn("[migrate/phase2] resolve dept_ids 失败，使用根部门", "user", u.Username, "err", err)
			deptIDs = nil
		}

		// 确定 OneID 登录名
		loginName := u.Username
		if validateOneIDUsername(u.Username) != nil {
			loginName = generateRandomLoginName()
		}

		// 随机初始密码（内存生成，不持久化；Phase 3 会覆盖为真实 bcrypt hash）
		initPwd := generateMigrateInitPassword()

		resp, err := OneIDCreateUser(ctx, OneIDCreateUserReq{
			Name:          u.Username,
			Username:      loginName,
			Password:      initPwd,
			DepartmentIDs: deptIDs,
		})
		if err != nil {
			// 用户名冲突：换随机登录名重试一次
			if strings.Contains(err.Error(), "username") || strings.Contains(err.Error(), "already") {
				loginName = generateRandomLoginName()
				resp, err = OneIDCreateUser(ctx, OneIDCreateUserReq{
					Name:          u.Username,
					Username:      loginName,
					Password:      initPwd,
					DepartmentIDs: deptIDs,
				})
			}
			if err != nil {
				return fmt.Errorf("OneID 创建用户失败: %w", err)
			}
		}
		unionID = resp.UnionID

		// UpdateColumn 不 bump updated_at，避免 Phase 3 将本次写入误判为“密码更新”
		if err := model.DB(ctx).Model(&model.User{}).
			Where("id = ?", u.ID).
			UpdateColumns(map[string]interface{}{
				"one_id_sub":       unionID,
				"oneid_login_name": loginName,
			}).Error; err != nil {
			return fmt.Errorf("回填 one_id_sub 失败: %w", err)
		}
	}

	// admin 角色绑定（幂等可重入：新建或复用 union_id 均执行，确保角色存在；
	// 与正常建用户流程对齐，失败即视为迁移失败，下次 reconcile 重试）。
	if u.Role == "admin" {
		if err := OneIDAddRoleUsers(ctx, []string{unionID}); err != nil {
			return fmt.Errorf("OneID admin 角色绑定失败: %w", err)
		}
	}

	// 更新内存 mirror（仅在 admin 角色绑定成功后才标记 role 已同步）
	groupIDs := getUserGroupIDs(ctx, u.ID)
	job.mu.Lock()
	job.UserMirror[u.ID] = UserSnapshot{Role: u.Role, GroupIDs: groupIDs}
	job.mu.Unlock()

	slog.Info("[migrate/phase2] 用户迁移成功",
		"username", u.Username, "union_id", unionID)
	return nil
}

// resolveDeptIDsForUser 查出用户所属 manual 分组对应的 OneID dept_ids。
func resolveDeptIDsForUser(ctx context.Context, userID uint) ([]string, error) {
	var groups []model.UserGroup
	if err := model.DB(ctx).
		Joins("JOIN user_group_members ON user_group_members.user_group_id = user_groups.id").
		Where("user_group_members.user_id = ? AND user_groups.source = ? AND user_groups.source_ref != ''",
			userID, model.GroupSourceManual).
		Find(&groups).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var deptIDs []string
	for _, g := range groups {
		if g.SourceRef != "" && !seen[g.SourceRef] {
			seen[g.SourceRef] = true
			deptIDs = append(deptIDs, g.SourceRef)
		}
	}
	return deptIDs, nil
}

// getUserGroupIDs 查用户当前所属的所有 manual 分组 ID。
func getUserGroupIDs(ctx context.Context, userID uint) []uint {
	var members []model.UserGroupMember
	model.DB(ctx).Where("user_id = ?", userID).Find(&members)
	ids := make([]uint, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserGroupID)
	}
	return ids
}

// generateMigrateInitPassword 生成随机初始密码（仅用于创建，不持久化；Phase 3 会覆盖为真实 bcrypt hash）。
// 满足常见密码策略：8-16位，包含大写字母、小写字母、数字、特殊字符各至少 1 个。
func generateMigrateInitPassword() string {
	const lower = "abcdefghjkmnpqrstuvwxyz"
	const upper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const digits = "23456789"
	const special = "!@#$"

	pick := func(charset string) byte {
		b := make([]byte, 1)
		_, _ = rand.Read(b)
		return charset[int(b[0])%len(charset)]
	}

	// 各取 1 个保证复杂度
	pwd := []byte{pick(lower), pick(upper), pick(digits), pick(special)}

	// 再随机补到 12 位
	all := lower + upper + digits + special
	extra := make([]byte, 8)
	_, _ = rand.Read(extra)
	for _, b := range extra {
		pwd = append(pwd, all[int(b)%len(all)])
	}

	// 打乱顺序
	for i := len(pwd) - 1; i > 0; i-- {
		jb := make([]byte, 1)
		_, _ = rand.Read(jb)
		j := int(jb[0]) % (i + 1)
		pwd[i], pwd[j] = pwd[j], pwd[i]
	}
	return string(pwd)
}

// ── Phase 3: 密码批量同步 ─────────────────────────────────────────────────────

const phase3BatchSize = 100

func runMigratePhase3(ctx context.Context, job *JobState) PhaseResult {
	job.mu.Lock()
	phaseStartAt := time.Now()
	watermark := job.Phase3At
	job.mu.Unlock()

	var users []model.User
	// 排除超管：超管绑定的是 OneID 预置超管账号（AdminUnionID），不参与改密。
	superAdminID := initialAdminID(ctx)
	q := model.DB(ctx).
		Select("id, password, one_id_sub, updated_at").
		Where("one_id_sub IS NOT NULL AND one_id_sub != ''").
		Where("password != ''")
	if superAdminID != 0 {
		q = q.Where("id != ?", superAdminID)
	}
	if !watermark.IsZero() {
		q = q.Where("updated_at > ?", watermark)
	}
	if err := q.Find(&users).Error; err != nil {
		slog.Error("[migrate/phase3] 查询 users 失败", "err", err)
		return PhaseResult{}
	}

	result := PhaseResult{Total: len(users)}
	if len(users) == 0 {
		job.mu.Lock()
		job.Phase3At = phaseStartAt
		job.mu.Unlock()
		return result
	}

	// 串行批次，每批 100
	for i := 0; i < len(users); i += phase3BatchSize {
		end := i + phase3BatchSize
		if end > len(users) {
			end = len(users)
		}
		batch := users[i:end]
		result.Batches++

		items := make([]PasswordResetItem, 0, len(batch))
		for _, u := range batch {
			if u.OneIDSub == nil || *u.OneIDSub == "" || u.Password == "" {
				continue
			}
			items = append(items, PasswordResetItem{
				UnionID:  *u.OneIDSub,
				Password: wrapBcryptPassword(u.Password),
			})
		}
		if len(items) == 0 {
			continue
		}

		batchResult, err := OneIDBatchResetPassword(ctx, items)
		if err != nil {
			slog.Error("[migrate/phase3] 批量改密失败", "batch_start", i, "err", err)
			for _, u := range batch {
				job.addFailure(MigrateFailureRecord{
					Phase: 3, TargetID: u.ID, TargetName: u.Username,
					Error: migrateFailureMessage(ctx, err), At: time.Now(),
				})
				result.Failed++
			}
			continue
		}

		// 解析部分失败
		failedUnionIDs := make(map[string]string, len(batchResult.Failures))
		for _, f := range batchResult.Failures {
			failedUnionIDs[f.UnionID] = f.ErrorMsg
		}
		for _, u := range batch {
			sub := ""
			if u.OneIDSub != nil {
				sub = *u.OneIDSub
			}
			if errMsg, failed := failedUnionIDs[sub]; failed {
				job.addFailure(MigrateFailureRecord{
					Phase: 3, TargetID: u.ID, TargetName: u.Username,
					Error: fmt.Sprintf("code=%d msg=%s", 0, errMsg), At: time.Now(),
				})
				result.Failed++
			} else {
				result.Success++
			}
		}
	}

	// 用 phase 开始时刻，防止漏掉"phase 跑期间新改密"的用户
	job.mu.Lock()
	job.Phase3At = phaseStartAt
	job.mu.Unlock()

	return result
}

// wrapBcryptPassword 将本地 bcrypt hash 包装为 OneID 接受的 JSON 格式。
func wrapBcryptPassword(hash string) string {
	payload := map[string]interface{}{
		"hash": map[string]string{
			"algorithm": "bcrypt",
			"value":     hash,
		},
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

// ── Reconcile ─────────────────────────────────────────────────────────────────

func runMigrateReconcile(ctx context.Context, job *JobState) ReconcileResult {
	result := ReconcileResult{OK: true}

	// 1. Phase 1 增量
	result.Phase1 = runMigratePhase1(ctx, job)

	// 1.5 超管绑定：直接将 admin 用户的 one_id_sub 设为 AdminUnionID，跳过 Phase 2 创建
	job.mu.Lock()
	adminUnionID := job.Config.AdminUnionID
	job.mu.Unlock()
	if adminUnionID != "" {
		bindAdminUser(ctx, job, adminUnionID, job.Config.AdminLoginName)
	}

	// 2. Phase 2 增量：未绑定用户（超管已由 bindAdminUser 绑定，自动跳过）
	result.Phase2 = runMigratePhase2(ctx, job)

	// 3. Phase 3 增量（密码）
	result.Phase3 = runMigratePhase3(ctx, job)

	// 4. Mirror diff：检测 role / 分组变化
	roleChanged, deptChanged := runMirrorDiff(ctx, job)
	result.MirrorDiff.RoleChanged = roleChanged
	result.MirrorDiff.DeptChanged = deptChanged

	// 5. 软删增量
	result.Disabled = runSoftDeleteSync(ctx, job)

	return result
}

// bindAdminUser 将超管（id 最小的 admin）的 one_id_sub 直接绑定为 AdminUnionID，
// 跳过 OneID 创建流程。超管对应 OneID 套件中预置的超管账号，故只绑定不创建。
// 普通不在此处理，由 Phase 2 创建 OneID 账号后调用 OneIDAddRoleUsers 绑定 admin 角色。
func bindAdminUser(ctx context.Context, job *JobState, adminUnionID, adminLoginName string) {
	initial := model.GetInitialAdmin(ctx)
	if initial == nil {
		slog.Warn("[migrate/admin-bind] 未找到超管（id 最小的 admin），跳过超管绑定")
		return
	}
	u := *initial
	// 仅处理尚未绑定的超管，避免重复绑定覆盖已有 one_id_sub。
	var existing model.User
	if err := model.DB(ctx).Select("id, one_id_sub").
		Where("id = ?", u.ID).First(&existing).Error; err != nil {
		slog.Error("[migrate/admin-bind] 查询超管失败", "user", u.Username, "err", err)
		return
	}
	if existing.OneIDSub != nil && *existing.OneIDSub != "" {
		return
	}
	if err := model.DB(ctx).Model(&model.User{}).Where("id = ?", u.ID).
		UpdateColumns(map[string]interface{}{
			"one_id_sub":       adminUnionID,
			"oneid_login_name": &adminLoginName,
		}).Error; err != nil {
		slog.Error("[migrate/admin-bind] 绑定超管 one_id_sub 失败", "user", u.Username, "err", err)
		job.addFailure(MigrateFailureRecord{
			Phase: 2, TargetID: u.ID, TargetName: u.Username,
			Error: migrateFailureMessage(ctx, err), At: time.Now(),
		})
		return
	}
	// 更新内存 mirror
	groupIDs := getUserGroupIDs(ctx, u.ID)
	job.mu.Lock()
	job.UserMirror[u.ID] = UserSnapshot{Role: u.Role, GroupIDs: groupIDs}
	job.mu.Unlock()
	slog.Info("[migrate/admin-bind] 超管绑定成功", "user", u.Username, "union_id", adminUnionID, "login_name", adminLoginName)
}

// runMirrorDiff 遍历所有有 one_id_sub 的用户，与内存 mirror 对比，推送 role/dept 变化。
// mirror 为空时（进程重启后首次 reconcile），对所有用户全量推送一次（幂等）。
//
// mirror 可靠性：仅当 role/dept 真正同步成功时才更新对应字段，失败则保留旧 mirror 值，
// 使下次 reconcile 仍能重试；避免“失败却标记已同步”导致永久漏推。
func runMirrorDiff(ctx context.Context, job *JobState) (roleChanged, deptChanged int) {
	superAdminID := initialAdminID(ctx)
	var users []model.User
	q := model.DB(ctx).
		Where("one_id_sub IS NOT NULL AND one_id_sub != ''")
	if superAdminID != 0 {
		q = q.Where("id != ?", superAdminID)
	}
	if err := q.Find(&users).Error; err != nil {
		slog.Error("[migrate/reconcile] 查询 users 失败", "err", err)
		return
	}

	for _, u := range users {
		if u.OneIDSub == nil || *u.OneIDSub == "" {
			continue
		}
		unionID := *u.OneIDSub
		currentGroupIDs := getUserGroupIDs(ctx, u.ID)

		job.mu.Lock()
		last, hasMirror := job.UserMirror[u.ID]
		job.mu.Unlock()

		roleSynced := hasMirror && last.Role == u.Role // 是否需更新 mirror role 的依据
		deptSynced := hasMirror && groupIDsEqual(last.GroupIDs, currentGroupIDs)

		// role 变化
		if !hasMirror || last.Role != u.Role {
			if hasMirror {
				// 有 mirror：精确判断方向
				if u.Role == "admin" && last.Role != "admin" {
					if err := OneIDAddRoleUsers(ctx, []string{unionID}); err != nil {
						slog.Warn("[migrate/reconcile] add-role 失败", "user", u.Username, "err", err)
						roleSynced = false
					} else {
						roleChanged++
						roleSynced = true
					}
				} else if u.Role != "admin" && last.Role == "admin" {
					if err := OneIDRemoveRoleUsers(ctx, []string{unionID}); err != nil {
						slog.Warn("[migrate/reconcile] remove-role 失败", "user", u.Username, "err", err)
						roleSynced = false
					} else {
						roleChanged++
						roleSynced = true
					}
				}
			} else {
				// 无 mirror（重启后第一次）：全量同步 role
				if u.Role == "admin" {
					if err := OneIDAddRoleUsers(ctx, []string{unionID}); err != nil {
						slog.Warn("[migrate/reconcile] add-role 失败（全量）", "user", u.Username, "err", err)
						roleSynced = false
					} else {
						roleChanged++
						roleSynced = true
					}
				} else {
					// 非 admin 首次：无需同步角色，标记为已同步
					roleSynced = true
				}
			}
		}

		// 分组变化：全量计算当前 dept_ids 并推
		if !hasMirror || !groupIDsEqual(last.GroupIDs, currentGroupIDs) {
			deptIDs, err := resolveDeptIDsForUser(ctx, u.ID)
			if err != nil {
				slog.Warn("[migrate/reconcile] resolve dept_ids 失败", "user", u.Username, "err", err)
				deptSynced = false
			} else {
				if len(deptIDs) == 0 {
					// 确保有根部门
					token, _ := getOneIDAppToken(ctx)
					if rootID, err := getOneIDRootDepartmentID(ctx, token); err == nil && rootID != "" {
						deptIDs = []string{rootID}
					}
				}
				if err := OneIDUpdateUser(ctx, unionID, map[string]interface{}{
					"department_ids": deptIDs,
				}); err != nil {
					slog.Warn("[migrate/reconcile] update dept 失败", "user", u.Username, "err", err)
					deptSynced = false
				} else {
					deptChanged++
					deptSynced = true
				}
			}
		}

		// 更新 mirror：仅写入已同步成功的字段，失败字段保留旧值以便下次重试。
		// 无 mirror（首次）且某字段未执行同步时，Role 默认 ""、GroupIDs nil 表示“未同步”。
		job.mu.Lock()
		prev := job.UserMirror[u.ID]
		next := UserSnapshot{GroupIDs: currentGroupIDs}
		if roleSynced {
			next.Role = u.Role
		} else if hasMirror {
			next.Role = prev.Role
		}
		if !deptSynced && hasMirror {
			next.GroupIDs = prev.GroupIDs
		}
		job.UserMirror[u.ID] = next
		job.mu.Unlock()
	}
	return
}

// groupIDsEqual 判断两个 group ID 切片是否相同（不关心顺序）。
func groupIDsEqual(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[uint]bool, len(a))
	for _, id := range a {
		set[id] = true
	}
	for _, id := range b {
		if !set[id] {
			return false
		}
	}
	return true
}

// runSoftDeleteSync 同步软删用户到 OneID（调用 batch_disable）。
func runSoftDeleteSync(ctx context.Context, job *JobState) PhaseResult {
	job.mu.Lock()
	watermark := job.Phase3At
	job.mu.Unlock()

	q := model.DB(ctx).Unscoped().
		Where("deleted_at IS NOT NULL AND one_id_sub IS NOT NULL AND one_id_sub != ''")
	if !watermark.IsZero() {
		q = q.Where("deleted_at > ?", watermark)
	}

	var users []model.User
	if err := q.Find(&users).Error; err != nil {
		slog.Error("[migrate/reconcile] 查询软删 users 失败", "err", err)
		return PhaseResult{}
	}

	result := PhaseResult{Total: len(users)}
	for _, u := range users {
		if u.OneIDSub == nil || *u.OneIDSub == "" {
			continue
		}
		if err := OneIDDisableUser(ctx, []string{*u.OneIDSub}); err != nil {
			slog.Warn("[migrate/reconcile] 停用用户失败", "user", u.Username, "err", err)
			job.addFailure(MigrateFailureRecord{
				Phase: 2, TargetID: u.ID, TargetName: u.Username,
				Error: migrateFailureMessage(ctx, err), At: time.Now(),
			})
			result.Failed++
		} else {
			result.Success++
		}
	}
	return result
}

// migrateFailureMessage 把 error 转成迁移失败记录文案。
// 两层分离（符合 i18n 规范）：用户可见消息走 i18n（RichError.ErrorMessage），
// 底层技术错误（如 OneID 返回的原始错误）原样拼接，不国际化——避免 RichError.Error()
// 只输出 i18n 文案而丢失真实失败原因（如 token endpoint 的 401 body）。
func migrateFailureMessage(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var re *hcommon.RichError
	if errors.As(err, &re) {
		withCauses := re.ErrorMessageWithCauses(ctx)
		if withCauses != "" {
			return withCauses
		}
		// 无底层 cause：仅返回 i18n 文案。
		return re.ErrorMessage(ctx)
	}
	return err.Error()
}

// initialAdminID 返回当前租户的超管 id（id 最小的 admin）。
// 超管对应 OneID 预置超管账号，迁移时只做 one_id_sub 绑定（bindAdminUser），
// 不参与密码同步（phase3）和角色/部门推送（mirror diff）——避免改超管密码、
// 给已有超管权限的账号重复加角色或推送部门。
// 无超管时返回 0（不会误排任何用户，因为用户 id 从 1 开始）。
func initialAdminID(ctx context.Context) uint {
	initial := model.GetInitialAdmin(ctx)
	if initial == nil {
		return 0
	}
	return initial.ID
}
