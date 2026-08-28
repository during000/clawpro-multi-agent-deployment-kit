package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

type contractAdjustmentCloud struct {
	*fakeAdjustmentCloud
	describeInstancesErr error
	describeDisksErr     error
	availabilityErr      error
	quotaByDisk          map[string]adjustmentDiskQuota
	quotaErrByDisk       map[string]error
	availabilityCalls    int
	quotaCalls           int
	deniedCalls          int
}

func (f *contractAdjustmentCloud) DescribeInstances(ctx context.Context, ids []string) (map[string]*adjustmentCloudInstance, error) {
	if f.describeInstancesErr != nil {
		return nil, f.describeInstancesErr
	}
	return f.fakeAdjustmentCloud.DescribeInstances(ctx, ids)
}

func (f *contractAdjustmentCloud) DescribeDisks(ctx context.Context, ids []string) (map[string]*adjustmentCloudDisk, error) {
	if f.describeDisksErr != nil {
		return nil, f.describeDisksErr
	}
	return f.fakeAdjustmentCloud.DescribeDisks(ctx, ids)
}

func (f *contractAdjustmentCloud) CheckInstanceTypeAvailable(context.Context, *adjustmentCloudInstance, string) (bool, error) {
	f.availabilityCalls++
	return f.available, f.availabilityErr
}

func (f *contractAdjustmentCloud) GetSystemDiskQuota(_ context.Context, _ *adjustmentCloudInstance, disk *adjustmentCloudDisk) (adjustmentDiskQuota, error) {
	f.quotaCalls++
	if err := f.quotaErrByDisk[disk.DiskID]; err != nil {
		return adjustmentDiskQuota{}, err
	}
	if quota, ok := f.quotaByDisk[disk.DiskID]; ok {
		return quota, nil
	}
	return f.quota, nil
}

func (f *contractAdjustmentCloud) DeniedActions(ctx context.Context, instanceID string, actions []string) ([]deniedAction, error) {
	f.deniedCalls++
	return f.fakeAdjustmentCloud.DeniedActions(ctx, instanceID, actions)
}

func newContractAdjustmentCloud(instanceID string) *contractAdjustmentCloud {
	return &contractAdjustmentCloud{
		fakeAdjustmentCloud: newHappyAdjustmentCloud(instanceID),
		quotaByDisk:         map[string]adjustmentDiskQuota{},
		quotaErrByDisk:      map[string]error{},
	}
}

func validateInstanceTypeContract(t *testing.T, instance model.Instance, cloud instanceAdjustmentCloudGateway, target string) instanceAdjustmentResult {
	t.Helper()
	results, err := validateResolvedAdjustmentTargets(context.Background(), instanceAdjustmentRequest{
		AdjustmentType: adjustmentTypeInstanceType, TargetInstanceType: &target,
	}, []resolvedAdjustmentTarget{{Instance: &instance}}, cloud, false)
	if err != nil {
		t.Fatalf("validate instance type: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%+v", results)
	}
	return results[0]
}

func validateDiskContract(t *testing.T, instance model.Instance, cloud instanceAdjustmentCloudGateway, target int64, mode string) instanceAdjustmentResult {
	t.Helper()
	results, err := validateResolvedAdjustmentTargets(context.Background(), instanceAdjustmentRequest{
		AdjustmentType: adjustmentTypeSystemDisk, TargetSystemDiskSize: &target, ResizeMode: mode,
	}, []resolvedAdjustmentTarget{{Instance: &instance}}, cloud, false)
	if err != nil {
		t.Fatalf("validate system disk: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%+v", results)
	}
	return results[0]
}

func adjustmentCloudContractInstance(instanceID string) model.Instance {
	return model.Instance{
		InstanceId: instanceID, Source: model.InstanceSourceCVM,
		LastKnownStatus: model.StatusRunning, LastStableState: model.StatusRunning,
	}
}
func TestAdjustmentCloudContract_DistinguishesInstanceTypeDirection(t *testing.T) {
	instanceID := "ins-instance-type-direction"
	instance := adjustmentCloudContractInstance(instanceID)
	cases := []struct {
		name        string
		target      string
		wantCode    string
		wantMessage string
	}{
		{name: "unchanged", target: "Ai2.MEDIUM4", wantCode: reasonInstanceTypeUnchanged, wantMessage: "已是目标规格"},
		{name: "downgrade", target: "Ai2.MEDIUM2", wantCode: reasonInstanceTypeDowngrade, wantMessage: "不支持降配"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := newContractAdjustmentCloud(instanceID)
			result := validateInstanceTypeContract(t, instance, cloud, test.target)
			if result.ReasonCode != test.wantCode || result.ReasonMessage != test.wantMessage {
				t.Fatalf("result=%+v want code=%q message=%q", result, test.wantCode, test.wantMessage)
			}
		})
	}
}

func TestAdjustmentCloudContract_RejectsUnsupportedRealtimeStates(t *testing.T) {
	cases := []struct {
		name string
		edit func(*adjustmentCloudInstance)
		want string
	}{
		{name: "running allowed", edit: func(*adjustmentCloudInstance) {}, want: ""},
		{name: "stopped allowed", edit: func(i *adjustmentCloudInstance) { i.State = "STOPPED" }, want: ""},
		{name: "transient state", edit: func(i *adjustmentCloudInstance) { i.State = "STOPPING" }, want: reasonInstanceStatusNotSupported},
		{name: "restricted", edit: func(i *adjustmentCloudInstance) { i.RestrictState = "EXPIRED" }, want: reasonCVMRestricted},
		{name: "cloud operation running", edit: func(i *adjustmentCloudInstance) { i.LatestOperationState = "OPERATING" }, want: reasonCVMOperationInProgress},
		{name: "stop charging", edit: func(i *adjustmentCloudInstance) { i.StopChargingMode = "STOP_CHARGING" }, want: reasonStopChargingNotSupported},
		{name: "unsupported charge", edit: func(i *adjustmentCloudInstance) { i.ChargeType = "CDHPAID" }, want: reasonUnsupportedChargeType},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instanceID := "ins-state-" + test.name
			instance := adjustmentCloudContractInstance(instanceID)
			cloud := newContractAdjustmentCloud(instanceID)
			test.edit(cloud.instances[instanceID])
			result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
			if result.ReasonCode != test.want {
				t.Fatalf("reason=%q want=%q result=%+v", result.ReasonCode, test.want, result)
			}
			if test.want == "" && !result.Adjustable {
				t.Fatalf("allowed state not adjustable: %+v", result)
			}
		})
	}
}

func TestAdjustmentCloudContract_ValidatesInstanceTypeDiskCompatibility(t *testing.T) {
	cases := []struct {
		name string
		edit func(*contractAdjustmentCloud, string)
		want string
	}{
		{name: "local system disk", edit: func(c *contractAdjustmentCloud, id string) { c.instances[id].SystemDiskType = "LOCAL_BASIC" }, want: reasonCloudDiskRequired},
		{name: "unsupported cloud system disk", edit: func(c *contractAdjustmentCloud, id string) {
			c.instances[id].SystemDiskType = "CLOUD_TSSD"
			c.disks["disk-system"].DiskType = "CLOUD_TSSD"
		}, want: reasonSystemDiskTypeNotSupported},
		{name: "local data disk", edit: func(c *contractAdjustmentCloud, id string) {
			c.instances[id].DataDisks = []adjustmentCloudDataDisk{{DiskID: "local-data", DiskType: "LOCAL_BASIC", DiskSize: 10}}
		}, want: reasonCloudDiskRequired},
		{name: "missing data disk fact", edit: func(c *contractAdjustmentCloud, id string) {
			c.instances[id].DataDisks = []adjustmentCloudDataDisk{{DiskID: "disk-data", DiskType: "CLOUD_BSSD", DiskSize: 10}}
		}, want: reasonDiskNotReady},
		{name: "detached data disk", edit: func(c *contractAdjustmentCloud, id string) {
			c.instances[id].DataDisks = []adjustmentCloudDataDisk{{DiskID: "disk-data", DiskType: "CLOUD_BSSD", DiskSize: 10}}
			c.disks["disk-data"] = &adjustmentCloudDisk{DiskID: "disk-data", DiskType: "CLOUD_BSSD", DiskUsage: "DATA_DISK", InstanceID: id, Attached: false, DiskState: "UNATTACHED"}
		}, want: reasonDiskNotReady},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instanceID := "ins-compat-" + test.name
			instance := adjustmentCloudContractInstance(instanceID)
			cloud := newContractAdjustmentCloud(instanceID)
			test.edit(cloud, instanceID)
			result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
			if result.ReasonCode != test.want || len(cloud.inquiries) != 0 {
				t.Fatalf("result=%+v inquiries=%d want=%s", result, len(cloud.inquiries), test.want)
			}
		})
	}
}

func TestAdjustmentCloudContract_ChecksAvailabilityAndMandatoryInquiry(t *testing.T) {
	instanceID := "ins-cloud-gates"
	instance := adjustmentCloudContractInstance(instanceID)

	t.Run("not sold", func(t *testing.T) {
		cloud := newContractAdjustmentCloud(instanceID)
		cloud.available = false
		result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
		if result.ReasonCode != reasonTargetInstanceTypeUnavailable || len(cloud.inquiries) != 0 {
			t.Fatalf("result=%+v inquiries=%d", result, len(cloud.inquiries))
		}
	})
	t.Run("availability lookup failure is conservative", func(t *testing.T) {
		cloud := newContractAdjustmentCloud(instanceID)
		cloud.availabilityErr = errors.New("SELL lookup failed")
		result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
		if result.ReasonCode != reasonTargetInstanceTypeUnavailable || len(cloud.inquiries) != 0 {
			t.Fatalf("result=%+v inquiries=%d", result, len(cloud.inquiries))
		}
	})
	t.Run("explicit denied action short circuits inquiry", func(t *testing.T) {
		cloud := newContractAdjustmentCloud(instanceID)
		cloud.denied = []deniedAction{{Action: "ResetInstancesType", Code: "VpcLimit", Message: "base network incompatible"}}
		result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
		if result.ReasonCode != reasonInstanceNetworkIncompatible || len(cloud.inquiries) != 0 {
			t.Fatalf("result=%+v inquiries=%d", result, len(cloud.inquiries))
		}
	})
	t.Run("denied action unavailable still requires inquiry", func(t *testing.T) {
		cloud := newContractAdjustmentCloud(instanceID)
		cloud.deniedErr = errors.New("unsupported common action")
		result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
		if !result.Adjustable || cloud.deniedCalls != 1 || len(cloud.inquiries) != 1 {
			t.Fatalf("result=%+v deniedCalls=%d inquiries=%d", result, cloud.deniedCalls, len(cloud.inquiries))
		}
	})
	t.Run("inquiry rejection is authoritative", func(t *testing.T) {
		cloud := newContractAdjustmentCloud(instanceID)
		cloud.inquiryErr = sdkerrors.NewTencentCloudSDKError("InvalidAccount.InsufficientBalance", "sensitive upstream message", "req-sensitive")
		result := validateInstanceTypeContract(t, instance, cloud, "Ai2.LARGE8")
		if result.ReasonCode != reasonInsufficientBalance || result.ReasonMessage == "sensitive upstream message" {
			t.Fatalf("result=%+v", result)
		}
	})
}

func addDiskContractInstance(cloud *contractAdjustmentCloud, instanceID, diskID, zone string) model.Instance {
	cloud.instances[instanceID] = &adjustmentCloudInstance{
		InstanceID: instanceID, State: "RUNNING", RestrictState: "NORMAL", ChargeType: "PREPAID", StopChargingMode: "NOT_APPLICABLE",
		InstanceType: "Ai2.MEDIUM4", CPU: 2, MemoryGB: 4, Zone: zone,
		SystemDiskID: diskID, SystemDiskType: "CLOUD_BSSD", SystemDiskSize: 50,
	}
	cloud.disks[diskID] = &adjustmentCloudDisk{
		DiskID: diskID, DiskType: "CLOUD_BSSD", DiskSize: 50, DiskUsage: "SYSTEM_DISK", DiskChargeType: "PREPAID",
		InstanceID: instanceID, Attached: true, DiskState: "ATTACHED",
	}
	return adjustmentCloudContractInstance(instanceID)
}

func TestAdjustmentCloudContract_CachesQuotaPerZoneAndIsolatesFailures(t *testing.T) {
	cloud := newContractAdjustmentCloud("ins-quota-a")
	cloud.instances = map[string]*adjustmentCloudInstance{}
	cloud.disks = map[string]*adjustmentCloudDisk{}
	a := addDiskContractInstance(cloud, "ins-quota-a", "disk-quota-a", "zone-a")
	b := addDiskContractInstance(cloud, "ins-quota-b", "disk-quota-b", "zone-a")
	c := addDiskContractInstance(cloud, "ins-quota-c", "disk-quota-c", "zone-b")
	cloud.quotaErrByDisk["disk-quota-a"] = errors.New("quota unavailable")
	target := int64(60)
	results, err := validateResolvedAdjustmentTargets(context.Background(), instanceAdjustmentRequest{
		AdjustmentType: adjustmentTypeSystemDisk, TargetSystemDiskSize: &target, ResizeMode: adjustmentResizeOnline,
	}, []resolvedAdjustmentTarget{{Instance: &a}, {Instance: &b}, {Instance: &c}}, cloud, false)
	if err != nil {
		t.Fatalf("validate batch: %v", err)
	}
	if cloud.quotaCalls != 2 {
		t.Fatalf("quotaCalls=%d want 2 for two cache keys", cloud.quotaCalls)
	}
	if results[0].ReasonCode != reasonDiskQuotaUnavailable || results[1].ReasonCode != reasonDiskQuotaUnavailable || !results[2].Adjustable {
		t.Fatalf("isolated quota results=%+v", results)
	}
}

func TestAdjustmentCloudContract_ValidatesDiskCapacityBoundaries(t *testing.T) {
	instanceID := "ins-disk-boundaries"
	instance := adjustmentCloudContractInstance(instanceID)
	cases := []struct {
		target      int64
		valid       bool
		wantCode    string
		wantMessage string
	}{
		{target: 49, wantCode: reasonDiskShrinkNotSupported, wantMessage: "不支持缩容"},
		{target: 50, wantCode: reasonDiskSizeUnchanged, wantMessage: "目标容量需大于当前系统盘容量"},
		{target: 55, wantCode: reasonInvalidDiskSize, wantMessage: "目标系统盘容量不符合扩容范围或步长"},
		{target: 60, valid: true},
		{target: 100, valid: true},
		{target: 101, wantCode: reasonInvalidDiskSize, wantMessage: "目标系统盘容量不符合扩容范围或步长"},
	}
	for _, test := range cases {
		t.Run(fmt.Sprintf("target-%d", test.target), func(t *testing.T) {
			cloud := newContractAdjustmentCloud(instanceID)
			cloud.quota = adjustmentDiskQuota{Available: true, MinSize: 50, MaxSize: 100, StepSize: 10}
			result := validateDiskContract(t, instance, cloud, test.target, adjustmentResizeOnline)
			if result.Adjustable != test.valid || result.ReasonCode != test.wantCode || result.ReasonMessage != test.wantMessage {
				t.Fatalf("result=%+v want valid=%v code=%q message=%q", result, test.valid, test.wantCode, test.wantMessage)
			}
			if result.MinDiskSize != 60 || result.MaxDiskSize != 100 || result.StepSize != 10 {
				t.Fatalf("bounds=%+v", result)
			}
		})
	}
}

func TestAdjustmentCloudContract_DerivesOnlineAndOfflineModes(t *testing.T) {
	instanceID := "ins-disk-modes"
	instance := adjustmentCloudContractInstance(instanceID)
	cases := []struct {
		name       string
		state      string
		mode       string
		wantOnline bool
		wantForce  bool
	}{
		{name: "running online", state: "RUNNING", mode: adjustmentResizeOnline, wantOnline: true},
		{name: "running offline", state: "RUNNING", mode: adjustmentResizeOffline, wantForce: true},
		{name: "stopped online request becomes offline", state: "STOPPED", mode: adjustmentResizeOnline},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := newContractAdjustmentCloud(instanceID)
			cloud.instances[instanceID].State = test.state
			result := validateDiskContract(t, instance, cloud, 60, test.mode)
			if !result.Adjustable || result.ReasonCode != "" {
				t.Fatalf("result=%+v", result)
			}
			if len(cloud.inquiries) != 0 {
				t.Fatalf("system disk validation unexpectedly used price inquiry: %+v", cloud.inquiries)
			}
			operation := result.operation
			if operation.ResizeOnline != test.wantOnline || operation.ForceStop != test.wantForce {
				t.Fatalf("operation=%+v", operation)
			}
		})
	}
}

func TestAdjustmentCloudContract_OnlineExecutionErrorMapping(t *testing.T) {
	err := sdkerrors.NewTencentCloudSDKError(
		"UnsupportedOperation.InstanceStateStopped",
		"online resize is unavailable",
		"request-online-unsupported",
	)
	online := adjustmentOperation{Type: adjustmentTypeSystemDisk, ResizeOnline: true}
	if got := mapAdjustmentExecutionError(online, err); got != reasonOnlineResizeNotSupported {
		t.Fatalf("online execution reason=%q", got)
	}
	offline := adjustmentOperation{Type: adjustmentTypeSystemDisk}
	if got := mapAdjustmentExecutionError(offline, err); got != reasonCloudAdjustmentFailed {
		t.Fatalf("offline execution reason=%q", got)
	}
}

func TestAdjustmentCloudContract_AppliesDeterministicDiskGates(t *testing.T) {
	cases := []struct {
		name string
		edit func(*contractAdjustmentCloud, string)
		want string
	}{
		{name: "missing", edit: func(c *contractAdjustmentCloud, _ string) { delete(c.disks, "disk-system") }, want: reasonDiskNotReady},
		{name: "wrong usage", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].DiskUsage = "DATA_DISK" }, want: reasonDiskNotReady},
		{name: "portable", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].Portable = true }, want: reasonDiskNotReady},
		{name: "not attached", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].Attached = false }, want: reasonDiskNotReady},
		{name: "wrong instance", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].InstanceID = "ins-other" }, want: reasonDiskNotReady},
		{name: "wrong state", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].DiskState = "UNATTACHED" }, want: reasonDiskNotReady},
		{name: "migrating", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].Migrating = true }, want: reasonDiskNotReady},
		{name: "rollbacking", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].Rollbacking = true }, want: reasonDiskNotReady},
		{name: "deadline error", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].DeadlineError = true }, want: reasonDiskNotReady},
		{name: "auto renew error", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].AutoRenewError = true }, want: reasonDiskNotReady},
		{name: "error prompt", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].ErrorPrompt = "billing blocked" }, want: reasonDiskNotReady},
		{name: "size drift", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].DiskSize = 60 }, want: reasonDiskNotReady},
		{name: "unsupported billing", edit: func(c *contractAdjustmentCloud, _ string) { c.disks["disk-system"].DiskChargeType = "CDHPAID" }, want: reasonUnsupportedChargeType},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instanceID := "ins-disk-gate"
			instance := adjustmentCloudContractInstance(instanceID)
			cloud := newContractAdjustmentCloud(instanceID)
			test.edit(cloud, instanceID)
			result := validateDiskContract(t, instance, cloud, 100, adjustmentResizeOnline)
			if result.ReasonCode != test.want || cloud.quotaCalls != 0 || len(cloud.inquiries) != 0 {
				t.Fatalf("result=%+v quotaCalls=%d inquiries=%d want=%s", result, cloud.quotaCalls, len(cloud.inquiries), test.want)
			}
		})
	}
}

func TestAdjustmentCloudContract_ClassifiesOfficialErrorFamilies(t *testing.T) {
	cases := []struct {
		name    string
		code    string
		message string
		want    string
	}{
		{name: "balance", code: "InvalidAccount.InsufficientBalance", want: reasonInsufficientBalance},
		{name: "unpaid order", code: "InvalidAccount.UnpaidOrder", want: reasonUnpaidOrder},
		{name: "sold out", code: "ResourcesSoldOut", want: reasonResourceSoldOut},
		{name: "base network", code: "InvalidInstance.BaseNetwork", want: reasonInstanceNetworkIncompatible},
		{name: "VPC", code: "InvalidVpcId.NotFound", want: reasonInstanceNetworkIncompatible},
		{name: "ENI", code: "LimitExceeded.Eni", want: reasonInstanceResourceLimitExceeded},
		{name: "EIP", code: "LimitExceeded.Eip", want: reasonInstanceResourceLimitExceeded},
		{name: "bandwidth", code: "LimitExceeded.Bandwidth", want: reasonInstanceResourceLimitExceeded},
		{name: "quota", code: "LimitExceeded.InstanceQuota", want: reasonResourceQuotaExceeded},
		{name: "image", code: "InvalidImageId.NotSupported", want: reasonInstanceImageNotSupported},
		{name: "RedHat", code: "InvalidInstance.RedHat", want: reasonInstanceImageNotSupported},
		{name: "ARM", code: "InvalidInstance.Arm", want: reasonInstanceFeatureNotSupported},
		{name: "heterogeneous", code: "InvalidInstance.Heterogeneous", want: reasonInstanceFeatureNotSupported},
		{name: "cross family", code: "InvalidInstance.Family", want: reasonInstanceFeatureNotSupported},
		{name: "swap", code: "InvalidInstance.Swap", want: reasonInstanceFeatureNotSupported},
		{name: "local disk", code: "InvalidInstance.LocalDisk", want: reasonInstanceFeatureNotSupported},
		{name: "special", code: "InvalidInstance.Special", want: reasonInstanceFeatureNotSupported},
		{name: "spot charge", code: "InvalidChargeType.Spot", want: reasonUnsupportedChargeType},
		{name: "promotion", code: "InvalidInstance.Promotion", want: reasonPromotionRestricted},
		{name: "application role", code: "InvalidInstance.ApplicationRole", want: reasonInstanceFeatureNotSupported},
		{name: "EMR", code: "InvalidInstance.Emr", want: reasonInstanceFeatureNotSupported},
		{name: "disk unavailable", code: "InvalidDisk.NotAvailable", want: reasonCloudDiskUnavailable},
		{name: "disk migrating", code: "InvalidDisk.Migrating", want: reasonDiskNotReady},
		{name: "disk rollback", code: "InvalidDisk.Rollbacking", want: reasonDiskNotReady},
		{name: "invalid instance state", code: "InvalidInstanceState", want: reasonCVMOperationInProgress},
		{name: "rescue", code: "InvalidInstance.Rescue", want: reasonCVMOperationInProgress},
		{name: "unknown sanitized", code: "InternalError", message: "secret upstream trace", want: reasonCloudAdjustmentFailed},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := sdkerrors.NewTencentCloudSDKError(test.code, test.message, "request-secret")
			if got := mapAdjustmentCloudError(err); got != test.want {
				t.Fatalf("mapAdjustmentCloudError(%s, %s)=%s want=%s", test.code, test.message, got, test.want)
			}
		})
	}
	deniedCases := []struct {
		action deniedAction
		want   string
	}{
		{action: deniedAction{Code: "LimitExceeded.InstanceQuota"}, want: reasonResourceQuotaExceeded},
		{action: deniedAction{Code: "LimitExceeded.Eni"}, want: reasonInstanceResourceLimitExceeded},
	}
	for _, test := range deniedCases {
		if got := deniedActionReason([]deniedAction{test.action}); got != test.want {
			t.Fatalf("deniedActionReason(%+v)=%s want=%s", test.action, got, test.want)
		}
	}
}

func newAdjustmentSDKContractGateway(t *testing.T) (*tencentInstanceAdjustmentGateway, map[string][]map[string]any) {
	t.Helper()
	requests := map[string][]map[string]any{}
	var mu sync.Mutex
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("X-TC-Action")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode %s request: %v", action, err)
		}
		mu.Lock()
		requests[action] = append(requests[action], body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "DescribeInstances":
			_, _ = w.Write([]byte(`{"Response":{"InstanceSet":[{"InstanceId":"ins-sdk-contract","InstanceState":"RUNNING","RestrictState":"NORMAL","InstanceChargeType":"PREPAID","StopChargingMode":"NOT_APPLICABLE","InstanceType":"Ai2.MEDIUM4","CPU":2,"Memory":4,"ImageId":"img-sdk","Placement":{"Zone":"ap-guangzhou-6"},"SystemDisk":{"DiskId":"disk-sdk-system","DiskType":"CLOUD_BSSD","DiskSize":50},"DataDisks":[{"DiskId":"disk-sdk-data","DiskType":"CLOUD_BSSD","DiskSize":20}],"LatestOperation":"ResetInstancesType","LatestOperationState":"SUCCESS","LatestOperationRequestId":"req-operation","LatestOperationErrorMsg":""}],"TotalCount":1,"RequestId":"req-describe-instance"}}`))
		case "DescribeDisks":
			_, _ = w.Write([]byte(`{"Response":{"DiskSet":[{"DiskId":"disk-sdk-system","DiskType":"CLOUD_BSSD","DiskSize":50,"DiskUsage":"SYSTEM_DISK","DiskChargeType":"PREPAID","InstanceId":"ins-sdk-contract","InstanceIdList":["ins-sdk-contract"],"Portable":false,"Attached":true,"DiskState":"ATTACHED","Migrating":false,"Rollbacking":false,"DeadlineError":false,"AutoRenewFlagError":false,"ErrorPrompt":""}],"TotalCount":1,"RequestId":"req-describe-disk"}}`))
		case "DescribeZoneInstanceConfigInfos":
			_, _ = w.Write([]byte(`{"Response":{"InstanceTypeQuotaSet":[{"Zone":"ap-guangzhou-6","InstanceType":"Ai2.LARGE8","InstanceChargeType":"PREPAID","Status":"SELL"}],"RequestId":"req-sell"}}`))
		case "DescribeDiskConfigQuota":
			_, _ = w.Write([]byte(`{"Response":{"DiskConfigSet":[{"Zone":"ap-guangzhou-6","InstanceFamily":"AI2","DiskType":"CLOUD_BSSD","DiskUsage":"SYSTEM_DISK","DiskChargeType":"PREPAID","Available":true,"MinDiskSize":50,"MaxDiskSize":500,"StepSize":10}],"RequestId":"req-quota"}}`))
		case "ResetInstancesType":
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-reset"}}`))
		case "ResizeInstanceDisks":
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-resize"}}`))
		default:
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-generic"}}`))
		}
	}))
	t.Cleanup(server.Close)

	credential := tccommon.NewCredential("fake-secret-id", "fake-secret-key")
	newProfile := func() *profile.ClientProfile {
		clientProfile := profile.NewClientProfile()
		clientProfile.HttpProfile.Endpoint = strings.TrimPrefix(server.URL, "https://")
		clientProfile.HttpProfile.ReqMethod = http.MethodPost
		clientProfile.HttpProfile.Scheme = "https"
		return clientProfile
	}
	cvmClient, err := cvm.NewClient(credential, "ap-guangzhou", newProfile())
	if err != nil {
		t.Fatalf("create CVM client: %v", err)
	}
	cvmClient.WithHttpTransport(server.Client().Transport)
	cbsClient, err := cbs.NewClient(credential, "ap-guangzhou", newProfile())
	if err != nil {
		t.Fatalf("create CBS client: %v", err)
	}
	cbsClient.WithHttpTransport(server.Client().Transport)
	return &tencentInstanceAdjustmentGateway{cvmClient: cvmClient, cbsClient: cbsClient}, requests
}

func adjustmentSDKRequest(t *testing.T, requests map[string][]map[string]any, action string) map[string]any {
	t.Helper()
	values := requests[action]
	if len(values) != 1 {
		t.Fatalf("%s requests=%+v", action, values)
	}
	return values[0]
}

func TestAdjustmentSDKContract_TypedGatewayRequestsAndResponses(t *testing.T) {
	gateway, requests := newAdjustmentSDKContractGateway(t)
	ctx := context.Background()

	instances, err := gateway.DescribeInstances(ctx, []string{"ins-sdk-contract"})
	if err != nil {
		t.Fatalf("DescribeInstances: %v", err)
	}
	instance := instances["ins-sdk-contract"]
	if instance == nil || instance.State != "RUNNING" || instance.InstanceType != "Ai2.MEDIUM4" || instance.CPU != 2 || instance.MemoryGB != 4 ||
		instance.Zone != "ap-guangzhou-6" || instance.SystemDiskID != "disk-sdk-system" || instance.SystemDiskSize != 50 || len(instance.DataDisks) != 1 ||
		instance.LatestOperationRequestID != "req-operation" {
		t.Fatalf("instance=%+v", instance)
	}
	if got := adjustmentSDKRequest(t, requests, "DescribeInstances")["InstanceIds"]; got == nil {
		t.Fatalf("DescribeInstances request=%+v", requests["DescribeInstances"])
	}

	disks, err := gateway.DescribeDisks(ctx, []string{"disk-sdk-system"})
	if err != nil {
		t.Fatalf("DescribeDisks: %v", err)
	}
	disk := disks["disk-sdk-system"]
	if disk == nil || disk.DiskUsage != "SYSTEM_DISK" || disk.DiskSize != 50 || !disk.Attached || disk.Portable || !diskBelongsToInstance(disk, instance.InstanceID) {
		t.Fatalf("disk=%+v", disk)
	}

	available, err := gateway.CheckInstanceTypeAvailable(ctx, instance, "Ai2.LARGE8")
	if err != nil || !available {
		t.Fatalf("available=%v err=%v", available, err)
	}
	quota, err := gateway.GetSystemDiskQuota(ctx, instance, disk)
	if err != nil || !quota.Available || quota.MinSize != 50 || quota.MaxSize != 500 || quota.StepSize != 10 {
		t.Fatalf("quota=%+v err=%v", quota, err)
	}

	instanceTypeOperation := adjustmentOperation{
		Type: adjustmentTypeInstanceType, InstanceID: instance.InstanceID, TargetInstanceType: "Ai2.LARGE8", ForceStop: true,
	}
	if err := gateway.InquiryInstanceType(ctx, instanceTypeOperation); err != nil {
		t.Fatalf("instance-type inquiry: %v", err)
	}
	requestID, err := gateway.Execute(ctx, instanceTypeOperation)
	if err != nil || requestID != "req-reset" {
		t.Fatalf("instance-type execute requestID=%q err=%v", requestID, err)
	}
	inquiryReset := adjustmentSDKRequest(t, requests, "InquiryPriceResetInstancesType")
	reset := adjustmentSDKRequest(t, requests, "ResetInstancesType")
	if inquiryReset["InstanceType"] != reset["InstanceType"] || inquiryReset["InstanceIds"] == nil || reset["InstanceIds"] == nil || reset["ForceStop"] != true {
		t.Fatalf("inquiry=%+v reset=%+v", inquiryReset, reset)
	}
	if _, exists := inquiryReset["ForceStop"]; exists {
		t.Fatalf("SDK inquiry unexpectedly accepted ForceStop: %+v", inquiryReset)
	}

	diskOperation := adjustmentOperation{
		Type: adjustmentTypeSystemDisk, InstanceID: instance.InstanceID, DiskID: disk.DiskID, TargetDiskSize: 100, ForceStop: false, ResizeOnline: true,
	}
	requestID, err = gateway.Execute(ctx, diskOperation)
	if err != nil || requestID != "req-resize" {
		t.Fatalf("disk execute requestID=%q err=%v", requestID, err)
	}
	if len(requests["InquiryPriceResizeInstanceDisks"]) != 0 {
		t.Fatalf("system disk validation unexpectedly called price inquiry: %+v", requests["InquiryPriceResizeInstanceDisks"])
	}
	resize := adjustmentSDKRequest(t, requests, "ResizeInstanceDisks")
	if resize["InstanceId"] != instance.InstanceID || resize["ForceStop"] != false || resize["ResizeOnline"] != true {
		t.Fatalf("resize request=%+v", resize)
	}
	writeDisk, writeOK := resize["SystemDisk"].(map[string]any)
	if !writeOK || writeDisk["DiskId"] != disk.DiskID || writeDisk["DiskSize"] != float64(100) {
		t.Fatalf("resize request=%+v", resize)
	}

	if err := gateway.StartInstance(ctx, instance.InstanceID); err != nil {
		t.Fatalf("StartInstance: %v", err)
	}
	if err := gateway.StopInstance(ctx, instance.InstanceID, "KEEP_CHARGING"); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}
	stop := adjustmentSDKRequest(t, requests, "StopInstances")
	if stop["StopType"] != "SOFT_FIRST" || stop["StoppedMode"] != "KEEP_CHARGING" {
		t.Fatalf("stop request=%+v", stop)
	}
}

func TestAdjustmentSDKContract_UnsupportedOperationFailsBeforeSDK(t *testing.T) {
	gateway, requests := newAdjustmentSDKContractGateway(t)
	if err := gateway.InquiryInstanceType(context.Background(), adjustmentOperation{Type: "future"}); err == nil {
		t.Fatal("unsupported inquiry accepted")
	}
	if _, err := gateway.Execute(context.Background(), adjustmentOperation{Type: "future"}); err == nil {
		t.Fatal("unsupported execute accepted")
	}
	if len(requests) != 0 {
		t.Fatalf("unsupported operations called SDK: %+v", requests)
	}
}

func TestAdjustmentGateContract_ActionQPSAndKeyIsolation(t *testing.T) {
	for _, test := range []struct {
		action string
		want   int
	}{
		{action: "ResetInstancesType", want: 8},
		{action: "InquiryPriceResetInstancesType", want: 8},
		{action: "ResizeInstanceDisks", want: 8},
		{action: "DescribeDiskConfigQuota", want: 15},
		{action: "DescribeDisks", want: 8},
		{action: "DescribeInstances", want: 8},
	} {
		if got := cloudActionQPS(test.action); got != test.want {
			t.Errorf("cloudActionQPS(%q)=%d want=%d", test.action, got, test.want)
		}
	}

	oldRegion := CVMRegion
	CVMRegion = "ap-guangzhou"
	t.Cleanup(func() { CVMRegion = oldRegion })
	ctxA := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Uin: "uin-a"})
	ctxB := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Uin: "uin-b"})
	key := cloudActionGateKey(ctxA, "ResetInstancesType")
	if key == "" || key != cloudActionGateKey(ctxA, "ResetInstancesType") {
		t.Fatalf("same UIN/region/action must produce a stable key: %q", key)
	}
	if key == cloudActionGateKey(ctxB, "ResetInstancesType") {
		t.Fatal("different UINs shared a cloud gate key")
	}
	if key == cloudActionGateKey(ctxA, "ResizeInstanceDisks") {
		t.Fatal("different actions shared a cloud gate key")
	}
	CVMRegion = "ap-shanghai"
	if key == cloudActionGateKey(ctxA, "ResetInstancesType") {
		t.Fatal("different regions shared a cloud gate key")
	}
}

func TestAdjustmentGateContract_NoBurstCadenceAndErrorPropagation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(model.UseDBForTestWithDriver(db, "sqlite"))

	for _, test := range []struct {
		action     string
		minimumAge time.Duration
		fnErr      error
	}{
		{action: "ResetInstancesType", minimumAge: 110 * time.Millisecond},
		{action: "DescribeDiskConfigQuota", minimumAge: 55 * time.Millisecond, fnErr: errors.New("quota failure")},
	} {
		t.Run(test.action, func(t *testing.T) {
			calls := 0
			started := time.Now()
			err := withCloudActionGate(context.Background(), test.action, func() error {
				calls++
				return test.fnErr
			})
			elapsed := time.Since(started)
			if calls != 1 || !errors.Is(err, test.fnErr) {
				t.Fatalf("calls=%d err=%v want=%v", calls, err, test.fnErr)
			}
			if elapsed < test.minimumAge {
				t.Fatalf("action %s returned after %s, want no-burst minimum >=%s", test.action, elapsed, test.minimumAge)
			}
		})
	}
}
