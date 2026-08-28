package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

var resetInstancesTypeSupportedDiskTypes = map[string]struct{}{
	"CLOUD_BASIC":   {},
	"CLOUD_PREMIUM": {},
	"CLOUD_SSD":     {},
	"CLOUD_BSSD":    {},
	"CLOUD_HSSD":    {}, // ResetInstancesType 实际支持，但公开文档未列出
}

type adjustmentCloudInstance struct {
	InstanceID                  string
	State                       string
	RestrictState               string
	ChargeType                  string
	StopChargingMode            string
	InstanceType                string
	CPU                         int64
	MemoryGB                    int64
	Zone                        string
	ImageID                     string
	SystemDiskID                string
	SystemDiskType              string
	SystemDiskSize              int64
	DataDisks                   []adjustmentCloudDataDisk
	LatestOperation             string
	LatestOperationState        string
	LatestOperationRequestID    string
	LatestOperationErrorMessage string
}

type adjustmentCloudDataDisk struct {
	DiskID   string
	DiskType string
	DiskSize int64
}

type adjustmentCloudDisk struct {
	DiskID         string
	DiskType       string
	DiskSize       int64
	DiskUsage      string
	DiskChargeType string
	InstanceID     string
	InstanceIDs    []string
	Portable       bool
	Attached       bool
	DiskState      string
	Migrating      bool
	Rollbacking    bool
	DeadlineError  bool
	AutoRenewError bool
	ErrorPrompt    string
}

type adjustmentDiskQuota struct {
	Available bool
	MinSize   int64
	MaxSize   int64
	StepSize  int64
}

type adjustmentOperation struct {
	Type               string
	InstanceID         string
	TargetInstanceType string
	DiskID             string
	TargetDiskSize     int64
	ForceStop          bool
	ResizeOnline       bool
}

type instanceAdjustmentCloudGateway interface {
	DescribeInstances(context.Context, []string) (map[string]*adjustmentCloudInstance, error)
	DescribeDisks(context.Context, []string) (map[string]*adjustmentCloudDisk, error)
	CheckInstanceTypeAvailable(context.Context, *adjustmentCloudInstance, string) (bool, error)
	GetSystemDiskQuota(context.Context, *adjustmentCloudInstance, *adjustmentCloudDisk) (adjustmentDiskQuota, error)
	DeniedActions(context.Context, string, []string) ([]deniedAction, error)
	InquiryInstanceType(context.Context, adjustmentOperation) error
	Execute(context.Context, adjustmentOperation) (string, error)
	StartInstance(context.Context, string) error
	StopInstance(context.Context, string, string) error
}

type tencentInstanceAdjustmentGateway struct {
	cvmClient *cvm.Client
	cbsClient *cbs.Client
}

func newTencentInstanceAdjustmentGateway(ctx context.Context) (*tencentInstanceAdjustmentGateway, error) {
	cvmClient, err := GetCVMClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cvm client: %w", err)
	}
	cbsClient, err := GetCBSClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cbs client: %w", err)
	}
	return &tencentInstanceAdjustmentGateway{cvmClient: cvmClient, cbsClient: cbsClient}, nil
}

func cloudActionQPS(action string) int {
	switch action {
	case "ResetInstancesType", "InquiryPriceResetInstancesType", "ResizeInstanceDisks":
		return 8
	case "DescribeDiskConfigQuota":
		return 15
	default:
		return 8
	}
}

func cloudActionGateKey(ctx context.Context, action string) string {
	hash := sha256.Sum256([]byte(hcommon.CVMUinFromCtx(ctx) + "|" + CVMRegion + "|" + action))
	return fmt.Sprintf("cloud-gate:%x", hash[:8])
}

func withCloudActionGate(ctx context.Context, action string, fn func() error) error {
	qps := cloudActionQPS(action)
	if qps <= 0 {
		qps = 1
	}
	lock, err := model.AcquireLock(ctx, cloudActionGateKey(ctx, action), 30*time.Second)
	if err != nil {
		return err
	}
	startedAt := time.Now()
	defer func() {
		minimumInterval := time.Second / time.Duration(qps)
		if remaining := minimumInterval - time.Since(startedAt); remaining > 0 {
			timer := time.NewTimer(remaining)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
			}
		}
		lock.Release()
	}()
	return fn()
}

func (g *tencentInstanceAdjustmentGateway) describeInstancesOnce(ctx context.Context, instanceIDs []string) (*cvm.DescribeInstancesResponse, error) {
	req := cvm.NewDescribeInstancesRequest()
	req.InstanceIds = tccommon.StringPtrs(instanceIDs)
	req.Limit = tccommon.Int64Ptr(int64(len(instanceIDs)))
	var resp *cvm.DescribeInstancesResponse
	err := withCloudActionGate(ctx, "DescribeInstances", func() error {
		return RetryCloudCall(ctx, func() error {
			var callErr error
			resp, callErr = CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.DescribeInstances)
			return callErr
		})
	})
	return resp, err
}

func (g *tencentInstanceAdjustmentGateway) DescribeInstances(ctx context.Context, instanceIDs []string) (map[string]*adjustmentCloudInstance, error) {
	result := make(map[string]*adjustmentCloudInstance, len(instanceIDs))
	for start := 0; start < len(instanceIDs); start += 100 {
		end := start + 100
		if end > len(instanceIDs) {
			end = len(instanceIDs)
		}
		batch := instanceIDs[start:end]
		resp, err := g.describeInstancesOnce(ctx, batch)
		if err != nil && isTencentInstanceNotFound(err) {
			for _, instanceID := range batch {
				singleResp, singleErr := g.describeInstancesOnce(ctx, []string{instanceID})
				if isTencentInstanceNotFound(singleErr) {
					continue
				}
				if singleErr != nil {
					return nil, singleErr
				}
				extractAdjustmentInstances(result, singleResp)
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		extractAdjustmentInstances(result, resp)
	}
	return result, nil
}

func extractAdjustmentInstances(dst map[string]*adjustmentCloudInstance, resp *cvm.DescribeInstancesResponse) {
	if resp == nil || resp.Response == nil {
		return
	}
	for _, item := range resp.Response.InstanceSet {
		if item == nil || item.InstanceId == nil {
			continue
		}
		info := &adjustmentCloudInstance{
			InstanceID:                  StrVal(item.InstanceId),
			State:                       StrVal(item.InstanceState),
			RestrictState:               StrVal(item.RestrictState),
			ChargeType:                  StrVal(item.InstanceChargeType),
			StopChargingMode:            StrVal(item.StopChargingMode),
			InstanceType:                StrVal(item.InstanceType),
			ImageID:                     StrVal(item.ImageId),
			LatestOperation:             StrVal(item.LatestOperation),
			LatestOperationState:        StrVal(item.LatestOperationState),
			LatestOperationRequestID:    StrVal(item.LatestOperationRequestId),
			LatestOperationErrorMessage: StrVal(item.LatestOperationErrorMsg),
		}
		if item.CPU != nil {
			info.CPU = *item.CPU
		}
		if item.Memory != nil {
			info.MemoryGB = *item.Memory
		}
		if item.Placement != nil {
			info.Zone = StrVal(item.Placement.Zone)
		}
		if item.SystemDisk != nil {
			info.SystemDiskID = StrVal(item.SystemDisk.DiskId)
			info.SystemDiskType = StrVal(item.SystemDisk.DiskType)
			if item.SystemDisk.DiskSize != nil {
				info.SystemDiskSize = *item.SystemDisk.DiskSize
			}
		}
		for _, disk := range item.DataDisks {
			if disk == nil {
				continue
			}
			dataDisk := adjustmentCloudDataDisk{DiskID: StrVal(disk.DiskId), DiskType: StrVal(disk.DiskType)}
			if disk.DiskSize != nil {
				dataDisk.DiskSize = *disk.DiskSize
			}
			info.DataDisks = append(info.DataDisks, dataDisk)
		}
		dst[info.InstanceID] = info
	}
}

func (g *tencentInstanceAdjustmentGateway) DescribeDisks(ctx context.Context, diskIDs []string) (map[string]*adjustmentCloudDisk, error) {
	result := make(map[string]*adjustmentCloudDisk, len(diskIDs))
	for start := 0; start < len(diskIDs); start += 100 {
		end := start + 100
		if end > len(diskIDs) {
			end = len(diskIDs)
		}
		req := cbs.NewDescribeDisksRequest()
		req.DiskIds = tccommon.StringPtrs(diskIDs[start:end])
		req.Limit = tccommon.Uint64Ptr(uint64(end - start))
		var resp *cbs.DescribeDisksResponse
		err := withCloudActionGate(ctx, "DescribeDisks", func() error {
			return RetryCloudCall(ctx, func() error {
				var callErr error
				resp, callErr = CallSDKAPITyped(ctx, SDKComponentCBS, req, g.cbsClient.DescribeDisks)
				return callErr
			})
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Response == nil {
			continue
		}
		for _, item := range resp.Response.DiskSet {
			if item == nil || item.DiskId == nil {
				continue
			}
			disk := &adjustmentCloudDisk{
				DiskID:         StrVal(item.DiskId),
				DiskType:       StrVal(item.DiskType),
				DiskUsage:      StrVal(item.DiskUsage),
				DiskChargeType: StrVal(item.DiskChargeType),
				InstanceID:     StrVal(item.InstanceId),
				InstanceIDs:    stringPtrValues(item.InstanceIdList),
				Portable:       boolValue(item.Portable),
				Attached:       boolValue(item.Attached),
				DiskState:      StrVal(item.DiskState),
				Migrating:      boolValue(item.Migrating),
				Rollbacking:    boolValue(item.Rollbacking),
				DeadlineError:  boolValue(item.DeadlineError),
				AutoRenewError: boolValue(item.AutoRenewFlagError),
				ErrorPrompt:    StrVal(item.ErrorPrompt),
			}
			if item.DiskSize != nil {
				disk.DiskSize = int64(*item.DiskSize)
			}
			result[disk.DiskID] = disk
		}
	}
	return result, nil
}

func (g *tencentInstanceAdjustmentGateway) CheckInstanceTypeAvailable(ctx context.Context, instance *adjustmentCloudInstance, target string) (bool, error) {
	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = []*cvm.Filter{
		{Name: tccommon.StringPtr("zone"), Values: tccommon.StringPtrs([]string{instance.Zone})},
		{Name: tccommon.StringPtr("instance-type"), Values: tccommon.StringPtrs([]string{target})},
		{Name: tccommon.StringPtr("instance-charge-type"), Values: tccommon.StringPtrs([]string{instance.ChargeType})},
	}
	var resp *cvm.DescribeZoneInstanceConfigInfosResponse
	err := withCloudActionGate(ctx, "DescribeZoneInstanceConfigInfos", func() error {
		return RetryCloudCall(ctx, func() error {
			var callErr error
			resp, callErr = CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.DescribeZoneInstanceConfigInfos)
			return callErr
		})
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Response == nil {
		return false, nil
	}
	for _, item := range resp.Response.InstanceTypeQuotaSet {
		if item != nil && StrVal(item.Zone) == instance.Zone && StrVal(item.InstanceType) == target &&
			StrVal(item.InstanceChargeType) == instance.ChargeType && StrVal(item.Status) == "SELL" {
			return true, nil
		}
	}
	return false, nil
}

func (g *tencentInstanceAdjustmentGateway) GetSystemDiskQuota(ctx context.Context, instance *adjustmentCloudInstance, disk *adjustmentCloudDisk) (adjustmentDiskQuota, error) {
	req := cbs.NewDescribeDiskConfigQuotaRequest()
	req.InquiryType = tccommon.StringPtr("INQUIRY_CVM_CONFIG")
	req.DiskChargeType = tccommon.StringPtr(disk.DiskChargeType)
	req.InstanceFamilies = tccommon.StringPtrs([]string{instanceFamily(instance.InstanceType)})
	req.DiskTypes = tccommon.StringPtrs([]string{disk.DiskType})
	req.Zones = tccommon.StringPtrs([]string{instance.Zone})
	req.Memory = tccommon.Uint64Ptr(uint64(instance.MemoryGB))
	req.CPU = tccommon.Uint64Ptr(uint64(instance.CPU))
	req.DiskUsage = tccommon.StringPtr("SYSTEM_DISK")
	var resp *cbs.DescribeDiskConfigQuotaResponse
	err := withCloudActionGate(ctx, "DescribeDiskConfigQuota", func() error {
		return RetryCloudCall(ctx, func() error {
			var callErr error
			resp, callErr = CallSDKAPITyped(ctx, SDKComponentCBS, req, g.cbsClient.DescribeDiskConfigQuota)
			return callErr
		})
	})
	if err != nil {
		return adjustmentDiskQuota{}, err
	}
	if resp == nil || resp.Response == nil {
		return adjustmentDiskQuota{}, nil
	}
	for _, item := range resp.Response.DiskConfigSet {
		if item == nil || StrVal(item.Zone) != instance.Zone || StrVal(item.InstanceFamily) != instanceFamily(instance.InstanceType) ||
			StrVal(item.DiskType) != disk.DiskType || StrVal(item.DiskUsage) != "SYSTEM_DISK" ||
			StrVal(item.DiskChargeType) != disk.DiskChargeType {
			continue
		}
		quota := adjustmentDiskQuota{Available: boolValue(item.Available), StepSize: 1}
		if item.MinDiskSize != nil {
			quota.MinSize = int64(*item.MinDiskSize)
		}
		if item.MaxDiskSize != nil {
			quota.MaxSize = int64(*item.MaxDiskSize)
		}
		if item.StepSize != nil && *item.StepSize > 0 {
			quota.StepSize = int64(*item.StepSize)
		}
		return quota, nil
	}
	return adjustmentDiskQuota{}, nil
}

func (g *tencentInstanceAdjustmentGateway) DeniedActions(ctx context.Context, instanceID string, actions []string) ([]deniedAction, error) {
	var result map[string][]deniedAction
	err := withCloudActionGate(ctx, "DescribeInstancesDeniedActions", func() error {
		var callErr error
		result, callErr = describeInstancesDeniedActions(ctx, []string{instanceID}, actions)
		return callErr
	})
	return result[instanceID], err
}

func (g *tencentInstanceAdjustmentGateway) InquiryInstanceType(ctx context.Context, operation adjustmentOperation) error {
	if operation.Type != adjustmentTypeInstanceType {
		return fmt.Errorf("unsupported instance-type inquiry for adjustment type %q", operation.Type)
	}
	req := cvm.NewInquiryPriceResetInstancesTypeRequest()
	req.InstanceIds = tccommon.StringPtrs([]string{operation.InstanceID})
	req.InstanceType = tccommon.StringPtr(operation.TargetInstanceType)
	return withCloudActionGate(ctx, "InquiryPriceResetInstancesType", func() error {
		return RetryCloudCall(ctx, func() error {
			_, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.InquiryPriceResetInstancesType)
			return err
		})
	})
}

func mapAdjustmentExecutionError(operation adjustmentOperation, err error) string {
	reason := mapAdjustmentCloudError(err)
	if operation.Type == adjustmentTypeSystemDisk && operation.ResizeOnline && reason == reasonCloudAdjustmentFailed {
		return reasonOnlineResizeNotSupported
	}
	return reason
}

func (g *tencentInstanceAdjustmentGateway) Execute(ctx context.Context, operation adjustmentOperation) (string, error) {
	switch operation.Type {
	case adjustmentTypeInstanceType:
		req := cvm.NewResetInstancesTypeRequest()
		req.InstanceIds = tccommon.StringPtrs([]string{operation.InstanceID})
		req.InstanceType = tccommon.StringPtr(operation.TargetInstanceType)
		req.ForceStop = tccommon.BoolPtr(operation.ForceStop)
		var resp *cvm.ResetInstancesTypeResponse
		err := withCloudActionGate(ctx, "ResetInstancesType", func() error {
			var callErr error
			resp, callErr = CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.ResetInstancesType)
			return callErr
		})
		if err != nil || resp == nil || resp.Response == nil {
			return "", err
		}
		return StrVal(resp.Response.RequestId), nil
	case adjustmentTypeSystemDisk:
		req := cvm.NewResizeInstanceDisksRequest()
		req.InstanceId = tccommon.StringPtr(operation.InstanceID)
		req.SystemDisk = &cvm.SystemDisk{DiskId: tccommon.StringPtr(operation.DiskID), DiskSize: tccommon.Int64Ptr(operation.TargetDiskSize)}
		req.ForceStop = tccommon.BoolPtr(operation.ForceStop)
		req.ResizeOnline = tccommon.BoolPtr(operation.ResizeOnline)
		var resp *cvm.ResizeInstanceDisksResponse
		err := withCloudActionGate(ctx, "ResizeInstanceDisks", func() error {
			var callErr error
			resp, callErr = CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.ResizeInstanceDisks)
			return callErr
		})
		if err != nil || resp == nil || resp.Response == nil {
			return "", err
		}
		return StrVal(resp.Response.RequestId), nil
	default:
		return "", fmt.Errorf("unsupported adjustment type %q", operation.Type)
	}
}

func (g *tencentInstanceAdjustmentGateway) StartInstance(ctx context.Context, instanceID string) error {
	req := cvm.NewStartInstancesRequest()
	req.InstanceIds = tccommon.StringPtrs([]string{instanceID})
	return withCloudActionGate(ctx, "StartInstances", func() error {
		_, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.StartInstances)
		return err
	})
}

func (g *tencentInstanceAdjustmentGateway) StopInstance(ctx context.Context, instanceID, stoppedMode string) error {
	req := cvm.NewStopInstancesRequest()
	req.InstanceIds = tccommon.StringPtrs([]string{instanceID})
	req.StopType = tccommon.StringPtr("SOFT_FIRST")
	if stoppedMode == "KEEP_CHARGING" || stoppedMode == "STOP_CHARGING" {
		req.StoppedMode = tccommon.StringPtr(stoppedMode)
	}
	return withCloudActionGate(ctx, "StopInstances", func() error {
		_, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, g.cvmClient.StopInstances)
		return err
	})
}

func isTencentInstanceNotFound(err error) bool {
	sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError)
	return ok && strings.Contains(strings.ToLower(sdkErr.GetCode()), "instanceid.notfound")
}

func mapAdjustmentCloudError(err error) string {
	if err == nil {
		return ""
	}
	code := strings.ToLower(err.Error())
	if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
		code = strings.ToLower(sdkErr.GetCode() + " " + sdkErr.GetMessage())
	}
	switch {
	case strings.Contains(code, "instanceid.notfound"), strings.Contains(code, "resourcenotfound.instance"):
		return reasonCVMInstanceNotFound
	case strings.Contains(code, "balance"), strings.Contains(code, "accountarrears"):
		return reasonInsufficientBalance
	case strings.Contains(code, "unpaid"), strings.Contains(code, "order"):
		return reasonUnpaidOrder
	case strings.Contains(code, "quota"):
		return reasonResourceQuotaExceeded
	case strings.Contains(code, "soldout"), strings.Contains(code, "stock"), strings.Contains(code, "resourceinsufficient"):
		return reasonResourceSoldOut
	case strings.Contains(code, "promotion"):
		return reasonPromotionRestricted
	case strings.Contains(code, "base_network"), strings.Contains(code, "basenetwork"), strings.Contains(code, "vpc"):
		return reasonInstanceNetworkIncompatible
	case strings.Contains(code, "eni"), strings.Contains(code, "eip"), strings.Contains(code, "bandwidth"):
		return reasonInstanceResourceLimitExceeded
	case strings.Contains(code, "redhat"), strings.Contains(code, "image"):
		return reasonInstanceImageNotSupported
	case strings.Contains(code, "disk") && (strings.Contains(code, "unavailable") || strings.Contains(code, "notavailable")):
		return reasonCloudDiskUnavailable
	case strings.Contains(code, "disk") && (strings.Contains(code, "migrat") || strings.Contains(code, "rollback")):
		return reasonDiskNotReady
	case strings.Contains(code, "invalidinstancestate"), strings.Contains(code, "operationinprogress"),
		strings.Contains(code, "operation in progress"), strings.Contains(code, "migrat"), strings.Contains(code, "rescue"):
		return reasonCVMOperationInProgress
	case strings.Contains(code, "charge"), strings.Contains(code, "spot"):
		return reasonUnsupportedChargeType
	case strings.Contains(code, "arm"), strings.Contains(code, "heterogeneous"), strings.Contains(code, "family"),
		strings.Contains(code, "swap"), strings.Contains(code, "localdisk"), strings.Contains(code, "special"),
		strings.Contains(code, "applicationrole"), strings.Contains(code, "emr"):
		return reasonInstanceFeatureNotSupported
	default:
		return reasonCloudAdjustmentFailed
	}
}

func deniedActionReason(actions []deniedAction) string {
	for _, action := range actions {
		text := strings.ToLower(action.Code + " " + action.Message)
		switch {
		case strings.Contains(text, "quota"):
			return reasonResourceQuotaExceeded
		case strings.Contains(text, "network"), strings.Contains(text, "vpc"):
			return reasonInstanceNetworkIncompatible
		case strings.Contains(text, "eni"), strings.Contains(text, "eip"), strings.Contains(text, "bandwidth"):
			return reasonInstanceResourceLimitExceeded
		case strings.Contains(text, "image"), strings.Contains(text, "redhat"):
			return reasonInstanceImageNotSupported
		case strings.Contains(text, "promotion"):
			return reasonPromotionRestricted
		default:
			return reasonInstanceFeatureNotSupported
		}
	}
	return ""
}

func isCloudDiskType(diskType string) bool {
	return strings.HasPrefix(diskType, "CLOUD_")
}

func diskBelongsToInstance(disk *adjustmentCloudDisk, instanceID string) bool {
	if disk.InstanceID == instanceID {
		return true
	}
	for _, id := range disk.InstanceIDs {
		if id == instanceID {
			return true
		}
	}
	return false
}

func diskIsReady(disk *adjustmentCloudDisk, instanceID string) bool {
	return disk != nil && disk.Attached && disk.DiskState == "ATTACHED" && !disk.Migrating && !disk.Rollbacking &&
		!disk.DeadlineError && !disk.AutoRenewError && disk.ErrorPrompt == "" && diskBelongsToInstance(disk, instanceID)
}

func instanceFamily(instanceType string) string {
	family, _, _ := strings.Cut(instanceType, ".")
	return strings.ToUpper(family)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringPtrValues(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func logBestEffortDeniedActionFailure(ctx context.Context, instanceID string, err error) {
	if err != nil {
		slog.WarnContext(ctx, "[InstanceAdjustment] denied-action 补充检查失败，继续必选询价", "instance_id", instanceID, "error", err)
	}
}
