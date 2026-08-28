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

// maxScopeGroupIDs 单次更新 CLS 采集范围的最大分组数量限制，与平台分组上限对齐。
var maxScopeGroupIDs = model.MaxUserGroupsPerPlatform

// ---------- GET /admin/cls/scope ----------

// HandleAdminGetCLSScope 查询当前 CLS 采集范围配置。
//
// 响应：
//
//	{
//	  "ok": true,
//	  "scope_type": "all" | "group",
//	  "group_ids": [1, 2, 3],
//	  "groups": [{"id": 1, "name": "研发组", "full_path": "根组/研发组", "install_stats": {...}}],
//	  "total_instance_count": 30,
//	  "total_install_stats": {"not_installed": 5, "installing": 3, "installed": 20, "uninstalling": 0, "skipped": 2}
//	}
func HandleAdminGetCLSScope(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	groupIDs, err := model.GetCLSCollectScopeGroupIDs(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCLSScopeFailed))
		return
	}

	config := model.GetSiteConfig(r.Context())

	if len(groupIDs) == 0 {
		scopeType := config.CLSScopeMode
		if config.CLSEnabled != 1 {
			scopeType = "off"
		}
		jsonOK(w, map[string]interface{}{
			"ok":          true,
			"cls_enabled": config.CLSEnabled == 1,
			"scope_type":  scopeType,
			"group_ids":   []uint{},
			"groups":      []interface{}{},
		})
		return
	}

	// 查询分组详情
	groups, rerr := model.GetGroupsByIDs(r.Context(), groupIDs)
	if rerr != nil {
		slog.Error("[CLS Scope] 查询分组详情失败", "error", rerr)
		groups = nil
	}

	// 批量展开所有 scope 分组的子孙，一次性查询 closure 表
	allDescendants, err := model.ExpandGroupIDsWithDescendants(r.Context(), groupIDs)
	if err != nil {
		slog.Warn("[CLS Scope] 展开分组子孙失败", "error", err)
	}

	// 一次性查询所有实例 ID
	var allScopeInstanceIDs []string
	if len(allDescendants) > 0 {
		allScopeInstanceIDs, rerr = model.GetGroupsCVMInstanceIDs(r.Context(), allDescendants)
		if rerr != nil {
			slog.Warn("[CLS Scope] 查询实例失败", "error", rerr)
		}
	}
	allScopeInstanceSet := make(map[string]struct{}, len(allScopeInstanceIDs))
	for _, id := range allScopeInstanceIDs {
		allScopeInstanceSet[id] = struct{}{}
	}
	totalInstanceCount := len(allScopeInstanceSet)

	// 一次 closure 查询同时用于：1) 统计子孙数量 2) 展开子孙集合
	perGroupDescCount := make(map[uint]int, len(groups))
	perGroupInstances := make(map[uint]map[string]struct{}, len(groups))
	for _, g := range groups {
		perGroupInstances[g.ID] = make(map[string]struct{})
	}
	perGroupDescendantSet := make(map[uint]map[uint]struct{}, len(groups))
	for _, g := range groups {
		perGroupDescendantSet[g.ID] = map[uint]struct{}{g.ID: {}} // 含自身
	}
	if len(groups) > 0 {
		scopeGroupIDs := make([]uint, 0, len(groups))
		for _, g := range groups {
			scopeGroupIDs = append(scopeGroupIDs, g.ID)
		}
		var closureRows []model.GroupClosure
		if err := model.DB(r.Context()).Model(&model.GroupClosure{}).
			Select("ancestor_id, descendant_id").
			Where("ancestor_id IN ?", scopeGroupIDs).
			Find(&closureRows).Error; err != nil {
			slog.Warn("[CLS Scope] 批量查询子孙失败", "error", err)
		} else {
			for _, row := range closureRows {
				// 子孙数量不含自身
				if row.AncestorID != row.DescendantID {
					perGroupDescCount[row.AncestorID]++
				}
				// 子孙集合含自身（初始化时已加入）
				if s, ok := perGroupDescendantSet[row.AncestorID]; ok {
					s[row.DescendantID] = struct{}{}
				}
			}
		}

		// 一次查询：直接通过 instances.group_id 查询分组->实例映射
		type groupInstance struct {
			GroupID    uint
			InstanceId string
		}
		var giRows []groupInstance
		if err := model.DB(r.Context()).Model(&model.Instance{}).
			Select("group_id, instance_id").
			Where("group_id IN ? AND instance_id != ''", allDescendants).
			Scan(&giRows).Error; err != nil {
			slog.Warn("[CLS Scope] 批量查询分组实例映射失败", "error", err)
		} else {
			// groupID -> instanceID set
			groupInstanceMap := make(map[uint]map[string]struct{})
			for _, row := range giRows {
				if groupInstanceMap[row.GroupID] == nil {
					groupInstanceMap[row.GroupID] = make(map[string]struct{})
				}
				groupInstanceMap[row.GroupID][row.InstanceId] = struct{}{}
			}
			// 汇总：每个 scope 分组的实例 = 其所有子孙分组实例的并集
			for _, g := range groups {
				descSet := perGroupDescendantSet[g.ID]
				instSet := perGroupInstances[g.ID]
				for descID := range descSet {
					if gInsts, ok := groupInstanceMap[descID]; ok {
						for instID := range gInsts {
							instSet[instID] = struct{}{}
						}
					}
				}
			}
		}
	}

	// 查询所有 scope 内实例的 cls_agent_status，用于统计安装进度
	type instanceStatus struct {
		InstanceId     string
		CLSAgentStatus int
	}
	var statusRows []instanceStatus
	if totalInstanceCount > 0 {
		uniqueIDs := make([]string, 0, len(allScopeInstanceSet))
		for id := range allScopeInstanceSet {
			uniqueIDs = append(uniqueIDs, id)
		}
		if err := model.DB(r.Context()).Model(&model.Instance{}).
			Select("instance_id, cls_agent_status").
			Where("instance_id IN ? AND deleted_at IS NULL", uniqueIDs).
			Scan(&statusRows).Error; err != nil {
			slog.Warn("[CLS Scope] 查询实例安装状态失败", "error", err)
		}
	}
	// 建立 instanceID → status 映射
	instanceStatusMap := make(map[string]int, len(statusRows))
	for _, row := range statusRows {
		instanceStatusMap[row.InstanceId] = row.CLSAgentStatus
	}

	// 辅助函数：统计一组实例的安装状态分布
	buildInstallStats := func(instSet map[string]struct{}) map[string]int {
		stats := map[string]int{
			"not_installed": 0,
			"installing":    0,
			"installed":     0,
			"uninstalling":  0,
			"skipped":       0,
		}
		for instID := range instSet {
			switch instanceStatusMap[instID] {
			case model.CLSAgentNotInstalled:
				stats["not_installed"]++
			case model.CLSAgentInstalling:
				stats["installing"]++
			case model.CLSAgentInstalled:
				stats["installed"]++
			case model.CLSAgentUninstalling:
				stats["uninstalling"]++
			case model.CLSAgentSkipped:
				stats["skipped"]++
			default:
				stats["not_installed"]++
			}
		}
		return stats
	}

	// 计算总体安装状态统计
	totalInstallStats := buildInstallStats(allScopeInstanceSet)

	groupInfos := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		info := map[string]interface{}{
			"id":               g.ID,
			"name":             g.Name,
			"full_path":        g.FullPath,
			"source":           g.Source,
			"descendant_count": perGroupDescCount[g.ID],
			"instance_count":   len(perGroupInstances[g.ID]),
			"install_stats":    buildInstallStats(perGroupInstances[g.ID]),
		}
		groupInfos = append(groupInfos, info)
	}

	jsonOK(w, map[string]interface{}{
		"ok":                   true,
		"cls_enabled":          config.CLSEnabled == 1,
		"scope_type":           "group",
		"group_ids":            groupIDs,
		"groups":               groupInfos,
		"total_instance_count": totalInstanceCount,
		"total_install_stats":  totalInstallStats,
	})
}

// ---------- POST /admin/cls/scope ----------

// clsScopeRequest 更新 CLS 采集范围的请求体。
type clsScopeRequest struct {
	ScopeType string `json:"scope_type"` // 采集范围模式: "all"=全量, "group"=分组
	GroupIDs  []uint `json:"group_ids"`  // 分组 ID 列表，scope_type="group" 时生效（可为空）
}

// HandleAdminUpdateCLSScope 更新 CLS 采集范围，自动 diff 并对新增/移除的分组下发 TAT。
//
// 请求体：
//
//	{ "scope_type": "all" | "group", "group_ids": [1, 2, 3] }
//
// scope_type="all" 表示全量模式（忽略 group_ids）。
// scope_type="group" 表示分组模式，group_ids 可为空（不安装任何机器）。
//
// 响应：
//
//	{
//	  "ok": true,
//	  "added_groups": [3],
//	  "removed_groups": [1],
//	  "added_instance_count": 5,
//	  "removed_instance_count": 2
//	}
func HandleAdminUpdateCLSScope(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.CLSEnabled != 1 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSServiceNotEnabled))
		return
	}

	var req clsScopeRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB 请求体大小限制
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	// 校验 scope_type
	if req.ScopeType != "all" && req.ScopeType != "group" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidCLSScopeType))
		return
	}

	// 全量模式下忽略 group_ids
	if req.ScopeType == "all" {
		req.GroupIDs = nil
	}

	// 校验分组数量上限
	if len(req.GroupIDs) > maxScopeGroupIDs {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSScopeGroupCountExceed, maxScopeGroupIDs))
		return
	}

	// 校验 group_ids 存在性
	if len(req.GroupIDs) > 0 {
		if err := validateGroupIDs(r.Context(), req.GroupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 持久化 scope_mode
	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{"cls_scope_mode": req.ScopeType}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateCLSScopeModeFailed))
		return
	}

	// 原子地计算 diff 并更新 scope（在同一事务中，避免并发竞态）
	added, removed, err := model.DiffAndSetCLSCollectScope(r.Context(), req.GroupIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateCLSScopeFailed))
		return
	}

	slog.Info("[CLS Scope] 更新采集范围", "new_group_ids", req.GroupIDs, "added", added, "removed", removed)

	// 对新增分组的实例标记为待安装（仅更新未处于活跃状态的实例）
	var addedInstanceCount int
	var warnings []string
	if len(added) > 0 {
		allAdded, err := expandAndGetCVMIDs(r.Context(), added)
		if err != nil {
			slog.Error("[CLS Scope] 展开新增分组实例失败", "error", err)
		} else if len(allAdded) > 0 {
			addedInstanceCount = len(allAdded)
			if err := markInstancesCLSStatusSafe(r.Context(), allAdded, model.CLSAgentNotInstalled); err != nil {
				slog.Error("[CLS Scope] 标记新增实例待安装失败", "error", err)
				warnings = append(warnings, "标记新增实例待安装失败，后台任务将自动补偿")
			}
			slog.Info("[CLS Scope] 已标记新增分组实例为待安装（跳过活跃状态实例）", "group_ids", added, "instance_count", len(allAdded))
		}
	}

	// 对移除分组的独占实例：不主动标记状态，交由后台 runCLSAgentScopeUninstall 定时任务处理。
	// 该任务会查询 cls_agent_status=CLSAgentInstalled 且不在当前 scope 范围内的实例，
	// 自动标记为 CLSAgentUninstalling 并执行卸载脚本。
	// 这里仅统计受影响的实例数用于前端展示。
	var removedInstanceCount int
	if len(removed) > 0 {
		removedInstances, err := getExclusiveRemovedInstances(r.Context(), removed, req.GroupIDs)
		if err != nil {
			slog.Error("[CLS Scope] 计算移除分组独占实例失败", "error", err)
		} else if len(removedInstances) > 0 {
			removedInstanceCount = len(removedInstances)
			slog.Info("[CLS Scope] 移除分组的独占实例将由后台任务自动卸载", "group_ids", removed, "instance_count", len(removedInstances))
		}
	}

	resp := map[string]interface{}{
		"ok":                     true,
		"added_groups":           added,
		"removed_groups":         removed,
		"added_instance_count":   addedInstanceCount,
		"removed_instance_count": removedInstanceCount,
	}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	jsonOK(w, resp)
}

// validateGroupIDs 校验分组 ID 列表的存在性。委托给 usergroup.ValidateGroupIDs 统一实现。
func validateGroupIDs(ctx context.Context, groupIDs []uint) error {
	return usergroup.ValidateGroupIDs(ctx, groupIDs)
}

// expandAndGetCVMIDs 展开分组（含子孙）并查询对应的 CVM 实例 ID。
// 通过 instances.group_id 直接关联（实例所属分组在 scope 子树中即命中）。
func expandAndGetCVMIDs(ctx context.Context, groupIDs []uint) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	// 统一使用 model 层的展开逻辑
	allGroupIDs, err := model.ExpandGroupIDsWithDescendants(ctx, groupIDs)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSDescendantExpandFailed)
	}
	if len(allGroupIDs) == 0 {
		return nil, nil
	}

	return model.GetCVMInstanceIDsInGroups(ctx, allGroupIDs)
}

// getExclusiveRemovedInstances 获取被移除分组中「不再属于任何 scope 分组」的实例。
// 避免多归属用户的实例被误卸载。
func getExclusiveRemovedInstances(ctx context.Context, removedGroupIDs, remainingGroupIDs []uint) ([]string, error) {
	// 获取被移除分组的实例
	removedInstances, err := expandAndGetCVMIDs(ctx, removedGroupIDs)
	if err != nil {
		return nil, err
	}
	if len(removedInstances) == 0 {
		return nil, nil
	}

	if len(remainingGroupIDs) == 0 {
		// 没有剩余 scope 分组，所有被移除实例都是独占的
		return removedInstances, nil
	}

	// 获取剩余 scope 分组的实例
	remainingInstances, err := expandAndGetCVMIDs(ctx, remainingGroupIDs)
	if err != nil {
		return nil, err
	}
	remainingSet := make(map[string]struct{}, len(remainingInstances))
	for _, id := range remainingInstances {
		remainingSet[id] = struct{}{}
	}

	// 过滤：只保留不在剩余 scope 中的实例
	var exclusive []string
	for _, id := range removedInstances {
		if _, ok := remainingSet[id]; !ok {
			exclusive = append(exclusive, id)
		}
	}
	return exclusive, nil
}

// markInstancesCLSStatus 批量将指定 CVM 实例标记为指定的 CLS Agent 状态。
// 无条件更新，用于需要强制覆盖的场景。
//
// Deprecated: 生产代码应使用 markInstancesCLSStatusSafe，它会跳过活跃状态实例。
// 此函数仅保留供测试使用。
func markInstancesCLSStatus(ctx context.Context, instanceIDs []string, status int) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	return model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id IN ?", instanceIDs).
		Updates(map[string]interface{}{
			"cls_agent_status":    status,
			"cls_agent_status_at": nil, // 清空冷却时间，让定时任务立即拾取
		}).Error
}

// markInstancesCLSStatusSafe 批量将指定 CVM 实例标记为指定的 CLS Agent 状态，
// 但跳过正处于活跃状态（安装中、卸载中、已安装）的实例，避免覆盖进行中的操作。
func markInstancesCLSStatusSafe(ctx context.Context, instanceIDs []string, status int) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	// 排除活跃状态：已安装、安装中、卸载中
	activeStatuses := []int{model.CLSAgentInstalled, model.CLSAgentInstalling, model.CLSAgentUninstalling}

	return model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id IN ? AND cls_agent_status NOT IN ?", instanceIDs, activeStatuses).
		Updates(map[string]interface{}{
			"cls_agent_status":    status,
			"cls_agent_status_at": nil,
		}).Error
}

// inheritCLSScopeForNewInstance 新实例创建后，检查其 group_id 是否命中 CLS 采集范围。
// 若命中且 CLS 已开启，则将该实例标记为待安装（cls_agent_status=0），后台任务会拾取并安装。
// 返回 error 以便调用方感知失败（如需要记录日志或重试）。
func inheritCLSScopeForNewInstance(ctx context.Context, groupID uint, instanceID string) error {
	config := model.GetSiteConfig(ctx)
	if config.CLSEnabled != 1 {
		return nil
	}

	inScope, err := model.IsInstanceGroupInCLSCollectScope(ctx, groupID)
	if err != nil {
		slog.Warn("[CLS Scope] 新实例继承检查失败", "group_id", groupID, "instance_id", instanceID, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgCheckCLSScopeForInstanceFailed)
	}
	if !inScope {
		return nil
	}

	// 实例命中 CLS 采集范围，安全标记实例为待安装（跳过活跃状态）
	if err := markInstancesCLSStatusSafe(ctx, []string{instanceID}, model.CLSAgentNotInstalled); err != nil {
		slog.Error("[CLS Scope] 新实例标记待安装失败", "group_id", groupID, "instance_id", instanceID, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgMarkInstanceCLSPendingFailed)
	}
	slog.Info("[CLS Scope] 新实例命中 CLS 采集范围，已标记待安装", "group_id", groupID, "instance_id", instanceID)
	return nil
}
