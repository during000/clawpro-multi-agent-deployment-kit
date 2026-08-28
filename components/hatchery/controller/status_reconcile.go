package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"hatchery/model"
)

// cvmAPIErrorState 与 batchFetchCVMInfoMap 中标记 CVM API 查询失败时写入的 State 保持一致。
const cvmAPIErrorState = "API_ERROR"

// ReconcileInstanceStatuses 后台 cvm-status-reconcile 任务的核心逻辑。
// 全量查实例 → 全量查 CVM → 计算语义状态 → diff 过滤 side-effect → 批量写缓存。
func ReconcileInstanceStatuses(ctx context.Context) {
	roundStartedAt := time.Now()

	// ① 全量轻量查询（不带过滤条件，后台需处理所有实例）
	allLight, err := queryAllLightInstancesWithFilter(ctx, adminQueryFilter{})
	if err != nil {
		slog.Error("[Reconcile] 轻量全量查询失败", "error", err)
		return
	}
	if len(allLight) == 0 {
		slog.Info("[Reconcile] 无实例，跳过")
		setFullSyncFinished(ctx, roundStartedAt)
		return
	}

	// ② 全量调 CVM
	var allCvmIds []string
	for _, item := range allLight {
		if item.InstanceId != "" {
			allCvmIds = append(allCvmIds, item.InstanceId)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(ctx, allCvmIds)

	// 统计本轮 CVM API 失败数：若需查询的实例全部失败，视为 CVM API 整体不可用，
	// 本轮不标记 ready，避免用 API 抖动期的兜底状态污染缓存并误导降级判断。
	errorCount := 0
	for _, info := range cvmInfoMap {
		if info != nil && info.State == cvmAPIErrorState {
			errorCount++
		}
	}
	cvmAPIAllFailed := len(allCvmIds) > 0 && errorCount == len(allCvmIds)
	slog.Info("[Reconcile] CVM 查询完成", "instance_count", len(allLight), "cvm_count", len(cvmInfoMap), "api_error", errorCount)

	// 🆕 批量预查实例状态依赖数据，消除 N+1（本地实例在循环内已 skip，不需要 localInfoMap）
	reconcileSiteConfig := model.GetSiteConfig(ctx)
	reconcileInstIDs := make([]uint, 0, len(allLight))
	for _, it := range allLight {
		if it.Source != model.InstanceSourceLocal {
			reconcileInstIDs = append(reconcileInstIDs, it.ID)
		}
	}
	reconcileInstallingMap := batchHasInstallingSkillInstallations(ctx, reconcileInstIDs)
	reconcileBatch := &InstanceStatusBatchLookup{SiteConfig: reconcileSiteConfig, InstallingSkillMap: reconcileInstallingMap}

	// ③ 遍历：计算语义状态 + diff 过滤 side-effect + 收集缓存更新
	var updates []model.InstanceStatusCacheItem
	var destroyedIDs []uint

	for _, it := range allLight {
		// 本地实例不走 reconcile：状态由 reporter report 主动更新，并有独立 sweep 负责失联后的 stopped 回收。
		if it.Source == model.InstanceSourceLocal {
			continue
		}
		info := cvmInfoMap[it.InstanceId]

		// 构造临时 Instance 用于状态计算
		tmpInst := lightToInstance(it)
		statusResp := ResolveInstanceStatus(ctx, &tmpInst, info, reconcileBatch)

		// side-effect diff 预过滤（加固 7.1）：
		// 仅有在途操作/CVM 消失/CVM 状态变化时才触发
		if shouldRunSideEffects(it, info) {
			handleStatusSideEffects(ctx, model.DB(ctx), &tmpInst, info, statusResp.Status)
		}

		// API_ERROR：本实例 CVM 查询失败，ResolveInstanceStatus 仅做了 running 兜底（用于实时展示），
		// 不应据此覆盖缓存。保留旧的 last_known_status 与 cvm_tags_json，跳过本条更新。
		if info != nil && info.State == cvmAPIErrorState {
			continue
		}

		// 收集已销毁实例（用于清理）
		if statusResp.Status == model.StatusDestroyed {
			destroyedIDs = append(destroyedIDs, it.ID)
		}

		// diff 过滤：状态变化，或 tags/image_id 为空时补齐，或资源字段发生变化。
		// tags 和 image_id 主要由各 handler（create/reinstall/upgrade）在操作时直写，
		// 后台 reconcile 仅对空值进行补齐（覆盖存量实例），已有值的不覆盖。
		statusChanged := statusResp.Status != it.LastKnownStatus
		// 存量补齐：仅当 DB 中为空且 CVM 有值时才写入。
		needFillTags := it.CVMTagsJSON == "" && info != nil && len(info.Tags) > 0
		needFillImage := it.ImgId == "" && info != nil && info.ImageId != ""
		// 资源缓存反映云上实时值，发生变化时更新。
		resourceChanged := info != nil && (it.CVMInstanceType != info.InstanceType ||
			it.CVMCPU != info.CPU ||
			it.CVMMemoryGB != info.MemoryGB ||
			it.SystemDiskType != info.SystemDiskType ||
			it.SystemDiskSize != info.SystemDiskSize ||
			it.CVMPublicIP != info.PublicIP ||
			it.CVMInternetChargeType != info.InternetChargeType ||
			it.CVMInternetMaxBandwidthOut != info.InternetMaxBandwidthOut)

		if statusChanged || needFillTags || needFillImage || resourceChanged {
			item := model.InstanceStatusCacheItem{
				ID:     it.ID,
				Status: statusResp.Status,
			}
			if needFillTags {
				tagsJSON := reconcileTagsToJSON(info, "")
				item.TagsJSON = &tagsJSON
			}
			if needFillImage {
				item.ImageId = &info.ImageId
			}
			if resourceChanged {
				item.CVMInstanceType = &info.InstanceType
				item.CVMCPU = &info.CPU
				item.CVMMemoryGB = &info.MemoryGB
				item.SystemDiskType = &info.SystemDiskType
				item.SystemDiskSize = &info.SystemDiskSize
				item.CVMPublicIP = &info.PublicIP
				item.CVMInternetChargeType = &info.InternetChargeType
				item.CVMInternetMaxBandwidthOut = &info.InternetMaxBandwidthOut
			}
			slog.Debug("[Reconcile] diff/补齐",
				"id", it.ID, "instance_id", it.InstanceId,
				"status_changed", statusChanged, "fill_tags", needFillTags, "fill_image", needFillImage,
				"resource_changed", resourceChanged,
			)
			updates = append(updates, item)
		}
	}

	// ④ 批量写回缓存
	if len(updates) > 0 {
		model.BatchUpdateInstanceStatusCache(ctx, updates, roundStartedAt)
	}

	// ⑤ 清理已销毁超过 1 天的实例
	if len(destroyedIDs) > 0 {
		model.CleanupDestroyedInstances(ctx, destroyedIDs, 24*time.Hour)
	}

	// ⑥ 标记整轮成功完成（CVM API 整体不可用时本轮不标记，等待恢复后的轮次）
	if cvmAPIAllFailed {
		slog.Warn("[Reconcile] CVM API 整体不可用，本轮不标记 ready", "cvm_count", len(allCvmIds))
	} else {
		setFullSyncFinished(ctx, roundStartedAt)
	}

	// ② 本地实例 sweep：双向对账，以 last_report_at 为唯一事实源刷新 last_known_status（running ↔ stopped）。
	reconcileLocalInstances(ctx)

	slog.Info("[Reconcile] 完成", "total", len(allLight), "updated", len(updates), "duration", time.Since(roundStartedAt))
}

// reconcileLocalInstances 本地实例状态双向对账。
//
// 前提：report 接口通过事务保证 `instances` 和 `local_instance_infos` 原子写入，
// 每个本地 agent 实例在 local_instance_infos 里必然有对应行且 last_report_at 非空。
// 因此本函数不需要处理「info 行缺失」或「last_report_at IS NULL」等边界。
//
// 对账规则（以 last_report_at 为唯一事实源，两方向对称）：
//   - 心跳过期 → last_known_status = stopped
//   - 心跳新鲜 → last_known_status = running
//
// 实现：先 JOIN 查出不一致行的 id，再按 id 批量 UPDATE
// （SQLite 不支持 UPDATE ... JOIN，两步走同时兼容 MySQL 与 SQLite 测试环境）。
func reconcileLocalInstances(ctx context.Context) {
	cutoff := time.Now().Add(-model.LocalInstanceOfflineThreshold)
	now := time.Now()

	pluckIDs := func(currentStatus, cmp string) []uint {
		var ids []uint
		if err := model.DB(ctx).
			Model(&model.Instance{}).
			Joins("JOIN local_instance_infos ON local_instance_infos.instance_id = instances.id AND local_instance_infos.deleted_at IS NULL").
			Where("instances.source = ?", model.InstanceSourceLocal).
			Where("instances.last_known_status = ?", currentStatus).
			Where("local_instance_infos.last_report_at "+cmp+" ?", cutoff).
			Pluck("instances.id", &ids).Error; err != nil {
			slog.Warn("[Reconcile][Local] 查询失败", "current_status", currentStatus, "error", err)
			return nil
		}
		return ids
	}

	updateStatus := func(ids []uint, targetStatus string) {
		if len(ids) == 0 {
			return
		}
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"last_known_status": targetStatus,
				"status_synced_at":  now,
			}).Error; err != nil {
			slog.Warn("[Reconcile][Local] 批量写回失败", "target", targetStatus, "ids", ids, "error", err)
			return
		}
		slog.Info("[Reconcile][Local] 实例状态刷新", "target", targetStatus, "count", len(ids))
	}

	// ① running 但心跳过期 → stopped
	updateStatus(pluckIDs(model.StatusRunning, "<"), model.StatusStopped)

	// ② stopped 但心跳新鲜 → running（自愈历史脏数据 & Pluck/UPDATE 之间的竞态误伤）
	updateStatus(pluckIDs(model.StatusStopped, ">="), model.StatusRunning)
}

// IsStatusCacheReady 判断后台任务是否完整成功完成过一轮。
// List 接口据此决定走新逻辑（纯 DB 读）还是旧逻辑（全量 CVM）。
func IsStatusCacheReady(ctx context.Context) bool {
	return model.GetLastFullSyncFinishedAt(ctx) != nil
}

// shouldRunSideEffects 判断实例是否需要执行 side-effect（diff 预过滤，加固 7.1）。
func shouldRunSideEffects(it lightInstance, cvmInfo *CVMInstanceInfo) bool {
	if model.IsResourceAdjustmentOperation(it.CurrentOperation) {
		return false
	}
	// 有在途操作 → 需要超时检测/收敛
	if it.CurrentOperation != "" {
		return true
	}
	// CVM 消失 → 可能外部销毁
	if it.InstanceId != "" && cvmInfo == nil {
		return true
	}
	if cvmInfo != nil && cvmInfo.State != "API_ERROR" {
		// CVM 状态变化 → 需要更新 last_cvm_state
		if cvmInfo.State != "" && cvmInfo.State != it.LastCVMState {
			return true
		}
		// CVM 计费类型变化或存量 DB 为空 → 需要同步 instance_charge_type
		if cvmInfo.InstanceChargeType != "" && cvmInfo.InstanceChargeType != it.InstanceChargeType {
			return true
		}
	}
	return false
}

// lightToInstance 从 lightInstance 构造最小的 model.Instance 用于状态计算。
func lightToInstance(it lightInstance) model.Instance {
	inst := model.Instance{}
	inst.ID = it.ID
	inst.Name = it.Name
	inst.UserID = it.UserID
	inst.GroupID = it.GroupID
	inst.InstanceId = it.InstanceId
	inst.CurrentOperation = it.CurrentOperation
	inst.CurrentOperationState = it.CurrentOperationState
	inst.CurrentOperationUpdatedAt = it.CurrentOperationUpdatedAt
	inst.LastCVMState = it.LastCVMState
	inst.LastStableState = it.LastStableState
	inst.AgentReady = it.AgentReady
	inst.CLSAgentStatus = it.CLSAgentStatus
	inst.CLSAgentStatusAt = it.CLSAgentStatusAt
	inst.InstanceChargeType = it.InstanceChargeType
	inst.CVMInstanceType = it.CVMInstanceType
	inst.CVMCPU = it.CVMCPU
	inst.CVMMemoryGB = it.CVMMemoryGB
	inst.SystemDiskType = it.SystemDiskType
	inst.SystemDiskSize = it.SystemDiskSize
	return inst
}

// setFullSyncFinished 标记整轮同步成功完成。
func setFullSyncFinished(ctx context.Context, t time.Time) {
	if err := model.SetLastFullSyncFinishedAt(ctx, t); err != nil {
		slog.Error("[Reconcile] 设置同步完成标记失败", "error", err)
	}
}

// reconcileTagsToJSON 将 CVM 标签转为 JSON 字符串。
// API_ERROR（CVM API 查询失败）时保留旧缓存 oldJSON，避免用空标签覆盖真实数据。
func reconcileTagsToJSON(info *CVMInstanceInfo, oldJSON string) string {
	if info != nil && info.State == cvmAPIErrorState {
		return oldJSON
	}
	if info == nil || len(info.Tags) == 0 {
		return "[]"
	}
	data, err := json.Marshal(info.Tags)
	if err != nil {
		// 序列化异常也保留旧值，不冒险覆盖
		slog.Warn("[Reconcile] 标签序列化失败", "error", err, "tags", info.Tags)
		return oldJSON
	}
	return string(data)
}
