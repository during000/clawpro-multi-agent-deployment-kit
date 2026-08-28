package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/model"
)

const notificationRetentionDays = 30

// 启动通知过期清理定时任务，每天凌晨 3:00 执行一次。
// 删除 created_at 超过 30 天的通知记录（物理删除）。
func init() {
	RegisterTask(TaskDef{
		Name:         "notification-cleanup",
		Interval:     24 * time.Hour,
		RunFunc:      runNotificationCleanup,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: calcNextThreeAM(),
	})
}

// calcNextThreeAM 计算距下一个凌晨 3:00 的时长，保持原有延迟逻辑。
func calcNextThreeAM() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
	return time.Until(next)
}

// runNotificationCleanup 清理超过 30 天的通知记录（物理删除）。
func runNotificationCleanup(ctx context.Context) {
	affected, err := model.CleanupExpiredNotifications(ctx, notificationRetentionDays)
	if err != nil {
		slog.Error("[NotifCleanup] 清理失败", "error", err)
	} else if affected > 0 {
		slog.Info("[NotifCleanup] 清理完成", "deleted", affected, "retention_days", notificationRetentionDays)
	}
}
