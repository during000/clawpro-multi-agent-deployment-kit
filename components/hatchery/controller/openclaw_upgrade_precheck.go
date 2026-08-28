package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// 升级前通过 TAT 执行 precheck_upgrade_space.sh 探测 HOME 目录剩余空间是否足够存放压缩包。
//   - 前置（prepareInstanceForUpgrade step 5）：不足 → HTTP 400 拒绝，实例状态不变；
//   - 异步（backupAndUploadToSMH 打包前二次探测）：不足 → errUpgradeAborted 中止，清操作锁 + 发通知。
// TAT 通道错误 / unknown 一律放行，由后续实际步骤兜底。

// diskPrecheckResult 来自 precheck_upgrade_space.sh 的 KV 解析结果。
type diskPrecheckResult struct {
	SourceKB    int64  // exclude 后的状态目录估算大小（KB）
	EstimatedKB int64  // 压缩后估算大小（KB, source × 40%）
	RequiredKB  int64  // 所需总空间（KB, estimated × 1.5）
	HomeAvailKB int64  // HOME 目录剩余（KB）
	HomeFS      string // HOME 目录文件系统类型（ext4/xfs/...）
	Result      string // ok | insufficient | unknown
	Reason      string // Result != ok 时的原因
	RawOutput   string // 原始脚本输出，仅 debug 用
}

// OK 仅 insufficient 返回 false，unknown 和其他值放行。
func (r *diskPrecheckResult) OK() bool {
	return r == nil || r.Result != "insufficient"
}

// HumanRequired 返回所需空间的可读文本（如 "512 MB"）。
func (r *diskPrecheckResult) HumanRequired() string {
	if r == nil {
		return ""
	}
	return humanKB(r.RequiredKB)
}

// HumanHomeAvail 返回 HOME 目录剩余空间的人类可读字符串。
func (r *diskPrecheckResult) HumanHomeAvail() string {
	if r == nil {
		return ""
	}
	return humanKB(r.HomeAvailKB)
}

// errUpgradeAborted 表示因外部条件不足中止升级（如磁盘空间不足）。
// finalizeUpgradeResult 通过 errors.As 识别，走清操作锁 + 发通知分支，不写 state=failed。
// 仅异步阶段（backupAndUploadToSMH 二次探测）产生；前置阶段直接返回 HTTP 400。
type errUpgradeAborted struct {
	Reason  string               // 短原因，英文，用于日志
	UserMsg string               // 面向用户的完整文案（已渲染 i18n）
	Detail  *diskPrecheckResult  // 探测详情，可为 nil
}

// Error 实现 error 接口。返回英文 Reason，避免污染日志中的关键字过滤。
func (e *errUpgradeAborted) Error() string {
	if e == nil {
		return "upgrade aborted"
	}
	return "upgrade aborted: " + e.Reason
}

// isUpgradeAbortedErr 判断 err 是否为 errUpgradeAborted（支持 %w 包装）。
func isUpgradeAbortedErr(err error) (*errUpgradeAborted, bool) {
	if err == nil {
		return nil, false
	}
	var aborted *errUpgradeAborted
	if errors.As(err, &aborted) {
		return aborted, true
	}
	return nil, false
}

// precheckUpgradeDiskSpace 通过 TAT 执行 precheck_upgrade_space.sh，解析 KV 输出。
// err != nil 时调用方应放行（由后续步骤兜底），不阻断升级。
var precheckUpgradeDiskSpace = func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
	if instance == nil || instance.InstanceId == "" {
		return nil, errors.New("precheckUpgradeDiskSpace: instance/InstanceId is empty")
	}
	output, err := runScriptFn(ctx, instance.InstanceId, "precheck_upgrade_space.sh", 60, instance.RuntimeUser, nil, nil)
	if err != nil {
		return nil, err
	}
	res := parseDiskPrecheckOutput(output)
	return res, nil
}

// parseDiskPrecheckOutput 解析 precheck_upgrade_space.sh 的 KV 输出。
func parseDiskPrecheckOutput(output string) *diskPrecheckResult {
	res := &diskPrecheckResult{RawOutput: output, Result: "unknown"}
	if output == "" {
		return res
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "PRECHECK_") {
			continue
		}
		// 拆分 "KEY:VALUE"，只按第一个冒号切分（值内可能含冒号）
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := line[idx+1:]
		switch key {
		case "PRECHECK_SOURCE_KB":
			res.SourceKB = parseInt64(val)
		case "PRECHECK_ESTIMATED_KB":
			res.EstimatedKB = parseInt64(val)
		case "PRECHECK_REQUIRED_KB":
			res.RequiredKB = parseInt64(val)
		case "PRECHECK_HOME_AVAIL_KB":
			res.HomeAvailKB = parseInt64(val)
		case "PRECHECK_HOME_FS":
			res.HomeFS = val
		case "PRECHECK_RESULT":
			// 仅接受已知值，其它一律视为 unknown
			switch val {
			case "ok", "insufficient", "unknown":
				res.Result = val
			default:
				res.Result = "unknown"
			}
		case "PRECHECK_REASON":
			res.Reason = val
		}
	}
	return res
}

// parseInt64 安全解析字符串为 int64，失败返回 0。
func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// humanKB 把 KB 数值格式化为可读字符串（KB/MB/GB）。
func humanKB(kb int64) string {
	if kb <= 0 {
		return "0 KB"
	}
	if kb < 1024 {
		return fmt.Sprintf("%d KB", kb)
	}
	if kb < 1024*1024 {
		// MB, 保留一位小数
		return fmt.Sprintf("%.1f MB", float64(kb)/1024.0)
	}
	// GB
	return fmt.Sprintf("%.2f GB", float64(kb)/(1024.0*1024.0))
}

// precheckUpgradeDiskSpaceStep 是 prepareInstanceForUpgrade 的 step 5。
// insufficient → HTTP 400 / batch=failed；unknown / TAT 错误 → 放行。
func precheckUpgradeDiskSpaceStep(ctx context.Context, instance *model.Instance, logPrefix string) upgradePrereqOutcome {
	log := Logger(ctx)
	res, err := precheckUpgradeDiskSpace(ctx, instance)
	if err != nil {
		// TAT 通道错误：只记 warn，放行
		log.Warn(logPrefix+" 磁盘空间预探测通道错误，放行让后续兜底",
			"instance_id", instance.InstanceId, "error", err)
		return upgradePrereqOutcome{OK: true}
	}
	if res == nil || res.OK() {
		if res != nil {
			log.Info(logPrefix+" 磁盘空间预探测通过",
				"instance_id", instance.InstanceId,
				"source_kb", res.SourceKB,
				"estimated_kb", res.EstimatedKB,
				"required_kb", res.RequiredKB,
				"home_avail_kb", res.HomeAvailKB,
				"home_fs", res.HomeFS,
				"result", res.Result)
		}
		return upgradePrereqOutcome{OK: true}
	}
	// insufficient：直接拒绝
	log.Warn(logPrefix+" 磁盘空间预探测不足，拒绝升级",
		"instance_id", instance.InstanceId,
		"source_kb", res.SourceKB,
		"estimated_kb", res.EstimatedKB,
		"required_kb", res.RequiredKB,
		"home_avail_kb", res.HomeAvailKB,
		"home_fs", res.HomeFS,
		"reason", res.Reason)
	return upgradePrereqOutcome{
		OK:          false,
		Err:         hcommon.I18nError(i18n.MsgUpgradePrecheckDiskInsufficient, res.HumanRequired(), res.HumanHomeAvail()),
		HTTPCode:    http.StatusBadRequest,
		BatchStatus: "failed",
	}
}

// buildAbortedByDiskInsufficient 根据探测结果构造 errUpgradeAborted。
// 调用方（backupAndUploadToSMH 二次探测）拿到后直接 return，由 finalizeUpgradeResult 处理。
func buildAbortedByDiskInsufficient(ctx context.Context, res *diskPrecheckResult) *errUpgradeAborted {
	msg := i18n.T(ctx, i18n.MsgUpgradeAbortedDiskInsufficient, res.HumanRequired(), res.HumanHomeAvail())
	return &errUpgradeAborted{
		Reason:  "disk_insufficient",
		UserMsg: msg,
		Detail:  res,
	}
}

// buildAbortedByDBUnrecoverable 构造“备份阶段本地数据库损坏且无法无损修复”的中止错误。
// 调用方（backupAndUploadToSMH）拿到后直接 return，由 finalizeUpgradeResult 走中止分支
// （清操作锁 + 发中止通知，原实例保持可用）。重装尚未发生，原盘数据完整，交人工离线抢救。
func buildAbortedByDBUnrecoverable(ctx context.Context) *errUpgradeAborted {
	return &errUpgradeAborted{
		Reason:  "db_unrecoverable",
		UserMsg: i18n.T(ctx, i18n.MsgUpgradeAbortedDBUnrecoverable),
		Detail:  nil,
	}
}
