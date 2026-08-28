package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

const roleSyncScanInterval = 2 * time.Minute
const roleSyncScanBatch = 100

// 周期任务：每 2 分钟扫描 role_sync_status='updating' 的实例，逐个调
// refreshRoleRecord 兜底聚合状态。用于捕获 SOUL/技能异步任务成败但
// refresh 未及时触发的边缘情况。
func init() {
	RegisterTask(TaskDef{
		Name:         "role-sync-refresh",
		Interval:     roleSyncScanInterval,
		RunFunc:      runRoleSyncRefresh,
		NeedDistLock: true, // scheduler 会自动加分布式锁
		PerTenant:    true,
		InitialDelay: 30 * time.Second,
	})
}

// runRoleSyncRefresh 扫描 updating 状态的实例并逐个刷新聚合 record 状态。
func runRoleSyncRefresh(ctx context.Context) {
	logger := slog.With("task", "RoleSyncRefresh")

	var instances []model.Instance
	if err := model.DB(ctx).
		Select("id").
		Where("role_sync_status = ?", model.RoleSyncStatusUpdating).
		Limit(roleSyncScanBatch).Find(&instances).Error; err != nil {
		logger.Error("查询 updating 实例失败", "error", err)
		return
	}
	if len(instances) == 0 {
		return
	}

	logger.Info("开始聚合 updating 实例状态", "count", len(instances))
	for _, inst := range instances {
		controller.RefreshRoleRecord(ctx, inst.ID)
	}
}
