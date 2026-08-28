package usergroup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────
// Step 1.8：部门组落地（oneid_dept）
// ──────────────────────────────────────────────

// LandDepartmentsResult 返回 Step 1.8 的执行结果。
type LandDepartmentsResult struct {
	// NewlyMarkedToBeDeleted 本次新打 to_be_deleted=1 的 oneid_dept 组明细。
	// 🔄 v6.13：字段类型由 []string（仅 full_path）升级为 []AffectedDeptGroup
	// （含 group_id + full_path），前端可直接拿 group_id 发起 /admin/user-groups/delete。
	NewlyMarkedToBeDeleted []AffectedDeptGroup

	// ChangedParentGroupIDs 本次同步中发生了父节点切换的 oneid_dept 组 ID 列表。
	// 🆕 v6.13：仅在 Step A 检测到 existing.ParentID != 解析出的 parentGroupID
	// 且 UpdateUserGroupExtForOneIDDept 成功落盘后记一条。仅改名不换父不算。
	ChangedParentGroupIDs []uint

	// LandingFailures 本次 landing 过程中**未能落地**的 OneID 部门明细。
	// 任何原因（创建失败、更新失败、父解析异常等）造成 user_groups 没产出
	// 对应 (source='oneid_dept', source_ref=DepartmentID) 行的 dept 都会被记在这里。
	// 响应/日志可据此暴露问题，避免类似 idx_ug_identifier_name 残留索引导致的
	// "静默漏掉 2 个同名子部门"这种 bug 再次被吞掉。
	LandingFailures []LandingFailure
}

// AffectedDeptGroup 受影响的 oneid_dept 组简要信息。
// 用于 /admin/oneid-sync-users 响应里标识本次被标 to_be_deleted 的分组。
type AffectedDeptGroup struct {
	GroupID  uint   `json:"group_id"`
	FullPath string `json:"full_path"`
}

// MovedGroupUser 同步过程中被移出某个 oneid_dept 组的成员条目。
// 🆕 v6.13：用户在 OneID 侧被调出某部门后，本次同步把对应 user_group_members
// 行物理删除，这里记下 (user_id, 移出前所在的 group_id) 供前端提醒/审计。
type MovedGroupUser struct {
	UserID      uint `json:"user_id"`
	FromGroupID uint `json:"from_group_id"`
}

// MembershipsResult 是 SyncOneIDMemberships 的返回值。
// 🆕 v6.13：用于把"用户被移出分组"的事件告知上层。
type MembershipsResult struct {
	MovedUsers []MovedGroupUser
}

// LandingFailure 单条 landing 失败记录。
type LandingFailure struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	Stage          string `json:"stage"` // "create" / "update" / "rename_parent" / "clear_tbd"
	Err            string `json:"err"`
}

// LandOneIDDepartmentsToGroups 执行 Step 1.8：
//
//	A. 按父→子拓扑序 Upsert oneid_departments 到 user_groups(source='oneid_dept', source_ref=DepartmentID)
//	B. 对比出"OneID 已消失但本地仍有"的 oneid_dept 组子树
//	C. 子树级处理：每个组独立 CanDelete，全通过→实删；任一失败→整子树 to_be_deleted=1
//	D. 对于上次 to_be_deleted=1 但本次 OneID 又返回的组 → 清 to_be_deleted=0
//
// 本函数不抛错中断，而是累积日志：只要主干流程能推进，单个部门失败只记 WARN 并继续。
func LandOneIDDepartmentsToGroups(ctx context.Context) (*LandDepartmentsResult, error) {
	result := &LandDepartmentsResult{}

	// 读 OneID 部门记录（已经由同步主流程拉全）
	var depts []model.OneIDDepartmentRecord
	if err := model.DB(ctx).Find(&depts).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptLandingReadDepts)
	}

	activeDeptIDSet := make(map[string]bool, len(depts))
	for _, d := range depts {
		activeDeptIDSet[d.DepartmentID] = true
	}

	// 排好拓扑序：depth 升序（父先于子）
	ordered := topologicalOrderByParent(depts)

	// ── A. Upsert 部门组（按父→子顺序） ────────────────────
	// deptID → 本地 user_group.ID（含新建 + 已存在）
	deptToGroupID := make(map[string]uint, len(ordered))
	// failedDeptIDs 收集本轮已创建失败（或因父链失败而跳过）的部门，用于子孙判定：
	// 若部门 X 的 parent_dept_id 在这里，则 X 也不能落地（避免被错误地挂到根）。
	failedDeptIDs := make(map[string]bool)
	// activeInOrdered 标记 parent_dept_id 是否真的出现在本次数据集里；
	// 用于区分两种"parent 不在 deptToGroupID"的情况：
	//   1) parent 根本没在本次数据（授权范围裁剪）→ 允许当作根节点挂
	//   2) parent 在本次数据但落地失败（如 depth 超限）→ 子孙必须跳过，不挂根
	activeInOrdered := make(map[string]bool, len(ordered))
	for _, d := range ordered {
		activeInOrdered[d.DepartmentID] = true
	}

	for _, d := range ordered {
		// 父组解析
		var parentGroupID uint
		if d.DepartmentParentID != "" {
			if gid, ok := deptToGroupID[d.DepartmentParentID]; ok {
				parentGroupID = gid
			} else if failedDeptIDs[d.DepartmentParentID] || activeInOrdered[d.DepartmentParentID] {
				// 父在本次数据里但没落地成功（depth 超限 / 其它 create 失败 /
				// 父自身也因 parent 失败被跳过）→ 当前部门必须跳过，不能静默挂根，
				// 否则组织树会错位（例如 OneID 第 12 层被挂到本地根，视觉上看就是
				// "一个叶子节点和总部并列"）。
				slog.Warn("LandOneIDDepartmentsToGroups: 父部门落地失败，连锁跳过当前部门",
					"dept_id", d.DepartmentID, "parent_dept_id", d.DepartmentParentID)
				result.LandingFailures = append(result.LandingFailures, LandingFailure{
					DepartmentID:   d.DepartmentID,
					DepartmentName: d.DepartmentName,
					Stage:          "skipped_due_to_parent",
					Err:            fmt.Sprintf("父部门 %s 落地失败，本部门一并跳过", d.DepartmentParentID),
				})
				failedDeptIDs[d.DepartmentID] = true
				continue
			} else {
				// 父不在本次数据里（部门授权范围裁剪），直接挂根
				slog.Debug("LandOneIDDepartmentsToGroups: 父部门在本次数据中缺失，挂到根",
					"dept_id", d.DepartmentID, "parent_dept_id", d.DepartmentParentID)
			}
		}

		// 查本地是否已有（按 source_ref = DepartmentID）
		existing, err := model.GroupBySourceRef(ctx, model.GroupSourceOneIDDept, d.DepartmentID)
		if err != nil {
			slog.Warn("LandOneIDDepartmentsToGroups: GroupBySourceRef 失败，跳过",
				"dept_id", d.DepartmentID, "err", err)
			result.LandingFailures = append(result.LandingFailures, LandingFailure{
				DepartmentID:   d.DepartmentID,
				DepartmentName: d.DepartmentName,
				Stage:          "lookup",
				Err:            hcommon.ErrorMessageWithCausesCtx(ctx, err),
			})
			// 标记失败，防止子孙部门被错误挂根
			failedDeptIDs[d.DepartmentID] = true
			continue
		}

		if existing == nil {
			// 新建
			name := strings.TrimSpace(d.DepartmentName)
			if name == "" {
				name = d.DepartmentID // 兜底用 ID，避免空名
			}
			if err := model.ValidateGroupName(name); err != nil {
				// 名字含 / 或超长时用 ID 兜底
				slog.Warn("LandOneIDDepartmentsToGroups: 部门名非法，使用 ID 作为 name",
					"dept_id", d.DepartmentID, "name", name, "err", err)
				name = d.DepartmentID
			}
			g, err := model.CreateUserGroupWithOpts(ctx, name, "", parentGroupID,
				model.GroupSourceOneIDDept, d.DepartmentID)
			if err != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: 创建 oneid_dept 组失败",
					"dept_id", d.DepartmentID, "name", name, "parent_group_id", parentGroupID, "err", err)
				result.LandingFailures = append(result.LandingFailures, LandingFailure{
					DepartmentID:   d.DepartmentID,
					DepartmentName: d.DepartmentName,
					Stage:          "create",
					Err:            hcommon.ErrorMessageWithCausesCtx(ctx, err),
				})
				// 标记失败：本部门没有产出 group_id，其子孙不能错误挂根（例如 depth
				// 超限场景：第 11 层 create 失败 → 第 12 层应整体跳过而非回到根）。
				failedDeptIDs[d.DepartmentID] = true
				continue
			}
			deptToGroupID[d.DepartmentID] = g.ID
			continue
		}

		// 已存在：D. 如果上次打了 to_be_deleted，本次 OneID 又回来了 → 清标记
		if existing.ToBeDeleted {
			if err := model.DB(ctx).Model(&model.UserGroup{}).
				Where("id = ?", existing.ID).
				Update("to_be_deleted", false).Error; err != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: 清除 to_be_deleted 标记失败",
					"group_id", existing.ID, "err", err)
				result.LandingFailures = append(result.LandingFailures, LandingFailure{
					DepartmentID:   d.DepartmentID,
					DepartmentName: d.DepartmentName,
					Stage:          "clear_tbd",
					Err:            err.Error(),
				})
			} else {
				slog.Info("LandOneIDDepartmentsToGroups: 清除 to_be_deleted 标记",
					"group_id", existing.ID, "dept_id", d.DepartmentID)
			}
		}
		// 同步名字 / 父组变化
		curName := strings.TrimSpace(d.DepartmentName)
		if curName == "" {
			curName = d.DepartmentID
		}
		if err := model.ValidateGroupName(curName); err != nil {
			curName = d.DepartmentID
		}
		nameChanged := existing.Name != curName
		parentChanged := existing.ParentID != parentGroupID

		// 无论仅改名、仅换父、或两者都变，统一走 UpdateUserGroupExtForOneIDDept，
		// 它内部会重算 full_path（含子孙级联）+ depth + closure 一致性。
		// 不能只做 DB.Updates({"name": ...}) —— 那样 full_path 会停在老值，
		// 子孙的 full_path 也会过期（现网 bug：name="1组" 但 full_path 还是 ".../一组"）。
		if nameChanged || parentChanged {
			opts := model.UpdateGroupOpts{}
			if nameChanged {
				nameOpt := curName
				opts.Name = &nameOpt
			}
			if parentChanged {
				parentPtr := parentGroupID
				opts.NewParentIDPtr = &parentPtr
			}
			if _, err := model.UpdateUserGroupExtForOneIDDept(ctx, existing.ID, opts); err != nil {
				stage := "update"
				if parentChanged {
					stage = "rename_parent"
				}
				slog.Warn("LandOneIDDepartmentsToGroups: oneid_dept 更新失败",
					"group_id", existing.ID, "stage", stage, "err", err)
				result.LandingFailures = append(result.LandingFailures, LandingFailure{
					DepartmentID:   d.DepartmentID,
					DepartmentName: d.DepartmentName,
					Stage:          stage,
					Err:            hcommon.ErrorMessageWithCausesCtx(ctx, err),
				})
			} else if parentChanged {
				// 🆕 v6.13：只有 parent 切换且更新成功才记一条
				result.ChangedParentGroupIDs = append(result.ChangedParentGroupIDs, existing.ID)
			}
		}
		deptToGroupID[d.DepartmentID] = existing.ID
	}

	// ── B+C. 对比出 OneID 消失的 oneid_dept 组子树，级联删除或标记 ──
	var localOneIDGroups []model.UserGroup
	if err := model.DB(ctx).Where("source = ?", model.GroupSourceOneIDDept).
		Find(&localOneIDGroups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptLandingReadLocal)
	}

	// 本地缺少对应 OneID 部门（按 source_ref 匹配）。
	//
	// 🐛 v6 hotfix：不再用 !g.ToBeDeleted 过滤。
	// 过去的逻辑是"上次已标 to_be_deleted 的组本次跳过"，但这会让"用户解绑
	// 绑定后再次同步应清理"的路径走不通——已标记的组永远不会再被重新评估
	// 为可删除。现在让已标 to_be_deleted 的组也参与 CanDelete 重判：
	//   - 仍不可删 → 继续保留（markOneIDSubtreeToBeDeleted 对 to_be_deleted=1
	//     的行天然幂等，不会重复记入 NewlyMarkedToBeDeleted）
	//   - 变成可删 → 走 DeleteUserGroupForOneIDDept 实际清理
	var missingRoots []model.UserGroup
	for _, g := range localOneIDGroups {
		if !activeDeptIDSet[g.SourceRef] {
			missingRoots = append(missingRoots, g)
		}
	}

	// 只处理"子树根"：即父组仍活跃（或无父），避免重复处理子节点
	localActiveGroupIDs := make(map[uint]bool)
	for deptID, gid := range deptToGroupID {
		if activeDeptIDSet[deptID] {
			localActiveGroupIDs[gid] = true
		}
	}
	subtreeRoots := filterSubtreeRoots(missingRoots, localActiveGroupIDs)

	// 预查询 CLS 采集范围分组（不依赖循环变量，避免重复查询）
	clsScopeIDs, err := model.GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		slog.Warn("LandOneIDDepartmentsToGroups: 查询 CLS 采集范围失败，告警功能可能失效", "error", err)
	}
	clsScopeSet := make(map[uint]struct{}, len(clsScopeIDs))
	for _, sid := range clsScopeIDs {
		clsScopeSet[sid] = struct{}{}
	}

	for _, root := range subtreeRoots {
		// 收集子树（祖先在前、叶子在后），全部都是 oneid_dept
		descendantIDs, err := model.ClosureDescendants(ctx, root.ID, true)
		if err != nil {
			slog.Warn("LandOneIDDepartmentsToGroups: 读取子树后代失败，跳过",
				"root_id", root.ID, "err", err)
			continue
		}

		// 🔄 v6 语义：**逐个组独立判断**而不是整子树 all-or-nothing。
		//
		// 流程：
		//  1. post-order 遍历（叶子先、祖先后）
		//  2. 对每个组独立调 CanDeleteUserGroup
		//     - 能删 && 当前已无"未删子孙"占位 → 物理删除，加入 deletedSet
		//     - 否则 → 标 to_be_deleted
		//
		//  "父级级联保留"规则：即使某个祖先本身 CanDeleteUserGroup=true，
		//   只要它有任何一个子孙没被物理删（被标了 to_be_deleted 保留），
		//   就必须保留（否则子孙变孤儿），一起标 to_be_deleted。
		post := reverseUintSlice(descendantIDs) // 叶子先
		deleted := make(map[uint]bool, len(post))
		var kept []uint // 被保留（需要标 to_be_deleted）的组 id
		for _, gid := range post {
			// 先查该组是否还有"未被删除"的子孙（基于当前轮的 deleted 集合 + DB 里仍在的非候选子孙）
			hasSurvivingChild, err := hasSurvivingChildGroup(ctx, gid, deleted)
			if err != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: 查询子孙失败，保留",
					"group_id", gid, "err", err)
				kept = append(kept, gid)
				continue
			}
			if hasSurvivingChild {
				// 有子孙留下 → 必须保留做父节点
				kept = append(kept, gid)
				continue
			}
			canDel, rerr := model.CanDeleteUserGroup(ctx, gid)
			if rerr != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: CanDeleteUserGroup 失败，保留",
					"group_id", gid, "err", rerr)
				kept = append(kept, gid)
				continue
			}
			if !canDel {
				kept = append(kept, gid)
				continue
			}
			// 物理删
			if err := model.DeleteUserGroupForOneIDDept(ctx, gid); err != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: 物理删除失败，降级为标记",
					"group_id", gid, "err", err)
				kept = append(kept, gid)
				continue
			}
			deleted[gid] = true
		}

		if len(deleted) > 0 {
			slog.Info("LandOneIDDepartmentsToGroups: 子树部分物理删除",
				"root_id", root.ID, "root_full_path", root.FullPath,
				"deleted_count", len(deleted), "kept_count", len(kept))
		}
		if len(kept) > 0 {
			marked, err := markOneIDSubtreeToBeDeleted(ctx, kept)
			if err != nil {
				slog.Warn("LandOneIDDepartmentsToGroups: 标记 to_be_deleted 失败",
					"root_id", root.ID, "err", err)
				continue
			}
			result.NewlyMarkedToBeDeleted = append(result.NewlyMarkedToBeDeleted, marked...)
			slog.Info("LandOneIDDepartmentsToGroups: 部分组标记 to_be_deleted",
				"root_id", root.ID, "root_full_path", root.FullPath, "count", len(kept))

			// 告警：标记待删的组如果在 CLS 采集范围中，提醒管理员手动处理
			if len(clsScopeSet) > 0 {
				var overlap []uint
				for _, kid := range kept {
					if _, ok := clsScopeSet[kid]; ok {
						overlap = append(overlap, kid)
					}
				}
				if len(overlap) > 0 {
					slog.Warn("LandOneIDDepartmentsToGroups: 标记待删的 OneID 组仍在 CLS 采集范围中，请管理员手动更新 scope",
						"root_id", root.ID, "cls_scope_overlap_group_ids", overlap)
				}
			}
		}
	}

	return result, nil
}

// reverseUintSlice 反转 uint 切片（非原地，返回新切片）。
// 用于把 ClosureDescendants 的"祖先在前"序列翻成"叶子在前"的 post-order 序列。
func reverseUintSlice(in []uint) []uint {
	out := make([]uint, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

// hasSurvivingChildGroup 判断 groupID 是否有"尚未被物理删除"的直属子组（parent_id = groupID）。
// deleted 是本轮已在内存里物理删除的 id 集合（它们可能还在事务可见范围内，但逻辑上视为已删）。
//
// 语义：
//   - 只看直属子组（parent_id = groupID），不递归 —— 因为 post-order 保证更深的子孙
//     如果能删早已被删除、如果保留就会作为本 groupID 的直属子组的子孙存在，而
//     "直属子组存在 ⇒ 必然有后代未被删"，所以直属子组的存活性就够了。
//   - 同时排除本轮 deleted 集合里的 id。
func hasSurvivingChildGroup(ctx context.Context, groupID uint, deleted map[uint]bool) (bool, error) {
	var children []model.UserGroup
	if err := model.DB(ctx).
		Where("parent_id = ?", groupID).
		Find(&children).Error; err != nil {
		return false, err
	}
	for _, c := range children {
		if !deleted[c.ID] {
			return true, nil
		}
	}
	return false, nil
}

// topologicalOrderByParent 按父→子拓扑序返回部门；
// 在本数据集内部（不跨数据集）解析父子关系，孤儿（父不在数据集中）按根处理。
func topologicalOrderByParent(depts []model.OneIDDepartmentRecord) []model.OneIDDepartmentRecord {
	idToDept := make(map[string]model.OneIDDepartmentRecord, len(depts))
	for _, d := range depts {
		idToDept[d.DepartmentID] = d
	}

	// 对每个部门计算 "本数据集内的 depth"
	depthOf := make(map[string]int, len(depts))
	var resolve func(id string, visiting map[string]bool) int
	resolve = func(id string, visiting map[string]bool) int {
		if dp, ok := depthOf[id]; ok {
			return dp
		}
		if visiting[id] {
			// 环：当作根
			depthOf[id] = 0
			return 0
		}
		d, ok := idToDept[id]
		if !ok {
			return 0
		}
		if d.DepartmentParentID == "" {
			depthOf[id] = 0
			return 0
		}
		if _, pok := idToDept[d.DepartmentParentID]; !pok {
			// 父不在本数据集 → 当作根
			depthOf[id] = 0
			return 0
		}
		visiting[id] = true
		depthOf[id] = resolve(d.DepartmentParentID, visiting) + 1
		delete(visiting, id)
		return depthOf[id]
	}
	for _, d := range depts {
		_ = resolve(d.DepartmentID, map[string]bool{})
	}

	sorted := make([]model.OneIDDepartmentRecord, len(depts))
	copy(sorted, depts)
	sort.SliceStable(sorted, func(i, j int) bool {
		di := depthOf[sorted[i].DepartmentID]
		dj := depthOf[sorted[j].DepartmentID]
		if di != dj {
			return di < dj
		}
		return sorted[i].DepartmentID < sorted[j].DepartmentID
	})
	return sorted
}

// filterSubtreeRoots 从 missingRoots 中滤除那些"父组也在 missingRoots 中"的节点，
// 只保留每棵子树的最顶端。以避免对同一子树重复操作。
func filterSubtreeRoots(missing []model.UserGroup, _ map[uint]bool) []model.UserGroup {
	missingIDs := make(map[uint]bool, len(missing))
	for _, g := range missing {
		missingIDs[g.ID] = true
	}
	roots := make([]model.UserGroup, 0, len(missing))
	for _, g := range missing {
		if g.ParentID == 0 || !missingIDs[g.ParentID] {
			roots = append(roots, g)
		}
	}
	return roots
}

// markOneIDSubtreeToBeDeleted 把整棵子树的 to_be_deleted 打 1，返回新标记组明细（id + full_path）。
// 只返回"本次从 0 → 1"的组，避免重复计数。
func markOneIDSubtreeToBeDeleted(ctx context.Context, descendantIDs []uint) ([]AffectedDeptGroup, error) {
	if len(descendantIDs) == 0 {
		return nil, nil
	}
	// 先取出"当前 to_be_deleted=0"的组，它们会被本次标记
	var newly []model.UserGroup
	if err := model.DB(ctx).
		Where("id IN ? AND to_be_deleted = ?", descendantIDs, false).
		Find(&newly).Error; err != nil {
		return nil, err
	}
	if len(newly) == 0 {
		return nil, nil
	}
	newlyIDs := make([]uint, len(newly))
	items := make([]AffectedDeptGroup, len(newly))
	for i, g := range newly {
		newlyIDs[i] = g.ID
		items[i] = AffectedDeptGroup{GroupID: g.ID, FullPath: g.FullPath}
	}
	if err := model.DB(ctx).Model(&model.UserGroup{}).
		Where("id IN ?", newlyIDs).
		Update("to_be_deleted", true).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ──────────────────────────────────────────────
// Step 2.6：部门成员落地（user_group_members(source='oneid_dept', is_main)）
// ──────────────────────────────────────────────

// SyncOneIDMemberships 执行 Step 2.6：
//
//	A. 遍历 oneid_user_profiles.DepartmentsJSON，
//	   Upsert user_group_members(source='oneid_dept', is_main 按 IsMainDepartment)
//	B. 删除"profile 中不再出现"的 oneid_dept 成员行
//
// 仅处理本地 users 表里存在 one_id_sub 的用户。
//
// 🆕 v6.13：返回值新增 *MembershipsResult，其中 MovedUsers 记录本次被移出分组的
// (user_id, from_group_id) 列表 —— 用于 /admin/oneid-sync-users 响应。调用方
// 可以传 nil result，仅关心 error（兼容旧行为）。
func SyncOneIDMemberships(ctx context.Context) (*MembershipsResult, error) {
	result := &MembershipsResult{}
	// 1. 拉所有 profile
	var profiles []model.OneIDUserProfile
	if err := model.DB(ctx).Find(&profiles).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptSyncReadProfiles)
	}
	if len(profiles) == 0 {
		return result, nil
	}

	// 2. 拉 sub → user_id 映射（只含 one_id_sub 非空的本地用户）
	//
	// 必须用 Unscoped()：被 OneID 上游标记 Suspended/Disabled/LockedOut 的用户，
	// 在 ClawPro 走 model.DB(ctx).Delete 软删（deleted_at != NULL）。如果不绕开 GORM 的
	// 软删过滤，禁用用户的 sub 不会进 subToUserID，下面 desired map 跳过他的所有
	// 部门，step 6 差集就把他的 oneid_dept membership 全删了 —— 表现为"OneID 上
	// 仍在组织结构内的禁用用户在 ClawPro 上突然变成未分组用户"。
	type userRow struct {
		ID       uint
		OneIDSub string `gorm:"column:one_id_sub"`
	}
	var users []userRow
	if err := model.DB(ctx).Unscoped().Model(&model.User{}).
		Where("one_id_sub IS NOT NULL AND one_id_sub != ''").
		Select("id, one_id_sub").
		Scan(&users).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptSyncReadUsers)
	}
	subToUserID := make(map[string]uint, len(users))
	for _, u := range users {
		subToUserID[u.OneIDSub] = u.ID
	}

	// 3. 拉 oneid_dept 组 map：DepartmentID → group_id
	var oneidGroups []model.UserGroup
	if err := model.DB(ctx).Where("source = ?", model.GroupSourceOneIDDept).
		Find(&oneidGroups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptSyncReadDeptGroups)
	}
	deptIDToGroupID := make(map[string]uint, len(oneidGroups))
	for _, g := range oneidGroups {
		if g.SourceRef != "" {
			deptIDToGroupID[g.SourceRef] = g.ID
		}
	}
	if len(deptIDToGroupID) == 0 {
		// 部门还没落地就没必要同步成员
		return result, nil
	}

	// 4. 组装期望的成员关系：{user_id, group_id, is_main}
	type memberKey struct {
		UserGroupID uint
		UserID      uint
	}
	desired := make(map[memberKey]bool) // value = is_main

	for _, p := range profiles {
		uid, ok := subToUserID[p.OneIDSub]
		if !ok {
			// 用户还未落地到 users 表（同步流程的 Step 2 失败？）→ 跳过
			continue
		}
		if p.DepartmentsJSON == "" || p.DepartmentsJSON == "[]" {
			continue
		}
		var depts []model.OneIDDepartment
		if err := json.Unmarshal([]byte(p.DepartmentsJSON), &depts); err != nil {
			slog.Warn("SyncOneIDMemberships: 解析 departments_json 失败", "sub", p.OneIDSub, "err", err)
			continue
		}
		for _, d := range depts {
			gid, gok := deptIDToGroupID[d.DepartmentID]
			if !gok {
				// 部门未落地本地（可能 landing 时被裁）→ 跳过
				continue
			}
			k := memberKey{UserGroupID: gid, UserID: uid}
			// 多个来源同一 (user, group) 时，任一 is_main=true 即主部门
			desired[k] = desired[k] || d.IsMainDepartment
		}
	}

	// 5. 拉本地当前 oneid_dept 成员
	var current []model.UserGroupMember
	if err := model.DB(ctx).Where("source = ?", model.MemberSourceOneIDDept).
		Find(&current).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDDeptSyncReadMembers)
	}
	currentMap := make(map[memberKey]model.UserGroupMember, len(current))
	for _, m := range current {
		currentMap[memberKey{UserGroupID: m.UserGroupID, UserID: m.UserID}] = m
	}

	// 6. 事务：新增 / 更新 is_main / 删除
	//
	// 🆕 v6.13：在事务里收集被移出分组的 (user_id, from_group_id) 对，
	// 成功提交后才追加到 result.MovedUsers（失败回滚时不应报告）。
	var movedBuf []MovedGroupUser
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除
		toDelete := make([]uint, 0)
		localMoved := make([]MovedGroupUser, 0)
		for k, m := range currentMap {
			if _, stillDesired := desired[k]; !stillDesired {
				toDelete = append(toDelete, m.ID)
				localMoved = append(localMoved, MovedGroupUser{
					UserID:      k.UserID,
					FromGroupID: k.UserGroupID,
				})
			}
		}
		if len(toDelete) > 0 {
			if err := tx.Where("id IN ?", toDelete).Delete(&model.UserGroupMember{}).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgOneIDDeptDeleteStaleMember)
			}
		}
		movedBuf = localMoved

		// 新增 / 更新 is_main
		for k, isMain := range desired {
			cur, exists := currentMap[k]
			if !exists {
				// 新增
				m := model.UserGroupMember{
					UserGroupID: k.UserGroupID,
					UserID:      k.UserID,
					IsMain:      isMain,
					Source:      model.MemberSourceOneIDDept,
					CreatedAt:   time.Now(),
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "identifier"}, {Name: "user_group_id"}, {Name: "user_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"is_main", "source"}),
				}).Create(&m).Error; err != nil {
					return hcommon.I18nRichError(err, i18n.MsgOneIDDeptInsertMember,
						k.UserGroupID, k.UserID)
				}
				continue
			}
			// 已存在：is_main 或 source 不一致 → 更新
			if cur.IsMain != isMain || cur.Source != model.MemberSourceOneIDDept {
				if err := tx.Model(&model.UserGroupMember{}).
					Where("id = ?", cur.ID).
					Updates(map[string]any{
						"is_main": isMain,
						"source":  model.MemberSourceOneIDDept,
					}).Error; err != nil {
					return hcommon.I18nRichError(err, i18n.MsgOneIDDeptUpdateIsMain, cur.ID)
				}
			}
		}

		// is_main 互斥约束：同一用户至多一个主部门（取第一个，其余清 false）
		if err := enforceSingleMainPerUserTx(tx); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	result.MovedUsers = movedBuf
	return result, nil
}

// enforceSingleMainPerUserTx 保证每个用户在 oneid_dept 组中至多 1 个 is_main=true。
// 策略：若有多个 is_main=true，保留 group_id 最小的那条，其余置 false。
func enforceSingleMainPerUserTx(tx *gorm.DB) error {
	type row struct {
		UserID uint
	}
	var rows []row
	err := tx.Raw(`
		SELECT user_id FROM user_group_members
		WHERE source = ? AND is_main = ?
		GROUP BY user_id
		HAVING COUNT(*) >= 2
	`, model.MemberSourceOneIDDept, true).Scan(&rows).Error
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDDeptQueryMultiMain)
	}
	if len(rows) == 0 {
		return nil
	}
	for _, r := range rows {
		// 先拿所有主部门行，按 user_group_id ASC
		var mainRows []model.UserGroupMember
		if err := tx.Where("user_id = ? AND source = ? AND is_main = ?",
			r.UserID, model.MemberSourceOneIDDept, true).
			Order("user_group_id ASC").
			Find(&mainRows).Error; err != nil {
			return err
		}
		if len(mainRows) <= 1 {
			continue
		}
		// 保留第一条，其余清 is_main
		keepIDs := []uint{mainRows[0].ID}
		var resetIDs []uint
		for i := 1; i < len(mainRows); i++ {
			resetIDs = append(resetIDs, mainRows[i].ID)
		}
		if len(resetIDs) > 0 {
			if err := tx.Model(&model.UserGroupMember{}).
				Where("id IN ?", resetIDs).
				Update("is_main", false).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgOneIDDeptClearExtraMain, keepIDs)
			}
			slog.Info("SyncOneIDMemberships: 用户有多主部门，保留最小 group_id",
				"user_id", r.UserID, "keep_member_id", keepIDs[0], "reset_count", len(resetIDs))
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

// ErrOneIDDeptLandingSkipped landing 跳过（例如部门数据为空）。
var ErrOneIDDeptLandingSkipped = hcommon.I18nError(i18n.MsgOneIDDeptLandingSkipped)
