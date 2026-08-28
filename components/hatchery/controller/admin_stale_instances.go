package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// admin_stale_instances.go — 管理端 3 个端点
//
//   POST /admin/stale-instances/config-diff   配置差异（弹窗"查看配置差异"）
//   POST /admin/stale-instances/apply         应用处理（混合 4 种 action）
//   GET  /admin/stale-instances/records       处理记录列表

// staleConfigDiffMaxBatch 单次 config-diff 请求允许的实例数上限。
// 与 adminDeleteMaxBatch 保持一致；超过则返回 400。
// 主要原因：每个实例在响应中独立产出 instance_configs[] 一项，加上
// target_config 仅构建一次但每实例仍需走 buildInstanceCategoriesView 等逻辑，
// 防止上限过大时拖垮 DB / 序列化。
const staleConfigDiffMaxBatch = 100

// staleApplyMaxBatch 单次 apply 请求允许的实例数总和上限（actions 中所有
// instance_ids 长度求和）。500 是经验值：apply 主要瓶颈在 per-instance 的
// detectScenario / writeICGRTx 事务，比 config-diff 更重，但 admin 端批量
// 处理 stale 实例的常见场景（OneID 同步、批量分组调整）一次几百条很正常。
const staleApplyMaxBatch = 500

// HandleAdminStaleInstancesConfigDiff 配置差异接口。
func HandleAdminStaleInstancesConfigDiff(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req configDiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_ids"))
		return
	}
	if len(req.InstanceIDs) > staleConfigDiffMaxBatch {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, staleConfigDiffMaxBatch))
		return
	}
	if req.TargetGroupID == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "target_group_id"))
		return
	}
	targetGroupID := *req.TargetGroupID

	var instances []model.Instance
	if err := model.DB(r.Context()).Where("id IN ?", req.InstanceIDs).Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	// 按请求顺序保留
	instMap := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		instMap[instances[i].ID] = &instances[i]
	}

	siteConfig := model.GetSiteConfig(r.Context())
	// 目标分组视图只构建一次，所有实例共享。
	targetCats := buildCategoriesForView(r.Context(), targetGroupID, &siteConfig)
	targetConfig := buildTargetConfig(r.Context(), targetCats)

	// 把所有实例用到的 AIModelID / RoleID 一次性批量查回（最多 2 条 SQL）。
	orderedInsts := make([]*model.Instance, 0, len(req.InstanceIDs))
	for _, id := range req.InstanceIDs {
		if inst := instMap[id]; inst != nil {
			orderedInsts = append(orderedInsts, inst)
		}
	}
	lookups := loadInstanceLookups(r.Context(), orderedInsts)

	// T28：一次性批量拉 CVM 信息（含公网三字段），单批 100 内符合上限。
	// 空值/API 失败自动降级为 map 缺失，computeConfigDiff 会跳过公网子行的实例侧填充。
	cvmIDs := make([]string, 0, len(orderedInsts))
	for _, inst := range orderedInsts {
		if inst.InstanceId != "" {
			cvmIDs = append(cvmIDs, inst.InstanceId)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	instanceIDs := make([]uint, 0, len(orderedInsts))
	for _, inst := range orderedInsts {
		instanceIDs = append(instanceIDs, inst.ID)
	}
	memoryPluginMap := batchFetchMemoryPluginMap(r.Context(), cvmIDs)
	driveSpaceMap := batchFetchDriveSpaceMap(r.Context(), instanceIDs)
	cwpSecurityMap := batchFetchCWPSecurityMap(r.Context(), cvmIDs)

	instanceConfigs := make([]configDiffPerInstance, 0, len(req.InstanceIDs))
	notFound := make([]uint, 0)
	for _, id := range req.InstanceIDs {
		inst := instMap[id]
		if inst == nil {
			notFound = append(notFound, id)
			continue
		}
		instanceConfigs = append(instanceConfigs, computeConfigDiff(r.Context(), inst, targetCats, lookups, cvmInfoMap[inst.InstanceId], &siteConfig, nil, memoryPluginMap, driveSpaceMap, cwpSecurityMap))
	}

	jsonOK(w, map[string]interface{}{
		"ok":                     true,
		"target_group_id":        targetGroupID,
		"target_group_path":      groupFullPath(r.Context(), targetGroupID),
		"target_config":          targetConfig,
		"instance_configs":       instanceConfigs,
		"not_found_instance_ids": notFound,
	})
}

// HandleAdminStaleInstancesApply 应用处理接口（混合 action）。
func HandleAdminStaleInstancesApply(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req applyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if !validTriggerSource(req.TriggerSource) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "trigger_source"))
		return
	}
	if len(req.Actions) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "actions"))
		return
	}

	// 校验各 action：枚举值 + instance_ids 非空 + 必填 target 字段。
	// fail-fast：任一 action 不合法立即 400，避免后续每实例都 marked failed。
	for i, item := range req.Actions {
		if !validStaleAction(item.Action) {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamFormatError, fmt.Sprintf("actions[%d].action", i)))
			return
		}
		if len(item.InstanceIDs) == 0 {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamRequired, fmt.Sprintf("actions[%d].instance_ids", i)))
			return
		}
		// migrate 必须显式给 target_group_id（哪怕是 0，对应场景 B）；
		// 用 *uint 区分"未传"和"显式 0"，未传 → 400。
		if item.Action == StaleActionMigrate && item.TargetGroupID == nil {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamRequired, fmt.Sprintf("actions[%d].target_group_id", i)))
			return
		}
		// handover 必须给非 0 的 target_user_id（0 在 handover 场景永远非法）。
		if item.Action == StaleActionHandover && item.TargetUserID == 0 {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamRequired, fmt.Sprintf("actions[%d].target_user_id", i)))
			return
		}
	}

	// 校验 instance_ids 总数上限（actions 跨条目求和），防止单次请求拖垮 DB。
	totalInstanceIDs := 0
	for _, item := range req.Actions {
		totalInstanceIDs += len(item.InstanceIDs)
	}
	if totalInstanceIDs > staleApplyMaxBatch {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, staleApplyMaxBatch))
		return
	}

	// 批量预查 target_group_id / target_user_id 存在性（最多 2 条 IN 查询）。
	// 给 applyEngine 传 map，使其在 detectScenario 之前就能拒绝引用了不存在
	// 资源的请求，避免被 noop 早返回吞错。
	groupIDSet := make(map[uint]struct{})
	userIDSet := make(map[uint]struct{})
	for _, item := range req.Actions {
		// migrate 与 handover 都可能携带 target_group_id（migrate 是迁移目标；
		// handover 是从目标 user 的多分组里指定一个）。任一非零都需校验存在。
		gid := item.targetGroupIDValue()
		if gid > 0 {
			groupIDSet[gid] = struct{}{}
		}
		if item.Action == StaleActionHandover && item.TargetUserID > 0 {
			userIDSet[item.TargetUserID] = struct{}{}
		}
	}
	existingGroupIDs := loadExistingGroupIDs(r.Context(), groupIDSet)
	existingUserIDs := loadExistingUserIDs(r.Context(), userIDSet)

	// 取当前管理员 ID 作为 actor_id
	actorID := uint(0)
	if u, _ := getLoginUser(r); u != nil {
		actorID = u.ID
	}

	engine := newApplyEngine(r.Context(), actorID, req.TriggerSource, existingGroupIDs, existingUserIDs)
	engine.run(req.Actions)

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"results": engine.results,
	})
}

// HandleAdminStaleInstancesRecords 处理记录列表查询。
func HandleAdminStaleInstancesRecords(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	q := r.URL.Query()
	params := model.ListICGRsParams{
		Action:        q.Get("action"),
		ActorType:     q.Get("actor_type"),
		TriggerSource: q.Get("trigger_source"),
	}
	if v := q.Get("instance_id"); v != "" {
		// 同时支持 instance_pk（数字）与 CVM ins-xxx 字符串
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			params.InstancePK = uint(id)
		} else {
			params.InstanceID = v
		}
	}
	if v := q.Get("user_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			params.UserID = uint(id)
		}
	}
	if v := q.Get("group_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			gid := uint(id)
			params.GroupID = &gid
		}
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.To = &t
		}
	}
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Page = n
		}
	}
	if v := q.Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.PageSize = n
		}
	}
	// 分页参数兜底：未传 / 非法值时使用合理默认。
	// 与 model.normalizePagination 内部行为对齐：page≥1, 1≤page_size≤100。
	// 在 handler 层显式做一遍，确保响应里回显的是真正生效的值，
	// 而不是请求里的原始 0。
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	} else if params.PageSize > 100 {
		params.PageSize = 100
	}

	rows, total, err := model.ListICGRs(r.Context(), params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":       true,
		"total":    total,
		"page":     params.Page,
		"page_size": params.PageSize,
		"records":  rows,
	})
}

// loadExistingGroupIDs 批量校验 user_groups 中存在的 ID。
// idSet 为空 → 立即返回空 map，不查 DB。结果 map 仅含真实存在的 ID。
func loadExistingGroupIDs(ctx context.Context, idSet map[uint]struct{}) map[uint]bool {
	out := make(map[uint]bool, len(idSet))
	if len(idSet) == 0 {
		return out
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var rows []struct {
		ID uint `gorm:"column:id"`
	}
	_ = model.DB(ctx).Model(&model.UserGroup{}).Select("id").Where("id IN ?", ids).Find(&rows).Error
	for _, r := range rows {
		out[r.ID] = true
	}
	return out
}

// loadExistingUserIDs 批量校验 users 中存在的 ID（含未软删）。
func loadExistingUserIDs(ctx context.Context, idSet map[uint]struct{}) map[uint]bool {
	out := make(map[uint]bool, len(idSet))
	if len(idSet) == 0 {
		return out
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var rows []struct {
		ID uint `gorm:"column:id"`
	}
	_ = model.DB(ctx).Model(&model.User{}).Select("id").Where("id IN ?", ids).Find(&rows).Error
	for _, r := range rows {
		out[r.ID] = true
	}
	return out
}
