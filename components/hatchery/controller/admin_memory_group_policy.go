package controller

import (
	"encoding/json"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// HandleAdminMemoryGroupPolicies GET /admin/memory/group-policies
// 查询所有分组策略（联表过滤已删除的分组）。
func HandleAdminMemoryGroupPolicies(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 联表查询策略（过滤已删除的分组）
	type policyRow struct {
		GroupID  uint   `json:"group_id"`
		Plan     string `json:"plan"`
		Priority int    `json:"priority"`
		// 来自 user_groups 表
		GroupName string `json:"group_name"`
		FullPath  string `json:"full_path"`
	}

	var rows []policyRow
	model.DB(r.Context()).
		Table("memory_plan_group_policies p").
		Select("p.group_id, p.plan, p.priority, g.name AS group_name, g.full_path").
		Joins("INNER JOIN user_groups g ON g.id = p.group_id").
		Order("p.priority ASC, p.id ASC").
		Scan(&rows)

	// 按 priority 分组
	type groupInfo struct {
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
		FullPath  string `json:"full_path"`
	}
	type policyResp struct {
		Priority int         `json:"priority"`
		Plan     string      `json:"plan"`
		Groups   []groupInfo `json:"groups"`
	}

	policiesMap := map[int]*policyResp{}
	for _, row := range rows {
		if _, ok := policiesMap[row.Priority]; !ok {
			policiesMap[row.Priority] = &policyResp{
				Priority: row.Priority,
				Plan:     row.Plan,
				Groups:   []groupInfo{},
			}
		}
		policiesMap[row.Priority].Groups = append(policiesMap[row.Priority].Groups, groupInfo{
			GroupID:   row.GroupID,
			GroupName: row.GroupName,
			FullPath:  row.FullPath,
		})
	}

	// 按 priority 顺序输出
	policies := []policyResp{}
	for _, p := range []int{1, 2} {
		if pr, ok := policiesMap[p]; ok {
			policies = append(policies, *pr)
		}
	}

	jsonOK(w, map[string]any{
		"ok":       true,
		"policies": policies,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// POST + PUT /admin/memory/group-policy
// ─────────────────────────────────────────────────────────────────────────────

type createGroupPolicyRequest struct {
	Priority int    `json:"priority"`
	Plan     string `json:"plan"`
	GroupIDs []uint `json:"group_ids"`
}

type updateGroupPolicyRequest struct {
	Priority int    `json:"priority"`
	Plan     string `json:"plan"`
	GroupIDs []uint `json:"group_ids"`
}

type deleteGroupPolicyRequest struct {
	Priority int `json:"priority"`
}

// HandleAdminMemoryGroupPolicy POST+PUT /admin/memory/group-policy
func HandleAdminMemoryGroupPolicy(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	switch r.Method {
	case http.MethodPost:
		handleCreateGroupPolicy(w, r)
	case http.MethodPut:
		handleUpdateGroupPolicy(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAdminMemoryGroupPolicyDelete POST /admin/memory/group-policy/delete
func HandleAdminMemoryGroupPolicyDelete(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req deleteGroupPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestWithError, err))
		return
	}

	if req.Priority != 1 && req.Priority != 2 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPriorityMustBe1Or2))
		return
	}

	log := Logger(r.Context())

	// 删除前先记录旧数据（用于审计/恢复）
	var oldPolicies []model.MemoryPlanGroupPolicy
	model.DB(r.Context()).Where("priority = ?", req.Priority).Find(&oldPolicies)
	oldGroupIDs := make([]uint, len(oldPolicies))
	oldPlan := ""
	for i, p := range oldPolicies {
		oldGroupIDs[i] = p.GroupID
		oldPlan = p.Plan
	}

	log.Info("[GroupPolicy:Delete] 删除分组策略",
		"priority", req.Priority,
		"old_plan", oldPlan,
		"old_group_ids", oldGroupIDs)

	// 删除该 priority 的所有行
	model.DB(r.Context()).Where("priority = ?", req.Priority).Delete(&model.MemoryPlanGroupPolicy{})

	// 如果删的是 priority=1 且 priority=2 还有数据，降级为 1
	if req.Priority == 1 {
		result := model.DB(r.Context()).Model(&model.MemoryPlanGroupPolicy{}).
			Where("priority = ?", 2).
			Update("priority", 1)
		if result.RowsAffected > 0 {
			log.Info("[GroupPolicy:Delete] priority=2 已降级为 priority=1")
		}
	}

	jsonOK(w, map[string]any{"ok": true})
}

// ─────────────────────────────────────────────────────────────────────────────

func handleCreateGroupPolicy(w http.ResponseWriter, r *http.Request) {
	var req createGroupPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestWithError, err))
		return
	}

	// 1. 校验 plan
	if req.Plan != "off" && req.Plan != "free" && req.Plan != "pro" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPlanMustBeOffFreePro))
		return
	}
	cfg := model.GetSiteConfig(r.Context())
	defaultPlan := cfg.MemoryDefaultPlan
	if defaultPlan == "" {
		if cfg.MemoryTDAIEnable {
			defaultPlan = "free"
		} else {
			defaultPlan = "off"
		}
	}
	if req.Plan == defaultPlan {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPlanSameAsDefault, defaultPlan))
		return
	}

	// 2. 校验 priority
	if req.Priority != 1 && req.Priority != 2 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPriorityMustBe1Or2))
		return
	}

	// 3. 校验该 priority 下尚无数据
	var existCount int64
	model.DB(r.Context()).Model(&model.MemoryPlanGroupPolicy{}).Where("priority = ?", req.Priority).Count(&existCount)
	if existCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgGroupPolicyPriorityAlreadyExists, req.Priority))
		return
	}

	// 4. 校验 group_ids 非空
	if len(req.GroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyGroupIDsCannotBeEmpty))
		return
	}

	// 校验分组存在
	var groupCount int64
	model.DB(r.Context()).Model(&model.UserGroup{}).Where("id IN ?", req.GroupIDs).Count(&groupCount)
	if groupCount != int64(len(req.GroupIDs)) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPartialGroupIDNotExist))
		return
	}

	// 5. 批量 INSERT（UNIQUE 冲突由数据库兜底）
	policies := make([]model.MemoryPlanGroupPolicy, len(req.GroupIDs))
	for i, gid := range req.GroupIDs {
		policies[i] = model.MemoryPlanGroupPolicy{
			GroupID:  gid,
			Plan:     req.Plan,
			Priority: req.Priority,
		}
	}

	if err := model.DB(r.Context()).Create(&policies).Error; err != nil {
		// UNIQUE 冲突
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgGroupPolicyPartialGroupOccupied).WithDetail(err.Error()))
		return
	}

	log := Logger(r.Context())
	log.Info("[GroupPolicy:Create] 创建分组策略成功",
		"priority", req.Priority,
		"plan", req.Plan,
		"group_ids", req.GroupIDs)

	jsonOK(w, map[string]any{"ok": true})
}

func handleUpdateGroupPolicy(w http.ResponseWriter, r *http.Request) {
	var req updateGroupPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestWithError, err))
		return
	}

	// 1. 校验 priority
	if req.Priority != 1 && req.Priority != 2 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPriorityMustBe1Or2))
		return
	}

	// 校验该 priority 存在
	var existCount int64
	model.DB(r.Context()).Model(&model.MemoryPlanGroupPolicy{}).Where("priority = ?", req.Priority).Count(&existCount)
	if existCount == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgGroupPolicyPriorityNotExist, req.Priority))
		return
	}

	// 2. 校验 plan
	if req.Plan != "off" && req.Plan != "free" && req.Plan != "pro" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPlanMustBeOffFreePro))
		return
	}
	cfg := model.GetSiteConfig(r.Context())
	defaultPlan := cfg.MemoryDefaultPlan
	if defaultPlan == "" {
		if cfg.MemoryTDAIEnable {
			defaultPlan = "free"
		} else {
			defaultPlan = "off"
		}
	}
	if req.Plan == defaultPlan {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPlanSameAsDefault, defaultPlan))
		return
	}

	// 校验不与另一条策略的 plan 冲突
	otherPriority := 1
	if req.Priority == 1 {
		otherPriority = 2
	}
	var otherPolicy model.MemoryPlanGroupPolicy
	if err := model.DB(r.Context()).Where("priority = ?", otherPriority).First(&otherPolicy).Error; err == nil {
		if otherPolicy.Plan == req.Plan {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPlanUsedByOther, req.Plan, otherPriority))
			return
		}
	}

	// 3. 校验 group_ids 非空
	if len(req.GroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyGroupIDsCannotBeEmpty))
		return
	}

	// 校验分组存在
	var groupCount int64
	model.DB(r.Context()).Model(&model.UserGroup{}).Where("id IN ?", req.GroupIDs).Count(&groupCount)
	if groupCount != int64(len(req.GroupIDs)) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupPolicyPartialGroupIDNotExist))
		return
	}

	// 记录旧数据（用于审计/恢复）
	var oldPolicies []model.MemoryPlanGroupPolicy
	model.DB(r.Context()).Where("priority = ?", req.Priority).Find(&oldPolicies)
	oldGroupIDs := make([]uint, len(oldPolicies))
	oldPlan := ""
	for i, p := range oldPolicies {
		oldGroupIDs[i] = p.GroupID
		oldPlan = p.Plan
	}

	// 4. 事务内全量替换
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 删除该 priority 的旧数据
		if err := tx.Where("priority = ?", req.Priority).Delete(&model.MemoryPlanGroupPolicy{}).Error; err != nil {
			return err
		}
		// 插入新数据
		policies := make([]model.MemoryPlanGroupPolicy, len(req.GroupIDs))
		for i, gid := range req.GroupIDs {
			policies[i] = model.MemoryPlanGroupPolicy{
				GroupID:  gid,
				Plan:     req.Plan,
				Priority: req.Priority,
			}
		}
		return tx.Create(&policies).Error
	})

	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgGroupPolicyUpdateFailedMaybeOccupied).WithDetail(err.Error()))
		return
	}

	log := Logger(r.Context())
	log.Info("[GroupPolicy:Update] 修改分组策略成功",
		"priority", req.Priority,
		"old_plan", oldPlan,
		"old_group_ids", oldGroupIDs,
		"new_plan", req.Plan,
		"new_group_ids", req.GroupIDs)

	jsonOK(w, map[string]any{"ok": true})
}
