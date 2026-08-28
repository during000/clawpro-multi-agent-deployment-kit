package controller

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ========== 端点 10: GET /admin/mcp/instances — 实例安装情况 ==========

// HandleAdminMcpInstances 查询 MCP 实例安装情况（分页）。
// 对齐 HandleAdminSkillInstances 的 4 步流程：
//  1. 全量查询（SQL 基础查询 + 过滤条件）
//  2. 批量查询 CVM 实时状态
//  3. 计算语义状态，过滤出 running 实例
//  4. 内存分页 + 批量加载用户组 + 组装响应
func HandleAdminMcpInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	serviceID := r.URL.Query().Get("service_id")
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	instanceType := r.URL.Query().Get("instance_type")
	agentVersionMin := r.URL.Query().Get("agent_version_min")
	agentVersionMax := r.URL.Query().Get("agent_version_max")
	slog.Info("查询 MCP 实例安装情况", "service_id", serviceID, "status", statusFilter, "instance_type", instanceType)
	if serviceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	// 查找 MCP
	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", serviceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	// 获取最新版本号
	var latestVersion string
	if server.LatestVersionID > 0 {
		var lv model.McpVersion
		if err := model.DB(r.Context()).Where("id = ?", server.LatestVersionID).First(&lv).Error; err == nil {
			latestVersion = lv.Version
		}
	}

	page, pageSize := parsePagination(r, 500)

	type instResp struct {
		InstanceID            uint       `json:"instance_id"              gorm:"column:instance_id"`
		CVMInstanceID         string     `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName          string     `json:"instance_name"            gorm:"column:instance_name"`
		InstanceType          string     `json:"instance_type"            gorm:"column:instance_type"`
		UserID                uint       `json:"user_id"                  gorm:"column:user_id"`
		Source                string     `json:"-"                        gorm:"column:source"`
		Username              string     `json:"username"                 gorm:"column:username"`
		LastCVMState          string     `json:"last_cvm_state"           gorm:"column:last_cvm_state"`
		LastStableState       string     `json:"-"                        gorm:"column:last_stable_state"`
		CurrentOperation      string     `json:"-"                        gorm:"column:current_operation"`
		CurrentOperationState string     `json:"-"                        gorm:"column:current_operation_state"`
		AgentReady            int        `json:"-"                        gorm:"column:agent_ready"`
		CLSAgentStatus        int        `json:"-"                        gorm:"column:cls_agent_status"`
		CLSAgentStatusAt      *time.Time `json:"-"                        gorm:"column:cls_agent_status_at"`
		AgentVersion          string     `json:"agent_version"            gorm:"column:agent_version"`
		Status                string     `json:"status"                   gorm:"column:install_status"`
		Version               string     `json:"version"                  gorm:"column:version"`
		LatestVersion         string     `json:"latest_version"           gorm:"column:latest_version"`
	}

	// 构造基础查询：LEFT JOIN users + mcp_installations，在 SQL 层推导安装状态
	baseQuery := model.BuildMcpInstanceQueryV2(r.Context(), serviceID, latestVersion)

	// 按用户组筛选实例（辅助筛选，支持逗号分隔多个 group_id）
	// group_id=0 表示未分组用户的实例，可与正常 group_id 组合使用，如 group_id=0,1,3
	if groupIDStr := r.URL.Query().Get("group_id"); groupIDStr != "" {
		var groupIDs []int
		includeUngrouped := false
		for _, s := range strings.Split(groupIDStr, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				continue
			}
			if id == 0 {
				includeUngrouped = true
			} else if id > 0 {
				groupIDs = append(groupIDs, id)
			}
		}
		if includeUngrouped && len(groupIDs) > 0 {
			// 未分组 + 指定分组：OR 语义
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?) OR instances.user_id IN (?)", ungroupedSubQ, groupedSubQ)
		} else if includeUngrouped {
			// 仅未分组
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?)", ungroupedSubQ)
		} else if len(groupIDs) > 0 {
			// 仅指定分组（使用子查询避免 JOIN 产生重复行）
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
		}
	}

	if search != "" {
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 按实例类型筛选（支持逗号分隔多类型，如 instance_type=openclaw,Hermes）
	if instanceType != "" {
		types := strings.Split(instanceType, ",")
		trimmed := make([]string, 0, len(types))
		for _, t := range types {
			if s := strings.TrimSpace(t); s != "" {
				trimmed = append(trimmed, s)
			}
		}
		if len(trimmed) > 0 {
			baseQuery = baseQuery.Where("instances.agent_type IN ?", trimmed)
		}
	}

	// 安装状态筛选（SQL 层预过滤，减少全量数据量）
	// 复用 model.McpInstallStatusCase 确保与 SELECT 中的 CASE 逻辑一致
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		baseQuery = baseQuery.Where(model.McpInstallStatusCase(latestVersion)+" IN ?", statuses)
	}

	// ── 第一步：全量查询（不分页），用于批量计算实例语义状态后内存过滤 ──
	var allResults []instResp
	if err := baseQuery.Order("instances.created_at DESC").Scan(&allResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreQueryInstancesFail, err))
		return
	}
	if allResults == nil {
		allResults = []instResp{}
	}

	// ── 第二步：批量查询 CVM 实时状态 ──
	var cvmIDs []string
	for _, row := range allResults {
		if row.CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, row.CVMInstanceID)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// ── 批量预查：消除循环内 N+1 ──
	siteConfig := model.GetSiteConfig(r.Context())
	preInstIDs := make([]uint, 0, len(allResults))
	localInstIDs := make([]uint, 0)
	for _, row := range allResults {
		preInstIDs = append(preInstIDs, row.InstanceID)
		if row.Source == model.InstanceSourceLocal {
			localInstIDs = append(localInstIDs, row.InstanceID)
		}
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), preInstIDs)
	localInfoMap := batchResolveLocalInstanceStatus(r.Context(), localInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap, LocalInfoMap: localInfoMap}

	// ── 第三步：计算每个实例的语义状态，过滤出 running 的实例 ──
	type instWithStatus struct {
		instResp
		InstanceStatus      string
		InstanceStatusLabel string
		Transient           bool
	}
	var runningResults []instWithStatus
	for _, row := range allResults {
		tmpInst := model.Instance{
			LastCVMState:          row.LastCVMState,
			LastStableState:       row.LastStableState,
			CurrentOperation:      row.CurrentOperation,
			CurrentOperationState: row.CurrentOperationState,
			AgentReady:            row.AgentReady,
			CLSAgentStatus:        row.CLSAgentStatus,
			CLSAgentStatusAt:      row.CLSAgentStatusAt,
			InstanceId:            row.CVMInstanceID,
			Source:                row.Source,
		}
		tmpInst.ID = row.InstanceID
		cvmInfo := cvmInfoMap[row.CVMInstanceID]
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfo, batch)
		// 只保留 instance_status=running 的实例
		if statusResp.Status != model.StatusRunning {
			continue
		}
		runningResults = append(runningResults, instWithStatus{
			instResp:            row,
			InstanceStatus:      statusResp.Status,
			InstanceStatusLabel: statusResp.Label,
			Transient:           statusResp.Transient,
		})
	}

	// ── 第三步（续）：按实例版本（agent_version）过滤 ──
	// agent_version 格式为日期版本号（如 2026.4.11），比较时按 YYYY.M.D 逐段数值比较
	// agent_version_min: 闭区间下界（>=）
	// agent_version_max: 闭区间上界（<=）
	// 精确匹配可通过 min=max 实现，如 agent_version_min=2026.4.11&agent_version_max=2026.4.11
	if agentVersionMin != "" || agentVersionMax != "" {
		var filtered []instWithStatus
		for _, row := range runningResults {
			if agentVersionMin != "" && compareAgentVersion(row.AgentVersion, agentVersionMin) < 0 {
				continue
			}
			if agentVersionMax != "" && compareAgentVersion(row.AgentVersion, agentVersionMax) > 0 {
				continue
			}
			filtered = append(filtered, row)
		}
		runningResults = filtered
	}

	// ── 第四步：内存分页 ──
	total := int64(len(runningResults))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(runningResults) {
		start = len(runningResults)
	}
	if end > len(runningResults) {
		end = len(runningResults)
	}
	pageResults := runningResults[start:end]

	// 批量加载用户所属分组
	userIDSet := make(map[uint]bool)
	for _, row := range pageResults {
		if row.UserID > 0 {
			userIDSet[row.UserID] = true
		}
	}
	userGroupMap := make(map[uint][]model.UserGroup)
	if len(userIDSet) > 0 {
		userIDs := make([]uint, 0, len(userIDSet))
		for uid := range userIDSet {
			userIDs = append(userIDs, uid)
		}
		if m, err := model.GetUserGroupsByUserIDs(r.Context(), userIDs); err == nil {
			userGroupMap = m
		} else {
			slog.Error("[McpInstances] 批量查询用户分组失败", "error", err)
		}
	}

	type groupInfo struct {
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
	}
	type instFinalResp struct {
		instResp
		UserGroups          []groupInfo `json:"user_groups"`
		InstanceStatus      string      `json:"instance_status"`
		InstanceStatusLabel string      `json:"instance_status_label"`
		Transient           bool        `json:"transient"`
	}
	finalResults := make([]instFinalResp, 0, len(pageResults))
	for _, row := range pageResults {
		item := instFinalResp{
			instResp:            row.instResp,
			InstanceStatus:      row.InstanceStatus,
			InstanceStatusLabel: row.InstanceStatusLabel,
			Transient:           row.Transient,
		}
		if groups, ok := userGroupMap[row.UserID]; ok {
			for _, g := range groups {
				item.UserGroups = append(item.UserGroups, groupInfo{GroupID: g.ID, GroupName: g.Name})
			}
		}
		if item.UserGroups == nil {
			item.UserGroups = []groupInfo{}
		}
		finalResults = append(finalResults, item)
	}

	jsonOK(w, map[string]interface{}{
		"instances": finalResults,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// compareAgentVersion 按日期版本号逐段数值比较（格式 YYYY.M.D，如 2026.4.11）。
// 返回 -1 / 0 / +1。无法解析的版本号视为 0.0.0（排最前）。
func compareAgentVersion(a, b string) int {
	pa := parseAgentVersionParts(a)
	pb := parseAgentVersionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

// parseAgentVersionParts 将 "2026.4.11" 解析为 [2026, 4, 11]。
// 不足 3 段的部分补 0，解析失败的段视为 0。
func parseAgentVersionParts(v string) [3]int {
	var parts [3]int
	segs := strings.Split(v, ".")
	for i := 0; i < 3 && i < len(segs); i++ {
		n, _ := strconv.Atoi(segs[i])
		parts[i] = n
	}
	return parts
}
