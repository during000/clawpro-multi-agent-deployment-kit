package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// 角色切换/分发的执行模式：用户端单实例切换 vs 管理端批量分发。
type applyMode int

const (
	applyModeSwitch applyMode = iota
	applyModeDistribute
)

// 角色切换/分发被跳过的机器可读 reason 枚举（前端按 token 翻译展示）。
const (
	skipReasonNotFound             = "not_found"
	skipReasonAgentTypeUnsupported = "agent_type_unsupported"
	skipReasonNotRunning           = "not_running"
	skipReasonOperationInProgress  = "operation_in_progress"
	skipReasonRoleMismatch         = "role_mismatch"
	skipReasonAlreadyUpdated       = "already_updated"
	skipReasonNoRoleToRemove       = "no_role_to_remove"
	skipReasonUpdatingInProgress   = "updating_in_progress"
	skipReasonRoleNotVisible       = "role_not_visible"
)

// SkippedItem 被跳过的实例信息。reason 是 token，供前端翻译。
type SkippedItem struct {
	InstanceID uint   `json:"instance_id"`
	Reason     string `json:"reason"`
}

// ApplyResult applyRoleToInstances 的最终输出。
type ApplyResult struct {
	Accepted int           `json:"accepted"`
	Skipped  []SkippedItem `json:"skipped"`
}

// applyOptions 承载 apply 时的元数据（写入 role_distribution_records 快照）。
type applyOptions struct {
	OperatorID uint   // 触发操作的用户 ID（AdminToken 场景为 0）
	Source     string // switch / distribute / create
}

// applyRoleToInstances 是 switch-role / distribute 共用的核心 service。
//
//   - mode = applyModeSwitch     : 用户端单实例切换；targetRoleID = 0 表示移除角色
//   - mode = applyModeDistribute : 管理端批量分发；targetRoleID 必须 > 0
//
// 同步阶段：校验管线 + DB 事务写入 role_id / distributed_role_version / role_sync_status='updating' /
// soul_set_at + 创建/关联 SkillInstallation 差集 + 创建 RoleDistributionRecord。
// 异步阶段：fire & forget 执行 SetInstanceSoul / RemoveInstanceSoul + installSkillsAsync。
// SOUL 完成后自动 refreshRoleRecord 聚合状态；技能状态由周期任务兜底聚合。
func applyRoleToInstances(
	ctx context.Context,
	instanceIDs []uint,
	targetRoleID uint,
	mode applyMode,
	resolver instanceStatusResolver,
	opts applyOptions,
) (*ApplyResult, error) {
	role, err := loadTargetRole(ctx, targetRoleID)
	if err != nil {
		return nil, err
	}

	instances, missing := loadInstancesForApply(ctx, instanceIDs)
	result := &ApplyResult{Skipped: make([]SkippedItem, 0)}
	for _, id := range missing {
		result.Skipped = append(result.Skipped, SkippedItem{InstanceID: id, Reason: skipReasonNotFound})
	}

	accepted := filterInstancesForApply(ctx, instances, role, mode, resolver, result)
	for _, inst := range accepted {
		applyRoleSingle(ctx, inst, role, mode, opts)
		result.Accepted++
	}
	return result, nil
}

// loadTargetRole targetRoleID > 0 时加载角色；= 0 (switch 模式专属) 返回 nil。
func loadTargetRole(ctx context.Context, targetRoleID uint) (*model.OpenClawRole, error) {
	if targetRoleID == 0 {
		return nil, nil
	}
	var role model.OpenClawRole
	if err := model.DB(ctx).First(&role, targetRoleID).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgRoleApplyLoadRoleFailed)
	}
	return &role, nil
}

// loadInstancesForApply 批量加载实例，返回 (找到的实例切片, 缺失的 ID 列表)。
// 缺失既包含真不存在，也包含被多租户 identifier 过滤掉的（视作 not_found）。
func loadInstancesForApply(ctx context.Context, ids []uint) ([]model.Instance, []uint) {
	if len(ids) == 0 {
		return nil, nil
	}
	var found []model.Instance
	model.DB(ctx).Where("id IN ?", ids).Find(&found)

	foundSet := make(map[uint]bool, len(found))
	for _, inst := range found {
		foundSet[inst.ID] = true
	}
	var missing []uint
	for _, id := range ids {
		if !foundSet[id] {
			missing = append(missing, id)
		}
	}
	return found, missing
}

// filterInstancesForApply 走校验管线，把通过的实例返回，被跳过的写入 result.Skipped。
//
// 管线顺序（按用户拍板的设计）：
//
//	① agent_type_unsupported
//	② not_running
//	③ distribute 专属：role_mismatch
//	④ distribute 专属：already_updated
//	⑤ switch + role_id=0 专属：no_role_to_remove
func filterInstancesForApply(
	ctx context.Context,
	instances []model.Instance,
	role *model.OpenClawRole,
	mode applyMode,
	resolver instanceStatusResolver,
	result *ApplyResult,
) []model.Instance {
	accepted := make([]model.Instance, 0, len(instances))
	for i := range instances {
		inst := instances[i]
		if reason := pipelineCheckInstance(ctx, &inst, role, mode, resolver); reason != "" {
			result.Skipped = append(result.Skipped, SkippedItem{InstanceID: inst.ID, Reason: reason})
			continue
		}
		accepted = append(accepted, inst)
	}
	return accepted
}

// pipelineCheckInstance 按管线顺序检查单个实例，命中第一条不通过的规则即返回对应 reason；
// 全部通过返回空字符串。
func pipelineCheckInstance(
	ctx context.Context,
	inst *model.Instance,
	role *model.OpenClawRole,
	mode applyMode,
	resolver instanceStatusResolver,
) string {
	if !model.AgentTypeSupportsRole(ctx, inst.AgentType) {
		return skipReasonAgentTypeUnsupported
	}
	if err := requireNoResourceAdjustment(inst); err != nil {
		return skipReasonOperationInProgress
	}
	if _, err := requireInstanceRunning(ctx, inst, resolver); err != nil {
		return skipReasonNotRunning
	}
	// updating_in_progress：任何时候实例上一次角色更新还在进行中都拒绝新的 apply，避免撞车。
	if inst.RoleSyncStatus == model.RoleSyncStatusUpdating {
		return skipReasonUpdatingInProgress
	}
	if mode == applyModeDistribute {
		if inst.RoleID != role.ID {
			return skipReasonRoleMismatch
		}
		// already_updated 收紧：仅当版本号 >= 且 role_sync_status=updated 时才判定"已完成"。
		// updating（进行中）已在上面拦截；failed / pending 允许重发。
		if compareRoleVersions(inst.DistributedRoleVersion, role.Version) >= 0 &&
			inst.RoleSyncStatus == model.RoleSyncStatusUpdated {
			return skipReasonAlreadyUpdated
		}
	}
	if mode == applyModeSwitch && role == nil && inst.RoleID == 0 {
		return skipReasonNoRoleToRemove
	}
	// switch 模式下检查角色对实例所在分组是否可见
	// distribute 是管理员操作不需要校验
	if mode == applyModeSwitch && role != nil {
		if !isRoleVisibleToInstanceGroup(ctx, role, inst.GroupID) {
			return skipReasonRoleNotVisible
		}
	}
	return ""
}

// isRoleVisibleToInstanceGroup 检查角色对实例所在分组是否可见。
// visibility_type=all → 可见
// visibility_type=group → 检查 instanceGroupID 是否在角色的可见分组列表中
// instanceGroupID=0（未指定分组）→ group 类型角色不可见
func isRoleVisibleToInstanceGroup(ctx context.Context, role *model.OpenClawRole, instanceGroupID uint) bool {
	if role.VisibilityType == model.VisibilityAll {
		return true
	}
	if instanceGroupID == 0 {
		return false // 实例未指定分组，group 类型角色不可见
	}
	groupIDs, err := model.GetRoleVisibilityGroupIDs(ctx, []uint{role.ID})
	if err != nil {
		slog.Warn("[isRoleVisibleToInstanceGroup] 查询角色可见分组失败", "role_id", role.ID, "error", err)
		return false
	}
	for _, gid := range groupIDs[role.ID] {
		if gid == instanceGroupID {
			return true
		}
	}
	return false
}

// applyRoleSingle 对单个实例落库 + 触发异步下发。tx 内只做必须原子化的事：
//  1. finalize 老 record（若存在）为 cancelled
//  2. 更新 instance 字段（role_id / distributed_role_version / role_sync_status='updating' / soul_set_at=nil）
//  3. 计算并创建 SkillInstallation 差集
//  4. 创建新 record 并写入 SkillInstallationIDs JSON
//
// tx 外触发异步：SetInstanceSoul / RemoveInstanceSoul（带新 record.ID）+ installSkillsAsync。
func applyRoleSingle(ctx context.Context, inst model.Instance, role *model.OpenClawRole, mode applyMode, opts applyOptions) {
	logger := slog.With("task", "applyRoleSingle", "instance_id", inst.ID, "mode", mode)

	recordID, err := writeRoleFieldsTx(ctx, &inst, role, opts)
	if err != nil {
		logger.Error("更新实例角色字段失败", "error", err)
		return
	}

	detached := hcommon.DetachContext(ctx)
	if role == nil {
		// role_id=0 场景（switch role_id=0）：清空 role 相关字段，异步走 RemoveInstanceSoul。
		// tx 内已经把 old record finalize 为 cancelled，无 recordID 传给 SOUL 侧。
		go fireRemoveInstanceSoul(detached, inst.ID, 0)
		return
	}
	go fireRoleDistribution(detached, ctx, inst, role.ID, recordID)
}

// writeRoleFieldsTx 在事务内更新实例角色字段 + 创建/finalize record + 创建 SkillInstallation 差集。
// 返回新创建的 record.ID（role == nil 时返回 0）。
func writeRoleFieldsTx(ctx context.Context, inst *model.Instance, role *model.OpenClawRole, opts applyOptions) (uint, error) {
	var newRecordID uint
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. finalize 老 record 为 cancelled（如果存在）
		finalizeActiveRecordAsCancelled(tx, inst.ID)

		// 2. 更新 instance 字段
		updates := buildRoleFieldUpdates(role)
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(updates).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRoleApplyDBUpdateFailed)
		}
		// 同步本地副本
		inst.RoleID = pickRoleID(role)
		inst.DistributedRoleVersion = pickRoleVersion(role)
		inst.RoleSyncStatus = pickInitialSyncStatus(role)

		// 3. role == nil 时到此为止，不用建新 record
		if role == nil {
			return nil
		}

		// 4. 创建 SkillInstallation 差集，收集 IDs
		skillIDs, err := createSkillDiffInstallations(tx, inst.ID, role.ID)
		if err != nil {
			return err
		}

		// 5. 创建新 record（含 SkillInstallationIDs JSON）
		record, err := createRoleDistributionRecord(tx, inst, role, opts, skillIDs)
		if err != nil {
			return err
		}
		newRecordID = record.ID
		return nil
	})
	return newRecordID, err
}

// buildRoleFieldUpdates 根据目标 role 构造 GORM Updates map。
//   - role == nil → role_id=0, distributed_role_version="", role_sync_status=""
//   - role != nil → role_id=role.ID, distributed_role_version=role.Version, role_sync_status="updating"
//
// 两种情况都把 soul_set_at 置 nil，触发周期任务兜底重下发。
func buildRoleFieldUpdates(role *model.OpenClawRole) map[string]interface{} {
	return map[string]interface{}{
		"soul_set_at":              nil,
		"role_id":                  pickRoleID(role),
		"distributed_role_version": pickRoleVersion(role),
		"role_sync_status":         pickInitialSyncStatus(role),
	}
}

func pickRoleID(role *model.OpenClawRole) uint {
	if role == nil {
		return 0
	}
	return role.ID
}

func pickRoleVersion(role *model.OpenClawRole) string {
	if role == nil {
		return ""
	}
	return role.Version
}

// pickInitialSyncStatus apply 时实例的初始 role_sync_status。
//   - role == nil → "" (无角色)
//   - role != nil → "updating" (下发中，待异步收敛)
func pickInitialSyncStatus(role *model.OpenClawRole) string {
	if role == nil {
		return model.RoleSyncStatusEmpty
	}
	return model.RoleSyncStatusUpdating
}

// createSkillDiffInstallations 计算"目标角色技能 - 实例已成功安装技能"差集（按 slug 比对），
// 把差集每条作为一条 SkillInstallation(status=None) 写入，由 installSkillsAsync 后续装载。
// 返回本次新建的 SkillInstallation ID 列表，供 record 冗余记录。
//
// 策略（用户拍板）：技能不删；已有的 slug 用新版本覆盖装；没有的 slug 新装。
// 这里不区分"已是同版本"——TAT 安装幂等，多跑一次成本可接受。
func createSkillDiffInstallations(tx *gorm.DB, instanceID uint, roleID uint) ([]uint, error) {
	var roleSkills []model.OpenClawRoleSkill
	if err := tx.Where("open_claw_role_id = ?", roleID).Find(&roleSkills).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgRoleApplyDBUpdateFailed)
	}
	if len(roleSkills) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(roleSkills))
	for _, rs := range roleSkills {
		rec := model.SkillInstallation{
			InstanceID:    instanceID,
			Name:          rs.Name,
			Slug:          rs.Slug,
			Version:       rs.Version,
			CosZipKey:     rs.CosZipKey,
			InstallStatus: model.SkillInstallNone,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgRoleApplyDBUpdateFailed)
		}
		ids = append(ids, rec.ID)
	}
	return ids, nil
}

// createRoleDistributionRecord 创建本次 apply 的 role_distribution_records 记录。
// SkillInstallationIDs 用 JSON 数组格式存储，便于 refresh 时反查。
func createRoleDistributionRecord(
	tx *gorm.DB,
	inst *model.Instance,
	role *model.OpenClawRole,
	opts applyOptions,
	skillIDs []uint,
) (*model.RoleDistributionRecord, error) {
	skillIDsJSON, err := marshalSkillInstallationIDs(skillIDs)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgRoleApplyDBUpdateFailed)
	}
	record := &model.RoleDistributionRecord{
		InstanceID:           inst.ID,
		InstanceCID:          inst.InstanceId,
		RoleID:               role.ID,
		RoleName:             role.Name,
		Version:              role.Version,
		OperatorID:           opts.OperatorID,
		Source:               opts.Source,
		Status:               model.RoleRecordStatusUpdating,
		SoulStatus:           model.RoleSubStatusPending,
		SkillStatus:          model.RoleSubStatusPending,
		SkillInstallationIDs: skillIDsJSON,
	}
	if err := tx.Create(record).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgRoleApplyDBUpdateFailed)
	}
	return record, nil
}

// marshalSkillInstallationIDs 把 IDs 编码成 JSON 数组字符串（"[1,2,3]"）。
// 空切片返回空串（record.SkillInstallationIDs 默认值也是空串，保持一致）。
func marshalSkillInstallationIDs(ids []uint) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	buf, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// fireRemoveInstanceSoul 包一层日志，避免在 goroutine 里裸吞错误。
func fireRemoveInstanceSoul(ctx context.Context, instanceID uint, recordID uint) {
	if err := RemoveInstanceSoul(ctx, instanceID, recordID); err != nil {
		slog.Warn("[RoleApply] RemoveInstanceSoul 失败", "instance_id", instanceID, "error", err)
	}
}

// fireRoleDistribution 异步执行角色下发的完整流程：
//  1. SetInstanceSoul — SOUL 注入，成败写入 record.soul_status
//  2. waitForRoleSkillsSynced — 等待角色技能 cos_zip_key 同步完成
//  3. refreshSkillInstallationsCosZipKey — 刷新 SkillInstallation 快照中的 cos_zip_key
//  4. installSkillsAsync — 技能安装，结束后 defer refreshRoleRecord 聚合状态
//
// detached 是脱离请求生命周期的 context；origCtx 用于提取 i18n printer，
// 使 installSkillsAsync 中写入 DB 的 error_message 能按调用者语言国际化。
func fireRoleDistribution(detached context.Context, origCtx context.Context, inst model.Instance, roleID, recordID uint) {
	ctx := i18n.WithPrinter(detached, origCtx)
	logger := slog.With("task", "fireRoleDistribution",
		"instance_id", inst.ID, "role_id", roleID)

	// 1. SOUL 注入
	if err := SetInstanceSoul(ctx, inst.ID, recordID); err != nil {
		logger.Warn("SetInstanceSoul 失败，靠周期任务兜底", "error", err)
	}

	// 2. 等待角色技能 cos_zip_key 同步完成。
	// syncRoleSkillsCosZipKey 在角色创建/更新时异步触发，可能尚未完成。
	// 等待期间 SkillInstallation 保持 None 状态，refreshRoleRecord 会将 record 视为 updating。
	if !waitForRoleSkillsSynced(ctx, roleID) {
		logger.Warn("等待角色技能同步超时，继续安装（空 cos_zip_key 会被标 failed）",
			"instance_id", inst.ID, "role_id", roleID)
	}

	// 3. 刷新 SkillInstallation 的 cos_zip_key（快照时可能为空）
	refreshSkillInstallationsCosZipKey(ctx, inst.ID, roleID)

	// 4. 技能安装（内部 defer refreshRoleRecord 聚合状态）
	installSkillsAsync(ctx, inst.ID, inst.InstanceId, inst.AgentType, waitModeRetry)
}

// waitForRoleSkillsSynced 等待角色技能的 cos_zip_key 同步完成。
//
// syncRoleSkillsCosZipKey 在角色创建/更新时异步触发（go 关键字），下载 zip 并上传到
// SMH common space 后回写 open_claw_role_skills.cos_zip_key。在 cos_zip_key 就绪前
// 去安装技能没有意义（下载 URL 无法生成）。
//
// 本函数在 applyRoleSingle 的异步 goroutine 中调用，不阻塞 API 响应。
// 策略：
//  1. 快速 DB 查询：该角色是否还有 cos_zip_key='' 的技能？没有则立即返回。
//  2. 有空值 → 主动同步调用 syncRoleSkillsCosZipKey（内部先尝试复用已有记录，毫秒级）
//  3. 同步后仍有空值 → 轮询等待（等角色创建时的异步 sync 完成），最长等 5 分钟
//  4. 超时返回 false，调用方继续走 installSkillsAsync（空 cos_zip_key 会被标 failed）
//
// 等待角色技能 SMH 同步的超时与轮询间隔，包级变量便于测试覆盖。
var (
	roleSkillsSyncMaxWait      = 5 * time.Minute
	roleSkillsSyncPollInterval = 5 * time.Second
)

func waitForRoleSkillsSynced(ctx context.Context, roleID uint) bool {
	countEmpty := func() int64 {
		var count int64
		model.DB(ctx).Model(&model.OpenClawRoleSkill{}).
			Where("open_claw_role_id = ? AND cos_zip_key = ''", roleID).
			Count(&count)
		return count
	}

	// 1. 快速检查
	if countEmpty() == 0 {
		return true
	}

	// 2. 主动触发同步（角色创建时的异步 sync 可能尚未完成或已失败）
	slog.Info("[waitForRoleSkillsSynced] 角色技能 cos_zip_key 未就绪，主动触发同步", "role_id", roleID)
	syncRoleSkillsCosZipKey(ctx, roleID)

	if countEmpty() == 0 {
		return true
	}

	// 3. 轮询等待
	deadline := time.Now().Add(roleSkillsSyncMaxWait)
	for time.Now().Before(deadline) {
		time.Sleep(roleSkillsSyncPollInterval)
		empty := countEmpty()
		if empty == 0 {
			return true
		}
		slog.Info("[waitForRoleSkillsSynced] 等待角色技能 SMH 同步",
			"role_id", roleID, "empty_count", empty)
	}

	slog.Warn("[waitForRoleSkillsSynced] 等待角色技能同步超时", "role_id", roleID)
	return false
}

// refreshSkillInstallationsCosZipKey 从 open_claw_role_skills 刷新 SkillInstallation 的 cos_zip_key。
//
// createSkillDiffInstallations 在事务内快照 cos_zip_key，此时可能还是空值。
// waitForRoleSkillsSynced 确认同步完成后，本函数把最新的 cos_zip_key 回填到 SkillInstallation，
// 使后续 installSkillsAsync 能正常生成下载 URL。
func refreshSkillInstallationsCosZipKey(ctx context.Context, instanceID uint, roleID uint) {
	var roleSkills []model.OpenClawRoleSkill
	model.DB(ctx).Where("open_claw_role_id = ?", roleID).Find(&roleSkills)
	slugToZipKey := make(map[string]string, len(roleSkills))
	for _, rs := range roleSkills {
		if rs.CosZipKey != "" {
			slugToZipKey[rs.Slug] = rs.CosZipKey
		}
	}

	var installs []model.SkillInstallation
	model.DB(ctx).Where("instance_id = ? AND install_status = ? AND cos_zip_key = ''",
		instanceID, model.SkillInstallNone).Find(&installs)
	for _, si := range installs {
		if zipKey, ok := slugToZipKey[si.Slug]; ok {
			model.DB(ctx).Model(&si).Update("cos_zip_key", zipKey)
		}
	}
}
