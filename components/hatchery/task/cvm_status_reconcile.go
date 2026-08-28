package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hatchery/controller"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "cvm-status-reconcile",
		Interval:     60 * time.Second, // 可配置，核心场景可降到 30s
		RunFunc:      runCVMStatusReconcile,
		NeedDistLock: true,  // 多副本互斥
		PerTenant:    true,  // 多租户分发
		InitialDelay: 5 * time.Second, // 启动后尽快跑第一轮（缓解冷启动）
	})
}

func runCVMStatusReconcile(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[cvm-status-reconcile] panic recovered", "error", fmt.Sprintf("%v", r))
		}
	}()
	controller.ReconcileInstanceStatuses(ctx)
	slog.Debug("[cvm-status-reconcile] 本轮完成")
}
