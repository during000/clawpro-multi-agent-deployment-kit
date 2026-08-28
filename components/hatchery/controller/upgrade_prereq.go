package controller

import (
	"context"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// upgradePrereqOutcome 是「升级前置检查」的统一返回结果。
//
// 设计目的：
//   - 单实例升级（HandleUpgrade）和批量升级（HandleAdminBatchUpgrade）需要执行
//     一组相同的实例级本地检查与预处理（拒绝官方镜像降级 / providerKeys / 防重入 /
//     SupportsUpgrade / 官方镜像 runtime_user 校正 等）。过去每新增一项，都要在两处
//     分别 inline 一次，极易遗漏其中一侧。
//   - 抽取统一入口 prepareInstanceForUpgrade 后，两处统一调用，本结构用于把
//     「执行结果」无歧义地交回调用方，由调用方自行决定如何回写响应：
//   - 单实例：根据 HTTPCode + Err 调 writeError；
//   - 批量：根据 BatchStatus + Err.Error() 拼一项 upgradeResult。
//
// 说明：本结构不包含「状态准入（必须 running）」与「龙虾医生节点」检查。
// 原因是这两项在两个入口的执行时机/数据来源/响应格式均不同：
//   - 单实例走 requireInstanceRunning(resolver)，并在失败时使用 writeAgentGuardError；
//   - 批量复用已批量查好的 cvmInfoMap，并使用 agentStatusRejectMessage。
//
// 强行抽到一起反而会把 cvmInfoMap、resolver 等参数挤进通用函数，得不偿失。
type upgradePrereqOutcome struct {
	// OK 为 true 表示全部通过；false 时下方字段才有意义。
	OK bool

	// Err 为不通过时的原始错误，单实例直接交给 writeError，批量取 Err.Error() 写入 Message。
	Err error

	// HTTPCode 为单实例侧 writeError 应使用的 HTTP 状态码。
	HTTPCode int

	// BatchStatus 为批量侧 upgradeResult.Status，取值 "failed" / "skipped"，
	// 与现有批量入口保持一致。
	BatchStatus string
}

// prepareInstanceForUpgrade 是单实例 / 批量升级共用的「实例级前置入口」。
// 调用方只需根据返回的 upgradePrereqOutcome 输出错误。
//
//	【只读检查】（任一失败即返回）
//	 0. rejectDowngradeOnOfficialImage —— agent_version 高于目标镜像版本时拒绝降级。
//	    覆盖官方公共镜像和自定义启用镜像，文案在被调函数内区分。
//	    失败语义：版本判定为降级；HTTP 400；批量记 failed。
//	 1. checkConfigProviderKeys —— agent 配置中 provider key 合法性校验（按 agent_type 分派）。
//	    失败语义：实例当前配置不允许升级；HTTP 400；批量记 failed。
//	 2. 防重入 —— current_operation 正在执行时拒绝。
//	    失败语义：实例上有正在执行的操作；HTTP 409；批量记 skipped。
//	 3. checkInstanceSupportsUpgrade —— agent_type 是否启用一键升级。
//	    失败语义：该类型不支持一键升级；HTTP 400；批量记 failed。
//
//	【预处理写动作】（只读检查全部通过后执行）
//	 4. ensureOfficialImageRuntimeUserForUpgrade —— 官方公共镜像强制校正 runtime_user 为 root，
//	    避免重装后 restore 脚本以不存在的用户身份执行。
//	    失败语义：DB 写失败；HTTP 500；批量记 failed。
//
// 参数：
//   - defaultImage: 本次升级的目标镜像，允许为 nil（内部做 nil 兜底）。
//   - logPrefix: 日志前缀，建议传 "[Upgrade]" / "[BatchUpgrade]"。
//
// 后续新增「单实例与批量都要做的实例级前置项」时，请直接在本函数内追加。
func prepareInstanceForUpgrade(ctx context.Context, instance *model.Instance, defaultImage *model.AIImage, logPrefix string) upgradePrereqOutcome {
	// 0) 拒绝降级：agent_version 高于目标镜像版本时拒绝升级。
	//    覆盖官方镜像和自定义镜像两类来源，文案在被调函数内区分。
	if rerr := rejectDowngradeOnOfficialImage(ctx, instance, defaultImage); rerr != nil {
		return upgradePrereqOutcome{
			OK:          false,
			Err:         rerr,
			HTTPCode:    http.StatusBadRequest,
			BatchStatus: "failed",
		}
	}

	// 1) agent 配置检查（provider keys 合法性，按 agent_type 内部分派）
	if rerr := checkOpenclawConfigProviderKeys(ctx, instance); rerr != nil {
		return upgradePrereqOutcome{
			OK:          false,
			Err:         rerr,
			HTTPCode:    http.StatusBadRequest,
			BatchStatus: "failed",
		}
	}

	// 2) 防重入：升级 / 重装 等操作进行中时拒绝
	if instance.CurrentOperation != "" && instance.CurrentOperationState == model.OpStateProcessing {
		return upgradePrereqOutcome{
			OK:          false,
			Err:         hcommon.I18nError(i18n.MsgInstanceOperationInProgress, instance.CurrentOperation),
			HTTPCode:    http.StatusConflict,
			BatchStatus: "skipped",
		}
	}

	// 3) 仅 SupportsUpgrade=true 的类型允许一键升级
	if err := checkInstanceSupportsUpgrade(ctx, instance); err != nil {
		return upgradePrereqOutcome{
			OK:          false,
			Err:         hcommon.EnsureRichErrorOrPanic(err),
			HTTPCode:    http.StatusBadRequest,
			BatchStatus: "failed",
		}
	}

	// 4) 官方镜像 runtime_user 强制校正为 root（写 DB，在所有只读检查通过后执行）。
	var targetImageId string
	if defaultImage != nil {
		targetImageId = defaultImage.ImageId
	}
	if rerr := ensureOfficialImageRuntimeUserForUpgrade(ctx, instance, targetImageId, logPrefix); rerr != nil {
		return upgradePrereqOutcome{
			OK:          false,
			Err:         rerr,
			HTTPCode:    http.StatusInternalServerError,
			BatchStatus: "failed",
		}
	}

	// 5) 磁盘空间预探测（空间不足时直接拒绝，TAT 错误 / unknown 放行）。
	if outcome := precheckUpgradeDiskSpaceStep(ctx, instance, logPrefix); !outcome.OK {
		return outcome
	}

	return upgradePrereqOutcome{OK: true}
}

// startUpgradeOutcome 是「启动升级」入口的统一返回结果。
//
// 与 upgradePrereqOutcome 的分工：
//   - prepareInstanceForUpgrade 负责「能不能升」的实例级前置检查（providerKeys /
//     防重入 / SupportsUpgrade / runtime_user 校正）。
//   - startUpgradeForInstance 负责「现在就开始升」的执行链路（needUpgrade 判定 /
//     设置操作锁 / 启动异步 performUpgrade goroutine）。
//
// 调用方按下表读取本结构：
//
//	Started=true                  → 升级已成功提交，单实例返回 200 "升级已开始"，
//	                                批量记 started + "升级已开始"。
//	AlreadyLatest=true            → 已是最新版本，单实例返回 200 "实例已是最新版本，无需升级"，
//	                                批量记 skipped + "实例已是最新版本，无需升级"。
//	Started=false && Err!=nil     → 失败，单实例 writeError(HTTPCode, Err)，
//	                                批量记 BatchStatus + Err.Error()。
type startUpgradeOutcome struct {
	// Started 为 true 表示已通过所有判定、设置了操作锁、并启动了异步升级 goroutine。
	Started bool

	// AlreadyLatest 为 true 表示 checkNeedsUpgrade 判定无需升级（镜像/版本一致），
	// 调用方应作为「成功但未启动」的特例处理（单实例 200，批量 skipped）。
	AlreadyLatest bool

	// Err 为执行链路失败时的原始错误。Started/AlreadyLatest 任一为 true 时 Err 必为 nil。
	Err error

	// HTTPCode 为单实例侧 writeError 应使用的 HTTP 状态码。
	HTTPCode int

	// BatchStatus 为批量侧 upgradeResult.Status，取值 "failed" / "skipped"，
	// 与现有批量入口保持一致。
	BatchStatus string

	// CurrentImageId / TargetImageId 仅供调用方日志使用，函数内部已打过日志。
	CurrentImageId string
	TargetImageId  string
}

// startUpgradeForInstance 在 prepareInstanceForUpgrade 通过后，统一负责「启动升级」三件套：
//  1. checkNeedsUpgrade —— 判定镜像/版本是否真的需要升级（不需要时短路返回 AlreadyLatest）。
//  2. setOperation       —— 在 DB 上设置 OpUpgrade 操作锁，确保状态查询能立即反映"升级中"。
//  3. go performUpgrade  —— 在 hcommon.DetachContext 之上启动异步升级 goroutine。
//
// 参数：
//   - defaultImage: prepareInstanceForUpgrade 之前已查到的目标镜像，必须非 nil。
//   - cvmInfoMap:   批量入口已查好的 instanceId → CVMInstanceInfo 映射；
//     单实例入口直接传 nil，函数内部会让 checkNeedsUpgrade 自查 CVM。
//   - logPrefix:    内部日志前缀，建议传 "[Upgrade]" / "[BatchUpgrade]"，与入口一致便于排查。
//
// 后续如要修改「升级启动顺序 / 操作锁策略 / 异步上下文派生方式」，请只改本函数，
// 单实例和批量入口将自动同步生效。
func startUpgradeForInstance(
	ctx context.Context,
	instance *model.Instance,
	defaultImage *model.AIImage,
	cvmInfoMap map[string]*CVMInstanceInfo,
	logPrefix string,
) startUpgradeOutcome {
	log := Logger(ctx)

	// 1) 判定是否需要升级。批量侧传入预查的 cvmInfoMap 复用，单实例侧传 nil 让其自查。
	var (
		instanceImageId string
		needUpgrade     bool
		rerr            error
	)
	if cvmInfoMap != nil {
		instanceImageId, needUpgrade, rerr = checkNeedsUpgrade(ctx, instance, defaultImage, cvmInfoMap)
	} else {
		instanceImageId, needUpgrade, rerr = checkNeedsUpgrade(ctx, instance, defaultImage)
	}
	if rerr != nil {
		log.Error(logPrefix+" 检查升级状态失败",
			"instance_id", instance.InstanceId, "db_id", instance.ID, "error", rerr)
		return startUpgradeOutcome{
			Err:           hcommon.I18nRichError(rerr, i18n.MsgUpgradeCheckUpgradeStatusFailed),
			HTTPCode:      http.StatusInternalServerError,
			BatchStatus:   "failed",
			TargetImageId: defaultImage.ImageId,
		}
	}
	log.Info(logPrefix+" 升级检查结果",
		"instanceId", instance.InstanceId,
		"currentImageId", instanceImageId,
		"targetImageId", defaultImage.ImageId,
		"needUpgrade", needUpgrade)

	if !needUpgrade {
		return startUpgradeOutcome{
			AlreadyLatest:  true,
			CurrentImageId: instanceImageId,
			TargetImageId:  defaultImage.ImageId,
		}
	}

	// 2) 设置操作锁。失败一律视为冲突类（HTTP 409 / 批量 failed）。
	if rerr := setOperation(model.DB(ctx), instance, model.OpUpgrade); rerr != nil {
		log.Warn(logPrefix+" 设置操作锁失败",
			"instance_id", instance.InstanceId, "db_id", instance.ID, "error", rerr)
		return startUpgradeOutcome{
			Err:            hcommon.I18nRichError(rerr, i18n.MsgFailedToSetUpgradeOpLock),
			HTTPCode:       http.StatusConflict,
			BatchStatus:    "failed",
			CurrentImageId: instanceImageId,
			TargetImageId:  defaultImage.ImageId,
		}
	}

	// 3) 启动异步升级 goroutine。使用 DetachContext 让升级生命周期与 HTTP 请求脱钩。
	log.Info(logPrefix+" 启动异步升级",
		"instance_id", instance.InstanceId, "db_id", instance.ID,
		"agent_type", instance.AgentType,
		"current_image", instanceImageId, "target_image", defaultImage.ImageId)
	go func(ctx context.Context, inst *model.Instance, target, current string) {
		if err := startUpgradePerformFn(ctx, inst, target, current); err != nil {
			log.Error(logPrefix+" 异步升级失败",
				"instance_id", inst.InstanceId, "error", err)
		} else {
			log.Info(logPrefix+" 异步升级成功", "instance_id", inst.InstanceId)
		}
	}(hcommon.DetachContext(ctx), instance, defaultImage.ImageId, instanceImageId)
	clearAdjustmentFailure(ctx, instance.ID)

	return startUpgradeOutcome{
		Started:        true,
		CurrentImageId: instanceImageId,
		TargetImageId:  defaultImage.ImageId,
	}
}

// startUpgradePerformFn 是 startUpgradeForInstance 启动异步升级 goroutine 时
// 实际执行升级动作的钩子，默认指向 performUpgrade。
//
// 抽出此变量的唯一目的是让单测可以替换它，避免测试用例（例如
// TestStartUpgradeForInstance_StartedAndOperationLocked）在断言"操作锁已设置 +
// 返回 Started"后，让真实的 performUpgrade 在后台 goroutine 里继续访问
// notifications / site_configs / skill_installations 等表 —— 那些表在多数
// 单测的内存 SQLite 中并未迁移，且 goroutine 可能跨越测试边界，把 SQL 落到
// 下一个测试的 DB 上，造成测试间相互污染、出现 "no such table" 的红线。
//
// 注意：本钩子仅作用于 startUpgradeForInstance 启动的异步路径。
// openclaw_upgrade.go 内 performUpgrade 在其他场景的直调（HandleUpgrade、
// performUpgradeResume 失败降级等）不经过此钩子，行为保持不变。
var startUpgradePerformFn = performUpgrade
