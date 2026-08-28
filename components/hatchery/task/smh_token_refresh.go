package task

import (
	"context"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "smh-token-refresh",
		Interval:     12 * time.Hour,
		RunFunc:      runSMHTokenRefresh,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 1 * time.Minute,
	})
}

// runSMHTokenRefresh 刷新 SMH Access Token。
// SMH 未配置则跳过。
func runSMHTokenRefresh(ctx context.Context) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return
	}
	controller.InitSMHTokenRefresher(ctx, smhConfig)
}
