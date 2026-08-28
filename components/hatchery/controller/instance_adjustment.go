package controller

import (
	"context"
	"log/slog"

	"hatchery/model"
)

const (
	adjustmentTypeInstanceType = "instance_type"
	adjustmentTypeSystemDisk   = "system_disk"
	adjustmentResizeOnline     = "online"
	adjustmentResizeOffline    = "offline"

	adjustmentStatusProcessing = "processing"
	adjustmentStatusFailed     = "failed"

	adjustmentPhaseQueued         = "queued"
	adjustmentPhaseSubmitting     = "submitting"
	adjustmentPhasePolling        = "polling"
	adjustmentPhaseRestoreSuccess = "restore_success"
	adjustmentPhaseRestoreFailure = "restore_failure"
)

const (
	reasonInstanceNotFound              = "instance_not_found"
	reasonCloudInstanceRequired         = "cloud_instance_required"
	reasonDoctorNodeNotAllowed          = "doctor_node_not_allowed"
	reasonOperationInProgress           = "operation_in_progress"
	reasonInstanceStatusNotSupported    = "instance_status_not_supported"
	reasonCVMInstanceNotFound           = "cvm_instance_not_found"
	reasonCVMRestricted                 = "cvm_restricted"
	reasonCVMOperationInProgress        = "cvm_operation_in_progress"
	reasonCVMQueryFailed                = "cvm_query_failed"
	reasonStopChargingNotSupported      = "stop_charging_not_supported"
	reasonInvalidTarget                 = "invalid_target"
	reasonUnsupportedInstanceType       = "unsupported_instance_type"
	reasonInstanceTypeNotUpgrade        = "instance_type_not_upgrade"
	reasonInstanceTypeUnchanged         = "instance_type_unchanged"
	reasonInstanceTypeDowngrade         = "instance_type_downgrade_not_supported"
	reasonCloudDiskRequired             = "cloud_disk_required"
	reasonSystemDiskTypeNotSupported    = "system_disk_type_not_supported"
	reasonTargetInstanceTypeUnavailable = "target_instance_type_unavailable"
	reasonDiskQuotaUnavailable          = "disk_quota_unavailable"
	reasonUnsupportedChargeType         = "unsupported_charge_type"
	reasonDiskNotReady                  = "disk_not_ready"
	reasonCloudDiskUnavailable          = "cloud_disk_unavailable"
	reasonInstanceNetworkIncompatible   = "instance_network_incompatible"
	reasonInstanceResourceLimitExceeded = "instance_resource_limit_exceeded"
	reasonResourceQuotaExceeded         = "resource_quota_exceeded"
	reasonInstanceImageNotSupported     = "instance_image_not_supported"
	reasonInstanceFeatureNotSupported   = "instance_feature_not_supported"
	reasonPromotionRestricted           = "promotion_restricted"
	reasonInvalidDiskSize               = "invalid_disk_size"
	reasonDiskSizeUnchanged             = "disk_size_unchanged"
	reasonDiskShrinkNotSupported        = "disk_shrink_not_supported"
	reasonOnlineResizeNotSupported      = "online_resize_not_supported"
	reasonInsufficientBalance           = "insufficient_balance"
	reasonUnpaidOrder                   = "unpaid_order"
	reasonResourceSoldOut               = "resource_sold_out"
	reasonInternalError                 = "internal_error"
	reasonCloudAdjustmentFailed         = "cloud_adjustment_failed"
	reasonAdjustmentTimeout             = "adjustment_timeout"
	reasonAdjustmentRestoreFailed       = "adjustment_restore_failed"
)

// clearAdjustmentFailure 删除单个实例的失败调整记录。
func clearAdjustmentFailure(ctx context.Context, instanceID uint) {
	clearAdjustmentFailures(ctx, instanceID)
}

// clearAdjustmentFailures 分批删除实例的失败调整记录；失败仅告警，不影响已成功下发的实例操作。
func clearAdjustmentFailures(ctx context.Context, instanceIDs ...uint) {
	for _, chunk := range chunkUintIDs(instanceIDs, batchINChunkSize) {
		if err := model.DB(ctx).
			Where("instance_id IN ? AND status = ?", chunk, adjustmentStatusFailed).
			Delete(&model.InstanceAdjustment{}).Error; err != nil {
			slog.WarnContext(ctx, "[InstanceAdjustment] 清除失败调整记录失败",
				"instance_db_ids", chunk,
				"error", err,
			)
		}
	}
}
