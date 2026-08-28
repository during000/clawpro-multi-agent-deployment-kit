package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

const (
	instanceAdjustmentWorkerConcurrency = 5
	instanceAdjustmentTimeout           = 15 * time.Minute
	instanceAdjustmentCrashObservations = 3
	instanceAdjustmentCrashInterval     = 5 * time.Second
)

type adjustmentWorkItem struct {
	Instance *model.Instance
	Task     *model.InstanceAdjustment
	Payload  model.InstanceAdjustmentPayload
}

func (i *adjustmentWorkItem) executionStartedAt() time.Time {
	if i != nil && i.Task != nil && i.Task.ExecutionStartedAt != nil {
		return *i.Task.ExecutionStartedAt
	}
	if i != nil && i.Task != nil {
		return i.Task.CreatedAt
	}
	return time.Time{}
}

type adjustmentTaskUpdate struct {
	phase     *string
	requestID *string
	runAt     *time.Time
	attempt   *int
	errorCode *string
}

func adjustmentUpdateValue[T any](value T) *T {
	return &value
}

// RunInstanceAdjustmentWorkerOnce advances at most five persisted adjustment
// tasks for the tenant carried by ctx.
func RunInstanceAdjustmentWorkerOnce(ctx context.Context) {
	runInstanceAdjustmentWorkerOnce(ctx, defaultAdjustmentCloudFactory)
}

func runInstanceAdjustmentWorkerOnce(ctx context.Context, factory adjustmentCloudFactory) {
	db := model.DB(ctx)
	activePhases := []string{
		adjustmentPhaseSubmitting,
		adjustmentPhasePolling,
		adjustmentPhaseRestoreSuccess,
		adjustmentPhaseRestoreFailure,
	}
	var activeCount int64
	if err := db.Model(&model.InstanceAdjustment{}).
		Where("status = ? AND phase IN ?", adjustmentStatusProcessing, activePhases).
		Count(&activeCount).Error; err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 统计活动任务失败", "error", err)
		return
	}

	fresh := make(map[uint]struct{})
	slots := instanceAdjustmentWorkerConcurrency - int(activeCount)
	if slots > 0 {
		var queued []model.InstanceAdjustment
		if err := db.Where("status = ? AND phase = ?", adjustmentStatusProcessing, adjustmentPhaseQueued).
			Order("id ASC").Limit(slots).Find(&queued).Error; err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 查询排队任务失败", "error", err)
			return
		}
		now := time.Now()
		for i := range queued {
			updated := db.Model(&model.InstanceAdjustment{}).
				Where("id = ? AND status = ? AND phase = ?", queued[i].ID, adjustmentStatusProcessing, adjustmentPhaseQueued).
				Updates(map[string]any{
					"phase":                adjustmentPhaseSubmitting,
					"run_at":               now,
					"attempt":              0,
					"execution_started_at": now,
				})
			if updated.Error != nil {
				slog.ErrorContext(ctx, "[InstanceAdjustment] 领取任务失败", "id", queued[i].ID, "error", updated.Error)
				continue
			}
			if updated.RowsAffected == 1 {
				fresh[queued[i].ID] = struct{}{}
			}
		}
	}

	now := time.Now()
	var due []model.InstanceAdjustment
	if err := db.Where("status = ? AND phase IN ? AND run_at <= ?", adjustmentStatusProcessing, activePhases, now).
		Order("id ASC").Limit(instanceAdjustmentWorkerConcurrency).Find(&due).Error; err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 查询到期任务失败", "error", err)
		return
	}
	if len(due) == 0 {
		return
	}
	instanceIDs := make([]uint, 0, len(due))
	for i := range due {
		instanceIDs = append(instanceIDs, due[i].InstanceID)
	}
	var instances []model.Instance
	if err := db.Where("id IN ?", instanceIDs).Find(&instances).Error; err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 查询任务实例失败", "error", err)
		return
	}
	instancesByID := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		instancesByID[instances[i].ID] = &instances[i]
	}

	gateway, err := factory(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 创建云客户端失败，等待下轮", "error", err)
		return
	}

	var wg sync.WaitGroup
	for i := range due {
		task := &due[i]
		instance := instancesByID[task.InstanceID]
		if instance == nil {
			now := time.Now()
			if err := db.Model(task).Updates(map[string]any{
				"status": adjustmentStatusFailed, "error_code": reasonInstanceNotFound, "finished_at": now,
			}).Error; err != nil {
				slog.ErrorContext(ctx, "[InstanceAdjustment] 标记孤立任务失败", "task_id", task.ID, "error", err)
			}
			continue
		}
		payload, payloadErr := task.Payload()
		if payloadErr != nil {
			item := &adjustmentWorkItem{Instance: instance, Task: task}
			finishAdjustmentFailure(ctx, item, nil, reasonCloudAdjustmentFailed)
			continue
		}
		item := &adjustmentWorkItem{
			Instance: instance,
			Task:     task,
			Payload:  payload,
		}
		_, isFresh := fresh[task.ID]
		wg.Add(1)
		go func() {
			defer wg.Done()
			advanceInstanceAdjustment(ctx, gateway, item, isFresh)
		}()
	}
	wg.Wait()
}

func advanceInstanceAdjustment(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem, fresh bool) {
	startedAt := item.executionStartedAt()
	if startedAt.IsZero() {
		finishAdjustmentFailure(ctx, item, nil, reasonCloudAdjustmentFailed)
		return
	}
	if time.Since(startedAt) >= instanceAdjustmentTimeout {
		if item.Task.Phase == adjustmentPhaseRestoreSuccess || item.Task.Phase == adjustmentPhaseRestoreFailure {
			cloud, err := describeOneAdjustmentInstance(ctx, gateway, item.Instance.InstanceId)
			if err != nil {
				slog.WarnContext(ctx, "[InstanceAdjustment] 恢复阶段达到总超时且云状态读取失败", "id", item.Instance.ID, "error", err)
			}
			finishAdjustmentFailure(ctx, item, cloud, reasonAdjustmentRestoreFailed)
			return
		}
		item.Task.Phase = adjustmentPhaseRestoreFailure
		item.Task.ErrorCode = reasonAdjustmentTimeout
		if err := persistAdjustmentPhase(ctx, item, 0); err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 写入超时阶段失败", "id", item.Instance.ID, "error", err)
			return
		}
	}

	switch item.Task.Phase {
	case adjustmentPhaseSubmitting:
		if fresh {
			submitFreshAdjustment(ctx, gateway, item)
		} else {
			reconcileAmbiguousSubmission(ctx, gateway, item)
		}
	case adjustmentPhasePolling:
		pollInstanceAdjustment(ctx, gateway, item)
	case adjustmentPhaseRestoreSuccess, adjustmentPhaseRestoreFailure:
		restoreInstanceState(ctx, gateway, item)
	}
}

func adjustmentRequestFromWorkItem(item *adjustmentWorkItem) instanceAdjustmentRequest {
	req := instanceAdjustmentRequest{AdjustmentType: item.Task.Type, ResizeMode: item.Payload.ResizeMode}
	if item.Task.Type == adjustmentTypeInstanceType {
		target := item.Payload.TargetInstanceType
		req.TargetInstanceType = &target
	} else {
		target := item.Payload.TargetDiskSize
		req.TargetSystemDiskSize = &target
	}
	return req
}

func jitValidateInstanceAdjustment(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem) (*instanceAdjustmentResult, error) {
	req := adjustmentRequestFromWorkItem(item)
	targets := []resolvedAdjustmentTarget{{DBID: item.Instance.ID, InstanceID: item.Instance.InstanceId, Instance: item.Instance, Adjustment: item.Task}}
	results, err := validateResolvedAdjustmentTargets(ctx, req, targets, gateway, true)
	if err != nil {
		return nil, err
	}
	if len(results) != 1 {
		return nil, errors.New("unexpected empty JIT result")
	}
	return &results[0], nil
}

func submitFreshAdjustment(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem) {
	result, err := jitValidateInstanceAdjustment(ctx, gateway, item)
	if err != nil {
		scheduleAdjustmentReadRetry(ctx, item, err)
		return
	}
	if !result.Adjustable {
		finishAdjustmentFailure(ctx, item, result.cloud, result.ReasonCode)
		return
	}
	requestID, executeErr := gateway.Execute(ctx, result.operation)
	if executeErr != nil {
		if isRetryableError(executeErr) {
			next := time.Now().Add(instanceAdjustmentCrashInterval)
			if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
				attempt: adjustmentUpdateValue(0),
				runAt:   &next,
			}); err != nil {
				slog.ErrorContext(ctx, "[InstanceAdjustment] 写入提交观察窗口失败", "id", item.Instance.ID, "error", err)
			}
			return
		}
		finishAdjustmentFailure(ctx, item, result.cloud, mapAdjustmentExecutionError(result.operation, executeErr))
		return
	}
	now := time.Now()
	next := now.Add(time.Second)
	if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
		requestID: &requestID,
		phase:     adjustmentUpdateValue(adjustmentPhasePolling),
		attempt:   adjustmentUpdateValue(0),
		runAt:     &next,
	}); err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] RequestId 落库失败，进入保守恢复窗口", "id", item.Instance.ID, "request_id", requestID, "error", err)
	}
}

func reconcileAmbiguousSubmission(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem) {
	cloud, err := describeOneAdjustmentInstance(ctx, gateway, item.Instance.InstanceId)
	if err != nil {
		scheduleAmbiguousSubmissionReadRetry(ctx, item, err)
		return
	}
	if cloud == nil {
		finishAdjustmentFailure(ctx, item, nil, reasonCVMInstanceNotFound)
		return
	}
	if adjustmentTargetReached(item, cloud) {
		transitionAdjustmentToRestore(ctx, item, adjustmentPhaseRestoreSuccess, "")
		return
	}
	if relevantLatestOperation(item, cloud) && cloud.LatestOperationState == "OPERATING" {
		now := time.Now()
		next := now.Add(time.Second)
		requestID := cloud.LatestOperationRequestID
		if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
			requestID: &requestID,
			phase:     adjustmentUpdateValue(adjustmentPhasePolling),
			attempt:   adjustmentUpdateValue(0),
			runAt:     &next,
		}); err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 写入已观测云操作失败", "id", item.Instance.ID, "request_id", requestID, "error", err)
		}
		return
	}
	if relevantLatestOperation(item, cloud) && cloud.LatestOperationState == "FAILED" {
		reason := mapAdjustmentCloudError(errors.New(cloud.LatestOperationErrorMessage))
		transitionAdjustmentToRestore(ctx, item, adjustmentPhaseRestoreFailure, reason)
		return
	}

	now := time.Now()
	if !item.Task.UpdatedAt.IsZero() && now.Sub(item.Task.UpdatedAt) < instanceAdjustmentCrashInterval {
		next := item.Task.UpdatedAt.Add(instanceAdjustmentCrashInterval)
		if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{runAt: &next}); err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 延后提交观察失败", "id", item.Instance.ID, "error", err)
		}
		return
	}
	count := item.Task.Attempt + 1
	if count < instanceAdjustmentCrashObservations {
		next := now.Add(instanceAdjustmentCrashInterval)
		if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
			attempt: &count,
			runAt:   &next,
		}); err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 写入提交观察次数失败", "id", item.Instance.ID, "error", err)
		}
		return
	}
	item.Task.Attempt = count
	submitFreshAdjustment(ctx, gateway, item)
}

func pollInstanceAdjustment(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem) {
	cloud, err := describeOneAdjustmentInstance(ctx, gateway, item.Instance.InstanceId)
	if err != nil {
		scheduleAdjustmentReadRetry(ctx, item, err)
		return
	}
	if cloud == nil {
		transitionAdjustmentToRestore(ctx, item, adjustmentPhaseRestoreFailure, reasonCVMInstanceNotFound)
		return
	}
	if adjustmentTargetReached(item, cloud) && (cloud.LatestOperationState == "SUCCESS" || cloud.LatestOperationState == "") {
		transitionAdjustmentToRestore(ctx, item, adjustmentPhaseRestoreSuccess, "")
		return
	}
	if item.Task.RequestID != "" && cloud.LatestOperationRequestID != "" && cloud.LatestOperationRequestID != item.Task.RequestID {
		scheduleNextAdjustmentPoll(ctx, item, cloud, time.Second)
		return
	}
	if relevantLatestOperation(item, cloud) && cloud.LatestOperationState == "FAILED" {
		reason := mapAdjustmentCloudError(errors.New(cloud.LatestOperationErrorMessage))
		transitionAdjustmentToRestore(ctx, item, adjustmentPhaseRestoreFailure, reason)
		return
	}
	scheduleNextAdjustmentPoll(ctx, item, cloud, time.Second)
}

func transitionAdjustmentToRestore(ctx context.Context, item *adjustmentWorkItem, phase, reason string) {
	now := time.Now()
	update := adjustmentTaskUpdate{
		phase:   &phase,
		runAt:   &now,
		attempt: adjustmentUpdateValue(0),
	}
	if reason != "" {
		update.errorCode = &reason
	}
	if err := updateProcessingAdjustment(ctx, item, update); err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 写入恢复阶段失败", "id", item.Instance.ID, "phase", phase, "error", err)
	}
}

func restoreInstanceState(ctx context.Context, gateway instanceAdjustmentCloudGateway, item *adjustmentWorkItem) {
	cloud, err := describeOneAdjustmentInstance(ctx, gateway, item.Instance.InstanceId)
	if err != nil {
		scheduleAdjustmentReadRetry(ctx, item, err)
		return
	}
	if cloud == nil {
		reason := item.Task.ErrorCode
		if reason == "" {
			reason = reasonAdjustmentRestoreFailed
		}
		finishAdjustmentFailure(ctx, item, nil, reason)
		return
	}
	desiredState := item.Payload.OriginalCVMState
	switch desiredState {
	case "RUNNING":
		if cloud.State == "RUNNING" {
			finishAdjustmentRestore(ctx, item, cloud)
			return
		}
		if cloud.State == "STOPPED" {
			if err := gateway.StartInstance(ctx, item.Instance.InstanceId); err != nil {
				finishAdjustmentFailure(ctx, item, cloud, reasonAdjustmentRestoreFailed)
				return
			}
			scheduleNextAdjustmentPoll(ctx, item, cloud, 5*time.Second)
			return
		}
	case "STOPPED":
		if cloud.State == "STOPPED" {
			finishAdjustmentRestore(ctx, item, cloud)
			return
		}
		if cloud.State == "RUNNING" {
			if err := gateway.StopInstance(ctx, item.Instance.InstanceId, item.Payload.OriginalStopChargingMode); err != nil {
				finishAdjustmentFailure(ctx, item, cloud, reasonAdjustmentRestoreFailed)
				return
			}
			scheduleNextAdjustmentPoll(ctx, item, cloud, 5*time.Second)
			return
		}
	default:
		finishAdjustmentFailure(ctx, item, cloud, reasonAdjustmentRestoreFailed)
		return
	}
	if model.CVMTransientStates[cloud.State] || cloud.LatestOperationState == "OPERATING" {
		scheduleNextAdjustmentPoll(ctx, item, cloud, 2*time.Second)
		return
	}
	finishAdjustmentFailure(ctx, item, cloud, reasonAdjustmentRestoreFailed)
}

func finishAdjustmentRestore(ctx context.Context, item *adjustmentWorkItem, cloud *adjustmentCloudInstance) {
	if item.Task.Phase == adjustmentPhaseRestoreSuccess {
		finishAdjustmentSuccess(ctx, item, cloud)
		return
	}
	reason := item.Task.ErrorCode
	if reason == "" {
		reason = reasonCloudAdjustmentFailed
	}
	finishAdjustmentFailure(ctx, item, cloud, reason)
}

func finishAdjustmentSuccess(ctx context.Context, item *adjustmentWorkItem, cloud *adjustmentCloudInstance) {
	now := time.Now()
	updates := terminalInstanceClearUpdates()
	updates["last_known_status"] = strings.ToLower(item.Payload.OriginalCVMState)
	updates["status_synced_at"] = now
	if cloud != nil {
		updates["cvm_instance_type"] = cloud.InstanceType
		updates["cvm_cpu"] = cloud.CPU
		updates["cvm_memory_gb"] = cloud.MemoryGB
		updates["system_disk_type"] = cloud.SystemDiskType
		updates["system_disk_size"] = cloud.SystemDiskSize
		updates["last_cvm_state"] = cloud.State
	}
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Instance{}).
			Where("id = ? AND current_operation IN ?", item.Instance.ID, []string{model.OpAdjustInstanceType, model.OpAdjustSystemDisk}).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("adjustment instance CAS lost for %d", item.Instance.ID)
		}
		return tx.Where("id = ? AND status = ?", item.Task.ID, adjustmentStatusProcessing).
			Delete(&model.InstanceAdjustment{}).Error
	})
	if err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 成功终态落库失败", "id", item.Instance.ID, "error", err)
	}
}

func finishAdjustmentFailure(ctx context.Context, item *adjustmentWorkItem, cloud *adjustmentCloudInstance, reason string) {
	if reason == "" {
		reason = reasonCloudAdjustmentFailed
	}
	now := time.Now()
	updates := terminalInstanceClearUpdates()
	if cloud == nil && reason == reasonCVMInstanceNotFound {
		updates["last_cvm_state"] = "NOTFOUND"
		updates["last_known_status"] = model.StatusDestroyed
		updates["status_synced_at"] = now
	}
	if cloud != nil {
		updates["cvm_instance_type"] = cloud.InstanceType
		updates["cvm_cpu"] = cloud.CPU
		updates["cvm_memory_gb"] = cloud.MemoryGB
		updates["system_disk_type"] = cloud.SystemDiskType
		updates["system_disk_size"] = cloud.SystemDiskSize
		updates["last_cvm_state"] = cloud.State
		updates["status_synced_at"] = now
		if cloud.State == "RUNNING" || cloud.State == "STOPPED" {
			updates["last_known_status"] = strings.ToLower(cloud.State)
		}
	}
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Instance{}).
			Where("id = ? AND current_operation IN ?", item.Instance.ID, []string{model.OpAdjustInstanceType, model.OpAdjustSystemDisk}).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("adjustment instance CAS lost for %d", item.Instance.ID)
		}
		taskResult := tx.Model(&model.InstanceAdjustment{}).
			Where("id = ? AND status = ?", item.Task.ID, adjustmentStatusProcessing).
			Updates(map[string]any{
				"status":      adjustmentStatusFailed,
				"error_code":  reason,
				"finished_at": now,
				"run_at":      now,
			})
		return taskResult.Error
	})
	if err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 失败终态落库失败", "id", item.Instance.ID, "reason", reason, "error", err)
	}
}

func terminalInstanceClearUpdates() map[string]any {
	return map[string]any{
		"current_operation":            model.OpNone,
		"current_operation_state":      model.OpStateNone,
		"current_operation_updated_at": nil,
	}
}

func adjustmentTargetReached(item *adjustmentWorkItem, cloud *adjustmentCloudInstance) bool {
	if item.Task.Type == adjustmentTypeInstanceType {
		return cloud.InstanceType == item.Payload.TargetInstanceType
	}
	return cloud.SystemDiskSize >= item.Payload.TargetDiskSize
}

func relevantLatestOperation(item *adjustmentWorkItem, cloud *adjustmentCloudInstance) bool {
	if item.Task.Type == adjustmentTypeInstanceType {
		return strings.Contains(cloud.LatestOperation, "ResetInstancesType")
	}
	return strings.Contains(cloud.LatestOperation, "ResizeInstanceDisks")
}

func describeOneAdjustmentInstance(ctx context.Context, gateway instanceAdjustmentCloudGateway, instanceID string) (*adjustmentCloudInstance, error) {
	instances, err := gateway.DescribeInstances(ctx, []string{instanceID})
	if err != nil {
		return nil, err
	}
	return instances[instanceID], nil
}

func persistAdjustmentPhase(ctx context.Context, item *adjustmentWorkItem, delay time.Duration) error {
	runAt := time.Now().Add(delay)
	return updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
		phase:     &item.Task.Phase,
		errorCode: &item.Task.ErrorCode,
		runAt:     &runAt,
	})
}

func scheduleNextAdjustmentPoll(ctx context.Context, item *adjustmentWorkItem, cloud *adjustmentCloudInstance, delay time.Duration) {
	now := time.Now()
	runAt := now.Add(delay)
	if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
		attempt: adjustmentUpdateValue(0),
		runAt:   &runAt,
	}); err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 安排下次轮询失败", "id", item.Instance.ID, "error", err)
		return
	}
	if cloud == nil {
		return
	}
	updates := map[string]any{"last_cvm_state": cloud.State}
	if cloud.State == "RUNNING" || cloud.State == "STOPPED" {
		updates["last_known_status"] = strings.ToLower(cloud.State)
		updates["status_synced_at"] = now
	}
	if err := model.DB(ctx).Model(&model.Instance{}).Where("id = ?", item.Instance.ID).Updates(updates).Error; err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 写回调整期间实例状态失败", "id", item.Instance.ID, "error", err)
	}
}

func scheduleAmbiguousSubmissionReadRetry(ctx context.Context, item *adjustmentWorkItem, readErr error) {
	runAt := time.Now().Add(time.Second)
	if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
		attempt: &item.Task.Attempt,
		runAt:   &runAt,
	}); err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 安排提交观察重试失败", "id", item.Instance.ID, "error", err)
	}
	slog.WarnContext(ctx, "[InstanceAdjustment] 提交观察读取失败，等待重试", "id", item.Instance.ID, "error", readErr)
}

func scheduleAdjustmentReadRetry(ctx context.Context, item *adjustmentWorkItem, readErr error) {
	attempt := item.Task.Attempt + 1
	delay := time.Second << min(attempt-1, 3)
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	runAt := time.Now().Add(delay)
	if err := updateProcessingAdjustment(ctx, item, adjustmentTaskUpdate{
		attempt: &attempt,
		runAt:   &runAt,
	}); err != nil {
		slog.ErrorContext(ctx, "[InstanceAdjustment] 读取退避落库失败", "id", item.Instance.ID, "error", err)
	}
	slog.WarnContext(ctx, "[InstanceAdjustment] 云状态读取失败，等待重试", "id", item.Instance.ID, "delay", delay, "error", readErr)
}

func updateProcessingAdjustment(ctx context.Context, item *adjustmentWorkItem, update adjustmentTaskUpdate) error {
	if item == nil || item.Task == nil {
		return errors.New("missing adjustment task")
	}
	updates := make(map[string]any, 5)
	if update.phase != nil {
		updates["phase"] = *update.phase
	}
	if update.requestID != nil {
		updates["request_id"] = *update.requestID
	}
	if update.runAt != nil {
		updates["run_at"] = *update.runAt
	}
	if update.attempt != nil {
		updates["attempt"] = *update.attempt
	}
	if update.errorCode != nil {
		updates["error_code"] = *update.errorCode
	}
	if len(updates) == 0 {
		return nil
	}
	result := model.DB(ctx).Model(&model.InstanceAdjustment{}).
		Where("id = ? AND status = ?", item.Task.ID, adjustmentStatusProcessing).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("adjustment CAS lost for task %d", item.Task.ID)
	}
	if update.phase != nil {
		item.Task.Phase = *update.phase
	}
	if update.requestID != nil {
		item.Task.RequestID = *update.requestID
	}
	if update.runAt != nil {
		item.Task.RunAt = *update.runAt
	}
	if update.attempt != nil {
		item.Task.Attempt = *update.attempt
	}
	if update.errorCode != nil {
		item.Task.ErrorCode = *update.errorCode
	}
	item.Task.UpdatedAt = time.Now()
	return nil
}
