package task

import (
	"context"
	"time"

	"hatchery/model"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "clean-expired-sessions",
		Interval:     1 * time.Hour,
		RunFunc:      runCleanExpiredSessions,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runCleanExpiredSessions 清理过期的 OneID session 黑名单。
func runCleanExpiredSessions(ctx context.Context) {
	model.CleanExpiredSessions(ctx)
}
