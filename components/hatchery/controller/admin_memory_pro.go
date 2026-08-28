package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	sdk "hatchery/internal/tdaimemorysdk"
)

// ========== 1. 服务概览 ==========

// HandleAdminMemoryOverview GET /admin/memory/overview
// 返回实例计划分布 + Pro 容量信息。
func HandleAdminMemoryOverview(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())
	log.Info("[MemoryOverview] 收到请求")

	memorySupportedTypes := model.GetMemorySupportedAgentTypes(r.Context())

	// 实例总数：仅统计支持记忆的类型
	var totalInstances int64
	model.DB(r.Context()).Model(&model.Instance{}).
		Where("instance_id != '' AND agent_type IN ?", memorySupportedTypes).
		Count(&totalInstances)

	// 按 current_plan 分组统计：以 DB.Model(&Instance{}) 起头，保证 GORM 回调自动注入
	// identifier + deleted_at IS NULL，与 total_instances 的口径完全一致。
	// LEFT JOIN plugin 拿 current_plan，没 plugin 行或值为空一律视为 OFF。
	planStats := map[string]int64{"OFF": 0, "FREE": 0, "PRO": 0}
	var rows []struct {
		CurrentPlan string
		Count       int64
	}
	model.DB(r.Context()).Model(&model.Instance{}).
		Select("COALESCE(NULLIF(p.current_plan, ''), 'OFF') AS current_plan, count(*) AS count").
		Joins("LEFT JOIN memory_tda_iplugins AS p ON p.instance_id = instances.instance_id AND p.deleted_at IS NULL").
		Where("instances.instance_id != '' AND instances.agent_type IN ?", memorySupportedTypes).
		Group("current_plan").
		Find(&rows)
	for _, row := range rows {
		if _, ok := planStats[row.CurrentPlan]; ok {
			// 累加而非覆盖：SQLite 等数据库对 GROUP BY 别名的处理方式不同，
			// 可能把 NULL 与 'OFF' 分成两组返回，这里通过累加保证结果正确。
			planStats[row.CurrentPlan] += row.Count
		} else {
			// 兜底：未识别/空的 plan 值（NULL、空字符串、脏数据）归入 OFF
			planStats["OFF"] += row.Count
		}
	}

	// Pro 容量（调 Agent Memory API 获取池信息，used 从本地 plan_stats 取保证口径一致）
	proCapacity := map[string]any{
		"total": 0,
		"used":  planStats["PRO"], // 从本地元数据取，与 plan_stats.PRO 一致
	}
	client, err := NewMemorySDKClient(r.Context())
	if err == nil {
		resp, err := client.DescribeMemoryProInstances(r.Context(), &sdk.DescribeMemoryProInstancesRequest{})
		if err == nil && resp.TotalCount > 0 && len(resp.Items) > 0 {
			inst := resp.Items[0]
			proCapacity["total"] = inst.MemoryLimit
			proCapacity["remote_used"] = inst.MemoryUsed // 远端全池已分配数（含其他站点占用 + 泄露），仅供运维参考；与 used（本站 PRO 数）的差值可辅助发现泄露
			proCapacity["status"] = inst.Status
			proCapacity["memory_pro_id"] = inst.MemoryProId
			proCapacity["vdb_instance_id"] = inst.VDBInstanceId
			proCapacity["vdb_vip"] = inst.VDBVip
			proCapacity["vdb_port"] = inst.VDBPort
		} else if err != nil {
			log.Warn("[MemoryOverview] 查询 Pro 容量失败", "error", err)
		}
	}

	// 默认计划配置
	cfg := model.GetSiteConfig(r.Context())

	jsonOK(w, map[string]any{
		"total_instances":     totalInstances,
		"plan_stats":          planStats,
		"pro_capacity":        proCapacity,
		"memory_default_plan": cfg.MemoryDefaultPlan,
	})
}

// ========== 2. 开通 Pro 服务 ==========

type activateProRequest struct {
	MemoryLimit int `json:"memory_limit"` // 记忆空间上限
}

// HandleAdminMemoryProActivate POST /admin/memory/pro/activate
func HandleAdminMemoryProActivate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPostMethodSupported))
		return
	}

	var req activateProRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if req.MemoryLimit <= 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMemoryLimitMustGTZero))
		return
	}

	log := Logger(r.Context())
	log.Info("[MemoryProActivate] 收到开通请求", "memory_limit", req.MemoryLimit)

	// 检查是否已开通
	client, err := NewMemorySDKClient(r.Context())
	if err != nil {
		log.Error("[MemoryProActivate] 初始化 SDK 失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInitMemorySDKFailed))
		return
	}
	existing, _ := client.DescribeMemoryProInstances(r.Context(), &sdk.DescribeMemoryProInstancesRequest{})
	if existing != nil && existing.TotalCount > 0 {
		log.Warn("[MemoryProActivate] Pro 服务已开通，拒绝重复创建")
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMemoryProAlreadyActivated))
		return
	}

	// 获取网络参数（复用 CVM 默认 VPC/子网）
	cfg := model.GetSiteConfig(r.Context())
	vpcId, subnetId, sgIds, err := resolveNetworkParams(r.Context(), &cfg)
	if err != nil {
		log.Error("[MemoryProActivate] 获取网络参数失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMemoryGetNetworkParamsFail))
		return
	}

	log.Info("[MemoryProActivate] 开始创建 VDB 实例",
		"memory_limit", req.MemoryLimit,
		"vpc_id", vpcId,
		"subnet_id", subnetId,
		"security_groups", sgIds,
	)

	resp, err := client.CreateMemoryProInstance(r.Context(), &sdk.CreateMemoryProInstanceRequest{
		VpcId:            vpcId,
		SubnetId:         subnetId,
		SecurityGroupIds: sgIds,
		MemoryLimit:      req.MemoryLimit,
	})
	if err != nil {
		log.Error("[MemoryProActivate] 创建 VDB 实例失败", "error", err,
			"vpc_id", vpcId, "subnet_id", subnetId)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMemoryCreateVDBFail))
		return
	}

	log.Info("[MemoryProActivate] VDB 实例创建成功",
		"memory_pro_id", resp.MemoryProId,
		"vdb_instance_id", resp.VDBInstanceId,
	)

	jsonOK(w, map[string]any{
		"memory_pro_id":   resp.MemoryProId,
		"vdb_instance_id": resp.VDBInstanceId,
	})
}

// resolveNetworkParams 从 SiteConfig 获取 VPC/子网/安全组。
//
// 与 CVM 创建逻辑对齐（openclaw.go），保证 VPC 和 Subnet 始终来自同一套配置：
//   - 优先级 1：管理员手动配置的 VpcId + SubnetIds（全局 VPC）
//   - 优先级 2：系统自动创建的 DefaultVpcId + DefaultSubnetIds
//
// 修复前的 bug：VPC 优先取 DefaultVpcId，但 Subnet 可能回退到 SubnetIds，
// 导致 VPC 和 Subnet 分属不同网络，CreateMemoryProInstance 报 "SubnetId not found"。
func resolveNetworkParams(ctx context.Context, cfg *model.SiteConfig) (vpcId, subnetId string, sgIds []string, err error) {
	// 优先级 1：管理员手动配置的全局 VPC + 子网
	globalSubnetMap := cfg.GetSubnetMap() // map[string][]string (zone -> []subnetId)
	if cfg.VpcId != "" && len(globalSubnetMap) > 0 {
		vpcId = cfg.VpcId
		for _, sids := range globalSubnetMap {
			if len(sids) > 0 {
				subnetId = sids[0]
				break
			}
		}
	}

	// 优先级 2：系统自动创建的默认 VPC + 子网
	if vpcId == "" || subnetId == "" {
		defaultSubnetMap := cfg.GetDefaultSubnetMap() // map[string][]string (zone -> []subnetId)
		if cfg.DefaultVpcId != "" && len(defaultSubnetMap) > 0 {
			vpcId = cfg.DefaultVpcId
			subnetId = "" // 重置，确保从同一套取
			for _, sids := range defaultSubnetMap {
				if len(sids) > 0 {
					subnetId = sids[0]
					break
				}
			}
		}
	}

	if vpcId == "" {
		return "", "", nil, hcommon.I18nError(i18n.MsgVPCNotConfigured)
	}
	if subnetId == "" {
		return "", "", nil, hcommon.I18nError(i18n.MsgSubnetNotConfigured)
	}

	// 安全组（新模型：从 DefaultRuleSet 的 ACTIVE 池里选）
	// TODO(sg-ruleset-user-group): 该 change 落地后改为按 user.group_id → user_group.rule_set_id
	// 推导目标 RuleSet，而不是硬编码 DefaultRuleSet。
	if true { // model.DB is always available
		if defaultRS, err := model.GetDefaultRuleSet(ctx); err != nil {
			slog.Warn("[MemoryPro] GetDefaultRuleSet failed, sgIds will be empty", "error", err)
		} else if defaultRS != nil {
			if ids, lerr := listActiveSGIDsByRuleSet(ctx, defaultRS.ID); lerr != nil {
				slog.Warn("[MemoryPro] listActiveSGIDsByRuleSet failed, sgIds will be empty", "rule_set_id", defaultRS.ID, "error", lerr)
			} else {
				sgIds = ids
			}
		}
	}

	return vpcId, subnetId, sgIds, nil
}

// ========== 3. 关闭 Pro 服务 ==========

// HandleAdminMemoryProRelease POST /admin/memory/pro/release
// 关闭 Pro 服务（释放 VDB 池）。
// 前置条件：所有龙虾实例的 Pro 记忆库必须已关闭（既不能处于 PRO，也不能残留 pool_id），
// 否则返回 409 错误，提示管理员先在用户端将所有实例切到 OFF 并清空残留绑定信息。
func HandleAdminMemoryProRelease(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPostMethodSupported))
		return
	}

	log := Logger(r.Context())
	log.Info("[MemoryProRelease] 收到关闭 Pro 服务请求")

	client, err := NewMemorySDKClient(r.Context())
	if err != nil {
		log.Error("[MemoryProRelease] 初始化 SDK 失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInitMemorySDKFailed))
		return
	}

	// 查询 Pro 实例
	proResp, err := client.DescribeMemoryProInstances(r.Context(), &sdk.DescribeMemoryProInstancesRequest{})
	if err != nil {
		log.Error("[MemoryProRelease] 查询 Pro 实例失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMemoryQueryProInstanceFail))
		return
	}
	if proResp.TotalCount == 0 {
		log.Warn("[MemoryProRelease] 未找到 Pro 服务实例")
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMemoryProInstanceNotFound))
		return
	}

	// 前置校验：必须所有 Pro 记忆库已关闭（current_plan=PRO 或仍有残留 pool_id 均视为在用）
	var proPlugins []model.MemoryTDAIPlugin
	model.DB(r.Context()).Where("current_plan = ? OR (pool_id IS NOT NULL AND pool_id != '')", model.MemoryPlanPro).Find(&proPlugins)

	if len(proPlugins) > 0 {
		// 收集仍在使用 Pro 的实例 ID（最多显示 10 个）
		inUseIDs := make([]string, 0, len(proPlugins))
		for _, p := range proPlugins {
			inUseIDs = append(inUseIDs, p.InstanceID)
			if len(inUseIDs) >= 10 {
				break
			}
		}
		suffix := ""
		if len(proPlugins) > 10 {
			suffix = i18n.T(r.Context(), i18n.MsgMemoryProMoreSuffixFmt, len(proPlugins))
		}

		log.Warn("[MemoryProRelease] 仍有 Pro 记忆库在用，拒绝释放",
			"in_use_count", len(proPlugins), "sample_ids", inUseIDs)

		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMemoryProInUseRefuseRelease,
			len(proPlugins), strings.Join(inUseIDs, ", "), suffix))
		return
	}

	// 所有 Pro 记忆库已关闭，释放 VDB 实例
	failedReleaseIDs := make([]string, 0)
	for _, inst := range proResp.Items {
		_, err = client.DeleteMemoryProInstance(r.Context(), &sdk.DeleteMemoryProInstanceRequest{
			MemoryProId: inst.MemoryProId,
		})
		if err != nil {
			failedReleaseIDs = append(failedReleaseIDs, inst.MemoryProId)
			log.Warn("[MemoryProRelease] 释放 VDB 实例失败",
				"memory_pro_id", inst.MemoryProId, "error", err)
		} else {
			log.Info("[MemoryProRelease] VDB 实例已释放", "memory_pro_id", inst.MemoryProId)
		}
	}
	if len(failedReleaseIDs) > 0 {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMemoryReleaseProFailed,
			strings.Join(failedReleaseIDs, ", ")))
		return
	}

	// 关闭 Pro 服务后，清空分组策略表（plan=pro 的策略已无法生效）
	result := model.DB(r.Context()).Where("1 = 1").Delete(&model.MemoryPlanGroupPolicy{})
	if result.RowsAffected > 0 {
		log.Info("[MemoryProRelease] 已清空分组策略表", "rows_deleted", result.RowsAffected)
	}

	log.Info("[MemoryProRelease] Pro 服务已关闭")

	jsonOK(w, map[string]any{
		"ok": true,
	})
}

// ========== 4. 切换记忆计划（批量） ==========

type batchSwitchPlanRequest struct {
	InstanceIDs []string `json:"instance_ids"`
	TargetPlan  string   `json:"target_plan"` // off / free / pro
}

// HandleAdminMemoryPlanSwitch POST /admin/memory/plan/switch
func HandleAdminMemoryPlanSwitch(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPostMethodSupported))
		return
	}

	var req batchSwitchPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	desiredPlan, jobType, switchStatus, ok := resolveMemoryPlanTransition(req.TargetPlan)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidTargetPlan, req.TargetPlan))
		return
	}

	operator := ""
	if user, err := RequestUser(r); user != nil && err == nil {
		operator = user.Username
	}

	log := Logger(r.Context())
	log.Info("[AdminMemoryPlanSwitch] 收到批量切换请求",
		"target_plan", req.TargetPlan,
		"instance_count", len(req.InstanceIDs),
		"operator", operator,
	)

	type submitResult struct {
		InstanceID string            `json:"instance_id"`
		Status     string            `json:"status,omitempty"` // accepted / rejected
		TaskID     uint              `json:"task_id,omitempty"`
		Reason     string            `json:"reason,omitempty"`  // 拒绝原因枚举
		Message    string            `json:"message,omitempty"` // 面向用户的提示文案
		Detail     map[string]string `json:"detail,omitempty"`  // 补充信息
		Error      string            `json:"error,omitempty"`   // 兼容老字段
	}
	var results []submitResult

	// ===== 网络连通性前置预检（仅 target_plan=pro） =====
	precheckResults := map[string]MemoryPrecheckResult{}
	if desiredPlan == model.MemoryPlanPro {
		// 收集需要预检的实例：current_plan 非 PRO 且无进行中的切换
		var precheckIDs []string
		for _, instID := range req.InstanceIDs {
			instID = strings.TrimSpace(instID)
			if instID == "" {
				continue
			}
			var plugin model.MemoryTDAIPlugin
			if err := model.DB(r.Context()).Where("instance_id = ?", instID).First(&plugin).Error; err == nil {
				if plugin.CurrentPlan == model.MemoryPlanPro || plugin.SwitchStatus != "" {
					continue // 已是 PRO 或有进行中切换，不预检
				}
			}
			precheckIDs = append(precheckIDs, instID)
		}
		if len(precheckIDs) > 0 {
			precheckResults = PrecheckBatchForProSwitch(r.Context(), precheckIDs)
		}
	}

	for _, instID := range req.InstanceIDs {
		instID = strings.TrimSpace(instID)
		if instID == "" {
			continue
		}

		// 校验 instance_id 对应的实例是否存在，并检查是否支持记忆功能
		var inst model.Instance
		if err := model.DB(r.Context()).Where("instance_id = ?", instID).First(&inst).Error; err != nil {
			log.Warn("[AdminMemoryPlanSwitch] 实例不存在", "instance_id", instID)
			results = append(results, submitResult{
				InstanceID: instID,
				Error:      i18n.T(r.Context(), i18n.MsgInstanceNotFound),
			})
			continue
		}
		if !model.AgentTypeSupportsMemory(r.Context(), inst.AgentType) {
			typeName := model.GetAgentTypeDisplayName(r.Context(), inst.AgentType)
			log.Warn("[AdminMemoryPlanSwitch] 实例类型不支持记忆功能",
				"instance_id", instID, "agent_type", inst.AgentType)
			results = append(results, submitResult{
				InstanceID: instID,
				Error:      i18n.T(r.Context(), i18n.MsgMemoryProAgentTypeNoMemoryFmt, typeName),
			})
			continue
		}

		model.EnsureMemoryTDAIPluginRow(r.Context(), instID)

		// 检查 switch_status
		var plugin model.MemoryTDAIPlugin
		if err := model.DB(r.Context()).Where("instance_id = ?", instID).First(&plugin).Error; err == nil {
			if plugin.SwitchStatus != "" {
				log.Warn("[AdminMemoryPlanSwitch] 有进行中的切换操作",
					"instance_id", instID, "switch_status", plugin.SwitchStatus)
				results = append(results, submitResult{
					InstanceID: instID,
					Error:      i18n.T(r.Context(), i18n.MsgMemPluginSwitchInProgress, plugin.SwitchStatus),
				})
				continue
			}
			// PRO → FREE 不支持，需先切到 OFF
			if plugin.CurrentPlan == model.MemoryPlanPro && desiredPlan == model.MemoryPlanFree {
				log.Warn("[AdminMemoryPlanSwitch] PRO→FREE 不支持，需先切到 OFF",
					"instance_id", instID, "current_plan", plugin.CurrentPlan)
				results = append(results, submitResult{
					InstanceID: instID,
					Error:      i18n.T(r.Context(), i18n.MsgProToFreeNotSupported),
				})
				continue
			}
		}

		// 网络预检结果检查：不通则拒绝创建任务
		if pr, ok := precheckResults[instID]; ok && !pr.Reachable {
			log.Warn("[AdminMemoryPlanSwitch] 网络预检不通过",
				"instance_id", instID, "reason", pr.Reason, "vdb_instance_id", pr.VDBInstanceID)
			results = append(results, submitResult{
				InstanceID: instID,
				Status:     "rejected",
				Reason:     pr.Reason,
				Message:    pr.Message,
				Detail:     map[string]string{"cvm_id": instID, "vdb_instance_id": pr.VDBInstanceID},
				Error:      pr.Message,
			})
			continue
		}

		bizKey := fmt.Sprintf("switch:%s", instID)
		job, err := model.SubmitJob(r.Context(), jobType, bizKey, instID, "{}", operator, "")
		if err != nil {
			log.Warn("[AdminMemoryPlanSwitch] 提交任务失败",
				"instance_id", instID, "job_type", jobType, "error", err)
			results = append(results, submitResult{InstanceID: instID, Error: err.Error()})
			continue
		}

		model.DB(r.Context()).Model(&model.MemoryTDAIPlugin{}).
			Where("instance_id = ?", instID).
			Updates(map[string]any{
				"desired_plan":  desiredPlan,
				"switch_status": switchStatus,
				"last_task_id":  job.ID,
			})

		log.Info("[AdminMemoryPlanSwitch] 任务已提交",
			"instance_id", instID, "job_id", job.ID, "target_plan", req.TargetPlan)
		results = append(results, submitResult{InstanceID: instID, Status: "accepted", TaskID: job.ID})
	}

	// 如果所有实例都被拒绝（零 accepted），用 422 返回包含所有失败 CVM ID 的错误信息，
	// 方便前端通用错误拦截器直接展示完整的 error message。
	hasAccepted := false
	var rejectedIDs []string
	var vdbInfo string // VDB 实例 ID + endpoint，取第一个 rejected 的信息
	for _, r := range results {
		if r.Status == "accepted" {
			hasAccepted = true
			break
		}
		if r.Status == "rejected" {
			rejectedIDs = append(rejectedIDs, r.InstanceID)
			if vdbInfo == "" && r.Message != "" {
				// 直接复用 results[].message 里已经拼好的 VDB 信息
				// message 格式: "...VDB (vdb-xxx, http://x.x.x.x:80)..."
				// 从中提取 "VDB (...)" 部分
				if idx := strings.Index(r.Message, "VDB ("); idx >= 0 {
					end := strings.Index(r.Message[idx:], ")")
					if end > 0 {
						vdbInfo = r.Message[idx : idx+end+1] // "VDB (vdb-xxx, http://x.x.x.x:80)"
					}
				}
			}
		}
	}
	if !hasAccepted && len(rejectedIDs) > 0 {
		if vdbInfo == "" {
			vdbInfo = "VDB"
		}
		writeError(w, r, http.StatusUnprocessableEntity,
			hcommon.I18nError(i18n.MsgMemoryProAllRejectedTip, strings.Join(rejectedIDs, ", "), vdbInfo).
				WithCustomData(map[string]any{
					"target_plan": req.TargetPlan,
					"results":     results,
				}),
		)
		return
	}

	jsonOK(w, map[string]any{
		"target_plan": req.TargetPlan,
		"results":     results,
	})
}

// ========== 5. 龙虾列表 ==========

// HandleAdminMemoryInstances GET /admin/memory/instances?page=&page_size=&keyword=&plan=
func HandleAdminMemoryInstances(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	log := Logger(r.Context())
	log.Debug("[AdminMemoryInstances] 收到列表查询请求",
		"page", r.URL.Query().Get("page"),
		"page_size", r.URL.Query().Get("page_size"),
		"keyword", r.URL.Query().Get("keyword"),
		"plan", r.URL.Query().Get("plan"))

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	keyword := r.URL.Query().Get("keyword")
	planFilter := r.URL.Query().Get("plan")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	// 不传 page/page_size 时默认返回 10 条；传了则按常规分页，单页上限 10000
	if pageStr == "" && pageSizeStr == "" {
		page = 1
		pageSize = 10
	} else {
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 10000 {
			pageSize = 10000
		}
	}

	type instanceRow struct {
		InstanceID     string  `json:"instance_id"`
		InstanceName   string  `json:"instance_name"`
		AgentType      string  `json:"agent_type"`
		CreatorName    string  `json:"creator_name"`
		CurrentPlan    string  `json:"current_plan"`
		SwitchStatus   string  `json:"switch_status"`
		LastSwitchedAt *string `json:"last_switched_at"`
		// 实例网络信息（方便排查 VDB 连通性问题）
		VpcId              string `json:"vpc_id"`
		SubnetId           string `json:"subnet_id"`
		SecurityGroupId    string `json:"security_group_id"`
		AgentVersion       string `json:"agent_version"`
		PluginVersionsJSON string `json:"plugin_versions_json"`
		GroupID            uint   `json:"group_id"`
		GroupFullPath      string `json:"group_full_path"`
		// 记忆插件详情
		DesiredPlan string `json:"desired_plan"`
		LastTaskID  uint   `json:"last_task_id"`
		LastError   string `json:"last_error"`
		PoolID      string `json:"pool_id"`
	}

	memorySupportedTypes := model.GetMemorySupportedAgentTypes(r.Context())

	// 基础查询：使用 DB.Model(&Instance{}) 起头，保证 GORM 回调自动注入
	// identifier + deleted_at IS NULL，与概览接口口径完全一致。
	baseQuery := model.DB(r.Context()).Model(&model.Instance{}).
		Select(`instances.instance_id, 
				instances.name as instance_name,
				instances.agent_type,
				COALESCE(users.username, '') as creator_name,
				COALESCE(memory_tda_iplugins.current_plan, 'OFF') as current_plan,
				COALESCE(memory_tda_iplugins.switch_status, '') as switch_status,
				memory_tda_iplugins.last_switched_at,
				COALESCE(instances.vpc_id, '') as vpc_id,
				COALESCE(instances.subnet_id, '') as subnet_id,
				COALESCE(instances.security_group_id, '') as security_group_id,
				COALESCE(instances.agent_version, '') as agent_version,
				COALESCE(instances.plugin_versions_json, '') as plugin_versions_json,
				COALESCE(instances.group_id, 0) as group_id,
				COALESCE(memory_tda_iplugins.desired_plan, 'OFF') as desired_plan,
				COALESCE(memory_tda_iplugins.last_task_id, 0) as last_task_id,
				COALESCE(memory_tda_iplugins.last_error, '') as last_error,
				COALESCE(memory_tda_iplugins.pool_id, '') as pool_id`).
		Joins("LEFT JOIN memory_tda_iplugins ON memory_tda_iplugins.instance_id = instances.instance_id AND memory_tda_iplugins.deleted_at IS NULL").
		Joins("LEFT JOIN users ON users.id = instances.user_id AND users.deleted_at IS NULL").
		Where("instances.instance_id != ''").
		Where("instances.agent_type IN ?", memorySupportedTypes)

	if keyword != "" {
		kw := "%" + keyword + "%"
		baseQuery = baseQuery.Where("(instances.instance_id LIKE ? OR instances.name LIKE ?)", kw, kw)
	}
	if planFilter != "" {
		baseQuery = baseQuery.Where("COALESCE(memory_tda_iplugins.current_plan, 'OFF') = ?", strings.ToUpper(planFilter))
	}

	var total int64
	// count 只统计行数：memory_tda_iplugins 与 instance 是 1:1（唯一索引 identifier+instance_id），
	// LEFT JOIN 不会放大行数。仅当 planFilter 非空（需按 plugin.current_plan 过滤）时才 JOIN，
	// 否则去掉无意义的 JOIN，避免 nested loop 扫描 memory_tda_iplugins 全表。
	countQuery := model.DB(r.Context()).Model(&model.Instance{}).
		Where("instances.instance_id != ''").
		Where("instances.agent_type IN ?", memorySupportedTypes)
	if keyword != "" {
		kw := "%" + keyword + "%"
		countQuery = countQuery.Where("(instances.instance_id LIKE ? OR instances.name LIKE ?)", kw, kw)
	}
	if planFilter != "" {
		countQuery = countQuery.
			Joins("LEFT JOIN memory_tda_iplugins ON memory_tda_iplugins.instance_id = instances.instance_id AND memory_tda_iplugins.deleted_at IS NULL").
			Where("COALESCE(memory_tda_iplugins.current_plan, 'OFF') = ?", strings.ToUpper(planFilter))
	}
	countQuery.Count(&total)

	var items []instanceRow
	baseQuery.
		Order("instances.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items)

	// 批量回填 GroupFullPath
	if len(items) > 0 {
		ids := make([]uint, 0, len(items))
		for _, it := range items {
			if it.GroupID != 0 {
				ids = append(ids, it.GroupID)
			}
		}
		pathMap := fetchGroupFullPathMap(r.Context(), ids)
		for i := range items {
			if items[i].GroupID != 0 {
				items[i].GroupFullPath = pathMap[items[i].GroupID]
			}
		}
	}

	jsonOK(w, map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     items,
	})
}

// ========== 6. 默认计划配置 ==========

type updateDefaultPlanRequest struct {
	MemoryDefaultPlan string `json:"memory_default_plan"` // off / free / pro
	ClearPolicies     *bool  `json:"clear_policies"`      // 可选，默认 false；显式传 true 时清空分组策略表
}

// HandleAdminMemoryDefaultPlan GET+PUT /admin/memory/default-plan
func HandleAdminMemoryDefaultPlan(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())

	switch r.Method {
	case http.MethodGet:
		cfg := model.GetSiteConfig(r.Context())
		plan := cfg.MemoryDefaultPlan
		if plan == "" {
			// 兼容旧开关
			if cfg.MemoryTDAIEnable {
				plan = "free"
			} else {
				plan = "off"
			}
		}
		jsonOK(w, map[string]any{
			"memory_default_plan": plan,
		})

	case http.MethodPut:
		var req updateDefaultPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
			return
		}
		switch req.MemoryDefaultPlan {
		case "off", "free", "pro":
		default:
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgInvalidTargetPlan, req.MemoryDefaultPlan))
			return
		}

		// 同步更新旧 bool 开关（兼容）
		enableBool := req.MemoryDefaultPlan != "off"
		if err := model.UpdateSiteConfig(r.Context(), map[string]any{
			"memory_default_plan": req.MemoryDefaultPlan,
			"memory_tdai_enable":  enableBool,
		}); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMemoryUpdateDefaultPlanFail))
			return
		}

		// 只在显式传 clear_policies=true 时，清空分组策略表
		if req.ClearPolicies != nil && *req.ClearPolicies {
			result := model.DB(r.Context()).Where("1 = 1").Delete(&model.MemoryPlanGroupPolicy{})
			if result.RowsAffected > 0 {
				log.Info("[AdminMemoryDefaultPlan] 已清空分组策略表", "rows_deleted", result.RowsAffected)
			}
		}

		log.Info("[AdminMemoryDefaultPlan] 默认记忆计划已更新",
			"memory_default_plan", req.MemoryDefaultPlan)

		jsonOK(w, map[string]any{
			"ok":                  true,
			"memory_default_plan": req.MemoryDefaultPlan,
		})

	default:
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMemoryOnlyGetPutMethod))
	}
}

// ========== helper ==========

// NewMemorySDKClient 创建 Agent Memory SDK 客户端。
// 凭证策略：
//   - 若设置了 MEMORY_API_SECRET_ID / MEMORY_API_SECRET_KEY 环境变量，使用独立凭证（测试环境）
//   - 否则走 getCredential()，与 CVM/CLS 保持一致（CVMUin 非空时走 STS）
//
// Endpoint 通过环境变量 MEMORY_API_ENDPOINT 配置，未设置时使用 SDK 默认值（tdai.tencentcloudapi.com）。
// Region 优先级：环境变量 MEMORY_API_REGION > --region 启动参数（CVMRegion）> SDK 默认值（ap-guangzhou）。
func NewMemorySDKClient(ctx context.Context) (*sdk.Client, error) {
	var secretId, secretKey, token, credType string

	// 优先使用独立环境变量（测试环境隔离凭证）
	if envId := os.Getenv("MEMORY_API_SECRET_ID"); envId != "" {
		secretId = envId
		secretKey = os.Getenv("MEMORY_API_SECRET_KEY")
		credType = "env_override"
	} else {
		cred, err := getCredential(ctx)
		if err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgAPIGatewayGetCredFailed)
		}
		secretId = cred.SecretId
		secretKey = cred.SecretKey
		token = cred.Token
		if token != "" {
			credType = "sts_token"
		} else {
			credType = "permanent_aksk"
		}
	}

	sdkCfg := sdk.Config{
		SecretID:  secretId,
		SecretKey: secretKey,
		Token:     token,
	}
	if ep := os.Getenv("MEMORY_API_ENDPOINT"); ep != "" {
		sdkCfg.Endpoint = ep
	}
	// Region 优先级：环境变量 > --region 启动参数（CVMRegion）> SDK 默认值（ap-guangzhou）
	if rg := os.Getenv("MEMORY_API_REGION"); rg != "" {
		sdkCfg.Region = rg
	} else if CVMRegion != "" {
		sdkCfg.Region = CVMRegion
	}
	slog.Debug("[MemorySDK] 创建客户端",
		"cred_type", credType,
		"endpoint", sdkCfg.Endpoint,
		"region", sdkCfg.Region,
		"secret_id_prefix", secretId[:min(8, len(secretId))]+"...",
	)
	return sdk.NewClient(sdkCfg)
}

// ReleaseProMemSpaceForInstance 以严格模式释放指定实例的远端 Pro 记忆库，并返回是否可以安全删除本地 plugin 行。
// 返回值：
//   - true：导出/禁用成功且远端资源已成功释放，或本地无残留 pool_id，可继续删除 memory_tda_iplugins 行
//   - false：导出/禁用失败或远端资源释放失败，本地绑定信息必须保留，便于后续补偿清理
func ReleaseProMemSpaceForInstance(ctx context.Context, instanceID string) bool {
	return releaseProMemSpaceForInstance(ctx, instanceID, true)
}

// ReleaseProMemSpaceForMissingInstance 用于 CVM 已不存在/不可达的场景。
// 此时本地导出大概率已不可能，故跳过导出/禁用步骤，直接尽力释放远端 VDB 库。
func ReleaseProMemSpaceForMissingInstance(ctx context.Context, instanceID string) bool {
	return releaseProMemSpaceForInstance(ctx, instanceID, false)
}

// releaseProMemSpaceForInstance 释放远端 Pro 记忆库，并返回是否可以安全删除本地 plugin 行。
// requireExport=true：必须先完成导出/禁用；false：跳过导出，直接尽力释放远端资源。
func releaseProMemSpaceForInstance(ctx context.Context, instanceID string, requireExport bool) bool {
	log := slog.Default().With("instance_id", instanceID)
	// agent_type 不支持记忆的实例直接放行（不可能持有 Pro 记忆库）
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).First(&inst).Error; err == nil {
		if !model.AgentTypeSupportsMemory(ctx, inst.AgentType) {
			return true
		}
	}

	plugin := model.GetMemoryTDAIPlugin(ctx, instanceID)
	if plugin == nil {
		return true
	}
	if plugin.PoolID == "" {
		return true
	}

	log.Info("[ReleaseProMemSpace] 实例删除前释放 Pro 记忆库",
		"current_plan", plugin.CurrentPlan,
		"pool_id", plugin.PoolID,
		"database", plugin.DatabaseName)

	// Step 1: 当前仍在 PRO 时，优先在 CVM 上完成导出 + 禁用插件。
	// requireExport=true：若导出失败，则中止整个释放流程，避免未备份就删除远端数据。
	// requireExport=false：说明 CVM 已不存在/不可达，跳过导出步骤，直接尽力释放远端记忆库。
	// 若已不在 PRO，但残留了 pool_id，则只释放远端记忆库，不再做本地导出/清配置。
	if plugin.CurrentPlan == model.MemoryPlanPro {
		if requireExport {
			_, backupErr := RunScript(ctx, instanceID, "memory_tdai_disable.sh", 600, "", nil, map[string]string{
				"plugin":           model.DefaultMemoryTDAIPluginName,
				"clear_pro_config": "true",
				"vdb_endpoint":     plugin.Endpoint,
				"vdb_database":     plugin.DatabaseName,
				"vdb_api_key":      plugin.ApiKeySecretRef,
				"vdb_username":     plugin.VdbUsername,
				"job_id":           "",
			})
			if backupErr != nil {
				log.Warn("[ReleaseProMemSpace] 导出/禁用脚本执行失败，保留远端记忆库和本地绑定信息",
					"error", backupErr)
				return false
			}
			log.Info("[ReleaseProMemSpace] 导出/禁用脚本执行成功")
		} else {
			log.Warn("[ReleaseProMemSpace] CVM 已不存在或不可达，跳过导出/禁用，直接释放远端记忆库",
				"pool_id", plugin.PoolID)
		}
	} else {
		log.Warn("[ReleaseProMemSpace] 检测到残留 Pro 绑定信息，跳过本地导出/禁用，仅释放远端记忆库",
			"current_plan", plugin.CurrentPlan,
			"pool_id", plugin.PoolID)
	}

	// Step 2: 释放远端 VDB 库
	client, err := NewMemorySDKClient(ctx)
	if err != nil {
		log.Warn("[ReleaseProMemSpace] 初始化 SDK 失败，保留本地绑定信息以便后续清理",
			"error", err)
		return false
	}

	_, err = client.DeleteMemSpace(ctx, &sdk.DeleteMemSpaceRequest{
		SpaceId: plugin.PoolID,
	})
	if err != nil {
		// 远端释放失败：保留本地绑定信息，避免"泄漏但本地无记录"
		log.Warn("[ReleaseProMemSpace] 释放远端记忆库失败，保留本地绑定信息以便后续清理",
			"pool_id", plugin.PoolID, "error", err)
		return false
	}

	log.Info("[ReleaseProMemSpace] 远端记忆库已释放",
		"pool_id", plugin.PoolID)

	// Step 3: 只在远端释放成功后才清理 DB 绑定信息
	result := model.DB(ctx).Model(plugin).Updates(map[string]any{
		"current_plan":       model.MemoryPlanOff,
		"desired_plan":       model.MemoryPlanOff,
		"switch_status":      "",
		"pool_id":            "",
		"database_name":      "",
		"endpoint":           "",
		"api_key_secret_ref": "",
		"vdb_username":       "",
		"embedding_model":    "",
	})
	if result.Error != nil {
		log.Error("[ReleaseProMemSpace] 清理 DB 绑定信息失败",
			"error", result.Error)
	} else {
		log.Info("[ReleaseProMemSpace] DB 绑定信息已清理",
			"rows_affected", result.RowsAffected)
	}
	return true
}

// resubmitProSwitchAfterReinstall 重装实例后，若实例当前是 Pro 模式（有完整绑定信息），
// 自动提交一个 SWITCH_TO_PRO 任务，重新下发 VDB 配置到记忆插件。
// 由于绑定信息保留在 DB 中，handleSwitchToPro 走幂等复用分支，直接下发配置而不重新创建记忆库。
func resubmitProSwitchAfterReinstall(ctx context.Context, instanceID string) {
	log := slog.Default().With("instance_id", instanceID)
	plugin := model.GetMemoryTDAIPlugin(ctx, instanceID)
	if plugin == nil || plugin.CurrentPlan != model.MemoryPlanPro || plugin.PoolID == "" {
		return
	}

	// 若有进行中的切换任务，不重复提交
	if plugin.SwitchStatus != "" {
		log.Info("[ReinstallProResubmit] 有进行中的切换任务，跳过",
			"switch_status", plugin.SwitchStatus)
		return
	}

	bizKey := fmt.Sprintf("switch:%s", instanceID)
	job, err := model.SubmitJob(ctx, model.TdaiJobTypeSwitchToPro, bizKey, instanceID, "{}", "system:reinstall_pro_resubmit", "")
	if err != nil {
		log.Warn("[ReinstallProResubmit] 提交 Pro 配置重新下发任务失败（不阻塞重装）",
			"error", err)
		return
	}

	model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).
		Where("instance_id = ?", instanceID).
		Updates(map[string]any{
			"switch_status": model.MemorySwitchStatusSwitchingToPro,
			"last_task_id":  job.ID,
		})

	log.Info("[ReinstallProResubmit] 已提交 Pro 配置重新下发任务",
		"job_id", job.ID)
}
