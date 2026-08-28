package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

const (
	powerActionStart = "start"
	powerActionStop  = "stop"

	powerBatchMaxTargets = 1000
	cvmPowerBatchSize    = 100
)

type powerTargetRequest struct {
	ID          uint     `json:"id"`
	InstanceID  string   `json:"instance_id"`
	IDs         []uint   `json:"ids"`
	InstanceIDs []string `json:"instance_ids"`
}

func (req powerTargetRequest) IsBatch() bool {
	return len(req.IDs) > 0 || len(req.InstanceIDs) > 0
}

type powerTarget struct {
	InputID         uint
	InputInstanceID string
	Instance        *model.Instance
	Err             error
}

type powerActionResult struct {
	ID         uint   `json:"id,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

type powerActionResponse struct {
	OK      bool                `json:"ok"`
	Results []powerActionResult `json:"results"`
}

type powerTargetError struct {
	HTTPStatus int
	Status     InstanceStatusResponse
	Err        error
}

// HandleStartInstance POST /openclaw/start - 用户端开机（支持单个和批量）。
func HandleStartInstance(w http.ResponseWriter, r *http.Request) {
	handleUserPowerInstance(w, r, powerActionStart, defaultStatusResolver)
}

// HandleStopInstance POST /openclaw/stop - 用户端关机（支持单个和批量）。
func HandleStopInstance(w http.ResponseWriter, r *http.Request) {
	handleUserPowerInstance(w, r, powerActionStop, defaultStatusResolver)
}

func handleUserPowerInstance(w http.ResponseWriter, r *http.Request, action string, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	handlePowerInstanceAction(w, r, action, false, user.ID, resolver)
}

func handlePowerInstanceAction(w http.ResponseWriter, r *http.Request, action string, admin bool, userID uint, resolver instanceStatusResolver) {
	req, err := parsePowerTargetRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	targets, err := resolvePowerTargets(r, req, userID)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if len(targets) == 0 {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if !req.IsBatch() && targets[0].Instance != nil {
		w = WrapInstanceId(w, targets[0].Instance.InstanceId)
	}

	resp := &powerActionResponse{OK: true, Results: make([]powerActionResult, 0, len(targets))}
	readyTargets := make([]powerTarget, 0, len(targets))
	var singleErr *powerTargetError
	for _, target := range targets {
		if targetErr := validateAndLockPowerTarget(r, action, admin, target, resolver); targetErr != nil {
			if !req.IsBatch() {
				singleErr = targetErr
				break
			}
			resp.Results = append(resp.Results, powerResultFromError(r, powerResultFromTarget(target), targetErr))
			continue
		}
		readyTargets = append(readyTargets, target)
	}
	if singleErr != nil {
		writeSinglePowerError(w, r, singleErr)
		return
	}
	if len(readyTargets) == 0 {
		jsonOK(w, resp)
		return
	}

	client, cvmErr := GetCVMClient(r.Context())
	if cvmErr != nil {
		richErr := hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed)
		resp.Results = append(resp.Results, powerResultsFromCloudError(r, readyTargets, richErr)...)
		if !req.IsBatch() {
			writeSinglePowerError(w, r, &powerTargetError{HTTPStatus: http.StatusInternalServerError, Err: richErr})
			return
		}
		jsonOK(w, resp)
		return
	}

	results, cloudErr := executePowerCloudCalls(r, client, action, readyTargets)
	resp.Results = append(resp.Results, results...)
	clearAdjustmentFailuresOnSuccessfulPowerTargets(r.Context(), readyTargets, results)

	// 管理端开机成功后：清除 stale-instances v1.0 全部标记
	// (stale_group / pending_user_action / allow_migrate / allow_same_group_handover)。
	// 语义："admin 主动开机 = 该实例分组归属处理完毕"，用户下次开机不再被任何 stale 标拦。
	// 仅 admin && start 且 CVM 调用成功。
	if admin && action == powerActionStart && cloudErr == nil {
		clearStaleFlagsOnStartedTargets(r.Context(), readyTargets, results)
	}

	if cloudErr != nil && !req.IsBatch() {
		writeSinglePowerError(w, r, &powerTargetError{HTTPStatus: http.StatusInternalServerError, Err: hcommon.EnsureRichErrorOrPanic(cloudErr)})
		return
	}
	if req.IsBatch() {
		jsonOK(w, resp)
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

func writeSinglePowerError(w http.ResponseWriter, r *http.Request, targetErr *powerTargetError) {
	if errors.Is(targetErr.Err, ErrAgentNotAllowed) {
		writeAgentGuardError(w, r, targetErr.Err)
		return
	}
	writeError(w, r, targetErr.HTTPStatus, hcommon.EnsureRichErrorOrPanic(targetErr.Err))
}

func parsePowerTargetRequest(r *http.Request) (powerTargetRequest, error) {
	var body struct {
		IDs         *[]uint   `json:"ids"`
		ID          *uint     `json:"id"`
		InstanceIDs *[]string `json:"instance_ids"`
		InstanceID  *string   `json:"instance_id"`
	}
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	if body.IDs != nil {
		if len(*body.IDs) > powerBatchMaxTargets {
			return powerTargetRequest{}, hcommon.I18nError(i18n.MsgIDsCountExceed, powerBatchMaxTargets)
		}
		seen := make(map[uint]struct{}, len(*body.IDs))
		ids := make([]uint, 0, len(*body.IDs))
		for _, id := range *body.IDs {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return powerTargetRequest{}, hcommon.I18nError(i18n.MsgIDsEmptyList)
		}
		return powerTargetRequest{IDs: ids}, nil
	}
	if body.InstanceIDs != nil {
		if len(*body.InstanceIDs) > powerBatchMaxTargets {
			return powerTargetRequest{}, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, powerBatchMaxTargets)
		}
		seen := make(map[string]struct{}, len(*body.InstanceIDs))
		instanceIDs := make([]string, 0, len(*body.InstanceIDs))
		for _, instanceID := range *body.InstanceIDs {
			instanceID = strings.TrimSpace(instanceID)
			if instanceID == "" {
				continue
			}
			if _, ok := seen[instanceID]; ok {
				continue
			}
			seen[instanceID] = struct{}{}
			instanceIDs = append(instanceIDs, instanceID)
		}
		if len(instanceIDs) == 0 {
			return powerTargetRequest{}, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty)
		}
		return powerTargetRequest{InstanceIDs: instanceIDs}, nil
	}
	if body.ID != nil && *body.ID > 0 {
		return powerTargetRequest{ID: *body.ID}, nil
	}
	if raw := strings.TrimSpace(r.FormValue("id")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			return powerTargetRequest{}, hcommon.I18nError(i18n.MsgInvalidID)
		}
		return powerTargetRequest{ID: uint(id)}, nil
	}
	if body.InstanceID != nil && strings.TrimSpace(*body.InstanceID) != "" {
		return powerTargetRequest{InstanceID: strings.TrimSpace(*body.InstanceID)}, nil
	}
	if raw := strings.TrimSpace(r.FormValue("instance_id")); raw != "" {
		return powerTargetRequest{InstanceID: raw}, nil
	}
	return powerTargetRequest{}, hcommon.I18nError(i18n.MsgMissingIDOrInstanceID)
}

func resolvePowerTargets(r *http.Request, req powerTargetRequest, userID uint) ([]powerTarget, error) {
	if !req.IsBatch() {
		inst, err := findInstanceByIDOrCVMID(r.Context(), userID, req.ID, req.InstanceID)
		if err != nil {
			return nil, err
		}
		return []powerTarget{{InputID: req.ID, InputInstanceID: req.InstanceID, Instance: inst}}, nil
	}

	if len(req.IDs) > 0 {
		return resolvePowerTargetsByIDs(r, req.IDs, userID)
	}
	return resolvePowerTargetsByCVMIDs(r, req.InstanceIDs, userID)
}

func resolvePowerTargetsByIDs(r *http.Request, ids []uint, userID uint) ([]powerTarget, error) {
	q := model.DB(r.Context())
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	var instances []model.Instance
	if err := q.Where("id IN ?", ids).Find(&instances).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed)
	}
	byID := instancesByID(instances)
	targets := make([]powerTarget, 0, len(ids))
	for _, id := range ids {
		target := powerTarget{InputID: id}
		if inst := byID[id]; inst != nil {
			target.Instance = inst
		} else {
			target.Err = ErrInstanceNotFound
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func resolvePowerTargetsByCVMIDs(r *http.Request, instanceIDs []string, userID uint) ([]powerTarget, error) {
	q := model.DB(r.Context())
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	var instances []model.Instance
	if err := q.Where("instance_id IN ?", instanceIDs).Find(&instances).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryInstancesByIDsFailed)
	}
	byCVMID := instancesByCVMID(instances)
	targets := make([]powerTarget, 0, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		target := powerTarget{InputInstanceID: instanceID}
		if inst := byCVMID[instanceID]; inst != nil {
			target.Instance = inst
		} else {
			target.Err = ErrInstanceNotFound
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func instancesByID(instances []model.Instance) map[uint]*model.Instance {
	byID := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		byID[instances[i].ID] = &instances[i]
	}
	return byID
}

func instancesByCVMID(instances []model.Instance) map[string]*model.Instance {
	byCVMID := make(map[string]*model.Instance, len(instances))
	for i := range instances {
		byCVMID[instances[i].InstanceId] = &instances[i]
	}
	return byCVMID
}

func powerResultFromTarget(target powerTarget) powerActionResult {
	if target.Instance == nil {
		return powerActionResult{ID: target.InputID, InstanceID: target.InputInstanceID}
	}
	return powerActionResult{ID: target.Instance.ID, InstanceID: target.Instance.InstanceId, Name: target.Instance.Name}
}

func validateAndLockPowerTarget(r *http.Request, action string, admin bool, target powerTarget, resolver instanceStatusResolver) *powerTargetError {
	if target.Err != nil {
		return &powerTargetError{HTTPStatus: instanceErrStatus(target.Err), Err: target.Err}
	}
	inst := target.Instance
	if inst == nil {
		return &powerTargetError{HTTPStatus: http.StatusNotFound, Err: ErrInstanceNotFound}
	}
	if inst.IsDoctorNode {
		return &powerTargetError{HTTPStatus: http.StatusBadRequest, Err: hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed)}
	}
	if rerr := rejectLocalInstance(inst); rerr != nil {
		// 本地 agent 实例（source=local）不走 CVM/TAT，不支持远程 start/stop/reboot/reset。
		return &powerTargetError{HTTPStatus: http.StatusBadRequest, Err: rerr}
	}
	if inst.InstanceId == "" {
		return &powerTargetError{HTTPStatus: http.StatusBadRequest, Err: hcommon.I18nError(i18n.MsgInstanceNoCVM)}
	}

	// 用户端开机：带 stale-instances v1.0 标的实例禁止直接开机，管理端不受限。
	//   - pending_user_action：实例等待 owner 自处理，引导用户先在弹窗完成迁移或移交
	//   - stale_group：分组归属异常，需管理员先处理（管理端开机成功后自动清此标）
	// 两个标的检查独立；命中任一即拒绝，返回具体消息帮助前端定位。
	if !admin && action == powerActionStart {
		if ok, _ := model.HasInstanceFlag(r.Context(), inst.ID, model.InstanceFlagPendingUserAction); ok {
			return &powerTargetError{HTTPStatus: http.StatusConflict, Err: hcommon.I18nError(i18n.MsgInstancePendingUserAction)}
		}
		if ok, _ := model.HasInstanceFlag(r.Context(), inst.ID, model.InstanceFlagStaleGroup); ok {
			return &powerTargetError{HTTPStatus: http.StatusConflict, Err: hcommon.I18nError(i18n.MsgInstanceStaleGroup)}
		}
	}

	var status InstanceStatusResponse
	var err error
	if admin {
		status, err = requireActionAllowedForAdmin(r.Context(), inst, action, resolver)
	} else {
		status, err = requireActionAllowedForUser(r.Context(), inst, action, resolver)
	}
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrAgentNotAllowed) || errors.Is(err, ErrOperationInProgress) {
			statusCode = http.StatusConflict
		}
		return &powerTargetError{HTTPStatus: statusCode, Status: status, Err: hcommon.EnsureRichErrorOrPanic(err)}
	}

	if action == powerActionStart {
		err = setOperationWithAgentReset(model.DB(r.Context()), inst, model.OpReboot)
	} else {
		err = setOperation(model.DB(r.Context()), inst, model.OpReboot)
	}
	if err != nil {
		return &powerTargetError{HTTPStatus: http.StatusConflict, Status: status, Err: hcommon.I18nRichError(err, i18n.MsgOperationConflict)}
	}
	return nil
}

func powerResultFromError(r *http.Request, result powerActionResult, targetErr *powerTargetError) powerActionResult {
	if errors.Is(targetErr.Err, ErrAgentNotAllowed) {
		result.Status = "skipped"
		key, args := agentStatusRejectMessage(targetErr.Status)
		result.Message = i18n.T(r.Context(), key, args...)
		return result
	}
	result.Status = "failed"
	result.Message = hcommon.ErrorMessageWithCtx(r.Context(), targetErr.Err)
	return result
}

func powerResultsFromCloudError(r *http.Request, targets []powerTarget, err error) []powerActionResult {
	results := make([]powerActionResult, 0, len(targets))
	for _, target := range targets {
		result := powerResultFromTarget(target)
		result.Status = "failed"
		result.Message = hcommon.ErrorMessageWithCtx(r.Context(), err)
		if target.Instance != nil {
			clearOperation(model.DB(r.Context()), target.Instance, model.OpStateFailed)
		}
		results = append(results, result)
	}
	return results
}

func executePowerCloudCalls(r *http.Request, client *cvm.Client, action string, targets []powerTarget) ([]powerActionResult, error) {
	results := make([]powerActionResult, 0, len(targets))
	var firstErr error
	for start := 0; start < len(targets); start += cvmPowerBatchSize {
		end := start + cvmPowerBatchSize
		if end > len(targets) {
			end = len(targets)
		}
		chunk := targets[start:end]
		chunkIDs := make([]string, 0, len(chunk))
		for _, target := range chunk {
			chunkIDs = append(chunkIDs, target.Instance.InstanceId)
		}

		err := callPowerInstances(client, action, chunkIDs)
		var richErr error
		if err != nil {
			richErr = hcommon.I18nRichError(err, powerFailureKey(action))
			if firstErr == nil {
				firstErr = richErr
			}
		}
		for _, target := range chunk {
			result := powerResultFromTarget(target)
			if richErr != nil {
				result.Status = "failed"
				result.Message = hcommon.ErrorMessageWithCtx(r.Context(), richErr)
				if target.Instance != nil {
					clearOperation(model.DB(r.Context()), target.Instance, model.OpStateFailed)
				}
			} else {
				result.Status = "started"
				if action == powerActionStop {
					result.Message = i18n.T(r.Context(), i18n.MsgShutdownStarted)
				} else {
					result.Message = i18n.T(r.Context(), i18n.MsgPowerOnStarted)
				}
			}
			results = append(results, result)
		}
	}
	return results, firstErr
}

func callPowerInstances(client *cvm.Client, action string, instanceIDs []string) error {
	if action == powerActionStart {
		return callStartInstances(client, instanceIDs)
	}
	return callStopInstances(client, instanceIDs)
}

func powerFailureKey(action string) i18n.Key {
	if action == powerActionStop {
		return i18n.MsgShutdownFailed
	}
	return i18n.MsgPowerOnFailed
}

func callStartInstances(client *cvm.Client, instanceIDs []string) error {
	req := cvm.NewStartInstancesRequest()
	req.InstanceIds = sdkcommon.StringPtrs(instanceIDs)
	_, err := client.StartInstances(req)
	return err
}

func callStopInstances(client *cvm.Client, instanceIDs []string) error {
	req := cvm.NewStopInstancesRequest()
	req.InstanceIds = sdkcommon.StringPtrs(instanceIDs)
	req.StopType = sdkcommon.StringPtr("SOFT_FIRST")
	// 腾讯云文档说明 StoppedMode 仅对支持关机不收费的按量计费实例生效；
	// 包年包月等不适用实例会保持普通关机语义。
	req.StoppedMode = sdkcommon.StringPtr("STOP_CHARGING")
	_, err := client.StopInstances(req)
	return err
}

func clearAdjustmentFailuresOnSuccessfulPowerTargets(ctx context.Context, targets []powerTarget, results []powerActionResult) {
	if len(targets) != len(results) {
		return
	}
	instanceIDs := make([]uint, 0, len(results))
	for i, result := range results {
		if result.Status == "started" && targets[i].Instance != nil {
			instanceIDs = append(instanceIDs, targets[i].Instance.ID)
		}
	}
	clearAdjustmentFailures(ctx, instanceIDs...)
}

// clearStaleFlagsOnStartedTargets 管理端开机成功后清 stale-instances v1.0 全部标记：
// stale_group / pending_user_action / allow_migrate / allow_same_group_handover。
//
// 语义："admin 主动开机 = 该实例分组归属处理完毕"，用户下次开机不再被任何 stale 标拦。
// 与 stale_instances_apply.go 里的 clearStaleFlagsTx 保持一致（apply 完成路径也是清全部）。
//
// results 与 targets 一一对应；只清 CVM 调用返回 status="started" 的目标。
// 单条清标失败仅日志告警，不影响开机响应。
func clearStaleFlagsOnStartedTargets(ctx context.Context, targets []powerTarget, results []powerActionResult) {
	if len(targets) == 0 || len(targets) != len(results) {
		return
	}
	for i, target := range targets {
		if results[i].Status != "started" || target.Instance == nil {
			continue
		}
		err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
			return clearStaleFlagsTx(tx, target.Instance.ID)
		})
		if err != nil {
			slog.Warn("[AdminStart] clear stale flags failed",
				"instance_id", target.Instance.ID, "cvm_id", target.Instance.InstanceId, "err", err)
		}
	}
}
