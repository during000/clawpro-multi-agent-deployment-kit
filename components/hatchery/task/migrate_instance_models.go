package task

import (
	"context"

	"hatchery/model"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "migrate-instance-models",
		Interval:     0, // 一次性
		RunFunc:      runMigrateInstanceModels,
		NeedDistLock: true,
		PerTenant:    true,
		Priority:     PriorityStartup,
	})
}

// runMigrateInstanceModels 执行多模型 Fallback 存量迁移（幂等）。
func runMigrateInstanceModels(ctx context.Context) {
	model.MigrateInstanceModels(ctx)
}
