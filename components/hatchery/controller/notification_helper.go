package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	hcommon "hatchery/common"
	"hatchery/model"
)

// createErrorNotification 从 error 提取结构化信息，创建 error 类通知。
// 自动识别 RichError 并提取 Detail/RequestId/InstanceId。
// title 作为通知标题，同时也作为通知消息正文（Message）。
// error_detail JSON 中包含完整的错误信息供"复制详情"使用。
// 如果 RichError 中 InstanceId 为空，自动从 DB 回查 CVM 实例 ID 补全。
// 可选传入 context.Context，自动提取 request_id 做兜底。
//
// 声明为 var 而非 func，以便测试中可替换为同步/阻塞版本，
// 保证 go createErrorNotification(...) 的异步 goroutine 在测试结束前完成，避免 DB 替换后触发 "no such table"。
var createErrorNotification = func(
	userID, instanceID uint,
	instanceName, notifyType, title string,
	err error,
	ctx context.Context,
) {
	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(ctx, err)}
	var richErr *hcommon.RichError
	if errors.As(err, &richErr) {
		detail.Detail = hcommon.ErrorDetailWithCtx(ctx, richErr)
		// 优先使用业务请求 ID（方便查日志），其次用 TAT/SDK RequestId
		if richErr.BizRequestId != "" {
			detail.RequestId = richErr.BizRequestId
		} else {
			detail.RequestId = richErr.RequestId
		}
		detail.InstanceId = richErr.InstanceId
	}
	// 兜底：从 context 中提取 request_id
	if detail.RequestId == "" && ctx != nil {
		if reqID := hcommon.GetRequestID(ctx); reqID != "" {
			detail.RequestId = reqID
		}
	}
	// 兜底：RichError 中无 InstanceId 时，从 DB 回查 CVM 实例 ID
	if detail.InstanceId == "" && instanceID > 0 {
		var inst model.Instance
		if model.DB(ctx).Select("instance_id").First(&inst, instanceID).Error == nil && inst.InstanceId != "" {
			detail.InstanceId = inst.InstanceId
		}
	}
	model.CreateNotificationWithCategory(
		ctx,
		userID, instanceID, instanceName,
		notifyType, model.NotifCategoryError,
		title, title, detail,
	)
}

// notifyQuotaExceeded 配额超限通知（自然日去重：当天只通知一次）。
// 配额按自然日重置（0 点清零），通知也按自然日去重。
// 使用分布式锁防止多副本并发写入重复通知。
func notifyQuotaExceeded(ctx context.Context, userID, instanceID uint, instanceName, message string) {
	lockKey := fmt.Sprintf("notify:quota_exceeded:%d", userID)
	lock, err := model.TryLock(ctx, lockKey)
	if err != nil {
		slog.Debug("[notifyQuotaExceeded] 获取锁失败，跳过", "user_id", userID, "error", err)
		return
	}
	defer lock.Release()

	todayStart := model.LocalToday()
	var count int64
	model.DB(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND type = ? AND created_at >= ?", userID, model.NotifyTypeQuotaExceeded, todayStart).
		Count(&count)
	if count > 0 {
		return // 今天已通知过，跳过
	}
	model.CreateNotificationWithCategory(
		ctx,
		userID, instanceID, instanceName,
		model.NotifyTypeQuotaExceeded, model.NotifCategoryError,
		"Token 配额已用尽", message, nil,
	)
}
