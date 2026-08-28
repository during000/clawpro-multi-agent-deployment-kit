package controller

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// gatewayRestartTasks 跟踪会重启 openclaw gateway 的异步任务。
// Key: 实例 DB ID (uint)；Value: *int32 (计数器)。
// 仅在创建实例流程中使用，阻止 /openclaw/status 在初始化任务完成前返回 "running"。
var gatewayRestartTasks sync.Map // map[uint]*int32

// trackGatewayRestartTask 增加实例的待完成任务计数。
func trackGatewayRestartTask(instanceID uint) {
	val, _ := gatewayRestartTasks.LoadOrStore(instanceID, new(int32))
	atomic.AddInt32(val.(*int32), 1)
}

// untrackGatewayRestartTask 减少待完成任务计数。
func untrackGatewayRestartTask(instanceID uint) {
	val, ok := gatewayRestartTasks.Load(instanceID)
	if !ok {
		return
	}
	atomic.AddInt32(val.(*int32), -1)
}

// hasPendingGatewayRestartTasks 检查实例是否还有未完成的异步任务。
// 同时清理计数器 <= 0 的过期条目，防止内存泄漏。
func hasPendingGatewayRestartTasks(instanceID uint) bool {
	val, ok := gatewayRestartTasks.Load(instanceID)
	if !ok {
		return false
	}
	ptr := val.(*int32)
	v := atomic.LoadInt32(ptr)
	if v <= 0 {
		gatewayRestartTasks.Delete(instanceID)
		return false
	}
	return true
}

// InstanceStatusResponse 用户端状态响应
type InstanceStatusResponse struct {
	Status             string   `json:"status"`
	Label              string   `json:"label"`
	Tooltip            string   `json:"tooltip"`
	Actions            []string `json:"actions"`
	Transient          bool     `json:"transient"`
	InstanceChargeType string   `json:"instance_charge_type"`
}

// CVMTag CVM 实例标签键值对
type CVMTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CVMInstanceInfo CVM 实例信息（从 DescribeInstances 提取）
type CVMInstanceInfo struct {
	State                       string   // CVM 状态
	LatestOperation             string   // 最近操作
	LatestOperationState        string   // SUCCESS/OPERATING/FAILED
	LatestOperationRequestID    string   // 最近操作 RequestId
	LatestOperationErrorMessage string   // 最近操作内部错误，仅供脱敏日志/映射
	RestrictState               string   // 实例业务状态；非 NORMAL 即限制
	StopChargingMode            string   // KEEP_CHARGING/STOP_CHARGING/NOT_APPLICABLE
	ImageId                     string   // 实例当前使用的镜像 ID
	OsName                      string   // 实例操作系统名称
	InstanceChargeType          string   // 实例计费类型
	InstanceType                string   // CVM 规格
	CPU                         int64    // CPU 核数
	MemoryGB                    int64    // 内存 GiB
	Zone                        string   // 可用区
	SystemDiskID                string   // 系统盘 ID
	SystemDiskType              string   // 系统盘介质
	SystemDiskSize              int64    // 系统盘 GiB
	Tags                        []CVMTag // 实例绑定的标签

	// 公网信息来自 DescribeInstances，并由状态 reconcile 写入实例资源缓存。
	PublicIP                string // 首个公网 IP，未分配时为空串
	InternetChargeType      string // 公网计费类型（BANDWIDTH_POSTPAID_BY_HOUR / TRAFFIC_POSTPAID_BY_HOUR / BANDWIDTH_PACKAGE 等）
	InternetMaxBandwidthOut int64  // 公网带宽上限（Mbps），0 = 未开通
}

// InstanceStatusBatchLookup 批量实例状态预查缓存。
// 调用方在循环前通过 batchHasInstallingSkillInstallations 等预查，
// 将结果装入此结构体传入 ResolveInstanceStatus，消除 N+1 循环内 DB 查询。
// 为 nil 时各步骤按需逐条查 DB（向后兼容现有调用）。
type InstanceStatusBatchLookup struct {
	SiteConfig         model.SiteConfig                  // 预查的站点配置（避免循环内 GetSiteConfig DB 查询）
	InstallingSkillMap map[uint]bool                     // instanceID → 是否有 installing 状态的技能安装
	LocalInfoMap       map[uint]*model.LocalInstanceInfo // instanceID → 本地实例信息
}

func cvmInstanceInfoFromSDK(inst *cvm.Instance) *CVMInstanceInfo {
	if inst == nil {
		return nil
	}
	info := &CVMInstanceInfo{
		State:                       StrVal(inst.InstanceState),
		LatestOperation:             StrVal(inst.LatestOperation),
		LatestOperationState:        StrVal(inst.LatestOperationState),
		LatestOperationRequestID:    StrVal(inst.LatestOperationRequestId),
		LatestOperationErrorMessage: StrVal(inst.LatestOperationErrorMsg),
		RestrictState:               StrVal(inst.RestrictState),
		StopChargingMode:            StrVal(inst.StopChargingMode),
		ImageId:                     StrVal(inst.ImageId),
		OsName:                      StrVal(inst.OsName),
		InstanceChargeType:          StrVal(inst.InstanceChargeType),
		InstanceType:                StrVal(inst.InstanceType),
		Tags:                        extractCVMTags(inst.Tags),
		PublicIP:                    firstPublicIP(inst.PublicIpAddresses),
		InternetChargeType:          internetChargeTypeFromCVM(inst.InternetAccessible),
		InternetMaxBandwidthOut:     internetBandwidthFromCVM(inst.InternetAccessible),
	}
	if inst.CPU != nil {
		info.CPU = *inst.CPU
	}
	if inst.Memory != nil {
		info.MemoryGB = *inst.Memory
	}
	if inst.Placement != nil {
		info.Zone = StrVal(inst.Placement.Zone)
	}
	if inst.SystemDisk != nil {
		info.SystemDiskID = StrVal(inst.SystemDisk.DiskId)
		info.SystemDiskType = StrVal(inst.SystemDisk.DiskType)
		if inst.SystemDisk.DiskSize != nil {
			info.SystemDiskSize = *inst.SystemDisk.DiskSize
		}
	}
	return info
}

// ResolveInstanceStatus 状态映射引擎（9 步）
// 根据 DB 记录和 CVM 状态，计算 OpenClaw 语义状态。
//
// batch 参数为可选的批量预查缓存：
//   - 传入非 nil 时，使用预查结果避免循环内 N+1 DB 查询（admin 列表路径）
//   - 传入 nil 时，按需逐条查 DB（单实例路径如 /openclaw/status）
//
// 注意：调用方若传入的是临时拼装的 instance（而非完整 DB 记录），必须至少补齐以下字段，
// 否则会导致状态误判：
//   - ID
//   - InstanceId
//   - CurrentOperation
//   - CurrentOperationState
//   - CurrentOperationUpdatedAt
//   - LastCVMState
//   - LastStableState
//   - AgentReady
//   - CLSAgentStatus
//   - CLSAgentStatusAt
func ResolveInstanceStatus(ctx context.Context, instance *model.Instance, cvmInfo *CVMInstanceInfo, batch *InstanceStatusBatchLookup) InstanceStatusResponse {
	var status string
	var prevState = instance.LastCVMState

	// Step -1: 本地实例不走 CVM 状态机。
	// 状态仅有两种：running / stopped（阈值 LocalInstanceOfflineThreshold）。
	// 需要 reporter 拉 last_report_at 上报才能判定。
	if instance.Source == model.InstanceSourceLocal {
		if batch != nil && batch.LocalInfoMap != nil {
			return resolveLocalInstanceStatusCached(ctx, instance, batch.LocalInfoMap)
		}
		return resolveLocalInstanceStatus(ctx, instance)
	}

	if model.IsResourceAdjustmentOperation(instance.CurrentOperation) {
		return resourceAdjustmentStableStatus(ctx, instance, cvmInfo)
	}

	// Step 0: 操作失败状态判断
	// delete 操作失败不在此处处理（由 Step 1 根据 CVM 状态决定）
	// 注意：仅当 CVM 曾运行过且现已不存在（外部销毁）时，才不设置失败状态，
	// 交由 Step 2 判定为 destroyed；其余情况（含 CVM 从未创建成功、InstanceId 为空）
	// 保持 load_failed/upgrade_failed，保证用户可通过 /openclaw/retry 重试。
	if instance.CurrentOperationState == model.OpStateFailed && instance.CurrentOperation != "" {
		externallyDestroyed := cvmInfo == nil && instance.InstanceId != "" &&
			(isPostCreationCVMState(instance.LastStableState) || isPostCreationCVMState(instance.LastCVMState))
		if instance.CurrentOperation == model.OpUpgrade {
			if !externallyDestroyed {
				// 升级失败 → upgrade_failed（实例仍为 RUNNING，区别于 load_failed）
				status = model.StatusUpgradeFailed
				slog.Debug("[StateMap] Step0: 升级失败 → upgrade_failed", "id", instance.ID)
			}
			// externallyDestroyed：CVM 曾运行过但已被外部销毁，交由 Step 2 判定为 destroyed
		} else if instance.CurrentOperation != model.OpDelete && instance.CurrentOperation != model.OpMigrate {
			if !externallyDestroyed {
				status = model.StatusLoadFailed
				slog.Debug("[StateMap] Step0: 操作失败 → load_failed", "id", instance.ID, "operation", instance.CurrentOperation)
			}
			// externallyDestroyed：CVM 曾运行过但已被外部销毁，交由 Step 2 判定为 destroyed
		}
	}

	// Step 0.5: CVM API 查询失败（非"CVM 不存在"），使用 DB 缓存的历史状态兜底，不做销毁判定。
	// 场景：CVM API 返回 InternalError / 超时 / 客户端创建失败，batchFetchCVMInfoMap 标记为 API_ERROR。
	// 例外：正在执行删除操作时不兜底，让 Step 1 正常完成删除流程。
	if status == "" && cvmInfo != nil && cvmInfo.State == "API_ERROR" {
		if instance.CurrentOperation == model.OpDelete {
			cvmInfo = nil // 视同 CVM 不存在，走 Step 1 正常删除
		} else {
			// 保守返回 running，避免因 API 抖动误判为 destroyed。
			// 注意：必须从 UserStatusMap 取完整展示信息（label/tooltip/actions/transient），
			// 不能只返回裸 Status，否则前端会收到空 label / nil actions 导致 UI 异常。
			Logger(ctx).Warn("[StateMap] Step0.5: CVM API 查询失败，兜底返回 running",
				"id", instance.ID,
				"instance_id", instance.InstanceId,
				"last_cvm_state", instance.LastCVMState,
			)
			chargeType := instanceChargeTypeOrDefault(instance.InstanceChargeType)
			defaultLang := hcommon.DefaultLangFromCtx(ctx)
			if def, ok := model.UserStatusMap[model.StatusRunning]; ok {
				if defaultLang == "zh" {
					return InstanceStatusResponse{
						Status:             def.Status,
						Label:              def.Label,
						Tooltip:            def.Tooltip,
						Actions:            def.Actions,
						Transient:          def.Transient,
						InstanceChargeType: chargeType,
					}
				}
				return InstanceStatusResponse{
					Status:             def.Status,
					Label:              def.LabelEn,
					Tooltip:            def.TooltipEn,
					Actions:            def.Actions,
					Transient:          def.Transient,
					InstanceChargeType: chargeType,
				}
			}
			// 兜底：UserStatusMap 无 running 条目（理论上不会发生）
			return InstanceStatusResponse{Status: model.StatusRunning, InstanceChargeType: chargeType}
		}
	}

	// Step 1: currentOperation=delete 且 CVM 消失 → destroyed
	if status == "" && instance.CurrentOperation == model.OpDelete {
		if cvmInfo == nil || cvmInfo.State == "" || cvmInfo.State == "NOTFOUND" {
			status = model.StatusDestroyed
			slog.Debug("[StateMap] Step1: delete + CVM消失 → destroyed", "id", instance.ID)
		} else {
			status = model.StatusDestroying // 销毁进行中
			slog.Debug(
				"[StateMap] Step1: delete 进行中 → destroying",
				"id", instance.ID,
				"cvm_state", cvmInfo.State,
			)
		}
	} else if status == "" && cvmInfo == nil {
		// Step 2: CVM 实例不存在
		if instance.InstanceId == "" {
			// 2.1 DB 中 instance_id 为空 → creating
			status = model.StatusCreating
			slog.Debug("[StateMap] Step2.1: 无 CVM instance_id → creating", "id", instance.ID)
		} else if isPostCreationCVMState(instance.LastStableState) || isPostCreationCVMState(instance.LastCVMState) {
			// 2.2a DB 中有 instance_id，CVM 不存在，但实例曾经运行过 → destroyed（外部销毁）
			status = model.StatusDestroyed
			slog.Debug("[StateMap] Step2.2a: CVM 不存在但曾运行过 → destroyed（外部销毁）", "id", instance.ID,
				"last_stable_state", instance.LastStableState, "last_cvm_state", instance.LastCVMState)
		} else if isCreationPhaseCVMState(instance.LastCVMState) {
			// 2.2a2 DB 中有 instance_id，CVM 不存在，LastCVMState 仍处于创建阶段（PENDING/LAUNCHING）→ create_failed
			status = model.StatusCreateFailed
			slog.Debug("[StateMap] Step2.2a2: CVM 不存在且 LastCVMState 处于创建阶段 → create_failed", "id", instance.ID,
				"last_cvm_state", instance.LastCVMState)
		} else if instance.CurrentOperation == model.OpCreate {
			// 2.2b DB 中有 instance_id 但 CVM 不存在，且正在执行创建操作 → create_failed
			status = model.StatusCreateFailed
			slog.Debug("[StateMap] Step2.2b: CVM 不存在且创建中 → create_failed", "id", instance.ID)
		} else if instance.LastStableState == "" && instance.LastCVMState == "" {
			// 2.2b2 DB 中有 instance_id，CVM 不存在，但从未记录过任何 CVM 状态 → create_failed（从未成功运行过）
			status = model.StatusCreateFailed
			slog.Debug("[StateMap] Step2.2b2: CVM 不存在且从未有状态记录 → create_failed", "id", instance.ID)
		} else {
			// 2.2c DB 中有 instance_id，CVM 不存在，非创建中（存量数据兜底）→ destroyed
			status = model.StatusDestroyed
			slog.Debug("[StateMap] Step2.2c: CVM 不存在，非创建中（存量兜底）→ destroyed", "id", instance.ID,
				"current_operation", instance.CurrentOperation)
		}
	} else if status == "" {
		// CVM 存在，根据状态映射
		cvmState := cvmInfo.State

		// Step 2.5: RestrictState 非 NORMAL → pending（实例被隔离：过期/安全隔离等）
		// 无论 CVM InstanceState 是什么，RestrictState 非 NORMAL 都视为隔离
		if cvmInfo.RestrictState != "" && cvmInfo.RestrictState != "NORMAL" {
			status = model.StatusPending
			slog.Debug("[StateMap] Step2.5: RestrictState非NORMAL → pending", "id", instance.ID, "restrict_state", cvmInfo.RestrictState, "cvm_state", cvmState)
		} else if cvmState == "LAUNCH_FAILED" {
			// Step 3: LAUNCH_FAILED → create_failed
			status = model.StatusCreateFailed
			slog.Debug("[StateMap] Step3: LAUNCH_FAILED → create_failed", "id", instance.ID)
		} else if model.CVMLiveMigrateStates[cvmState] {
			// Step 4: 热迁移态 → running（热迁移期间实例仍在运行）
			status = model.StatusRunning
			slog.Debug("[StateMap] Step4: 热迁移 → running", "id", instance.ID, "cvm_state", cvmState)
		} else if model.CVMPlatformLimitStates[cvmState] {
			// Step 5: 平台限制态 → pending
			status = model.StatusPending
			slog.Debug("[StateMap] Step5: 平台限制态 → pending", "id", instance.ID, "cvm_state", cvmState)
		} else if model.CVMRescueModeStates[cvmState] {
			// Step 6: 维护态（救援模式）→ maintaining
			status = model.StatusMaintaining
			slog.Debug("[StateMap] Step6: 维护态 → maintaining", "id", instance.ID, "cvm_state", cvmState)
		} else if cvmState == "PENDING" {
			// Step 6.5: PENDING → creating（CVM 尚未启动，属于创建中）
			status = model.StatusCreating
			slog.Debug("[StateMap] Step6.5: PENDING → creating", "id", instance.ID)
		} else if model.CVMTransientStates[cvmState] {
			// Step 7: 其他过渡态 → loading
			status = model.StatusLoading
			slog.Debug("[StateMap] Step7: 过渡态 → loading", "id", instance.ID, "cvm_state", cvmState)
		} else if cvmState == "RUNNING" {
			// Step 8: RUNNING 态判断
			if instance.CurrentOperation == model.OpUpgrade && instance.CurrentOperationState == model.OpStateProcessing {
				// 升级进行中 → upgrading
				status = model.StatusUpgrading
				slog.Debug("[StateMap] Step8-upgrade: RUNNING + upgrade进行中 → upgrading", "id", instance.ID)
			} else if instance.CurrentOperation == model.OpMigrate && instance.CurrentOperationState == model.OpStateProcessing {
				// 迁移进行中 → loading（对外不暴露迁移状态）
				status = model.StatusLoading
				slog.Debug("[StateMap] Step8-migrate: RUNNING + migrate进行中 → loading", "id", instance.ID)
			} else if instance.AgentReady == 0 {
				// Agent 未就绪 → loading
				status = model.StatusLoading
				slog.Debug("[StateMap] Step8a: RUNNING + Agent未就绪 → loading", "id", instance.ID)
			} else if (batch != nil && isCLSPendingInstallationWithConfig(batch.SiteConfig, instance)) ||
				(batch == nil && isCLSPendingInstallation(ctx, instance)) {
				// CLS 服务已开通但 CLS Agent 未安装，且不在冷却期内 → loading
				status = model.StatusLoading
				slog.Debug("[StateMap] Step8-cls: RUNNING + CLS Agent待安装 → loading", "id", instance.ID, "cls_agent_status", instance.CLSAgentStatus, "cls_agent_status_at", instance.CLSAgentStatusAt)
			} else if (batch != nil && batch.InstallingSkillMap[instance.ID]) ||
				(batch == nil && hasInstallingSkillInstallations(ctx, instance.ID)) {
				// Agent 已就绪，但初始技能仍在安装中 → loading
				status = model.StatusLoading
				slog.Debug("[StateMap] Step8b: RUNNING + 初始技能安装中 → loading", "id", instance.ID)
			} else if hasPendingGatewayRestartTasks(instance.ID) {
				// 创建时的异步任务（技能/插件/模型/SMH）尚未完成，gateway 可能即将被重启 → loading
				status = model.StatusLoading
				slog.Debug("[StateMap] Step8c: RUNNING + 异步初始化任务进行中 → loading", "id", instance.ID)
			} else {
				// Agent 就绪 → running
				status = model.StatusRunning
				slog.Debug("[StateMap] Step8d: RUNNING + Agent就绪 → running", "id", instance.ID)
			}
		} else if cvmState == "STOPPED" {
			// Step 9: STOPPED 态判断
			// reinstall/upgrade 操作中经过 STOPPED 是正常过渡，应显示对应状态
			if instance.CurrentOperation == model.OpUpgrade && instance.CurrentOperationState == model.OpStateProcessing {
				status = model.StatusUpgrading
				slog.Debug("[StateMap] Step9-upgrade: upgrade + STOPPED → upgrading（重装过渡态）", "id", instance.ID)
			} else if instance.CurrentOperation == model.OpReinstall {
				status = model.StatusLoading
				slog.Debug("[StateMap] Step9a: reinstall + STOPPED → loading（重装过渡态）", "id", instance.ID)
			} else {
				status = model.StatusStopped
				slog.Debug("[StateMap] Step9b: STOPPED → stopped", "id", instance.ID)
			}
		} else {
			// Step 10: 未知状态 → maintaining（兜底）
			status = model.StatusMaintaining
			slog.Debug("[StateMap] Step10: 未知状态 → maintaining", "id", instance.ID, "cvm_state", cvmState)
		}
	}

	// 记录状态变化
	if prevState != cvmInfoState(cvmInfo) {
		slog.Info("[StateMap] 状态变化", "id", instance.ID, "prev_cvm", prevState, "status", status)
	}

	chargeType := instanceChargeTypeOrDefault(instance.InstanceChargeType)
	if cvmInfo != nil && cvmInfo.InstanceChargeType != "" {
		chargeType = cvmInfo.InstanceChargeType
	}

	defaultLang := hcommon.DefaultLangFromCtx(ctx)

	// 从状态映射表获取展示信息
	if def, ok := model.UserStatusMap[status]; ok {
		if defaultLang == "zh" {
			return InstanceStatusResponse{
				Status:             def.Status,
				Label:              def.Label,
				Tooltip:            def.Tooltip,
				Actions:            def.Actions,
				Transient:          def.Transient,
				InstanceChargeType: chargeType,
			}
		} else {
			return InstanceStatusResponse{
				Status:             def.Status,
				Label:              def.LabelEn,
				Tooltip:            def.TooltipEn,
				Actions:            def.Actions,
				Transient:          def.Transient,
				InstanceChargeType: chargeType,
			}
		}
	}

	// 兜底：返回 maintaining
	return InstanceStatusResponse{
		Status:             model.StatusMaintaining,
		Label:              i18n.T(ctx, i18n.MsgInstanceStatusMaintaining),
		Tooltip:            i18n.T(ctx, i18n.MsgInstanceStatusMaintainingTooltip),
		Actions:            []string{"delete"},
		Transient:          true,
		InstanceChargeType: chargeType,
	}
}

func resourceAdjustmentStableStatus(ctx context.Context, instance *model.Instance, cvmInfo *CVMInstanceInfo) InstanceStatusResponse {
	status := ""
	if cvmInfo != nil && (cvmInfo.State == "RUNNING" || cvmInfo.State == "STOPPED") {
		status = strings.ToLower(cvmInfo.State)
	}
	if status == "" {
		status = instance.LastStableState
	}
	if status != model.StatusRunning && status != model.StatusStopped {
		status = model.StatusMaintaining
	}
	chargeType := instanceChargeTypeOrDefault(instance.InstanceChargeType)
	if cvmInfo != nil && cvmInfo.InstanceChargeType != "" {
		chargeType = cvmInfo.InstanceChargeType
	}
	label := i18n.T(ctx, i18n.MsgInstanceStatusMaintaining)
	tooltip := i18n.T(ctx, i18n.MsgInstanceStatusMaintainingTooltip)
	if def, ok := model.UserStatusMap[status]; ok {
		label, tooltip = def.Label, def.Tooltip
		if hcommon.DefaultLangFromCtx(ctx) != "zh" {
			label, tooltip = def.LabelEn, def.TooltipEn
		}
	}
	return InstanceStatusResponse{
		Status:             status,
		Label:              label,
		Tooltip:            tooltip,
		Actions:            []string{},
		Transient:          false,
		InstanceChargeType: chargeType,
	}
}

func hasInstallingSkillInstallations(ctx context.Context, instanceID uint) bool {
	if instanceID == 0 || model.DB(ctx) == nil {
		return false
	}

	var count int64
	if err := model.DB(ctx).Model(&model.SkillInstallation{}).
		Where("instance_id = ? AND install_status = ?", instanceID, model.SkillInstalling).
		Limit(1).
		Count(&count).Error; err != nil {
		slog.Warn("[StateMap] 查询初始技能安装状态失败", "instance_id", instanceID, "error", err)
		return false
	}

	return count > 0
}

// batchINChunkSize 批量 IN 查询的分片大小。
// 远低于 MySQL max_prepared_stmt_count (65535) 和 max_allowed_packet (64MB) 上限，
// 同时保证单次 IN 扫描在合理时间内完成。
const batchINChunkSize = 500

// batchHasInstallingSkillInstallations 批量查询哪些实例有 installing 状态的技能安装记录。
// 返回 map[instanceID]bool，true 表示该实例存在 installing 状态的安装记录。
// 用于替代循环内逐条调用 hasInstallingSkillInstallations，消除 N+1 查询。
func batchHasInstallingSkillInstallations(ctx context.Context, instanceIDs []uint) map[uint]bool {
	result := make(map[uint]bool)
	if len(instanceIDs) == 0 || model.DB(ctx) == nil {
		return result
	}

	for _, chunk := range chunkUintIDs(instanceIDs, batchINChunkSize) {
		var installingIDs []uint
		if err := model.DB(ctx).Model(&model.SkillInstallation{}).
			Where("instance_id IN ? AND install_status = ?", chunk, model.SkillInstalling).
			Distinct("instance_id").
			Pluck("instance_id", &installingIDs).Error; err != nil {
			slog.WarnContext(ctx, "[StateMap] 批量查询初始技能安装状态失败", "error", err)
			continue
		}
		for _, id := range installingIDs {
			result[id] = true
		}
	}
	return result
}

// batchResolveLocalInstanceStatus 批量查询本地实例的 LocalInstanceInfo。
// 返回 map[instanceID]*model.LocalInstanceInfo，用于替代循环内逐条查询。
func batchResolveLocalInstanceStatus(ctx context.Context, instanceIDs []uint) map[uint]*model.LocalInstanceInfo {
	result := make(map[uint]*model.LocalInstanceInfo)
	if len(instanceIDs) == 0 || model.DB(ctx) == nil {
		return result
	}

	for _, chunk := range chunkUintIDs(instanceIDs, batchINChunkSize) {
		var infos []model.LocalInstanceInfo
		if err := model.DB(ctx).Where("instance_id IN ?", chunk).Find(&infos).Error; err != nil {
			slog.WarnContext(ctx, "[LocalInstance] 批量查询 LocalInstanceInfo 失败", "error", err)
			continue
		}
		for i := range infos {
			result[infos[i].InstanceID] = &infos[i]
		}
	}
	return result
}

// chunkUintIDs 将 ID 列表按 size 分片，避免 IN 子句超出 MySQL 参数限制。
func chunkUintIDs(ids []uint, size int) [][]uint {
	var chunks [][]uint
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[i:end])
	}
	return chunks
}

// isCLSPendingInstallationWithConfig 是 isCLSPendingInstallation 的缓存版，
// 接受预查的 SiteConfig 而非每次查 DB。
func isCLSPendingInstallationWithConfig(siteConfig model.SiteConfig, instance *model.Instance) bool {
	if instance == nil {
		return false
	}
	if siteConfig.CLSEnabled != 1 {
		return false
	}
	return instance.CLSAgentStatus == model.CLSAgentInstalling
}

// resolveLocalInstanceStatusCached 是 resolveLocalInstanceStatus 的缓存版，
// 使用预查的 localInfoMap 替代逐条 DB 查询。
// map 未命中等价于原函数的 RecordNotFound，fallback 为 stopped。
func resolveLocalInstanceStatusCached(ctx context.Context, instance *model.Instance, localInfoMap map[uint]*model.LocalInstanceInfo) InstanceStatusResponse {
	status := model.StatusStopped

	if info, ok := localInfoMap[instance.ID]; ok {
		if info.LastReportAt != nil &&
			time.Since(*info.LastReportAt) <= model.LocalInstanceOfflineThreshold {
			status = model.StatusRunning
		}
	}
	// 未命中 → stopped（等价于原函数 RecordNotFound 分支）

	resp := InstanceStatusResponse{
		Status:  status,
		Actions: []string{}, // 本地实例无可用动作
	}
	if def, ok := model.UserStatusMap[status]; ok {
		resp.Transient = def.Transient
		if hcommon.DefaultLangFromCtx(ctx) == "zh" {
			resp.Label = def.Label
			resp.Tooltip = def.Tooltip
		} else {
			resp.Label = def.LabelEn
			resp.Tooltip = def.TooltipEn
		}
	}
	return resp
}

// isCLSPendingInstallation 判断 CLS Agent 是否处于安装中（CLSAgentInstalling=2）。
// 仅当 CLS 服务已开通（site_config.cls_enabled=1）且后台任务已将实例标记为安装中时返回 true。
// status=0（未安装）的实例等待后台任务调度，不提前进入 loading，
// 避免大批量待安装实例同时显示 loading 状态。
func isCLSPendingInstallation(ctx context.Context, instance *model.Instance) bool {
	if instance == nil {
		return false
	}

	// CLS 服务未开通
	if model.GetSiteConfig(ctx).CLSEnabled != 1 {
		return false
	}

	// 只有后台任务已明确标记为"安装中"时才显示 loading
	return instance.CLSAgentStatus == model.CLSAgentInstalling
}

// isCreationPhaseCVMState 判断 CVM 状态是否仍处于创建阶段
// PENDING、LAUNCHING 表示 CVM 尚未成功启动，若此时 CVM 消失则为创建失败
func isCreationPhaseCVMState(state string) bool {
	return state == "PENDING" || state == "LAUNCHING"
}

// isPostCreationCVMState 判断 CVM 状态是否为创建后的状态
// 用于区分"从未成功创建"和"曾经运行过后被外部销毁"
func isPostCreationCVMState(state string) bool {
	// 创建后才会出现的状态（排除 PENDING、LAUNCHING、LAUNCH_FAILED 等创建阶段状态）
	postCreationStates := map[string]bool{
		"RUNNING":      true,
		"STOPPED":      true,
		"STOPPING":     true,
		"STARTING":     true,
		"REBOOTING":    true,
		"REINSTALLING": true,
		"SHUTDOWN":     true,
		"TERMINATING":  true,
	}
	return postCreationStates[state]
}

// cvmInfoState 安全获取 CVM 状态
func cvmInfoState(cvmInfo *CVMInstanceInfo) string {
	if cvmInfo == nil {
		return ""
	}
	return cvmInfo.State
}

// handleStatusSideEffects 副作用处理（同步部分）
// 注意：Agent 检测由后台 AgentChecker 异步完成，不在本函数中处理
// ctx 用于内部异步 goroutine 继承 TenantSnapshot(通过 DetachContext 派生)
func handleStatusSideEffects(ctx context.Context, db *gorm.DB, instance *model.Instance, cvmInfo *CVMInstanceInfo, status string) {
	if instance != nil && model.IsResourceAdjustmentOperation(instance.CurrentOperation) {
		return
	}
	// 使用事务确保数据一致性
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			slog.Error("[SideEffect] panic, rollback", "error", r)
		}
	}()

	var needsUpdate bool
	updates := make(map[string]interface{})

	// 1. 更新 LastCVMState 缓存（排除 API_ERROR，避免污染缓存）
	if cvmInfo != nil && cvmInfo.State != "" && cvmInfo.State != "API_ERROR" {
		updates["last_cvm_state"] = cvmInfo.State
		instance.LastCVMState = cvmInfo.State
		needsUpdate = true
	}
	if cvmInfo != nil && cvmInfo.InstanceChargeType != "" && instance.InstanceChargeType != cvmInfo.InstanceChargeType {
		updates["instance_charge_type"] = cvmInfo.InstanceChargeType
		instance.InstanceChargeType = cvmInfo.InstanceChargeType
		needsUpdate = true
	}

	// 2. 操作超时检测
	// 注意：upgrade/migrate 操作不在此处标记超时，由各自的异步 goroutine 自行管理生命周期
	// （与 Step 2.5 "操作超时自动恢复" 和 Step 3 "操作收敛" 保持一致）
	if isOperationTimedOut(instance) &&
		instance.CurrentOperation != model.OpUpgrade &&
		instance.CurrentOperation != model.OpMigrate {
		updates["current_operation_state"] = model.OpStateFailed
		now := time.Now()
		updates["current_operation_updated_at"] = &now
		instance.CurrentOperationState = model.OpStateFailed
		instance.CurrentOperationUpdatedAt = &now
		needsUpdate = true
		slog.Info("[SideEffect] 操作超时", "id", instance.ID, "operation", instance.CurrentOperation)
	}

	// 2.5 操作超时自动恢复：清除 OpStateFailed，避免实例永久卡在 load_failed
	// 仅处理会映射为 load_failed 的操作；delete/upgrade/migrate 保持各自专用失败语义。
	// Case A: CVM RUNNING + Agent 就绪 + CLS 正常 → 恢复为 running
	// Case B: CVM 已正常关机（STOPPED）→ 收敛为 stopped
	isRecoverableLoadFailedOp := instance.CurrentOperation != "" &&
		instance.CurrentOperation != model.OpDelete &&
		instance.CurrentOperation != model.OpUpgrade &&
		instance.CurrentOperation != model.OpMigrate
	if isRecoverableLoadFailedOp && instance.CurrentOperationState == model.OpStateFailed && cvmInfo != nil && cvmInfo.State != "API_ERROR" {
		cvmState := cvmInfo.State
		prevOp := instance.CurrentOperation
		recovered := false

		if cvmState == "RUNNING" &&
			instance.AgentReady == 1 &&
			(instance.CLSAgentStatus == model.CLSAgentInstalled || instance.CLSAgentStatus == model.CLSAgentNotInstalled) {
			// Case A: CVM 正常运行且 Agent 就绪
			now := time.Now()
			updates["current_operation"] = model.OpNone
			updates["current_operation_state"] = model.OpStateSuccess
			updates["current_operation_updated_at"] = &now
			updates["last_stable_state"] = "RUNNING"
			instance.CurrentOperation = model.OpNone
			instance.CurrentOperationState = model.OpStateSuccess
			instance.CurrentOperationUpdatedAt = &now
			instance.LastStableState = "RUNNING"
			recovered = true
			slog.Info("[SideEffect] 操作超时自动恢复(Case A)：CVM RUNNING + Agent就绪",
				"id", instance.ID,
				"operation", prevOp,
				"agent_ready", instance.AgentReady,
				"cls_status", instance.CLSAgentStatus,
			)
		} else if cvmState == "STOPPED" {
			// Case B: CVM 已正常关机，操作已无意义，收敛为 stopped
			now := time.Now()
			updates["current_operation"] = model.OpNone
			updates["current_operation_state"] = model.OpStateSuccess
			updates["current_operation_updated_at"] = &now
			updates["last_stable_state"] = cvmState
			instance.CurrentOperation = model.OpNone
			instance.CurrentOperationState = model.OpStateSuccess
			instance.CurrentOperationUpdatedAt = &now
			instance.LastStableState = cvmState
			recovered = true
			slog.Info("[SideEffect] 操作超时自动恢复(Case B)：CVM 已正常关机",
				"id", instance.ID,
				"operation", prevOp,
				"cvm_state", cvmState,
			)
		}

		if recovered {
			needsUpdate = true
		}
	}

	// 3. 操作收敛（非 RUNNING 稳定态）
	// RUNNING 态的收敛由后台 AgentChecker 完成
	// 注意：delete 操作不在此处收敛（由 Step 4 删除确认处理）
	// 注意：upgrade 操作不在此处收敛（由 performUpgrade 异步流程自行管理锁的释放）
	// 注意：已超时（failed）的操作不再覆盖为 success
	if instance.CurrentOperation != "" &&
		instance.CurrentOperation != model.OpDelete &&
		instance.CurrentOperation != model.OpUpgrade &&
		instance.CurrentOperation != model.OpMigrate &&
		instance.CurrentOperationState != model.OpStateFailed &&
		cvmInfo != nil && cvmInfo.State != "API_ERROR" {
		cvmState := cvmInfo.State
		// 非过渡态且非 RUNNING → 收敛
		if !model.CVMTransientStates[cvmState] && cvmState != "RUNNING" {
			now := time.Now()
			updates["current_operation"] = model.OpNone
			updates["current_operation_state"] = model.OpStateSuccess
			updates["current_operation_updated_at"] = &now
			if instance.LastCVMState != "" {
				updates["last_stable_state"] = instance.LastCVMState
			}
			prevOp := instance.CurrentOperation
			instance.CurrentOperation = model.OpNone
			instance.CurrentOperationState = model.OpStateSuccess
			instance.CurrentOperationUpdatedAt = &now
			needsUpdate = true
			slog.Info("[SideEffect] 操作收敛", "id", instance.ID, "operation", prevOp, "cvm_state", cvmState)
		}
	}

	// 批量更新
	if needsUpdate && len(updates) > 0 {
		if err := tx.Model(instance).Updates(updates).Error; err != nil {
			tx.Rollback()
			slog.Warn("[SideEffect] 更新失败，rollback", "id", instance.ID, "error", err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Warn("[SideEffect] commit 失败", "id", instance.ID, "error", err)
		return
	}

	// 4. 删除确认 + Purge（事务完成后异步执行）
	if instance.CurrentOperation == model.OpDelete {
		if cvmInfo == nil {
			// CVM 已消失：跳过本地导出，直接尽力释放远端 Pro 记忆库；失败则保留 DB 记录等待后续补偿
			go func(ctx context.Context, inst model.Instance) {
				if !ReleaseProMemSpaceForMissingInstance(ctx, inst.InstanceId) {
					slog.Warn("[SideEffect] Pro 记忆库释放失败，保留 DB 记录等待后续补偿",
						"id", inst.ID, "instance_id", inst.InstanceId)
					return
				}
				slog.Info("[SideEffect] CVM 已消失，清理 DB 记录", "id", inst.ID)
				model.DB(ctx).Where("instance_id = ?", inst.ID).Delete(&model.SkillInstallation{})
				if err := model.DB(ctx).Delete(&inst).Error; err != nil {
					slog.Error("[SideEffect] 清理 DB 记录失败，将在下次轮询重试", "id", inst.ID, "error", err)
				}
				model.DeleteMemoryTDAIPluginRow(ctx, inst.InstanceId)
				MarkPersonalSpaceToBeDeleted(ctx, inst.ID)
			}(hcommon.DetachContext(ctx), *instance)
		} else if cvmInfo.RestrictState != "" && cvmInfo.RestrictState != "NORMAL" {
			// CVM 已隔离（RestrictState 非 NORMAL）→ TAT 大概率不可达，跳过导出，直接尽力释放远端 VDB
			go func(ctx context.Context, inst model.Instance, restrictState string) {
				slog.Info("[SideEffect] 异步 Purge 开始（RestrictState 非 NORMAL）", "instance_id", inst.InstanceId, "restrict_state", restrictState)
				if !ReleaseProMemSpaceForMissingInstance(ctx, inst.InstanceId) {
					slog.Warn("[SideEffect] Pro 记忆库释放失败，跳过 Purge 并保留 DB 记录等待后续补偿",
						"instance_id", inst.InstanceId, "restrict_state", restrictState)
					return
				}

				// 带重试的 Purge 操作
				var purgeErr error
				const maxRetries = 3
				for attempt := 1; attempt <= maxRetries; attempt++ {
					if purgeErr = destroyCVMInstance(ctx, inst.InstanceId); purgeErr == nil {
						slog.Info("[SideEffect] Purge 成功，清理 DB 记录", "instance_id", inst.InstanceId)
						break
					}
					slog.Warn("[SideEffect] Purge 失败，准备重试",
						"instance_id", inst.InstanceId, "attempt", attempt, "max_retries", maxRetries, "error", purgeErr)
					if attempt < maxRetries {
						time.Sleep(time.Duration(attempt) * 2 * time.Second) // 退避重试：2s, 4s
					}
				}

				if purgeErr != nil {
					slog.Error("[SideEffect] Purge 重试耗尽，仍清理 DB 记录（CVM 已隔离，资源已释放）",
						"instance_id", inst.InstanceId, "error", purgeErr)
				}
				// 无论 Purge 成功与否，均清 DB，避免每次轮询重复触发 Purge
				model.DB(ctx).Where("instance_id = ?", inst.ID).Delete(&model.SkillInstallation{})
				if err := model.DB(ctx).Delete(&inst).Error; err != nil {
					slog.Error("[SideEffect] Purge 后清理 DB 记录失败，将在下次轮询重试", "instance_id", inst.InstanceId, "error", err)
				}
				model.DeleteMemoryTDAIPluginRow(ctx, inst.InstanceId)
				MarkPersonalSpaceToBeDeleted(ctx, inst.ID)
			}(hcommon.DetachContext(ctx), *instance, cvmInfo.RestrictState)
		}
	}

	// 5. 外部感知（currentOp 空 + CVM 不存在 → 外部销毁）
	if instance.CurrentOperation == "" && cvmInfo == nil && instance.InstanceId != "" {
		slog.Info("[SideEffect] 外部销毁感知", "id", instance.ID, "instance_id", instance.InstanceId)
		// 幂等检查：避免重复发送外部销毁通知
		var existCount int64
		model.DB(ctx).Model(&model.Notification{}).
			Where("instance_id = ? AND type = ?", instance.ID, model.NotifyTypeExternalDestroy).
			Count(&existCount)
		if existCount == 0 {
			go func(ctx context.Context, inst model.Instance) {
				if err := model.CreateNotification(
					ctx,
					inst.UserID, inst.ID, inst.Name,
					model.NotifyTypeExternalDestroy,
					i18n.T(ctx, i18n.MsgInstanceExternallyDestroyedTitle),
					instanceDestroyMessage(ctx, inst.Name),
				); err != nil {
					slog.Warn("[SideEffect] 创建外部销毁通知失败", "id", inst.ID, "error", err)
				}
			}(hcommon.DetachContext(ctx), *instance)
		} else {
			slog.Debug("[SideEffect] 外部销毁通知已存在，跳过", "id", instance.ID)
		}
	}

	// 6. 外部感知（currentOp 空 + CVM 过渡态）
	if instance.CurrentOperation == "" && cvmInfo != nil && model.CVMTransientStates[cvmInfo.State] {
		slog.Info("[SideEffect] 外部操作感知",
			"id", instance.ID,
			"cvm_state", cvmInfo.State,
			"latest_operation", cvmInfo.LatestOperation,
		)
	}
}

// fetchCVMInstanceInfo 查询 CVM 实例信息
func fetchCVMInstanceInfo(ctx context.Context, instanceId string) (*CVMInstanceInfo, error) {
	if instanceId == "" {
		return nil, nil
	}

	client, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return nil, rerr
	}

	req := cvm.NewDescribeInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instanceId})

	var resp *cvm.DescribeInstancesResponse
	err := RetryCloudCall(ctx, func() error {
		var callErr error
		resp, callErr = client.DescribeInstances(req)
		return callErr
	})
	if err != nil {
		// 实例不存在
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
			if sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
				return nil, nil
			}
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryCVMInstanceFailed)
	}

	if resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
		return nil, nil
	}

	inst := resp.Response.InstanceSet[0]
	return cvmInstanceInfoFromSDK(inst), nil
}

// extractCVMTags 从 CVM SDK 的 Tag 指针切片中提取标签键值对
func extractCVMTags(sdkTags []*cvm.Tag) []CVMTag {
	tags := make([]CVMTag, 0, len(sdkTags))
	for _, t := range sdkTags {
		if t != nil && t.Key != nil && t.Value != nil {
			tags = append(tags, CVMTag{Key: *t.Key, Value: *t.Value})
		}
	}
	return tags
}

// destroyCVMInstance 彻底销毁已隔离的 CVM 实例
// 对已隔离（RestrictState 非 NORMAL）的实例调用 TerminateInstances 即为彻底销毁（跳过回收站直接释放）
// 注意：对正常运行中的实例调用 TerminateInstances 是「退还实例」（进入回收站）
func destroyCVMInstance(ctx context.Context, instanceId string) error {
	client, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return rerr
	}

	req := cvm.NewTerminateInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instanceId})

	_, err := client.TerminateInstances(req)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
			// 实例已不存在，视为成功
			if sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
				return nil
			}
		}
		return hcommon.I18nRichError(err, i18n.MsgInstanceStateDestroyCVMFailed)
	}
	return nil
}

// instanceDestroyMessage 生成外部销毁通知消息
func instanceDestroyMessage(ctx context.Context, name string) string {
	return i18n.T(ctx, i18n.MsgInstanceExternallyDestroyed, name)
}

// resolveLocalInstanceStatus 派生本地实例状态。
//
// 本地 agent 实例只有「心跳活」/「心跳丢」两种状态，复用 CVM 现有状态枚举：
//   - 心跳活　→ StatusRunning  （运行中）
//   - 心跳丢　→ StatusStopped  （已关机/丢联统一表示）
//
// 判定规则：
//   - 无 LocalInstanceInfo 记录或 last_report_at 为空 → stopped
//   - last_report_at 距今超过 LocalInstanceOfflineThreshold → stopped
//   - 否则 → running
//
// label/tooltip 从 UserStatusMap 查表，与 CVM 实例一致；但 actions 不能直接套用
// CVM 的列表——hatchery 无法对本地机器远程 reboot/reinstall/start/terminal，这些动作被裁
// 剔；delete 也由 hatchery 接管不了（应该从用户本地卸载 agent，hatchery 只能被动
// 运 reporter 超时后清理），所以 actions 返回空列表。
func resolveLocalInstanceStatus(ctx context.Context, instance *model.Instance) InstanceStatusResponse {
	status := model.StatusStopped

	var info model.LocalInstanceInfo
	err := model.DB(ctx).Where("instance_id = ?", instance.ID).First(&info).Error
	switch {
	case err == nil:
		if info.LastReportAt != nil &&
			time.Since(*info.LastReportAt) <= model.LocalInstanceOfflineThreshold {
			status = model.StatusRunning
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 首次上报前尚无 LocalInstanceInfo 行（或已被清理），正常回退 stopped。
	default:
		// DB 抛非预期错误（非 record-not-found）：不阻断接口，仍回 fallback stopped，
		// 但记录 warn 方便排查 DB 抖动 / 连接孤立环境。
		slog.WarnContext(ctx, "[LocalInstance] 查询 LocalInstanceInfo 失败，fallback 为 stopped",
			"instance_id", instance.ID,
			"error", err)
	}

	resp := InstanceStatusResponse{
		Status:  status,
		Actions: []string{}, // 本地实例无可用动作：hatchery 无法远程 reboot/reinstall/start/terminal/delete
	}
	if def, ok := model.UserStatusMap[status]; ok {
		resp.Transient = def.Transient
		if hcommon.DefaultLangFromCtx(ctx) == "zh" {
			resp.Label = def.Label
			resp.Tooltip = def.Tooltip
		} else {
			resp.Label = def.LabelEn
			resp.Tooltip = def.TooltipEn
		}
	}
	return resp
}
