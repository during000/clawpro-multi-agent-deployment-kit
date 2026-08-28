package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

func seedAdjustmentWorkerInstance(t *testing.T, instanceID, adjustmentType, phase string) adjustmentWorkItem {
	t.Helper()
	now := time.Now().Add(-time.Second)
	instance := createAdjustmentTestInstance(t, instanceID)
	instance.CurrentOperation = model.OpAdjustInstanceType
	if adjustmentType == adjustmentTypeSystemDisk {
		instance.CurrentOperation = model.OpAdjustSystemDisk
	}
	instance.CurrentOperationState = model.OpStateProcessing
	instance.CurrentOperationUpdatedAt = &now
	if err := model.DB(context.Background()).Save(&instance).Error; err != nil {
		t.Fatalf("save worker instance: %v", err)
	}

	payload := model.InstanceAdjustmentPayload{
		OriginalCVMState:         "RUNNING",
		OriginalStopChargingMode: "NOT_APPLICABLE",
	}
	if adjustmentType == adjustmentTypeInstanceType {
		payload.TargetInstanceType = "Ai2.LARGE8"
	} else {
		payload.TargetDiskSize = 60
		payload.ResizeMode = adjustmentResizeOnline
	}
	task := model.InstanceAdjustment{
		CreatedAt:  now,
		UpdatedAt:  now,
		Identifier: instance.Identifier,
		InstanceID: instance.ID,
		Status:     adjustmentStatusProcessing,
		Type:       adjustmentType,
		Phase:      phase,
		RunAt:      now,
	}
	if err := task.SetPayload(payload); err != nil {
		t.Fatalf("encode worker payload: %v", err)
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create worker task: %v", err)
	}
	return adjustmentWorkItem{
		Instance: &instance,
		Task:     &task,
		Payload:  payload,
	}
}

func saveAdjustmentWorkerItem(t *testing.T, item *adjustmentWorkItem) {
	t.Helper()
	if err := model.DB(context.Background()).Save(item.Instance).Error; err != nil {
		t.Fatalf("save worker instance: %v", err)
	}
	if item.Task == nil {
		return
	}
	if err := item.Task.SetPayload(item.Payload); err != nil {
		t.Fatalf("encode worker payload: %v", err)
	}
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).
		Where("id = ?", item.Task.ID).
		UpdateColumns(map[string]any{
			"created_at":      item.Task.CreatedAt,
			"updated_at":      item.Task.UpdatedAt,
			"status":          item.Task.Status,
			"adjustment_type": item.Task.Type,
			"phase":           item.Task.Phase,
			"payload_json":    item.Task.PayloadJSON,
			"request_id":      item.Task.RequestID,
			"run_at":          item.Task.RunAt,
			"attempt":         item.Task.Attempt,
			"error_code":      item.Task.ErrorCode,
		}).Error; err != nil {
		t.Fatalf("save worker task: %v", err)
	}
}

func TestInstanceAdjustmentWorker_QueuedJITThenExecute(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-submit", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	cloud := newHappyAdjustmentCloud(item.Instance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }

	runInstanceAdjustmentWorkerOnce(context.Background(), factory)

	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Phase != adjustmentPhasePolling || stored.Task.RequestID != cloud.requestID {
		t.Fatalf("stored phase=%s requestID=%s", stored.Task.Phase, stored.Task.RequestID)
	}
	if len(cloud.inquiries) != 1 || len(cloud.executions) != 1 {
		t.Fatalf("inquiries=%+v executions=%+v", cloud.inquiries, cloud.executions)
	}
	if cloud.inquiries[0] != cloud.executions[0] {
		t.Fatalf("inquiry/write operation mismatch: inquiry=%+v write=%+v", cloud.inquiries[0], cloud.executions[0])
	}
}

func TestInstanceAdjustmentWorker_JITFailureSkipsWrite(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-jit-fail", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	cloud := newHappyAdjustmentCloud(item.Instance.InstanceId)
	cloud.instances[item.Instance.InstanceId].StopChargingMode = "STOP_CHARGING"
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }

	runInstanceAdjustmentWorkerOnce(context.Background(), factory)

	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonStopChargingNotSupported {
		t.Fatalf("stored status=%s reason=%s", stored.Task.Status, stored.Task.ErrorCode)
	}
	if len(cloud.executions) != 0 {
		t.Fatalf("write API called after failed JIT: %+v", cloud.executions)
	}
}

func TestInstanceAdjustmentWorker_PollSuccessRestoresAndClears(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-success", adjustmentTypeInstanceType, adjustmentPhasePolling)
	item.Task.RequestID = "req-adjustment"
	saveAdjustmentWorkerItem(t, &item)
	cloud := newHappyAdjustmentCloud(item.Instance.InstanceId)
	cloud.instances[item.Instance.InstanceId].InstanceType = "Ai2.LARGE8"
	cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
	cloud.instances[item.Instance.InstanceId].LatestOperationState = "SUCCESS"
	cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = "req-adjustment"
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }

	runInstanceAdjustmentWorkerOnce(context.Background(), factory)
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).
		Where("instance_id = ?", item.Instance.ID).
		Update("run_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatalf("make restore due: %v", err)
	}
	runInstanceAdjustmentWorkerOnce(context.Background(), factory)

	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.CurrentOperation != model.OpNone || stored.Task.Status != "" || stored.Instance.CVMInstanceType != "Ai2.LARGE8" {
		t.Fatalf("terminal state not cleared: current=%s adjustment=%s type=%s", stored.Instance.CurrentOperation, stored.Task.Status, stored.Instance.CVMInstanceType)
	}
}

func TestReconcileAmbiguousSubmission_RequiresThreeSpacedObservations(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-ambiguous", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
	cloud := newHappyAdjustmentCloud(item.Instance.InstanceId)

	for observation := 1; observation <= 3; observation++ {
		old := time.Now().Add(-instanceAdjustmentCrashInterval - time.Second)
		if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).
			Where("instance_id = ?", item.Instance.ID).
			Updates(map[string]any{"updated_at": old, "run_at": old}).Error; err != nil {
			t.Fatalf("age observation: %v", err)
		}
		item = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		reconcileAmbiguousSubmission(context.Background(), cloud, &item)
		if observation < 3 && len(cloud.executions) != 0 {
			t.Fatalf("write replayed after observation %d", observation)
		}
	}
	if len(cloud.executions) != 1 {
		t.Fatalf("expected one write after three observations, got %d", len(cloud.executions))
	}
}

type workerContractCloud struct {
	*fakeAdjustmentCloud
	describeErr error
	startErr    error
	stopErr     error
	stopModes   []string
}

func (f *workerContractCloud) DescribeInstances(ctx context.Context, ids []string) (map[string]*adjustmentCloudInstance, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	return f.fakeAdjustmentCloud.DescribeInstances(ctx, ids)
}

func (f *workerContractCloud) StartInstance(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCalls++
	return f.startErr
}

func (f *workerContractCloud) StopInstance(_ context.Context, _ string, mode string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	f.stopModes = append(f.stopModes, mode)
	return f.stopErr
}

func newWorkerContractCloud(instanceID string) *workerContractCloud {
	return &workerContractCloud{fakeAdjustmentCloud: newHappyAdjustmentCloud(instanceID)}
}

func reloadAdjustmentWorkerInstance(t *testing.T, id uint) adjustmentWorkItem {
	t.Helper()
	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance, id).Error; err != nil {
		t.Fatalf("reload instance %d: %v", id, err)
	}
	var task model.InstanceAdjustment
	err := model.DB(context.Background()).Where("instance_id = ?", id).Take(&task).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("reload adjustment task %d: %v", id, err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		task.InstanceID = id
		return adjustmentWorkItem{Instance: &instance, Task: &task}
	}
	payload, err := task.Payload()
	if err != nil {
		t.Fatalf("decode adjustment task %d: %v", id, err)
	}
	return adjustmentWorkItem{
		Instance: &instance,
		Task:     &task,
		Payload:  payload,
	}
}

func makeAdjustmentDue(t *testing.T, id uint) adjustmentWorkItem {
	t.Helper()
	past := time.Now().Add(-time.Second)
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).
		Where("instance_id = ?", id).Update("run_at", past).Error; err != nil {
		t.Fatalf("make adjustment due: %v", err)
	}
	return reloadAdjustmentWorkerInstance(t, id)
}

func TestAdjustmentWorker_SchedulerConcurrencyCap(t *testing.T) {
	initTestDB(t)
	cloud := newWorkerContractCloud("unused")
	cloud.instances = map[string]*adjustmentCloudInstance{}
	cloud.disks = map[string]*adjustmentCloudDisk{}
	ids := make([]uint, 0, 12)
	for i := range 12 {
		instanceID := fmt.Sprintf("ins-worker-cap-%02d", i)
		item := seedAdjustmentWorkerInstance(t, instanceID, adjustmentTypeInstanceType, adjustmentPhaseQueued)
		ids = append(ids, item.Instance.ID)
		diskID := fmt.Sprintf("disk-worker-cap-%02d", i)
		cloud.instances[instanceID] = &adjustmentCloudInstance{
			InstanceID: instanceID, State: "RUNNING", RestrictState: "NORMAL", ChargeType: "PREPAID", StopChargingMode: "NOT_APPLICABLE",
			InstanceType: "Ai2.MEDIUM4", CPU: 2, MemoryGB: 4, Zone: "zone-a", SystemDiskID: diskID, SystemDiskType: "CLOUD_BSSD", SystemDiskSize: 50,
		}
		cloud.disks[diskID] = &adjustmentCloudDisk{
			DiskID: diskID, DiskType: "CLOUD_BSSD", DiskSize: 50, DiskUsage: "SYSTEM_DISK", DiskChargeType: "PREPAID",
			InstanceID: instanceID, Attached: true, DiskState: "ATTACHED",
		}
	}
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	runInstanceAdjustmentWorkerOnce(context.Background(), factory)

	var polling, queued int64
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).Where("instance_id IN ? AND phase = ?", ids, adjustmentPhasePolling).Count(&polling).Error; err != nil {
		t.Fatalf("count polling: %v", err)
	}
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).Where("instance_id IN ? AND phase = ?", ids, adjustmentPhaseQueued).Count(&queued).Error; err != nil {
		t.Fatalf("count queued: %v", err)
	}
	if polling != instanceAdjustmentWorkerConcurrency || queued != 12-instanceAdjustmentWorkerConcurrency || len(cloud.executions) != instanceAdjustmentWorkerConcurrency {
		t.Fatalf("polling=%d queued=%d executions=%d", polling, queued, len(cloud.executions))
	}
}

func TestAdjustmentWorker_RunningInstanceTypeSuccess(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-running-type", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	runInstanceAdjustmentWorkerOnce(context.Background(), factory)
	if len(cloud.executions) != 1 || !cloud.executions[0].ForceStop || cloud.executions[0].TargetInstanceType != "Ai2.LARGE8" {
		t.Fatalf("execute operations=%+v", cloud.executions)
	}

	stored := makeAdjustmentDue(t, item.Instance.ID)
	cloud.instances[item.Instance.InstanceId].InstanceType = "Ai2.LARGE8"
	cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
	cloud.instances[item.Instance.InstanceId].LatestOperationState = "SUCCESS"
	cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = stored.Task.RequestID
	pollInstanceAdjustment(context.Background(), cloud, &stored)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Phase != adjustmentPhaseRestoreSuccess {
		t.Fatalf("phase=%s", stored.Task.Phase)
	}
	restoreInstanceState(context.Background(), cloud, &stored)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.CurrentOperation != model.OpNone || stored.Task.Status != "" || stored.Instance.CVMInstanceType != "Ai2.LARGE8" || stored.Instance.LastKnownStatus != model.StatusRunning {
		t.Fatalf("terminal instance=%+v", stored)
	}
}

func TestAdjustmentWorker_StoppedInstanceTypeRestoresOriginalMode(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-stopped-type", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	item.Payload.OriginalCVMState = "STOPPED"
	item.Payload.OriginalStopChargingMode = "KEEP_CHARGING"
	item.Instance.LastKnownStatus = model.StatusStopped
	saveAdjustmentWorkerItem(t, &item)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.instances[item.Instance.InstanceId].State = "STOPPED"
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	runInstanceAdjustmentWorkerOnce(context.Background(), factory)
	if len(cloud.executions) != 1 || cloud.executions[0].ForceStop {
		t.Fatalf("stopped reset operation=%+v", cloud.executions)
	}

	stored := makeAdjustmentDue(t, item.Instance.ID)
	cloud.instances[item.Instance.InstanceId].InstanceType = "Ai2.LARGE8"
	cloud.instances[item.Instance.InstanceId].State = "RUNNING"
	cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
	cloud.instances[item.Instance.InstanceId].LatestOperationState = "SUCCESS"
	cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = stored.Task.RequestID
	pollInstanceAdjustment(context.Background(), cloud, &stored)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	restoreInstanceState(context.Background(), cloud, &stored)
	if cloud.stopCalls != 1 || len(cloud.stopModes) != 1 || cloud.stopModes[0] != "KEEP_CHARGING" {
		t.Fatalf("stopCalls=%d modes=%v", cloud.stopCalls, cloud.stopModes)
	}
	cloud.instances[item.Instance.InstanceId].State = "STOPPED"
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	restoreInstanceState(context.Background(), cloud, &stored)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.CurrentOperation != model.OpNone || stored.Instance.LastKnownStatus != model.StatusStopped || stored.Instance.CVMInstanceType != "Ai2.LARGE8" {
		t.Fatalf("terminal stopped instance=%+v", stored)
	}
}

func TestAdjustmentWorker_SystemDiskExecutionModes(t *testing.T) {
	cases := []struct {
		name       string
		state      string
		mode       string
		wantOnline bool
		wantForce  bool
	}{
		{name: "running online", state: "RUNNING", mode: adjustmentResizeOnline, wantOnline: true},
		{name: "running offline", state: "RUNNING", mode: adjustmentResizeOffline, wantForce: true},
		{name: "originally stopped", state: "STOPPED", mode: adjustmentResizeOffline},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			initTestDB(t)
			item := seedAdjustmentWorkerInstance(t, "ins-worker-disk-mode", adjustmentTypeSystemDisk, adjustmentPhaseQueued)
			item.Payload.ResizeMode = test.mode
			item.Payload.OriginalCVMState = test.state
			saveAdjustmentWorkerItem(t, &item)
			cloud := newWorkerContractCloud(item.Instance.InstanceId)
			cloud.instances[item.Instance.InstanceId].State = test.state
			factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
			runInstanceAdjustmentWorkerOnce(context.Background(), factory)
			if len(cloud.executions) != 1 {
				t.Fatalf("executions=%+v", cloud.executions)
			}
			operation := cloud.executions[0]
			if operation.ResizeOnline != test.wantOnline || operation.ForceStop != test.wantForce || operation.DiskID != "disk-system" || operation.TargetDiskSize != item.Payload.TargetDiskSize {
				t.Fatalf("operation=%+v", operation)
			}
			if len(cloud.inquiries) != 0 {
				t.Fatalf("system disk JIT unexpectedly used price inquiry: %+v", cloud.inquiries)
			}
		})
	}
}

func TestAdjustmentWorker_RequestIDPollingRules(t *testing.T) {
	cases := []struct {
		name      string
		requestID string
		latestID  string
		latest    string
		target    bool
		wantPhase string
		wantError string
	}{
		{name: "matching success", requestID: "req-1", latestID: "req-1", latest: "SUCCESS", target: true, wantPhase: adjustmentPhaseRestoreSuccess},
		{name: "mismatched failure waits", requestID: "req-1", latestID: "req-other", latest: "FAILED", wantPhase: adjustmentPhasePolling},
		{name: "missing latest ID plus target converges", requestID: "req-1", latestID: "", latest: "", target: true, wantPhase: adjustmentPhaseRestoreSuccess},
		{name: "matching failure restores", requestID: "req-1", latestID: "req-1", latest: "FAILED", wantPhase: adjustmentPhaseRestoreFailure, wantError: reasonResourceSoldOut},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			initTestDB(t)
			item := seedAdjustmentWorkerInstance(t, "ins-worker-poll-id", adjustmentTypeInstanceType, adjustmentPhasePolling)
			item.Task.RequestID = test.requestID
			saveAdjustmentWorkerItem(t, &item)
			cloud := newWorkerContractCloud(item.Instance.InstanceId)
			cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
			cloud.instances[item.Instance.InstanceId].LatestOperationState = test.latest
			cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = test.latestID
			cloud.instances[item.Instance.InstanceId].LatestOperationErrorMessage = "ResourcesSoldOut"
			if test.target {
				cloud.instances[item.Instance.InstanceId].InstanceType = "Ai2.LARGE8"
			}
			pollInstanceAdjustment(context.Background(), cloud, &item)
			stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
			if stored.Task.Phase != test.wantPhase || stored.Task.ErrorCode != test.wantError {
				t.Fatalf("phase=%s error=%s want phase=%s error=%s", stored.Task.Phase, stored.Task.ErrorCode, test.wantPhase, test.wantError)
			}
		})
	}
}

func TestAdjustmentWorker_AmbiguousSubmissionConvergenceAndReadErrors(t *testing.T) {
	t.Run("target already reached never replays", func(t *testing.T) {
		initTestDB(t)
		item := seedAdjustmentWorkerInstance(t, "ins-worker-ambiguous-target", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
		cloud := newWorkerContractCloud(item.Instance.InstanceId)
		cloud.instances[item.Instance.InstanceId].InstanceType = "Ai2.LARGE8"
		reconcileAmbiguousSubmission(context.Background(), cloud, &item)
		stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		if stored.Task.Phase != adjustmentPhaseRestoreSuccess || len(cloud.executions) != 0 {
			t.Fatalf("stored=%+v executions=%+v", stored, cloud.executions)
		}
	})
	t.Run("operating request is adopted without replay", func(t *testing.T) {
		initTestDB(t)
		item := seedAdjustmentWorkerInstance(t, "ins-worker-ambiguous-operating", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
		cloud := newWorkerContractCloud(item.Instance.InstanceId)
		cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
		cloud.instances[item.Instance.InstanceId].LatestOperationState = "OPERATING"
		cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = "req-adopted"
		reconcileAmbiguousSubmission(context.Background(), cloud, &item)
		stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		if stored.Task.Phase != adjustmentPhasePolling || stored.Task.RequestID != "req-adopted" || len(cloud.executions) != 0 {
			t.Fatalf("stored=%+v executions=%+v", stored, cloud.executions)
		}
	})
	t.Run("failed request is recorded without replay", func(t *testing.T) {
		initTestDB(t)
		item := seedAdjustmentWorkerInstance(t, "ins-worker-ambiguous-failed", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
		item.Task.Attempt = instanceAdjustmentCrashObservations - 1
		item.Task.UpdatedAt = time.Now().Add(-instanceAdjustmentCrashInterval)
		saveAdjustmentWorkerItem(t, &item)
		cloud := newWorkerContractCloud(item.Instance.InstanceId)
		cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
		cloud.instances[item.Instance.InstanceId].LatestOperationState = "FAILED"
		cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = "req-failed-before-persist"
		cloud.instances[item.Instance.InstanceId].LatestOperationErrorMessage = "ResourcesSoldOut"
		reconcileAmbiguousSubmission(context.Background(), cloud, &item)
		stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		if stored.Task.Phase != adjustmentPhaseRestoreFailure || stored.Task.ErrorCode != reasonResourceSoldOut || len(cloud.executions) != 0 {
			t.Fatalf("stored=%+v executions=%+v", stored, cloud.executions)
		}
	})
	t.Run("read errors do not count as no-trace observations", func(t *testing.T) {
		initTestDB(t)
		item := seedAdjustmentWorkerInstance(t, "ins-worker-ambiguous-read", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
		item.Task.Attempt = 2
		saveAdjustmentWorkerItem(t, &item)
		cloud := newWorkerContractCloud(item.Instance.InstanceId)
		cloud.describeErr = errors.New("temporary DescribeInstances failure")
		reconcileAmbiguousSubmission(context.Background(), cloud, &item)
		stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		if stored.Task.Attempt != 2 || len(cloud.executions) != 0 {
			t.Fatalf("retry count=%d executions=%+v", stored.Task.Attempt, cloud.executions)
		}
	})
}

func TestAdjustmentWorker_PersistedRequestIDNeverReplaysOnReadError(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-request-read-error", adjustmentTypeInstanceType, adjustmentPhasePolling)
	item.Task.RequestID = "req-persisted"
	saveAdjustmentWorkerItem(t, &item)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.describeErr = errors.New("temporary DescribeInstances failure")
	pollInstanceAdjustment(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Phase != adjustmentPhasePolling || stored.Task.RequestID != "req-persisted" || len(cloud.executions) != 0 || stored.Task.Attempt != 1 {
		t.Fatalf("stored=%+v executions=%+v", stored, cloud.executions)
	}
}

func TestAdjustmentWorker_TimeoutBoundary(t *testing.T) {
	for _, test := range []struct {
		name     string
		age      time.Duration
		timedOut bool
	}{
		{name: "fourteen minutes fifty nine seconds", age: 14*time.Minute + 59*time.Second},
		{name: "fifteen minutes", age: 15*time.Minute + time.Second, timedOut: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			initTestDB(t)
			item := seedAdjustmentWorkerInstance(t, "ins-worker-timeout", adjustmentTypeInstanceType, adjustmentPhasePolling)
			started := time.Now().Add(-test.age)
			item.Task.CreatedAt = started
			saveAdjustmentWorkerItem(t, &item)
			cloud := newWorkerContractCloud(item.Instance.InstanceId)
			advanceInstanceAdjustment(context.Background(), cloud, &item, false)
			stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
			if test.timedOut {
				if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonAdjustmentTimeout || stored.Instance.CurrentOperation != model.OpNone {
					t.Fatalf("timed-out stored=%+v", stored)
				}
			} else if stored.Task.Status != adjustmentStatusProcessing || stored.Task.Phase != adjustmentPhasePolling {
				t.Fatalf("pre-boundary stored=%+v", stored)
			}
		})
	}
	t.Run("expired restore terminates with truthful failure", func(t *testing.T) {
		initTestDB(t)
		item := seedAdjustmentWorkerInstance(t, "ins-worker-restore-timeout", adjustmentTypeInstanceType, adjustmentPhaseRestoreSuccess)
		started := time.Now().Add(-instanceAdjustmentTimeout - time.Second)
		item.Task.CreatedAt = started
		saveAdjustmentWorkerItem(t, &item)
		cloud := newWorkerContractCloud(item.Instance.InstanceId)
		cloud.instances[item.Instance.InstanceId].State = "STARTING"
		advanceInstanceAdjustment(context.Background(), cloud, &item, false)
		stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
		if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonAdjustmentRestoreFailed ||
			stored.Instance.CurrentOperation != model.OpNone || stored.Instance.LastCVMState != "STARTING" {
			t.Fatalf("expired restore stored=%+v", stored)
		}
	})
}

func TestAdjustmentWorker_CloudFailureRestoresAndPersistsProductReason(t *testing.T) {
	for _, originalState := range []string{"RUNNING", "STOPPED"} {
		t.Run(originalState, func(t *testing.T) {
			initTestDB(t)
			item := seedAdjustmentWorkerInstance(t, "ins-worker-cloud-failure", adjustmentTypeInstanceType, adjustmentPhasePolling)
			item.Task.RequestID = "req-failed"
			item.Payload.OriginalCVMState = originalState
			saveAdjustmentWorkerItem(t, &item)
			cloud := newWorkerContractCloud(item.Instance.InstanceId)
			cloud.instances[item.Instance.InstanceId].State = originalState
			cloud.instances[item.Instance.InstanceId].LatestOperation = "ResetInstancesType"
			cloud.instances[item.Instance.InstanceId].LatestOperationState = "FAILED"
			cloud.instances[item.Instance.InstanceId].LatestOperationRequestID = "req-failed"
			cloud.instances[item.Instance.InstanceId].LatestOperationErrorMessage = "InvalidAccount.InsufficientBalance"
			pollInstanceAdjustment(context.Background(), cloud, &item)
			stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
			restoreInstanceState(context.Background(), cloud, &stored)
			stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
			if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonInsufficientBalance || stored.Instance.CurrentOperation != model.OpNone || stored.Instance.LastKnownStatus != stringLower(originalState) {
				t.Fatalf("terminal failure=%+v", stored)
			}
		})
	}
}

func stringLower(state string) string {
	if state == "RUNNING" {
		return model.StatusRunning
	}
	return model.StatusStopped
}

func TestAdjustmentWorker_RestoreFailureKeepsTruthfulCloudState(t *testing.T) {
	cases := []struct {
		name          string
		originalState string
		cloudState    string
		startErr      error
		stopErr       error
	}{
		{name: "start failure", originalState: "RUNNING", cloudState: "STOPPED", startErr: errors.New("start failed")},
		{name: "stop failure", originalState: "STOPPED", cloudState: "RUNNING", stopErr: errors.New("stop failed")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			initTestDB(t)
			item := seedAdjustmentWorkerInstance(t, "ins-worker-restore-failure", adjustmentTypeInstanceType, adjustmentPhaseRestoreSuccess)
			item.Payload.OriginalCVMState = test.originalState
			saveAdjustmentWorkerItem(t, &item)
			cloud := newWorkerContractCloud(item.Instance.InstanceId)
			cloud.instances[item.Instance.InstanceId].State = test.cloudState
			cloud.startErr = test.startErr
			cloud.stopErr = test.stopErr
			restoreInstanceState(context.Background(), cloud, &item)
			stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
			if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonAdjustmentRestoreFailed || stored.Instance.LastKnownStatus != stringLower(test.cloudState) || stored.Instance.CurrentOperation != model.OpNone {
				t.Fatalf("truthful restore failure=%+v", stored)
			}
		})
	}
}

func TestAdjustmentWorker_JITFailureSkipsCloudWrite(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-jit-contract-fail", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.instances[item.Instance.InstanceId].StopChargingMode = "STOP_CHARGING"
	submitFreshAdjustment(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonStopChargingNotSupported || len(cloud.executions) != 0 {
		t.Fatalf("stored=%+v executions=%+v", stored, cloud.executions)
	}
}

func TestAdjustmentWorker_SystemDiskJITSkipsPriceInquiry(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-jit-contract-success", adjustmentTypeSystemDisk, adjustmentPhaseSubmitting)
	item.Payload.ResizeMode = adjustmentResizeOffline
	saveAdjustmentWorkerItem(t, &item)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	submitFreshAdjustment(context.Background(), cloud, &item)
	if len(cloud.inquiries) != 0 || len(cloud.executions) != 1 {
		t.Fatalf("inquiries=%+v executions=%+v", cloud.inquiries, cloud.executions)
	}
}

func TestAdjustmentWorker_CloudFactoryFailurePreservesClaimedTask(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-factory-failure", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	runInstanceAdjustmentWorkerOnce(context.Background(), func(context.Context) (instanceAdjustmentCloudGateway, error) {
		return nil, errors.New("client unavailable")
	})
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusProcessing || stored.Task.Phase != adjustmentPhaseSubmitting {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_MissingExecutionStartFailsTask(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-missing-start", adjustmentTypeInstanceType, adjustmentPhasePolling)
	item.Task.CreatedAt = time.Time{}
	saveAdjustmentWorkerItem(t, &item)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	advanceInstanceAdjustment(context.Background(), cloud, &item, false)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonCloudAdjustmentFailed {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_RetryableExecuteErrorSchedulesObservation(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-retryable-write", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.executeErr = errors.New("network timeout")
	submitFreshAdjustment(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusProcessing || stored.Task.Phase != adjustmentPhaseSubmitting ||
		stored.Task.RequestID != "" || stored.Task.RunAt.IsZero() {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_NonRetryableExecuteErrorPersistsProductReason(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-nonretryable-write", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.executeErr = errors.New("account balance unavailable")
	submitFreshAdjustment(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonInsufficientBalance {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_MissingCVMFailsAmbiguousSubmission(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-cloud-gone", adjustmentTypeInstanceType, adjustmentPhaseSubmitting)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	delete(cloud.instances, item.Instance.InstanceId)
	reconcileAmbiguousSubmission(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonCVMInstanceNotFound {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_TransientRestoreStateKeepsTaskProcessing(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-transient-restore", adjustmentTypeInstanceType, adjustmentPhaseRestoreSuccess)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	cloud.instances[item.Instance.InstanceId].State = "STARTING"
	restoreInstanceState(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusProcessing || stored.Task.Phase != adjustmentPhaseRestoreSuccess ||
		stored.Instance.CurrentOperation != model.OpAdjustInstanceType {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_InvalidOriginalStateFailsRestore(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-invalid-original", adjustmentTypeInstanceType, adjustmentPhaseRestoreSuccess)
	item.Payload.OriginalCVMState = "UNKNOWN"
	saveAdjustmentWorkerItem(t, &item)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	restoreInstanceState(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonAdjustmentRestoreFailed {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAdjustmentWorker_DiskTargetAndOperationRecognition(t *testing.T) {
	diskInstance := &adjustmentWorkItem{
		Instance: &model.Instance{},
		Task:     &model.InstanceAdjustment{Type: adjustmentTypeSystemDisk},
		Payload:  model.InstanceAdjustmentPayload{TargetDiskSize: 100},
	}
	diskCloud := &adjustmentCloudInstance{SystemDiskSize: 100, LatestOperation: "ResizeInstanceDisks"}
	if !adjustmentTargetReached(diskInstance, diskCloud) || !relevantLatestOperation(diskInstance, diskCloud) {
		t.Fatalf("disk target/latest operation not recognized: instance=%+v cloud=%+v", diskInstance, diskCloud)
	}
}

func TestAdjustmentWorker_ObservedStableStateUsesPollWrite(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-observed-state", adjustmentTypeSystemDisk, adjustmentPhasePolling)

	scheduleNextAdjustmentPoll(context.Background(), &item, &adjustmentCloudInstance{State: "STOPPED"}, time.Second)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.LastKnownStatus != model.StatusStopped || stored.Instance.LastCVMState != "STOPPED" || stored.Instance.StatusSyncedAt == nil {
		t.Fatalf("stopped observation not persisted: %+v", stored)
	}

	scheduleNextAdjustmentPoll(context.Background(), &stored, &adjustmentCloudInstance{State: "STARTING"}, time.Second)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.LastKnownStatus != model.StatusStopped || stored.Instance.LastCVMState != "STARTING" {
		t.Fatalf("transient observation changed stable display: %+v", stored)
	}

	scheduleNextAdjustmentPoll(context.Background(), &stored, &adjustmentCloudInstance{State: "RUNNING"}, time.Second)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if stored.Instance.LastKnownStatus != model.StatusRunning || stored.Instance.LastCVMState != "RUNNING" {
		t.Fatalf("running observation not persisted: %+v", stored)
	}
}

func TestAdjustmentWorker_QueuedWaitDoesNotConsumeExecutionTimeout(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-queued-timeout", adjustmentTypeInstanceType, adjustmentPhaseQueued)
	queuedAt := time.Now().Add(-instanceAdjustmentTimeout - time.Minute)
	if err := model.DB(context.Background()).Model(&model.InstanceAdjustment{}).
		Where("instance_id = ?", item.Instance.ID).
		UpdateColumn("created_at", queuedAt).Error; err != nil {
		t.Fatalf("age queued task: %v", err)
	}

	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	runInstanceAdjustmentWorkerOnce(context.Background(), factory)

	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	if len(cloud.executions) != 1 {
		t.Fatalf("queued task executions=%d, want 1", len(cloud.executions))
	}
	if stored.Task.Phase != adjustmentPhasePolling {
		t.Fatalf("phase=%s, want %s", stored.Task.Phase, adjustmentPhasePolling)
	}
	if stored.Task.ExecutionStartedAt == nil || stored.Task.ExecutionStartedAt.Before(queuedAt.Add(instanceAdjustmentTimeout)) {
		t.Fatalf("execution_started_at=%v queued_at=%v", stored.Task.ExecutionStartedAt, queuedAt)
	}
}

func TestAdjustmentWorker_MissingCVMFailurePersistsDestroyedProjection(t *testing.T) {
	initTestDB(t)
	item := seedAdjustmentWorkerInstance(t, "ins-worker-missing-cvm", adjustmentTypeInstanceType, adjustmentPhasePolling)
	cloud := newWorkerContractCloud(item.Instance.InstanceId)
	delete(cloud.instances, item.Instance.InstanceId)

	pollInstanceAdjustment(context.Background(), cloud, &item)
	stored := reloadAdjustmentWorkerInstance(t, item.Instance.ID)
	restoreInstanceState(context.Background(), cloud, &stored)
	stored = reloadAdjustmentWorkerInstance(t, item.Instance.ID)

	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonCVMInstanceNotFound {
		t.Fatalf("adjustment status=%s error=%s", stored.Task.Status, stored.Task.ErrorCode)
	}
	if stored.Instance.CurrentOperation != model.OpNone || stored.Instance.LastKnownStatus != model.StatusDestroyed || stored.Instance.LastCVMState != "NOTFOUND" {
		t.Fatalf("instance operation=%s status=%s cvm_state=%s", stored.Instance.CurrentOperation, stored.Instance.LastKnownStatus, stored.Instance.LastCVMState)
	}
}
