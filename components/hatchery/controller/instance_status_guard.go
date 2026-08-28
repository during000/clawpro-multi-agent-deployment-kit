package controller

import (
	"context"
	"errors"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── Agent 写操作状态准入统一 guard ─────────────────────────────────────
//
// 背景：Agent 在 10 种非 running 状态下不应接受写操作。前端按钮启用/禁用
// 完全依赖 /openclaw/status 与 /admin/instances 列表返回的 actions[]，
// 权威源是 model.UserStatusMap / model.AdminStatusMap。本 guard 复用同样
// 的状态解析（FetchCVMInstanceInfo + ResolveInstanceStatus）与同样的白名单
// 表，保证 "UI 通过但 API 拒绝" / "UI 拒绝但 API 通过" 不会发生。
//
// 依赖注入：handler 双层化——大写 HandleXxx(w,r) 调小写
// handleXxx(w,r,resolver)，测试直接调小写 handler 传入 mock resolver。
//
// ─────────────────────────────────────────────────────────────────────

// ErrAgentNotAllowed 表示因 Agent 当前状态不允许该操作而被 guard 拒绝。
// handler 拿到包含此 sentinel 的错误时应返回 HTTP 409；其它错误（如
// CVM 查询失败）属于基础设施错误，应返回 500。
var ErrAgentNotAllowed = hcommon.I18nError(i18n.MsgAgentStatusNotAllowed)

// instanceStatusResolver 提供实例状态解析能力。
// 生产实现内部调用 fetchCVMInstanceInfo + ResolveInstanceStatus；
// 测试 mock 直接返回期望的状态结果，不接触真实 CVM。
type instanceStatusResolver interface {
	// ResolveStatus 解析实例当前的 OpenClaw 语义状态，与 /openclaw/status 口径一致。
	ResolveStatus(ctx context.Context, instance *model.Instance) (InstanceStatusResponse, error)
}

// defaultInstanceStatusResolver 是生产环境实现。
type defaultInstanceStatusResolver struct{}

func (defaultInstanceStatusResolver) ResolveStatus(ctx context.Context, instance *model.Instance) (InstanceStatusResponse, error) {
	if instance == nil {
		return InstanceStatusResponse{}, hcommon.I18nError(i18n.MsgInstanceIsNil)
	}
	// 本地实例不走 CVM API。本地实例的 InstanceId 不是 CVM 格式
	// （不是 ins-xxxxxxxx），fetchCVMInstanceInfo 会报「实例ID不合要求」。
	// ResolveInstanceStatus 内部 Step -1 会识别 InstanceSourceLocal 并派发到
	// resolveLocalInstanceStatus，本地实例状态只看 last_report_at，不依赖 cvmInfo。
	if instance.Source == model.InstanceSourceLocal {
		return ResolveInstanceStatus(ctx, instance, nil, nil), nil
	}
	cvmInfo, err := fetchCVMInstanceInfo(ctx, instance.InstanceId)
	if err != nil {
		return InstanceStatusResponse{}, err
	}
	return ResolveInstanceStatus(ctx, instance, cvmInfo, nil), nil
}

// defaultStatusResolver 是所有 handler 默认使用的生产 resolver 实例。
var defaultStatusResolver instanceStatusResolver = defaultInstanceStatusResolver{}

// agentStatusRejectMessage 根据当前语义状态生成统一的"非 running 拒绝"
// 文案。规则：
//   - stopped：明确引导用户开机
//   - 过渡态（creating/loading/upgrading/maintaining/pending）：引导等待
//   - 失败/已销毁（load_failed/create_failed/upgrade_failed/destroyed）：
//     明确告知无法执行
//
// label 取自 UserStatusMap，确保用户端/管控端展示文案一致。
func agentStatusRejectMessage(status InstanceStatusResponse) (i18n.Key, []any) {
	label := status.Label
	if label == "" {
		if def, ok := model.UserStatusMap[status.Status]; ok {
			label = def.Label
		} else {
			label = status.Status
		}
	}

	switch status.Status {
	case model.StatusStopped:
		return i18n.MsgAgentStatusStopped, nil
	case model.StatusCreating, model.StatusLoading, model.StatusUpgrading,
		model.StatusMaintaining, model.StatusPending:
		return i18n.MsgAgentStatusTransition, []any{label}
	case model.StatusLoadFailed, model.StatusCreateFailed,
		model.StatusUpgradeFailed, model.StatusDestroyed:
		return i18n.MsgAgentStatusFailed, []any{label}
	default:
		return i18n.MsgAgentStatusFailed, []any{label}
	}
}

// agentNotAllowedError 是 guard 业务拒绝时返回的错误类型，包装
// ErrAgentNotAllowed sentinel。Error() 仅返回用户文案，方便直接透传到
// writeError 的 error 字段。
type agentNotAllowedError struct {
	msg string
}

func (e agentNotAllowedError) Error() string { return e.msg }
func (e agentNotAllowedError) Unwrap() error { return ErrAgentNotAllowed }

func newAgentNotAllowedError(status InstanceStatusResponse) error {
	key, args := agentStatusRejectMessage(status)
	return hcommon.I18nError(key, args...).
		WithCause(ErrAgentNotAllowed)
}

// actionAllowed 判断 action 是否在给定的 Actions 列表中。
func actionAllowed(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

// requireNoResourceAdjustment 拒绝资源调整期间的其他实例写操作。
// current_operation 是调整 worker 持有的操作锁，不能被生命周期或配置操作覆盖。
func requireNoResourceAdjustment(instance *model.Instance) error {
	if instance != nil && model.IsResourceAdjustmentOperation(instance.CurrentOperation) {
		return ErrOperationInProgress
	}
	return nil
}

// requireActionAllowedForUser 校验当前实例状态下 action 是否在用户端
// 白名单（UserStatusMap）中。
//
// 返回的 error：
//   - errors.Is(err, ErrAgentNotAllowed) 为 true → 状态拒绝，handler 返回 409
//   - 其它非 nil error → 基础设施错误（如 CVM 查询失败），handler 返回 500
//   - nil → 放行
func requireActionAllowedForUser(ctx context.Context, instance *model.Instance, action string, resolver instanceStatusResolver) (InstanceStatusResponse, error) {
	if err := requireNoResourceAdjustment(instance); err != nil {
		return InstanceStatusResponse{}, err
	}
	status, err := resolver.ResolveStatus(ctx, instance)
	if err != nil {
		return status, err
	}
	def, ok := model.UserStatusMap[status.Status]
	if !ok || !actionAllowed(def.Actions, action) {
		return status, newAgentNotAllowedError(status)
	}
	return status, nil
}

// requireActionAllowedForAdmin 同 requireActionAllowedForUser，但查
// AdminStatusMap。
func requireActionAllowedForAdmin(ctx context.Context, instance *model.Instance, action string, resolver instanceStatusResolver) (InstanceStatusResponse, error) {
	if err := requireNoResourceAdjustment(instance); err != nil {
		return InstanceStatusResponse{}, err
	}
	status, err := resolver.ResolveStatus(ctx, instance)
	if err != nil {
		return status, err
	}
	def, ok := model.AdminStatusMap[status.Status]
	if !ok || !actionAllowed(def.Actions, action) {
		return status, newAgentNotAllowedError(status)
	}
	return status, nil
}

// requireInstanceRunning 用于 UserStatusMap / AdminStatusMap 未列出的
// 写操作（如 upgrade/add-skill/set-model/set-channel/set-env 等，前端
// 入口仅在 running 时启用）。非 running 时返回包含 ErrAgentNotAllowed
// 的错误。
func requireInstanceRunning(ctx context.Context, instance *model.Instance, resolver instanceStatusResolver) (InstanceStatusResponse, error) {
	if err := requireNoResourceAdjustment(instance); err != nil {
		return InstanceStatusResponse{}, err
	}
	status, err := resolver.ResolveStatus(ctx, instance)
	if err != nil {
		return status, hcommon.I18nRichError(err, i18n.MsgAgentNotAllowed)
	}
	if status.Status != model.StatusRunning {
		return status, newAgentNotAllowedError(status)
	}
	return status, nil
}

// writeAgentGuardError 把 guard 返回的状态或操作冲突转换为 HTTP 409，
// 基础设施错误返回 500。
//
// HTML 响应（少数 handler 在 isJSON=false 时走 renderInstanceListWithError）
// 需要 handler 自行分支处理。
func writeAgentGuardError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrAgentNotAllowed) || errors.Is(err, ErrOperationInProgress) {
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
}
