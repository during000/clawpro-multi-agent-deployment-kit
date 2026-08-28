package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// HandleAdminSMHConfigAPI returns the current SMH configuration as JSON.
//
// GET /admin/smh/config
func HandleAdminSMHConfigAPI(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	smhConfig := model.GetSMHConfig(r.Context())
	siteConfig := model.GetSiteConfig(r.Context())

	// 密钥脱敏：仅显示前4位和后4位
	maskedSecret := siteConfig.SMHLibrarySecret
	if len(maskedSecret) > 8 {
		maskedSecret = maskedSecret[:4] + "****" + maskedSecret[len(maskedSecret)-4:]
	}

	jsonOK(w, map[string]interface{}{
		"smh_enabled":                  siteConfig.SMHEnabled,
		"smh_library_id":               siteConfig.SMHLibraryId,
		"smh_library_secret":           maskedSecret,
		"smh_endpoint":                 siteConfig.SMHEndpoint,
		"smh_common_space":             smhConfig.CommonSpace,
		"smh_skillhub_space":           smhConfig.SkillhubSpace,
		"smh_auto_provision_on_create": siteConfig.SMHAutoProvisionOnCreate,
		"is_configured":                smhConfig.IsConfigured(),
		"provision_error":              siteConfig.SMHProvisionError,
	})
}

// HandleUpdateSMHConfig handles manual SMH configuration updates.
//
// POST /admin/config/smh
//
// Supported fields (all optional, omitted fields are not modified):
//   - smh_library_id:              Library ID
//   - smh_library_secret:          Library secret (sensitive field)
//   - smh_endpoint:                Library access domain
//   - smh_enabled:                 Service switch ("0" or "1")
//   - smh_auto_provision_on_create: Auto-create space on instance creation ("0" or "1")
//
// Permission: requireAdmin (admin role or admin-token only)
func HandleUpdateSMHConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	r.ParseForm()

	updates := map[string]interface{}{}
	if r.Form.Has("smh_library_id") {
		updates["smh_library_id"] = r.FormValue("smh_library_id")
	}
	if r.Form.Has("smh_library_secret") {
		updates["smh_library_secret"] = r.FormValue("smh_library_secret")
	}
	if r.Form.Has("smh_endpoint") {
		updates["smh_endpoint"] = r.FormValue("smh_endpoint")
	}
	if v := r.FormValue("smh_enabled"); v == "0" || v == "1" {
		val, _ := strconv.Atoi(v)
		updates["smh_enabled"] = val
	}
	if v := r.FormValue("smh_auto_provision_on_create"); v == "0" || v == "1" {
		updates["smh_auto_provision_on_create"] = v == "1"
	}

	if len(updates) > 0 {
		if err := model.UpdateSiteConfig(r.Context(), updates); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateSiteConfigFailed))
			return
		}
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminSMHAutoProvision 设置创建实例时是否自动创建个人空间。
// POST /admin/smh/auto-provision
//
// 参数：
//   - enabled: "0" 或 "1"
func HandleAdminSMHAutoProvision(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	r.ParseForm()
	v := r.FormValue("enabled")
	if v != "0" && v != "1" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnabledMustBe01))
		return
	}

	enabled := v == "1"
	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{"smh_auto_provision_on_create": enabled}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateConfigFailed))
		return
	}

	slog.Info("[SMH] 自动创建个人空间配置已更新", "enabled", enabled)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminPersonalSpaces 查询个人空间列表（分页）
// GET /admin/smh/personal-spaces
func HandleAdminPersonalSpaces(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	q := r.URL.Query()

	query := model.DB(r.Context()).Model(&model.SMHPersonalSpace{})

	// 按用户 ID 过滤
	if userID := q.Get("user"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// 仅回收站（实例已删除）
	if q.Get("instance_deleted_only") == "1" {
		query = query.Where("to_be_deleted_at IS NOT NULL")
	}

	// 仅激活（实例未删除）
	if q.Get("instance_not_deleted_only") == "1" {
		query = query.Where("to_be_deleted_at IS NULL")
	}

	var total int64
	query.Count(&total)

	var spaces []model.SMHPersonalSpace
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&spaces)

	// 批量查询空间用量
	var spaceIds []string
	for _, s := range spaces {
		if s.SpaceId != "" {
			spaceIds = append(spaceIds, s.SpaceId)
		}
	}
	usageMap := fetchPersonalSpaceUsage(r.Context(), spaceIds)

	items := make([]map[string]interface{}, len(spaces))
	for i, s := range spaces {
		items[i] = map[string]interface{}{
			"id":                 s.ID,
			"space_id":           s.SpaceId,
			"user_id":            s.UserId,
			"username":           s.UserName,
			"instance_id":        s.InstanceId,
			"instance_name":      s.InstanceName,
			"cvm_instance_id":    s.CVMInstanceId,
			"storage_quota":      s.StorageQuota,
			"free_storage_quota": s.FreeStorageQuota,
			"used_storage":       usageMap[s.SpaceId].size,
			"bound_at":           s.CreatedAt,
			"expires_at":         s.ExpiresAt,
			"instance_deleted":   s.ToBeDeletedAt != nil, // TODO: 应该是实例解绑，而不是实例删除
			"to_be_deleted_at":   s.ToBeDeletedAt,
		}
	}

	jsonOK(w, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// HandleAdminSMHInstances 查询实例及个人空间状态列表（分页）
// GET /admin/smh/instances
func HandleAdminSMHInstances(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)

	type instanceWithSpace struct {
		InstanceDbID     uint       `gorm:"column:id"`
		InstanceName     string     `gorm:"column:name"`
		InstanceId       string     `gorm:"column:instance_id"`
		AgentType        string     `gorm:"column:agent_type"`
		UserID           uint       `gorm:"column:user_id"`
		GroupID          uint       `gorm:"column:group_id"`
		Username         string     `gorm:"column:username"`
		SpaceDbID        *uint      `gorm:"column:space_id_pk"`
		SpaceId          *string    `gorm:"column:space_id"`
		StorageQuota     *int64     `gorm:"column:storage_quota"`
		FreeStorageQuota *int64     `gorm:"column:free_storage_quota"`
		SpaceCreatedAt   *time.Time `gorm:"column:space_created_at"`
		ExpiresAt        *time.Time `gorm:"column:expires_at"`
		ToBeDeletedAt    *time.Time `gorm:"column:to_be_deleted_at"`
	}

	baseQuery := model.DB(r.Context()).Model(&model.Instance{}).
		Select(`instances.id, instances.name, instances.instance_id, instances.agent_type, instances.user_id, instances.group_id,
			u.username,
			s.id as space_id_pk, s.space_id,
			s.storage_quota, s.free_storage_quota,
			s.created_at as space_created_at, s.expires_at, s.to_be_deleted_at`).
		Joins("LEFT JOIN smh_personal_spaces s ON s.instance_id = instances.id AND s.deleted_at IS NULL AND s.identifier = instances.identifier").
		Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL AND u.identifier = instances.identifier").
		Where("instances.agent_type IN ?", model.GetSMHSupportedAgentTypes(r.Context()))

	if userID := r.URL.Query().Get("user"); userID != "" {
		baseQuery = baseQuery.Where("instances.user_id = ?", userID)
	}
	if agentType := r.URL.Query().Get("agent_type"); agentType != "" {
		baseQuery = baseQuery.Where("instances.agent_type = ?", agentType)
	}
	if groupID := r.URL.Query().Get("group_id"); groupID != "" {
		baseQuery = baseQuery.Where("instances.group_id = ?", groupID)
	}

	// exclude_recycling=1 时过滤掉 space 在回收站中的实例
	if r.URL.Query().Get("exclude_recycling") == "1" {
		baseQuery = baseQuery.Where("s.to_be_deleted_at IS NULL")
	}

	var total int64
	baseQuery.Session(&gorm.Session{}).Count(&total)

	var rows []instanceWithSpace
	baseQuery.Order("instances.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	// 批量查询有空间的用量
	var spaceIds []string
	for _, row := range rows {
		if row.SpaceId != nil && *row.SpaceId != "" {
			spaceIds = append(spaceIds, *row.SpaceId)
		}
	}
	usageMap := fetchPersonalSpaceUsage(r.Context(), spaceIds)

	// 批量查询 GroupID → full_path
	var groupIDs []uint
	for _, row := range rows {
		if row.GroupID != 0 {
			groupIDs = append(groupIDs, row.GroupID)
		}
	}
	groupPathMap := fetchGroupFullPathMap(r.Context(), groupIDs)

	items := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		status := "none"
		if row.SpaceDbID != nil {
			if row.ToBeDeletedAt != nil {
				status = "recycling"
			} else {
				status = "active"
			}
		}

		agentType := model.NormalizeAgentType(row.AgentType)

		item := map[string]interface{}{
			"instance_id":     row.InstanceDbID,
			"instance_name":   row.InstanceName,
			"cvm_instance_id": row.InstanceId,
			"agent_type":      agentType,
			"user_id":         row.UserID,
			"username":        row.Username,
			"group_id":        row.GroupID,
			"group_full_path": groupPathMap[row.GroupID],
			"space_status":    status,
		}

		if row.SpaceDbID != nil {
			spaceId := ""
			if row.SpaceId != nil {
				spaceId = *row.SpaceId
			}
			item["space_id"] = *row.SpaceDbID
			item["smh_space_id"] = spaceId
			item["storage_quota"] = row.StorageQuota
			item["free_storage_quota"] = row.FreeStorageQuota
			item["used_storage"] = usageMap[spaceId].size
			item["bound_at"] = row.SpaceCreatedAt
			item["expires_at"] = row.ExpiresAt
			item["to_be_deleted_at"] = row.ToBeDeletedAt
		}

		items[i] = item
	}

	jsonOK(w, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// HandleAdminPersonalSpaceToken 获取指定个人空间的临时访问 Token
// GET /admin/smh/personal-spaces/token?id=xxx
func HandleAdminPersonalSpaceToken(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var space model.SMHPersonalSpace
	if model.DB(r.Context()).First(&space, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPersonalSpaceNotFound))
		return
	}

	smhConfig := model.GetSMHConfig(r.Context())

	accessToken, expiresAt, _, err := ensurePersonalSpaceToken(r.Context(), space.SpaceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgGetSMHTokenFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"token":      accessToken,
		"space_id":   space.SpaceId,
		"library_id": smhConfig.LibraryId,
		"endpoint":   smhConfig.Endpoint,
		"expires_at": expiresAt.UnixMilli(),
	})
}

// HandleAdminSMHStat 查询 SMH 存储的整体统计信息
// GET /admin/smh/stat
func HandleAdminSMHStat(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	// 空间总数 = 全局空间（skillhub + common）+ 活跃个人空间数
	var personalCount int64
	model.DB(r.Context()).Model(&model.SMHPersonalSpace{}).Where("to_be_deleted_at IS NULL").Count(&personalCount)
	spaceCount := 2 + personalCount // skillhub + common

	// 活跃个人空间配额总和
	var personalQuotaSum int64
	model.DB(r.Context()).Model(&model.SMHPersonalSpace{}).Where("to_be_deleted_at IS NULL").Select("COALESCE(SUM(storage_quota), 0)").Scan(&personalQuotaSum)
	publicQuotaBytes := int64(commonSpaceQuotaBytes + skillhubSpaceQuotaBytes)
	totalQuotaBytes := publicQuotaBytes + personalQuotaSum

	// 查询 Library 总已用存储
	smhConfig := model.GetSMHConfig(r.Context())
	var totalUsedBytes int64
	var commonUsedBytes int64
	var skillhubUsedBytes int64
	if smhConfig.IsConfigured() {
		if total, err := describeLibraryUsage(r.Context(), smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret); err != nil {
			slog.Warn("[SMH] 查询 Library 用量失败", "error", err)
		} else {
			totalUsedBytes = total
		}
		// 查询 common 和 skillhub 空间已用存储
		var querySpaceIds []string
		if smhConfig.CommonSpace != "" {
			querySpaceIds = append(querySpaceIds, smhConfig.CommonSpace)
		}
		if smhConfig.SkillhubSpace != "" {
			querySpaceIds = append(querySpaceIds, smhConfig.SkillhubSpace)
		}
		if len(querySpaceIds) > 0 {
			if usageMap, err := describeSpaceUsage(r.Context(), smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret, querySpaceIds); err != nil {
				slog.Warn("[SMH] 查询空间用量失败", "error", err)
			} else {
				if info, ok := usageMap[smhConfig.CommonSpace]; ok {
					commonUsedBytes = info.size
				}
				if info, ok := usageMap[smhConfig.SkillhubSpace]; ok {
					skillhubUsedBytes = info.size
				}
			}
		}
	}

	// 个人空间已用（含回收站）= Library 总已用 - 公共空间已用
	personalUsedAllBytes := totalUsedBytes - commonUsedBytes - skillhubUsedBytes

	// 扣除回收站空间用量
	var recycleBinUsedBytes int64
	if smhConfig.IsConfigured() {
		var recycleBinSpaceIds []string
		model.DB(r.Context()).Model(&model.SMHPersonalSpace{}).
			Where("to_be_deleted_at IS NOT NULL AND space_id != ''").
			Pluck("space_id", &recycleBinSpaceIds)
		if len(recycleBinSpaceIds) > 0 {
			recycleBinUsedBytes = sumSpaceUsage(r.Context(), smhConfig, recycleBinSpaceIds)
		}
	}
	personalUsedBytes := personalUsedAllBytes - recycleBinUsedBytes

	publicUsedBytes := commonUsedBytes + skillhubUsedBytes

	jsonOK(w, map[string]interface{}{
		"space_count":   spaceCount,
		"used_storage":  publicUsedBytes + personalUsedBytes,
		"storage_quota": totalQuotaBytes,
		"public_space": map[string]interface{}{
			"used_storage":  publicUsedBytes,
			"storage_quota": publicQuotaBytes,
		},
		"common_space": map[string]interface{}{
			"used_storage":  commonUsedBytes,
			"storage_quota": commonSpaceQuotaBytes,
		},
		"skillhub_space": map[string]interface{}{
			"used_storage":  skillhubUsedBytes,
			"storage_quota": skillhubSpaceQuotaBytes,
		},
		"personal_space": map[string]interface{}{
			"count":         personalCount,
			"used_storage":  personalUsedBytes,
			"storage_quota": personalQuotaSum,
		},
	})
}

// smhSpaceUsageBatchSize 单次查询空间用量的最大空间数（URL 长度限制）
const smhSpaceUsageBatchSize = 100

// sumSpaceUsage 分批查询空间用量并求和，每批最多 100 个。
func sumSpaceUsage(ctx context.Context, smhConfig model.SMHConfig, spaceIds []string) int64 {
	var total int64
	for i := 0; i < len(spaceIds); i += smhSpaceUsageBatchSize {
		end := i + smhSpaceUsageBatchSize
		if end > len(spaceIds) {
			end = len(spaceIds)
		}
		batch := spaceIds[i:end]
		usageMap, err := describeSpaceUsage(ctx, smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret, batch)
		if err != nil {
			slog.Warn("[SMH] 批量查询空间用量失败", "batch_start", i, "error", err)
			continue
		}
		for _, info := range usageMap {
			total += info.size
		}
	}
	return total
}

// fetchPersonalSpaceUsage 分批查询多个 space 的用量，返回 map[spaceId]info。
func fetchPersonalSpaceUsage(ctx context.Context, spaceIds []string) map[string]spaceUsageInfo {
	result := make(map[string]spaceUsageInfo)
	if len(spaceIds) == 0 {
		return result
	}
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return result
	}
	for i := 0; i < len(spaceIds); i += smhSpaceUsageBatchSize {
		end := i + smhSpaceUsageBatchSize
		if end > len(spaceIds) {
			end = len(spaceIds)
		}
		m, err := describeSpaceUsage(ctx, smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret, spaceIds[i:end])
		if err != nil {
			slog.Warn("[SMH] 批量查询个人空间用量失败", "batch_start", i, "error", err)
			continue
		}
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// HandleAdminInstancePersonalSpace 批量管理实例个人空间开关。
// POST /admin/smh/instance-space
//
// 请求体（JSON）：
//
//	{"instance_ids": [1, 2, 3], "action": "enable" | "disable"}
//	{"select_all": true, "action": "enable" | "disable"}
//
// 逐个处理，部分失败不影响其他，返回每个实例的处理结果。
func HandleAdminInstancePersonalSpace(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	var req struct {
		InstanceIDs []uint `json:"instance_ids"`
		Action      string `json:"action"`
		SelectAll   bool   `json:"select_all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.Action != "enable" && req.Action != "disable" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgActionMustBeEnableDisable))
		return
	}
	if !req.SelectAll && len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	type itemResult struct {
		InstanceID uint   `json:"instance_id"`
		OK         bool   `json:"ok"`
		Error      string `json:"error,omitempty"`
	}

	var instances []model.Instance
	if req.SelectAll {
		model.DB(r.Context()).Find(&instances)
	} else {
		model.DB(r.Context()).Where("id IN ?", req.InstanceIDs).Find(&instances)
	}
	results := make([]itemResult, 0, len(instances))
	for i := range instances {
		instance := &instances[i]
		var opErr error
		if req.Action == "enable" {
			opErr = enableInstancePersonalSpace(r.Context(), instance)
		} else {
			opErr = disableInstancePersonalSpace(r.Context(), instance)
		}
		if opErr != nil {
			results = append(results, itemResult{InstanceID: instance.ID, OK: false, Error: opErr.Error()})
		} else {
			results = append(results, itemResult{InstanceID: instance.ID, OK: true})
		}
	}

	jsonOK(w, map[string]interface{}{"results": results})
}

func enableInstancePersonalSpace(ctx context.Context, instance *model.Instance) error {
	// 校验实例类型是否支持网盘
	if !model.AgentTypeSupportsSMH(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		return hcommon.I18nError(i18n.MsgSMHAgentTypeNotSupported, typeName)
	}

	var user model.User
	if model.DB(ctx).First(&user, instance.UserID).Error != nil {
		return hcommon.I18nError(i18n.MsgSMHInstanceUserNotFound)
	}

	var space model.SMHPersonalSpace
	err := model.DB(ctx).Unscoped().Where("instance_id = ?", instance.ID).First(&space).Error
	if err != nil {
		// 无记录：同步创建
		if _, err = createPersonalSpaceAndInitEnv(ctx, instance, &user); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSMHCreatePersonalSpaceFailed)
		}
		return nil // createPersonalSpaceAndInitEnv 会触发环境初始化
	}

	if space.ToBeDeletedAt == nil {
		// 已开启：幂等返回
		return nil
	}

	// 在回收站：乐观锁恢复
	changed, err := RestorePersonalSpace(ctx, &space)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSMHRestorePersonalSpaceFailed)
	}
	if !changed {
		return hcommon.I18nError(i18n.MsgSMHSpaceStatusChanged)
	}

	go func(ctx context.Context) {
		state, err := fetchCVMState(ctx, space.CVMInstanceId)
		if err != nil {
			slog.Warn("[SMH] 查询 CVM 状态失败，跳过立即初始化，后台任务兜底", "instance_id", space.CVMInstanceId, "error", err)
			return
		}
		if state != "RUNNING" {
			slog.Info("[SMH] CVM 非 RUNNING，跳过立即初始化，后台任务兜底", "instance_id", space.CVMInstanceId, "state", state)
			return
		}
		if err := TriggerSyncPersonalSpaceEnv(ctx, &space, true); err != nil {
			slog.Warn("[SMH] 个人空间恢复后触发环境初始化失败", "instance_id", space.CVMInstanceId, "error", err)
		}
	}(hcommon.DetachContext(ctx))
	slog.Info("[SMH] 个人空间已从回收站恢复", "instance_id", instance.InstanceId, "space_id", space.SpaceId)
	return nil
}

func disableInstancePersonalSpace(ctx context.Context, instance *model.Instance) error {
	var space model.SMHPersonalSpace
	if model.DB(ctx).Where("instance_id = ? AND to_be_deleted_at IS NULL", instance.ID).First(&space).Error != nil {
		return hcommon.I18nError(i18n.MsgSMHNoActivePersonalSpace)
	}

	// 乐观锁：to_be_deleted_at 仍为空才标记回收
	changed, err := RecyclePersonalSpace(ctx, &space)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSMHMarkRecycleFailed)
	}
	if !changed {
		return hcommon.I18nError(i18n.MsgSMHSpaceStatusChanged)
	}

	go func(ctx context.Context) {
		state, err := fetchCVMState(ctx, space.CVMInstanceId)
		if err != nil {
			slog.Warn("[SMH] 查询 CVM 状态失败，跳过立即卸载，后台任务兜底", "instance_id", space.CVMInstanceId, "error", err)
			return
		}
		if state != "RUNNING" {
			slog.Info("[SMH] CVM 非 RUNNING，跳过立即卸载，后台任务兜底", "instance_id", space.CVMInstanceId, "state", state)
			return
		}
		if err := TriggerSyncPersonalSpaceEnv(ctx, &space, false); err != nil {
			slog.Warn("[SMH] 触发个人空间环境卸载失败", "instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "error", err)
		}
	}(hcommon.DetachContext(ctx))
	slog.Info("[SMH] 个人空间已标记待回收", "instance_id", instance.InstanceId, "space_id", space.SpaceId)
	return nil
}

// createPersonalSpaceAndInitEnv 创建空间并异步触发环境初始化（单次 TAT：skill 安装 + token 注入）。
// 异步等待 CVM RUNNING + TAT Agent Online 后再触发，提高首发成功率；
// 等待超时仍失败由后台 personalSpaceEnvSyncInterval 服务兜底。
func createPersonalSpaceAndInitEnv(ctx context.Context, instance *model.Instance, user *model.User) (string, error) {
	spaceId, err := CreatePersonalSpaceForInstance(ctx, instance, user)
	if err != nil {
		return "", err
	}
	go func(ctx context.Context, inst model.Instance) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("[SMH] 初始化个人空间环境 panic", "instance_id", inst.InstanceId, "error", r)
			}
		}()
		if !waitForCVMRunning(ctx, inst.InstanceId, 10*time.Minute) {
			slog.Warn("[SMH] 等待 CVM 就绪超时，后台服务兜底", "instance_id", inst.InstanceId)
			return
		}
		if !waitForTATAgentOnline(ctx, inst.InstanceId, 3*time.Minute) {
			slog.Warn("[SMH] 等待 TAT Agent 就绪超时，后台服务兜底", "instance_id", inst.InstanceId)
			return
		}
		var space model.SMHPersonalSpace
		if model.DB(ctx).Where("instance_id = ?", inst.ID).First(&space).Error == nil {
			if err := TriggerSyncPersonalSpaceEnv(ctx, &space, true); err != nil {
				slog.Warn("[SMH] 触发环境初始化失败", "instance_id", inst.InstanceId, "error", err)
			}
		}
	}(hcommon.DetachContext(ctx), *instance)
	return spaceId, nil
}
