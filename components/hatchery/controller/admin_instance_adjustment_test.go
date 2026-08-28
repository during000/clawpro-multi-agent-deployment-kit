package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/gorilla/sessions"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"gorm.io/gorm"
)

type fakeAdjustmentCloud struct {
	mu sync.Mutex

	instances  map[string]*adjustmentCloudInstance
	disks      map[string]*adjustmentCloudDisk
	available  bool
	quota      adjustmentDiskQuota
	denied     []deniedAction
	deniedErr  error
	inquiryErr error
	executeErr error
	requestID  string

	inquiries  []adjustmentOperation
	executions []adjustmentOperation
	startCalls int
	stopCalls  int
}

func (f *fakeAdjustmentCloud) DescribeInstances(_ context.Context, ids []string) (map[string]*adjustmentCloudInstance, error) {
	result := make(map[string]*adjustmentCloudInstance, len(ids))
	for _, id := range ids {
		if instance := f.instances[id]; instance != nil {
			copyValue := *instance
			result[id] = &copyValue
		}
	}
	return result, nil
}

func (f *fakeAdjustmentCloud) DescribeDisks(_ context.Context, ids []string) (map[string]*adjustmentCloudDisk, error) {
	result := make(map[string]*adjustmentCloudDisk, len(ids))
	for _, id := range ids {
		if disk := f.disks[id]; disk != nil {
			copyValue := *disk
			result[id] = &copyValue
		}
	}
	return result, nil
}

func (f *fakeAdjustmentCloud) CheckInstanceTypeAvailable(context.Context, *adjustmentCloudInstance, string) (bool, error) {
	return f.available, nil
}

func (f *fakeAdjustmentCloud) GetSystemDiskQuota(context.Context, *adjustmentCloudInstance, *adjustmentCloudDisk) (adjustmentDiskQuota, error) {
	return f.quota, nil
}

func (f *fakeAdjustmentCloud) DeniedActions(context.Context, string, []string) ([]deniedAction, error) {
	return f.denied, f.deniedErr
}

func (f *fakeAdjustmentCloud) InquiryInstanceType(_ context.Context, operation adjustmentOperation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inquiries = append(f.inquiries, operation)
	return f.inquiryErr
}

func (f *fakeAdjustmentCloud) Execute(_ context.Context, operation adjustmentOperation) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.executions = append(f.executions, operation)
	return f.requestID, f.executeErr
}

func (f *fakeAdjustmentCloud) StartInstance(context.Context, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCalls++
	return nil
}

func (f *fakeAdjustmentCloud) StopInstance(context.Context, string, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	return nil
}

func newHappyAdjustmentCloud(instanceID string) *fakeAdjustmentCloud {
	return &fakeAdjustmentCloud{
		instances: map[string]*adjustmentCloudInstance{
			instanceID: {
				InstanceID:       instanceID,
				State:            "RUNNING",
				RestrictState:    "NORMAL",
				ChargeType:       "PREPAID",
				StopChargingMode: "NOT_APPLICABLE",
				InstanceType:     "Ai2.MEDIUM4",
				CPU:              2,
				MemoryGB:         4,
				Zone:             "ap-guangzhou-6",
				SystemDiskID:     "disk-system",
				SystemDiskType:   "CLOUD_BSSD",
				SystemDiskSize:   50,
			},
		},
		disks: map[string]*adjustmentCloudDisk{
			"disk-system": {
				DiskID:         "disk-system",
				DiskType:       "CLOUD_BSSD",
				DiskSize:       50,
				DiskUsage:      "SYSTEM_DISK",
				DiskChargeType: "PREPAID",
				InstanceID:     instanceID,
				Attached:       true,
				DiskState:      "ATTACHED",
			},
		},
		available: true,
		quota:     adjustmentDiskQuota{Available: true, MinSize: 50, MaxSize: 500, StepSize: 10},
		requestID: "req-adjustment",
	}
}

func createAdjustmentTestInstance(t *testing.T, instanceID string) model.Instance {
	t.Helper()
	instance := model.Instance{
		Name:            "adjustment-test",
		InstanceId:      instanceID,
		Source:          model.InstanceSourceCVM,
		AgentType:       model.AgentTypeOpenClaw,
		AgentReady:      1,
		LastKnownStatus: model.StatusRunning,
		LastStableState: model.StatusRunning,
		LastCVMState:    "RUNNING",
	}
	if err := model.DB(context.Background()).Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return instance
}

func TestDecodeAdminInstanceAdjustmentRequest_StrictAndNormalize(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantError bool
		check     func(t *testing.T, req instanceAdjustmentRequest)
	}{
		{
			name:      "unknown field",
			body:      `{"ids":[1],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8","unknown":true}`,
			wantError: true,
		},
		{
			name: "known irrelevant fields are ignored",
			body: `{"ids":[1,1],"adjustment_type":"instance_type","target_instance_type":" Ai2.LARGE8 ","target_system_disk_size":100,"resize_mode":"garbage"}`,
			check: func(t *testing.T, req instanceAdjustmentRequest) {
				if len(*req.IDs) != 1 || *req.TargetInstanceType != "Ai2.LARGE8" {
					t.Fatalf("unexpected normalized request: %+v", req)
				}
				if req.TargetSystemDiskSize != nil || req.ResizeMode != "" {
					t.Fatalf("irrelevant fields were not cleared: %+v", req)
				}
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config/validate", bytes.NewBufferString(test.body))
			w := httptest.NewRecorder()
			decoded, richErr := decodeAdminInstanceAdjustmentRequest(w, req)
			if (richErr != nil) != test.wantError {
				t.Fatalf("error=%v wantError=%v", richErr, test.wantError)
			}
			if richErr == nil && test.check != nil {
				test.check(t, decoded)
			}
		})
	}
}

func TestValidateResolvedAdjustmentTargets_CommonCVMGates(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-adjust")
	target := "Ai2.LARGE8"
	req := instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}

	t.Run("happy path and mandatory inquiry", func(t *testing.T) {
		cloud := newHappyAdjustmentCloud(instance.InstanceId)
		results, err := validateResolvedAdjustmentTargets(context.Background(), req,
			[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
		if err != nil || len(results) != 1 || !results[0].Adjustable {
			t.Fatalf("result=%+v err=%v", results, err)
		}
		if len(cloud.inquiries) != 1 {
			t.Fatalf("inquiries=%+v", cloud.inquiries)
		}
		if cloud.inquiries[0].TargetInstanceType != target || !cloud.inquiries[0].ForceStop {
			t.Fatalf("unexpected operation: %+v", cloud.inquiries[0])
		}
	})

	t.Run("denied action failure falls back to mandatory inquiry", func(t *testing.T) {
		cloud := newHappyAdjustmentCloud(instance.InstanceId)
		cloud.deniedErr = errors.New("action unavailable")
		results, err := validateResolvedAdjustmentTargets(context.Background(), req,
			[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
		if err != nil || !results[0].Adjustable || len(cloud.inquiries) != 1 {
			t.Fatalf("result=%+v inquiries=%d err=%v", results, len(cloud.inquiries), err)
		}
	})

	t.Run("stop charging is rejected before inquiry", func(t *testing.T) {
		cloud := newHappyAdjustmentCloud(instance.InstanceId)
		cloud.instances[instance.InstanceId].StopChargingMode = "STOP_CHARGING"
		results, err := validateResolvedAdjustmentTargets(context.Background(), req,
			[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
		if err != nil || results[0].ReasonCode != reasonStopChargingNotSupported || len(cloud.inquiries) != 0 {
			t.Fatalf("result=%+v inquiries=%d err=%v", results, len(cloud.inquiries), err)
		}
	})

	t.Run("inquiry balance error is productized", func(t *testing.T) {
		cloud := newHappyAdjustmentCloud(instance.InstanceId)
		cloud.inquiryErr = sdkerrors.NewTencentCloudSDKError("InvalidAccount.InsufficientBalance", "balance", "req")
		results, err := validateResolvedAdjustmentTargets(context.Background(), req,
			[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
		if err != nil || results[0].ReasonCode != reasonInsufficientBalance {
			t.Fatalf("result=%+v err=%v", results, err)
		}
	})
}

func TestValidateSystemDiskAdjustment_QuotaStepAndOperation(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-disk")
	cloud := newHappyAdjustmentCloud(instance.InstanceId)

	invalidSize := int64(55)
	req := instanceAdjustmentRequest{
		AdjustmentType:       adjustmentTypeSystemDisk,
		TargetSystemDiskSize: &invalidSize,
		ResizeMode:           adjustmentResizeOnline,
	}
	results, err := validateResolvedAdjustmentTargets(context.Background(), req,
		[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
	if err != nil || results[0].ReasonCode != reasonInvalidDiskSize {
		t.Fatalf("result=%+v err=%v", results, err)
	}
	if results[0].MinDiskSize != 60 || results[0].MaxDiskSize != 500 || results[0].StepSize != 10 {
		t.Fatalf("unexpected quota response: %+v", results[0])
	}

	validSize := int64(60)
	req.TargetSystemDiskSize = &validSize
	results, err = validateResolvedAdjustmentTargets(context.Background(), req,
		[]resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}, cloud, false)
	if err != nil || !results[0].Adjustable || len(cloud.inquiries) != 0 {
		t.Fatalf("result=%+v inquiries=%+v err=%v", results, cloud.inquiries, err)
	}
	operation := results[0].operation
	if operation.DiskID != "disk-system" || operation.TargetDiskSize != 60 || !operation.ResizeOnline || operation.ForceStop {
		t.Fatalf("unexpected resize operation: %+v", operation)
	}
}

func TestHandleAdminInstanceAdjustment_PartialAcceptanceAndIdempotency(t *testing.T) {
	initTestDB(t)
	cloudInstance := createAdjustmentTestInstance(t, "ins-submit")
	localInstance := model.Instance{Name: "local", InstanceId: "local-submit", Source: model.InstanceSourceLocal, LastKnownStatus: model.StatusRunning}
	if err := model.DB(context.Background()).Create(&localInstance).Error; err != nil {
		t.Fatalf("create local instance: %v", err)
	}
	cloud := newHappyAdjustmentCloud(cloudInstance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }

	body := map[string]any{
		"ids":                  []uint{cloudInstance.ID, localInstance.ID},
		"adjustment_type":      adjustmentTypeInstanceType,
		"target_instance_type": "Ai2.LARGE8",
	}
	encoded, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config", bytes.NewReader(encoded))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	handleAdminInstanceAdjustment(w, req, factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var response instanceAdjustmentSubmitResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AcceptedCount != 1 || response.RejectedCount != 1 {
		t.Fatalf("response=%+v", response)
	}
	var stored model.Instance
	if err := model.DB(context.Background()).First(&stored, cloudInstance.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	var task model.InstanceAdjustment
	if err := model.DB(context.Background()).Where("instance_id = ?", stored.ID).Take(&task).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if stored.CurrentOperation != model.OpAdjustInstanceType || task.Phase != adjustmentPhaseQueued {
		t.Fatalf("stored=%+v", stored)
	}

	encoded, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config", bytes.NewReader(encoded))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w = httptest.NewRecorder()
	handleAdminInstanceAdjustment(w, req, factory)
	if w.Code != http.StatusOK {
		t.Fatalf("idempotent status=%d body=%s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode idempotent response: %v", err)
	}
	if response.AlreadyProcessingCount != 1 {
		t.Fatalf("idempotent response=%+v", response)
	}
}

func TestHandleAdminInstanceAdjustment_AcceptanceFailureIsPerInstance(t *testing.T) {
	initTestDB(t)
	first := createAdjustmentTestInstance(t, "ins-accept-first")
	second := createAdjustmentTestInstance(t, "ins-accept-fails")
	third := createAdjustmentTestInstance(t, "ins-accept-third")

	cloud := newContractAdjustmentCloud(first.InstanceId)
	addDiskContractInstance(cloud, second.InstanceId, "disk-accept-fails", "ap-guangzhou-6")
	addDiskContractInstance(cloud, third.InstanceId, "disk-accept-third", "ap-guangzhou-6")
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }

	db := model.DB(context.Background())
	const callbackName = "test:fail_middle_adjustment_acceptance"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		task, ok := tx.Statement.Dest.(*model.InstanceAdjustment)
		if ok && task.InstanceID == second.ID {
			tx.AddError(errors.New("injected adjustment acceptance failure"))
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(callbackName)
	})

	body := adjustmentContractBody(
		[]uint{first.ID, second.ID, third.ID},
		adjustmentTypeInstanceType,
		"Ai2.LARGE8",
	)
	rr := httptest.NewRecorder()
	handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var response instanceAdjustmentSubmitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AcceptedCount != 2 || response.RejectedCount != 1 || response.AlreadyProcessingCount != 0 {
		t.Fatalf("response counts=%+v", response)
	}
	if len(response.Results) != 3 ||
		!response.Results[0].Accepted ||
		response.Results[1].Status != "rejected" ||
		response.Results[1].ReasonCode != reasonInternalError ||
		response.Results[1].ReasonMessage != i18n.T(context.Background(), i18n.MsgAdjustmentReasonInternalError) ||
		!response.Results[2].Accepted {
		t.Fatalf("ordered results=%+v", response.Results)
	}

	firstStored := reloadAdjustmentWorkerInstance(t, first.ID)
	secondStored := reloadAdjustmentWorkerInstance(t, second.ID)
	thirdStored := reloadAdjustmentWorkerInstance(t, third.ID)
	if firstStored.Instance.CurrentOperation != model.OpAdjustInstanceType ||
		thirdStored.Instance.CurrentOperation != model.OpAdjustInstanceType ||
		secondStored.Instance.CurrentOperation != model.OpNone ||
		secondStored.Task.Status != "" {
		t.Fatalf("first=%+v second=%+v third=%+v", firstStored, secondStored, thirdStored)
	}
}

func TestResourceAdjustmentLockBlocksDeleteAndHidesActions(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-lock")
	now := time.Now()
	instance.CurrentOperation = model.OpAdjustInstanceType
	instance.CurrentOperationState = model.OpStateProcessing
	instance.CurrentOperationUpdatedAt = &now
	if err := model.DB(context.Background()).Save(&instance).Error; err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := setOperation(model.DB(context.Background()), &instance, model.OpDelete); !errors.Is(err, ErrOperationInProgress) {
		t.Fatalf("delete should be blocked, err=%v", err)
	}
	status := ResolveInstanceStatus(context.Background(), &instance, &CVMInstanceInfo{State: "STOPPING"}, nil)
	if status.Status != model.StatusRunning || len(status.Actions) != 0 {
		t.Fatalf("status=%+v", status)
	}
}

func adjustmentContractBody(ids []uint, adjustmentType string, target any) map[string]any {
	body := map[string]any{"ids": ids, "adjustment_type": adjustmentType}
	if adjustmentType == adjustmentTypeInstanceType {
		body["target_instance_type"] = target
	} else {
		body["target_system_disk_size"] = target
		body["resize_mode"] = adjustmentResizeOnline
	}
	return body
}

func adjustmentContractRequest(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createAdjustmentTaskForTest(
	t *testing.T,
	instance *model.Instance,
	status string,
	adjustmentType string,
	phase string,
	payload model.InstanceAdjustmentPayload,
) model.InstanceAdjustment {
	t.Helper()
	task := model.InstanceAdjustment{
		Identifier: instance.Identifier,
		InstanceID: instance.ID,
		Status:     status,
		Type:       adjustmentType,
		Phase:      phase,
		RunAt:      time.Now(),
	}
	if err := task.SetPayload(payload); err != nil {
		t.Fatalf("encode adjustment task: %v", err)
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create adjustment task: %v", err)
	}
	return task
}

func TestAdjustmentRequestContract_AuthorizationMethodAndStrictJSON(t *testing.T) {
	initTestDB(t)
	cloud := newHappyAdjustmentCloud("unused")
	var factoryCalls atomic.Int64
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) {
		factoryCalls.Add(1)
		return cloud, nil
	}

	t.Run("method rejected before cloud", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/instances/adjust-config/validate", nil)
		req.Header.Set("Authorization", "Bearer "+AdminToken)
		rr := httptest.NewRecorder()
		handleAdminInstanceAdjustmentValidate(rr, req, factory)
		if rr.Code != http.StatusMethodNotAllowed || factoryCalls.Load() != 0 {
			t.Fatalf("status=%d factoryCalls=%d body=%s", rr.Code, factoryCalls.Load(), rr.Body.String())
		}
	})

	t.Run("unauthorized rejected before cloud", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config/validate", bytes.NewBufferString(`{}`))
		rr := httptest.NewRecorder()
		handleAdminInstanceAdjustmentValidate(rr, req, factory)
		if rr.Code != http.StatusUnauthorized || factoryCalls.Load() != 0 {
			t.Fatalf("status=%d factoryCalls=%d body=%s", rr.Code, factoryCalls.Load(), rr.Body.String())
		}
	})

	t.Run("non admin forbidden before cloud", func(t *testing.T) {
		oldStore := Store
		Store = sessions.NewCookieStore([]byte("adjustment-contract-session-secret"))
		t.Cleanup(func() { Store = oldStore })
		user := model.User{Username: "adjustment-contract-user", Password: "x", Role: "user"}
		if err := model.DB(context.Background()).Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		req := adminCreateSessionReq(t, user.Username, adjustmentContractBody([]uint{1}, adjustmentTypeInstanceType, "Ai2.LARGE8"))
		rr := httptest.NewRecorder()
		handleAdminInstanceAdjustmentValidate(rr, req, factory)
		if rr.Code != http.StatusForbidden || factoryCalls.Load() != 0 {
			t.Fatalf("status=%d factoryCalls=%d body=%s", rr.Code, factoryCalls.Load(), rr.Body.String())
		}
	})

	for _, test := range []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"ids":[1]`},
		{name: "unknown field", body: `{"ids":[1],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8","extra":true}`},
		{name: "trailing value", body: `{"ids":[1],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"} {}`},
		{name: "wrong JSON type", body: `{"ids":"1","adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config/validate", bytes.NewBufferString(test.body))
			req.Header.Set("Authorization", "Bearer "+AdminToken)
			rr := httptest.NewRecorder()
			handleAdminInstanceAdjustmentValidate(rr, req, factory)
			if rr.Code != http.StatusBadRequest || factoryCalls.Load() != 0 {
				t.Fatalf("status=%d factoryCalls=%d body=%s", rr.Code, factoryCalls.Load(), rr.Body.String())
			}
		})
	}
}

func TestAdjustmentRequestContract_EnvelopeDedupCountAndOrder(t *testing.T) {
	oneHundredOne := make([]uint, 101)
	for i := range oneHundredOne {
		oneHundredOne[i] = uint(i + 1)
	}
	tooMany, _ := json.Marshal(adjustmentContractBody(oneHundredOne, adjustmentTypeInstanceType, "Ai2.LARGE8"))
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "both ID forms", body: `{"ids":[1],"instance_ids":["ins-1"],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`},
		{name: "neither ID form", body: `{"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`},
		{name: "empty IDs", body: `{"ids":[],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`},
		{name: "all zero IDs", body: `{"ids":[0,0],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`},
		{name: "too many IDs", body: string(tooMany)},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(test.body))
			if _, richErr := decodeAdminInstanceAdjustmentRequest(httptest.NewRecorder(), req); richErr == nil {
				t.Fatal("invalid envelope accepted")
			}
		})
	}

	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{"ids":[3,1,3,0,2,1],"adjustment_type":"instance_type","target_instance_type":" Ai2.LARGE8 "}`))
	decoded, richErr := decodeAdminInstanceAdjustmentRequest(httptest.NewRecorder(), req)
	if richErr != nil {
		t.Fatalf("decode valid IDs: %v", richErr)
	}
	if !reflect.DeepEqual(*decoded.IDs, []uint{3, 1, 2}) || *decoded.TargetInstanceType != "Ai2.LARGE8" {
		t.Fatalf("normalized request=%+v IDs=%v", decoded, *decoded.IDs)
	}

	req = httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{"instance_ids":[" ins-b ","ins-a","ins-b"," "],"adjustment_type":"instance_type","target_instance_type":"Ai2.LARGE8"}`))
	decoded, richErr = decodeAdminInstanceAdjustmentRequest(httptest.NewRecorder(), req)
	if richErr != nil || !reflect.DeepEqual(*decoded.InstanceIDs, []string{"ins-b", "ins-a"}) {
		t.Fatalf("normalized instance IDs=%v err=%v", decoded.InstanceIDs, richErr)
	}
}

func TestAdjustmentRequestContract_BusinessValuesArePerInstanceReasons(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-business-values")
	cloud := newHappyAdjustmentCloud(instance.InstanceId)
	targets := []resolvedAdjustmentTarget{{DBID: instance.ID, Instance: &instance}}

	unknown := "Ai2.UNKNOWN"
	results, err := validateResolvedAdjustmentTargets(context.Background(), instanceAdjustmentRequest{
		AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &unknown,
	}, targets, cloud, false)
	if err != nil || results[0].ReasonCode != reasonUnsupportedInstanceType {
		t.Fatalf("unknown target result=%+v err=%v", results, err)
	}

	zero := int64(0)
	results, err = validateResolvedAdjustmentTargets(context.Background(), instanceAdjustmentRequest{
		AdjustmentType: adjustmentTypeSystemDisk, TargetSystemDiskSize: &zero, ResizeMode: adjustmentResizeOnline,
	}, targets, cloud, false)
	if err != nil || results[0].ReasonCode != reasonDiskShrinkNotSupported {
		t.Fatalf("zero disk target result=%+v err=%v", results, err)
	}
}

func TestAdjustmentRequestContract_FirstFailurePriority(t *testing.T) {
	target := "Ai2.UNKNOWN"
	req := instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}
	base := model.Instance{InstanceId: "ins-priority", Source: model.InstanceSourceCVM, LastKnownStatus: model.StatusRunning}
	cases := []struct {
		name       string
		edit       func(*model.Instance)
		adjustment *model.InstanceAdjustment
		want       string
	}{
		{name: "local before all other failures", edit: func(i *model.Instance) {
			i.Source = model.InstanceSourceLocal
			i.IsDoctorNode = true
			i.CurrentOperation = model.OpDelete
			i.LastKnownStatus = model.StatusLoading
		}, want: reasonCloudInstanceRequired},
		{name: "doctor before busy", edit: func(i *model.Instance) {
			i.IsDoctorNode = true
			i.CurrentOperation = model.OpDelete
			i.LastKnownStatus = model.StatusLoading
		}, want: reasonDoctorNodeNotAllowed},
		{name: "busy before cached status", edit: func(i *model.Instance) { i.CurrentOperation = model.OpDelete; i.LastKnownStatus = model.StatusLoading }, want: reasonOperationInProgress},
		{name: "cached status before cloud and target", edit: func(i *model.Instance) { i.LastKnownStatus = model.StatusLoading }, want: reasonInstanceStatusNotSupported},
		{name: "processing adjustment before cached status", edit: func(i *model.Instance) {
			i.LastKnownStatus = model.StatusLoading
		}, adjustment: &model.InstanceAdjustment{Status: adjustmentStatusProcessing}, want: reasonOperationInProgress},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instance := base
			test.edit(&instance)
			cloud := newHappyAdjustmentCloud(instance.InstanceId)
			results, err := validateResolvedAdjustmentTargets(context.Background(), req, []resolvedAdjustmentTarget{{Instance: &instance, Adjustment: test.adjustment}}, cloud, false)
			if err != nil || results[0].ReasonCode != test.want {
				t.Fatalf("result=%+v err=%v want=%s", results, err, test.want)
			}
		})
	}
}

func TestAdjustmentRequestContract_ValidateMixedPartialResultsPreserveOrder(t *testing.T) {
	initTestDB(t)
	cloudInstance := createAdjustmentTestInstance(t, "ins-validate-mixed")
	local := model.Instance{Name: "local-mixed", InstanceId: "local-mixed", Source: model.InstanceSourceLocal, LastKnownStatus: model.StatusRunning}
	if err := model.DB(context.Background()).Create(&local).Error; err != nil {
		t.Fatalf("create local: %v", err)
	}
	cloud := newHappyAdjustmentCloud(cloudInstance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	body := adjustmentContractBody([]uint{local.ID, cloudInstance.ID, 999999}, adjustmentTypeInstanceType, "Ai2.LARGE8")
	rr := httptest.NewRecorder()
	handleAdminInstanceAdjustmentValidate(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config/validate", body), factory)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response instanceAdjustmentValidateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AdjustableCount != 1 || response.NonAdjustableCount != 2 || len(response.Results) != 3 {
		t.Fatalf("response=%+v", response)
	}
	if response.Results[0].ID != local.ID || response.Results[0].ReasonCode != reasonCloudInstanceRequired ||
		response.Results[1].ID != cloudInstance.ID || !response.Results[1].Adjustable ||
		response.Results[2].ID != 999999 || response.Results[2].ReasonCode != reasonInstanceNotFound {
		t.Fatalf("ordered results=%+v", response.Results)
	}
}
func TestAdjustmentRequestContract_ValidateReturnsProductMessages(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-product-messages")
	cloud := newHappyAdjustmentCloud(instance.InstanceId)
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	cases := []struct {
		name           string
		adjustmentType string
		target         any
		wantCode       string
		wantMessage    string
	}{
		{name: "instance type unchanged", adjustmentType: adjustmentTypeInstanceType, target: "Ai2.MEDIUM4", wantCode: reasonInstanceTypeUnchanged, wantMessage: "已是目标规格"},
		{name: "instance type downgrade", adjustmentType: adjustmentTypeInstanceType, target: "Ai2.MEDIUM2", wantCode: reasonInstanceTypeDowngrade, wantMessage: "不支持降配"},
		{name: "disk size unchanged", adjustmentType: adjustmentTypeSystemDisk, target: int64(50), wantCode: reasonDiskSizeUnchanged, wantMessage: "目标容量需大于当前系统盘容量"},
		{name: "disk shrink", adjustmentType: adjustmentTypeSystemDisk, target: int64(40), wantCode: reasonDiskShrinkNotSupported, wantMessage: "不支持缩容"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			body := adjustmentContractBody([]uint{instance.ID}, test.adjustmentType, test.target)
			rr := httptest.NewRecorder()
			handleAdminInstanceAdjustmentValidate(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config/validate", body), factory)
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var response instanceAdjustmentValidateResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(response.Results) != 1 || response.Results[0].ReasonCode != test.wantCode || response.Results[0].ReasonMessage != test.wantMessage {
				t.Fatalf("response=%+v want code=%q message=%q", response, test.wantCode, test.wantMessage)
			}
		})
	}
}

func TestAdjustmentRequestContract_SubmitRevalidationPartiallyAccepts(t *testing.T) {
	initTestDB(t)
	first := createAdjustmentTestInstance(t, "ins-submit-revalidate-a")
	second := createAdjustmentTestInstance(t, "ins-submit-revalidate-b")
	cloud := newContractAdjustmentCloud(first.InstanceId)
	addDiskContractInstance(cloud, second.InstanceId, "disk-submit-revalidate-b", "ap-guangzhou-6")
	cloud.instances[second.InstanceId].State = "STOPPING"
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	body := adjustmentContractBody([]uint{first.ID, second.ID}, adjustmentTypeInstanceType, "Ai2.LARGE8")
	rr := httptest.NewRecorder()
	handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response instanceAdjustmentSubmitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AcceptedCount != 1 || response.RejectedCount != 1 || !response.Results[0].Accepted ||
		response.Results[1].ReasonCode != reasonInstanceStatusNotSupported {
		t.Fatalf("response=%+v", response)
	}
	firstStored := reloadAdjustmentWorkerInstance(t, first.ID)
	secondStored := reloadAdjustmentWorkerInstance(t, second.ID)
	if firstStored.Task.Phase != adjustmentPhaseQueued || secondStored.Instance.CurrentOperation != model.OpNone || secondStored.Task.Status != "" {
		t.Fatalf("first=%+v second=%+v", firstStored, secondStored)
	}
}

func TestAdjustmentRequestContract_ProcessingTaskWithoutCurrentOperationRejectsSubmit(t *testing.T) {
	t.Run("submit preflight", func(t *testing.T) {
		initTestDB(t)
		instance := createAdjustmentTestInstance(t, "ins-submit-processing-status")
		createAdjustmentTaskForTest(t, &instance, adjustmentStatusProcessing, adjustmentTypeInstanceType, adjustmentPhasePolling,
			model.InstanceAdjustmentPayload{TargetInstanceType: "Ai2.MEDIUM4"})
		factoryCalled := false
		factory := func(context.Context) (instanceAdjustmentCloudGateway, error) {
			factoryCalled = true
			return newHappyAdjustmentCloud(instance.InstanceId), nil
		}
		body := adjustmentContractBody([]uint{instance.ID}, adjustmentTypeInstanceType, "Ai2.LARGE8")
		rr := httptest.NewRecorder()
		handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var response instanceAdjustmentSubmitResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.RejectedCount != 1 || response.Results[0].ReasonCode != reasonOperationInProgress || factoryCalled {
			t.Fatalf("response=%+v factoryCalled=%v", response, factoryCalled)
		}
		stored := reloadAdjustmentWorkerInstance(t, instance.ID)
		if stored.Task.Status != adjustmentStatusProcessing || stored.Instance.CurrentOperation != model.OpNone ||
			stored.Payload.TargetInstanceType != "Ai2.MEDIUM4" {
			t.Fatalf("processing row overwritten: %+v", stored)
		}
	})

	t.Run("accept CAS", func(t *testing.T) {
		initTestDB(t)
		instance := createAdjustmentTestInstance(t, "ins-accept-processing-status")
		createAdjustmentTaskForTest(t, &instance, adjustmentStatusProcessing, adjustmentTypeInstanceType, adjustmentPhasePolling,
			model.InstanceAdjustmentPayload{TargetInstanceType: "Ai2.MEDIUM4"})
		target := "Ai2.LARGE8"
		req := instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}
		cloud := newHappyAdjustmentCloud(instance.InstanceId)
		result := instanceAdjustmentResult{
			ID:            instance.ID,
			instance:      &instance,
			CurrentStatus: model.StatusRunning,
			cloud:         cloud.instances[instance.InstanceId],
			operation: adjustmentOperation{
				Type:               adjustmentTypeInstanceType,
				InstanceID:         instance.InstanceId,
				TargetInstanceType: target,
				ForceStop:          true,
			},
		}
		accepted, err := acceptInstanceAdjustment(context.Background(), req, &result)
		if err != nil || accepted {
			t.Fatalf("accepted=%v err=%v", accepted, err)
		}
		stored := reloadAdjustmentWorkerInstance(t, instance.ID)
		if stored.Task.Status != adjustmentStatusProcessing || stored.Instance.CurrentOperation != model.OpNone {
			t.Fatalf("processing row overwritten: %+v", stored)
		}
	})
}

func TestAdjustmentRequestContract_DerivesStopBehaviorWithoutConfirmation(t *testing.T) {
	t.Run("legacy confirmation flag is rejected", func(t *testing.T) {
		initTestDB(t)
		instance := createAdjustmentTestInstance(t, "ins-legacy-force-stop")
		body := adjustmentContractBody([]uint{instance.ID}, adjustmentTypeInstanceType, "Ai2.LARGE8")
		body["force_stop_confirmed"] = true
		factoryCalled := false
		factory := func(context.Context) (instanceAdjustmentCloudGateway, error) {
			factoryCalled = true
			return newHappyAdjustmentCloud(instance.InstanceId), nil
		}
		rr := httptest.NewRecorder()
		handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
		if rr.Code != http.StatusBadRequest || factoryCalled {
			t.Fatalf("status=%d body=%s factoryCalled=%v", rr.Code, rr.Body.String(), factoryCalled)
		}
	})

	cases := []struct {
		name           string
		adjustmentType string
		state          string
		resizeMode     string
	}{
		{name: "running instance type is accepted", adjustmentType: adjustmentTypeInstanceType, state: "RUNNING"},
		{name: "stopped instance type is accepted", adjustmentType: adjustmentTypeInstanceType, state: "STOPPED"},
		{name: "running online disk is accepted", adjustmentType: adjustmentTypeSystemDisk, state: "RUNNING", resizeMode: adjustmentResizeOnline},
		{name: "running offline disk is accepted", adjustmentType: adjustmentTypeSystemDisk, state: "RUNNING", resizeMode: adjustmentResizeOffline},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			initTestDB(t)
			instance := createAdjustmentTestInstance(t, "ins-derived-stop")
			cloud := newHappyAdjustmentCloud(instance.InstanceId)
			cloud.instances[instance.InstanceId].State = test.state
			if test.state == "STOPPED" {
				instance.LastKnownStatus = model.StatusStopped
				if err := model.DB(context.Background()).Save(&instance).Error; err != nil {
					t.Fatalf("save stopped instance: %v", err)
				}
			}
			body := adjustmentContractBody([]uint{instance.ID}, test.adjustmentType, "Ai2.LARGE8")
			if test.adjustmentType == adjustmentTypeSystemDisk {
				body["target_system_disk_size"] = int64(60)
				body["resize_mode"] = test.resizeMode
			}
			factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
			rr := httptest.NewRecorder()
			handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var response instanceAdjustmentSubmitResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if response.AcceptedCount != 1 || response.RejectedCount != 0 || !response.Results[0].Accepted {
				t.Fatalf("response=%+v", response)
			}
		})
	}
}

func TestAdjustmentRequestContract_SameTargetIdempotenceAndConflict(t *testing.T) {
	target := "Ai2.LARGE8"
	other := "Ai2.2XLARGE16"
	instance := model.Instance{CurrentOperation: model.OpAdjustInstanceType}
	adjustment := model.InstanceAdjustment{Status: adjustmentStatusProcessing}
	if err := adjustment.SetPayload(model.InstanceAdjustmentPayload{TargetInstanceType: target}); err != nil {
		t.Fatalf("encode instance-type payload: %v", err)
	}
	if !sameAdjustmentRequest(&instance, &adjustment, instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}) {
		t.Fatal("same instance-type target was not idempotent")
	}
	if sameAdjustmentRequest(&instance, &adjustment, instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &other}) {
		t.Fatal("different instance-type target was treated as idempotent")
	}
	instance.CurrentOperation = model.OpReboot
	if sameAdjustmentRequest(&instance, &adjustment, instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}) {
		t.Fatal("ordinary operation was treated as idempotent adjustment")
	}

	size := int64(100)
	instance = model.Instance{CurrentOperation: model.OpAdjustSystemDisk}
	adjustment = model.InstanceAdjustment{Status: adjustmentStatusProcessing}
	if err := adjustment.SetPayload(model.InstanceAdjustmentPayload{TargetDiskSize: size, ResizeMode: adjustmentResizeOnline}); err != nil {
		t.Fatalf("encode disk payload: %v", err)
	}
	if !sameAdjustmentRequest(&instance, &adjustment, instanceAdjustmentRequest{AdjustmentType: adjustmentTypeSystemDisk, TargetSystemDiskSize: &size, ResizeMode: adjustmentResizeOnline}) {
		t.Fatal("same disk target/mode was not idempotent")
	}
	if sameAdjustmentRequest(&instance, &adjustment, instanceAdjustmentRequest{AdjustmentType: adjustmentTypeSystemDisk, TargetSystemDiskSize: &size, ResizeMode: adjustmentResizeOffline}) {
		t.Fatal("different disk mode was treated as idempotent")
	}
}

func TestAdjustmentRequestContract_ConcurrentAcceptanceUsesSingleCASWinner(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-concurrent-adjustment")
	target := "Ai2.LARGE8"
	req := instanceAdjustmentRequest{AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target}
	cloud := newHappyAdjustmentCloud(instance.InstanceId).instances[instance.InstanceId]
	base := instanceAdjustmentResult{
		ID: instance.ID, instance: &instance, cloud: cloud, CurrentStatus: model.StatusRunning,
		CurrentSystemDiskType: "CLOUD_BSSD", CurrentSystemDiskSize: 50,
		operation: adjustmentOperation{Type: adjustmentTypeInstanceType, InstanceID: instance.InstanceId, TargetInstanceType: target, ForceStop: true},
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners atomic.Int64
	var failures atomic.Int64
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			accepted, err := acceptInstanceAdjustment(context.Background(), req, &base)
			if err != nil {
				failures.Add(1)
				return
			}
			if accepted {
				winners.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if failures.Load() != 0 || winners.Load() != 1 {
		t.Fatalf("winners=%d failures=%d", winners.Load(), failures.Load())
	}
	stored := reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Instance.CurrentOperation != model.OpAdjustInstanceType || stored.Task.Phase != adjustmentPhaseQueued ||
		stored.Payload.TargetInstanceType != target {
		t.Fatalf("stored adjustment=%+v", stored)
	}
}

func TestAdjustmentSurfaceContract_ObservedStateAndActionHiding(t *testing.T) {
	for _, test := range []struct {
		original string
		cloud    string
		want     string
	}{
		{original: "RUNNING", cloud: "STOPPED", want: model.StatusStopped},
		{original: "RUNNING", cloud: "STOPPING", want: model.StatusRunning},
		{original: "STOPPED", cloud: "RUNNING", want: model.StatusRunning},
		{original: "STOPPED", cloud: "STARTING", want: model.StatusStopped},
	} {
		t.Run(test.original+"_"+test.cloud, func(t *testing.T) {
			instance := model.Instance{
				Source: model.InstanceSourceCVM, LastKnownStatus: test.want, LastStableState: strings.ToLower(test.original),
				CurrentOperation: model.OpAdjustInstanceType, CurrentOperationState: model.OpStateProcessing,
			}
			status := ResolveInstanceStatus(context.Background(), &instance, &CVMInstanceInfo{State: test.cloud}, nil)
			if status.Status != test.want || len(status.Actions) != 0 || status.Transient {
				t.Fatalf("status=%+v want=%s", status, test.want)
			}
			response := buildAdminInstanceFromCache(context.Background(), adminInstanceItem{
				Instance:   instance,
				Adjustment: &model.InstanceAdjustment{Status: adjustmentStatusProcessing},
			})
			if response.Status != test.want || response.AdjustmentStatus != adjustmentStatusProcessing || len(response.Actions) != 0 {
				t.Fatalf("admin response=%+v", response)
			}
		})
	}
}

func TestAdjustmentSurfaceContract_FailureClearsOnlyOnAcceptedWrite(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-failed-adjustment-clear")
	task := createAdjustmentTaskForTest(t, &instance, adjustmentStatusFailed, adjustmentTypeInstanceType, adjustmentPhaseRestoreFailure,
		model.InstanceAdjustmentPayload{})
	if err := model.DB(context.Background()).Model(&task).Update("error_code", reasonInsufficientBalance).Error; err != nil {
		t.Fatalf("save failure: %v", err)
	}
	cloud := newHappyAdjustmentCloud(instance.InstanceId)
	var factoryCalls int
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) {
		factoryCalls++
		return cloud, nil
	}

	rr := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/admin/instances/adjust-config", bytes.NewBufferString(`{"ids":[1],"unknown":true}`))
	badReq.Header.Set("Authorization", "Bearer "+AdminToken)
	handleAdminInstanceAdjustment(rr, badReq, factory)
	if rr.Code != http.StatusBadRequest || factoryCalls != 0 {
		t.Fatalf("bad request status=%d factoryCalls=%d", rr.Code, factoryCalls)
	}
	stored := reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonInsufficientBalance {
		t.Fatalf("parameter rejection cleared failure: %+v", stored)
	}

	body := adjustmentContractBody([]uint{instance.ID}, adjustmentTypeInstanceType, "Ai2.LARGE8")
	rr = httptest.NewRecorder()
	handleAdminInstanceAdjustment(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config", body), factory)
	if rr.Code != http.StatusOK {
		t.Fatalf("accepted write status=%d body=%s", rr.Code, rr.Body.String())
	}
	stored = reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Task.Status != adjustmentStatusProcessing || stored.Task.ErrorCode != "" || stored.Instance.CurrentOperation != model.OpAdjustInstanceType {
		t.Fatalf("accepted write did not atomically replace failure: %+v", stored)
	}
}

func adjustmentSurfacePointer[T any](value T) *T { return &value }

func TestAdjustmentSurfaceContract_ResourceCacheBackfillAndRaceProtection(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-resource-reconcile")
	oldRound := time.Now().Add(-2 * time.Minute)
	instance.CVMInstanceType = "Ai2.MEDIUM4"
	instance.CVMCPU = 2
	instance.CVMMemoryGB = 4
	instance.SystemDiskType = "CLOUD_BSSD"
	instance.SystemDiskSize = 50
	instance.CVMPublicIP = "198.51.100.1"
	instance.CVMInternetChargeType = "BANDWIDTH_POSTPAID_BY_HOUR"
	instance.CVMInternetMaxBandwidthOut = 10
	instance.StatusSyncedAt = &oldRound
	if err := model.DB(context.Background()).Save(&instance).Error; err != nil {
		t.Fatalf("save cache: %v", err)
	}

	newRound := time.Now()
	model.BatchUpdateInstanceStatusCache(context.Background(), []model.InstanceStatusCacheItem{{
		ID: instance.ID, Status: model.StatusRunning,
		CVMInstanceType: adjustmentSurfacePointer("Ai2.LARGE8"), CVMCPU: adjustmentSurfacePointer(int64(4)), CVMMemoryGB: adjustmentSurfacePointer(int64(8)),
		SystemDiskType: adjustmentSurfacePointer("CLOUD_SSD"), SystemDiskSize: adjustmentSurfacePointer(int64(100)),
		CVMPublicIP: adjustmentSurfacePointer("203.0.113.10"), CVMInternetChargeType: adjustmentSurfacePointer("TRAFFIC_POSTPAID_BY_HOUR"),
		CVMInternetMaxBandwidthOut: adjustmentSurfacePointer(int64(100)),
	}}, newRound)
	stored := reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Instance.CVMInstanceType != "Ai2.LARGE8" || stored.Instance.CVMCPU != 4 || stored.Instance.CVMMemoryGB != 8 ||
		stored.Instance.SystemDiskType != "CLOUD_SSD" || stored.Instance.SystemDiskSize != 100 ||
		stored.Instance.CVMPublicIP != "203.0.113.10" || stored.Instance.CVMInternetChargeType != "TRAFFIC_POSTPAID_BY_HOUR" ||
		stored.Instance.CVMInternetMaxBandwidthOut != 100 || !stored.Instance.StatusSyncedAt.Equal(newRound) {
		t.Fatalf("resource backfill=%+v", stored)
	}

	staleRound := newRound.Add(-time.Minute)
	model.BatchUpdateInstanceStatusCache(context.Background(), []model.InstanceStatusCacheItem{{
		ID: instance.ID, Status: model.StatusStopped,
		CVMInstanceType: adjustmentSurfacePointer("Ai2.MEDIUM2"), SystemDiskSize: adjustmentSurfacePointer(int64(40)),
		CVMPublicIP: adjustmentSurfacePointer("192.0.2.1"), CVMInternetMaxBandwidthOut: adjustmentSurfacePointer(int64(20)),
	}}, staleRound)
	stored = reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Instance.CVMInstanceType != "Ai2.LARGE8" || stored.Instance.SystemDiskSize != 100 ||
		stored.Instance.CVMPublicIP != "203.0.113.10" || stored.Instance.CVMInternetMaxBandwidthOut != 100 ||
		!stored.Instance.StatusSyncedAt.Equal(newRound) {
		t.Fatalf("stale reconcile overwrote worker cache: %+v", stored)
	}

	apiErrorRound := newRound.Add(time.Minute)
	model.BatchUpdateInstanceStatusCache(context.Background(), []model.InstanceStatusCacheItem{{ID: instance.ID, Status: cvmAPIErrorState}}, apiErrorRound)
	stored = reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Instance.CVMInstanceType != "Ai2.LARGE8" || stored.Instance.SystemDiskSize != 100 ||
		stored.Instance.CVMPublicIP != "203.0.113.10" || stored.Instance.CVMInternetMaxBandwidthOut != 100 ||
		!stored.Instance.StatusSyncedAt.Equal(apiErrorRound) {
		t.Fatalf("API_ERROR resource-less update changed cache: %+v", stored)
	}
}

func TestAdjustmentSurfaceContract_ResourceFieldsFiltersPaginationAndStats(t *testing.T) {
	initTestDB(t)
	query := url.Values{}
	query.Set("cvm_instance_type", "Ai2.MEDIUM4, Ai2.LARGE8")
	query.Set("system_disk_size", "50,100")
	var filter adminQueryFilter
	if err := parseAdminInstanceResourceFilters(query, &filter); err != nil {
		t.Fatalf("parse filters: %v", err)
	}
	if len(filter.CVMInstanceTypes) != 2 || len(filter.SystemDiskSizes) != 2 {
		t.Fatalf("filter=%+v", filter)
	}
	invalidQueries := []url.Values{
		{"system_disk_size": []string{"50,nope"}},
		{"system_disk_size_lt": []string{"0"}},
		{"system_disk_size_gt": []string{"nope"}},
		{"system_disk_size": []string{"50"}, "system_disk_size_lt": []string{"100"}},
		{"system_disk_size_lt": []string{"50"}, "system_disk_size_gt": []string{"100"}},
	}
	for _, invalid := range invalidQueries {
		if err := parseAdminInstanceResourceFilters(invalid, &adminQueryFilter{}); err == nil {
			t.Fatalf("invalid disk size filter accepted: %v", invalid)
		}
	}

	instances := []model.Instance{
		{Name: "resource-a", InstanceId: "ins-resource-a", Source: model.InstanceSourceCVM, CVMInstanceType: "Ai2.MEDIUM4", SystemDiskSize: 50, LastKnownStatus: model.StatusRunning},
		{Name: "resource-b", InstanceId: "ins-resource-b", Source: model.InstanceSourceCVM, CVMInstanceType: "Ai2.LARGE8", SystemDiskSize: 100, LastKnownStatus: model.StatusStopped},
		{Name: "resource-c", InstanceId: "ins-resource-c", Source: model.InstanceSourceCVM, CVMInstanceType: "Ai2.MEDIUM2", SystemDiskSize: 100, LastKnownStatus: model.StatusRunning},
	}
	if err := model.DB(context.Background()).Create(&instances).Error; err != nil {
		t.Fatalf("create resource instances: %v", err)
	}
	assertFilter := func(t *testing.T, query url.Values, wantTotal, wantRunning, wantStopped int) {
		t.Helper()
		var parsed adminQueryFilter
		if err := parseAdminInstanceResourceFilters(query, &parsed); err != nil {
			t.Fatalf("parse %v: %v", query, err)
		}
		var count int64
		if err := whereBuilderForCache(context.Background(), parsed).Count(&count).Error; err != nil {
			t.Fatalf("count %v: %v", query, err)
		}
		items, total := queryInstancesPageFromCache(context.Background(), 1, 10, parsed, "")
		stats := queryAdminStatsFromCache(context.Background(), parsed)
		if count != int64(wantTotal) || total != int64(wantTotal) || len(items) != wantTotal ||
			stats.Total != wantTotal || stats.Running != wantRunning || stats.Stopped != wantStopped {
			t.Fatalf("query=%v count=%d total=%d items=%d stats=%+v", query, count, total, len(items), stats)
		}
	}
	assertFilter(t, query, 2, 1, 1)
	assertFilter(t, url.Values{"system_disk_size": []string{"100"}}, 2, 1, 1)
	assertFilter(t, url.Values{"system_disk_size_lt": []string{"100"}}, 1, 1, 0)
	assertFilter(t, url.Values{"system_disk_size_lt": []string{"50"}}, 0, 0, 0)
	assertFilter(t, url.Values{"system_disk_size_gt": []string{"50"}}, 2, 1, 1)
	assertFilter(t, url.Values{"system_disk_size_gt": []string{"100"}}, 0, 0, 0)

	local := model.Instance{Name: "local-resource", InstanceId: "local-resource", Source: model.InstanceSourceLocal, LastKnownStatus: model.StatusRunning, CVMInstanceType: "must-not-leak", SystemDiskSize: 999}
	encoded, err := json.Marshal(buildAdminInstanceFromCache(context.Background(), adminInstanceItem{Instance: local}))
	if err != nil {
		t.Fatalf("marshal local response: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("decode local response: %v", err)
	}
	for _, field := range []string{"cvm_instance_type", "cpu", "memory_gb", "system_disk_type", "system_disk_size", "public_ip", "internet_charge_type", "internet_max_bandwidth_out", "adjustment_status"} {
		if _, exists := object[field]; exists {
			t.Errorf("local response leaked %s: %s", field, encoded)
		}
	}
}

func TestAdjustmentSurfaceContract_StatusCompatibilityAndReadOnlyFailurePreservation(t *testing.T) {
	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-status-contract")
	task := createAdjustmentTaskForTest(t, &instance, adjustmentStatusFailed, adjustmentTypeSystemDisk, adjustmentPhaseRestoreFailure,
		model.InstanceAdjustmentPayload{})
	if err := model.DB(context.Background()).Model(&task).Update("error_code", reasonResourceSoldOut).Error; err != nil {
		t.Fatalf("save failed adjustment: %v", err)
	}
	fetchCalls := 0
	fetcher := func(context.Context, string) (*CVMInstanceInfo, error) {
		fetchCalls++
		return &CVMInstanceInfo{State: "RUNNING", InstanceType: "Ai2.LARGE8", CPU: 4, MemoryGB: 8, SystemDiskType: "CLOUD_BSSD", SystemDiskSize: 100, PublicIP: "203.0.113.10", InternetChargeType: "TRAFFIC_POSTPAID_BY_HOUR", InternetMaxBandwidthOut: 100}, nil
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/instances/status?id="+strconv.FormatUint(uint64(instance.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr := httptest.NewRecorder()
	handleAdminInstanceStatus(rr, req, fetcher)
	if rr.Code != http.StatusOK || fetchCalls != 1 {
		t.Fatalf("status=%d fetchCalls=%d body=%s", rr.Code, fetchCalls, rr.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["state"] != "RUNNING" ||
		response["cvm_instance_type"] != "Ai2.LARGE8" ||
		response["system_disk_size"] != float64(100) ||
		response["public_ip"] != "203.0.113.10" ||
		response["internet_charge_type"] != "TRAFFIC_POSTPAID_BY_HOUR" ||
		response["internet_max_bandwidth_out"] != float64(100) ||
		response["adjustment_error_code"] != reasonResourceSoldOut {
		t.Fatalf("response=%+v", response)
	}
	stored := reloadAdjustmentWorkerInstance(t, instance.ID)
	if stored.Task.Status != adjustmentStatusFailed || stored.Task.ErrorCode != reasonResourceSoldOut {
		t.Fatalf("read-only status cleared failure: %+v", stored)
	}

	local := model.Instance{Name: "local-status-contract", InstanceId: "local-status-contract", Source: model.InstanceSourceLocal, LastKnownStatus: model.StatusRunning}
	if err := model.DB(context.Background()).Create(&local).Error; err != nil {
		t.Fatalf("create local: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/admin/instances/status?id="+strconv.FormatUint(uint64(local.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr = httptest.NewRecorder()
	handleAdminInstanceStatus(rr, req, fetcher)
	if rr.Code == http.StatusOK || fetchCalls != 1 {
		t.Fatalf("local status=%d fetchCalls=%d body=%s", rr.Code, fetchCalls, rr.Body.String())
	}
}

func TestAdjustmentSurfaceContract_StableReasonI18n(t *testing.T) {
	reasons := []string{
		reasonInstanceNotFound, reasonCloudInstanceRequired, reasonDoctorNodeNotAllowed, reasonOperationInProgress,
		reasonInstanceStatusNotSupported, reasonCVMInstanceNotFound, reasonCVMRestricted, reasonCVMOperationInProgress,
		reasonCVMQueryFailed, reasonStopChargingNotSupported, reasonInvalidTarget, reasonUnsupportedInstanceType,
		reasonInstanceTypeNotUpgrade, reasonInstanceTypeUnchanged, reasonInstanceTypeDowngrade,
		reasonCloudDiskRequired, reasonSystemDiskTypeNotSupported, reasonTargetInstanceTypeUnavailable,
		reasonDiskQuotaUnavailable, reasonUnsupportedChargeType, reasonDiskNotReady, reasonCloudDiskUnavailable,
		reasonInstanceNetworkIncompatible, reasonInstanceResourceLimitExceeded, reasonResourceQuotaExceeded,
		reasonInstanceImageNotSupported, reasonInstanceFeatureNotSupported, reasonPromotionRestricted,
		reasonInvalidDiskSize, reasonDiskSizeUnchanged, reasonDiskShrinkNotSupported, reasonOnlineResizeNotSupported,
		reasonInsufficientBalance, reasonUnpaidOrder, reasonResourceSoldOut, reasonInternalError,
		reasonCloudAdjustmentFailed, reasonAdjustmentTimeout, reasonAdjustmentRestoreFailed,
	}
	zhCtx := context.WithValue(context.Background(), i18n.CtxKey{}, message.NewPrinter(language.Chinese))
	enCtx := context.WithValue(context.Background(), i18n.CtxKey{}, message.NewPrinter(language.English))
	for _, reason := range reasons {
		key := adjustmentReasonKey(reason)
		zh := i18n.T(zhCtx, key)
		en := i18n.T(enCtx, key)
		if strings.TrimSpace(zh) == "" || strings.TrimSpace(en) == "" || zh == en || en == key.String() {
			t.Errorf("reason=%s key=%q zh=%q en=%q", reason, key.String(), zh, en)
		}
	}
	unknown := adjustmentReasonKey("future_unknown_reason")
	if unknown != i18n.MsgAdjustmentReasonCloudFailed {
		t.Fatalf("unknown reason key=%q", unknown.String())
	}
}

func TestAdjustmentSurfaceContract_SubmitOnlyAuditAndSecretSafeError(t *testing.T) {
	rule, ok := auditRules["/admin/instances/adjust-config"]
	if !ok || rule.Action != "instance_adjust_config" || rule.Resource != "instance" {
		t.Fatalf("submit audit rule=%+v ok=%v", rule, ok)
	}
	if _, ok := auditRules["/admin/instances/adjust-config/validate"]; ok {
		t.Fatal("side-effect-free validate endpoint must not write an audit record")
	}

	initTestDB(t)
	instance := createAdjustmentTestInstance(t, "ins-secret-safe-error")
	cloud := newHappyAdjustmentCloud(instance.InstanceId)
	cloud.inquiryErr = errors.New("credential secret-id=AKID-SHOULD-NOT-LEAK")
	factory := func(context.Context) (instanceAdjustmentCloudGateway, error) { return cloud, nil }
	body := adjustmentContractBody([]uint{instance.ID}, adjustmentTypeInstanceType, "Ai2.LARGE8")
	rr := httptest.NewRecorder()
	handleAdminInstanceAdjustmentValidate(rr, adjustmentContractRequest(t, "/admin/instances/adjust-config/validate", body), factory)
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), "AKID-SHOULD-NOT-LEAK") || !strings.Contains(rr.Body.String(), reasonCloudAdjustmentFailed) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
