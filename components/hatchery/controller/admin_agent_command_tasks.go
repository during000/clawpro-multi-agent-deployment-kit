package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ============================================================================
// 请求体 / 响应体
// ============================================================================

// agentDispatchReq 下发请求体。
//
// 端点 `POST /admin/agent-commands/dispatch` 复用三种模式，由入参组合区分：
//
//   - **A 启动新 dispatch**：填 CommandID + InstanceIDs (+ 可选 TestFirst/TestTargetInstanceID/ParamValues)；
//     必须不带 DispatchSlug。
//   - **B 续跑（人为闸门通过）**：仅填 DispatchSlug；A 模式所有字段必须为零值（否则 400 dispatch_slug_with_extra_params）。
//   - **C 终止下发**：填 DispatchSlug + Abort=true；A 模式所有字段必须为零值。
//
// 详见 design.md §6.4「dispatch 接口三模式」。
type agentDispatchReq struct {
	CommandID            uint              `json:"command_id"`
	InstanceIDs          []uint            `json:"instance_ids"`
	TestFirst            bool              `json:"test_first"`
	TestTargetInstanceID uint              `json:"test_target_instance_id"`
	ParamValues          map[string]string `json:"param_values"`

	// 续跑 / 终止 用：携带已存在的 dispatch_slug
	DispatchSlug string `json:"dispatch_slug,omitempty"`
	// 仅在 DispatchSlug 非空时有效：true 表示终止 pending 批次（C 模式），否则触发续跑（B 模式）
	Abort bool `json:"abort,omitempty"`
}

// agentDispatchResp 下发响应体（同步阶段）。
type agentDispatchResp struct {
	DispatchSlug         string            `json:"dispatch_slug"`
	Status               string            `json:"status"`
	TargetCount          uint              `json:"target_count"`
	TestFirst            bool              `json:"test_first"`
	TestTargetInstanceID *uint             `json:"test_target_instance_id"`
	ParamValues          map[string]string `json:"param_values"`
	InvocationsPlanned   uint              `json:"invocations_planned"`
	StartedAt            time.Time         `json:"started_at"`
}

// agentTaskListItem 执行记录列表行（dispatch 粒度）。
type agentTaskListItem struct {
	DispatchSlug          string     `json:"dispatch_slug"`
	CommandName           string     `json:"command_name"`
	CommandContentPreview string     `json:"command_content_preview"`
	TriggeredByUserID     uint       `json:"triggered_by_user_id"`
	TriggeredByUsername   string     `json:"triggered_by_username"`
	TriggeredByEmail      string     `json:"triggered_by_email"`
	TestFirst             bool       `json:"test_first"`
	Status                string     `json:"status"`
	TargetCount           uint       `json:"target_count"`
	SuccessCount          uint       `json:"success_count"`
	FailedCount           uint       `json:"failed_count"`
	InvocationCount       uint       `json:"invocation_count"`
	StartedAt             time.Time  `json:"started_at"`
	FinishedAt            *time.Time `json:"finished_at"`
}

type agentTaskListResp struct {
	Tasks    []agentTaskListItem `json:"tasks"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// agentTaskDetailResp 执行记录详情响应。
type agentTaskDetailResp struct {
	DispatchSlug         string                  `json:"dispatch_slug"`
	Status               string                  `json:"status"`
	TestFirst            bool                    `json:"test_first"`
	TestTargetInstanceID *uint                   `json:"test_target_instance_id"`
	TargetCount          uint                    `json:"target_count"`
	SuccessCount         uint                    `json:"success_count"`
	FailedCount          uint                    `json:"failed_count"`
	TriggeredByUserID    uint                    `json:"triggered_by_user_id"`
	TriggeredByUsername  string                  `json:"triggered_by_username"`
	TriggeredByEmail     string                  `json:"triggered_by_email"`
	StartedAt            time.Time               `json:"started_at"`
	FinishedAt           *time.Time              `json:"finished_at"`
	CommandSnapshot      map[string]any          `json:"command_snapshot"`
	ParamValues          map[string]string       `json:"param_values"`
	TestSummary          *agentTestSummary       `json:"test_summary"`
	Invocations          []agentInvocationDetail `json:"invocations"`
}

type agentTestSummary struct {
	Passed        bool   `json:"passed"`
	InstanceID    uint   `json:"instance_id"`
	ExitCode      *int   `json:"exit_code"`
	ElapsedMs     *uint  `json:"elapsed_ms"`
	OutputPreview string `json:"output_preview"`
}

type agentInvocationDetail struct {
	TATInvocationID string              `json:"tat_invocation_id"`
	IsTestRun       bool                `json:"is_test_run"`
	BatchIndex      uint                `json:"batch_index"`
	Status          string              `json:"status"`
	TargetCount     uint                `json:"target_count"`
	SuccessCount    uint                `json:"success_count"`
	FailedCount     uint                `json:"failed_count"`
	StartedAt       *time.Time          `json:"started_at"`
	FinishedAt      *time.Time          `json:"finished_at"`
	Tasks           []agentTaskInDetail `json:"tasks"`
}

type agentTaskInDetail struct {
	TATInvocationTaskID string  `json:"tat_invocation_task_id"`
	InstanceID          uint    `json:"instance_id"`
	CVMInstanceID       string  `json:"cvm_instance_id"`
	AgentName           string  `json:"agent_name"`
	OwnerUsername       string  `json:"owner_username"`
	IsTestTarget        bool    `json:"is_test_target"`
	Status              string  `json:"status"`
	ExitCode            *int    `json:"exit_code"`
	ElapsedMs           *uint   `json:"elapsed_ms"`
	Stdout              *string `json:"stdout"`
	Stderr              *string `json:"stderr"`
	StdoutTruncated     bool    `json:"stdout_truncated"`
	StderrTruncated     bool    `json:"stderr_truncated"`
	OutputExpired       bool    `json:"output_expired"`
	// ErrorInfo 来自 TAT 顶层 ErrorInfo 字段。当 status=unreachable / failed 而 stdout/stderr
	// 为空时，前端可展示该字段定位启动阶段失败原因（如 "user xxx does not exist"）。
	// 正常执行的 task 此字段为空字符串。
	ErrorInfo  string     `json:"error_info"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// ============================================================================
// 常量
// ============================================================================

// AgentTaskOutputMaxBytes 单字段（stdout / stderr）的截断阈值（决策 Q5：64KB）。
const AgentTaskOutputMaxBytes = 64 * 1024

// AgentTaskOutputPreviewBytes 测试机黄条提示的输出前缀长度。
const AgentTaskOutputPreviewBytes = 200

// agentPollerInterval 全局轮询协程的检查间隔。
const agentPollerInterval = 5 * time.Second

// agentPollerBatchLimit 单次轮询拉取的 task 数上限（避免一次性查太多）。
const agentPollerBatchLimit = 200

// agentDispatchAsyncWG 可选的 WaitGroup，用于测试等待 HandleDispatchAgentCommand
// 的后台 goroutine 完成。生产环境为 nil（不使用），测试中通过赋值启用。
//
// 详见 docs/testing.md「异步任务」节，参考 skillDistributeWG 范式：
// 测试 setup 时 `agentDispatchAsyncWG = &wg`，cleanup 调 `wg.Wait()` + 置 nil。
var agentDispatchAsyncWG *sync.WaitGroup

// ============================================================================
// Handler: POST /admin/agent-commands/dispatch
// ============================================================================

// HandleDispatchAgentCommand 下发命令到一批 Agent。
//
// 同步阶段：
//   - 校验 → 组装 param_values_json → 生成 dispatch_slug → 事务内预创建
//     1 条测试 invocation + 1 条 test task（如启用）+ N 条生产 invocation + M 条 task
//   - 立即返回 dispatch_slug
//
// 异步阶段（goroutine + DetachContext）：
//   - 测试机优先：调 RunInlineCommandBatchAsync 拿 (inv-, [(test_instance, invt-)])
//     轮询测试 task 至终态 → 失败标记整 dispatch failed；成功进入生产阶段
//   - 生产阶段：每个预创建批次调 RunInlineCommandBatchAsync 异步发出
//   - 全局轮询协程（task/agent_command_poller.go 注册到 scheduler）持续刷新 task 状态
func HandleDispatchAgentCommand(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	var req agentDispatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	ctx := r.Context()

	// ===== 0. 三模式分流 =====
	//
	// 携带 dispatch_slug 即为 B/C 模式：续跑 / 终止；不允许同时携带 A 模式独有字段。
	// design.md §6.4。
	if req.DispatchSlug != "" {
		if hasNewDispatchFields(&req) {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgDispatchSlugWithExtraParams).WithPrefix("dispatch_slug_with_extra_params"))
			return
		}
		if req.Abort {
			handleDispatchAbort(w, r, user, req.DispatchSlug)
		} else {
			handleDispatchContinue(w, r, user, req.DispatchSlug)
		}
		return
	}
	if req.Abort {
		// A 模式禁止单独携带 abort
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgDispatchSlugRequired).WithPrefix("dispatch_slug_required"))
		return
	}

	// ===== A 模式：启动新 dispatch（核心逻辑抽到 startDispatch 供定时任务复用）=====
	res, err := startDispatch(ctx, startDispatchInput{
		CommandID:            req.CommandID,
		InstanceIDs:          req.InstanceIDs,
		TestFirst:            req.TestFirst,
		TestTargetInstanceID: req.TestTargetInstanceID,
		ParamValues:          req.ParamValues,
		TriggeredByUserID:    user.ID,
	})
	if err != nil {
		var de *dispatchStartError
		if errors.As(err, &de) {
			writeError(w, r, de.code, de.err)
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		}
		return
	}

	// 同步响应
	resp := agentDispatchResp{
		DispatchSlug:       res.DispatchSlug,
		Status:             model.AgentInvocationStatusInProgress,
		TargetCount:        res.TargetCount,
		TestFirst:          res.TestFirst,
		ParamValues:        res.ParamValues,
		InvocationsPlanned: res.InvocationsPlanned,
		StartedAt:          res.StartedAt,
	}
	if res.TestFirst && res.TestTargetInstanceID != 0 {
		tt := res.TestTargetInstanceID
		resp.TestTargetInstanceID = &tt
	}
	jsonOK(w, resp)
}

// ============================================================================
// startDispatch：A 模式下发核心逻辑（HTTP handler 与定时任务 runner 共用）
// ============================================================================

// startDispatchInput startDispatch 的入参。TriggeredByUserID 由调用方决定：
// 手工下发 = 登录用户；定时任务 = schedule 创建者。
type startDispatchInput struct {
	CommandID            uint
	InstanceIDs          []uint
	TestFirst            bool
	TestTargetInstanceID uint
	ParamValues          map[string]string
	TriggeredByUserID    uint
	// AllowPartialOffline=true 时（仅定时任务）：离线目标不再整批拒绝，而是剔除离线机器、
	// 仅对在线机器下发，离线机器单独记为 unreachable 失败。全部离线仍拒绝。
	AllowPartialOffline bool
}

// startDispatchResult startDispatch 成功后的同步结果。
type startDispatchResult struct {
	DispatchSlug         string
	TargetCount          uint
	TestFirst            bool
	TestTargetInstanceID uint
	ParamValues          map[string]string
	InvocationsPlanned   uint
	StartedAt            time.Time
}

// dispatchStartError 携带建议 HTTP 状态码的下发校验错误，便于 handler 精确映射 code。
// 定时任务 runner 忽略 code，仅取 message 记入 schedule.last_error。
// err 为 *hcommon.RichError（带 i18n key + 机器码 prefix），手动下发与定时任务均可正确国际化。
type dispatchStartError struct {
	code int
	err  *hcommon.RichError
}

func (e *dispatchStartError) Error() string { return e.err.Error() }
func (e *dispatchStartError) Unwrap() error { return e.err }

func newDispatchStartError(code int, err *hcommon.RichError) *dispatchStartError {
	return &dispatchStartError{code: code, err: err}
}

// startDispatch 组装并启动一次 dispatch（A 模式），返回同步结果。
//
// 只依赖 ctx（model.DB(ctx) 已注入租户），不依赖 *http.Request，可被定时任务 runner 复用。
// 校验失败返回 *dispatchStartError（含建议 HTTP code）；系统级错误返回普通 error。
func startDispatch(ctx context.Context, in startDispatchInput) (*startDispatchResult, error) {
	if in.CommandID == 0 {
		return nil, newDispatchStartError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgCommandRequired).WithPrefix("command_required"))
	}
	if len(in.InstanceIDs) == 0 {
		return nil, newDispatchStartError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgTargetsRequired).WithPrefix("targets_required"))
	}
	uniqIDs := dedupUintSlice(in.InstanceIDs)
	if len(uniqIDs) > model.AgentDispatchMaxTargets {
		return nil, newDispatchStartError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgTooManyTargets, model.AgentDispatchMaxTargets).WithPrefix("too_many_targets"))
	}

	cmd, err := model.FindAgentCommandByID(ctx, in.CommandID)
	if err != nil {
		if errors.Is(err, model.ErrAgentCommandNotFound) {
			return nil, newDispatchStartError(http.StatusNotFound, hcommon.I18nError(i18n.MsgCommandNotFound).WithPrefix("command_not_found"))
		}
		return nil, err
	}

	// 取实例：仅返回当前租户内的（identifier 回调过滤）+ 校验存在
	var instances []model.Instance
	if err := model.DB(ctx).Where("id IN ?", uniqIDs).Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("query instances: %w", err)
	}
	if len(instances) != len(uniqIDs) {
		foundSet := make(map[uint]struct{}, len(instances))
		for _, ins := range instances {
			foundSet[ins.ID] = struct{}{}
		}
		missing := make([]uint, 0)
		for _, id := range uniqIDs {
			if _, ok := foundSet[id]; !ok {
				missing = append(missing, id)
			}
		}
		return nil, newDispatchStartError(http.StatusNotFound,
			hcommon.I18nError(i18n.MsgInstanceNotFoundInTenant, missing).WithPrefix("instance_not_found"))
	}

	// 拒绝本地 agent 实例：dispatch 会走 TAT RunCommand，本地实例 ID 不是 ins-* 格式，
	// 会被 CVM/TAT 参数校验拒。提前报错并列出具体名字，让用户调整选项。
	localNames := make([]string, 0)
	for _, ins := range instances {
		if ins.Source == model.InstanceSourceLocal {
			localNames = append(localNames, ins.Name)
		}
	}
	if len(localNames) > 0 {
		return nil, newDispatchStartError(http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgLocalInstanceTargetUnsupported, strings.Join(localNames, ", ")).
				WithPrefix("local_instance_target_unsupported"))
	}

	for i := range instances {
		if err := requireNoResourceAdjustment(&instances[i]); err != nil {
			return nil, newDispatchStartError(http.StatusConflict,
				hcommon.I18nError(i18n.MsgOperationInProgress).
					WithPrefix("instance_adjustment_in_progress"))
		}
	}

	// 决策 Q6：跨地域不允许（每条 Instance 没有 region 字段，单租户单 region 假设；如需扩展再加）
	// 当前管控端整体绑定 CVMRegion，所有 Instance 都在该 region；此处校验入口数据。
	// 如未来 Instance 增加 region 字段，再加 cross-region check。
	// 校验在线（非 RUNNING 视为不可下发）
	offlineNames := make([]string, 0)
	offlineSet := make(map[uint]struct{})
	for _, ins := range instances {
		if !isInstanceRunning(&ins) {
			offlineNames = append(offlineNames, ins.Name)
			offlineSet[ins.ID] = struct{}{}
		}
	}
	if len(offlineSet) > 0 {
		// 即时下发（默认）：任一离线即整批拒绝，保持原行为
		if !in.AllowPartialOffline {
			return nil, newDispatchStartError(http.StatusConflict,
				hcommon.I18nError(i18n.MsgTargetOffline, strings.Join(offlineNames, ", ")).WithPrefix("target_offline"))
		}
		// 尽力而为（定时任务）：全部离线则无可下发，仍拒绝（让 schedule 记 last_error）
		if len(offlineSet) == len(uniqIDs) {
			return nil, newDispatchStartError(http.StatusConflict,
				hcommon.I18nError(i18n.MsgAllTargetsOffline).WithPrefix("all_targets_offline"))
		}
		// 否则离线机器在下方从生产批次剔除、单独记为失败
	}

	// 测试机校验
	if in.TestFirst {
		if in.TestTargetInstanceID == 0 {
			return nil, newDispatchStartError(http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgTestTargetRequired).WithPrefix("test_target_required"))
		}
		found := false
		for _, id := range uniqIDs {
			if id == in.TestTargetInstanceID {
				found = true
				break
			}
		}
		if !found {
			return nil, newDispatchStartError(http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgTestTargetInvalid).WithPrefix("test_target_invalid"))
		}
	}

	// 参数取值组装与校验
	finalParams, perr := assembleParamValues(cmd, in.ParamValues)
	if perr != nil {
		return nil, newDispatchStartError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(perr))
	}

	// ===== 生成 dispatch_slug + 事务内预创建 =====
	dispatchSlug, err := generateUniqueDispatchSlug(ctx, 8)
	if err != nil {
		return nil, err
	}
	cmdSnapshotJSON, err := buildCommandSnapshot(cmd, dispatchSlug)
	if err != nil {
		return nil, err
	}
	paramValuesJSON, err := json.Marshal(finalParams)
	if err != nil {
		return nil, fmt.Errorf("marshal param_values: %w", err)
	}

	// 由于 AgentDispatchMaxTargets == TATRunCommandBatchMax == 200，
	// chunkUint 必然返回至多 1 批 —— 一次 dispatch 在生产阶段只触发 1 次 RunCommand。
	prodInstanceIDs := make([]uint, 0, len(uniqIDs))
	offlineInstanceIDs := make([]uint, 0, len(offlineSet))
	for _, id := range uniqIDs {
		if in.TestFirst && id == in.TestTargetInstanceID {
			continue
		}
		if _, off := offlineSet[id]; off {
			offlineInstanceIDs = append(offlineInstanceIDs, id) // 仅 AllowPartialOffline 时非空
			continue
		}
		prodInstanceIDs = append(prodInstanceIDs, id)
	}
	prodBatches := chunkUint(prodInstanceIDs, TATRunCommandBatchMax)
	totalInvocations := uint(len(prodBatches))
	if in.TestFirst {
		totalInvocations++
	}
	if len(offlineInstanceIDs) > 0 {
		totalInvocations++
	}

	now := time.Now()
	plan := &dispatchPlan{
		dispatchSlug:       dispatchSlug,
		commandID:          cmd.ID,
		commandSnapshot:    string(cmdSnapshotJSON),
		paramValuesJSON:    string(paramValuesJSON),
		triggeredByUID:     in.TriggeredByUserID,
		testFirst:          in.TestFirst,
		testInstanceID:     in.TestTargetInstanceID,
		prodBatches:        prodBatches,
		offlineInstanceIDs: offlineInstanceIDs,
		instancesByID:      indexInstancesByID(instances),
		startedAt:          now,
	}

	if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		return preCreateInvocationsAndTasks(tx, plan)
	}); err != nil {
		return nil, fmt.Errorf("pre-create invocations: %w", err)
	}

	// ===== 异步派发 =====
	asyncCtx := hcommon.WithTaskTrace(hcommon.DetachContext(ctx), "agent_command_dispatch")
	if agentDispatchAsyncWG != nil {
		agentDispatchAsyncWG.Add(1)
	}
	go func() {
		if agentDispatchAsyncWG != nil {
			defer agentDispatchAsyncWG.Done()
		}
		runDispatchAsync(asyncCtx, plan, cmd.Content, cmd.TimeoutSec, cmd.RunUser, cmd.Workdir, finalParams)
	}()

	return &startDispatchResult{
		DispatchSlug:         dispatchSlug,
		TargetCount:          uint(len(uniqIDs)),
		TestFirst:            in.TestFirst,
		TestTargetInstanceID: in.TestTargetInstanceID,
		ParamValues:          finalParams,
		InvocationsPlanned:   totalInvocations,
		StartedAt:            now,
	}, nil
}

// ============================================================================
// dispatch 端点的续跑 / 终止两条同步路径
// ============================================================================

// hasNewDispatchFields 判断 B/C 模式下是否同时携带了 A 模式独有的字段。
// 用于产生 400 dispatch_slug_with_extra_params。
func hasNewDispatchFields(req *agentDispatchReq) bool {
	if req.CommandID != 0 {
		return true
	}
	if len(req.InstanceIDs) > 0 {
		return true
	}
	if req.TestFirst {
		return true
	}
	if req.TestTargetInstanceID != 0 {
		return true
	}
	if len(req.ParamValues) > 0 {
		return true
	}
	return false
}

// loadDispatchForControl 根据 slug 拉 dispatch 行（不走识 dispatch_slug 反向查 invocation）。
// 返回 ErrDispatchNotFound 表示 slug 在当前租户下不存在。
func loadDispatchForControl(ctx context.Context, slug string) (*model.AgentCommandDispatch, error) {
	return model.FindDispatchBySlug(ctx, slug)
}

// canControlDispatch 校验调用者是否有权限对 dispatch 续跑 / 终止。
// 规则：必须是 dispatch 的原 triggered_by 或当前租户的初始管理员。
func canControlDispatch(ctx context.Context, user *model.User, d *model.AgentCommandDispatch) bool {
	if user == nil || d == nil {
		return false
	}
	if user.IsInitialAdmin(ctx) {
		return true
	}
	return d.TriggeredByUserID == user.ID
}

// handleDispatchContinue 处理 B 模式：用户在测试机阶段后点【继续下发剩余 N 台】。
//
// 同步部分：校验 dispatch.status = awaiting_confirmation + 权限；启动 goroutine 异步触发剩余 prod 批次。
// 同步响应同 A 模式的 agentDispatchResp 形状（status=in_progress）。
func handleDispatchContinue(w http.ResponseWriter, r *http.Request, user *model.User, slug string) {
	ctx := r.Context()

	d, err := loadDispatchForControl(ctx, slug)
	if err != nil {
		if errors.Is(err, model.ErrDispatchNotFound) {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgDispatchNotFound).WithPrefix("dispatch_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if !canControlDispatch(ctx, user, d) {
		writeError(w, r, http.StatusForbidden,
			hcommon.I18nError(i18n.MsgDispatchPermissionDenied).WithPrefix("permission_denied"))
		return
	}

	if err := requireDispatchAwaitingConfirmation(ctx, d); err != nil {
		writeError(w, r, statusCodeForControlErr(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 状态变化：awaiting_confirmation → in_progress
	if err := model.DB(ctx).Model(&model.AgentCommandDispatch{}).
		Where("id = ?", d.ID).
		Updates(map[string]any{
			"status":     model.AgentDispatchStatusInProgress,
			"updated_at": time.Now(),
		}).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 启动异步 goroutine 跑剩余 pending prod invocation
	asyncCtx := hcommon.WithTaskTrace(hcommon.DetachContext(ctx), "agent_command_dispatch_continue")
	if agentDispatchAsyncWG != nil {
		agentDispatchAsyncWG.Add(1)
	}
	go func() {
		if agentDispatchAsyncWG != nil {
			defer agentDispatchAsyncWG.Done()
		}
		runProdPhaseAsync(asyncCtx, slug)
	}()

	// 计数：本次会触发多少条 invocation（参考字段，前端可不依赖）
	invs, _ := model.FindInvocationsByDispatchID(ctx, d.ID)
	var pendingProd uint
	for i := range invs {
		inv := &invs[i]
		if !inv.IsTestRun && inv.Status == model.AgentInvocationStatusPending && inv.TATInvocationID == "" {
			pendingProd++
		}
	}
	jsonOK(w, agentDispatchResp{
		DispatchSlug:       slug,
		Status:             model.AgentDispatchStatusInProgress,
		InvocationsPlanned: pendingProd,
		StartedAt:          time.Now(),
	})
}

// handleDispatchAbort 处理 C 模式：用户点【终止下发】。
//
// 把所有 pending 非 test invocation + 它们的 pending task 标 cancelled，dispatch 整体进入 cancelled 终态。
// 已经 in_progress / 已终态的 invocation 不动 —— abort 只对「闸门后还没发出的」生效。
func handleDispatchAbort(w http.ResponseWriter, r *http.Request, user *model.User, slug string) {
	ctx := r.Context()

	d, err := loadDispatchForControl(ctx, slug)
	if err != nil {
		if errors.Is(err, model.ErrDispatchNotFound) {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgDispatchNotFound).WithPrefix("dispatch_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if !canControlDispatch(ctx, user, d) {
		writeError(w, r, http.StatusForbidden,
			hcommon.I18nError(i18n.MsgDispatchAbortPermissionDenied).WithPrefix("permission_denied"))
		return
	}
	if err := requireDispatchAwaitingConfirmation(ctx, d); err != nil {
		writeError(w, r, statusCodeForControlErr(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 批量更新 pending 非 test invocation → cancelled
	now := time.Now()
	invRes := model.DB(ctx).Model(&model.AgentCommandInvocation{}).
		Where("dispatch_id = ? AND is_test_run = ? AND status = ? AND tat_invocation_id = ?",
			d.ID, false, model.AgentInvocationStatusPending, "").
		Updates(map[string]any{
			"status":      model.AgentInvocationStatusCancelled,
			"finished_at": now,
		})
	if invRes.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(invRes.Error, i18n.MsgCancelInvocationsFailed))
		return
	}
	if invRes.RowsAffected == 0 {
		// 并发：要么已经被 abort，要么已经被 continue 启动。两种情况都已不再是 awaiting_confirmation。
		writeError(w, r, http.StatusConflict,
			hcommon.I18nError(i18n.MsgNothingToAbort).WithPrefix("nothing_to_abort"))
		return
	}

	// 同步把 pending task 也标 cancelled
	taskRes := model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("dispatch_id = ? AND is_test_target = ? AND status = ?",
			d.ID, false, model.AgentTaskStatusPending).
		Updates(map[string]any{
			"status":      model.AgentTaskStatusCancelled,
			"finished_at": now,
		})
	if taskRes.Error != nil {
		// 这里失败不阻断响应：invocation 已经标 cancelled，poller / detail handler 会兜底
		slog.Error("[AgentCommand] cancel pending tasks failed", "dispatch", slug, "error", taskRes.Error)
	}

	// 推 dispatch.status → cancelled
	recalcDispatchStatus(ctx, d.ID)

	jsonOK(w, map[string]any{
		"dispatch_slug":   slug,
		"status":          model.AgentDispatchStatusCancelled,
		"cancelled_count": uint(taskRes.RowsAffected),
	})
}

// requireDispatchAwaitingConfirmation 校验 dispatch 当前状态必须是 awaiting_confirmation；不是则返回带语义的 error。
//
// 错误的 HTTP code 约定：
//   - test_phase_in_progress → 409
//   - test_phase_failed / already_completed → 409
//   - already_continued → 409
//   - 其它边界 → 409 not_awaiting
//
// 优先读 dispatch.status；为给前端更友好提示，内部按需加载 invocation 切片做细分。
func requireDispatchAwaitingConfirmation(ctx context.Context, d *model.AgentCommandDispatch) error {
	if d.Status == model.AgentDispatchStatusAwaitingConfirmation {
		return nil
	}

	// 终态情况：已成功 / 已部分 / 已失败 / 已取消
	if d.IsTerminal() {
		return hcommon.I18nError(i18n.MsgCmdTaskAlreadyCompleted).WithPrefix("already_completed")
	}

	// dispatch.status == in_progress 时，再细分两种语义：
	//   - 测试机仍在跑（test_phase_in_progress）
	//   - 生产批已发出（already_continued）
	invs, err := model.FindInvocationsByDispatchID(ctx, d.ID)
	if err != nil {
		return hcommon.I18nError(i18n.MsgCmdTaskNotAwaiting).WithPrefix("not_awaiting")
	}
	var hasInProgressTest, hasFailedTest, hasInProgressProd bool
	for i := range invs {
		inv := &invs[i]
		if inv.IsTestRun {
			switch inv.Status {
			case model.AgentInvocationStatusInProgress, model.AgentInvocationStatusPending:
				hasInProgressTest = true
			case model.AgentInvocationStatusFailed, model.AgentInvocationStatusPartial:
				hasFailedTest = true
			}
			continue
		}
		if inv.Status == model.AgentInvocationStatusInProgress || inv.TATInvocationID != "" {
			hasInProgressProd = true
		}
	}
	switch {
	case hasInProgressTest:
		return hcommon.I18nError(i18n.MsgCmdTaskTestPhaseInProgress).WithPrefix("test_phase_in_progress")
	case hasFailedTest:
		return hcommon.I18nError(i18n.MsgCmdTaskTestPhaseFailed).WithPrefix("test_phase_failed")
	case hasInProgressProd:
		return hcommon.I18nError(i18n.MsgCmdTaskAlreadyContinued).WithPrefix("already_continued")
	default:
		return hcommon.I18nError(i18n.MsgCmdTaskNotAwaiting).WithPrefix("not_awaiting")
	}
}

// statusCodeForControlErr 根据 requireDispatchAwaitingConfirmation 返回的错误体推 HTTP code。
// 当前所有可识别错误都映射 409 Conflict。
func statusCodeForControlErr(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return http.StatusConflict
}

// runProdPhaseAsync 在 goroutine 内为指定 dispatch_slug 触发剩余生产批次。
//
// 进入条件：handleDispatchContinue 已确认 dispatch 处于 awaiting_confirmation。
// 函数内部从 DB 重建批次执行所需的脚本内容、参数、目标实例 —— 不依赖 in-memory dispatchPlan。
func runProdPhaseAsync(ctx context.Context, slug string) {
	defer func() {
		if rcv := recover(); rcv != nil {
			slog.Error("[AgentCommand] runProdPhaseAsync panic", "dispatch", slug, "error", rcv)
		}
	}()

	invs, err := model.FindInvocationsByDispatchSlug(ctx, slug)
	if err != nil {
		slog.Error("[AgentCommand] runProdPhaseAsync find invocations failed", "dispatch", slug, "error", err)
		return
	}
	for i := range invs {
		inv := &invs[i]
		if inv.IsTestRun {
			continue
		}
		if inv.Status != model.AgentInvocationStatusPending || inv.TATInvocationID != "" {
			continue
		}
		executeProdBatchByInvocationID(ctx, inv)
	}
	recalcDispatchStatusBySlug(ctx, slug)
}

// executeProdBatchByInvocationID 与 executeProdBatch 等价，但从 dispatch 行已存的
// snapshot / param_values 还原参数（v2 数据模型：snapshot 上提到 dispatch）。
//
// 不复用 dispatchPlan in-memory 结构 —— 续跑场景下原 plan 已随第一次 handler 的 goroutine 退出消失。
func executeProdBatchByInvocationID(ctx context.Context, inv *model.AgentCommandInvocation) {
	tasks, err := model.FindTasksByInvocationID(ctx, inv.ID)
	if err != nil {
		slog.Error("[AgentCommand] continue: load tasks failed", "invocation", inv.ID, "error", err)
		_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusFailed, "")
		return
	}
	if len(tasks) == 0 {
		// 无 task：异常空批，直接标 success（与原 markDispatchAllFailed 一致的兜底）
		_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusSuccess, "")
		return
	}

	// snapshot / param_values 上提到 dispatch 行
	d, err := model.FindDispatchByID(ctx, inv.DispatchID)
	if err != nil {
		slog.Error("[AgentCommand] continue: load dispatch failed",
			"invocation", inv.ID, "dispatch_id", inv.DispatchID, "error", err)
		_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusFailed, "")
		return
	}

	snap := decodeSnapshot(d.CommandSnapshot)
	scriptContent, _ := snap["content"].(string)
	runUser := runUserOrDefault(stringFromAny(snap["run_user"]))
	workdir := workdirOrDefault(stringFromAny(snap["workdir"]))
	timeoutSec := uint64FromAny(snap["timeout_sec"])
	if timeoutSec == 0 {
		timeoutSec = 60
	}
	finalParams := decodeParamValues(d.ParamValuesJSON)

	cvmIDs := make([]string, 0, len(tasks))
	for i := range tasks {
		if tasks[i].CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, tasks[i].CVMInstanceID)
		}
	}
	if len(cvmIDs) == 0 {
		_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusFailed, "")
		for i := range tasks {
			_ = setTaskFailed(ctx, tasks[i].ID, model.AgentTaskStatusUnreachable, nil)
		}
		updateInvocationCountsFromTasks(ctx, inv.ID)
		return
	}

	tatInvID, bindings, err := RunInlineCommandBatchAsync(ctx,
		cvmIDs, scriptContent, timeoutSec, runUser, workdir, finalParams)
	if err != nil {
		slog.Error("[AgentCommand] continue: prod RunCommand 失败",
			"dispatch", inv.DispatchSlug, "invocation", inv.ID, "error", err)
		_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusFailed, "")
		for i := range tasks {
			_ = setTaskFailed(ctx, tasks[i].ID, model.AgentTaskStatusUnreachable, nil)
		}
		updateInvocationCountsFromTasks(ctx, inv.ID)
		return
	}

	_ = setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusInProgress, tatInvID)

	cvmToTaskID := make(map[string]uint, len(tasks))
	for i := range tasks {
		if tasks[i].CVMInstanceID != "" {
			cvmToTaskID[tasks[i].CVMInstanceID] = tasks[i].ID
		}
	}
	for _, b := range bindings {
		tid := cvmToTaskID[b.InstanceID]
		if tid == 0 {
			continue
		}
		_ = setTaskTATID(ctx, tid, b.InvocationTaskID)
		_ = setTaskStatus(ctx, tid, model.AgentTaskStatusInProgress)
	}
}

// stringFromAny 帮助从 snapshot map[string]any 取字符串字段。
func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

// uint64FromAny 帮助从 snapshot map[string]any 取数值字段（json.Number/float64 都接受）。
func uint64FromAny(v any) uint64 {
	switch n := v.(type) {
	case float64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case int:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case uint:
		return uint64(n)
	case int64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case uint64:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil && i > 0 {
			return uint64(i)
		}
	}
	return 0
}

// decodeParamValues 解码 invocation.param_values_json 字段。
func decodeParamValues(s string) map[string]string {
	if s == "" {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return map[string]string{}
	}
	if m == nil {
		return map[string]string{}
	}
	return m
}

// ============================================================================
// 内部类型 / 辅助
// ============================================================================

// dispatchPlan 一次 dispatch 的执行计划。
type dispatchPlan struct {
	dispatchID      uint
	dispatchSlug    string
	commandID       uint
	commandSnapshot string
	paramValuesJSON string
	triggeredByUID  uint
	testFirst       bool
	testInstanceID  uint
	prodBatches     [][]uint
	// offlineInstanceIDs 尽力而为下发时被剔除的离线目标；预创建为 unreachable 终态 task，不发 TAT。
	offlineInstanceIDs []uint
	instancesByID      map[uint]*model.Instance
	startedAt          time.Time

	// 预创建后回填的主键，便于异步阶段精确更新
	testInvocationID uint // 0 表示无测试机阶段
	testTaskID       uint
	prodInvocations  []uint   // len = len(prodBatches)
	prodTaskIDs      [][]uint // 与 prodBatches 同形状
}

// preCreateInvocationsAndTasks 在事务内预创建本 dispatch 涉及的所有行：
//
//	1 行 agent_command_dispatch
//	+ 1~2 行 agent_command_invocations
//	+ N 行 agent_command_tasks
//
// 写入后通过 plan 的 dispatchID / testInvocationID / testTaskID /
// prodInvocations / prodTaskIDs 回填主键。
func preCreateInvocationsAndTasks(tx *gorm.DB, plan *dispatchPlan) error {
	now := plan.startedAt

	// 总目标数（含测试机 + 离线被剔除的目标）
	totalTargets := 0
	for _, b := range plan.prodBatches {
		totalTargets += len(b)
	}
	if plan.testFirst {
		totalTargets++
	}
	totalTargets += len(plan.offlineInstanceIDs)

	// dispatch 顶层行
	d := &model.AgentCommandDispatch{
		Slug:                 plan.dispatchSlug,
		CommandID:            plan.commandID,
		CommandSnapshot:      plan.commandSnapshot,
		ParamValuesJSON:      plan.paramValuesJSON,
		TriggeredByUserID:    plan.triggeredByUID,
		TestFirst:            plan.testFirst,
		TestTargetInstanceID: plan.testInstanceID,
		TargetCount:          uint(totalTargets),
		Status:               model.AgentDispatchStatusInProgress,
		StartedAt:            now,
	}
	if err := tx.Create(d).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCmdTaskCreateDispatchFailed)
	}
	plan.dispatchID = d.ID

	// 测试机阶段（如启用）
	if plan.testFirst && plan.testInstanceID != 0 {
		inv := &model.AgentCommandInvocation{
			DispatchID:   plan.dispatchID,
			DispatchSlug: plan.dispatchSlug,
			IsTestRun:    true,
			BatchIndex:   0,
			TargetCount:  1,
			Status:       model.AgentInvocationStatusPending,
			StartedAt:    &now,
		}
		if err := tx.Create(inv).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCmdTaskCreateTestInvocationFailed)
		}
		plan.testInvocationID = inv.ID

		ins := plan.instancesByID[plan.testInstanceID]
		t := &model.AgentCommandTask{
			DispatchID:   plan.dispatchID,
			InvocationID: inv.ID,
			DispatchSlug: plan.dispatchSlug,
			InstanceID:   plan.testInstanceID,
			IsTestTarget: true,
			Status:       model.AgentTaskStatusPending,
		}
		fillTaskAgentMeta(t, ins)
		if err := tx.Create(t).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCmdTaskCreateTestTaskFailed)
		}
		plan.testTaskID = t.ID
	}

	// 生产阶段
	plan.prodInvocations = make([]uint, len(plan.prodBatches))
	plan.prodTaskIDs = make([][]uint, len(plan.prodBatches))
	for i, batch := range plan.prodBatches {
		inv := &model.AgentCommandInvocation{
			DispatchID:   plan.dispatchID,
			DispatchSlug: plan.dispatchSlug,
			IsTestRun:    false,
			BatchIndex:   uint(i + 1),
			TargetCount:  uint(len(batch)),
			Status:       model.AgentInvocationStatusPending,
			StartedAt:    &now,
		}
		if err := tx.Create(inv).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCmdTaskCreateProdInvocationFailed, i)
		}
		plan.prodInvocations[i] = inv.ID

		taskIDs := make([]uint, 0, len(batch))
		for _, instanceID := range batch {
			ins := plan.instancesByID[instanceID]
			t := &model.AgentCommandTask{
				DispatchID:   plan.dispatchID,
				InvocationID: inv.ID,
				DispatchSlug: plan.dispatchSlug,
				InstanceID:   instanceID,
				IsTestTarget: false,
				Status:       model.AgentTaskStatusPending,
			}
			fillTaskAgentMeta(t, ins)
			if err := tx.Create(t).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgCmdTaskCreateProdTaskFailed)
			}
			taskIDs = append(taskIDs, t.ID)
		}
		plan.prodTaskIDs[i] = taskIDs
	}

	// 离线目标（尽力而为下发）：单独建一条终态失败 invocation + unreachable task，不发 TAT。
	// HTTP 即时下发 offlineInstanceIDs 恒为空，此分支不触发，行为与之前一致。
	if len(plan.offlineInstanceIDs) > 0 {
		inv := &model.AgentCommandInvocation{
			DispatchID:   plan.dispatchID,
			DispatchSlug: plan.dispatchSlug,
			IsTestRun:    false,
			BatchIndex:   uint(len(plan.prodBatches) + 1),
			TargetCount:  uint(len(plan.offlineInstanceIDs)),
			FailedCount:  uint(len(plan.offlineInstanceIDs)),
			Status:       model.AgentInvocationStatusFailed,
			StartedAt:    &now,
			FinishedAt:   &now,
		}
		if err := tx.Create(inv).Error; err != nil {
			return fmt.Errorf("create offline invocation: %w", err)
		}
		for _, instanceID := range plan.offlineInstanceIDs {
			ins := plan.instancesByID[instanceID]
			t := &model.AgentCommandTask{
				DispatchID:   plan.dispatchID,
				InvocationID: inv.ID,
				DispatchSlug: plan.dispatchSlug,
				InstanceID:   instanceID,
				IsTestTarget: false,
				Status:       model.AgentTaskStatusUnreachable,
				FinishedAt:   &now,
			}
			fillTaskAgentMeta(t, ins)
			if err := tx.Create(t).Error; err != nil {
				return fmt.Errorf("create offline task: %w", err)
			}
		}
	}
	return nil
}

// fillTaskAgentMeta 把 instance 元信息冗余到 task 行。
func fillTaskAgentMeta(t *model.AgentCommandTask, ins *model.Instance) {
	if ins == nil {
		return
	}
	t.CVMInstanceID = ins.InstanceId
	t.AgentName = ins.Name
	// owner username 在异步阶段补；同步链路如已有 user 表查询可省，避免阻塞 dispatch
}

// runDispatchAsync 在 goroutine 里执行实际的 TAT RunCommand 调用与状态推进。
//
// 注意：所有 DB 调用走 model.DB(ctx) 让 identifier 回调注入；不允许 raw SQL（hatchery/CLAUDE.md §4）。
func runDispatchAsync(
	ctx context.Context,
	plan *dispatchPlan,
	scriptContent string,
	timeoutSec uint,
	runUser, workdir string,
	finalParams map[string]string,
) {
	defer func() {
		if rcv := recover(); rcv != nil {
			slog.Error("[AgentCommand] runDispatchAsync panic", "dispatch", plan.dispatchSlug, "error", rcv)
		}
	}()

	timeout := uint64(timeoutSec)

	// 1. 测试机阶段
	if plan.testFirst && plan.testInvocationID != 0 {
		ok := executeAndWaitTestRun(ctx, plan, scriptContent, timeout, runUser, workdir, finalParams)
		if !ok {
			// 测试机失败：整 dispatch 标记 failed；剩余 invocation/task 保持 pending（"未触发"）
			markDispatchAllFailed(ctx, plan)
			return
		}
		// ★ 测试机成功后到此为止 —— 不再自动衔接生产批。
		// dispatch.status 由 recalcDispatchStatus 推到 awaiting_confirmation，
		// 等待用户调 `POST /admin/agent-commands/dispatch` 携带 dispatch_slug 触发续跑或终止。
		recalcDispatchStatus(ctx, plan.dispatchID)
		return
	}

	// 2. 非测试机模式：直接跑全部生产批次（与改造前一致）
	for i, batch := range plan.prodBatches {
		executeProdBatch(ctx, plan, i, batch, scriptContent, timeout, runUser, workdir, finalParams)
	}
	recalcDispatchStatus(ctx, plan.dispatchID)
}

// executeAndWaitTestRun 调 TAT 下发测试机命令，轮询至终态，返回是否成功。
func executeAndWaitTestRun(
	ctx context.Context,
	plan *dispatchPlan,
	scriptContent string, timeout uint64,
	runUser, workdir string, finalParams map[string]string,
) bool {
	ins := plan.instancesByID[plan.testInstanceID]
	if ins == nil {
		slog.Error("[AgentCommand] test instance not found in plan", "dispatch", plan.dispatchSlug)
		return false
	}
	invID, bindings, err := RunInlineCommandBatchAsync(ctx,
		[]string{ins.InstanceId}, scriptContent, timeout, runUser, workdir, finalParams)
	if err != nil {
		slog.Error("[AgentCommand] test RunCommand 失败", "dispatch", plan.dispatchSlug, "error", err)
		_ = setInvocationStatus(ctx, plan.testInvocationID, model.AgentInvocationStatusFailed, "")
		_ = setTaskFailed(ctx, plan.testTaskID, model.AgentTaskStatusUnreachable, nil)
		return false
	}

	// 回写 TAT IDs
	_ = setInvocationStatus(ctx, plan.testInvocationID, model.AgentInvocationStatusInProgress, invID)
	if len(bindings) > 0 {
		_ = setTaskTATID(ctx, plan.testTaskID, bindings[0].InvocationTaskID)
	}

	// 同步轮询测试任务到终态（最多等待 timeout + 30s 安全余量）
	deadline := time.Now().Add(time.Duration(timeout+30) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(2 * time.Second):
		}
		t, err := loadTaskByID(ctx, plan.testTaskID)
		if err != nil {
			continue
		}
		if t.TATInvocationTaskID == "" {
			// binding 还没回写，跳过本轮
			continue
		}
		// 拉一次 TAT
		details, rerr := DescribeInvocationTasksBatch(ctx, []string{t.TATInvocationTaskID})
		if rerr != nil {
			continue
		}
		d, ok := details[t.TATInvocationTaskID]
		if !ok {
			continue
		}
		applyTATDetailToTask(ctx, t, d)
		if model.IsTerminalAgentTaskStatus(mapTATToAgentStatusFromDetail(d)) {
			updateInvocationCountsFromTasks(ctx, plan.testInvocationID)
			final, _ := loadTaskByID(ctx, plan.testTaskID)
			return final != nil && final.Status == model.AgentTaskStatusSuccess
		}
	}
	// 超时未到终态视为失败
	_ = setTaskFailed(ctx, plan.testTaskID, model.AgentTaskStatusTimeout, nil)
	updateInvocationCountsFromTasks(ctx, plan.testInvocationID)
	return false
}

// executeProdBatch 调用 TAT 下发一批生产 task 并写回 TAT IDs；不轮询，状态推进交给全局 poller。
func executeProdBatch(
	ctx context.Context,
	plan *dispatchPlan,
	batchIdx int,
	batchInstanceIDs []uint,
	scriptContent string, timeout uint64,
	runUser, workdir string, finalParams map[string]string,
) {
	cvmIDs := make([]string, 0, len(batchInstanceIDs))
	for _, iid := range batchInstanceIDs {
		ins := plan.instancesByID[iid]
		if ins != nil {
			cvmIDs = append(cvmIDs, ins.InstanceId)
		}
	}
	invocationID := plan.prodInvocations[batchIdx]
	taskIDs := plan.prodTaskIDs[batchIdx]

	invID, bindings, err := RunInlineCommandBatchAsync(ctx,
		cvmIDs, scriptContent, timeout, runUser, workdir, finalParams)
	if err != nil {
		slog.Error("[AgentCommand] prod RunCommand 失败",
			"dispatch", plan.dispatchSlug, "batch", batchIdx, "error", err)
		_ = setInvocationStatus(ctx, invocationID, model.AgentInvocationStatusFailed, "")
		for _, tid := range taskIDs {
			_ = setTaskFailed(ctx, tid, model.AgentTaskStatusUnreachable, nil)
		}
		updateInvocationCountsFromTasks(ctx, invocationID)
		return
	}

	_ = setInvocationStatus(ctx, invocationID, model.AgentInvocationStatusInProgress, invID)

	// 把 binding 按 cvm_instance_id 匹配回填 task
	cvmToTaskID := make(map[string]uint, len(taskIDs))
	for i, iid := range batchInstanceIDs {
		ins := plan.instancesByID[iid]
		if ins != nil {
			cvmToTaskID[ins.InstanceId] = taskIDs[i]
		}
	}
	for _, b := range bindings {
		tid := cvmToTaskID[b.InstanceID]
		if tid == 0 {
			continue
		}
		_ = setTaskTATID(ctx, tid, b.InvocationTaskID)
		_ = setTaskStatus(ctx, tid, model.AgentTaskStatusInProgress)
	}
}

// ============================================================================
// 状态写入工具
// ============================================================================

func setInvocationStatus(ctx context.Context, id uint, status, tatID string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if tatID != "" {
		updates["tat_invocation_id"] = tatID
	}
	if status == model.AgentInvocationStatusSuccess ||
		status == model.AgentInvocationStatusPartial ||
		status == model.AgentInvocationStatusFailed {
		now := time.Now()
		updates["finished_at"] = &now
	}
	return model.DB(ctx).Model(&model.AgentCommandInvocation{}).
		Where("id = ?", id).Updates(updates).Error
}

func setTaskTATID(ctx context.Context, id uint, tatTaskID string) error {
	return model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("id = ?", id).Update("tat_invocation_task_id", tatTaskID).Error
}

func setTaskStatus(ctx context.Context, id uint, status string) error {
	updates := map[string]any{"status": status, "updated_at": time.Now()}
	if model.IsTerminalAgentTaskStatus(status) {
		now := time.Now()
		updates["finished_at"] = &now
	}
	return model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("id = ?", id).Updates(updates).Error
}

func setTaskFailed(ctx context.Context, id uint, status string, exitCode *int) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	now := time.Now()
	updates["finished_at"] = &now
	if exitCode != nil {
		updates["exit_code"] = exitCode
	}
	return model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("id = ?", id).Updates(updates).Error
}

func loadTaskByID(ctx context.Context, id uint) (*model.AgentCommandTask, error) {
	var t model.AgentCommandTask
	if err := model.DB(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// applyTATDetailToTask 把 TAT 查询返回的详情写回 task 行（不写 stdout/stderr —— 决策 Q10）。
func applyTATDetailToTask(ctx context.Context, t *model.AgentCommandTask, d InvocationTaskDetail) {
	newStatus := MapTATTaskStatusToAgentTaskStatus(d.TaskStatus)
	updates := map[string]any{
		"status":     newStatus,
		"updated_at": time.Now(),
	}
	if d.ExitCode != nil {
		ec := int(*d.ExitCode)
		updates["exit_code"] = ec
	}
	if d.StartTime != nil {
		updates["started_at"] = d.StartTime
	}
	if d.EndTime != nil {
		updates["finished_at"] = d.EndTime
		if d.StartTime != nil {
			ms := uint(d.EndTime.Sub(*d.StartTime).Milliseconds())
			// 即使 start_time == end_time（TAT 秒级精度且任务很快）也写入 0，
			// 避免 elapsed_ms 字段为 null 让前端无法判断"已结束"。
			updates["elapsed_ms"] = ms
		}
	}
	_ = model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("id = ?", t.ID).Updates(updates).Error
}

func mapTATToAgentStatusFromDetail(d InvocationTaskDetail) string {
	if d.TaskStatus == "" {
		return model.AgentTaskStatusInProgress
	}
	return MapTATTaskStatusToAgentTaskStatus(d.TaskStatus)
}

// updateInvocationCountsFromTasks 重新统计某 invocation 下 task 的 success/failed_count，并按规则推 status。
//
// 调用末尾会自动触发 dispatch 状态重算（recalcDispatchStatusByInvocationID）。
func updateInvocationCountsFromTasks(ctx context.Context, invocationID uint) {
	updateInvocationCountsFromTasksNoRecalc(ctx, invocationID)
	// 推 dispatch.status：invocation 状态变化是 dispatch 状态变化的最常见触发源。
	recalcDispatchStatusByInvocationID(ctx, invocationID)
}

// updateInvocationCountsFromTasksNoRecalc 与 updateInvocationCountsFromTasks 等价，
// 但不触发 dispatch 重算。专给 recalcDispatchStatus 内部调用，避免递归。
func updateInvocationCountsFromTasksNoRecalc(ctx context.Context, invocationID uint) {
	var tasks []model.AgentCommandTask
	if err := model.DB(ctx).Where("invocation_id = ?", invocationID).Find(&tasks).Error; err != nil {
		slog.Error("[AgentCommand] reload tasks failed", "invocation_id", invocationID, "error", err)
		return
	}
	var (
		success, failed, terminal, total int
	)
	total = len(tasks)
	for _, t := range tasks {
		if t.Status == model.AgentTaskStatusSuccess {
			success++
			terminal++
		} else if model.IsFailureAgentTaskStatus(t.Status) {
			failed++
			terminal++
		} else if t.Status == model.AgentTaskStatusCancelled {
			terminal++
		}
	}
	// 决定 invocation status
	newStatus := model.AgentInvocationStatusInProgress
	if terminal == total && total > 0 {
		switch {
		case failed == 0:
			newStatus = model.AgentInvocationStatusSuccess
		case success == 0:
			newStatus = model.AgentInvocationStatusFailed
		default:
			newStatus = model.AgentInvocationStatusPartial
		}
	}
	updates := map[string]any{
		"success_count": uint(success),
		"failed_count":  uint(failed),
		"status":        newStatus,
		"updated_at":    time.Now(),
	}
	if newStatus != model.AgentInvocationStatusInProgress {
		now := time.Now()
		updates["finished_at"] = &now
	}
	_ = model.DB(ctx).Model(&model.AgentCommandInvocation{}).
		Where("id = ?", invocationID).Updates(updates).Error
}

// markDispatchAllFailed 测试机失败时，把整 dispatch 的所有 invocation 状态设为 failed（保留 task pending="未触发"）。
//
// 实现细节：
//   - 测试 invocation 已经在 executeAndWaitTestRun 内更新过 status；
//   - 这里只把所有"还在 pending"的 invocation 置 failed，不动它们的 task（保持 pending = "未触发"）
//   - 末尾调 recalcDispatchStatus 推 dispatch 终态
func markDispatchAllFailed(ctx context.Context, plan *dispatchPlan) {
	_ = model.DB(ctx).Model(&model.AgentCommandInvocation{}).
		Where("dispatch_id = ? AND status = ?",
			plan.dispatchID, model.AgentInvocationStatusPending).
		Updates(map[string]any{
			"status":      model.AgentInvocationStatusFailed,
			"finished_at": time.Now(),
			"updated_at":  time.Now(),
		}).Error
	recalcDispatchStatus(ctx, plan.dispatchID)
}

// agentDispatchReconcileBatchLimit 单次 reconcile 拉取的 dispatch 行数上限。
const agentDispatchReconcileBatchLimit = 200

// RunAgentCommandDispatchReconcileOnce 单次 reconcile：
//  1. 查 status IN (in_progress, awaiting_confirmation) 的 dispatch
//  2. 对每行调 recalcDispatchStatus 重新计算状态 + 计数 + finished_at
//
// 由 task scheduler 每 60s 调一次，作为事件式状态推进的兜底。
//
// 参考 RunAgentCommandPollerOnce 的注释：scheduler 已注入完整 TenantSnapshot，
// 下游 model.DB(ctx) 走 identifier 回调按当前租户过滤。
func RunAgentCommandDispatchReconcileOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[AgentDispatchReconcile] tick panic", "error", r)
		}
	}()

	dispatches, err := model.FindUnfinishedDispatches(ctx, agentDispatchReconcileBatchLimit)
	if err != nil {
		slog.Error("[AgentDispatchReconcile] 查询未终态 dispatch 失败", "error", err)
		return
	}
	for i := range dispatches {
		recalcDispatchStatus(ctx, dispatches[i].ID)
	}
}

// ============================================================================
// 全局轮询：单次执行
// ============================================================================
//
// 真正的调度由 task scheduler 框架（task/agent_command_poller.go 注册）负责。
// 本 package 只负责实现"一次轮询做什么"，与 task 包解耦避免循环依赖。

// RunAgentCommandPollerOnce 单次轮询：
//  1. 查 status='in_progress' 且 tat_invocation_task_id != "" 的 task
//  2. 批量调 DescribeInvocationTasksBatch
//  3. 对终态 task 写回 status / exit_code / elapsed_ms
//  4. 触发其所属 invocation 的状态重计算
//
// 由 task scheduler 每 5s 调一次，scheduler 已注入完整 TenantSnapshot
// （含 Uin / AKSK / Domain），下游 getCredential 走 STS 路径正常。
// ⚠️ 不要再回到「用 DBGlobal 跨租户拉 + InjectTenant({Identifier:...}) stub
// 替换 ctx」的写法：那样会丢掉 Uin → 永久凭证路径 → STS-only 部署 TAT
// 调用静默失败 → task 永远 in_progress。
func RunAgentCommandPollerOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[AgentCommandPoller] tick panic", "error", r)
		}
	}()

	// 用 model.DB(ctx) 走 GORM identifier 回调，按 poller 启动时继承的
	// FixedSnapshot.Identifier 自动过滤（stage-1 单租户场景）。
	//
	// ⚠️ 不要再用 DBGlobal + 按 identifier 分组 + InjectTenant({Identifier:...})
	// 这条路：那种写法会把原 ctx 上完整的 TenantSnapshot（含 Uin / AKSK /
	// Domain）替换成只剩 Identifier 的 stub，下游 getCredential 拿不到
	// Uin → 走永久凭证路径；STS-only 部署里凭证根本无效，TAT 调用静默失败，
	// task 状态永远停在 in_progress（用户体感"detail 调一次就好了"的根因）。
	var tasks []model.AgentCommandTask
	if err := model.DB(ctx).Model(&model.AgentCommandTask{}).
		Where("status = ? AND tat_invocation_task_id <> ?",
			model.AgentTaskStatusInProgress, "").
		Order("id asc").
		Limit(agentPollerBatchLimit).
		Find(&tasks).Error; err != nil {
		slog.Error("[AgentCommandPoller] 查询未终态 task 失败", "error", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.TATInvocationTaskID)
	}
	details, err := DescribeInvocationTasksBatch(ctx, ids)
	if err != nil {
		slog.Warn("[AgentCommandPoller] DescribeInvocationTasksBatch 失败",
			"error", err)
		return
	}
	affectedInvocations := make(map[uint]struct{})
	affectedDispatches := make(map[uint]struct{})
	for i := range tasks {
		t := &tasks[i]
		d, ok := details[t.TATInvocationTaskID]
		if !ok {
			continue
		}
		// 仅在状态变化时写回
		newStatus := MapTATTaskStatusToAgentTaskStatus(d.TaskStatus)
		if newStatus == t.Status && d.ExitCode == nil {
			continue
		}
		applyTATDetailToTask(ctx, t, d)
		affectedInvocations[t.InvocationID] = struct{}{}
		if t.DispatchID != 0 {
			affectedDispatches[t.DispatchID] = struct{}{}
		}
	}
	for invID := range affectedInvocations {
		updateInvocationCountsFromTasks(ctx, invID)
	}
	// updateInvocationCountsFromTasks 内部已会触发 recalcDispatchStatus（按 invocation 反查
	// dispatch_id），但若某些 invocation 没有 task 状态变化但 dispatch.cancelled_count
	// 等聚合需要重算，这里再批量推一次保证一致性。
	for dID := range affectedDispatches {
		recalcDispatchStatus(ctx, dID)
	}
}

// ============================================================================
// 列表 / 详情 handler
// ============================================================================

// HandleListAgentCommandTasks 执行记录列表（分页 dispatch 表）。
//
// v2 数据模型：dispatch 是顶层实体，分页直接基于 agent_command_dispatch 表，
// 不再扫描 invocation 行内存折叠。`status` / 时间过滤完全下推到 SQL。
func HandleListAgentCommandTasks(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	ctx := r.Context()
	page, pageSize := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	triggeredBy, _ := strconv.ParseUint(r.URL.Query().Get("triggered_by"), 10, 64)

	tx := model.DB(ctx).Model(&model.AgentCommandDispatch{})
	if triggeredBy > 0 {
		tx = tx.Where("triggered_by_user_id = ?", uint(triggeredBy))
	}
	if started := r.URL.Query().Get("started_after"); started != "" {
		if ts, err := time.Parse(time.RFC3339, started); err == nil {
			tx = tx.Where("started_at >= ?", ts)
		}
	}
	if started := r.URL.Query().Get("started_before"); started != "" {
		if ts, err := time.Parse(time.RFC3339, started); err == nil {
			tx = tx.Where("started_at < ?", ts)
		}
	}
	if statusFilter != "" {
		tx = tx.Where("status = ?", statusFilter)
	}
	if q != "" {
		// q 匹配 dispatch.slug / command_snapshot 文本（含命令名 / 命令内容）
		// 或操作人用户名命中的 user_id。
		matchedUserIDs := findUserIDsByUsernameLike(ctx, q)
		like := "%" + q + "%"
		if len(matchedUserIDs) > 0 {
			tx = tx.Where("slug LIKE ? OR command_snapshot LIKE ? OR triggered_by_user_id IN ?",
				like, like, matchedUserIDs)
		} else {
			tx = tx.Where("slug LIKE ? OR command_snapshot LIKE ?", like, like)
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	var dispatches []model.AgentCommandDispatch
	if err := tx.
		Order("created_at DESC, id DESC").
		Offset(offset).Limit(pageSize).
		Find(&dispatches).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 自愈：对当前页非终态的 dispatch 重算一次 status / counts，
	// 兜底事件式更新可能漏掉的情况。仅当前页（O(pageSize) 次 SQL）。
	for i := range dispatches {
		if !dispatches[i].IsTerminal() {
			recalcDispatchStatus(ctx, dispatches[i].ID)
		}
	}
	// 重读受影响的 dispatch（仅当存在非终态行时）
	if needRefresh := func() bool {
		for i := range dispatches {
			if !dispatches[i].IsTerminal() {
				return true
			}
		}
		return false
	}(); needRefresh {
		ids := make([]uint, 0, len(dispatches))
		for i := range dispatches {
			ids = append(ids, dispatches[i].ID)
		}
		var refreshed []model.AgentCommandDispatch
		if err := model.DB(ctx).Where("id IN ?", ids).Find(&refreshed).Error; err == nil {
			byID := make(map[uint]model.AgentCommandDispatch, len(refreshed))
			for _, d := range refreshed {
				byID[d.ID] = d
			}
			for i := range dispatches {
				if r2, ok := byID[dispatches[i].ID]; ok {
					dispatches[i] = r2
				}
			}
		}
	}

	// invocation_count 批量统计：当前页所有 dispatch
	invCountByDispatchID := make(map[uint]uint, len(dispatches))
	if len(dispatches) > 0 {
		ids := make([]uint, 0, len(dispatches))
		for i := range dispatches {
			ids = append(ids, dispatches[i].ID)
		}
		type invCntRow struct {
			DispatchID uint
			Cnt        uint
		}
		var rows []invCntRow
		if err := model.DB(ctx).
			Model(&model.AgentCommandInvocation{}).
			Select("dispatch_id, count(*) AS cnt").
			Where("dispatch_id IN ?", ids).
			Group("dispatch_id").
			Find(&rows).Error; err == nil {
			for _, r := range rows {
				invCountByDispatchID[r.DispatchID] = r.Cnt
			}
		}
	}

	// 取 trigger user 信息
	uidSet := make(map[uint]struct{}, len(dispatches))
	for i := range dispatches {
		uidSet[dispatches[i].TriggeredByUserID] = struct{}{}
	}
	uidList := make([]uint, 0, len(uidSet))
	for u := range uidSet {
		uidList = append(uidList, u)
	}
	users := batchUsersByIDs(ctx, uidList)

	tasks := make([]agentTaskListItem, 0, len(dispatches))
	for i := range dispatches {
		d := &dispatches[i]
		snap := decodeSnapshot(d.CommandSnapshot)
		name, _ := snap["name"].(string)
		content, _ := snap["content"].(string)
		preview := previewContent(content, 80)

		creator := users[d.TriggeredByUserID]
		tasks = append(tasks, agentTaskListItem{
			DispatchSlug:          d.Slug,
			CommandName:           name,
			CommandContentPreview: preview,
			TriggeredByUserID:     d.TriggeredByUserID,
			TriggeredByUsername:   creator.Username,
			TriggeredByEmail:      creator.Email,
			TestFirst:             d.TestFirst,
			Status:                d.Status,
			TargetCount:           d.TargetCount,
			SuccessCount:          d.SuccessCount,
			FailedCount:           d.FailedCount,
			InvocationCount:       invCountByDispatchID[d.ID],
			StartedAt:             d.StartedAt,
			FinishedAt:            d.FinishedAt,
		})
	}

	jsonOK(w, agentTaskListResp{Tasks: tasks, Total: total, Page: page, PageSize: pageSize})
}

// HandleAgentCommandTaskDetail 执行记录详情。
//
// with_output=true 时同步调 TAT 拉 stdout/stderr；TAT 整体失败 → 502。
//
// v2 数据模型：dispatch 是顶层实体；invocations[] 节点上保留 command_snapshot /
// param_values / triggered_by 等冗余字段（写时从 dispatch 复制），保持响应结构兼容。
func HandleAgentCommandTaskDetail(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	ctx := r.Context()
	dispatchSlug := r.URL.Query().Get("dispatch_slug")
	if dispatchSlug == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgDispatchSlugRequiredDetail).WithPrefix("dispatch_slug_required"))
		return
	}
	withOutput := strings.EqualFold(r.URL.Query().Get("with_output"), "true") ||
		r.URL.Query().Get("with_output") == ""

	d, err := model.FindDispatchBySlug(ctx, dispatchSlug)
	if err != nil {
		if errors.Is(err, model.ErrDispatchNotFound) {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgDispatchNotFoundDetail).WithPrefix("dispatch_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	invs, err := model.FindInvocationsByDispatchID(ctx, d.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	tasks, err := model.FindTasksByDispatchSlug(ctx, dispatchSlug)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 拉 TAT stdout/stderr（按需）
	var tatDetails map[string]InvocationTaskDetail
	if withOutput {
		ids := make([]string, 0, len(tasks))
		for _, t := range tasks {
			if t.TATInvocationTaskID != "" {
				ids = append(ids, t.TATInvocationTaskID)
			}
		}
		details, err := DescribeInvocationTasksBatch(ctx, ids)
		if err != nil {
			writeError(w, r, http.StatusBadGateway,
				hcommon.I18nError(i18n.MsgTATUnavailable).WithPrefix("tat_unavailable"))
			return
		}
		tatDetails = details
		// 顺便：TAT 终态而本地仍 in_progress 时同步刷写
		affected := make(map[uint]struct{})
		for i := range tasks {
			t := &tasks[i]
			dt, ok := details[t.TATInvocationTaskID]
			if !ok {
				continue
			}
			if t.Status == model.AgentTaskStatusInProgress &&
				model.IsTerminalAgentTaskStatus(MapTATTaskStatusToAgentTaskStatus(dt.TaskStatus)) {
				applyTATDetailToTask(ctx, t, dt)
				affected[t.InvocationID] = struct{}{}
			}
		}
		for invID := range affected {
			updateInvocationCountsFromTasks(ctx, invID)
		}
	}

	// 推 dispatch.status（事件式 + 当前请求自愈）
	recalcDispatchStatus(ctx, d.ID)

	// 重新加载 dispatch / invocation / task 拿到刚刚回写的最新值
	d, _ = model.FindDispatchBySlug(ctx, dispatchSlug)
	invs, _ = model.FindInvocationsByDispatchID(ctx, d.ID)
	tasks, _ = model.FindTasksByDispatchSlug(ctx, dispatchSlug)

	// 构造响应
	resp := buildDispatchDetail(ctx, d, invs, tasks, tatDetails, withOutput)
	jsonOK(w, resp)
}

// buildDispatchDetail 用 dispatch 顶层行 + invocations + tasks 构造详情响应。
//
// 兼容性：invocations[] 节点上保留 command_snapshot / param_values 等冗余字段
// （从 dispatch 行复制），保持与 v1 响应结构一致。
func buildDispatchDetail(
	ctx context.Context,
	d *model.AgentCommandDispatch,
	invs []model.AgentCommandInvocation,
	tasks []model.AgentCommandTask,
	tatDetails map[string]InvocationTaskDetail,
	withOutput bool,
) agentTaskDetailResp {
	commandSnapshot := decodeSnapshot(d.CommandSnapshot)
	var paramsValues map[string]string
	_ = json.Unmarshal([]byte(d.ParamValuesJSON), &paramsValues)

	// 测试机汇总
	tasksByInvocationID := make(map[uint][]model.AgentCommandTask)
	for _, t := range tasks {
		tasksByInvocationID[t.InvocationID] = append(tasksByInvocationID[t.InvocationID], t)
	}
	var (
		testSummary     *agentTestSummary
		testTargetIDPtr *uint
	)
	if d.TestFirst {
		for _, t := range tasks {
			if t.IsTestTarget {
				tt := t.InstanceID
				testTargetIDPtr = &tt
				summary := &agentTestSummary{InstanceID: t.InstanceID}
				summary.Passed = (t.Status == model.AgentTaskStatusSuccess)
				if t.ExitCode != nil {
					ec := *t.ExitCode
					summary.ExitCode = &ec
				}
				if t.ElapsedMs != nil {
					ms := *t.ElapsedMs
					summary.ElapsedMs = &ms
				}
				if dd, ok := tatDetails[t.TATInvocationTaskID]; ok {
					summary.OutputPreview = trimRunes(dd.Stdout, AgentTaskOutputPreviewBytes)
				}
				testSummary = summary
				break
			}
		}
	}

	// invocations[].tasks[]
	invocations := make([]agentInvocationDetail, 0, len(invs))
	for _, inv := range invs {
		taskDetails := make([]agentTaskInDetail, 0)
		for _, t := range tasksByInvocationID[inv.ID] {
			taskDetails = append(taskDetails, buildTaskInDetail(t, tatDetails, withOutput))
		}
		invocations = append(invocations, agentInvocationDetail{
			TATInvocationID: inv.TATInvocationID,
			IsTestRun:       inv.IsTestRun,
			BatchIndex:      inv.BatchIndex,
			Status:          inv.Status,
			TargetCount:     inv.TargetCount,
			SuccessCount:    inv.SuccessCount,
			FailedCount:     inv.FailedCount,
			StartedAt:       inv.StartedAt,
			FinishedAt:      inv.FinishedAt,
			Tasks:           taskDetails,
		})
	}
	// 排序：is_test_run desc, batch_index asc
	sort.SliceStable(invocations, func(i, j int) bool {
		if invocations[i].IsTestRun != invocations[j].IsTestRun {
			return invocations[i].IsTestRun
		}
		return invocations[i].BatchIndex < invocations[j].BatchIndex
	})

	creator := batchUsersByIDs(ctx, []uint{d.TriggeredByUserID})[d.TriggeredByUserID]
	return agentTaskDetailResp{
		DispatchSlug:         d.Slug,
		Status:               d.Status,
		TestFirst:            d.TestFirst,
		TestTargetInstanceID: testTargetIDPtr,
		TargetCount:          d.TargetCount,
		SuccessCount:         d.SuccessCount,
		FailedCount:          d.FailedCount,
		TriggeredByUserID:    d.TriggeredByUserID,
		TriggeredByUsername:  creator.Username,
		TriggeredByEmail:     creator.Email,
		StartedAt:            d.StartedAt,
		FinishedAt:           d.FinishedAt,
		CommandSnapshot:      commandSnapshot,
		ParamValues:          paramsValues,
		TestSummary:          testSummary,
		Invocations:          invocations,
	}
}

func buildTaskInDetail(t model.AgentCommandTask, tatDetails map[string]InvocationTaskDetail, withOutput bool) agentTaskInDetail {
	out := agentTaskInDetail{
		TATInvocationTaskID: t.TATInvocationTaskID,
		InstanceID:          t.InstanceID,
		CVMInstanceID:       t.CVMInstanceID,
		AgentName:           t.AgentName,
		OwnerUsername:       t.OwnerUsername,
		IsTestTarget:        t.IsTestTarget,
		Status:              t.Status,
		StartedAt:           t.StartedAt,
		FinishedAt:          t.FinishedAt,
	}
	if t.ExitCode != nil {
		ec := *t.ExitCode
		out.ExitCode = &ec
	}
	if t.ElapsedMs != nil {
		ms := *t.ElapsedMs
		out.ElapsedMs = &ms
	}
	if !withOutput {
		return out
	}
	// 取 TAT detail
	if t.TATInvocationTaskID == "" {
		return out
	}
	d, ok := tatDetails[t.TATInvocationTaskID]
	if !ok {
		out.OutputExpired = true
		return out
	}
	// 透传 TAT 顶层 ErrorInfo（DELIVER_FAILED / START_FAILED 时承载错误描述）
	out.ErrorInfo = d.ErrorInfo
	stdout := d.Stdout
	if len(stdout) > AgentTaskOutputMaxBytes {
		stdout = stdout[:AgentTaskOutputMaxBytes]
		out.StdoutTruncated = true
	}
	out.Stdout = &stdout
	stderr := d.Stderr
	if len(stderr) > AgentTaskOutputMaxBytes {
		stderr = stderr[:AgentTaskOutputMaxBytes]
		out.StderrTruncated = true
	}
	out.Stderr = &stderr
	return out
}

// ============================================================================
// 工具函数
// ============================================================================

func decodeSnapshot(s string) map[string]any {
	out := make(map[string]any)
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// previewContent 按 Unicode 字符数截取命令内容预览，超长加省略号。
// 必须按 rune 截断而非字节切片（s[:n]），否则中文等多字节 UTF-8 字符会
// 被切到字节中间，产生乱码。
func previewContent(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return model.TruncateRunes(s, n) + "..."
}

// trimRunes 按 Unicode 字符数截断字符串（不加省略号），用于 stdout/stderr
// 预览。同样不能用 s[:n]，否则中文会被截断成乱码。
func trimRunes(s string, n int) string {
	return model.TruncateRunes(s, n)
}

func dedupUintSlice(in []uint) []uint {
	seen := make(map[uint]struct{}, len(in))
	out := make([]uint, 0, len(in))
	for _, v := range in {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func chunkUint(in []uint, size int) [][]uint {
	if size <= 0 || len(in) == 0 {
		return nil
	}
	out := make([][]uint, 0, (len(in)+size-1)/size)
	for i := 0; i < len(in); i += size {
		end := i + size
		if end > len(in) {
			end = len(in)
		}
		out = append(out, in[i:end])
	}
	return out
}

func indexInstancesByID(in []model.Instance) map[uint]*model.Instance {
	out := make(map[uint]*model.Instance, len(in))
	for i := range in {
		out[in[i].ID] = &in[i]
	}
	return out
}

// isInstanceRunning 当前根据 LastCVMState / LastStableState 判断是否运行中。
// 项目里不同地方对状态字段的写法不统一，这里以 LastCVMState 优先。
func isInstanceRunning(ins *model.Instance) bool {
	if ins == nil {
		return false
	}
	if ins.LastCVMState != "" {
		return strings.EqualFold(ins.LastCVMState, "RUNNING")
	}
	if ins.LastStableState != "" {
		return strings.EqualFold(ins.LastStableState, "running")
	}
	// 缺乏稳定状态信息时保守视为未就绪，避免空跑 TAT
	return false
}

// generateUniqueDispatchSlug 生成同租户唯一的 dispatch_slug，最多重试 retries 次。
func generateUniqueDispatchSlug(ctx context.Context, retries int) (string, error) {
	for i := 0; i < retries; i++ {
		slug := model.GenerateAgentDispatchSlug()
		var n int64
		if err := model.DB(ctx).Model(&model.AgentCommandDispatch{}).
			Where("slug = ?", slug).Count(&n).Error; err != nil {
			return "", hcommon.I18nRichError(err, i18n.MsgCmdTaskCheckDispatchSlugFailed)
		}
		if n == 0 {
			return slug, nil
		}
	}
	return "", model.ErrDispatchSlugConflict
}

// buildCommandSnapshot 把命令模板字段冻结成 JSON 字符串入 invocation.command_snapshot。
//
// 决策 §8.1：snapshot 内的 content / params 全部使用 raw 文本，base64 不参与。
func buildCommandSnapshot(cmd *model.AgentCommand, dispatchSlug string) ([]byte, error) {
	snapshot := map[string]any{
		"command_id":     cmd.ID,
		"slug":           cmd.Slug,
		"name":           cmd.Name,
		"description":    cmd.Description,
		"type":           cmd.Type,
		"content":        cmd.Content, // raw
		"timeout_sec":    cmd.TimeoutSec,
		"run_user":       cmd.RunUser,
		"workdir":        cmd.Workdir,
		"params":         cmd.Params(),
		"_dispatch_slug": dispatchSlug,
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCmdTaskBuildSnapshotFailed)
	}
	return data, nil
}

// assembleParamValues 按命令模板 params 定义 + 用户填值组装最终 {name: value}。
//
// 用 map 的 `ok` 区分两种"无值"场景：
//   - key 不在 map 里：用户没传 → 走 default 兜底；无 default 时 param_value_required
//   - key 在 map 里但 value=""：用户**显式传了空字符串** → 透传为空，不走 default
//
// 这样产品交互上「在表单里把字段清空」和「根本没动那个字段」语义就分开了。
//
// 错误：
//   - param_value_required: 缺少必填参数（命令声明、无 default、key 也没在 map 里）
//   - param_unknown: 提供了命令未声明的参数
func assembleParamValues(cmd *model.AgentCommand, userValues map[string]string) (map[string]string, error) {
	params := cmd.Params()
	declared := make(map[string]model.AgentCommandParam, len(params))
	for _, p := range params {
		declared[p.Name] = p
	}
	final := make(map[string]string, len(params))
	missing := make([]string, 0)
	for _, p := range params {
		v, ok := userValues[p.Name]
		if !ok {
			if p.Default != "" {
				final[p.Name] = p.Default
				continue
			}
			missing = append(missing, p.Name)
			continue
		}
		// 用户显式传了（哪怕是空字符串）就尊重原值，不再回退 default
		final[p.Name] = v
	}
	if len(missing) > 0 {
		return nil, hcommon.I18nError(i18n.MsgCmdTaskParamValueRequired, strings.Join(missing, ", ")).WithPrefix("param_value_required")
	}
	unknown := make([]string, 0)
	for k := range userValues {
		if _, ok := declared[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, hcommon.I18nError(i18n.MsgCmdTaskParamUnknown, strings.Join(unknown, ", ")).WithPrefix("param_unknown")
	}
	return final, nil
}

// aggregateDispatchStatus 根据 invocation 列表 + sum 决定 dispatch 整体状态。
//
// 这是 recalcDispatchStatus 内部使用的纯函数，不读 DB。
//
// 优先级（从高到低）：
//  1. 测试 invocation 已终态成功 ∧ 至少 1 条非 test invocation 仍是 pending（即 tat_invocation_id 为空）→ awaiting_confirmation
//  2. 任意 invocation 非终态（pending/in_progress 都算非终态）→ in_progress
//  3. 任意 invocation 是 cancelled（用户主动放弃）→ cancelled
//  4. 否则按 success_count / failed_count 推 success / partial / failed
//
// 参数 anyInProgress 由调用方判定 (`!inv.IsTerminal()`)。
// pending 也会让 anyInProgress = true，所以 awaiting_confirmation 的判定要先于 anyInProgress。
func aggregateDispatchStatus(invs []model.AgentCommandInvocation, anyInProgress bool, success, failed uint) string {
	// 先看是否处于「测试机已通过 + 等待人为确认」状态
	if isAwaitingConfirmation(invs) {
		return model.AgentDispatchStatusAwaitingConfirmation
	}
	if anyInProgress {
		return model.AgentDispatchStatusInProgress
	}
	// 全部终态后，cancelled 优先于 success/partial/failed —— 用户视角：「我中止了」
	for i := range invs {
		if invs[i].Status == model.AgentInvocationStatusCancelled {
			return model.AgentDispatchStatusCancelled
		}
	}
	if failed > 0 && success > 0 {
		return model.AgentDispatchStatusPartial
	}
	if failed > 0 {
		return model.AgentDispatchStatusFailed
	}
	return model.AgentDispatchStatusSuccess
}

// isAwaitingConfirmation 判断 invocation 切片是否处于 awaiting_confirmation 状态。
//
// 条件：测试 invocation 已经终态成功（success / partial），且存在非 test invocation 仍是 pending
// 且尚未拿到 tat_invocation_id（即「预创建但 RunCommand 未发出」）。
//
// 测试 invocation 终态为 failed/cancelled 不进入 awaiting_confirmation —— 测试失败应直接走 failed 分支。
func isAwaitingConfirmation(invs []model.AgentCommandInvocation) bool {
	var hasTestSuccess bool
	var hasPendingProd bool
	for i := range invs {
		inv := &invs[i]
		if inv.IsTestRun {
			if inv.Status == model.AgentInvocationStatusSuccess ||
				inv.Status == model.AgentInvocationStatusPartial {
				hasTestSuccess = true
			}
			continue
		}
		// 非 test invocation
		if inv.Status == model.AgentInvocationStatusPending && inv.TATInvocationID == "" {
			hasPendingProd = true
		}
	}
	return hasTestSuccess && hasPendingProd
}

// recalcDispatchStatus 重新计算 dispatch 的整体状态、success/failed/cancelled 计数与
// finished_at，并把结果写回 dispatch 行。
//
// 调用时机（事件式）：
//   - invocation 状态变化（updateInvocationCountsFromTasks 末尾）
//   - markDispatchAllFailed 后
//   - handleDispatchAbort 后
//   - poller / detail / list 自愈刷新后
//
// 后台 reconcile 协程也会定期对未终态 dispatch 调本函数兜底。
//
// 计数语义：success_count / failed_count / cancelled_count 基于 task 行计数。
//   - success_count: task.status == success
//   - failed_count: TAT 报告的失败（failed/timeout/unreachable）
//   - cancelled_count: 用户终止下发的 task
//
// 终态时写入 finished_at = max(invocation.finished_at)；非终态清空 finished_at。
func recalcDispatchStatus(ctx context.Context, dispatchID uint) {
	if dispatchID == 0 {
		return
	}

	var d model.AgentCommandDispatch
	if err := model.DB(ctx).Where("id = ?", dispatchID).First(&d).Error; err != nil {
		slog.Warn("[AgentCommand] recalcDispatchStatus: load dispatch failed",
			"dispatch_id", dispatchID, "error", err)
		return
	}

	invs, err := model.FindInvocationsByDispatchID(ctx, dispatchID)
	if err != nil {
		slog.Warn("[AgentCommand] recalcDispatchStatus: load invocations failed",
			"dispatch_id", dispatchID, "error", err)
		return
	}

	var tasks []model.AgentCommandTask
	if err := model.DB(ctx).Where("dispatch_id = ?", dispatchID).
		Find(&tasks).Error; err != nil {
		slog.Warn("[AgentCommand] recalcDispatchStatus: load tasks failed",
			"dispatch_id", dispatchID, "error", err)
		return
	}

	// 自愈：先看是否有 invocation 非终态但其 task 已全部终态 —— 这种"卡死"行
	// 通常是 poller 写回了 task 但 updateInvocationCountsFromTasks 没跑成。
	// 修一下，再继续聚合到 dispatch。
	tasksByInv := make(map[uint][]model.AgentCommandTask)
	for _, t := range tasks {
		tasksByInv[t.InvocationID] = append(tasksByInv[t.InvocationID], t)
	}
	healed := false
	for i := range invs {
		inv := &invs[i]
		if inv.IsTerminal() {
			continue
		}
		group := tasksByInv[inv.ID]
		if len(group) == 0 {
			continue
		}
		allTerminal := true
		for _, tk := range group {
			if !model.IsTerminalAgentTaskStatus(tk.Status) {
				allTerminal = false
				break
			}
		}
		if allTerminal {
			updateInvocationCountsFromTasksNoRecalc(ctx, inv.ID)
			healed = true
		}
	}
	if healed {
		// 重新加载 invocation 拿到最新 status / counts
		invs, _ = model.FindInvocationsByDispatchID(ctx, dispatchID)
	}

	var (
		anyInProgress         bool
		invSuccess, invFailed uint
		latestFinished        *time.Time
	)
	for i := range invs {
		inv := &invs[i]
		if !inv.IsTerminal() {
			anyInProgress = true
		}
		invSuccess += inv.SuccessCount
		invFailed += inv.FailedCount
		if inv.FinishedAt != nil {
			if latestFinished == nil || inv.FinishedAt.After(*latestFinished) {
				ft := *inv.FinishedAt
				latestFinished = &ft
			}
		}
	}

	// task 维度计数（更准确）
	var successCnt, failedCnt, cancelledCnt uint
	for i := range tasks {
		switch tasks[i].Status {
		case model.AgentTaskStatusSuccess:
			successCnt++
		case model.AgentTaskStatusCancelled:
			cancelledCnt++
		default:
			if model.IsFailureAgentTaskStatus(tasks[i].Status) {
				failedCnt++
			}
		}
	}

	newStatus := aggregateDispatchStatus(invs, anyInProgress, invSuccess, invFailed)

	updates := map[string]any{
		"status":          newStatus,
		"success_count":   successCnt,
		"failed_count":    failedCnt,
		"cancelled_count": cancelledCnt,
		"updated_at":      time.Now(),
	}

	// finished_at: 终态时写入最晚 invocation.finished_at；非终态清空
	switch newStatus {
	case model.AgentDispatchStatusSuccess,
		model.AgentDispatchStatusPartial,
		model.AgentDispatchStatusFailed,
		model.AgentDispatchStatusCancelled:
		if latestFinished != nil {
			updates["finished_at"] = latestFinished
		} else {
			now := time.Now()
			updates["finished_at"] = &now
		}
	default:
		// in_progress / awaiting_confirmation 视为活跃
		updates["finished_at"] = nil
	}

	if err := model.DB(ctx).Model(&model.AgentCommandDispatch{}).
		Where("id = ?", dispatchID).Updates(updates).Error; err != nil {
		slog.Warn("[AgentCommand] recalcDispatchStatus: update dispatch failed",
			"dispatch_id", dispatchID, "error", err)
	}
}

// recalcDispatchStatusBySlug 便捷封装，按 dispatch_slug 反查 ID 后调 recalc。
func recalcDispatchStatusBySlug(ctx context.Context, slug string) {
	var d model.AgentCommandDispatch
	if err := model.DB(ctx).Select("id").Where("slug = ?", slug).First(&d).Error; err != nil {
		return
	}
	recalcDispatchStatus(ctx, d.ID)
}

// recalcDispatchStatusByInvocationID 便捷封装，按 invocation 反查 dispatch_id 后调 recalc。
func recalcDispatchStatusByInvocationID(ctx context.Context, invocationID uint) {
	var inv model.AgentCommandInvocation
	if err := model.DB(ctx).Select("dispatch_id").
		Where("id = ?", invocationID).First(&inv).Error; err != nil {
		return
	}
	if inv.DispatchID == 0 {
		return
	}
	recalcDispatchStatus(ctx, inv.DispatchID)
}
