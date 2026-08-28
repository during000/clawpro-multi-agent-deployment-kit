package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

type instanceAdjustmentRequest struct {
	IDs                  *[]uint   `json:"ids"`
	InstanceIDs          *[]string `json:"instance_ids"`
	AdjustmentType       string    `json:"adjustment_type"`
	TargetInstanceType   *string   `json:"target_instance_type"`
	TargetSystemDiskSize *int64    `json:"target_system_disk_size"`
	ResizeMode           string    `json:"resize_mode"`
}

type instanceAdjustmentResult struct {
	ID                    uint   `json:"id,omitempty"`
	InstanceID            string `json:"instance_id,omitempty"`
	CurrentInstanceType   string `json:"current_instance_type,omitempty"`
	CurrentSystemDiskType string `json:"current_system_disk_type,omitempty"`
	CurrentSystemDiskSize int64  `json:"current_system_disk_size,omitempty"`
	CurrentStatus         string `json:"current_status,omitempty"`
	Adjustable            bool   `json:"adjustable"`
	ReasonCode            string `json:"reason_code"`
	ReasonMessage         string `json:"reason_message"`
	MinDiskSize           int64  `json:"min_disk_size,omitempty"`
	MaxDiskSize           int64  `json:"max_disk_size,omitempty"`
	StepSize              int64  `json:"step_size,omitempty"`

	Status            string `json:"status,omitempty"`
	Accepted          bool   `json:"accepted,omitempty"`
	AlreadyProcessing bool   `json:"already_processing,omitempty"`

	instance  *model.Instance          `json:"-"`
	cloud     *adjustmentCloudInstance `json:"-"`
	operation adjustmentOperation      `json:"-"`
}

type instanceAdjustmentValidateResponse struct {
	AdjustableCount    int                        `json:"adjustable_count"`
	NonAdjustableCount int                        `json:"non_adjustable_count"`
	Results            []instanceAdjustmentResult `json:"results"`
}

type instanceAdjustmentSubmitResponse struct {
	AcceptedCount          int                        `json:"accepted_count"`
	RejectedCount          int                        `json:"rejected_count"`
	AlreadyProcessingCount int                        `json:"already_processing_count"`
	Results                []instanceAdjustmentResult `json:"results"`
}

type resolvedAdjustmentTarget struct {
	DBID       uint
	InstanceID string
	Instance   *model.Instance
	Adjustment *model.InstanceAdjustment
}

type instanceTypeAvailabilityResult struct {
	available bool
	err       error
}

type diskQuotaLookupResult struct {
	quota adjustmentDiskQuota
	err   error
}

const (
	adjustmentResultAccepted          = "accepted"
	adjustmentResultRejected          = "rejected"
	adjustmentResultAlreadyProcessing = "already_processing"
)

type adjustmentCloudFactory func(context.Context) (instanceAdjustmentCloudGateway, error)

func defaultAdjustmentCloudFactory(ctx context.Context) (instanceAdjustmentCloudGateway, error) {
	return newTencentInstanceAdjustmentGateway(ctx)
}

func HandleAdminInstanceAdjustmentValidate(w http.ResponseWriter, r *http.Request) {
	handleAdminInstanceAdjustmentValidate(w, r, defaultAdjustmentCloudFactory)
}

func handleAdminInstanceAdjustmentValidate(w http.ResponseWriter, r *http.Request, factory adjustmentCloudFactory) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	req, decodeErr := decodeAdminInstanceAdjustmentRequest(w, r)
	if decodeErr != nil {
		writeError(w, r, http.StatusBadRequest, decodeErr)
		return
	}
	targets, loadErr := resolveAdjustmentTargets(r.Context(), req)
	if loadErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(loadErr, i18n.MsgAdjustmentCloudUnavailable))
		return
	}
	gateway, factoryErr := factory(r.Context())
	if factoryErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(factoryErr, i18n.MsgAdjustmentCloudUnavailable))
		return
	}
	results, validateErr := validateResolvedAdjustmentTargets(r.Context(), req, targets, gateway, false)
	if validateErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(validateErr, i18n.MsgAdjustmentCloudUnavailable))
		return
	}
	resp := instanceAdjustmentValidateResponse{Results: results}
	for _, result := range results {
		if result.Adjustable {
			resp.AdjustableCount++
		} else {
			resp.NonAdjustableCount++
		}
	}
	jsonOK(w, resp)
}

func HandleAdminInstanceAdjustment(w http.ResponseWriter, r *http.Request) {
	handleAdminInstanceAdjustment(w, r, defaultAdjustmentCloudFactory)
}

func handleAdminInstanceAdjustment(w http.ResponseWriter, r *http.Request, factory adjustmentCloudFactory) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	ctx := r.Context()
	req, decodeErr := decodeAdminInstanceAdjustmentRequest(w, r)
	if decodeErr != nil {
		writeError(w, r, http.StatusBadRequest, decodeErr)
		return
	}
	targets, loadErr := resolveAdjustmentTargets(ctx, req)
	if loadErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(loadErr, i18n.MsgAdjustmentCloudUnavailable))
		return
	}

	// 提交阶段先识别不存在、重复提交和操作冲突，避免无效目标触发云 API。
	results := make([]instanceAdjustmentResult, len(targets))
	pendingTargets := make([]resolvedAdjustmentTarget, 0, len(targets))
	pendingIndexes := make([]int, 0, len(targets))
	for i, target := range targets {
		result := resultForTarget(target)
		switch {
		case target.Instance == nil:
			result.Status = adjustmentResultRejected
			rejectAdjustmentResult(ctx, &result, reasonInstanceNotFound)
		case target.Instance.CurrentOperation != "" ||
			target.Adjustment != nil && target.Adjustment.Status == adjustmentStatusProcessing:
			if sameAdjustmentRequest(target.Instance, target.Adjustment, req) {
				result.Status = adjustmentResultAlreadyProcessing
				result.AlreadyProcessing = true
			} else {
				result.Status = adjustmentResultRejected
				rejectAdjustmentResult(ctx, &result, reasonOperationInProgress)
			}
		default:
			pendingTargets = append(pendingTargets, target)
			pendingIndexes = append(pendingIndexes, i)
		}
		results[i] = result
	}

	// 仅对仍可受理的目标查询云上实时状态并执行完整校验。
	if len(pendingTargets) > 0 {
		gateway, err := factory(ctx)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAdjustmentCloudUnavailable))
			return
		}
		validated, err := validateResolvedAdjustmentTargets(ctx, req, pendingTargets, gateway, false)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAdjustmentCloudUnavailable))
			return
		}
		for i := range validated {
			results[pendingIndexes[i]] = validated[i]
		}
	}

	response := instanceAdjustmentSubmitResponse{Results: results}
	for i := range results {
		result := &results[i]
		if result.Status == adjustmentResultAlreadyProcessing {
			response.AlreadyProcessingCount++
			continue
		}
		if result.Status == adjustmentResultRejected || !result.Adjustable {
			result.Status = adjustmentResultRejected
			response.RejectedCount++
			continue
		}

		// 校验通过后再用事务抢占实例操作锁；并发请求只有一个能受理成功。
		accepted, err := acceptInstanceAdjustment(ctx, req, result)
		if err != nil {
			slog.ErrorContext(ctx, "[InstanceAdjustment] 受理单实例调整失败",
				"instance_db_id", result.ID,
				"instance_id", result.InstanceID,
				"error", err,
			)
			result.Status = adjustmentResultRejected
			result.Accepted = false
			rejectAdjustmentResult(ctx, result, reasonInternalError)
			response.RejectedCount++
			continue
		}
		if accepted {
			result.Status = adjustmentResultAccepted
			result.Accepted = true
			result.ReasonCode = ""
			result.ReasonMessage = ""
			response.AcceptedCount++
			continue
		}

		// CAS 抢锁失败时重新读库，区分相同请求的幂等提交和其他操作冲突。
		var instance model.Instance
		var adjustment model.InstanceAdjustment
		db := model.DB(ctx)
		sameRequest := db.First(&instance, result.ID).Error == nil &&
			db.Where("instance_id = ?", result.ID).First(&adjustment).Error == nil &&
			sameAdjustmentRequest(&instance, &adjustment, req)
		if sameRequest {
			result.Status = adjustmentResultAlreadyProcessing
			result.AlreadyProcessing = true
			result.Adjustable = false
			result.ReasonCode = ""
			result.ReasonMessage = ""
			response.AlreadyProcessingCount++
			continue
		}

		result.Status = adjustmentResultRejected
		result.Accepted = false
		rejectAdjustmentResult(ctx, result, reasonOperationInProgress)
		response.RejectedCount++
	}
	jsonOK(w, response)
}

func decodeAdminInstanceAdjustmentRequest(w http.ResponseWriter, r *http.Request) (instanceAdjustmentRequest, *hcommon.RichError) {
	var req instanceAdjustmentRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, hcommon.I18nRichError(err, i18n.MsgAdjustmentInvalidEnvelope)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return req, hcommon.I18nRichError(err, i18n.MsgAdjustmentInvalidEnvelope)
	}
	if (req.IDs == nil) == (req.InstanceIDs == nil) {
		return req, hcommon.I18nError(i18n.MsgAdjustmentIDsExclusive)
	}
	if req.IDs != nil {
		values := deduplicateUintTargets(*req.IDs)
		if len(values) < 1 || len(values) > 100 {
			return req, hcommon.I18nError(i18n.MsgAdjustmentTargetCount)
		}
		req.IDs = &values
	} else {
		values := deduplicateStringTargets(*req.InstanceIDs)
		if len(values) < 1 || len(values) > 100 {
			return req, hcommon.I18nError(i18n.MsgAdjustmentTargetCount)
		}
		req.InstanceIDs = &values
	}
	switch req.AdjustmentType {
	case adjustmentTypeInstanceType:
		if req.TargetInstanceType == nil {
			return req, hcommon.I18nError(i18n.MsgAdjustmentMissingInstanceType)
		}
		trimmed := strings.TrimSpace(*req.TargetInstanceType)
		req.TargetInstanceType = &trimmed
		req.TargetSystemDiskSize = nil
		req.ResizeMode = ""
	case adjustmentTypeSystemDisk:
		if req.TargetSystemDiskSize == nil {
			return req, hcommon.I18nError(i18n.MsgAdjustmentMissingDiskSize)
		}
		if req.ResizeMode != adjustmentResizeOnline && req.ResizeMode != adjustmentResizeOffline {
			return req, hcommon.I18nError(i18n.MsgAdjustmentInvalidResizeMode)
		}
		req.TargetInstanceType = nil
	default:
		return req, hcommon.I18nError(i18n.MsgAdjustmentInvalidType)
	}
	return req, nil
}

func deduplicateUintTargets(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func deduplicateStringTargets(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// resolveAdjustmentTargets 批量加载实例和调整任务，并按请求顺序保留不存在的目标。
func resolveAdjustmentTargets(ctx context.Context, req instanceAdjustmentRequest) ([]resolvedAdjustmentTarget, error) {
	var instances []model.Instance
	db := model.DB(ctx)
	if req.IDs != nil {
		if err := db.Where("id IN ?", *req.IDs).Find(&instances).Error; err != nil {
			return nil, err
		}
	} else if err := db.Where("instance_id IN ?", *req.InstanceIDs).Find(&instances).Error; err != nil {
		return nil, err
	}

	instanceIDs := make([]uint, 0, len(instances))
	byID := make(map[uint]*model.Instance, len(instances))
	byCloudID := make(map[string]*model.Instance, len(instances))
	for i := range instances {
		instance := &instances[i]
		instanceIDs = append(instanceIDs, instance.ID)
		byID[instance.ID] = instance
		byCloudID[instance.InstanceId] = instance
	}
	var adjustments []model.InstanceAdjustment
	if len(instanceIDs) > 0 {
		if err := db.Where("instance_id IN ?", instanceIDs).Find(&adjustments).Error; err != nil {
			return nil, err
		}
	}
	byAdjustmentInstanceID := make(map[uint]*model.InstanceAdjustment, len(adjustments))
	for i := range adjustments {
		byAdjustmentInstanceID[adjustments[i].InstanceID] = &adjustments[i]
	}

	if req.IDs != nil {
		result := make([]resolvedAdjustmentTarget, 0, len(*req.IDs))
		for _, id := range *req.IDs {
			result = append(result, resolvedAdjustmentTarget{DBID: id, Instance: byID[id], Adjustment: byAdjustmentInstanceID[id]})
		}
		return result, nil
	}
	result := make([]resolvedAdjustmentTarget, 0, len(*req.InstanceIDs))
	for _, instanceID := range *req.InstanceIDs {
		instance := byCloudID[instanceID]
		var adjustment *model.InstanceAdjustment
		if instance != nil {
			adjustment = byAdjustmentInstanceID[instance.ID]
		}
		result = append(result, resolvedAdjustmentTarget{InstanceID: instanceID, Instance: instance, Adjustment: adjustment})
	}
	return result, nil
}

// validateResolvedAdjustmentTargets 先做本地校验，再批量查询云实例、云盘并校验调整条件。
func validateResolvedAdjustmentTargets(ctx context.Context, req instanceAdjustmentRequest, targets []resolvedAdjustmentTarget, gateway instanceAdjustmentCloudGateway, allowOwnAdjustment bool) ([]instanceAdjustmentResult, error) {
	results, cloudIDs := precheckAdjustmentTargets(ctx, req, targets, allowOwnAdjustment)

	cloudInstances, err := gateway.DescribeInstances(ctx, cloudIDs)
	if err != nil {
		return nil, err
	}
	cloudDisks, err := gateway.DescribeDisks(ctx, collectAdjustmentDiskIDs(cloudInstances))
	if err != nil {
		return nil, err
	}

	// 同一批次复用机型可用性和云盘配额查询，避免重复调用云 API。
	availabilityCache := map[string]instanceTypeAvailabilityResult{}
	quotaCache := map[string]diskQuotaLookupResult{}
	for i := range results {
		result := &results[i]
		if result.ReasonCode != "" || result.instance == nil {
			continue
		}
		cloud := cloudInstances[result.instance.InstanceId]
		if !validateAdjustmentCloudInstance(ctx, result, cloud) {
			continue
		}
		switch req.AdjustmentType {
		case adjustmentTypeInstanceType:
			validateInstanceTypeAdjustment(ctx, req, result, gateway, cloudDisks, availabilityCache)
		case adjustmentTypeSystemDisk:
			validateSystemDiskAdjustment(ctx, req, result, gateway, cloudDisks, quotaCache)
		}
	}
	return results, nil
}

// precheckAdjustmentTargets 处理无需查询云 API 即可确定的拒绝条件。
func precheckAdjustmentTargets(ctx context.Context, req instanceAdjustmentRequest, targets []resolvedAdjustmentTarget, allowOwnAdjustment bool) ([]instanceAdjustmentResult, []string) {
	results := make([]instanceAdjustmentResult, len(targets))
	cloudIDs := make([]string, 0, len(targets))
	seenCloudIDs := make(map[string]struct{}, len(targets))

	for i, target := range targets {
		result := resultForTarget(target)
		results[i] = result
		if target.Instance == nil {
			rejectAdjustmentResult(ctx, &results[i], reasonInstanceNotFound)
			continue
		}
		instance := target.Instance
		switch {
		case instance.Source == model.InstanceSourceLocal:
			rejectAdjustmentResult(ctx, &results[i], reasonCloudInstanceRequired)
		case instance.IsDoctorNode:
			rejectAdjustmentResult(ctx, &results[i], reasonDoctorNodeNotAllowed)
		case (instance.CurrentOperation != "" ||
			target.Adjustment != nil && target.Adjustment.Status == adjustmentStatusProcessing) &&
			!(allowOwnAdjustment && sameAdjustmentRequest(instance, target.Adjustment, req)):
			rejectAdjustmentResult(ctx, &results[i], reasonOperationInProgress)
		case instance.LastKnownStatus != "" &&
			instance.LastKnownStatus != model.StatusRunning &&
			instance.LastKnownStatus != model.StatusStopped:
			rejectAdjustmentResult(ctx, &results[i], reasonInstanceStatusNotSupported)
		case instance.InstanceId == "":
			rejectAdjustmentResult(ctx, &results[i], reasonCVMInstanceNotFound)
		default:
			if _, seen := seenCloudIDs[instance.InstanceId]; !seen {
				seenCloudIDs[instance.InstanceId] = struct{}{}
				cloudIDs = append(cloudIDs, instance.InstanceId)
			}
		}
	}
	return results, cloudIDs
}

func collectAdjustmentDiskIDs(instances map[string]*adjustmentCloudInstance) []string {
	diskIDs := make([]string, 0, len(instances))
	seen := make(map[string]struct{})
	add := func(diskID string) {
		if diskID == "" {
			return
		}
		if _, exists := seen[diskID]; exists {
			return
		}
		seen[diskID] = struct{}{}
		diskIDs = append(diskIDs, diskID)
	}
	for _, instance := range instances {
		add(instance.SystemDiskID)
		for _, disk := range instance.DataDisks {
			add(disk.DiskID)
		}
	}
	return diskIDs
}

// validateAdjustmentCloudInstance 以云上实时状态为准，补齐结果并拦截不可调整状态。
func validateAdjustmentCloudInstance(ctx context.Context, result *instanceAdjustmentResult, cloud *adjustmentCloudInstance) bool {
	if cloud == nil {
		rejectAdjustmentResult(ctx, result, reasonCVMInstanceNotFound)
		return false
	}
	result.cloud = cloud
	result.InstanceID = cloud.InstanceID
	result.CurrentInstanceType = cloud.InstanceType
	result.CurrentSystemDiskType = cloud.SystemDiskType
	result.CurrentSystemDiskSize = cloud.SystemDiskSize
	result.CurrentStatus = strings.ToLower(cloud.State)

	switch {
	case cloud.State != "RUNNING" && cloud.State != "STOPPED":
		rejectAdjustmentResult(ctx, result, reasonInstanceStatusNotSupported)
	case cloud.RestrictState != "NORMAL":
		rejectAdjustmentResult(ctx, result, reasonCVMRestricted)
	case cloud.LatestOperationState == "OPERATING":
		rejectAdjustmentResult(ctx, result, reasonCVMOperationInProgress)
	case cloud.StopChargingMode == "STOP_CHARGING":
		rejectAdjustmentResult(ctx, result, reasonStopChargingNotSupported)
	case cloud.ChargeType != "PREPAID" && cloud.ChargeType != "POSTPAID_BY_HOUR":
		rejectAdjustmentResult(ctx, result, reasonUnsupportedChargeType)
	default:
		return true
	}
	return false
}

// validateInstanceTypeAdjustment 只允许 AI2 规格向上调整，并验证云盘、售卖和询价条件。
func validateInstanceTypeAdjustment(ctx context.Context, req instanceAdjustmentRequest, result *instanceAdjustmentResult, gateway instanceAdjustmentCloudGateway, disks map[string]*adjustmentCloudDisk, availabilityCache map[string]instanceTypeAvailabilityResult) {
	cloud := result.cloud
	target := *req.TargetInstanceType
	targetRank, ok := model.AI2InstanceTypeRank(target)
	if !ok {
		rejectAdjustmentResult(ctx, result, reasonUnsupportedInstanceType)
		return
	}
	currentRank, ok := model.AI2InstanceTypeRank(cloud.InstanceType)
	if !ok {
		rejectAdjustmentResult(ctx, result, reasonInstanceTypeNotUpgrade)
		return
	}
	switch {
	case targetRank == currentRank:
		rejectAdjustmentResult(ctx, result, reasonInstanceTypeUnchanged)
		return
	case targetRank < currentRank:
		rejectAdjustmentResult(ctx, result, reasonInstanceTypeDowngrade)
		return
	}
	if !isCloudDiskType(cloud.SystemDiskType) || cloud.SystemDiskID == "" {
		rejectAdjustmentResult(ctx, result, reasonCloudDiskRequired)
		return
	}
	systemDisk := disks[cloud.SystemDiskID]
	if systemDisk == nil || systemDisk.DiskUsage != "SYSTEM_DISK" || !diskIsReady(systemDisk, cloud.InstanceID) {
		rejectAdjustmentResult(ctx, result, reasonDiskNotReady)
		return
	}
	if cloud.SystemDiskSize > 0 && systemDisk.DiskSize != cloud.SystemDiskSize {
		rejectAdjustmentResult(ctx, result, reasonDiskNotReady)
		return
	}
	if !isCloudDiskType(systemDisk.DiskType) {
		rejectAdjustmentResult(ctx, result, reasonCloudDiskRequired)
		return
	}
	if _, ok := resetInstancesTypeSupportedDiskTypes[systemDisk.DiskType]; !ok {
		rejectAdjustmentResult(ctx, result, reasonSystemDiskTypeNotSupported)
		return
	}
	result.CurrentSystemDiskType = systemDisk.DiskType
	result.CurrentSystemDiskSize = systemDisk.DiskSize
	for _, dataDisk := range cloud.DataDisks {
		if !isCloudDiskType(dataDisk.DiskType) || dataDisk.DiskID == "" {
			rejectAdjustmentResult(ctx, result, reasonCloudDiskRequired)
			return
		}
		if disk := disks[dataDisk.DiskID]; disk == nil || !isCloudDiskType(disk.DiskType) || disk.DiskUsage != "DATA_DISK" || !diskIsReady(disk, cloud.InstanceID) {
			rejectAdjustmentResult(ctx, result, reasonDiskNotReady)
			return
		}
	}
	cacheKey := strings.Join([]string{cloud.Zone, cloud.ChargeType, target}, "|")
	availability, ok := availabilityCache[cacheKey]
	if !ok {
		available, err := gateway.CheckInstanceTypeAvailable(ctx, cloud, target)
		availability = instanceTypeAvailabilityResult{available: available, err: err}
		availabilityCache[cacheKey] = availability
	}
	if availability.err != nil || !availability.available {
		rejectAdjustmentResult(ctx, result, reasonTargetInstanceTypeUnavailable)
		return
	}
	actions, deniedErr := gateway.DeniedActions(ctx, cloud.InstanceID, []string{"ResetInstancesType"})
	if deniedErr != nil {
		logBestEffortDeniedActionFailure(ctx, cloud.InstanceID, deniedErr)
	} else if reason := deniedActionReason(actions); reason != "" {
		rejectAdjustmentResult(ctx, result, reason)
		return
	}
	operation := adjustmentOperation{
		Type:               adjustmentTypeInstanceType,
		InstanceID:         cloud.InstanceID,
		TargetInstanceType: target,
		ForceStop:          cloud.State == "RUNNING",
	}
	if err := gateway.InquiryInstanceType(ctx, operation); err != nil {
		rejectAdjustmentResult(ctx, result, mapAdjustmentCloudError(err))
		return
	}
	result.operation = operation
	result.Adjustable = true
}

// validateSystemDiskAdjustment 校验系统盘实时属性、配额和扩容步长，生成执行参数。
func validateSystemDiskAdjustment(ctx context.Context, req instanceAdjustmentRequest, result *instanceAdjustmentResult, gateway instanceAdjustmentCloudGateway, disks map[string]*adjustmentCloudDisk, quotaCache map[string]diskQuotaLookupResult) {
	cloud := result.cloud
	if !isCloudDiskType(cloud.SystemDiskType) || cloud.SystemDiskID == "" {
		rejectAdjustmentResult(ctx, result, reasonCloudDiskRequired)
		return
	}
	disk := disks[cloud.SystemDiskID]
	if disk == nil || disk.DiskUsage != "SYSTEM_DISK" || disk.Portable || !diskIsReady(disk, cloud.InstanceID) {
		rejectAdjustmentResult(ctx, result, reasonDiskNotReady)
		return
	}
	if cloud.SystemDiskSize > 0 && disk.DiskSize != cloud.SystemDiskSize {
		rejectAdjustmentResult(ctx, result, reasonDiskNotReady)
		return
	}
	result.CurrentSystemDiskType = disk.DiskType
	result.CurrentSystemDiskSize = disk.DiskSize
	if disk.DiskChargeType != "PREPAID" && disk.DiskChargeType != "POSTPAID_BY_HOUR" {
		rejectAdjustmentResult(ctx, result, reasonUnsupportedChargeType)
		return
	}
	cacheKey := fmt.Sprintf("%s|%s|%s|%s|%d|%d", cloud.Zone, instanceFamily(cloud.InstanceType), disk.DiskType, disk.DiskChargeType, cloud.CPU, cloud.MemoryGB)
	quotaEntry, ok := quotaCache[cacheKey]
	if !ok {
		quota, err := gateway.GetSystemDiskQuota(ctx, cloud, disk)
		quotaEntry = diskQuotaLookupResult{quota: quota, err: err}
		quotaCache[cacheKey] = quotaEntry
	}
	if quotaEntry.err != nil || !quotaEntry.quota.Available || quotaEntry.quota.MaxSize <= 0 {
		rejectAdjustmentResult(ctx, result, reasonDiskQuotaUnavailable)
		return
	}
	quota := quotaEntry.quota
	if quota.StepSize <= 0 {
		quota.StepSize = 1
	}
	result.MinDiskSize = firstValidDiskSizeAbove(disk.DiskSize, quota)
	result.MaxDiskSize = quota.MaxSize
	result.StepSize = quota.StepSize
	target := *req.TargetSystemDiskSize
	switch {
	case target == disk.DiskSize:
		rejectAdjustmentResult(ctx, result, reasonDiskSizeUnchanged)
		return
	case target < disk.DiskSize:
		rejectAdjustmentResult(ctx, result, reasonDiskShrinkNotSupported)
		return
	case target < quota.MinSize || target > quota.MaxSize || (target-quota.MinSize)%quota.StepSize != 0:
		rejectAdjustmentResult(ctx, result, reasonInvalidDiskSize)
		return
	}
	operation := adjustmentOperation{
		Type:           adjustmentTypeSystemDisk,
		InstanceID:     cloud.InstanceID,
		DiskID:         disk.DiskID,
		TargetDiskSize: target,
		ResizeOnline:   cloud.State == "RUNNING" && req.ResizeMode == adjustmentResizeOnline,
		ForceStop:      cloud.State == "RUNNING" && req.ResizeMode == adjustmentResizeOffline,
	}
	actions, deniedErr := gateway.DeniedActions(ctx, cloud.InstanceID, []string{"ResizeInstanceDisks"})
	if deniedErr != nil {
		logBestEffortDeniedActionFailure(ctx, cloud.InstanceID, deniedErr)
	} else if reason := deniedActionReason(actions); reason != "" {
		rejectAdjustmentResult(ctx, result, reason)
		return
	}
	result.operation = operation
	result.Adjustable = true
}

func firstValidDiskSizeAbove(current int64, quota adjustmentDiskQuota) int64 {
	step := quota.StepSize
	if step <= 0 {
		step = 1
	}
	candidate := quota.MinSize
	if candidate <= current {
		delta := current + 1 - quota.MinSize
		candidate = quota.MinSize + ((delta+step-1)/step)*step
	}
	if candidate > quota.MaxSize {
		return 0
	}
	return candidate
}

func resultForTarget(target resolvedAdjustmentTarget) instanceAdjustmentResult {
	result := instanceAdjustmentResult{ID: target.DBID, InstanceID: target.InstanceID}
	if target.Instance != nil {
		result.ID = target.Instance.ID
		result.InstanceID = target.Instance.InstanceId
		result.CurrentInstanceType = target.Instance.CVMInstanceType
		result.CurrentSystemDiskType = target.Instance.SystemDiskType
		result.CurrentSystemDiskSize = target.Instance.SystemDiskSize
		result.CurrentStatus = target.Instance.LastKnownStatus
		result.instance = target.Instance
	}
	return result
}

func sameAdjustmentRequest(instance *model.Instance, adjustment *model.InstanceAdjustment, req instanceAdjustmentRequest) bool {
	if instance == nil || adjustment == nil || adjustment.Status != adjustmentStatusProcessing {
		return false
	}
	payload, err := adjustment.Payload()
	if err != nil {
		return false
	}
	switch req.AdjustmentType {
	case adjustmentTypeInstanceType:
		return instance.CurrentOperation == model.OpAdjustInstanceType && req.TargetInstanceType != nil &&
			payload.TargetInstanceType == *req.TargetInstanceType
	case adjustmentTypeSystemDisk:
		return instance.CurrentOperation == model.OpAdjustSystemDisk && req.TargetSystemDiskSize != nil &&
			payload.TargetDiskSize == *req.TargetSystemDiskSize && payload.ResizeMode == req.ResizeMode
	default:
		return false
	}
}

// acceptInstanceAdjustment 在一个事务内抢占操作锁并创建调整任务。
func acceptInstanceAdjustment(ctx context.Context, req instanceAdjustmentRequest, result *instanceAdjustmentResult) (bool, error) {
	now := time.Now()
	operationName := model.OpAdjustInstanceType
	if req.AdjustmentType == adjustmentTypeSystemDisk {
		operationName = model.OpAdjustSystemDisk
	}
	task := model.InstanceAdjustment{
		Identifier: result.instance.Identifier,
		InstanceID: result.ID,
		Status:     adjustmentStatusProcessing,
		Type:       req.AdjustmentType,
		Phase:      adjustmentPhaseQueued,
		RunAt:      now,
	}
	if err := task.SetPayload(model.InstanceAdjustmentPayload{
		TargetInstanceType:       result.operation.TargetInstanceType,
		TargetDiskSize:           result.operation.TargetDiskSize,
		ResizeMode:               req.ResizeMode,
		OriginalCVMState:         result.cloud.State,
		OriginalStopChargingMode: result.cloud.StopChargingMode,
	}); err != nil {
		return false, err
	}

	accepted := false
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 任务表和实例操作锁双重判断，兼容历史异常数据并保证同实例只有一个活动任务。
		var activeTaskCount int64
		if err := tx.Model(&model.InstanceAdjustment{}).
			Where("instance_id = ? AND status = ?", result.ID, adjustmentStatusProcessing).
			Count(&activeTaskCount).Error; err != nil {
			return err
		}
		if activeTaskCount > 0 {
			return nil
		}
		// current_operation 为空才允许抢锁；RowsAffected 是并发受理的最终判定。
		update := tx.Model(&model.Instance{}).
			Where("id = ? AND current_operation = ?", result.ID, model.OpNone).
			Updates(map[string]any{
				"current_operation":            operationName,
				"current_operation_state":      model.OpStateProcessing,
				"current_operation_updated_at": now,
				"last_stable_state":            result.CurrentStatus,
				"cvm_instance_type":            result.cloud.InstanceType,
				"cvm_cpu":                      result.cloud.CPU,
				"cvm_memory_gb":                result.cloud.MemoryGB,
				"system_disk_type":             result.CurrentSystemDiskType,
				"system_disk_size":             result.CurrentSystemDiskSize,
				"status_synced_at":             now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return nil
		}
		// 新任务受理成功前清理旧失败记录，保持每个实例至多一条调整任务。
		if err := tx.Where("instance_id = ? AND status = ?", result.ID, adjustmentStatusFailed).
			Delete(&model.InstanceAdjustment{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		accepted = true
		return nil
	})
	return accepted, err
}

func rejectAdjustmentResult(ctx context.Context, result *instanceAdjustmentResult, code string) {
	result.Adjustable = false
	result.ReasonCode = code
	result.ReasonMessage = i18n.T(ctx, adjustmentReasonKey(code))
}

var adjustmentReasonKeys = map[string]i18n.Key{
	reasonInstanceNotFound:              i18n.MsgInstanceNotFound,
	reasonCloudInstanceRequired:         i18n.MsgAdjustmentReasonCloudRequired,
	reasonDoctorNodeNotAllowed:          i18n.MsgAdjustmentReasonDoctorNode,
	reasonOperationInProgress:           i18n.MsgAdjustmentReasonOperationInProgress,
	reasonInstanceStatusNotSupported:    i18n.MsgAdjustmentReasonStatus,
	reasonCVMInstanceNotFound:           i18n.MsgAdjustmentReasonCVMNotFound,
	reasonCVMRestricted:                 i18n.MsgAdjustmentReasonCVMRestricted,
	reasonCVMOperationInProgress:        i18n.MsgAdjustmentReasonCVMOperation,
	reasonCVMQueryFailed:                i18n.MsgAdjustmentReasonCVMQuery,
	reasonStopChargingNotSupported:      i18n.MsgAdjustmentReasonStopCharging,
	reasonInvalidTarget:                 i18n.MsgAdjustmentReasonInvalidTarget,
	reasonUnsupportedInstanceType:       i18n.MsgAdjustmentReasonUnsupportedType,
	reasonInstanceTypeNotUpgrade:        i18n.MsgAdjustmentReasonNotUpgrade,
	reasonInstanceTypeUnchanged:         i18n.MsgAdjustmentReasonInstanceTypeUnchanged,
	reasonInstanceTypeDowngrade:         i18n.MsgAdjustmentReasonInstanceTypeDowngrade,
	reasonCloudDiskRequired:             i18n.MsgAdjustmentReasonCloudDisk,
	reasonSystemDiskTypeNotSupported:    i18n.MsgAdjustmentReasonDiskType,
	reasonTargetInstanceTypeUnavailable: i18n.MsgAdjustmentReasonTypeUnavailable,
	reasonDiskQuotaUnavailable:          i18n.MsgAdjustmentReasonDiskQuota,
	reasonUnsupportedChargeType:         i18n.MsgAdjustmentReasonChargeType,
	reasonDiskNotReady:                  i18n.MsgAdjustmentReasonDiskNotReady,
	reasonCloudDiskUnavailable:          i18n.MsgAdjustmentReasonCloudDiskUnavailable,
	reasonInstanceNetworkIncompatible:   i18n.MsgAdjustmentReasonNetwork,
	reasonInstanceResourceLimitExceeded: i18n.MsgAdjustmentReasonResourceLimit,
	reasonResourceQuotaExceeded:         i18n.MsgAdjustmentReasonResourceQuota,
	reasonInstanceImageNotSupported:     i18n.MsgAdjustmentReasonImage,
	reasonInstanceFeatureNotSupported:   i18n.MsgAdjustmentReasonFeature,
	reasonPromotionRestricted:           i18n.MsgAdjustmentReasonPromotion,
	reasonInvalidDiskSize:               i18n.MsgAdjustmentReasonDiskSize,
	reasonDiskSizeUnchanged:             i18n.MsgAdjustmentReasonDiskMustGrow,
	reasonDiskShrinkNotSupported:        i18n.MsgAdjustmentReasonDiskShrink,
	reasonOnlineResizeNotSupported:      i18n.MsgAdjustmentReasonOnlineResize,
	reasonInsufficientBalance:           i18n.MsgAdjustmentReasonBalance,
	reasonUnpaidOrder:                   i18n.MsgAdjustmentReasonUnpaidOrder,
	reasonResourceSoldOut:               i18n.MsgAdjustmentReasonSoldOut,
	reasonInternalError:                 i18n.MsgAdjustmentReasonInternalError,
	reasonAdjustmentTimeout:             i18n.MsgAdjustmentReasonTimeout,
	reasonAdjustmentRestoreFailed:       i18n.MsgAdjustmentReasonRestoreFailed,
}

func adjustmentReasonKey(code string) i18n.Key {
	if key, ok := adjustmentReasonKeys[code]; ok {
		return key
	}
	return i18n.MsgAdjustmentReasonCloudFailed
}
