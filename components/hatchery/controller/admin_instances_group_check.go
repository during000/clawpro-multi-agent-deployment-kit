package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

const groupCheckMaxIDs = 500

type groupCheckRequest struct {
	IDs              []uint `json:"ids"`
	CheckUserGroup   bool   `json:"check_user_group"`
	CheckConfigDrift bool   `json:"check_config_drift"`
}

type groupCheckResultItem struct {
	ID                uint `json:"id"`
	UserGroupMismatch bool `json:"user_group_mismatch"`
	HasConfigDrift    bool `json:"has_config_drift"`
}

// groupCheckItem 是 enrichAdminInstancesWithGroupCheck 内部使用的轻量计算载体，
// 不依赖 adminInstanceItemWithStatus（后者是 /admin/instances 列表响应专用结构体）。
type groupCheckItem struct {
	ID                uint
	UserID            uint
	GroupID           uint
	UserGroupMismatch bool
	HasConfigDrift    bool
}

// HandleAdminInstancesGroupCheck POST /admin/instances/group-check
// 异步 check 端点：前端渲染列表后调此端点获取红黄点数据，与主列表解耦。
func HandleAdminInstancesGroupCheck(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req groupCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	// 去重 + 去零
	ids := dedupUintSlice(req.IDs)
	if len(ids) == 0 {
		jsonOK(w, map[string]interface{}{"results": []groupCheckResultItem{}})
		return
	}
	if len(ids) > groupCheckMaxIDs {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 从 DB 加载实例基本字段（UserID + GroupID），构造最小 items 列表
	var rows []struct {
		ID      uint `gorm:"column:id"`
		UserID  uint `gorm:"column:user_id"`
		GroupID uint `gorm:"column:group_id"`
	}
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("id, user_id, group_id").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	items := make([]groupCheckItem, len(rows))
	for i, row := range rows {
		items[i].ID = row.ID
		items[i].UserID = row.UserID
		items[i].GroupID = row.GroupID
	}

	enrichAdminInstancesWithGroupCheck(r.Context(), items, req.CheckUserGroup, req.CheckConfigDrift)

	results := make([]groupCheckResultItem, len(items))
	for i, item := range items {
		results[i] = groupCheckResultItem{
			ID:                item.ID,
			UserGroupMismatch: item.UserGroupMismatch,
			HasConfigDrift:    item.HasConfigDrift,
		}
	}
	jsonOK(w, map[string]interface{}{"results": results})
}

// admin_instances_group_check.go —— /admin/instances 分组关系 & 配置漂移双字段计算
//
// 两个字段互相独立，可通过请求参数分别开关：
//   check_user_group=true    → user_group_mismatch = user 已不在实例 group_id / group_id=0 但 user 已加入某组
//   check_config_drift=true  → has_config_drift    = 实例配置相对 group_id 组配置存在 different 行（跳过 not_check）
//
// 两个都不传 → 保持默认 false，零额外查询开销（保护 /admin/instances 常规链路性能）。
//
// 批量查询策略：
//   1) 一条 SQL 拉所有涉及 user_id 的 (user_id, user_group_id) 关系
//   2) 一条 SQL 拉所有涉及 instance ID 的完整 Instance 记录（config-drift 需要 RoleID / VpcId 等）
//   3) 每个 unique group_id 调 1 次 buildCategoriesForView（内含固定条 SQL；同页实例通常集中于少数 group）
//   4) 一次 loadInstanceLookups 批量拉 model_name / role_name
//   5) 一次 batchFetchCVMInfoMap 批量拉 CVM 公网三字段（用于 T28 公网子行 drift 判定）

// enrichAdminInstancesWithGroupCheck 按开关计算 user_group_mismatch 与 has_config_drift。
//
// 空 items 或两开关都为 false 时立即返回，零查询开销。
func enrichAdminInstancesWithGroupCheck(ctx context.Context, items []groupCheckItem, checkUserGroup, checkConfigDrift bool) {
	if len(items) == 0 {
		return
	}
	if !checkUserGroup && !checkConfigDrift {
		return
	}

	// ── Field 1: user_group_mismatch ──
	if checkUserGroup {
		userGroupMap := loadUserGroupMemberships(ctx, collectUniqueUserIDs(items))
		for i := range items {
			items[i].UserGroupMismatch = computeUserGroupMismatch(items[i].UserID, items[i].GroupID, userGroupMap)
		}
	}

	// ── Field 2: has_config_drift（成本较高，独立开关） ──
	if !checkConfigDrift {
		return
	}

	// 拉全 Instance 记录：config-diff 需要 RoleID / VpcId / SubnetId / SecurityGroupId 等 items 中未带的字段
	instanceMap := loadInstancesByIDs(ctx, collectInstanceIDs(items))
	if len(instanceMap) == 0 {
		return
	}

	// 站点全局配置（buildCategoriesForView 里各种 site_configs 兜底会用到）
	siteConfig := model.GetSiteConfig(ctx)

	// 缓存每个 unique group_id 的目标视图（含 group_id=0 → 全局兜底视角）
	groupIDs := collectUniqueGroupIDs(items)
	targetCatsByGroup := make(map[uint][]usergroup.ConfigCategoryResult, len(groupIDs))
	for _, gid := range groupIDs {
		targetCatsByGroup[gid] = buildCategoriesForView(ctx, gid, &siteConfig)
	}

	// 批量拉 lookups（AIModelID → name / RoleID → name）
	instances := make([]*model.Instance, 0, len(instanceMap))
	for _, inst := range instanceMap {
		instances = append(instances, inst)
	}
	lookups := loadInstanceLookups(ctx, instances)

	// 批量拉 CVM 公网三字段：drift 判定需要覆盖 T28（公网 IP / 计费模式 / 带宽上限）。
	// check_config_drift 是 opt-in 开关，一次批量 CVM API 调用符合预期成本。
	cvmIDs := make([]string, 0, len(instances))
	for _, inst := range instances {
		if inst.InstanceId != "" {
			cvmIDs = append(cvmIDs, inst.InstanceId)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(ctx, cvmIDs)

	instanceIDs := make([]uint, 0, len(instances))
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.ID)
	}
	memoryPluginMap := batchFetchMemoryPluginMap(ctx, cvmIDs)
	driveSpaceMap := batchFetchDriveSpaceMap(ctx, instanceIDs)
	cwpSecurityMap := batchFetchCWPSecurityMap(ctx, cvmIDs)

	// 实例组视图缓存（drift 场景 target group == inst.GroupID，可直接从 targetCatsByGroup
	// 过滤出 5 类跟随组 category，避免 buildInstanceCategoriesView 逐实例调
	// buildCategoriesForGroup —— 这是 config-drift 主要瓶颈来源）
	groupCache := buildInstanceGroupInheritedCacheFromTargetCats(targetCatsByGroup)

	for i := range items {
		inst, ok := instanceMap[items[i].ID]
		if !ok || inst == nil {
			continue
		}
		targetCats, ok := targetCatsByGroup[items[i].GroupID]
		if !ok {
			continue
		}
		diff := computeConfigDiff(ctx, inst, targetCats, lookups, cvmInfoMap[inst.InstanceId], &siteConfig, groupCache, memoryPluginMap, driveSpaceMap, cwpSecurityMap)
		items[i].HasConfigDrift = anyDifferentRow(diff.Categories)
	}
}

// computeUserGroupMismatch 判断实例的 user_id 与 group_id 是否失配。
// 语义（对齐 stale-instances 场景 A/B/C/D）：
//   - group_id != 0 且 userGroupMap[userID] 不含 group_id → mismatch=true（场景 A/B/D）
//   - group_id == 0 且 userGroupMap[userID] 非空 → mismatch=true（场景 C：用户已加入分组但实例仍未分组）
//   - 其他 → false
func computeUserGroupMismatch(userID, groupID uint, userGroupMap map[uint]map[uint]struct{}) bool {
	groups := userGroupMap[userID]
	if groupID == 0 {
		return len(groups) > 0
	}
	if _, ok := groups[groupID]; !ok {
		return true
	}
	return false
}

// anyDifferentRow 检查 config-diff 结果里是否含 status=different 行。
// not_check / same / contained_in_target 都不视作 drift。
func anyDifferentRow(rows []instanceConfigRow) bool {
	for _, r := range rows {
		if r.Status == "different" {
			return true
		}
	}
	return false
}

// collectUniqueUserIDs / collectUniqueGroupIDs / collectInstanceIDs — 从响应列表提取批量查询 IDs。
func collectUniqueUserIDs(items []groupCheckItem) []uint {
	seen := make(map[uint]struct{}, len(items))
	out := make([]uint, 0, len(items))
	for i := range items {
		uid := items[i].UserID
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		out = append(out, uid)
	}
	return out
}

func collectUniqueGroupIDs(items []groupCheckItem) []uint {
	seen := make(map[uint]struct{}, len(items))
	out := make([]uint, 0, len(items))
	for i := range items {
		gid := items[i].GroupID
		if _, ok := seen[gid]; ok {
			continue
		}
		seen[gid] = struct{}{}
		out = append(out, gid)
	}
	return out
}

func collectInstanceIDs(items []groupCheckItem) []uint {
	out := make([]uint, 0, len(items))
	for i := range items {
		out = append(out, items[i].ID)
	}
	return out
}

// loadUserGroupMemberships 一次 SQL 拉出所有涉及 user_id 的 (user_id, user_group_id) 关系。
// 返回 map：user_id → set<user_group_id>。
func loadUserGroupMemberships(ctx context.Context, userIDs []uint) map[uint]map[uint]struct{} {
	out := make(map[uint]map[uint]struct{}, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	var rows []struct {
		UserID      uint `gorm:"column:user_id"`
		UserGroupID uint `gorm:"column:user_group_id"`
	}
	if err := model.DB(ctx).Model(&model.UserGroupMember{}).
		Select("user_id, user_group_id").
		Where("user_id IN ?", userIDs).
		Find(&rows).Error; err != nil {
		slog.Warn("[GroupCheck] user_group_members query failed", "err", err)
		return out
	}
	for _, r := range rows {
		if _, ok := out[r.UserID]; !ok {
			out[r.UserID] = make(map[uint]struct{})
		}
		out[r.UserID][r.UserGroupID] = struct{}{}
	}
	return out
}

// loadInstancesByIDs 一次 SQL 拉出全部 Instance 完整记录，返回 id → *Instance。
// config-drift 计算需要 items 中不带的字段（RoleID / VpcId / SubnetId / SecurityGroupId 等）。
func loadInstancesByIDs(ctx context.Context, ids []uint) map[uint]*model.Instance {
	out := make(map[uint]*model.Instance, len(ids))
	if len(ids) == 0 {
		return out
	}
	var rows []model.Instance
	if err := model.DB(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		slog.Warn("[GroupCheck] instances load failed", "err", err)
		return out
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out
}
