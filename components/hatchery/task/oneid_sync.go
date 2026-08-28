package task

import (
	"context"
	"time"

	"hatchery/controller"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "oneid-profile-sync",
		Interval:     6 * time.Hour,
		RunFunc:      runOneIDSync,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
		InitialDelay: 30 * time.Second,
	})
}

// runOneIDSync 执行一轮 OneID 通讯录同步。
func runOneIDSync(ctx context.Context) {
	controller.SyncViaGateway(ctx)
}
